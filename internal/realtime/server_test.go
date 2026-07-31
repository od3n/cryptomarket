package realtime

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/crypto-market-platform/internal/stream"
	"github.com/crypto-market-platform/internal/subscriber"
)

func testLogger() *slog.Logger {
	return slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
}

// newTestMetrics creates Metrics with a custom registry to avoid promauto duplicate panics.
func newTestMetrics() *Metrics {
	return &Metrics{
		ActiveConnections: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "realtime_active_connections",
			Help: "Number of currently connected SSE clients.",
		}),
		TotalConnections: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "realtime_connections_total",
			Help: "Total number of SSE client connections since start.",
		}),
		ConnectionDuration: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name:    "realtime_connection_duration_seconds",
			Help:    "Duration of SSE client connections in seconds.",
			Buckets: []float64{1, 5, 15, 30, 60, 120, 300, 600, 1800, 3600},
		}),
		EventsConsumed: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "realtime_events_consumed_total",
			Help: "Total number of events consumed from Redis Streams.",
		}),
		EventsBroadcast: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "realtime_events_broadcast_total",
			Help: "Total number of events broadcast to clients.",
		}),
		ValidationFailures: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "realtime_event_validation_failures_total",
			Help: "Total number of events that failed validation.",
		}),
		RedisReconnects: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "realtime_redis_reconnects_total",
			Help: "Total number of Redis reconnection attempts.",
		}),
		ConsumerLag: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "realtime_stream_consumer_lag",
			Help: "Approximate number of pending messages in the consumer group.",
		}),
		DroppedMessages: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "realtime_dropped_messages_total",
			Help: "Total number of messages dropped due to slow clients.",
		}),
		HeartbeatFailures: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "realtime_heartbeat_failures_total",
			Help: "Total number of heartbeat write failures.",
		}),
	}
}

func testContext() (context.Context, context.CancelFunc) {
	return context.WithCancel(context.Background())
}

func testCtx() context.Context {
	return context.Background()
}

func testPriceEvent() *stream.PriceEvent {
	return &stream.PriceEvent{
		EventID:     "test-evt-001",
		EventType:   stream.EventTypePriceUpdated,
		Symbol:      "BTC",
		PriceUSD:    "65000.50",
		MarketCap:   "1300000000000",
		Volume24h:   "50000000000",
		Change24h:   "2.5",
		Provider:    "coingecko",
		ObservedAt:  time.Now().UTC().Format(time.RFC3339),
		PublishedAt: time.Now().UTC().Format(time.RFC3339),
	}
}

func testStreamMessage(event *stream.PriceEvent) stream.StreamMessage {
	return stream.StreamMessage{
		StreamID: "1234567890-0",
		Event:    event,
	}
}

func TestHealthEndpoint(t *testing.T) {
	hub := subscriber.NewHub(subscriber.DefaultHubConfig(), nil, testLogger())
	metrics := newTestMetrics()
	server := NewServer(hub, nil, metrics, testLogger())

	req := httptest.NewRequestWithContext(context.Background(), "GET", "/health", nil)
	w := httptest.NewRecorder()

	server.Health(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if w.Body.String() != `{"status":"ok"}` {
		t.Errorf("unexpected body: %s", w.Body.String())
	}
}

func TestSSEHeaders(t *testing.T) {
	hub := subscriber.NewHub(subscriber.DefaultHubConfig(), nil, testLogger())
	metrics := newTestMetrics()
	server := NewServer(hub, nil, metrics, testLogger())

	req := httptest.NewRequestWithContext(context.Background(), "GET", "/events/markets", nil)
	w := httptest.NewRecorder()

	// The SSE handler will block, so we test via a goroutine and cancel
	ctx, cancel := testContext()
	defer cancel()
	req = req.WithContext(ctx)

	done := make(chan struct{})
	go func() {
		server.EventsMarkets(w, req)
		close(done)
	}()

	// Give it a moment to write headers
	time.Sleep(50 * time.Millisecond)
	cancel()
	<-done

	contentType := w.Header().Get("Content-Type")
	if contentType != "text/event-stream" {
		t.Errorf("expected Content-Type text/event-stream, got %s", contentType)
	}
	cacheControl := w.Header().Get("Cache-Control")
	if cacheControl != "no-cache" {
		t.Errorf("expected Cache-Control no-cache, got %s", cacheControl)
	}
}

func TestConnectionLimitReached(t *testing.T) {
	cfg := subscriber.HubConfig{MaxClients: 0, ClientBuffer: 8}
	hub := subscriber.NewHub(cfg, nil, testLogger())
	metrics := newTestMetrics()
	server := NewServer(hub, nil, metrics, testLogger())

	req := httptest.NewRequestWithContext(context.Background(), "GET", "/events/markets", nil)
	w := httptest.NewRecorder()

	server.EventsMarkets(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503 when connection limit reached, got %d", w.Code)
	}
}

func TestRouter(t *testing.T) {
	hub := subscriber.NewHub(subscriber.DefaultHubConfig(), nil, testLogger())
	metrics := newTestMetrics()
	server := NewServer(hub, nil, metrics, testLogger())

	router := server.Router()

	// Test /health route
	req := httptest.NewRequestWithContext(context.Background(), "GET", "/health", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for /health, got %d", w.Code)
	}
}

func TestHandleEvent(t *testing.T) {
	hub := subscriber.NewHub(subscriber.DefaultHubConfig(), nil, testLogger())
	metrics := newTestMetrics()
	server := NewServer(hub, nil, metrics, testLogger())

	client := hub.Subscribe()
	if client == nil {
		t.Fatal("expected client")
	}

	event := testPriceEvent()
	msg := testStreamMessage(event)

	err := server.HandleEvent(testCtx(), msg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	select {
	case received := <-client.Events():
		if received.Symbol != "BTC" {
			t.Errorf("expected BTC, got %s", received.Symbol)
		}
	case <-time.After(time.Second):
		t.Error("timed out waiting for broadcast event")
	}
}
