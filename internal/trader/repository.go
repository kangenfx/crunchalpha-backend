package trader

import (
	"github.com/google/uuid"
	"time"
	"database/sql"
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

// GetUserAccounts retrieves all trading accounts for a user
func (r *Repository) GetUserAccounts(userID string) ([]TraderAccount, error) {
	rows, err := r.db.Query(`
		SELECT id, user_id, COALESCE(nickname, account_number) as nickname, 
		       broker, platform, COALESCE(currency, 'USD') as currency,
		       account_number, account_type, status, updated_at, created_at,
		       COALESCE(about, '') as about,
		       COALESCE(ea_verified, false) as ea_verified,
		       COALESCE(connection_status, 'pending') as connection_status,
		       last_sync_at,
		       ea_first_push_at
		FROM trader_accounts
		WHERE user_id = $1
		ORDER BY 
			CASE status 
				WHEN 'linked' THEN 1
				WHEN 'active' THEN 1
				WHEN 'pending' THEN 2
				WHEN 'disabled' THEN 3
				ELSE 4
			END,
			created_at DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var accounts []TraderAccount
	for rows.Next() {
		var acc TraderAccount
		
		err := rows.Scan(
			&acc.ID, &acc.UserID, &acc.Nickname, &acc.Broker,
			&acc.Platform, &acc.Currency, &acc.AccountNumber,
			&acc.AccountType, &acc.Status, &acc.UpdatedAt, &acc.CreatedAt,
			&acc.About, &acc.EaVerified, &acc.LastSyncAt, &acc.EaFirstPushAt, &acc.ConnectionStatus,
		)
		if err != nil {
			return nil, err
		}
		
		accounts = append(accounts, acc)
	}

	return accounts, rows.Err()
}

// GetAccountByID retrieves specific account
func (r *Repository) GetAccountByID(accountID, userID string) (*TraderAccount, error) {
	var acc TraderAccount
	
	err := r.db.QueryRow(`
		SELECT id, user_id, COALESCE(nickname, account_number) as nickname,
		       broker, platform, COALESCE(currency, 'USD') as currency,
		       account_number, account_type, status, updated_at, created_at
		FROM trader_accounts
		WHERE id = $1 AND user_id = $2
	`, accountID, userID).Scan(
		&acc.ID, &acc.UserID, &acc.Nickname, &acc.Broker,
		&acc.Platform, &acc.Currency, &acc.AccountNumber,
		&acc.AccountType, &acc.Status, &acc.UpdatedAt, &acc.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, sql.ErrNoRows
	}
	
	return &acc, err
}

// CreateDummyAccount creates a test account (using correct enum values)
func (r *Repository) CreateDummyAccount(userID, nickname, broker, platform string) (*TraderAccount, error) {
	now := time.Now()
	acc := &TraderAccount{
		UserID:        userID,
		Nickname:      nickname,
		Broker:        broker,
		Platform:      platform,
		Currency:      "USD",
		AccountNumber: "12345678",
		AccountType:   "demo",
		Status:        "active",
		UpdatedAt:     now,
		CreatedAt:     now,
	}

	// Use 'provider' as the role (from the enum), not 'master'
	err := r.db.QueryRow(`
		INSERT INTO trader_accounts 
		(user_id, nickname, broker, platform, currency, account_number, 
		 account_type, role, status, investor_password, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, 'provider', $8, 'dummy123', $9, $10)
		RETURNING id
	`, acc.UserID, acc.Nickname, acc.Broker, acc.Platform, acc.Currency,
		acc.AccountNumber, acc.AccountType, acc.Status, acc.CreatedAt, acc.UpdatedAt,
	).Scan(&acc.ID)

	return acc, err
}

// GetTradesByAccount returns all trades for an account
func (r *Repository) GetTradesByAccount(accountID string) ([]map[string]interface{}, error) {
	rows, err := r.db.Query(`
		SELECT
			id, account_id, symbol, type, lots,
			open_price, close_price,
			open_time, close_time,
			profit, commission, swap, status
		FROM trades
		WHERE account_id = $1
		ORDER BY close_time DESC NULLS LAST, open_time DESC
	`, accountID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	trades := []map[string]interface{}{}
	for rows.Next() {
		var (
			id, accountID, symbol, tradeType string
			lots, openPrice, closePrice      float64
			profit, commission, swap         float64
			status                           string
			openTime, closeTime              *time.Time
		)

		err := rows.Scan(
			&id, &accountID, &symbol, &tradeType, &lots,
			&openPrice, &closePrice,
			&openTime, &closeTime,
			&profit, &commission, &swap, &status,
		)
		if err != nil {
			continue
		}

		trade := map[string]interface{}{
			"id":          id,
			"account_id":  accountID,
			"symbol":      symbol,
			"type":        tradeType,
			"lots":        lots,
			"open_price":  openPrice,
			"close_price": closePrice,
			"profit":      profit,
			"commission":  commission,
			"swap":        swap,
			"status":      status,
			"open_time":   nil,
			"close_time":  nil,
		}

		if openTime != nil {
			trade["open_time"] = openTime.Format(time.RFC3339)
		}
		if closeTime != nil {
			trade["close_time"] = closeTime.Format(time.RFC3339)
		}

		trades = append(trades, trade)
	}

	return trades, nil
}



// DeleteAccount deletes a trader account (CASCADE handles related data)
func (r *Repository) DeleteAccount(accountID, userID string) error {
	result, err := r.db.Exec(`
		DELETE FROM trader_accounts 
		WHERE id = $1 AND user_id = $2
	`, accountID, userID)
	
	if err != nil {
		return err
	}
	
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return sql.ErrNoRows
	}
	
	return nil
}


// GetAccountByNumber finds account by number and user
func (r *Repository) GetAccountByNumber(accountNumber, userID string) (*TraderAccount, error) {
	var account TraderAccount
	query := `SELECT id, account_number, broker, platform FROM trader_accounts WHERE account_number = $1 AND user_id = $2`
	err := r.db.QueryRow(query, accountNumber, userID).Scan(&account.ID, &account.AccountNumber, &account.Broker, &account.Platform)
	if err != nil {
		return nil, err
	}
	return &account, nil
}

// generateUUID creates a simple UUID
// CreateAccount creates a new trading account
func (r *Repository) CreateAccount(userID, accountNumber, broker, platform, nickname, currency string) (*TraderAccount, error) {
	if currency == "" {
		currency = "USD"
	}
	
	account := &TraderAccount{
		ID:            generateUUID(),
		UserID:        userID,
		AccountNumber: accountNumber,
		Broker:        broker,
		Platform:      platform,
		Nickname:      nickname,
		Currency:      currency,
		Balance:       0,
		Equity:        0,
		Status:        "active",
	}

	query := `
		INSERT INTO trader_accounts (
			id, user_id, account_number, broker, platform, 
			nickname, currency, balance, equity, status, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, NOW())
		RETURNING id, account_number, broker, platform, created_at
	`

	err := r.db.QueryRow(
		query,
		account.ID, account.UserID, account.AccountNumber, account.Broker, account.Platform,
		account.Nickname, account.Currency, account.Balance, account.Equity, account.Status,
	).Scan(&account.ID, &account.AccountNumber, &account.Broker, &account.Platform, &account.CreatedAt)

	return account, err
}
// CreateAccountFull creates a new trading account with all fields


func (r *Repository) GetTradesByAccountPaginated(accountID string, limit, offset int) ([]map[string]interface{}, int, error) {
	// Get total count
	var total int
	err := r.db.QueryRow(`SELECT COUNT(*) FROM trades WHERE account_id = $1 AND status = 'closed'`, accountID).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	rows, err := r.db.Query(`
		SELECT id, symbol, type, lots,
		       open_time, close_time,
		       open_price, close_price,
		       COALESCE(profit,0)+COALESCE(swap,0)+COALESCE(commission,0) as net_profit
		FROM trades
		WHERE account_id = $1 AND status = 'closed'
		ORDER BY close_time DESC
		LIMIT $2 OFFSET $3
	`, accountID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var trades []map[string]interface{}
	for rows.Next() {
		var id, symbol, tradeType string
		var lots, openPrice, closePrice, netProfit float64
		var openTime, closeTime interface{}
		if err := rows.Scan(&id, &symbol, &tradeType, &lots, &openTime, &closeTime, &openPrice, &closePrice, &netProfit); err != nil {
			continue
		}
		trades = append(trades, map[string]interface{}{
			"id":          id,
			"symbol":      symbol,
			"type":        tradeType,
			"lots":        lots,
			"open_time":   openTime,
			"close_time":  closeTime,
			"open_price":  openPrice,
			"close_price": closePrice,
			"profit":      netProfit,
		})
	}
	if trades == nil {
		trades = []map[string]interface{}{}
	}
	return trades, total, rows.Err()
}

func (r *Repository) UpdateAccountMeta(accountID, userID, nickname, about string) error {
	_, err := r.db.Exec(`
		UPDATE trader_accounts
		SET nickname = $1, about = $2
		WHERE id = $3 AND user_id = $4
	`, nickname, about, accountID, userID)
	return err
}


func (r *Repository) CreateAccountFull(userID, accountNumber, broker, platform, server, investorPassword, nickname, currency, role, about string) (*TraderAccount, error) {
	if currency == "" {
		currency = "USD"
	}
	if role == "" {
		role = "trader"
	}
	
	accountID := generateUUID()

	// TODO: Encrypt investor_password before storing (for production security)
	// For now, store as-is (will add encryption later)
	
	query := `
		INSERT INTO trader_accounts (
			id, user_id, account_number, broker, platform, 
			server, investor_password, nickname, currency, 
			account_type, role, balance, equity, status, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, 'provider', $11, $12, $13, NOW())
		RETURNING id, account_number, broker, platform, server, account_type, created_at
	`

	var account TraderAccount
	err := r.db.QueryRow(
		query,
		accountID, userID, accountNumber, broker, platform,
		server, investorPassword, nickname, currency,
		role, 0, 0, "active",
	).Scan(&account.ID, &account.AccountNumber, &account.Broker, &account.Platform, &account.Server, &account.AccountType, &account.CreatedAt)

	if err != nil {
		return nil, err
	}

	return &account, nil
}

// generateUUID creates a proper UUID v4
func generateUUID() string {
	return uuid.New().String()
}

// GetAccountSummary - deposit, withdrawal, net profit from DB
func (r *Repository) GetAccountSummary(accountID, userID string) (deposit, withdraw, netProfit, roi float64) {
	r.db.QueryRow(`
		SELECT
			COALESCE(SUM(CASE WHEN transaction_type='deposit' THEN amount ELSE 0 END),0),
			COALESCE(SUM(CASE WHEN transaction_type='withdrawal' THEN amount ELSE 0 END),0)
		FROM account_transactions
		WHERE account_id=$1::uuid`,
		accountID).Scan(&deposit, &withdraw)
	// net_pnl & roi dari alpha_ranks (single source of truth, zero on-the-fly)
	r.db.QueryRow(`SELECT COALESCE(net_pnl,0), COALESCE(roi,0) FROM alpha_ranks WHERE account_id=$1::uuid AND symbol='ALL'`, accountID).Scan(&netProfit, &roi)

	return
}
