package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/network-collector/backend/internal/services/storage"
)

// ListVolumes lists all volumes
func ListVolumes(repo *storage.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		volumes, err := repo.ListVolumes()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to list volumes",
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"data": volumes,
		})
	}
}

// GetVolume gets a specific volume by ID
func GetVolume(repo *storage.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")

		volume, err := repo.GetVolumeByID(id)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Volume not found",
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"data": volume,
		})
	}
}

