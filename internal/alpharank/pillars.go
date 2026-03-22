package alpharank

import (
	"fmt"
	"math"
	"time"
)

// P1: Return vs Drawdown (20%)
func CalculateP1(netProfit, totalDeposits, maxDDPct float64) PillarScore {
	pillar := PillarScore{
		Code:   "P1",
		Name:   "Profitability",
		Weight: 20,
	}

	if netProfit <= 0 {
		pillar.Score = 0
		pillar.Reason = "Net loss"
		return pillar
	}

	growthPct := (netProfit / totalDeposits) * 100

	if maxDDPct == 0 {
		// No drawdown = perfect risk control
		pillar.Score = 100
		pillar.Reason = "No drawdown recorded"
		return pillar
	}

	R := growthPct / maxDDPct

	switch {
	case R <= 0.5:
		pillar.Score = 0
		pillar.Reason = "Return < half DD"
	case R <= 1.0:
		pillar.Score = 30
		pillar.Reason = "Return = DD"
	case R <= 2.0:
		pillar.Score = 60
		pillar.Reason = "Return 2x DD"
	case R <= 3.0:
		pillar.Score = 85
		pillar.Reason = "Return 3x DD"
	default:
		pillar.Score = 100
		pillar.Reason = fmt.Sprintf("Excellent (%.0fx DD)", R)
	}

	return pillar
}

// P2: Consistency (20%) - Fixed year+week grouping + inactive weeks
func CalculateP2(trades []TradeData) PillarScore {
	pillar := PillarScore{
		Code:   "P2",
		Name:   "Consistency",
		Weight: 20,
	}

	if len(trades) == 0 {
		pillar.Score = 0
		pillar.Reason = "No trades"
		return pillar
	}

	// Get date range
	firstTrade := trades[0].CloseTime
	lastTrade := trades[0].CloseTime
	for _, t := range trades {
		if t.CloseTime.Before(firstTrade) {
			firstTrade = t.CloseTime
		}
		if t.CloseTime.After(lastTrade) {
			lastTrade = t.CloseTime
		}
	}

	// Build weekly profit map (year+week key - FIX!)
	weeklyMap := make(map[string]float64)
	for _, trade := range trades {
		year, week := trade.CloseTime.ISOWeek()
		key := fmt.Sprintf("%d-%d", year, week)
		weeklyMap[key] += trade.Profit
	}

	// Fill inactive weeks with $0 (first trade to last trade only)
	// P7 handles inactive penalty - P2 only measures active period
	for d := firstTrade; d.Before(lastTrade.AddDate(0, 0, 7)); d = d.AddDate(0, 0, 7) {
		year, week := d.ISOWeek()
		key := fmt.Sprintf("%d-%d", year, week)
		if _, exists := weeklyMap[key]; !exists {
			weeklyMap[key] = 0 // Inactive week = $0
		}
	}

	weeklyReturns := make([]float64, 0, len(weeklyMap))
	for _, v := range weeklyMap {
		weeklyReturns = append(weeklyReturns, v)
	}

	if len(weeklyReturns) < 4 {
		pillar.Score = 0
		pillar.Reason = "Need 4+ weeks of history"
		return pillar
	}

	mean := 0.0
	for _, r := range weeklyReturns {
		mean += r
	}
	mean /= float64(len(weeklyReturns))

	variance := 0.0
	for _, r := range weeklyReturns {
		diff := r - mean
		variance += diff * diff
	}
	variance /= float64(len(weeklyReturns))
	stdDev := math.Sqrt(variance)

	if mean == 0 {
		pillar.Score = 0
		pillar.Reason = "Zero mean weekly profit"
		return pillar
	}

	CV := stdDev / math.Abs(mean)
	pillar.Score = 100.0 / (1.0 + CV)

	if pillar.Score > 100 {
		pillar.Score = 100
	}

	pillar.Reason = fmt.Sprintf("%.0f weeks analyzed (incl. inactive)", float64(len(weeklyReturns)))
	return pillar
}

// P3: Risk Flags (25%)
func CalculateP3(flags []RiskFlag) PillarScore {
	pillar := PillarScore{
		Code:   "P3",
		Name:   "Risk Management",
		Weight: 25,
	}

	score := 100.0
	for _, flag := range flags {
		score -= flag.Penalty
	}

	if score < 0 {
		score = 0
	}

	pillar.Score = score
	pillar.Reason = fmt.Sprintf("%d risk flags detected", len(flags))
	return pillar
}

