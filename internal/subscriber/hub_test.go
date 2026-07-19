package subscriber

import (
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/crypto-market-platform/internal/stream"
)

func testLogger() *slog.Logger {
	return slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
}

func testEvent(symbol string) *stream.PriceEvent {
	return &stream.PriceEvent{
		EventID:     "evt-" + symbol + "-" + time.Now().Format("150405.000"),
		EventType:   stream.EventTypePriceUpdated,
		Symbol:      symbol,
		PriceUSD:    "50000.00",
		MarketCap:   "1000000000000",
		Volume24h:   "30000000000",
		Change24h:   "1.5",
		Provider:    "coingecko",
		ObservedAt:  time.Now().UTC().Format(time.RFC3339),
		PublishedAt: time.Now().UTC().Format(time.RFC3339),
	}
}

func TestHubSubscribeUnsubscribe(t *testing.T) {
	hub := NewHub(DefaultHubConfig(), nil, testLogger())

	client := hub.Subscribe()
	if client == nil {
		t.Fatal("expected client, got nil")
	}
	if hub.ClientCount() != 1 {
		t.Errorf("expected 1 client, got %d", hub.ClientCount())
	}

	hub.Unsubscribe(client)
	if hub.ClientCount() != 0 {
		t.Errorf("expected 0 clients after unsubscribe, got %d", hub.ClientCount())
	}
}

func TestHubMaxClients(t *testing.T) {
	cfg := HubConfig{MaxClients: 2, ClientBuffer: 8}
	hub := NewHub(cfg, nil, testLogger())

	c1 := hub.Subscribe()
	c2 := hub.Subscribe()
	c3 := hub.Subscribe() // Should be rejected

	if c1 == nil || c2 == nil {
		t.Fatal("expected first two clients to succeed")
	}
	if c3 != nil {
		t.Error("expected third client to be rejected (max 2)")
	}
	if hub.ClientCount() != 2 {
		t.Errorf("expected 2 clients, got %d", hub.ClientCount())
	}
}

func TestHubBroadcast(t *testing.T) {
	hub := NewHub(DefaultHubConfig(), nil, testLogger())

	client := hub.Subscribe()
	if client == nil {
		t.Fatal("expected client")
	}

	event := testEvent("BTC")
	hub.Broadcast(event)

	select {
	case received := <-client.Events():
		if received.Symbol != "BTC" {
			t.Errorf("expected BTC, got %s", received.Symbol)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for broadcast event")
	}
}

func TestHubBroadcastMultipleClients(t *testing.T) {
	hub := NewHub(DefaultHubConfig(), nil, testLogger())

	c1 := hub.Subscribe()
	c2 := hub.Subscribe()

	event := testEvent("ETH")
	hub.Broadcast(event)

	for i, c := range []*Client{c1, c2} {
		select {
		case received := <-c.Events():
			if received.Symbol != "ETH" {
				t.Errorf("client %d: expected ETH, got %s", i, received.Symbol)
			}
		case <-time.After(time.Second):
			t.Fatalf("client %d: timed out", i)
		}
	}
}

func TestHubCloseAll(t *testing.T) {
	hub := NewHub(DefaultHubConfig(), nil, testLogger())

	hub.Subscribe()
	hub.Subscribe()
	hub.Subscribe()

	hub.CloseAll()
	if hub.ClientCount() != 0 {
		t.Errorf("expected 0 clients after CloseAll, got %d", hub.ClientCount())
	}
}

func TestClientSendLatestStateWins(t *testing.T) {
	logger := testLogger()
	// Small buffer to trigger compaction
	client := newClient("test-client", 2, logger)

	// Fill buffer with 2 events for different symbols
	e1 := testEvent("BTC")
	e2 := testEvent("ETH")
	client.send(e1)
	client.send(e2)

	// Buffer is now full. Sending another BTC event should compact.
	e3 := testEvent("BTC")
	e3.PriceUSD = "99999.00"
	result := client.send(e3)
	if !result {
		t.Error("expected send to succeed with compaction")
	}

	// Drain and verify we have latest state per symbol
	events := make(map[string]*stream.PriceEvent)
	for {
		select {
		case ev := <-client.events:
			events[ev.Symbol] = ev
		default:
			goto done
		}
	}
done:

	if len(events) != 2 {
		t.Errorf("expected 2 events after compaction, got %d", len(events))
	}
	if btc, ok := events["BTC"]; ok {
		if btc.PriceUSD != "99999.00" {
			t.Errorf("expected latest BTC price 99999.00, got %s", btc.PriceUSD)
		}
	} else {
		t.Error("expected BTC event in compacted buffer")
	}
}

func TestClientSendToClosedClient(t *testing.T) {
	logger := testLogger()
	client := newClient("test-client", 8, logger)
	client.close()

	result := client.send(testEvent("BTC"))
	if result {
		t.Error("expected send to closed client to return false")
	}
}

func TestClientDone(t *testing.T) {
	logger := testLogger()
	client := newClient("test-client", 8, logger)

	select {
	case <-client.Done():
		t.Fatal("Done channel should not be closed initially")
	default:
	}

	client.close()

	select {
	case <-client.Done():
		// Expected
	case <-time.After(time.Second):
		t.Fatal("Done channel should be closed after close()")
	}
}
