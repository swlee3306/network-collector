package models

import (
	"time"

	"gorm.io/gorm"
)

// InstanceMetrics represents time-series metrics for an instance
type InstanceMetrics struct {
	ID              string    `gorm:"type:char(36);primaryKey" json:"id"`
	InstanceID      string    `gorm:"type:char(36);index:idx_instance_timestamp;not null" json:"instance_id"`
	Timestamp       time.Time `gorm:"index:idx_instance_timestamp;index:idx_timestamp;not null" json:"timestamp"`
	CPUUsagePercent float64   `gorm:"type:double" json:"cpu_usage_percent"`
	MemoryUsageMB   int64     `gorm:"type:bigint" json:"memory_usage_mb"`
	MemoryTotalMB   int64     `gorm:"type:bigint" json:"memory_total_mb"`
	DiskReadBytes   int64     `gorm:"type:bigint" json:"disk_read_bytes"`
	DiskWriteBytes  int64     `gorm:"type:bigint" json:"disk_write_bytes"`
	NetworkRxBytes  int64     `gorm:"type:bigint" json:"network_rx_bytes"`
	NetworkTxBytes  int64     `gorm:"type:bigint" json:"network_tx_bytes"`
	CreatedAt       time.Time `json:"created_at"`

	// Relationships
	Instance Instance `gorm:"foreignKey:InstanceID" json:"instance,omitempty"`
}

// TableName specifies the table name
func (InstanceMetrics) TableName() string {
	return "instance_metrics"
}

// BeforeCreate hook
func (im *InstanceMetrics) BeforeCreate(tx *gorm.DB) error {
	if im.ID == "" {
		im.ID = generateUUID()
	}
	return nil
}

// NetworkMetrics represents time-series metrics for a network
type NetworkMetrics struct {
	ID                string    `gorm:"type:char(36);primaryKey" json:"id"`
	NetworkID         string    `gorm:"type:char(36);index:idx_network_timestamp;not null" json:"network_id"`
	Timestamp         time.Time `gorm:"index:idx_network_timestamp;index:idx_timestamp;not null" json:"timestamp"`
	TotalBytes        int64     `gorm:"type:bigint" json:"total_bytes"`
	PacketsDropped    int       `gorm:"default:0" json:"packets_dropped"`
	ConnectedVMsCount int       `gorm:"default:0" json:"connected_vms_count"`
	CreatedAt         time.Time `json:"created_at"`

	// Relationships
	Network Network `gorm:"foreignKey:NetworkID" json:"network,omitempty"`
}

// TableName specifies the table name
func (NetworkMetrics) TableName() string {
	return "network_metrics"
}

// BeforeCreate hook
func (nm *NetworkMetrics) BeforeCreate(tx *gorm.DB) error {
	if nm.ID == "" {
		nm.ID = generateUUID()
	}
	return nil
}

// HypervisorMetrics represents time-series metrics for a hypervisor
type HypervisorMetrics struct {
	ID              string    `gorm:"type:char(36);primaryKey" json:"id"`
	HypervisorID    string    `gorm:"type:char(36);index:idx_hypervisor_timestamp;not null" json:"hypervisor_id"`
	Timestamp       time.Time `gorm:"index:idx_hypervisor_timestamp;index:idx_timestamp;not null" json:"timestamp"`
	VCPUsUsed       int       `gorm:"default:0" json:"vcpus_used"`
	VCPUsTotal      int       `gorm:"default:0" json:"vcpus_total"`
	MemoryUsedMB    int64     `gorm:"type:bigint;default:0" json:"memory_used_mb"`
	MemoryTotalMB   int64     `gorm:"type:bigint;default:0" json:"memory_total_mb"`
	RunningVMsCount int       `gorm:"default:0" json:"running_vms_count"`
	CreatedAt       time.Time `json:"created_at"`

	// Relationships
	Hypervisor Hypervisor `gorm:"foreignKey:HypervisorID" json:"hypervisor,omitempty"`
}

// TableName specifies the table name
func (HypervisorMetrics) TableName() string {
	return "hypervisor_metrics"
}

// BeforeCreate hook
func (hm *HypervisorMetrics) BeforeCreate(tx *gorm.DB) error {
	if hm.ID == "" {
		hm.ID = generateUUID()
	}
	return nil
}

