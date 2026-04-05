package alpharank

import (
	"fmt"
	"math"
	"time"
)

func DetectRiskFlags(metrics AccountMetrics) []RiskFlag {
	var flags []RiskFlag

	// Use peak balance for leverage calculations (not current balance!)
	peakBalance := metrics.PeakBalance
	if peakBalance <= 0 {
		peakBalance = metrics.TotalDeposits
	}
	if peakBalance <= 0 {
		peakBalance = metrics.CurrentBalance
	}

	flags = append(flags, detectNoStopLoss(metrics.Trades, metrics.MaxDrawdownPct)...)
	flags = append(flags, detectExcessivePositionSize(metrics.Trades, peakBalance)...)
	flags = append(flags, detectRevengeTrading(metrics.Trades)...)
	flags = append(flags, detectConsistencyVolatility(metrics.Trades)...)
	// flags = append(flags, detectHighCorrelation(metrics.Trades)...) // REMOVED: Owner request
	flags = append(flags, detectLotInconsistency(metrics.Trades)...)
	flags = append(flags, detectMartingale(metrics.Trades)...)
	flags = append(flags, detectExtremeDrawdown(metrics.MaxDrawdownPct)...)
	flags = append(flags, detectStrategyChange(metrics.Trades)...)
	flags = append(flags, detectLongFloatingLoss(metrics.Trades)...)
	flags = append(flags, detectWeekendExposure(metrics.Trades)...)

	return flags
}

// CRITICAL: No Stop Loss - Sliding scale based on DD
func detectNoStopLoss(trades []TradeData, maxDD float64) []RiskFlag {
	if len(trades) == 0 {
		return nil
	}

	withSL := 0
	for _, t := range trades {
		if t.StopLoss != 0 {
			withSL++
		}
	}
	slPct := float64(withSL) / float64(len(trades)) * 100

	if slPct >= 30 {
		return nil
	}

	// Sliding scale based on actual DD
	if maxDD < 10 {
		return []RiskFlag{{
			FlagType: "NO_STOP_LOSS",
			Severity: "MINOR",
			Penalty:  10.0,
			Title:    "No Stop Loss",
			Desc:     fmt.Sprintf("%.0f%% trades without SL - DD controlled at %.2f%%", 100-slPct, maxDD),
			SoftTitle: "Open-Exit Strategy",
			SoftDesc:  "Trader manages exits without fixed stop loss — drawdown remains controlled.",
		}}
	} else if maxDD < 20 {
		return []RiskFlag{{
			FlagType: "NO_STOP_LOSS",
			Severity: "MAJOR",
			Penalty:  18.0,
			Title:    "No Stop Loss",
			Desc:     fmt.Sprintf("%.0f%% trades without SL - DD moderate at %.2f%%", 100-slPct, maxDD),
			SoftTitle: "Unprotected Exit Risk",
			SoftDesc:  "Limited use of stop loss with moderate drawdown — monitor closely.",
		}}
	}

	return []RiskFlag{{
		FlagType: "NO_STOP_LOSS",
		Severity: "CRITICAL",
		Penalty:  25.0,
		Title:    "No Stop Loss",
		Desc:     fmt.Sprintf("%.0f%% trades without SL - DD high at %.2f%%", 100-slPct, maxDD),
		SoftTitle: "High Exit Risk",
		SoftDesc:  "Trades without stop loss protection — significant drawdown exposure.",
	}}
}

// CRITICAL: Overleveraging - Use peak balance
func detectExcessivePositionSize(trades []TradeData, peakBalance float64) []RiskFlag {
	if len(trades) == 0 || peakBalance <= 0 {
		return nil
	}

	// Standard: 1 lot = $100,000 notional, safe = 0.01 lot per $1000 (1:100 leverage safe usage)
	// lotRatio = lots / (peakBalance / 100000) — how many "standard lots" relative to balance
	// ratio > 2.0 = using more than 2x safe standard = excessive
	excessiveCount := 0
	maxRatio := 0.0

	for _, t := range trades {
		if t.Lots <= 0 {
			continue
		}
		safeMaxLots := peakBalance / 100000.0 * 2.0 // 2x safe threshold
		if safeMaxLots <= 0 {
			safeMaxLots = 0.01
		}
		ratio := t.Lots / safeMaxLots
		if ratio > maxRatio {
			maxRatio = ratio
		}
		if ratio > 1.0 {
			excessiveCount++
		}
	}

	excessivePct := float64(excessiveCount) / float64(len(trades)) * 100

	if excessivePct < 10 {
		return nil
	}

	if excessivePct > 30 || maxRatio > 5.0 {
		return []RiskFlag{{
			FlagType: "EXCESSIVE_POSITION_SIZE",
			Severity: "CRITICAL",
			Penalty:  25.0,
			Title:    "Excessive Position Size",
			Desc:     fmt.Sprintf("%.0f%% trades use lot size too large for balance ($%.0f)", excessivePct, peakBalance),
		}}
	}

	return []RiskFlag{{
		FlagType: "EXCESSIVE_POSITION_SIZE",
		Severity: "MAJOR",
		Penalty:  15.0,
		Title:    "Large Position Size",
		Desc:     fmt.Sprintf("%.0f%% trades exceed safe lot size for balance ($%.0f)", excessivePct, peakBalance),
		SoftTitle: "High Position Sizing",
		SoftDesc:  "Position sizes above typical safe levels relative to account balance.",
	}}
}

