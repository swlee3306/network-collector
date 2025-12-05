package models

import (
	"time"

	"gorm.io/gorm"
)

// Project represents an OpenStack project
type Project struct {
	ID          string    `gorm:"type:char(36);primaryKey" json:"id"`
	OpenStackID string    `gorm:"type:varchar(255);uniqueIndex;not null" json:"openstack_id"`
	Name        string    `gorm:"type:varchar(255);not null" json:"name"`
	Description string    `gorm:"type:text" json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	CollectedAt time.Time `gorm:"index" json:"collected_at"`

	// Relationships
	Instances []Instance `gorm:"foreignKey:ProjectID" json:"instances,omitempty"`
	Networks  []Network  `gorm:"foreignKey:ProjectID" json:"networks,omitempty"`
	Volumes   []Volume   `gorm:"foreignKey:ProjectID" json:"volumes,omitempty"`
}

// TableName specifies the table name
func (Project) TableName() string {
	return "projects"
}

// BeforeCreate hook
func (p *Project) BeforeCreate(tx *gorm.DB) error {
	if p.ID == "" {
		p.ID = generateUUID()
	}
	return nil
}

