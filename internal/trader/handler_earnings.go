package trader

import (
"database/sql"
"net/http"
"time"

"github.com/gin-gonic/gin"
)

// ─── Response structs ───────────────────────────────────────────────────────

type EarningsSummary struct {
PendingEarnings   float64            `json:"pending_earnings"`
PaidEarnings      float64            `json:"paid_earnings"`
TotalEarned       float64            `json:"total_earned"`
ActiveInvestors   int                `json:"active_investors"`
PerInvestor       []InvestorEarnings `json:"per_investor"`
MonthlyChart      []MonthlyEarning   `json:"monthly_chart"`
PendingWithdrawal float64            `json:"pending_withdrawal"`
}

type InvestorEarnings struct {
InvestorName  string  `json:"investor_name"`
TraderAccount string  `json:"trader_account"`
AumUSD        float64 `json:"aum_usd"`
AccruedFee    float64 `json:"accrued_fee"`
TraderShare   float64 `json:"trader_share"`
Status        string  `json:"status"`
PeriodStart   string  `json:"period_start"`
}

type MonthlyEarning struct {
Month       string  `json:"month"`
TraderShare float64 `json:"trader_share"`
}

type WithdrawRequest struct {
Amount float64 `json:"amount" binding:"required,gt=0"`
Method string  `json:"method"`
Notes  string  `json:"notes"`
}

// ─── GET /api/trader/earnings ────────────────────────────────────────────────

func (h *Handler) GetEarnings(c *gin.Context) {
userID, exists := c.Get("user_id")
if !exists {
c.JSON(http.StatusUnauthorized, gin.H{"ok": false, "error": "unauthorized"})
return
}
uid := userID.(string)

var summary EarningsSummary
summary.PerInvestor = []InvestorEarnings{}
summary.MonthlyChart = []MonthlyEarning{}

db := h.service.repo.db

// Pending earnings — dari fee_periods belum closed, trader share dari transactions
db.QueryRow(`
SELECT COALESCE(SUM(ft.trader_share), 0)
FROM investor_fee_transactions ft
JOIN trader_accounts ta ON ta.id = ft.trader_account_id
WHERE ta.user_id = $1::uuid AND ft.status = 'pending'
`, uid).Scan(&summary.PendingEarnings)

// Paid earnings
db.QueryRow(`
SELECT COALESCE(SUM(ft.trader_share), 0)
FROM investor_fee_transactions ft
JOIN trader_accounts ta ON ta.id = ft.trader_account_id
WHERE ta.user_id = $1::uuid AND ft.status = 'paid'
`, uid).Scan(&summary.PaidEarnings)

summary.TotalEarned = summary.PendingEarnings + summary.PaidEarnings

// Active investors count
db.QueryRow(`
SELECT COUNT(DISTINCT fp.user_id)
FROM investor_fee_periods fp
JOIN trader_accounts ta ON ta.id = fp.trader_account_id
WHERE ta.user_id = $1::uuid AND fp.is_closed = false
`, uid).Scan(&summary.ActiveInvestors)

// Pending withdrawal amount
db.QueryRow(`
SELECT COALESCE(SUM(amount), 0)
FROM earnings_withdrawals
WHERE user_id = $1::uuid AND role = 'trader' AND status = 'pending'
`, uid).Scan(&summary.PendingWithdrawal)

// Per-investor breakdown — dari fee_periods (sumber kebenaran)
rows, err := db.Query(`
SELECT
COALESCE(u.name, u.email, 'Investor') AS investor_name,
COALESCE(ta.nickname, ta.account_number, 'Account') AS trader_account,
COALESCE(fp.aum_usd, 0) AS aum_usd,
COALESCE(fp.accrued_fee, 0) AS accrued_fee,
COALESCE(ft.trader_share, 0) AS trader_share,
COALESCE(ft.status, 'accruing') AS status,
TO_CHAR(fp.period_start, 'YYYY-MM-DD') AS period_start
FROM investor_fee_periods fp
JOIN trader_accounts ta ON ta.id = fp.trader_account_id
JOIN users u ON u.id = fp.user_id
LEFT JOIN investor_fee_transactions ft ON ft.fee_period_id = fp.id
WHERE ta.user_id = $1::uuid
ORDER BY fp.created_at DESC
LIMIT 100
`, uid)
if err == nil && rows != nil {
defer rows.Close()
for rows.Next() {
var inv InvestorEarnings
rows.Scan(
&inv.InvestorName, &inv.TraderAccount,
&inv.AumUSD, &inv.AccruedFee, &inv.TraderShare,
&inv.Status, &inv.PeriodStart,
)
summary.PerInvestor = append(summary.PerInvestor, inv)
}
}

// Monthly chart — 12 bulan terakhir
chartRows, err := db.Query(`
SELECT
TO_CHAR(DATE_TRUNC('month', ft.created_at), 'YYYY-MM') AS month,
COALESCE(SUM(ft.trader_share), 0) AS trader_share
FROM investor_fee_transactions ft
JOIN trader_accounts ta ON ta.id = ft.trader_account_id
WHERE ta.user_id = $1::uuid
  AND ft.created_at >= NOW() - INTERVAL '12 months'
GROUP BY DATE_TRUNC('month', ft.created_at)
ORDER BY DATE_TRUNC('month', ft.created_at) ASC
`, uid)
if err == nil && chartRows != nil {
defer chartRows.Close()
for chartRows.Next() {
var m MonthlyEarning
chartRows.Scan(&m.Month, &m.TraderShare)
summary.MonthlyChart = append(summary.MonthlyChart, m)
}
}

c.JSON(http.StatusOK, gin.H{"ok": true, "data": summary})
}

