package api

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/network-collector/backend/internal/api/handlers"
	"github.com/network-collector/backend/internal/api/middleware"
	"github.com/network-collector/backend/internal/config"
	"github.com/network-collector/backend/internal/services/events"
	"github.com/network-collector/backend/internal/services/storage"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"gorm.io/gorm"
)

// Server represents the API server
type Server struct {
	router     *gin.Engine
	config     *config.Config
	repository *storage.Repository
	httpServer *http.Server
}

// NewServer creates a new API server
func NewServer(cfg *config.Config, db *gorm.DB, broadcaster *events.Broadcaster) (*Server, error) {
	// Set Gin mode
	if cfg.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	} else {
		gin.SetMode(gin.DebugMode)
	}

	router := gin.New()

	// Initialize repository
	repository := storage.NewRepository(db)

	server := &Server{
		router:     router,
		config:     cfg,
		repository: repository,
	}

	// Setup middleware
	server.setupMiddleware()

	// Set event broadcaster for handlers
	handlers.SetEventBroadcaster(broadcaster)

	// Setup routes
	server.setupRoutes()

	return server, nil
}

// setupMiddleware configures middleware
func (s *Server) setupMiddleware() {
	// Recovery middleware (panic recovery)
	s.router.Use(gin.Recovery())

	// Metrics middleware (should be early to capture all requests)
	s.router.Use(middleware.Metrics())

	// Logging middleware
	s.router.Use(middleware.Logger())

	// CORS middleware
	s.router.Use(middleware.CORS())

	// Request ID middleware
	s.router.Use(middleware.RequestID())

	// Error handler middleware (must be last)
	s.router.Use(middleware.ErrorHandler())
}

// setupRoutes configures API routes
func (s *Server) setupRoutes() {
	// Health check endpoint (no auth required)
	s.router.GET("/health", handlers.HealthCheck)

	// Prometheus metrics endpoint (no auth required)
	s.router.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// API v1 routes
	v1 := s.router.Group("/api/v1")
	{
		// Authentication
		v1.POST("/auth/login", handlers.Login(s.config))

		// Protected routes (require authentication)
		protected := v1.Group("")
		protected.Use(middleware.Auth(s.config))
		{
			// Resource endpoints
			protected.GET("/instances", handlers.ListInstances(s.repository))
			protected.GET("/instances/:id", handlers.GetInstance(s.repository))
			protected.GET("/projects", handlers.ListProjects(s.repository))
			// Register specific routes before parameterized routes to avoid route conflicts
			protected.GET("/projects/compare", handlers.CompareProjects(s.repository))
			protected.GET("/projects/:id/summary", handlers.GetProjectResourceSummary(s.repository))
			protected.GET("/projects/:id", handlers.GetProject(s.repository))
			protected.GET("/networks", handlers.ListNetworks(s.repository))
			protected.GET("/networks/:id", handlers.GetNetwork(s.repository))
			protected.GET("/hypervisors", handlers.ListHypervisors(s.repository))
			protected.GET("/hypervisors/:id", handlers.GetHypervisor(s.repository))
			protected.GET("/flavors", handlers.ListFlavors(s.repository))
			protected.GET("/flavors/:id", handlers.GetFlavor(s.repository))
			protected.GET("/volumes", handlers.ListVolumes(s.repository))
			protected.GET("/volumes/:id", handlers.GetVolume(s.repository))

			// Topology endpoints
			protected.GET("/topology/instances/:id", handlers.GetInstanceTopology(s.repository))
			protected.GET("/topology/hosts/:id", handlers.GetHostTopology(s.repository))
			protected.GET("/topology/networks/:id", handlers.GetNetworkTopology(s.repository))

			// Metrics endpoints
			protected.GET("/metrics/instances/:id", handlers.GetInstanceMetrics(s.repository))
			protected.GET("/metrics/networks/:id", handlers.GetNetworkMetrics(s.repository))
			protected.GET("/metrics/hypervisors/:id", handlers.GetHypervisorMetrics(s.repository))

			// Real-time events (SSE)
			protected.GET("/events/stream", handlers.StreamEvents(s.repository))
		}
	}
}

// Start starts the HTTP server
func (s *Server) Start() error {
	addr := fmt.Sprintf(":%d", s.config.Server.Port)
	log.Printf("Starting API server on %s", addr)

	s.httpServer = &http.Server{
		Addr:         addr,
		Handler:      s.router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	return s.httpServer.ListenAndServe()
}

// Stop gracefully stops the server
func (s *Server) Stop(ctx context.Context) error {
	log.Println("Stopping API server...")
	if s.httpServer == nil {
		return nil
	}
	return s.httpServer.Shutdown(ctx)
}

// Router returns the Gin router (for testing purposes)
func (s *Server) Router() *gin.Engine {
	return s.router
}

