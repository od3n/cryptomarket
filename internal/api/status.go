package api

import (
	"context"
	"net/http"
	"time"

	"github.com/crypto-market-platform/internal/market"
	"github.com/crypto-market-platform/internal/provider"
)

// DegradedState represents the overall platform health state.
type DegradedState string

const (
	// StateHealthy indicates all systems operating normally.
	StateHealthy DegradedState = "healthy"
	// StateDegraded indicates reduced functionality (e.g., fallback provider active).
	StateDegraded DegradedState = "degraded"
	// StateStale indicates data is stale but service is available.
	StateStale DegradedState = "stale"
	// StateUnavailable indicates the service cannot fulfill requests.
	StateUnavailable DegradedState = "unavailable"
)

// OperationsStatus represents the full operational status response.
type OperationsStatus struct {
	Status          DegradedState               `json:"status"`
	Timestamp       time.Time                   `json:"timestamp"`
	Ingestion       IngestionStatus             `json:"ingestion"`
	Provider        provider.OrchestratorStatus `json:"provider"`
	Freshness       *market.FreshnessSummary    `json:"freshness,omitempty"`
	Dependencies    DependencyStatus            `json:"dependencies"`
	RecentAnomalies []market.Anomaly            `json:"recent_anomalies,omitempty"`
}

// IngestionStatus represents the ingestion subsystem status.
type IngestionStatus struct {
	Active         bool   `json:"active"`
	ActiveProvider string `json:"active_provider"`
	Degraded       bool   `json:"degraded"`
}

// DependencyStatus represents the health of external dependencies.
type DependencyStatus struct {
	PostgreSQL string `json:"postgresql"`
	Redis      string `json:"redis"`
}

// StatusReporter provides operational status information.
type StatusReporter struct {
	orchestrator     *provider.FallbackOrchestrator
	freshnessTracker *market.FreshnessTracker
	anomalyDetector  *market.AnomalyDetector
}

// NewStatusReporter creates a new StatusReporter.
func NewStatusReporter(
	orchestrator *provider.FallbackOrchestrator,
	freshnessTracker *market.FreshnessTracker,
	anomalyDetector *market.AnomalyDetector,
) *StatusReporter {
	return &StatusReporter{
		orchestrator:     orchestrator,
		freshnessTracker: freshnessTracker,
		anomalyDetector:  anomalyDetector,
	}
}

// OperationsStatus returns the full operational status of the platform.
func (h *Handler) OperationsStatus(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	now := time.Now()

	status := OperationsStatus{
		Timestamp: now,
		Status:    StateHealthy,
	}

	// Check dependency health.
	status.Dependencies = h.checkDependencies(ctx)
	if status.Dependencies.PostgreSQL == "unavailable" || status.Dependencies.Redis == "unavailable" {
		status.Status = StateUnavailable
	}

	// Provider / ingestion status.
	if h.statusReporter != nil && h.statusReporter.orchestrator != nil {
		orchStatus := h.statusReporter.orchestrator.Status()
		status.Provider = orchStatus
		status.Ingestion = IngestionStatus{
			Active:         true,
			ActiveProvider: orchStatus.ActiveProvider,
			Degraded:       orchStatus.Degraded,
		}

		if orchStatus.Degraded && status.Status == StateHealthy {
			status.Status = StateDegraded
		}

		// Check if all circuit breakers are open.
		allOpen := len(orchStatus.CircuitStates) > 0
		for _, state := range orchStatus.CircuitStates {
			if state != "open" {
				allOpen = false
				break
			}
		}
		if allOpen {
			status.Status = StateUnavailable
		}
	} else {
		status.Ingestion = IngestionStatus{Active: false}
	}

	// Freshness status.
	if h.statusReporter != nil && h.statusReporter.freshnessTracker != nil {
		summary := h.statusReporter.freshnessTracker.Summary()
		status.Freshness = &summary

		if summary.OverallState == market.FreshnessStale && status.Status == StateHealthy {
			status.Status = StateStale
		}
	}

	// Recent anomalies.
	if h.statusReporter != nil && h.statusReporter.anomalyDetector != nil {
		anomalies := h.statusReporter.anomalyDetector.RecentAnomalies(10)
		if len(anomalies) > 0 {
			status.RecentAnomalies = anomalies
		}
	}

	writeJSON(w, http.StatusOK, status)
}

// checkDependencies checks the health of PostgreSQL and Redis.
func (h *Handler) checkDependencies(ctx context.Context) DependencyStatus {
	deps := DependencyStatus{
		PostgreSQL: "healthy",
		Redis:      "healthy",
	}

	if h.db != nil {
		if err := h.db.PingContext(ctx); err != nil {
			deps.PostgreSQL = "unavailable"
		}
	}

	if h.cache != nil {
		if err := h.cache.Ping(ctx); err != nil {
			deps.Redis = "unavailable"
		}
	}

	return deps
}
