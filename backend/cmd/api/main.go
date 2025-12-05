package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/network-collector/backend/internal/api"
	"github.com/network-collector/backend/internal/config"
	"github.com/network-collector/backend/internal/database"
	"github.com/network-collector/backend/internal/database/migrations"
	"github.com/network-collector/backend/internal/services/events"
)

func main() {
	log.Println("OpenStack Monitoring API Service starting...")

	// Load configuration
	cfg := config.Load()

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
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// Run migrations
	if err := migrations.RunMigrations(db); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	// Initialize event broadcaster
	broadcaster := events.NewBroadcaster()

	// Initialize API server
	server, err := api.NewServer(cfg, db, broadcaster)
	if err != nil {
		log.Fatalf("Failed to create API server: %v", err)
	}

	// Start server in a goroutine
	go func() {
		if err := server.Start(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	log.Printf("API server started on port %d", cfg.Server.Port)

	// Graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Println("OpenStack Monitoring API Service shutting down...")

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Stop(ctx); err != nil {
		log.Printf("Error during server shutdown: %v", err)
	}

	// Close database connection after server shutdown
	if err := database.Close(); err != nil {
		log.Printf("Error closing database connection: %v", err)
	}

	log.Println("API Service shutdown complete")
}

