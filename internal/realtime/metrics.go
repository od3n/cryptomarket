package realtime

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Metrics holds Prometheus metrics for the realtime gateway service.
type Metrics struct {
	ActiveConnections  prometheus.Gauge
	TotalConnections   prometheus.Counter
	ConnectionDuration prometheus.Histogram
	EventsConsumed     prometheus.Counter
	EventsBroadcast    prometheus.Counter
	ValidationFailures prometheus.Counter
	RedisReconnects    prometheus.Counter
	ConsumerLag        prometheus.Gauge
	DroppedMessages    prometheus.Counter
	HeartbeatFailures  prometheus.Counter
}

// NewMetrics creates and registers all realtime gateway Prometheus metrics.
func NewMetrics() *Metrics {
	return &Metrics{
		ActiveConnections: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "realtime_active_connections",
			Help: "Number of currently connected SSE clients.",
		}),
		TotalConnections: promauto.NewCounter(prometheus.CounterOpts{
			Name: "realtime_connections_total",
			Help: "Total number of SSE client connections since start.",
		}),
		ConnectionDuration: promauto.NewHistogram(prometheus.HistogramOpts{
			Name:    "realtime_connection_duration_seconds",
			Help:    "Duration of SSE client connections in seconds.",
			Buckets: []float64{1, 5, 15, 30, 60, 120, 300, 600, 1800, 3600},
		}),
		EventsConsumed: promauto.NewCounter(prometheus.CounterOpts{
			Name: "realtime_events_consumed_total",
			Help: "Total number of events consumed from Redis Streams.",
		}),
		EventsBroadcast: promauto.NewCounter(prometheus.CounterOpts{
			Name: "realtime_events_broadcast_total",
			Help: "Total number of events broadcast to clients.",
		}),
		ValidationFailures: promauto.NewCounter(prometheus.CounterOpts{
			Name: "realtime_event_validation_failures_total",
			Help: "Total number of events that failed validation.",
		}),
		RedisReconnects: promauto.NewCounter(prometheus.CounterOpts{
			Name: "realtime_redis_reconnects_total",
			Help: "Total number of Redis reconnection attempts.",
		}),
		ConsumerLag: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "realtime_stream_consumer_lag",
			Help: "Approximate number of pending messages in the consumer group.",
		}),
		DroppedMessages: promauto.NewCounter(prometheus.CounterOpts{
			Name: "realtime_dropped_messages_total",
			Help: "Total number of messages dropped due to slow clients.",
		}),
		HeartbeatFailures: promauto.NewCounter(prometheus.CounterOpts{
			Name: "realtime_heartbeat_failures_total",
			Help: "Total number of heartbeat write failures.",
		}),
	}
}

// Implement stream.MetricsReporter interface.
func (m *Metrics) IncEventsConsumed()         { m.EventsConsumed.Inc() }
func (m *Metrics) IncValidationFailures()     { m.ValidationFailures.Inc() }
func (m *Metrics) IncRedisReconnects()        { m.RedisReconnects.Inc() }
func (m *Metrics) SetConsumerLag(lag float64) { m.ConsumerLag.Set(lag) }

// Implement subscriber.HubMetrics interface.
func (m *Metrics) SetActiveConnections(n int) { m.ActiveConnections.Set(float64(n)) }
func (m *Metrics) IncTotalConnections()       { m.TotalConnections.Inc() }
func (m *Metrics) IncDroppedMessages()        { m.DroppedMessages.Inc() }
