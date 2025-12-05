package handlers

import (
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/network-collector/backend/internal/models"
	"github.com/network-collector/backend/internal/services/storage"
)

// ProjectComparison represents resource counts for a project
type ProjectComparison struct {
	ProjectID   string `json:"project_id"`
	ProjectName string `json:"project_name"`
	Instances   int    `json:"instances"`
	Networks    int    `json:"networks"`
	Volumes     int    `json:"volumes"`
	ActiveInstances int `json:"active_instances"`
}

// CompareProjects compares resource usage across multiple projects
func CompareProjects(repo *storage.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get project IDs from query parameter (comma-separated)
		projectIDsParam := c.Query("project_ids")
		if projectIDsParam == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "project_ids parameter is required (comma-separated)",
			})
			return
		}

		// Parse project IDs
		var projectIDs []string
		for _, id := range splitCommaSeparated(projectIDsParam) {
			if id != "" {
				projectIDs = append(projectIDs, id)
			}
		}

		if len(projectIDs) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "At least one project ID is required",
			})
			return
		}

		// Get comparison data for each project
		var comparisons []ProjectComparison
		for _, projectID := range projectIDs {
			project, err := repo.GetProjectByID(projectID)
			if err != nil {
				// Skip projects that don't exist
				continue
			}

			instances, err := repo.GetInstancesByProjectID(projectID)
			if err != nil {
				log.Printf("Failed to get instances for project %s: %v", projectID, err)
				instances = []models.Instance{} // Use empty slice on error
			}
			networks, err := repo.GetNetworksByProjectID(projectID)
			if err != nil {
				log.Printf("Failed to get networks for project %s: %v", projectID, err)
				networks = []models.Network{} // Use empty slice on error
			}
			volumes, err := repo.GetVolumesByProjectID(projectID)
			if err != nil {
				log.Printf("Failed to get volumes for project %s: %v", projectID, err)
				volumes = []models.Volume{} // Use empty slice on error
			}

			activeInstances := 0
			for _, instance := range instances {
				if instance.Status == "ACTIVE" {
					activeInstances++
				}
			}

			comparisons = append(comparisons, ProjectComparison{
				ProjectID:       projectID,
				ProjectName:     project.Name,
				Instances:       len(instances),
				Networks:        len(networks),
				Volumes:         len(volumes),
				ActiveInstances: activeInstances,
			})
		}

		c.JSON(http.StatusOK, gin.H{
			"data": comparisons,
		})
	}
}

// GetProjectResourceSummary gets resource summary for a single project
func GetProjectResourceSummary(repo *storage.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		projectID := c.Param("id")

		project, err := repo.GetProjectByID(projectID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Project not found",
			})
			return
		}

		instances, err := repo.GetInstancesByProjectID(projectID)
		if err != nil {
			log.Printf("Failed to get instances for project %s: %v", projectID, err)
			instances = []models.Instance{} // Use empty slice on error
		}
		networks, err := repo.GetNetworksByProjectID(projectID)
		if err != nil {
			log.Printf("Failed to get networks for project %s: %v", projectID, err)
			networks = []models.Network{} // Use empty slice on error
		}
		volumes, err := repo.GetVolumesByProjectID(projectID)
		if err != nil {
			log.Printf("Failed to get volumes for project %s: %v", projectID, err)
			volumes = []models.Volume{} // Use empty slice on error
		}

		activeInstances := 0
		for _, instance := range instances {
			if instance.Status == "ACTIVE" {
				activeInstances++
			}
		}

		summary := ProjectComparison{
			ProjectID:       projectID,
			ProjectName:     project.Name,
			Instances:       len(instances),
			Networks:        len(networks),
			Volumes:         len(volumes),
			ActiveInstances: activeInstances,
		}

		c.JSON(http.StatusOK, gin.H{
			"data": summary,
		})
	}
}

// Helper function to split comma-separated string
func splitCommaSeparated(s string) []string {
	var result []string
	var current string
	for _, char := range s {
		if char == ',' {
			// Trim whitespace and only add if not empty after trimming
			trimmed := strings.TrimSpace(current)
			if trimmed != "" {
				result = append(result, trimmed)
			}
			current = ""
		} else {
			current += string(char)
		}
	}
	// Trim whitespace and only add if not empty after trimming
	trimmed := strings.TrimSpace(current)
	if trimmed != "" {
		result = append(result, trimmed)
	}
	return result
}

