// +build integration

package integration

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/network-collector/backend/internal/api"
	appconfig "github.com/network-collector/backend/internal/config"
	"github.com/network-collector/backend/internal/database"
	"github.com/network-collector/backend/internal/database/migrations"
	"github.com/network-collector/backend/internal/models"
	"github.com/network-collector/backend/internal/services/events"
	"github.com/network-collector/backend/internal/services/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

var testDB *gorm.DB
var testServer *api.Server
var testToken string

func setupAPITest(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Set Gin to test mode
	gin.SetMode(gin.TestMode)

	// Setup test database
	config := database.Config{
		Host:     getEnvOrDefault("TEST_DB_HOST", "localhost"),
		Port:     getEnvAsIntOrDefault("TEST_DB_PORT", 3306),
		User:     getEnvOrDefault("TEST_DB_USER", "root"),
		Password: getEnvOrDefault("TEST_DB_PASSWORD", ""),
		Database: getEnvOrDefault("TEST_DB_NAME", "openstack_monitor_test"),
	}

	var err error
	testDB, err = database.Connect(config)
	require.NoError(t, err, "Failed to connect to test database")

	// Run migrations
	err = migrations.RunMigrations(testDB)
	require.NoError(t, err, "Failed to run migrations")

	// Setup test data
	setupTestData(t, testDB)

	// Create API server
	cfg := &appconfig.Config{
		Server: appconfig.ServerConfig{
			Port:      8080,
			JWTSecret: "test-secret",
		},
		Auth: appconfig.AuthConfig{
			Token: "test-token",
		},
		Logging: appconfig.LoggingConfig{
			Level: "info",
		},
		Environment: "test",
	}

	broadcaster := events.NewBroadcaster()
	testServer, err = api.NewServer(cfg, testDB, broadcaster)
	require.NoError(t, err, "Failed to create API server")

	testToken = "test-token"
}

func teardownAPITest(t *testing.T) {
	if testDB != nil {
		// Clean up test data
		cleanupTestData(t, testDB)
		database.Close()
	}
}

func setupTestData(t *testing.T, db *gorm.DB) {
	// Create test instance
	instance := &models.Instance{
		OpenStackID: "test-instance-api",
		Name:        "test-vm-api",
		Status:      "ACTIVE",
	}
	err := db.Save(instance).Error
	require.NoError(t, err)

	// Create test network
	network := &models.Network{
		OpenStackID: "test-network-api",
		Name:        "test-network-api",
		Status:      "ACTIVE",
	}
	err = db.Save(network).Error
	require.NoError(t, err)
}

// cleanupTestData is defined in database_test.go

func TestAPI_HealthCheck(t *testing.T) {
	setupAPITest(t)
	defer teardownAPITest(t)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/health", nil)
	testServer.Router().ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "healthy", response["status"])
}

func TestAPI_Login(t *testing.T) {
	setupAPITest(t)
	defer teardownAPITest(t)

	w := httptest.NewRecorder()
	loginReq := map[string]string{
		"username": "operator",
		"password": "password",
	}
	body, _ := json.Marshal(loginReq)
	req, _ := http.NewRequest("POST", "/api/v1/auth/login", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	testServer.Router().ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.NotEmpty(t, response["token"])
	assert.Equal(t, "operator", response["role"])
}

func TestAPI_ListInstances(t *testing.T) {
	setupAPITest(t)
	defer teardownAPITest(t)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/instances", nil)
	req.Header.Set("Authorization", "Bearer "+testToken)
	testServer.Router().ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.NotNil(t, response["data"])
}

func TestAPI_GetInstance(t *testing.T) {
	setupAPITest(t)
	defer teardownAPITest(t)

	// First, get instance ID from database
	repo := storage.NewRepository(testDB)
	instance, err := repo.GetInstanceByOpenStackID("test-instance-api")
	require.NoError(t, err)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/instances/"+instance.ID, nil)
	req.Header.Set("Authorization", "Bearer "+testToken)
	testServer.Router().ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.NotNil(t, response["data"])
}

func TestAPI_UnauthorizedAccess(t *testing.T) {
	setupAPITest(t)
	defer teardownAPITest(t)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/instances", nil)
	// No Authorization header
	testServer.Router().ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

