// Package appsettings provides application-level settings for Verstak desktop.
// App settings are stored locally (NOT inside the vault) and contain installation-specific
// configuration like the current vault path, recent vaults, theme, dev mode, etc.
package appsettings

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Config represents the application settings stored in ~/.config/verstak/config.json.
type Config struct {
	SchemaVersion    int                     `json:"schemaVersion"`
	CurrentVaultPath string                  `json:"currentVaultPath"`
	RecentVaults     []string                `json:"recentVaults"`
	Theme            string                  `json:"theme"`
	DevMode          bool                    `json:"devMode"`
	UserPluginsDir   string                  `json:"userPluginsDir"`
	Workbench        WorkbenchPreferences    `json:"workbench,omitempty"`
	Sync             SyncSettings            `json:"sync,omitempty"`
	BrowserReceiver  BrowserReceiverSettings `json:"browserReceiver,omitempty"`
	WindowState      *WindowState            `json:"windowState,omitempty"`
	LastOpenedAt     string                  `json:"lastOpenedAt"`
}

type WorkbenchPreferences struct {
	DefaultTextEditorProvider          string `json:"defaultTextEditorProvider,omitempty"`
	DefaultMarkdownEditorProvider      string `json:"defaultMarkdownEditorProvider,omitempty"`
	DefaultNotesMarkdownEditorProvider string `json:"defaultNotesMarkdownEditorProvider,omitempty"`
}

// SyncSettings holds sync configuration for the current vault.
type SyncSettings struct {
	Enabled      bool   `json:"enabled"`
	ServerURL    string `json:"serverUrl"`
	DeviceID     string `json:"deviceId"`
	DeviceName   string `json:"deviceName"`
	SyncInterval int    `json:"syncInterval"`
	LastStatus   string `json:"lastStatus"`
	LastSyncAt   string `json:"lastSyncAt"`
	LastError    string `json:"lastError,omitempty"`
}

// BrowserReceiverSettings holds the installation-local browser capture pairing secret.
type BrowserReceiverSettings struct {
	Token string `json:"token,omitempty"`
}

// WindowState stores the last window position and size.
type WindowState struct {
	Width     int  `json:"width"`
	Height    int  `json:"height"`
	Maximized bool `json:"maximized"`
}

// Manager provides thread-safe access to app settings.
type Manager struct {
	mu         sync.RWMutex
	config     *Config
	configPath string
}

// DefaultConfigPath returns the default path for app settings.
func DefaultConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	return filepath.Join(home, ".config", "verstak", "config.json")
}

// NewManager creates a new app settings manager.
func NewManager(configPath string) *Manager {
	return &Manager{
		config:     nil,
		configPath: configPath,
	}
}

// NewDefaultManager creates a manager with the default config path.
func NewDefaultManager() *Manager {
	return NewManager(DefaultConfigPath())
}

// Load reads app settings from disk, creating defaults if missing.
func (m *Manager) Load() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	data, err := os.ReadFile(m.configPath)
	if err != nil {
		if os.IsNotExist(err) {
			m.config = defaultConfig()
			return m.saveLocked()
		}
		return fmt.Errorf("failed to read app settings: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		// Corrupt config: backup and create defaults
		backupPath := m.configPath + ".corrupt." + time.Now().Format("20060102-150405")
		os.WriteFile(backupPath, data, 0o600)
		m.config = defaultConfig()
		if saveErr := m.saveLocked(); saveErr != nil {
			return fmt.Errorf("corrupt config (backed up to %s), failed to save defaults: %w", backupPath, saveErr)
		}
		return fmt.Errorf("corrupt config (backed up to %s), defaults created", backupPath)
	}

	if cfg.SchemaVersion != 1 {
		cfg.SchemaVersion = 1
	}
	if cfg.Theme == "" {
		cfg.Theme = "dark"
	}
	if cfg.RecentVaults == nil {
		cfg.RecentVaults = []string{}
	}

	m.config = &cfg
	return nil
}

// Save writes app settings to disk.
func (m *Manager) Save() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.saveLocked()
}

func (m *Manager) saveLocked() error {
	dir := filepath.Dir(m.configPath)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("failed to create config dir: %w", err)
	}

	data, err := json.MarshalIndent(m.config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal app settings: %w", err)
	}

	tmpFile := m.configPath + ".tmp"
	if err := os.WriteFile(tmpFile, data, 0o600); err != nil {
		return fmt.Errorf("failed to write app settings: %w", err)
	}
	return os.Rename(tmpFile, m.configPath)
}

// Get returns a copy of the current config.
func (m *Manager) Get() *Config {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.config == nil {
		return defaultConfig()
	}
	return copyConfig(m.config)
}