// ─── POST /api/trader/earnings/withdraw ──────────────────────────────────────

func (h *Handler) RequestWithdraw(c *gin.Context) {
userID, exists := c.Get("user_id")
if !exists {
c.JSON(http.StatusUnauthorized, gin.H{"ok": false, "error": "unauthorized"})
return
}
uid := userID.(string)

var req WithdrawRequest
if err := c.ShouldBindJSON(&req); err != nil {
c.JSON(http.StatusBadRequest, gin.H{"ok": false, "error": "amount required"})
return
}

if req.Method == "" {
req.Method = "bank_transfer"
}

// Cek available balance (pending earnings - pending withdrawal)
var available float64
h.service.repo.db.QueryRow(`
SELECT
COALESCE((
SELECT SUM(ft.trader_share)
FROM investor_fee_transactions ft
JOIN trader_accounts ta ON ta.id = ft.trader_account_id
WHERE ta.user_id = $1::uuid AND ft.status = 'pending'
), 0)
-
COALESCE((
SELECT SUM(amount)
FROM earnings_withdrawals
WHERE user_id = $1::uuid AND role = 'trader' AND status = 'pending'
), 0)
`, uid).Scan(&available)

if req.Amount > available {
c.JSON(http.StatusBadRequest, gin.H{
"ok":        false,
"error":     "Insufficient available balance",
"available": available,
})
return
}

var withdrawID string
err := h.service.repo.db.QueryRow(`
INSERT INTO earnings_withdrawals (user_id, role, amount, method, notes, status, requested_at)
VALUES ($1::uuid, 'trader', $2, $3, $4, 'pending', $5)
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

// ─── GET /api/trader/earnings/withdrawals ────────────────────────────────────

func (h *Handler) GetWithdrawals(c *gin.Context) {
userID, exists := c.Get("user_id")
if !exists {
c.JSON(http.StatusUnauthorized, gin.H{"ok": false, "error": "unauthorized"})
return
}
uid := userID.(string)

rows, err := h.service.repo.db.Query(`
SELECT id, amount, method, notes, status,
       TO_CHAR(requested_at, 'YYYY-MM-DD HH24:MI') AS requested_at,
       TO_CHAR(processed_at, 'YYYY-MM-DD HH24:MI') AS processed_at
FROM earnings_withdrawals
WHERE user_id = $1::uuid AND role = 'trader'
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
