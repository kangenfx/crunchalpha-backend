package alpharank

import (
	"fmt"
	"math"
	"time"
)

// SavePerformanceMetrics saves all performance data to DB - single source of truth
func (s *Service) SavePerformanceMetrics(accountID string, metrics AccountMetrics, result *AlphaRankResult) error {
	trades := metrics.Trades
	if len(trades) == 0 {
		return nil
	}

	// Calculate all metrics
	var grossProfit, grossLoss, totalProfit float64
	var wins, losses int
	var totalDuration float64

	for _, t := range trades {
		net := t.Profit + t.Swap + t.Commission
		totalProfit += net
		if net > 0 {
			grossProfit += net
			wins++
		} else {
			grossLoss += math.Abs(net)
			losses++
		}
		totalDuration += t.CloseTime.Sub(t.OpenTime).Hours()
	}

	totalTrades := len(trades)
	winRate := 0.0
	if totalTrades > 0 {
		winRate = float64(wins) / float64(totalTrades) * 100
	}

	profitFactor := 0.0
	if grossLoss > 0 {
		profitFactor = grossProfit / grossLoss
	}

	avgWin := 0.0
	if wins > 0 {
		avgWin = grossProfit / float64(wins)
	}

	avgLoss := 0.0
	if losses > 0 {
		avgLoss = grossLoss / float64(losses)
	}

	rewardRisk := 0.0
	if avgLoss > 0 {
		rewardRisk = avgWin / avgLoss
	}

	expectancy := 0.0
	if totalTrades > 0 {
		expectancy = totalProfit / float64(totalTrades)
	}

	avgDuration := 0.0
	if totalTrades > 0 {
		avgDuration = totalDuration / float64(totalTrades)
	}

	// Trades per month
	tradesPerMonth := 0.0
	if len(trades) > 0 {
		firstTrade := trades[0].CloseTime
		lastTrade := trades[len(trades)-1].CloseTime
		months := lastTrade.Sub(firstTrade).Hours() / 24 / 30
		if months > 0 {
			tradesPerMonth = float64(totalTrades) / months
		}
	}

	// Period start/end
	periodStart := trades[0].CloseTime
	periodEnd := trades[len(trades)-1].CloseTime


	query := `
		INSERT INTO performance_metrics (
			account_id, account_type, period, symbol,
			net_profit, gross_profit, gross_loss,
			profit_factor, expectancy,
			max_drawdown, max_drawdown_pct,
			win_rate, loss_rate,
			avg_win, avg_loss,
			reward_risk_ratio,
			total_trades, winning_trades, losing_trades,
			avg_trade_duration, trades_per_month,
			period_start, period_end,
			calculated_at
		) VALUES (
			$1, 'trader', 'ALL', 'ALL',
			$2, $3, $4,
			$5, $6,
			$7, $8,
			$9, $10,
			$11, $12,
			$13,
			$14, $15, $16,
			$17, $18,
			$19, $20,
			NOW()
		)
		ON CONFLICT (account_id, period, symbol)
		DO UPDATE SET
			net_profit = EXCLUDED.net_profit,
			gross_profit = EXCLUDED.gross_profit,
			gross_loss = EXCLUDED.gross_loss,
			profit_factor = EXCLUDED.profit_factor,
			expectancy = EXCLUDED.expectancy,
			max_drawdown = EXCLUDED.max_drawdown,
			max_drawdown_pct = EXCLUDED.max_drawdown_pct,
			win_rate = EXCLUDED.win_rate,
			loss_rate = EXCLUDED.loss_rate,
			avg_win = EXCLUDED.avg_win,
			avg_loss = EXCLUDED.avg_loss,
			reward_risk_ratio = EXCLUDED.reward_risk_ratio,
			total_trades = EXCLUDED.total_trades,
			winning_trades = EXCLUDED.winning_trades,
			losing_trades = EXCLUDED.losing_trades,
			avg_trade_duration = EXCLUDED.avg_trade_duration,
			trades_per_month = EXCLUDED.trades_per_month,
			period_start = EXCLUDED.period_start,
			period_end = EXCLUDED.period_end,
			calculated_at = NOW()
	`

	_, err := s.db.Exec(query,
		accountID,
		totalProfit, grossProfit, grossLoss,
		profitFactor, expectancy,
		0, metrics.MaxDrawdownPct,
		winRate, 100-winRate,
		avgWin, avgLoss,
		rewardRisk,
		totalTrades, wins, losses,
		avgDuration, tradesPerMonth,
		periodStart, periodEnd,
	)
	if err != nil {
		return fmt.Errorf("failed to save performance metrics: %w", err)
	}

	// Also save per-symbol performance
	bySymbol := make(map[string][]TradeData)
	for _, t := range trades {
		bySymbol[t.Symbol] = append(bySymbol[t.Symbol], t)
	}

	for symbol, symbolTrades := range bySymbol {
		if len(symbolTrades) < 10 {
			continue
		}
		s.saveSymbolPerformance(accountID, symbol, symbolTrades)
	}

	return nil
}

