package b2b

import (
	"fmt"
	"log"
	"time"
)

type ManagerConnector struct {
	repo *Repository
}

func NewManagerConnector(repo *Repository) *ManagerConnector {
	return &ManagerConnector{repo: repo}
}

func (m *ManagerConnector) Sync(broker *BrokerConfig) (*SyncResult, error) {
	result := &SyncResult{
		BrokerID:   broker.ID,
		BrokerCode: broker.BrokerCode,
		SyncedAt:   time.Now(),
	}
	log.Printf("[B2B Manager] Starting sync for broker: %s (%s)", broker.BrokerName, broker.ManagerVersion)
	switch broker.ManagerVersion {
	case "mt4":
		return m.syncMT4(broker, result)
	case "mt5":
		return m.syncMT5(broker, result)
	default:
		return nil, fmt.Errorf("unsupported manager version: %s", broker.ManagerVersion)
	}
}

func (m *ManagerConnector) syncMT4(broker *BrokerConfig, result *SyncResult) (*SyncResult, error) {
	log.Printf("[B2B MT4] Broker: %s | Server: %s | Login: %d",
		broker.BrokerName, broker.ManagerServer, broker.ManagerLogin)
	mockAccounts := m.getMockAccounts(broker.ID)
	for _, acc := range mockAccounts {
		if err := m.repo.UpsertBrokerAccount(&acc); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("account %s: %v", acc.AccountNumber, err))
			continue
		}
		result.AccountsSynced++
	}
	mockTrades := m.getMockTrades(broker.ID)
	for _, trade := range mockTrades {
		if err := m.repo.UpsertBrokerTrade(&trade); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("trade %d: %v", trade.Ticket, err))
			continue
		}
		result.TradesSynced++
	}
	log.Printf("[B2B MT4] Sync complete — accounts: %d, trades: %d", result.AccountsSynced, result.TradesSynced)
	return result, nil
}

func (m *ManagerConnector) syncMT5(broker *BrokerConfig, result *SyncResult) (*SyncResult, error) {
	log.Printf("[B2B MT5] Broker: %s | Server: %s | Login: %d",
		broker.BrokerName, broker.ManagerServer, broker.ManagerLogin)
	mockAccounts := m.getMockAccounts(broker.ID)
	for _, acc := range mockAccounts {
		if err := m.repo.UpsertBrokerAccount(&acc); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("account %s: %v", acc.AccountNumber, err))
			continue
		}
		result.AccountsSynced++
	}
	mockTrades := m.getMockTrades(broker.ID)
	for _, trade := range mockTrades {
		if err := m.repo.UpsertBrokerTrade(&trade); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("trade %d: %v", trade.Ticket, err))
			continue
		}
		result.TradesSynced++
	}
	log.Printf("[B2B MT5] Sync complete — accounts: %d, trades: %d", result.AccountsSynced, result.TradesSynced)
	return result, nil
}

func (m *ManagerConnector) getMockAccounts(brokerID int64) []BrokerAccount {
	return []BrokerAccount{
		{
			BrokerID:      brokerID,
			AccountNumber: "MOCK-001",
			AccountName:   "Test Trader 1",
			Currency:      "USD",
			Balance:       10000.00,
			Equity:        10250.00,
			Margin:        500.00,
			FreeMargin:    9750.00,
			Leverage:      100,
			IsActive:      true,
		},
		{
			BrokerID:      brokerID,
			AccountNumber: "MOCK-002",
			AccountName:   "Test Trader 2",
			Currency:      "USD",
			Balance:       25000.00,
			Equity:        24800.00,
			Margin:        1200.00,
			FreeMargin:    23600.00,
			Leverage:      200,
			IsActive:      true,
		},
	}
}

func (m *ManagerConnector) getMockTrades(brokerID int64) []BrokerTrade {
	now := time.Now()
	return []BrokerTrade{
		{
			BrokerID:      brokerID,
			AccountNumber: "MOCK-001",
			Ticket:        100001,
			Symbol:        "XAUUSD",
			Type:          "buy",
			Lots:          0.10,
			OpenPrice:     2320.50,
			Profit:        25.00,
			OpenTime:      now.Add(-24 * time.Hour),
			Status:        "open",
		},
		{
			BrokerID:      brokerID,
			AccountNumber: "MOCK-001",
			Ticket:        100002,
			Symbol:        "EURUSD",
			Type:          "sell",
			Lots:          0.20,
			OpenPrice:     1.0850,
			ClosePrice:    1.0820,
			Profit:        60.00,
			OpenTime:      now.Add(-48 * time.Hour),
			CloseTime:     now.Add(-24 * time.Hour),
			Status:        "closed",
		},
	}
}
