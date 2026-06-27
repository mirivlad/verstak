// Package storage provides a safe, namespace-isolated JSON storage API for plugins.
// All data is stored within the vault's .verstak directory, scoped per plugin.
package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/verstak/verstak-desktop/internal/core/vault"
)

// Storage provides plugin-scoped JSON storage (settings, data, cache).
type Storage struct {
	mu    sync.RWMutex
	vault *vault.Vault
}

// New creates a new Storage instance backed by the given vault.
func New(v *vault.Vault) *Storage {
	return &Storage{vault: v}
}

// ─── Plugin ID validation ─────────────────────────────────

func validatePluginID(pluginID string) error {
	if pluginID == "" {
		return fmt.Errorf("plugin ID is empty")
	}
	if strings.ContainsAny(pluginID, `/\`) {
		return fmt.Errorf("plugin ID %q contains path separators", pluginID)
	}
	if pluginID == "." || pluginID == ".." {
		return fmt.Errorf("plugin ID %q is a path traversal reference", pluginID)
	}
	cleaned := filepath.Clean(pluginID)
	if cleaned != pluginID {
		return fmt.Errorf("plugin ID %q contains path traversal", pluginID)
	}
	return nil
}

func validateStorageName(kind, name string) error {
	if name == "" {
		return fmt.Errorf("%s name is empty", kind)
	}
	if strings.ContainsAny(name, `/\`) {
		return fmt.Errorf("%s name %q contains path separators", kind, name)
	}
	if name == "." || name == ".." {
		return fmt.Errorf("%s name %q is a path traversal reference", kind, name)
	}
	cleaned := filepath.Clean(name)
	if cleaned != name {
		return fmt.Errorf("%s name %q contains path traversal", kind, name)
	}
	return nil
}

// ─── Atomic write helper ──────────────────────────────────

func atomicWrite(path string, data []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("failed to create dir %s: %w", dir, err)
	}
	tmpFile := filepath.Join(dir, fmt.Sprintf(".tmp.%d", time.Now().UnixNano()))
	if err := os.WriteFile(tmpFile, data, 0o644); err != nil {
		return fmt.Errorf("failed to write temp file: %w", err)
	}
	if err := os.Rename(tmpFile, path); err != nil {
		os.Remove(tmpFile) // best-effort cleanup
		return fmt.Errorf("failed to rename temp file: %w", err)
	}
	return nil
}

// ─── Settings API ─────────────────────────────────────────

// ReadPluginSettings reads all settings for a plugin.
// Returns empty map if settings.json does not exist.
func (s *Storage) ReadPluginSettings(pluginID string) (map[string]interface{}, error) {
	if err := validatePluginID(pluginID); err != nil {
		return nil, err
	}

	dir := s.vault.GetPluginSettingsPath(pluginID)
	path := filepath.Join(dir, "settings.json")

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return make(map[string]interface{}), nil
		}
		return nil, fmt.Errorf("failed to read settings for plugin %s: %w", pluginID, err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("corrupt settings.json for plugin %s: %w", pluginID, err)
	}
	return result, nil
}

// WritePluginSettings writes all settings for a plugin atomically.
func (s *Storage) WritePluginSettings(pluginID string, data map[string]interface{}) error {
	if err := validatePluginID(pluginID); err != nil {
		return err
	}

	dir := s.vault.GetPluginSettingsPath(pluginID)
	path := filepath.Join(dir, "settings.json")

	encoded, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal settings for plugin %s: %w", pluginID, err)
	}
	return atomicWrite(path, encoded)
}

// ReadPluginSetting reads a single setting key.
func (s *Storage) ReadPluginSetting(pluginID, key string) (interface{}, error) {
	settings, err := s.ReadPluginSettings(pluginID)
	if err != nil {
		return nil, err
	}
	val, ok := settings[key]
	if !ok {
		return nil, nil
	}
	return val, nil
}

// WritePluginSetting writes a single setting key.
func (s *Storage) WritePluginSetting(pluginID, key string, value interface{}) error {
	settings, err := s.ReadPluginSettings(pluginID)
	if err != nil {
		return err
	}
	settings[key] = value
	return s.WritePluginSettings(pluginID, settings)
}

// ─── Data JSON API ────────────────────────────────────────

// ReadPluginDataJSON reads a named JSON data file for a plugin.
func (s *Storage) ReadPluginDataJSON(pluginID, name string) (map[string]interface{}, error) {
	if err := validatePluginID(pluginID); err != nil {
		return nil, err
	}
	if err := validateStorageName("data", name); err != nil {
		return nil, err
	}

	dir := s.vault.GetPluginDataPath(pluginID)
	path := filepath.Join(dir, name+".json")

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return make(map[string]interface{}), nil
		}
		return nil, fmt.Errorf("failed to read data %s for plugin %s: %w", name, pluginID, err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("corrupt data file %s.json for plugin %s: %w", name, pluginID, err)
	}
	return result, nil
}

// WritePluginDataJSON writes a named JSON data file for a plugin atomically.
func (s *Storage) WritePluginDataJSON(pluginID, name string, data map[string]interface{}) error {
	if err := validatePluginID(pluginID); err != nil {
		return err
	}
	if err := validateStorageName("data", name); err != nil {
		return err
	}

	dir := s.vault.GetPluginDataPath(pluginID)
	path := filepath.Join(dir, name+".json")

	encoded, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal data %s for plugin %s: %w", name, pluginID, err)
	}
	return atomicWrite(path, encoded)
}

// ─── Cache JSON API ───────────────────────────────────────

// ReadPluginCacheJSON reads a named JSON cache file for a plugin.
func (s *Storage) ReadPluginCacheJSON(pluginID, name string) (map[string]interface{}, error) {
	if err := validatePluginID(pluginID); err != nil {
		return nil, err
	}
	if err := validateStorageName("cache", name); err != nil {
		return nil, err
	}

	dir := s.vault.GetPluginCachePath(pluginID)
	path := filepath.Join(dir, name+".json")

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return make(map[string]interface{}), nil
		}
		return nil, fmt.Errorf("failed to read cache %s for plugin %s: %w", name, pluginID, err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("corrupt cache file %s.json for plugin %s: %w", name, pluginID, err)
	}
	return result, nil
}

// WritePluginCacheJSON writes a named JSON cache file for a plugin atomically.
func (s *Storage) WritePluginCacheJSON(pluginID, name string, data map[string]interface{}) error {
	if err := validatePluginID(pluginID); err != nil {
		return err
	}
	if err := validateStorageName("cache", name); err != nil {
		return err
	}

	dir := s.vault.GetPluginCachePath(pluginID)
	path := filepath.Join(dir, name+".json")

	encoded, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal cache %s for plugin %s: %w", name, pluginID, err)
	}
	return atomicWrite(path, encoded)
}
