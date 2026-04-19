package investor

import (
	"strings"
	"database/sql"
	"encoding/json"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

func getUID(c *gin.Context) (string, bool) {
	v, exists := c.Get("user_id")
	if !exists { return "", false }
	s, ok := v.(string)
	return s, ok && s != ""
}

// GET /api/investor/analyst-sets
func (h *Handler) GetAnalystSets(c *gin.Context) {
	uid, ok := getUID(c)
	if !ok { c.JSON(401, gin.H{"ok": false, "error": "unauthorized"}); return }

	rows, err := h.service.repo.DB.Query(`
		SELECT ss.id, ss.name, ss.analyst_id,
		       COALESCE(u.name, u.email, '') as analyst_name,
		       COALESCE(s.allocation_pct, 0) as allocation_pct,
		       COALESCE(sub.status,'') as sub_status,
		       COALESCE(sub.auto_follow, false) as auto_follow,
		       COALESCE(sub.started_at::text,'') as started_at,
		       COALESCE(sub.expires_at::text,'') as expires_at,
		       COALESCE(ss.alpha_score,0) as alpha_score,
		       COALESCE(ss.alpha_grade,'D') as alpha_grade,
			       COALESCE(sub.id::text,'') as subscription_id
		FROM analyst_signal_sets ss
		LEFT JOIN users u ON u.id = ss.analyst_id::uuid
		LEFT JOIN analyst_subscriptions sub ON sub.set_id=ss.id AND sub.investor_id=$1 AND sub.status='ACTIVE'
		WHERE ss.status='Active' AND sub.id IS NOT NULL
		ORDER BY ss.created_at DESC`, uid)
	if err != nil { c.JSON(500, gin.H{"ok": false, "error": "db error"}); return }
	defer rows.Close()

	type SetRow struct {
		ID             string  `json:"id"`
		Name           string  `json:"name"`
		AnalystName    string  `json:"analystName"`
		SubStatus      string  `json:"subStatus"`
		AutoFollow     bool    `json:"autoFollow"`
		Subscribed     bool    `json:"subscribed"`
		StartedAt      string  `json:"startedAt"`
		ExpiresAt      string  `json:"expiresAt"`
		AlphaScore     float64 `json:"alphaScore"`
		AlphaGrade     string  `json:"alphaGrade"`
		SubscriptionID string  `json:"subscriptionId"`
	}
	var sets []SetRow
	for rows.Next() {
		var s SetRow
		rows.Scan(&s.ID, &s.Name, new(string), &s.AnalystName, &s.SubStatus, &s.AutoFollow,
			&s.StartedAt, &s.ExpiresAt, &s.AlphaScore, &s.AlphaGrade, &s.SubscriptionID)
		s.Subscribed = s.SubStatus == "ACTIVE"
		sets = append(sets, s)
	}
	if sets == nil { sets = []SetRow{} }
	c.JSON(200, gin.H{"ok": true, "sets": sets})
}

// GET /api/investor/analyst-subscriptions
func (h *Handler) GetAnalystSubscriptions(c *gin.Context) {
	uid, ok := getUID(c)
	if !ok { c.JSON(401, gin.H{"ok": false, "error": "unauthorized"}); return }

	rows, err := h.service.repo.DB.Query(`
		SELECT s.id, s.set_id, s.status, s.auto_follow, s.created_at,
		       ss.name as set_name,
		       COALESCE(u.name, u.email, '') as analyst_name,
		       COALESCE(s.allocation_pct, 0) as allocation_pct,
		       COALESCE(ss.alpha_score, 0) as alpha_score,
		       COALESCE(ss.alpha_grade, 'D') as alpha_grade
		FROM analyst_subscriptions s
		JOIN analyst_signal_sets ss ON ss.id = s.set_id
		LEFT JOIN users u ON u.id = ss.analyst_id::uuid
		WHERE s.investor_id=$1 AND s.status='ACTIVE'
		ORDER BY s.created_at DESC`, uid)
	if err != nil { c.JSON(500, gin.H{"ok": false, "error": "db error"}); return }
	defer rows.Close()

	type SubRow struct {
		ID            string  `json:"id"`
		SetID         string  `json:"setId"`
		SetName       string  `json:"setName"`
		AnalystName   string  `json:"analystName"`
		Status        string  `json:"status"`
		AutoFollow    bool    `json:"autoFollow"`
		CreatedAt     string  `json:"createdAt"`
		AllocationPct float64 `json:"allocationPct"`
		AlphaScore    float64 `json:"alphaScore"`
		AlphaGrade    string  `json:"alphaGrade"`
	}
	var subs []SubRow
	for rows.Next() {
		var s SubRow
		var createdAt time.Time
		rows.Scan(&s.ID, &s.SetID, &s.Status, &s.AutoFollow, &createdAt, &s.SetName, &s.AnalystName, &s.AllocationPct, &s.AlphaScore, &s.AlphaGrade)
		s.CreatedAt = createdAt.Format("2006-01-02")
		subs = append(subs, s)
	}
	if subs == nil { subs = []SubRow{} }
	c.JSON(200, gin.H{"ok": true, "subscriptions": subs})
}

// POST /api/investor/analyst-subscribe
func (h *Handler) SubscribeAnalystSet(c *gin.Context) {
	uid, ok := getUID(c)
	if !ok { c.JSON(401, gin.H{"ok": false, "error": "unauthorized"}); return }

	var req struct {
		SetID      string `json:"setId"`
		AutoFollow bool   `json:"autoFollow"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.SetID == "" {
		c.JSON(400, gin.H{"ok": false, "error": "setId required"}); return
	}

	var analystID string
	err := h.service.repo.DB.QueryRow(`SELECT analyst_id FROM analyst_signal_sets WHERE id=$1`, req.SetID).Scan(&analystID)
	if err != nil { c.JSON(404, gin.H{"ok": false, "error": "signal set not found"}); return }

	// Update existing row jika ada (any status), kalau tidak ada baru insert
	res, err := h.service.repo.DB.Exec(`UPDATE analyst_subscriptions
		SET status='ACTIVE', auto_follow=$3, cancelled_at=NULL, created_at=now(),
		    started_at=now(), expires_at=now() + interval '30 days'
		WHERE id = (
			SELECT id FROM analyst_subscriptions
			WHERE investor_id=$1::uuid AND set_id=$2
			ORDER BY created_at DESC LIMIT 1
		)`,
		uid, req.SetID, req.AutoFollow)
	if err != nil { c.JSON(500, gin.H{"ok": false, "error": "subscribe failed update: "+err.Error()}); return }
	rows, _ := res.RowsAffected()
	if rows == 0 {
		_, err = h.service.repo.DB.Exec(`INSERT INTO analyst_subscriptions
			(investor_id, analyst_id, set_id, status, auto_follow, created_at, cancelled_at, started_at, expires_at)
			VALUES ($1::uuid,$2::uuid,$3,'ACTIVE',$4,now(),NULL,now(),now() + interval '30 days')`,
			uid, analystID, req.SetID, req.AutoFollow)
		if err != nil { c.JSON(500, gin.H{"ok": false, "error": "subscribe failed insert: "+err.Error()}); return }
	}
	c.JSON(200, gin.H{"ok": true, "message": "Subscribed successfully"})
}

// POST /api/investor/analyst-unsubscribe
func (h *Handler) UnsubscribeAnalystSet(c *gin.Context) {
	uid, ok := getUID(c)
	if !ok { c.JSON(401, gin.H{"ok": false, "error": "unauthorized"}); return }

	var req struct { SetID string `json:"setId"` }
	if err := c.ShouldBindJSON(&req); err != nil || req.SetID == "" {
		c.JSON(400, gin.H{"ok": false, "error": "setId required"}); return
	}

	_, err := h.service.repo.DB.Exec(`UPDATE analyst_subscriptions SET status='CANCELLED', cancelled_at=now(), expires_at=now()
		WHERE investor_id=$1 AND set_id=$2 AND status='ACTIVE'`, uid, req.SetID)
	if err != nil { c.JSON(500, gin.H{"ok": false, "error": "unsubscribe failed"}); return }
	c.JSON(200, gin.H{"ok": true, "message": "Unsubscribed"})
}

// GET /api/investor/analyst-feed
func (h *Handler) GetAnalystFeed(c *gin.Context) {
	uid, ok := getUID(c)
	if !ok { c.JSON(401, gin.H{"ok": false, "error": "unauthorized"}); return }

	rows, err := h.service.repo.DB.Query(`
		SELECT sig.id, sig.pair, sig.direction, sig.entry, sig.sl, sig.tp,
		       sig.status, sig.issued_at, COALESCE(sig.notes,''),
		       ss.name as set_name,
		       COALESCE(sig.running_at::text,'') as running_at,
		       COALESCE(sig.closed_at::text,'') as closed_at
		FROM analyst_signals sig
		JOIN analyst_signal_sets ss ON ss.id = sig.set_id
		JOIN analyst_subscriptions sub ON sub.set_id = sig.set_id
		WHERE sub.investor_id=$1 AND sub.status='ACTIVE'
		ORDER BY sig.id DESC LIMIT 200`, uid)
	if err != nil { c.JSON(500, gin.H{"ok": false, "error": "db error"}); return }
	defer rows.Close()

	type SigRow struct {
		ID        int64  `json:"id"`
		Pair      string `json:"pair"`
		Direction string `json:"direction"`
		Entry     string `json:"entry"`
		SL        string `json:"sl"`
		TP        string `json:"tp"`
		Status    string `json:"status"`
		IssuedAt  string `json:"issuedAt"`
		Notes     string `json:"notes"`
		SetName   string `json:"setName"`
		RunningAt string `json:"runningAt"`
		ClosedAt  string `json:"closedAt"`
	}
	var sigs []SigRow
	for rows.Next() {
		var s SigRow
		rows.Scan(&s.ID, &s.Pair, &s.Direction, &s.Entry, &s.SL, &s.TP,
			&s.Status, &s.IssuedAt, &s.Notes, &s.SetName, &s.RunningAt, &s.ClosedAt)
		sigs = append(sigs, s)
	}
	if sigs == nil { sigs = []SigRow{} }
	c.JSON(200, gin.H{"ok": true, "signals": sigs, "count": len(sigs)})
}

var _ = sql.ErrNoRows
var _ = http.StatusOK

// GET /api/investor/subscription-history — only CANCELLED subscriptions (completed periods)
func (h *Handler) GetSubscriptionHistory(c *gin.Context) {
	uid, ok := getUID(c)
	if !ok { c.JSON(401, gin.H{"ok": false, "error": "unauthorized"}); return }

	rows, err := h.service.repo.DB.Query(`
		SELECT 
			sub.id::text, sub.set_id, sub.status, sub.auto_follow,
			COALESCE(sub.started_at::text,'') as started_at,
			COALESCE(sub.expires_at::text,'') as expires_at,
			COALESCE(sub.cancelled_at::text,'') as cancelled_at,
			COALESCE(ss.name,'') as set_name,
			COALESCE(NULLIF(u.name,''), u.email, '') as analyst_name,
			COALESCE(ss.alpha_score,0) as alpha_score,
			COALESCE(ss.alpha_grade,'') as alpha_grade,
			COUNT(o.id) as total_orders,
			COUNT(o.id) FILTER (WHERE o.status='CLOSED_TP') as wins,
			COUNT(o.id) FILTER (WHERE o.status='CLOSED_SL') as losses,
			COALESCE(SUM(CASE 
				WHEN o.status='CLOSED_TP' THEN (o.close_price - o.open_price) * o.lot_size * 100
				WHEN o.status='CLOSED_SL' THEN (o.close_price - o.open_price) * o.lot_size * 100
				ELSE 0 END), 0) as total_pnl
		FROM analyst_subscriptions sub
		LEFT JOIN analyst_signal_sets ss ON ss.id = sub.set_id
		LEFT JOIN users u ON u.id::text = ss.analyst_id
		LEFT JOIN investor_signal_orders o ON o.set_id=sub.set_id 
			AND o.investor_id=sub.investor_id
			AND o.created_at >= sub.started_at
			AND (sub.cancelled_at IS NULL OR o.created_at <= sub.cancelled_at)
		WHERE sub.investor_id=$1::uuid AND sub.status='CANCELLED'
		GROUP BY sub.id, sub.set_id, sub.status, sub.auto_follow,
		         sub.started_at, sub.expires_at, sub.cancelled_at,
		         ss.name, u.name, u.email, ss.alpha_score, ss.alpha_grade
		ORDER BY sub.cancelled_at DESC`, uid)
	if err != nil { c.JSON(500, gin.H{"ok": false, "error": err.Error()}); return }
	defer rows.Close()

	type HistRow struct {
		ID          string  `json:"id"`
		SetID       string  `json:"setId"`
		SetName     string  `json:"setName"`
		AnalystName string  `json:"analystName"`
		Status      string  `json:"status"`
		AutoFollow  bool    `json:"autoFollow"`
		StartedAt   string  `json:"startedAt"`
		ExpiresAt   string  `json:"expiresAt"`
		CancelledAt string  `json:"cancelledAt"`
		AlphaScore  float64 `json:"alphaScore"`
		AlphaGrade  string  `json:"alphaGrade"`
		TotalOrders int     `json:"totalOrders"`
		Wins        int     `json:"wins"`
		Losses      int     `json:"losses"`
		WinRate     float64 `json:"winRate"`
		TotalPnL    float64 `json:"totalPnl"`
	}
	var items []HistRow
	for rows.Next() {
		var row HistRow
		err := rows.Scan(&row.ID, &row.SetID, &row.Status, &row.AutoFollow,
			&row.StartedAt, &row.ExpiresAt, &row.CancelledAt,
			&row.SetName, &row.AnalystName, &row.AlphaScore, &row.AlphaGrade,
			&row.TotalOrders, &row.Wins, &row.Losses, &row.TotalPnL)
		if err != nil { continue }
		if row.TotalOrders > 0 {
			row.WinRate = float64(row.Wins) / float64(row.TotalOrders) * 100
		}
		items = append(items, row)
	}
	if items == nil { items = []HistRow{} }
	c.JSON(200, gin.H{"ok": true, "history": items, "count": len(items)})
}

// POST /api/investor/copy-trader-subscribe
func (h *Handler) CopyTraderSubscribe(c *gin.Context) {
	uid, ok := getUID(c)
	if !ok { c.JSON(401, gin.H{"ok": false, "error": "unauthorized"}); return }
	var req struct {
		TraderAccountID string  `json:"traderAccountId"`
		LotMode         string  `json:"lotMode"`
		LotSize         float64 `json:"lotSize"`
		RiskPercent     float64 `json:"riskPercent"`
		MaxLot          float64 `json:"maxLot"`
		CopySL          bool    `json:"copySl"`
		CopyTP          bool    `json:"copyTp"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.TraderAccountID == "" {
		c.JSON(400, gin.H{"ok": false, "error": "traderAccountId required"}); return
	}
	lotMethod := req.LotMode
	if lotMethod == "" { lotMethod = "AUM" }
	lotSize := req.LotSize
	if lotSize == 0 { lotSize = 0.01 }
	maxLot := req.MaxLot
	if maxLot == 0 { maxLot = 1.0 }

	// Get investor own trader account (FK requirement for copy_subscriptions)
	var followerAccountID string
	dbErr := h.service.repo.DB.QueryRow(`
		SELECT id::text FROM trader_accounts
		WHERE user_id=$1::uuid AND status='active'
		ORDER BY created_at ASC LIMIT 1`, uid).Scan(&followerAccountID)
	if dbErr != nil || followerAccountID == "" {
		c.JSON(400, gin.H{"ok": false, "error": "no_account", "message": "Link an MT5/MT4 account first. Go to Trader Dashboard → Add Account."}); return
	}
	if followerAccountID == req.TraderAccountID {
		c.JSON(400, gin.H{"ok": false, "error": "Cannot copy your own account"}); return
	}

	_, dbErr = h.service.repo.DB.Exec(`
		INSERT INTO copy_subscriptions
			(id, follower_account_id, provider_account_id, lot_multiplier, max_lot, min_lot,
			 copy_sl, copy_tp, status, lot_calculation_method, max_risk_percent, created_at, updated_at)
		VALUES (gen_random_uuid(), $1::uuid, $2::uuid, $3, $4, 0.01, $5, $6, 'ACTIVE', $7, $8, now(), now())
		ON CONFLICT (follower_account_id, provider_account_id) DO UPDATE SET
			status='ACTIVE', lot_multiplier=$3, max_lot=$4, copy_sl=$5, copy_tp=$6,
			lot_calculation_method=$7, max_risk_percent=$8, updated_at=now()`,
		followerAccountID, req.TraderAccountID, lotSize, maxLot,
		req.CopySL, req.CopyTP, lotMethod, req.RiskPercent)
	if dbErr != nil {
		c.JSON(500, gin.H{"ok": false, "error": "subscribe failed: " + dbErr.Error()}); return
	}
	// Upsert user_allocations for AUM tracking
	h.service.repo.DB.Exec(`
		INSERT INTO user_allocations
			(user_id, trader_account_id, allocation_mode, allocation_value, max_risk_pct, max_positions, status, created_at, updated_at)
		VALUES ($1::uuid, $2::uuid, 'PERCENT', 10, 5, 10, 'ACTIVE', now(), now())
		ON CONFLICT (user_id, trader_account_id) DO UPDATE SET status='ACTIVE', updated_at=now()`,
		followerAccountID, req.TraderAccountID)
	c.JSON(200, gin.H{"ok": true, "message": "Subscribed to copy trader"})
}


// POST /api/investor/copy-trader-unsubscribe
func (h *Handler) CopyTraderUnsubscribe(c *gin.Context) {
	uid, ok := getUID(c)
	if !ok { c.JSON(401, gin.H{"ok": false, "error": "unauthorized"}); return }
	var req struct { TraderAccountID string `json:"traderAccountId"` }
	if err := c.ShouldBindJSON(&req); err != nil || req.TraderAccountID == "" {
		c.JSON(400, gin.H{"ok": false, "error": "traderAccountId required"}); return
	}
	var followerAccountID string
	h.service.repo.DB.QueryRow(
		`SELECT id::text FROM trader_accounts WHERE user_id=$1::uuid AND status='active' LIMIT 1`,
		uid).Scan(&followerAccountID)
	if followerAccountID == "" {
		c.JSON(400, gin.H{"ok": false, "error": "no active broker account found"}); return
	}
	_, err := h.service.repo.DB.Exec(`
		UPDATE copy_subscriptions SET status='inactive', updated_at=now()
		WHERE follower_account_id=$1::uuid AND provider_account_id=$2::uuid`,
		followerAccountID, req.TraderAccountID)
	if err != nil { c.JSON(500, gin.H{"ok": false, "error": err.Error()}); return }
	c.JSON(200, gin.H{"ok": true, "message": "Unsubscribed"})
}

// GET /api/investor/copy-trader-subscriptions
func (h *Handler) GetCopyTraderSubscriptions(c *gin.Context) {
	uid, ok := getUID(c)
	if !ok { c.JSON(401, gin.H{"ok": false, "error": "unauthorized"}); return }
	// Get investor's trader accounts first
	var followerAccountID string
	h.service.repo.DB.QueryRow(`SELECT id::text FROM trader_accounts WHERE user_id=$1::uuid AND status='active' LIMIT 1`, uid).Scan(&followerAccountID)

	if followerAccountID == "" {
		c.JSON(200, gin.H{"ok": true, "subscriptions": []interface{}{}, "count": 0})
		return
	}

	rows, err := h.service.repo.DB.Query(`
		SELECT cs.id::text, cs.provider_account_id::text, cs.status,
			cs.lot_calculation_method, cs.lot_multiplier, cs.max_lot, cs.max_risk_percent,
			cs.copy_sl, cs.copy_tp, cs.created_at::text,
			COALESCE(u.name, ta.nickname, ta.account_number) as trader_name,
			COALESCE(ta.broker,'') as broker, COALESCE(ta.platform::text,'') as platform,
			COALESCE(ar.alpha_score,0) as alpha_score, COALESCE(ar.grade,'') as grade,
			COALESCE(ar.risk_level,'MEDIUM') as risk_level,
			COALESCE(ar.layer3_multiplier,1.0) as layer3_multiplier,
			COALESCE(ar.layer3_status,'NEUTRAL') as layer3_status,
			COALESCE(ar.layer3_detail->>'system_mode','FULL_ACTIVE') as layer3_system_mode,
			COALESCE(ar.layer3_reason,'') as layer3_reason
		FROM copy_subscriptions cs
		LEFT JOIN trader_accounts ta ON ta.id = cs.provider_account_id
		LEFT JOIN users u ON u.id = ta.user_id
		LEFT JOIN alpha_ranks ar ON ar.account_id = cs.provider_account_id AND ar.symbol='ALL'
		WHERE cs.follower_account_id=$1::uuid
		ORDER BY cs.created_at DESC`, followerAccountID)
	if err != nil { c.JSON(500, gin.H{"ok": false, "error": err.Error()}); return }
	defer rows.Close()
	type SubRow struct {
		ID         string  `json:"id"`
		TraderID   string  `json:"traderId"`
		Status     string  `json:"status"`
		LotMode    string  `json:"lotMode"`
		LotSize    float64 `json:"lotSize"`
		MaxLot     float64 `json:"maxLot"`
		RiskPct    float64 `json:"riskPct"`
		CopySL     bool    `json:"copySl"`
		CopyTP     bool    `json:"copyTp"`
		CreatedAt  string  `json:"createdAt"`
		TraderName string  `json:"traderName"`
		Broker     string  `json:"broker"`
		Platform   string  `json:"platform"`
		AlphaScore float64 `json:"alphaScore"`
		Grade      string  `json:"grade"`
			RiskLevel        string  `json:"riskLevel"`
			Layer3Multiplier float64 `json:"layer3Multiplier"`
			Layer3Status     string  `json:"layer3Status"`
			Layer3SystemMode string  `json:"layer3SystemMode"`
			Layer3Reason     string  `json:"layer3Reason"`
	}
	var subs []SubRow
	for rows.Next() {
		var s SubRow
		if err := rows.Scan(&s.ID,&s.TraderID,&s.Status,&s.LotMode,&s.LotSize,&s.MaxLot,
			&s.RiskPct,&s.CopySL,&s.CopyTP,&s.CreatedAt,&s.TraderName,&s.Broker,&s.Platform,
			&s.AlphaScore,&s.Grade,
                        &s.RiskLevel,&s.Layer3Multiplier,&s.Layer3Status,
                        &s.Layer3SystemMode,&s.Layer3Reason); err != nil { continue }
		subs = append(subs, s)
	}
	if subs == nil { subs = []SubRow{} }
	c.JSON(200, gin.H{"ok": true, "subscriptions": subs, "count": len(subs)})
}

// GET /api/investor/trader-profile/:account_id
func (h *Handler) GetTraderProfile(c *gin.Context) {
	accountID := c.Param("account_id")
	if accountID == "" { c.JSON(400, gin.H{"ok":false,"error":"account_id required"}); return }

	// Basic info + alpharank
	var profile struct {
		ID             string  `json:"id"`
		AccountNumber  string  `json:"account_number"`
		Broker         string  `json:"broker"`
		Platform       string  `json:"platform"`
		TraderName     string  `json:"trader_name"`
		Nickname       string  `json:"nickname"`
		Country        string  `json:"country"`
		Bio            string  `json:"bio"`
		Equity         float64 `json:"equity"`
		Balance        float64 `json:"balance"`
		AlphaScore     float64 `json:"alpha_score"`
		Grade          string  `json:"grade"`
		MaxDrawdownPct float64 `json:"max_drawdown_pct"`
		WinRate        float64 `json:"win_rate"`
		NetProfit      float64 `json:"net_profit"`
		TotalTrades    int     `json:"total_trades"`
		ProfitFactor   float64 `json:"profit_factor"`
		Survivability  float64 `json:"survivability"`
		Scalability    float64 `json:"scalability"`
			ROI           float64 `json:"roi"`
			TotalDeposit  float64 `json:"total_deposit"`
			TotalWithdraw float64 `json:"total_withdraw"`
				RiskLevel     string  `json:"risk_level"`
				Strategy      string  `json:"strategy"`
	}
	err := h.service.repo.DB.QueryRow(`
		SELECT ta.id::text, ta.account_number, COALESCE(ta.broker,''), COALESCE(ta.platform::text,''),
				COALESCE(u.name, ta.nickname, ta.account_number) as trader_name,
				COALESCE(ta.nickname,''), COALESCE(u.country,'') as country, COALESCE(u.bio,'') as bio,
				COALESCE(ta.about,'') as strategy,
				COALESCE(ta.equity,0), COALESCE(ta.balance,0),
			COALESCE(ar.alpha_score,0), COALESCE(ar.grade,'N/A'),
			COALESCE(ar.max_drawdown_pct,0)
		FROM trader_accounts ta
		LEFT JOIN alpha_ranks ar ON ar.account_id=ta.id AND ar.symbol='ALL'
		LEFT JOIN users u ON u.id = ta.user_id
		WHERE ta.id=$1::uuid AND ta.status='active'`, accountID).Scan(
		&profile.ID, &profile.AccountNumber, &profile.Broker, &profile.Platform,
			&profile.TraderName, &profile.Nickname, &profile.Country, &profile.Bio, &profile.Strategy,
				&profile.Equity, &profile.Balance,
		&profile.AlphaScore, &profile.Grade, &profile.MaxDrawdownPct)
	if err != nil { c.JSON(404, gin.H{"ok":false,"error":"trader not found: "+err.Error()}); return }

	// Get performance from alpha_ranks (single source of truth)
	h.service.repo.DB.QueryRow(`
			SELECT COALESCE(win_rate,0), COALESCE(net_pnl,0),
					COALESCE(total_trades_all,0), COALESCE(profit_factor,0),
					COALESCE(risk_level,'MEDIUM')
			FROM alpha_ranks
			WHERE account_id=$1::uuid AND symbol='ALL'`, accountID).Scan(
				&profile.WinRate, &profile.NetProfit, &profile.TotalTrades, &profile.ProfitFactor, &profile.RiskLevel)

	// Get deposit/withdraw for ROI
	h.service.repo.DB.QueryRow(`
			SELECT COALESCE(SUM(CASE WHEN transaction_type='deposit' THEN amount ELSE 0 END),0),
				COALESCE(SUM(CASE WHEN transaction_type='withdrawal' THEN amount ELSE 0 END),0)
			FROM account_transactions WHERE account_id=$1::uuid`, accountID).Scan(
			&profile.TotalDeposit, &profile.TotalWithdraw)
	if profile.TotalDeposit > 0 {
		profile.ROI = ((profile.Equity + profile.TotalWithdraw - profile.TotalDeposit) / profile.TotalDeposit) * 100
	}

	// Calculate survivability + scalability from DB (same formula as trader service)
	// Survivability = alphaScore - penalty(maxDD)
	survScore := profile.AlphaScore
	if profile.MaxDrawdownPct > 50 { survScore -= 30 } else if profile.MaxDrawdownPct > 30 { survScore -= 15 }
	if survScore < 0 { survScore = 0 }
	profile.Survivability = survScore

	// Scalability = alphaScore*0.7 + bonus(balance)
	scalScore := profile.AlphaScore * 0.7
	if profile.Balance >= 100000 { scalScore += 20 } else if profile.Balance >= 10000 { scalScore += 10 } else if profile.Balance >= 1000 { scalScore += 5 }
	if scalScore > 100 { scalScore = 100 }
	if scalScore < 0 { scalScore = 0 }
	profile.Scalability = scalScore

	// Pillars - stored as JSONB in alpha_ranks
	type Pillar struct {
		Code   string  `json:"code"`
		Name   string  `json:"name"`
		Score  float64 `json:"score"`
		Weight int     `json:"weight"`
		Reason string  `json:"reason"`
	}
	var pillarsJSON []byte
	var pillars []Pillar
	h.service.repo.DB.QueryRow(`
		SELECT COALESCE(pillars::text,'[]') FROM alpha_ranks
		WHERE account_id=$1::uuid AND symbol='ALL' LIMIT 1`, accountID).Scan(&pillarsJSON)
	if pillarsJSON != nil {
		json.Unmarshal(pillarsJSON, &pillars)
	}

	// Per pair
	type PairItem struct {
		Symbol    string      `json:"symbol"`
		Trades    int         `json:"trades"`
		WinRate   float64     `json:"win_rate"`
		NetProfit float64     `json:"net_profit"`
		AvgRR     float64     `json:"avg_rr"`
		Flags     interface{} `json:"flags"`
	}
	ppRows, _ := h.service.repo.DB.Query(`
			SELECT ar.symbol,
				COALESCE(ar.total_trades_all, ar.trade_count,0),
				COALESCE(ar.win_rate,0), COALESCE(ar.net_pnl,0),
				COALESCE(ar.risk_reward,0),
				COALESCE(ar.risk_flags::text,'[]'::text)
			FROM alpha_ranks ar
			WHERE ar.account_id=$1::uuid AND ar.symbol!='ALL' AND ar.trade_count >= 20
			ORDER BY ar.net_pnl DESC`, accountID)
	var pairs []PairItem
	if ppRows != nil {
		defer ppRows.Close()
		for ppRows.Next() {
			var p PairItem
			var flagsRaw string
			ppRows.Scan(&p.Symbol,&p.Trades,&p.WinRate,&p.NetProfit,&p.AvgRR,&flagsRaw)
			var flagsParsed []interface{}
			json.Unmarshal([]byte(flagsRaw), &flagsParsed)
			if flagsParsed == nil { flagsParsed = []interface{}{} }
			p.Flags = flagsParsed
			pairs = append(pairs, p)
		}
	}

	// Monthly performance
	type MonthRow struct {
		Year  int     `json:"year"`
		Month int     `json:"month"`
		Profit float64 `json:"profit"`
		Trades int     `json:"trades"`
		WinRate float64 `json:"winRate"`
	}
	mRows, _ := h.service.repo.DB.Query(`
		SELECT year, month, COALESCE(net_profit,0), COALESCE(total_trades,0), COALESCE(win_rate,0)
		FROM performance_metrics
		WHERE account_id=$1::uuid AND period='MONTHLY' AND account_type='trader'
		ORDER BY year, month DESC LIMIT 24`, accountID)
	var monthly []MonthRow
	if mRows != nil {
		defer mRows.Close()
		for mRows.Next() {
			var m MonthRow
			mRows.Scan(&m.Year,&m.Month,&m.Profit,&m.Trades,&m.WinRate)
			monthly = append(monthly, m)
		}
	}

	if pillars == nil { pillars = []Pillar{} }
	if pairs == nil { pairs = []PairItem{} }
	if monthly == nil { monthly = []MonthRow{} }

	// Risk flags from alpha_ranks
	var flagsJSON []byte
	h.service.repo.DB.QueryRow(`
		SELECT COALESCE(risk_flags::text,'[]') FROM alpha_ranks
		WHERE account_id=$1::uuid AND symbol='ALL' LIMIT 1`, accountID).Scan(&flagsJSON)

	// Statistics dari alpha_ranks
	var stats struct {
		TotalTrades   int     `json:"totalTrades"`
		WinningTrades int     `json:"winningTrades"`
		LosingTrades  int     `json:"losingTrades"`
		WinRate       float64 `json:"winRate"`
		ProfitFactor  float64 `json:"profitFactor"`
		AvgWin        float64 `json:"avgWin"`
		AvgLoss       float64 `json:"avgLoss"`
		RiskReward    float64 `json:"riskReward"`
		Expectancy    float64 `json:"expectancy"`
	}
	h.service.repo.DB.QueryRow(`
		SELECT COALESCE(total_trades_all,0), COALESCE(winning_trades,0), COALESCE(losing_trades,0),
			COALESCE(win_rate,0), COALESCE(profit_factor,0),
			COALESCE(avg_win,0), COALESCE(avg_loss,0),
			COALESCE(risk_reward,0), COALESCE(expectancy,0)
		FROM alpha_ranks WHERE account_id=$1::uuid AND symbol='ALL' LIMIT 1`, accountID).Scan(
		&stats.TotalTrades, &stats.WinningTrades, &stats.LosingTrades,
		&stats.WinRate, &stats.ProfitFactor,
		&stats.AvgWin, &stats.AvgLoss, &stats.RiskReward, &stats.Expectancy)

	c.JSON(200, gin.H{"ok":true,"profile":profile,"pillars":pillars,"pairs":pairs,
		"monthly":monthly,"risk_flags_raw":string(flagsJSON),"statistics":stats})
}

// GET /api/investor/trade-copies — copy executions for this investor
func (h *Handler) GetTradeCopies(c *gin.Context) {
	uid, ok := getUID(c)
	if !ok { c.JSON(401, gin.H{"ok":false,"error":"unauthorized"}); return }

	// Get investor's trader accounts
	var followerAccountID string
	h.service.repo.DB.QueryRow(`SELECT id::text FROM trader_accounts WHERE user_id=$1::uuid AND status='active' LIMIT 1`, uid).Scan(&followerAccountID)

	if followerAccountID == "" {
		c.JSON(200, gin.H{"ok":true,"copies":[]interface{}{}})
		return
	}

	rows, err := h.service.repo.DB.Query(`
		SELECT ce.id::text, ce.follower_ticket, ce.executed_lots, ce.executed_price::text,
		       ce.success, ce.executed_at::text, ce.error_message,
		       COALESCE(u.name, ta.nickname, ta.account_number) as trader_name
		FROM copy_executions ce
		JOIN copy_subscriptions cs ON cs.id = ce.subscription_id
		JOIN trader_accounts ta ON ta.id = cs.provider_account_id
		LEFT JOIN users u ON u.id = ta.user_id
		WHERE cs.follower_account_id = $1::uuid
		ORDER BY ce.executed_at DESC
		LIMIT 100`, followerAccountID)
	if err != nil { c.JSON(500, gin.H{"ok":false,"error":err.Error()}); return }
	defer rows.Close()

	type CopyRow struct {
		ID            string  `json:"id"`
		FollowerTicket *int64 `json:"followerTicket"`
		ExecutedLots  float64 `json:"executedLots"`
		ExecutedPrice string  `json:"executedPrice"`
		Success       bool    `json:"success"`
		ExecutedAt    string  `json:"executedAt"`
		ErrorMessage  *string `json:"errorMessage"`
		TraderName    string  `json:"traderName"`
	}

	var copies []CopyRow
	for rows.Next() {
		var r CopyRow
		rows.Scan(&r.ID, &r.FollowerTicket, &r.ExecutedLots, &r.ExecutedPrice,
			&r.Success, &r.ExecutedAt, &r.ErrorMessage, &r.TraderName)
		copies = append(copies, r)
	}
	if copies == nil { copies = []CopyRow{} }
	c.JSON(200, gin.H{"ok":true,"copies":copies})
}

// PUT /api/investor/analyst-subscription/:id/allocation — set allocation pct
func (h *Handler) UpdateSubscriptionAllocation(c *gin.Context) {
	uid, ok := getUID(c)
	if !ok { c.JSON(401, gin.H{"ok":false,"error":"unauthorized"}); return }
	subID := c.Param("id")
	var req struct {
		AllocationPct float64 `json:"allocationPct"`
		AlphaScore    float64 `json:"alphaScore"`
		AlphaGrade    string  `json:"alphaGrade"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"ok":false,"error":"invalid json"}); return
	}
	if req.AllocationPct < 0 || req.AllocationPct > 100 {
		c.JSON(400, gin.H{"ok":false,"error":"allocation must be 0-100"}); return
	}
	_, err := h.service.repo.DB.Exec(`
		UPDATE analyst_subscriptions
		SET allocation_pct=$1
		WHERE id=$2 AND investor_id=$3::uuid AND status='ACTIVE'`,
		req.AllocationPct, subID, uid)
	if err != nil { c.JSON(500, gin.H{"ok":false,"error":err.Error()}); return }
	c.JSON(200, gin.H{"ok":true,"allocationPct":req.AllocationPct})
}
// PUT /api/investor/analyst-subscription/:id/mode — switch auto/manual
func (h *Handler) UpdateSubscriptionMode(c *gin.Context) {
	uid, ok := getUID(c)
	if !ok { c.JSON(401, gin.H{"ok":false,"error":"unauthorized"}); return }

	subID := c.Param("id")
	var req struct {
		AutoFollow bool `json:"autoFollow"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"ok":false,"error":"invalid json"}); return
	}

	execMode := "MANUAL"
	if req.AutoFollow { execMode = "AUTO" }

	result, err := h.service.repo.DB.Exec(`
		UPDATE analyst_subscriptions 
		SET auto_follow=$1, execution_mode=$2
		WHERE id=$3 AND investor_id=$4::uuid AND status='ACTIVE'`,
		req.AutoFollow, execMode, subID, uid)
	if err != nil { c.JSON(500, gin.H{"ok":false,"error":err.Error()}); return }

	rows, _ := result.RowsAffected()
	if rows == 0 { c.JSON(404, gin.H{"ok":false,"error":"subscription not found"}); return }

	c.JSON(200, gin.H{"ok":true, "autoFollow":req.AutoFollow, "executionMode":execMode})
}

