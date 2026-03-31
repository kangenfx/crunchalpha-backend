package b2b

import (
	"database/sql"
	"time"
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) SaveBrokerConfig(cfg *BrokerConfig) error {
	query := `
		INSERT INTO b2b_brokers (
			broker_name, broker_code, integration_type, is_active,
			manager_server, manager_login, manager_password, manager_version,
			rest_base_url, rest_api_key, rest_secret,
			created_at, updated_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)
		ON CONFLICT (broker_code) DO UPDATE SET
			broker_name = EXCLUDED.broker_name,
			integration_type = EXCLUDED.integration_type,
			is_active = EXCLUDED.is_active,
			manager_server = EXCLUDED.manager_server,
			manager_login = EXCLUDED.manager_login,
			manager_password = EXCLUDED.manager_password,
			manager_version = EXCLUDED.manager_version,
			rest_base_url = EXCLUDED.rest_base_url,
			rest_api_key = EXCLUDED.rest_api_key,
			rest_secret = EXCLUDED.rest_secret,
			updated_at = EXCLUDED.updated_at
		RETURNING id`
	now := time.Now()
	return r.db.QueryRow(query,
		cfg.BrokerName, cfg.BrokerCode, cfg.IntegrationType, cfg.IsActive,
		cfg.ManagerServer, cfg.ManagerLogin, cfg.ManagerPassword, cfg.ManagerVersion,
		cfg.RestBaseURL, cfg.RestAPIKey, cfg.RestSecret,
		now, now,
	).Scan(&cfg.ID)
}

func (r *Repository) GetActiveBrokers() ([]BrokerConfig, error) {
	query := `
		SELECT id, broker_name, broker_code, integration_type, is_active,
			manager_server, manager_login, manager_password, manager_version,
			rest_base_url, rest_api_key, rest_secret,
			created_at, updated_at
		FROM b2b_brokers WHERE is_active = true`
	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var brokers []BrokerConfig
	for rows.Next() {
		var b BrokerConfig
		err := rows.Scan(
			&b.ID, &b.BrokerName, &b.BrokerCode, &b.IntegrationType, &b.IsActive,
			&b.ManagerServer, &b.ManagerLogin, &b.ManagerPassword, &b.ManagerVersion,
			&b.RestBaseURL, &b.RestAPIKey, &b.RestSecret,
			&b.CreatedAt, &b.UpdatedAt,
		)
		if err != nil {
			continue
		}
		brokers = append(brokers, b)
	}
	return brokers, nil
}

func (r *Repository) UpsertBrokerAccount(acc *BrokerAccount) error {
	query := `
		INSERT INTO b2b_accounts (
			broker_id, account_number, account_name, currency,
			balance, equity, margin, free_margin, leverage,
			is_active, last_sync_at, created_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
		ON CONFLICT (broker_id, account_number) DO UPDATE SET
			account_name = EXCLUDED.account_name,
			balance = EXCLUDED.balance,
			equity = EXCLUDED.equity,
			margin = EXCLUDED.margin,
			free_margin = EXCLUDED.free_margin,
			leverage = EXCLUDED.leverage,
			last_sync_at = EXCLUDED.last_sync_at
		RETURNING id`
	now := time.Now()
	return r.db.QueryRow(query,
		acc.BrokerID, acc.AccountNumber, acc.AccountName, acc.Currency,
		acc.Balance, acc.Equity, acc.Margin, acc.FreeMargin, acc.Leverage,
		acc.IsActive, now, now,
	).Scan(&acc.ID)
}

func (r *Repository) UpsertBrokerTrade(trade *BrokerTrade) error {
	query := `
		INSERT INTO b2b_trades (
			broker_id, account_number, ticket, symbol, type,
			lots, open_price, close_price, profit, swap, commission,
			open_time, close_time, status
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)
		ON CONFLICT (broker_id, ticket) DO UPDATE SET
			close_price = EXCLUDED.close_price,
			profit = EXCLUDED.profit,
			swap = EXCLUDED.swap,
			commission = EXCLUDED.commission,
			close_time = EXCLUDED.close_time,
			status = EXCLUDED.status
		RETURNING id`
	return r.db.QueryRow(query,
		trade.BrokerID, trade.AccountNumber, trade.Ticket, trade.Symbol, trade.Type,
		trade.Lots, trade.OpenPrice, trade.ClosePrice, trade.Profit, trade.Swap, trade.Commission,
		trade.OpenTime, trade.CloseTime, trade.Status,
	).Scan(&trade.ID)
}

func (r *Repository) SaveWhiteLabel(wl *WhiteLabelConfig) error {
	query := `
		INSERT INTO b2b_whitelabel (
			broker_id, brand_name, logo_url, primary_color,
			domain, is_active, created_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7)
		ON CONFLICT (broker_id) DO UPDATE SET
			brand_name = EXCLUDED.brand_name,
			logo_url = EXCLUDED.logo_url,
			primary_color = EXCLUDED.primary_color,
			domain = EXCLUDED.domain,
			is_active = EXCLUDED.is_active
		RETURNING id`
	return r.db.QueryRow(query,
		wl.BrokerID, wl.BrandName, wl.LogoURL, wl.PrimaryColor,
		wl.Domain, wl.IsActive, time.Now(),
	).Scan(&wl.ID)
}

func (r *Repository) GetWhiteLabelByDomain(domain string) (*WhiteLabelConfig, error) {
	query := `
		SELECT id, broker_id, brand_name, logo_url, primary_color, domain, is_active, created_at
		FROM b2b_whitelabel WHERE domain = $1 AND is_active = true`
	var wl WhiteLabelConfig
	err := r.db.QueryRow(query, domain).Scan(
		&wl.ID, &wl.BrokerID, &wl.BrandName, &wl.LogoURL,
		&wl.PrimaryColor, &wl.Domain, &wl.IsActive, &wl.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &wl, nil
}
