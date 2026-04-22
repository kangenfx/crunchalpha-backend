package analyst

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	DB *sql.DB
}

type SignalSetDTO struct {
	ID            string  `json:"id"`
	Name          string  `json:"name"`
	Subscribers   int     `json:"subscribers"`
	AutoFollowers int     `json:"autoFollowers"`
	WinRate       float64 `json:"winRate"`
	PF            float64 `json:"profitFactor"`
	Status        string  `json:"status"`
}

type SignalRow struct {
	ID        int64   `json:"id"`
	SetID     string  `json:"set_id"`
	SetName   string  `json:"set_name"`
	Pair      string  `json:"pair"`
	Direction string  `json:"direction"`
	Entry     string  `json:"entry"`
	SL        string  `json:"sl"`
	TP        string  `json:"tp"`
	Status    string  `json:"status"`
	IssuedAt  string  `json:"issuedAt"`
	ClosedAt  string  `json:"closedAt"`
	Analyst   string  `json:"analyst"`
	Notes     string  `json:"notes"`
	RR        float64 `json:"rr"`
	Result    string  `json:"result"`
}

type AnalystFlag struct {
	Code     string  `json:"code"`
	Severity string  `json:"severity"`
	Title    string  `json:"title"`
	Desc     string  `json:"desc"`
	Penalty  float64 `json:"penalty"`
}

func mustAnalyst(c *gin.Context) (string, bool) {
	uid, _ := c.Get("user_id")
	role, _ := c.Get("user_role")
	uidStr, _ := uid.(string)
	roleStr, _ := role.(string)
	if uidStr == "" || strings.ToUpper(roleStr) != "ANALYST" {
		c.JSON(http.StatusUnauthorized, gin.H{"ok": false, "error": "unauthorized"})
		return "", false
	}
	return uidStr, true
}

func genSetID(name string) string {
	s := strings.ToUpper(strings.TrimSpace(name))
	s = strings.ReplaceAll(s, " ", "-")
	s = strings.ReplaceAll(s, "_", "-")
	if s == "" { s = "SET" }
	if len(s) > 24 { s = s[:24] }
	return "SET-" + time.Now().UTC().Format("20060102-150405") + "-" + s
}

func calcRR(entry, sl, tp, direction string) float64 {
	e, err1 := strconv.ParseFloat(entry, 64)
	s, err2 := strconv.ParseFloat(sl, 64)
	t, err3 := strconv.ParseFloat(tp, 64)
	if err1 != nil || err2 != nil || err3 != nil { return 0 }
	var risk, reward float64
	if strings.ToUpper(direction) == "BUY" {
		risk = e - s; reward = t - e
	} else {
		risk = s - e; reward = e - t
	}
	if risk <= 0 { return 0 }
	return math.Round(reward/risk*100) / 100
}

func calcGrade(score float64) string {
	switch {
	case score >= 90: return "A+"
	case score >= 85: return "A"
	case score >= 80: return "A-"
	case score >= 75: return "B+"
	case score >= 70: return "B"
	case score >= 65: return "B-"
	case score >= 60: return "C+"
	case score >= 55: return "C"
	case score >= 50: return "C-"
	case score >= 40: return "D"
	default:          return "F"
	}
}

func (h *Handler) Dashboard(c *gin.Context) {
	uid, ok := mustAnalyst(c)
	if !ok { return }
	var totalSubs, totalAuto int
	h.DB.QueryRow(`SELECT COUNT(*), COUNT(*) FILTER (WHERE execution_mode='AUTO')
		FROM analyst_subscriptions WHERE analyst_id=$1 AND status='ACTIVE'`, uid).Scan(&totalSubs, &totalAuto)
	rows, err := h.DB.Query(`SELECT s.id, s.name, s.status,
		COALESCE(s.win_rate,0), COALESCE(s.profit_factor,0),
		COALESCE(s.subscribers,0),
		COUNT(sub.id) FILTER (WHERE sub.execution_mode='AUTO' AND sub.status='ACTIVE')
		FROM analyst_signal_sets s
		LEFT JOIN analyst_subscriptions sub ON sub.set_id=s.id AND sub.status='ACTIVE'
		WHERE s.analyst_id=$1
		GROUP BY s.id, s.name, s.status, s.win_rate, s.profit_factor, s.subscribers
		ORDER BY s.created_at DESC LIMIT 200`, uid)
	if err != nil { c.JSON(500, gin.H{"ok": false, "error": "db query failed"}); return }
	defer rows.Close()
	sets := make([]SignalSetDTO, 0)
	for rows.Next() {
		var s SignalSetDTO
		if err := rows.Scan(&s.ID, &s.Name, &s.Status, &s.WinRate, &s.PF, &s.Subscribers, &s.AutoFollowers); err != nil {
			c.JSON(500, gin.H{"ok": false, "error": "db scan failed"}); return
		}
		sets = append(sets, s)
	}
	c.JSON(200, gin.H{"ok": true, "signalSets": sets,
		"summary": gin.H{"totalSignalPaket": len(sets), "subscribers": totalSubs, "autoFollowers": totalAuto}})
}

func (h *Handler) ListSignalSets(c *gin.Context) {
	uid, ok := mustAnalyst(c)
	if !ok { return }
	rows, err := h.DB.Query(`SELECT id, name, status,
		COALESCE(win_rate,0), COALESCE(profit_factor,0)
		FROM analyst_signal_sets
		WHERE analyst_id=$1 ORDER BY created_at DESC LIMIT 200`, uid)
	if err != nil { c.JSON(500, gin.H{"ok": false, "error": "db query failed"}); return }
	defer rows.Close()
	out := make([]SignalSetDTO, 0)
	for rows.Next() {
		var s SignalSetDTO
		if err := rows.Scan(&s.ID, &s.Name, &s.Status, &s.WinRate, &s.PF); err != nil {
			c.JSON(500, gin.H{"ok": false, "error": "db scan failed"}); return
		}
		out = append(out, s)
	}
	c.JSON(200, gin.H{"ok": true, "signalSets": out})
}

func (h *Handler) CreateSignalSet(c *gin.Context) {
	uid, ok := mustAnalyst(c)
	if !ok { return }
	var count int
	h.DB.QueryRow(`SELECT COUNT(*) FROM analyst_signal_sets WHERE analyst_id=$1`, uid).Scan(&count)
	if count >= 2 { c.JSON(400, gin.H{"ok": false, "error": "max 2 signal sets per analyst"}); return }
	var req struct{ Name string `json:"name"`; Description string `json:"description"` }
	if err := c.ShouldBindJSON(&req); err != nil { c.JSON(400, gin.H{"ok": false, "error": "invalid json"}); return }
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" { c.JSON(400, gin.H{"ok": false, "error": "name required"}); return }
	id := genSetID(req.Name)
	var s SignalSetDTO
	err := h.DB.QueryRow(`INSERT INTO analyst_signal_sets (id, analyst_id, name, description, status)
		VALUES ($1, $2, $3, $4, 'Active') RETURNING id, name, status`, id, uid, req.Name, strings.TrimSpace(req.Description)).Scan(&s.ID, &s.Name, &s.Status)
	if err != nil { c.JSON(500, gin.H{"ok": false, "error": "db insert failed"}); return }
	c.JSON(200, gin.H{"ok": true, "signalSet": s})
}

