package apikey

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	repo *Repository
}

func NewHandler(repo *Repository) *Handler {
	return &Handler{repo: repo}
}

// CreateAPIKey generates new API key for user
func (h *Handler) CreateAPIKey(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req CreateAPIKeyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Generate API key
	key, err := GenerateAPIKey()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate key"})
		return
	}

	// Hash for storage
	keyHash := HashAPIKey(key)
	keyPrefix := GetKeyPrefix(key)

	// Store in database
	keyID, err := h.repo.CreateAPIKey(
		userID.(string),
		keyHash,
		keyPrefix,
		req.Name,
		req.AccountIDs,
		req.AllowedIPs,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create API key"})
		return
	}

	// Return full key (ONLY TIME it's shown!)
	c.JSON(http.StatusCreated, CreateAPIKeyResponse{
		ID:        keyID,
		Key:       key,
		KeyPrefix: keyPrefix,
		Name:      req.Name,
	})
}

// ListAPIKeys returns all API keys for user
func (h *Handler) ListAPIKeys(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	keys, err := h.repo.ListAPIKeys(userID.(string))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list keys"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"api_keys": keys,
		"count":    len(keys),
	})
}

// RevokeAPIKey disables an API key
func (h *Handler) RevokeAPIKey(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	keyID := c.Param("key_id")
	if keyID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "key_id required"})
		return
	}

	if err := h.repo.RevokeAPIKey(keyID, userID.(string)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to revoke key"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"ok":      true,
		"message": "API key revoked",
	})
}
