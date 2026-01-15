package middleware

import (
	"os"
	"strings"

	"github.com/gin-gonic/gin"
)

// CORS configures Cross-Origin Resource Sharing settings.
// Reads allowed origins from CORS_ALLOWED_ORIGINS env var (comma-separated).
// Defaults to allowing all origins if not specified (for development).
func CORS() gin.HandlerFunc {
	allowedOrigins := os.Getenv("CORS_ALLOWED_ORIGINS")
	if allowedOrigins == "" {
		allowedOrigins = "*"
	}

	origins := strings.Split(allowedOrigins, ",")
	originsMap := make(map[string]bool)
	for _, origin := range origins {
		originsMap[strings.TrimSpace(origin)] = true
	}

	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")

		// If wildcard or origin is in allowed list
		if originsMap["*"] || originsMap[origin] {
			c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
			if origin == "" {
				c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
			}
		}

		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With, X-API-Key")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}