func (h *Handler) UpdateSignalSet(c *gin.Context) {
	uid, ok := mustAnalyst(c)
	if !ok { return }
	setID := c.Param("id")
	var req struct{ Name string `json:"name"`; Description string `json:"description"` }
	if err := c.ShouldBindJSON(&req); err != nil { c.JSON(400, gin.H{"ok": false, "error": "invalid json"}); return }
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" { c.JSON(400, gin.H{"ok": false, "error": "name required"}); return }
	res, err := h.DB.Exec(`UPDATE analyst_signal_sets SET name=$1, description=$2, updated_at=now()
		WHERE id=$3 AND analyst_id=$4`, req.Name, strings.TrimSpace(req.Description), setID, uid)
	if err != nil { c.JSON(500, gin.H{"ok": false, "error": "db update failed"}); return }
	aff, _ := res.RowsAffected()
	if aff == 0 { c.JSON(404, gin.H{"ok": false, "error": "signal set not found"}); return }
	c.JSON(200, gin.H{"ok": true, "id": setID, "name": req.Name, "description": req.Description})
}

func (h *Handler) ListSignals(c *gin.Context) {
	uid, ok := mustAnalyst(c)
	if !ok { return }
	rows, err := h.DB.Query(`
SELECT s.id, COALESCE(s.set_id,''), COALESCE(ss.name,''), s.pair, s.direction,
  s.entry, s.sl, s.tp, s.status, COALESCE(s.issued_at,''), COALESCE(s.closed_at::text,''), COALESCE(s.analyst_name,''),
  COALESCE(s.notes,'')
FROM analyst_signals s
LEFT JOIN analyst_signal_sets ss ON ss.id = s.set_id
WHERE s.analyst_id=$1 ORDER BY s.issued_at DESC NULLS LAST, s.id DESC LIMIT 500`, uid)
	if err != nil { c.JSON(500, gin.H{"ok": false, "error": "db query failed"}); return }
	defer rows.Close()
	out := make([]SignalRow, 0)
	for rows.Next() {
		var s SignalRow
		if err := rows.Scan(&s.ID, &s.SetID, &s.SetName, &s.Pair, &s.Direction,
			&s.Entry, &s.SL, &s.TP, &s.Status, &s.IssuedAt, &s.ClosedAt, &s.Analyst, &s.Notes); err != nil {
			c.JSON(500, gin.H{"ok": false, "error": "db scan failed"}); return
		}
		s.RR = calcRR(s.Entry, s.SL, s.TP, s.Direction)
		if s.Status == "CLOSED_TP" { s.Result = "WIN" } else if s.Status == "CLOSED_SL" { s.Result = "LOSS" }
		out = append(out, s)
	}
	c.JSON(200, gin.H{"ok": true, "signals": out})
}

func (h *Handler) CreateSignal(c *gin.Context) {
	uid, ok := mustAnalyst(c)
	if !ok { return }
	var req struct {
		Pair        string `json:"pair"`
		Direction   string `json:"direction"`
		Entry       string `json:"entry"`
		SL          string `json:"sl"`
		TP          string `json:"tp"`
		SetID       string `json:"setId"`
		AnalystName string `json:"analystName"`
		Notes       string `json:"notes"`
		MarketPrice string `json:"marketPrice"`
	}
	if err := c.ShouldBindJSON(&req); err != nil { c.JSON(400, gin.H{"ok": false, "error": "invalid json"}); return }
	req.Pair = strings.ToUpper(strings.TrimSpace(req.Pair))
	req.Direction = strings.ToUpper(strings.TrimSpace(req.Direction))
	req.Entry = strings.TrimSpace(req.Entry)
	req.SL = strings.TrimSpace(req.SL)
	req.TP = strings.TrimSpace(req.TP)
	req.SetID = strings.TrimSpace(req.SetID)
	if req.Pair == "" || (req.Direction != "BUY" && req.Direction != "SELL") ||
		req.Entry == "" || req.SL == "" || req.TP == "" {
		c.JSON(400, gin.H{"ok": false, "error": "pair/direction/entry/sl/tp required"}); return
	}
	if req.SetID == "" { c.JSON(400, gin.H{"ok": false, "error": "setId required"}); return }

	// Hard floor R:R >= 0.5
	rr := calcRR(req.Entry, req.SL, req.TP, req.Direction)
	if rr < 0.5 {
		c.JSON(400, gin.H{"ok": false, "error": fmt.Sprintf("R:R %.2f below minimum 0.5 — rejected", rr)}); return
	}

	// Spam: max 10 signals/24h
	var recentCount int
	h.DB.QueryRow(`SELECT COUNT(*) FROM analyst_signals
		WHERE analyst_id=$1 AND created_at > NOW() - INTERVAL '24 hours'`, uid).Scan(&recentCount)
	if recentCount >= 10 { c.JSON(400, gin.H{"ok": false, "error": "max 10 signals per 24 hours"}); return }

	var setOwner string
	err := h.DB.QueryRow(`SELECT analyst_id FROM analyst_signal_sets WHERE id=$1`, req.SetID).Scan(&setOwner)
	if err != nil || setOwner != uid { c.JSON(400, gin.H{"ok": false, "error": "signal set not found or unauthorized"}); return }

	analystName := strings.TrimSpace(req.AnalystName)
	if analystName == "" { analystName = "Analyst" }
	issuedAt := time.Now().Format("2006-01-02 15:04")
	var id int64
	// Auto-lookup market price from EA cache
	marketPrice := strings.TrimSpace(req.MarketPrice)
	if marketPrice == "" {
		var bid, ask float64
		err2 := h.DB.QueryRow(`SELECT bid, ask FROM ea_price_cache WHERE pair=$1`,
			req.Pair).Scan(&bid, &ask)
		if err2 == nil {
			// Use mid price
			mid := (bid + ask) / 2.0
			marketPrice = strconv.FormatFloat(mid, 'f', 5, 64)
		} else {
			marketPrice = "0"
		}
	}
	err = h.DB.QueryRow(`INSERT INTO analyst_signals
		(analyst_id, set_id, pair, direction, entry, sl, tp, status, issued_at, analyst_name, notes, market_price_at_creation)
		VALUES ($1,$2,$3,$4,$5,$6,$7,'PENDING',$8,$9,$10,$11) RETURNING id`,
		uid, req.SetID, req.Pair, req.Direction, req.Entry, req.SL, req.TP,
		issuedAt, analystName, strings.TrimSpace(req.Notes), marketPrice).Scan(&id)
	if err != nil { c.JSON(500, gin.H{"ok": false, "error": fmt.Sprintf("db insert failed: %v", err)}); return }
	c.JSON(200, gin.H{"ok": true, "signal": SignalRow{
		ID: id, SetID: req.SetID, Pair: req.Pair, Direction: req.Direction,
		Entry: req.Entry, SL: req.SL, TP: req.TP,
		Status: "PENDING", IssuedAt: issuedAt, Analyst: analystName, Notes: req.Notes, RR: rr,
	}})
}

