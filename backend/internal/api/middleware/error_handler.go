package middleware

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/network-collector/backend/pkg/errors"
	"gorm.io/gorm"
)

// ErrorHandler is a middleware that handles errors consistently
func ErrorHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		// Check if there are any errors
		if len(c.Errors) > 0 {
			err := c.Errors.Last().Err

			// Handle different error types
			switch e := err.(type) {
			case *errors.APIError:
				c.JSON(e.Code, gin.H{
					"error": e,
				})
				return

			case *errors.DBError:
				log.Printf("Database error: %v", err)
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": errors.NewAPIError(
						http.StatusInternalServerError,
						"Database operation failed",
						errors.ErrorTypeDatabase,
					),
				})
				return

			case *errors.OpenStackError:
				log.Printf("OpenStack error: %v", err)
				c.JSON(http.StatusBadGateway, gin.H{
					"error": errors.NewAPIErrorWithDetails(
						http.StatusBadGateway,
						"OpenStack service error",
						errors.ErrorTypeOpenStack,
						e.Error(),
					),
				})
				return

			default:
				// Check if it's a GORM record not found error
				if err == gorm.ErrRecordNotFound {
					c.JSON(http.StatusNotFound, gin.H{
						"error": errors.ErrNotFound,
					})
					return
				}
				// Other errors
				log.Printf("Unexpected error: %v", err)
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": errors.ErrInternal,
				})
				return
			}
		}
	}
}

