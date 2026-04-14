package analyst

import (
"database/sql"
"net/http"
"time"

"github.com/gin-gonic/gin"
)

type AnalystEarningsSummary struct {
TotalSubscribers    int                   `json:"total_subscribers"`
ActiveSubscriptions int                   `json:"active_subscriptions"`
PendingEarnings     float64               `json:"pending_earnings"`
PaidEarnings        float64               `json:"paid_earnings"`
TotalEarned         float64               `json:"total_earned"`
PendingWithdrawal   float64               `json:"pending_withdrawal"`
PerSubscriber       []SubscriberEarnings  `json:"per_subscriber"`
MonthlyChart        []AnalystMonthly      `json:"monthly_chart"`
}

type SubscriberEarnings struct {
InvestorName string  `json:"investor_name"`
SignalSetName string  `json:"signal_set_name"`
SubFee       float64 `json:"subscription_fee"`
AnalystShare float64 `json:"analyst_share"`
Status       string  `json:"status"`
StartedAt    string  `json:"started_at"`
AutoFollow   bool    `json:"auto_follow"`
}

type AnalystMonthly struct {
Month        string  `json:"month"`
AnalystShare float64 `json:"analyst_share"`
}

type AnalystWithdrawRequest struct {
Amount float64 `json:"amount" binding:"required,gt=0"`
Method string  `json:"method"`
Notes  string  `json:"notes"`
}

// ─── GET /api/analyst/earnings ───────────────────────────────────────────────

func (h *Handler) GetEarnings(c *gin.Context) {
userID, exists := c.Get("user_id")
if !exists {
c.JSON(http.StatusUnauthorized, gin.H{"ok": false, "error": "unauthorized"})
return
}
uid := userID.(string)

db := h.DB
var summary AnalystEarningsSummary
summary.PerSubscriber = []SubscriberEarnings{}
summary.MonthlyChart = []AnalystMonthly{}

// Cek platform fee config — analyst share pct (default 70%)
var analystSharePct float64 = 70.0
db.QueryRow(`
SELECT COALESCE(value::numeric, 70)
FROM platform_fee_config
WHERE key = 'analyst_share_pct'
LIMIT 1
`).Scan(&analystSharePct)

// Active subscriptions count
db.QueryRow(`
SELECT COUNT(*)
FROM analyst_subscriptions asub
JOIN analyst_signal_sets ass ON ass.id = asub.set_id
WHERE ass.analyst_id = $1
  AND asub.status = 'active'
`, uid).Scan(&summary.ActiveSubscriptions)

// Total subscribers (termasuk cancelled)
db.QueryRow(`
SELECT COUNT(*)
FROM analyst_subscriptions asub
JOIN analyst_signal_sets ass ON ass.id = asub.set_id
WHERE ass.analyst_id = $1
`, uid).Scan(&summary.TotalSubscribers)

// Pending earnings — dari total_fee_collected yang belum di-withdraw
db.QueryRow(`
SELECT COALESCE(SUM(asub.total_fee_collected * $2 / 100.0), 0)
FROM analyst_subscriptions asub
JOIN analyst_signal_sets ass ON ass.id = asub.set_id
WHERE ass.analyst_id = $1
`, uid, analystSharePct).Scan(&summary.PendingEarnings)

// Already withdrawn (paid)
db.QueryRow(`
SELECT COALESCE(SUM(amount), 0)
FROM earnings_withdrawals
WHERE user_id = $1::uuid AND role = 'analyst' AND status = 'paid'
`, uid).Scan(&summary.PaidEarnings)

// Pending withdrawal amount
db.QueryRow(`
SELECT COALESCE(SUM(amount), 0)
FROM earnings_withdrawals
WHERE user_id = $1::uuid AND role = 'analyst' AND status = 'pending'
`, uid).Scan(&summary.PendingWithdrawal)

summary.TotalEarned = summary.PendingEarnings + summary.PaidEarnings

// Per-subscriber breakdown
rows, err := db.Query(`
SELECT
COALESCE(u.name, u.email, 'Investor') AS investor_name,
COALESCE(ass.name, 'Signal Set') AS signal_set_name,
COALESCE(asub.subscription_fee, 0) AS subscription_fee,
COALESCE(asub.subscription_fee * $2 / 100.0, 0) AS analyst_share,
COALESCE(asub.status, 'active') AS status,
COALESCE(TO_CHAR(asub.started_at, 'YYYY-MM-DD'), '') AS started_at,
COALESCE(asub.auto_follow, false) AS auto_follow
FROM analyst_subscriptions asub
JOIN analyst_signal_sets ass ON ass.id = asub.set_id
JOIN users u ON u.id = asub.investor_id
WHERE ass.analyst_id = $1
ORDER BY asub.created_at DESC
LIMIT 100
`, uid, analystSharePct)
if err == nil && rows != nil {
defer rows.Close()
for rows.Next() {
var s SubscriberEarnings
rows.Scan(
&s.InvestorName, &s.SignalSetName,
&s.SubFee, &s.AnalystShare,
&s.Status, &s.StartedAt, &s.AutoFollow,
)
summary.PerSubscriber = append(summary.PerSubscriber, s)
}
}

// Monthly chart — group by bulan dari created_at subscription
chartRows, err := db.Query(`
SELECT
TO_CHAR(DATE_TRUNC('month', asub.created_at), 'YYYY-MM') AS month,
COALESCE(SUM(asub.subscription_fee * $2 / 100.0), 0) AS analyst_share
FROM analyst_subscriptions asub
JOIN analyst_signal_sets ass ON ass.id = asub.set_id
WHERE ass.analyst_id = $1
  AND asub.created_at >= NOW() - INTERVAL '12 months'
GROUP BY DATE_TRUNC('month', asub.created_at)
ORDER BY DATE_TRUNC('month', asub.created_at) ASC
`, uid, analystSharePct)
if err == nil && chartRows != nil {
defer chartRows.Close()
for chartRows.Next() {
var m AnalystMonthly
chartRows.Scan(&m.Month, &m.AnalystShare)
summary.MonthlyChart = append(summary.MonthlyChart, m)
}
}

c.JSON(http.StatusOK, gin.H{"ok": true, "data": summary})
}

