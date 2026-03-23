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
// P2: Consistency (20%) - Active weeks only, with loss-week multiplier
// P2: Consistency (20%)
// CV = stdDev / |mean| dari % return per minggu (bukan nominal dollar)
// % return minggu = profit minggu / balance awal minggu x 100
// Active weeks only - gap weeks = trader waiting for setup = discipline
// Bonus multiplier for low loss weeks
// P2: Consistency (20%)
// Prioritas: pakai equity snapshots jika ada (include floating)
// Fallback: initialDeposit + cumulative closed profit
// % return per week = (equity_end - equity_start) / equity_start x 100
// P2: Consistency (20%)
// Priority 1: equity snapshots per week (equity-based, include floating)
// Fallback: trade-by-trade P&L variance (per dokumen section 4.4)
// P2: Consistency (20%)
// Priority 1: equity snapshots per week (include floating)
// Fallback: weekly closed profit / running balance from initialDeposit
func CalculateP2(trades []TradeData, initialDeposit float64, snapshots []EquitySnapshot) PillarScore {
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

	var weeklyReturns []float64
	dataSource := ""

	// Priority 1: equity snapshots per week (include floating)
	// Only use if equity data has meaningful variance (not all same value)
	hasVariance := false
	if len(snapshots) >= 4 {
		firstEq := snapshots[0].Equity
		for _, s := range snapshots {
			if math.Abs(s.Equity-firstEq) > 1.0 {
				hasVariance = true
				break
			}
		}
	}
	if hasVariance {
		weeklyEquity := make(map[string]float64)
		weekOrder := []string{}
		weekSeen := make(map[string]bool)
		for _, snap := range snapshots {
			year, week := snap.SnapshotTime.ISOWeek()
			key := fmt.Sprintf("%d-%02d", year, week)
			if !weekSeen[key] {
				weekSeen[key] = true
				weekOrder = append(weekOrder, key)
			}
			weeklyEquity[key] = snap.Equity // last snapshot of week
		}
		if len(weekOrder) >= 4 {
			for i := 1; i < len(weekOrder); i++ {
				prevEq := weeklyEquity[weekOrder[i-1]]
				currEq := weeklyEquity[weekOrder[i]]
				if prevEq > 0 {
					weeklyReturns = append(weeklyReturns, (currEq-prevEq)/prevEq*100)
				}
			}
			dataSource = "equity snapshots"
		}
	}

	// Fallback: weekly closed profit / running balance from initialDeposit
	if len(weeklyReturns) < 4 {
		dataSource = "closed trades"
		// Sort trades
		sorted := make([]TradeData, len(trades))
		copy(sorted, trades)
		for i := 0; i < len(sorted)-1; i++ {
			for j := i + 1; j < len(sorted); j++ {
				if sorted[i].CloseTime.After(sorted[j].CloseTime) {
					sorted[i], sorted[j] = sorted[j], sorted[i]
				}
			}
		}

		// Group by week
		weeklyProfit := make(map[string]float64)
		weekOrder := []string{}
		weekSeen := make(map[string]bool)
		for _, t := range sorted {
			year, week := t.CloseTime.ISOWeek()
			key := fmt.Sprintf("%d-%02d", year, week)
			if !weekSeen[key] {
				weekSeen[key] = true
				weekOrder = append(weekOrder, key)
			}
			weeklyProfit[key] += t.Profit + t.Swap + t.Commission
		}

		// Calculate % return per week
		// Week 1: profit / initialDeposit
		// Week N: profit / (initialDeposit + sum of prev weeks profit)
		weeklyReturns = []float64{}
		runningBalance := initialDeposit
		if runningBalance <= 0 {
			runningBalance = 1000 // default fallback
		}
		for _, key := range weekOrder {
			profit := weeklyProfit[key]
			if runningBalance > 0 {
				ret := profit / runningBalance * 100
				weeklyReturns = append(weeklyReturns, ret)
			}
			runningBalance += profit
		}
	}

	if len(weeklyReturns) < 4 {
		pillar.Score = 0
		pillar.Reason = "Need 4+ weeks of history"
		return pillar
	}

	// Mean
	mean := 0.0
	for _, r := range weeklyReturns {
		mean += r
	}
	mean /= float64(len(weeklyReturns))

	if mean == 0 {
		pillar.Score = 0
		pillar.Reason = "Zero mean weekly return"
		return pillar
	}

	// StdDev & CV
	variance := 0.0
	for _, r := range weeklyReturns {
		diff := r - mean
		variance += diff * diff
	}
	variance /= float64(len(weeklyReturns))
	stdDev := math.Sqrt(variance)
	CV := stdDev / math.Abs(mean)

	// Base score
	baseScore := 100.0 / (1.0 + CV)

	score := baseScore
	if score > 100 {
		score = 100
	}

	pillar.Score = score
	pillar.Reason = fmt.Sprintf("%d weeks (%s), CV=%.2f", len(weeklyReturns), dataSource, CV)
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
// P4: Recovery Resilience (10%)
// P4 = (RV_score x 0.6) + (DD_score x 0.4)
// RV = TotalReturn% / MaxDD% (equity-based)
// DD Duration = avg days trader stuck underwater
func CalculateP4(trades []TradeData, maxDDPct float64, totalReturnPct float64) PillarScore {
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

	// Component A: Recovery Velocity (60%)
	// RV = TotalReturn% / MaxDD%
	rvScore := 0.0
	if maxDDPct <= 0 {
		// No DD = perfect
		rvScore = 100
	} else {
		RV := totalReturnPct / maxDDPct
		switch {
		case RV < 1:
			rvScore = 0
		case RV < 2:
			rvScore = 40
		case RV < 5:
			rvScore = 70
		case RV < 10:
			rvScore = 90
		default:
			rvScore = 100
		}
	}

	// Component B: DD Duration (40%)
	// Calculate avg days trader is "underwater" (below peak)
	type dayBalance struct {
		date string
		bal  float64
	}

	// Build daily balance map
	dailyMap := make(map[string]float64)
	runBal := 0.0
	for _, t := range trades {
		runBal += t.Profit + t.Swap + t.Commission
		day := t.CloseTime.Format("2006-01-02")
		dailyMap[day] = runBal
	}

	// Find underwater days
	peak := 0.0
	underwaterDays := 0
	totalDays := 0
	for _, t := range trades {
		day := t.CloseTime.Format("2006-01-02")
		bal := dailyMap[day]
		if bal > peak {
			peak = bal
		}
		if peak > 0 && bal < peak {
			underwaterDays++
		}
		totalDays++
	}

	// Avg underwater duration as % of trading days
	avgUnderwaterPct := 0.0
	if totalDays > 0 {
		avgUnderwaterPct = float64(underwaterDays) / float64(totalDays) * 100
	}

	ddScore := 0.0
	switch {
	case avgUnderwaterPct < 10:
		ddScore = 100
	case avgUnderwaterPct < 25:
		ddScore = 80
	case avgUnderwaterPct < 40:
		ddScore = 60
	case avgUnderwaterPct < 60:
		ddScore = 40
	default:
		ddScore = 0
	}

	// Final P4
	p4 := (rvScore * 0.6) + (ddScore * 0.4)
	pillar.Score = p4

	RV := 0.0
	if maxDDPct > 0 {
		RV = totalReturnPct / maxDDPct
	}
	pillar.Reason = fmt.Sprintf("RV=%.1fx DD, underwater %.0f%% of days", RV, avgUnderwaterPct)
	return pillar
}

// P5: Mathematical Edge (10%)
// P5: Mathematical Edge (10%)
// P5 = (E_score x 0.70) + (S_score x 0.30)
// E = (WinRate x AvgWin) - (LossRate x AvgLoss) -- Van Tharp Expectancy
// Sharpe = AvgWeeklyReturn / StdDev -- William Sharpe
func CalculateP5(trades []TradeData, winRate, avgWin, avgLoss float64) PillarScore {
	pillar := PillarScore{
		Code:   "P5",
		Name:   "Trading Edge",
		Weight: 10,
	}

	if len(trades) == 0 {
		pillar.Score = 0
		pillar.Reason = "No trades"
		return pillar
	}

	// Component A: Expectancy (70%) - Van Tharp
	// E = (WinRate x AvgWin) - (LossRate x AvgLoss)
	winRateDec := winRate / 100.0
	lossRateDec := 1.0 - winRateDec
	E := (winRateDec * avgWin) - (lossRateDec * avgLoss)

	Epct := 0.0
	if avgWin > 0 {
		Epct = (E / avgWin) * 100
	}

	eScore := 0.0
	switch {
	case Epct <= 0:
		eScore = 0
	case Epct <= 10:
		eScore = 30
	case Epct <= 25:
		eScore = 60
	case Epct <= 50:
		eScore = 85
	default:
		eScore = 100
	}

	// Component B: Sharpe Ratio (30%) - William Sharpe
	// Sharpe = AvgWeeklyReturn / StdDev of weekly returns
	weeklyMap := make(map[string]float64)
	for _, t := range trades {
		year, week := t.CloseTime.ISOWeek()
		key := fmt.Sprintf("%d-%d", year, week)
		weeklyMap[key] += t.Profit + t.Swap + t.Commission
	}

	weeklyReturns := make([]float64, 0, len(weeklyMap))
	for _, v := range weeklyMap {
		weeklyReturns = append(weeklyReturns, v)
	}

	sScore := 0.0
	if len(weeklyReturns) >= 4 {
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

		sharpe := 0.0
		if stdDev > 0 {
			sharpe = mean / stdDev
		}

		switch {
		case sharpe < 0.5:
			sScore = 0
		case sharpe < 1.0:
			sScore = 40
		case sharpe < 2.0:
			sScore = 70
		case sharpe < 3.0:
			sScore = 90
		default:
			sScore = 100
		}
	}

	p5 := (eScore * 0.70) + (sScore * 0.30)
	pillar.Score = p5
	pillar.Reason = fmt.Sprintf("Expectancy %.1f%%, E_score=%.0f, S_score=%.0f", Epct, eScore, sScore)
	return pillar
}

// P6: Discipline (8%)
// P6: Behavioral Discipline (8%)
// Base: 70
// Bonus: pause after DD, lot reduction after loss, consistent sizing
// Penalty: revenge trading, overtrading
func CalculateP6(trades []TradeData) PillarScore {
	pillar := PillarScore{
		Code:   "P6",
		Name:   "Discipline",
		Weight: 8,
	}

	if len(trades) == 0 {
		pillar.Score = 0
		pillar.Reason = "No trades"
		return pillar
	}

	score := 70.0
	reasons := []string{}

	// 1. Pause after DD >15% (+15)
	// Check if trader reduced trades after significant loss period
	dailyPnl := make(map[string]float64)
	for _, t := range trades {
		day := t.CloseTime.Format("2006-01-02")
		dailyPnl[day] += t.Profit
	}
	pausedAfterDD := false
	runningPnl := 0.0
	peak := 0.0
	var days []string
	for d := range dailyPnl {
		days = append(days, d)
	}
	// Sort days
	for i := 0; i < len(days)-1; i++ {
		for j := i + 1; j < len(days); j++ {
			if days[i] > days[j] {
				days[i], days[j] = days[j], days[i]
			}
		}
	}
	for idx, day := range days {
		runningPnl += dailyPnl[day]
		if runningPnl > peak {
			peak = runningPnl
		}
		dd := 0.0
		if peak > 0 {
			dd = (peak - runningPnl) / peak * 100
		}
		if dd > 15 && idx+1 < len(days) {
			// Check if next day has fewer trades
			curTrades := 0
			for _, t := range trades {
				if t.CloseTime.Format("2006-01-02") == day {
					curTrades++
				}
			}
			nextTrades := 0
			for _, t := range trades {
				if t.CloseTime.Format("2006-01-02") == days[idx+1] {
					nextTrades++
				}
			}
			if nextTrades < curTrades {
				pausedAfterDD = true
			}
		}
	}
	if pausedAfterDD {
		score += 15
		reasons = append(reasons, "pauses after DD")
	}

	// 2. Lot reduction after losses (+10)
	// Check if trader reduces lot size after losing trades
	lotReduction := false
	for i := 1; i < len(trades); i++ {
		if trades[i-1].Profit < 0 && trades[i].Lots < trades[i-1].Lots {
			lotReduction = true
			break
		}
	}
	if lotReduction {
		score += 10
		reasons = append(reasons, "lot reduction after loss")
	}

	// 3. Consistent position sizing (+10)
	// Low variance in lot sizes = consistent
	if len(trades) >= 10 {
		meanLot := 0.0
		for _, t := range trades {
			meanLot += t.Lots
		}
		meanLot /= float64(len(trades))
		varLot := 0.0
		for _, t := range trades {
			diff := t.Lots - meanLot
			varLot += diff * diff
		}
		varLot /= float64(len(trades))
		cvLot := 0.0
		if meanLot > 0 {
			cvLot = math.Sqrt(varLot) / meanLot * 100
		}
		if cvLot < 30 {
			score += 10
			reasons = append(reasons, fmt.Sprintf("consistent sizing CV=%.0f%%", cvLot))
		}
	}

	// 4. Revenge trading penalty (-25)
	// Lot spike >100% immediately after large loss
	revengeCount := 0
	for i := 1; i < len(trades); i++ {
		if trades[i-1].Profit < 0 && trades[i].Lots > trades[i-1].Lots*2 {
			revengeCount++
		}
	}
	if revengeCount >= 3 {
		score -= 25
		reasons = append(reasons, fmt.Sprintf("revenge trading %dx", revengeCount))
	} else if revengeCount >= 1 {
		score -= 10
		reasons = append(reasons, fmt.Sprintf("revenge trading %dx", revengeCount))
	}

	// 5. Overtrading penalty (-15)
	// Many trades in one day during loss
	dailyTrades := make(map[string][]TradeData)
	for _, t := range trades {
		day := t.CloseTime.Format("2006-01-02")
		dailyTrades[day] = append(dailyTrades[day], t)
	}
	overtradeDays := 0
	for _, dayT := range dailyTrades {
		if len(dayT) >= 5 {
			dayProfit := 0.0
			for _, t := range dayT {
				dayProfit += t.Profit
			}
			if dayProfit < 0 {
				overtradeDays++
			}
		}
	}
	if overtradeDays >= 3 {
		score -= 15
		reasons = append(reasons, fmt.Sprintf("overtrading %dd", overtradeDays))
	}

	if score > 100 {
		score = 100
	}
	if score < 0 {
		score = 0
	}

	pillar.Score = score
	if len(reasons) > 0 {
		pillar.Reason = reasons[0]
		for i := 1; i < len(reasons); i++ {
			pillar.Reason += ", " + reasons[i]
		}
	} else {
		pillar.Reason = "No behavioral issues detected"
	}
	return pillar
}

// P7: Track Record (7%) - With inactive penalty
func CalculateP7(totalTrades, daysSinceStart int, lastTradeTime time.Time) PillarScore {
	pillar := PillarScore{
		Code:   "P7",
		Name:   "Track Record",
		Weight: 7,
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