// GET /api/investor/performance-overview — investor performance summary
func (h *Handler) GetPerformanceOverview(c *gin.Context) {
	uid, ok := getUID(c)
	if !ok { c.JSON(401, gin.H{"ok":false,"error":"unauthorized"}); return }

	now := time.Now()
	weekStart := now.AddDate(0, 0, -7)
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())

	// Signal performance
	var totalSignalPnl, weekSignalPnl, monthSignalPnl float64
	var totalSignalTrades, winSignals, lossSignals int
	h.service.repo.DB.QueryRow(`
		SELECT 
			COALESCE(SUM(net_profit),0),
			COALESCE(SUM(CASE WHEN close_time >= $2 THEN net_profit ELSE 0 END),0),
			COALESCE(SUM(CASE WHEN close_time >= $3 THEN net_profit ELSE 0 END),0),
			COUNT(*),
			COALESCE(SUM(CASE WHEN outcome='WIN' THEN 1 ELSE 0 END),0),
			COALESCE(SUM(CASE WHEN outcome='LOSS' THEN 1 ELSE 0 END),0)
		FROM investor_signal_executions
		WHERE investor_id=$1::uuid AND close_time IS NOT NULL`,
		uid, weekStart, monthStart).Scan(
		&totalSignalPnl, &weekSignalPnl, &monthSignalPnl,
		&totalSignalTrades, &winSignals, &lossSignals)

	// Copy trade performance
	var totalCopyPnl, weekCopyPnl, monthCopyPnl float64
	var totalCopyTrades, winCopies int
	h.service.repo.DB.QueryRow(`
		SELECT
			COALESCE(SUM(t.profit),0),
			COALESCE(SUM(CASE WHEN t.close_time >= $2 THEN t.profit ELSE 0 END),0),
			COALESCE(SUM(CASE WHEN t.close_time >= $3 THEN t.profit ELSE 0 END),0),
			COUNT(*),
			COALESCE(SUM(CASE WHEN t.profit > 0 THEN 1 ELSE 0 END),0)
		FROM copy_executions ce
		JOIN copy_subscriptions cs ON cs.id = ce.subscription_id
		JOIN trades t ON t.ticket = ce.follower_ticket AND t.account_id = cs.follower_account_id
		WHERE cs.follower_account_id IN (
			SELECT id FROM trader_accounts WHERE user_id=$1::uuid AND status='active'
		) AND t.close_time IS NOT NULL`,
		uid, weekStart, monthStart).Scan(
		&totalCopyPnl, &weekCopyPnl, &monthCopyPnl,
		&totalCopyTrades, &winCopies)

	// Active counts
	var activeSignalSubs, activeCopyTrades int
	h.service.repo.DB.QueryRow(`SELECT COUNT(*) FROM analyst_subscriptions WHERE investor_id=$1::uuid AND status='ACTIVE'`, uid).Scan(&activeSignalSubs)
	h.service.repo.DB.QueryRow(`
		SELECT COUNT(*) FROM copy_subscriptions cs
		JOIN trader_accounts ta ON ta.id = cs.follower_account_id
		WHERE ta.user_id=$1::uuid AND cs.status='ACTIVE'`, uid).Scan(&activeCopyTrades)

	// Signal breakdown per set
	type SetPerf struct {
		SetID    string  `json:"setId"`
		SetName  string  `json:"setName"`
		Trades   int     `json:"trades"`
		Wins     int     `json:"wins"`
		Losses   int     `json:"losses"`
		TotalPnl float64 `json:"totalPnl"`
		WinRate  float64 `json:"winRate"`
	}
	setRows, _ := h.service.repo.DB.Query(`
		SELECT ase.set_id, ass.name,
			COUNT(*), 
			COALESCE(SUM(CASE WHEN ise.outcome='WIN' THEN 1 ELSE 0 END),0),
			COALESCE(SUM(CASE WHEN ise.outcome='LOSS' THEN 1 ELSE 0 END),0),
			COALESCE(SUM(ise.net_profit),0)
		FROM investor_signal_executions ise
		JOIN analyst_subscriptions ase ON ase.id = ise.subscription_id
		JOIN analyst_signal_sets ass ON ass.id = ase.set_id
		WHERE ise.investor_id=$1::uuid AND ise.close_time IS NOT NULL
		GROUP BY ase.set_id, ass.name`, uid)
	var setPerfs []SetPerf
	if setRows != nil {
		defer setRows.Close()
		for setRows.Next() {
			var s SetPerf
			setRows.Scan(&s.SetID, &s.SetName, &s.Trades, &s.Wins, &s.Losses, &s.TotalPnl)
			if s.Trades > 0 { s.WinRate = float64(s.Wins)/float64(s.Trades)*100 }
			setPerfs = append(setPerfs, s)
		}
	}
	if setPerfs == nil { setPerfs = []SetPerf{} }

	totalPnl := totalSignalPnl + totalCopyPnl
	weekPnl := weekSignalPnl + weekCopyPnl
	monthPnl := monthSignalPnl + monthCopyPnl

	c.JSON(200, gin.H{
		"ok": true,
		"overview": gin.H{
			"totalPnl":    totalPnl,
			"weekPnl":     weekPnl,
			"monthPnl":    monthPnl,
			"totalTrades": totalSignalTrades + totalCopyTrades,
			"activeSignalSubs":  activeSignalSubs,
			"activeCopyTrades":  activeCopyTrades,
		},
		"signals": gin.H{
			"totalPnl":    totalSignalPnl,
			"weekPnl":     weekSignalPnl,
			"monthPnl":    monthSignalPnl,
			"totalTrades": totalSignalTrades,
			"wins":        winSignals,
			"losses":      lossSignals,
			"breakdown":   setPerfs,
		},
		"copyTrades": gin.H{
			"totalPnl":    totalCopyPnl,
			"weekPnl":     weekCopyPnl,
			"monthPnl":    monthCopyPnl,
			"totalTrades": totalCopyTrades,
			"wins":        winCopies,
		},
	})
}

