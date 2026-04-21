package investor

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// GetPortfolio - GET /api/investor/portfolio
func (h *Handler) GetPortfolio(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	portfolio, err := h.service.GetPortfolio(c.Request.Context(), userID.(string))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": portfolio})
}

// GetAllocations - GET /api/investor/allocations
func (h *Handler) GetAllocations(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	settings, err := h.service.GetAllocations(c.Request.Context(), userID.(string))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": settings})
}

// SetAllocation - POST /api/investor/allocations
func (h *Handler) SetAllocation(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	var req AllocationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request", "message": err.Error()})
		return
	}
	if req.TraderAccountID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "trader_account_id is required"})
		return
	}
	if req.Mode == "" {
		req.Mode = "PERCENT"
	}
	err := h.service.SetAllocation(
		c.Request.Context(),
		userID.(string),
		req.TraderAccountID,
		req.FollowerAccountID,
		req.Mode,
		req.Value,
		req.MaxRiskPct,
		req.MaxPositions,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Allocation updated"})
}

// Subscribe - POST /api/investor/subscribe
func (h *Handler) Subscribe(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	var req SubscribeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request", "message": err.Error()})
		return
	}
	if req.TraderAccountID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "trader_account_id is required"})
		return
	}
	err := h.service.Subscribe(c.Request.Context(), userID.(string), req.TraderAccountID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Subscribed successfully"})
}

// GetSubscriptions - GET /api/investor/subscriptions
func (h *Handler) GetSubscriptions(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	portfolio, err := h.service.GetPortfolio(c.Request.Context(), userID.(string))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": portfolio.Subscriptions})
}

// GetTraderList - GET /api/investor/traders
func (h *Handler) GetTraderList(c *gin.Context) {
	traders, err := h.service.GetTraderList(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": traders})
}
