package investor

import (
	"github.com/gin-gonic/gin"
)

// GET /api/ea/investor/pending-copy-trades
// EA investor poll copy trade events yang perlu dieksekusi
func (h *Handler) EAGetPendingCopyTrades(c *gin.Context) {
	investorID := c.GetHeader("X-Investor-ID")
	if investorID == "" {
		c.JSON(401, gin.H{"ok": false, "error": "missing investor id"})
		return
	}
	engine := NewCopyTraderEngine(h.service.repo.DB)
	events, err := engine.GetPendingCopyEvents(investorID)
	if err != nil {
		c.JSON(500, gin.H{"ok": false, "error": "db error"})
		return
	}
	c.JSON(200, gin.H{"ok": true, "events": events, "count": len(events)})
}

// POST /api/ea/investor/copy-trade-update
// EA report hasil eksekusi copy trade
func (h *Handler) EACopyTradeUpdate(c *gin.Context) {
	investorID := c.GetHeader("X-Investor-ID")
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
	investorID := c.GetHeader("X-Investor-ID")
	if investorID == "" {
		c.JSON(401, gin.H{"ok": false, "error": "missing investor id"})
		return
	}
	var req struct {
		Equity  float64 `json:"equity"`
		Balance float64 `json:"balance"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.Equity <= 0 {
		c.JSON(400, gin.H{"ok": false, "error": "invalid equity"})
		return
	}
	_, err := h.service.repo.DB.Exec(
		`INSERT INTO investor_settings (investor_id, investor_equity, updated_at)
		 VALUES ($1::uuid, $2, now())
		 ON CONFLICT (investor_id) DO UPDATE SET
		   investor_equity = $2,
		   updated_at = now()`,
		investorID, req.Equity)
	if err != nil {
		c.JSON(500, gin.H{"ok": false, "error": "failed to update equity"})
		return
	}
	c.JSON(200, gin.H{"ok": true, "message": "equity updated", "equity": req.Equity})
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
