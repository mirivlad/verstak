// Package vault provides the core vault service for managing Verstak vaults.
// A vault is a directory that stores plugin data, settings, cache, and metadata.
package vault

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/verstak/verstak-desktop/internal/core/events"
)

// VaultStatus represents the current state of a vault.
type VaultStatus string

const (
	StatusNotCreated VaultStatus = "not-created"
	StatusClosed     VaultStatus = "closed"
	StatusOpen       VaultStatus = "open"
	StatusError      VaultStatus = "error"
)

// VaultMeta stores metadata about a vault, persisted in .verstak/vault.json.
type VaultMeta struct {
	SchemaVersion int    `json:"schemaVersion"`
	VaultID       string `json:"vaultId"`
	CreatedAt     string `json:"createdAt"`
	UpdatedAt     string `json:"updatedAt"`
	App           string `json:"app"`
}

// Vault manages a Verstak vault directory and its layout.
type Vault struct {
	mu       sync.RWMutex
	status   VaultStatus
	path     string
	meta     *VaultMeta
	eventBus *events.Bus
}

// NewVault creates a new Vault instance with the given event bus.
func NewVault(bus *events.Bus) *Vault {
	return &Vault{
		status:   StatusNotCreated,
		path:     "",
		meta:     nil,
		eventBus: bus,
	}
}

// GetVaultStatus returns the current vault status.
func (v *Vault) GetVaultStatus() VaultStatus {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return v.status
}

// GetVaultPath returns the current vault path.
func (v *Vault) GetVaultPath() string {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return v.path
}

// GetVaultMeta returns the current vault metadata.
func (v *Vault) GetVaultMeta() *VaultMeta {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return v.meta
}

// CreateVault creates a new vault at the given path.
func (v *Vault) CreateVault(path string) error {
	if err := ValidateVaultPath(path); err != nil {
		return fmt.Errorf("invalid vault path: %w", err)
	}

	vaultDir := filepath.Join(path, "VerstakVault")

	// Create VerstakVault directory
	if err := os.MkdirAll(vaultDir, 0o755); err != nil {
		return fmt.Errorf("failed to create vault directory: %w", err)
	}

	// Ensure .verstak layout
	if err := EnsureVaultLayout(vaultDir); err != nil {
		return fmt.Errorf("failed to create vault layout: %w", err)
	}

	// Generate metadata
	now := time.Now().UTC().Format(time.RFC3339)
	meta := &VaultMeta{
		SchemaVersion: 1,
		VaultID:       uuid.New().String(),
		CreatedAt:     now,
		UpdatedAt:     now,
		App:           "verstak",
	}

	// Write vault.json
	metaPath := filepath.Join(vaultDir, ".verstak", "vault.json")
	data, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal vault meta: %w", err)
	}
	if err := os.WriteFile(metaPath, data, 0o644); err != nil {
		return fmt.Errorf("failed to write vault.json: %w", err)
	}

	v.mu.Lock()
	v.status = StatusOpen
	v.path = vaultDir
	v.meta = meta
	v.mu.Unlock()

	// Publish event
	if v.eventBus != nil {
		v.eventBus.Publish(events.Event{
			Name:    "vault.created",
			Payload: map[string]string{"path": v.path, "vaultId": v.meta.VaultID},
		})
	}

	return nil
}

