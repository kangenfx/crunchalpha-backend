package trader

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// GetWeeklyPerformance returns weekly aggregated performance from real trades
func (h *Handler) GetWeeklyPerformance(c *gin.Context) {
	accountID := c.Query("account_id")
	if accountID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "account_id required"})
		return
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

	// Get weekly performance from database
	weeks, err := h.service.GetWeeklyPerformanceFromDB(accountID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"account_id": accountID,
		"weeks":      weeks,
		"total":      len(weeks),
	})
}
