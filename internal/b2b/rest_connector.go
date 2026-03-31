package b2b

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

type RestConnector struct {
	repo   *Repository
	client *http.Client
}

func NewRestConnector(repo *Repository) *RestConnector {
	return &RestConnector{
		repo:   repo,
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

func (r *RestConnector) Sync(broker *BrokerConfig) (*SyncResult, error) {
	result := &SyncResult{
		BrokerID:   broker.ID,
		BrokerCode: broker.BrokerCode,
		SyncedAt:   time.Now(),
	}
	log.Printf("[B2B REST] Starting sync for broker: %s | URL: %s", broker.BrokerName, broker.RestBaseURL)
	accounts, err := r.fetchAccounts(broker)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch accounts: %w", err)
	}
	for _, acc := range accounts {
		if err := r.repo.UpsertBrokerAccount(&acc); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("account %s: %v", acc.AccountNumber, err))
			continue
		}
		result.AccountsSynced++
	}
	trades, err := r.fetchTrades(broker)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch trades: %w", err)
	}
	for _, trade := range trades {
		if err := r.repo.UpsertBrokerTrade(&trade); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("trade %d: %v", trade.Ticket, err))
			continue
		}
		result.TradesSynced++
	}
	log.Printf("[B2B REST] Sync complete — accounts: %d, trades: %d", result.AccountsSynced, result.TradesSynced)
	return result, nil
}

func (r *RestConnector) fetchAccounts(broker *BrokerConfig) ([]BrokerAccount, error) {
	url := fmt.Sprintf("%s/accounts", broker.RestBaseURL)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+broker.RestAPIKey)
	req.Header.Set("Content-Type", "application/json")
	resp, err := r.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("broker API returned status: %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var result struct {
		Accounts []struct {
			AccountNumber string  `json:"account_number"`
			AccountName   string  `json:"account_name"`
			Currency      string  `json:"currency"`
			Balance       float64 `json:"balance"`
			Equity        float64 `json:"equity"`
			Margin        float64 `json:"margin"`
			FreeMargin    float64 `json:"free_margin"`
			Leverage      int     `json:"leverage"`
		} `json:"accounts"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse accounts: %w", err)
	}
	accounts := make([]BrokerAccount, 0, len(result.Accounts))
	for _, a := range result.Accounts {
		accounts = append(accounts, BrokerAccount{
			BrokerID:      broker.ID,
			AccountNumber: a.AccountNumber,
			AccountName:   a.AccountName,
			Currency:      a.Currency,
			Balance:       a.Balance,
			Equity:        a.Equity,
			Margin:        a.Margin,
			FreeMargin:    a.FreeMargin,
			Leverage:      a.Leverage,
			IsActive:      true,
		})
	}
	return accounts, nil
}

func (r *RestConnector) fetchTrades(broker *BrokerConfig) ([]BrokerTrade, error) {
	url := fmt.Sprintf("%s/trades", broker.RestBaseURL)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+broker.RestAPIKey)
	req.Header.Set("Content-Type", "application/json")
	resp, err := r.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("broker API returned status: %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var result struct {
		Trades []struct {
			Ticket     int64   `json:"ticket"`
			Account    string  `json:"account_number"`
			Symbol     string  `json:"symbol"`
			Type       string  `json:"type"`
			Lots       float64 `json:"lots"`
			OpenPrice  float64 `json:"open_price"`
			ClosePrice float64 `json:"close_price"`
			Profit     float64 `json:"profit"`
			Swap       float64 `json:"swap"`
			Commission float64 `json:"commission"`
			OpenTime   int64   `json:"open_time"`
			CloseTime  int64   `json:"close_time"`
			Status     string  `json:"status"`
		} `json:"trades"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse trades: %w", err)
	}
	trades := make([]BrokerTrade, 0, len(result.Trades))
	for _, t := range result.Trades {
		trade := BrokerTrade{
			BrokerID:      broker.ID,
			AccountNumber: t.Account,
			Ticket:        t.Ticket,
			Symbol:        t.Symbol,
			Type:          t.Type,
			Lots:          t.Lots,
			OpenPrice:     t.OpenPrice,
			ClosePrice:    t.ClosePrice,
			Profit:        t.Profit,
			Swap:          t.Swap,
			Commission:    t.Commission,
			Status:        t.Status,
		}
		if t.OpenTime > 0 {
			trade.OpenTime = time.Unix(t.OpenTime, 0)
		}
		if t.CloseTime > 0 {
			trade.CloseTime = time.Unix(t.CloseTime, 0)
		}
		trades = append(trades, trade)
	}
	return trades, nil
}
