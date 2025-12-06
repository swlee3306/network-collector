package collector

import (
	"fmt"

	"github.com/gophercloud/gophercloud/openstack/compute/v2/servers"
	"github.com/network-collector/backend/internal/models"
	"github.com/network-collector/backend/internal/services/storage"
	"github.com/network-collector/backend/pkg/openstack"
)

// InstanceCollector collects instance data from OpenStack
type InstanceCollector struct {
	client     *openstack.Client
	repository *storage.Repository
}

// NewInstanceCollector creates a new instance collector
func NewInstanceCollector(client *openstack.Client, repository *storage.Repository) *InstanceCollector {
	return &InstanceCollector{
		client:     client,
		repository: repository,
	}
}

// CollectInstances collects all instances from OpenStack
func (c *InstanceCollector) CollectInstances() error {
	openstackServers, err := c.client.ListServers()
	if err != nil {
		return fmt.Errorf("failed to list servers: %w", err)
	}

	var errors []error
	successCount := 0

	for _, server := range openstackServers {
		if err := c.saveInstance(server); err != nil {
			errors = append(errors, fmt.Errorf("failed to save instance %s: %w", server.ID, err))
			continue
		}
		successCount++
	}

	// Return error if all failed, otherwise return partial error info
	if len(errors) == len(openstackServers) {
		return fmt.Errorf("all instances failed to collect: %d errors", len(errors))
	}

	if len(errors) > 0 {
		// Partial failure - log but don't fail completely
		// This aligns with FR-012: partial failure shows partial data
		return fmt.Errorf("partial failure: %d succeeded, %d failed", successCount, len(errors))
	}

	return nil
}

// saveInstance saves a single instance to database
func (c *InstanceCollector) saveInstance(server servers.Server) error {
	// Get project - must exist (collected first)
	var projectID string
	if server.TenantID != "" {
		project, err := c.repository.GetProjectByOpenStackID(server.TenantID)
		if err != nil {
			// Project not found - leave empty to maintain foreign key integrity
			// The instance can be updated later when the project is collected
			projectID = ""
		} else {
			projectID = project.ID
		}
	}

	// Get flavor - must exist (collected first)
	var flavorID string
	if flavorIDStr, ok := server.Flavor["id"].(string); ok {
		flavor, err := c.repository.GetFlavorByOpenStackID(flavorIDStr)
		if err != nil {
			// Flavor not found - leave empty to maintain foreign key integrity
			// The instance can be updated later when the flavor is collected
			flavorID = ""
		} else {
			flavorID = flavor.ID
		}
	}

	// Get hypervisor ID from server details
	// server.HostID is a hash (56 chars), which doesn't directly map to hypervisor OpenStack ID
	// We need to find hypervisor by hostname or other means
	// For now, we'll leave it empty and it will be updated later when topology is built
	// The hypervisor relationship can be established through the topology analyzer
	var hypervisorID string
	// Note: server.HostID is not the same as hypervisor OpenStack ID
	// We'll leave hypervisorID empty for now and let it be populated later
	// when we have better hypervisor matching logic

	instance := &models.Instance{
		OpenStackID:  server.ID,
		Name:         server.Name,
		Status:       server.Status,
		ProjectID:    projectID,
		FlavorID:     flavorID,
		HypervisorID: hypervisorID,
		CreatedAt:    server.Created,
		UpdatedAt:    server.Updated,
	}

	return c.repository.UpsertInstance(instance)
}

