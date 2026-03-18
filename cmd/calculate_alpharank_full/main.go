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
	OpenTime    string  `json:"open_time"`
	CloseTime   string  `json:"close_time"`
	Symbol      string  `json:"symbol"`
	Type        string  `json:"type"`
	Lots        float64 `json:"lots"`
	OpenPrice   float64 `json:"open_price"`
	ClosePrice  float64 `json:"close_price"`
	StopLoss    float64 `json:"stop_loss"`
	TakeProfit  float64 `json:"take_profit"`
	Profit      float64 `json:"profit"`
}

type Account struct {
	AccountNumber string      `json:"account"`
	Nickname      string      `json:"nickname"`
	Trades        []TradeJSON `json:"trades"`
	Stats         struct {
		TotalTrades      int     `json:"total_trades"`
		WinningTrades    int     `json:"winning_trades"`
		LosingTrades     int     `json:"losing_trades"`
		WinRate          float64 `json:"win_rate"`
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
	fmt.Println("🔮 ALPHARANK WITH FULL TRADE DATA + FLAG DETECTION")
	fmt.Println(strings.Repeat("=", 70))
	
	for i, acc := range accounts {
		fmt.Printf("\n%s\n", strings.Repeat("=", 70))
		fmt.Printf("Account %d: %s (%s)\n", i+1, acc.Nickname, acc.AccountNumber)
		fmt.Printf("%s\n", strings.Repeat("=", 70))
		
		// Convert trades to AlphaRank format
		trades := make([]alpharank.TradeData, len(acc.Trades))
		for j, t := range acc.Trades {
			openTime, _ := time.Parse(time.RFC3339, t.OpenTime)
			closeTime, _ := time.Parse(time.RFC3339, t.CloseTime)
			
			trades[j] = alpharank.TradeData{
				OpenTime:   openTime,
				CloseTime:  closeTime,
				Symbol:     t.Symbol,
				Type:       t.Type,
				Lots:       t.Lots,
				OpenPrice:  t.OpenPrice,
				ClosePrice: t.ClosePrice,
				StopLoss:   t.StopLoss,
				TakeProfit: t.TakeProfit,
				Profit:     t.Profit,
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
			Trades:         trades, // ← FULL TRADE DATA!
			StartDate:      time.Now().AddDate(0, -4, 0),
			EndDate:        time.Now(),
		}
		
		result := calc.Calculate(metrics)
		
		fmt.Printf("\n🎯 ALPHARANK RESULTS:\n")
		fmt.Printf("   Score: %.1f/100 | Grade: %s | Tier: %s | Risk: %s\n", 
			result.AlphaScore, result.Grade, result.Tier, result.Risk)
		
		fmt.Println("\n📊 7 Pillars:")
		for _, p := range result.Pillars {
			fmt.Printf("   %s (%2d%%): %3.0f/100 - %s\n", p.Code, p.Weight, p.Score, p.Name)
		}
		
		fmt.Printf("\n🚨 Risk Flags: Critical=%d, Major=%d, Minor=%d\n", 
			result.RiskFlags.Counts.Critical, result.RiskFlags.Counts.Major, result.RiskFlags.Counts.Minor)
		
		if len(result.RiskFlags.Items) > 0 {
			fmt.Println("\n   ⚠️  DETECTED FLAGS:")
			for _, flag := range result.RiskFlags.Items {
				fmt.Printf("   - [%s] %s: %s\n", flag.Severity, flag.Title, flag.ScoreText)
				fmt.Printf("     → %s\n", flag.Desc)
			}
		} else {
			fmt.Println("   ✅ No risk flags detected - Clean trading!")
		}
		
		fmt.Printf("\n💪 Survivability: %d/100 (%s)\n", result.Survivability.Score, result.Survivability.Label)
		fmt.Printf("📈 Scalability: %d/100 (%s)\n", result.Scalability.Score, result.Scalability.Label)
	}
	
	fmt.Println("\n" + strings.Repeat("=", 70))
	fmt.Println("✅ ALPHARANK CALCULATION COMPLETE!")
	fmt.Println(strings.Repeat("=", 70))
	fmt.Println("\n🎉 All accounts scored with FULL trade data!")
	fmt.Println("📊 P2, P6, and FLAG DETECTION now ACTIVE!")
}
