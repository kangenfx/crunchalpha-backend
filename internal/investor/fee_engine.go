package investor

import (
"database/sql"
"fmt"
"log"
"time"
)

// FeeEngine handles performance fee calculation, invoicing, and enforcement
type FeeEngine struct {
db *sql.DB
}

func NewFeeEngine(db *sql.DB) *FeeEngine {
return &FeeEngine{db: db}
}

// UpdateEquityAndHWM — called every time EA pushes equity for investor
func (fe *FeeEngine) UpdateEquityAndHWM(investorID string, equityCurrent float64) error {
rows, err := fe.db.Query(`
SELECT id, trader_account_id, analyst_set_id,
       hwm_start, hwm_end, fee_pct, aum_usd
FROM investor_fee_periods
WHERE user_id = $1 AND is_closed = false
`, investorID)
if err != nil {
return fmt.Errorf("fee periods query: %w", err)
}
defer rows.Close()

for rows.Next() {
var (
periodID        string
traderAccountID sql.NullString
analystSetID    sql.NullString
hwmStart        float64
hwmEnd          sql.NullFloat64
feePct          float64
aumUSD          float64
)
if err := rows.Scan(&periodID, &traderAccountID, &analystSetID,
&hwmStart, &hwmEnd, &feePct, &aumUSD); err != nil {
continue
}

// Current HWM is max of hwm_start and hwm_end
currentHWM := hwmStart
if hwmEnd.Valid && hwmEnd.Float64 > currentHWM {
currentHWM = hwmEnd.Float64
}

// Equity for this period
equityForPeriod := equityCurrent

// Update HWM if equity increased
newHWM := currentHWM
if equityForPeriod > currentHWM {
newHWM = equityForPeriod
}

// Calculate accrued profit and fee
accruedProfit := 0.0
if equityForPeriod > newHWM {
accruedProfit = equityForPeriod - newHWM
}
accruedFee := accruedProfit * feePct / 100.0

// Update DB
_, err := fe.db.Exec(`
UPDATE investor_fee_periods
SET hwm_end = $1, accrued_fee = $2, updated_at = NOW()
WHERE id = $3
`, newHWM, accruedFee, periodID)
if err != nil {
log.Printf("[FeeEngine] update hwm error period %s: %v", periodID, err)
}
}
return nil
}

// ClosePeriodAndInvoice — close fee period and create invoice
func (fe *FeeEngine) ClosePeriodAndInvoice(periodID string) error {
var accruedFee float64
var userID, traderAccountID string
err := fe.db.QueryRow(`
SELECT user_id, COALESCE(trader_account_id::text,''), COALESCE(accrued_fee, 0)
FROM investor_fee_periods
WHERE id = $1
`, periodID).Scan(&userID, &traderAccountID, &accruedFee)
if err != nil {
return fmt.Errorf("get period: %w", err)
}

if accruedFee <= 0 {
return nil
}

// Create invoice
_, err = fe.db.Exec(`
INSERT INTO fee_invoices (user_id, trader_account_id, amount, status, created_at)
VALUES ($1, $2, $3, 'PENDING', NOW())
`, userID, traderAccountID, accruedFee)
if err != nil {
return fmt.Errorf("create invoice: %w", err)
}

// Close period
_, err = fe.db.Exec(`
UPDATE investor_fee_periods
SET is_closed = true, closed_at = $1, updated_at = NOW()
WHERE id = $2
`, time.Now(), periodID)
return err
}

// GetAccruedFees — get total accrued fees for investor
func (fe *FeeEngine) GetAccruedFees(investorID string) (float64, error) {
var total float64
err := fe.db.QueryRow(`
SELECT COALESCE(SUM(accrued_fee), 0)
FROM investor_fee_periods
WHERE user_id = $1 AND is_closed = false
`, investorID).Scan(&total)
return total, err
}

func (fe *FeeEngine) OpenPeriod(investorID, traderAccountID string, aumUSD, equityStart, hwmStart, feePct float64) error {
_, err := fe.db.Exec(`
INSERT INTO investor_fee_periods
(user_id, trader_account_id, aum_usd, equity_start, equity_current,
 hwm_start, hwm_end, accrued_profit, accrued_fee, fee_pct,
 performance_fee_pct, period_start, is_closed, fee_status, period_type)
VALUES ($1, $2::uuid, $3, $4, $4, $5, $5, 0, 0, $6, $6, now(), false, 'pending', 'normal')
ON CONFLICT DO NOTHING`,
investorID, traderAccountID, aumUSD, equityStart, hwmStart, feePct)
return err
}

func (fe *FeeEngine) MarkInvoicePaid(invoiceID string) error {
var investorID string
err := fe.db.QueryRow(`
UPDATE investor_fee_invoices
SET status='paid', paid_at=now(), updated_at=now()
WHERE id=$1 RETURNING investor_id`, invoiceID).Scan(&investorID)
if err != nil { return fmt.Errorf("mark paid: %w", err) }
fe.db.Exec(`UPDATE investor_restrictions SET active=false, resolved_at=now()
WHERE investor_id=$1 AND active=true`, investorID)
return nil
}

