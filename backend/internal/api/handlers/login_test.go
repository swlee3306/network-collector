package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/network-collector/backend/internal/config"
	"github.com/stretchr/testify/assert"
)

func TestLogin(t *testing.T) {
	// Set Gin to test mode
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		requestBody    LoginRequest
		config         *config.Config
		expectedStatus int
		expectToken    bool
	}{
		{
			name: "login with configured token",
			requestBody: LoginRequest{
				Username: "operator",
				Password: "password",
			},
			config: &config.Config{
				Auth: config.AuthConfig{
					Token: "configured-token",
				},
			},
			expectedStatus: http.StatusOK,
			expectToken:    true,
		},
		{
			name: "login without configured token",
			requestBody: LoginRequest{
				Username: "operator",
				Password: "password",
			},
			config: &config.Config{
				Auth: config.AuthConfig{
					Token: "",
				},
			},
			expectedStatus: http.StatusOK,
			expectToken:    true,
		},
		{
			name: "login with invalid request body",
			requestBody: LoginRequest{
				Username: "",
				Password: "",
			},
			config: &config.Config{
				Auth: config.AuthConfig{
					Token: "token",
				},
			},
			expectedStatus: http.StatusBadRequest,
			expectToken:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a response recorder
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			// Set request body
			body, _ := json.Marshal(tt.requestBody)
			c.Request, _ = http.NewRequest("POST", "/auth/login", bytes.NewBuffer(body))
			c.Request.Header.Set("Content-Type", "application/json")

			// Call the handler
			handler := Login(tt.config)
			handler(c)

			// Assertions
			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectToken {
				var response LoginResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.NotEmpty(t, response.Token)
				assert.Equal(t, "operator", response.Role)
			}
		})
	}
}

