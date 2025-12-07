package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/network-collector/backend/internal/config"
)

// Auth returns an authentication middleware
func Auth(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		var token string

		// First, try to get token from Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader != "" {
			// Check if it's a Bearer token
			parts := strings.Split(authHeader, " ")
			if len(parts) == 2 && parts[0] == "Bearer" {
				token = parts[1]
			}
		}

		// If no token from header, try query parameter (for EventSource/SSE)
		// EventSource doesn't support custom headers, so we accept token as query param
		if token == "" {
			token = c.Query("token")
		}

		// Validate token
		if token == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Authorization required (header or query parameter)",
			})
			c.Abort()
			return
		}

		// Validate token (simple check against configured token)
		// In production, this should use JWT or session-based auth
		if cfg.Auth.Token != "" && token != cfg.Auth.Token {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid token",
			})
			c.Abort()
			return
		}

		// Set user context (single operator role)
		c.Set("user", "operator")
		c.Set("role", "operator")

		c.Next()
	}
}

