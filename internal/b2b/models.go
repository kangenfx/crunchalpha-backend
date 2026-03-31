package b2b

import "time"

// BrokerConfig — konfigurasi broker partner
type BrokerConfig struct {
	ID              int64     `json:"id"`
	BrokerName      string    `json:"broker_name"`
	BrokerCode      string    `json:"broker_code"`
	IntegrationType string    `json:"integration_type"`
	IsActive        bool      `json:"is_active"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`

	// Manager API credentials (MT4/MT5)
	ManagerServer   string `json:"manager_server,omitempty"`
	ManagerLogin    int64  `json:"manager_login,omitempty"`
	ManagerPassword string `json:"manager_password,omitempty"`
	ManagerVersion  string `json:"manager_version,omitempty"`

	// REST API credentials
	RestBaseURL string `json:"rest_base_url,omitempty"`
	RestAPIKey  string `json:"rest_api_key,omitempty"`
	RestSecret  string `json:"rest_secret,omitempty"`
}

// BrokerAccount — akun trader yang di-pull dari broker
type BrokerAccount struct {
	ID            int64     `json:"id"`
	BrokerID      int64     `json:"broker_id"`
	AccountNumber string    `json:"account_number"`
	AccountName   string    `json:"account_name"`
	Currency      string    `json:"currency"`
	Balance       float64   `json:"balance"`
	Equity        float64   `json:"equity"`
	Margin        float64   `json:"margin"`
	FreeMargin    float64   `json:"free_margin"`
	Leverage      int       `json:"leverage"`
	IsActive      bool      `json:"is_active"`
	LastSyncAt    time.Time `json:"last_sync_at"`
	CreatedAt     time.Time `json:"created_at"`
}

// BrokerTrade — trade yang di-pull dari broker
type BrokerTrade struct {
	ID            int64     `json:"id"`
	BrokerID      int64     `json:"broker_id"`
	AccountNumber string    `json:"account_number"`
	Ticket        int64     `json:"ticket"`
	Symbol        string    `json:"symbol"`
	Type          string    `json:"type"`
	Lots          float64   `json:"lots"`
	OpenPrice     float64   `json:"open_price"`
	ClosePrice    float64   `json:"close_price,omitempty"`
	Profit        float64   `json:"profit,omitempty"`
	Swap          float64   `json:"swap,omitempty"`
	Commission    float64   `json:"commission,omitempty"`
	OpenTime      time.Time `json:"open_time"`
	CloseTime     time.Time `json:"close_time,omitempty"`
	Status        string    `json:"status"`
}

// SyncResult — hasil sync dari broker
type SyncResult struct {
	BrokerID       int64     `json:"broker_id"`
	BrokerCode     string    `json:"broker_code"`
	AccountsSynced int       `json:"accounts_synced"`
	TradesSynced   int       `json:"trades_synced"`
	Errors         []string  `json:"errors,omitempty"`
	SyncedAt       time.Time `json:"synced_at"`
}

// WhiteLabelConfig — konfigurasi white label per client
type WhiteLabelConfig struct {
	ID           int64     `json:"id"`
	BrokerID     int64     `json:"broker_id"`
	BrandName    string    `json:"brand_name"`
	LogoURL      string    `json:"logo_url"`
	PrimaryColor string    `json:"primary_color"`
	Domain       string    `json:"domain"`
	IsActive     bool      `json:"is_active"`
	CreatedAt    time.Time `json:"created_at"`
}
