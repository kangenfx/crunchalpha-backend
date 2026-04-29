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
COALESCE(iek.equity, 0),
inv.max_daily_loss_pct, COALESCE(inv.risk_level, 'balanced'),
ua.follower_account_id
 FROM user_allocations ua
 JOIN investor_settings inv ON inv.investor_id = ua.user_id
 JOIN investor_ea_keys iek ON iek.investor_id = ua.user_id
   AND iek.mt5_account = (
SELECT account_number FROM trader_accounts WHERE id = ua.follower_account_id
   )
 WHERE ua.trader_account_id = $1
   AND ua.status = 'ACTIVE'
   AND ua.allocation_value > 0
   AND ua.follower_account_id IS NOT NULL`,
pos.TraderAccountID)
if err != nil {
log.Printf("[CopyEngine] DB error: %v", err)
return
}
defer rows.Close()
for rows.Next() {
var investorID, riskLevel string
var followerAccountID string
var allocationPct, maxRiskPct, investorEquity, maxDailyLossPct float64
var maxPositions int
if err := rows.Scan(&investorID, &allocationPct, &maxRiskPct, &maxPositions,
&investorEquity, &maxDailyLossPct, &riskLevel, &followerAccountID); err != nil {
log.Printf("[CopyEngine] Scan error: %v", err)
continue
}
log.Printf("[CopyEngine] Loop — investorID=%s followerAccountID=%s", investorID, followerAccountID)
e.generateCopyEvent(investorID, followerAccountID, pos, allocationPct, maxPositions, investorEquity, maxDailyLossPct, riskLevel)
}
}

func riskLevelMaxRiskPct(level string) float64 {
switch level {
case "conservative":
return 0.5
case "aggressive":
return 3.0
default:
return 1.5
}
}

func riskLevelMaxDD(level string) float64 {
switch level {
case "conservative":
return 5.0
case "aggressive":
return 20.0
default:
return 10.0
}
}

func (e *CopyTraderEngine) generateCopyEvent(
investorID string, followerAccountID string, pos TraderPosition,
allocationPct float64, maxPositions int,
investorEquity float64, maxDailyLossPct float64, riskLevel string,
) {
aum := investorEquity * allocationPct / 100.0

// DD guard
var floatingProfit float64
e.db.QueryRow(`SELECT COALESCE(SUM(floating), 0) FROM investor_ea_keys WHERE investor_id=$1::uuid`, investorID).Scan(&floatingProfit)
if investorEquity > 0 && floatingProfit < 0 {
currentDD := math.Abs(floatingProfit) / investorEquity * 100.0
maxDD := riskLevelMaxDD(riskLevel)
if currentDD >= maxDD {
log.Printf("[CopyEngine] DD guard: investor %s DD=%.2f%% >= maxDD=%.2f%% (%s) — skip", investorID, currentDD, maxDD, riskLevel)
return
}
}

calculatedLot := pos.Lots * (aum / pos.TraderEquity)

// Risk cap per trade
maxRiskPct := riskLevelMaxRiskPct(riskLevel)
maxLotFromRisk := aum * maxRiskPct / 100.0
if maxLotFromRisk >= 0.01 && calculatedLot > maxLotFromRisk {
log.Printf("[CopyEngine] Risk cap: lot %.4f -> %.4f (%s %.1f%%)", calculatedLot, maxLotFromRisk, riskLevel, maxRiskPct)
calculatedLot = maxLotFromRisk
}

// Layer 3 multiplier dari DB
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
log.Printf("[CopyEngine] Lot too small (%.4f) for investor %s — skipping", calculatedLot, investorID)
return
}

reason := e.checkRejection(followerAccountID, investorID, calculatedLot, investorEquity, maxPositions, maxDailyLossPct)
status := "PENDING"
if reason != "" {
status = "REJECTED"
}

// DEBUG
var debugSubID string
e.db.QueryRow(`SELECT id FROM copy_subscriptions WHERE provider_account_id=$1 AND follower_account_id=$2::uuid LIMIT 1`, pos.TraderAccountID, followerAccountID).Scan(&debugSubID)
log.Printf("[CopyEngine] DEBUG followerAccountID=%s subID=%s", followerAccountID, debugSubID)
_, err := e.db.Exec(
`INSERT INTO copy_events
(id, subscription_id, provider_account_id, follower_account_id,
 action, symbol, type, lots, sl, tp, provider_ticket, status, error,
 calculated_lot, investor_equity, aum_used, rejection_reason, created_at)
 VALUES (
uuid_generate_v4(),
(SELECT id FROM copy_subscriptions WHERE provider_account_id=$2 AND follower_account_id=$3::uuid LIMIT 1),
$2, $3::uuid,
$4, $5, $6, $7, $8, $9, $10, $11, $12,
$13, $14, $15, $16, now())`,
investorID, pos.TraderAccountID, followerAccountID,
"OPEN", pos.Symbol, pos.Type, calculatedLot,
pos.SL, pos.TP, pos.Ticket, status, nullStrEngine(reason),
calculatedLot, investorEquity, aum, nullStrEngine(reason),
)
if err != nil {
log.Printf("[CopyEngine] Insert error investor %s: %v", investorID, err)
return
}
log.Printf("[CopyEngine] Event created — investor:%s follower:%s symbol:%s lot:%.2f aum:%.2f status:%s reason:%s",
investorID, followerAccountID, pos.Symbol, calculatedLot, aum, status, reason)
}

func (e *CopyTraderEngine) checkRejection(
followerAccountID string, investorID string, lot float64,
investorEquity float64, maxPositions int, maxDailyLossPct float64,
) string {
if lot < 0.01 {
return "Calculated lot below minimum (0.01)"
}
var openCount int
e.db.QueryRow(
`SELECT COUNT(*) FROM copy_events
 WHERE follower_account_id = $1::uuid
   AND action = 'OPEN'
   AND status = 'EXECUTED'
   AND provider_ticket NOT IN (
     SELECT provider_ticket FROM copy_events
     WHERE follower_account_id = $1::uuid
       AND action = 'CLOSE'
       AND status = 'EXECUTED'
   )`,
followerAccountID).Scan(&openCount)
if openCount >= maxPositions {
return fmt.Sprintf("Max open positions reached (%d)", maxPositions)
}
var totalAlloc float64
e.db.QueryRow(
`SELECT COALESCE(SUM(allocation_value), 0)
 FROM user_allocations
 WHERE user_id=$1::uuid AND follower_account_id=$2::uuid AND status='ACTIVE'`,
investorID, followerAccountID).Scan(&totalAlloc)
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
 WHERE ev.follower_account_id = $1::uuid
   AND DATE(ce.executed_at) = CURRENT_DATE
   AND ce.success = false`,
