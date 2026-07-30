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

	h.applyProviderStatus(&status)
	h.applyFreshnessStatus(&status)
	h.applyRecentAnomalies(&status)

	writeJSON(w, http.StatusOK, status)
}

// applyProviderStatus fills in provider and ingestion state from the orchestrator.
func (h *Handler) applyProviderStatus(status *OperationsStatus) {
	if h.statusReporter == nil || h.statusReporter.orchestrator == nil {
		status.Ingestion = IngestionStatus{Active: false}
		return
	}

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

	if allCircuitsOpen(orchStatus.CircuitStates) {
		status.Status = StateUnavailable
	}
}

// allCircuitsOpen reports whether every circuit breaker is open.
func allCircuitsOpen(states map[string]string) bool {
	if len(states) == 0 {
		return false
	}
	for _, state := range states {
		if state != "open" {
			return false
		}
	}
	return true
}

// applyFreshnessStatus fills in data freshness state.
func (h *Handler) applyFreshnessStatus(status *OperationsStatus) {
	if h.statusReporter == nil || h.statusReporter.freshnessTracker == nil {
		return
	}
	summary := h.statusReporter.freshnessTracker.Summary()
	status.Freshness = &summary

	if summary.OverallState == market.FreshnessStale && status.Status == StateHealthy {
		status.Status = StateStale
	}
}

// applyRecentAnomalies attaches recently detected anomalies.
func (h *Handler) applyRecentAnomalies(status *OperationsStatus) {
	if h.statusReporter == nil || h.statusReporter.anomalyDetector == nil {
		return
	}
	anomalies := h.statusReporter.anomalyDetector.RecentAnomalies(10)
	if len(anomalies) > 0 {
		status.RecentAnomalies = anomalies
	}
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
