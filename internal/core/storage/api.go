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

// NDJSONRetention bounds append-only plugin data. Records are compacted after
// a successful append so settings.json never becomes an event log.
type NDJSONRetention struct {
	TimestampField   string
	MaxAge           time.Duration
	MaxEntries       int
	MaxBytes         int64
	DeduplicateField string
	DeduplicateValue string
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
	s.mu.RLock()
	defer s.mu.RUnlock()
	if err := validatePluginID(pluginID); err != nil {
		return nil, err
	}
	var result map[string]interface{}
	err := s.withOpenVault(func(vaultPath string) error {
		var err error
		result, err = readPluginSettingsAt(vaultPath, pluginID)
		return err
	})
	return result, err
}

func readPluginSettingsAt(vaultPath, pluginID string) (map[string]interface{}, error) {
	path := filepath.Join(vaultPath, ".verstak", "plugin-settings", pluginID, "settings.json")
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
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := validatePluginID(pluginID); err != nil {
		return err
	}
	return s.withOpenVault(func(vaultPath string) error {
		return writePluginSettingsAt(vaultPath, pluginID, data)
	})
}

func writePluginSettingsAt(vaultPath, pluginID string, data map[string]interface{}) error {
	path := filepath.Join(vaultPath, ".verstak", "plugin-settings", pluginID, "settings.json")
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
	return s.UpdatePluginSettings(pluginID, func(settings map[string]interface{}) error {
		settings[key] = value
		return nil
	})
}