followerAccountID).Scan(&dailyLoss)
if dailyLoss >= limit {
return fmt.Sprintf("Max daily loss %.1f%% reached", maxDailyLossPct)
}
}
return ""
}

func (e *CopyTraderEngine) GetPendingCopyEvents(investorID string, mt5Account string) ([]CopyEvent, error) {
// Auto-expire PENDING events older than 3 minutes
e.db.Exec(`UPDATE copy_events SET status='REJECTED', error='auto-expired: not executed within 3 minutes'
 WHERE status='PENDING' AND created_at < now() - interval '3 minutes'`)
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
 JOIN trader_accounts ta ON ta.id = ce.follower_account_id
 WHERE ta.user_id = $1::uuid
   AND ta.account_number = $2
   AND ce.status = 'PENDING'
 ORDER BY ce.created_at ASC LIMIT 20`,
investorID, mt5Account)
if err != nil {
return nil, err
}
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
); err != nil {
continue
}
ev.InvestorID = investorID
ev.CreatedAt = createdAt.Format("2006-01-02 15:04:05")
events = append(events, ev)
}
if events == nil {
events = []CopyEvent{}
}
return events, nil
}

func (e *CopyTraderEngine) UpdateCopyEventStatus(eventID, status, rejectionReason string, followerTicket int64, executedLot, executedPrice, profit float64) error {
	log.Printf("[CopyEngine] UpdateStatus eventID=%s status=%s profit=%f", eventID, status, profit)
	_, err := e.db.Exec(
	`UPDATE copy_events SET
	status       = $2,
	error        = CASE WHEN $3 != '' THEN $3 ELSE error END,
	processed_at = now()
	 WHERE id = $1`,
	eventID, status, rejectionReason)
	if err != nil {
		return fmt.Errorf("update copy_event failed: %w", err)
	}
	if status == "EXECUTED" || status == "REJECTED" {
		_, execErr := e.db.Exec(
		`INSERT INTO copy_executions
		(subscription_id, signal_id, follower_ticket, executed_lots, executed_price, success, error_message, action, close_price, profit, executed_at)
		 SELECT ce.subscription_id, $1::uuid, $2::bigint, $3::numeric,
		        CASE WHEN ce.action='OPEN' THEN $4::numeric ELSE 0 END,
		        $5, $6, ce.action,
		        CASE WHEN ce.action='CLOSE' THEN $4::numeric ELSE NULL END,
		        CASE WHEN ce.action='CLOSE' THEN $7::numeric ELSE NULL END,
		        now()
		 FROM copy_events ce WHERE ce.id = $1`,
		eventID,
		followerTicket, executedLot, executedPrice,
		status == "EXECUTED",
		nullStrEngine(rejectionReason),
		profit,
		)
		if execErr != nil { log.Printf("[CopyEngine] copy_executions error: %v eventID=%s", execErr, eventID) } else { log.Printf("[CopyEngine] copy_executions OK eventID=%s profit=%f", eventID, profit) }
	}
	return nil
}

func nullStrEngine(s string) interface{} {
if s == "" {
return nil
}
return s
}
