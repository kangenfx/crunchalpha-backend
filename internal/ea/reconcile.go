package ea

import (
"log"
"math"
"time"
)

// StartReconciliationCron — jalankan setiap 2 menit
// Detect open trades master yang belum punya copy_event PENDING/EXECUTED di follower
func (r *Repository) StartReconciliationCron() {
go func() {
for {
time.Sleep(2 * time.Minute)
r.runReconciliation()
}
}()
log.Printf("[Reconcile] Cron started — interval 2 minutes")
}

func (r *Repository) runReconciliation() {
// Ambil semua open trades dari master yang punya subscriber ACTIVE
rows, err := r.db.Query(`
SELECT DISTINCT
t.ticket, t.symbol, t.type, t.lots, t.sl, t.tp,
t.account_id::text,
cs.id::text as subscription_id,
cs.follower_account_id::text,
cs.provider_account_id::text,
ta_inv.user_id::text as investor_id,
COALESCE(ua.allocation_value, 0) as allocation_value,
COALESCE(iek.equity, inv.investor_equity, 0) as acct_equity,
COALESCE(inv.risk_level, 'balanced') as risk_level,
COALESCE(inv.max_daily_loss_pct, 20) as max_daily_loss_pct,
COALESCE(ua.max_positions, 10) as max_positions
FROM trades t
JOIN copy_subscriptions cs ON cs.provider_account_id = t.account_id
AND cs.status = 'ACTIVE'
JOIN trader_accounts ta_inv ON ta_inv.id = cs.follower_account_id
JOIN investor_settings inv ON inv.investor_id = ta_inv.user_id
AND inv.copy_trader_enabled = true
JOIN user_allocations ua ON ua.user_id = ta_inv.user_id
AND ua.trader_account_id = t.account_id
AND ua.follower_account_id = cs.follower_account_id
AND ua.status = 'ACTIVE'
AND ua.allocation_value > 0
LEFT JOIN investor_ea_keys iek ON iek.investor_id = ta_inv.user_id
AND iek.mt5_account = ta_inv.account_number
WHERE t.status = 'open'
  AND NOT EXISTS (
SELECT 1 FROM copy_events ce
WHERE ce.subscription_id = cs.id
  AND ce.provider_ticket = t.ticket
  AND ce.action = 'OPEN'
  AND ce.status IN ('PENDING', 'EXECUTED', 'REJECTED')
  )`)
if err != nil {
log.Printf("[Reconcile] Query error: %v", err)
return
}
defer rows.Close()

count := 0
for rows.Next() {
var ticket int64
var symbol, tradeType, accountID, subscriptionID string
var followerAccountID, providerAccountID, investorID, riskLevel string
var lots, sl, tp, allocationPct, acctEquity, maxDailyLossPct float64
var maxPositions int

if err := rows.Scan(
&ticket, &symbol, &tradeType, &lots, &sl, &tp,
&accountID, &subscriptionID,
&followerAccountID, &providerAccountID, &investorID,
&allocationPct, &acctEquity, &riskLevel,
&maxDailyLossPct, &maxPositions,
); err != nil {
log.Printf("[Reconcile] Scan error: %v", err)
continue
}

// Direction
direction := 0
if tradeType == "sell" || tradeType == "1" {
direction = 1
}

// AUM
aum := acctEquity * allocationPct / 100.0

// Trader equity
var traderEquity float64
r.db.QueryRow(`SELECT COALESCE(equity, balance, 0) FROM trader_accounts WHERE id = $1::uuid`, providerAccountID).Scan(&traderEquity)
if traderEquity <= 0 {
traderEquity = 1
}

// Calc lot
propLot := math.Floor(lots*(aum/traderEquity)*100) / 100
if propLot < 0.01 {
log.Printf("[Reconcile] SKIP ticket=%d propLot=%.4f < 0.01", ticket, propLot)
continue
}

// Rejection check
reason := r.checkCopyRejection(investorID, followerAccountID, propLot, acctEquity, maxPositions, maxDailyLossPct)
if reason != "" {
log.Printf("[Reconcile] SKIP ticket=%d reason=%s", ticket, reason)
continue
}

// Insert copy_event dengan ON CONFLICT DO NOTHING (unique: subscription_id, provider_ticket, action)
_, err := r.db.Exec(`
INSERT INTO copy_events
(id, subscription_id, provider_account_id, follower_account_id,
 action, symbol, type, lots,
 sl, tp, provider_ticket, status, created_at,
 calculated_lot, prop_lot, risk_lot, final_lot,
 investor_equity, aum_used)
VALUES (
uuid_generate_v4(), $1::uuid, $2::uuid, $3::uuid,
'OPEN', $4, $5, $6,
$7, $8, $9, 'PENDING', now(),
$6, $6, $6, $6,
$10, $11
)
ON CONFLICT (subscription_id, provider_ticket, action) DO NOTHING`,
subscriptionID, providerAccountID, followerAccountID,
symbol, direction, propLot,
sl, tp, ticket,
acctEquity, aum,
)
if err != nil {
log.Printf("[Reconcile] Insert error ticket=%d: %v", ticket, err)
continue
}
count++
log.Printf("[Reconcile] Created copy_event ticket=%d symbol=%s lot=%.2f follower=%s",
ticket, symbol, propLot, followerAccountID)
}

if count > 0 {
log.Printf("[Reconcile] Done — %d missed trades queued", count)
}
}
