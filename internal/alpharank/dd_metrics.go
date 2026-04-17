package alpharank

import "log"

// UpdateDrawdownMetrics dipanggil setiap EA push — return maxDD, currentDD
func (s *Service) UpdateDrawdownMetrics(accountID string, equity, totalWithdrawals float64) (float64, float64) {
normalizedEquity := equity + totalWithdrawals

var trueMaxDD float64
var historicalPeak float64

err := s.db.QueryRow(`
WITH events AS (
    SELECT created_at as event_time,
        CASE WHEN transaction_type = 'deposit' THEN amount ELSE -amount END as delta
    FROM account_transactions WHERE account_id = $1
    UNION ALL
    SELECT close_time, profit + swap + commission
    FROM trades WHERE account_id = $1 AND status = 'closed'
),
running AS (
    SELECT event_time, SUM(delta) OVER (ORDER BY event_time ROWS UNBOUNDED PRECEDING) as running_balance
    FROM events
),
with_peak AS (
    SELECT running_balance, MAX(running_balance) OVER (ORDER BY event_time ROWS UNBOUNDED PRECEDING) as peak_so_far
    FROM running
),
with_dd AS (
    SELECT running_balance, peak_so_far,
        CASE WHEN peak_so_far > 0 THEN (peak_so_far - running_balance) / peak_so_far * 100 ELSE 0 END as dd_pct
    FROM with_peak
)
SELECT COALESCE(MAX(peak_so_far), 0), COALESCE(MAX(dd_pct), 0)
FROM with_dd
`, accountID).Scan(&historicalPeak, &trueMaxDD)

if err != nil {
log.Printf("[DD] query failed for %s: %v", accountID, err)
return 0, 0
}

peakForCurrent := historicalPeak
if normalizedEquity > peakForCurrent {
peakForCurrent = normalizedEquity
}
currentDD := 0.0
if peakForCurrent > 0 && normalizedEquity < peakForCurrent {
currentDD = (peakForCurrent - normalizedEquity) / peakForCurrent * 100
}

maxDD := trueMaxDD
if currentDD > maxDD {
maxDD = currentDD
}
if maxDD > 100 {
maxDD = 100
}

// Simpan ke DB
s.db.Exec(`
UPDATE alpha_ranks SET
    peak_equity      = $2,
    last_equity      = $3,
    current_dd       = $4,
    max_drawdown_pct = $5
WHERE account_id = $1
`, accountID, peakForCurrent, normalizedEquity, currentDD, maxDD)

return maxDD, currentDD
}
