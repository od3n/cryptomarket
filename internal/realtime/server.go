package realtime

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"

	"github.com/crypto-market-platform/internal/stream"
	"github.com/crypto-market-platform/internal/subscriber"
)

// Server is the realtime gateway HTTP server.
type Server struct {
	hub     *subscriber.Hub
	redis   *redis.Client
	metrics *Metrics
	logger  *slog.Logger
}

// NewServer creates a new realtime server.
func NewServer(hub *subscriber.Hub, redisClient *redis.Client, metrics *Metrics, logger *slog.Logger) *Server {
	return &Server{
		hub:     hub,
		redis:   redisClient,
		metrics: metrics,
		logger:  logger,
	}
}

// Router creates the HTTP router for the realtime service.
func (s *Server) Router() http.Handler {
	r := chi.NewRouter()

	r.Get("/health", s.Health)
	r.Get("/ready", s.Ready)
	r.Get("/events/markets", s.EventsMarkets)
	r.Method("GET", "/metrics", promhttp.Handler())

	return r
}

// Health returns a simple liveness check.
func (s *Server) Health(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"ok"}`))
}

// Ready checks Redis connectivity.
func (s *Server) Ready(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if err := s.redis.Ping(ctx).Err(); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte(`{"status":"not_ready","error":"redis unavailable"}`))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"ready"}`))
}

// EventsMarkets is the SSE endpoint for live market price updates.
func (s *Server) EventsMarkets(w http.ResponseWriter, r *http.Request) {
	client := s.hub.Subscribe()
	if client == nil {
		http.Error(w, "connection limit reached", http.StatusServiceUnavailable)
		return
	}

	start := time.Now()
	lastEventID := r.Header.Get("Last-Event-ID")

	s.logger.Info("client connected",
		slog.String("client_id", client.ID()),
		slog.String("last_event_id", lastEventID))

	// If Last-Event-ID is provided, replay missed events from the stream.
	if lastEventID != "" {
		s.replayEvents(r.Context(), client, lastEventID)
	}

	// Serve SSE (blocks until client disconnects or hub closes).
	client.ServeSSE(w, r, lastEventID)

	// Client disconnected.
	duration := time.Since(start)
	s.metrics.ConnectionDuration.Observe(duration.Seconds())
	s.hub.Unsubscribe(client)

	s.logger.Info("client disconnected",
		slog.String("client_id", client.ID()),
		slog.Duration("duration", duration))
}

// replayEvents sends missed events from the Redis Stream after the given stream ID.
func (s *Server) replayEvents(ctx context.Context, client *subscriber.Client, afterID string) {
	entries, err := s.redis.XRange(ctx, stream.StreamKey, afterID, "+").Result()
	if err != nil {
		s.logger.Warn("failed to replay events",
			slog.String("after_id", afterID),
			slog.String("error", err.Error()))
		return
	}

	// Limit replay to avoid flooding a reconnecting client.
	const maxReplay = 50
	if len(entries) > maxReplay {
		entries = entries[len(entries)-maxReplay:]
	}

	replayed := 0
	for _, entry := range entries {
		if entry.ID == afterID {
			continue
		}
		dataStr, ok := entry.Values["data"].(string)
		if !ok {
			continue
		}
		event, err := stream.ParseEvent([]byte(dataStr))
		if err != nil || event.Validate() != nil {
			continue
		}
		client.SendReplay(event)
		replayed++
	}

	if replayed > 0 {
		s.logger.Debug("replayed events",
			slog.String("client_id", client.ID()),
			slog.Int("count", replayed))
	}
}

// HandleEvent is the stream event handler that broadcasts to all clients.
func (s *Server) HandleEvent(ctx context.Context, msg stream.StreamMessage) error {
	s.hub.Broadcast(msg.Event)
	s.metrics.EventsBroadcast.Inc()
	return nil
}
