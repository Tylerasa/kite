package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// SecurityHeaders sets CORS headers for the configured frontend origin and
// adds standard security response headers on every request.
func SecurityHeaders(allowedOrigin string) gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")

		// Only echo back the origin header if it matches the configured allow-list.
		// This avoids a wildcard CORS policy while still supporting the frontend.
		if origin == allowedOrigin {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			c.Header("Access-Control-Allow-Headers", "Authorization, Content-Type, Idempotency-Key")
			c.Header("Access-Control-Max-Age", "86400")
			c.Header("Vary", "Origin")
		}

		// Handle CORS preflight — browsers send OPTIONS before the real request.
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		// Prevent MIME-type sniffing.
		c.Header("X-Content-Type-Options", "nosniff")
		// Deny embedding in iframes (clickjacking protection).
		c.Header("X-Frame-Options", "DENY")
		// Limit referrer information sent to third parties.
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")

		c.Next()
	}
}