func (s *Service) saveSymbolPerformance(accountID, symbol string, trades []TradeData) error {
	var grossProfit, grossLoss, totalProfit float64
	var wins, losses int

	for _, t := range trades {
		net := t.Profit + t.Swap + t.Commission
		totalProfit += net
		if net > 0 {
			grossProfit += net
			wins++
		} else {
			grossLoss += math.Abs(net)
			losses++
		}
	}

	totalTrades := len(trades)
	winRate := 0.0
	if totalTrades > 0 {
		winRate = float64(wins) / float64(totalTrades) * 100
	}

	profitFactor := 0.0
	if grossLoss > 0 {
		profitFactor = grossProfit / grossLoss
	}

	avgWin := 0.0
	if wins > 0 {
		avgWin = grossProfit / float64(wins)
	}

	avgLoss := 0.0
	if losses > 0 {
		avgLoss = grossLoss / float64(losses)
	}

	rewardRisk := 0.0
	if avgLoss > 0 {
		rewardRisk = avgWin / avgLoss
	}

	expectancy := 0.0
	if totalTrades > 0 {
		expectancy = totalProfit / float64(totalTrades)
	}

	// Get MaxDD from alpha_ranks DB - single source of truth
	maxDD := 0.0
	s.db.QueryRow(`
		SELECT COALESCE(max_drawdown_pct, 0)
		FROM alpha_ranks
		WHERE account_id = $1 AND symbol = $2
	`, accountID, symbol).Scan(&maxDD)

	periodStart := trades[0].CloseTime
	periodEnd := trades[len(trades)-1].CloseTime

	query := `
		INSERT INTO performance_metrics (
			account_id, account_type, period, symbol,
			net_profit, gross_profit, gross_loss,
			profit_factor, expectancy,
			max_drawdown, max_drawdown_pct,
			win_rate, loss_rate,
			avg_win, avg_loss,
			reward_risk_ratio,
			total_trades, winning_trades, losing_trades,
			period_start, period_end,
			calculated_at
		) VALUES (
			$1, 'trader', 'ALL', $2,
			$3, $4, $5,
			$6, $7,
			$8, $9,
			$10, $11,
			$12, $13,
			$14,
			$15, $16, $17,
			$18, $19,
			NOW()
		)
		ON CONFLICT (account_id, period, symbol)
		DO UPDATE SET
			net_profit = EXCLUDED.net_profit,
			gross_profit = EXCLUDED.gross_profit,
			gross_loss = EXCLUDED.gross_loss,
			profit_factor = EXCLUDED.profit_factor,
			expectancy = EXCLUDED.expectancy,
			max_drawdown = EXCLUDED.max_drawdown,
			max_drawdown_pct = EXCLUDED.max_drawdown_pct,
			win_rate = EXCLUDED.win_rate,
			loss_rate = EXCLUDED.loss_rate,
			avg_win = EXCLUDED.avg_win,
			avg_loss = EXCLUDED.avg_loss,
			reward_risk_ratio = EXCLUDED.reward_risk_ratio,
			total_trades = EXCLUDED.total_trades,
			winning_trades = EXCLUDED.winning_trades,
			losing_trades = EXCLUDED.losing_trades,
			period_start = EXCLUDED.period_start,
			period_end = EXCLUDED.period_end,
			calculated_at = NOW()
	`

	_, err := s.db.Exec(query,
		accountID, symbol,
		totalProfit, grossProfit, grossLoss,
		profitFactor, expectancy,
		0, maxDD,
		winRate, 100-winRate,
		avgWin, avgLoss,
		rewardRisk,
		totalTrades, wins, losses,
		periodStart, periodEnd,
	)
	return err
}

