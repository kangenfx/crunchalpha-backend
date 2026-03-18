package trader

import (
	"encoding/json"
	"database/sql"
	"errors"
	"fmt"

	"crunchalpha-v3/internal/alpharank"
)

// GetDashboardWithAlphaRank returns dashboard with AlphaRank from database
func (s *Service) GetDashboardWithAlphaRank(accountID, userID string) (*DashboardResponse, error) {
	account, err := s.repo.GetAccountByID(accountID, userID)
	if err == sql.ErrNoRows {
		return nil, errors.New("account not found")
	}
	if err != nil {
		return nil, err
	}

	brokerLabel := fmt.Sprintf("%s • %s • %s — Account %s (%s)",
		account.Broker, account.Platform, account.Currency,
		account.AccountNumber, account.Status)

	// Get AlphaRank from database (SINGLE SOURCE OF TRUTH!)
	var result struct {
		P1, P2, P3, P4, P5, P6, P7 float64
		AlphaScore                 float64
		Grade                      string
		Badge                      string
		TradeCount                 int
		RiskFlags                  []byte // jsonb
		CriticalCount              int
		MajorCount                 int
		MinorCount                 int
	}

	err = s.repo.db.QueryRow(`
		SELECT
			profitability_score, risk_score, consistency_score,
			stability_score, activity_score, duration_score, drawdown_score,
			alpha_score, grade, badge, trade_count,
			COALESCE(risk_flags, '[]'::jsonb),
			COALESCE(critical_count, 0),
			COALESCE(major_count, 0),
			COALESCE(minor_count, 0)
		FROM alpha_ranks
		WHERE account_id = $1 AND symbol = 'ALL'
	`, accountID).Scan(
		&result.P1, &result.P2, &result.P3, &result.P4,
		&result.P5, &result.P6, &result.P7,
		&result.AlphaScore, &result.Grade, &result.Badge, &result.TradeCount,
		&result.RiskFlags,
		&result.CriticalCount, &result.MajorCount, &result.MinorCount,
	)

	if err != nil {
		return nil, fmt.Errorf("AlphaRank not calculated yet for this account")
	}

	// Parse risk flags from JSON
	var flags []alpharank.RiskFlag
	if len(result.RiskFlags) > 0 {
		json.Unmarshal(result.RiskFlags, &flags)
	}

	// Get account balance/equity
	var balance, equity float64
	s.repo.db.QueryRow(`
		SELECT COALESCE(balance, 0), COALESCE(equity, 0)
		FROM trader_accounts WHERE id = $1
	`, accountID).Scan(&balance, &equity)

	// Read ALL performance data from DB - single source of truth
	var pm struct {
		TotalTrades   int
		WinningTrades int
		LosingTrades  int
		WinRate       float64
		ProfitFactor  float64
		MaxDD         float64
		AvgWin        float64
		AvgLoss       float64
		RiskReward    float64
		Expectancy    float64
	}

	err = s.repo.db.QueryRow(`
		SELECT
			COALESCE(total_trades, 0),
			COALESCE(winning_trades, 0),
			COALESCE(losing_trades, 0),
			COALESCE(win_rate, 0),
			COALESCE(profit_factor, 0),
			COALESCE(max_drawdown_pct, 0),
			COALESCE(avg_win, 0),
			COALESCE(avg_loss, 0),
			COALESCE(reward_risk_ratio, 0),
			COALESCE(expectancy, 0)
		FROM performance_metrics
		WHERE account_id = $1 AND period = 'ALL' AND symbol = 'ALL'
	`, accountID).Scan(
		&pm.TotalTrades, &pm.WinningTrades, &pm.LosingTrades,
		&pm.WinRate, &pm.ProfitFactor, &pm.MaxDD,
		&pm.AvgWin, &pm.AvgLoss, &pm.RiskReward, &pm.Expectancy,
	)
	if err != nil {
		return nil, fmt.Errorf("performance metrics not calculated yet, run recalculate first")
	}

	// Calculate survivability & scalability from DB values
	calculator := alpharank.NewCalculator()
	surv := calculator.CalculateSurvivability(pm.MaxDD, result.AlphaScore)
	scal := calculator.CalculateScalability(balance, result.AlphaScore)

	risk := "LOW"
	if result.CriticalCount > 0 {
		risk = "HIGH"
	} else if result.MajorCount >= 1 {
		risk = "MEDIUM"
	}

	tier := result.Badge
	if tier == "" {
		tier = getTierFromGrade(result.Grade)
	}

	winRate := pm.WinRate
	profitFactor := pm.ProfitFactor
	avgWin := pm.AvgWin
	avgLoss := pm.AvgLoss
	riskReward := pm.RiskReward
	expectancy := pm.Expectancy

	pillars := []Pillar{
		{Code: "P1", Name: "Profitability", Weight: 20, Score: int(result.P1)},
		{Code: "P2", Name: "Consistency", Weight: 20, Score: int(result.P2)},
		{Code: "P3", Name: "Risk Management", Weight: 25, Score: int(result.P3)},
		{Code: "P4", Name: "Recovery", Weight: 10, Score: int(result.P4)},
		{Code: "P5", Name: "Trading Edge", Weight: 10, Score: int(result.P5)},
		{Code: "P6", Name: "Discipline", Weight: 8, Score: int(result.P6)},
		{Code: "P7", Name: "Track Record", Weight: 7, Score: int(result.P7)},
	}

	// Build AlphaRankResult untuk convert
	alphaResult := alpharank.AlphaRankResult{
		AlphaScore: result.AlphaScore,
		Grade:      result.Grade,
		Tier:       tier,
		Risk:       risk,
	}
	alphaResult.RiskFlags.Counts.Critical = result.CriticalCount
	alphaResult.RiskFlags.Counts.Major = result.MajorCount
	alphaResult.RiskFlags.Counts.Minor = result.MinorCount
	alphaResult.RiskFlags.Items = flags

	dashboard := &DashboardResponse{
		Snapshot: DashboardSnapshot{
			Tier:         tier,
			Risk:         risk,
			Grade:        result.Grade,
			AlphaScore:   result.AlphaScore,
			Equity:       fmt.Sprintf("$%.2f", equity),
			Balance:      fmt.Sprintf("$%.2f", balance),
			PnLToday:     "+0.00 USD",
			TotalTrades:  pm.TotalTrades,
			WinRate:      winRate,
			ProfitFactor: profitFactor,
			MaxDD:        pm.MaxDD,
			AvgWin:       avgWin,
			AvgLoss:      avgLoss,
			RiskReward:   riskReward,
			Expectancy:   expectancy,
			BrokerLabel:  brokerLabel,
		},
		Survivability: SurvivabilityScore{
			Score: surv.Score,
			Label: surv.Label,
			Note:  surv.Note,
		},
		Scalability: ScalabilityScore{
			Score: scal.Score,
			Label: scal.Label,
			Note:  scal.Note,
		},
		Pillars:   pillars,
		RiskFlags: convertRiskFlags(alphaResult),
		Statistics: &MetricsStatistics{
			TotalTrades:   pm.TotalTrades,
			WinningTrades: pm.WinningTrades,
			LosingTrades:  pm.LosingTrades,
			WinRate:       winRate,
			ProfitFactor:  profitFactor,
			MaxDD:         pm.MaxDD,
			AvgWin:        avgWin,
			AvgLoss:       avgLoss,
			RiskReward:    riskReward,
			Expectancy:    expectancy,
		},
	}

	return dashboard, nil
}