// GET /api/affiliate/overview — affiliate dashboard overview
func (h *Handler) GetAffiliateOverview(c *gin.Context) {
	uid, ok := getUID(c)
	if !ok { c.JSON(401, gin.H{"ok":false,"error":"unauthorized"}); return }

	// Get or create affiliate record
	var affID, code, tier string
	var totalReferrals, activeReferrals int
	var eumUSD float64

	err := h.service.repo.DB.QueryRow(`
		SELECT id::text, code, tier, total_referrals, active_referrals, eum_usd
		FROM affiliates WHERE user_id=$1::uuid`, uid).
		Scan(&affID, &code, &tier, &totalReferrals, &activeReferrals, &eumUSD)

	if err != nil {
		// Create affiliate record
		name := uid[:8]
		code = "CA-" + name
		h.service.repo.DB.QueryRow(`
			INSERT INTO affiliates (user_id, code, tier)
			VALUES ($1::uuid, $2, 'BRONZE')
			RETURNING id::text, code, tier, total_referrals, active_referrals, eum_usd`,
			uid, code).Scan(&affID, &code, &tier, &totalReferrals, &activeReferrals, &eumUSD)
	}

	// Total payouts
	var totalPayout, pendingPayout float64
	h.service.repo.DB.QueryRow(`
		SELECT COALESCE(SUM(amount_usd),0) FROM affiliate_payouts
		WHERE affiliate_id=$1::uuid AND status='PAID'`, affID).Scan(&totalPayout)
	h.service.repo.DB.QueryRow(`
		SELECT COALESCE(SUM(amount_usd),0) FROM affiliate_payouts
		WHERE affiliate_id=$1::uuid AND status='PENDING'`, affID).Scan(&pendingPayout)

	// Affiliate config from DB
	var affiliateMode float64
	var flatPct float64
	var customPct *float64
	h.service.repo.DB.QueryRow(`SELECT value FROM platform_fee_config WHERE key='affiliate_mode'`).Scan(&affiliateMode)
	h.service.repo.DB.QueryRow(`SELECT value FROM platform_fee_config WHERE key='affiliate_flat_pct'`).Scan(&flatPct)
	h.service.repo.DB.QueryRow(`SELECT custom_commission_pct FROM affiliates WHERE id=$1::uuid`, affID).Scan(&customPct)

	commissionPct := flatPct
	if customPct != nil {
		commissionPct = *customPct
	}

	// Tier info (used only when mode=tier)
	tierPct := map[string]float64{"BRONZE":3,"SILVER":5,"GOLD":7,"PLATINUM":10}
	nextTier := map[string]string{"BRONZE":"SILVER","SILVER":"GOLD","GOLD":"PLATINUM","PLATINUM":"PLATINUM"}
	nextReq := map[string]string{
		"BRONZE":"5 referrals + $5k EUM",
		"SILVER":"20 referrals + $50k EUM",
		"GOLD":"50 referrals + $150k EUM",
		"PLATINUM":"Top tier achieved",
	}
	_ = tierPct

	c.JSON(200, gin.H{
		"ok": true,
		"affiliate": gin.H{
			"id": affID, "code": code, "tier": tier,
			"tierPct": tierPct[tier],
			"nextTier": nextTier[tier],
			"nextTierReq": nextReq[tier],
			"totalReferrals": totalReferrals,
			"activeReferrals": activeReferrals,
			"eumUSD": eumUSD,
			"totalPayout": totalPayout,
			"pendingPayout": pendingPayout,
			"referralLink": "https://crunchalpha.com/register?ref=" + code,
			"commissionPct": commissionPct,
			"affiliateMode": affiliateMode,
			"isCustomCommission": customPct != nil,
		},
	})
}