// OpenVault opens an existing vault at the given path.
// The path can be either the vault root (containing .verstak/vault.json)
// or the parent directory (containing VerstakVault/.verstak/vault.json).
func (v *Vault) OpenVault(path string) error {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("failed to resolve path: %w", err)
	}

	// Try direct path first: <path>/.verstak/vault.json
	metaPath := filepath.Join(absPath, ".verstak", "vault.json")
	data, err := os.ReadFile(metaPath)
	if err != nil {
		// Try VerstakVault subdirectory: <path>/VerstakVault/.verstak/vault.json
		vaultDir := filepath.Join(absPath, "VerstakVault")
		metaPath = filepath.Join(vaultDir, ".verstak", "vault.json")
		data, err = os.ReadFile(metaPath)
		if err != nil {
			return fmt.Errorf("failed to read vault.json: %w (looked in %s and %s)", err, filepath.Join(absPath, ".verstak"), filepath.Join(vaultDir, ".verstak"))
		}
		absPath = vaultDir
	}

	var meta VaultMeta
	if err := json.Unmarshal(data, &meta); err != nil {
		return fmt.Errorf("failed to parse vault.json: %w", err)
	}

	// Validate metadata
	if meta.SchemaVersion != 1 {
		return fmt.Errorf("unsupported schema version: %d", meta.SchemaVersion)
	}
	if meta.VaultID == "" {
		return errors.New("vault ID is empty")
	}

	// Ensure layout exists
	if err := EnsureVaultLayout(absPath); err != nil {
		return fmt.Errorf("failed to ensure vault layout: %w", err)
	}

	v.mu.Lock()
	v.status = StatusOpen
	v.path = absPath
	v.meta = &meta
	v.mu.Unlock()

	// Publish event
	if v.eventBus != nil {
		v.eventBus.Publish(events.Event{
			Name:    "vault.opened",
			Payload: map[string]string{"path": v.path, "vaultId": v.meta.VaultID},
		})
	}

	return nil
}

// CloseVault closes the current vault.
func (v *Vault) CloseVault() {
	v.mu.Lock()
	defer v.mu.Unlock()

	if v.status == StatusClosed {
		return
	}

	vaultID := ""
	if v.meta != nil {
		vaultID = v.meta.VaultID
	}

	v.status = StatusClosed
	v.path = ""
	v.meta = nil

	if v.eventBus != nil {
		v.eventBus.Publish(events.Event{
			Name:    "vault.closed",
			Payload: map[string]string{"vaultId": vaultID},
		})
	}
}

// EnsureVaultLayout creates the .verstak directory and standard subdirectories
// if they do not already exist.
func EnsureVaultLayout(basePath string) error {
	subdirs := []string{
		".verstak/plugin-data",
		".verstak/plugin-settings",
		".verstak/plugin-cache",
		".verstak/trash",
		".verstak/logs",
	}

	for _, sub := range subdirs {
		dir := filepath.Join(basePath, sub)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("failed to create %s: %w", sub, err)
		}
	}

	return nil
}

// ValidateVaultPath checks that the given path is a valid, safe vault path.
func ValidateVaultPath(path string) error {
	if path == "" {
		return errors.New("path is empty")
	}

	cleaned := filepath.Clean(path)

	if !filepath.IsAbs(cleaned) {
		return errors.New("path must be absolute")
	}

	// Check for null bytes
	if strings.Contains(cleaned, "\x00") {
		return errors.New("path contains null bytes")
	}

	return nil
}

// ResolveSafePath resolves a relative path within the vault, preventing
// path traversal attacks.
func (v *Vault) ResolveSafePath(relative string) (string, error) {
	v.mu.RLock()
	defer v.mu.RUnlock()

	if v.status != StatusOpen || v.path == "" {
		return "", errors.New("vault is not open")
	}

	result := filepath.Join(v.path, relative)
	result = filepath.Clean(result)

	if !strings.HasPrefix(result, v.path) {
		return "", errors.New("path traversal detected")
	}

	return result, nil
}

// GetPluginDataPath returns the data directory for a plugin, creating it if needed.
func (v *Vault) GetPluginDataPath(pluginID string) string {
	v.mu.RLock()
	defer v.mu.RUnlock()

	dir := filepath.Join(v.path, ".verstak", "plugin-data", pluginID)
	os.MkdirAll(dir, 0o755)
	return dir
}

// GetPluginSettingsPath returns the settings directory for a plugin, creating it if needed.
func (v *Vault) GetPluginSettingsPath(pluginID string) string {
	v.mu.RLock()
	defer v.mu.RUnlock()

	dir := filepath.Join(v.path, ".verstak", "plugin-settings", pluginID)
	os.MkdirAll(dir, 0o755)
	return dir
}

// GetPluginCachePath returns the cache directory for a plugin, creating it if needed.
func (v *Vault) GetPluginCachePath(pluginID string) string {
	v.mu.RLock()
	defer v.mu.RUnlock()

	dir := filepath.Join(v.path, ".verstak", "plugin-cache", pluginID)
	os.MkdirAll(dir, 0o755)
	return dir
}
