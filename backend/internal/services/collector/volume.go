package collector

import (
	"fmt"

	"github.com/gophercloud/gophercloud/openstack/blockstorage/v3/volumes"
	"github.com/network-collector/backend/internal/models"
	"github.com/network-collector/backend/internal/services/storage"
	"github.com/network-collector/backend/pkg/openstack"
)

// VolumeCollector collects volume data from OpenStack
type VolumeCollector struct {
	client     *openstack.Client
	repository *storage.Repository
}

// NewVolumeCollector creates a new volume collector
func NewVolumeCollector(client *openstack.Client, repository *storage.Repository) *VolumeCollector {
	return &VolumeCollector{
		client:     client,
		repository: repository,
	}
}

// CollectVolumes collects all volumes from OpenStack
func (c *VolumeCollector) CollectVolumes() error {
	openstackVolumes, err := c.client.ListVolumes()
	if err != nil {
		return fmt.Errorf("failed to list volumes: %w", err)
	}

	var errors []error
	successCount := 0

	for _, volume := range openstackVolumes {
		if err := c.saveVolume(volume); err != nil {
			errors = append(errors, fmt.Errorf("failed to save volume %s: %w", volume.ID, err))
			continue
		}
		successCount++
	}

	if len(errors) == len(openstackVolumes) {
		return fmt.Errorf("all volumes failed to collect: %d errors", len(errors))
	}

	if len(errors) > 0 {
		return fmt.Errorf("partial failure: %d succeeded, %d failed", successCount, len(errors))
	}

	return nil
}

// saveVolume saves a single volume to database
func (c *VolumeCollector) saveVolume(volume volumes.Volume) error {
	// Get project - volumes.Volume doesn't have TenantID, use UserID or Owner if available
	// For now, we'll leave projectID empty and it can be populated later
	var projectID string
	// Note: gophercloud volumes.Volume doesn't expose TenantID directly
	// In production, you might need to use volume metadata or other fields

	// Get attached instance if exists
	// Convert OpenStack ServerID (instance OpenStack ID) to internal Instance ID
	// Volume.AttachedTo should store the internal database Instance ID for foreign key relationship
	var instanceID string
	if len(volume.Attachments) > 0 {
		attachment := volume.Attachments[0]
		if attachment.ServerID != "" {
			instance, err := c.repository.GetInstanceByOpenStackID(attachment.ServerID)
			if err != nil {
				// Instance not found - leave empty to maintain foreign key integrity
				// The volume can be updated later when the instance is collected
				// Storing OpenStack ID here would break the foreign key relationship
				instanceID = ""
			} else {
				instanceID = instance.ID
			}
		}
	}

	dbVolume := &models.Volume{
		OpenStackID: volume.ID,
		Name:        volume.Name,
		Status:      volume.Status,
		Size:        volume.Size,
		VolumeType:  volume.VolumeType,
		ProjectID:   projectID,
		AttachedTo:  instanceID,
		CreatedAt:   volume.CreatedAt,
		UpdatedAt:   volume.UpdatedAt,
	}

	return c.repository.UpsertVolume(dbVolume)
}

