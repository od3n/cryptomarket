package provider

import (
	"fmt"
	"sync"
)

// Selector manages ordered provider selection with disablement support.
type Selector struct {
	mu       sync.RWMutex
	ordered  []Provider
	disabled map[string]bool
}

// NewSelector creates a provider selector from an ordered list of providers.
// The first provider is the primary; subsequent providers are fallbacks.
func NewSelector(providers []Provider, disabledNames []string) *Selector {
	disabled := make(map[string]bool, len(disabledNames))
	for _, name := range disabledNames {
		disabled[name] = true
	}
	return &Selector{
		ordered:  providers,
		disabled: disabled,
	}
}

// Eligible returns providers in priority order, excluding disabled ones.
func (s *Selector) Eligible() []Provider {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]Provider, 0, len(s.ordered))
	for _, p := range s.ordered {
		if !s.disabled[p.Name()] {
			result = append(result, p)
		}
	}
	return result
}

// Primary returns the highest-priority eligible provider.
func (s *Selector) Primary() (Provider, error) {
	eligible := s.Eligible()
	if len(eligible) == 0 {
		return nil, fmt.Errorf("no eligible providers available")
	}
	return eligible[0], nil
}

// Fallbacks returns all eligible providers except the primary.
func (s *Selector) Fallbacks() []Provider {
	eligible := s.Eligible()
	if len(eligible) <= 1 {
		return nil
	}
	return eligible[1:]
}

// Disable marks a provider as disabled.
func (s *Selector) Disable(name string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.disabled[name] = true
}

// Enable marks a provider as enabled.
func (s *Selector) Enable(name string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.disabled, name)
}

// IsDisabled returns whether a provider is disabled.
func (s *Selector) IsDisabled(name string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.disabled[name]
}

// All returns all registered providers regardless of disabled state.
func (s *Selector) All() []Provider {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]Provider, len(s.ordered))
	copy(result, s.ordered)
	return result
}

// Names returns the names of all providers in priority order.
func (s *Selector) Names() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	names := make([]string, len(s.ordered))
	for i, p := range s.ordered {
		names[i] = p.Name()
	}
	return names
}
