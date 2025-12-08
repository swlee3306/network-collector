package collector

import (
	"fmt"

	"github.com/gophercloud/gophercloud/openstack/identity/v3/projects"
	"github.com/network-collector/backend/internal/models"
	"github.com/network-collector/backend/internal/services/storage"
	"github.com/network-collector/backend/pkg/openstack"
)

// ProjectCollector collects project data from OpenStack
type ProjectCollector struct {
	client     *openstack.Client
	repository *storage.Repository
}

// NewProjectCollector creates a new project collector
func NewProjectCollector(client *openstack.Client, repository *storage.Repository) *ProjectCollector {
	return &ProjectCollector{
		client:     client,
		repository: repository,
	}
}

// CollectProjects collects all projects from OpenStack
func (c *ProjectCollector) CollectProjects() error {
	openstackProjects, err := c.client.ListProjects()
	if err != nil {
		return fmt.Errorf("failed to list projects: %w", err)
	}

	var errors []error
	successCount := 0
	openstackIDs := make([]string, 0, len(openstackProjects))

	for _, project := range openstackProjects {
		openstackIDs = append(openstackIDs, project.ID)
		if err := c.saveProject(project); err != nil {
			errors = append(errors, fmt.Errorf("failed to save project %s: %w", project.ID, err))
			continue
		}
		successCount++
	}

	// Delete projects that no longer exist in OpenStack
	if err := c.repository.DeleteProjectsNotIn(openstackIDs); err != nil {
		errors = append(errors, fmt.Errorf("failed to delete removed projects: %w", err))
	}

	if len(errors) == len(openstackProjects) {
		return fmt.Errorf("all projects failed to collect: %d errors", len(errors))
	}

	if len(errors) > 0 {
		return fmt.Errorf("partial failure: %d succeeded, %d failed", successCount, len(errors))
	}

	return nil
}

// saveProject saves a single project to database
func (c *ProjectCollector) saveProject(project projects.Project) error {
	dbProject := &models.Project{
		OpenStackID: project.ID,
		Name:        project.Name,
		Description: project.Description,
	}

	return c.repository.UpsertProject(dbProject)
}

