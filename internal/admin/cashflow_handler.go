package admin

import (
	"database/sql"
	"net/http"
	"github.com/gin-gonic/gin"
)

type CashflowHandler struct {
	DB *sql.DB
}

func NewCashflowHandler(db *sql.DB) *CashflowHandler {
	return &CashflowHandler{DB: db}
}

// GET /api/admin/cashflow/summary
func (h *CashflowHandler) Summary(c *gin.Context) {
	var totalSignalSubs, activeSignalSubs int
	var totalSignalRevenue, totalCopyRevenue float64

	h.DB.QueryRow(`SELECT COUNT(*), COUNT(*) FILTER (WHERE status='ACTIVE') FROM analyst_subscriptions`).Scan(&totalSignalSubs, &activeSignalSubs)
	h.DB.QueryRow(`SELECT COALESCE(SUM(subscription_fee),0) FROM analyst_subscriptions WHERE status='ACTIVE'`).Scan(&totalSignalRevenue)
	h.DB.QueryRow(`SELECT COUNT(*) FROM copy_subscriptions WHERE status='ACTIVE'`).Scan(&totalCopyRevenue)

	var weeklySignups, monthlySignups, yearlySignups, totalUsers int
	h.DB.QueryRow(`SELECT COUNT(*) FROM users WHERE created_at >= NOW() - INTERVAL '7 days'`).Scan(&weeklySignups)
	h.DB.QueryRow(`SELECT COUNT(*) FROM users WHERE created_at >= NOW() - INTERVAL '30 days'`).Scan(&monthlySignups)
	h.DB.QueryRow(`SELECT COUNT(*) FROM users WHERE created_at >= NOW() - INTERVAL '365 days'`).Scan(&yearlySignups)
	h.DB.QueryRow(`SELECT COUNT(*) FROM users`).Scan(&totalUsers)

	var totalAffiliatePaid, totalAffiliatePending float64
	h.DB.QueryRow(`SELECT COALESCE(SUM(amount_usd),0) FROM affiliate_payouts WHERE status='PAID'`).Scan(&totalAffiliatePaid)
	h.DB.QueryRow(`SELECT COALESCE(SUM(amount_usd),0) FROM affiliate_payouts WHERE status='PENDING'`).Scan(&totalAffiliatePending)

	c.JSON(http.StatusOK, gin.H{
		"signal_subscriptions": gin.H{
			"total":   totalSignalSubs,
			"active":  activeSignalSubs,
			"mrr_usd": totalSignalRevenue,
		},
		"copy_subscriptions": gin.H{
			"active": int(totalCopyRevenue),
		},
		"affiliate_payouts": gin.H{
			"paid_usd":    totalAffiliatePaid,
			"pending_usd": totalAffiliatePending,
		},
		"user_growth": gin.H{
			"total":   totalUsers,
			"weekly":  weeklySignups,
			"monthly": monthlySignups,
			"yearly":  yearlySignups,
		},
	})
}

// GET /api/admin/cashflow/signal-subscriptions
func (h *CashflowHandler) SignalSubscriptions(c *gin.Context) {
	rows, err := h.DB.Query(`
		SELECT s.id, s.subscription_fee, s.status, s.billing_cycle,
		       s.started_at, s.cancelled_at, s.expires_at,
		       inv.email, inv.name,
		       ana.email, ana.name,
		       ss.name
		FROM analyst_subscriptions s
		LEFT JOIN users inv ON inv.id = s.investor_id
		LEFT JOIN users ana ON ana.id = s.analyst_id
		LEFT JOIN analyst_signal_sets ss ON ss.id = s.set_id
		ORDER BY s.created_at DESC
	`)
	if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()}); return }
	defer rows.Close()

	type Row struct {
		ID              string   `json:"id"`
		Fee             float64  `json:"fee"`
		Status          string   `json:"status"`
		BillingCycle    string   `json:"billing_cycle"`
		StartedAt       string   `json:"started_at"`
		CancelledAt     *string  `json:"cancelled_at"`
		ExpiresAt       *string  `json:"expires_at"`
		InvestorEmail   *string  `json:"investor_email"`
		InvestorName    *string  `json:"investor_name"`
		AnalystEmail    *string  `json:"analyst_email"`
		AnalystName     *string  `json:"analyst_name"`
		SignalSetName   *string  `json:"signal_set_name"`
	}

	var subs []Row
	for rows.Next() {
		var r Row
		rows.Scan(&r.ID, &r.Fee, &r.Status, &r.BillingCycle,
			&r.StartedAt, &r.CancelledAt, &r.ExpiresAt,
			&r.InvestorEmail, &r.InvestorName,
			&r.AnalystEmail, &r.AnalystName, &r.SignalSetName)
		subs = append(subs, r)
	}
	if subs == nil { subs = []Row{} }
	c.JSON(http.StatusOK, gin.H{"data": subs})
}