// GET /api/affiliate/referrals
func (h *Handler) GetAffiliateReferrals(c *gin.Context) {
	uid, ok := getUID(c)
	if !ok { c.JSON(401, gin.H{"ok":false,"error":"unauthorized"}); return }

	var affID string
	h.service.repo.DB.QueryRow(`SELECT id::text FROM affiliates WHERE user_id=$1::uuid`, uid).Scan(&affID)
	if affID == "" { c.JSON(200, gin.H{"ok":true,"referrals":[]interface{}{}}); return }

	rows, err := h.service.repo.DB.Query(`
		SELECT r.referred_email, r.status, r.created_at::text,
		       COALESCE(u.primary_role,'') as role
		FROM affiliate_referrals r
		LEFT JOIN users u ON u.id = r.referred_user_id
		WHERE r.affiliate_id=$1::uuid
		ORDER BY r.created_at DESC LIMIT 100`, affID)
	if err != nil { c.JSON(500, gin.H{"ok":false,"error":err.Error()}); return }
	defer rows.Close()

	type ReferralRow struct {
		Email     string `json:"email"`
		Status    string `json:"status"`
		CreatedAt string `json:"createdAt"`
		Role      string `json:"role"`
	}
	var referrals []ReferralRow
	for rows.Next() {
		var r ReferralRow
		rows.Scan(&r.Email, &r.Status, &r.CreatedAt, &r.Role)
		// Anonymize email
		parts := strings.Split(r.Email, "@")
		if len(parts) == 2 && len(parts[0]) > 2 {
			r.Email = parts[0][:2] + "***@" + parts[1]
		}
		referrals = append(referrals, r)
	}
	if referrals == nil { referrals = []ReferralRow{} }
	c.JSON(200, gin.H{"ok":true,"referrals":referrals})
}

