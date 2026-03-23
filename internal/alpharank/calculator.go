package alpharank

import (
	"time"
)

type Calculator struct{}

func NewCalculator() *Calculator {
	return &Calculator{}
}

func (c *Calculator) Calculate(metrics AccountMetrics) AlphaRankResult {
	result := AlphaRankResult{
		CalculatedAt: time.Now(),
	}

	// Use peak balance for tier (not current balance which could be $0.28!)
	tierBalance := metrics.PeakBalance
	if tierBalance <= 0 {
		tierBalance = metrics.CurrentBalance
	}
	result.Tier = c.GetCapitalTier(tierBalance)

	flags := DetectRiskFlags(metrics)
	result.RiskFlags.Items = flags

	for _, flag := range flags {
		switch flag.Severity {
		case "CRITICAL":
			result.RiskFlags.Counts.Critical++
		case "MAJOR":
			result.RiskFlags.Counts.Major++
		case "MINOR":
			result.RiskFlags.Counts.Minor++
		}
	}

	result.Pillars = c.Calculate7Pillars(metrics, flags)
	result.AlphaScore = c.CalculateAlphaScore(result.Pillars)
	result.Grade = c.GetGrade(result.AlphaScore)
	result.Risk = c.GetRiskLevel(result.RiskFlags.Counts.Critical, result.RiskFlags.Counts.Major, result.RiskFlags.Counts.Minor, result.AlphaScore)

	surv := c.CalculateSurvivability(metrics.MaxDrawdownPct, result.AlphaScore)
	result.Survivability.Score = surv.Score
	result.Survivability.Label = surv.Label
	result.Survivability.Note = surv.Note

	scal := c.CalculateScalability(metrics.PeakBalance, result.AlphaScore)
	result.Scalability.Score = scal.Score
	result.Scalability.Label = scal.Label
	result.Scalability.Note = scal.Note

	return result
}

func (c *Calculator) GetCapitalTier(balance float64) string {
	switch {
	case balance >= 1_000_000:
		return "INSTITUTIONAL"
	case balance >= 100_000:
		return "PROFESSIONAL"
	case balance >= 10_000:
		return "SEMI-PRO"
	case balance >= 1_000:
		return "RETAIL"
	default:
		return "MICRO"
	}
}

func (c *Calculator) Calculate7Pillars(metrics AccountMetrics, flags []RiskFlag) []PillarScore {
	pillars := make([]PillarScore, 7)

	// P1: Use totalDeposits as base, fallback to initialDeposit
	depositBase := metrics.TotalDeposits
	if depositBase <= 0 {
		depositBase = metrics.InitialDeposit
	}
	pillars[0] = CalculateP1(metrics.NetProfit, depositBase, metrics.MaxDrawdownPct)

	// P2: Consistency with year+week fix + inactive weeks
	pillars[1] = CalculateP2(metrics.Trades)

	// P3: Risk flags
	pillars[2] = CalculateP3(flags)

	// P4: Recovery
	pillars[3] = CalculateP4(metrics.Trades, metrics.MaxDrawdownPct, metrics.NetProfit/metrics.TotalDeposits*100)

	// P5: Edge
	winRate := 0.0
	if metrics.TotalTrades > 0 {
		winRate = float64(metrics.WinningTrades) / float64(metrics.TotalTrades) * 100
	}
	if metrics.GrossLoss != 0 {
	}
	pillars[4] = CalculateP5(metrics.Trades, winRate, metrics.AvgWin, metrics.AvgLoss)

	// P6: Discipline
	pillars[5] = CalculateP6(metrics.Trades)

	// P7: Track record with lastTradeTime for inactive penalty
	days := 0
	lastTradeTime := time.Now()
	if !metrics.StartDate.IsZero() {
		days = int(time.Since(metrics.StartDate).Hours() / 24)
	}
	if !metrics.EndDate.IsZero() {
		lastTradeTime = metrics.EndDate
	}
	pillars[6] = CalculateP7(metrics.TotalTrades, days, lastTradeTime)

	return pillars
}

func (c *Calculator) CalculateAlphaScore(pillars []PillarScore) float64 {
	totalScore := 0.0
	totalWeight := 0.0

	for _, p := range pillars {
		weight := float64(p.Weight) / 100.0
		totalScore += p.Score * weight
		totalWeight += weight
	}

	if totalWeight == 0 {
		return 0
	}

	return totalScore / totalWeight
}

func (c *Calculator) GetGrade(score float64) string {
	switch {
	case score >= 95:
		return "A+"
	case score >= 90:
		return "A"
	case score >= 85:
		return "A-"
	case score >= 80:
		return "B+"
	case score >= 75:
		return "B"
	case score >= 70:
		return "B-"
	case score >= 65:
		return "C+"
	case score >= 60:
		return "C"
	case score >= 55:
		return "C-"
	case score >= 50:
		return "D+"
	case score >= 45:
		return "D"
	default:
		return "F"
	}
}

func (c *Calculator) GetRiskLevel(critical, major, minor int, alphaScore float64) string {
	totalFlags := critical + major + minor
	if critical > 0 || alphaScore < 30 {
		return "EXTREME"
	}
	if totalFlags >= 3 || (alphaScore >= 30 && alphaScore < 50) {
		return "HIGH"
	}
	if totalFlags == 2 || (alphaScore >= 50 && alphaScore < 70) {
		return "MEDIUM"
	}
	if critical == 0 && totalFlags <= 1 && alphaScore >= 70 {
		return "LOW"
	}
	return "MEDIUM"
}

type SurvScalResult struct {
	Score int
	Label string
	Note  string
}

func (c *Calculator) CalculateSurvivability(maxDD, alphaScore float64) SurvScalResult {
	score := int(alphaScore)

	if maxDD > 50 {
		score -= 30
	} else if maxDD > 30 {
		score -= 15
	}

	if score < 0 {
		score = 0
	}

	label := ""
	switch {
	case score >= 80:
		label = "Excellent"
	case score >= 60:
		label = "Good"
	case score >= 40:
		label = "Moderate"
	default:
		label = "Poor"
	}

	return SurvScalResult{
		Score: score,
		Label: label,
		Note:  "0–100 score (retail transparency)",
	}
}

func (c *Calculator) CalculateScalability(peakBalance, alphaScore float64) SurvScalResult {
	score := int(alphaScore * 0.7)

	if peakBalance >= 100_000 {
		score += 20
	} else if peakBalance >= 10_000 {
		score += 10
	} else if peakBalance >= 1_000 {
		score += 5
	}

	if score < 0 {
		score = 0
	}
	if score > 100 {
		score = 100
	}

	label := ""
	switch {
	case score >= 80:
		label = "Excellent"
	case score >= 60:
		label = "Good"
	case score >= 40:
		label = "Moderate"
	default:
		label = "Limited"
	}

	return SurvScalResult{
		Score: score,
		Label: label,
		Note:  "Capacity & growth readiness",
	}
}
