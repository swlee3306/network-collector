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
	DB   *gorm.DB
	once sync.Once
	mu   sync.Mutex
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
// Returns the same connection instance if already connected
func Connect(config Config) (*gorm.DB, error) {
	mu.Lock()
	defer mu.Unlock()

	// If already connected, return existing connection
	if DB != nil {
		// Verify connection is still alive
		sqlDB, err := DB.DB()
		if err == nil {
			if err := sqlDB.Ping(); err == nil {
				return DB, nil
			}
		}
		// Connection is dead, reset and reconnect
		DB = nil
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

	log.Println("Database connection established")
	return DB, nil
}

// Close closes the database connection
// Uses sync.Once to ensure it's only closed once, even if called multiple times
func Close() error {
	var closeErr error
	once.Do(func() {
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