// P4: Recovery (10%)
// P4: Recovery Resilience (10%)
// Measures how fast trader recovers from max drawdown to new peak
func CalculateP4(trades []TradeData, maxDDPct float64, initialDeposit float64) PillarScore {
	pillar := PillarScore{
		Code:   "P4",
		Name:   "Recovery Resilience",
		Weight: 10,
	}

	if len(trades) < 2 {
		pillar.Score = 50
		pillar.Reason = "Insufficient trades"
		return pillar
	}

	// No significant DD = perfect recovery
	if maxDDPct < 5 {
		pillar.Score = 100
		pillar.Reason = fmt.Sprintf("DD %.1f%% minimal - no recovery needed", maxDDPct)
		return pillar
	}

	// Find max drawdown point and recovery using running balance
	runningBalance := initialDeposit
	peak := initialDeposit
	maxDD := 0.0
	maxDDIdx := 0
	peakIdx := 0

	for i, trade := range trades {
		netProfit := trade.Profit + trade.Swap + trade.Commission
		runningBalance += netProfit
		if runningBalance > peak {
			peak = runningBalance
			peakIdx = i
		}
		if peak > 0 {
			dd := (peak - runningBalance) / peak * 100
			if dd > maxDD {
				maxDD = dd
				maxDDIdx = i
			}
		}
	}

	// Check if trader recovered after max DD point
	recoveryBalance := 0.0
	for _, trade := range trades[:maxDDIdx+1] {
		recoveryBalance += trade.Profit + trade.Swap + trade.Commission
	}
	peakBeforeDD := peak

	// Find recovery - did balance exceed peak after DD?
	recovered := false
	recoveryWeeks := 0
	runBal := recoveryBalance
	for i := maxDDIdx + 1; i < len(trades); i++ {
		runBal += trades[i].Profit + trades[i].Swap + trades[i].Commission
		if runBal >= peakBeforeDD {
			recovered = true
			// Estimate weeks from DD point to recovery
			duration := trades[i].CloseTime.Sub(trades[maxDDIdx].CloseTime)
			recoveryWeeks = int(duration.Hours() / 168)
			break
		}
	}
	_ = peakIdx

	if !recovered {
		// Not recovered yet
		if maxDDPct < 20 {
			pillar.Score = 50
			pillar.Reason = fmt.Sprintf("DD %.1f%% - recovering (not yet at peak)", maxDDPct)
		} else {
			pillar.Score = 20
			pillar.Reason = fmt.Sprintf("DD %.1f%% - not recovered to peak", maxDDPct)
		}
		return pillar
	}

	// Recovered - score based on speed
	switch {
	case recoveryWeeks <= 1:
		pillar.Score = 100
		pillar.Reason = fmt.Sprintf("Recovered from %.1f%% DD in %d week(s)", maxDDPct, recoveryWeeks)
	case recoveryWeeks <= 4:
		pillar.Score = 85
		pillar.Reason = fmt.Sprintf("Recovered from %.1f%% DD in %d weeks", maxDDPct, recoveryWeeks)
	case recoveryWeeks <= 12:
		pillar.Score = 65
		pillar.Reason = fmt.Sprintf("Recovered from %.1f%% DD in %d weeks", maxDDPct, recoveryWeeks)
	case recoveryWeeks <= 26:
		pillar.Score = 45
		pillar.Reason = fmt.Sprintf("Slow recovery from %.1f%% DD (%d weeks)", maxDDPct, recoveryWeeks)
	default:
		pillar.Score = 25
		pillar.Reason = fmt.Sprintf("Very slow recovery from %.1f%% DD (%d weeks)", maxDDPct, recoveryWeeks)
	}

	return pillar
}

// P5: Mathematical Edge (10%)
func CalculateP5(winRate, profitFactor float64) PillarScore {
	pillar := PillarScore{
		Code:   "P5",
		Name:   "Trading Edge",
		Weight: 10,
	}

	winScore := 0.0
	if winRate >= 60 {
		winScore = 100
	} else if winRate >= 55 {
		winScore = 80
	} else if winRate >= 50 {
		winScore = 60
	} else if winRate >= 45 {
		winScore = 40
	} else {
		winScore = 20
	}

	pfScore := 0.0
	if profitFactor >= 2.0 {
		pfScore = 100
	} else if profitFactor >= 1.5 {
		pfScore = 80
	} else if profitFactor >= 1.2 {
		pfScore = 60
	} else if profitFactor >= 1.0 {
		pfScore = 40
	} else {
		pfScore = 0
	}

	pillar.Score = (winScore + pfScore) / 2.0
	pillar.Reason = fmt.Sprintf("WinRate %.1f%%, PF %.2f", winRate, profitFactor)
	return pillar
}

