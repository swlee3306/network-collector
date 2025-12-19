package migrations

import (
	"fmt"
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
	// This MUST succeed - if it fails, the migration should fail
	if err := ensureColumnTypes(db); err != nil {
		log.Printf("Error: Failed to ensure column types: %v", err)
		// This is critical - return error to prevent incorrect schema
		return fmt.Errorf("failed to ensure correct column types: %w", err)
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
			// Execute each SQL statement separately - GORM Exec() can only handle one statement at a time
			if err := db.Exec(`SET FOREIGN_KEY_CHECKS = 0`).Error; err != nil {
				log.Printf("Warning: Failed to disable foreign key checks: %v", err)
			}
			// Use backticks for FK name to handle special characters
			if err := db.Exec("ALTER TABLE instances DROP FOREIGN KEY `" + fkName + "`").Error; err != nil {
				log.Printf("Warning: Failed to drop foreign key constraint %s: %v", fkName, err)
			} else {
				log.Printf("Successfully dropped foreign key constraint: %s", fkName)
			}
			if err := db.Exec(`SET FOREIGN_KEY_CHECKS = 1`).Error; err != nil {
				log.Printf("Warning: Failed to enable foreign key checks: %v", err)
			}
		}
	}

	return nil
}

// ensureColumnTypes ensures that specific columns have the correct type
// This is needed because GORM AutoMigrate might try to change column types
// based on model definitions, but we need to preserve VARCHAR(255) for these columns
// This function ALWAYS runs after AutoMigrate to fix any incorrect column types
func ensureColumnTypes(db *gorm.DB) error {
	// Ensure instances.hypervisor_id is VARCHAR(255)
	// This MUST run every time to fix GORM AutoMigrate's incorrect type inference
	if db.Migrator().HasTable(&models.Instance{}) {
		if db.Migrator().HasColumn(&models.Instance{}, "hypervisor_id") {
			// Always check and fix the column type, regardless of current type
			var columnType string
			err := db.Raw(`
				SELECT COLUMN_TYPE 
				FROM information_schema.COLUMNS 
				WHERE TABLE_SCHEMA = DATABASE() 
				AND TABLE_NAME = 'instances' 
				AND COLUMN_NAME = 'hypervisor_id'
			`).Scan(&columnType).Error

			if err == nil {
				// Check if it's NOT varchar(255) - if so, fix it
				if columnType != "varchar(255)" {
					log.Printf("Fixing instances.hypervisor_id from %s to VARCHAR(255)...", columnType)
					
					// ALWAYS drop foreign key constraint first (it might have been recreated by AutoMigrate)
					// This MUST succeed or column type change will fail
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
						// Disable foreign key checks first
						if err := db.Exec(`SET FOREIGN_KEY_CHECKS = 0`).Error; err != nil {
							log.Printf("Warning: Failed to disable foreign key checks: %v", err)
						}
						
						// Try to drop FK with backticks (safer for special characters)
						dropSuccess := false
						if err := db.Exec("ALTER TABLE instances DROP FOREIGN KEY `" + fkName + "`").Error; err != nil {
							log.Printf("Warning: Failed to drop foreign key constraint %s with backticks: %v", fkName, err)
							// Try without backticks as fallback
							if err2 := db.Exec(`ALTER TABLE instances DROP FOREIGN KEY ` + fkName).Error; err2 != nil {
								log.Printf("Error: Both attempts to drop FK failed: %v, %v", err, err2)
							} else {
								log.Printf("Successfully dropped foreign key constraint without backticks")
								dropSuccess = true
							}
						} else {
							log.Printf("Successfully dropped foreign key constraint: %s", fkName)
							dropSuccess = true
						}
						
						// Verify FK was actually dropped
						if dropSuccess {
							var verifyFkName string
							verifyErr := db.Raw(`
								SELECT CONSTRAINT_NAME
								FROM information_schema.KEY_COLUMN_USAGE
								WHERE TABLE_SCHEMA = DATABASE()
								AND TABLE_NAME = 'instances'
								AND COLUMN_NAME = 'hypervisor_id'
								AND REFERENCED_TABLE_NAME IS NOT NULL
								LIMIT 1
							`).Scan(&verifyFkName).Error
							
							if verifyErr == nil && verifyFkName != "" {
								log.Printf("Warning: Foreign key constraint still exists after drop attempt: %s", verifyFkName)
							}
						}
						
						// Re-enable foreign key checks
						if err := db.Exec(`SET FOREIGN_KEY_CHECKS = 1`).Error; err != nil {
							log.Printf("Warning: Failed to enable foreign key checks: %v", err)
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
					
					// Now modify the column type - this MUST succeed
					// Double-check that FK is gone before attempting MODIFY COLUMN
					var remainingFkName string
					checkErr := db.Raw(`
						SELECT CONSTRAINT_NAME
						FROM information_schema.KEY_COLUMN_USAGE
						WHERE TABLE_SCHEMA = DATABASE()
						AND TABLE_NAME = 'instances'
						AND COLUMN_NAME = 'hypervisor_id'
						AND REFERENCED_TABLE_NAME IS NOT NULL
						LIMIT 1
					`).Scan(&remainingFkName).Error
					
					if checkErr == nil && remainingFkName != "" {
						log.Printf("Warning: Foreign key constraint %s still exists, attempting final drop before MODIFY COLUMN", remainingFkName)
						if err := db.Exec(`SET FOREIGN_KEY_CHECKS = 0`).Error; err != nil {
							log.Printf("Warning: Failed to disable foreign key checks: %v", err)
						}
						// Final attempt to drop FK
						if err := db.Exec("ALTER TABLE instances DROP FOREIGN KEY `" + remainingFkName + "`").Error; err != nil {
							log.Printf("Error: Final attempt to drop FK failed: %v", err)
						}
					}
					
					// Disable foreign key checks before MODIFY COLUMN
					if err := db.Exec(`SET FOREIGN_KEY_CHECKS = 0`).Error; err != nil {
						log.Printf("Warning: Failed to disable foreign key checks: %v", err)
					}
					
					// Now modify the column type
					if err := db.Exec(`
						ALTER TABLE instances 
						MODIFY COLUMN hypervisor_id VARCHAR(255) NULL
					`).Error; err != nil {
						log.Printf("Error: Failed to modify hypervisor_id column type: %v", err)
						// Re-enable FK checks before returning error
						db.Exec(`SET FOREIGN_KEY_CHECKS = 1`)
						return fmt.Errorf("failed to modify hypervisor_id to VARCHAR(255): %w", err)
					}
					
					// Re-enable foreign key checks
					if err := db.Exec(`SET FOREIGN_KEY_CHECKS = 1`).Error; err != nil {
						log.Printf("Warning: Failed to enable foreign key checks: %v", err)
					}
					log.Println("Successfully fixed instances.hypervisor_id to VARCHAR(255)")
				} else {
					log.Println("instances.hypervisor_id is already VARCHAR(255) - no fix needed")
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

