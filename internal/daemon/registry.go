package daemon

import (
	"sync"

	"github.com/anthropics/claude_code_bridge/internal/daemon/adapter"
)

// Registry manages provider adapter registrations.
type Registry struct {
	mu       sync.RWMutex
	adapters map[string]adapter.Adapter
}

// NewRegistry creates a new provider registry.
func NewRegistry() *Registry {
	return &Registry{
		adapters: make(map[string]adapter.Adapter),
	}
}

// Register registers an adapter for a provider name.
func (r *Registry) Register(name string, a adapter.Adapter) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.adapters[name] = a
}

// Get returns the adapter for a provider name.
func (r *Registry) Get(name string) (adapter.Adapter, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	a, ok := r.adapters[name]
	return a, ok
}

// Names returns all registered provider names.
func (r *Registry) Names() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	names := make([]string, 0, len(r.adapters))
	for name := range r.adapters {
		names = append(names, name)
	}
	return names
}

// Count returns the number of registered adapters.
func (r *Registry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.adapters)
}
