package provider

import (
	"fmt"
	"time"
)

// Factory creates a provider instance with the given base URL and timeout.
type Factory func(baseURL string, timeout time.Duration) Provider

// Registry maps provider names to their factory functions.
type Registry struct {
	factories map[string]Factory
	urls      map[string]string
}

// NewRegistry creates a new provider registry with default providers.
func NewRegistry() *Registry {
	r := &Registry{
		factories: make(map[string]Factory),
		urls:      make(map[string]string),
	}
	// Register built-in providers.
	r.Register("coingecko", func(baseURL string, timeout time.Duration) Provider {
		return NewCoinGeckoProvider(baseURL, timeout)
	})
	r.Register("coincap", func(baseURL string, timeout time.Duration) Provider {
		return NewCoinCapProvider(baseURL, timeout)
	})
	return r
}

// Register adds a provider factory to the registry.
func (r *Registry) Register(name string, factory Factory) {
	r.factories[name] = factory
}

// SetBaseURL sets the base URL for a specific provider.
func (r *Registry) SetBaseURL(name, baseURL string) {
	r.urls[name] = baseURL
}

// Create instantiates a provider by name.
func (r *Registry) Create(name string, defaultURL string, timeout time.Duration) (Provider, error) {
	factory, ok := r.factories[name]
	if !ok {
		return nil, fmt.Errorf("unknown provider: %s", name)
	}
	baseURL := defaultURL
	if u, ok := r.urls[name]; ok {
		baseURL = u
	}
	return factory(baseURL, timeout), nil
}

// Available returns the list of registered provider names.
func (r *Registry) Available() []string {
	names := make([]string, 0, len(r.factories))
	for name := range r.factories {
		names = append(names, name)
	}
	return names
}
