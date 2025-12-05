package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/network-collector/backend/internal/services/storage"
)

// ListNetworks lists all networks
func ListNetworks(repo *storage.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		networks, err := repo.ListNetworks()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to list networks",
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"data": networks,
		})
	}
}

// GetNetwork gets a specific network by ID
func GetNetwork(repo *storage.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")

		network, err := repo.GetNetworkByID(id)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Network not found",
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"data": network,
		})
	}
}

