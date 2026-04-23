package ea

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"strings"
	"time"
)

type AlphaRankCalculator interface {
	CalculateForAccount(accountID string) error
}

type Repository struct {
	db              *sql.DB
	alphaRankSvc    AlphaRankCalculator
}

// epoch2000 = 2000-01-01 UTC
const epoch2000 = int64(946684800)

func sanitizeTimestamp(ts int64) int64 {
	if ts < epoch2000 {
		return time.Now().Unix()
	}
	return ts
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

	// Sanitize epoch-0 timestamps
	trade.OpenTime = sanitizeTimestamp(trade.OpenTime)
	if trade.Symbol == "" || trade.Symbol == "archived" {
		return nil
	}
	query := `
		INSERT INTO trades (
			account_id, ticket, symbol, type, lots,
			open_price, close_price, profit, swap, commission,
			open_time, close_time, status,
			sl, tp, min_equity, equity_at_open,
			created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10,
			to_timestamp($11), to_timestamp($12), $13,
			$14, $15, $16, $17,
			NOW())
		ON CONFLICT (account_id, ticket)
		DO UPDATE SET
			close_price    = EXCLUDED.close_price,
			profit         = EXCLUDED.profit,
			swap           = EXCLUDED.swap,
			commission     = EXCLUDED.commission,
			close_time     = EXCLUDED.close_time,
			status         = EXCLUDED.status,
			lots           = EXCLUDED.lots,
			sl             = EXCLUDED.sl,
			tp             = EXCLUDED.tp,
			min_equity     = CASE WHEN EXCLUDED.min_equity > 0 THEN EXCLUDED.min_equity ELSE trades.min_equity END,
			equity_at_open = CASE WHEN EXCLUDED.equity_at_open > 0 THEN EXCLUDED.equity_at_open ELSE trades.equity_at_open END,
				open_time = CASE WHEN EXCLUDED.open_time > '2000-01-01' THEN EXCLUDED.open_time ELSE trades.open_time END
	`

	var closeTime int64
	if trade.CloseTime > 0 {
		closeTime = trade.CloseTime
	}

	_, err := r.db.Exec(query,
		accountID, trade.Ticket, trade.Symbol, trade.Type, trade.Lots,
		trade.OpenPrice, trade.ClosePrice, trade.Profit, trade.Swap, trade.Commission,
		trade.OpenTime, closeTime, trade.Status,
		trade.SL, trade.TP, trade.MinEquity, trade.EquityAtOpen,
	)

	return err
}

func (r *Repository) SyncTrade(accountID string, trade *TradeData) error {
	query := `
			INSERT INTO trades (
				account_id, ticket, symbol, type, lots,
				open_price, close_price, profit, swap, commission,
				open_time, close_time, status,
				sl, tp, min_equity, equity_at_open,
				created_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10,
				to_timestamp($11), to_timestamp($12), $13,
				$14, $15, $16, $17,
				NOW())
			ON CONFLICT (account_id, ticket)
			DO UPDATE SET
				close_price    = EXCLUDED.close_price,
				profit         = EXCLUDED.profit,
				swap           = EXCLUDED.swap,
				commission     = EXCLUDED.commission,
				close_time     = EXCLUDED.close_time,
				status         = EXCLUDED.status,
				lots           = EXCLUDED.lots,
				sl             = EXCLUDED.sl,
				tp             = EXCLUDED.tp,
				min_equity     = CASE WHEN EXCLUDED.min_equity > 0 THEN EXCLUDED.min_equity ELSE trades.min_equity END,
				equity_at_open = CASE WHEN EXCLUDED.equity_at_open > 0 THEN EXCLUDED.equity_at_open ELSE trades.equity_at_open END,
				open_time = CASE WHEN EXCLUDED.open_time > '2000-01-01' THEN EXCLUDED.open_time ELSE trades.open_time END
	`
	// Sanitize epoch-0 timestamps
	trade.OpenTime = sanitizeTimestamp(trade.OpenTime)

	var closeTime int64
	if trade.CloseTime > 0 {
		closeTime = trade.CloseTime
	}
	_, err := r.db.Exec(query,
		accountID, trade.Ticket, trade.Symbol, trade.Type, trade.Lots,
		trade.OpenPrice, trade.ClosePrice, trade.Profit, trade.Swap, trade.Commission,
		trade.OpenTime, closeTime, trade.Status,
		trade.SL, trade.TP, trade.MinEquity, trade.EquityAtOpen,
	)
	return err
}

