package collector

import (
	"encoding/json"
	"fmt"

	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/layer3/routers"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/networks"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/ports"
	"github.com/network-collector/backend/internal/models"
	"github.com/network-collector/backend/internal/services/storage"
	"github.com/network-collector/backend/pkg/openstack"
)

// NetworkCollector collects network data from OpenStack
type NetworkCollector struct {
	client     *openstack.Client
	repository *storage.Repository
}

// NewNetworkCollector creates a new network collector
func NewNetworkCollector(client *openstack.Client, repository *storage.Repository) *NetworkCollector {
	return &NetworkCollector{
		client:     client,
		repository: repository,
	}
}

// CollectNetworks collects all networks from OpenStack
func (c *NetworkCollector) CollectNetworks() error {
	openstackNetworks, err := c.client.ListNetworks()
	if err != nil {
		return fmt.Errorf("failed to list networks: %w", err)
	}

	var errors []error
	successCount := 0

	for _, network := range openstackNetworks {
		if err := c.saveNetwork(network); err != nil {
			errors = append(errors, fmt.Errorf("failed to save network %s: %w", network.ID, err))
			continue
		}
		successCount++
	}

	if len(errors) == len(openstackNetworks) {
		return fmt.Errorf("all networks failed to collect: %d errors", len(errors))
	}

	if len(errors) > 0 {
		return fmt.Errorf("partial failure: %d succeeded, %d failed", successCount, len(errors))
	}

	return nil
}

// saveNetwork saves a single network to database
func (c *NetworkCollector) saveNetwork(network networks.Network) error {
	// Get project - must exist (collected first)
	var projectID string
	if network.TenantID != "" {
		project, err := c.repository.GetProjectByOpenStackID(network.TenantID)
		if err != nil {
			// Project not found - leave empty to maintain foreign key integrity
			// The network can be updated later when the project is collected
			projectID = ""
		} else {
			projectID = project.ID
		}
	}

	dbNetwork := &models.Network{
		OpenStackID: network.ID,
		Name:        network.Name,
		Status:      network.Status,
		ProjectID:   projectID,
		Shared:      network.Shared,
		CreatedAt:   network.CreatedAt,
		UpdatedAt:   network.UpdatedAt,
	}

	if err := c.repository.UpsertNetwork(dbNetwork); err != nil {
		return err
	}

	// Save subnets
	for _, subnetID := range network.Subnets {
		// Subnet details will be collected separately if needed
		// For now, we just store the network
		_ = subnetID
	}

	return nil
}

// CollectPorts collects all ports from OpenStack
func (c *NetworkCollector) CollectPorts() error {
	opts := ports.ListOpts{}
	openstackPorts, err := c.client.ListPorts(opts)
	if err != nil {
		return fmt.Errorf("failed to list ports: %w", err)
	}

	var errors []error
	successCount := 0

	for _, port := range openstackPorts {
		if err := c.savePort(port); err != nil {
			errors = append(errors, fmt.Errorf("failed to save port %s: %w", port.ID, err))
			continue
		}
		successCount++
	}

	if len(errors) == len(openstackPorts) {
		return fmt.Errorf("all ports failed to collect: %d errors", len(errors))
	}

	if len(errors) > 0 {
		return fmt.Errorf("partial failure: %d succeeded, %d failed", successCount, len(errors))
	}

	return nil
}

// savePort saves a single port to database
func (c *NetworkCollector) savePort(port ports.Port) error {
	// Get network - must exist (collected first)
	var networkID string
	if port.NetworkID != "" {
		network, err := c.repository.GetNetworkByOpenStackID(port.NetworkID)
		if err != nil {
			// Network not found - leave empty to maintain foreign key integrity
			// The port can be updated later when the network is collected
			networkID = ""
		} else {
			networkID = network.ID
		}
	}

	// Convert OpenStack DeviceID (instance OpenStack ID) to internal Instance ID
	// Port.DeviceID should store the internal database Instance ID for foreign key relationship
	var deviceID string
	if port.DeviceID != "" && port.DeviceOwner != "" {
		// Only convert if device owner indicates it's an instance (compute:nova)
		if port.DeviceOwner == "compute:nova" || port.DeviceOwner == "compute:None" {
			instance, err := c.repository.GetInstanceByOpenStackID(port.DeviceID)
			if err == nil {
				deviceID = instance.ID
			} else {
				// Instance not found - leave empty to maintain foreign key integrity
				// The port can be updated later when the instance is collected
				// Storing OpenStack ID here would break the foreign key relationship
				deviceID = ""
			}
		} else {
			// For non-instance devices (routers, etc.), store OpenStack ID directly
			// These don't have foreign key relationships to Instance
			deviceID = port.DeviceID
		}
	}

	// Serialize fixed IPs to JSON
	fixedIPsJSON, err := json.Marshal(port.FixedIPs)
	if err != nil {
		return fmt.Errorf("failed to marshal port fixed IPs: %w", err)
	}

	dbPort := &models.Port{
		OpenStackID: port.ID,
		NetworkID:   networkID,
		DeviceID:    deviceID,
		DeviceOwner: port.DeviceOwner,
		MACAddress:  port.MACAddress,
		Status:      port.Status,
		FixedIPs:    string(fixedIPsJSON),
		CreatedAt:   port.CreatedAt,
	}

	return c.repository.UpsertPort(dbPort)
}

// CollectRouters collects all routers from OpenStack
func (c *NetworkCollector) CollectRouters() error {
	openstackRouters, err := c.client.ListRouters()
	if err != nil {
		return fmt.Errorf("failed to list routers: %w", err)
	}

	var errors []error
	successCount := 0

	for _, router := range openstackRouters {
		if err := c.saveRouter(router); err != nil {
			errors = append(errors, fmt.Errorf("failed to save router %s: %w", router.ID, err))
			continue
		}
		successCount++
	}

	if len(errors) == len(openstackRouters) {
		return fmt.Errorf("all routers failed to collect: %d errors", len(errors))
	}

	if len(errors) > 0 {
		return fmt.Errorf("partial failure: %d succeeded, %d failed", successCount, len(errors))
	}

	return nil
}

// saveRouter saves a single router to database
func (c *NetworkCollector) saveRouter(router routers.Router) error {
	externalGatewayJSON, err := json.Marshal(router.GatewayInfo)
	if err != nil {
		return fmt.Errorf("failed to marshal router gateway info: %w", err)
	}

	dbRouter := &models.Router{
		OpenStackID:       router.ID,
		Name:              router.Name,
		Status:            router.Status,
		ExternalGatewayInfo: string(externalGatewayJSON),
	}

	return c.repository.UpsertRouter(dbRouter)
}

