package trader

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// DeleteAccountHandler handles account deletion
func (h *Handler) DeleteAccountHandler(c *gin.Context) {
	accountID := c.Param("account_id")
	if accountID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "account_id required"})
		return
	}

	// Get user ID from context
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	// Delete using service
	err := h.service.DeleteAccount(accountID, userID.(string))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to delete account",
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Account deleted successfully",
	})
}
