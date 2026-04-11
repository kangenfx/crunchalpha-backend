package alpharank

import (
	"database/sql"
	"encoding/json"
	"math"
	"time"
)

// Layer3Status represents the system intelligence status
type Layer3Status string

const (
	Layer3Reduce  Layer3Status = "REDUCE"
	Layer3Neutral Layer3Status = "NEUTRAL"
	Layer3Watch   Layer3Status = "WATCH"
)

// Layer3Detail breakdown per modul — disimpan ke DB sebagai JSON
type Layer3Detail struct {
	BehaviorScore   float64  `json:"behavior_score"`
	VolatilityScore float64  `json:"volatility_score"`
	DDScore         float64  `json:"dd_score"`
	BehaviorReasons []string `json:"behavior_reasons"`
	VolReasons      []string `json:"vol_reasons"`
	DDReasons       []string `json:"dd_reasons"`
	RegimeDetected  string   `json:"regime_detected"`
	SystemMode      string   `json:"system_mode"`
	SoftReasons     []string `json:"soft_reasons"`
	CalculatedAt    string   `json:"calculated_at"`
}

// Layer3Result final output
type Layer3Result struct {
	Multiplier float64      `json:"multiplier"`
	Status     Layer3Status `json:"status"`
	Reason     string       `json:"reason"`
	Detail     Layer3Detail `json:"detail"`
}

// CalculateLayer3 — main entry point, semua dari DB via metrics
// Dipanggil saat recalculate, hasil disimpan ke alpha_ranks
func CalculateLayer3(metrics AccountMetrics, maxDrawdownPct float64, activeFlags int) Layer3Result {
	detail := Layer3Detail{
		CalculatedAt: time.Now().Format(time.RFC3339),
	}

	// === MODUL 1: Behavior Shift Intelligence ===
	behaviorScore, behaviorReasons := calculateBehaviorScore(metrics)
	detail.BehaviorScore = behaviorScore
	detail.BehaviorReasons = behaviorReasons

	// === MODUL 2: Market Regime Detection (Volatility Proxy) ===
	volScore, volReasons, regimeDetected := calculateVolatilityScore(metrics)
	detail.VolatilityScore = volScore
	detail.VolReasons = volReasons
	detail.RegimeDetected = regimeDetected

	// === MODUL 3: Adaptive Position Scaling (DD-based) ===
	ddScore, ddReasons := calculateDDScore(maxDrawdownPct, activeFlags)
	detail.DDScore = ddScore
	detail.DDReasons = ddReasons

	// === FINAL MULTIPLIER = M1 × M2 × M3 ===
	multiplier := behaviorScore * volScore * ddScore

	// Cap: min 0.30, max 1.00 — tidak pernah expand lot
	if multiplier < 0.30 {
		multiplier = 0.30
	}
	if multiplier > 1.00 {
		multiplier = 1.00
	}
	// Round 4 decimal
	multiplier = math.Round(multiplier*10000) / 10000

	// Determine status
	status := Layer3Neutral
	reason := "All systems normal — full position size active"

	if multiplier <= 0.60 {
		status = Layer3Reduce
		reason = buildReason(behaviorReasons, volReasons, ddReasons, "REDUCE")
	} else if multiplier <= 0.85 {
		status = Layer3Watch
		reason = buildReason(behaviorReasons, volReasons, ddReasons, "WATCH")
	}

	systemMode := GetSystemMode(multiplier)
	softReasons := GetSoftReasons(detail.BehaviorReasons, detail.VolReasons, detail.DDReasons)
	detail.SystemMode = systemMode
	detail.SoftReasons = softReasons

	return Layer3Result{
		Multiplier: multiplier,
		Status:     status,
		Reason:     reason,
		Detail:     detail,
	}
}

