package main

import (
	"database/sql"
	"fmt"
	"os"
	"strings"
	"time"
	"encoding/json"
	
	_ "github.com/lib/pq"
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
	fmt.Println("💾 STORING FLAGS TO DATABASE")
	fmt.Println(strings.Repeat("=", 70))
	
	for i, acc := range accounts {
		fmt.Printf("\n[%d/%d] %s\n", i+1, len(accounts), acc.Nickname)
		
		var accountID string
		err := db.QueryRow(`
			SELECT id FROM trader_accounts 
			WHERE user_id = $1 AND account_number = $2
		`, userID, acc.AccountNumber).Scan(&accountID)
		
		if err != nil {
			fmt.Printf("   ⚠️  Account not found\n")
			continue
		}
		
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
		
		fmt.Printf("   📊 Score: %.1f | Grade: %s | Risk: %s\n", 
			result.AlphaScore, result.Grade, result.Risk)
		
		// Delete old flags for this account
		db.Exec(`DELETE FROM alpha_flags WHERE account_id = $1`, accountID)
		
		// Insert new flags with correct schema
		flagCount := 0
		for _, flag := range result.RiskFlags.Items {
			// Map severity to integer
			severityInt := 1 // Minor
			if flag.Severity == "MAJOR" {
				severityInt = 2
			} else if flag.Severity == "CRITICAL" {
				severityInt = 3
			}
			
			_, err := db.Exec(`
				INSERT INTO alpha_flags (
					account_id, flag_type, severity, title, description, value
				) VALUES ($1, $2, $3, $4, $5, $6)
				ON CONFLICT (account_id, flag_type) DO UPDATE SET
					severity = EXCLUDED.severity,
					title = EXCLUDED.title,
					description = EXCLUDED.description,
					triggered_at = NOW()
			`, accountID, flag.FlagType, severityInt, flag.Title, flag.Desc, flag.Penalty)
			
			if err == nil {
				flagCount++
				fmt.Printf("   🚨 [%s] %s (-%0.f points)\n", flag.Severity, flag.Title, flag.Penalty)
			} else {
				fmt.Printf("   ⚠️  Flag error: %v\n", err)
			}
		}
		
		if flagCount == 0 {
			fmt.Printf("   ✅ No flags - Clean trading!\n")
		} else {
			fmt.Printf("   📝 %d flags stored\n", flagCount)
		}
	}
	
	fmt.Println()
	fmt.Println(strings.Repeat("=", 70))
	fmt.Println("✅ FLAGS STORED!")
	fmt.Println(strings.Repeat("=", 70))
}
