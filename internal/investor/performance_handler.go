package investor

import (
	"strconv"

	"github.com/gin-gonic/gin"
)

// GetTraderMonthlyPerformance - investor lens: view any trader's monthly performance
func (h *Handler) GetTraderMonthlyPerformance(c *gin.Context) {
	accountID := c.Query("account_id")
	if accountID == "" {
		c.JSON(400, gin.H{"error": "account_id required"})
		return
	}
	rows, err := h.service.repo.DB.Query(`
		SELECT TO_CHAR(close_time, 'Mon') as month,
		       EXTRACT(YEAR FROM close_time) as year,
		       COUNT(*) as trades,
		       SUM(CASE WHEN (profit+COALESCE(swap,0)+COALESCE(commission,0)) > 0 THEN 1 ELSE 0 END) as wins,
		       SUM(CASE WHEN (profit+COALESCE(swap,0)+COALESCE(commission,0)) < 0 THEN 1 ELSE 0 END) as losses,
		       SUM(profit+COALESCE(swap,0)+COALESCE(commission,0)) as total_profit
		FROM trades
		WHERE account_id=$1::uuid AND status='closed'
		GROUP BY TO_CHAR(close_time,'Mon'), EXTRACT(YEAR FROM close_time), DATE_TRUNC('month',close_time)
		ORDER BY DATE_TRUNC('month',close_time) ASC`, accountID)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	var months []map[string]interface{}
	for rows.Next() {
		var month string
		var year int
		var trades, wins, losses int
		var totalProfit float64
		rows.Scan(&month, &year, &trades, &wins, &losses, &totalProfit)
		winRate := 0.0
		if trades > 0 { winRate = float64(wins) / float64(trades) * 100 }
		months = append(months, map[string]interface{}{
			"month": month, "year": year, "trades": trades,
			"wins": wins, "losses": losses, "profit": totalProfit, "winRate": winRate,
		})
	}
	if months == nil { months = []map[string]interface{}{} }
	c.JSON(200, gin.H{"account_id": accountID, "months": months})
}

// GetTraderWeeklyPerformance - investor lens: view any trader's weekly performance
func (h *Handler) GetTraderWeeklyPerformance(c *gin.Context) {
	accountID := c.Query("account_id")
	if accountID == "" {
		c.JSON(400, gin.H{"error": "account_id required"})
		return
	}
	rows, err := h.service.repo.DB.Query(`
		SELECT EXTRACT(WEEK FROM close_time) as week,
		       EXTRACT(YEAR FROM close_time) as year,
		       COUNT(*) as trades,
		       SUM(CASE WHEN (profit+COALESCE(swap,0)+COALESCE(commission,0)) > 0 THEN 1 ELSE 0 END) as wins,
		       SUM(profit+COALESCE(swap,0)+COALESCE(commission,0)) as total_profit
		FROM trades
		WHERE account_id=$1::uuid AND status='closed'
		GROUP BY EXTRACT(WEEK FROM close_time), EXTRACT(YEAR FROM close_time), DATE_TRUNC('week',close_time)
		ORDER BY DATE_TRUNC('week',close_time) ASC`, accountID)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	var weeks []map[string]interface{}
	for rows.Next() {
		var week, year int
		var trades, wins int
		var totalProfit float64
		rows.Scan(&week, &year, &trades, &wins, &totalProfit)
		weeks = append(weeks, map[string]interface{}{
			"week": week, "year": year, "trades": trades,
			"wins": wins, "profit": totalProfit,
		})
	}
	if weeks == nil { weeks = []map[string]interface{}{} }
	c.JSON(200, gin.H{"account_id": accountID, "weeks": weeks})
}

// GetTraderTrades - investor lens: trade history tanpa lot/entry/exit price
func (h *Handler) GetTraderTrades(c *gin.Context) {
	accountID := c.Query("account_id")
	if accountID == "" {
		c.JSON(400, gin.H{"error": "account_id required"})
		return
	}
	limitStr := c.DefaultQuery("limit", "20")
	limit := 20
	if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 1000 {
		limit = l
	}
	offsetStr := c.DefaultQuery("offset", "0")
	offset := 0
	if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
		offset = o
	}
	// Verify account exists and active
	var exists bool
	h.service.repo.DB.QueryRow(`
		SELECT EXISTS(SELECT 1 FROM trader_accounts WHERE id=$1::uuid AND status='active')`,
		accountID).Scan(&exists)
	if !exists {
		c.JSON(404, gin.H{"error": "trader not found"})
		return
	}
	// Total count
	var total int
	h.service.repo.DB.QueryRow(`
		SELECT COUNT(*) FROM trades WHERE account_id=$1::uuid AND status='closed'`,
		accountID).Scan(&total)
	// Query trades
	rows, err := h.service.repo.DB.Query(`
		SELECT symbol, type, open_time, close_time,
		       COALESCE(profit,0)+COALESCE(swap,0)+COALESCE(commission,0) as net_profit,
		       COALESCE(status,'closed')
		FROM trades
		WHERE account_id=$1::uuid AND status='closed'
		ORDER BY close_time DESC NULLS LAST, open_time DESC
		LIMIT $2 OFFSET $3`, accountID, limit, offset)
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
			"symbol": symbol, "type": tradeType,
			"open_time": openTime, "close_time": closeTime,
			"profit": profit, "status": status,
		})
	}
	if trades == nil { trades = []map[string]interface{}{} }
	c.JSON(200, gin.H{
		"account_id": accountID,
		"trades":     trades,
		"count":      len(trades),
		"total":      total,
		"limit":      limit,
		"offset":     offset,
	})
}
