// path: backend/common/telemetry/metrics.go
package telemetry

import (
	"log"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	// Rate: Total requests by service, operation, and HTTP/gRPC status
	RequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "artisan_requests_total",
			Help: "Total number of operations and requests handled across Artisan Market services (Rate).",
		},
		[]string{"service", "operation", "status"},
	)

	// Errors: Error counter across services and error types
	ErrorsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "artisan_errors_total",
			Help: "Total number of errors encountered during request execution (Errors).",
		},
		[]string{"service", "operation", "error_type"},
	)

	// Duration: Request latency histogram in seconds
	RequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "artisan_request_duration_seconds",
			Help:    "Execution duration histogram for requests and operations (Duration).",
			Buckets: []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10},
		},
		[]string{"service", "operation"},
	)
)

// RecordREDMetrics records the three RED metrics (Rate, Errors, Duration) for an operation.
func RecordREDMetrics(service, operation string, duration time.Duration, err error) {
	status := "success"
	if err != nil {
		status = "error"
		ErrorsTotal.WithLabelValues(service, operation, "internal").Inc()
	}

	RequestsTotal.WithLabelValues(service, operation, status).Inc()
	RequestDuration.WithLabelValues(service, operation).Observe(duration.Seconds())
}

// StartMetricsServer starts a standalone HTTP server on the specified address exposing Prometheus /metrics.
func StartMetricsServer(addr string) *http.Server {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())

	srv := &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	go func() {
		log.Printf("[RED Metrics] Prometheus metrics server listening on %s/metrics", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("[RED Metrics] Server error: %v", err)
		}
	}()

	return srv
}

