package alpharank

import "log"

// UpdateDrawdownMetrics dipanggil setiap EA push — return maxDD, currentDD
// DD dihitung dari equity_snapshots running peak-to-trough
// normalized_equity = equity + total_withdrawals (WD bukan loss, jangan hitung sebagai DD)
func (s *Service) UpdateDrawdownMetrics(accountID string, equity, totalWithdrawals float64) (float64, float64) {
	normalizedEquity := equity + totalWithdrawals

	var maxDD, peakEquity float64
	err := s.db.QueryRow(`
		WITH snapshots AS (
			SELECT snapshot_time,
				(equity + $2) AS norm_equity
			FROM equity_snapshots
			WHERE account_id = $1
			AND equity > 100
			ORDER BY snapshot_time ASC
		),
		with_peak AS (
			SELECT snapshot_time, norm_equity,
				MAX(norm_equity) OVER (ORDER BY snapshot_time ROWS UNBOUNDED PRECEDING) AS peak_so_far
			FROM snapshots
		),
		with_dd AS (
			SELECT norm_equity, peak_so_far,
				CASE WHEN peak_so_far > 0
					THEN (peak_so_far - norm_equity) / peak_so_far * 100
					ELSE 0
				END AS dd_pct
			FROM with_peak
		)
		SELECT COALESCE(MAX(peak_so_far), 0), COALESCE(MAX(dd_pct), 0)
		FROM with_dd
	`, accountID, totalWithdrawals).Scan(&peakEquity, &maxDD)

	if err != nil {
		log.Printf("[DD] query failed for %s: %v", accountID, err)
		return 0, 0
	}

	// Current DD: normalized equity vs peak
	if normalizedEquity > peakEquity {
		peakEquity = normalizedEquity
	}
	currentDD := 0.0
	if peakEquity > 0 && normalizedEquity < peakEquity {
		currentDD = (peakEquity - normalizedEquity) / peakEquity * 100
	}

	if maxDD > 100 {
		maxDD = 100
	}
	if currentDD > 100 {
		currentDD = 100
	}

	// Simpan ke DB — langsung overwrite, tidak pakai GREATEST
	s.db.Exec(`
		UPDATE alpha_ranks SET
			peak_equity      = $2,
			last_equity      = $3,
			current_dd       = $4,
			max_drawdown_pct = $5
		WHERE account_id = $1
	`, accountID, peakEquity, normalizedEquity, currentDD, maxDD)

	return maxDD, currentDD
}
