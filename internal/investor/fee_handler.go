package investor

import (
"github.com/gin-gonic/gin"
)

// GET /api/investor/fee-status
func (h *Handler) GetFeeStatus(c *gin.Context) {
uid, ok := getUID(c)
if !ok { c.JSON(401, gin.H{"ok":false,"error":"unauthorized"}); return }

fe := NewFeeEngine(h.service.repo.DB)
status, err := fe.GetInvestorFeeStatus(uid)
if err != nil { c.JSON(500, gin.H{"ok":false,"error":err.Error()}); return }

c.JSON(200, gin.H{"ok":true, "data": status})
}

// GET /api/investor/invoices
func (h *Handler) GetInvoices(c *gin.Context) {
uid, ok := getUID(c)
if !ok { c.JSON(401, gin.H{"ok":false,"error":"unauthorized"}); return }

rows, err := h.service.repo.DB.Query(`
SELECT i.id, i.amount, i.fee_pct, i.accrued_profit,
       i.due_date, i.status, i.paid_at, i.created_at,
       COALESCE(ta.nickname, ta.account_number, 'Unknown') as trader_name,
       EXTRACT(DAY FROM i.due_date - now())::int as days_left
FROM investor_fee_invoices i
LEFT JOIN trader_accounts ta ON ta.id = i.trader_account_id
WHERE i.investor_id = $1
ORDER BY i.created_at DESC
LIMIT 50`, uid)
if err != nil { c.JSON(500, gin.H{"ok":false,"error":err.Error()}); return }
defer rows.Close()

type InvoiceRow struct {
ID            string   `json:"id"`
Amount        float64  `json:"amount"`
FeePct        float64  `json:"feePct"`
AccruedProfit float64  `json:"accruedProfit"`
DueDate       string   `json:"dueDate"`
Status        string   `json:"status"`
PaidAt        *string  `json:"paidAt"`
CreatedAt     string   `json:"createdAt"`
TraderName    string   `json:"traderName"`
DaysLeft      int      `json:"daysLeft"`
}

invoices := []InvoiceRow{}
for rows.Next() {
var inv InvoiceRow
var paidAt *string
rows.Scan(&inv.ID, &inv.Amount, &inv.FeePct, &inv.AccruedProfit,
&inv.DueDate, &inv.Status, &paidAt, &inv.CreatedAt,
&inv.TraderName, &inv.DaysLeft)
inv.PaidAt = paidAt
invoices = append(invoices, inv)
}
c.JSON(200, gin.H{"ok":true,"invoices":invoices})
}

// POST /api/admin/invoices/:id/mark-paid
func (h *Handler) AdminMarkInvoicePaid(c *gin.Context) {
invoiceID := c.Param("id")
fe := NewFeeEngine(h.service.repo.DB)
if err := fe.MarkInvoicePaid(invoiceID); err != nil {
c.JSON(500, gin.H{"ok":false,"error":err.Error()}); return
}
c.JSON(200, gin.H{"ok":true,"message":"Invoice marked as paid"})
}

// GET /api/trader/fee-earnings
func (h *Handler) GetTraderFeeEarnings(c *gin.Context) {
uid, ok := getUID(c)
if !ok { c.JSON(401, gin.H{"ok":false,"error":"unauthorized"}); return }

// Get all invoices for trader's accounts
rows, err := h.service.repo.DB.Query(`
SELECT i.id, i.amount, i.fee_pct, i.accrued_profit,
       i.due_date, i.status, i.paid_at, i.created_at,
       COALESCE(u.name, u.email) as investor_name
FROM investor_fee_invoices i
JOIN users u ON u.id = i.investor_id
WHERE i.trader_account_id IN (
SELECT id FROM trader_accounts WHERE user_id = $1
)
ORDER BY i.created_at DESC
LIMIT 100`, uid)
if err != nil { c.JSON(500, gin.H{"ok":false,"error":err.Error()}); return }
defer rows.Close()

type EarningRow struct {
ID            string  `json:"id"`
Amount        float64 `json:"amount"`
FeePct        float64 `json:"feePct"`
AccruedProfit float64 `json:"accruedProfit"`
DueDate       string  `json:"dueDate"`
Status        string  `json:"status"`
PaidAt        *string `json:"paidAt"`
CreatedAt     string  `json:"createdAt"`
InvestorName  string  `json:"investorName"`
}

earnings := []EarningRow{}
totalPending := 0.0
totalPaid    := 0.0

for rows.Next() {
var e EarningRow
var paidAt *string
rows.Scan(&e.ID, &e.Amount, &e.FeePct, &e.AccruedProfit,
&e.DueDate, &e.Status, &paidAt, &e.CreatedAt, &e.InvestorName)
e.PaidAt = paidAt
if e.Status == "paid" { totalPaid += e.Amount } else { totalPending += e.Amount }
earnings = append(earnings, e)
}

c.JSON(200, gin.H{
"ok": true,
"earnings": earnings,
"summary": gin.H{
"totalPending": totalPending,
"totalPaid":    totalPaid,
"total":        totalPending + totalPaid,
},
})
}