// MAJOR: Revenge Trading - Per symbol, confirmed with outcome
func detectRevengeTrading(trades []TradeData) []RiskFlag {
	if len(trades) < 2 {
		return nil
	}

	// Group trades by symbol
	bySymbol := make(map[string][]TradeData)
	for _, t := range trades {
		bySymbol[t.Symbol] = append(bySymbol[t.Symbol], t)
	}

	revengeCount := 0
	confirmedRevenge := 0

	for _, symbolTrades := range bySymbol {
		for i := 1; i < len(symbolTrades); i++ {
			prev := symbolTrades[i-1]
			curr := symbolTrades[i]

			if prev.Profit >= 0 {
				continue
			}

			if math.Abs(prev.Profit) < 50 {
				continue
			}

			if prev.Lots <= 0 {
				continue
			}

			lotIncrease := (curr.Lots/prev.Lots - 1) * 100
			if lotIncrease > 100 {
				revengeCount++
				// Confirmed revenge: next trade also loss
				if curr.Profit < 0 {
					confirmedRevenge++
				}
			}
		}
	}

	if confirmedRevenge >= 3 {
		return []RiskFlag{{
			FlagType: "REVENGE_TRADING",
			Severity: "MAJOR",
			Penalty:  15.0,
			Title:    "Revenge Trading",
			Desc:     fmt.Sprintf("Lot spike after loss confirmed %dx (same symbol)", confirmedRevenge),
			SoftTitle: "Emotional Recovery Pattern",
			SoftDesc:  "Signs of increased trading activity following losses.",
		}}
	}

	if confirmedRevenge >= 1 {
		return []RiskFlag{{
			FlagType: "REVENGE_TRADING",
			Severity: "MINOR",
			Penalty:  8.0,
			Title:    "Revenge Trading",
			Desc:     fmt.Sprintf("Possible revenge trading detected %dx", confirmedRevenge),
			SoftTitle: "Active Loss Recovery",
			SoftDesc:  "Minor pattern of increased activity after drawdown periods.",
		}}
	}

	return nil
}

// MINOR: Consistency Volatility - Fixed year+week, check loss weeks
func detectConsistencyVolatility(trades []TradeData) []RiskFlag {
	// Fixed: include year in weekly grouping
	weeklyMap := make(map[string]float64)
	for _, trade := range trades {
		year, week := trade.CloseTime.ISOWeek()
		key := fmt.Sprintf("%d-%d", year, week)
		weeklyMap[key] += trade.Profit
	}

	weeklyReturns := make([]float64, 0, len(weeklyMap))
	for _, v := range weeklyMap {
		weeklyReturns = append(weeklyReturns, v)
	}

	if len(weeklyReturns) < 4 {
		return nil
	}

	mean := 0.0
	for _, r := range weeklyReturns {
		mean += r
	}
	mean /= float64(len(weeklyReturns))

	if mean <= 0 {
		return nil
	}

	variance := 0.0
	for _, r := range weeklyReturns {
		diff := r - mean
		variance += diff * diff
	}
	variance /= float64(len(weeklyReturns))
	stdDev := math.Sqrt(variance)

	cv := stdDev / math.Abs(mean)

	// Count loss weeks
	lossWeeks := 0
	for _, r := range weeklyReturns {
		if r < 0 {
			lossWeeks++
		}
	}
	lossPct := float64(lossWeeks) / float64(len(weeklyReturns)) * 100

	if cv > 2.0 && lossPct > 30 {
		return []RiskFlag{{
			FlagType: "CONSISTENCY_VOLATILITY",
			Severity: "MINOR",
			Penalty:  8.0,
			Title:    "High Consistency Volatility",
			Desc:     fmt.Sprintf("Weekly variance CV=%.1f, %.0f%% loss weeks", cv, lossPct),
			SoftTitle: "Flexible Position Sizing",
			SoftDesc:  "Some variation in lot sizes — common in adaptive strategies.",
		}}
	}

	if cv > 2.0 && lossPct <= 30 {
		return []RiskFlag{{
			FlagType: "CONSISTENCY_VOLATILITY",
			Severity: "MINOR",
			Penalty:  4.0,
			Title:    "High Weekly Variance",
			Desc:     fmt.Sprintf("Weekly variance CV=%.1f but %.0f%% weeks profitable", cv, 100-lossPct),
			SoftTitle: "Flexible Position Sizing",
			SoftDesc:  "Some variation in lot sizes — common in adaptive strategies.",
		}}
	}

	return nil
}

