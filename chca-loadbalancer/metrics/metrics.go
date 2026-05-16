package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// RequestsTotal counts total requests routed to each backend.
	RequestsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "chca_requests_total",
		Help: "Total number of requests routed to each backend.",
	}, []string{"backend"})

	// ActiveConnections tracks live active connections per backend.
	ActiveConnections = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "chca_active_connections",
		Help: "Number of currently active connections per backend.",
	}, []string{"backend"})

	// RequestDuration observes request latency per backend.
	RequestDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "chca_request_duration_seconds",
		Help:    "Histogram of request durations per backend.",
		Buckets: prometheus.DefBuckets,
	}, []string{"backend"})

	// CongestionSkips counts how many times a congested backend was skipped.
	CongestionSkips = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "chca_congestion_skips_total",
		Help: "Total number of times a congested backend was skipped.",
	}, []string{"backend"})

	// HealthyBackends tracks the number of healthy backends.
	HealthyBackends = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "chca_healthy_backends",
		Help: "Number of currently healthy backends.",
	})

	// FallbackEvents counts requests that fell back to least-loaded selection.
	FallbackEvents = promauto.NewCounter(prometheus.CounterOpts{
		Name: "chca_fallback_events_total",
		Help: "Total requests that used fallback (all backends overloaded).",
	})
)
