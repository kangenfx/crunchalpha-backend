package trader

import (
)

type DebugMetricsResponse struct {
	// Raw metrics from buildMetrics
	RawMetrics struct {
		AccountID      string  `json:"accountId"`
		CurrentBalance float64 `json:"currentBalance"`
		CurrentEquity  float64 `json:"currentEquity"`
		NetProfit      float64 `json:"netProfit"`
		GrossProfit    float64 `json:"grossProfit"`
		GrossLoss      float64 `json:"grossLoss"`
		TotalTrades    int     `json:"totalTrades"`
		WinningTrades  int     `json:"winningTrades"`
		LosingTrades   int     `json:"losingTrades"`
		MaxDrawdownPct float64 `json:"maxDrawdownPct"`
	} `json:"rawMetrics"`
	
	// DD Breakdown
	DDBreakdown struct {
		Level1_AbsoluteDD float64 `json:"level1_absoluteDD"`
		Level2_RelativeDD float64 `json:"level2_relativeDD"`
		Level3_RealDD     float64 `json:"level3_realDD"`
		FinalDD           float64 `json:"finalDD"`
		UsedLevel         string  `json:"usedLevel"`
	} `json:"ddBreakdown"`
	
	// Calculated Stats
	CalculatedStats struct {
		WinRate       float64 `json:"winRate"`
		ProfitFactor  float64 `json:"profitFactor"`
		AvgWin        float64 `json:"avgWin"`
		AvgLoss       float64 `json:"avgLoss"`
		RiskReward    float64 `json:"riskReward"`
		Expectancy    float64 `json:"expectancy"`
	} `json:"calculatedStats"`
	
	// AlphaRank
	AlphaRank struct {
		Score   float64 `json:"score"`
		Grade   string  `json:"grade"`
		Tier    string  `json:"tier"`
		Risk    string  `json:"risk"`
	} `json:"alphaRank"`
	
	// Pillars
	Pillars []struct {
		Code   string `json:"code"`
		Name   string `json:"name"`
		Score  int    `json:"score"`
		Weight int    `json:"weight"`
	} `json:"pillars"`
	
	// Database Info
	DatabaseInfo struct {
		InitialDeposit    float64 `json:"initialDeposit"`
		TotalDeposits     float64 `json:"totalDeposits"`
		TotalWithdrawals  float64 `json:"totalWithdrawals"`
		EquitySnapshots   int     `json:"equitySnapshots"`
		ClosedTrades      int     `json:"closedTrades"`
	} `json:"databaseInfo"`
}

func (s *Service) GetDebugMetrics(accountID, userID string) (*DebugMetricsResponse, error) {
	// Get account
	
	// Get trades
	trades, _ := s.getTradesForAccount(accountID)
	
	// Get balance/equity
	var balance, equity float64
	s.repo.db.QueryRow(`SELECT COALESCE(balance, 0), COALESCE(equity, 0) FROM trader_accounts WHERE id = $1`, accountID).Scan(&balance, &equity)
	
	// Build metrics
	metrics := s.buildMetrics(accountID, trades, balance, equity, "ALL")
	
	// Get DD breakdown
	level1 := CalculateAbsoluteDD(s.repo.db, accountID, equity)
	level2 := CalculateRelativeDD(s.repo.db, accountID, trades)
	level3 := CalculateRealDD(s.repo.db, accountID, trades)
	
	usedLevel := "Level 1"
	finalDD := level1
	if level2 > finalDD {
		finalDD = level2
		usedLevel = "Level 2"
	}
	if level3 > finalDD {
		finalDD = level3
		usedLevel = "Level 3"
	}
	
	// Get database info
	var initialDeposit, totalDeposits, totalWithdrawals float64
	var snapshotCount int
	
	s.repo.db.QueryRow(`SELECT COALESCE(amount, 0) FROM account_transactions WHERE account_id = $1 AND transaction_type = 'deposit' ORDER BY transaction_time ASC LIMIT 1`, accountID).Scan(&initialDeposit)
	s.repo.db.QueryRow(`SELECT COALESCE(SUM(amount), 0) FROM account_transactions WHERE account_id = $1 AND transaction_type = 'deposit'`, accountID).Scan(&totalDeposits)
	s.repo.db.QueryRow(`SELECT COALESCE(SUM(amount), 0) FROM account_transactions WHERE account_id = $1 AND transaction_type = 'withdrawal'`, accountID).Scan(&totalWithdrawals)
	s.repo.db.QueryRow(`SELECT COUNT(*) FROM equity_snapshots WHERE account_id = $1`, accountID).Scan(&snapshotCount)
	
	// Build response
	resp := &DebugMetricsResponse{}
	
	resp.RawMetrics.AccountID = metrics.AccountID
	resp.RawMetrics.CurrentBalance = metrics.CurrentBalance
	resp.RawMetrics.CurrentEquity = metrics.CurrentEquity
	resp.RawMetrics.NetProfit = metrics.NetProfit
	resp.RawMetrics.GrossProfit = metrics.GrossProfit
	resp.RawMetrics.GrossLoss = metrics.GrossLoss
	resp.RawMetrics.TotalTrades = metrics.TotalTrades
	resp.RawMetrics.WinningTrades = metrics.WinningTrades
	resp.RawMetrics.LosingTrades = metrics.LosingTrades
	resp.RawMetrics.MaxDrawdownPct = metrics.MaxDrawdownPct
	
	resp.DDBreakdown.Level1_AbsoluteDD = level1
	resp.DDBreakdown.Level2_RelativeDD = level2
	resp.DDBreakdown.Level3_RealDD = level3
	resp.DDBreakdown.FinalDD = finalDD
	resp.DDBreakdown.UsedLevel = usedLevel
	
	if metrics.TotalTrades > 0 {
		resp.CalculatedStats.WinRate = float64(metrics.WinningTrades) / float64(metrics.TotalTrades) * 100
		if metrics.GrossLoss != 0 {
			resp.CalculatedStats.ProfitFactor = metrics.GrossProfit / (-metrics.GrossLoss)
		}
		resp.CalculatedStats.Expectancy = metrics.NetProfit / float64(metrics.TotalTrades)
	}
	if metrics.WinningTrades > 0 {
		resp.CalculatedStats.AvgWin = metrics.GrossProfit / float64(metrics.WinningTrades)
	}
	if metrics.LosingTrades > 0 {
		resp.CalculatedStats.AvgLoss = metrics.GrossLoss / float64(metrics.LosingTrades)
	}
	if resp.CalculatedStats.AvgLoss != 0 {
		resp.CalculatedStats.RiskReward = resp.CalculatedStats.AvgWin / (-resp.CalculatedStats.AvgLoss)
	}
	
	resp.DatabaseInfo.InitialDeposit = initialDeposit
	resp.DatabaseInfo.TotalDeposits = totalDeposits
	resp.DatabaseInfo.TotalWithdrawals = totalWithdrawals
	resp.DatabaseInfo.EquitySnapshots = snapshotCount
	resp.DatabaseInfo.ClosedTrades = len(trades)
	
	return resp, nil
}
