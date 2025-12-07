package models

import (
	"time"

	"gorm.io/gorm"
)

// TopologyNodeType represents the type of topology node
type TopologyNodeType string

const (
	NodeTypeVM       TopologyNodeType = "VM"
	NodeTypePort     TopologyNodeType = "PORT"
	NodeTypeNetwork  TopologyNodeType = "NETWORK"
	NodeTypeRouter   TopologyNodeType = "ROUTER"
	NodeTypeHost     TopologyNodeType = "HOST"
	NodeTypeUnknown  TopologyNodeType = "UNKNOWN"
)

// TopologyNode represents a node in the network topology graph
type TopologyNode struct {
	ID           string          `gorm:"type:char(36);primaryKey" json:"id"`
	NodeType     TopologyNodeType `gorm:"type:varchar(20);not null;index" json:"node_type"`
	OpenStackID  string          `gorm:"type:varchar(255);index" json:"openstack_id"`
	Name         string          `gorm:"type:varchar(255);not null" json:"name"`
	InstanceID   *string         `gorm:"type:char(36);index" json:"instance_id"` // Nullable - only set for VM nodes
	PortID       *string         `gorm:"type:char(36);index" json:"port_id"` // Nullable - only set for Port nodes
	NetworkID    *string         `gorm:"type:char(36);index" json:"network_id"` // Nullable - only set for Network nodes
	RouterID     *string         `gorm:"type:char(36);index" json:"router_id"` // Nullable - only set for Router nodes
	HypervisorID *string         `gorm:"type:char(36);index" json:"hypervisor_id"` // Nullable - only set for Host nodes
	Metadata     string          `gorm:"type:json" json:"metadata"` // JSON object
	IsAccessible bool            `gorm:"default:true" json:"is_accessible"`
	CreatedAt    time.Time       `json:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at"`
	CollectedAt  time.Time       `gorm:"index" json:"collected_at"`

	// Relationships
	Instance   *Instance   `gorm:"foreignKey:InstanceID" json:"instance,omitempty"`
	Port       *Port       `gorm:"foreignKey:PortID" json:"port,omitempty"`
	Network    *Network    `gorm:"foreignKey:NetworkID" json:"network,omitempty"`
	Router     *Router     `gorm:"foreignKey:RouterID" json:"router,omitempty"`
	Hypervisor *Hypervisor `gorm:"foreignKey:HypervisorID" json:"hypervisor,omitempty"`
	
	// Edges
	SourceEdges []TopologyEdge `gorm:"foreignKey:SourceNodeID" json:"source_edges,omitempty"`
	TargetEdges []TopologyEdge `gorm:"foreignKey:TargetNodeID" json:"target_edges,omitempty"`
}

// TableName specifies the table name
func (TopologyNode) TableName() string {
	return "topology_nodes"
}

// BeforeCreate hook
func (tn *TopologyNode) BeforeCreate(tx *gorm.DB) error {
	if tn.ID == "" {
		tn.ID = generateUUID()
	}
	return nil
}

// TopologyEdge represents an edge (connection) in the network topology graph
type TopologyEdge struct {
	ID           string    `gorm:"type:char(36);primaryKey" json:"id"`
	SourceNodeID string    `gorm:"type:char(36);index;not null" json:"source_node_id"`
	TargetNodeID string    `gorm:"type:char(36);index;not null" json:"target_node_id"`
	EdgeType     string    `gorm:"type:varchar(50)" json:"edge_type"` // VIRTUAL, PHYSICAL, etc.
	Metadata     string    `gorm:"type:json" json:"metadata"` // JSON object
	CreatedAt    time.Time `json:"created_at"`
	CollectedAt  time.Time `gorm:"index" json:"collected_at"`

	// Relationships
	SourceNode TopologyNode `gorm:"foreignKey:SourceNodeID" json:"source_node,omitempty"`
	TargetNode TopologyNode `gorm:"foreignKey:TargetNodeID" json:"target_node,omitempty"`
}

// TableName specifies the table name
func (TopologyEdge) TableName() string {
	return "topology_edges"
}

// BeforeCreate hook
func (te *TopologyEdge) BeforeCreate(tx *gorm.DB) error {
	if te.ID == "" {
		te.ID = generateUUID()
	}
	// Validate that source and target are different
	if te.SourceNodeID == te.TargetNodeID {
		return gorm.ErrInvalidValue
	}
	return nil
}

// Index for performance optimization (path finding queries)
// This will be created in migration
// CREATE INDEX idx_topology_edges_source_target ON topology_edges(source_node_id, target_node_id);

