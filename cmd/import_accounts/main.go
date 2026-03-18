package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"
	
	_ "github.com/lib/pq"
	"crunchalpha-v3/internal/alpharank"
)

type TradeJSON struct {
	OpenTime    string  `json:"open_time"`
	CloseTime   string  `json:"close_time"`
	Symbol      string  `json:"symbol"`
	Type        string  `json:"type"`
	Lots        float64 `json:"lots"`
	Profit      float64 `json:"profit"`
	StopLoss    float64 `json:"stop_loss"`
}

type Account struct {
	AccountNumber string      `json:"account"`
	Nickname      string      `json:"nickname"`
	Name          string      `json:"name"`
	Broker        string      `json:"broker"`
	Platform      string      `json:"platform"`
	Currency      string      `json:"currency"`
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
	dbURL := "postgres://alpha_user:alpha_password@172.18.0.2:5432/crunchalpha?sslmode=disable"
	
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		fmt.Printf("❌ Connection failed: %v\n", err)
		return
	}
	defer db.Close()
	
	if err := db.Ping(); err != nil {
		fmt.Printf("❌ Ping failed: %v\n", err)
		return
	}
	
	fmt.Println("✅ Connected to database")
	
	data, _ := os.ReadFile("/tmp/accounts_full_trades.json")
	var accounts []Account
	json.Unmarshal(data, &accounts)
	
	calc := alpharank.NewCalculator()
	userID := "ecb6ef49-988d-4432-9895-48e0d88656b7"
	
	fmt.Println()
	fmt.Println(strings.Repeat("=", 70))
	fmt.Println("📥 IMPORTING 3 ACCOUNTS")
	fmt.Println(strings.Repeat("=", 70))
	
	successCount := 0
	
	for i, acc := range accounts {
		fmt.Printf("\n[%d/%d] %s (%s)\n", i+1, len(accounts), acc.Nickname, acc.AccountNumber)
		
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
		
		// Insert account - matching actual schema
		var accountID string
		err := db.QueryRow(`
			INSERT INTO trader_accounts (
				user_id, 
				platform, 
				account_number, 
				investor_password,
				broker, 
				server,
				role,
				status,
				nickname,
				currency,
				account_type
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
			ON CONFLICT (user_id, platform, account_number, server) 
			DO UPDATE SET 
				nickname = EXCLUDED.nickname,
				updated_at = NOW()
			RETURNING id
		`, userID, acc.Platform, acc.AccountNumber, "dummy_password", 
			acc.Broker, "", "provider", "active", acc.Nickname, acc.Currency, "real").Scan(&accountID)
		
		if err != nil {
			fmt.Printf("   ⚠️  Insert failed: %v\n", err)
			continue
		}
		
		fmt.Printf("   ✅ Saved (ID: %s...)\n", accountID[:8])
		fmt.Printf("   📊 AlphaScore: %.1f | Grade: %s | Tier: %s | Risk: %s", 
			result.AlphaScore, result.Grade, result.Tier, result.Risk)
		
		if result.RiskFlags.Counts.Critical > 0 {
			fmt.Printf(" 🚨")
		}
		fmt.Println()
		
		successCount++
	}
	
	fmt.Println()
	fmt.Println(strings.Repeat("=", 70))
	fmt.Println("✅ IMPORT COMPLETE!")
	fmt.Println(strings.Repeat("=", 70))
	fmt.Printf("\n📊 %d/%d accounts imported\n", successCount, len(accounts))
	fmt.Println("🚀 Accounts ready for API!")
}
