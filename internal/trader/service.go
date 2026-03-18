package trader

import (
	"database/sql"
	"errors"
	"fmt"
	"time"
	
	"crunchalpha-v3/internal/alpharank"
)

type Service struct {
	repo       *Repository
	alphaCalc  *alpharank.Calculator
}

func NewService(db *sql.DB) *Service {
	return &Service{
		repo:      NewRepository(db),
		alphaCalc: alpharank.NewCalculator(),
	}
}

func (s *Service) GetUserAccounts(userID string) ([]TraderAccount, error) {
	return s.repo.GetUserAccounts(userID)
}

func (s *Service) GetDashboard(accountID, userID string) (*DashboardResponse, error) {
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

	metrics := alpharank.AccountMetrics{
		AccountID:      accountID,
		CurrentBalance: 12500,
		CurrentEquity:  12500,
		InitialDeposit: 10000,
		TotalDeposits:  10000,
		NetProfit:      2500,
		GrossProfit:    2840.9,
		GrossLoss:      -340.9,
		TotalTrades:    164,
		WinningTrades:  89,
		LosingTrades:   75,
		MaxDrawdownPct: 18.7,
		StartDate:      time.Now().AddDate(0, -4, 0),
		EndDate:        time.Now(),
		Trades:         generateDummyTrades(164),
	}
	
	alphaResult := s.alphaCalc.Calculate(metrics)
	
	dashboard := &DashboardResponse{
		Snapshot: DashboardSnapshot{
			Tier:         alphaResult.Tier,
			Risk:         alphaResult.Risk,
			Grade:        alphaResult.Grade,
			AlphaScore:   alphaResult.AlphaScore,
			Equity:       "12,500 USD",
			Balance:      "12,000 USD",
			PnLToday:     "+135.40 USD",
			TotalTrades:  164,
			WinRate:      54.2,
			ProfitFactor: 1.34,
			MaxDD:        18.7,
			BrokerLabel:  brokerLabel,
		},
		Survivability: SurvivabilityScore{
			Score: alphaResult.Survivability.Score,
			Label: alphaResult.Survivability.Label,
			Note:  alphaResult.Survivability.Note,
		},
		Scalability: ScalabilityScore{
			Score: alphaResult.Scalability.Score,
			Label: alphaResult.Scalability.Label,
			Note:  alphaResult.Scalability.Note,
		},
		Pillars:   convertPillars(alphaResult.Pillars),
		RiskFlags: convertRiskFlags(alphaResult),
		Metrics: []Metric{
			{Label: "Total Return (Net)", Value: "25.0%"},
			{Label: "Profitability / Yield", Value: "2.98x"},
			{Label: "Avg Daily Return", Value: "0.18%"},
			{Label: "Max Drawdown", Value: "18.7%"},
			{Label: "Gross Profit", Value: "2,840.9 USD"},
			{Label: "Gross Loss", Value: "-340.9 USD"},
			{Label: "Profit Factor", Value: "1.34"},
			{Label: "Win Rate", Value: "54.2%"},
			{Label: "Avg R:R", Value: "1.28"},
			{Label: "Avg Hold", Value: "5.2 hours"},
			{Label: "Expectancy / Trade", Value: "1.62"},
			{Label: "Sharpe (proxy)", Value: "0.84"},
			{Label: "Sortino (proxy)", Value: "1.26"},
			{Label: "Tail Risk (proxy)", Value: "0.41"},
			{Label: "Max Consecutive Loss", Value: "6"},
			{Label: "Max Consecutive Win", Value: "9"},
			{Label: "Exposure (avg)", Value: "22%"},
			{Label: "Weekend Exposure", Value: "Low"},
			{Label: "Trade Frequency", Value: "12/day"},
			{Label: "Consistency Score", Value: "0.67"},
		},
		Regime: RegimeDetection{
			Regime:     "Choppy",
			Confidence: 0.78,
			Window:     "D1",
			Reason:     "Low slope + noisy sign-flips",
		},
	}

	return dashboard, nil
}

func (s *Service) CreateDummyAccount(userID, nickname, broker, platform string) (*TraderAccount, error) {
	return s.repo.CreateDummyAccount(userID, nickname, broker, platform)
}

func convertPillars(alphaPillars []alpharank.PillarScore) []Pillar {
	pillars := make([]Pillar, len(alphaPillars))
	for i, ap := range alphaPillars {
		pillars[i] = Pillar{
			Code:   ap.Code,
			Name:   ap.Name,
			Weight: ap.Weight,
			Score:  int(ap.Score),
		}
	}
	return pillars
}

func convertRiskFlags(alphaResult alpharank.AlphaRankResult) RiskFlags {
	rf := RiskFlags{}
	rf.Counts.Critical = alphaResult.RiskFlags.Counts.Critical
	rf.Counts.Major = alphaResult.RiskFlags.Counts.Major
	rf.Counts.Minor = alphaResult.RiskFlags.Counts.Minor
	rf.Counts.Total = len(alphaResult.RiskFlags.Items)
	
	rf.Items = make([]RiskFlag, len(alphaResult.RiskFlags.Items))
	for i, af := range alphaResult.RiskFlags.Items {
		rf.Items[i] = RiskFlag{
			Severity:  af.Severity,
			Title:     af.Title,
			ScoreText: fmt.Sprintf("-%.0f", af.Penalty),
			Desc:      af.Desc,
		}
	}
	
	return rf
}

