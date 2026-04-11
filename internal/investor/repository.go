package investor

import (
	"context"
	"database/sql"
	"fmt"
)

type Repository struct {
	DB *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{DB: db}
}

// GetUserInfo - get basic user info from users table
func (r *Repository) GetUserInfo(ctx context.Context, userID string) (*UserInfo, error) {
	var user UserInfo
	err := r.DB.QueryRowContext(ctx, `
		SELECT id, email, primary_role
		FROM users
		WHERE id = $1
	`, userID).Scan(&user.ID, &user.Email, &user.PrimaryRole)
	if err != nil {
		return nil, fmt.Errorf("error getting user info: %w", err)
	}
	return &user, nil
}

// GetInvestorCapital - sum equity from investor's own trader_accounts
func (r *Repository) GetInvestorCapital(ctx context.Context, userID string) (float64, error) {
	var total sql.NullFloat64
	err := r.DB.QueryRowContext(ctx, `
		SELECT COALESCE(SUM(equity), 0)
		FROM trader_accounts
		WHERE user_id = $1 AND status = 'active'
	`, userID).Scan(&total)
	if err != nil {
		return 0, fmt.Errorf("error getting investor capital: %w", err)
	}
	if total.Valid {
		return total.Float64, nil
	}
	return 0, nil
}

// GetAllocations - get all active allocations with real data from DB
func (r *Repository) GetAllocations(ctx context.Context, userID string) ([]AllocationWithTraderInfo, error) {
	query := `
		SELECT
			ua.trader_account_id,
			COALESCE(ta.account_number, '') as account_number,
			COALESCE(ta.broker, '') as broker,
			COALESCE(ta.platform::text, '') as platform,
			COALESCE(ta.equity, 0) as trader_equity,
			ua.allocation_mode,
			ua.allocation_value,
			ua.max_risk_pct,
			ua.max_positions,
			COALESCE(ar.alpha_score, 0) as alpha_score,
			COALESCE(ar.grade, 'N/A') as grade,
			COALESCE(ar.critical_count, 0) as critical_count,
		COALESCE(ar.major_count, 0) as major_count,
			COALESCE(ar.risk_level, 'MEDIUM') as risk_level,
			COALESCE(ar.layer3_multiplier, 1.0) as layer3_multiplier,
			COALESCE(ar.layer3_status, 'NEUTRAL') as layer3_status,
			COALESCE(ar.layer3_detail->>'system_mode', 'FULL_ACTIVE') as layer3_system_mode,
			COALESCE(ar.layer3_reason, '') as layer3_reason,
			ua.status
		FROM user_allocations ua
		LEFT JOIN trader_accounts ta ON ta.id = ua.trader_account_id
		LEFT JOIN alpha_ranks ar ON ar.account_id = ua.trader_account_id
			AND ar.symbol = 'ALL'
		WHERE ua.user_id = $1 AND ua.status = 'ACTIVE'
		ORDER BY ua.updated_at DESC
	`
	rows, err := r.DB.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("error getting allocations: %w", err)
	}
	defer rows.Close()

	var allocations []AllocationWithTraderInfo
	for rows.Next() {
		var alloc AllocationWithTraderInfo
		var riskScore float64
			var majorCount int
		err := rows.Scan(
			&alloc.TraderAccountID,
			&alloc.TraderAccountNumber,
			&alloc.Broker,
			&alloc.Platform,
			&alloc.TraderEquity,
			&alloc.AllocationMode,
			&alloc.AllocationValue,
			&alloc.MaxRiskPct,
			&alloc.MaxPositions,
			&alloc.AlphaScore,
			&alloc.Grade,
			&riskScore,
			&majorCount,
			&alloc.RiskLevel,
			&alloc.Layer3Multiplier,
			&alloc.Layer3Status,
			&alloc.Layer3SystemMode,
			&alloc.Layer3SoftReasons,
			&alloc.Status,
		)
		if err != nil {
			return nil, fmt.Errorf("error scanning allocation: %w", err)
		}
			// RiskLevel dari DB — MEDIUM" // default, tidak ada flags data di allocation query
		allocations = append(allocations, alloc)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating allocations: %w", err)
	}
	return allocations, nil
}

