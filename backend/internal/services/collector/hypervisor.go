package collector

import (
	"fmt"

	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/hypervisors"
	"github.com/network-collector/backend/internal/models"
	"github.com/network-collector/backend/internal/services/storage"
	"github.com/network-collector/backend/pkg/openstack"
)

// HypervisorCollector collects hypervisor data from OpenStack
type HypervisorCollector struct {
	client     *openstack.Client
	repository *storage.Repository
}

// NewHypervisorCollector creates a new hypervisor collector
func NewHypervisorCollector(client *openstack.Client, repository *storage.Repository) *HypervisorCollector {
	return &HypervisorCollector{
		client:     client,
		repository: repository,
	}
}

// CollectHypervisors collects all hypervisors from OpenStack
func (c *HypervisorCollector) CollectHypervisors() error {
	openstackHypervisors, err := c.client.ListHypervisors()
	if err != nil {
		return fmt.Errorf("failed to list hypervisors: %w", err)
	}

	var errors []error
	successCount := 0

	for _, hv := range openstackHypervisors {
		if err := c.saveHypervisor(hv); err != nil {
			errors = append(errors, fmt.Errorf("failed to save hypervisor %s: %w", hv.ID, err))
			continue
		}
		successCount++
	}

	if len(errors) == len(openstackHypervisors) {
		return fmt.Errorf("all hypervisors failed to collect: %d errors", len(errors))
	}

	if len(errors) > 0 {
		return fmt.Errorf("partial failure: %d succeeded, %d failed", successCount, len(errors))
	}

	return nil
}

// saveHypervisor saves a single hypervisor to database
func (c *HypervisorCollector) saveHypervisor(hv hypervisors.Hypervisor) error {
	dbHypervisor := &models.Hypervisor{
		OpenStackID:  hv.ID,
		Hostname:     hv.HypervisorHostname,
		HostIP:       hv.HostIP,
		Status:       hv.Status,
		State:        hv.State,
		VCPUsUsed:    hv.VCPUsUsed,
		VCPUsTotal:   hv.VCPUs,
		MemoryUsed:   int64(hv.MemoryMBUsed),
		MemoryTotal:  int64(hv.MemoryMB),
		LocalGBUsed:  int64(hv.LocalGBUsed),
		LocalGBTotal: int64(hv.LocalGB),
		RunningVMs:   hv.RunningVMs,
	}

	return c.repository.UpsertHypervisor(dbHypervisor)
}