// GET /api/affiliate/payouts
func (h *Handler) GetAffiliatePayouts(c *gin.Context) {
	uid, ok := getUID(c)
	if !ok { c.JSON(401, gin.H{"ok":false,"error":"unauthorized"}); return }

	var affID string
	h.service.repo.DB.QueryRow(`SELECT id::text FROM affiliates WHERE user_id=$1::uuid`, uid).Scan(&affID)
	if affID == "" { c.JSON(200, gin.H{"ok":true,"payouts":[]interface{}{}}); return }

	rows, err := h.service.repo.DB.Query(`
		SELECT amount_usd, source, status, COALESCE(period,''), created_at::text
		FROM affiliate_payouts
		WHERE affiliate_id=$1::uuid
		ORDER BY created_at DESC LIMIT 50`, affID)
	if err != nil { c.JSON(500, gin.H{"ok":false,"error":err.Error()}); return }
	defer rows.Close()

	type PayoutRow struct {
		Amount    float64 `json:"amount"`
		Source    string  `json:"source"`
		Status    string  `json:"status"`
		Period    string  `json:"period"`
		CreatedAt string  `json:"createdAt"`
	}
	var payouts []PayoutRow
	for rows.Next() {
		var p PayoutRow
		rows.Scan(&p.Amount, &p.Source, &p.Status, &p.Period, &p.CreatedAt)
		payouts = append(payouts, p)
	}
	if payouts == nil { payouts = []PayoutRow{} }
	c.JSON(200, gin.H{"ok":true,"payouts":payouts})
}