func (s *Service) getTradesForAccount(accountID string) ([]alpharank.TradeData, error) {
	query := `
		SELECT
			symbol, type, lots, open_price, close_price,
			profit, swap, commission, open_time, close_time
		FROM trades
		WHERE account_id = $1 AND status = 'closed'
		ORDER BY close_time ASC
	`

	rows, err := s.repo.db.Query(query, accountID)
	if err != nil {
		return []alpharank.TradeData{}, nil
	}
	defer rows.Close()

	var trades []alpharank.TradeData
	for rows.Next() {
		var t alpharank.TradeData
		var openTime, closeTime sql.NullTime

		err := rows.Scan(
			&t.Symbol, &t.Type, &t.Lots, &t.OpenPrice, &t.ClosePrice,
			&t.Profit, &t.Swap, &t.Commission, &openTime, &closeTime,
		)
		if err != nil {
			continue
		}

		if openTime.Valid {
			t.OpenTime = openTime.Time
		}
		if closeTime.Valid {
			t.CloseTime = closeTime.Time
		}

		trades = append(trades, t)
	}

	return trades, nil
}

func (s *Service) buildMetrics(accountID string, trades []alpharank.TradeData, balance, equity float64, symbol string) alpharank.AccountMetrics {
	var (
		grossProfit   float64
		grossLoss     float64
		winningTrades int
		losingTrades  int
		totalProfit   float64
		maxDD         float64
		peak          float64
	)

	for _, trade := range trades {
		netProfit := trade.Profit + trade.Swap + trade.Commission
		totalProfit += netProfit

		if netProfit > 0 {
			grossProfit += netProfit
			winningTrades++
		} else {
			grossLoss += netProfit
			losingTrades++
		}
	}
	// Calculate MaxDD from trades (per-symbol, not global!)
	maxDD = 0.0
	if len(trades) > 0 {
		runningBalance := 0.0
		peak = 0.0
		
		for _, trade := range trades {
			netProfit := trade.Profit + trade.Swap + trade.Commission
			runningBalance += netProfit
			
			if runningBalance > peak {
				peak = runningBalance
			}
			
			if peak > 0 {
				// Cap DD at 100% (cannot lose more than peak)
				dd := (peak - runningBalance) / peak * 100
				if dd > 100 {
					dd = 100
				}
				if dd > maxDD {
					maxDD = dd
				}
			}
		}

	}
	

	return alpharank.AccountMetrics{
		AccountID:      accountID,
		CurrentBalance: balance,
		CurrentEquity:  equity,
		NetProfit:      totalProfit,
		GrossProfit:    grossProfit,
		GrossLoss:      grossLoss,
		PeakBalance:    peak,
		TotalTrades:    len(trades),
		WinningTrades:  winningTrades,
		LosingTrades:   losingTrades,
		MaxDrawdownPct: maxDD,
		Trades:         trades,
	}
}

