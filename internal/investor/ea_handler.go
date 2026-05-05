package investor

import (
	"math"
	"strconv"
	"strings"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)



// GET /api/ea/investor/pending-signals
// EA polls this to get signals it should execute
func (h *Handler) EAGetPendingSignals(c *gin.Context) {
	// For now use X-Investor-ID header for testing
	// Later replace with EA key auth
	investorID := getEAInvestorID(c)
	if investorID == "" {
		c.JSON(401, gin.H{"ok": false, "error": "missing investor id"})
		return
	}

	rows, err := h.service.repo.DB.Query(`
		SELECT sig.id, sig.pair, sig.direction, sig.entry, sig.sl, sig.tp,
		       sig.status, sig.set_id, ss.name as set_name,
		       COALESCE(sub.allocation_pct, 10) as allocation_pct,
		       COALESCE(isk.equity, 0) as investor_equity,
		       COALESCE(isk.risk_level, 'balanced') as risk_level,
		       COALESCE(sub.follower_account_id::text, '') as follower_account_id
			FROM analyst_signals sig
			JOIN analyst_signal_sets ss ON ss.id = sig.set_id
			JOIN analyst_subscriptions sub ON sub.set_id = sig.set_id
			LEFT JOIN investor_ea_keys isk ON isk.investor_id = sub.investor_id
			  AND isk.trader_account_id = sub.follower_account_id
		WHERE sub.investor_id=$1::uuid
		  AND sub.status='ACTIVE'
		  AND sub.auto_follow=true
		  AND sig.status IN ('RUNNING','CLOSED_TP','CLOSED_SL')
		ORDER BY sig.id ASC`, investorID)
	if err != nil {
		c.JSON(500, gin.H{"ok": false, "error": "db error"})
		return
	}
	defer rows.Close()

	type SigRow struct {
		ID            int64   `json:"id"`
		Pair          string  `json:"pair"`
		Direction     string  `json:"direction"`
		Entry         string  `json:"entry"`
		SL            string  `json:"sl"`
		TP            string  `json:"tp"`
		Status        string  `json:"status"`
		SetID         string  `json:"setId"`
		SetName       string  `json:"setName"`
		AllocationPct float64 `json:"allocationPct"`
		Equity        float64 `json:"equity"`
		RiskLevel     string  `json:"riskLevel"`
		FollowerAccID string  `json:"followerAccountId"`
		CalculatedLot float64 `json:"calculatedLot"`
		OrderStatus   string  `json:"orderStatus"`
		Ticket        int64   `json:"ticket"`
	}

	var sigs []SigRow
	for rows.Next() {
		var s SigRow
		rows.Scan(&s.ID, &s.Pair, &s.Direction, &s.Entry, &s.SL, &s.TP,
			&s.Status, &s.SetID, &s.SetName, &s.AllocationPct, &s.Equity, &s.RiskLevel, &s.FollowerAccID)
		// Calculate lot based on risk
		riskPct := 1.5 // balanced default
		if s.RiskLevel == "conservative" { riskPct = 0.5 }
		if s.RiskLevel == "aggressive"   { riskPct = 3.0 }
		aum := s.Equity * s.AllocationPct / 100.0
		riskAmt := aum * riskPct / 100.0
		entry := 0.0; sl := 0.0
		if s.Entry != "" { entry, _ = strconv.ParseFloat(s.Entry, 64) }
		if s.SL != "" && s.SL != "0" { sl, _ = strconv.ParseFloat(s.SL, 64) }
		if entry > 0 && sl > 0 {
			slDist := entry - sl
			if slDist < 0 { slDist = -slDist }
			// contract size: gold=100, forex=100000, default=100000
			contractSize := 100000.0
			pairUp := strings.ToUpper(s.Pair)
			if strings.Contains(pairUp, "XAU") || strings.Contains(pairUp, "GOLD") { contractSize = 100.0 }
			if slDist > 0 { s.CalculatedLot = math.Round(riskAmt/(slDist*contractSize)*100)/100 }
		}
		if s.CalculatedLot < 0.01 { s.CalculatedLot = 0.01 }
		if s.CalculatedLot > 1.0  { s.CalculatedLot = 1.0 }

		// Check if investor already has order for this signal
		h.service.repo.DB.QueryRow(`
			SELECT COALESCE(status,''), COALESCE(ticket,0)
			FROM investor_signal_orders
			WHERE investor_id=$1::uuid AND signal_id=$2`,
			investorID, s.ID).Scan(&s.OrderStatus, &s.Ticket)

		sigs = append(sigs, s)
	}
	if sigs == nil {
		sigs = []SigRow{}
	}
	c.JSON(200, gin.H{"ok": true, "signals": sigs, "count": len(sigs)})
}