func (r *Repository) UpdateAccountBalance(accountID string, balance, equity float64) error {
	query := `
		UPDATE trader_accounts
		SET balance = $1, equity = $2, last_sync_at = NOW(), updated_at = NOW(),
		    ea_verified = true,
		    ea_first_push_at = COALESCE(ea_first_push_at, NOW())
		WHERE id = $3
	`
	_, err := r.db.Exec(query, balance, equity, accountID)
	return err
}

func (r *Repository) UpdateAccountFull(accountID string, data *AccountData) error {
	// Serialize floating_by_symbol to JSON
	floatingBySymbolJSON := []byte("{}")
	if len(data.FloatingBySymbol) > 0 {
		if b, err := json.Marshal(data.FloatingBySymbol); err == nil {
			floatingBySymbolJSON = b
		}
	}
	query := `
		UPDATE trader_accounts
		SET balance = $1, equity = $2,
		    margin = $3, free_margin = $4,
		    floating_profit = $5, open_lots = $6, open_positions = $7,
		    floating_by_symbol = $8,
		    last_sync_at = NOW(), updated_at = NOW(),
		    ea_verified = true,
		    ea_first_push_at = COALESCE(ea_first_push_at, NOW())
		WHERE id = $9
	`
	_, err := r.db.Exec(query,
		data.Balance, data.Equity,
		data.Margin, data.FreeMargin,
		data.FloatingProfit, data.OpenLots, data.OpenPositions,
		floatingBySymbolJSON,
		accountID,
	)
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


// ─── Risk Level Mapping ───────────────────────────────────────────────────────
type RiskLevelConfig struct {
	MaxRiskPerTrade float64 // % of AUM per trade
	MaxDD           float64 // % max drawdown
}

func getRiskLevelConfig(riskLevel string) RiskLevelConfig {
	switch riskLevel {
	case "conservative":
		return RiskLevelConfig{MaxRiskPerTrade: 0.5, MaxDD: 5.0}
	case "aggressive":
		return RiskLevelConfig{MaxRiskPerTrade: 3.0, MaxDD: 20.0}
	default: // balanced
		return RiskLevelConfig{MaxRiskPerTrade: 1.5, MaxDD: 10.0}
	}
}

// ─── Pip Value Estimation ─────────────────────────────────────────────────────
func getPipValue(symbol string) float64 {
	sym := strings.ToUpper(symbol)
	switch {
	case strings.Contains(sym, "XAU") || strings.Contains(sym, "GOLD"):
		return 1.0 // $1 per pip per 0.01 lot for Gold
	case strings.Contains(sym, "XAG"):
		return 0.5
	case strings.Contains(sym, "BTC"):
		return 1.0
	case strings.Contains(sym, "JPY"):
		return 0.01 // JPY pairs
	default:
		return 1.0 // Most forex pairs
	}
}

// ─── Estimated SL from trader history ────────────────────────────────────────
// est_SL = avg_loss / (avg_lots × pip_value)
func (r *Repository) estimateSL(traderAccountID, symbol string, currentLots float64) float64 {
	var avgLoss, avgLots float64
	r.db.QueryRow(`
		SELECT
			ABS(AVG(CASE WHEN profit < 0 THEN profit END)),
			AVG(CASE WHEN profit < 0 THEN lots END)
		FROM trades
		WHERE account_id=$1::uuid
		  AND symbol=$2
		  AND status='closed'
		  AND profit < 0
		  AND lots > 0
		LIMIT 50`,
		traderAccountID, symbol).Scan(&avgLoss, &avgLots)

	pipVal := getPipValue(symbol)

	if avgLoss > 0 && avgLots > 0 && pipVal > 0 {
		estSL := avgLoss / (avgLots * pipVal)
		if estSL > 0 {
			return estSL
		}
	}

	// Fallback: fixed % per symbol type
	sym := strings.ToUpper(symbol)
	switch {
	case strings.Contains(sym, "XAU") || strings.Contains(sym, "GOLD"):
		return 1500.0 // ~15 pip Gold (1500 points)
	case strings.Contains(sym, "BTC"):
		return 500.0
	case strings.Contains(sym, "JPY"):
		return 0.5
	default:
		return 0.0030 // ~30 pip Forex
	}
}

// ─── Final Lot Calculation ────────────────────────────────────────────────────
type LotCalcResult struct {
	PropLot     float64
	RiskLot     float64
	FinalLot    float64
	EstimatedSL float64
	RiskLevel   string
}

func (r *Repository) calcFinalLot(
	traderAccountID string,
	trade *TradeData,
	aum float64,
	traderEquity float64,
	riskLevel string,
	traderAvgLoss float64,
) LotCalcResult {
	cfg := getRiskLevelConfig(riskLevel)
	pipVal := getPipValue(trade.Symbol)

	// 1. Proportional lot
	propLot := trade.Lots * (aum / traderEquity)

	// 2. Estimate SL
	var estSL float64
	if trade.OpenPrice > 0 {
		// Get from trader history
		estSL = r.estimateSL(traderAccountID, trade.Symbol, trade.Lots)
	}
	if estSL <= 0 {
		estSL = r.estimateSL(traderAccountID, trade.Symbol, trade.Lots)
	}

	// 3. Risk lot from risk level
	// risk_lot = (AUM × max_risk_pct/100) / (SL × pip_value)
	var riskLot float64
	if estSL > 0 && pipVal > 0 {
		maxRiskAmt := aum * cfg.MaxRiskPerTrade / 100.0
		riskLot = maxRiskAmt / (estSL * pipVal)
	} else {
		// No SL estimate available — use conservative cap
		riskLot = aum * cfg.MaxRiskPerTrade / 100.0 / 10.0
	}

	// 4. Final lot = MIN(prop_lot, risk_lot)
	finalLot := math.Min(propLot, riskLot)

	// 5. Round down to 2 decimal places
	finalLot = math.Floor(finalLot*100) / 100
	propLot = math.Floor(propLot*100) / 100
	riskLot = math.Floor(riskLot*100) / 100

	// 6. Minimum lot
	if finalLot < 0.01 {
		finalLot = 0.01
	}

	return LotCalcResult{
		PropLot:     propLot,
		RiskLot:     riskLot,
		FinalLot:    finalLot,
		EstimatedSL: estSL,
		RiskLevel:   riskLevel,
	}
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

	// Get avg loss dari trader history untuk estimasi SL
	var traderAvgLoss float64
	r.db.QueryRow(`
		SELECT COALESCE(ABS(AVG(CASE WHEN profit < 0 THEN profit END)), 0)
		FROM trades WHERE account_id=$1::uuid AND status='closed' AND profit < 0`,
		traderAccountID).Scan(&traderAvgLoss)

	// Get all active investors following this trader
	rows, err := r.db.Query(`
		SELECT
			ta_inv.user_id::text,
			ua.allocation_value,
			ua.max_positions,
			inv.investor_equity,
			inv.max_daily_loss_pct,
			COALESCE(inv.risk_level, 'balanced'),
			cs.follower_account_id::text,
			COALESCE(iek.equity, inv.investor_equity) as acct_equity
		FROM copy_subscriptions cs
		JOIN trader_accounts ta_inv ON ta_inv.id = cs.follower_account_id
		JOIN user_allocations ua ON ua.user_id = ta_inv.user_id
			AND ua.trader_account_id = cs.provider_account_id
			AND ua.follower_account_id = cs.follower_account_id
			AND ua.status = 'ACTIVE'
		JOIN investor_settings inv ON inv.investor_id = ta_inv.user_id
		LEFT JOIN investor_ea_keys iek ON iek.investor_id = ta_inv.user_id
			AND iek.mt5_account = ta_inv.account_number
		WHERE cs.provider_account_id = $1::uuid
		  AND cs.status = 'ACTIVE'
		  AND inv.copy_trader_enabled = true
		  AND ua.allocation_value > 0`,
		traderAccountID)
	if err != nil {
		log.Printf("[CopyEngine] DB error querying followers: %v", err)
		return
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		var investorID, riskLevel, followerAccountID string
		var allocationPct, investorEquity, maxDailyLossPct, acctEquity float64
		var maxPositions int
		if err := rows.Scan(&investorID, &allocationPct, &maxPositions,
			&investorEquity, &maxDailyLossPct, &riskLevel, &followerAccountID, &acctEquity); err != nil {
			continue
		}

		// AUM calculation — per follower account equity
		aum := acctEquity * allocationPct / 100.0

		// Risk-normalized lot calculation
		lotResult := r.calcFinalLot(traderAccountID, trade, aum, traderEquity, riskLevel, traderAvgLoss)
		calculatedLot := lotResult.FinalLot

		// Rejection checks
		reason := r.checkCopyRejection(investorID, followerAccountID, calculatedLot, acctEquity, maxPositions, maxDailyLossPct)
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
				 prop_lot, risk_lot, estimated_sl, final_lot,
				 created_at)
			VALUES (
				uuid_generate_v4(),
				(SELECT cs.id FROM copy_subscriptions cs
				 JOIN trader_accounts ta ON ta.id = cs.follower_account_id
				 WHERE ta.user_id=$1::uuid AND cs.provider_account_id=$2::uuid LIMIT 1),
				$2::uuid,
				$17::uuid,
				'OPEN', $3, $4, $5,
				$6, $7, $8, $9, $10,
				$5, $11, $12, $10,
				$13, $14, $15, $16,
				now()
			)`,
			investorID, traderAccountID,
			trade.Symbol, direction, calculatedLot,
			0.0, 0.0, trade.Ticket, status, nullStr(reason),
			acctEquity, aum,
			lotResult.PropLot, lotResult.RiskLot, lotResult.EstimatedSL, lotResult.FinalLot,
			followerAccountID,
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
func (r *Repository) checkCopyRejection(investorID, followerAccountID string, lot, investorEquity float64, maxPositions int, maxDailyLossPct float64) string {
	if lot < 0.01 {
		return "Calculated lot below minimum (0.01)"
	}

	// Max open positions check
	var openCount int
	r.db.QueryRow(`
		SELECT COUNT(*) FROM copy_events
		WHERE follower_account_id = $2::uuid
		AND status='PENDING' AND action='OPEN'`,
		investorID, followerAccountID).Scan(&openCount)
	if maxPositions > 0 && openCount >= maxPositions {
		return fmt.Sprintf("Max open positions reached (%d)", maxPositions)
	}

	// Total allocation check — per follower account
	var totalAlloc float64
	r.db.QueryRow(`
		SELECT COALESCE(SUM(allocation_value), 0)
		FROM user_allocations
		WHERE user_id=$1::uuid AND status='ACTIVE'
		AND follower_account_id = $2::uuid`,
		investorID, followerAccountID).Scan(&totalAlloc)
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

// VerifyEAAccountNumber — cross-check account number yang di-push EA vs yang didaftarkan
func (r *Repository) VerifyEAAccountNumber(accountID, pushedAccountNumber string) error {
var registeredNumber string
err := r.db.QueryRow(`SELECT account_number FROM trader_accounts WHERE id=$1`, accountID).Scan(&registeredNumber)
if err != nil { return err }
if registeredNumber != pushedAccountNumber {
// Mismatch — suspend account
r.db.Exec(`UPDATE trader_accounts SET status='suspended', ea_verified=false WHERE id=$1`, accountID)
return fmt.Errorf("account number mismatch: registered=%s pushed=%s", registeredNumber, pushedAccountNumber)
}
r.db.Exec(`UPDATE trader_accounts SET ea_account_number_confirmed=$1 WHERE id=$2`, pushedAccountNumber, accountID)
return nil
}
