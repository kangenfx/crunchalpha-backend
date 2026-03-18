package alpharank

import "time"

// TradeData represents a single trade
type TradeData struct {
	OpenTime   time.Time
	CloseTime  time.Time
	Symbol     string
	Type       string
	Lots       float64
	OpenPrice  float64
	ClosePrice float64
	StopLoss   float64
	TakeProfit float64
	Profit     float64
	Commission float64
	Swap       float64
}

// AccountMetrics contains all metrics needed for calculation
type AccountMetrics struct {
	AccountID      string
	CurrentBalance float64
	CurrentEquity  float64
	InitialDeposit float64
	TotalDeposits  float64
	TotalWithdraws float64
	PeakBalance    float64  // NEW: peak balance for leverage calculations
	NetProfit      float64
	GrossProfit    float64
	GrossLoss      float64
	TotalTrades    int
	WinningTrades  int
	LosingTrades   int
	MaxDrawdownPct float64
	MaxDrawdownAbs float64
	Trades         []TradeData
	StartDate      time.Time
	EndDate        time.Time
}

// PillarScore represents individual pillar calculation
type PillarScore struct {
	Code   string  `json:"code"`
	Name   string  `json:"name"`
	Weight int     `json:"weight"`
	Score  float64 `json:"score"`
	Reason string  `json:"reason"`
}

// RiskFlag represents detected risk
type RiskFlag struct {
	FlagType string  `json:"flag_type"`
	Severity string  `json:"severity"`
	Penalty  float64 `json:"penalty"`
	Title    string  `json:"title"`
	Desc     string  `json:"desc"`
}

// AlphaRankResult complete calculation result
type AlphaRankResult struct {
	AlphaScore float64       `json:"alphaScore"`
	Grade      string        `json:"grade"`
	Tier       string        `json:"tier"`
	Risk       string        `json:"risk"`
	Pillars    []PillarScore `json:"pillars"`
	RiskFlags  struct {
		Counts struct {
			Critical int `json:"critical"`
			Major    int `json:"major"`
			Minor    int `json:"minor"`
		} `json:"counts"`
		Items []RiskFlag `json:"items"`
	} `json:"riskFlags"`
	Survivability struct {
		Score int    `json:"score"`
		Label string `json:"label"`
		Note  string `json:"note"`
	} `json:"survivability"`
	Scalability struct {
		Score int    `json:"score"`
		Label string `json:"label"`
		Note  string `json:"note"`
	} `json:"scalability"`
	CalculatedAt time.Time `json:"calculatedAt"`
}
