package trader

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// GetTrades returns paginated trades for an account
func (h *Handler) GetTrades(c *gin.Context) {
	accountID := c.Query("account_id")
	if accountID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "account_id required"})
		return
	}

	limitStr := c.DefaultQuery("limit", "50")
	limit, _ := strconv.Atoi(limitStr)
	if limit <= 0 || limit > 1000 {
		limit = 50
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

	// Get real trades from database
	trades, err := h.service.repo.GetTradesByAccount(accountID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch trades"})
		return
	}

	// Apply limit
	if len(trades) > limit {
		trades = trades[:limit]
	}

	c.JSON(http.StatusOK, gin.H{
		"account_id": accountID,
		"trades":     trades,
		"count":      len(trades),
		"limit":      limit,
	})
}