// GET /api/admin/cashflow/copy-subscriptions
func (h *CashflowHandler) CopySubscriptions(c *gin.Context) {
	rows, err := h.DB.Query(`
		SELECT cs.id, cs.status, cs.lot_multiplier, cs.created_at,
		       ta_f.account_number, u_f.email, u_f.name,
		       ta_p.account_number, u_p.email, u_p.name
		FROM copy_subscriptions cs
		LEFT JOIN trader_accounts ta_f ON ta_f.id = cs.follower_account_id
		LEFT JOIN users u_f ON u_f.id = ta_f.user_id
		LEFT JOIN trader_accounts ta_p ON ta_p.id = cs.provider_account_id
		LEFT JOIN users u_p ON u_p.id = ta_p.user_id
		ORDER BY cs.created_at DESC
	`)
	if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()}); return }
	defer rows.Close()

	type Row struct {
		ID                    string  `json:"id"`
		Status                string  `json:"status"`
		LotMultiplier         float64 `json:"lot_multiplier"`
		CreatedAt             string  `json:"created_at"`
		FollowerAccount       *string `json:"follower_account"`
		FollowerEmail         *string `json:"follower_email"`
		FollowerName          *string `json:"follower_name"`
		ProviderAccount       *string `json:"provider_account"`
		ProviderEmail         *string `json:"provider_email"`
		ProviderName          *string `json:"provider_name"`
	}

	var subs []Row
	for rows.Next() {
		var r Row
		rows.Scan(&r.ID, &r.Status, &r.LotMultiplier, &r.CreatedAt,
			&r.FollowerAccount, &r.FollowerEmail, &r.FollowerName,
			&r.ProviderAccount, &r.ProviderEmail, &r.ProviderName)
		subs = append(subs, r)
	}
	if subs == nil { subs = []Row{} }
	c.JSON(http.StatusOK, gin.H{"data": subs})
}

// GET /api/admin/cashflow/user-growth
func (h *CashflowHandler) UserGrowth(c *gin.Context) {
	// Monthly signups last 12 months
	rows, err := h.DB.Query(`
		SELECT TO_CHAR(DATE_TRUNC('month', created_at), 'YYYY-MM') as month,
		       COUNT(*) as signups,
		       COUNT(*) FILTER (WHERE primary_role='trader') as traders,
		       COUNT(*) FILTER (WHERE primary_role='analyst') as analysts,
		       COUNT(*) FILTER (WHERE primary_role='investor') as investors
		FROM users
		WHERE created_at >= NOW() - INTERVAL '12 months'
		GROUP BY DATE_TRUNC('month', created_at)
		ORDER BY DATE_TRUNC('month', created_at) ASC
	`)
	if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()}); return }
	defer rows.Close()

	type MonthRow struct {
		Month     string `json:"month"`
		Signups   int    `json:"signups"`
		Traders   int    `json:"traders"`
		Analysts  int    `json:"analysts"`
		Investors int    `json:"investors"`
	}

	var monthly []MonthRow
	for rows.Next() {
		var r MonthRow
		rows.Scan(&r.Month, &r.Signups, &r.Traders, &r.Analysts, &r.Investors)
		monthly = append(monthly, r)
	}
	if monthly == nil { monthly = []MonthRow{} }

	// Weekly signups last 8 weeks
	rows2, err := h.DB.Query(`
		SELECT TO_CHAR(DATE_TRUNC('week', created_at), 'YYYY-MM-DD') as week,
		       COUNT(*) as signups
		FROM users
		WHERE created_at >= NOW() - INTERVAL '8 weeks'
		GROUP BY DATE_TRUNC('week', created_at)
		ORDER BY DATE_TRUNC('week', created_at) ASC
	`)
	if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()}); return }
	defer rows2.Close()

	type WeekRow struct {
		Week    string `json:"week"`
		Signups int    `json:"signups"`
	}
	var weekly []WeekRow
	for rows2.Next() {
		var r WeekRow
		rows2.Scan(&r.Week, &r.Signups)
		weekly = append(weekly, r)
	}
	if weekly == nil { weekly = []WeekRow{} }

	c.JSON(http.StatusOK, gin.H{
		"monthly": monthly,
		"weekly":  weekly,
	})
}
