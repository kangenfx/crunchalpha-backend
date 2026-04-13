package investor

import (
	"database/sql"
	"fmt"
	"log"
	"math"
	"time"
)

type CopyTraderEngine struct {
	db *sql.DB
}

func NewCopyTraderEngine(db *sql.DB) *CopyTraderEngine {
	return &CopyTraderEngine{db: db}
}

type CopyEvent struct {
	ID              string  `json:"id"`
	InvestorID      string  `json:"investorId"`
	TraderAccountID string  `json:"traderAccountId"`
	Action          string  `json:"action"`
	Symbol          string  `json:"symbol"`
	Direction       int     `json:"direction"`
	ProviderTicket  int64   `json:"providerTicket"`
	CalculatedLot   float64 `json:"calculatedLot"`
	SL              float64 `json:"sl"`
	TP              float64 `json:"tp"`
	InvestorEquity  float64 `json:"investorEquity"`
	AUMUsed         float64 `json:"aumUsed"`
	AllocationPct   float64 `json:"allocationPct"`
	Status          string  `json:"status"`
	RejectionReason string  `json:"rejectionReason,omitempty"`
	CreatedAt       string  `json:"createdAt"`
}

type TraderPosition struct {
	Ticket          int64
	Symbol          string
	Type            int
	Lots            float64
	OpenPrice       float64
	SL              float64
	TP              float64
	TraderEquity    float64
	TraderAccountID string
}

func (e *CopyTraderEngine) OnTraderPositionOpen(pos TraderPosition) {
	rows, err := e.db.Query(
		`SELECT ua.user_id, ua.allocation_value, ua.max_risk_pct, ua.max_positions,
		        inv.investor_equity, inv.max_daily_loss_pct
		 FROM user_allocations ua
		 JOIN investor_settings inv ON inv.investor_id = ua.user_id
		 WHERE ua.trader_account_id = $1
		   AND ua.status = 'ACTIVE'
		   AND ua.allocation_value > 0
		   AND inv.investor_equity > 0`,
		pos.TraderAccountID)
	if err != nil {
		log.Printf("[CopyEngine] DB error: %v", err)
		return
	}
	defer rows.Close()
	for rows.Next() {
		var investorID string
		var allocationPct, maxRiskPct, investorEquity, maxDailyLossPct float64
		var maxPositions int
		if err := rows.Scan(&investorID, &allocationPct, &maxRiskPct, &maxPositions,
			&investorEquity, &maxDailyLossPct); err != nil {
			continue
		}
		e.generateCopyEvent(investorID, pos, allocationPct, maxPositions, investorEquity, maxDailyLossPct)
	}
}

func (e *CopyTraderEngine) generateCopyEvent(
	investorID string, pos TraderPosition,
	allocationPct float64, maxPositions int,
	investorEquity float64, maxDailyLossPct float64,
) {
	aum := investorEquity * allocationPct / 100.0
	calculatedLot := pos.Lots * (aum / pos.TraderEquity)

	// Layer 3: baca multiplier dari DB (zero on-the-fly)
	var layer3Multiplier float64 = 1.0
	e.db.QueryRow(`
		SELECT COALESCE(layer3_multiplier, 1.0)
		FROM alpha_ranks
		WHERE account_id = $1 AND symbol = 'ALL'
	`, pos.TraderAccountID).Scan(&layer3Multiplier)
	if layer3Multiplier < 0.30 {
		layer3Multiplier = 0.30
	}
	if layer3Multiplier > 1.00 {
		layer3Multiplier = 1.00
	}
	calculatedLot = calculatedLot * layer3Multiplier
	calculatedLot = math.Floor(calculatedLot*100) / 100
	if calculatedLot < 0.01 {
		// Reject — jangan naikan ke minimum, itu berbahaya (lot tidak proporsional)
		log.Printf("[CopyEngine] Lot too small (%.4f) for investor %s — skipping", calculatedLot, investorID)
		return
	}
	reason := e.checkRejection(investorID, calculatedLot, investorEquity, maxPositions, maxDailyLossPct)
	status := "PENDING"
	if reason != "" {
		status = "REJECTED"
	}
	_, err := e.db.Exec(
		`INSERT INTO copy_events
			(id, subscription_id, provider_account_id, follower_account_id,
			 action, symbol, type, lots, sl, tp, provider_ticket, status, error,
			 calculated_lot, investor_equity, aum_used, rejection_reason, created_at)
		 VALUES (
			uuid_generate_v4(),
			(SELECT id FROM user_allocations WHERE user_id=$1::uuid AND trader_account_id=$2 LIMIT 1),
			$2,
			COALESCE((SELECT id FROM trader_accounts WHERE user_id=$1::uuid AND status='active' LIMIT 1), $1::uuid),
			$3, $4, $5, $6, $7, $8, $9, $10, $11,
			$12, $13, $14, $15, now())`,
		investorID, pos.TraderAccountID,
		"OPEN", pos.Symbol, pos.Type, calculatedLot,
		pos.SL, pos.TP, pos.Ticket, status, nullStrEngine(reason),
		calculatedLot, investorEquity, aum, nullStrEngine(reason),
	)
	if err != nil {
		log.Printf("[CopyEngine] Insert error investor %s: %v", investorID, err)
		return
	}
	log.Printf("[CopyEngine] Event — investor:%s symbol:%s lot:%.2f aum:%.2f status:%s",
		investorID, pos.Symbol, calculatedLot, aum, status)
}

