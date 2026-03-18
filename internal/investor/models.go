package investor

import "time"

// UserInfo - basic user info
type UserInfo struct {
	ID          string `json:"id"`
	Email       string `json:"email"`
	PrimaryRole string `json:"primary_role"`
}

// AllocationWithTraderInfo - allocation with real trader data from DB
type AllocationWithTraderInfo struct {
	TraderAccountID     string  `json:"trader_account_id"`
	TraderAccountNumber string  `json:"trader_account_number"`
	Broker              string  `json:"broker"`
	Platform            string  `json:"platform"`
	TraderEquity        float64 `json:"trader_equity"`
	AllocationMode      string  `json:"allocation_mode"`
	AllocationValue     float64 `json:"allocation_value"`
	MaxRiskPct          float64 `json:"max_risk_pct"`
	MaxPositions        int     `json:"max_positions"`
	AlphaScore          float64 `json:"alpha_score"`
	Grade               string  `json:"grade"`
	RiskLevel           string  `json:"risk_level"`
	Status              string  `json:"status"`
}

// Subscription - user subscription record
type Subscription struct {
	ID               string    `json:"id"`
	UserID           string    `json:"user_id"`
	TraderAccountID  string    `json:"trader_account_id"`
	SubscriptionType string    `json:"subscription_type"`
	StartAt          time.Time `json:"start_at"`
	EndAt            time.Time `json:"end_at"`
	Status           string    `json:"status"`
	CreatedAt        time.Time `json:"created_at"`
}

// Portfolio - investor portfolio summary, all values from DB
type Portfolio struct {
	User               *UserInfo                  `json:"user"`
	TotalCapital       float64                    `json:"total_capital"`
	AllocatedCapital   float64                    `json:"allocated_capital"`
	TotalReturn        float64                    `json:"total_return"`
	MonthlyReturn      float64                    `json:"monthly_return"`
	SurvivabilityScore int                        `json:"survivability_score"`
	RiskLevel          string                     `json:"risk_level"`
	Allocations        []AllocationWithTraderInfo `json:"allocations"`
	Subscriptions      []Subscription             `json:"subscriptions"`
	TotalAllocatedPct  float64                    `json:"total_allocated_pct"`
}

// AllocationSettings - portfolio level settings
type AllocationSettings struct {
	Mode               string                     `json:"mode"`
	RebalanceFrequency string                     `json:"rebalance_frequency"`
	Allocations        []AllocationWithTraderInfo `json:"allocations"`
}

// AllocationRequest - request to set/update allocation
type AllocationRequest struct {
	TraderAccountID string  `json:"trader_account_id"`
	Mode            string  `json:"mode"`
	Value           float64 `json:"value"`
	MaxRiskPct      float64 `json:"max_risk_pct"`
	MaxPositions    int     `json:"max_positions"`
}

// SubscribeRequest - request to follow a trader
type SubscribeRequest struct {
	TraderAccountID string `json:"trader_account_id"`
}

// TraderListItem - trader account for marketplace/browse list
type TraderListItem struct {
	ID             string  `json:"id"`
	AccountNumber  string  `json:"account_number"`
	Broker         string  `json:"broker"`
	Platform       string  `json:"platform"`
	Nickname       string  `json:"nickname"`
	Equity         float64 `json:"equity"`
	AlphaScore     float64 `json:"alpha_score"`
	Grade          string  `json:"grade"`
	RiskLevel      string  `json:"risk_level"`
	MaxDrawdownPct float64 `json:"max_drawdown_pct"`
	WinRate        float64 `json:"win_rate"`
	NetProfit      float64 `json:"net_profit"`
	TotalTrades    int     `json:"total_trades"`
	ProfitFactor   float64 `json:"profit_factor"`
	BestPairSymbol string  `json:"best_pair_symbol"`
	BestPairScore  float64 `json:"best_pair_score"`
	BestPairGrade  string  `json:"best_pair_grade"`
	TotalDeposit   float64 `json:"total_deposit"`
	TotalWithdraw  float64 `json:"total_withdraw"`
	ROI            float64 `json:"roi"`
}
