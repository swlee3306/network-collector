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
	// Get server details from OpenStack
	_, err := c.client.GetServer(instance.OpenStackID)
	if err != nil {
		return fmt.Errorf("failed to get server: %w", err)
	}

	// Extract metrics from server status
	// Note: OpenStack Nova API doesn't provide detailed metrics directly
	// In production, you would use Ceilometer or Gnocchi for metrics
	// For now, we'll collect basic information from flavor and status

	var memoryTotalMB int64 = 0
	if instance.FlavorID != "" {
		flavor, err := c.repository.GetFlavorByID(instance.FlavorID)
		if err == nil && flavor != nil {
			memoryTotalMB = int64(flavor.RAM) // Flavor RAM is in MB
		}
	}

	metrics := &models.InstanceMetrics{
		InstanceID:      instance.ID,
		Timestamp:       time.Now(),
		CPUUsagePercent: 0, // Would be collected from Ceilometer/Gnocchi
		MemoryUsageMB:   0, // Would be collected from Ceilometer/Gnocchi
		MemoryTotalMB:   memoryTotalMB, // From flavor
		DiskReadBytes:   0, // Would be collected from Ceilometer/Gnocchi
		DiskWriteBytes:  0, // Would be collected from Ceilometer/Gnocchi
		NetworkRxBytes:  0, // Would be collected from Ceilometer/Gnocchi
		NetworkTxBytes:  0, // Would be collected from Ceilometer/Gnocchi
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

	// Get network with ports to count connected VMs
	networkWithPorts, err := c.repository.GetNetwork(network.ID)
	if err != nil {
		log.Printf("Failed to get network with ports: %v", err)
		networkWithPorts = &network
	}

	// Count connected VMs by checking ports with device_owner = "compute:nova"
	connectedVMsCount := 0
	vmInstanceIDs := make(map[string]bool)
	
	if networkWithPorts != nil {
		for _, port := range networkWithPorts.Ports {
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

