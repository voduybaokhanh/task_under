package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// HTTPRequestsTotal counts HTTP requests labelled by method, route and status.
	HTTPRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests.",
		},
		[]string{"method", "path", "status"},
	)

	// HTTPRequestDuration observes request latency in seconds.
	HTTPRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request latency in seconds.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "path", "status"},
	)

	// WSConnectionsActive tracks the number of currently connected WebSocket clients.
	WSConnectionsActive = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "ws_connections_active",
			Help: "Number of active WebSocket connections.",
		},
	)

	// TasksCreatedTotal counts created tasks (business metric).
	TasksCreatedTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "tasks_created_total",
			Help: "Total number of tasks created.",
		},
	)

	// TasksCompletedTotal counts completed tasks (business metric).
	TasksCompletedTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "tasks_completed_total",
			Help: "Total number of tasks completed.",
		},
	)
)