func (h *Handler) UpdateSignal(c *gin.Context) {
	uid, ok := mustAnalyst(c)
	if !ok { return }
	sigID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || sigID <= 0 { c.JSON(400, gin.H{"ok": false, "error": "invalid signal id"}); return }
	var currentStatus string
	err = h.DB.QueryRow(`SELECT status FROM analyst_signals WHERE id=$1 AND analyst_id=$2`, sigID, uid).Scan(&currentStatus)
	if err != nil { c.JSON(404, gin.H{"ok": false, "error": "signal not found"}); return }
	if currentStatus != "PENDING" && currentStatus != "OPEN" {
		c.JSON(400, gin.H{"ok": false, "error": "only PENDING signals can be edited"}); return
	}
	var req struct {
		Entry string `json:"entry"`; SL string `json:"sl"`; TP string `json:"tp"`; Notes string `json:"notes"`
	}
	if err := c.ShouldBindJSON(&req); err != nil { c.JSON(400, gin.H{"ok": false, "error": "invalid json"}); return }
	req.Entry = strings.TrimSpace(req.Entry); req.SL = strings.TrimSpace(req.SL); req.TP = strings.TrimSpace(req.TP)
	if req.Entry == "" || req.SL == "" || req.TP == "" { c.JSON(400, gin.H{"ok": false, "error": "entry/sl/tp required"}); return }
	_, err = h.DB.Exec(`UPDATE analyst_signals SET entry=$1, sl=$2, tp=$3, notes=$4, updated_at=now()
		WHERE id=$5 AND analyst_id=$6`, req.Entry, req.SL, req.TP, req.Notes, sigID, uid)
	if err != nil { c.JSON(500, gin.H{"ok": false, "error": "db update failed"}); return }
	c.JSON(200, gin.H{"ok": true, "updated": sigID})
}

func (h *Handler) CancelSignal(c *gin.Context) {
	uid, ok := mustAnalyst(c)
	if !ok { return }
	sigID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || sigID <= 0 { c.JSON(400, gin.H{"ok": false, "error": "invalid signal id"}); return }
	res, err := h.DB.Exec(`UPDATE analyst_signals SET status='CANCELLED_MANUAL', updated_at=now()
		WHERE id=$1 AND analyst_id=$2 AND status IN ('PENDING','OPEN')`, sigID, uid)
	if err != nil { c.JSON(500, gin.H{"ok": false, "error": "db update failed"}); return }
	aff, _ := res.RowsAffected()
	if aff == 0 { c.JSON(400, gin.H{"ok": false, "error": "signal not found or not cancellable"}); return }
	c.JSON(200, gin.H{"ok": true, "updated": aff})
}

func (h *Handler) Performance(c *gin.Context) {
	uid, ok := mustAnalyst(c)
	if !ok { return }
	var totalSignals, wins, losses int
	var totalRR, grossProfit, grossLoss float64
	rows, err := h.DB.Query(`SELECT status, entry, sl, tp, direction FROM analyst_signals
		WHERE analyst_id=$1 AND status IN ('CLOSED_TP','CLOSED_SL')`, uid)
	if err != nil { c.JSON(500, gin.H{"ok": false, "error": "db query failed"}); return }
	defer rows.Close()
	for rows.Next() {
		var status, entry, sl, tp, direction string
		rows.Scan(&status, &entry, &sl, &tp, &direction)
		totalSignals++
		rr := calcRR(entry, sl, tp, direction)
		totalRR += rr
		if status == "CLOSED_TP" { wins++; grossProfit += rr } else { losses++; grossLoss += 1.0 }
	}
	winRate := 0.0
	if totalSignals > 0 { winRate = math.Round(float64(wins)/float64(totalSignals)*10000) / 100 }
	avgRR := 0.0
	if totalSignals > 0 { avgRR = math.Round(totalRR/float64(totalSignals)*100) / 100 }
	pf := 0.0
	if grossLoss > 0 { pf = math.Round(grossProfit/grossLoss*100) / 100 }
	var pendingCount, runningCount int
	h.DB.QueryRow(`SELECT COUNT(*) FILTER (WHERE status='PENDING'), COUNT(*) FILTER (WHERE status='RUNNING')
		FROM analyst_signals WHERE analyst_id=$1`, uid).Scan(&pendingCount, &runningCount)
	c.JSON(200, gin.H{"ok": true, "performance": gin.H{
		"totalSignals": totalSignals, "wins": wins, "losses": losses,
		"winRate": winRate, "profitFactor": pf, "avgRR": avgRR,
		"pendingSignals": pendingCount, "runningSignals": runningCount,
	}})
}

