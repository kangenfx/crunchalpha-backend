package investor

import (
"log"
"strings"
"strconv"
"time"
"github.com/gin-gonic/gin"
)

type InvestorTradeData struct {
Ticket      int64   `json:"ticket"`
Symbol      string  `json:"symbol"`
Type        string  `json:"type"`
Lots        float64 `json:"lots"`
OpenPrice   float64 `json:"openPrice"`
ClosePrice  float64 `json:"closePrice"`
OpenTime    int64   `json:"openTime"`
CloseTime   int64   `json:"closeTime"`
Profit      float64 `json:"profit"`
Swap        float64 `json:"swap"`
Commission  float64 `json:"commission"`
Status      string  `json:"status"`
Comment     string  `json:"comment"`
}

// POST /api/ea/investor/sync-trades
func (h *Handler) EASyncInvestorTrades(c *gin.Context) {
investorID := getEAInvestorID(c)
if investorID == "" {
c.JSON(401, gin.H{"ok": false, "error": "unauthorized"})
return
}

// Get follower_account_id from ea key
var followerAccountID string
var mt5Account string
err := h.service.repo.DB.QueryRow(`
SELECT iek.id, iek.mt5_account 
FROM investor_ea_keys iek
WHERE iek.investor_id = $1::uuid
LIMIT 1`, investorID).Scan(&followerAccountID, &mt5Account)
if err != nil {
c.JSON(500, gin.H{"ok": false, "error": "ea key not found"})
return
}

// Get trader_accounts.id for this mt5_account
var followerAcctID string
h.service.repo.DB.QueryRow(`
SELECT id FROM trader_accounts 
WHERE account_number = $1 AND user_id = $2::uuid`,
mt5Account, investorID).Scan(&followerAcctID)

var req struct {
Trades []InvestorTradeData `json:"trades"`
}
if err := c.ShouldBindJSON(&req); err != nil {
c.JSON(400, gin.H{"ok": false, "error": "invalid json"})
return
}

upserted := 0
for _, t := range req.Trades {
// Skip non-CA trades
if !strings.HasPrefix(t.Comment, "CA-CT:") {
continue
}

// Extract provider_ticket from comment "CA-CT:123456"
provTicketStr := strings.TrimPrefix(t.Comment, "CA-CT:")
provTicket, _ := strconv.ParseInt(strings.TrimSpace(provTicketStr), 10, 64)

// Sanitize timestamps
openTime := time.Unix(t.OpenTime, 0)
if t.OpenTime < 946684800 {
openTime = time.Now()
}
var closeTime *time.Time
if t.CloseTime > 946684800 {
ct := time.Unix(t.CloseTime, 0)
closeTime = &ct
}

status := "open"
if t.Status == "closed" || t.CloseTime > 946684800 {
status = "closed"
}

_, err := h.service.repo.DB.Exec(`
INSERT INTO investor_trades
(investor_id, follower_account_id, ticket, symbol, type, lots,
 open_price, close_price, open_time, close_time,
 profit, swap, commission, status, comment, provider_ticket, updated_at)
VALUES ($1::uuid, $2::uuid, $3, $4, $5, $6,
$7, $8, $9, $10,
$11, $12, $13, $14, $15, $16, now())
ON CONFLICT (follower_account_id, ticket)
DO UPDATE SET
close_price  = EXCLUDED.close_price,
close_time   = EXCLUDED.close_time,
profit       = EXCLUDED.profit,
swap         = EXCLUDED.swap,
status       = EXCLUDED.status,
updated_at   = now()`,
investorID, followerAcctID, t.Ticket, t.Symbol, t.Type, t.Lots,
t.OpenPrice, t.ClosePrice, openTime, closeTime,
t.Profit, t.Swap, t.Commission, status, t.Comment, provTicket,
)
if err != nil {
log.Printf("[InvestorTrades] Upsert error ticket %d: %v", t.Ticket, err)
continue
}
upserted++
}

log.Printf("[InvestorTrades] Synced %d trades for investor %s", upserted, investorID)
c.JSON(200, gin.H{"ok": true, "synced": upserted})
}

// GET /api/investor/trade-history
func (h *Handler) GetInvestorTradeHistory(c *gin.Context) {
uid, ok := getUID(c)
if !ok {
c.JSON(401, gin.H{"ok": false, "error": "unauthorized"})
return
}

rows, err := h.service.repo.DB.Query(`
SELECT it.ticket, it.symbol, it.type, it.lots,
       it.open_price, COALESCE(it.close_price, 0),
       it.open_time::text, COALESCE(it.close_time::text, ''),
       COALESCE(it.profit, 0), it.status, it.comment,
       COALESCE(it.provider_ticket, 0),
       COALESCE(ta.nickname, ta.account_number, '') as trader_name
FROM investor_trades it
LEFT JOIN trader_accounts ta ON ta.ticket = it.provider_ticket
WHERE it.investor_id = $1::uuid
ORDER BY it.open_time DESC
LIMIT 200`, uid)
if err != nil {
c.JSON(500, gin.H{"ok": false, "error": err.Error()})
return
}
defer rows.Close()

type TradeRow struct {
Ticket         int64   `json:"ticket"`
Symbol         string  `json:"symbol"`
Type           string  `json:"type"`
Lots           float64 `json:"lots"`
OpenPrice      float64 `json:"openPrice"`
ClosePrice     float64 `json:"closePrice"`
OpenTime       string  `json:"openTime"`
CloseTime      string  `json:"closeTime"`
Profit         float64 `json:"profit"`
Status         string  `json:"status"`
Comment        string  `json:"comment"`
ProviderTicket int64   `json:"providerTicket"`
TraderName     string  `json:"traderName"`
}

var trades []TradeRow
for rows.Next() {
var t TradeRow
rows.Scan(&t.Ticket, &t.Symbol, &t.Type, &t.Lots,
&t.OpenPrice, &t.ClosePrice,
&t.OpenTime, &t.CloseTime,
&t.Profit, &t.Status, &t.Comment,
&t.ProviderTicket, &t.TraderName)
trades = append(trades, t)
}
if trades == nil {
trades = []TradeRow{}
}
c.JSON(200, gin.H{"ok": true, "trades": trades, "count": len(trades)})
}