// === MODUL 1: Behavior Shift Intelligence ===
// Deteksi perubahan perilaku trader vs baseline historisnya
func calculateBehaviorScore(metrics AccountMetrics) (float64, []string) {
	score := 1.0
	var reasons []string

	if len(metrics.Trades) < 40 {
		return 1.0, nil
	}

	// Split trades: recent 20% vs baseline 80%
	splitIdx := len(metrics.Trades) * 8 / 10
	if splitIdx < 5 {
		splitIdx = 5
	}
	baselineTrades := metrics.Trades[:splitIdx]
	recentTrades := metrics.Trades[splitIdx:]

	if len(recentTrades) == 0 {
		return 1.0, nil
	}

	// --- B1: Lot spike detection ---
	baselineLots := make([]float64, len(baselineTrades))
	for i, t := range baselineTrades {
		baselineLots[i] = t.Lots
	}
	recentLots := make([]float64, len(recentTrades))
	for i, t := range recentTrades {
		recentLots[i] = t.Lots
	}
	baselineAvgLot := calculateMean(baselineLots)
	recentAvgLot := calculateMean(recentLots)

	if baselineAvgLot > 0 {
		lotRatio := recentAvgLot / baselineAvgLot
		if lotRatio >= 2.5 {
			score *= 0.70
			reasons = append(reasons, "Lot size spike: recent avg 2.5x+ baseline")
		} else if lotRatio >= 1.8 {
			score *= 0.85
			reasons = append(reasons, "Lot size elevated: recent avg 1.8x baseline")
		}
	}

	// --- B2: Win rate deterioration ---
	baselineWins := 0
	for _, t := range baselineTrades {
		if t.Profit+t.Swap+t.Commission > 0 {
			baselineWins++
		}
	}
	recentWins := 0
	for _, t := range recentTrades {
		if t.Profit+t.Swap+t.Commission > 0 {
			recentWins++
		}
	}
	baselineWR := float64(baselineWins) / float64(len(baselineTrades)) * 100
	recentWR := float64(recentWins) / float64(len(recentTrades)) * 100

	if baselineWR > 0 {
		wrDrop := baselineWR - recentWR
		if wrDrop >= 30 {
			score *= 0.75
			reasons = append(reasons, "Win rate dropped 30%+ from baseline")
		} else if wrDrop >= 20 {
			score *= 0.88
			reasons = append(reasons, "Win rate dropped 20%+ from baseline")
		}
	}

	// --- B3: SL skip streak in recent trades ---
	slSkipCount := 0
	for _, t := range recentTrades {
		if t.StopLoss == 0 && t.Profit+t.Swap+t.Commission < 0 {
			slSkipCount++
		}
	}
	slSkipPct := float64(slSkipCount) / float64(len(recentTrades)) * 100
	if slSkipPct >= 70 {
		score *= 0.80
		reasons = append(reasons, "Recent trades: 70%+ losses without SL")
	} else if slSkipPct >= 50 {
		score *= 0.90
		reasons = append(reasons, "Recent trades: 50%+ losses without SL")
	}

	// --- B4: Erratic lot sizing (std dev / mean) ---
	allLots := make([]float64, len(metrics.Trades))
	for i, t := range metrics.Trades {
		allLots[i] = t.Lots
	}
	lotMean := calculateMean(allLots)
	lotStdDev := calculateStdDev(allLots)
	if lotMean > 0 {
		cv := lotStdDev / lotMean
		if cv >= 1.0 {
			score *= 0.80
			reasons = append(reasons, "Highly erratic lot sizing (CV >= 1.0)")
		} else if cv >= 0.6 {
			score *= 0.90
			reasons = append(reasons, "Inconsistent lot sizing (CV >= 0.6)")
		}
	}

	if score > 1.0 {
		score = 1.0
	}
	return math.Round(score*10000) / 10000, reasons
}