func (h *Handler) AnalystAlphaRank(c *gin.Context) {
	uid, ok := mustAnalyst(c)
	if !ok { return }

	setId := strings.TrimSpace(c.Query("setId"))

	// Verify ownership if setId provided
	if setId != "" {
		var owner string
		if err := h.DB.QueryRow(`SELECT analyst_id FROM analyst_signal_sets WHERE id=$1`, setId).Scan(&owner); err != nil || owner != uid {
			c.JSON(403, gin.H{"ok": false, "error": "signal set not found or unauthorized"})
			return
		}
	} else {
		// Get first signal set for this analyst
		if err := h.DB.QueryRow(`SELECT id FROM analyst_signal_sets WHERE analyst_id=$1 ORDER BY created_at DESC LIMIT 1`, uid).Scan(&setId); err != nil {
			c.JSON(200, gin.H{"ok": true, "alphaRank": gin.H{
				"alphaScore":0,"grade":"D","totalSignals":0,"closedSignals":0,
				"wins":0,"losses":0,"winRate":0,"avgRR":0,"profitFactor":0,
				"totalPipsWin":0,"totalPipsLoss":0,"maxConsecLoss":0,
				"pendingSignals":0,"runningSignals":0,
				"pillars":[]gin.H{},"flags":[]gin.H{},
			}})
			return
		}
	}

	// All data from DB
	var (
		alphaScore, winRate, profitFactor, avgRR, cumulativeR float64
		netPips, totalPipsWin, totalPipsLoss, avgTP, avgSL float64
		avgSignalMonth, avgSignalWeek float64
		totalSignals, closedSignals, wins, losses int
		pendingSignals, runningSignals, daysActive, maxConsecLoss int
		p1Score, p2Score, p3Score, p4Score float64
		p5Score, p6Score, p7Score float64
		p1Reason, p2Reason, p3Reason, p4Reason string
		p5Reason, p6Reason, p7Reason string
		flagsJSON, alphaGrade string
	)

	err := h.DB.QueryRow(`
		SELECT COALESCE(alpha_score,0), COALESCE(alpha_grade,'D'),
		       COALESCE(win_rate,0), COALESCE(profit_factor,0), COALESCE(avg_rr,0), COALESCE(cumulative_r,0),
		       COALESCE(net_pips,0), COALESCE(total_pips_win,0), COALESCE(total_pips_loss,0),
		       COALESCE(avg_tp,0), COALESCE(avg_sl,0),
		       COALESCE(avg_signal_month,0), COALESCE(avg_signal_week,0),
		       COALESCE(total_signals,0), COALESCE(closed_signals,0),
		       COALESCE(winning_signals,0), COALESCE(losses,0),
		       COALESCE(pending_signals,0), COALESCE(running_signals,0),
		       COALESCE(days_active,1), COALESCE(max_consec_loss,0),
		       COALESCE(p1_score,0), COALESCE(p2_score,0), COALESCE(p3_score,0), COALESCE(p4_score,0),
		       COALESCE(p5_score,0), COALESCE(p6_score,0), COALESCE(p7_score,0),
		       COALESCE(p1_reason,''), COALESCE(p2_reason,''), COALESCE(p3_reason,''), COALESCE(p4_reason,''),
		       COALESCE(p5_reason,''), COALESCE(p6_reason,''), COALESCE(p7_reason,''),
		       COALESCE(flags_json,'[]')
		FROM analyst_signal_sets WHERE id=$1
	`, setId).Scan(
		&alphaScore, &alphaGrade,
		&winRate, &profitFactor, &avgRR, &cumulativeR,
		&netPips, &totalPipsWin, &totalPipsLoss, &avgTP, &avgSL,
		&avgSignalMonth, &avgSignalWeek,
		&totalSignals, &closedSignals, &wins, &losses,
		&pendingSignals, &runningSignals, &daysActive, &maxConsecLoss,
		&p1Score, &p2Score, &p3Score, &p4Score,
		&p5Score, &p6Score, &p7Score,
		&p1Reason, &p2Reason, &p3Reason, &p4Reason,
		&p5Reason, &p6Reason, &p7Reason,
		&flagsJSON,
	)
	if err != nil {
		c.JSON(500, gin.H{"ok": false, "error": "db error"})
		return
	}

	var flags []map[string]interface{}
	json.Unmarshal([]byte(flagsJSON), &flags)
	if flags == nil { flags = []map[string]interface{}{} }

	_ = netPips; _ = avgTP; _ = avgSL; _ = avgSignalMonth; _ = avgSignalWeek; _ = daysActive; _ = cumulativeR

	c.JSON(200, gin.H{
		"ok": true,
		"alphaRank": gin.H{
			"setId":          setId,
			"alphaScore":     alphaScore,
			"grade":          alphaGrade,
			"totalSignals":   totalSignals,
			"closedSignals":  closedSignals,
			"wins":           wins,
			"losses":         losses,
			"winRate":        winRate,
			"avgRR":          avgRR,
			"profitFactor":   profitFactor,
			"totalPipsWin":   totalPipsWin,
			"totalPipsLoss":  totalPipsLoss,
			"maxConsecLoss":  maxConsecLoss,
			"pendingSignals": pendingSignals,
			"runningSignals": runningSignals,
			"pillars": []gin.H{
				{"code":"P1","name":"Profitability","weight":15,"score":p1Score,"reason":p1Reason},
				{"code":"P2","name":"Consistency","weight":15,"score":p2Score,"reason":p2Reason},
				{"code":"P3","name":"Risk Management","weight":15,"score":p3Score,"reason":p3Reason},
				{"code":"P4","name":"Recovery","weight":15,"score":p4Score,"reason":p4Reason},
				{"code":"P5","name":"Trading Edge","weight":15,"score":p5Score,"reason":p5Reason},
				{"code":"P6","name":"Discipline","weight":10,"score":p6Score,"reason":p6Reason},
				{"code":"P7","name":"Track Record","weight":10,"score":p7Score,"reason":p7Reason},
			},
			"flags":     flags,
			"flagCount": len(flags),
		},
	})
}

func (h *Handler) PublicMarketplace(c *gin.Context) {
	rows, err := h.DB.Query(`
		SELECT 
			s.id, s.name, COALESCE(s.market,'') || ' ' || COALESCE(s.style,'') as description, s.analyst_id,
			COALESCE(NULLIF(u.name,''), split_part(u.email,'@',1)) as analyst_name,
			COUNT(DISTINCT sub.investor_id) FILTER (WHERE sub.status='ACTIVE') as subscribers,
			COUNT(sig.id) FILTER (WHERE sig.status='CLOSED_TP' OR sig.status='CLOSED_SL') as total_signals,
			COUNT(sig.id) FILTER (WHERE sig.status='CLOSED_TP') as winning_signals,
			COALESCE(AVG(CASE WHEN sig.status='CLOSED_TP' AND sig.tp::numeric>0 AND sig.entry::numeric>0
				THEN ABS(sig.tp::numeric - sig.entry::numeric) END), 0) as avg_tp_dist,
			COALESCE(AVG(CASE WHEN sig.status='CLOSED_SL' AND sig.sl::numeric>0 AND sig.entry::numeric>0
				THEN ABS(sig.entry::numeric - sig.sl::numeric) END), 0) as avg_sl_dist,
			COALESCE(s.avg_rr, 0) as avg_rr,
			COALESCE(s.cumulative_r, 0) as cumulative_r,
			COALESCE(s.net_pips, 0) as net_pips,
			s.created_at,
			COALESCE(s.alpha_score, 0) as alpha_score,
			COALESCE(s.alpha_grade, 'D') as alpha_grade
		FROM analyst_signal_sets s
		LEFT JOIN users u ON u.id::text = s.analyst_id
		LEFT JOIN analyst_subscriptions sub ON sub.set_id = s.id
		LEFT JOIN analyst_signals sig ON sig.set_id = s.id
		WHERE UPPER(s.status) = 'ACTIVE' AND COALESCE(s.closed_signals, 0) >= 20
		GROUP BY s.id, s.name, COALESCE(s.market,'') || ' ' || COALESCE(s.style,''), s.analyst_id, u.name, u.email, s.created_at, s.alpha_score, s.alpha_grade
		ORDER BY s.alpha_score DESC, subscribers DESC
	`)
	if err != nil { c.JSON(500, gin.H{"ok": false, "error": err.Error()}); return }
	defer rows.Close()

	type SetItem struct {
		ID             string  `json:"id"`
		Name           string  `json:"name"`
		Description    string  `json:"description"`
		AnalystID      string  `json:"analystId"`
		AnalystName    string  `json:"analystName"`
		Subscribers    int     `json:"subscribers"`
		TotalSignals   int     `json:"totalSignals"`
		WinningSignals int     `json:"winningSignals"`
		WinRate        float64 `json:"winRate"`
		ProfitFactor   float64 `json:"profitFactor"`
		AlphaScore     float64 `json:"alphaScore"`
		AlphaGrade     string  `json:"alphaGrade"`
		CreatedAt      string  `json:"createdAt"`
		AvgRR          float64 `json:"avgRR"`
		CumulativeR    float64 `json:"cumulativeR"`
		NetPips        float64 `json:"netPips"`
	}

	var items []SetItem
	for rows.Next() {
		var it SetItem
		var avgTpDist, avgSlDist float64
		var analystName sql.NullString
		var createdAtTime sql.NullTime
		var dbAlphaScore float64
		var dbAlphaGrade string
		var avgRR, cumulativeR, netPips float64
		rows.Scan(&it.ID, &it.Name, &it.Description, &it.AnalystID, &analystName,
			&it.Subscribers, &it.TotalSignals, &it.WinningSignals,
			&avgTpDist, &avgSlDist, &avgRR, &cumulativeR, &netPips, &createdAtTime, &dbAlphaScore, &dbAlphaGrade)
		it.AvgRR = avgRR
		it.CumulativeR = cumulativeR
		it.NetPips = netPips
		if analystName.Valid { it.AnalystName = analystName.String }
		if createdAtTime.Valid { it.CreatedAt = createdAtTime.Time.Format("2006-01-02") }
		// Use DB alpha score if available, else recalc
		if dbAlphaScore > 0 {
			it.AlphaScore = dbAlphaScore
			it.AlphaGrade = dbAlphaGrade
		}

		if it.TotalSignals > 0 {
			it.WinRate = float64(it.WinningSignals) / float64(it.TotalSignals) * 100
		}
		if avgSlDist > 0 {
			it.ProfitFactor = avgTpDist / avgSlDist
		}

		// AlphaScore already loaded from DB above (if available)
		if it.AlphaScore == 0 {
			ar := h.calcAlphaRank(it.ID)
			it.AlphaScore = ar.Score
			it.AlphaGrade = ar.Grade
		}
		items = append(items, it)
	}
	if items == nil { items = []SetItem{} }

	// Sort by AlphaScore desc
	for i := 0; i < len(items)-1; i++ {
		for j := i+1; j < len(items); j++ {
			if items[j].AlphaScore > items[i].AlphaScore {
				items[i], items[j] = items[j], items[i]
			}
		}
	}

	c.JSON(200, gin.H{"ok": true, "sets": items, "count": len(items)})
}

