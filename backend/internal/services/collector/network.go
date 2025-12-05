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
	var projectID string
	if network.TenantID != "" {
		project, err := c.repository.GetProjectByOpenStackID(network.TenantID)
		if err != nil {
			projectID = network.TenantID
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
	// Get network
	var networkID string
	if port.NetworkID != "" {
		network, err := c.repository.GetNetworkByOpenStackID(port.NetworkID)
		if err != nil {
			networkID = port.NetworkID
		} else {
			networkID = network.ID
		}
	}

	// Store OpenStack device ID directly (Port.DeviceID stores OpenStack instance ID)
	deviceID := port.DeviceID

	// Serialize fixed IPs to JSON
	fixedIPsJSON, _ := json.Marshal(port.FixedIPs)

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
	externalGatewayJSON, _ := json.Marshal(router.GatewayInfo)

	dbRouter := &models.Router{
		OpenStackID:       router.ID,
		Name:              router.Name,
		Status:            router.Status,
		ExternalGatewayInfo: string(externalGatewayJSON),
	}

	return c.repository.UpsertRouter(dbRouter)
}

