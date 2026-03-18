package middleware

import (
	"net/http"
	"strings"

	"crunchalpha-v3/internal/apikey"
	"github.com/gin-gonic/gin"
)

// APIKeyAuth middleware validates API key and sets user context
func APIKeyAuth(repo *apikey.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get API key from header
		authHeader := c.GetHeader("X-API-Key")
		if authHeader == "" {
			// Try Authorization header as fallback
			authHeader = c.GetHeader("Authorization")
			if strings.HasPrefix(authHeader, "ApiKey ") {
				authHeader = strings.TrimPrefix(authHeader, "ApiKey ")
			} else {
				authHeader = ""
			}
		}

		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "API key required",
				"message": "Provide X-API-Key header",
			})
			c.Abort()
			return
		}

		// Validate key format
		if !apikey.ValidateKeyFormat(authHeader) {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "Invalid API key format",
				"message": "Key should start with 'crunch_'",
			})
			c.Abort()
			return
		}

		// Hash and lookup
		keyHash := apikey.HashAPIKey(authHeader)
		key, err := repo.GetAPIKeyByHash(keyHash)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "Invalid API key",
				"message": "Key not found or revoked",
			})
			c.Abort()
			return
		}

		// Check IP whitelist if configured
		if len(key.AllowedIPs) > 0 {
			clientIP := c.ClientIP()
			allowed := false
			for _, ip := range key.AllowedIPs {
				if ip == clientIP {
					allowed = true
					break
				}
			}
			if !allowed {
				c.JSON(http.StatusForbidden, gin.H{
					"error":   "IP not allowed",
					"message": "Your IP is not whitelisted for this key",
					"your_ip": clientIP,
				})
				c.Abort()
				return
			}
		}

		// Set context
		c.Set("user_id", key.UserID)
		c.Set("api_key_id", key.ID)
		c.Set("api_key_accounts", key.AccountIDs)

		// Update last used (async, don't block request)
		go func() {
			repo.UpdateLastUsed(key.ID)
			repo.LogAPIKeyUsage(key.ID, c.ClientIP(), c.Request.URL.Path, c.Request.Method, 0)
		}()

		c.Next()

		// Log final status code
		go func() {
			repo.LogAPIKeyUsage(key.ID, c.ClientIP(), c.Request.URL.Path, c.Request.Method, c.Writer.Status())
		}()
	}
}

// RequireAccount middleware checks if API key has access to requested account
func RequireAccount(c *gin.Context) {
	accountID := c.Query("account_id")
	if accountID == "" {
		accountID = c.Param("account_id")
	}

	if accountID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "account_id required",
			"message": "Specify which account to access",
		})
		c.Abort()
		return
	}

	// Get allowed accounts from context
	allowedAccounts, exists := c.Get("api_key_accounts")
	if !exists {
		// No API key context, skip check (JWT flow)
		c.Next()
		return
	}

	accounts := allowedAccounts.([]string)

	// Check if account is in allowed list
	allowed := false
	for _, acc := range accounts {
		if acc == accountID {
			allowed = true
			break
		}
	}

	if !allowed {
		c.JSON(http.StatusForbidden, gin.H{
			"error":            "Account access denied",
			"message":          "This API key cannot access this account",
			"requested":        accountID,
			"allowed_accounts": accounts,
		})
		c.Abort()
		return
	}

	c.Next()
}
