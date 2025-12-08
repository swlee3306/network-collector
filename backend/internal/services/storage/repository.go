package storage

import (
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
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
	
	// Check if instance exists by open_stack_id
	existing, err := r.GetInstanceByOpenStackID(instance.OpenStackID)
	if err == nil && existing != nil {
		// Update existing instance
		instance.ID = existing.ID
		// Use Updates to handle NULL values properly
		updateData := map[string]interface{}{
			"name":         instance.Name,
			"status":       instance.Status,
			"project_id":   instance.ProjectID,
			"flavor_id":    instance.FlavorID,
			"updated_at":   instance.UpdatedAt,
			"collected_at": instance.CollectedAt,
		}
		// Set hypervisor_id to NULL if empty, otherwise set the value
		if instance.HypervisorID == "" {
			updateData["hypervisor_id"] = nil
		} else {
			updateData["hypervisor_id"] = instance.HypervisorID
		}
		return r.db.Model(instance).Updates(updateData).Error
	}
	// Create new instance - use Updates for NULL handling
	if instance.ID == "" {
		instance.ID = uuid.New().String()
	}
	createData := map[string]interface{}{
		"id":           instance.ID,
		"open_stack_id": instance.OpenStackID,
		"name":         instance.Name,
		"status":       instance.Status,
		"project_id":   instance.ProjectID,
		"flavor_id":    instance.FlavorID,
		"created_at":   instance.CreatedAt,
		"updated_at":   instance.UpdatedAt,
		"collected_at": instance.CollectedAt,
	}
	// Set hypervisor_id to NULL if empty, otherwise set the value
	if instance.HypervisorID == "" {
		createData["hypervisor_id"] = nil
	} else {
		createData["hypervisor_id"] = instance.HypervisorID
	}
	return r.db.Model(instance).Create(createData).Error
}

