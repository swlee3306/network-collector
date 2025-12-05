package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/network-collector/backend/internal/metrics"
)

// Metrics returns a middleware that records Prometheus metrics
func Metrics() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.FullPath()
		if path == "" {
			path = c.Request.URL.Path
		}

		// Record request size
		requestSize := float64(c.Request.ContentLength)
		if requestSize < 0 {
			requestSize = 0
		}

		// Process request
		c.Next()

		// Calculate duration
		duration := time.Since(start).Seconds()

		// Record response size
		responseSize := float64(c.Writer.Size())
		if responseSize < 0 {
			responseSize = 0
		}

		// Record metrics
		statusCode := c.Writer.Status()
		metrics.RecordHTTPRequest(
			c.Request.Method,
			path,
			statusCode,
			duration,
			requestSize,
			responseSize,
		)
	}
}

// MetricsEndpoint returns a handler for the /metrics endpoint
func MetricsEndpoint() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Prometheus metrics endpoint will be handled by promhttp
		// This is just a placeholder - actual implementation will use promhttp.Handler()
		c.String(200, "# Metrics endpoint - use promhttp.Handler() in server setup")
	}
}

