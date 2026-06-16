// Package plugin provides plugin discovery, manifest parsing, and lifecycle management.
package plugin

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Manifest represents a Verstak plugin.json manifest.
type Manifest struct {
	SchemaVersion   int                `json:"schemaVersion"`
	ID              string             `json:"id"`
	Name            string             `json:"name"`
	Version         string             `json:"version"`
	APIVersion      string             `json:"apiVersion"`
	Description     string             `json:"description,omitempty"`
	Source          string             `json:"source,omitempty"`
	Icon            string             `json:"icon,omitempty"`
	Provides        []string           `json:"provides"`
	Requires        []string           `json:"requires,omitempty"`
	OptionalRequires []string           `json:"optionalRequires,omitempty"`
	Permissions     []string           `json:"permissions"`
	Frontend        *FrontendConfig    `json:"frontend,omitempty"`
	Backend         *BackendConfig     `json:"backend,omitempty"`
	Migrations      *MigrationConfig   `json:"migrations,omitempty"`
	Contributes     *Contributions     `json:"contributes,omitempty"`
	Sync            *SyncConfig        `json:"sync,omitempty"`
}

// FrontendConfig describes the plugin's frontend bundle.
type FrontendConfig struct {
	Entry string `json:"entry"`
	Style string `json:"style,omitempty"`
}

// BackendConfig describes the plugin's backend sidecar.
type BackendConfig struct {
	Type        string            `json:"type"`
	Entry       map[string]string `json:"entry"`
	HealthCheck *HealthCheckConfig `json:"healthCheck,omitempty"`
}

// HealthCheckConfig describes sidecar health check.
type HealthCheckConfig struct {
	Type    string `json:"type,omitempty"`
	Timeout int    `json:"timeout,omitempty"`
}

// MigrationConfig describes DB migrations.
type MigrationConfig struct {
	Path string `json:"path,omitempty"`
}

// Contributions describes UI and action contributions.
type Contributions struct {
	Views             []ContributionView              `json:"views,omitempty"`
	Commands          []ContributionCommand           `json:"commands,omitempty"`
	SettingsPanels    []ContributionSettingsPanel      `json:"settingsPanels,omitempty"`
	SidebarItems      []ContributionSidebarItem       `json:"sidebarItems,omitempty"`
	FileActions       []ContributionAction            `json:"fileActions,omitempty"`
	NoteActions       []ContributionAction            `json:"noteActions,omitempty"`
	ContextMenuEntries []ContributionContextMenuEntry `json:"contextMenuEntries,omitempty"`
	SearchProviders   []ContributionSearchProvider    `json:"searchProviders,omitempty"`
	ActivityProviders []ContributionActivityProvider  `json:"activityProviders,omitempty"`
	StatusBarItems    []ContributionStatusBarItem     `json:"statusBarItems,omitempty"`
}

// ContributionView represents a view contribution.
type ContributionView struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	Icon      string `json:"icon,omitempty"`
	Component string `json:"component"`
}

// ContributionCommand represents a command palette command.
type ContributionCommand struct {
	ID         string `json:"id"`
	Title      string `json:"title"`
	Keybinding string `json:"keybinding,omitempty"`
	Icon       string `json:"icon,omitempty"`
	Handler    string `json:"handler,omitempty"`
}

// ContributionSettingsPanel represents a settings panel.
type ContributionSettingsPanel struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	Component string `json:"component"`
	Icon      string `json:"icon,omitempty"`
}

// ContributionSidebarItem represents a sidebar item.
type ContributionSidebarItem struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Icon     string `json:"icon,omitempty"`
	View     string `json:"view"`
	Position int    `json:"position,omitempty"`
}

// ContributionAction represents a file or note action.
type ContributionAction struct {
	ID         string `json:"id"`
	Label      string `json:"label"`
	Icon       string `json:"icon,omitempty"`
	Capability string `json:"capability,omitempty"`
	Handler    string `json:"handler,omitempty"`
}

// ContributionContextMenuEntry represents a context menu entry.
type ContributionContextMenuEntry struct {
	ID         string `json:"id"`
	Label      string `json:"label"`
	Context    string `json:"context"`
	Group      string `json:"group,omitempty"`
	Capability string `json:"capability,omitempty"`
	Handler    string `json:"handler,omitempty"`
}

// ContributionSearchProvider represents a search provider.
type ContributionSearchProvider struct {
	ID      string `json:"id"`
	Label   string `json:"label"`
	Handler string `json:"handler"`
}

// ContributionActivityProvider represents an activity provider.
type ContributionActivityProvider struct {
	ID      string   `json:"id"`
	Events  []string `json:"events,omitempty"`
	Handler string   `json:"handler"`
}

