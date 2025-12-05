package errors

import (
	"fmt"
	"net/http"
)

// APIError represents an API error response
type APIError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
	Type    string `json:"type"`
}

// Error implements the error interface
func (e *APIError) Error() string {
	return fmt.Sprintf("[%d] %s: %s", e.Code, e.Type, e.Message)
}

// NewAPIError creates a new API error
func NewAPIError(code int, message, errorType string) *APIError {
	return &APIError{
		Code:    code,
		Message: message,
		Type:    errorType,
	}
}

// NewAPIErrorWithDetails creates a new API error with details
func NewAPIErrorWithDetails(code int, message, errorType, details string) *APIError {
	return &APIError{
		Code:    code,
		Message: message,
		Type:    errorType,
		Details: details,
	}
}

// DBError represents a database error
type DBError struct {
	Operation string
	Err       error
}

func (e *DBError) Error() string {
	return fmt.Sprintf("database error during %s: %v", e.Operation, e.Err)
}

func (e *DBError) Unwrap() error {
	return e.Err
}

// NewDBError creates a new database error
func NewDBError(operation string, err error) *DBError {
	return &DBError{
		Operation: operation,
		Err:       err,
	}
}

// OpenStackError represents an OpenStack API error
type OpenStackError struct {
	Service   string
	Operation string
	Err       error
	Retryable bool
}

func (e *OpenStackError) Error() string {
	return fmt.Sprintf("openstack error [%s/%s]: %v", e.Service, e.Operation, e.Err)
}

func (e *OpenStackError) Unwrap() error {
	return e.Err
}

// NewOpenStackError creates a new OpenStack error
func NewOpenStackError(service, operation string, err error, retryable bool) *OpenStackError {
	return &OpenStackError{
		Service:   service,
		Operation: operation,
		Err:       err,
		Retryable: retryable,
	}
}

// IsRetryable checks if an error is retryable
func IsRetryable(err error) bool {
	if openstackErr, ok := err.(*OpenStackError); ok {
		return openstackErr.Retryable
	}
	return false
}

// Common error types
const (
	ErrorTypeValidation   = "validation_error"
	ErrorTypeNotFound     = "not_found"
	ErrorTypeUnauthorized = "unauthorized"
	ErrorTypeForbidden    = "forbidden"
	ErrorTypeInternal     = "internal_error"
	ErrorTypeOpenStack    = "openstack_error"
	ErrorTypeDatabase     = "database_error"
	ErrorTypeTimeout      = "timeout_error"
)

// Common error constructors
var (
	ErrNotFound     = NewAPIError(http.StatusNotFound, "Resource not found", ErrorTypeNotFound)
	ErrUnauthorized = NewAPIError(http.StatusUnauthorized, "Unauthorized", ErrorTypeUnauthorized)
	ErrForbidden    = NewAPIError(http.StatusForbidden, "Forbidden", ErrorTypeForbidden)
	ErrInternal     = NewAPIError(http.StatusInternalServerError, "Internal server error", ErrorTypeInternal)
)
