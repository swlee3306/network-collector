package storage

import (
	"log"
	"time"

	"github.com/network-collector/backend/internal/models"
	"gorm.io/gorm"
)

// Repository handles database operations
type Repository struct {
	db *gorm.DB
}

// NewRepository creates a new repository instance
func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

// Instance operations

// UpsertInstance creates or updates an instance
func (r *Repository) UpsertInstance(instance *models.Instance) error {
	instance.CollectedAt = time.Now()
	return r.db.Save(instance).Error
}

// GetInstanceByOpenStackID gets an instance by OpenStack ID
func (r *Repository) GetInstanceByOpenStackID(openstackID string) (*models.Instance, error) {
	var instance models.Instance
	err := r.db.Where("openstack_id = ?", openstackID).First(&instance).Error
	if err != nil {
		return nil, err
	}
	return &instance, nil
}

// ListInstances lists all instances
func (r *Repository) ListInstances() ([]models.Instance, error) {
	var instances []models.Instance
	err := r.db.Find(&instances).Error
	return instances, err
}

// GetInstancesByProjectID gets all instances for a specific project
func (r *Repository) GetInstancesByProjectID(projectID string) ([]models.Instance, error) {
	var instances []models.Instance
	err := r.db.Where("project_id = ?", projectID).Find(&instances).Error
	return instances, err
}

// GetNetworksByProjectID gets all networks for a specific project
func (r *Repository) GetNetworksByProjectID(projectID string) ([]models.Network, error) {
	var networks []models.Network
	err := r.db.Where("project_id = ?", projectID).Find(&networks).Error
	return networks, err
}

// GetVolumesByProjectID gets all volumes for a specific project
func (r *Repository) GetVolumesByProjectID(projectID string) ([]models.Volume, error) {
	var volumes []models.Volume
	err := r.db.Where("project_id = ?", projectID).Find(&volumes).Error
	return volumes, err
}

// Project operations

// UpsertProject creates or updates a project
func (r *Repository) UpsertProject(project *models.Project) error {
	project.CollectedAt = time.Now()
	return r.db.Save(project).Error
}

// GetProjectByOpenStackID gets a project by OpenStack ID
func (r *Repository) GetProjectByOpenStackID(openstackID string) (*models.Project, error) {
	var project models.Project
	err := r.db.Where("openstack_id = ?", openstackID).First(&project).Error
	if err != nil {
		return nil, err
	}
	return &project, nil
}

// ListProjects lists all projects
func (r *Repository) ListProjects() ([]models.Project, error) {
	var projects []models.Project
	err := r.db.Find(&projects).Error
	return projects, err
}

// GetProjectByID gets a project by ID
func (r *Repository) GetProjectByID(projectID string) (*models.Project, error) {
	var project models.Project
	err := r.db.Where("id = ?", projectID).First(&project).Error
	if err != nil {
		return nil, err
	}
	return &project, nil
}

// Network operations

// UpsertNetwork creates or updates a network
func (r *Repository) UpsertNetwork(network *models.Network) error {
	network.CollectedAt = time.Now()
	return r.db.Save(network).Error
}

// UpsertSubnet creates or updates a subnet
func (r *Repository) UpsertSubnet(subnet *models.Subnet) error {
	subnet.CollectedAt = time.Now()
	return r.db.Save(subnet).Error
}

// UpsertPort creates or updates a port
func (r *Repository) UpsertPort(port *models.Port) error {
	port.CollectedAt = time.Now()
	return r.db.Save(port).Error
}

// UpsertRouter creates or updates a router
func (r *Repository) UpsertRouter(router *models.Router) error {
	router.CollectedAt = time.Now()
	return r.db.Save(router).Error
}

// GetNetworkByOpenStackID gets a network by OpenStack ID
func (r *Repository) GetNetworkByOpenStackID(openstackID string) (*models.Network, error) {
	var network models.Network
	err := r.db.Where("openstack_id = ?", openstackID).First(&network).Error
	if err != nil {
		return nil, err
	}
	return &network, nil
}

