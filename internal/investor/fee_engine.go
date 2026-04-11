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
