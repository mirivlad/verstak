// Package pluginstate manages the vault-level plugin state (enabled/disabled, desired plugins).
// This is stored inside the vault at .verstak/plugins.json, separate from app settings
// and separate from individual plugin settings.
package pluginstate

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/verstak/verstak-desktop/internal/core/vault"
)

// VaultPluginState represents the plugin state for a specific vault.
type VaultPluginState struct {
	SchemaVersion   int             `json:"schemaVersion"`
	EnabledPlugins  []string        `json:"enabledPlugins"`
	DisabledPlugins []string        `json:"disabledPlugins"`
	DesiredPlugins  []DesiredPlugin `json:"desiredPlugins"`
	UpdatedAt       string          `json:"updatedAt"`
}

// DesiredPlugin records a plugin that should be available in this vault.
type DesiredPlugin struct {
	ID      string `json:"id"`
	Version string `json:"version"`
	Source  string `json:"source"`
}

// Manager provides thread-safe access to vault plugin state.
type Manager struct {
	mu    sync.RWMutex
	state *VaultPluginState
	vault *vault.Vault
}

// NewManager creates a new vault plugin state manager.
func NewManager(v *vault.Vault) *Manager {
	return &Manager{
		vault: v,
	}
}

// Load reads the vault plugin state from .verstak/plugins.json.
func (m *Manager) Load() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.vault.GetVaultStatus() != vault.StatusOpen {
		return fmt.Errorf("vault is not open")
	}

	vaultPath := m.vault.GetVaultPath()
	statePath := filepath.Join(vaultPath, ".verstak", "plugins.json")

	data, err := os.ReadFile(statePath)
	if err != nil {
		if os.IsNotExist(err) {
			m.state = defaultState()
			return m.saveLocked()
		}
		return fmt.Errorf("failed to read vault plugin state: %w", err)
	}

	var state VaultPluginState
	if err := json.Unmarshal(data, &state); err != nil {
		// Corrupt: backup and create defaults
		backupPath := statePath + ".corrupt." + time.Now().Format("20060102-150405")
		os.WriteFile(backupPath, data, 0o600)
		m.state = defaultState()
		if saveErr := m.saveLocked(); saveErr != nil {
			return fmt.Errorf("corrupt plugins.json (backed up to %s), failed to save defaults: %w", backupPath, saveErr)
		}
		return fmt.Errorf("corrupt plugins.json (backed up to %s), defaults created", backupPath)
	}

	if state.SchemaVersion != 1 {
		state.SchemaVersion = 1
	}
	if state.EnabledPlugins == nil {
		state.EnabledPlugins = []string{}
	}
	if state.DisabledPlugins == nil {
		state.DisabledPlugins = []string{}
	}
	if state.DesiredPlugins == nil {
		state.DesiredPlugins = []DesiredPlugin{}
	}

	m.state = &state
	return nil
}

// Save writes the vault plugin state to disk.
func (m *Manager) Save() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.saveLocked()
}

func (m *Manager) saveLocked() error {
	if m.state == nil {
		m.state = defaultState()
	}

	m.state.UpdatedAt = time.Now().UTC().Format(time.RFC3339)

	vaultPath := m.vault.GetVaultPath()
	statePath := filepath.Join(vaultPath, ".verstak", "plugins.json")

	data, err := json.MarshalIndent(m.state, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal vault plugin state: %w", err)
	}

	tmpFile := statePath + ".tmp"
	if err := os.WriteFile(tmpFile, data, 0o644); err != nil {
		return fmt.Errorf("failed to write vault plugin state: %w", err)
	}
	return os.Rename(tmpFile, statePath)
}

// Get returns a copy of the current state.
func (m *Manager) Get() *VaultPluginState {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.state == nil {
		return defaultState()
	}
	return copyState(m.state)
}