// Helper: calculate AlphaRank for any setId (used by marketplace)
type AlphaResult struct { Score float64; Grade string }

func (h *Handler) calcAlphaRank(setId string) AlphaResult {
	rows, err := h.DB.Query(`SELECT direction, entry, sl, tp, status FROM analyst_signals WHERE set_id=$1`, setId)
	if err != nil { return AlphaResult{0, "D"} }
	defer rows.Close()

	var wins, losses, total int
	var grossProfit, grossLoss float64

	for rows.Next() {
		var dir, entry, sl, tp, status string
		rows.Scan(&dir, &entry, &sl, &tp, &status)
		if status != "CLOSED_TP" && status != "CLOSED_SL" { continue }
		total++
		e, _ := strconv.ParseFloat(entry, 64)
		s, _ := strconv.ParseFloat(sl, 64)
		t, _ := strconv.ParseFloat(tp, 64)
		if status == "CLOSED_TP" {
			wins++
			grossProfit += math.Abs(t - e)
		} else {
			losses++
			grossLoss += math.Abs(e - s)
		}
	}

	if total == 0 { return AlphaResult{0, "D"} }

	winRate := float64(wins) / float64(total) * 100
	pf := 0.0
	if grossLoss > 0 { pf = grossProfit / grossLoss }

	// Pillar scores (simplified 4 pillars for marketplace)
	p1 := winRate * 0.35                          // Win Rate 35%
	p2 := math.Min(pf/3.0, 1.0) * 30             // Profit Factor 30%
	p3 := math.Min(float64(total)/10.0, 1.0) * 20 // Volume 20%
	p4 := 0.0
	if pf >= 1.5 && winRate >= 50 { p4 = 15 } else if pf >= 1.0 { p4 = 7 } // Consistency 15%

	score := math.Round((p1+p2+p3+p4)*10) / 10
	if score > 100 { score = 100 }

	grade := "D"
	switch {
	case score >= 90: grade = "A+"
	case score >= 85: grade = "A"
	case score >= 80: grade = "A-"
	case score >= 75: grade = "B+"
	case score >= 70: grade = "B"
	case score >= 65: grade = "B-"
	case score >= 60: grade = "C+"
	case score >= 55: grade = "C"
	}
	return AlphaResult{score, grade}
}

