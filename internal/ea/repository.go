package ea

import (
	"database/sql"
	"log"
)

type AlphaRankCalculator interface {
	CalculateForAccount(accountID string) error
}

type Repository struct {
	db              *sql.DB
	alphaRankSvc    AlphaRankCalculator
}

func NewRepository(db *sql.DB, alphaRankSvc AlphaRankCalculator) *Repository {
	return &Repository{db: db, alphaRankSvc: alphaRankSvc}
}

func (r *Repository) GetAccountIDByNumber(accountNumber, userID string) (string, error) {
	var accountID string
	query := `SELECT id FROM trader_accounts WHERE account_number = $1 AND user_id = $2`
	err := r.db.QueryRow(query, accountNumber, userID).Scan(&accountID)
	return accountID, err
}

func (r *Repository) SaveTrade(accountID string, trade *TradeData) error {
	// Skip non-trading symbols (archived, empty)
	if trade.Symbol == "" || trade.Symbol == "archived" {
		return nil
	}
	query := `
		INSERT INTO trades (
			account_id, ticket, symbol, type, lots,
			open_price, close_price, profit, swap, commission,
			open_time, close_time, status, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10,
			to_timestamp($11), to_timestamp($12), $13, NOW())
		ON CONFLICT (account_id, ticket)
		DO UPDATE SET
			close_price = EXCLUDED.close_price,
			profit = EXCLUDED.profit,
			swap = EXCLUDED.swap,
			commission = EXCLUDED.commission,
			close_time = EXCLUDED.close_time,
				status = EXCLUDED.status
			WHERE trades.status != 'closed'
	`

	var closeTime int64
	if trade.CloseTime > 0 {
		closeTime = trade.CloseTime
	}

	_, err := r.db.Exec(query,
		accountID, trade.Ticket, trade.Symbol, trade.Type, trade.Lots,
		trade.OpenPrice, trade.ClosePrice, trade.Profit, trade.Swap, trade.Commission,
		trade.OpenTime, closeTime, trade.Status,
	)

	return err
}

func (r *Repository) SyncTrade(accountID string, trade *TradeData) error {
	query := `
			INSERT INTO trades (
				account_id, ticket, symbol, type, lots,
				open_price, close_price, profit, swap, commission,
				open_time, close_time, status, created_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10,
				to_timestamp($11), to_timestamp($12), $13, NOW())
			ON CONFLICT (account_id, ticket)
			DO UPDATE SET
				close_price = EXCLUDED.close_price,
				profit = EXCLUDED.profit,
				swap = EXCLUDED.swap,
				commission = EXCLUDED.commission,
				close_time = EXCLUDED.close_time,
				status = EXCLUDED.status
	`
	var closeTime int64
	if trade.CloseTime > 0 {
		closeTime = trade.CloseTime
	}
	_, err := r.db.Exec(query,
		accountID, trade.Ticket, trade.Symbol, trade.Type, trade.Lots,
		trade.OpenPrice, trade.ClosePrice, trade.Profit, trade.Swap, trade.Commission,
		trade.OpenTime, closeTime, trade.Status,
	)
	return err
}

func (r *Repository) UpdateAccountBalance(accountID string, balance, equity float64) error {
	query := `
		UPDATE trader_accounts
		SET balance = $1, equity = $2, last_sync_at = NOW(), updated_at = NOW()
		WHERE id = $3
	`
	_, err := r.db.Exec(query, balance, equity, accountID)
	return err
}

func (r *Repository) TriggerAlphaRankCalculation(accountID string) {
	if r.alphaRankSvc == nil {
		log.Printf("[AlphaRank] WARNING: alphaRankSvc is nil for account %s", accountID)
		return
	}
	if err := r.alphaRankSvc.CalculateForAccount(accountID); err != nil {
		log.Printf("[AlphaRank] Recalculate failed for account %s: %v", accountID, err)
	} else {
		log.Printf("[AlphaRank] Recalculate success for account %s", accountID)
	}
}

func (r *Repository) SaveAccountTransactions(accountID string, initialDeposit, totalDeposits, totalWithdrawals float64) error {
	if initialDeposit > 0 {
		query := `
			INSERT INTO account_transactions (account_id, transaction_type, amount, balance_after, description, transaction_time)
			SELECT $1, 'deposit', $2, $2, 'Initial deposit from EA',
			       COALESCE((SELECT MIN(open_time) FROM trades WHERE account_id = $1), NOW())
			ON CONFLICT (account_id, transaction_type, description)
			DO UPDATE SET amount = EXCLUDED.amount, balance_after = EXCLUDED.balance_after
		`
		_, err := r.db.Exec(query, accountID, initialDeposit)
		if err != nil {
			return err
		}
	}

	if totalDeposits > 0 {
		query := `
			INSERT INTO account_transactions (account_id, transaction_type, amount, balance_after, description, transaction_time)
			VALUES ($1, 'deposit', $2, 0, 'Total deposits from EA', NOW())
			ON CONFLICT (account_id, transaction_type, description)
			DO UPDATE SET amount = EXCLUDED.amount, balance_after = EXCLUDED.balance_after
		`
		_, err := r.db.Exec(query, accountID, totalDeposits)
		if err != nil {
			return err
		}
	}

	if totalWithdrawals > 0 {
		query := `
			INSERT INTO account_transactions (account_id, transaction_type, amount, balance_after, description, transaction_time)
			VALUES ($1, 'withdrawal', $2, 0, 'Total withdrawals from EA', NOW())
			ON CONFLICT (account_id, transaction_type, description)
			DO UPDATE SET amount = EXCLUDED.amount, balance_after = EXCLUDED.balance_after
		`
		_, err := r.db.Exec(query, accountID, totalWithdrawals)
		if err != nil {
			return err
		}
	}

	return nil
}

func (r *Repository) SaveEquitySnapshot(accountID string, balance, equity float64) error {
	query := `
		INSERT INTO equity_snapshots (account_id, equity, balance, snapshot_time)
		VALUES ($1, $2, $3, NOW())
	`
	_, err := r.db.Exec(query, accountID, equity, balance)
	return err
}
