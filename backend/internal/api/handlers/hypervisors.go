package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/network-collector/backend/internal/services/storage"
)

// ListHypervisors lists all hypervisors
func ListHypervisors(repo *storage.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		hypervisors, err := repo.ListHypervisors()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to list hypervisors",
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"data": hypervisors,
		})
	}
}

// GetHypervisor gets a specific hypervisor by ID
func GetHypervisor(repo *storage.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")

		hypervisor, err := repo.GetHypervisorByID(id)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Hypervisor not found",
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"data": hypervisor,
		})
	}
}