// RecalcAndSaveAlphaRank — identical 7-pillar logic, saves to DB
// Call this whenever a signal status changes to CLOSED_TP or CLOSED_SL
func (h *Handler) RecalcAndSaveAlphaRank(setId string) {
	type sigData struct{ direction,entry,sl,tp,status,pair string; createdAt time.Time }
	rows, err := h.DB.Query(`SELECT direction,entry,sl,tp,status,pair,created_at FROM analyst_signals WHERE set_id=$1 ORDER BY created_at ASC`, setId)
	if err != nil { return }
	defer rows.Close()
	var allSigs []sigData
	for rows.Next() {
		var sd sigData
		rows.Scan(&sd.direction,&sd.entry,&sd.sl,&sd.tp,&sd.status,&sd.pair,&sd.createdAt)
		allSigs = append(allSigs, sd)
	}
	totalSignals := len(allSigs)
	var closedSignals,wins,losses,pendingCount,runningCount int
	var grossProfit,grossLoss,totalClosedRR float64
	var totalPipsWin,totalPipsLoss float64
	var recentCount24h int
	slMap := make(map[string][]float64)
	weeklyMap := make(map[string]int)
	var firstSignalTime,lastSignalTime time.Time
	now := time.Now()
	var maxCL,cur int

	for i,sd := range allSigs {
		rr := calcRR(sd.entry,sd.sl,sd.tp,sd.direction)
		if i==0 { firstSignalTime=sd.createdAt }
		lastSignalTime=sd.createdAt
		y,w := sd.createdAt.ISOWeek()
		weeklyMap[fmt.Sprintf("%d-%d",y,w)]++
		if now.Sub(sd.createdAt)<=24*time.Hour { recentCount24h++ }
		eVal,_:=strconv.ParseFloat(sd.entry,64); slVal,_:=strconv.ParseFloat(sd.sl,64)
		if eVal>0&&slVal>0 { slMap[sd.pair]=append(slMap[sd.pair],math.Abs(eVal-slVal)) }
		switch sd.status {
		case "PENDING","OPEN": pendingCount++
		case "RUNNING": runningCount++
		}
		if sd.status=="CLOSED_TP"||sd.status=="CLOSED_SL" {
			closedSignals++; totalClosedRR+=rr
			e2,_:=strconv.ParseFloat(sd.entry,64)
			tp2,_:=strconv.ParseFloat(sd.tp,64)
			sl2,_:=strconv.ParseFloat(sd.sl,64)
			mult:=10000.0
			if strings.Contains(sd.pair,"JPY")||strings.Contains(sd.pair,"XAU") { mult=100.0 }
			if sd.status=="CLOSED_TP" {
				wins++; grossProfit+=math.Abs(tp2-e2)
				totalPipsWin+=math.Abs(tp2-e2)*mult; cur=0
			} else {
				losses++; grossLoss+=math.Abs(e2-sl2)
				totalPipsLoss+=math.Abs(e2-sl2)*mult
				cur++; if cur>maxCL { maxCL=cur }
			}
		}
	}

	winRate:=0.0; if closedSignals>0 { winRate=float64(wins)/float64(closedSignals)*100 }
	avgClosedRR:=0.0; if closedSignals>0 { avgClosedRR=totalClosedRR/float64(closedSignals) }
	pf:=0.0; if grossLoss>0 { pf=grossProfit/grossLoss }
	netPips:=totalPipsWin-totalPipsLoss
	avgTP:=0.0; if wins>0 { avgTP=totalPipsWin/float64(wins) }
	avgSL:=0.0; if losses>0 { avgSL=totalPipsLoss/float64(losses) }
	daysActive:=1; if !firstSignalTime.IsZero() { daysActive=int(now.Sub(firstSignalTime).Hours()/24)+1 }
	avgSignalMonth:=float64(totalSignals)/(float64(daysActive)/30.0)
	avgSignalWeek:=float64(totalSignals)/(float64(daysActive)/7.0)
	cumulativeR:=totalClosedRR

	// P1
	p1Raw:=winRate/100.0*avgClosedRR; var p1Score float64
	switch { case p1Raw>=1.2:p1Score=100; case p1Raw>=0.9:p1Score=85; case p1Raw>=0.6:p1Score=70; case p1Raw>=0.3:p1Score=45; case closedSignals==0:p1Score=0; default:p1Score=math.Max(0,p1Raw/0.3*45) }
	p1Reason:=fmt.Sprintf("WinRate %.1f%% x AvgRR %.2f = %.2f", winRate, avgClosedRR, p1Raw)
	// P2
	totalWeeks:=1
	var p2Score float64
	if closedSignals>=2 { totalWeeks=int(lastSignalTime.Sub(firstSignalTime).Hours()/168)+1; cr:=float64(len(weeklyMap))/float64(totalWeeks)*100; switch { case cr>=80:p2Score=100; case cr>=60:p2Score=80; case cr>=40:p2Score=60; case cr>=20:p2Score=35; default:p2Score=10 } }
	p2Reason:=fmt.Sprintf("%d active weeks of %d total weeks", len(weeklyMap), totalWeeks)
	// P3 + flags
	type flagItem struct{ Code,Severity,Title,Desc string; Penalty int }
	var flagsList []flagItem
	p3Score:=0.0; if closedSignals>0 { p3Score=100 }
	if closedSignals>=5&&winRate<40 { flagsList=append(flagsList,flagItem{"LOW_ACCURACY","MAJOR","Low Win Rate",fmt.Sprintf("Win rate %.1f%% below 40%% threshold",winRate),15}); p3Score-=15 }
	if closedSignals>=3&&avgClosedRR<0.8 { flagsList=append(flagsList,flagItem{"POOR_RR","MAJOR","Poor Risk-Reward",fmt.Sprintf("Avg R:R 1:%.2f below 0.8 threshold",avgClosedRR),12}); p3Score-=12 }
	if recentCount24h>10 { flagsList=append(flagsList,flagItem{"SPAM_SIGNALS","MINOR","Excessive Signals",fmt.Sprintf("%d signals in last 24h",recentCount24h),8}); p3Score-=8 }
	for pair,slList:=range slMap { if len(slList)>=3 { avg:=0.0; for _,v:=range slList{avg+=v}; avg/=float64(len(slList)); maxDev:=0.0; for _,v:=range slList{if d:=math.Abs(v-avg)/avg*100;d>maxDev{maxDev=d}}; if maxDev>200{flagsList=append(flagsList,flagItem{"INCONSISTENT_SIZING","MINOR","Inconsistent Risk Sizing",fmt.Sprintf("%s SL varies by %.0f%%",pair,maxDev),5}); p3Score-=5; break} } }
	if p3Score<0 { p3Score=0 }
	if closedSignals<5 { flagsList=append(flagsList,flagItem{"INSUFFICIENT_DATA","INFO","Insufficient Data",fmt.Sprintf("Only %d closed signal(s). Need 20 for full score.",closedSignals),0}) } else if closedSignals<20 { flagsList=append(flagsList,flagItem{"LIMITED_SAMPLE","INFO","Limited Sample",fmt.Sprintf("%d closed signals. Need 20 for full score.",closedSignals),0}) }
	p3Reason:=fmt.Sprintf("%d flag(s) detected", len(flagsList))
	// P4
	var p4Score float64
	if closedSignals>0 { switch { case maxCL==0:p4Score=100; case maxCL<=2:p4Score=85; case maxCL<=4:p4Score=65; case maxCL<=6:p4Score=40; default:p4Score=math.Max(0,100-float64(maxCL)*10) } }
	p4Reason:=fmt.Sprintf("Max consecutive losses: %d", maxCL)
	// P5
	var p5Score float64
	switch { case pf>=2.0:p5Score=100; case pf>=1.5:p5Score=85; case pf>=1.2:p5Score=70; case pf>=1.0:p5Score=50; case closedSignals==0:p5Score=0; default:p5Score=math.Max(0,pf/1.0*50) }
	p5Reason:=fmt.Sprintf("Profit factor %.2f", pf)
	// P6
	p6Score:=0.0; if closedSignals>0 { p6Score=math.Round(math.Min(100,avgClosedRR/1.0*100)*10)/10 }
	p6Reason:=fmt.Sprintf("Avg R:R closed signals 1:%.2f", avgClosedRR)
	// P7
	var p7Score float64
	if closedSignals>0 {
		sigScore:=0.0; switch { case closedSignals>=50:sigScore=100; case closedSignals>=20:sigScore=80; case closedSignals>=10:sigScore=60; case closedSignals>=5:sigScore=40; default:sigScore=float64(closedSignals)/5.0*40 }
		dayScore:=0.0; switch { case daysActive>=365:dayScore=100; case daysActive>=180:dayScore=85; case daysActive>=90:dayScore=70; case daysActive>=30:dayScore=50; case daysActive>=7:dayScore=30; default:dayScore=10 }
		p7Score=sigScore*0.5+dayScore*0.5
	}
	p7Reason:=fmt.Sprintf("%d days active, %d signals", daysActive, totalSignals)

	alphaScore:=math.Round((p1Score*0.15+p2Score*0.15+p3Score*0.15+p4Score*0.15+p5Score*0.15+p6Score*0.10+p7Score*0.10+p6Score*0.05)*10)/10
	grade:=calcGrade(alphaScore)

	flagsJSON:="["
	for i,f:=range flagsList { if i>0 { flagsJSON+="," }; flagsJSON+=fmt.Sprintf(`{"code":%q,"severity":%q,"title":%q,"desc":%q,"penalty":%d}`,f.Code,f.Severity,f.Title,f.Desc,f.Penalty) }
	flagsJSON+="]"

	var subscriberCount int
	h.DB.QueryRow(`SELECT COUNT(*) FROM analyst_subscriptions WHERE set_id=$1 AND status='ACTIVE'`, setId).Scan(&subscriberCount)

	h.DB.Exec(`UPDATE analyst_signal_sets SET
		alpha_score=$1, alpha_grade=$2, alpha_updated_at=now(),
		win_rate=$3, total_signals=$4, winning_signals=$5, profit_factor=$6,
		closed_signals=$7, losses=$8, avg_rr=$9, cumulative_r=$10,
		net_pips=$11, total_pips_win=$12, total_pips_loss=$13,
		avg_tp=$14, avg_sl=$15, max_consec_loss=$16,
		avg_signal_month=$17, avg_signal_week=$18,
		pending_signals=$19, running_signals=$20, days_active=$21,
		p1_score=$22, p2_score=$23, p3_score=$24, p4_score=$25,
		p5_score=$26, p6_score=$27, p7_score=$28,
		p1_reason=$29, p2_reason=$30, p3_reason=$31, p4_reason=$32,
		p5_reason=$33, p6_reason=$34, p7_reason=$35,
		flags_json=$36, subscribers=$37
		WHERE id=$38`,
		alphaScore, grade,
		math.Round(winRate*100)/100, totalSignals, wins, math.Round(pf*100)/100,
		closedSignals, losses, math.Round(avgClosedRR*100)/100, math.Round(cumulativeR*100)/100,
		math.Round(netPips*10)/10, math.Round(totalPipsWin*10)/10, math.Round(totalPipsLoss*10)/10,
		math.Round(avgTP*10)/10, math.Round(avgSL*10)/10, maxCL,
		math.Round(avgSignalMonth*10)/10, math.Round(avgSignalWeek*10)/10,
		pendingCount, runningCount, daysActive,
		math.Round(p1Score*10)/10, math.Round(p2Score*10)/10, math.Round(p3Score*10)/10, math.Round(p4Score*10)/10,
		math.Round(p5Score*10)/10, math.Round(p6Score*10)/10, math.Round(p7Score*10)/10,
		p1Reason, p2Reason, p3Reason, p4Reason, p5Reason, p6Reason, p7Reason,
		flagsJSON, subscriberCount,
		setId)
}