// GetInstanceByOpenStackID gets an instance by OpenStack ID
func (r *Repository) GetInstanceByOpenStackID(openstackID string) (*models.Instance, error) {
	var instance models.Instance
	err := r.db.Where("open_stack_id = ?", openstackID).First(&instance).Error
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

// deleteTopologyNodesAndEdges deletes topology nodes and their associated edges
// This is a helper function to ensure edges are deleted before nodes
func (r *Repository) deleteTopologyNodesAndEdges(nodeIDs []string) error {
	if len(nodeIDs) == 0 {
		return nil
	}

	// First, delete edges that reference these nodes (as source or target)
	if err := r.db.Where("source_node_id IN ? OR target_node_id IN ?", nodeIDs, nodeIDs).Delete(&models.TopologyEdge{}).Error; err != nil {
		return fmt.Errorf("failed to delete topology edges: %w", err)
	}

	// Then, delete the topology nodes themselves
	if err := r.db.Where("id IN ?", nodeIDs).Delete(&models.TopologyNode{}).Error; err != nil {
		return fmt.Errorf("failed to delete topology nodes: %w", err)
	}

	return nil
}

// DeleteInstancesNotIn deletes instances that are not in the provided OpenStack ID list
// Also deletes related child records (metrics, topology_nodes) and updates volumes
func (r *Repository) DeleteInstancesNotIn(openstackIDs []string) error {
	// First, find instances to be deleted
	var instancesToDelete []models.Instance
	query := r.db
	if len(openstackIDs) == 0 {
		query = query.Find(&instancesToDelete)
	} else {
		query = query.Where("open_stack_id NOT IN ?", openstackIDs).Find(&instancesToDelete)
	}
	if query.Error != nil {
		return query.Error
	}

	if len(instancesToDelete) == 0 {
		return nil
	}

	// Get instance IDs
	instanceIDs := make([]string, 0, len(instancesToDelete))
	for _, i := range instancesToDelete {
		instanceIDs = append(instanceIDs, i.ID)
	}

	// Delete related child records first
	// 1. Delete instance metrics
	if err := r.db.Where("instance_id IN ?", instanceIDs).Delete(&models.InstanceMetrics{}).Error; err != nil {
		return fmt.Errorf("failed to delete instance metrics: %w", err)
	}

	// 2. Find and delete topology nodes that reference these instances (and their edges)
	var topologyNodes []models.TopologyNode
	if err := r.db.Where("instance_id IN ?", instanceIDs).Find(&topologyNodes).Error; err == nil && len(topologyNodes) > 0 {
		nodeIDs := make([]string, 0, len(topologyNodes))
		for _, n := range topologyNodes {
			nodeIDs = append(nodeIDs, n.ID)
		}
		if err := r.deleteTopologyNodesAndEdges(nodeIDs); err != nil {
			return err
		}
	}

	// 3. Update volumes that are attached to these instances (set AttachedTo to NULL)
	if err := r.db.Model(&models.Volume{}).Where("attached_to IN ?", instanceIDs).Update("attached_to", nil).Error; err != nil {
		return fmt.Errorf("failed to update volumes: %w", err)
	}

	// Finally, delete the instances themselves
	result := r.db.Where("id IN ?", instanceIDs).Delete(&models.Instance{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected > 0 {
		log.Printf("Deleted %d instances that no longer exist in OpenStack", result.RowsAffected)
	}
	return nil
}

// DeleteNetworksNotIn deletes networks that are not in the provided OpenStack ID list
// Also deletes related child records (metrics, ports, subnets, topology_nodes)
func (r *Repository) DeleteNetworksNotIn(openstackIDs []string) error {
	// First, find networks to be deleted
	var networksToDelete []models.Network
	query := r.db
	if len(openstackIDs) == 0 {
		query = query.Find(&networksToDelete)
	} else {
		query = query.Where("open_stack_id NOT IN ?", openstackIDs).Find(&networksToDelete)
	}
	if query.Error != nil {
		return query.Error
	}

	if len(networksToDelete) == 0 {
		return nil
	}

	// Get network IDs
	networkIDs := make([]string, 0, len(networksToDelete))
	for _, n := range networksToDelete {
		networkIDs = append(networkIDs, n.ID)
	}

	// Delete related child records first
	// 1. Delete network metrics
	if err := r.db.Where("network_id IN ?", networkIDs).Delete(&models.NetworkMetrics{}).Error; err != nil {
		return fmt.Errorf("failed to delete network metrics: %w", err)
	}

	// 2. Delete ports (they reference networks)
	if err := r.db.Where("network_id IN ?", networkIDs).Delete(&models.Port{}).Error; err != nil {
		return fmt.Errorf("failed to delete ports: %w", err)
	}

	// 3. Delete subnets
	if err := r.db.Where("network_id IN ?", networkIDs).Delete(&models.Subnet{}).Error; err != nil {
		return fmt.Errorf("failed to delete subnets: %w", err)
	}

	// 4. Find and delete topology nodes that reference these networks (and their edges)
	var topologyNodes []models.TopologyNode
	if err := r.db.Where("network_id IN ?", networkIDs).Find(&topologyNodes).Error; err == nil && len(topologyNodes) > 0 {
		nodeIDs := make([]string, 0, len(topologyNodes))
		for _, n := range topologyNodes {
			nodeIDs = append(nodeIDs, n.ID)
		}
		if err := r.deleteTopologyNodesAndEdges(nodeIDs); err != nil {
			return err
		}
	}

	// Finally, delete the networks themselves
	result := r.db.Where("id IN ?", networkIDs).Delete(&models.Network{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected > 0 {
		log.Printf("Deleted %d networks that no longer exist in OpenStack", result.RowsAffected)
	}
	return nil
}

// DeletePortsNotIn deletes ports that are not in the provided OpenStack ID list
// Also deletes related topology_nodes
func (r *Repository) DeletePortsNotIn(openstackIDs []string) error {
	// First, find ports to be deleted
	var portsToDelete []models.Port
	query := r.db
	if len(openstackIDs) == 0 {
		query = query.Find(&portsToDelete)
	} else {
		query = query.Where("open_stack_id NOT IN ?", openstackIDs).Find(&portsToDelete)
	}
	if query.Error != nil {
		return query.Error
	}

	if len(portsToDelete) == 0 {
		return nil
	}

	// Get port IDs
	portIDs := make([]string, 0, len(portsToDelete))
	for _, p := range portsToDelete {
		portIDs = append(portIDs, p.ID)
	}

	// Find and delete topology nodes that reference these ports (and their edges)
	var topologyNodes []models.TopologyNode
	if err := r.db.Where("port_id IN ?", portIDs).Find(&topologyNodes).Error; err == nil && len(topologyNodes) > 0 {
		nodeIDs := make([]string, 0, len(topologyNodes))
		for _, n := range topologyNodes {
			nodeIDs = append(nodeIDs, n.ID)
		}
		if err := r.deleteTopologyNodesAndEdges(nodeIDs); err != nil {
			return err
		}
	}

	// Finally, delete the ports themselves
	result := r.db.Where("id IN ?", portIDs).Delete(&models.Port{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected > 0 {
		log.Printf("Deleted %d ports that no longer exist in OpenStack", result.RowsAffected)
	}
	return nil
}

// DeleteRoutersNotIn deletes routers that are not in the provided OpenStack ID list
// Also deletes related topology_nodes
func (r *Repository) DeleteRoutersNotIn(openstackIDs []string) error {
	// First, find routers to be deleted
	var routersToDelete []models.Router
	query := r.db
	if len(openstackIDs) == 0 {
		query = query.Find(&routersToDelete)
	} else {
		query = query.Where("open_stack_id NOT IN ?", openstackIDs).Find(&routersToDelete)
	}
	if query.Error != nil {
		return query.Error
	}

	if len(routersToDelete) == 0 {
		return nil
	}

	// Get router IDs
	routerIDs := make([]string, 0, len(routersToDelete))
	for _, rt := range routersToDelete {
		routerIDs = append(routerIDs, rt.ID)
	}

	// Find and delete topology nodes that reference these routers (and their edges)
	var topologyNodes []models.TopologyNode
	if err := r.db.Where("router_id IN ?", routerIDs).Find(&topologyNodes).Error; err == nil && len(topologyNodes) > 0 {
		nodeIDs := make([]string, 0, len(topologyNodes))
		for _, n := range topologyNodes {
			nodeIDs = append(nodeIDs, n.ID)
		}
		if err := r.deleteTopologyNodesAndEdges(nodeIDs); err != nil {
			return err
		}
	}

	// Finally, delete the routers themselves
	result := r.db.Where("id IN ?", routerIDs).Delete(&models.Router{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected > 0 {
		log.Printf("Deleted %d routers that no longer exist in OpenStack", result.RowsAffected)
	}
	return nil
}

// DeleteVolumesNotIn deletes volumes that are not in the provided OpenStack ID list
func (r *Repository) DeleteVolumesNotIn(openstackIDs []string) error {
	if len(openstackIDs) == 0 {
		result := r.db.Delete(&models.Volume{})
		return result.Error
	}
	result := r.db.Where("open_stack_id NOT IN ?", openstackIDs).Delete(&models.Volume{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected > 0 {
		log.Printf("Deleted %d volumes that no longer exist in OpenStack", result.RowsAffected)
	}
	return nil
}

// DeleteProjectsNotIn deletes projects that are not in the provided OpenStack ID list
// Also deletes related child records (instances, networks, volumes)
// Note: This will cascade delete all resources in those projects
func (r *Repository) DeleteProjectsNotIn(openstackIDs []string) error {
	// First, find projects to be deleted
	var projectsToDelete []models.Project
	query := r.db
	if len(openstackIDs) == 0 {
		query = query.Find(&projectsToDelete)
	} else {
		query = query.Where("open_stack_id NOT IN ?", openstackIDs).Find(&projectsToDelete)
	}
	if query.Error != nil {
		return query.Error
	}

	if len(projectsToDelete) == 0 {
		return nil
	}

	// Get project IDs
	projectIDs := make([]string, 0, len(projectsToDelete))
	for _, p := range projectsToDelete {
		projectIDs = append(projectIDs, p.ID)
	}

	// Delete related child records first
	// 1. Delete instances (and their metrics, topology_nodes will be handled by instance deletion)
	var instancesToDelete []models.Instance
	if err := r.db.Where("project_id IN ?", projectIDs).Find(&instancesToDelete).Error; err == nil && len(instancesToDelete) > 0 {
		instanceIDs := make([]string, 0, len(instancesToDelete))
		for _, i := range instancesToDelete {
			instanceIDs = append(instanceIDs, i.ID)
		}
		// Delete instance metrics
		if err := r.db.Where("instance_id IN ?", instanceIDs).Delete(&models.InstanceMetrics{}).Error; err != nil {
			return fmt.Errorf("failed to delete instance metrics: %w", err)
		}
		// Find and delete topology nodes (and their edges)
		var topologyNodes []models.TopologyNode
		if err := r.db.Where("instance_id IN ?", instanceIDs).Find(&topologyNodes).Error; err == nil && len(topologyNodes) > 0 {
			nodeIDs := make([]string, 0, len(topologyNodes))
			for _, n := range topologyNodes {
				nodeIDs = append(nodeIDs, n.ID)
			}
			if err := r.deleteTopologyNodesAndEdges(nodeIDs); err != nil {
				return err
			}
		}
		// Update volumes
		if err := r.db.Model(&models.Volume{}).Where("attached_to IN ?", instanceIDs).Update("attached_to", nil).Error; err != nil {
			return fmt.Errorf("failed to update volumes: %w", err)
		}
		// Delete instances
		if err := r.db.Where("project_id IN ?", projectIDs).Delete(&models.Instance{}).Error; err != nil {
			return fmt.Errorf("failed to delete instances: %w", err)
		}
	}

	// 2. Delete networks (and their metrics, ports, subnets, topology_nodes)
	var networksToDelete []models.Network
	if err := r.db.Where("project_id IN ?", projectIDs).Find(&networksToDelete).Error; err == nil && len(networksToDelete) > 0 {
		networkIDs := make([]string, 0, len(networksToDelete))
		for _, n := range networksToDelete {
			networkIDs = append(networkIDs, n.ID)
		}
		// Delete network metrics
		if err := r.db.Where("network_id IN ?", networkIDs).Delete(&models.NetworkMetrics{}).Error; err != nil {
			return fmt.Errorf("failed to delete network metrics: %w", err)
		}
		// Delete ports
		if err := r.db.Where("network_id IN ?", networkIDs).Delete(&models.Port{}).Error; err != nil {
			return fmt.Errorf("failed to delete ports: %w", err)
		}
		// Delete subnets
		if err := r.db.Where("network_id IN ?", networkIDs).Delete(&models.Subnet{}).Error; err != nil {
			return fmt.Errorf("failed to delete subnets: %w", err)
		}
		// Find and delete topology nodes (and their edges)
		var topologyNodes []models.TopologyNode
		if err := r.db.Where("network_id IN ?", networkIDs).Find(&topologyNodes).Error; err == nil && len(topologyNodes) > 0 {
			nodeIDs := make([]string, 0, len(topologyNodes))
			for _, n := range topologyNodes {
				nodeIDs = append(nodeIDs, n.ID)
			}
			if err := r.deleteTopologyNodesAndEdges(nodeIDs); err != nil {
				return err
			}
		}
		// Delete networks
		if err := r.db.Where("project_id IN ?", projectIDs).Delete(&models.Network{}).Error; err != nil {
			return fmt.Errorf("failed to delete networks: %w", err)
		}
	}

	// 3. Delete volumes
	if err := r.db.Where("project_id IN ?", projectIDs).Delete(&models.Volume{}).Error; err != nil {
		return fmt.Errorf("failed to delete volumes: %w", err)
	}

	// Finally, delete the projects themselves
	result := r.db.Where("id IN ?", projectIDs).Delete(&models.Project{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected > 0 {
		log.Printf("Deleted %d projects that no longer exist in OpenStack", result.RowsAffected)
	}
	return nil
}

// DeleteFlavorsNotIn deletes flavors that are not in the provided OpenStack ID list
// Also updates instances that reference these flavors (set FlavorID to NULL)
func (r *Repository) DeleteFlavorsNotIn(openstackIDs []string) error {
	// First, find flavors to be deleted
	var flavorsToDelete []models.Flavor
	query := r.db
	if len(openstackIDs) == 0 {
		query = query.Find(&flavorsToDelete)
	} else {
		query = query.Where("open_stack_id NOT IN ?", openstackIDs).Find(&flavorsToDelete)
	}
	if query.Error != nil {
		return query.Error
	}

	if len(flavorsToDelete) == 0 {
		return nil
	}

	// Get flavor IDs
	flavorIDs := make([]string, 0, len(flavorsToDelete))
	for _, f := range flavorsToDelete {
		flavorIDs = append(flavorIDs, f.ID)
	}

	// Update instances that reference these flavors (set FlavorID to NULL)
	if err := r.db.Model(&models.Instance{}).Where("flavor_id IN ?", flavorIDs).Update("flavor_id", nil).Error; err != nil {
		return fmt.Errorf("failed to update instances: %w", err)
	}

	// Finally, delete the flavors themselves
	result := r.db.Where("id IN ?", flavorIDs).Delete(&models.Flavor{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected > 0 {
		log.Printf("Deleted %d flavors that no longer exist in OpenStack", result.RowsAffected)
	}
	return nil
}

// DeleteHypervisorsNotIn deletes hypervisors that are not in the provided OpenStack ID list
// Also deletes related child records (metrics, topology_nodes) and updates instances
func (r *Repository) DeleteHypervisorsNotIn(openstackIDs []string) error {
	// First, find hypervisors to be deleted
	var hypervisorsToDelete []models.Hypervisor
	query := r.db
	if len(openstackIDs) == 0 {
		query = query.Find(&hypervisorsToDelete)
	} else {
		query = query.Where("open_stack_id NOT IN ?", openstackIDs).Find(&hypervisorsToDelete)
	}
	if query.Error != nil {
		return query.Error
	}

	if len(hypervisorsToDelete) == 0 {
		return nil
	}

	// Get hypervisor IDs
	hypervisorIDs := make([]string, 0, len(hypervisorsToDelete))
	for _, h := range hypervisorsToDelete {
		hypervisorIDs = append(hypervisorIDs, h.ID)
	}

	// Delete related child records first
	// 1. Delete hypervisor metrics
	if err := r.db.Where("hypervisor_id IN ?", hypervisorIDs).Delete(&models.HypervisorMetrics{}).Error; err != nil {
		return fmt.Errorf("failed to delete hypervisor metrics: %w", err)
	}

	// 2. Find and delete topology nodes that reference these hypervisors (and their edges)
	var topologyNodes []models.TopologyNode
	if err := r.db.Where("hypervisor_id IN ?", hypervisorIDs).Find(&topologyNodes).Error; err == nil && len(topologyNodes) > 0 {
		nodeIDs := make([]string, 0, len(topologyNodes))
		for _, n := range topologyNodes {
			nodeIDs = append(nodeIDs, n.ID)
		}
		if err := r.deleteTopologyNodesAndEdges(nodeIDs); err != nil {
			return err
		}
	}

	// 3. Update instances that reference these hypervisors (set HypervisorID to NULL)
	if err := r.db.Model(&models.Instance{}).Where("hypervisor_id IN ?", hypervisorIDs).Update("hypervisor_id", nil).Error; err != nil {
		return fmt.Errorf("failed to update instances: %w", err)
	}

	// Finally, delete the hypervisors themselves
	result := r.db.Where("id IN ?", hypervisorIDs).Delete(&models.Hypervisor{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected > 0 {
		log.Printf("Deleted %d hypervisors that no longer exist in OpenStack", result.RowsAffected)
	}
	return nil
}

// Project operations

// UpsertProject creates or updates a project
func (r *Repository) UpsertProject(project *models.Project) error {
	project.CollectedAt = time.Now()
	// Check if project exists by open_stack_id
	existing, err := r.GetProjectByOpenStackID(project.OpenStackID)
	if err == nil && existing != nil {
		// Update existing project
		project.ID = existing.ID
		return r.db.Save(project).Error
	}
	// Create new project
	return r.db.Create(project).Error
}

// GetProjectByOpenStackID gets a project by OpenStack ID
func (r *Repository) GetProjectByOpenStackID(openstackID string) (*models.Project, error) {
	var project models.Project
	err := r.db.Where("open_stack_id = ?", openstackID).First(&project).Error
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
	// Check if network exists by open_stack_id
	existing, err := r.GetNetworkByOpenStackID(network.OpenStackID)
	if err == nil && existing != nil {
		// Update existing network
		network.ID = existing.ID
		return r.db.Save(network).Error
	}
	// Create new network
	return r.db.Create(network).Error
}

// UpsertSubnet creates or updates a subnet
func (r *Repository) UpsertSubnet(subnet *models.Subnet) error {
	subnet.CollectedAt = time.Now()
	return r.db.Save(subnet).Error
}

// UpsertPort creates or updates a port
func (r *Repository) UpsertPort(port *models.Port) error {
	port.CollectedAt = time.Now()
	// Check if port exists by open_stack_id
	var existing models.Port
	err := r.db.Where("open_stack_id = ?", port.OpenStackID).First(&existing).Error
	if err == nil {
		// Update existing port
		port.ID = existing.ID
		return r.db.Save(port).Error
	}
	// Create new port
	return r.db.Create(port).Error
}

// UpsertRouter creates or updates a router
func (r *Repository) UpsertRouter(router *models.Router) error {
	router.CollectedAt = time.Now()
	// Check if router exists by open_stack_id
	var existing models.Router
	err := r.db.Where("open_stack_id = ?", router.OpenStackID).First(&existing).Error
	if err == nil {
		// Update existing router
		router.ID = existing.ID
		return r.db.Save(router).Error
	}
	// Create new router
	return r.db.Create(router).Error
}

// GetNetworkByOpenStackID gets a network by OpenStack ID
func (r *Repository) GetNetworkByOpenStackID(openstackID string) (*models.Network, error) {
	var network models.Network
	err := r.db.Where("open_stack_id = ?", openstackID).First(&network).Error
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
	// Check if hypervisor exists by open_stack_id
	existing, err := r.GetHypervisorByOpenStackID(hypervisor.OpenStackID)
	if err == nil && existing != nil {
		// Update existing hypervisor
		hypervisor.ID = existing.ID
		return r.db.Save(hypervisor).Error
	}
	// Create new hypervisor
	return r.db.Create(hypervisor).Error
}

// GetHypervisorByOpenStackID gets a hypervisor by OpenStack ID
func (r *Repository) GetHypervisorByOpenStackID(openstackID string) (*models.Hypervisor, error) {
	var hypervisor models.Hypervisor
	err := r.db.Where("open_stack_id = ?", openstackID).First(&hypervisor).Error
	if err != nil {
		return nil, err
	}
	return &hypervisor, nil
}

// Flavor operations

// UpsertFlavor creates or updates a flavor
func (r *Repository) UpsertFlavor(flavor *models.Flavor) error {
	flavor.CollectedAt = time.Now()
	// Check if flavor exists by open_stack_id
	existing, err := r.GetFlavorByOpenStackID(flavor.OpenStackID)
	if err == nil && existing != nil {
		// Update existing flavor
		flavor.ID = existing.ID
		return r.db.Save(flavor).Error
	}
	// Create new flavor
	return r.db.Create(flavor).Error
}

// GetFlavorByOpenStackID gets a flavor by OpenStack ID
func (r *Repository) GetFlavorByOpenStackID(openstackID string) (*models.Flavor, error) {
	var flavor models.Flavor
	err := r.db.Where("open_stack_id = ?", openstackID).First(&flavor).Error
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
	
	// Ensure empty strings are converted to NULL for pointer fields
	// This prevents foreign key constraint violations
	if node.InstanceID != nil && *node.InstanceID == "" {
		node.InstanceID = nil
	}
	if node.PortID != nil && *node.PortID == "" {
		node.PortID = nil
	}
	if node.NetworkID != nil && *node.NetworkID == "" {
		node.NetworkID = nil
	}
	if node.RouterID != nil && *node.RouterID == "" {
		node.RouterID = nil
	}
	if node.HypervisorID != nil && *node.HypervisorID == "" {
		node.HypervisorID = nil
	}
	
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
// First finds the instance by OpenStack ID, then finds ports by internal instance ID
func (r *Repository) GetPortsByOpenStackDeviceID(openstackDeviceID string) ([]models.Port, error) {
	// First, find the instance by OpenStack ID
	instance, err := r.GetInstanceByOpenStackID(openstackDeviceID)
	if err != nil {
		// Instance not found, return empty slice
		return []models.Port{}, nil
	}
	
	// Find ports by internal instance ID
	var ports []models.Port
	err = r.db.Where("device_id = ?", instance.ID).Find(&ports).Error
	return ports, err
}

// GetPortsByNetworkID gets all ports for a specific network
func (r *Repository) GetPortsByNetworkID(networkID string) ([]models.Port, error) {
	var ports []models.Port
	err := r.db.Where("network_id = ?", networkID).Find(&ports).Error
	return ports, err
}

// GetRoutersByNetwork gets routers connected to a network (simplified - in production use Neutron API)
func (r *Repository) GetRoutersByNetwork(networkOpenStackID string) ([]models.Router, error) {
	// This is a simplified implementation
	// In production, we'd query Neutron API for router-network connections
	// Router-network connection is through Ports with device_owner like "network:router_interface" or "network:router_gateway"
	
	// First, find the network by OpenStack ID
	var network models.Network
	if err := r.db.Where("open_stack_id = ?", networkOpenStackID).First(&network).Error; err != nil {
		// Network not found, return empty slice
		return []models.Router{}, nil
	}
	
	// Find ports on this network that are router interfaces or gateways
	// Router ports have device_owner starting with "network:router"
	var routerPorts []models.Port
	if err := r.db.Where("network_id = ? AND device_owner LIKE ?", network.ID, "network:router%").Find(&routerPorts).Error; err != nil {
		// No router ports found, return empty slice
		return []models.Router{}, nil
	}
	
	// Router ports don't store router ID in device_id anymore (it's NULL)
	// Instead, we need to find routers by matching their OpenStack ID with port device_owner info
	// For now, return all routers - in production, use Neutron API to get router-network connections
	// This is a simplified implementation
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