// MAJOR: High Correlation - Time window based (basket trades)
func detectHighCorrelation(trades []TradeData) []RiskFlag {
	if len(trades) < 5 {
		return nil
	}

	maxSimultaneous := 0

	for i, t := range trades {
		count := 1
		for j, other := range trades {
			if i == j {
				continue
			}
			// Same direction within 60 seconds = basket trade
			timeDiff := math.Abs(t.OpenTime.Sub(other.OpenTime).Seconds())
			if timeDiff <= 60 {
				continue // Skip basket trades (same time = intentional)
			}
			// Check if positions overlap
			if t.OpenTime.Before(other.CloseTime) && other.OpenTime.Before(t.CloseTime) {
				if t.Type == other.Type {
					count++
				}
			}
		}
		if count > maxSimultaneous {
			maxSimultaneous = count
		}
	}

	if maxSimultaneous >= 10 {
		return []RiskFlag{{
			FlagType: "HIGH_CORRELATION",
			Severity: "MAJOR",
			Penalty:  15.0,
			Title:    "High Correlated Positions",
			Desc:     fmt.Sprintf("Up to %d same-direction positions open simultaneously", maxSimultaneous),
		}}
	}

	return nil
}

// MINOR: Lot Inconsistency - Per symbol
func detectLotInconsistency(trades []TradeData) []RiskFlag {
	if len(trades) < 30 {
		return nil
	}

	// Group by symbol
	bySymbol := make(map[string][]float64)
	for _, t := range trades {
		if t.Lots > 0 {
			bySymbol[t.Symbol] = append(bySymbol[t.Symbol], t.Lots)
		}
	}

	for symbol, lots := range bySymbol {
		if len(lots) < 10 {
			continue
		}

		var sum, sumSq float64
		for _, lot := range lots {
			sum += lot
			sumSq += lot * lot
		}
		mean := sum / float64(len(lots))
		variance := (sumSq / float64(len(lots))) - (mean * mean)
		stdDev := math.Sqrt(variance)
		cv := (stdDev / mean) * 100

		if cv > 150 {
			return []RiskFlag{{
				FlagType: "LOT_INCONSISTENCY",
				Severity: "MINOR",
				Penalty:  10.0,
				Title:    "Lot Size Inconsistency",
				Desc:     fmt.Sprintf("%s lot sizes vary %.0f%% (erratic sizing)", symbol, cv),
				SoftTitle: "Adaptive Lot Sizing",
				SoftDesc:  "Lot sizes vary — may reflect strategy adaptation to market conditions.",
			}}
		}
	}

	return nil
}

// CRITICAL: Max Position Size - Use peak balance

// CRITICAL: Martingale detection
func detectMartingale(trades []TradeData) []RiskFlag {
	if len(trades) < 10 {
		return nil
	}

	consecutiveDoubles := 0
	maxConsecutive := 0

	for i := 1; i < len(trades); i++ {
		if trades[i-1].Lots > 0 && trades[i].Lots >= trades[i-1].Lots*1.8 {
			consecutiveDoubles++
			if consecutiveDoubles > maxConsecutive {
				maxConsecutive = consecutiveDoubles
			}
		} else {
			consecutiveDoubles = 0
		}
	}

	if maxConsecutive >= 3 {
		return []RiskFlag{{
			FlagType: "MARTINGALE",
			Severity: "CRITICAL",
			Penalty:  30.0,
			Title:    "Martingale Pattern",
			Desc:     fmt.Sprintf("Lot doubling pattern detected (%dx consecutive)", maxConsecutive),
			SoftTitle: "Progressive Position Strategy",
			SoftDesc:  "Uses increasing position sizes after losses — high risk in adverse markets.",
		}}
	}

	return nil
}

