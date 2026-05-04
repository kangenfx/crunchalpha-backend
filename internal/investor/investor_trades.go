package investor

import (
"log"
"strings"
"strconv"
"time"
"database/sql"
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

	// Get follower_account_id from ea key used in this request
	keyID, _ := c.Get("ea_key_id")
	keyIDStr := ""
	if keyID != nil { keyIDStr = keyID.(string) }
	var mt5Account string
	if keyIDStr != "" {
		h.service.repo.DB.QueryRow(`SELECT mt5_account FROM investor_ea_keys WHERE id=$1`, keyIDStr).Scan(&mt5Account)
	} else {
		h.service.repo.DB.QueryRow(`SELECT mt5_account FROM investor_ea_keys WHERE investor_id=$1::uuid ORDER BY last_used DESC LIMIT 1`, investorID).Scan(&mt5Account)
	}
	var followerAcctID string
	h.service.repo.DB.QueryRow(`SELECT id::text FROM trader_accounts WHERE account_number=$1 AND user_id=$2::uuid AND status='active' LIMIT 1`, mt5Account, investorID).Scan(&followerAcctID)
	if followerAcctID == "" {
		log.Printf("[InvestorTrades] follower_account_id not found for mt5_account=%s investor=%s", mt5Account, investorID)
		c.JSON(400, gin.H{"ok": false, "error": "account not found"})
		return
	}

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

		// Extract provider_ticket from comment "CA-CT:123456" or "CA-CT:123456[sl]"
		provTicketStr := strings.TrimPrefix(t.Comment, "CA-CT:")
		provTicketStr = strings.Split(provTicketStr, "[")[0]
		provTicket, _ := strconv.ParseInt(strings.TrimSpace(provTicketStr), 10, 64)
		log.Printf("[InvestorTrades] deal ticket=%d openTime=%d closeTime=%d openPrice=%f closePrice=%f profit=%f comment=%s", t.Ticket, t.OpenTime, t.CloseTime, t.OpenPrice, t.ClosePrice, t.Profit, t.Comment)

		if t.CloseTime > 946684800 && t.OpenTime == 0 {
			// MT5 CLOSE deal — update existing row by provider_ticket
			closeTime := time.Unix(t.CloseTime, 0)
			log.Printf("[InvestorTrades] CLOSE deal ticket=%d provTicket=%d closePrice=%f follower=%s", t.Ticket, provTicket, t.ClosePrice, followerAcctID)
			_, err := h.service.repo.DB.Exec(`
UPDATE investor_trades SET
  close_price = $1,
  close_time  = $2,
  profit      = $3,
  swap        = swap + $4,
  status      = 'closed',
  updated_at  = now()
WHERE follower_account_id = $5::uuid
  AND provider_ticket = $6
  AND status = 'open'`,
				t.ClosePrice, closeTime, t.Profit, t.Swap,
				followerAcctID, provTicket)
			if err != nil {
				log.Printf("[InvestorTrades] Close update error ticket %d: %v", t.Ticket, err)
			} else {
				upserted++
			}
			// Sync profit ke copy_executions
			_, ceErr := h.service.repo.DB.Exec(`
UPDATE copy_executions SET
  close_price = $1,
  profit      = $2
WHERE follower_ticket = $3
  AND action = 'CLOSE'
  AND (profit IS NULL OR profit = 0)`,
				t.ClosePrice, t.Profit, t.Ticket)
			if ceErr != nil {
				log.Printf("[InvestorTrades] copy_executions sync error ticket %d: %v", t.Ticket, ceErr)
			} else {
				log.Printf("[InvestorTrades] copy_executions synced ticket=%d profit=%f", t.Ticket, t.Profit)
			}
			continue
		}

		// OPEN deal or MT4 full trade — INSERT/UPSERT
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
  close_price = EXCLUDED.close_price,
  close_time  = EXCLUDED.close_time,
  profit      = EXCLUDED.profit,
  swap        = EXCLUDED.swap,
  status      = EXCLUDED.status,
  updated_at  = now()`,
			investorID, followerAcctID, t.Ticket, t.Symbol, t.Type, t.Lots,
			t.OpenPrice, t.ClosePrice, openTime, closeTime,
			t.Profit, t.Swap, t.Commission, status, t.Comment, provTicket,
		)
		if err != nil {
			log.Printf("[InvestorTrades] Upsert error ticket %d: %v", t.Ticket, err)
			continue
		}
		upserted++
		// Auto-close copy_events jika trade sudah closed
		if status == "closed" && provTicket > 0 {
			h.service.repo.DB.Exec(`
				INSERT INTO copy_events
				(id, subscription_id, provider_account_id, follower_account_id,
				 action, symbol, type, lots, provider_ticket, status, created_at)
				SELECT uuid_generate_v4(), ce.subscription_id, ce.provider_account_id,
					ce.follower_account_id, 'CLOSE', ce.symbol, ce.type,
					ce.calculated_lot, ce.provider_ticket, 'EXECUTED', now()
				FROM copy_events ce
				WHERE ce.follower_account_id = $1::uuid
				  AND ce.provider_ticket = $2
				  AND ce.action = 'OPEN' AND ce.status = 'EXECUTED'
				  AND NOT EXISTS (
					SELECT 1 FROM copy_events x
					WHERE x.follower_account_id = $1::uuid
					  AND x.provider_ticket = $2
					  AND x.action = 'CLOSE'
				)`,
				followerAcctID, provTicket)
		}
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
	accountID := c.Query("account_id")

	query := `
SELECT
	ce.provider_ticket as ticket,
	ce.symbol,
	CASE WHEN ce.type=0 THEN 'buy' ELSE 'sell' END as type,
	ce.calculated_lot as lots,
	COALESCE(ox.executed_price, 0) as open_price,
	COALESCE(cx.close_price, 0) as close_price,
	ce.created_at::text as open_time,
	COALESCE(cc.created_at::text, '') as close_time,
	COALESCE(cx.profit, 0) as profit,
	CASE WHEN cx.close_price IS NOT NULL THEN 'closed' ELSE 'open' END as status,
	COALESCE(ta.nickname, ta.account_number, '') as trader_name,
	COALESCE(fa.account_number, '') as follower_account
FROM copy_events ce
LEFT JOIN copy_executions ox ON ox.signal_id = ce.id AND ox.action = 'OPEN'
LEFT JOIN copy_events cc ON cc.provider_ticket = ce.provider_ticket
	AND cc.action = 'CLOSE' AND cc.follower_account_id = ce.follower_account_id
LEFT JOIN copy_executions cx ON cx.signal_id = cc.id AND cx.action = 'CLOSE'
LEFT JOIN trader_accounts ta ON ta.id = ce.provider_account_id
LEFT JOIN trader_accounts fa ON fa.id = ce.follower_account_id
WHERE ce.action = 'OPEN' AND ce.status = 'EXECUTED'
AND fa.user_id = $1::uuid
AND ox.executed_price > 0
AND cx.close_price > 0
AND cx.profit IS NOT NULL`

	var rows *sql.Rows
	var err error
	if accountID != "" {
		rows, err = h.service.repo.DB.Query(query + ` AND ce.follower_account_id = $2::uuid ORDER BY ce.created_at DESC LIMIT 200`, uid, accountID)
	} else {
		rows, err = h.service.repo.DB.Query(query + ` ORDER BY ce.created_at DESC LIMIT 200`, uid)
	}
	if err != nil {
		c.JSON(500, gin.H{"ok": false, "error": err.Error()})
		return
	}
	defer rows.Close()

	type TradeRow struct {
		Ticket          int64   `json:"ticket"`
		Symbol          string  `json:"symbol"`
		Type            string  `json:"type"`
		Lots            float64 `json:"lots"`
		OpenPrice       float64 `json:"openPrice"`
		ClosePrice      float64 `json:"closePrice"`
		OpenTime        string  `json:"openTime"`
		CloseTime       string  `json:"closeTime"`
		Profit          float64 `json:"profit"`
		Status          string  `json:"status"`
		TraderName      string  `json:"traderName"`
		FollowerAccount string  `json:"followerAccount"`
	}

	var trades []TradeRow
	for rows.Next() {
		var t TradeRow
		rows.Scan(&t.Ticket, &t.Symbol, &t.Type, &t.Lots,
			&t.OpenPrice, &t.ClosePrice,
			&t.OpenTime, &t.CloseTime,
			&t.Profit, &t.Status,
			&t.TraderName, &t.FollowerAccount)
		trades = append(trades, t)
	}
	if trades == nil {
		trades = []TradeRow{}
	}
	c.JSON(200, gin.H{"ok": true, "trades": trades, "count": len(trades)})
}
