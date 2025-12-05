package metrics

import (
	"strconv"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// Collection metrics
	CollectionTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "openstack_collection_total",
			Help: "Total number of collection operations",
		},
		[]string{"resource_type", "status"}, // status: success, failure
	)

	CollectionDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "openstack_collection_duration_seconds",
			Help:    "Duration of collection operations in seconds",
			Buckets: prometheus.ExponentialBuckets(0.1, 2, 10), // 0.1s to ~102.4s
		},
		[]string{"resource_type"},
	)

	CollectionErrors = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "openstack_collection_errors_total",
			Help: "Total number of collection errors",
		},
		[]string{"resource_type", "error_type"},
	)

	// API metrics
	HTTPRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "openstack_api_http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "endpoint", "status_code"},
	)

	HTTPRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "openstack_api_http_request_duration_seconds",
			Help:    "HTTP request duration in seconds",
			Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10},
		},
		[]string{"method", "endpoint"},
	)

	HTTPRequestSize = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "openstack_api_http_request_size_bytes",
			Help:    "HTTP request size in bytes",
			Buckets: prometheus.ExponentialBuckets(100, 10, 6), // 100B to ~100MB
		},
		[]string{"method", "endpoint"},
	)

	HTTPResponseSize = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "openstack_api_http_response_size_bytes",
			Help:    "HTTP response size in bytes",
			Buckets: prometheus.ExponentialBuckets(100, 10, 6), // 100B to ~100MB
		},
		[]string{"method", "endpoint"},
	)

	// Database metrics
	DatabaseQueriesTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "openstack_database_queries_total",
			Help: "Total number of database queries",
		},
		[]string{"operation", "status"}, // operation: select, insert, update, delete
	)

	DatabaseQueryDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "openstack_database_query_duration_seconds",
			Help:    "Database query duration in seconds",
			Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5},
		},
		[]string{"operation"},
	)
)

// RecordCollection records a collection operation
func RecordCollection(resourceType string, success bool, duration float64) {
	status := "failure"
	if success {
		status = "success"
	}
	CollectionTotal.WithLabelValues(resourceType, status).Inc()
	CollectionDuration.WithLabelValues(resourceType).Observe(duration)
}

// RecordCollectionError records a collection error
func RecordCollectionError(resourceType string, errorType string) {
	CollectionErrors.WithLabelValues(resourceType, errorType).Inc()
}

// RecordHTTPRequest records an HTTP request
func RecordHTTPRequest(method, endpoint string, statusCode int, duration, requestSize, responseSize float64) {
	statusCodeStr := strconv.Itoa(statusCode)
	HTTPRequestsTotal.WithLabelValues(method, endpoint, statusCodeStr).Inc()
	HTTPRequestDuration.WithLabelValues(method, endpoint).Observe(duration)
	HTTPRequestSize.WithLabelValues(method, endpoint).Observe(requestSize)
	HTTPResponseSize.WithLabelValues(method, endpoint).Observe(responseSize)
}

// RecordDatabaseQuery records a database query
func RecordDatabaseQuery(operation string, success bool, duration float64) {
	status := "failure"
	if success {
		status = "success"
	}
	DatabaseQueriesTotal.WithLabelValues(operation, status).Inc()
	DatabaseQueryDuration.WithLabelValues(operation).Observe(duration)
}

