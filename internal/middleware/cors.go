package middleware

import (
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"

	"api-recaptcha/internal/logger"
)

// corsAllowedMethods are the methods the router actually exposes. Keep in sync with the routes
// registered in cmd/server/main.go; do not advertise verbs nobody can call.
const corsAllowedMethods = "GET, POST, OPTIONS"

// corsAllowedHeaders are the request headers a browser may send. X-API-Key carries the API key
// and Content-Type is set by every JSON client; anything else is not part of the contract.
const corsAllowedHeaders = "Content-Type, X-API-Key"

// CORS configures Cross-Origin Resource Sharing.
//
// CORS_ALLOWED_ORIGINS is a comma-separated allowlist of exact origins. When it is unset the
// middleware denies every cross-origin request: the served clients are servers holding a shared
// key, so allowing a browser origin is always an explicit decision, never a default.
//
// The request Origin is echoed only on an exact allowlist match, together with Vary: Origin so
// caches key on it. A literal "*" is not an allowlist and is ignored with a warning.
//
// Credentials are never advertised: this API authenticates with the X-API-Key header, uses no
// cookie, and "Access-Control-Allow-Origin: *" next to "Allow-Credentials: true" is invalid for
// browsers in any case.
func CORS() gin.HandlerFunc {
	allowedOrigins := os.Getenv("CORS_ALLOWED_ORIGINS")
	if strings.TrimSpace(allowedOrigins) == "" {
		logger.Log.Warn("CORS_ALLOWED_ORIGINS not set, denying every cross-origin request")
	}

	origins := make(map[string]struct{})
	for _, origin := range strings.Split(allowedOrigins, ",") {
		origin = strings.TrimSpace(origin)
		switch {
		case origin == "":
			continue
		case origin == "*":
			logger.Log.Warn("CORS_ALLOWED_ORIGINS contains a wildcard, which is ignored; list explicit origins instead")
			continue
		default:
			origins[origin] = struct{}{}
		}
	}

	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")

		if len(origins) > 0 && origin != "" {
			// The response now depends on the request Origin, whether or not it matches.
			c.Writer.Header().Add("Vary", "Origin")
		}

		if _, ok := origins[origin]; ok {
			c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
			c.Writer.Header().Set("Access-Control-Allow-Methods", corsAllowedMethods)
			c.Writer.Header().Set("Access-Control-Allow-Headers", corsAllowedHeaders)
		}

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}
