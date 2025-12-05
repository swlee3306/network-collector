package models

import (
	"time"

	"gorm.io/gorm"
)

// Volume represents an OpenStack volume
type Volume struct {
	ID          string    `gorm:"type:char(36);primaryKey" json:"id"`
	OpenStackID string    `gorm:"type:varchar(255);uniqueIndex;not null" json:"openstack_id"`
	Name        string    `gorm:"type:varchar(255)" json:"name"`
	Status      string    `gorm:"type:varchar(50);not null" json:"status"`
	Size        int       `gorm:"not null" json:"size"` // GB
	VolumeType  string    `gorm:"type:varchar(100)" json:"volume_type"`
	ProjectID   string    `gorm:"type:char(36);index" json:"project_id"`
	AttachedTo  string    `gorm:"type:char(36);index" json:"attached_to"` // Instance ID
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	CollectedAt time.Time `gorm:"index" json:"collected_at"`

	// Relationships
	Project    Project  `gorm:"foreignKey:ProjectID" json:"project,omitempty"`
	Instance   *Instance `gorm:"foreignKey:AttachedTo" json:"instance,omitempty"`
}

// TableName specifies the table name
func (Volume) TableName() string {
	return "volumes"
}

// BeforeCreate hook
func (v *Volume) BeforeCreate(tx *gorm.DB) error {
	if v.ID == "" {
		v.ID = generateUUID()
	}
	return nil
}

