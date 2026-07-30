package subscriber

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/crypto-market-platform/internal/stream"
)

const (
	// HeartbeatInterval is how often SSE heartbeat comments are sent.
	HeartbeatInterval = 15 * time.Second
	// WriteTimeout is the maximum time to wait for a write before considering the client dead.
	WriteTimeout = 10 * time.Second
)

// Client represents a connected SSE client with a bounded event buffer.
// Buffer policy: latest-state-wins per symbol. When the buffer is full,
// older events for the same symbol are replaced with the latest.
type Client struct {
	id     string
	events chan *stream.PriceEvent
	done   chan struct{}
	logger *slog.Logger

	mu     sync.Mutex
	closed bool

	// latestPerSymbol tracks the latest event per symbol for dedup in buffer.
	latestPerSymbol map[string]*stream.PriceEvent
	bufferSize      int
}

func newClient(id string, bufferSize int, logger *slog.Logger) *Client {
	return &Client{
		id:              id,
		events:          make(chan *stream.PriceEvent, bufferSize),
		done:            make(chan struct{}),
		logger:          logger,
		latestPerSymbol: make(map[string]*stream.PriceEvent),
		bufferSize:      bufferSize,
	}
}

// ID returns the client identifier.
func (c *Client) ID() string {
	return c.id
}

// Events returns the channel of events for this client.
func (c *Client) Events() <-chan *stream.PriceEvent {
	return c.events
}

// Done returns a channel closed when the client is disconnected.
func (c *Client) Done() <-chan struct{} {
	return c.done
}

// send attempts to enqueue an event. Implements latest-state-wins:
// if the buffer is full, it replaces the oldest event for the same symbol.
// Returns false if the event was dropped entirely.
func (c *Client) send(event *stream.PriceEvent) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return false
	}

	// Try non-blocking send first.
	select {
	case c.events <- event:
		c.latestPerSymbol[event.Symbol] = event
		return true
	default:
	}

	// Buffer full: apply latest-state-wins policy.
	// Drain the buffer, keeping only the latest per symbol.
	latest := make(map[string]*stream.PriceEvent)
	close(c.events)
	for ev := range c.events {
		latest[ev.Symbol] = ev
	}
	// Update with the new event.
	latest[event.Symbol] = event

	// Recreate the channel with compacted state.
	c.events = make(chan *stream.PriceEvent, c.bufferSize)
	for _, ev := range latest {
		c.events <- ev
	}
	c.latestPerSymbol = latest

	return true
}

// SendReplay sends an event directly to the client buffer (used for Last-Event-ID replay).
func (c *Client) SendReplay(event *stream.PriceEvent) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return
	}
	select {
	case c.events <- event:
	default:
		// Drop if buffer full during replay.
	}
}

func (c *Client) close() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.closed {
		c.closed = true
		close(c.done)
	}
}

// ServeSSE writes SSE events to the HTTP response writer.
// It handles heartbeats, client disconnects, and graceful shutdown.
// The last event ID is accepted for future replay support but not yet used
// (see ADR-005: latest-state delivery policy).
func (c *Client) ServeSSE(w http.ResponseWriter, r *http.Request, _ string) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}

	// Set SSE headers.
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)
	flusher.Flush()

	heartbeat := time.NewTicker(HeartbeatInterval)
	defer heartbeat.Stop()

	ctx := r.Context()

	for {
		select {
		case <-ctx.Done():
			return
		case <-c.done:
			return
		case <-heartbeat.C:
			// Send heartbeat comment.
			if _, err := fmt.Fprintf(w, ":heartbeat\n\n"); err != nil {
				return
			}
			flusher.Flush()
		case event, ok := <-c.events:
			if !ok {
				return
			}
			if err := writeSSEEvent(w, event); err != nil {
				return
			}
			flusher.Flush()
		}
	}
}

// writeSSEEvent writes a single SSE-formatted event.
func writeSSEEvent(w http.ResponseWriter, event *stream.PriceEvent) error {
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}

	_, err = fmt.Fprintf(w, "event: %s\nid: %s\ndata: %s\n\n", event.EventType, event.EventID, string(data))
	return err
}
