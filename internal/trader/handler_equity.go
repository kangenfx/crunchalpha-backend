package trader

import (
	"net/http"
	"strconv"
	"time"
	
	"github.com/gin-gonic/gin"
)

// GetEquityCurve returns equity curve data points
func (h *Handler) GetEquityCurve(c *gin.Context) {
	accountID := c.Query("account_id")
	if accountID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "account_id required"})
		return
	}
	
	daysStr := c.DefaultQuery("days", "30")
	days, _ := strconv.Atoi(daysStr)
	if days <= 0 || days > 365 {
		days = 30
	}
	
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	
	// Verify account ownership
	account, err := h.service.repo.GetAccountByID(accountID, userID.(string))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "account not found"})
		return
	}
	
	// Generate equity curve data
	// Will be replaced with real balance snapshots from trades
	points := generateEquityCurve(account.Balance, account.Equity, days)
	
	c.JSON(http.StatusOK, gin.H{
		"account_id": accountID,
		"points":     points,
		"days":       days,
		"start":      points[0]["equity"],
		"end":        points[len(points)-1]["equity"],
		"growth":     ((points[len(points)-1]["equity"].(float64) - points[0]["equity"].(float64)) / points[0]["equity"].(float64)) * 100,
	})
}

func generateEquityCurve(balance, equity float64, days int) []map[string]interface{} {
	points := []map[string]interface{}{}
	now := time.Now()
	
	currentEquity := balance
	currentBalance := balance
	
	for i := days; i >= 0; i-- {
		date := now.AddDate(0, 0, -i)
		
		// Simulate growth with some volatility
		growth := (float64(days-i) / float64(days)) * (equity - balance)
		volatility := float64((i % 7) - 3) * 50 // Random fluctuation
		
		currentEquity = balance + growth + volatility
		currentBalance = balance + (growth * 0.8) // Balance grows slower
		
		points = append(points, map[string]interface{}{
			"date":    date.Format("2006-01-02"),
			"equity":  currentEquity,
			"balance": currentBalance,
		})
	}
	
	return points
}
