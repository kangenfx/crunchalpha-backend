package alpharank

import (
	"math"
)

// RegimeType represents market regime
type RegimeType string

const (
	RegimeTrending RegimeType = "TRENDING"
	RegimeRanging  RegimeType = "RANGING"
	RegimeVolatile RegimeType = "VOLATILE"
	RegimeMixed    RegimeType = "MIXED"
)

// RegimeDetection result
type RegimeDetection struct {
	CurrentRegime  RegimeType `json:"current_regime"`
	Confidence     float64    `json:"confidence"`
	TrendingPct    float64    `json:"trending_pct"`
	RangingPct     float64    `json:"ranging_pct"`
	VolatilePct    float64    `json:"volatile_pct"`
	Description    string     `json:"description"`
	Recommendation string     `json:"recommendation"`
}

// DetectRegime analyzes trading patterns to determine market regime
func DetectRegime(metrics AccountMetrics) RegimeDetection {
	if len(metrics.Trades) < 20 {
		return RegimeDetection{
			CurrentRegime:  RegimeMixed,
			Confidence:     0,
			Description:    "Insufficient data for regime detection",
			Recommendation: "Need at least 20 trades",
		}
	}

	// Analyze win/loss patterns
	trendingScore := calculateTrendingScore(metrics)
	rangingScore := calculateRangingScore(metrics)
	volatileScore := calculateVolatileScore(metrics)

	// Normalize to percentages
	total := trendingScore + rangingScore + volatileScore
	if total == 0 {
		total = 1
	}

	trendingPct := (trendingScore / total) * 100
	rangingPct := (rangingScore / total) * 100
	volatilePct := (volatileScore / total) * 100

	// Determine dominant regime
	var regime RegimeType
	var confidence float64
	var description string
	var recommendation string

	maxScore := math.Max(trendingPct, math.Max(rangingPct, volatilePct))

	if maxScore == trendingPct {
		regime = RegimeTrending
		confidence = trendingPct
		description = "Market shows strong directional movement"
		recommendation = "Focus on trend-following strategies, use trailing stops"
	} else if maxScore == rangingPct {
		regime = RegimeRanging
		confidence = rangingPct
		description = "Market is range-bound with support/resistance"
		recommendation = "Use mean-reversion strategies, take profits at key levels"
	} else {
		regime = RegimeVolatile
		confidence = volatilePct
		description = "Market shows high volatility and unpredictability"
		recommendation = "Reduce position sizes, widen stops, avoid overtrading"
	}

	// If no clear dominant regime
	if maxScore < 50 {
		regime = RegimeMixed
		confidence = maxScore
		description = "Mixed market conditions, no clear regime"
		recommendation = "Use adaptive strategies, be cautious with entries"
	}

	return RegimeDetection{
		CurrentRegime:  regime,
		Confidence:     confidence,
		TrendingPct:    trendingPct,
		RangingPct:     rangingPct,
		VolatilePct:    volatilePct,
		Description:    description,
		Recommendation: recommendation,
	}
}

// calculateTrendingScore analyzes if trades follow trends
func calculateTrendingScore(metrics AccountMetrics) float64 {
	score := 0.0

	// Calculate average trade duration
	var totalDuration float64
	for _, trade := range metrics.Trades {
		duration := trade.CloseTime.Sub(trade.OpenTime).Hours()
		totalDuration += duration
	}
	avgDuration := totalDuration / float64(len(metrics.Trades))

	// Longer average duration = more trending
	if avgDuration > 24 {
		score += 30 // Holding for days = trend following
	} else if avgDuration > 4 {
		score += 15 // Several hours = swing trading
	}

	// Check for consecutive wins/losses (trend runs)
	consecutiveWins := 0
	consecutiveLosses := 0
	maxConsecutiveWins := 0
	maxConsecutiveLosses := 0

	for _, trade := range metrics.Trades {
		netProfit := trade.Profit + trade.Swap + trade.Commission
		if netProfit > 0 {
			consecutiveWins++
			consecutiveLosses = 0
			if consecutiveWins > maxConsecutiveWins {
				maxConsecutiveWins = consecutiveWins
			}
		} else {
			consecutiveLosses++
			consecutiveWins = 0
			if consecutiveLosses > maxConsecutiveLosses {
				maxConsecutiveLosses = consecutiveLosses
			}
		}
	}

	// Long winning/losing streaks indicate trending
	if maxConsecutiveWins >= 5 || maxConsecutiveLosses >= 5 {
		score += 40
	} else if maxConsecutiveWins >= 3 || maxConsecutiveLosses >= 3 {
		score += 20
	}

	// Check if big wins are bigger than big losses (trend capture)
	if metrics.GrossProfit > math.Abs(metrics.GrossLoss)*1.5 {
		score += 30
	}

	return score
}