// IsEnabled checks if a plugin is enabled.
func (m *Manager) IsEnabled(pluginID string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.state == nil {
		return false
	}
	for _, id := range m.state.EnabledPlugins {
		if id == pluginID {
			return true
		}
	}
	return false
}

// IsDisabled checks if a plugin is explicitly disabled.
func (m *Manager) IsDisabled(pluginID string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.state == nil {
		return false
	}
	for _, id := range m.state.DisabledPlugins {
		if id == pluginID {
			return true
		}
	}
	return false
}

// EnablePlugin enables a plugin.
func (m *Manager) EnablePlugin(pluginID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.state == nil {
		m.state = defaultState()
	}

	// Remove from disabled
	m.state.DisabledPlugins = removeString(m.state.DisabledPlugins, pluginID)

	// Add to enabled if not already there
	if !containsString(m.state.EnabledPlugins, pluginID) {
		m.state.EnabledPlugins = append(m.state.EnabledPlugins, pluginID)
	}

	return m.saveLocked()
}

// DisablePlugin disables a plugin.
func (m *Manager) DisablePlugin(pluginID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.state == nil {
		m.state = defaultState()
	}

	// Remove from enabled
	m.state.EnabledPlugins = removeString(m.state.EnabledPlugins, pluginID)

	// Add to disabled if not already there
	if !containsString(m.state.DisabledPlugins, pluginID) {
		m.state.DisabledPlugins = append(m.state.DisabledPlugins, pluginID)
	}

	return m.saveLocked()
}

// RecordDesiredPlugin adds or updates a desired plugin entry.
func (m *Manager) RecordDesiredPlugin(id, version, source string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.state == nil {
		m.state = defaultState()
	}

	// Update if exists
	for i, dp := range m.state.DesiredPlugins {
		if dp.ID == id {
			m.state.DesiredPlugins[i].Version = version
			m.state.DesiredPlugins[i].Source = source
			return m.saveLocked()
		}
	}

	// Add new
	m.state.DesiredPlugins = append(m.state.DesiredPlugins, DesiredPlugin{
		ID:      id,
		Version: version,
		Source:  source,
	})

	return m.saveLocked()
}

// ListMissingInstalled returns desired plugins that are not currently installed.
func (m *Manager) ListMissingInstalled(installedIDs []string) []DesiredPlugin {
	m.mu.RLock()
	defer m.mu.RUnlock()

	installed := make(map[string]bool)
	for _, id := range installedIDs {
		installed[id] = true
	}

	var missing []DesiredPlugin
	for _, dp := range m.state.DesiredPlugins {
		if !installed[dp.ID] {
			missing = append(missing, dp)
		}
	}
	return missing
}

func defaultState() *VaultPluginState {
	return &VaultPluginState{
		SchemaVersion:   1,
		EnabledPlugins:  []string{},
		DisabledPlugins: []string{},
		DesiredPlugins:  []DesiredPlugin{},
		UpdatedAt:       time.Now().UTC().Format(time.RFC3339),
	}
}

func copyState(s *VaultPluginState) *VaultPluginState {
	enabled := make([]string, len(s.EnabledPlugins))
	copy(enabled, s.EnabledPlugins)
	disabled := make([]string, len(s.DisabledPlugins))
	copy(disabled, s.DisabledPlugins)
	desired := make([]DesiredPlugin, len(s.DesiredPlugins))
	copy(desired, s.DesiredPlugins)
	return &VaultPluginState{
		SchemaVersion:   s.SchemaVersion,
		EnabledPlugins:  enabled,
		DisabledPlugins: disabled,
		DesiredPlugins:  desired,
		UpdatedAt:       s.UpdatedAt,
	}
}

func containsString(list []string, s string) bool {
	for _, item := range list {
		if item == s {
			return true
		}
	}
	return false
}

func removeString(list []string, s string) []string {
	result := make([]string, 0, len(list))
	for _, item := range list {
		if item != s {
			result = append(result, item)
		}
	}
	return result
}