func generateDummyTrades(count int) []alpharank.TradeData {
	trades := make([]alpharank.TradeData, count)
	baseTime := time.Now().AddDate(0, -4, 0)
	
	for i := 0; i < count; i++ {
		openTime := baseTime.Add(time.Duration(i*2) * time.Hour)
		closeTime := openTime.Add(time.Duration(3+i%10) * time.Hour)
		
		profit := float64(10 + (i % 50))
		if i%3 == 0 {
			profit = -profit
		}
		
		trades[i] = alpharank.TradeData{
			OpenTime:   openTime,
			CloseTime:  closeTime,
			Symbol:     "EURUSD",
			Type:       "BUY",
			Lots:       0.1,
			OpenPrice:  1.1000,
			ClosePrice: 1.1010,
			StopLoss:   1.0990,
			TakeProfit: 1.1020,
			Profit:     profit,
		}
	}
	
	return trades
}
// Add this method to service.go



// DeleteAccount removes an account
func (s *Service) DeleteAccount(accountID, userID string) error {
	return s.repo.DeleteAccount(accountID, userID)
}

// CreateAccount registers a new trading account
func (s *Service) CreateAccount(userID, accountNumber, broker, platform, nickname, currency string) (*TraderAccount, error) {
	return s.repo.CreateAccount(userID, accountNumber, broker, platform, nickname, currency)
}

// GetAccountByNumber finds account by number
func (s *Service) GetAccountByNumber(accountNumber, userID string) (*TraderAccount, error) {
	return s.repo.GetAccountByNumber(accountNumber, userID)
}
// CreateAccountFull creates account with all fields (PRODUCTION)
func (s *Service) CreateAccountFull(userID, accountNumber, broker, platform, server, investorPassword, nickname, currency, role string) (*TraderAccount, error) {
	return s.repo.CreateAccountFull(userID, accountNumber, broker, platform, server, investorPassword, nickname, currency, role)
}


// GetMonthlyPerformanceFromDB returns monthly aggregated data from trades
func (s *Service) GetMonthlyPerformanceFromDB(accountID string) ([]map[string]interface{}, error) {
	query := `
		SELECT 
			TO_CHAR(close_time, 'Mon') as month,
			EXTRACT(YEAR FROM close_time) as year,
			COUNT(*) as trades,
			SUM(CASE WHEN (profit + swap + commission) > 0 THEN 1 ELSE 0 END) as wins,
			SUM(CASE WHEN (profit + swap + commission) < 0 THEN 1 ELSE 0 END) as losses,
			SUM(profit + swap + commission) as total_profit
		FROM trades
		WHERE account_id = $1 AND status = 'closed'
		GROUP BY TO_CHAR(close_time, 'Mon'), EXTRACT(YEAR FROM close_time), DATE_TRUNC('month', close_time)
		ORDER BY DATE_TRUNC('month', close_time) ASC
	`

	rows, err := s.repo.db.Query(query, accountID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var months []map[string]interface{}
	for rows.Next() {
		var month string
		var year int
		var trades, wins, losses int
		var totalProfit float64

		if err := rows.Scan(&month, &year, &trades, &wins, &losses, &totalProfit); err != nil {
			continue
		}

		winRate := 0.0
		if trades > 0 {
			winRate = (float64(wins) / float64(trades)) * 100
		}

		months = append(months, map[string]interface{}{
			"month":   month,
			"year":    year,
			"profit":  totalProfit,
			"trades":  trades,
			"wins":    wins,
			"losses":  losses,
			"winRate": winRate,
		})
	}

	return months, nil
}

// GetWeeklyPerformanceFromDB returns weekly aggregated data from trades
func (s *Service) GetWeeklyPerformanceFromDB(accountID string) ([]map[string]interface{}, error) {
	query := `
		SELECT 
			EXTRACT(WEEK FROM close_time) as week_num,
			EXTRACT(YEAR FROM close_time) as year,
			COUNT(*) as trades,
			SUM(CASE WHEN (profit + swap + commission) > 0 THEN 1 ELSE 0 END) as wins,
			SUM(CASE WHEN (profit + swap + commission) < 0 THEN 1 ELSE 0 END) as losses,
			SUM(profit + swap + commission) as total_profit
		FROM trades
		WHERE account_id = $1 AND status = 'closed'
		GROUP BY EXTRACT(WEEK FROM close_time), EXTRACT(YEAR FROM close_time), DATE_TRUNC('week', close_time)
		ORDER BY DATE_TRUNC('week', close_time) ASC
	`

	rows, err := s.repo.db.Query(query, accountID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var weeks []map[string]interface{}
	for rows.Next() {
		var weekNum, year int
		var trades, wins, losses int
		var totalProfit float64

		if err := rows.Scan(&weekNum, &year, &trades, &wins, &losses, &totalProfit); err != nil {
			continue
		}

		winRate := 0.0
		if trades > 0 {
			winRate = (float64(wins) / float64(trades)) * 100
		}

		weeks = append(weeks, map[string]interface{}{
			"week":    weekNum,
			"year":    year,
			"profit":  totalProfit,
			"trades":  trades,
			"wins":    wins,
			"losses":  losses,
			"winRate": winRate,
		})
	}

	return weeks, nil
}
