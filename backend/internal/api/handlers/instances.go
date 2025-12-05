package handlers

import (
	"github.com/gin-gonic/gin"
	"github.com/network-collector/backend/internal/services/storage"
	"github.com/network-collector/backend/pkg/errors"
	"gorm.io/gorm"
)

// ListInstances lists all instances
func ListInstances(repo *storage.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		instances, err := repo.ListInstances()
		if err != nil {
			c.Error(errors.NewDBError("list instances", err))
			return
		}

		c.JSON(200, gin.H{
			"data": instances,
		})
	}
}

// GetInstance gets a specific instance by ID
func GetInstance(repo *storage.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")

		instance, err := repo.GetInstanceByID(id)
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				c.Error(errors.ErrNotFound)
			} else {
				c.Error(errors.NewDBError("get instance", err))
			}
			return
		}

		c.JSON(200, gin.H{
			"data": instance,
		})
	}
}

