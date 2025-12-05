package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/network-collector/backend/internal/services/storage"
)

// GetInstanceMetrics gets metrics for a specific instance
func GetInstanceMetrics(repo *storage.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")

		// Parse query parameters for time range
		startTime := c.Query("start_time")
		endTime := c.Query("end_time")

		var start, end time.Time
		var err error

		if startTime != "" {
			start, err = time.Parse(time.RFC3339, startTime)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": "Invalid start_time format (use RFC3339)",
				})
				return
			}
		} else {
			// Default: last 24 hours
			start = time.Now().Add(-24 * time.Hour)
		}

		if endTime != "" {
			end, err = time.Parse(time.RFC3339, endTime)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": "Invalid end_time format (use RFC3339)",
				})
				return
			}
		} else {
			end = time.Now()
		}

		metrics, err := repo.GetInstanceMetrics(id, start, end)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to get instance metrics",
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"data": metrics,
		})
	}
}

// GetNetworkMetrics gets metrics for a specific network
func GetNetworkMetrics(repo *storage.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")

		// Parse query parameters for time range
		startTime := c.Query("start_time")
		endTime := c.Query("end_time")

		var start, end time.Time
		var err error

		if startTime != "" {
			start, err = time.Parse(time.RFC3339, startTime)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": "Invalid start_time format (use RFC3339)",
				})
				return
			}
		} else {
			start = time.Now().Add(-24 * time.Hour)
		}

		if endTime != "" {
			end, err = time.Parse(time.RFC3339, endTime)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": "Invalid end_time format (use RFC3339)",
				})
				return
			}
		} else {
			end = time.Now()
		}

		metrics, err := repo.GetNetworkMetrics(id, start, end)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to get network metrics",
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"data": metrics,
		})
	}
}

// GetHypervisorMetrics gets metrics for a specific hypervisor
func GetHypervisorMetrics(repo *storage.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")

		// Parse query parameters for time range
		startTime := c.Query("start_time")
		endTime := c.Query("end_time")

		var start, end time.Time
		var err error

		if startTime != "" {
			start, err = time.Parse(time.RFC3339, startTime)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": "Invalid start_time format (use RFC3339)",
				})
				return
			}
		} else {
			start = time.Now().Add(-24 * time.Hour)
		}

		if endTime != "" {
			end, err = time.Parse(time.RFC3339, endTime)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": "Invalid end_time format (use RFC3339)",
				})
				return
			}
		} else {
			end = time.Now()
		}

		metrics, err := repo.GetHypervisorMetrics(id, start, end)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to get hypervisor metrics",
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"data": metrics,
		})
	}
}