// ─── POST /api/analyst/earnings/withdraw ─────────────────────────────────────

func (h *Handler) RequestWithdraw(c *gin.Context) {
userID, exists := c.Get("user_id")
if !exists {
c.JSON(http.StatusUnauthorized, gin.H{"ok": false, "error": "unauthorized"})
return
}
uid := userID.(string)

var req AnalystWithdrawRequest
if err := c.ShouldBindJSON(&req); err != nil {
c.JSON(http.StatusBadRequest, gin.H{"ok": false, "error": "amount required"})
return
}
if req.Method == "" {
req.Method = "bank_transfer"
}

// Available = pending earnings - pending withdrawal
var available float64
var analystSharePct float64 = 70.0
h.DB.QueryRow(`SELECT COALESCE(value::numeric,70) FROM platform_fee_config WHERE key='analyst_share_pct' LIMIT 1`).Scan(&analystSharePct)

h.DB.QueryRow(`
SELECT
COALESCE((
SELECT SUM(asub.subscription_fee * $2 / 100.0)
FROM analyst_subscriptions asub
JOIN analyst_signal_sets ass ON ass.id = asub.set_id
WHERE ass.analyst_id = $1
), 0)
-
COALESCE((
SELECT SUM(amount)
FROM earnings_withdrawals
WHERE user_id = $1::uuid AND role = 'analyst' AND status IN ('pending','approved')
), 0)
`, uid, analystSharePct).Scan(&available)

if req.Amount > available {
c.JSON(http.StatusBadRequest, gin.H{
"ok":        false,
"error":     "Insufficient available balance",
"available": available,
})
return
}

var withdrawID string
err := h.DB.QueryRow(`
INSERT INTO earnings_withdrawals (user_id, role, amount, method, notes, status, requested_at)
VALUES ($1::uuid, 'analyst', $2, $3, $4, 'pending', $5)
RETURNING id
`, uid, req.Amount, req.Method, req.Notes, time.Now()).Scan(&withdrawID)

if err != nil {
c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "error": "Failed to submit request"})
return
}

c.JSON(http.StatusOK, gin.H{
"ok":      true,
"message": "Withdrawal request submitted",
"id":      withdrawID,
})
}

// ─── GET /api/analyst/earnings/withdrawals ───────────────────────────────────

func (h *Handler) GetWithdrawals(c *gin.Context) {
userID, exists := c.Get("user_id")
if !exists {
c.JSON(http.StatusUnauthorized, gin.H{"ok": false, "error": "unauthorized"})
return
}
uid := userID.(string)

rows, err := h.DB.Query(`
SELECT id, amount, method, notes, status,
       TO_CHAR(requested_at, 'YYYY-MM-DD HH24:MI') AS requested_at,
       TO_CHAR(processed_at, 'YYYY-MM-DD HH24:MI') AS processed_at
FROM earnings_withdrawals
WHERE user_id = $1::uuid AND role = 'analyst'
ORDER BY requested_at DESC
LIMIT 50
`, uid)
if err != nil {
c.JSON(http.StatusOK, gin.H{"ok": true, "withdrawals": []interface{}{}})
return
}
defer rows.Close()

type WRow struct {
ID          string  `json:"id"`
Amount      float64 `json:"amount"`
Method      string  `json:"method"`
Notes       string  `json:"notes"`
Status      string  `json:"status"`
RequestedAt string  `json:"requested_at"`
ProcessedAt *string `json:"processed_at"`
}
var list []WRow
for rows.Next() {
var w WRow
var processedAt sql.NullString
rows.Scan(&w.ID, &w.Amount, &w.Method, &w.Notes, &w.Status, &w.RequestedAt, &processedAt)
if processedAt.Valid {
w.ProcessedAt = &processedAt.String
}
list = append(list, w)
}
if list == nil {
list = []WRow{}
}
c.JSON(http.StatusOK, gin.H{"ok": true, "withdrawals": list})
}
