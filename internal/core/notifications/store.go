package notifications

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type storedSchedules struct {
	Items []Item `json:"items"`
}

func schedulePath(vaultPath string) string {
	return filepath.Join(vaultPath, ".verstak", "notifications", "schedules.json")
}

func (m *Manager) loadLocked() error {
	if m.loaded {
		return nil
	}
	if m.vault == nil {
		return fmt.Errorf("notification vault is not initialized")
	}
	var loaded []Item
	if err := m.vault.WithOpenPath(func(vaultPath string) error {
		path := schedulePath(vaultPath)
		data, err := os.ReadFile(path)
		if os.IsNotExist(err) {
			loaded = []Item{}
			return nil
		}
		if err != nil {
			return fmt.Errorf("read notification schedules: %w", err)
		}
		var stored storedSchedules
		if err := json.Unmarshal(data, &stored); err != nil {
			return fmt.Errorf("decode notification schedules: %w", err)
		}
		for _, item := range stored.Items {
			if err := validateRequests(item.PluginID, []Request{{ID: item.ID, DueAt: item.DueAt, Title: item.Title, Body: item.Body}}); err != nil {
				return fmt.Errorf("invalid stored notification schedule: %w", err)
			}
			loaded = append(loaded, item)
		}
		sortItems(loaded)
		return nil
	}); err != nil {
		return err
	}
	m.items = loaded
	m.loaded = true
	return nil
}

func (m *Manager) writeLocked(items []Item) error {
	if m.vault == nil {
		return fmt.Errorf("notification vault is not initialized")
	}
	return m.vault.WithOpenPath(func(vaultPath string) error {
		data, err := json.MarshalIndent(storedSchedules{Items: items}, "", "  ")
		if err != nil {
			return fmt.Errorf("encode notification schedules: %w", err)
		}
		path := schedulePath(vaultPath)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return fmt.Errorf("create notification schedule directory: %w", err)
		}
		temporary := filepath.Join(filepath.Dir(path), fmt.Sprintf(".schedules.%d.tmp", time.Now().UnixNano()))
		if err := os.WriteFile(temporary, data, 0o644); err != nil {
			return fmt.Errorf("write notification schedules: %w", err)
		}
		if err := os.Rename(temporary, path); err != nil {
			_ = os.Remove(temporary)
			return fmt.Errorf("replace notification schedules: %w", err)
		}
		return nil
	})
}
