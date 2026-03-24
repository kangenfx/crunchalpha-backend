package alpharank

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"time"
)

type Service struct {
	db *sql.DB
}

func NewService(db *sql.DB) *Service {
	return &Service{db: db}
}

func (s *Service) CalculateForAccount(accountID string) error {
	trades, err := s.getTradesForAccount(accountID)
	if err != nil {
		return fmt.Errorf("failed to get trades: %w", err)
	}

	if len(trades) < 10 {
		return fmt.Errorf("insufficient trades: need at least 10, have %d", len(trades))
	}

	var balance, equity float64
	err = s.db.QueryRow(`
		SELECT
			COALESCE(balance, 0) as balance,
			COALESCE(equity, 0) as equity
		FROM trader_accounts
		WHERE id = $1
	`, accountID).Scan(&balance, &equity)
	if err != nil {
		return fmt.Errorf("failed to get account info: %w", err)
	}

	// Get deposit/withdrawal data
	var totalDeposits, totalWithdrawals float64
	s.db.QueryRow(`
		SELECT COALESCE(SUM(amount), 0) 
		FROM account_transactions 
		WHERE account_id = $1 AND transaction_type = 'deposit'
	`, accountID).Scan(&totalDeposits)
	s.db.QueryRow(`
		SELECT COALESCE(SUM(amount), 0) 
		FROM account_transactions 
		WHERE account_id = $1 AND transaction_type = 'withdrawal'
	`, accountID).Scan(&totalWithdrawals)

	metrics := s.buildMetrics(accountID, trades, balance, equity, totalDeposits, totalWithdrawals)
	calculator := NewCalculator()
	result := calculator.Calculate(metrics)

	// Also calculate and save per-pair
	symbols, err := s.getDistinctSymbols(accountID)
	if err == nil {
		for _, symbol := range symbols {
			// Get trades for this symbol
			symbolTrades := []TradeData{}
			for _, t := range trades {
				if t.Symbol == symbol {
					symbolTrades = append(symbolTrades, t)
				}
			}
			
			// Skip if insufficient trades
				// Skip non-trading symbols
				if symbol == "archived" || symbol == "" {
					continue
				}
				if len(symbolTrades) < 1 {
				continue
			}
			
                        // Calculate metrics for this symbol
                        // Pass ALL trades for accurate running balance, DD only for this symbol
                        symbolMetrics := s.buildMetricsForSymbol(accountID, symbol, symbolTrades, trades, balance, equity, totalDeposits, totalWithdrawals)
                        symbolResult := calculator.Calculate(symbolMetrics)
			
			// Save per-pair (ignore errors)
                        // net_pnl per-pair = closed profit + floating profit per symbol
			symbolNetPnl := symbolMetrics.NetProfit
                        s.saveAlphaRankForSymbol(accountID, symbol, &symbolResult, len(symbolTrades), symbolMetrics.MaxDrawdownPct, symbolNetPnl, &symbolMetrics)
		}
	}


	// Save performance metrics to DB - single source of truth
	s.SavePerformanceMetrics(accountID, metrics, &result)
	s.SaveMonthlyPerformance(accountID, metrics.Trades)
	s.SaveWeeklyPerformance(accountID, metrics.Trades)

	return s.saveAlphaRankWithMetrics(accountID, &result, len(trades), metrics.MaxDrawdownPct, &metrics)
}