// POST /api/ea/investor/order-update
// EA reports order status back to backend
func (h *Handler) EAOrderUpdate(c *gin.Context) {
	investorID := getEAInvestorID(c)
	if investorID == "" {
		c.JSON(401, gin.H{"ok": false, "error": "missing investor id"})
		return
	}

	var req struct {
		SignalID   int64   `json:"signalId"`
		Ticket     int64   `json:"ticket"`
		Status     string  `json:"status"` // OPENED, CLOSED_TP, CLOSED_SL, CLOSED_MANUAL
		OpenPrice  float64 `json:"openPrice"`
		ClosePrice float64 `json:"closePrice"`
		LotSize    float64 `json:"lotSize"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.SignalID == 0 {
		c.JSON(400, gin.H{"ok": false, "error": "invalid request"})
		return
	}

	// Get set_id for this signal
	var setID string
	h.service.repo.DB.QueryRow(`SELECT COALESCE(set_id,'') FROM analyst_signals WHERE id=$1`, req.SignalID).Scan(&setID)

	now := time.Now()
	var openedAt, closedAt *time.Time
	if req.Status == "OPENED" {
		openedAt = &now
	} else if req.Status == "CLOSED_TP" || req.Status == "CLOSED_SL" || req.Status == "CLOSED_MANUAL" {
		closedAt = &now
	}

	_, err := h.service.repo.DB.Exec(`
		INSERT INTO investor_signal_orders
			(investor_id, signal_id, set_id, ticket, status, open_price, close_price, lot_size, created_at, opened_at, closed_at)
		VALUES ($1::uuid, $2, $3, $4, $5, $6, $7, $8, now(), $9, $10)
		ON CONFLICT (investor_id, signal_id) DO UPDATE
		SET ticket=$4, status=$5, open_price=$6, close_price=$7, lot_size=$8,
		    opened_at=COALESCE(investor_signal_orders.opened_at, $9),
		    closed_at=$10`,
		investorID, req.SignalID, setID, req.Ticket, req.Status,
		req.OpenPrice, req.ClosePrice, req.LotSize, openedAt, closedAt)

	if err != nil {
		c.JSON(500, gin.H{"ok": false, "error": "update failed: "+err.Error()})
		return
	}

	c.JSON(200, gin.H{"ok": true, "message": "order updated"})
}

// GET /api/investor/signal-orders — investor sees their order history
func (h *Handler) GetSignalOrders(c *gin.Context) {
	uid, ok := getUID(c)
	if !ok {
		c.JSON(401, gin.H{"ok": false, "error": "unauthorized"})
		return
	}

	rows, err := h.service.repo.DB.Query(`
		SELECT o.id, o.signal_id, o.set_id, o.ticket, o.status,
		       o.open_price, o.close_price, o.lot_size, o.created_at,
		       sig.pair, sig.direction, sig.entry, sig.sl, sig.tp,
		       ss.name as set_name
		FROM investor_signal_orders o
		JOIN analyst_signals sig ON sig.id = o.signal_id
		WHERE o.investor_id=$1::uuid
		ORDER BY o.id DESC LIMIT 100`, uid)
	if err != nil {
		c.JSON(500, gin.H{"ok": false, "error": "db error"})
		return
	}
	defer rows.Close()

	type OrderRow struct {
		ID         int64   `json:"id"`
		SignalID   int64   `json:"signalId"`
		SetID      string  `json:"setId"`
		SetName    string  `json:"setName"`
		Ticket     int64   `json:"ticket"`
		Status     string  `json:"status"`
		OpenPrice  float64 `json:"openPrice"`
		ClosePrice float64 `json:"closePrice"`
		LotSize    float64 `json:"lotSize"`
		CreatedAt  string  `json:"createdAt"`
		Pair       string  `json:"pair"`
		Direction  string  `json:"direction"`
		Entry      string  `json:"entry"`
		SL         string  `json:"sl"`
		TP         string  `json:"tp"`
	}

	var orders []OrderRow
	for rows.Next() {
		var o OrderRow
		var createdAt time.Time
		rows.Scan(&o.ID, &o.SignalID, &o.SetID, &o.Ticket, &o.Status,
			&o.OpenPrice, &o.ClosePrice, &o.LotSize, &createdAt,
			&o.Pair, &o.Direction, &o.Entry, &o.SL, &o.TP, &o.SetName)
		o.CreatedAt = createdAt.Format("2006-01-02 15:04")
		orders = append(orders, o)
	}
	if orders == nil {
		orders = []OrderRow{}
	}
	c.JSON(200, gin.H{"ok": true, "orders": orders, "count": len(orders)})
}

var _ = http.StatusOK