// UpdatePluginSettings atomically reads, mutates, and writes a plugin's settings.
func (s *Storage) UpdatePluginSettings(pluginID string, update func(map[string]interface{}) error) error {
	if update == nil {
		return fmt.Errorf("settings update is nil")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := validatePluginID(pluginID); err != nil {
		return err
	}
	return s.withOpenVault(func(vaultPath string) error {
		settings, err := readPluginSettingsAt(vaultPath, pluginID)
		if err != nil {
			return err
		}
		if err := update(settings); err != nil {
			return err
		}
		return writePluginSettingsAt(vaultPath, pluginID, settings)
	})
}

func (s *Storage) withOpenVault(operation func(string) error) error {
	if s == nil || s.vault == nil {
		return fmt.Errorf("vault is not initialized")
	}
	return s.vault.WithOpenPath(operation)
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

// ReadPluginDataNDJSON reads an append-only named data file. A missing file is
// represented by an empty slice.
func (s *Storage) ReadPluginDataNDJSON(pluginID, name string) ([]map[string]interface{}, error) {
	if err := validatePluginID(pluginID); err != nil {
		return nil, err
	}
	if err := validateStorageName("data", name); err != nil {
		return nil, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	var records []map[string]interface{}
	err := s.withOpenVault(func(vaultPath string) error {
		var err error
		records, err = readPluginDataNDJSONAt(vaultPath, pluginID, name)
		return err
	})
	return records, err
}

// AppendPluginDataNDJSON appends records durably and then applies bounded
// retention. It returns false without writing when the supplied idempotency
// value is already present in the retained log.
func (s *Storage) AppendPluginDataNDJSON(pluginID, name string, records []map[string]interface{}, retention NDJSONRetention) (bool, error) {
	if err := validatePluginID(pluginID); err != nil {
		return false, err
	}
	if err := validateStorageName("data", name); err != nil {
		return false, err
	}
	if len(records) == 0 {
		return false, fmt.Errorf("NDJSON records are empty")
	}
	if retention.MaxEntries < 0 || retention.MaxBytes < 0 || retention.MaxAge < 0 {
		return false, fmt.Errorf("NDJSON retention values must not be negative")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	stored := false
	err := s.withOpenVault(func(vaultPath string) error {
		existing, err := readPluginDataNDJSONAt(vaultPath, pluginID, name)
		if err != nil {
			return err
		}
		if retention.DeduplicateField != "" && retention.DeduplicateValue != "" {
			for _, record := range existing {
				if fmt.Sprint(record[retention.DeduplicateField]) == retention.DeduplicateValue {
					return nil
				}
			}
		}
		path := pluginDataNDJSONPath(vaultPath, pluginID, name)
		if err := appendNDJSON(path, records); err != nil {
			return err
		}
		stored = true
		compacted := compactNDJSONRecords(append(existing, records...), retention, time.Now().UTC())
		if !sameNDJSONRecords(append(existing, records...), compacted) {
			return writeNDJSON(path, compacted)
		}
		return nil
	})
	return stored, err
}

// WritePluginDataNDJSON replaces a named append-only data file. It is reserved
// for explicit user actions such as clearing Activity, never normal event
// ingestion.
func (s *Storage) WritePluginDataNDJSON(pluginID, name string, records []map[string]interface{}) error {
	if err := validatePluginID(pluginID); err != nil {
		return err
	}
	if err := validateStorageName("data", name); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.withOpenVault(func(vaultPath string) error {
		return writeNDJSON(pluginDataNDJSONPath(vaultPath, pluginID, name), records)
	})
}

func pluginDataNDJSONPath(vaultPath, pluginID, name string) string {
	return filepath.Join(vaultPath, ".verstak", "plugin-data", pluginID, name+".ndjson")
}

func readPluginDataNDJSONAt(vaultPath, pluginID, name string) ([]map[string]interface{}, error) {
	data, err := os.ReadFile(pluginDataNDJSONPath(vaultPath, pluginID, name))
	if err != nil {
		if os.IsNotExist(err) {
			return []map[string]interface{}{}, nil
		}
		return nil, fmt.Errorf("failed to read NDJSON data %s for plugin %s: %w", name, pluginID, err)
	}
	if len(data) == 0 {
		return []map[string]interface{}{}, nil
	}
	lines := strings.Split(strings.TrimSuffix(string(data), "\n"), "\n")
	records := make([]map[string]interface{}, 0, len(lines))
	for index, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		var record map[string]interface{}
		if err := json.Unmarshal([]byte(line), &record); err != nil {
			return nil, fmt.Errorf("corrupt NDJSON data %s line %d for plugin %s: %w", name, index+1, pluginID, err)
		}
		records = append(records, record)
	}
	return records, nil
}

func appendNDJSON(path string, records []map[string]interface{}) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("failed to create NDJSON data directory: %w", err)
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("failed to open NDJSON data file: %w", err)
	}
	defer file.Close()
	for _, record := range records {
		encoded, err := json.Marshal(record)
		if err != nil {
			return fmt.Errorf("failed to marshal NDJSON record: %w", err)
		}
		if _, err := file.Write(append(encoded, '\n')); err != nil {
			return fmt.Errorf("failed to append NDJSON record: %w", err)
		}
	}
	if err := file.Sync(); err != nil {
		return fmt.Errorf("failed to sync NDJSON data: %w", err)
	}
	return nil
}

func writeNDJSON(path string, records []map[string]interface{}) error {
	data := make([]byte, 0)
	for _, record := range records {
		encoded, err := json.Marshal(record)
		if err != nil {
			return fmt.Errorf("failed to marshal NDJSON record: %w", err)
		}
		data = append(data, encoded...)
		data = append(data, '\n')
	}
	return atomicWrite(path, data)
}

func compactNDJSONRecords(records []map[string]interface{}, retention NDJSONRetention, now time.Time) []map[string]interface{} {
	kept := make([]map[string]interface{}, 0, len(records))
	cutoff := now.Add(-retention.MaxAge)
	for _, record := range records {
		if retention.MaxAge > 0 && retention.TimestampField != "" {
			if raw, ok := record[retention.TimestampField].(string); ok {
				if timestamp, err := time.Parse(time.RFC3339, raw); err == nil && timestamp.Before(cutoff) {
					continue
				}
			}
		}
		kept = append(kept, record)
	}
	if retention.MaxEntries > 0 && len(kept) > retention.MaxEntries {
		kept = kept[len(kept)-retention.MaxEntries:]
	}
	if retention.MaxBytes > 0 {
		for len(kept) > 0 && ndjsonSize(kept) > retention.MaxBytes {
			kept = kept[1:]
		}
	}
	return kept
}

func ndjsonSize(records []map[string]interface{}) int64 {
	var total int64
	for _, record := range records {
		encoded, err := json.Marshal(record)
		if err == nil {
			total += int64(len(encoded) + 1)
		}
	}
	return total
}

func sameNDJSONRecords(left, right []map[string]interface{}) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		leftJSON, leftErr := json.Marshal(left[index])
		rightJSON, rightErr := json.Marshal(right[index])
		if leftErr != nil || rightErr != nil || string(leftJSON) != string(rightJSON) {
			return false
		}
	}
	return true
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
