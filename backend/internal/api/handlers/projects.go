package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/network-collector/backend/internal/services/storage"
)

// ListProjects lists all projects
func ListProjects(repo *storage.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		projects, err := repo.ListProjects()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to list projects",
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"data": projects,
		})
	}
}

// GetProject gets a specific project by ID
func GetProject(repo *storage.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")

		project, err := repo.GetProjectByID(id)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Project not found",
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"data": project,
		})
	}
}

