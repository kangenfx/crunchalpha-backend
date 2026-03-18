package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"
	
	"crunchalpha-v3/internal/alpharank"
)

type TradeJSON struct {
	OpenTime  string  `json:"open_time"`
	CloseTime string  `json:"close_time"`
	Symbol    string  `json:"symbol"`
	Type      string  `json:"type"`
	Lots      float64 `json:"lots"`
	Profit    float64 `json:"profit"`
	StopLoss  float64 `json:"stop_loss"`
}

type Account struct {
	AccountNumber string      `json:"account"`
	Nickname      string      `json:"nickname"`
	Trades        []TradeJSON `json:"trades"`
	Stats         struct {
		TotalTrades      int     `json:"total_trades"`
		WinningTrades    int     `json:"winning_trades"`
		LosingTrades     int     `json:"losing_trades"`
		GrossProfit      float64 `json:"gross_profit"`
		GrossLoss        float64 `json:"gross_loss"`
		TotalProfit      float64 `json:"total_profit"`
		TotalDeposits    float64 `json:"total_deposits"`
		TotalWithdrawals float64 `json:"total_withdrawals"`
	} `json:"stats"`
}

func main() {
	data, _ := os.ReadFile("/tmp/accounts_full_trades.json")
	var accounts []Account
	json.Unmarshal(data, &accounts)
	
	calc := alpharank.NewCalculator()
	
	fmt.Println(strings.Repeat("=", 70))
	fmt.Println("🚨 TESTING ALL RISK FLAG DETECTORS")
	fmt.Println(strings.Repeat("=", 70))
	
	for i, acc := range accounts {
		fmt.Printf("\n[%d/%d] %s (%d trades)\n", i+1, len(accounts), acc.Nickname, len(acc.Trades))
		
		trades := make([]alpharank.TradeData, len(acc.Trades))
		for j, t := range acc.Trades {
			openTime, _ := time.Parse(time.RFC3339, t.OpenTime)
			closeTime, _ := time.Parse(time.RFC3339, t.CloseTime)
			
			trades[j] = alpharank.TradeData{
				OpenTime:  openTime,
				CloseTime: closeTime,
				Symbol:    t.Symbol,
				Type:      t.Type,
				Lots:      t.Lots,
				StopLoss:  t.StopLoss,
				Profit:    t.Profit,
			}
		}
		
		balance := acc.Stats.TotalDeposits - acc.Stats.TotalWithdrawals + acc.Stats.TotalProfit
		deposits := acc.Stats.TotalDeposits
		if deposits == 0 {
			deposits = 1000
		}
		
		metrics := alpharank.AccountMetrics{
			AccountID:      acc.AccountNumber,
			CurrentBalance: balance,
			CurrentEquity:  balance,
			InitialDeposit: deposits,
			TotalDeposits:  deposits,
			TotalWithdraws: acc.Stats.TotalWithdrawals,
			NetProfit:      acc.Stats.TotalProfit,
			GrossProfit:    acc.Stats.GrossProfit,
			GrossLoss:      acc.Stats.GrossLoss,
			TotalTrades:    acc.Stats.TotalTrades,
			WinningTrades:  acc.Stats.WinningTrades,
			LosingTrades:   acc.Stats.LosingTrades,
			MaxDrawdownPct: 20.0,
			Trades:         trades,
			StartDate:      time.Now().AddDate(0, -4, 0),
			EndDate:        time.Now(),
		}
		
		result := calc.Calculate(metrics)
		
		fmt.Printf("📊 Score: %.1f | Grade: %s | Risk: %s\n", 
			result.AlphaScore, result.Grade, result.Risk)
		
		if len(result.RiskFlags.Items) == 0 {
			fmt.Println("✅ No flags detected\n")
		} else {
			fmt.Printf("🚨 %d FLAGS: Critical=%d Major=%d Minor=%d\n", 
				len(result.RiskFlags.Items),
				result.RiskFlags.Counts.Critical,
				result.RiskFlags.Counts.Major,
				result.RiskFlags.Counts.Minor)
			
			for _, flag := range result.RiskFlags.Items {
				fmt.Printf("   [%s] %s (%.0f pts)\n", flag.Severity, flag.Title, flag.Penalty)
			}
			fmt.Println()
		}
	}
	
	fmt.Println(strings.Repeat("=", 70))
	fmt.Println("✅ TEST COMPLETE")
	fmt.Println(strings.Repeat("=", 70))
}