// UpdateAffiliateTiers — called periodically or on-demand
// Calculates EUM and upgrades tiers
func (h *Handler) RecalcAffiliateTiers(c *gin.Context) {
	rows, err := h.service.repo.DB.Query(`SELECT id::text, user_id::text FROM affiliates`)
	if err != nil { c.JSON(500, gin.H{"ok":false,"error":err.Error()}); return }
	defer rows.Close()

	updated := 0
	for rows.Next() {
		var affID, userID string
		rows.Scan(&affID, &userID)

		// Count active referrals
		var activeReferrals int
		h.service.repo.DB.QueryRow(`
			SELECT COUNT(*) FROM affiliate_referrals 
			WHERE affiliate_id=$1::uuid AND status='ACTIVE'`, affID).Scan(&activeReferrals)

		// Calculate EUM — sum of equity from referred users' subscriptions + copy trades
		var eumUSD float64
		h.service.repo.DB.QueryRow(`
			SELECT COALESCE(SUM(es.equity),0)
			FROM equity_snapshots es
			JOIN trader_accounts ta ON ta.id = es.account_id
			JOIN affiliate_referrals ar ON ar.referred_user_id = ta.user_id
			WHERE ar.affiliate_id=$1::uuid AND ar.status='ACTIVE'
			AND es.created_at = (
				SELECT MAX(es2.created_at) FROM equity_snapshots es2 
				WHERE es2.account_id = es.account_id
			)`, affID).Scan(&eumUSD)

		// Determine tier
		tier := "BRONZE"
		switch {
		case eumUSD >= 150000 || (activeReferrals >= 50 && eumUSD >= 50000):
			tier = "PLATINUM"
		case activeReferrals >= 20 && eumUSD >= 50000:
			tier = "GOLD"
		case activeReferrals >= 5 && eumUSD >= 5000:
			tier = "SILVER"
		}

		h.service.repo.DB.Exec(`
			UPDATE affiliates SET tier=$1, active_referrals=$2, eum_usd=$3
			WHERE id=$4::uuid`, tier, activeReferrals, eumUSD, affID)
		updated++
	}

	c.JSON(200, gin.H{"ok":true,"updated":updated})
}