func (e *CopyTraderEngine) checkRejection(
	investorID string, lot float64,
	investorEquity float64, maxPositions int, maxDailyLossPct float64,
) string {
	if lot < 0.01 {
		return "Calculated lot below minimum (0.01)"
	}
	var openCount int
	e.db.QueryRow(
		`SELECT COUNT(*) FROM copy_events
		 WHERE follower_account_id = (
		   SELECT id FROM trader_accounts WHERE user_id=$1::uuid AND status='active' LIMIT 1
		 ) AND status='PENDING'`,
		investorID).Scan(&openCount)
	if openCount >= maxPositions {
		return fmt.Sprintf("Max open positions reached (%d)", maxPositions)
	}
	var totalAlloc float64
	e.db.QueryRow(
		`SELECT COALESCE(SUM(allocation_value), 0)
		 FROM user_allocations
		 WHERE user_id=$1::uuid AND status='ACTIVE'`,
		investorID).Scan(&totalAlloc)
	if totalAlloc > 100 {
		return fmt.Sprintf("Total allocation %.0f%% exceeds 100%%", totalAlloc)
	}
	if maxDailyLossPct > 0 && investorEquity > 0 {
		limit := investorEquity * maxDailyLossPct / 100.0
		var dailyLoss float64
		e.db.QueryRow(
			`SELECT COALESCE(SUM(ABS(executed_lots * executed_price)), 0)
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

func (e *CopyTraderEngine) GetPendingCopyEvents(investorID string) ([]CopyEvent, error) {
	rows, err := e.db.Query(
		`SELECT ce.id, ce.provider_account_id, ce.action, ce.symbol, ce.type,
		        COALESCE(ce.calculated_lot, ce.lots),
		        COALESCE(ce.sl, 0), COALESCE(ce.tp, 0),
		        ce.provider_ticket,
		        COALESCE(ce.investor_equity, 0),
		        COALESCE(ce.aum_used, 0),
		        ce.status,
		        COALESCE(ce.rejection_reason, ''),
		        ce.created_at
		 FROM copy_events ce
		 WHERE ce.follower_account_id = (
		   SELECT id FROM trader_accounts WHERE user_id=$1::uuid AND status='active' LIMIT 1
		 ) AND ce.status = 'PENDING'
		 ORDER BY ce.created_at ASC LIMIT 20`,
		investorID)
	if err != nil { return nil, err }
	defer rows.Close()
	var events []CopyEvent
	for rows.Next() {
		var ev CopyEvent
		var createdAt time.Time
		if err := rows.Scan(
			&ev.ID, &ev.TraderAccountID, &ev.Action,
			&ev.Symbol, &ev.Direction, &ev.CalculatedLot,
			&ev.SL, &ev.TP, &ev.ProviderTicket,
			&ev.InvestorEquity, &ev.AUMUsed,
			&ev.Status, &ev.RejectionReason, &createdAt,
		); err != nil { continue }
		ev.InvestorID = investorID
		ev.CreatedAt = createdAt.Format("2006-01-02 15:04:05")
		events = append(events, ev)
	}
	if events == nil { events = []CopyEvent{} }
	return events, nil
}

func (e *CopyTraderEngine) UpdateCopyEventStatus(eventID, status, rejectionReason string, followerTicket int64, executedLot, executedPrice float64) error {
	_, err := e.db.Exec(
		`UPDATE copy_events SET
			status       = $2,
			error        = CASE WHEN $3 != '' THEN $3 ELSE error END,
			processed_at = now()
		 WHERE id = $1`,
		eventID, status, rejectionReason)
	if err != nil { return fmt.Errorf("update copy_event failed: %w", err) }
	if status == "EXECUTED" || status == "REJECTED" {
		e.db.Exec(
			`INSERT INTO copy_executions
				(subscription_id, signal_id, follower_ticket, executed_lots, executed_price, success, error_message, executed_at)
			 SELECT ce.subscription_id, $1::uuid, $3, $4, $5, $6, $7, now()
			 FROM copy_events ce WHERE ce.id = $1`,
			eventID, eventID,
			followerTicket, executedLot, executedPrice,
			status == "EXECUTED",
			nullStrEngine(rejectionReason),
		)
	}
	return nil
}

func nullStrEngine(s string) interface{} {
	if s == "" { return nil }
	return s
}