// ListNetworks lists all networks
func (r *Repository) ListNetworks() ([]models.Network, error) {
	var networks []models.Network
	err := r.db.Find(&networks).Error
	return networks, err
}

// Hypervisor operations

// UpsertHypervisor creates or updates a hypervisor
func (r *Repository) UpsertHypervisor(hypervisor *models.Hypervisor) error {
	hypervisor.CollectedAt = time.Now()
	return r.db.Save(hypervisor).Error
}

// GetHypervisorByOpenStackID gets a hypervisor by OpenStack ID
func (r *Repository) GetHypervisorByOpenStackID(openstackID string) (*models.Hypervisor, error) {
	var hypervisor models.Hypervisor
	err := r.db.Where("openstack_id = ?", openstackID).First(&hypervisor).Error
	if err != nil {
		return nil, err
	}
	return &hypervisor, nil
}

// Flavor operations

// UpsertFlavor creates or updates a flavor
func (r *Repository) UpsertFlavor(flavor *models.Flavor) error {
	flavor.CollectedAt = time.Now()
	return r.db.Save(flavor).Error
}

// GetFlavorByOpenStackID gets a flavor by OpenStack ID
func (r *Repository) GetFlavorByOpenStackID(openstackID string) (*models.Flavor, error) {
	var flavor models.Flavor
	err := r.db.Where("openstack_id = ?", openstackID).First(&flavor).Error
	if err != nil {
		return nil, err
	}
	return &flavor, nil
}

// ListFlavors lists all flavors
func (r *Repository) ListFlavors() ([]models.Flavor, error) {
	var flavors []models.Flavor
	err := r.db.Find(&flavors).Error
	return flavors, err
}

// GetFlavorByID gets a flavor by ID
func (r *Repository) GetFlavorByID(flavorID string) (*models.Flavor, error) {
	var flavor models.Flavor
	err := r.db.Where("id = ?", flavorID).First(&flavor).Error
	if err != nil {
		return nil, err
	}
	return &flavor, nil
}

// Volume operations

// UpsertVolume creates or updates a volume
func (r *Repository) UpsertVolume(volume *models.Volume) error {
	volume.CollectedAt = time.Now()
	return r.db.Save(volume).Error
}

// ListVolumes lists all volumes
func (r *Repository) ListVolumes() ([]models.Volume, error) {
	var volumes []models.Volume
	err := r.db.Find(&volumes).Error
	return volumes, err
}

// GetVolumeByID gets a volume by ID
func (r *Repository) GetVolumeByID(volumeID string) (*models.Volume, error) {
	var volume models.Volume
	err := r.db.Where("id = ?", volumeID).First(&volume).Error
	if err != nil {
		return nil, err
	}
	return &volume, nil
}

// Metrics operations

// SaveInstanceMetrics saves instance metrics
func (r *Repository) SaveInstanceMetrics(metrics *models.InstanceMetrics) error {
	return r.db.Create(metrics).Error
}

// SaveNetworkMetrics saves network metrics
func (r *Repository) SaveNetworkMetrics(metrics *models.NetworkMetrics) error {
	return r.db.Create(metrics).Error
}

// SaveHypervisorMetrics saves hypervisor metrics
func (r *Repository) SaveHypervisorMetrics(metrics *models.HypervisorMetrics) error {
	return r.db.Create(metrics).Error
}

// Topology operations

// UpsertTopologyNode creates or updates a topology node
func (r *Repository) UpsertTopologyNode(node *models.TopologyNode) error {
	node.CollectedAt = time.Now()
	return r.db.Save(node).Error
}

// UpsertTopologyEdge creates or updates a topology edge
func (r *Repository) UpsertTopologyEdge(edge *models.TopologyEdge) error {
	edge.CollectedAt = time.Now()
	return r.db.Save(edge).Error
}

// GetTopologyNodeByInstanceID gets a topology node by instance ID
func (r *Repository) GetTopologyNodeByInstanceID(instanceID string) (*models.TopologyNode, error) {
	var node models.TopologyNode
	err := r.db.Where("instance_id = ? AND node_type = ?", instanceID, models.NodeTypeVM).First(&node).Error
	if err != nil {
		return nil, err
	}
	return &node, nil
}

