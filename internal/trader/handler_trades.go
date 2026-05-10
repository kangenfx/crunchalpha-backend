package trader

import (
	"net/http"
	"strconv"
	"fmt"

	"github.com/gin-gonic/gin"
)

// GetTrades returns paginated trades for an account
func (h *Handler) GetTrades(c *gin.Context) {
	accountID := c.Query("account_id")
	if accountID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "account_id required"})
		return
	}

	limitStr := c.DefaultQuery("limit", "20")
	limit, _ := strconv.Atoi(limitStr)
	if limit <= 0 || limit > 1000 {
		limit = 20
	}

	offsetStr := c.DefaultQuery("offset", "0")
	offset, _ := strconv.Atoi(offsetStr)
	if offset < 0 {
		offset = 0
	}

	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	// Verify account ownership
	_, err := h.service.repo.GetAccountByID(accountID, userID.(string))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "account not found"})
		return
	}

	trades, total, err := h.service.repo.GetTradesByAccountPaginated(accountID, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch trades"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"account_id": accountID,
		"trades":     trades,
		"count":      len(trades),
		"total":      total,
		"limit":      limit,
		"offset":     offset,
	})
}

// GetOpenTrades returns open positions for an account
func (h *Handler) GetOpenTrades(c *gin.Context) {
accountID := c.Query("account_id")
if accountID == "" {
c.JSON(400, gin.H{"error": "account_id required"})
return
}
rows, err := h.service.repo.db.Query(`
SELECT ticket, symbol, type, lots, open_price, profit, swap, open_time
FROM trades
WHERE account_id = $1::uuid AND status = 'open'
ORDER BY open_time DESC`, accountID)
if err != nil {
c.JSON(500, gin.H{"error": err.Error()})
return
}
defer rows.Close()
type OpenTrade struct {
Ticket    int64   `json:"ticket"`
Symbol    string  `json:"symbol"`
Type      string  `json:"type"`
Lots      float64 `json:"lots"`
OpenPrice float64 `json:"openPrice"`
Profit    float64 `json:"profit"`
Swap      float64 `json:"swap"`
OpenTime  string  `json:"openTime"`
}
var trades []OpenTrade
for rows.Next() {
var t OpenTrade
var openTime interface{}
rows.Scan(&t.Ticket, &t.Symbol, &t.Type, &t.Lots, &t.OpenPrice, &t.Profit, &t.Swap, &openTime)
if openTime != nil {
t.OpenTime = fmt.Sprintf("%v", openTime)
}
trades = append(trades, t)
}
if trades == nil {
trades = []OpenTrade{}
}
c.JSON(200, gin.H{"ok": true, "trades": trades, "total": len(trades)})
}
