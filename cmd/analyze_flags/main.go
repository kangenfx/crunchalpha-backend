package main

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"strings"
	"time"
)

type TradeJSON struct {
	OpenTime  string  `json:"open_time"`
	CloseTime string  `json:"close_time"`
	Lots      float64 `json:"lots"`
	Profit    float64 `json:"profit"`
	StopLoss  float64 `json:"stop_loss"`
}

type Account struct {
	AccountNumber string      `json:"account"`
	Nickname      string      `json:"nickname"`
	Trades        []TradeJSON `json:"trades"`
	Stats         struct {
		TotalProfit      float64 `json:"total_profit"`
		TotalDeposits    float64 `json:"total_deposits"`
		TotalWithdrawals float64 `json:"total_withdrawals"`
	} `json:"stats"`
}

func main() {
	data, _ := os.ReadFile("/tmp/accounts_full_trades.json")
	var accounts []Account
	json.Unmarshal(data, &accounts)
	
	fmt.Println(strings.Repeat("=", 70))
	fmt.Println("🔍 FLAG CONDITION ANALYSIS")
	fmt.Println(strings.Repeat("=", 70))
	
	for i := 0; i < 2; i++ {
		acc := accounts[i]
		fmt.Printf("\n%s - %d trades\n", acc.Nickname, len(acc.Trades))
		
		equity := acc.Stats.TotalDeposits - acc.Stats.TotalWithdrawals + acc.Stats.TotalProfit
		if equity == 0 {
			equity = 1000
		}
		
		// MARTINGALE
		doublingCount := 0
		for j := 1; j < len(acc.Trades); j++ {
			if acc.Trades[j-1].Profit < 0 && acc.Trades[j-1].Lots > 0 {
				ratio := acc.Trades[j].Lots / acc.Trades[j-1].Lots
				if ratio >= 1.9 && ratio <= 2.1 {
					doublingCount++
				}
			}
		}
		fmt.Printf("  Martingale: %d doublings (need ≥3)\n", doublingCount)
		
		// OVERLEVERAGING
		overleveraged := 0
		for _, t := range acc.Trades {
			if t.Profit < 0 {
				lossPct := math.Abs(t.Profit) / equity * 100
				if lossPct > 5 {
					overleveraged++
				}
			}
		}
		pct := float64(overleveraged) / float64(len(acc.Trades)) * 100
		fmt.Printf("  Overleveraging: %.1f%% trades (need >20%%)\n", pct)
		
		// REVENGE_TRADING
		revengeCount := 0
		for j := 1; j < len(acc.Trades); j++ {
			if acc.Trades[j-1].Profit < 0 && acc.Trades[j-1].Lots > 0 {
				lotIncrease := (acc.Trades[j].Lots/acc.Trades[j-1].Lots - 1) * 100
				if math.Abs(acc.Trades[j-1].Profit) > 50 && lotIncrease > 100 {
					revengeCount++
				}
			}
		}
		fmt.Printf("  Revenge trading: %d incidents\n", revengeCount)
		
		// WEEKEND_EXPOSURE
		weekendCount := 0
		for _, t := range acc.Trades {
			openTime, _ := time.Parse(time.RFC3339, t.OpenTime)
			if openTime.Weekday() == time.Friday {
				weekendCount++
			}
		}
		fmt.Printf("  Weekend exposure: %d trades\n", weekendCount)
		
		// NO_STOP_LOSS
		noSL := 0
		for _, t := range acc.Trades {
			if t.StopLoss == 0 {
				noSL++
			}
		}
		slPct := float64(noSL) / float64(len(acc.Trades)) * 100
		fmt.Printf("  No Stop Loss: %.1f%% (threshold >70%%)\n", slPct)
	}
	
	fmt.Println(strings.Repeat("=", 70))
}
