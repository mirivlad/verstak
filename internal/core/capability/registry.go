// Package capability provides a registry for plugin capabilities.
package capability

import (
	"fmt"
	"sort"
	"sync"
)

// Registry tracks available capabilities and which plugins provide them.
type Registry struct {
	mu           sync.RWMutex
	capabilities map[string]*Entry // capability name -> entry
}

// Entry represents a capability and its provider.
type Entry struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	PluginID    string `json:"pluginId"`
	Status      string `json:"status"` // "stable", "draft", "deprecated"
}

// NewRegistry creates a new capability registry.
func NewRegistry() *Registry {
	return &Registry{
		capabilities: make(map[string]*Entry),
	}
}

// Register adds a capability provided by a plugin.
func (r *Registry) Register(pluginID string, capabilities []string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, name := range capabilities {
		if existing, ok := r.capabilities[name]; ok {
			return fmt.Errorf("capability %q already registered by plugin %q", name, existing.PluginID)
		}
		r.capabilities[name] = &Entry{
			Name:     name,
			PluginID: pluginID,
			Status:   "draft",
		}
	}
	return nil
}

// UnregisterAll removes all capabilities (used before reload).
func (r *Registry) UnregisterAll() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.capabilities = make(map[string]*Entry)
}

// Unregister removes all capabilities provided by a plugin.
func (r *Registry) Unregister(pluginID string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for name, entry := range r.capabilities {
		if entry.PluginID == pluginID {
			delete(r.capabilities, name)
		}
	}
}

// Has checks if a capability is registered.
func (r *Registry) Has(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, ok := r.capabilities[name]
	return ok
}

// Get returns a capability entry by name.
func (r *Registry) Get(name string) *Entry {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.capabilities[name]
}

// List returns all registered capabilities, sorted by name.
func (r *Registry) List() []Entry {
	r.mu.RLock()
	defer r.mu.RUnlock()

	entries := make([]Entry, 0, len(r.capabilities))
	for _, e := range r.capabilities {
		entries = append(entries, *e)
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name < entries[j].Name
	})
	return entries
}

// Available returns the set of available capability names.
func (r *Registry) Available() map[string]bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make(map[string]bool, len(r.capabilities))
	for name := range r.capabilities {
		result[name] = true
	}
	return result
}

// CheckRequired verifies that all required capabilities are present.
// Returns missing capabilities.
func (r *Registry) CheckRequired(required []string) []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var missing []string
	for _, capName := range required {
		if _, ok := r.capabilities[capName]; !ok {
			missing = append(missing, capName)
		}
	}
	return missing
}