// POST /api/investor/affiliate/calculate-payout — calculate monthly payout
func (h *Handler) CalculateAffiliatePayout(c *gin.Context) {
	var req struct {
		Period string `json:"period"` // e.g. "2026-03"
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.Period == "" {
		c.JSON(400, gin.H{"ok":false,"error":"period required (e.g. 2026-03)"}); return
	}

	// Get all affiliates
	rows, err := h.service.repo.DB.Query(`SELECT id::text, tier FROM affiliates`)
	if err != nil { c.JSON(500, gin.H{"ok":false,"error":err.Error()}); return }
	defer rows.Close()

	tierPct := map[string]float64{"BRONZE":0.03,"SILVER":0.05,"GOLD":0.07,"PLATINUM":0.10}
	created := 0

	for rows.Next() {
		var affID, tier string
		rows.Scan(&affID, &tier)
		pct := tierPct[tier]

		// Platform revenue from this affiliate's referrals this period
		// = 20% of performance fees + subscription fees from referred users
		var perfFeeRevenue float64
		h.service.repo.DB.QueryRow(`
			SELECT COALESCE(SUM(net_profit * 0.20), 0)
			FROM investor_signal_executions ise
			JOIN affiliate_referrals ar ON ar.referred_user_id = ise.investor_id
			WHERE ar.affiliate_id=$1::uuid AND ar.status='ACTIVE'
			AND TO_CHAR(ise.close_time, 'YYYY-MM') = $2
			AND ise.net_profit > 0`, affID, req.Period).Scan(&perfFeeRevenue)

		var subFeeRevenue float64
		h.service.repo.DB.QueryRow(`
			SELECT COALESCE(COUNT(*) * 10 * 0.20, 0)
			FROM analyst_subscriptions sub
			JOIN affiliate_referrals ar ON ar.referred_user_id = sub.investor_id
			WHERE ar.affiliate_id=$1::uuid AND ar.status='ACTIVE'
			AND sub.status='ACTIVE'
			AND TO_CHAR(sub.started_at, 'YYYY-MM') <= $2`, affID, req.Period).Scan(&subFeeRevenue)

		totalRevenue := perfFeeRevenue + subFeeRevenue
		affiliatePayout := totalRevenue * pct

		if affiliatePayout > 0 {
			h.service.repo.DB.Exec(`
				INSERT INTO affiliate_payouts (affiliate_id, amount_usd, source, status, period)
				VALUES ($1::uuid, $2, 'revenue_share', 'PENDING', $3)`,
				affID, affiliatePayout, req.Period)
			created++
		}
	}

	c.JSON(200, gin.H{"ok":true,"period":req.Period,"payoutsCreated":created})
}
