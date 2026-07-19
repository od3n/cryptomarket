package telemetry

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Metrics holds all Prometheus metrics for the platform.
type Metrics struct {
	HTTPRequestDuration     *prometheus.HistogramVec
	HTTPResponseTotal       *prometheus.CounterVec
	IngestionDuration       prometheus.Histogram
	IngestionSuccess        prometheus.Counter
	IngestionFailure        prometheus.Counter
	ProviderRequestDuration *prometheus.HistogramVec
	DataFreshness           prometheus.Gauge
}

// NewMetrics creates and registers all Prometheus metrics.
func NewMetrics() *Metrics {
	return &Metrics{
		HTTPRequestDuration: promauto.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "Duration of HTTP requests in seconds.",
			Buckets: prometheus.DefBuckets,
		}, []string{"method", "path", "status"}),

		HTTPResponseTotal: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "http_responses_total",
			Help: "Total number of HTTP responses by status code.",
		}, []string{"method", "path", "status"}),

		IngestionDuration: promauto.NewHistogram(prometheus.HistogramOpts{
			Name:    "ingestion_duration_seconds",
			Help:    "Duration of ingestion cycles in seconds.",
			Buckets: []float64{0.1, 0.25, 0.5, 1, 2.5, 5, 10, 30},
		}),

		IngestionSuccess: promauto.NewCounter(prometheus.CounterOpts{
			Name: "ingestion_success_total",
			Help: "Total number of successful ingestion cycles.",
		}),

		IngestionFailure: promauto.NewCounter(prometheus.CounterOpts{
			Name: "ingestion_failure_total",
			Help: "Total number of failed ingestion cycles.",
		}),

		ProviderRequestDuration: promauto.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "provider_request_duration_seconds",
			Help:    "Duration of provider API requests in seconds.",
			Buckets: prometheus.DefBuckets,
		}, []string{"provider", "status"}),

		DataFreshness: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "data_freshness_seconds",
			Help: "Seconds since the last successful data ingestion.",
		}),
	}
}
