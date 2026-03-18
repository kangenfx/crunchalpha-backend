package investor

import (
	"net/http"
	"strconv"
	"github.com/gin-gonic/gin"
	"crunchalpha-v3/internal/trader"
)

// GetTraderMonthlyPerformance - investor lens: view any trader's monthly performance
func (h *Handler) GetTraderMonthlyPerformance(c *gin.Context) {
	accountID := c.Query("account_id")
	if accountID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "account_id required"})
		return
	}
	// Verify trader account exists and is active (no ownership check)
	var exists bool
	err := h.service.repo.DB.QueryRow(`
		SELECT EXISTS(SELECT 1 FROM trader_accounts WHERE id=$1::uuid AND status='active')`,
		accountID).Scan(&exists)
	if err != nil || !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "trader not found"})
		return
	}
	months, err := trader.GetMonthlyPerformanceFromDB(h.service.repo.DB, accountID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"account_id": accountID, "months": months, "total": len(months)})
}

// GetTraderWeeklyPerformance - investor lens: view any trader's weekly performance
func (h *Handler) GetTraderWeeklyPerformance(c *gin.Context) {
	accountID := c.Query("account_id")
	if accountID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "account_id required"})
		return
	}
	var exists bool
	err := h.service.repo.DB.QueryRow(`
		SELECT EXISTS(SELECT 1 FROM trader_accounts WHERE id=$1::uuid AND status='active')`,
		accountID).Scan(&exists)
	if err != nil || !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "trader not found"})
		return
	}
	weeks, err := trader.GetWeeklyPerformanceFromDB(h.service.repo.DB, accountID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"account_id": accountID, "weeks": weeks, "total": len(weeks)})
}

// GetTraderTrades - investor lens: trade history tanpa lot/entry/exit price
func (h *Handler) GetTraderTrades(c *gin.Context) {
	accountID := c.Query("account_id")
	if accountID == "" {
		c.JSON(400, gin.H{"error": "account_id required"})
		return
	}
	limitStr := c.DefaultQuery("limit", "50")
	limit := 50
	if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 1000 {
		limit = l
	}
	// Verify account exists and active - no ownership check
	var exists bool
	h.service.repo.DB.QueryRow(`
		SELECT EXISTS(SELECT 1 FROM trader_accounts WHERE id=$1::uuid AND status='active')`,
		accountID).Scan(&exists)
	if !exists {
		c.JSON(404, gin.H{"error": "trader not found"})
		return
	}
	// Query trades - investor lens: no lot_size, no open_price, no close_price
	rows, err := h.service.repo.DB.Query(`
		SELECT symbol, type, open_time, close_time,
			COALESCE(profit, 0), COALESCE(status, 'CLOSED')
		FROM trades
		WHERE account_id=$1::uuid
		ORDER BY close_time DESC NULLS LAST, open_time DESC
		LIMIT $2`, accountID, limit)
	if err != nil {
		c.JSON(500, gin.H{"error": "failed to fetch trades"})
		return
	}
	defer rows.Close()
	var trades []map[string]interface{}
	for rows.Next() {
		var symbol, tradeType, status string
		var openTime, closeTime *string
		var profit float64
		rows.Scan(&symbol, &tradeType, &openTime, &closeTime, &profit, &status)
		trades = append(trades, map[string]interface{}{
			"symbol":     symbol,
			"type":       tradeType,
			"open_time":  openTime,
			"close_time": closeTime,
			"profit":     profit,
			"status":     status,
		})
	}
	if trades == nil { trades = []map[string]interface{}{} }
	c.JSON(200, gin.H{"account_id": accountID, "trades": trades, "count": len(trades)})
}
