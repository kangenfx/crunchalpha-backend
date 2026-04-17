package alpharank

import "log"

// UpdateDrawdownMetrics dipanggil setiap EA push
func (s *Service) UpdateDrawdownMetrics(accountID string, equity, totalWithdrawals float64) error {
normalizedEquity := equity + totalWithdrawals

// Hitung true max DD sequential (peak-to-trough) dari trades
var trueMaxDD float64
var historicalPeak float64

err := s.db.QueryRow(`
WITH initial_deposit AS (
SELECT COALESCE(SUM(amount), 0) as deposit
FROM account_transactions
WHERE account_id = $1 AND transaction_type = 'deposit'
),
running AS (
SELECT close_time,
(SELECT deposit FROM initial_deposit) +
SUM(profit + swap + commission) OVER (ORDER BY close_time ROWS UNBOUNDED PRECEDING) as running_balance
FROM trades
WHERE account_id = $1 AND status = 'closed'
ORDER BY close_time
),
with_peak AS (
SELECT close_time, running_balance,
MAX(running_balance) OVER (ORDER BY close_time ROWS UNBOUNDED PRECEDING) as peak_so_far
FROM running
),
with_dd AS (
SELECT running_balance, peak_so_far,
CASE WHEN peak_so_far > 0
THEN (peak_so_far - running_balance) / peak_so_far * 100
ELSE 0
END as dd_pct
FROM with_peak
)
SELECT
COALESCE(MAX(peak_so_far), 0),
COALESCE(MAX(dd_pct), 0)
FROM with_dd
`, accountID).Scan(&historicalPeak, &trueMaxDD)

if err != nil {
log.Printf("[DD] DD query failed for %s: %v", accountID, err)
historicalPeak = 0
trueMaxDD = 0
}

// current_dd: dari peak (termasuk equity sekarang) ke equity sekarang
peakForCurrent := historicalPeak
if normalizedEquity > peakForCurrent {
peakForCurrent = normalizedEquity
}
currentDD := 0.0
if peakForCurrent > 0 && normalizedEquity < peakForCurrent {
currentDD = (peakForCurrent - normalizedEquity) / peakForCurrent * 100
}

// max_drawdown_pct = GREATEST dari trueMaxDD vs currentDD
_, err = s.db.Exec(`
UPDATE alpha_ranks SET
peak_equity      = $2,
last_equity      = $3,
current_dd       = $4,
max_drawdown_pct = GREATEST($5, $4)
WHERE account_id = $1
`, accountID, peakForCurrent, normalizedEquity, currentDD, trueMaxDD)

return err
}
