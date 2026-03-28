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