// POST /api/admin/recalc-alpharank — one-time recalc all sets
func (h *Handler) AdminRecalcAllAlphaRank(c *gin.Context) {
	rows, err := h.DB.Query(`SELECT id FROM analyst_signal_sets WHERE UPPER(status)='ACTIVE'`)
	if err != nil { c.JSON(500, gin.H{"ok": false, "error": err.Error()}); return }
	defer rows.Close()
	var setIDs []string
	for rows.Next() {
		var id string; rows.Scan(&id)
		setIDs = append(setIDs, id)
	}
	for _, id := range setIDs {
		h.RecalcAndSaveAlphaRank(id)
	}
	c.JSON(200, gin.H{"ok": true, "recalculated": len(setIDs), "setIds": setIDs})
}

// GET /api/public/analyst-profile/:setId — no auth required
// GET /api/public/analyst-profile/:setId — no auth, all data from DB
func (h *Handler) GetPublicAnalystProfile(c *gin.Context) {
	setId := strings.TrimSpace(c.Param("setId"))
	if setId == "" {
		c.JSON(400, gin.H{"ok": false, "error": "setId required"})
		return
	}

	// All data from DB — zero on-the-fly calculation
	var (
		setName, analystName, analystCountry, analystBio, alphaGrade, setStatus string
		alphaScore float64
		setCreatedAt time.Time
		winRate, profitFactor, avgRR, cumulativeR float64
		netPips, totalPipsWin, totalPipsLoss float64
		avgTP, avgSL float64
		avgSignalMonth, avgSignalWeek float64
		totalSignals, closedSignals, wins, losses int
		pendingSignals, runningSignals, daysActive int
		maxConsecLoss, subscribers int
		p1Score, p2Score, p3Score, p4Score float64
		p5Score, p6Score, p7Score float64
		p1Reason, p2Reason, p3Reason, p4Reason string
		p5Reason, p6Reason, p7Reason string
		flagsJSON string
	)

	err := h.DB.QueryRow(`
		SELECT s.name, s.alpha_score, s.alpha_grade, s.status, s.created_at,
		       COALESCE(u.name, u.email, 'Unknown'),
			       COALESCE(u.country,''), COALESCE(u.bio,''),
		       COALESCE(s.win_rate,0), COALESCE(s.profit_factor,0),
		       COALESCE(s.avg_rr,0), COALESCE(s.cumulative_r,0),
		       COALESCE(s.net_pips,0), COALESCE(s.total_pips_win,0), COALESCE(s.total_pips_loss,0),
		       COALESCE(s.avg_tp,0), COALESCE(s.avg_sl,0),
		       COALESCE(s.avg_signal_month,0), COALESCE(s.avg_signal_week,0),
		       COALESCE(s.total_signals,0), COALESCE(s.closed_signals,0),
		       COALESCE(s.winning_signals,0), COALESCE(s.losses,0),
		       COALESCE(s.pending_signals,0), COALESCE(s.running_signals,0),
		       COALESCE(s.days_active,1), COALESCE(s.max_consec_loss,0),
		       COALESCE(s.subscribers,0),
		       COALESCE(s.p1_score,0), COALESCE(s.p2_score,0), COALESCE(s.p3_score,0), COALESCE(s.p4_score,0),
		       COALESCE(s.p5_score,0), COALESCE(s.p6_score,0), COALESCE(s.p7_score,0),
		       COALESCE(s.p1_reason,''), COALESCE(s.p2_reason,''), COALESCE(s.p3_reason,''), COALESCE(s.p4_reason,''),
		       COALESCE(s.p5_reason,''), COALESCE(s.p6_reason,''), COALESCE(s.p7_reason,''),
		       COALESCE(s.flags_json,'[]')
		FROM analyst_signal_sets s
		LEFT JOIN users u ON u.id = s.analyst_id::uuid
		WHERE s.id = $1
	`, setId).Scan(
		&setName, &alphaScore, &alphaGrade, &setStatus, &setCreatedAt,
		&analystName, &analystCountry, &analystBio,
		&winRate, &profitFactor, &avgRR, &cumulativeR,
		&netPips, &totalPipsWin, &totalPipsLoss,
		&avgTP, &avgSL, &avgSignalMonth, &avgSignalWeek,
		&totalSignals, &closedSignals, &wins, &losses,
		&pendingSignals, &runningSignals, &daysActive, &maxConsecLoss,
		&subscribers,
		&p1Score, &p2Score, &p3Score, &p4Score,
		&p5Score, &p6Score, &p7Score,
		&p1Reason, &p2Reason, &p3Reason, &p4Reason,
		&p5Reason, &p6Reason, &p7Reason,
		&flagsJSON,
	)
	if err != nil {
		c.JSON(404, gin.H{"ok": false, "error": "signal set not found"})
		return
	}

	// Parse flags from JSON string
	var flags []map[string]interface{}
	json.Unmarshal([]byte(flagsJSON), &flags)
	if flags == nil { flags = []map[string]interface{}{} }

	// History — closed signals from DB, newest first
	type historyRow struct {
		ID        int64   `json:"id"`
		Pair      string  `json:"pair"`
		Direction string  `json:"direction"`
		Entry     string  `json:"entry"`
		SL        string  `json:"sl"`
		TP        string  `json:"tp"`
		RR        float64 `json:"rr"`
		Pips      float64 `json:"pips"`
		Result    string  `json:"result"`
		IssuedAt  string  `json:"issuedAt"`
		ClosedAt  string  `json:"closedAt"`
	}

	histRows, err := h.DB.Query(`
		SELECT id, pair, direction, entry, sl, tp,
		       COALESCE(issued_at, to_char(created_at, 'YYYY-MM-DD HH24:MI')),
		       status,
		       COALESCE(closed_at::text, '')
		FROM analyst_signals
		WHERE set_id=$1 AND status IN ('CLOSED_TP','CLOSED_SL')
		ORDER BY issued_at DESC NULLS LAST
		LIMIT 100
	`, setId)
	var history []historyRow
	if err == nil {
		defer histRows.Close()
		for histRows.Next() {
			var h2 historyRow
			var status string
			histRows.Scan(&h2.ID, &h2.Pair, &h2.Direction, &h2.Entry, &h2.SL, &h2.TP, &h2.IssuedAt, &status, &h2.ClosedAt)
			rr := calcRR(h2.Entry, h2.SL, h2.TP, h2.Direction)
			h2.RR = math.Round(rr*100) / 100
			e2, _ := strconv.ParseFloat(h2.Entry, 64)
			tp2, _ := strconv.ParseFloat(h2.TP, 64)
			sl2, _ := strconv.ParseFloat(h2.SL, 64)
			mult := 10000.0
			if strings.Contains(h2.Pair, "JPY") || strings.Contains(h2.Pair, "XAU") { mult = 100.0 }
			if status == "CLOSED_TP" {
				h2.Pips = math.Round(math.Abs(tp2-e2)*mult*10) / 10
				h2.Result = "WIN"
			} else {
				h2.Pips = math.Round(-math.Abs(e2-sl2)*mult*10) / 10
				h2.Result = "LOSS"
			}
			history = append(history, h2)
		}
	}
	if history == nil { history = []historyRow{} }

	c.JSON(200, gin.H{
		"ok": true,
		"set": gin.H{
			"id": setId, "name": setName, "analystName": analystName, "analystCountry": analystCountry, "analystBio": analystBio,
			"alphaScore": alphaScore, "alphaGrade": alphaGrade,
			"status": setStatus, "createdAt": setCreatedAt.Format(time.RFC3339),
		},
		"stats": gin.H{
			"totalSignals": totalSignals, "closedSignals": closedSignals,
			"wins": wins, "winRate": winRate,
			"runningSignals": runningSignals, "pendingSignals": pendingSignals,
			"subscribers": subscribers,
		},
		"metrics": gin.H{
			"avgRR": avgRR, "profitFactor": profitFactor,
			"netPips": netPips, "totalPipsWin": totalPipsWin, "totalPipsLoss": totalPipsLoss,
			"cumulativeR": cumulativeR,
			"avgSignalMonth": avgSignalMonth, "avgSignalWeek": avgSignalWeek,
			"avgTP": avgTP, "avgSL": avgSL, "maxConsecLoss": maxConsecLoss,
		},
		"pillars": []gin.H{
			{"code":"P1","name":"Profitability","weight":15,"score":p1Score,"reason":p1Reason},
			{"code":"P2","name":"Consistency","weight":15,"score":p2Score,"reason":p2Reason},
			{"code":"P3","name":"Risk Management","weight":15,"score":p3Score,"reason":p3Reason},
			{"code":"P4","name":"Recovery","weight":15,"score":p4Score,"reason":p4Reason},
			{"code":"P5","name":"Trading Edge","weight":15,"score":p5Score,"reason":p5Reason},
			{"code":"P6","name":"Discipline","weight":10,"score":p6Score,"reason":p6Reason},
			{"code":"P7","name":"Track Record","weight":10,"score":p7Score,"reason":p7Reason},
		},
		"flags":   flags,
		"history": history,
	})
}