// === MODUL 2: Market Regime Detection (Volatility Proxy) ===
// Tanpa ATR — pakai |open_price - close_price| dari trade data
func calculateVolatilityScore(metrics AccountMetrics) (float64, []string, string) {
	score := 1.0
	var reasons []string
	regime := "NORMAL"

	if len(metrics.Trades) < 40 {
		return 1.0, nil, regime
	}

	// Proxy volatility = |open_price - close_price| / open_price (% move per trade)
	allMoves := make([]float64, 0, len(metrics.Trades))
	for _, t := range metrics.Trades {
		if t.OpenPrice > 0 {
			move := math.Abs(t.ClosePrice-t.OpenPrice) / t.OpenPrice * 100
			allMoves = append(allMoves, move)
		}
	}

	if len(allMoves) < 10 {
		return 1.0, nil, regime
	}

	// Split: recent 30% vs baseline 70%
	splitIdx := len(allMoves) * 7 / 10
	if splitIdx < 5 {
		splitIdx = 5
	}
	baselineMoves := allMoves[:splitIdx]
	recentMoves := allMoves[splitIdx:]

	baselineVolatility := calculateMean(baselineMoves)
	recentVolatility := calculateMean(recentMoves)

	if baselineVolatility > 0 && len(recentMoves) > 0 {
		volRatio := recentVolatility / baselineVolatility

		if volRatio >= 2.5 {
			score *= 0.60
			regime = "EXTREME_VOLATILE"
			reasons = append(reasons, "Market volatility 2.5x+ above baseline")
		} else if volRatio >= 1.8 {
			score *= 0.75
			regime = "HIGH_VOLATILE"
			reasons = append(reasons, "Market volatility 1.8x above baseline")
		} else if volRatio >= 1.4 {
			score *= 0.88
			regime = "ELEVATED_VOLATILE"
			reasons = append(reasons, "Market volatility 1.4x above baseline")
		} else if volRatio <= 0.5 {
			// Very low volatility — normal, no penalty
			regime = "LOW_VOLATILE"
		}
	}

	// Tambahan: loss streak di recent trades → regime buruk
	recentTrades := metrics.Trades[len(metrics.Trades)*7/10:]
	if len(recentTrades) >= 4 {
		consecutiveLoss := 0
		maxConsLoss := 0
		for _, t := range recentTrades {
			if t.Profit+t.Swap+t.Commission < 0 {
				consecutiveLoss++
				if consecutiveLoss > maxConsLoss {
					maxConsLoss = consecutiveLoss
				}
			} else {
				consecutiveLoss = 0
			}
		}
		if maxConsLoss >= 5 {
			score *= 0.80
			reasons = append(reasons, "Recent loss streak: 5+ consecutive losses")
		} else if maxConsLoss >= 3 {
			score *= 0.90
			reasons = append(reasons, "Recent loss streak: 3+ consecutive losses")
		}
	}

	if score > 1.0 {
		score = 1.0
	}
	return math.Round(score*10000) / 10000, reasons, regime
}

// === MODUL 3: Adaptive Position Scaling (DD-based) ===
// Berdasarkan max_drawdown_pct dari DB — zero on-the-fly
func calculateDDScore(maxDrawdownPct float64, activeFlags int) (float64, []string) {
	score := 1.0
	var reasons []string

	// DD tiers
	switch {
	case maxDrawdownPct >= 50:
		score *= 0.50
		reasons = append(reasons, "Drawdown >= 50% — critical protection active")
	case maxDrawdownPct >= 35:
		score *= 0.65
		reasons = append(reasons, "Drawdown >= 35% — significant risk reduction")
	case maxDrawdownPct >= 25:
		score *= 0.75
		reasons = append(reasons, "Drawdown >= 25% — moderate risk reduction")
	case maxDrawdownPct >= 15:
		score *= 0.88
		reasons = append(reasons, "Drawdown >= 15% — mild risk reduction")
	}

	// Active critical/major flags tambah penalti
	if activeFlags >= 5 {
		score *= 0.75
		reasons = append(reasons, "5+ active risk flags detected")
	} else if activeFlags >= 3 {
		score *= 0.85
		reasons = append(reasons, "3+ active risk flags detected")
	} else if activeFlags >= 1 {
		score *= 0.95
		reasons = append(reasons, "Active risk flags present")
	}

	if score > 1.0 {
		score = 1.0
	}
	return math.Round(score*10000) / 10000, reasons
}