// Helper
func avgTradeDuration(trades []TradeData) float64 {
	if len(trades) == 0 {
		return 0
	}
	total := 0.0
	for _, t := range trades {
		total += t.CloseTime.Sub(t.OpenTime).Hours()
	}
	return total / float64(len(trades))
}

// Unused but required to avoid import error
var _ = time.Now

// SaveMonthlyPerformance — group trades by year+month, save to performance_metrics
func (s *Service) SaveMonthlyPerformance(accountID string, trades []TradeData) error {
	type monthKey struct{ year, month int }
	byMonth := make(map[monthKey][]TradeData)
	for _, t := range trades {
		if t.CloseTime.IsZero() { continue }
		k := monthKey{t.CloseTime.Year(), int(t.CloseTime.Month())}
		byMonth[k] = append(byMonth[k], t)
	}
	for k, mTrades := range byMonth {
		var grossProfit, grossLoss, totalProfit float64
		var wins, losses int
		for _, t := range mTrades {
			net := t.Profit + t.Swap + t.Commission
			totalProfit += net
			if net > 0 { grossProfit += net; wins++ } else { grossLoss += math.Abs(net); losses++ }
		}
		total := len(mTrades)
		winRate := 0.0; if total > 0 { winRate = float64(wins)/float64(total)*100 }
		pf := 0.0; if grossLoss > 0 { pf = grossProfit/grossLoss }
		avgWin := 0.0; if wins > 0 { avgWin = grossProfit/float64(wins) }
		avgLoss := 0.0; if losses > 0 { avgLoss = grossLoss/float64(losses) }
		rr := 0.0; if avgLoss > 0 { rr = avgWin/avgLoss }
		exp := 0.0; if total > 0 { exp = totalProfit/float64(total) }
		period := fmt.Sprintf("%04d-%02d", k.year, k.month)
		pStart := mTrades[0].CloseTime; pEnd := mTrades[len(mTrades)-1].CloseTime
		s.db.Exec(`
			INSERT INTO performance_metrics (
				account_id, account_type, period, symbol,
				net_profit, gross_profit, gross_loss, profit_factor, expectancy,
				max_drawdown, max_drawdown_pct, win_rate, loss_rate,
				avg_win, avg_loss, reward_risk_ratio,
				total_trades, winning_trades, losing_trades,
				period_start, period_end, calculated_at
			) VALUES ($1,'trader',$2,'ALL',$3,$4,$5,$6,$7,0,0,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,NOW())
			ON CONFLICT (account_id, period, symbol) DO UPDATE SET
				net_profit=EXCLUDED.net_profit, gross_profit=EXCLUDED.gross_profit,
				gross_loss=EXCLUDED.gross_loss, profit_factor=EXCLUDED.profit_factor,
				expectancy=EXCLUDED.expectancy, win_rate=EXCLUDED.win_rate,
				loss_rate=EXCLUDED.loss_rate, avg_win=EXCLUDED.avg_win,
				avg_loss=EXCLUDED.avg_loss, reward_risk_ratio=EXCLUDED.reward_risk_ratio,
				total_trades=EXCLUDED.total_trades, winning_trades=EXCLUDED.winning_trades,
				losing_trades=EXCLUDED.losing_trades,
				period_start=EXCLUDED.period_start, period_end=EXCLUDED.period_end,
				calculated_at=NOW()`,
			accountID, period,
			totalProfit, grossProfit, grossLoss, pf, exp,
			winRate, 100-winRate, avgWin, avgLoss, rr,
			total, wins, losses, pStart, pEnd)
	}
	return nil
}

