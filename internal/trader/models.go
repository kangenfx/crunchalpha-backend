package trader

import "time"

// TraderAccount represents a trading account
type TraderAccount struct {
	ID            string    `json:"id" db:"id"`
	UserID        string    `json:"-" db:"user_id"`
	Nickname      string    `json:"nickname" db:"nickname"`
	Broker        string    `json:"broker" db:"broker"`
	Platform      string    `json:"platform" db:"platform"`
	Server        string    `json:"server" db:"server"`
	AccountType   string    `json:"account_type" db:"account_type"`
	Currency      string    `json:"currency" db:"currency"`
	AccountNumber string    `json:"account_number" db:"account_number"`
	Status        string    `json:"status" db:"status"`
	Balance       float64   `json:"balance" db:"balance"`
	Equity        float64   `json:"equity" db:"equity"`
	UpdatedAt     time.Time `json:"updated_at" db:"updated_at"`
	CreatedAt     time.Time `json:"created_at" db:"created_at"`
}

// AccountsResponse for listing accounts
type AccountsResponse struct {
	Accounts []TraderAccount `json:"accounts"`
}

// DashboardSnapshot - Capital overview
type DashboardSnapshot struct {
	Tier       string  `json:"tier"`
	Risk       string  `json:"risk"`
	Grade      string  `json:"grade"`
	AlphaScore float64 `json:"alphaScore"`

	Equity     string `json:"equity"`
	Balance    string `json:"balance"`
	PnLToday   string `json:"pnlToday"`

	TotalTrades  int     `json:"totalTrades"`
	WinRate      float64 `json:"winRate"`
	ProfitFactor float64 `json:"profitFactor"`
	MaxDD        float64 `json:"maxDD"`
	NetPnl       float64 `json:"netPnl"`
	AvgWin       float64 `json:"avgWin"`
	AvgLoss      float64 `json:"avgLoss"`
	RiskReward   float64 `json:"riskReward"`
	Expectancy   float64 `json:"expectancy"`

	BrokerLabel string `json:"brokerLabel"`
	RiskLevel   string `json:"riskLevel"`

	Statistics map[string]interface{} `json:"statistics,omitempty"`
}

type SurvivabilityScore struct {
	Score int    `json:"score"`
	Label string `json:"label"`
	Note  string `json:"note"`
}

type ScalabilityScore struct {
	Score int    `json:"score"`
	Label string `json:"label"`
	Note  string `json:"note"`
}

type Pillar struct {
	Code   string `json:"code"`
	Name   string `json:"name"`
	Weight int    `json:"weight"`
	Score  int    `json:"score"`
	Reason string `json:"reason"`
}

type RiskFlag struct {
	Severity  string `json:"severity"`
	Title     string `json:"title"`
	ScoreText string `json:"scoreText"`
	Desc      string `json:"desc"`
}

type RiskFlags struct {
	Counts struct {
		Critical int `json:"critical"`
		Major    int `json:"major"`
		Minor    int `json:"minor"`
	Total    int `json:"total"`
	} `json:"counts"`
	Items []RiskFlag `json:"items"`
}

type Metric struct {
	Label string `json:"label"`
	Value string `json:"value"`
}

type RegimeDetection struct {
	Regime     string  `json:"regime"`
	Confidence float64 `json:"confidence"`
	Window     string  `json:"window"`
	Reason     string  `json:"reason"`
}

type DashboardResponse struct {
	Snapshot      DashboardSnapshot  `json:"snapshot"`
	Survivability SurvivabilityScore `json:"survivability"`
	Scalability   ScalabilityScore   `json:"scalability"`
	Pillars       []Pillar           `json:"pillars"`
	RiskFlags     RiskFlags          `json:"riskFlags"`
	Metrics       []Metric           `json:"metrics"`
    Statistics    *MetricsStatistics `json:"statistics,omitempty"`
	Regime        RegimeDetection    `json:"regime"`
}

type MetricsStatistics struct {
	TotalTrades   int     `json:"totalTrades"`
	WinningTrades int     `json:"winningTrades"`
	LosingTrades  int     `json:"losingTrades"`
	WinRate       float64 `json:"winRate"`
	ProfitFactor  float64 `json:"profitFactor"`
	MaxDD         float64 `json:"maxDD"`
	AvgWin        float64 `json:"avgWin"`
	AvgLoss       float64 `json:"avgLoss"`
	RiskReward    float64 `json:"riskReward"`
	Expectancy    float64 `json:"expectancy"`
}
