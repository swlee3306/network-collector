package collector

import (
	"fmt"
	"log"
	"time"

	"github.com/network-collector/backend/internal/config"
	"github.com/network-collector/backend/internal/metrics"
	"github.com/network-collector/backend/internal/services/storage"
	"github.com/network-collector/backend/internal/services/topology"
	"github.com/network-collector/backend/pkg/openstack"
)

// OpenStackCollector coordinates collection from all OpenStack services
type OpenStackCollector struct {
	client             *openstack.Client
	repository         *storage.Repository
	instanceCollector  *InstanceCollector
	networkCollector   *NetworkCollector
	hypervisorCollector *HypervisorCollector
	projectCollector   *ProjectCollector
	flavorCollector    *FlavorCollector
	volumeCollector    *VolumeCollector
	metricsCollector   *MetricsCollector
	topologyAnalyzer   *topology.Analyzer
}

// NewOpenStackCollector creates a new OpenStack collector
func NewOpenStackCollector(cfg *config.Config, repository *storage.Repository) (*OpenStackCollector, error) {
	openstackConfig := openstack.Config{
		AuthURL:    cfg.OpenStack.AuthURL,
		Username:   cfg.OpenStack.Username,
		Password:   cfg.OpenStack.Password,
		ProjectID:  cfg.OpenStack.ProjectID,
		DomainName: cfg.OpenStack.DomainName,
	}

	client, err := openstack.NewClient(openstackConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create OpenStack client: %w", err)
	}

	return &OpenStackCollector{
		client:             client,
		repository:         repository,
		instanceCollector:  NewInstanceCollector(client, repository),
		networkCollector:   NewNetworkCollector(client, repository),
		hypervisorCollector: NewHypervisorCollector(client, repository),
		projectCollector:   NewProjectCollector(client, repository),
		flavorCollector:    NewFlavorCollector(client, repository),
		volumeCollector:    NewVolumeCollector(client, repository),
		metricsCollector:   NewMetricsCollector(client, repository),
		topologyAnalyzer:   topology.NewAnalyzer(repository),
	}, nil
}

