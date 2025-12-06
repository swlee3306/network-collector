package migrations

import (
	"log"

	"github.com/network-collector/backend/internal/models"
	"gorm.io/gorm"
)

// RunMigrations runs all database migrations
func RunMigrations(db *gorm.DB) error {
	log.Println("Running database migrations...")

	// Fix existing data before migration to prevent column type change errors
	if err := fixExistingData(db); err != nil {
		log.Printf("Warning: Failed to fix existing data: %v", err)
		// Continue with migration even if data fix fails
	}

	// Auto-migrate all models
	allModels := models.AllModels()
	if err := db.AutoMigrate(allModels...); err != nil {
		return err
	}

	// Ensure hypervisor_id and device_id columns are VARCHAR(255) after migration
	// This prevents GORM from trying to change them back to char(36)
	if err := ensureColumnTypes(db); err != nil {
		log.Printf("Warning: Failed to ensure column types: %v", err)
		// Continue even if column type fix fails
	}

	// Create indexes for performance optimization
	if err := createIndexes(db); err != nil {
		return err
	}

	log.Println("Database migrations completed successfully")
	return nil
}

// ensureColumnTypes ensures that specific columns have the correct type
// This is needed because GORM AutoMigrate might try to change column types
// based on model definitions, but we need to preserve VARCHAR(255) for these columns
func ensureColumnTypes(db *gorm.DB) error {
	// Ensure instances.hypervisor_id is VARCHAR(255)
	if db.Migrator().HasTable(&models.Instance{}) {
		if db.Migrator().HasColumn(&models.Instance{}, "hypervisor_id") {
			// Check current type
			var columnType string
			err := db.Raw(`
				SELECT COLUMN_TYPE 
				FROM information_schema.COLUMNS 
				WHERE TABLE_SCHEMA = DATABASE() 
				AND TABLE_NAME = 'instances' 
				AND COLUMN_NAME = 'hypervisor_id'
			`).Scan(&columnType).Error

			if err == nil && columnType != "varchar(255)" {
				log.Println("Ensuring instances.hypervisor_id is VARCHAR(255)...")
				// Use raw SQL to modify column type, ignoring data truncation
				// This will set long values to NULL if needed
				if err := db.Exec(`
					UPDATE instances 
					SET hypervisor_id = NULL 
					WHERE hypervisor_id IS NOT NULL 
					AND LENGTH(hypervisor_id) > 255
				`).Error; err != nil {
					log.Printf("Warning: Failed to truncate long hypervisor_id values: %v", err)
				}
				// Now modify the column type
				if err := db.Exec(`
					ALTER TABLE instances 
					MODIFY COLUMN hypervisor_id VARCHAR(255) NULL
				`).Error; err != nil {
					return err
				}
			}
		}
	}

	// Ensure ports.device_id is VARCHAR(255)
	if db.Migrator().HasTable(&models.Port{}) {
		if db.Migrator().HasColumn(&models.Port{}, "device_id") {
			var columnType string
			err := db.Raw(`
				SELECT COLUMN_TYPE 
				FROM information_schema.COLUMNS 
				WHERE TABLE_SCHEMA = DATABASE() 
				AND TABLE_NAME = 'ports' 
				AND COLUMN_NAME = 'device_id'
			`).Scan(&columnType).Error

			if err == nil && columnType != "varchar(255)" {
				log.Println("Ensuring ports.device_id is VARCHAR(255)...")
				if err := db.Exec(`
					UPDATE ports 
					SET device_id = NULL 
					WHERE device_id IS NOT NULL 
					AND LENGTH(device_id) > 255
				`).Error; err != nil {
					log.Printf("Warning: Failed to truncate long device_id values: %v", err)
				}
				if err := db.Exec(`
					ALTER TABLE ports 
					MODIFY COLUMN device_id VARCHAR(255) NULL
				`).Error; err != nil {
					return err
				}
			}
		}
	}

	return nil
}

// fixExistingData fixes existing data that might cause migration failures
func fixExistingData(db *gorm.DB) error {
	// Check if instances table exists
	if !db.Migrator().HasTable(&models.Instance{}) {
		return nil // Table doesn't exist yet, nothing to fix
	}

	// Check current column type
	var columnType string
	err := db.Raw(`
		SELECT COLUMN_TYPE 
		FROM information_schema.COLUMNS 
		WHERE TABLE_SCHEMA = DATABASE() 
		AND TABLE_NAME = 'instances' 
		AND COLUMN_NAME = 'hypervisor_id'
	`).Scan(&columnType).Error

	if err != nil {
		// Column might not exist yet, that's okay
		return nil
	}

	// If column is already VARCHAR(255), no need to fix
	if columnType == "varchar(255)" {
		return nil
	}

	// If column is char(36) or smaller, we need to handle long values
	// Truncate or set to NULL for values longer than 36 characters
	log.Println("Fixing existing hypervisor_id values that are too long...")
	result := db.Exec(`
		UPDATE instances 
		SET hypervisor_id = NULL 
		WHERE hypervisor_id IS NOT NULL 
		AND LENGTH(hypervisor_id) > 36
	`)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected > 0 {
		log.Printf("Fixed %d instances with hypervisor_id values longer than 36 characters", result.RowsAffected)
	}

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

