package main

import (
	"flag"
	"log"
	"os"

	"github.com/network-collector/backend/internal/database"
	"github.com/network-collector/backend/internal/database/migrations"
)

func main() {
	var (
		host     = flag.String("host", "localhost", "Database host")
		port     = flag.Int("port", 3306, "Database port")
		user     = flag.String("user", "root", "Database user")
		password = flag.String("password", "", "Database password")
		dbName   = flag.String("db", "openstack_monitor", "Database name")
		action   = flag.String("action", "up", "Migration action: up or down")
	)
	flag.Parse()

	// Get password from environment if not provided
	if *password == "" {
		*password = os.Getenv("DB_PASSWORD")
	}

	config := database.Config{
		Host:     *host,
		Port:     *port,
		User:     *user,
		Password: *password,
		Database: *dbName,
	}

	db, err := database.Connect(config)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer database.Close()

	switch *action {
	case "up":
		if err := migrations.RunMigrations(db); err != nil {
			log.Fatalf("Migration failed: %v", err)
		}
		log.Println("Migrations completed successfully")
	case "down":
		if err := migrations.RollbackMigrations(db); err != nil {
			log.Fatalf("Rollback failed: %v", err)
		}
		log.Println("Migrations rolled back successfully")
	default:
		log.Fatalf("Unknown action: %s. Use 'up' or 'down'", *action)
	}
}

