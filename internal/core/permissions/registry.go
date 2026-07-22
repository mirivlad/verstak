// Package permissions provides a registry for plugin permissions.
package permissions

import (
	"sort"
	"sync"
)

// Entry describes a known permission.
type Entry struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Dangerous   bool   `json:"dangerous"`
}

// Registry tracks known permissions and their safety levels.
type Registry struct {
	mu          sync.RWMutex
	permissions map[string]Entry
}

// NewRegistry creates a new permissions registry.
func NewRegistry() *Registry {
	r := &Registry{
		permissions: make(map[string]Entry),
	}
	r.registerDefaults()
	return r
}

func (r *Registry) registerDefaults() {
	defaults := []Entry{
		{Name: "vault.read", Description: "Read vault files and metadata", Dangerous: false},
		{Name: "vault.write", Description: "Write vault files and metadata", Dangerous: true},
		{Name: "vault.watch", Description: "Watch vault file changes", Dangerous: false},
		{Name: "files.read", Description: "List files and read text files through the vault Files API", Dangerous: false},
		{Name: "files.write", Description: "Create folders, write text files, and move paths through the vault Files API", Dangerous: true},
		{Name: "files.delete", Description: "Trash vault files and folders through the vault Files API", Dangerous: true},
		{Name: "files.openExternal", Description: "Open vault files and folders in external OS applications", Dangerous: true},
		{Name: "storage.namespace", Description: "Read/write plugin's own storage namespace", Dangerous: false},
		{Name: "storage.migrations", Description: "Run database migrations in plugin namespace", Dangerous: false},
		{Name: "notifications.schedule", Description: "Schedule native notifications in the plugin's own namespace", Dangerous: false},
		{Name: "events.publish", Description: "Publish events to the event bus", Dangerous: false},
		{Name: "events.subscribe", Description: "Subscribe to events on the event bus", Dangerous: false},
		{Name: "ui.register", Description: "Register UI components and contributions", Dangerous: false},
		{Name: "commands.register", Description: "Register command palette commands", Dangerous: false},
		{Name: "workbench.open", Description: "Request Workbench open/edit routing for vault resources", Dangerous: false},
		{Name: "network.local", Description: "Connect to localhost network services", Dangerous: false},
		{Name: "network.remote", Description: "Connect to remote network services", Dangerous: true},
		{Name: "process.spawn", Description: "Spawn external processes", Dangerous: true},
		{Name: "secrets.read", Description: "Read secrets from the secret store", Dangerous: true},
		{Name: "secrets.write", Description: "Write secrets to the secret store", Dangerous: true},
		{Name: "sync.participate", Description: "Participate in vault sync", Dangerous: true},
		{Name: "browser.receiver.manage", Description: "View and rotate the local browser receiver pairing token", Dangerous: true},
		{Name: "imports.readExternal", Description: "Select and read external folders or backup archives for import", Dangerous: true},
		{Name: "imports.apply", Description: "Create reviewed imported folders, workspaces, notes, and files in the vault", Dangerous: true},
	}
	for _, e := range defaults {
		r.permissions[e.Name] = e
	}
}

// Get returns permission info by name.
func (r *Registry) Get(name string) (Entry, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	e, ok := r.permissions[name]
	return e, ok
}

// IsDangerous checks if a permission is marked dangerous.
func (r *Registry) IsDangerous(name string) bool {
	e, ok := r.Get(name)
	return ok && e.Dangerous
}

// List returns all known permissions, sorted by name.
func (r *Registry) List() []Entry {
	r.mu.RLock()
	defer r.mu.RUnlock()

	entries := make([]Entry, 0, len(r.permissions))
	for _, e := range r.permissions {
		entries = append(entries, e)
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name < entries[j].Name
	})
	return entries
}
