// Package api provides Wails-bound methods for the frontend.
package api

import (
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
	return nil
}

// ─── Plugin Manager API ─────────────────────────────────────

// GetPlugins returns all discovered plugins.
func (a *App) GetPlugins() []plugin.Plugin {
	return a.plugins
}

// GetCapabilities returns all registered capabilities.
func (a *App) GetCapabilities() []capability.Entry {
	return a.capRegistry.List()
}

// GetPermissions returns all known permissions.
func (a *App) GetPermissions() []permissions.Entry {
	return a.permRegistry.List()
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

// ReloadPlugins re-discovers plugins from disk.
func (a *App) ReloadPlugins() {
	discoveryDirs := []string{
		"~/.config/verstak/plugins",
		"./plugins",
	}
	plugins, _ := plugin.DiscoverPlugins(discoveryDirs)
	a.plugins = plugins
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
