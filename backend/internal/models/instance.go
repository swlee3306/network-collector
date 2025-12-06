package models

import (
	"time"

	"gorm.io/gorm"
)

// Instance represents an OpenStack server instance
type Instance struct {
	ID           string    `gorm:"type:char(36);primaryKey" json:"id"`
	OpenStackID  string    `gorm:"type:varchar(255);uniqueIndex;not null" json:"openstack_id"`
	Name         string    `gorm:"type:varchar(255);not null" json:"name"`
	Status       string    `gorm:"type:varchar(50);not null" json:"status"`
	ProjectID    string    `gorm:"type:char(36);index" json:"project_id"`
	FlavorID     string    `gorm:"type:char(36);index" json:"flavor_id"`
	HypervisorID string    `gorm:"type:varchar(255);index" json:"hypervisor_id"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	CollectedAt  time.Time `gorm:"index" json:"collected_at"`

	// Relationships
	// Note: Hypervisor relationship doesn't use foreign key constraint because HypervisorID
	// needs to be VARCHAR(255) to support OpenStack Host IDs (56-character hashes),
	// not just internal UUIDs. GORM would create it as char(36) if we use foreign key constraint.
	Project    Project           `gorm:"foreignKey:ProjectID" json:"project,omitempty"`
	Flavor     Flavor            `gorm:"foreignKey:FlavorID" json:"flavor,omitempty"`
	Hypervisor Hypervisor        `gorm:"foreignKey:HypervisorID;constraint:-" json:"hypervisor,omitempty"` // No FK constraint
	Metrics    []InstanceMetrics `gorm:"foreignKey:InstanceID" json:"metrics,omitempty"`
}

// TableName specifies the table name
func (Instance) TableName() string {
	return "instances"
}

// BeforeCreate hook
func (i *Instance) BeforeCreate(tx *gorm.DB) error {
	if i.ID == "" {
		i.ID = generateUUID()
	}
	return nil
}