// SaveWeeklyPerformance — group trades by year+week, save to performance_metrics
func (s *Service) SaveWeeklyPerformance(accountID string, trades []TradeData) error {
	type weekKey struct{ year, week int }
	byWeek := make(map[weekKey][]TradeData)
	for _, t := range trades {
		if t.CloseTime.IsZero() { continue }
		y, w := t.CloseTime.ISOWeek()
		byWeek[weekKey{y, w}] = append(byWeek[weekKey{y, w}], t)
	}
	for k, wTrades := range byWeek {
		var grossProfit, grossLoss, totalProfit float64
		var wins, losses int
		for _, t := range wTrades {
			net := t.Profit + t.Swap + t.Commission
			totalProfit += net
			if net > 0 { grossProfit += net; wins++ } else { grossLoss += math.Abs(net); losses++ }
		}
		total := len(wTrades)
		winRate := 0.0; if total > 0 { winRate = float64(wins)/float64(total)*100 }
		pf := 0.0; if grossLoss > 0 { pf = grossProfit/grossLoss }
		avgWin := 0.0; if wins > 0 { avgWin = grossProfit/float64(wins) }
		avgLoss := 0.0; if losses > 0 { avgLoss = grossLoss/float64(losses) }
		rr := 0.0; if avgLoss > 0 { rr = avgWin/avgLoss }
		exp := 0.0; if total > 0 { exp = totalProfit/float64(total) }
		period := fmt.Sprintf("W%04d-%02d", k.year, k.week)
		pStart := wTrades[0].CloseTime; pEnd := wTrades[len(wTrades)-1].CloseTime
		s.db.Exec(`
			INSERT INTO performance_metrics (
				account_id, account_type, period, symbol,
				net_profit, gross_profit, gross_loss, profit_factor, expectancy,
				max_drawdown, max_drawdown_pct, win_rate, loss_rate,
				avg_win, avg_loss, reward_risk_ratio,
				total_trades, winning_trades, losing_trades,
				period_start, period_end, calculated_at
			) VALUES ($1,'trader',$2,'ALL',$3,$4,$5,$6,$7,0,0,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,NOW())
			ON CONFLICT (account_id, period, symbol) DO UPDATE SET
				net_profit=EXCLUDED.net_profit, gross_profit=EXCLUDED.gross_profit,
				gross_loss=EXCLUDED.gross_loss, profit_factor=EXCLUDED.profit_factor,
				expectancy=EXCLUDED.expectancy, win_rate=EXCLUDED.win_rate,
				loss_rate=EXCLUDED.loss_rate, avg_win=EXCLUDED.avg_win,
				avg_loss=EXCLUDED.avg_loss, reward_risk_ratio=EXCLUDED.reward_risk_ratio,
				total_trades=EXCLUDED.total_trades, winning_trades=EXCLUDED.winning_trades,
				losing_trades=EXCLUDED.losing_trades,
				period_start=EXCLUDED.period_start, period_end=EXCLUDED.period_end,
				calculated_at=NOW()`,
			accountID, period,
			totalProfit, grossProfit, grossLoss, pf, exp,
			winRate, 100-winRate, avgWin, avgLoss, rr,
			total, wins, losses, pStart, pEnd)
	}
	return nil
}