// calculateRangingScore analyzes mean-reversion patterns
func calculateRangingScore(metrics AccountMetrics) float64 {
	score := 0.0

	// Quick in-and-out trades = ranging/scalping
	var totalDuration float64
	quickTrades := 0
	for _, trade := range metrics.Trades {
		duration := trade.CloseTime.Sub(trade.OpenTime).Hours()
		totalDuration += duration
		if duration < 1 {
			quickTrades++
		}
	}
	avgDuration := totalDuration / float64(len(metrics.Trades))

	if avgDuration < 1 {
		score += 40 // Scalping = ranging
	} else if avgDuration < 4 {
		score += 20 // Intraday = possible ranging
	}

	// High win rate with small avg win = ranging/mean reversion
	winRate := float64(metrics.WinningTrades) / float64(metrics.TotalTrades) * 100
	avgWin := metrics.GrossProfit / float64(metrics.WinningTrades)
	avgLoss := math.Abs(metrics.GrossLoss) / float64(metrics.LosingTrades)

	if winRate > 60 && avgWin < avgLoss*2 {
		score += 30 // High win rate, small wins = ranging
	}

	// Consistent lot sizes = systematic ranging strategy
	var lotSizes []float64
	for _, trade := range metrics.Trades {
		lotSizes = append(lotSizes, trade.Lots)
	}
	lotStdDev := calculateStdDev(lotSizes)
	lotMean := calculateMean(lotSizes)
	
	if lotMean > 0 && lotStdDev/lotMean < 0.3 {
		score += 30 // Consistent sizing = disciplined ranging
	}

	return score
}

// calculateVolatileScore analyzes erratic patterns
func calculateVolatileScore(metrics AccountMetrics) float64 {
	score := 0.0

	// Large drawdown = volatile
	if metrics.MaxDrawdownPct > 50 {
		score += 50
	} else if metrics.MaxDrawdownPct > 30 {
		score += 30
	}

	// Low win rate = volatile/gambling
	winRate := float64(metrics.WinningTrades) / float64(metrics.TotalTrades) * 100
	if winRate < 40 {
		score += 30
	}

	// Erratic lot sizing = volatile
	var lotSizes []float64
	for _, trade := range metrics.Trades {
		lotSizes = append(lotSizes, trade.Lots)
	}
	lotStdDev := calculateStdDev(lotSizes)
	lotMean := calculateMean(lotSizes)
	
	if lotMean > 0 && lotStdDev/lotMean > 0.5 {
		score += 40 // Erratic lot sizes
	}

	// Big swings in P/L = volatile
	avgWin := metrics.GrossProfit / float64(metrics.WinningTrades)
	avgLoss := math.Abs(metrics.GrossLoss) / float64(metrics.LosingTrades)
	
	if avgLoss > avgWin*3 {
		score += 30 // Huge losses compared to wins
	}

	return score
}

func calculateMean(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

func calculateStdDev(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	mean := calculateMean(values)
	variance := 0.0
	for _, v := range values {
		variance += math.Pow(v-mean, 2)
	}
	return math.Sqrt(variance / float64(len(values)))
}