func (fe *FeeEngine) GetInvestorFeeStatus(investorID string) (map[string]interface{}, error) {
type Period struct {
ID            string  `json:"id"`
TraderName    string  `json:"traderName"`
AUMUSD        float64 `json:"aumUsd"`
HWM           float64 `json:"hwm"`
EquityCurrent float64 `json:"equityCurrent"`
AccruedProfit float64 `json:"accruedProfit"`
AccruedFee    float64 `json:"accruedFee"`
FeePct        float64 `json:"feePct"`
}
rows, err := fe.db.Query(`
SELECT fp.id,
       COALESCE(ta.nickname, ta.account_number, 'Unknown') as trader_name,
       fp.aum_usd, COALESCE(fp.hwm_end, fp.hwm_start) as hwm,
       COALESCE(fp.equity_current, 0),
       COALESCE(fp.accrued_profit, 0),
       COALESCE(fp.accrued_fee, 0),
       COALESCE(fp.fee_pct, 20)
FROM investor_fee_periods fp
LEFT JOIN trader_accounts ta ON ta.id = fp.trader_account_id
WHERE fp.user_id = $1 AND fp.is_closed = false
ORDER BY fp.created_at DESC`, investorID)
if err != nil { return nil, err }
defer rows.Close()
periods := []Period{}
totalAccruedFee := 0.0
for rows.Next() {
var p Period
rows.Scan(&p.ID, &p.TraderName, &p.AUMUSD, &p.HWM,
&p.EquityCurrent, &p.AccruedProfit, &p.AccruedFee, &p.FeePct)
totalAccruedFee += p.AccruedFee
periods = append(periods, p)
}

type Invoice struct {
ID       string  `json:"id"`
Amount   float64 `json:"amount"`
DueDate  string  `json:"dueDate"`
Status   string  `json:"status"`
DaysLeft int     `json:"daysLeft"`
}
invRows, _ := fe.db.Query(`
SELECT id, amount, due_date::text, status,
       GREATEST(0, EXTRACT(DAY FROM due_date - now())::int) as days_left
FROM investor_fee_invoices
WHERE investor_id=$1 AND status IN ('pending','overdue')
ORDER BY due_date ASC`, investorID)
invoices := []Invoice{}
totalOutstanding := 0.0
if invRows != nil {
defer invRows.Close()
for invRows.Next() {
var inv Invoice
invRows.Scan(&inv.ID, &inv.Amount, &inv.DueDate, &inv.Status, &inv.DaysLeft)
totalOutstanding += inv.Amount
invoices = append(invoices, inv)
}
}

type Restriction struct {
Type   string `json:"type"`
Reason string `json:"reason"`
}
restRows, _ := fe.db.Query(`
SELECT restriction_type, reason FROM investor_restrictions
WHERE investor_id=$1 AND active=true`, investorID)
restrictions := []Restriction{}
if restRows != nil {
defer restRows.Close()
for restRows.Next() {
var r Restriction
restRows.Scan(&r.Type, &r.Reason)
restrictions = append(restrictions, r)
}
}

return map[string]interface{}{
"activePeriods":    periods,
"totalAccruedFee":  totalAccruedFee,
"invoices":         invoices,
"totalOutstanding": totalOutstanding,
"restrictions":     restrictions,
"hasRestrictions":  len(restrictions) > 0,
}, nil
}

func (fe *FeeEngine) ApplyOverdueRestrictions() {
rows, err := fe.db.Query(`
SELECT id, investor_id FROM investor_fee_invoices
WHERE status='pending' AND due_date < now() - interval '1 day'`)
if err == nil {
defer rows.Close()
for rows.Next() {
var invoiceID, investorID string
if rows.Scan(&invoiceID, &investorID) == nil {
fe.db.Exec(`UPDATE investor_fee_invoices SET status='overdue' WHERE id=$1`, invoiceID)
fe.applyRestriction(investorID, "risk_reduced", "Invoice overdue", invoiceID)
}
}
}
}

func (fe *FeeEngine) applyRestriction(investorID, rType, reason, invoiceID string) {
var count int
fe.db.QueryRow(`SELECT COUNT(*) FROM investor_restrictions
WHERE investor_id=$1 AND restriction_type=$2 AND active=true`,
investorID, rType).Scan(&count)
if count > 0 { return }
fe.db.Exec(`INSERT INTO investor_restrictions
(investor_id, restriction_type, reason, invoice_id, active)
VALUES ($1,$2,$3,$4,true)`,
investorID, rType, reason, invoiceID)
}

func (fe *FeeEngine) MonthlyClosePeriods() {
rows, err := fe.db.Query(`
SELECT id FROM investor_fee_periods
WHERE is_closed=false AND period_start < date_trunc('month', now())`)
if err != nil { return }
defer rows.Close()
for rows.Next() {
var periodID string
if rows.Scan(&periodID) == nil {
fe.ClosePeriodAndInvoice(periodID)
}
}
}