// GetTopologyNodeByPortID gets a topology node by port ID
func (r *Repository) GetTopologyNodeByPortID(portID string) (*models.TopologyNode, error) {
	var node models.TopologyNode
	err := r.db.Where("port_id = ? AND node_type = ?", portID, models.NodeTypePort).First(&node).Error
	if err != nil {
		return nil, err
	}
	return &node, nil
}

// GetTopologyNodeByNetworkID gets a topology node by network ID
func (r *Repository) GetTopologyNodeByNetworkID(networkID string) (*models.TopologyNode, error) {
	var node models.TopologyNode
	err := r.db.Where("network_id = ? AND node_type = ?", networkID, models.NodeTypeNetwork).First(&node).Error
	if err != nil {
		return nil, err
	}
	return &node, nil
}

// GetTopologyNodeByRouterID gets a topology node by router ID
func (r *Repository) GetTopologyNodeByRouterID(routerID string) (*models.TopologyNode, error) {
	var node models.TopologyNode
	err := r.db.Where("router_id = ? AND node_type = ?", routerID, models.NodeTypeRouter).First(&node).Error
	if err != nil {
		return nil, err
	}
	return &node, nil
}

// GetTopologyNodeByHypervisorID gets a topology node by hypervisor ID
func (r *Repository) GetTopologyNodeByHypervisorID(hypervisorID string) (*models.TopologyNode, error) {
	var node models.TopologyNode
	err := r.db.Where("hypervisor_id = ? AND node_type = ?", hypervisorID, models.NodeTypeHost).First(&node).Error
	if err != nil {
		return nil, err
	}
	return &node, nil
}

// GetTopologyEdge gets a topology edge by source and target
func (r *Repository) GetTopologyEdge(sourceID, targetID string) (*models.TopologyEdge, error) {
	var edge models.TopologyEdge
	err := r.db.Where("source_node_id = ? AND target_node_id = ?", sourceID, targetID).First(&edge).Error
	if err != nil {
		return nil, err
	}
	return &edge, nil
}

// GetPortsByOpenStackDeviceID gets ports by OpenStack device ID (instance OpenStack ID)
// Note: Port.DeviceID stores OpenStack instance ID, not internal UUID
func (r *Repository) GetPortsByOpenStackDeviceID(openstackDeviceID string) ([]models.Port, error) {
	var ports []models.Port
	err := r.db.Where("device_id = ?", openstackDeviceID).Find(&ports).Error
	return ports, err
}

// GetRoutersByNetwork gets routers connected to a network (simplified - in production use Neutron API)
func (r *Repository) GetRoutersByNetwork(networkOpenStackID string) ([]models.Router, error) {
	// This is a simplified implementation
	// In production, we'd query Neutron API for router-network connections
	var routers []models.Router
	err := r.db.Find(&routers).Error
	return routers, err
}

// GetInstanceByID gets an instance by ID
func (r *Repository) GetInstanceByID(instanceID string) (*models.Instance, error) {
	var instance models.Instance
	err := r.db.Where("id = ?", instanceID).First(&instance).Error
	if err != nil {
		return nil, err
	}
	return &instance, nil
}

// GetNetworkByID gets a network by ID
func (r *Repository) GetNetworkByID(networkID string) (*models.Network, error) {
	var network models.Network
	err := r.db.Where("id = ?", networkID).First(&network).Error
	if err != nil {
		return nil, err
	}
	return &network, nil
}

// GetHypervisorByID gets a hypervisor by ID
func (r *Repository) GetHypervisorByID(hypervisorID string) (*models.Hypervisor, error) {
	var hypervisor models.Hypervisor
	err := r.db.Where("id = ?", hypervisorID).First(&hypervisor).Error
	if err != nil {
		return nil, err
	}
	return &hypervisor, nil
}

// ListHypervisors lists all hypervisors
func (r *Repository) ListHypervisors() ([]models.Hypervisor, error) {
	var hypervisors []models.Hypervisor
	err := r.db.Find(&hypervisors).Error
	return hypervisors, err
}

