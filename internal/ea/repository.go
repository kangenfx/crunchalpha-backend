package ea

import (
	"database/sql"
	"fmt"
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
				status = EXCLUDED.status,
				lots = EXCLUDED.lots
			WHERE trades.status != 'closed' OR EXCLUDED.status = 'open'
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
				status = EXCLUDED.status,
				lots = EXCLUDED.lots
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

// TriggerCopyEngine — dipanggil saat trader buka posisi baru
// Generate copy_events untuk semua investor yang follow trader ini
func (r *Repository) TriggerCopyEngine(traderAccountID string, trade *TradeData) {
	// Get trader equity dari DB
	var traderEquity float64
	err := r.db.QueryRow(`
		SELECT COALESCE(equity, 0) FROM trader_accounts WHERE id=$1::uuid`,
		traderAccountID).Scan(&traderEquity)
	if err != nil || traderEquity <= 0 {
		log.Printf("[CopyEngine] Skip — trader equity 0 or not found for %s", traderAccountID)
		return
	}

	// Direction: 0=BUY, 1=SELL
	direction := 0
	if trade.Type == "sell" || trade.Type == "SELL" {
		direction = 1
	}

	// Get all active investors following this trader via copy_subscriptions
	// follower_account_id di copy_subscriptions adalah trader_account milik investor
	// kita perlu join ke trader_accounts untuk dapat user_id investor
	rows, err := r.db.Query(`
		SELECT
			ta_inv.user_id::text,
			ua.allocation_value,
			ua.max_risk_pct,
			ua.max_positions,
			inv.investor_equity,
			inv.max_daily_loss_pct
		FROM copy_subscriptions cs
		JOIN trader_accounts ta_inv ON ta_inv.id = cs.follower_account_id
		JOIN user_allocations ua ON ua.user_id = ta_inv.user_id
			AND ua.trader_account_id = cs.provider_account_id
			AND ua.status = 'ACTIVE'
		JOIN investor_settings inv ON inv.investor_id = ta_inv.user_id
		WHERE cs.provider_account_id = $1::uuid
		  AND cs.status = 'ACTIVE'
		  AND inv.copy_trader_enabled = true
		  AND inv.investor_equity > 0
		  AND ua.allocation_value > 0`,
		traderAccountID)
	if err != nil {
		log.Printf("[CopyEngine] DB error querying followers: %v", err)
		return
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		var investorID string
		var allocationPct, maxRiskPct, investorEquity, maxDailyLossPct float64
		var maxPositions int
		if err := rows.Scan(&investorID, &allocationPct, &maxRiskPct, &maxPositions,
			&investorEquity, &maxDailyLossPct); err != nil {
			continue
		}

		// AUM calculation
		aum := investorEquity * allocationPct / 100.0

		// Proportional lot: Lot = trader_lot × (AUM / trader_equity)
		calculatedLot := trade.Lots * (aum / traderEquity)
		// Round down to 2 decimal places
		calculatedLot = float64(int(calculatedLot*100)) / 100
		if calculatedLot < 0.01 {
			calculatedLot = 0.01
		}

		// Rejection checks
		reason := r.checkCopyRejection(investorID, calculatedLot, investorEquity, maxPositions, maxDailyLossPct)
		status := "PENDING"
		if reason != "" {
			status = "REJECTED"
		}

		// Insert copy_event — simpan ke DB (single source of truth)
		_, err := r.db.Exec(`
			INSERT INTO copy_events
				(id, subscription_id, provider_account_id, follower_account_id,
				 action, symbol, type, lots,
				 sl, tp, provider_ticket, status, error,
				 calculated_lot, investor_equity, aum_used, rejection_reason,
				 created_at)
			VALUES (
				uuid_generate_v4(),
				(SELECT cs.id FROM copy_subscriptions cs
				 JOIN trader_accounts ta ON ta.id = cs.follower_account_id
				 WHERE ta.user_id=$1::uuid AND cs.provider_account_id=$2::uuid LIMIT 1),
				$2::uuid,
				COALESCE((SELECT id FROM trader_accounts WHERE user_id=$1::uuid AND status='active' LIMIT 1), $1::uuid),
				'OPEN', $3, $4, $5,
				$6, $7, $8, $9, $10,
				$5, $11, $12, $10,
				now()
			)`,
			investorID, traderAccountID,
			trade.Symbol, direction, calculatedLot,
			trade.OpenPrice, 0.0, trade.Ticket, status, nullStr(reason),
			investorEquity, aum,
		)
		if err != nil {
			log.Printf("[CopyEngine] Insert error investor %s: %v", investorID, err)
			continue
		}
		count++
		log.Printf("[CopyEngine] Event created — investor:%s symbol:%s lot:%.2f aum:%.2f status:%s reason:%s",
			investorID, trade.Symbol, calculatedLot, aum, status, reason)
	}
	log.Printf("[CopyEngine] Done — %d events created for trader %s", count, traderAccountID)
}

// TriggerCopyEngineClose — dipanggil saat trader close posisi
func (r *Repository) TriggerCopyEngineClose(traderAccountID string, providerTicket int64) {
	// Update copy_events yang PENDING → CLOSE_PENDING untuk posisi ini
	_, err := r.db.Exec(`
		INSERT INTO copy_events
			(id, subscription_id, provider_account_id, follower_account_id,
			 action, symbol, type, lots, provider_ticket, status, created_at)
		SELECT
			uuid_generate_v4(),
			ce.subscription_id,
			ce.provider_account_id,
			ce.follower_account_id,
			'CLOSE', ce.symbol, ce.type, ce.calculated_lot,
			$2, 'PENDING', now()
		FROM copy_events ce
		WHERE ce.provider_account_id = $1::uuid
		  AND ce.provider_ticket = $2
		  AND ce.action = 'OPEN'
		  AND ce.status = 'EXECUTED'`,
		traderAccountID, providerTicket)
	if err != nil {
		log.Printf("[CopyEngine] Close event error: %v", err)
	}
}

// checkCopyRejection — cek apakah copy event perlu di-reject
func (r *Repository) checkCopyRejection(investorID string, lot, investorEquity float64, maxPositions int, maxDailyLossPct float64) string {
	if lot < 0.01 {
		return "Calculated lot below minimum (0.01)"
	}

	// Max open positions check
	var openCount int
	r.db.QueryRow(`
		SELECT COUNT(*) FROM copy_events
		WHERE follower_account_id = (
			SELECT id FROM trader_accounts WHERE user_id=$1::uuid AND status='active' LIMIT 1
		) AND status='PENDING' AND action='OPEN'`,
		investorID).Scan(&openCount)
	if openCount >= maxPositions {
		return fmt.Sprintf("Max open positions reached (%d)", maxPositions)
	}

	// Total allocation check
	var totalAlloc float64
	r.db.QueryRow(`
		SELECT COALESCE(SUM(allocation_value), 0)
		FROM user_allocations
		WHERE user_id=$1::uuid AND status='ACTIVE'`,
		investorID).Scan(&totalAlloc)
	if totalAlloc > 100 {
		return fmt.Sprintf("Total allocation %.0f%% exceeds 100%%", totalAlloc)
	}

	// Max daily loss check
	if maxDailyLossPct > 0 && investorEquity > 0 {
		limit := investorEquity * maxDailyLossPct / 100.0
		var dailyLoss float64
		r.db.QueryRow(`
			SELECT COALESCE(SUM(ABS(executed_lots * executed_price)), 0)
			FROM copy_executions ce
			JOIN copy_events ev ON ev.id::text = ce.signal_id::text
			WHERE ev.follower_account_id = (
				SELECT id FROM trader_accounts WHERE user_id=$1::uuid AND status='active' LIMIT 1
			) AND DATE(ce.executed_at) = CURRENT_DATE
			  AND ce.success = false`,
			investorID).Scan(&dailyLoss)
		if dailyLoss >= limit {
			return fmt.Sprintf("Max daily loss %.1f%% reached", maxDailyLossPct)
		}
	}
	return ""
}

func nullStr(s string) interface{} {
	if s == "" { return nil }
	return s
}
