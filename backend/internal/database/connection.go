package database

import (
	"fmt"
	"log"
	"sync"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var (
	DB       *gorm.DB
	closeOnce sync.Once
	mu       sync.Mutex
)

// Config holds database configuration
type Config struct {
	Host     string
	Port     int
	User     string
	Password string
	Database string
}

// Connect initializes database connection
// Returns the same connection instance if already connected and alive
// If connection was closed, creates a new connection and resets the closeOnce state
func Connect(config Config) (*gorm.DB, error) {
	mu.Lock()
	defer mu.Unlock()

	// If already connected, verify connection is still alive
	if DB != nil {
		sqlDB, err := DB.DB()
		if err == nil {
			if err := sqlDB.Ping(); err == nil {
				return DB, nil
			}
		}
		// Connection is dead, reset and reconnect
		DB = nil
		// Reset closeOnce to allow Close to be called again after reconnection
		closeOnce = sync.Once{}
	}

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		config.User,
		config.Password,
		config.Host,
		config.Port,
		config.Database,
	)

	var err error
	DB, err = gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})

	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Reset closeOnce when establishing a new connection
	// This allows Close to be called again for this new connection
	closeOnce = sync.Once{}

	log.Println("Database connection established")
	return DB, nil
}

// Close closes the database connection
// Uses sync.Once to ensure it's only closed once per connection instance, even if called multiple times
// After closing, a new connection can be established by calling Connect again
func Close() error {
	var closeErr error
	closeOnce.Do(func() {
		mu.Lock()
		defer mu.Unlock()
		if DB != nil {
			sqlDB, err := DB.DB()
			if err != nil {
				closeErr = err
				return
			}
			closeErr = sqlDB.Close()
			DB = nil
		}
	})
	return closeErr
}

// ResetCloseState resets the closeOnce state to allow reconnection after Close
// This should only be used in testing or when you need to explicitly reset the connection state
func ResetCloseState() {
	mu.Lock()
	defer mu.Unlock()
	// Create a new sync.Once to allow Close to be called again after reconnection
	closeOnce = sync.Once{}
}

