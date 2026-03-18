package trader

import (
	"database/sql"
	"crunchalpha-v3/internal/alpharank"
)

// GetMaxDrawdown - Calculate from EQUITY CURVE (trades only)
// IGNORE deposits/withdrawals to match backend alpharank calculation
// This shows trading performance DD, not account balance DD
func GetMaxDrawdown(db *sql.DB, accountID string, currentEquity float64, trades []alpharank.TradeData) float64 {
	if len(trades) == 0 {
		return 0
	}

	// Start from 0, build equity curve from closed trades only
	runningBalance := 0.0
	peak := 0.0
	maxDD := 0.0

	for _, trade := range trades {
		// Add trade P/L
		netProfit := trade.Profit + trade.Swap + trade.Commission
		runningBalance += netProfit

		// Update peak
		if runningBalance > peak {
			peak = runningBalance
		}

		// Calculate DD from equity curve
		if peak > 0 {
			dd := (peak - runningBalance) / peak * 100
			if dd > maxDD {
				maxDD = dd
			}
		}
	}

	return maxDD
}

// Dummy functions to avoid errors
func CalculateAbsoluteDD(db *sql.DB, accountID string, currentEquity float64) float64 {
	return 0
}

func CalculateRelativeDD(db *sql.DB, accountID string, trades []alpharank.TradeData) float64 {
	return 0
}

func CalculateRealDD(db *sql.DB, accountID string, trades []alpharank.TradeData) float64 {
	return 0
}
