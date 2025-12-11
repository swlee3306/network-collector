package config

import (
	"fmt"
	"os"
)

// DatabaseConfig holds database configuration
type DatabaseConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	Database string
}

// OpenStackConfig holds OpenStack API configuration
type OpenStackConfig struct {
	AuthURL    string
	Username   string
	Password   string
	ProjectID  string
	DomainName string
}

// ServerConfig holds server configuration
type ServerConfig struct {
	Port         int
	JWTSecret    string
	SSEEnabled   bool
}

// AuthConfig holds authentication configuration
type AuthConfig struct {
	Token     string
	LoginType string // "internal" or "public" - controls login endpoint access
}

// LoggingConfig holds logging configuration
type LoggingConfig struct {
	Level string
}

// Config holds application configuration
type Config struct {
	Database   DatabaseConfig
	OpenStack  OpenStackConfig
	Server     ServerConfig
	Auth       AuthConfig
	Logging    LoggingConfig
	Environment string
}

// Load loads configuration from environment variables
func Load() *Config {
	return &Config{
		Database: DatabaseConfig{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnvAsInt("DB_PORT", 3306),
			User:     getEnv("DB_USER", "root"),
			Password: getEnv("DB_PASSWORD", ""),
			Database: getEnv("DB_NAME", "openstack_monitor"),
		},
		OpenStack: OpenStackConfig{
			AuthURL:    getEnv("OPENSTACK_AUTH_URL", ""),
			Username:   getEnv("OPENSTACK_USERNAME", ""),
			Password:   getEnv("OPENSTACK_PASSWORD", ""),
			ProjectID:  getEnv("OPENSTACK_PROJECT_ID", ""),
			DomainName: getEnv("OPENSTACK_DOMAIN_NAME", "default"),
		},
		Server: ServerConfig{
			Port:       getEnvAsInt("SERVER_PORT", 8080),
			JWTSecret:  getEnv("JWT_SECRET", "change-me-in-production"),
			SSEEnabled: getEnvAsBool("SSE_ENABLED", true),
		},
		Auth: AuthConfig{
			Token:     getEnv("AUTH_TOKEN", ""),
			LoginType: getEnv("LOGIN_TYPE", "internal"), // "internal" (no auth) or "public" (requires auth)
		},
		Logging: LoggingConfig{
			Level: getEnv("LOG_LEVEL", "info"),
		},
		Environment: getEnv("ENVIRONMENT", "development"),
	}
}

// Helper functions
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	// Simple conversion - in production, use strconv.Atoi with error handling
	// For now, return default if conversion fails
	var result int
	if _, err := fmt.Sscanf(value, "%d", &result); err != nil {
		return defaultValue
	}
	return result
}

func getEnvAsBool(key string, defaultValue bool) bool {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value == "true" || value == "1"
}

