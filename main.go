package main

import (
	"context"
	"embed"
	"log"
	"os"
	"time"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"

	"github.com/verstak/verstak-desktop/internal/api"
	"github.com/verstak/verstak-desktop/internal/core/appsettings"
	"github.com/verstak/verstak-desktop/internal/core/browserreceiver"
	"github.com/verstak/verstak-desktop/internal/core/capability"
	"github.com/verstak/verstak-desktop/internal/core/contribution"
	"github.com/verstak/verstak-desktop/internal/core/events"
	corefiles "github.com/verstak/verstak-desktop/internal/core/files"
	"github.com/verstak/verstak-desktop/internal/core/notifications"
	"github.com/verstak/verstak-desktop/internal/core/permissions"
	"github.com/verstak/verstak-desktop/internal/core/plugin"
	"github.com/verstak/verstak-desktop/internal/core/pluginstate"
	"github.com/verstak/verstak-desktop/internal/core/storage"
	syncsvc "github.com/verstak/verstak-desktop/internal/core/sync"
	"github.com/verstak/verstak-desktop/internal/core/vault"
	"github.com/verstak/verstak-desktop/internal/core/workspace"
	"github.com/verstak/verstak-desktop/internal/shell/debug"
	"github.com/verstak/verstak-desktop/internal/shell/tray"
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
	corePluginID := capability.CorePluginID
	coreCaps := capability.CorePlatformCapabilities()
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
	plugin.ResolveLifecycle(plugins, capRegistry, func(pluginID string) bool {
		return pluginStateMgr != nil && pluginStateMgr.IsDisabled(pluginID)
	})
	for i := range plugins {
		p := &plugins[i]

		if debugEnabled {
			debug.Logf("[main] lifecycle[%d]: id=%s status=%s enabled=%v", i, p.Manifest.ID, p.Status, p.Enabled)
		}

		if p.Status != plugin.StatusLoaded && p.Status != plugin.StatusDegraded {
			if p.Error != "" {
				log.Printf("[plugin] %s: status=%s: %s", p.Manifest.ID, p.Status, p.Error)
			}
			continue
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
	receiverToken, err := appSettingsMgr.EnsureBrowserReceiverToken()
	if err != nil {
		log.Printf("[browserreceiver] local receiver disabled: %v", err)
	}
	var app *api.App
	var browserReceiver *browserreceiver.Receiver
	if receiverToken != "" {
		browserReceiver = browserreceiver.NewWithOptions(eventBus, browserreceiver.Options{
			RequireToken:  true,
			ReceiverToken: receiverToken,
		}, func() string {
			if app == nil {
				return ""
			}
			current := app.GetCurrentWorkspace()
			if root, ok := current["rootPath"].(string); ok {
				return root
			}
			return ""
		})
	}
	app = api.NewApp(capRegistry, contribRegistry, permRegistry, eventBus, plugins, vaultService, storageService, filesService, appSettingsMgr, pluginStateMgr, workspaceMgr, syncService, browserReceiver, debugEnabled)
	app.SetNotificationService(notifications.New(vaultService, api.NewNativeNotificationSender(), time.Now))
	trayController := tray.New(tray.NewNativeBackend(), tray.DefaultIcon())
	trayController.SetReadyChangedHandler(app.SetTrayReady)
	trayLabels := func(language string) tray.Labels {
		return tray.LabelsForPreference(language, os.Getenv("LC_ALL"), os.Getenv("LC_MESSAGES"), os.Getenv("LANG"))
	}
	trayController.SetLabels(trayLabels(cfg.Language))
	appSettingsMgr.SetLanguageChangedHandler(func(language string) {
		trayController.SetLabels(trayLabels(language))
	})
	if browserReceiver != nil {
		browserReceiverServer, err := browserreceiver.Start(browserreceiver.DefaultAddr, browserReceiver)
		if err != nil {
			log.Printf("[browserreceiver] local receiver disabled: %v", err)
		} else {
			defer browserReceiverServer.Close()
			log.Printf("[browserreceiver] paired local receiver listening at %s", browserReceiverServer.URL())
		}
	}

	// ─── Wails App ───────────────────────────────────────────
	appOptions := &options.App{
		Title:            "Verstak",
		Width:            1200,
		Height:           800,
		MinWidth:         800,
		MinHeight:        600,
		WindowStartState: options.Normal,
		OnStartup:        app.Startup,
		OnDomReady: func(ctx context.Context) {
			app.DomReady(ctx)
			if err := trayController.Start(tray.Actions{Show: app.ShowWindow, Quit: app.Quit}); err != nil {
				log.Printf("[tray] disabled: %v; normal window close will exit", err)
			}
		},
		OnShutdown: func(ctx context.Context) {
			trayController.Stop()
			app.Shutdown(ctx)
		},
		OnBeforeClose: app.BeforeClose,
		SingleInstanceLock: &options.SingleInstanceLock{
			UniqueId: "605fba28-7cbf-4f14-9d1b-a4da0c1723f8",
			OnSecondInstanceLaunch: func(options.SecondInstanceData) {
				app.ShowWindow()
			},
		},
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		Bind: []interface{}{
			app,
		},
	}
	err = wails.Run(appOptions)

	if err != nil {
		log.Fatalf("Error: %v", err)
	}
}
