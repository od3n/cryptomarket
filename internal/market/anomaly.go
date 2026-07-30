package market

import (
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var anomalyDetectedTotal = promauto.NewCounterVec(prometheus.CounterOpts{
	Name: "anomaly_detected_total",
	Help: "Total number of anomalies detected by type and provider.",
}, []string{"type", "provider"})

// AnomalyType represents the type of anomaly detected.
type AnomalyType string

const (
	// AnomalyPriceDivergence indicates provider price difference above threshold.
	AnomalyPriceDivergence AnomalyType = "price_divergence"
	// AnomalyTimestampRegression indicates a sudden timestamp regression.
	AnomalyTimestampRegression AnomalyType = "timestamp_regression"
	// AnomalyUnchangedPrice indicates unchanged price for unusually long duration.
	AnomalyUnchangedPrice AnomalyType = "unchanged_price"
	// AnomalyMissingAssets indicates missing assets from provider response.
	AnomalyMissingAssets AnomalyType = "missing_assets"
	// AnomalyHighLatency indicates provider response latency above threshold.
	AnomalyHighLatency AnomalyType = "high_latency"
	// AnomalySustainedFallback indicates sustained fallback usage.
	AnomalySustainedFallback AnomalyType = "sustained_fallback"
)

// AnomalyConfig holds configuration for anomaly detection.
type AnomalyConfig struct {
	// PriceDivergenceThreshold is the percentage difference that triggers an anomaly.
	PriceDivergenceThreshold float64
	// UnchangedPriceDuration is how long a price can remain unchanged before anomaly.
	UnchangedPriceDuration time.Duration
	// HighLatencyThreshold is the response time that triggers a latency anomaly.
	HighLatencyThreshold time.Duration
	// SustainedFallbackDuration is how long fallback can be active before anomaly.
	SustainedFallbackDuration time.Duration
	// ConsecutiveEventThreshold is how many consecutive events before alerting.
	ConsecutiveEventThreshold int
}

// DefaultAnomalyConfig returns sensible defaults.
func DefaultAnomalyConfig() AnomalyConfig {
	return AnomalyConfig{
		PriceDivergenceThreshold:  5.0, // 5%
		UnchangedPriceDuration:    10 * time.Minute,
		HighLatencyThreshold:      5 * time.Second,
		SustainedFallbackDuration: 5 * time.Minute,
		ConsecutiveEventThreshold: 3,
	}
}

// Anomaly represents a detected anomaly.
type Anomaly struct {
	Type       AnomalyType `json:"type"`
	Provider   string      `json:"provider"`
	Symbol     string      `json:"symbol,omitempty"`
	Message    string      `json:"message"`
	DetectedAt time.Time   `json:"detected_at"`
	Value      float64     `json:"value,omitempty"`
}

// AnomalyDetector detects anomalies in market data.
type AnomalyDetector struct {
	mu     sync.RWMutex
	config AnomalyConfig

	// Track state for anomaly detection
	lastPrices       map[string]priceRecord
	lastTimestamps   map[string]time.Time
	fallbackStart    time.Time
	consecutiveCount map[string]int
	recentAnomalies  []Anomaly
}

type priceRecord struct {
	price     float64
	provider  string
	timestamp time.Time
}

// NewAnomalyDetector creates a new anomaly detector.
func NewAnomalyDetector(config AnomalyConfig) *AnomalyDetector {
	return &AnomalyDetector{
		config:           config,
		lastPrices:       make(map[string]priceRecord),
		lastTimestamps:   make(map[string]time.Time),
		consecutiveCount: make(map[string]int),
		recentAnomalies:  make([]Anomaly, 0, 100),
	}
}

// CheckPriceDivergence checks if prices from different providers diverge too much.
func (ad *AnomalyDetector) CheckPriceDivergence(symbol string, provider string, price float64) *Anomaly {
	ad.mu.Lock()
	defer ad.mu.Unlock()

	key := symbol
	lastRecord, exists := ad.lastPrices[key]

	if exists && lastRecord.provider != provider {
		// Calculate percentage difference
		avgPrice := (price + lastRecord.price) / 2
		if avgPrice > 0 {
			diff := abs(price-lastRecord.price) / avgPrice * 100
			if diff > ad.config.PriceDivergenceThreshold {
				anomaly := Anomaly{
					Type:       AnomalyPriceDivergence,
					Provider:   provider,
					Symbol:     symbol,
					Message:    "Price divergence between providers exceeds threshold",
					DetectedAt: time.Now(),
					Value:      diff,
				}
				ad.recordAnomaly(anomaly)
				return &anomaly
			}
		}
	}

	// Update last price
	ad.lastPrices[key] = priceRecord{
		price:     price,
		provider:  provider,
		timestamp: time.Now(),
	}
	return nil
}

// CheckTimestampRegression checks if timestamp went backwards.
func (ad *AnomalyDetector) CheckTimestampRegression(symbol string, provider string, timestamp time.Time) *Anomaly {
	ad.mu.Lock()
	defer ad.mu.Unlock()

	key := symbol + ":" + provider
	lastTs, exists := ad.lastTimestamps[key]

	if exists && timestamp.Before(lastTs) {
		anomaly := Anomaly{
			Type:       AnomalyTimestampRegression,
			Provider:   provider,
			Symbol:     symbol,
			Message:    "Timestamp regression detected",
			DetectedAt: time.Now(),
		}
		ad.recordAnomaly(anomaly)
		ad.lastTimestamps[key] = timestamp
		return &anomaly
	}

	ad.lastTimestamps[key] = timestamp
	return nil
}

// CheckUnchangedPrice checks if price has been unchanged for too long.
func (ad *AnomalyDetector) CheckUnchangedPrice(symbol string, provider string, price float64) *Anomaly {
	ad.mu.Lock()
	defer ad.mu.Unlock()

	key := symbol + ":" + provider
	lastRecord, exists := ad.lastPrices[key]

	if exists && lastRecord.price == price && lastRecord.provider == provider {
		duration := time.Since(lastRecord.timestamp)
		if duration > ad.config.UnchangedPriceDuration {
			anomaly := Anomaly{
				Type:       AnomalyUnchangedPrice,
				Provider:   provider,
				Symbol:     symbol,
				Message:    "Price unchanged for unusually long duration",
				DetectedAt: time.Now(),
				Value:      duration.Seconds(),
			}
			ad.recordAnomaly(anomaly)
			return &anomaly
		}
	}

	return nil
}

// CheckMissingAssets checks if expected assets are missing from response.
func (ad *AnomalyDetector) CheckMissingAssets(provider string, expectedSymbols []string, receivedSymbols []string) *Anomaly {
	received := make(map[string]bool)
	for _, s := range receivedSymbols {
		received[s] = true
	}

	var missing []string
	for _, s := range expectedSymbols {
		if !received[s] {
			missing = append(missing, s)
		}
	}

	if len(missing) > 0 {
		anomaly := Anomaly{
			Type:       AnomalyMissingAssets,
			Provider:   provider,
			Message:    "Missing assets from provider response",
			DetectedAt: time.Now(),
			Value:      float64(len(missing)),
		}
		ad.recordAnomaly(anomaly)
		return &anomaly
	}
	return nil
}

// CheckHighLatency checks if provider response latency is too high.
func (ad *AnomalyDetector) CheckHighLatency(provider string, latency time.Duration) *Anomaly {
	if latency > ad.config.HighLatencyThreshold {
		anomaly := Anomaly{
			Type:       AnomalyHighLatency,
			Provider:   provider,
			Message:    "Provider response latency above threshold",
			DetectedAt: time.Now(),
			Value:      latency.Seconds(),
		}
		ad.mu.Lock()
		ad.recordAnomaly(anomaly)
		ad.mu.Unlock()
		return &anomaly
	}
	return nil
}

// RecordFallbackStart records when fallback mode started.
func (ad *AnomalyDetector) RecordFallbackStart() {
	ad.mu.Lock()
	defer ad.mu.Unlock()
	if ad.fallbackStart.IsZero() {
		ad.fallbackStart = time.Now()
	}
}

// RecordFallbackEnd records when fallback mode ended.
func (ad *AnomalyDetector) RecordFallbackEnd() {
	ad.mu.Lock()
	defer ad.mu.Unlock()
	ad.fallbackStart = time.Time{}
}

// CheckSustainedFallback checks if fallback has been active too long.
func (ad *AnomalyDetector) CheckSustainedFallback(activeProvider string) *Anomaly {
	ad.mu.Lock()
	defer ad.mu.Unlock()

	if !ad.fallbackStart.IsZero() {
		duration := time.Since(ad.fallbackStart)
		if duration > ad.config.SustainedFallbackDuration {
			anomaly := Anomaly{
				Type:       AnomalySustainedFallback,
				Provider:   activeProvider,
				Message:    "Sustained fallback usage detected",
				DetectedAt: time.Now(),
				Value:      duration.Seconds(),
			}
			ad.recordAnomaly(anomaly)
			return &anomaly
		}
	}
	return nil
}

// recordAnomaly records an anomaly and updates metrics. Must be called with lock held.
func (ad *AnomalyDetector) recordAnomaly(anomaly Anomaly) {
	anomalyDetectedTotal.WithLabelValues(string(anomaly.Type), anomaly.Provider).Inc()

	// Keep only recent anomalies
	ad.recentAnomalies = append(ad.recentAnomalies, anomaly)
	if len(ad.recentAnomalies) > 100 {
		ad.recentAnomalies = ad.recentAnomalies[len(ad.recentAnomalies)-100:]
	}
}

// RecentAnomalies returns recent anomalies.
func (ad *AnomalyDetector) RecentAnomalies(limit int) []Anomaly {
	ad.mu.RLock()
	defer ad.mu.RUnlock()

	if limit <= 0 || limit > len(ad.recentAnomalies) {
		limit = len(ad.recentAnomalies)
	}

	start := len(ad.recentAnomalies) - limit
	result := make([]Anomaly, limit)
	copy(result, ad.recentAnomalies[start:])
	return result
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}