// P6: Discipline (8%)
func CalculateP6(trades []TradeData) PillarScore {
	pillar := PillarScore{
		Code:   "P6",
		Name:   "Discipline",
		Weight: 12,
	}

	if len(trades) == 0 {
		pillar.Score = 0
		pillar.Reason = "No trades"
		return pillar
	}

	score := 100.0
	reasons := []string{}

	// 1. Stop Loss usage (max penalty -35)
	withSL := 0
	for _, t := range trades {
		if t.StopLoss != 0 {
			withSL++
		}
	}
	slPct := float64(withSL) / float64(len(trades)) * 100
	if slPct < 30 {
		score -= 35
		reasons = append(reasons, fmt.Sprintf("SL %.0f%%", slPct))
	} else if slPct < 60 {
		score -= 15
		reasons = append(reasons, fmt.Sprintf("SL %.0f%%", slPct))
	} else {
		reasons = append(reasons, fmt.Sprintf("SL %.0f%%", slPct))
	}

	// 2. Take Profit usage (max penalty -25)
	withTP := 0
	for _, t := range trades {
		if t.TakeProfit != 0 {
			withTP++
		}
	}
	tpPct := float64(withTP) / float64(len(trades)) * 100
	if tpPct < 30 {
		score -= 25
		reasons = append(reasons, fmt.Sprintf("TP %.0f%%", tpPct))
	} else if tpPct < 60 {
		score -= 10
		reasons = append(reasons, fmt.Sprintf("TP %.0f%%", tpPct))
	} else {
		reasons = append(reasons, fmt.Sprintf("TP %.0f%%", tpPct))
	}

	// 3. Overtrading detection - trades per day after loss (max penalty -20)
	dailyMap := make(map[string][]TradeData)
	for _, t := range trades {
		day := t.CloseTime.Format("2006-01-02")
		dailyMap[day] = append(dailyMap[day], t)
	}
	overtradeDays := 0
	for _, dayTrades := range dailyMap {
		if len(dayTrades) >= 5 {
			// Check if previous day was loss
			dayProfit := 0.0
			for _, t := range dayTrades {
				dayProfit += t.Profit
			}
			if dayProfit < 0 {
				overtradeDays++
			}
		}
	}
	if overtradeDays >= 5 {
		score -= 20
		reasons = append(reasons, fmt.Sprintf("overtrading %dd", overtradeDays))
	} else if overtradeDays >= 2 {
		score -= 10
		reasons = append(reasons, fmt.Sprintf("overtrading %dd", overtradeDays))
	}

	if score < 0 {
		score = 0
	}

	pillar.Score = score
	if len(reasons) > 0 {
		pillar.Reason = fmt.Sprintf("SL/TP/OT: %s/%s", reasons[0], reasons[1])
		if len(reasons) > 2 {
			pillar.Reason += " " + reasons[2]
		}
	}
	return pillar
}

// P7: Track Record (7%) - With inactive penalty
func CalculateP7(totalTrades, daysSinceStart int, lastTradeTime time.Time) PillarScore {
	pillar := PillarScore{
		Code:   "P7",
		Name:   "Track Record",
		Weight: 3,
	}

	tradesScore := 0.0
	if totalTrades >= 200 {
		tradesScore = 100
	} else if totalTrades >= 100 {
		tradesScore = 80
	} else if totalTrades >= 50 {
		tradesScore = 60
	} else if totalTrades >= 20 {
		tradesScore = 40
	} else {
		tradesScore = 20
	}

	historyScore := 0.0
	if daysSinceStart >= 180 {
		historyScore = 100
	} else if daysSinceStart >= 90 {
		historyScore = 80
	} else if daysSinceStart >= 30 {
		historyScore = 60
	} else {
		historyScore = 40
	}

	// Inactive penalty - weeks since last trade
	weeksSinceLastTrade := time.Since(lastTradeTime).Hours() / 168
	inactivePenalty := 1.0
	inactiveNote := ""

	if weeksSinceLastTrade > 12 {
		inactivePenalty = 0.4
		inactiveNote = fmt.Sprintf(" (inactive %.0f weeks!)", weeksSinceLastTrade)
	} else if weeksSinceLastTrade > 8 {
		inactivePenalty = 0.6
		inactiveNote = fmt.Sprintf(" (inactive %.0f weeks)", weeksSinceLastTrade)
	} else if weeksSinceLastTrade > 4 {
		inactivePenalty = 0.8
		inactiveNote = fmt.Sprintf(" (inactive %.0f weeks)", weeksSinceLastTrade)
	}

	historyScore *= inactivePenalty

	pillar.Score = (tradesScore + historyScore) / 2.0
	pillar.Reason = fmt.Sprintf("%d trades, %d days history%s", totalTrades, daysSinceStart, inactiveNote)
	return pillar
}

// Helper: Calculate weekly returns (year+week fixed)
func calculateWeeklyReturns(trades []TradeData) []float64 {
	if len(trades) == 0 {
		return []float64{}
	}

	weeklyProfits := make(map[string]float64)
	for _, trade := range trades {
		year, week := trade.CloseTime.ISOWeek()
		key := fmt.Sprintf("%d-%d", year, week)
		weeklyProfits[key] += trade.Profit
	}

	returns := make([]float64, 0, len(weeklyProfits))
	for _, profit := range weeklyProfits {
		returns = append(returns, profit)
	}

	return returns
}
