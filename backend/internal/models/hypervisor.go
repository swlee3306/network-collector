package models

import (
	"time"

	"gorm.io/gorm"
)

// Hypervisor represents an OpenStack hypervisor
type Hypervisor struct {
	ID            string    `gorm:"type:char(36);primaryKey" json:"id"`
	OpenStackID   string    `gorm:"type:varchar(255);uniqueIndex;not null" json:"openstack_id"`
	Hostname      string    `gorm:"type:varchar(255);uniqueIndex;not null" json:"hostname"`
	HostIP        string    `gorm:"type:varchar(50)" json:"host_ip"`
	Status        string    `gorm:"type:varchar(50);not null" json:"status"`
	State         string    `gorm:"type:varchar(20);not null" json:"state"` // up/down
	VCPUsUsed     int       `gorm:"default:0" json:"vcpus_used"`
	VCPUsTotal    int       `gorm:"default:0" json:"vcpus_total"`
	MemoryUsed    int64     `gorm:"type:bigint;default:0" json:"memory_used"` // MB
	MemoryTotal   int64     `gorm:"type:bigint;default:0" json:"memory_total"` // MB
	LocalGBUsed   int64     `gorm:"type:bigint;default:0" json:"local_gb_used"` // GB
	LocalGBTotal  int64     `gorm:"type:bigint;default:0" json:"local_gb_total"` // GB
	RunningVMs    int       `gorm:"default:0" json:"running_vms"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
	CollectedAt   time.Time `gorm:"index" json:"collected_at"`

	// Relationships
	Instances []Instance `gorm:"foreignKey:HypervisorID" json:"instances,omitempty"`
	Metrics   []HypervisorMetrics `gorm:"foreignKey:HypervisorID" json:"metrics,omitempty"`
}

// TableName specifies the table name
func (Hypervisor) TableName() string {
	return "hypervisors"
}

// BeforeCreate hook
func (h *Hypervisor) BeforeCreate(tx *gorm.DB) error {
	if h.ID == "" {
		h.ID = generateUUID()
	}
	return nil
}

