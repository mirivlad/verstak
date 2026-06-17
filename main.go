package main

import (
	"embed"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"

	"github.com/verstak/verstak-desktop/internal/api"
	"github.com/verstak/verstak-desktop/internal/core/appsettings"
	"github.com/verstak/verstak-desktop/internal/core/capability"
	"github.com/verstak/verstak-desktop/internal/core/contribution"
	"github.com/verstak/verstak-desktop/internal/core/events"
	"github.com/verstak/verstak-desktop/internal/core/permissions"
	"github.com/verstak/verstak-desktop/internal/core/plugin"
	"github.com/verstak/verstak-desktop/internal/core/pluginstate"
	"github.com/verstak/verstak-desktop/internal/core/storage"
	"github.com/verstak/verstak-desktop/internal/core/vault"
)

//go:embed frontend/dist
var assets embed.FS

// expandPath resolves "~" to the user's home directory.
func expandPath(path string) string {
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			log.Printf("[main] expandPath: cannot get home dir: %v", err)
			return path
		}
		return filepath.Join(home, path[2:])
	}
	return path
}

func main() {
	// ─── Initialize Core Registries ──────────────────────────
	capRegistry := capability.NewRegistry()
	contribRegistry := contribution.NewRegistry()
	permRegistry := permissions.NewRegistry()
	eventBus := events.NewBus()

	// ─── Initialize Vault ────────────────────────────────────
	vaultService := vault.NewVault(eventBus)

	// ─── Register Core Capabilities ─────────────────────────
	// These are provided by the desktop core itself, not by plugins.
	// Registered before plugin discovery so that plugins can resolve
	// required capabilities (e.g. verstak/core/plugin-manager/v1) at load time.
	corePluginID := "verstak-desktop"
	coreCaps := []string{
		"verstak/core/plugin-manager/v1",
		"verstak/core/capability-registry/v1",
		"verstak/core/contribution-registry/v1",
		"verstak/core/permissions/v1",
		"verstak/core/events/v1",
	}
	if err := capRegistry.Register(corePluginID, coreCaps); err != nil {
		log.Fatalf("[main] failed to register core capabilities: %v", err)
	}
	log.Printf("[main] registered %d core capabilities", len(coreCaps))

	// Register vault capability (vault is available as a core service)
	if err := capRegistry.Register(corePluginID, []string{"verstak/core/vault/v1"}); err != nil {
		log.Fatalf("[main] failed to register vault capability: %v", err)
	}
	log.Printf("[main] registered vault capability")

	// ─── Initialize App Settings ─────────────────────────────
	appSettingsMgr := appsettings.NewDefaultManager()
	if err := appSettingsMgr.Load(); err != nil {
		log.Printf("[main] app settings: %v", err)
	}

	// ─── Vault Auto-Open ─────────────────────────────────────
	// If currentVaultPath is set in app settings, try to open it.
	cfg := appSettingsMgr.Get()
	if cfg.CurrentVaultPath != "" {
		if err := vaultService.OpenVault(cfg.CurrentVaultPath); err != nil {
			log.Printf("[main] failed to auto-open vault at %s: %v", cfg.CurrentVaultPath, err)
		} else {
			log.Printf("[main] auto-opened vault at %s", cfg.CurrentVaultPath)
		}
	}

	// ─── Initialize Vault Plugin State ───────────────────────
	pluginStateMgr := pluginstate.NewManager(vaultService)
	if vaultService.GetVaultStatus() == vault.StatusOpen {
		if err := pluginStateMgr.Load(); err != nil {
			log.Printf("[main] vault plugin state: %v", err)
		}
	}

	// ─── Plugin Discovery ───────────────────────────────────
	// Resolve plugin directories relative to the binary location,
	// not CWD (Wails may launch from a different directory).
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

	log.Printf("[main] plugin dirs: %v", discoveryDirs)

	plugins, discErrors := plugin.DiscoverPlugins(discoveryDirs)
	for _, err := range discErrors {
		log.Printf("[plugin] discovery warning: %v", err)
	}

	log.Printf("[plugin] discovered %d plugins", len(plugins))

	// ─── Plugin Lifecycle: Register Capabilities + Contributions ──
	for i := range plugins {
		p := &plugins[i]

		// Check if plugin is disabled in vault plugin state
		if pluginStateMgr != nil && pluginStateMgr.IsDisabled(p.Manifest.ID) {
			log.Printf("[plugin] %s: disabled in vault plugin state — skipping", p.Manifest.ID)
			p.Status = plugin.StatusDisabled
			p.Enabled = false
			continue
		}

		// Register provided capabilities
		if len(p.Manifest.Provides) > 0 {
			if err := capRegistry.Register(p.Manifest.ID, p.Manifest.Provides); err != nil {
				log.Printf("[plugin] %s: capability registration failed: %v", p.Manifest.ID, err)
				p.Status = plugin.StatusFailed
				p.Error = err.Error()
				continue
			}
			log.Printf("[plugin] %s: registered %d capabilities", p.Manifest.ID, len(p.Manifest.Provides))
		}

		// Resolve required capabilities
		missingRequired := capRegistry.CheckRequired(p.Manifest.Requires)
		if len(missingRequired) > 0 {
			log.Printf("[plugin] %s: missing required capabilities: %v", p.Manifest.ID, missingRequired)
			p.Status = plugin.StatusMissingRequiredCapability
			p.Error = fmt.Sprintf("missing required: %s", strings.Join(missingRequired, ", "))
			continue
		}

		// Check optional capabilities for degraded mode
		missingOptional := capRegistry.CheckRequired(p.Manifest.OptionalRequires)
		if len(missingOptional) > 0 {
			log.Printf("[plugin] %s: missing optional capabilities (degraded): %v", p.Manifest.ID, missingOptional)
			p.Status = plugin.StatusDegraded
		} else {
			p.Status = plugin.StatusLoaded
		}

		// Register contributions
		if p.Manifest.Contributes != nil {
			contribRegistry.Register(p.Manifest.ID, p.Manifest.Contributes)
			log.Printf("[plugin] %s: contributions registered", p.Manifest.ID)
		}

		// Record as desired plugin in vault state (only if vault is open)
		if pluginStateMgr != nil && vaultService.GetVaultStatus() == vault.StatusOpen {
			source := p.Manifest.Source
			if source == "" {
				source = "unknown"
			}
			if err := pluginStateMgr.RecordDesiredPlugin(p.Manifest.ID, p.Manifest.Version, source); err != nil {
				log.Printf("[plugin] %s: failed to record desired: %v", p.Manifest.ID, err)
			}
		}

		log.Printf("[plugin] %s: status=%s", p.Manifest.ID, p.Status)
	}

	// ─── Log Summary ───────────────────────────────────────
	loaded := 0
	degraded := 0
	failed := 0
	for _, p := range plugins {
		switch p.Status {
		case plugin.StatusLoaded:
			loaded++
		case plugin.StatusDegraded:
			degraded++
		default:
			failed++
		}
	}
	log.Printf("[main] lifecycle summary: loaded=%d degraded=%d failed=%d vault=%s",
		loaded, degraded, failed, vaultService.GetVaultStatus())

	// Create the App struct
	storageService := storage.New(vaultService)
	app := api.NewApp(capRegistry, contribRegistry, permRegistry, eventBus, plugins, vaultService, storageService, appSettingsMgr, pluginStateMgr)

	// ─── Wails App ───────────────────────────────────────────
	err := wails.Run(&options.App{
		Title:            "Verstak",
		Width:            1200,
		Height:           800,
		MinWidth:         800,
		MinHeight:        600,
		WindowStartState: options.Normal,
		OnStartup:        app.Startup,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		Bind: []interface{}{
			app,
		},
	})

	if err != nil {
		log.Fatalf("Error: %v", err)
	}
}
