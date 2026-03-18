package ratelimit

import (
	"net/http"
	"github.com/gin-gonic/gin"
)

func (l *Limiter) RateLimitMiddleware(prefix string, config Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		clientIP := c.ClientIP()
		key := prefix + ":" + clientIP
		
		allowed, err := l.Check(key, config)
		
		if err != nil && err.Error() == "rate limit exceeded" {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": "Too many requests",
				"message": "You have exceeded the rate limit. Please try again later.",
				"retry_after": int(config.Window.Seconds()),
			})
			c.Abort()
			return
		}
		
		if err != nil {
			// Log error but don't block (fail open for now)
			c.Next()
			return
		}
		
		if !allowed {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": "Too many requests",
				"message": "Please try again later.",
			})
			c.Abort()
			return
		}
		
		c.Next()
	}
}