// GetPortfolioStats - get survivability and risk from followed traders' alpha_ranks
func (r *Repository) GetPortfolioStats(ctx context.Context, userID string) (survivability int, riskLevel string, err error) {
	var avgRisk sql.NullFloat64
	err = r.DB.QueryRowContext(ctx, `
		SELECT AVG(ar.risk_score)
		FROM user_allocations ua
		JOIN alpha_ranks ar ON ar.account_id = ua.trader_account_id
			AND ar.symbol = 'ALL'
		WHERE ua.user_id = $1 AND ua.status = 'ACTIVE'
	`, userID).Scan(&avgRisk)
	if err != nil {
		return 50, "MODERATE", nil
	}
	if !avgRisk.Valid || avgRisk.Float64 == 0 {
		return 50, "MODERATE", nil
	}
	score := int(avgRisk.Float64)
	return score, riskLevelFromAvgScore(avgRisk.Float64), nil
}

// GetSubscriptions - get all subscriptions (no subscription_type filter - column doesn't exist)
func (r *Repository) GetSubscriptions(ctx context.Context, userID string) ([]Subscription, error) {
	query := `
		SELECT id, user_id, trader_account_id, start_at, end_at, status, created_at
		FROM user_subscriptions
		WHERE user_id = $1 AND status = 'ACTIVE'
		ORDER BY created_at DESC
	`
	rows, err := r.DB.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("error getting subscriptions: %w", err)
	}
	defer rows.Close()

	var subs []Subscription
	for rows.Next() {
		var sub Subscription
		err := rows.Scan(
			&sub.ID, &sub.UserID, &sub.TraderAccountID,
			&sub.StartAt, &sub.EndAt, &sub.Status, &sub.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("error scanning subscription: %w", err)
		}
		sub.SubscriptionType = "TRADER"
		subs = append(subs, sub)
	}
	return subs, nil
}

// UpsertAllocation - create or update allocation
func (r *Repository) UpsertAllocation(ctx context.Context, userID, traderAccountID string, mode string, value, maxRisk float64, maxPos int) error {
	query := `
		INSERT INTO user_allocations
			(user_id, trader_account_id, allocation_mode, allocation_value, max_risk_pct, max_positions, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, 'ACTIVE', NOW(), NOW())
		ON CONFLICT (user_id, trader_account_id)
		DO UPDATE SET
			allocation_mode  = EXCLUDED.allocation_mode,
			allocation_value = EXCLUDED.allocation_value,
			max_risk_pct     = EXCLUDED.max_risk_pct,
			max_positions    = EXCLUDED.max_positions,
			status           = 'ACTIVE',
			updated_at       = NOW()
	`
	_, err := r.DB.ExecContext(ctx, query, userID, traderAccountID, mode, value, maxRisk, maxPos)
	if err != nil {
		return fmt.Errorf("error upserting allocation: %w", err)
	}
	return nil
}

// FollowTrader - create follow relationship
func (r *Repository) FollowTrader(ctx context.Context, userID, traderAccountID string) error {
	query := `
		INSERT INTO user_follows (user_id, trader_account_id, status, created_at, updated_at)
		VALUES ($1, $2, 'ACTIVE', NOW(), NOW())
		ON CONFLICT (user_id, trader_account_id) DO NOTHING
	`
	_, err := r.DB.ExecContext(ctx, query, userID, traderAccountID)
	if err != nil {
		return fmt.Errorf("error following trader: %w", err)
	}
	return nil
}

// CreateSubscription - insert without subscription_type (column doesn't exist in schema)
func (r *Repository) CreateSubscription(ctx context.Context, userID, traderAccountID string) error {
	query := `
		INSERT INTO user_subscriptions
			(user_id, trader_account_id, start_at, end_at, status, created_at, updated_at)
		VALUES ($1, $2, NOW(), NOW() + INTERVAL '30 days', 'ACTIVE', NOW(), NOW())
	`
	_, err := r.DB.ExecContext(ctx, query, userID, traderAccountID)
	if err != nil {
		return fmt.Errorf("error creating subscription: %w", err)
	}
	return nil
}

