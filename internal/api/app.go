// Package api provides Wails-bound methods for the frontend.
package api

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/verstak/verstak-desktop/internal/core/capability"
	"github.com/verstak/verstak-desktop/internal/core/contribution"
	"github.com/verstak/verstak-desktop/internal/core/events"
	"github.com/verstak/verstak-desktop/internal/core/permissions"
	"github.com/verstak/verstak-desktop/internal/core/plugin"
	"github.com/verstak/verstak-desktop/internal/core/vault"
)

// App is the main application struct exposed to the Wails frontend.
type App struct {
	capRegistry     *capability.Registry
	contribRegistry *contribution.Registry
	permRegistry    *permissions.Registry
	eventBus        *events.Bus
	plugins         []plugin.Plugin
	vault           *vault.Vault
}

// NewApp creates a new App instance.
func NewApp(
	capReg *capability.Registry,
	contribReg *contribution.Registry,
	permReg *permissions.Registry,
	bus *events.Bus,
	plugins []plugin.Plugin,
	vaultService *vault.Vault,
) *App {
	return &App{
		capRegistry:     capReg,
		contribRegistry: contribReg,
		permRegistry:    permReg,
		eventBus:        bus,
		plugins:         plugins,
		vault:           vaultService,
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
	// Resolve plugin directories relative to the binary location
	binDir := filepath.Dir(os.Args[0])
	pluginDir := filepath.Join(binDir, "plugins")

	discoveryDirs := []string{
		"~/.config/verstak/plugins",
		pluginDir,
	}

	// Expand tilde in all paths
	for i, d := range discoveryDirs {
		discoveryDirs[i] = expandPath(d)
	}

	log.Printf("[api] ReloadPlugins: scanning dirs: %v", discoveryDirs)

	// Unregister all non-core capabilities
	a.capRegistry.UnregisterAll()

	// Re-register core capabilities
	coreCaps := []string{
		"verstak/core/plugin-manager/v1",
		"verstak/core/capability-registry/v1",
		"verstak/core/contribution-registry/v1",
		"verstak/core/permissions/v1",
		"verstak/core/events/v1",
	}
	if err := a.capRegistry.Register("verstak-desktop", coreCaps); err != nil {
		log.Printf("[api] ReloadPlugins: failed to re-register core capabilities: %v", err)
	}

	// Re-register vault capability if vault is open
	if a.vault != nil && a.vault.GetVaultStatus() == vault.StatusOpen {
		if err := a.capRegistry.Register("verstak-desktop", []string{"verstak/core/vault/v1"}); err != nil {
			log.Printf("[api] ReloadPlugins: failed to re-register vault capability: %v", err)
		}
	}

	plugins, errs := plugin.DiscoverPlugins(discoveryDirs)

	// Plugin lifecycle: register capabilities + contributions
	for i := range plugins {
		p := &plugins[i]

		if len(p.Manifest.Provides) > 0 {
			if err := a.capRegistry.Register(p.Manifest.ID, p.Manifest.Provides); err != nil {
				log.Printf("[plugin] %s: capability registration failed: %v", p.Manifest.ID, err)
				p.Status = plugin.StatusFailed
				p.Error = err.Error()
				continue
			}
		}

		missingRequired := a.capRegistry.CheckRequired(p.Manifest.Requires)
		if len(missingRequired) > 0 {
			p.Status = plugin.StatusMissingRequiredCapability
			p.Error = fmt.Sprintf("missing required: %s", strings.Join(missingRequired, ", "))
			continue
		}

		missingOptional := a.capRegistry.CheckRequired(p.Manifest.OptionalRequires)
		if len(missingOptional) > 0 {
			p.Status = plugin.StatusDegraded
		} else {
			p.Status = plugin.StatusLoaded
		}

		if p.Manifest.Contributes != nil {
			a.contribRegistry.Register(p.Manifest.ID, p.Manifest.Contributes)
		}
	}

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

// ─── Vault API ──────────────────────────────────────────────

// GetVaultStatus returns the current vault status, path, and vault ID.
func (a *App) GetVaultStatus() map[string]string {
	status := "not-created"
	path := ""
	vaultID := ""

	if a.vault != nil {
		status = string(a.vault.GetVaultStatus())
		path = a.vault.GetVaultPath()
		meta := a.vault.GetVaultMeta()
		if meta != nil {
			vaultID = meta.VaultID
		}
	}

	return map[string]string{
		"status":  status,
		"path":    path,
		"vaultId": vaultID,
	}
}

// CreateVault creates a new vault at the given path.
func (a *App) CreateVault(path string) error {
	if a.vault == nil {
		return fmt.Errorf("vault service not initialized")
	}
	return a.vault.CreateVault(path)
}

// OpenVault opens an existing vault at the given path.
func (a *App) OpenVault(path string) error {
	if a.vault == nil {
		return fmt.Errorf("vault service not initialized")
	}
	return a.vault.OpenVault(path)
}

// CloseVault closes the current vault.
func (a *App) CloseVault() error {
	if a.vault == nil {
		return fmt.Errorf("vault service not initialized")
	}
	a.vault.CloseVault()
	return nil
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
