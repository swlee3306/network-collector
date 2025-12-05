package migrations

import (
	"log"

	"github.com/network-collector/backend/internal/models"
	"gorm.io/gorm"
)

// RunMigrations runs all database migrations
func RunMigrations(db *gorm.DB) error {
	log.Println("Running database migrations...")

	// Auto-migrate all models
	allModels := models.AllModels()
	if err := db.AutoMigrate(allModels...); err != nil {
		return err
	}

	// Create indexes for performance optimization
	if err := createIndexes(db); err != nil {
		return err
	}

	log.Println("Database migrations completed successfully")
	return nil
}

// createIndexes creates additional indexes for performance optimization
func createIndexes(db *gorm.DB) error {
	// Composite index for topology edge path finding
	if err := db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_topology_edges_source_target 
		ON topology_edges(source_node_id, target_node_id)
	`).Error; err != nil {
		return err
	}

	// Index for timestamp-based queries (for data retention)
	// Each index must have a unique name across the database
	if err := db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_instance_metrics_timestamp 
		ON instance_metrics(timestamp)
	`).Error; err != nil {
		return err
	}

	if err := db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_network_metrics_timestamp 
		ON network_metrics(timestamp)
	`).Error; err != nil {
		return err
	}

	if err := db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_hypervisor_metrics_timestamp 
		ON hypervisor_metrics(timestamp)
	`).Error; err != nil {
		return err
	}

	return nil
}

// RollbackMigrations rolls back all migrations (for development/testing)
func RollbackMigrations(db *gorm.DB) error {
	log.Println("Rolling back database migrations...")

	allModels := models.AllModels()
	for _, model := range allModels {
		if err := db.Migrator().DropTable(model); err != nil {
			return err
		}
	}

	log.Println("Database migrations rolled back")
	return nil
}

