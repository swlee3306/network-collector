package topology

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/network-collector/backend/internal/models"
	"github.com/network-collector/backend/internal/services/storage"
)

// Analyzer analyzes network topology and builds graph model
type Analyzer struct {
	repository *storage.Repository
}

// NewAnalyzer creates a new topology analyzer
func NewAnalyzer(repository *storage.Repository) *Analyzer {
	return &Analyzer{
		repository: repository,
	}
}

// BuildTopologyForVM builds topology graph for a specific VM
func (a *Analyzer) BuildTopologyForVM(instanceID string) error {
	// Get instance
	instance, err := a.repository.GetInstanceByID(instanceID)
	if err != nil {
		return fmt.Errorf("failed to get instance: %w", err)
	}

	log.Printf("Building topology for VM: %s (%s)", instance.Name, instance.ID)

	// Create VM node
	vmNode, err := a.createOrUpdateNode(models.NodeTypeVM, instance.ID, instance.Name, instance.OpenStackID, nil)
	if err != nil {
		return fmt.Errorf("failed to create VM node: %w", err)
	}

	// Find ports connected to this VM (by OpenStack instance ID)
	ports, err := a.repository.GetPortsByOpenStackDeviceID(instance.OpenStackID)
	if err != nil {
		return fmt.Errorf("failed to get ports: %w", err)
	}

	if len(ports) == 0 {
		log.Printf("No ports found for VM %s", instance.ID)
		return nil
	}

	// For each port, trace the path
	for _, port := range ports {
		if err := a.tracePathFromPort(&port, vmNode); err != nil {
			log.Printf("Failed to trace path from port %s: %v", port.ID, err)
			// Continue with other ports even if one fails
			continue
		}
	}

	return nil
}

// tracePathFromPort traces network path from a port to physical host
func (a *Analyzer) tracePathFromPort(port *models.Port, sourceNode *models.TopologyNode) error {
	// Create port node
	portNode, err := a.createOrUpdateNode(models.NodeTypePort, port.ID, port.OpenStackID, port.OpenStackID, map[string]interface{}{
		"mac_address": port.MACAddress,
		"device_owner": port.DeviceOwner,
	})
	if err != nil {
		return err
	}

	// Create edge: VM -> Port
	if err := a.createOrUpdateEdge(sourceNode.ID, portNode.ID, "VIRTUAL", nil); err != nil {
		return err
	}

	// Get network
	if port.NetworkID == "" {
		return fmt.Errorf("port %s has no network", port.ID)
	}

	network, err := a.repository.GetNetworkByID(port.NetworkID)
	if err != nil {
		// Network might not be accessible - mark as unknown
		log.Printf("Network %s not found for port %s", port.NetworkID, port.ID)
		unknownNode, err := a.createOrUpdateNode(models.NodeTypeUnknown, "", fmt.Sprintf("Unknown Network: %s", port.NetworkID), "", map[string]interface{}{
			"network_id": port.NetworkID,
		})
		if err != nil {
			return err
		}
		unknownNode.IsAccessible = false
		if err := a.repository.UpsertTopologyNode(unknownNode); err != nil {
			return err
		}

		// Create edge: Port -> Unknown Network
		return a.createOrUpdateEdge(portNode.ID, unknownNode.ID, "VIRTUAL", nil)
	}

	// Create network node
	networkNode, err := a.createOrUpdateNode(models.NodeTypeNetwork, network.ID, network.Name, network.OpenStackID, map[string]interface{}{
		"status": network.Status,
		"shared": network.Shared,
	})
	if err != nil {
		return err
	}

	// Create edge: Port -> Network
	if err := a.createOrUpdateEdge(portNode.ID, networkNode.ID, "VIRTUAL", nil); err != nil {
		return err
	}

	// Find routers connected to this network
	routers, err := a.repository.GetRoutersByNetwork(network.OpenStackID)
	if err != nil {
		// No routers found - this is OK, network might not have external gateway
		log.Printf("No routers found for network %s", network.OpenStackID)
		return nil
	}

	// For each router, trace to external network and physical host
	for _, router := range routers {
		if err := a.tracePathFromRouter(&router, networkNode); err != nil {
			log.Printf("Failed to trace path from router %s: %v", router.ID, err)
			continue
		}
	}

	// Also try to find direct path to hypervisor (for internal networks)
	// port.DeviceID contains OpenStack instance ID, need to find instance
	if port.DeviceID != "" {
		instance, err := a.repository.GetInstanceByOpenStackID(port.DeviceID)
		if err == nil && instance.HypervisorID != "" {
			hypervisor, err := a.repository.GetHypervisorByID(instance.HypervisorID)
			if err == nil {
				hostNode, err := a.createOrUpdateNode(models.NodeTypeHost, hypervisor.ID, hypervisor.Hostname, hypervisor.OpenStackID, map[string]interface{}{
					"host_ip": hypervisor.HostIP,
					"status": hypervisor.Status,
				})
				if err == nil {
					// Create edge: Network -> Host (direct connection for internal networks)
					if err := a.createOrUpdateEdge(networkNode.ID, hostNode.ID, "PHYSICAL", nil); err != nil {
						log.Printf("Failed to create edge from network to host: %v", err)
					}
				}
			}
		}
	}

	return nil
}

