// Package notifications persists plugin-owned reminder schedules and delivers
// due entries through a desktop-native sender.
package notifications

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	maxRequestsPerPlugin = 500
	maxIDLength          = 256
	maxTitleLength       = 512
	maxBodyLength        = 4096
)

// VaultPath provides a stable operation while a vault is open.
type VaultPath interface {
	WithOpenPath(func(string) error) error
}

// Request is a desired scheduled notification in a plugin's own namespace.
type Request struct {
	ID    string `json:"id"`
	DueAt string `json:"dueAt"`
	Title string `json:"title"`
	Body  string `json:"body,omitempty"`
}

// Item is the persisted form of a plugin notification schedule.
type Item struct {
	PluginID     string `json:"pluginId"`
	ID           string `json:"id"`
	DueAt        string `json:"dueAt"`
	Title        string `json:"title"`
	Body         string `json:"body,omitempty"`
	SentForDueAt string `json:"sentForDueAt,omitempty"`
}

// Sender delivers one native desktop notification.
type Sender interface {
	Send(context.Context, Item) error
}

// Manager owns canonical schedules for every authorized plugin.
type Manager struct {
	mu     sync.Mutex
	vault  VaultPath
	sender Sender
	now    func() time.Time

	items  []Item
	loaded bool
	stop   chan struct{}
}

// New creates a notification manager. The supplied clock is used by tests; a
// nil clock uses the current time.
func New(vault VaultPath, sender Sender, now func() time.Time) *Manager {
	if now == nil {
		now = time.Now
	}
	return &Manager{vault: vault, sender: sender, now: now}
}

// Replace atomically replaces one plugin's desired schedules.
func (m *Manager) Replace(pluginID string, requests []Request) error {
	if err := validateRequests(pluginID, requests); err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := m.loadLocked(); err != nil {
		return err
	}

	previous := make(map[string]Item)
	kept := make([]Item, 0, len(m.items)+len(requests))
	for _, item := range m.items {
		if item.PluginID == pluginID {
			previous[item.ID] = item
			continue
		}
		kept = append(kept, item)
	}
	for _, request := range requests {
		item := Item{
			PluginID: pluginID,
			ID:       request.ID,
			DueAt:    request.DueAt,
			Title:    request.Title,
			Body:     request.Body,
		}
		if old, ok := previous[request.ID]; ok && old.DueAt == request.DueAt {
			item.SentForDueAt = old.SentForDueAt
		}
		kept = append(kept, item)
	}
	sortItems(kept)
	if err := m.writeLocked(kept); err != nil {
		return err
	}
	m.items = kept
	return nil
}

// Clear removes every schedule owned by a plugin.
func (m *Manager) Clear(pluginID string) error {
	return m.Replace(pluginID, nil)
}

// Items returns a snapshot of the current schedule state.
func (m *Manager) Items() []Item {
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := m.loadLocked(); err != nil {
		return nil
	}
	return append([]Item(nil), m.items...)
}

// Path reports the canonical schedule path while the vault is open.
func (m *Manager) Path() string {
	if m == nil || m.vault == nil {
		return ""
	}
	var path string
	if err := m.vault.WithOpenPath(func(vaultPath string) error {
		path = schedulePath(vaultPath)
		return nil
	}); err != nil {
		return ""
	}
	return path
}

// Start immediately checks due schedules, then repeats every 30 seconds.
func (m *Manager) Start(ctx context.Context) {
	if m == nil {
		return
	}
	m.mu.Lock()
	if m.stop != nil {
		m.mu.Unlock()
		return
	}
	stop := make(chan struct{})
	m.stop = stop
	m.mu.Unlock()

	go func() {
		m.Tick(ctx)
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				m.Tick(ctx)
			case <-stop:
				return
			case <-ctx.Done():
				return
			}
		}
	}()
}

// Stop stops a running scheduler. It is safe to call repeatedly.
func (m *Manager) Stop() {
	if m == nil {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.stop == nil {
		return
	}
	close(m.stop)
	m.stop = nil
}

// Tick delivers each overdue schedule once. A sender failure leaves the item
// pending for a later tick.
func (m *Manager) Tick(ctx context.Context) {
	if m == nil || m.sender == nil {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := m.loadLocked(); err != nil {
		return
	}
	now := m.now().UTC()
	for index, item := range m.items {
		dueAt, err := time.Parse(time.RFC3339, item.DueAt)
		if err != nil || dueAt.After(now) || item.SentForDueAt == item.DueAt {
			continue
		}
		if err := m.sender.Send(ctx, item); err != nil {
			continue
		}
		updated := append([]Item(nil), m.items...)
		updated[index].SentForDueAt = item.DueAt
		if err := m.writeLocked(updated); err != nil {
			continue
		}
		m.items = updated
	}
}

func validateRequests(pluginID string, requests []Request) error {
	if strings.TrimSpace(pluginID) == "" || strings.ContainsAny(pluginID, "/\\") {
		return fmt.Errorf("invalid plugin ID %q", pluginID)
	}
	if len(requests) > maxRequestsPerPlugin {
		return fmt.Errorf("plugin %q has more than %d notification schedules", pluginID, maxRequestsPerPlugin)
	}
	seen := make(map[string]bool, len(requests))
	for _, request := range requests {
		if request.ID == "" || len(request.ID) > maxIDLength {
			return fmt.Errorf("notification ID is empty or too long")
		}
		if seen[request.ID] {
			return fmt.Errorf("duplicate notification ID %q", request.ID)
		}
		seen[request.ID] = true
		if !strings.HasSuffix(request.DueAt, "Z") {
			return fmt.Errorf("notification %q dueAt must be UTC RFC3339", request.ID)
		}
		if _, err := time.Parse(time.RFC3339, request.DueAt); err != nil {
			return fmt.Errorf("notification %q has invalid dueAt: %w", request.ID, err)
		}
		if strings.TrimSpace(request.Title) == "" || len(request.Title) > maxTitleLength {
			return fmt.Errorf("notification %q title is empty or too long", request.ID)
		}
		if len(request.Body) > maxBodyLength {
			return fmt.Errorf("notification %q body is too long", request.ID)
		}
	}
	return nil
}

func sortItems(items []Item) {
	sort.Slice(items, func(left, right int) bool {
		if items[left].PluginID != items[right].PluginID {
			return items[left].PluginID < items[right].PluginID
		}
		return items[left].ID < items[right].ID
	})
}