// GET /api/analyst/my-subscribers — investors subscribed to analyst's signal sets
func (h *Handler) GetMySubscribers(c *gin.Context) {
	uid, ok := c.Get("user_id")
	if !ok { c.JSON(401, gin.H{"ok":false,"error":"unauthorized"}); return }

	rows, err := h.DB.Query(`
		SELECT
			s.id, s.set_id, ass.name as set_name,
			s.status, s.execution_mode, s.auto_follow,
			s.created_at,
			COALESCE(s.total_signals_during,0),
			COALESCE(s.wins_during,0),
			COALESCE(s.losses_during,0),
			COALESCE(s.expires_at::text,'')
		FROM analyst_subscriptions s
		JOIN analyst_signal_sets ass ON ass.id = s.set_id
		WHERE ass.analyst_id = $1
		ORDER BY s.created_at DESC`, uid)
	if err != nil { c.JSON(500, gin.H{"ok":false,"error":err.Error()}); return }
	defer rows.Close()

	type SubRow struct {
		ID            string `json:"id"`
		SetID         string `json:"setId"`
		SetName       string `json:"setName"`
		Status        string `json:"status"`
		ExecutionMode string `json:"executionMode"`
		AutoFollow    bool   `json:"autoFollow"`
		CreatedAt     string `json:"createdAt"`
		TotalSignals  int    `json:"totalSignals"`
		Wins          int    `json:"wins"`
		Losses        int    `json:"losses"`
		ExpiresAt     string `json:"expiresAt"`
	}

	var subs []SubRow
	for rows.Next() {
		var s SubRow
		rows.Scan(&s.ID, &s.SetID, &s.SetName,
			&s.Status, &s.ExecutionMode, &s.AutoFollow,
			&s.CreatedAt, &s.TotalSignals, &s.Wins, &s.Losses, &s.ExpiresAt)
		subs = append(subs, s)
	}
	if subs == nil { subs = []SubRow{} }

	// Summary per set
	setMap := make(map[string]gin.H)
	for _, s := range subs {
		if _, ok := setMap[s.SetID]; !ok {
			setMap[s.SetID] = gin.H{"setId": s.SetID, "setName": s.SetName, "total": 0, "active": 0, "auto": 0, "manual": 0}
		}
		m := setMap[s.SetID]
		m["total"] = m["total"].(int) + 1
		if s.Status == "ACTIVE" {
			m["active"] = m["active"].(int) + 1
			if s.AutoFollow { m["auto"] = m["auto"].(int) + 1 } else { m["manual"] = m["manual"].(int) + 1 }
		}
		setMap[s.SetID] = m
	}
	var setSummary []gin.H
	for _, v := range setMap { setSummary = append(setSummary, v) }
	if setSummary == nil { setSummary = []gin.H{} }

	c.JSON(200, gin.H{"ok":true, "subscribers":subs, "setSummary":setSummary})
}
