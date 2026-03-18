package middleware

import (
	"net/http"
	"strings"

	"crunchalpha-v3/internal/auth"
	"github.com/gin-gonic/gin"
)

// AuthRequired middleware validates JWT token
func AuthRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Authorization header required",
				"message": "Please provide a valid access token",
			})
			c.Abort()
			return
		}

		// Check Bearer format
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid authorization format",
				"message": "Format should be: Bearer <token>",
			})
			c.Abort()
			return
		}

		token := parts[1]

		// Validate token
		claims, err := auth.ValidateToken(token)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid token",
				"message": err.Error(),
			})
			c.Abort()
			return
		}

		// Set user info in context for handlers to use
		c.Set("user_id", claims.UserID)
		c.Set("user_email", claims.Email)
		c.Set("user_role", claims.Role)

		c.Next()
	}
}

// RequireRole middleware checks if user has required role
func RequireRole(allowedRoles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userRole, exists := c.Get("user_role")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Unauthorized",
				"message": "User role not found in context",
			})
			c.Abort()
			return
		}

		role := userRole.(string)
		
		// Check if user's role is in allowed roles
		allowed := false
		for _, allowedRole := range allowedRoles {
			if role == allowedRole {
				allowed = true
				break
			}
		}

		if !allowed {
			c.JSON(http.StatusForbidden, gin.H{
				"error": "Forbidden",
				"message": "You don't have permission to access this resource",
				"required_role": allowedRoles,
			})
			c.Abort()
			return
		}

		c.Next()
	}
}
