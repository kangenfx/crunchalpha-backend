package investor

import (
	"github.com/gin-gonic/gin"
)

// GET /api/ea/investor/pending-copy-trades
// EA investor poll copy trade events yang perlu dieksekusi
func (h *Handler) EAGetPendingCopyTrades(c *gin.Context) {
	investorID := getEAInvestorID(c)
	if investorID == "" {
		c.JSON(401, gin.H{"ok": false, "error": "missing investor id"})
		return
	}
	mt5Account, _ := c.Get("mt5_account")
	mt5Str := ""
	if mt5Account != nil { mt5Str = mt5Account.(string) }
	engine := NewCopyTraderEngine(h.service.repo.DB)
	events, err := engine.GetPendingCopyEvents(investorID, mt5Str)
	if err != nil {
		c.JSON(500, gin.H{"ok": false, "error": "db error"})
		return
	}
	c.JSON(200, gin.H{"ok": true, "events": events, "count": len(events)})
}

// POST /api/ea/investor/copy-trade-update
// EA report hasil eksekusi copy trade
func (h *Handler) EACopyTradeUpdate(c *gin.Context) {
	investorID := getEAInvestorID(c)
	if investorID == "" {
		c.JSON(401, gin.H{"ok": false, "error": "missing investor id"})
		return
	}
	var req struct {
		EventID         string  `json:"eventId"`
		Status          string  `json:"status"`
		RejectionReason string  `json:"rejectionReason"`
		FollowerTicket  int64   `json:"followerTicket"`
		ExecutedLot     float64 `json:"executedLot"`
		ExecutedPrice   float64 `json:"executedPrice"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.EventID == "" {
		c.JSON(400, gin.H{"ok": false, "error": "invalid request"})
		return
	}
	engine := NewCopyTraderEngine(h.service.repo.DB)
	err := engine.UpdateCopyEventStatus(
		req.EventID, req.Status, req.RejectionReason,
		req.FollowerTicket, req.ExecutedLot, req.ExecutedPrice,
	)
	if err != nil {
		c.JSON(500, gin.H{"ok": false, "error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"ok": true, "message": "copy trade updated"})
}

// POST /api/ea/investor/push-equity
// EA investor push equity terkini ke backend (untuk AUM calculation)
func (h *Handler) EAPushEquity(c *gin.Context) {
	investorID := getEAInvestorID(c)
	if investorID == "" {
		c.JSON(401, gin.H{"ok": false, "error": "unauthorized"})
		return
	}
	var req struct {
		Equity        float64 `json:"equity"`
		Balance       float64 `json:"balance"`
		Margin        float64 `json:"margin"`
		FreeMargin    float64 `json:"free_margin"`
		Floating      float64 `json:"floating_profit"`
		OpenLots      float64 `json:"open_lots"`
		OpenPositions int     `json:"open_positions"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.Equity <= 0 {
		c.JSON(400, gin.H{"ok": false, "error": "invalid equity"})
		return
	}
	keyID, _ := c.Get("ea_key_id")
	if keyID != nil && keyID.(string) != "" {
		h.service.repo.DB.Exec(
			`UPDATE investor_ea_keys SET equity=$1, balance=$2, margin=$3, free_margin=$4, floating=$5, open_lots=$6, open_positions=$7, last_equity_at=now() WHERE id=$8`,
			req.Equity, req.Balance, req.Margin, req.FreeMargin, req.Floating, req.OpenLots, req.OpenPositions, keyID)
	}
	h.service.repo.DB.Exec(
		`UPDATE investor_settings SET investor_equity=(SELECT COALESCE(SUM(equity),0) FROM investor_ea_keys WHERE investor_id=$1::uuid), updated_at=now() WHERE investor_id=$1::uuid`,
		investorID)
	c.JSON(200, gin.H{"ok": true, "equity": req.Equity})
}

// GET /api/investor/copy-trade-history
// Investor lihat history copy trade events (executed + rejected)
func (h *Handler) GetCopyTradeHistory(c *gin.Context) {
	uid, ok := getUID(c)
	if !ok {
		c.JSON(401, gin.H{"ok": false, "error": "unauthorized"})
		return
	}
	rows, err := h.service.repo.DB.Query(
		`SELECT
			ce.id, ce.action, ce.symbol, ce.type,
			COALESCE(ce.calculated_lot, ce.lots),
			COALESCE(ce.sl, 0), COALESCE(ce.tp, 0),
			ce.status,
			COALESCE(ce.rejection_reason, ''),
			COALESCE(ce.aum_used, 0),
			COALESCE(ce.investor_equity, 0),
			ce.created_at,
			COALESCE(ce.processed_at, ce.created_at),
			COALESCE(ta.nickname, ta.account_number, '') as trader_name
		 FROM copy_events ce
		 LEFT JOIN trader_accounts ta ON ta.id = ce.provider_account_id
		 WHERE ce.follower_account_id = (
		   SELECT id FROM trader_accounts WHERE user_id=$1::uuid AND status='active' LIMIT 1
		 )
		 ORDER BY ce.created_at DESC LIMIT 100`,
		uid)
	if err != nil {
		c.JSON(500, gin.H{"ok": false, "error": "db error"})
		return
	}
	defer rows.Close()

	type Row struct {
		ID              string  `json:"id"`
		Action          string  `json:"action"`
		Symbol          string  `json:"symbol"`
		Direction       int     `json:"direction"`
		Lot             float64 `json:"lot"`
		SL              float64 `json:"sl"`
		TP              float64 `json:"tp"`
		Status          string  `json:"status"`
		RejectionReason string  `json:"rejectionReason"`
		AUMUsed         float64 `json:"aumUsed"`
		InvestorEquity  float64 `json:"investorEquity"`
		CreatedAt       string  `json:"createdAt"`
		ProcessedAt     string  `json:"processedAt"`
		TraderName      string  `json:"traderName"`
	}
	var result []Row
	for rows.Next() {
		var r Row
		var createdAt, processedAt interface{}
		if err := rows.Scan(
			&r.ID, &r.Action, &r.Symbol, &r.Direction,
			&r.Lot, &r.SL, &r.TP,
			&r.Status, &r.RejectionReason,
			&r.AUMUsed, &r.InvestorEquity,
			&createdAt, &processedAt, &r.TraderName,
		); err != nil { continue }
		if t, ok := createdAt.(interface{ Format(string) string }); ok {
			r.CreatedAt = t.Format("2006-01-02 15:04")
		}
		result = append(result, r)
	}
	if result == nil { result = []Row{} }
	c.JSON(200, gin.H{"ok": true, "events": result, "count": len(result)})
}
