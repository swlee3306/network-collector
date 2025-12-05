package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/network-collector/backend/internal/services/storage"
)

// ListFlavors lists all flavors
func ListFlavors(repo *storage.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		flavors, err := repo.ListFlavors()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to list flavors",
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"data": flavors,
		})
	}
}

// GetFlavor gets a specific flavor by ID
func GetFlavor(repo *storage.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")

		flavor, err := repo.GetFlavorByID(id)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Flavor not found",
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"data": flavor,
		})
	}
}