func (s *Service) getTradesForAccount(accountID string) ([]TradeData, error) {
	query := `
		SELECT
			symbol, type, lots,
			open_price, close_price,
			profit, swap, commission,
			open_time, close_time
		FROM trades
		WHERE account_id = $1
		AND status = 'closed'
		AND close_time IS NOT NULL
		ORDER BY close_time ASC
	`

	rows, err := s.db.Query(query, accountID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var trades []TradeData
	for rows.Next() {
		var t TradeData
		var openTime, closeTime sql.NullTime

		err := rows.Scan(
			&t.Symbol,
			&t.Type,
			&t.Lots,
			&t.OpenPrice,
			&t.ClosePrice,
			&t.Profit,
			&t.Swap,
			&t.Commission,
			&openTime,
			&closeTime,
		)
		if err != nil {
			return nil, err
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

func (s *Service) buildMetrics(accountID string, trades []TradeData, balance, equity, totalDeposits, totalWithdrawals float64) AccountMetrics {
	// Query floating profit dari DB (open trades) - zero on-the-fly
	var floatingProfit float64
	s.db.QueryRow(`
		SELECT COALESCE(SUM(profit + swap + commission), 0)
		FROM trades
		WHERE account_id = $1 AND status = 'open'
	`, accountID).Scan(&floatingProfit)
	var (
		grossProfit   float64
		grossLoss     float64
		winningTrades int
		losingTrades  int
		totalProfit   float64
		maxDD         float64
		peakBalance   float64
		startDate     time.Time
		endDate       time.Time
	)

	if len(trades) > 0 {
		startDate = trades[0].OpenTime
		endDate = trades[len(trades)-1].CloseTime
	}

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

	// Initial deposit = first deposit before first trade
	initialDeposit := 0.0
	s.db.QueryRow(`
		SELECT COALESCE(SUM(amount), 0)
		FROM account_transactions
		WHERE account_id = $1
		AND transaction_type = 'deposit'
		AND transaction_time <= (SELECT MIN(close_time) FROM trades WHERE account_id = $1 AND status = 'closed')
	`, accountID).Scan(&initialDeposit)
	if initialDeposit <= 0 {
		initialDeposit = balance - totalProfit
	}
	if initialDeposit <= 0 {
		initialDeposit = balance
	}

	maxDD = 0.0
	peakBalance = initialDeposit

	// DD calculation: loop all events (deposits/withdrawals/trades) ordered by time
	// DD per trade = abs(loss) / balance_before_trade
	type Event struct {
		EventTime time.Time
		EventType string
		Amount    float64
	}
	var events []Event

	// Add deposit/withdrawal events
	depRows, depErr := s.db.Query(`
		SELECT transaction_type, amount, created_at
		FROM account_transactions
		WHERE account_id = $1
		ORDER BY created_at ASC
	`, accountID)
	if depErr == nil {
		defer depRows.Close()
		for depRows.Next() {
			var txType string
			var amount float64
			var createdAt time.Time
			if err := depRows.Scan(&txType, &amount, &createdAt); err == nil {
				events = append(events, Event{EventTime: createdAt, EventType: txType, Amount: amount})
			}
		}
	}

	// Add trade close events
	for _, trade := range trades {
		net := trade.Profit + trade.Swap + trade.Commission
		events = append(events, Event{EventTime: trade.CloseTime, EventType: "trade", Amount: net})
	}

	// Sort events by time (bubble sort)
	for i := 0; i < len(events); i++ {
		for j := i + 1; j < len(events); j++ {
			if events[j].EventTime.Before(events[i].EventTime) {
				events[i], events[j] = events[j], events[i]
			}
		}
	}

	// Loop events - DD = loss / balance_before_trade
	runningBalance := initialDeposit
	for _, event := range events {
		switch event.EventType {
		case "deposit":
			runningBalance += event.Amount
			if runningBalance > peakBalance {
				peakBalance = runningBalance
			}
		case "withdrawal":
			runningBalance -= event.Amount
		case "trade":
			if event.Amount < 0 && runningBalance > 0 {
				dd := (math.Abs(event.Amount) / runningBalance) * 100
				if dd > maxDD {
					maxDD = dd
				}
			}
			runningBalance += event.Amount
			if runningBalance > peakBalance {
				peakBalance = runningBalance
			}
		}
	}

	// Floating DD
	if balance > 0 && equity < balance {
		floatingDD := (balance - equity) / balance * 100
		if floatingDD > maxDD {
			maxDD = floatingDD
		}
	}

	// Cap DD at 100%
	if maxDD > 100 {
		maxDD = 100
	}

	// Peak balance
	if totalDeposits > peakBalance {
		peakBalance = totalDeposits
	}

	// Derived metrics
	avgWin := 0.0
	if winningTrades > 0 { avgWin = grossProfit / float64(winningTrades) }
	avgLoss := 0.0
	if losingTrades > 0 { avgLoss = math.Abs(grossLoss) / float64(losingTrades) }
	riskReward := 0.0
	if avgLoss > 0 { riskReward = avgWin / avgLoss }
	expectancy := 0.0
	if len(trades) > 0 { expectancy = totalProfit / float64(len(trades)) }

	return AccountMetrics{
		AccountID:        accountID,
		CurrentBalance:   balance,
		CurrentEquity:    equity,
		InitialDeposit:   initialDeposit,
		TotalDeposits:    totalDeposits,
		TotalWithdraws:   totalWithdrawals,
		PeakBalance:      peakBalance,
		NetProfit:        totalProfit + floatingProfit,
		ClosedNetProfit:  totalProfit,
		GrossProfit:      grossProfit,
		GrossLoss:        grossLoss,
		AvgWin:           avgWin,
		AvgLoss:          avgLoss,
		RiskReward:       riskReward,
		Expectancy:       expectancy,
		TotalTrades:      len(trades),
		WinningTrades:    winningTrades,
		LosingTrades:     losingTrades,
		MaxDrawdownPct:   maxDD,
		MaxDrawdownAbs:   0,
		Trades:           trades,
		EquitySnapshots:  s.loadEquitySnapshots(accountID),
		StartDate:        startDate,
		EndDate:          endDate,
	}
}


// loadEquitySnapshots loads equity snapshots for an account
func (s *Service) loadEquitySnapshots(accountID string) []EquitySnapshot {
	rows, err := s.db.Query(`
		SELECT snapshot_time, equity, balance
		FROM equity_snapshots
		WHERE account_id = $1
		ORDER BY snapshot_time ASC`, accountID)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var snapshots []EquitySnapshot
	for rows.Next() {
		var s EquitySnapshot
		rows.Scan(&s.SnapshotTime, &s.Equity, &s.Balance)
		snapshots = append(snapshots, s)
	}
	return snapshots
}

func (s *Service) saveAlphaRank(accountID string, result *AlphaRankResult, tradeCount int, maxDD float64) error {
	return s.saveAlphaRankWithMetrics(accountID, result, tradeCount, maxDD, nil)
}

func (s *Service) saveAlphaRankWithMetrics(accountID string, result *AlphaRankResult, tradeCount int, maxDD float64, metrics *AccountMetrics) error {
	var p1, p2, p3, p4, p5, p6, p7 float64
	for _, pillar := range result.Pillars {
		switch pillar.Code {
		case "P1": p1 = pillar.Score
		case "P2": p2 = pillar.Score
		case "P3": p3 = pillar.Score
		case "P4": p4 = pillar.Score
		case "P5": p5 = pillar.Score
		case "P6": p6 = pillar.Score
		case "P7": p7 = pillar.Score
		}
	}

	flagsJSON, err := json.Marshal(result.RiskFlags.Items)
	if err != nil {
		return fmt.Errorf("failed to marshal flags: %w", err)
	}
	pillarsJSON, _ := json.Marshal(result.Pillars)

	// Calc global stats from metrics if available
	winRate := 0.0
	profitFactor := 0.0
	netPnl := 0.0
	roi := 0.0
	totalTradesAll := tradeCount
	if metrics != nil {
		if metrics.TotalTrades > 0 {
			winRate = float64(metrics.WinningTrades) / float64(metrics.TotalTrades) * 100
		}
		if metrics.GrossLoss != 0 {
			profitFactor = math.Abs(metrics.GrossProfit / metrics.GrossLoss)
		}
		// net_pnl = equity + withdraw - deposit (REAL, termasuk floating open positions)
		netPnl = metrics.NetProfit
			if metrics.TotalDeposits > 0 { roi = (netPnl / metrics.TotalDeposits) * 100 }
		totalTradesAll = metrics.TotalTrades
	}

	query := `
		INSERT INTO alpha_ranks (
			account_id, account_type,
			profitability_score, risk_score, consistency_score,
			stability_score, activity_score, duration_score, drawdown_score,
			alpha_score, grade, badge, tier,
			symbol, status, min_trades_met, trade_count,
			risk_flags, critical_count, major_count, minor_count,
			max_drawdown_pct, pillars,
				win_rate, total_trades_all, profit_factor, net_pnl, roi,
				winning_trades, losing_trades, avg_win, avg_loss, risk_reward, expectancy,
                        risk_level,
			survivability_score, scalability_score,
			calculated_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23,$24,$25,$26,$27,$28,$29,$30,$31,$32,$33,$34,$35,$36,$37,NOW())
		ON CONFLICT (account_id, symbol)
		DO UPDATE SET
			profitability_score=EXCLUDED.profitability_score, risk_score=EXCLUDED.risk_score,
			consistency_score=EXCLUDED.consistency_score, stability_score=EXCLUDED.stability_score,
			activity_score=EXCLUDED.activity_score, duration_score=EXCLUDED.duration_score,
			drawdown_score=EXCLUDED.drawdown_score, alpha_score=EXCLUDED.alpha_score,
			grade=EXCLUDED.grade, badge=EXCLUDED.badge, tier=EXCLUDED.tier,
			trade_count=EXCLUDED.trade_count, risk_flags=EXCLUDED.risk_flags,
			critical_count=EXCLUDED.critical_count, major_count=EXCLUDED.major_count,
			minor_count=EXCLUDED.minor_count, max_drawdown_pct=EXCLUDED.max_drawdown_pct,
			pillars=EXCLUDED.pillars, win_rate=EXCLUDED.win_rate,
			total_trades_all=EXCLUDED.total_trades_all, profit_factor=EXCLUDED.profit_factor,
				net_pnl=EXCLUDED.net_pnl, roi=EXCLUDED.roi,
				winning_trades=EXCLUDED.winning_trades, losing_trades=EXCLUDED.losing_trades,
				avg_win=EXCLUDED.avg_win, avg_loss=EXCLUDED.avg_loss,
				risk_reward=EXCLUDED.risk_reward, expectancy=EXCLUDED.expectancy,
				risk_level=EXCLUDED.risk_level,
			survivability_score=EXCLUDED.survivability_score,
			scalability_score=EXCLUDED.scalability_score,
                        calculated_at=NOW()
	`

	_, err = s.db.Exec(query,
		accountID, "trader",
		p1, p2, p3, p4, p5, p6, p7,
		result.AlphaScore, result.Grade, result.Tier, result.Tier,
		"ALL", "ACTIVE", true, tradeCount,
		flagsJSON,
		result.RiskFlags.Counts.Critical, result.RiskFlags.Counts.Major, result.RiskFlags.Counts.Minor,
		maxDD, pillarsJSON,
			winRate, totalTradesAll, profitFactor, netPnl, roi,
			int(metrics.WinningTrades), int(metrics.LosingTrades),
			metrics.AvgWin, metrics.AvgLoss, metrics.RiskReward, metrics.Expectancy,
			result.Risk,
			result.Survivability.Score, result.Scalability.Score,
	)
	return err
}

func (s *Service) getDistinctSymbols(accountID string) ([]string, error) {
	query := `
		SELECT DISTINCT symbol
		FROM trades
		WHERE account_id = $1 AND status = 'closed'
		ORDER BY symbol
	`
	rows, err := s.db.Query(query, accountID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var symbols []string
	for rows.Next() {
		var symbol string
		rows.Scan(&symbol)
		symbols = append(symbols, symbol)
	}
	return symbols, nil
}

func (s *Service) saveAlphaRankForSymbol(accountID, symbol string, result *AlphaRankResult, tradeCount int, maxDD float64, netPnl float64, metrics *AccountMetrics) error {
	winRate := 0.0
	profitFactor := 0.0
	if metrics != nil {
		if metrics.TotalTrades > 0 {
			winRate = float64(metrics.WinningTrades) / float64(metrics.TotalTrades) * 100
		}
		if metrics.GrossLoss != 0 {
			profitFactor = math.Abs(metrics.GrossProfit / metrics.GrossLoss)
		}
	}
	var p1, p2, p3, p4, p5, p6, p7 float64
	for _, pillar := range result.Pillars {
		switch pillar.Code {
		case "P1":
			p1 = pillar.Score
		case "P2":
			p2 = pillar.Score
		case "P3":
			p3 = pillar.Score
		case "P4":
			p4 = pillar.Score
		case "P5":
			p5 = pillar.Score
		case "P6":
			p6 = pillar.Score
		case "P7":
			p7 = pillar.Score
		}
	}

	flagsJSON, err := json.Marshal(result.RiskFlags.Items)
	if err != nil {
		return fmt.Errorf("failed to marshal flags: %w", err)
	}

	pillarsJSON, _ := json.Marshal(result.Pillars)

	query := `
			INSERT INTO alpha_ranks (
				account_id, account_type,
				profitability_score, risk_score, consistency_score,
				stability_score, activity_score, duration_score, drawdown_score,
				alpha_score, grade, badge, tier,
				symbol, status, min_trades_met, trade_count,
				risk_flags, critical_count, major_count, minor_count,
				max_drawdown_pct, pillars,
					net_pnl, win_rate, total_trades_all, profit_factor,
				risk_level,
				calculated_at
			) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23,$24,$25,$26,$27,$28,NOW())
			ON CONFLICT (account_id, symbol)
			DO UPDATE SET
				profitability_score = EXCLUDED.profitability_score,
				risk_score = EXCLUDED.risk_score,
				consistency_score = EXCLUDED.consistency_score,
				stability_score = EXCLUDED.stability_score,
				activity_score = EXCLUDED.activity_score,
				duration_score = EXCLUDED.duration_score,
				drawdown_score = EXCLUDED.drawdown_score,
				alpha_score = EXCLUDED.alpha_score,
				grade = EXCLUDED.grade,
				badge = EXCLUDED.badge,
				tier = EXCLUDED.tier,
				trade_count = EXCLUDED.trade_count,
				risk_flags = EXCLUDED.risk_flags,
				critical_count = EXCLUDED.critical_count,
				major_count = EXCLUDED.major_count,
				minor_count = EXCLUDED.minor_count,
				max_drawdown_pct = EXCLUDED.max_drawdown_pct,
				pillars = EXCLUDED.pillars,
					net_pnl = EXCLUDED.net_pnl,
					win_rate = EXCLUDED.win_rate,
					total_trades_all = EXCLUDED.total_trades_all,
					profit_factor = EXCLUDED.profit_factor,
					risk_level = EXCLUDED.risk_level,
				calculated_at = NOW()
	`

	_, err = s.db.Exec(query,
		accountID, "trader",
		p1, p2, p3, p4, p5, p6, p7,
		result.AlphaScore, result.Grade, result.Tier, result.Tier,
		symbol, "ACTIVE", true, tradeCount,
		flagsJSON,
		result.RiskFlags.Counts.Critical,
		result.RiskFlags.Counts.Major,
		result.RiskFlags.Counts.Minor,
		maxDD, pillarsJSON,
		netPnl,
		winRate, tradeCount, profitFactor,
		GetRiskLevelFromCounts(result.RiskFlags.Counts.Critical, result.RiskFlags.Counts.Major, result.RiskFlags.Counts.Minor, result.AlphaScore),
	)
	return err
}

// buildMetricsForSymbol - same as buildMetrics but DD only calculated for specific symbol
// Uses ALL trades for accurate running balance, but only counts DD for target symbol
func (s *Service) buildMetricsForSymbol(accountID, symbol string, symbolTrades []TradeData, allTrades []TradeData, balance, equity, totalDeposits, totalWithdrawals float64) AccountMetrics {
	// Calculate symbol-specific stats from symbolTrades only
	var grossProfit, grossLoss float64
	var winningTrades, losingTrades int
	var totalProfit float64
	var startDate, endDate time.Time

	if len(symbolTrades) > 0 {
		startDate = symbolTrades[0].OpenTime
		endDate = symbolTrades[len(symbolTrades)-1].CloseTime
	}

	for _, trade := range symbolTrades {
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

	// Starting balance = current balance minus all trades profit
	allTotalProfit := 0.0
	for _, t := range allTrades {
		allTotalProfit += t.Profit + t.Swap + t.Commission
	}
	initialDeposit := balance - allTotalProfit
	if initialDeposit <= 0 {
		initialDeposit = totalDeposits
	}
	if initialDeposit <= 0 {
		initialDeposit = balance
	}

	// DD calculation using ALL events for accurate balance
	// but only count DD when THIS symbol has a loss
	type Event struct {
		EventTime  time.Time
		EventType  string
		Amount     float64
		Symbol     string
	}

	var events []Event

	// Add deposit/withdrawal events
	depRows, depErr := s.db.Query(`
		SELECT transaction_type, amount, created_at
		FROM account_transactions
		WHERE account_id = $1
		ORDER BY created_at ASC
	`, accountID)
	if depErr == nil {
		defer depRows.Close()
		for depRows.Next() {
			var txType string
			var amount float64
			var createdAt time.Time
			if err := depRows.Scan(&txType, &amount, &createdAt); err == nil {
				events = append(events, Event{EventTime: createdAt, EventType: txType, Amount: amount})
			}
		}
	}

	// Add ALL trades (all symbols) for accurate balance tracking
	for _, trade := range allTrades {
		net := trade.Profit + trade.Swap + trade.Commission
		events = append(events, Event{EventTime: trade.CloseTime, EventType: "trade", Amount: net, Symbol: trade.Symbol})
	}

	// Sort by time
	for i := 0; i < len(events); i++ {
		for j := i + 1; j < len(events); j++ {
			if events[j].EventTime.Before(events[i].EventTime) {
				events[i], events[j] = events[j], events[i]
			}
		}
	}

	// Loop events - DD only for target symbol loss
	runningBalance := initialDeposit
	maxDD := 0.0
	peakBalance := 0.0

	for _, event := range events {
		switch event.EventType {
		case "deposit":
			runningBalance += event.Amount
			if runningBalance > peakBalance {
				peakBalance = runningBalance
			}
		case "withdrawal":
			runningBalance -= event.Amount
		case "trade":
			// Only calculate DD for target symbol loss
			if event.Symbol == symbol && event.Amount < 0 && runningBalance > 0 {
				dd := (math.Abs(event.Amount) / runningBalance) * 100
				if dd > maxDD {
					maxDD = dd
				}
			}
			runningBalance += event.Amount
			if runningBalance > peakBalance {
				peakBalance = runningBalance
			}
		}
	}

	// Floating DD per-pair = abs(floating_profit_pair) / balance
	// Konsisten untuk 1 pair maupun multi-pair (proporsional dari balance)
	var symbolFloatingProfit float64
	s.db.QueryRow(`
		SELECT COALESCE(SUM(profit), 0)
		FROM trades
		WHERE account_id = $1 AND symbol = $2 AND status = 'open'
	`, accountID, symbol).Scan(&symbolFloatingProfit)

	if symbolFloatingProfit < 0 && balance > 0 {
		floatingDD := (math.Abs(symbolFloatingProfit) / balance) * 100
		if floatingDD > maxDD {
			maxDD = floatingDD
		}
	}
	// Cap at 100%
	if maxDD > 100 {
		maxDD = 100
	}

	return AccountMetrics{
		AccountID:        accountID,
		CurrentBalance:   balance,
		CurrentEquity:    equity,
		GrossProfit:      grossProfit,
		GrossLoss:        grossLoss,
		NetProfit:        totalProfit + symbolFloatingProfit,
		ClosedNetProfit:  totalProfit,
		TotalDeposits:    totalDeposits,
		InitialDeposit:   initialDeposit,
		TotalWithdraws:   totalWithdrawals,
		WinningTrades:    winningTrades,
		LosingTrades:     losingTrades,
		TotalTrades:      len(symbolTrades),
		MaxDrawdownPct:   maxDD,
		PeakBalance:      peakBalance,
		StartDate:        startDate,
		EndDate:          endDate,
		Trades:           symbolTrades,
	}
}

// GetRiskLevelFromCounts - standalone helper untuk compute risk_level dari counts
func GetRiskLevelFromCounts(critical, major, minor int, alphaScore float64) string {
	totalFlags := critical + major + minor
	// EXTREME: Any critical flag OR AlphaScore < 30
	if critical > 0 || alphaScore < 30 {
		return "EXTREME"
	}
	// HIGH: 3+ flags OR AlphaScore 30-50
	if totalFlags >= 3 || (alphaScore >= 30 && alphaScore < 50) {
		return "HIGH"
	}
	// MEDIUM: 2 flags OR AlphaScore 50-70
	if totalFlags >= 2 || (alphaScore >= 50 && alphaScore < 70) {
		return "MEDIUM"
	}
	// VERIFIED_SAFE: AlphaScore >= 85, no flags
	if alphaScore >= 85 && totalFlags == 0 {
		return "VERIFIED_SAFE"
	}
	// LOW: 0-1 flags + AlphaScore >= 70
	if critical == 0 && alphaScore >= 70 {
		return "LOW"
	}
	return "MEDIUM"
}