// ContributionStatusBarItem represents a status bar item.
type ContributionStatusBarItem struct {
	ID       string `json:"id"`
	Label    string `json:"label"`
	Position string `json:"position,omitempty"`
	Handler  string `json:"handler,omitempty"`
}

// SyncConfig describes plugin sync configuration.
type SyncConfig struct {
	Namespaces  []string `json:"namespaces,omitempty"`
	Participate bool     `json:"participate,omitempty"`
}

// Status represents the current state of a plugin.
type Status string

const (
	StatusDiscovered              Status = "discovered"
	StatusDisabled                Status = "disabled"
	StatusLoading                 Status = "loading"
	StatusLoaded                  Status = "loaded"
	StatusDegraded                Status = "degraded"
	StatusFailed                  Status = "failed"
	StatusIncompatible            Status = "incompatible"
	StatusMissingRequiredCapability Status = "missing-required-capability"
)

// Plugin represents a loaded plugin instance.
type Plugin struct {
	Manifest Manifest `json:"manifest"`
	Status   Status   `json:"status"`
	Error    string   `json:"error,omitempty"`
	Enabled  bool     `json:"enabled"`
	RootPath string   `json:"rootPath"`
}

// validationErrors tracks manifest validation issues.
type validationErrors struct {
	errors []string
}

func (v *validationErrors) add(format string, args ...interface{}) {
	v.errors = append(v.errors, fmt.Sprintf(format, args...))
}

// ValidateManifest checks a manifest for required fields and valid values.
func ValidateManifest(m *Manifest) []string {
	var errs validationErrors

	if m.SchemaVersion != 1 {
		errs.add("schemaVersion must be 1, got %d", m.SchemaVersion)
	}
	if m.ID == "" {
		errs.add("id is required")
	} else if !isValidPluginID(m.ID) {
		errs.add("id %q must match pattern: alphanumeric, dots, hyphens", m.ID)
	}
	if m.Name == "" {
		errs.add("name is required")
	}
	if m.Version == "" {
		errs.add("version is required")
	}
	if m.APIVersion == "" {
		errs.add("apiVersion is required")
	}
	if len(m.Provides) == 0 {
		errs.add("provides must have at least one capability")
	}
	if len(m.Permissions) == 0 {
		errs.add("permissions must have at least one permission")
	}

	return errs.errors
}

func isValidPluginID(id string) bool {
	if id == "" {
		return false
	}
	for _, r := range id {
		if !isAllowedInID(r) {
			return false
		}
	}
	return true
}

func isAllowedInID(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') ||
		(r >= '0' && r <= '9') || r == '.' || r == '-'
}

// ─── Discovery ──────────────────────────────────────────────

// DiscoverPlugins scans the given directories for plugin.json manifests.
func DiscoverPlugins(dirs []string) ([]Plugin, []error) {
	var plugins []Plugin
	var errs []error

	seen := make(map[string]bool)

	for _, dir := range dirs {
		entries, err := os.ReadDir(dir)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			errs = append(errs, fmt.Errorf("reading plugin directory %s: %w", dir, err))
			continue
		}

		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}

			pluginDir := filepath.Join(dir, entry.Name())
			manifestPath := filepath.Join(pluginDir, "plugin.json")

			if _, err := os.Stat(manifestPath); os.IsNotExist(err) {
				continue
			}

			plugin, err := loadPlugin(pluginDir)
			if err != nil {
				errs = append(errs, fmt.Errorf("plugin %s: %w", entry.Name(), err))
				continue
			}

			if seen[plugin.Manifest.ID] {
				errs = append(errs, fmt.Errorf("duplicate plugin ID %q in %s", plugin.Manifest.ID, pluginDir))
				continue
			}
			seen[plugin.Manifest.ID] = true
			plugins = append(plugins, plugin)
		}
	}

	return plugins, errs
}

// loadPlugin reads and validates a plugin from its directory.
func loadPlugin(pluginDir string) (Plugin, error) {
	manifestPath := filepath.Join(pluginDir, "plugin.json")
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return Plugin{}, fmt.Errorf("reading manifest: %w", err)
	}

	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return Plugin{}, fmt.Errorf("parsing manifest: %w", err)
	}

	if errs := ValidateManifest(&m); len(errs) > 0 {
		return Plugin{}, fmt.Errorf("invalid manifest: %s", strings.Join(errs, "; "))
	}

	return Plugin{
		Manifest: m,
		Status:   StatusDiscovered,
		Enabled:  true,
		RootPath: pluginDir,
	}, nil
}