// tracePathFromRouter traces path from router to physical host
func (a *Analyzer) tracePathFromRouter(router *models.Router, sourceNode *models.TopologyNode) error {
	// Create router node
	routerNode, err := a.createOrUpdateNode(models.NodeTypeRouter, router.ID, router.Name, router.OpenStackID, map[string]interface{}{
		"status": router.Status,
	})
	if err != nil {
		return err
	}

	// Create edge: Network -> Router
	if err := a.createOrUpdateEdge(sourceNode.ID, routerNode.ID, "VIRTUAL", nil); err != nil {
		return err
	}

	// Router has external gateway - this connects to physical network
	// For now, we'll create a placeholder for physical network
	// In a real implementation, we'd query Neutron agents to find physical network details

	// Find hypervisor that might be hosting VMs on this network
	// This is a simplified approach - in production, we'd use Neutron agents
	hypervisors, err := a.repository.ListHypervisors()
	if err != nil {
		return err
	}

	// For each hypervisor, check if it has VMs on networks connected to this router
	// This is a simplified topology - in production, we'd use more sophisticated mapping
	for _, hypervisor := range hypervisors {
		hostNode, err := a.createOrUpdateNode(models.NodeTypeHost, hypervisor.ID, hypervisor.Hostname, hypervisor.OpenStackID, map[string]interface{}{
			"host_ip": hypervisor.HostIP,
			"status": hypervisor.Status,
		})
		if err != nil {
			continue
		}

		// Create edge: Router -> Host
		// Note: In production, this would be more sophisticated, using Neutron agents
		if err := a.createOrUpdateEdge(routerNode.ID, hostNode.ID, "PHYSICAL", map[string]interface{}{
			"note": "simplified topology - use Neutron agents for accurate mapping",
		}); err != nil {
			log.Printf("Failed to create edge from router to host: %v", err)
		}
	}

	return nil
}

// createOrUpdateNode creates or updates a topology node
func (a *Analyzer) createOrUpdateNode(nodeType models.TopologyNodeType, id, name, openstackID string, metadata map[string]interface{}) (*models.TopologyNode, error) {
	// Try to find existing node
	var node *models.TopologyNode
	var err error

	switch nodeType {
	case models.NodeTypeVM:
		if id != "" {
			node, err = a.repository.GetTopologyNodeByInstanceID(id)
		}
	case models.NodeTypePort:
		if id != "" {
			node, err = a.repository.GetTopologyNodeByPortID(id)
		}
	case models.NodeTypeNetwork:
		if id != "" {
			node, err = a.repository.GetTopologyNodeByNetworkID(id)
		}
	case models.NodeTypeRouter:
		if id != "" {
			node, err = a.repository.GetTopologyNodeByRouterID(id)
		}
	case models.NodeTypeHost:
		if id != "" {
			node, err = a.repository.GetTopologyNodeByHypervisorID(id)
		}
	}

	// If node exists, update it; otherwise create new
	if node == nil || err != nil {
		node = &models.TopologyNode{
			NodeType:    nodeType,
			OpenStackID: openstackID,
			Name:        name,
			IsAccessible: true,
		}

		// Set appropriate foreign key based on node type
		switch nodeType {
		case models.NodeTypeVM:
			node.InstanceID = id
		case models.NodeTypePort:
			node.PortID = id
		case models.NodeTypeNetwork:
			node.NetworkID = id
		case models.NodeTypeRouter:
			node.RouterID = id
		case models.NodeTypeHost:
			node.HypervisorID = id
		}
	}

	// Update metadata
	if metadata != nil {
		metadataJSON, _ := json.Marshal(metadata)
		node.Metadata = string(metadataJSON)
	}

	if err := a.repository.UpsertTopologyNode(node); err != nil {
		return nil, err
	}

	return node, nil
}

// createOrUpdateEdge creates or updates a topology edge
func (a *Analyzer) createOrUpdateEdge(sourceID, targetID string, edgeType string, metadata map[string]interface{}) error {
	// Check if edge already exists
	edge, err := a.repository.GetTopologyEdge(sourceID, targetID)
	if err != nil {
		// Edge doesn't exist, create new
		edge = &models.TopologyEdge{
			SourceNodeID: sourceID,
			TargetNodeID: targetID,
			EdgeType:     edgeType,
		}
	}

	if metadata != nil {
		metadataJSON, _ := json.Marshal(metadata)
		edge.Metadata = string(metadataJSON)
	}

	return a.repository.UpsertTopologyEdge(edge)
}