func getTierFromGrade(grade string) string {
	switch grade {
	case "S+", "S", "S-":
		return "LEGEND"
	case "A+", "A", "A-":
		return "ELITE"
	case "B+", "B", "B-":
		return "PRO"
	case "C+", "C", "C-":
		return "INTERMEDIATE"
	case "D+", "D", "D-", "F":
		return "NOVICE"
	default:
		return "MICRO"
	}
}

// getTradesForSymbol returns trades for specific symbol
func (s *Service) getTradesForSymbol(accountID, symbol string) ([]alpharank.TradeData, error) {
	query := `
		SELECT
			symbol, type, lots, open_price, close_price,
			profit, swap, commission, open_time, close_time
		FROM trades
		WHERE account_id = $1 AND symbol = $2 AND status = 'closed'
		ORDER BY close_time ASC
	`

	rows, err := s.repo.db.Query(query, accountID, symbol)
	if err != nil {
		return []alpharank.TradeData{}, nil
	}
	defer rows.Close()

	var trades []alpharank.TradeData
	for rows.Next() {
		var t alpharank.TradeData
		var openTime, closeTime sql.NullTime

		err := rows.Scan(
			&t.Symbol, &t.Type, &t.Lots, &t.OpenPrice, &t.ClosePrice,
			&t.Profit, &t.Swap, &t.Commission, &openTime, &closeTime,
		)
		if err != nil {
			continue
		}

		if openTime.Valid {
			t.OpenTime = openTime.Time
		}
		if closeTime.Valid {
			t.CloseTime = closeTime.Time
		}

		trades = append(trades, t)
	}

	return trades, nil
}
