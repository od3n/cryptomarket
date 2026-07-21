// Package featureflags provides runtime feature toggles for gradual rollout.
// Flags are configured via environment variables and can be checked at runtime
// without redeployment.
//
// Usage:
//
//	flags := featureflags.New()
//	if flags.IsEnabled("new_pricing_algorithm") {
//	    // new code path
//	}
//
// Configuration via environment:
//
//	FEATURE_FLAGS=new_pricing_algorithm=true,experimental_ui=false,beta_providers=true
package featureflags

import (
	"os"
	"strings"
	"sync"
)

// Flags holds the current feature flag state.
type Flags struct {
	mu    sync.RWMutex
	flags map[string]bool
}

// New creates a Flags instance loaded from the FEATURE_FLAGS environment variable.
// Format: comma-separated key=value pairs (e.g., "flag_a=true,flag_b=false").
func New() *Flags {
	f := &Flags{
		flags: make(map[string]bool),
	}
	f.loadFromEnv()
	return f
}

// NewWithDefaults creates a Flags instance with predefined defaults,
// overridden by environment configuration.
func NewWithDefaults(defaults map[string]bool) *Flags {
	f := &Flags{
		flags: make(map[string]bool, len(defaults)),
	}
	for k, v := range defaults {
		f.flags[k] = v
	}
	f.loadFromEnv()
	return f
}

// IsEnabled returns true if the named feature flag is enabled.
// Unknown flags return false (fail-closed).
func (f *Flags) IsEnabled(name string) bool {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.flags[name]
}

// Set enables or disables a feature flag at runtime.
// Useful for testing or admin endpoints.
func (f *Flags) Set(name string, enabled bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.flags[name] = enabled
}

// All returns a copy of all current flag states.
func (f *Flags) All() map[string]bool {
	f.mu.RLock()
	defer f.mu.RUnlock()
	result := make(map[string]bool, len(f.flags))
	for k, v := range f.flags {
		result[k] = v
	}
	return result
}

// Reload re-reads flags from the environment variable.
// Call periodically or on config change signal.
func (f *Flags) Reload() {
	f.loadFromEnv()
}

// loadFromEnv parses the FEATURE_FLAGS environment variable.
func (f *Flags) loadFromEnv() {
	raw := os.Getenv("FEATURE_FLAGS")
	if raw == "" {
		return
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	pairs := strings.Split(raw, ",")
	for _, pair := range pairs {
		pair = strings.TrimSpace(pair)
		if pair == "" {
			continue
		}
		parts := strings.SplitN(pair, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(strings.ToLower(parts[1]))
		f.flags[key] = value == "true" || value == "1" || value == "yes"
	}
}

// Well-known feature flag names.
const (
	// FlagNewPricing enables the new pricing algorithm.
	FlagNewPricing = "new_pricing_algorithm"

	// FlagBetaProviders enables beta data providers.
	FlagBetaProviders = "beta_providers"

	// FlagExperimentalCache enables experimental Redis caching strategy.
	FlagExperimentalCache = "experimental_cache"

	// FlagDetailedMetrics enables high-cardinality debug metrics.
	FlagDetailedMetrics = "detailed_metrics"

	// FlagMaintenanceMode enables maintenance mode (returns 503).
	FlagMaintenanceMode = "maintenance_mode"
)

// DefaultFlags returns the standard set of feature flags with safe defaults.
func DefaultFlags() *Flags {
	return NewWithDefaults(map[string]bool{
		FlagNewPricing:        false,
		FlagBetaProviders:     false,
		FlagExperimentalCache: false,
		FlagDetailedMetrics:   false,
		FlagMaintenanceMode:   false,
	})
}
