package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/network-collector/backend/internal/config"
)

// LoginRequest represents login request body
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// LoginResponse represents login response
type LoginResponse struct {
	Token string `json:"token"`
	Role  string `json:"role"`
}

// Login handles login requests
func Login(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req LoginRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Invalid request body",
			})
			return
		}

		// Simple authentication (single operator role)
		// In production, this should validate against a user database
		// For now, we'll use a simple token-based auth
		if cfg.Auth.Token != "" {
			// If token is configured, return it
			c.JSON(http.StatusOK, LoginResponse{
				Token: cfg.Auth.Token,
				Role:  "operator",
			})
			return
		}

		// Otherwise, return a default token (for development)
		c.JSON(http.StatusOK, LoginResponse{
			Token: "default-token",
			Role:  "operator",
		})
	}
}