// buildReason — compile human-readable reason string
func buildReason(behaviorReasons, volReasons, ddReasons []string, level string) string {
	all := append(append(behaviorReasons, volReasons...), ddReasons...)
	if len(all) == 0 {
		return "System monitoring active"
	}
	result := ""
	for i, r := range all {
		if i > 0 {
			result += "; "
		}
		result += r
		if i >= 2 {
			break // max 3 reasons in string
		}
	}
	return result
}

// SaveLayer3ToDBAtomic — update layer3 kolom di alpha_ranks
// Dipanggil setelah saveAlphaRankWithMetrics selesai
func SaveLayer3ToDB(db *sql.DB, accountID string, result Layer3Result) error {
	detailJSON, err := json.Marshal(result.Detail)
	if err != nil {
		return err
	}

	_, err = db.Exec(`
		UPDATE alpha_ranks
		SET
			layer3_multiplier    = $1,
			layer3_status        = $2,
			layer3_reason        = $3,
			layer3_detail        = $4,
			layer3_calculated_at = NOW()
		WHERE account_id = $5
		AND symbol = 'ALL'
	`, result.Multiplier, string(result.Status), result.Reason, detailJSON, accountID)
	return err
}

// GetSystemMode — convert multiplier ke user-friendly mode
func GetSystemMode(multiplier float64) string {
	switch {
	case multiplier >= 0.90:
		return "FULL_ACTIVE"
	case multiplier >= 0.75:
		return "MONITORING"
	case multiplier >= 0.55:
		return "DEFENSIVE"
	default:
		return "PROTECTED"
	}
}

// GetSystemModeMessage — investor-friendly message
func GetSystemModeMessage(mode string, reasons []string) string {
	switch mode {
	case "FULL_ACTIVE":
		return "System running at full capacity"
	case "MONITORING":
		return "System detecting early signals — position size slightly adjusted"
	case "DEFENSIVE":
		return "Unusual patterns detected — system reducing exposure to protect your capital"
	default:
		return "High risk detected — system has reduced position size to minimum to protect your funds"
	}
}

// GetSoftReasons — convert technical reasons ke investor-friendly language
func GetSoftReasons(behaviorReasons, volReasons, ddReasons []string) []string {
	softMap := map[string]string{
		"Lot size spike: recent avg 2.5x+ baseline":  "Unusual position sizing detected",
		"Lot size elevated: recent avg 1.8x baseline": "Position size above normal range",
		"Win rate dropped 30%+ from baseline":         "Recent performance declining significantly",
		"Win rate dropped 20%+ from baseline":         "Recent performance below historical average",
		"Recent trades: 70%+ losses without SL":       "Limited risk protection on recent trades",
		"Recent trades: 50%+ losses without SL":       "Limited risk protection on recent trades",
		"Highly erratic lot sizing (CV >= 1.0)":       "Inconsistent trading behavior",
		"Inconsistent lot sizing (CV >= 0.6)":         "Slightly inconsistent trading behavior",
		"Market volatility 2.5x+ above baseline":      "Extreme market conditions detected",
		"Market volatility 1.8x above baseline":       "High market volatility detected",
		"Market volatility 1.4x above baseline":       "Elevated market volatility",
		"Recent loss streak: 5+ consecutive losses":   "Recent performance declining",
		"Recent loss streak: 3+ consecutive losses":   "Short-term performance under pressure",
		"Drawdown >= 50% — critical protection active": "Significant drawdown detected",
		"Drawdown >= 35% — significant risk reduction": "Elevated drawdown level",
		"Drawdown >= 25% — moderate risk reduction":    "Moderate drawdown detected",
		"Drawdown >= 15% — mild risk reduction":        "Mild drawdown present",
		"5+ active risk flags detected":               "Multiple risk indicators active",
		"3+ active risk flags detected":               "Several risk indicators active",
		"Active risk flags present":                   "Risk indicator active",
	}

	seen := map[string]bool{}
	var soft []string
	all := append(append(behaviorReasons, volReasons...), ddReasons...)
	for _, r := range all {
		msg, ok := softMap[r]
		if !ok {
			msg = r
		}
		if !seen[msg] {
			seen[msg] = true
			soft = append(soft, msg)
		}
		if len(soft) >= 3 {
			break
		}
	}
	return soft
}
