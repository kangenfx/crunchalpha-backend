package analyst

import (
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// ── EA Auth Middleware ────────────────────────────────────────────────────────
func (h *Handler) EAAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		key := c.GetHeader("X-EA-Key")
		if key == "" {
			key = c.Query("ea_key")
		}
		if key == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"ok": false, "error": "missing EA key"})
			c.Abort()
			return
		}
		var id string
		err := h.DB.QueryRow(`
			SELECT id FROM analyst_ea_keys 
			WHERE key_hash=$1`, key).Scan(&id)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"ok": false, "error": "invalid EA key"})
			c.Abort()
			return
		}
		// Update last_used
		h.DB.Exec(`UPDATE analyst_ea_keys SET last_used=now() WHERE id=$1`, id)
		c.Set("ea_key_id", id)
		c.Next()
	}
}

// ── GET /api/ea/analyst/pending-signals ──────────────────────────────────────
// EA polls this every tick to get all PENDING + RUNNING signals
func (h *Handler) EAGetSignals(c *gin.Context) {
	rows, err := h.DB.Query(`
		SELECT id, pair, direction, entry, sl, tp, status, COALESCE(market_price_at_creation,'0')
		FROM analyst_signals
		WHERE status IN ('PENDING','RUNNING')
		ORDER BY id ASC`)
	if err != nil {
		c.JSON(500, gin.H{"ok": false, "error": "db error"})
		return
	}
	defer rows.Close()

	type EASignal struct {
		ID          int64   `json:"id"`
		Pair        string  `json:"pair"`
		Direction   string  `json:"direction"`
		Entry       float64 `json:"entry"`
		SL          float64 `json:"sl"`
		TP          float64 `json:"tp"`
		Status      string  `json:"status"`
		MarketPrice float64 `json:"marketPrice"`
	}

	signals := make([]EASignal, 0)
	for rows.Next() {
		var s EASignal
		var entryS, slS, tpS, mktS string
		rows.Scan(&s.ID, &s.Pair, &s.Direction, &entryS, &slS, &tpS, &s.Status, &mktS)
		s.MarketPrice, _ = strconv.ParseFloat(mktS, 64)
		s.Entry, _ = strconv.ParseFloat(entryS, 64)
		s.SL, _    = strconv.ParseFloat(slS, 64)
		s.TP, _    = strconv.ParseFloat(tpS, 64)
		signals = append(signals, s)
	}
	c.JSON(200, gin.H{"ok": true, "signals": signals, "count": len(signals)})
}

// ── POST /api/ea/analyst/update-signal ───────────────────────────────────────
// EA posts price updates → backend decides status change
type EAUpdateReq struct {
	SignalID  int64   `json:"signal_id"`
	Pair      string  `json:"pair"`
	BidPrice  float64 `json:"bid"`
	AskPrice  float64 `json:"ask"`
}

func (h *Handler) EAUpdateSignal(c *gin.Context) {
	var req EAUpdateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"ok": false, "error": "invalid json"})
		return
	}

	// Fetch current signal
	var id int64
	var direction, entryS, slS, tpS, status string
	err := h.DB.QueryRow(`
		SELECT id, direction, entry, sl, tp, status
		FROM analyst_signals WHERE id=$1 AND status IN ('PENDING','RUNNING')`,
		req.SignalID).Scan(&id, &direction, &entryS, &slS, &tpS, &status)
	if err != nil {
		c.JSON(404, gin.H{"ok": false, "error": "signal not found or not active"})
		return
	}

	entry, _ := strconv.ParseFloat(entryS, 64)
	sl, _    := strconv.ParseFloat(slS, 64)
	tp, _    := strconv.ParseFloat(tpS, 64)

	// Use bid for SELL, ask for BUY
	price := req.BidPrice
	if strings.ToUpper(direction) == "BUY" {
		price = req.AskPrice
	}

	newStatus := ""
	now := time.Now()

	if status == "PENDING" {
		// Trigger when price TOUCHES entry from any direction
		// BUY uses ASK, SELL uses BID (already set above)
		// Entry touched = price has reached entry level
		if direction == "BUY"  && price >= entry { newStatus = "RUNNING" } // Buy Stop
		if direction == "BUY"  && price <= entry { newStatus = "RUNNING" } // Buy Limit
		if direction == "SELL" && price <= entry { newStatus = "RUNNING" } // Sell Stop
		if direction == "SELL" && price >= entry { newStatus = "RUNNING" } // Sell Limit
	}

	if status == "RUNNING" || newStatus == "RUNNING" {
		// Check TP/SL hit
		if direction == "BUY" {
			if price >= tp {
				newStatus = "CLOSED_TP"
			} else if price <= sl {
				newStatus = "CLOSED_SL"
			}
		} else { // SELL
			if price <= tp {
				newStatus = "CLOSED_TP"
			} else if price >= sl {
				newStatus = "CLOSED_SL"
			}
		}
	}

	if newStatus == "" {
		c.JSON(200, gin.H{"ok": true, "action": "no_change", "signal_id": id, "status": status})
		return
	}

	// Update DB
	var dbErr error
	if newStatus == "RUNNING" {
		_, dbErr = h.DB.Exec(`
			UPDATE analyst_signals 
			SET status='RUNNING', running_at=$1, updated_at=now()
			WHERE id=$2`, now, id)
	} else {
		// CLOSED_TP or CLOSED_SL
		rr := calcRR(entryS, slS, tpS, direction)
		_ = rr
		_, dbErr = h.DB.Exec(`
			UPDATE analyst_signals 
			SET status=$1, closed_at=$2, updated_at=now()
			WHERE id=$3`, newStatus, now, id)
	}
	if dbErr == nil && (newStatus == "CLOSED_TP" || newStatus == "CLOSED_SL") {
		var setId string
		h.DB.QueryRow(`SELECT COALESCE(set_id,'') FROM analyst_signals WHERE id=$1`, id).Scan(&setId)
		if setId != "" { go h.RecalcAndSaveAlphaRank(setId) }
		go h.DB.Exec(`UPDATE investor_signal_orders SET status=$1, closed_at=now() WHERE signal_id=$2 AND status='OPENED'`, newStatus, id)
	
	}

	if dbErr != nil {
		c.JSON(500, gin.H{"ok": false, "error": fmt.Sprintf("db update failed: %v", dbErr)})
		return
	}

	c.JSON(200, gin.H{
		"ok":        true,
		"action":    "updated",
		"signal_id": id,
		"old_status": status,
		"new_status": newStatus,
		"price":     math.Round(price*100000) / 100000,
		"timestamp": now.Format(time.RFC3339),
	})
}

