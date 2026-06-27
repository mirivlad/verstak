package main

import (
	"embed"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"

	"github.com/verstak/verstak-desktop/internal/api"
	"github.com/verstak/verstak-desktop/internal/core/appsettings"
	"github.com/verstak/verstak-desktop/internal/core/capability"
	"github.com/verstak/verstak-desktop/internal/core/contribution"
	"github.com/verstak/verstak-desktop/internal/core/events"
	corefiles "github.com/verstak/verstak-desktop/internal/core/files"
	"github.com/verstak/verstak-desktop/internal/core/permissions"
	"github.com/verstak/verstak-desktop/internal/core/plugin"
	"github.com/verstak/verstak-desktop/internal/core/pluginstate"
	"github.com/verstak/verstak-desktop/internal/core/storage"
	syncsvc "github.com/verstak/verstak-desktop/internal/core/sync"
	"github.com/verstak/verstak-desktop/internal/core/vault"
	"github.com/verstak/verstak-desktop/internal/core/workspace"
	"github.com/verstak/verstak-desktop/internal/shell/debug"
)

//go:embed frontend/dist
var assets embed.FS

func main() {
	// ─── Debug Logging ───────────────────────────────────────
	debugEnabled := debug.Init(os.Args)
	if debugEnabled {
		log.Printf("[main] debug mode enabled — logging to file")
	}

	// ─── Initialize Core Registries ──────────────────────────
	capRegistry := capability.NewRegistry()
	contribRegistry := contribution.NewRegistry()
	permRegistry := permissions.NewRegistry()
	eventBus := events.NewBus()

	// ─── Initialize Vault ────────────────────────────────────
	vaultService := vault.NewVault(eventBus)

	// ─── Initialize App Settings ─────────────────────────────
	appSettingsMgr := appsettings.NewDefaultManager()
	if err := appSettingsMgr.Load(); err != nil {
		log.Printf("[main] app settings: %v", err)
	}

	// ─── Vault Auto-Open ─────────────────────────────────────
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

	// ─── Initialize Workspace ────────────────────────────────
	var workspaceMgr *workspace.Manager
	if vaultService.GetVaultStatus() == vault.StatusOpen {
		workspaceMgr = workspace.NewManager(vaultService.GetVaultPath())
		if err := workspaceMgr.Load(); err != nil {
			log.Printf("[main] workspace: %v", err)
			workspaceMgr = nil
		} else {
			log.Printf("[main] workspace loaded: %d nodes", len(workspaceMgr.GetTree().Nodes))
		}
	}

	// ─── Register Core Capabilities ─────────────────────────
	corePluginID := "verstak-desktop"
	coreCaps := []string{
		"verstak/core/plugin-manager/v1",
		"verstak/core/capability-registry/v1",
		"verstak/core/contribution-registry/v1",
		"verstak/core/permissions/v1",
		"verstak/core/events/v1",
		"verstak/core/files/v1",
		"verstak/core/workbench/v1",
	}
	if err := capRegistry.Register(corePluginID, coreCaps); err != nil {
		log.Fatalf("[main] failed to register core capabilities: %v", err)
	}
	log.Printf("[main] registered %d core capabilities", len(coreCaps))

	// Register vault capability
	if err := capRegistry.Register(corePluginID, []string{"verstak/core/vault/v1"}); err != nil {
		log.Fatalf("[main] failed to register vault capability: %v", err)
	}
	log.Printf("[main] registered vault capability")

	// Register workspace capability (only when vault is open and workspace initialized)
	if workspaceMgr != nil && workspaceMgr.IsInitialized() {
		if err := capRegistry.Register(corePluginID, []string{"verstak/core/workspace/v1"}); err != nil {
			log.Fatalf("[main] failed to register workspace capability: %v", err)
		}
		log.Printf("[main] registered workspace capability")
	}

	// ─── Plugin Discovery ───────────────────────────────────
	discoveryDirs := plugin.DefaultDiscoveryDirs()
	log.Printf("[main] plugin dirs: %v", discoveryDirs)
	if debugEnabled {
		debug.Logf("[main] plugin dirs: %v", discoveryDirs)
	}

	plugins, discErrors := plugin.DiscoverPlugins(discoveryDirs)
	for _, err := range discErrors {
		log.Printf("[plugin] discovery warning: %v", err)
		if debugEnabled {
			debug.Logf("[plugin] discovery warning: %v", err)
		}
	}

	log.Printf("[plugin] discovered %d plugins", len(plugins))
	if debugEnabled {
		for i, p := range plugins {
			debug.Logf("[plugin] discovered[%d]: id=%s name=%s version=%s source=%s root=%s",
				i, p.Manifest.ID, p.Manifest.Name, p.Manifest.Version, p.Manifest.Source, p.RootPath)
		}
	}

	// ─── Plugin Lifecycle: Register Capabilities + Contributions ──
	if debugEnabled {
		debug.Logf("[main] starting plugin lifecycle for %d plugins", len(plugins))
	}
	for i := range plugins {
		p := &plugins[i]

		if debugEnabled {
			debug.Logf("[main] lifecycle[%d]: id=%s status=%s enabled=%v", i, p.Manifest.ID, p.Status, p.Enabled)
		}

		// Check if plugin is disabled in vault plugin state
		if pluginStateMgr != nil && pluginStateMgr.IsDisabled(p.Manifest.ID) {
			log.Printf("[plugin] %s: disabled in vault plugin state — skipping", p.Manifest.ID)
			if debugEnabled {
				debug.Logf("[main] lifecycle: %s disabled in vault state, skipping", p.Manifest.ID)
			}
			p.Status = plugin.StatusDisabled
			p.Enabled = false
			continue
		}

		// Register provided capabilities
		if len(p.Manifest.Provides) > 0 {
			if err := capRegistry.Register(p.Manifest.ID, p.Manifest.Provides); err != nil {
				log.Printf("[plugin] %s: capability registration failed: %v", p.Manifest.ID, err)
				if debugEnabled {
					debug.Logf("[main] lifecycle: %s capability registration failed: %v", p.Manifest.ID, err)
				}
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
			if debugEnabled {
				debug.Logf("[main] lifecycle: %s missing required: %v", p.Manifest.ID, missingRequired)
			}
			p.Status = plugin.StatusMissingRequiredCapability
			p.Error = fmt.Sprintf("missing required: %s", strings.Join(missingRequired, ", "))
			continue
		}

		// Check optional capabilities for degraded mode
		missingOptional := capRegistry.CheckRequired(p.Manifest.OptionalRequires)
		if len(missingOptional) > 0 {
			log.Printf("[plugin] %s: missing optional capabilities (degraded): %v", p.Manifest.ID, missingOptional)
			if debugEnabled {
				debug.Logf("[main] lifecycle: %s missing optional (degraded): %v", p.Manifest.ID, missingOptional)
			}
			p.Status = plugin.StatusDegraded
		} else {
			p.Status = plugin.StatusLoaded
		}

		// Register contributions
		if p.Manifest.Contributes != nil {
			contribRegistry.Register(p.Manifest.ID, p.Manifest.Contributes)
			log.Printf("[plugin] %s: contributions registered", p.Manifest.ID)
			if debugEnabled {
				c := p.Manifest.Contributes
				debug.Logf("[main] lifecycle: %s contributions: views=%d commands=%d sidebar=%d settings=%d statusbar=%d",
					p.Manifest.ID, len(c.Views), len(c.Commands), len(c.SidebarItems), len(c.SettingsPanels), len(c.StatusBarItems))
			}
		}

		// Record as desired plugin in vault state (only if vault is open)
		if pluginStateMgr != nil && vaultService.GetVaultStatus() == vault.StatusOpen {
			source := p.Manifest.Source
			if source == "" {
				source = "unknown"
			}
			if err := pluginStateMgr.RecordDesiredPlugin(p.Manifest.ID, p.Manifest.Version, source); err != nil {
				log.Printf("[plugin] %s: failed to record desired: %v", p.Manifest.ID, err)
				if debugEnabled {
					debug.Logf("[main] lifecycle: %s failed to record desired: %v", p.Manifest.ID, err)
				}
			}
		}

		log.Printf("[plugin] %s: status=%s", p.Manifest.ID, p.Status)
		if debugEnabled {
			debug.Logf("[main] lifecycle: %s final status=%s enabled=%v", p.Manifest.ID, p.Status, p.Enabled)
		}
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
	filesService := corefiles.NewService(vaultService)
	var syncService *syncsvc.Service
	if vaultService.GetVaultStatus() == vault.StatusOpen {
		syncService = syncsvc.NewService(vaultService.GetVaultPath(), "")
	}
	app := api.NewApp(capRegistry, contribRegistry, permRegistry, eventBus, plugins, vaultService, storageService, filesService, appSettingsMgr, pluginStateMgr, workspaceMgr, syncService, debugEnabled)

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
