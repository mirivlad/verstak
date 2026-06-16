// Package api provides Wails-bound methods for the frontend.
package api

import (
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/verstak/verstak-desktop/internal/core/capability"
	"github.com/verstak/verstak-desktop/internal/core/contribution"
	"github.com/verstak/verstak-desktop/internal/core/events"
	"github.com/verstak/verstak-desktop/internal/core/permissions"
	"github.com/verstak/verstak-desktop/internal/core/plugin"
)

// App is the main application struct exposed to the Wails frontend.
type App struct {
	capRegistry     *capability.Registry
	contribRegistry *contribution.Registry
	permRegistry    *permissions.Registry
	eventBus        *events.Bus
	plugins         []plugin.Plugin
}

// NewApp creates a new App instance.
func NewApp(
	capReg *capability.Registry,
	contribReg *contribution.Registry,
	permReg *permissions.Registry,
	bus *events.Bus,
	plugins []plugin.Plugin,
) *App {
	return &App{
		capRegistry:     capReg,
		contribRegistry: contribReg,
		permRegistry:    permReg,
		eventBus:        bus,
		plugins:         plugins,
	}
}

// Startup is called when the app starts.
func (a *App) Startup() error {
	log.Printf("[api] App.Startup: initialized with %d plugins", len(a.plugins))
	return nil
}

// ─── Plugin Manager API ─────────────────────────────────────

// GetPlugins returns all discovered plugins.
func (a *App) GetPlugins() []plugin.Plugin {
	log.Printf("[api] GetPlugins: returning %d plugins", len(a.plugins))
	return a.plugins
}

// GetCapabilities returns all registered capabilities.
func (a *App) GetCapabilities() []capability.Entry {
	entries := a.capRegistry.List()
	log.Printf("[api] GetCapabilities: returning %d entries", len(entries))
	return entries
}

// GetPermissions returns all known permissions.
func (a *App) GetPermissions() []permissions.Entry {
	entries := a.permRegistry.List()
	log.Printf("[api] GetPermissions: returning %d entries", len(entries))
	return entries
}

// GetContributions returns all registered contributions.
func (a *App) GetContributions() ContributionSummary {
	return ContributionSummary{
		Views:           a.contribRegistry.Views(),
		Commands:        a.contribRegistry.Commands(),
		SettingsPanels:  a.contribRegistry.SettingsPanels(),
		SidebarItems:    a.contribRegistry.SidebarItems(),
		FileActions:     a.contribRegistry.FileActions(),
		NoteActions:     a.contribRegistry.NoteActions(),
		SearchProviders: a.contribRegistry.SearchProviders(),
	}
}

// expandPath resolves "~" to the user's home directory.
func expandPath(path string) string {
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			log.Printf("[api] expandPath: cannot get home dir: %v", err)
			return path
		}
		return filepath.Join(home, path[2:])
	}
	return path
}

// ReloadPlugins re-discovers plugins from disk and returns a summary.
func (a *App) ReloadPlugins() (int, string) {
	discoveryDirs := []string{
		"~/.config/verstak/plugins",
		"./plugins",
	}

	// Expand tilde in all paths
	for i, d := range discoveryDirs {
		discoveryDirs[i] = expandPath(d)
	}

	log.Printf("[api] ReloadPlugins: scanning dirs: %v", discoveryDirs)

	plugins, errs := plugin.DiscoverPlugins(discoveryDirs)
	a.plugins = plugins

	var buf strings.Builder
	buf.WriteString("discovery complete")
	if len(plugins) > 0 {
		buf.WriteString(": ")
		buf.WriteString(plugin.FormatDiscoverySummary(plugins))
	}

	if len(errs) > 0 {
		log.Printf("[api] ReloadPlugins: %d warning(s)", len(errs))
		for _, e := range errs {
			log.Printf("[api]   discovery warning: %v", e)
		}
	}

	log.Printf("[api] ReloadPlugins: discovered %d plugin(s)", len(plugins))

	discoveryDirsStr := strings.Join(discoveryDirs, ", ")
	summary := buf.String()

	log.Printf("[api] ReloadPlugins: dirs=[%s] %s", discoveryDirsStr, summary)

	return len(plugins), summary
}

// ContributionSummary aggregates all contribution types for the frontend.
type ContributionSummary struct {
	Views           []contribution.ContributionView           `json:"views"`
	Commands        []contribution.ContributionCommand        `json:"commands"`
	SettingsPanels  []contribution.ContributionSettingsPanel  `json:"settingsPanels"`
	SidebarItems    []contribution.ContributionSidebarItem    `json:"sidebarItems"`
	FileActions     []contribution.ContributionAction         `json:"fileActions"`
	NoteActions     []contribution.ContributionAction         `json:"noteActions"`
	SearchProviders []contribution.ContributionSearchProvider `json:"searchProviders"`
}
