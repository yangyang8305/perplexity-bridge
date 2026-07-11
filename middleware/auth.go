package middleware

import (
	"pplx2api/config"
	"strings"

	"github.com/gin-gonic/gin"
)

// AuthMiddleware validates the Bearer token against APIKEY.
// A1 fix: when APIKEY is empty, all requests are allowed through (no-auth mode).
// A4 fix: read APIKey under RLock to avoid race with hot-reload.
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		config.ConfigInstance.RwMutex.RLock()
		apiKey := config.ConfigInstance.APIKey
		config.ConfigInstance.RwMutex.RUnlock()

		// A1 fix: empty APIKEY means no-auth mode — let everything through
		if apiKey == "" {
			c.Next()
			return
		}

		key := strings.TrimPrefix(c.GetHeader("Authorization"), "Bearer ")
		if key == "" {
			c.JSON(401, gin.H{"error": "Missing Authorization header"})
			c.Abort()
			return
		}
		if key != apiKey {
			c.JSON(401, gin.H{"error": "Invalid API key"})
			c.Abort()
			return
		}
		c.Next()
	}
}