// CRITICAL: Extreme Drawdown
func detectExtremeDrawdown(maxDD float64) []RiskFlag {
	if maxDD >= 50 {
		return []RiskFlag{{
			FlagType: "EXTREME_DRAWDOWN",
			Severity: "CRITICAL",
			Penalty:  25.0,
			Title:    "Extreme Drawdown",
			Desc:     fmt.Sprintf("Max drawdown %.1f%% is extremely high", maxDD),
			SoftTitle: "High Capital Exposure Period",
			SoftDesc:  "Account experienced significant drawdown — recovery required.",
		}}
	}
	if maxDD >= 30 {
		return []RiskFlag{{
			FlagType: "EXTREME_DRAWDOWN",
			Severity: "MAJOR",
			Penalty:  15.0,
			Title:    "High Drawdown",
			Desc:     fmt.Sprintf("Max drawdown %.1f%% is high", maxDD),
			SoftTitle: "Notable Drawdown Period",
			SoftDesc:  "Moderate drawdown recorded — within recoverable range.",
		}}
	}
	return nil
}

// MAJOR: Strategy Change
func detectStrategyChange(trades []TradeData) []RiskFlag {
	if len(trades) < 20 {
		return nil
	}

	mid := len(trades) / 2
	avg1 := avgDuration(trades[:mid])
	avg2 := avgDuration(trades[mid:])

	if avg1 == 0 {
		return nil
	}

	change := math.Abs(avg2/avg1-1) * 100
	if change > 300 {
		return []RiskFlag{{
			FlagType: "STRATEGY_CHANGE",
			Severity: "MAJOR",
			Penalty:  15.0,
			Title:    "Strategy Change Detected",
			Desc:     fmt.Sprintf("Trading duration changed %.0f%% between periods", change),
			SoftTitle: "Evolving Strategy",
			SoftDesc:  "Trading approach has shifted — recent performance may not reflect history.",
		}}
	}

	return nil
}

// Helpers
func avgDuration(trades []TradeData) float64 {
	if len(trades) == 0 {
		return 0
	}
	total := 0.0
	for _, t := range trades {
		total += t.CloseTime.Sub(t.OpenTime).Hours()
	}
	return total / float64(len(trades))
}

func isWeekend(t time.Time) bool {
	return t.Weekday() == time.Saturday || t.Weekday() == time.Sunday
}

// MAJOR: Long Floating Loss - position in loss >5 days
func detectLongFloatingLoss(trades []TradeData) []RiskFlag {
	if len(trades) == 0 {
		return nil
	}
	longFloatCount := 0
	for _, t := range trades {
		if t.Profit < 0 {
			duration := t.CloseTime.Sub(t.OpenTime)
			if duration.Hours() > 5*24 {
				longFloatCount++
			}
		}
	}
	longFloatPct := float64(longFloatCount) / float64(len(trades)) * 100
	if longFloatPct > 20 {
		return []RiskFlag{{
			FlagType: "LONG_FLOATING_LOSS",
			Severity: "MAJOR",
			Penalty:  20.0,
			Title:    "Long Floating Loss",
			Desc:     fmt.Sprintf("%.0f%% trades held in loss >5 days", longFloatPct),
			SoftTitle: "Extended Open Position",
			SoftDesc:  "Holds losing positions for extended periods — floating risk present.",
		}}
	}
	if longFloatPct > 10 {
		return []RiskFlag{{
			FlagType: "LONG_FLOATING_LOSS",
			Severity: "MINOR",
			Penalty:  8.0,
			Title:    "Long Floating Loss",
			Desc:     fmt.Sprintf("%.0f%% trades held in loss >5 days", longFloatPct),
			SoftTitle: "Patience-Based Exit",
			SoftDesc:  "Occasionally holds trades longer than average — monitor exposure.",
		}}
	}
	return nil
}

// MINOR: Weekend Exposure - positions held over weekend
func detectWeekendExposure(trades []TradeData) []RiskFlag {
	if len(trades) == 0 {
		return nil
	}
	weekendCount := 0
	for _, t := range trades {
		openDay := t.OpenTime.Weekday()
		duration := t.CloseTime.Sub(t.OpenTime)
		if openDay == 5 || (duration.Hours() > 48) {
			weekendCount++
		}
	}
	weekendPct := float64(weekendCount) / float64(len(trades)) * 100
	if weekendPct > 50 {
		return []RiskFlag{{
			FlagType: "WEEKEND_EXPOSURE",
			Severity: "MINOR",
			Penalty:  10.0,
			Title:    "Weekend Exposure",
			Desc:     fmt.Sprintf("%.0f%% trades held over weekend (gap risk)", weekendPct),
			SoftTitle: "Weekend Position Holder",
			SoftDesc:  "Keeps positions open over weekends — exposed to gap risk.",
		}}
	}
	return nil
}
