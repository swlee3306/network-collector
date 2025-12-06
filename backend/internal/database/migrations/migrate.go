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

	// Drop foreign key constraints that might cause GORM to create wrong column types
	// GORM AutoMigrate creates foreign key columns based on referenced table's primary key type
	// We need to prevent this for hypervisor_id and device_id which should be VARCHAR(255)
	if err := dropProblematicForeignKeys(db); err != nil {
		log.Printf("Warning: Failed to drop problematic foreign keys: %v", err)
		// Continue with migration even if FK drop fails
	}

	// Auto-migrate all models
	allModels := models.AllModels()
	if err := db.AutoMigrate(allModels...); err != nil {
		return err
	}

	// Ensure hypervisor_id and device_id columns are VARCHAR(255) after migration
	// GORM AutoMigrate might have created them as char(36) due to foreign key relationships
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

// dropProblematicForeignKeys drops foreign key constraints that cause GORM to create wrong column types
// This should be called BEFORE AutoMigrate to prevent GORM from creating char(36) columns
func dropProblematicForeignKeys(db *gorm.DB) error {
	// Drop fk_hypervisors_instances if it exists
	// This prevents GORM from creating hypervisor_id as char(36)
	if db.Migrator().HasTable(&models.Instance{}) {
		var fkName string
		err := db.Raw(`
			SELECT CONSTRAINT_NAME
			FROM information_schema.KEY_COLUMN_USAGE
			WHERE TABLE_SCHEMA = DATABASE()
			AND TABLE_NAME = 'instances'
			AND COLUMN_NAME = 'hypervisor_id'
			AND REFERENCED_TABLE_NAME IS NOT NULL
			LIMIT 1
		`).Scan(&fkName).Error

		if err == nil && fkName != "" {
			log.Printf("Dropping foreign key constraint before migration: %s", fkName)
			if err := db.Exec(`
				SET FOREIGN_KEY_CHECKS = 0;
				ALTER TABLE instances DROP FOREIGN KEY ` + fkName + `;
				SET FOREIGN_KEY_CHECKS = 1;
			`).Error; err != nil {
				log.Printf("Warning: Failed to drop foreign key constraint: %v", err)
			}
		}
	}

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
				
				// First, find and drop the foreign key constraint if it exists
				var fkName string
				err = db.Raw(`
					SELECT CONSTRAINT_NAME
					FROM information_schema.KEY_COLUMN_USAGE
					WHERE TABLE_SCHEMA = DATABASE()
					AND TABLE_NAME = 'instances'
					AND COLUMN_NAME = 'hypervisor_id'
					AND REFERENCED_TABLE_NAME IS NOT NULL
					LIMIT 1
				`).Scan(&fkName).Error
				
				if err == nil && fkName != "" {
					log.Printf("Dropping foreign key constraint: %s", fkName)
					if err := db.Exec(`
						SET FOREIGN_KEY_CHECKS = 0;
						ALTER TABLE instances DROP FOREIGN KEY ` + fkName + `;
						SET FOREIGN_KEY_CHECKS = 1;
					`).Error; err != nil {
						log.Printf("Warning: Failed to drop foreign key constraint: %v", err)
					}
				}
				
				// Set empty strings to NULL and truncate long values
				if err := db.Exec(`
					UPDATE instances 
					SET hypervisor_id = NULL 
					WHERE (hypervisor_id = '' OR hypervisor_id IS NOT NULL AND LENGTH(hypervisor_id) > 255)
				`).Error; err != nil {
					log.Printf("Warning: Failed to fix hypervisor_id values: %v", err)
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

