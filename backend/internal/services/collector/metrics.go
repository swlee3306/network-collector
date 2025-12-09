package collector

import (
	"fmt"
	"log"
	"time"

	"github.com/network-collector/backend/internal/models"
	"github.com/network-collector/backend/internal/services/storage"
	"github.com/network-collector/backend/pkg/openstack"
)

// MetricsCollector collects metrics from OpenStack
type MetricsCollector struct {
	client     *openstack.Client
	repository *storage.Repository
}

// NewMetricsCollector creates a new metrics collector
func NewMetricsCollector(client *openstack.Client, repository *storage.Repository) *MetricsCollector {
	return &MetricsCollector{
		client:     client,
		repository: repository,
	}
}

// CollectInstanceMetrics collects metrics for all instances
func (c *MetricsCollector) CollectInstanceMetrics() error {
	instances, err := c.repository.ListInstances()
	if err != nil {
		return fmt.Errorf("failed to list instances: %w", err)
	}

	var errors []error
	successCount := 0

	for _, instance := range instances {
		if err := c.collectInstanceMetric(instance); err != nil {
			log.Printf("Failed to collect metrics for instance %s: %v", instance.ID, err)
			errors = append(errors, err)
			continue
		}
		successCount++
	}

	if len(errors) == len(instances) {
		return fmt.Errorf("all instance metrics failed to collect: %d errors", len(errors))
	}

	if len(errors) > 0 {
		return fmt.Errorf("partial failure: %d succeeded, %d failed", successCount, len(errors))
	}

	return nil
}

// collectInstanceMetric collects metrics for a single instance
func (c *MetricsCollector) collectInstanceMetric(instance models.Instance) error {
	// Get memory total from flavor (fallback if diagnostics doesn't provide it)
	var memoryTotalMB int64 = 0
	if instance.FlavorID != "" {
		flavor, err := c.repository.GetFlavorByID(instance.FlavorID)
		if err == nil && flavor != nil {
			memoryTotalMB = int64(flavor.RAM) // Flavor RAM is in MB
		}
	}

	// Try to get diagnostics from OpenStack Nova API
	// Diagnostics provides CPU, memory, disk, and network metrics
	diagnostics, err := c.client.GetServerDiagnostics(instance.OpenStackID)
	if err != nil {
		// If diagnostics fails (e.g., insufficient permissions or server not running),
		// log the error but continue with default values
		log.Printf("Warning: Failed to get diagnostics for instance %s (%s): %v. Using default values.", 
			instance.ID, instance.OpenStackID, err)
		
		// Use default values (0) but keep memory total from flavor
		metrics := &models.InstanceMetrics{
			InstanceID:      instance.ID,
			Timestamp:       time.Now(),
			CPUUsagePercent: 0,
			MemoryUsageMB:   0,
			MemoryTotalMB:   memoryTotalMB,
			DiskReadBytes:   0,
			DiskWriteBytes:  0,
			NetworkRxBytes:  0,
			NetworkTxBytes:  0,
		}
		return c.repository.SaveInstanceMetrics(metrics)
	}

	// Use diagnostics data, but fallback to flavor memory if diagnostics doesn't provide it
	if diagnostics.MemoryTotalMB == 0 && memoryTotalMB > 0 {
		diagnostics.MemoryTotalMB = memoryTotalMB
	}

	metrics := &models.InstanceMetrics{
		InstanceID:      instance.ID,
		Timestamp:       time.Now(),
		CPUUsagePercent: diagnostics.CPUUsagePercent,
		MemoryUsageMB:   diagnostics.MemoryUsageMB,
		MemoryTotalMB:   diagnostics.MemoryTotalMB,
		DiskReadBytes:   diagnostics.DiskReadBytes,
		DiskWriteBytes:  diagnostics.DiskWriteBytes,
		NetworkRxBytes:  diagnostics.NetworkRxBytes,
		NetworkTxBytes:  diagnostics.NetworkTxBytes,
	}

	return c.repository.SaveInstanceMetrics(metrics)
}

// CollectNetworkMetrics collects metrics for all networks
func (c *MetricsCollector) CollectNetworkMetrics() error {
	networks, err := c.repository.ListNetworks()
	if err != nil {
		return fmt.Errorf("failed to list networks: %w", err)
	}

	var errors []error
	successCount := 0

	for _, network := range networks {
		if err := c.collectNetworkMetric(network); err != nil {
			log.Printf("Failed to collect metrics for network %s: %v", network.ID, err)
			errors = append(errors, err)
			continue
		}
		successCount++
	}

	if len(errors) == len(networks) {
		return fmt.Errorf("all network metrics failed to collect: %d errors", len(errors))
	}

	if len(errors) > 0 {
		return fmt.Errorf("partial failure: %d succeeded, %d failed", successCount, len(errors))
	}

	return nil
}

