package models

import (
	"time"

	"gorm.io/gorm"
)

// Network represents an OpenStack network resource
type Network struct {
	ID          string    `gorm:"type:char(36);primaryKey" json:"id"`
	OpenStackID string    `gorm:"type:varchar(255);uniqueIndex;not null" json:"openstack_id"`
	Name        string    `gorm:"type:varchar(255);not null" json:"name"`
	Status      string    `gorm:"type:varchar(50);not null" json:"status"`
	ProjectID   string    `gorm:"type:char(36);index" json:"project_id"`
	Shared      bool      `gorm:"default:false" json:"shared"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	CollectedAt time.Time `gorm:"index" json:"collected_at"`

	// Relationships
	Project Project `gorm:"foreignKey:ProjectID" json:"project,omitempty"`
	Subnets []Subnet `gorm:"foreignKey:NetworkID" json:"subnets,omitempty"`
	Ports   []Port   `gorm:"foreignKey:NetworkID" json:"ports,omitempty"`
	Metrics []NetworkMetrics `gorm:"foreignKey:NetworkID" json:"metrics,omitempty"`
}

// TableName specifies the table name
func (Network) TableName() string {
	return "networks"
}

// BeforeCreate hook
func (n *Network) BeforeCreate(tx *gorm.DB) error {
	if n.ID == "" {
		n.ID = generateUUID()
	}
	return nil
}

// Subnet represents an OpenStack subnet
type Subnet struct {
	ID          string    `gorm:"type:char(36);primaryKey" json:"id"`
	OpenStackID string    `gorm:"type:varchar(255);uniqueIndex;not null" json:"openstack_id"`
	NetworkID   string    `gorm:"type:char(36);index;not null" json:"network_id"`
	Name        string    `gorm:"type:varchar(255)" json:"name"`
	CIDR        string    `gorm:"type:varchar(50);not null" json:"cidr"`
	GatewayIP   string    `gorm:"type:varchar(50)" json:"gateway_ip"`
	CreatedAt   time.Time `json:"created_at"`
	CollectedAt time.Time `gorm:"index" json:"collected_at"`

	// Relationships
	Network Network `gorm:"foreignKey:NetworkID" json:"network,omitempty"`
}

// TableName specifies the table name
func (Subnet) TableName() string {
	return "subnets"
}

// BeforeCreate hook
func (s *Subnet) BeforeCreate(tx *gorm.DB) error {
	if s.ID == "" {
		s.ID = generateUUID()
	}
	return nil
}

// Port represents an OpenStack network port
type Port struct {
	ID          string    `gorm:"type:char(36);primaryKey" json:"id"`
	OpenStackID string    `gorm:"type:varchar(255);uniqueIndex;not null" json:"openstack_id"`
	NetworkID   string    `gorm:"type:char(36);index;not null" json:"network_id"`
	DeviceID    *string   `gorm:"type:varchar(255);index" json:"device_id"` // Nullable - can be instance ID or router ID
	DeviceOwner string    `gorm:"type:varchar(100)" json:"device_owner"`
	MACAddress  string    `gorm:"type:varchar(17)" json:"mac_address"`
	Status      string    `gorm:"type:varchar(50)" json:"status"`
	FixedIPs    string    `gorm:"type:json" json:"fixed_ips"` // JSON array of IPs
	CreatedAt   time.Time `json:"created_at"`
	CollectedAt time.Time `gorm:"index" json:"collected_at"`

	// Relationships
	Network Network `gorm:"foreignKey:NetworkID" json:"network,omitempty"`
	// Note: DeviceID can reference either Instance or Router, so no foreign key constraint
}

// TableName specifies the table name
func (Port) TableName() string {
	return "ports"
}

// BeforeCreate hook
func (p *Port) BeforeCreate(tx *gorm.DB) error {
	if p.ID == "" {
		p.ID = generateUUID()
	}
	return nil
}

// Router represents an OpenStack router
type Router struct {
	ID                string    `gorm:"type:char(36);primaryKey" json:"id"`
	OpenStackID       string    `gorm:"type:varchar(255);uniqueIndex;not null" json:"openstack_id"`
	Name              string    `gorm:"type:varchar(255);not null" json:"name"`
	Status            string    `gorm:"type:varchar(50);not null" json:"status"`
	ExternalGatewayInfo string  `gorm:"type:json" json:"external_gateway_info"` // JSON object
	CreatedAt         time.Time `json:"created_at"`
	CollectedAt       time.Time `gorm:"index" json:"collected_at"`
}

// TableName specifies the table name
func (Router) TableName() string {
	return "routers"
}

// BeforeCreate hook
func (r *Router) BeforeCreate(tx *gorm.DB) error {
	if r.ID == "" {
		r.ID = generateUUID()
	}
	return nil
}