// Update patches the config with non-zero values and saves.
func (m *Manager) Update(patch *Config) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.config == nil {
		m.config = defaultConfig()
	}

	if patch.CurrentVaultPath != "" {
		m.config.CurrentVaultPath = patch.CurrentVaultPath
	}
	if patch.Theme != "" {
		m.config.Theme = patch.Theme
	}
	if patch.UserPluginsDir != "" {
		m.config.UserPluginsDir = patch.UserPluginsDir
	}
	if patch.WindowState != nil {
		m.config.WindowState = patch.WindowState
	}
	if patch.Workbench.DefaultTextEditorProvider != "" {
		m.config.Workbench.DefaultTextEditorProvider = patch.Workbench.DefaultTextEditorProvider
	}
	if patch.Workbench.DefaultMarkdownEditorProvider != "" {
		m.config.Workbench.DefaultMarkdownEditorProvider = patch.Workbench.DefaultMarkdownEditorProvider
	}
	if patch.Workbench.DefaultNotesMarkdownEditorProvider != "" {
		m.config.Workbench.DefaultNotesMarkdownEditorProvider = patch.Workbench.DefaultNotesMarkdownEditorProvider
	}
	m.config.DevMode = patch.DevMode

	m.config.LastOpenedAt = time.Now().UTC().Format(time.RFC3339)
	return m.saveLocked()
}

// UpdateSync replaces sync settings without changing unrelated app settings.
func (m *Manager) UpdateSync(syncSettings SyncSettings) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.config == nil {
		m.config = defaultConfig()
	}

	m.config.Sync = syncSettings
	m.config.LastOpenedAt = time.Now().UTC().Format(time.RFC3339)
	return m.saveLocked()
}

// EnsureBrowserReceiverToken returns the persisted pairing token, creating it when absent.
func (m *Manager) EnsureBrowserReceiverToken() (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.updateBrowserReceiverTokenLocked(false)
}

// RotateBrowserReceiverToken replaces the persisted pairing token.
func (m *Manager) RotateBrowserReceiverToken() (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.updateBrowserReceiverTokenLocked(true)
}

func (m *Manager) updateBrowserReceiverTokenLocked(force bool) (string, error) {
	if m.config == nil {
		m.config = defaultConfig()
	}
	current := strings.TrimSpace(m.config.BrowserReceiver.Token)
	if current != "" && !force {
		return current, nil
	}

	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("generate browser receiver token: %w", err)
	}
	token := base64.RawURLEncoding.EncodeToString(bytes)
	m.config.BrowserReceiver.Token = token
	if err := m.saveLocked(); err != nil {
		m.config.BrowserReceiver.Token = current
		return "", err
	}
	return token, nil
}

// SetCurrentVault updates the current vault path and adds to recents.
func (m *Manager) SetCurrentVault(path string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.config == nil {
		m.config = defaultConfig()
	}

	m.config.CurrentVaultPath = path
	m.config.LastOpenedAt = time.Now().UTC().Format(time.RFC3339)

	// Add to recents (deduplicate, keep max 10)
	m.config.RecentVaults = addRecent(m.config.RecentVaults, path, 10)

	return m.saveLocked()
}

// ClearCurrentVault clears the current vault path (e.g. on close).
func (m *Manager) ClearCurrentVault() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.config == nil {
		m.config = defaultConfig()
	}

	m.config.CurrentVaultPath = ""
	return m.saveLocked()
}

func defaultConfig() *Config {
	return &Config{
		SchemaVersion:    1,
		CurrentVaultPath: "",
		RecentVaults:     []string{},
		Theme:            "dark",
		DevMode:          false,
		UserPluginsDir:   filepath.Join(os.Getenv("HOME"), ".config", "verstak", "plugins"),
		Workbench:        WorkbenchPreferences{},
		WindowState:      &WindowState{Width: 1200, Height: 800},
		LastOpenedAt:     time.Now().UTC().Format(time.RFC3339),
	}
}

func copyConfig(c *Config) *Config {
	recent := make([]string, len(c.RecentVaults))
	copy(recent, c.RecentVaults)
	cfg := &Config{
		SchemaVersion:    c.SchemaVersion,
		CurrentVaultPath: c.CurrentVaultPath,
		RecentVaults:     recent,
		Theme:            c.Theme,
		DevMode:          c.DevMode,
		UserPluginsDir:   c.UserPluginsDir,
		Workbench:        c.Workbench,
		Sync:             c.Sync,
		BrowserReceiver:  c.BrowserReceiver,
		LastOpenedAt:     c.LastOpenedAt,
	}
	if c.WindowState != nil {
		cfg.WindowState = &WindowState{
			Width:     c.WindowState.Width,
			Height:    c.WindowState.Height,
			Maximized: c.WindowState.Maximized,
		}
	}
	return cfg
}

func addRecent(list []string, path string, max int) []string {
	// Remove if already exists
	filtered := make([]string, 0, len(list))
	for _, p := range list {
		if p != path {
			filtered = append(filtered, p)
		}
	}
	// Prepend (most recent first)
	result := append([]string{path}, filtered...)
	// Trim
	if len(result) > max {
		result = result[:max]
	}
	return result
}
