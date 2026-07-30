// Package subscriber implements the SSE client hub for broadcasting market events.
package subscriber

import (
	"log/slog"
	"sync"
	"sync/atomic"

	"github.com/crypto-market-platform/internal/stream"
)

const (
	// DefaultMaxClients is the default maximum number of concurrent SSE connections.
	DefaultMaxClients = 100
	// DefaultClientBufferSize is the per-client event buffer size.
	DefaultClientBufferSize = 64
)

// HubConfig holds configuration for the subscriber hub.
type HubConfig struct {
	MaxClients   int
	ClientBuffer int
}

// DefaultHubConfig returns sensible defaults.
func DefaultHubConfig() HubConfig {
	return HubConfig{
		MaxClients:   DefaultMaxClients,
		ClientBuffer: DefaultClientBufferSize,
	}
}

// HubMetrics allows the hub to report metrics.
type HubMetrics interface {
	SetActiveConnections(n int)
	IncTotalConnections()
	IncDroppedMessages()
}

// Hub manages SSE client subscriptions and broadcasts events.
// It implements a latest-state-wins policy: when a client buffer is full,
// older intermediate updates per symbol are dropped in favor of the latest.
type Hub struct {
	cfg     HubConfig
	logger  *slog.Logger
	metrics HubMetrics

	mu      sync.RWMutex
	clients map[string]*Client
	nextID  atomic.Int64
}

// NewHub creates a new subscriber hub.
func NewHub(cfg HubConfig, metrics HubMetrics, logger *slog.Logger) *Hub {
	return &Hub{
		cfg:     cfg,
		logger:  logger,
		metrics: metrics,
		clients: make(map[string]*Client),
	}
}

// Subscribe registers a new client and returns it. Returns nil if at capacity.
func (h *Hub) Subscribe() *Client {
	h.mu.Lock()
	defer h.mu.Unlock()

	if len(h.clients) >= h.cfg.MaxClients {
		h.logger.Warn("connection limit reached, rejecting client",
			slog.Int("max", h.cfg.MaxClients))
		return nil
	}

	id := h.nextID.Add(1)
	clientID := formatClientID(id)
	client := newClient(clientID, h.cfg.ClientBuffer, h.logger)

	h.clients[clientID] = client
	if h.metrics != nil {
		h.metrics.IncTotalConnections()
		h.metrics.SetActiveConnections(len(h.clients))
	}

	h.logger.Debug("client subscribed", slog.String("client_id", clientID), slog.Int("total", len(h.clients)))
	return client
}

// Unsubscribe removes a client from the hub.
func (h *Hub) Unsubscribe(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if _, ok := h.clients[client.ID()]; ok {
		delete(h.clients, client.ID())
		client.close()
		if h.metrics != nil {
			h.metrics.SetActiveConnections(len(h.clients))
		}
		h.logger.Debug("client unsubscribed", slog.String("client_id", client.ID()), slog.Int("total", len(h.clients)))
	}
}

// Broadcast sends an event to all connected clients using latest-state-wins policy.
func (h *Hub) Broadcast(event *stream.PriceEvent) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for _, client := range h.clients {
		if !client.send(event) {
			if h.metrics != nil {
				h.metrics.IncDroppedMessages()
			}
		}
	}
}

// ClientCount returns the current number of connected clients.
func (h *Hub) ClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

// CloseAll disconnects all clients (used during shutdown).
func (h *Hub) CloseAll() {
	h.mu.Lock()
	defer h.mu.Unlock()

	for id, client := range h.clients {
		client.close()
		delete(h.clients, id)
	}
	if h.metrics != nil {
		h.metrics.SetActiveConnections(0)
	}
	h.logger.Info("all clients disconnected")
}

func formatClientID(id int64) string {
	return "client-" + itoa(id)
}

func itoa(n int64) string {
	if n == 0 {
		return "0"
	}
	buf := make([]byte, 0, 20)
	for n > 0 {
		buf = append(buf, byte('0'+n%10))
		n /= 10
	}
	// Reverse.
	for i, j := 0, len(buf)-1; i < j; i, j = i+1, j-1 {
		buf[i], buf[j] = buf[j], buf[i]
	}
	return string(buf)
}
