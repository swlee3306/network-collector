package main

import (
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/network-collector/backend/internal/config"
	"github.com/network-collector/backend/internal/database"
	"github.com/network-collector/backend/internal/database/migrations"
	"github.com/network-collector/backend/internal/services/collector"
	"github.com/network-collector/backend/internal/services/retention"
	"github.com/network-collector/backend/internal/services/storage"
	"github.com/network-collector/backend/pkg/logger"
	"go.uber.org/zap"
)

func main() {
	// Load configuration
	cfg := config.Load()

	// Initialize logger
	if err := logger.Init(cfg.Logging.Level, cfg.Environment); err != nil {
		panic("Failed to initialize logger: " + err.Error())
	}
	defer logger.Sync()

	log := logger.GetLogger()
	log.Info("OpenStack Collector Service starting...")

	// Initialize database connection
	dbConfig := database.Config{
		Host:     cfg.Database.Host,
		Port:     cfg.Database.Port,
		User:     cfg.Database.User,
		Password: cfg.Database.Password,
		Database: cfg.Database.Database,
	}

	db, err := database.Connect(dbConfig)
	if err != nil {
		log.Fatal("Failed to connect to database", zap.Error(err))
	}
	defer database.Close()

	// Run migrations
	if err := migrations.RunMigrations(db); err != nil {
		log.Fatal("Failed to run migrations", zap.Error(err))
	}

	// Initialize repository
	repository := storage.NewRepository(db)

	// Initialize OpenStack collector
	openstackCollector, err := collector.NewOpenStackCollector(cfg, repository)
	if err != nil {
		log.Fatal("Failed to create OpenStack collector", zap.Error(err))
	}

	// Initial collection
	log.Info("Performing initial collection...")
	if err := openstackCollector.CollectAll(); err != nil {
		log.Warn("Initial collection completed with errors", zap.Error(err))
		// Continue even if initial collection fails (partial failure is acceptable)
	}

	// Start collection scheduler (1 minute interval)
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	// Collection goroutine
	go func() {
		for {
			select {
			case <-ticker.C:
				log.Info("Starting scheduled collection...")
				if err := openstackCollector.CollectAll(); err != nil {
					log.Warn("Collection completed with errors", zap.Error(err))
					// Continue collecting even if there are errors (partial failure handling)
				}
			}
		}
	}()

	log.Info("OpenStack Collector Service started", zap.String("collection_interval", "1 minute"))

	// Start data retention cleanup service (daily cleanup)
	cleanupService := retention.NewCleanupService(repository, 30) // 30 days retention
	go cleanupService.RunPeriodicCleanup(24 * time.Hour)

	log.Info("Data retention cleanup service started", zap.Int("retention_days", 30), zap.String("cleanup_interval", "24 hours"))

	// Graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Info("OpenStack Collector Service shutting down...")
}