// Topology query operations

// GetTopologyEdgesByNodeID gets all edges connected to a node (as source or target)
func (r *Repository) GetTopologyEdgesByNodeID(nodeID string) ([]models.TopologyEdge, error) {
	var edges []models.TopologyEdge
	err := r.db.Where("source_node_id = ? OR target_node_id = ?", nodeID, nodeID).Find(&edges).Error
	return edges, err
}

// GetTopologyNodesByIDs gets multiple topology nodes by IDs
func (r *Repository) GetTopologyNodesByIDs(nodeIDs []string) ([]models.TopologyNode, error) {
	if len(nodeIDs) == 0 {
		return []models.TopologyNode{}, nil
	}
	var nodes []models.TopologyNode
	err := r.db.Where("id IN ?", nodeIDs).Find(&nodes).Error
	return nodes, err
}

// GetTopologyNodeByID gets a topology node by ID
func (r *Repository) GetTopologyNodeByID(nodeID string) (*models.TopologyNode, error) {
	var node models.TopologyNode
	err := r.db.Where("id = ?", nodeID).First(&node).Error
	if err != nil {
		return nil, err
	}
	return &node, nil
}

// Metrics query operations

// GetInstanceMetrics gets instance metrics within a time range
func (r *Repository) GetInstanceMetrics(instanceID string, start, end time.Time) ([]models.InstanceMetrics, error) {
	var metrics []models.InstanceMetrics
	err := r.db.Where("instance_id = ? AND timestamp >= ? AND timestamp <= ?", instanceID, start, end).
		Order("timestamp ASC").
		Find(&metrics).Error
	return metrics, err
}

// GetNetworkMetrics gets network metrics within a time range
func (r *Repository) GetNetworkMetrics(networkID string, start, end time.Time) ([]models.NetworkMetrics, error) {
	var metrics []models.NetworkMetrics
	err := r.db.Where("network_id = ? AND timestamp >= ? AND timestamp <= ?", networkID, start, end).
		Order("timestamp ASC").
		Find(&metrics).Error
	return metrics, err
}

// GetHypervisorMetrics gets hypervisor metrics within a time range
func (r *Repository) GetHypervisorMetrics(hypervisorID string, start, end time.Time) ([]models.HypervisorMetrics, error) {
	var metrics []models.HypervisorMetrics
	err := r.db.Where("hypervisor_id = ? AND timestamp >= ? AND timestamp <= ?", hypervisorID, start, end).
		Order("timestamp ASC").
		Find(&metrics).Error
	return metrics, err
}

// Data retention operations

// DeleteInstanceMetricsBefore deletes instance metrics before the specified date
func (r *Repository) DeleteInstanceMetricsBefore(cutoffDate time.Time) error {
	result := r.db.Where("timestamp < ?", cutoffDate).Delete(&models.InstanceMetrics{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected > 0 {
		log.Printf("Deleted %d instance metrics records older than %s", result.RowsAffected, cutoffDate.Format(time.RFC3339))
	}
	return nil
}

// DeleteNetworkMetricsBefore deletes network metrics before the specified date
func (r *Repository) DeleteNetworkMetricsBefore(cutoffDate time.Time) error {
	result := r.db.Where("timestamp < ?", cutoffDate).Delete(&models.NetworkMetrics{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected > 0 {
		log.Printf("Deleted %d network metrics records older than %s", result.RowsAffected, cutoffDate.Format(time.RFC3339))
	}
	return nil
}

// DeleteHypervisorMetricsBefore deletes hypervisor metrics before the specified date
func (r *Repository) DeleteHypervisorMetricsBefore(cutoffDate time.Time) error {
	result := r.db.Where("timestamp < ?", cutoffDate).Delete(&models.HypervisorMetrics{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected > 0 {
		log.Printf("Deleted %d hypervisor metrics records older than %s", result.RowsAffected, cutoffDate.Format(time.RFC3339))
	}
	return nil
}