// GetTraderList - get all public trader accounts with AlphaRank for marketplace
func (r *Repository) GetTraderList(ctx context.Context) ([]TraderListItem, error) {
	query := `
		SELECT
			ta.id,
			ta.account_number,
			COALESCE(ta.broker, '') as broker,
			COALESCE(ta.platform::text, '') as platform,
			COALESCE(ta.nickname, '') as nickname,
			COALESCE(ta.equity, 0) as equity,
			COALESCE(ar.alpha_score, 0) as alpha_score,
			COALESCE(ar.grade, 'N/A') as grade,
			COALESCE(ar.critical_count, 0) as critical_count,
                        COALESCE(ar.major_count, 0) as major_count,
                        COALESCE(ar.minor_count, 0) as minor_count,
                        COALESCE(ar.risk_level, 'MEDIUM') as risk_level,
                        COALESCE(ar.max_drawdown_pct, 0) as max_drawdown_pct,
			COALESCE(ar.win_rate, 0) as win_rate,
			COALESCE(ar.net_profit, 0) as net_profit,
			COALESCE(ar.total_trades_all, ar.trade_count, 0) as total_trades,
			COALESCE(ar.profit_factor, 0) as profit_factor,
			COALESCE(bp.symbol, '') as best_pair_symbol,
			COALESCE(bp.alpha_score, 0) as best_pair_score,
			COALESCE(bp.grade, '') as best_pair_grade,
			COALESCE((SELECT SUM(amount) FROM account_transactions WHERE account_id=ta.id AND transaction_type='deposit'), 0) as total_deposit,
			COALESCE((SELECT SUM(amount) FROM account_transactions WHERE account_id=ta.id AND transaction_type='withdrawal'), 0) as total_withdraw
		FROM trader_accounts ta
		LEFT JOIN alpha_ranks ar ON ar.account_id = ta.id AND ar.symbol = 'ALL'
		LEFT JOIN LATERAL (
			SELECT symbol, alpha_score, grade FROM alpha_ranks
			WHERE account_id = ta.id AND symbol != 'ALL'
			ORDER BY alpha_score DESC LIMIT 1
		) bp ON true
		WHERE ta.status = 'active'
		ORDER BY ar.alpha_score DESC NULLS LAST
	`
	rows, err := r.DB.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("error getting trader list: %w", err)
	}
	defer rows.Close()
	var traders []TraderListItem
	for rows.Next() {
		var t TraderListItem
		var criticalCount, majorCount, minorCount int
		var riskLevelDB string
		err := rows.Scan(
			&t.ID, &t.AccountNumber, &t.Broker, &t.Platform,
			&t.Nickname, &t.Equity, &t.AlphaScore, &t.Grade,
			&criticalCount, &majorCount, &minorCount, &riskLevelDB,
			&t.MaxDrawdownPct, &t.WinRate,
			&t.NetProfit, &t.TotalTrades, &t.ProfitFactor,
			&t.BestPairSymbol, &t.BestPairScore, &t.BestPairGrade,
			&t.TotalDeposit,
			&t.TotalWithdraw,
		)
		if err != nil {
			return nil, fmt.Errorf("error scanning trader: %w", err)
		}
		if riskLevelDB != "" {
			t.RiskLevel = riskLevelDB
		} else {
			t.RiskLevel = riskLevelFromFlags(criticalCount, majorCount, minorCount, t.AlphaScore)
		}
		if t.TotalDeposit > 0 {
			t.ROI = ((t.Equity + t.TotalWithdraw - t.TotalDeposit) / t.TotalDeposit) * 100
		}
		traders = append(traders, t)
	}
		return traders, nil
}

// riskLevelFromFlags - full logic with alpha_score
func riskLevelFromFlags(critical, major, minor int, alphaScore float64) string {
	totalFlags := critical + major + minor
	if critical > 0 || alphaScore < 30 {
		return "EXTREME"
	}
	if totalFlags >= 3 || (alphaScore >= 30 && alphaScore < 50) {
		return "HIGH"
	}
	if totalFlags == 2 || (alphaScore >= 50 && alphaScore < 70) {
		return "MEDIUM"
	}
	if critical == 0 && totalFlags <= 1 && alphaScore >= 70 {
		return "LOW"
	}
	return "MEDIUM"
}

// riskLevelFromAvgScore - for portfolio avg risk score
func riskLevelFromAvgScore(score float64) string {
	switch {
	case score >= 80:
		return "LOW"
	case score >= 60:
		return "MODERATE"
	case score >= 40:
		return "HIGH"
	default:
		return "CRITICAL"
	}
}