// ── POST /api/ea/analyst/batch-update ────────────────────────────────────────
// EA sends all current prices at once — more efficient than per-signal
type EAPriceTick struct {
	Pair     string  `json:"pair"`
	Bid      float64 `json:"bid"`
	Ask      float64 `json:"ask"`
}

type EABatchReq struct {
	Prices []EAPriceTick `json:"prices"`
}

func (h *Handler) EABatchUpdate(c *gin.Context) {
	var req EABatchReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"ok": false, "error": "invalid json"})
		return
	}
	if len(req.Prices) == 0 {
		c.JSON(400, gin.H{"ok": false, "error": "no prices provided"})
		return
	}

	// Build price map + update cache
	priceMap := make(map[string]EAPriceTick)
	for _, p := range req.Prices {
		pair := strings.ToUpper(p.Pair)
		priceMap[pair] = p
		// Upsert price cache
		h.DB.Exec(`INSERT INTO ea_price_cache (pair, bid, ask, updated_at)
			VALUES ($1,$2,$3,now())
			ON CONFLICT (pair) DO UPDATE SET bid=$2, ask=$3, updated_at=now()`,
			pair, p.Bid, p.Ask)
	}

	// Fetch all active signals
	rows, err := h.DB.Query(`
		SELECT id, pair, direction, entry, sl, tp, status, COALESCE(market_price_at_creation,'0')
		FROM analyst_signals
		WHERE status IN ('PENDING','RUNNING')`)
	if err != nil {
		c.JSON(500, gin.H{"ok": false, "error": "db error"})
		return
	}
	defer rows.Close()

	type UpdateResult struct {
		SignalID  int64  `json:"signal_id"`
		OldStatus string `json:"old_status"`
		NewStatus string `json:"new_status"`
	}
	updates := make([]UpdateResult, 0)
	now := time.Now()

	for rows.Next() {
		var id int64
		var pair, direction, entryS, slS, tpS, status, mktS string
		rows.Scan(&id, &pair, &direction, &entryS, &slS, &tpS, &status, &mktS)
		mktPrice, _ := strconv.ParseFloat(mktS, 64)

		tick, ok := priceMap[strings.ToUpper(pair)]
		if !ok { continue } // No price for this pair

		entry, _ := strconv.ParseFloat(entryS, 64)
		sl, _    := strconv.ParseFloat(slS, 64)
		tp, _    := strconv.ParseFloat(tpS, 64)

		price := tick.Bid
		if strings.ToUpper(direction) == "BUY" {
			price = tick.Ask
		}

		newStatus := ""

		if status == "PENDING" {
			if mktPrice <= 0 {
				// No market price stored — touch from any direction
				if direction == "BUY"  { newStatus = "RUNNING" }
				if direction == "SELL" { newStatus = "RUNNING" }
			} else if direction == "BUY" {
				if entry <= mktPrice && price <= entry { newStatus = "RUNNING" } // Buy Limit: entry below market, wait for price to drop
				if entry >= mktPrice && price >= entry { newStatus = "RUNNING" } // Buy Stop: entry above market, wait for price to rise
			} else { // SELL
				if entry >= mktPrice && price >= entry { newStatus = "RUNNING" } // Sell Limit: entry above market, wait for price to rise
				if entry <= mktPrice && price <= entry { newStatus = "RUNNING" } // Sell Stop: entry below market, wait for price to drop
			}
		}

		checkPrice := price
		checkStatus := status
		if newStatus == "RUNNING" { checkStatus = "RUNNING" }

		if checkStatus == "RUNNING" {
			if direction == "BUY" {
				if checkPrice >= tp { newStatus = "CLOSED_TP" } else if checkPrice <= sl { newStatus = "CLOSED_SL" }
			} else {
				if checkPrice <= tp { newStatus = "CLOSED_TP" } else if checkPrice >= sl { newStatus = "CLOSED_SL" }
			}
		}

		if newStatus == "" { continue }

		var dbErr error
		if newStatus == "RUNNING" {
			_, dbErr = h.DB.Exec(`UPDATE analyst_signals SET status='RUNNING', running_at=$1, updated_at=now() WHERE id=$2`, now, id)
		} else {
			_, dbErr = h.DB.Exec(`UPDATE analyst_signals SET status=$1, closed_at=$2, updated_at=now() WHERE id=$3`, newStatus, now, id)
		}
		if dbErr == nil {
			updates = append(updates, UpdateResult{SignalID: id, OldStatus: status, NewStatus: newStatus})
		}
	}

	c.JSON(200, gin.H{
		"ok":      true,
		"updated": len(updates),
		"changes": updates,
	})
}
