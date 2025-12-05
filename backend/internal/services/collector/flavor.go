package collector

import (
	"fmt"

	"github.com/gophercloud/gophercloud/openstack/compute/v2/flavors"
	"github.com/network-collector/backend/internal/models"
	"github.com/network-collector/backend/internal/services/storage"
	"github.com/network-collector/backend/pkg/openstack"
)

// FlavorCollector collects flavor data from OpenStack
type FlavorCollector struct {
	client     *openstack.Client
	repository *storage.Repository
}

// NewFlavorCollector creates a new flavor collector
func NewFlavorCollector(client *openstack.Client, repository *storage.Repository) *FlavorCollector {
	return &FlavorCollector{
		client:     client,
		repository: repository,
	}
}

// CollectFlavors collects all flavors from OpenStack
func (c *FlavorCollector) CollectFlavors() error {
	openstackFlavors, err := c.client.ListFlavors()
	if err != nil {
		return fmt.Errorf("failed to list flavors: %w", err)
	}

	var errors []error
	successCount := 0

	for _, flavor := range openstackFlavors {
		if err := c.saveFlavor(flavor); err != nil {
			errors = append(errors, fmt.Errorf("failed to save flavor %s: %w", flavor.ID, err))
			continue
		}
		successCount++
	}

	if len(errors) == len(openstackFlavors) {
		return fmt.Errorf("all flavors failed to collect: %d errors", len(errors))
	}

	if len(errors) > 0 {
		return fmt.Errorf("partial failure: %d succeeded, %d failed", successCount, len(errors))
	}

	return nil
}

// saveFlavor saves a single flavor to database
func (c *FlavorCollector) saveFlavor(flavor flavors.Flavor) error {
	dbFlavor := &models.Flavor{
		OpenStackID: flavor.ID,
		Name:        flavor.Name,
		VCPUs:       flavor.VCPUs,
		RAM:         flavor.RAM,
		Disk:        flavor.Disk,
		Ephemeral:   flavor.Ephemeral,
		Swap:        flavor.Swap,
		IsPublic:    flavor.IsPublic,
	}

	return c.repository.UpsertFlavor(dbFlavor)
}