// CollectAll collects all basic resources (instances, networks, hypervisors)
// Collection order is important for foreign key relationships:
// 1. Projects (no dependencies)
// 2. Flavors (no dependencies)
// 3. Hypervisors (no dependencies)
// 4. Instances (depends on Projects, Flavors, Hypervisors)
// 5. Networks (depends on Projects)
// 6. Ports (depends on Networks, Instances)
// 7. Routers (no dependencies)
// 8. Volumes (depends on Projects, Instances)
func (c *OpenStackCollector) CollectAll() error {
	log.Println("Starting OpenStack resource collection...")
	startTime := time.Now()

	var errors []error
	partialFailures := 0

	// 1. Collect projects first (no dependencies)
	log.Println("Collecting projects...")
	projectStart := time.Now()
	if err := c.projectCollector.CollectProjects(); err != nil {
		log.Printf("Project collection error: %v", err)
		errors = append(errors, fmt.Errorf("projects: %w", err))
		partialFailures++
		metrics.RecordCollection("projects", false, time.Since(projectStart).Seconds())
		metrics.RecordCollectionError("projects", "collection_failed")
	} else {
		log.Println("Projects collected successfully")
		metrics.RecordCollection("projects", true, time.Since(projectStart).Seconds())
	}

	// 2. Collect flavors (no dependencies, needed by instances)
	log.Println("Collecting flavors...")
	flavorStart := time.Now()
	if err := c.flavorCollector.CollectFlavors(); err != nil {
		log.Printf("Flavor collection error: %v", err)
		errors = append(errors, fmt.Errorf("flavors: %w", err))
		partialFailures++
		metrics.RecordCollection("flavors", false, time.Since(flavorStart).Seconds())
		metrics.RecordCollectionError("flavors", "collection_failed")
	} else {
		log.Println("Flavors collected successfully")
		metrics.RecordCollection("flavors", true, time.Since(flavorStart).Seconds())
	}

	// 3. Collect hypervisors (no dependencies, needed by instances)
	log.Println("Collecting hypervisors...")
	hypervisorStart := time.Now()
	if err := c.hypervisorCollector.CollectHypervisors(); err != nil {
		log.Printf("Hypervisor collection error: %v", err)
		errors = append(errors, fmt.Errorf("hypervisors: %w", err))
		partialFailures++
		metrics.RecordCollection("hypervisors", false, time.Since(hypervisorStart).Seconds())
		metrics.RecordCollectionError("hypervisors", "collection_failed")
	} else {
		log.Println("Hypervisors collected successfully")
		metrics.RecordCollection("hypervisors", true, time.Since(hypervisorStart).Seconds())
	}

	// 4. Collect instances (depends on projects, flavors, hypervisors)
	log.Println("Collecting instances...")
	instanceStart := time.Now()
	if err := c.instanceCollector.CollectInstances(); err != nil {
		log.Printf("Instance collection error: %v", err)
		errors = append(errors, fmt.Errorf("instances: %w", err))
		partialFailures++
		metrics.RecordCollection("instances", false, time.Since(instanceStart).Seconds())
		metrics.RecordCollectionError("instances", "collection_failed")
	} else {
		log.Println("Instances collected successfully")
		metrics.RecordCollection("instances", true, time.Since(instanceStart).Seconds())
	}

	// 5. Collect networks (depends on projects)
	log.Println("Collecting networks...")
	networkStart := time.Now()
	if err := c.networkCollector.CollectNetworks(); err != nil {
		log.Printf("Network collection error: %v", err)
		errors = append(errors, fmt.Errorf("networks: %w", err))
		partialFailures++
		metrics.RecordCollection("networks", false, time.Since(networkStart).Seconds())
		metrics.RecordCollectionError("networks", "collection_failed")
	} else {
		log.Println("Networks collected successfully")
		metrics.RecordCollection("networks", true, time.Since(networkStart).Seconds())
	}

	// 6. Collect ports (depends on networks, instances)
	log.Println("Collecting ports...")
	portStart := time.Now()
	if err := c.networkCollector.CollectPorts(); err != nil {
		log.Printf("Port collection error: %v", err)
		errors = append(errors, fmt.Errorf("ports: %w", err))
		partialFailures++
		metrics.RecordCollection("ports", false, time.Since(portStart).Seconds())
		metrics.RecordCollectionError("ports", "collection_failed")
	} else {
		log.Println("Ports collected successfully")
		metrics.RecordCollection("ports", true, time.Since(portStart).Seconds())
	}

	// 7. Collect routers (no dependencies)
	log.Println("Collecting routers...")
	routerStart := time.Now()
	if err := c.networkCollector.CollectRouters(); err != nil {
		log.Printf("Router collection error: %v", err)
		errors = append(errors, fmt.Errorf("routers: %w", err))
		partialFailures++
		metrics.RecordCollection("routers", false, time.Since(routerStart).Seconds())
		metrics.RecordCollectionError("routers", "collection_failed")
	} else {
		log.Println("Routers collected successfully")
		metrics.RecordCollection("routers", true, time.Since(routerStart).Seconds())
	}

	// Collect volumes
	log.Println("Collecting volumes...")
	volumeStart := time.Now()
	if err := c.volumeCollector.CollectVolumes(); err != nil {
		log.Printf("Volume collection error: %v", err)
		errors = append(errors, fmt.Errorf("volumes: %w", err))
		partialFailures++
		metrics.RecordCollection("volumes", false, time.Since(volumeStart).Seconds())
		metrics.RecordCollectionError("volumes", "collection_failed")
	} else {
		log.Println("Volumes collected successfully")
		metrics.RecordCollection("volumes", true, time.Since(volumeStart).Seconds())
	}

	// Determine if this is a complete failure or partial failure
	// Total: instances, networks, ports, routers, hypervisors, projects, flavors, volumes = 8
	if len(errors) == 8 {
		// All collections failed - complete failure
		return fmt.Errorf("complete collection failure: all resources failed to collect")
	}

	if len(errors) > 0 {
		// Partial failure - some succeeded, some failed
		// This aligns with FR-012: partial failure shows partial data
		log.Printf("Partial collection failure: %d resources failed, %d succeeded", len(errors), 8-len(errors))
		return fmt.Errorf("partial collection failure: %d errors", len(errors))
	}

	log.Println("All resources collected successfully")

	// Build topology for all instances
	log.Println("Building network topology...")
	if err := c.buildTopologyForAllInstances(); err != nil {
		log.Printf("Topology building completed with errors: %v", err)
		// Don't fail collection if topology building fails
	} else {
		log.Println("Topology building completed successfully")
		// Broadcast topology update event
		c.broadcastTopologyUpdate()
	}

	// Collect metrics
	log.Println("Collecting metrics...")
	metricsStart := time.Now()
	if err := c.metricsCollector.CollectAllMetrics(); err != nil {
		log.Printf("Metrics collection completed with errors: %v", err)
		metrics.RecordCollection("metrics", false, time.Since(metricsStart).Seconds())
		metrics.RecordCollectionError("metrics", "collection_failed")
		// Don't fail collection if metrics collection fails
	} else {
		log.Println("Metrics collection completed successfully")
		metrics.RecordCollection("metrics", true, time.Since(metricsStart).Seconds())
	}

	// Record overall collection duration
	totalDuration := time.Since(startTime).Seconds()
	if len(errors) == 0 {
		metrics.RecordCollection("all", true, totalDuration)
	} else {
		metrics.RecordCollection("all", false, totalDuration)
	}

	// Broadcast collection end event
	c.broadcastCollectionEnd()

	return nil
}

// broadcastTopologyUpdate broadcasts topology update event
// Note: In a distributed system, this would use a message queue or shared event bus
func (c *OpenStackCollector) broadcastTopologyUpdate() {
	// This is a placeholder - in production, use a shared event bus or message queue
	// For now, events are handled within the API service
}

// broadcastCollectionEnd broadcasts collection end event
func (c *OpenStackCollector) broadcastCollectionEnd() {
	// This is a placeholder - in production, use a shared event bus or message queue
	// For now, events are handled within the API service
}

// buildTopologyForAllInstances builds topology for all collected instances
func (c *OpenStackCollector) buildTopologyForAllInstances() error {
	instances, err := c.repository.ListInstances()
	if err != nil {
		return fmt.Errorf("failed to list instances: %w", err)
	}

	var errors []error
	successCount := 0

	for _, instance := range instances {
		if err := c.topologyAnalyzer.BuildTopologyForVM(instance.ID); err != nil {
			log.Printf("Failed to build topology for instance %s: %v", instance.ID, err)
			errors = append(errors, err)
			continue
		}
		successCount++
	}

	if len(errors) == len(instances) {
		return fmt.Errorf("all topology builds failed: %d errors", len(errors))
	}

	if len(errors) > 0 {
		return fmt.Errorf("partial topology build failure: %d succeeded, %d failed", successCount, len(errors))
	}

	return nil
}