// collectNetworkMetric collects metrics for a single network
func (c *MetricsCollector) collectNetworkMetric(network models.Network) error {
	// Network metrics would typically come from Neutron agents or monitoring tools
	// For now, we'll calculate basic metrics from available data

	// Get ports for this network
	ports, err := c.repository.GetPortsByNetworkID(network.ID)
	if err != nil {
		log.Printf("Failed to get ports for network %s: %v", network.ID, err)
		ports = []models.Port{}
	}

	// Count connected VMs by checking ports with device_owner = "compute:nova"
	connectedVMsCount := 0
	vmInstanceIDs := make(map[string]bool)
	
	for _, port := range ports {
		// Ports with device_owner starting with "compute:" are VM interfaces
		if port.DeviceOwner != "" && (port.DeviceOwner == "compute:nova" || 
			(len(port.DeviceOwner) > 8 && port.DeviceOwner[:8] == "compute:")) {
			// Try to find instance by device_id (could be OpenStack ID or internal ID)
			if port.DeviceID != nil && *port.DeviceID != "" {
				// Check if we've already counted this instance
				if !vmInstanceIDs[*port.DeviceID] {
					// Try to find instance by OpenStack ID first
					instance, err := c.repository.GetInstanceByOpenStackID(*port.DeviceID)
					if err != nil {
						// Try by internal ID
						instance, err = c.repository.GetInstanceByID(*port.DeviceID)
					}
					if err == nil && instance != nil {
						vmInstanceIDs[instance.ID] = true
						connectedVMsCount++
					}
				}
			}
		}
	}

	metrics := &models.NetworkMetrics{
		NetworkID:         network.ID,
		Timestamp:         time.Now(),
		TotalBytes:        0, // Would be collected from monitoring tools
		PacketsDropped:    0, // Would be collected from monitoring tools
		ConnectedVMsCount: connectedVMsCount, // Calculated from connected instances
	}

	return c.repository.SaveNetworkMetrics(metrics)
}

// CollectHypervisorMetrics collects metrics for all hypervisors
func (c *MetricsCollector) CollectHypervisorMetrics() error {
	hypervisors, err := c.repository.ListHypervisors()
	if err != nil {
		return fmt.Errorf("failed to list hypervisors: %w", err)
	}

	var errors []error
	successCount := 0

	for _, hypervisor := range hypervisors {
		if err := c.collectHypervisorMetric(hypervisor); err != nil {
			log.Printf("Failed to collect metrics for hypervisor %s: %v", hypervisor.ID, err)
			errors = append(errors, err)
			continue
		}
		successCount++
	}

	if len(errors) == len(hypervisors) {
		return fmt.Errorf("all hypervisor metrics failed to collect: %d errors", len(errors))
	}

	if len(errors) > 0 {
		return fmt.Errorf("partial failure: %d succeeded, %d failed", successCount, len(errors))
	}

	return nil
}

// collectHypervisorMetric collects metrics for a single hypervisor
func (c *MetricsCollector) collectHypervisorMetric(hypervisor models.Hypervisor) error {
	// Hypervisor metrics are already collected in the hypervisor resource
	// We can use the existing data to create metrics entries

	metrics := &models.HypervisorMetrics{
		HypervisorID:    hypervisor.ID,
		Timestamp:       time.Now(),
		VCPUsUsed:       hypervisor.VCPUsUsed,
		VCPUsTotal:      hypervisor.VCPUsTotal,
		MemoryUsedMB:    hypervisor.MemoryUsed,
		MemoryTotalMB:   hypervisor.MemoryTotal,
		RunningVMsCount: hypervisor.RunningVMs,
	}

	return c.repository.SaveHypervisorMetrics(metrics)
}

// CollectAllMetrics collects all metrics
func (c *MetricsCollector) CollectAllMetrics() error {
	log.Println("Collecting instance metrics...")
	if err := c.CollectInstanceMetrics(); err != nil {
		log.Printf("Instance metrics collection error: %v", err)
	}

	log.Println("Collecting network metrics...")
	if err := c.CollectNetworkMetrics(); err != nil {
		log.Printf("Network metrics collection error: %v", err)
	}

	log.Println("Collecting hypervisor metrics...")
	if err := c.CollectHypervisorMetrics(); err != nil {
		log.Printf("Hypervisor metrics collection error: %v", err)
	}

	return nil
}

