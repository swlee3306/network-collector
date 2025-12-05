package models

import (
	"time"

	"gorm.io/gorm"
)

// Flavor represents an OpenStack flavor (instance type)
type Flavor struct {
	ID          string    `gorm:"type:char(36);primaryKey" json:"id"`
	OpenStackID string    `gorm:"type:varchar(255);uniqueIndex;not null" json:"openstack_id"`
	Name        string    `gorm:"type:varchar(255);uniqueIndex;not null" json:"name"`
	VCPUs       int       `gorm:"not null" json:"vcpus"`
	RAM         int       `gorm:"not null" json:"ram"` // MB
	Disk        int       `gorm:"not null" json:"disk"` // GB
	Ephemeral   int       `gorm:"default:0" json:"ephemeral"` // GB
	Swap        int       `gorm:"default:0" json:"swap"` // MB
	IsPublic    bool      `gorm:"default:true" json:"is_public"`
	CreatedAt   time.Time `json:"created_at"`
	CollectedAt time.Time `gorm:"index" json:"collected_at"`

	// Relationships
	Instances []Instance `gorm:"foreignKey:FlavorID" json:"instances,omitempty"`
}

// TableName specifies the table name
func (Flavor) TableName() string {
	return "flavors"
}

// BeforeCreate hook
func (f *Flavor) BeforeCreate(tx *gorm.DB) error {
	if f.ID == "" {
		f.ID = generateUUID()
	}
	return nil
}

