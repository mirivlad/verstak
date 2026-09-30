// Package api provides Wails-bound methods for the frontend.
package api

import (
	"context"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"github.com/verstak/verstak-desktop/internal/core/appsettings"
	"github.com/verstak/verstak-desktop/internal/core/browserreceiver"
	"github.com/verstak/verstak-desktop/internal/core/capability"
	"github.com/verstak/verstak-desktop/internal/core/contribution"
	"github.com/verstak/verstak-desktop/internal/core/events"
	"github.com/verstak/verstak-desktop/internal/core/externalopen"
	corefiles "github.com/verstak/verstak-desktop/internal/core/files"
	"github.com/verstak/verstak-desktop/internal/core/filewatcher"
	"github.com/verstak/verstak-desktop/internal/core/importservice"
	"github.com/verstak/verstak-desktop/internal/core/notifications"
	"github.com/verstak/verstak-desktop/internal/core/permissions"
	"github.com/verstak/verstak-desktop/internal/core/plugin"
	"github.com/verstak/verstak-desktop/internal/core/pluginstate"
	coresecrets "github.com/verstak/verstak-desktop/internal/core/secrets"
	"github.com/verstak/verstak-desktop/internal/core/storage"
	syncsvc "github.com/verstak/verstak-desktop/internal/core/sync"
	"github.com/verstak/verstak-desktop/internal/core/vault"
	coreworkbench "github.com/verstak/verstak-desktop/internal/core/workbench"
	"github.com/verstak/verstak-desktop/internal/core/workspacetree"
)

var emitFrontendEvent = runtime.EventsEmit
var initializeNativeNotifications = runtime.InitializeNotifications
var cleanupNativeNotifications = runtime.CleanupNotifications
var sendNativeNotification = runtime.SendNotification
var hideNativeWindow = runtime.WindowHide
var showNativeWindow = runtime.WindowShow
var quitNativeApplication = runtime.Quit

type notificationService interface {
	Replace(pluginID string, requests []notifications.Request) error
	Clear(pluginID string) error
	Start(ctx context.Context)
	Stop()
}

// App is the main application struct exposed to the Wails frontend.
type App struct {
	ctx                   context.Context
	capRegistry           *capability.Registry
	contribRegistry       *contribution.Registry
	permRegistry          *permissions.Registry
	eventBus              *events.Bus
	plugins               []plugin.Plugin
	vault                 *vault.Vault
	storage               *storage.Storage
	files                 *corefiles.Service
	externalOpen          externalOpenService
	appSettings           *appsettings.Manager
	pluginState           *pluginstate.Manager
	workbench             *coreworkbench.Router
	treeV2                *workspacetree.Service
	syncSvc               *syncsvc.Service
	browserReceiver       *browserreceiver.Receiver
	secretsSession        *coresecrets.VaultSession
	fileWatcher           *filewatcher.Service
	importsMu             sync.RWMutex
	imports               *importservice.Service
	selectImportDirectory func(context.Context, runtime.OpenDialogOptions) (string, error)
	selectImportArchive   func(context.Context, runtime.OpenDialogOptions) (string, error)
	syncRunMu             sync.Mutex
	syncRunning           atomic.Bool
	transfersMu           sync.Mutex
	cancelledTransfers    map[string]bool
	syncTimerMu           sync.Mutex
	syncScanTimer         *time.Timer
	notifications         notificationService
	debug                 bool
	activityEvents        map[string]bool
	browserInboxEvents    map[string]bool
	browserInboxEnabled   atomic.Bool
	allowQuit             atomic.Bool
	trayReady             atomic.Bool
	quitOnce              sync.Once
}

// SetNotificationService attaches the core-owned plugin notification scheduler.
func (a *App) SetNotificationService(service notificationService) {
	if a == nil {
		return
	}
	a.notifications = service
}

// NewApp creates a new App instance.
func NewApp(
	capReg *capability.Registry,
	contribReg *contribution.Registry,
	permReg *permissions.Registry,
	bus *events.Bus,
	plugins []plugin.Plugin,
	vaultService *vault.Vault,
	storageService *storage.Storage,
	filesService *corefiles.Service,
	appSettingsMgr *appsettings.Manager,
	pluginStateMgr *pluginstate.Manager,
	syncService *syncsvc.Service,
	browserReceiverService *browserreceiver.Receiver,
	debugEnabled bool,
) *App {
	app := &App{
		capRegistry:           capReg,
		contribRegistry:       contribReg,
		permRegistry:          permReg,
		eventBus:              bus,
		plugins:               plugins,
		vault:                 vaultService,
		storage:               storageService,
		files:                 filesService,
		externalOpen:          externalopen.NewService(),
		appSettings:           appSettingsMgr,
		pluginState:           pluginStateMgr,
		workbench:             coreworkbench.NewRouter(workbenchPrefsFromSettings(appSettingsMgr)),
		syncSvc:               syncService,
		browserReceiver:       browserReceiverService,
		fileWatcher:           filewatcher.NewService(bus, 0),
		selectImportDirectory: runtime.OpenDirectoryDialog,
		selectImportArchive:   runtime.OpenFileDialog,
		debug:                 debugEnabled,
		activityEvents:        make(map[string]bool),
		browserInboxEvents:    make(map[string]bool),
	}
	if app.syncSvc == nil {
		app.rebindSyncService()
	}
	app.ensureActivityProviderSubscriptions()
	app.ensureBrowserInboxSubscriptions()
	if app.browserReceiver != nil {
		app.browserReceiver.SetPersistence(app.browserInboxAvailable, app.recordBrowserCapture)
		app.browserReceiver.SetActivityPersistence(app.activityAvailable, app.recordBrowserActivityBatch)
	}
	app.startFileWatcherForOpenVault()
	return app
}

// Startup is called when the app starts. Sets the Wails context for dialogs.
func (a *App) Startup(ctx context.Context) {
	a.ctx = ctx
	a.ensureActivityProviderSubscriptions()
	a.ensureBrowserInboxSubscriptions()
	log.Printf("[api] App.Startup: initialized with %d plugins", len(a.plugins))
}

// DomReady initializes the native notification runtime before starting schedules.
func (a *App) DomReady(ctx context.Context) {
	if a.notifications == nil {
		return
	}
	if err := initializeNativeNotifications(ctx); err != nil {
		log.Printf("[api] native notifications unavailable: %v", err)
		return
	}
	a.notifications.Start(ctx)
}

// Shutdown stops scheduled delivery before releasing native notification resources.
func (a *App) Shutdown(ctx context.Context) {
	if service := a.currentImportService(); service != nil {
		service.CloseAll()
	}
	if a.notifications != nil {
		a.notifications.Stop()
	}
	cleanupNativeNotifications(ctx)
}

// SetTrayReady reports whether the native tray can safely return a hidden window.
func (a *App) SetTrayReady(ready bool) {
	if a != nil {
		a.trayReady.Store(ready)
	}
}

// BeforeClose hides the primary window only after the native tray has confirmed
// that it can return the window. Otherwise the normal Wails close path exits.
func (a *App) BeforeClose(ctx context.Context) bool {
	if a.allowQuit.Load() {
		return false
	}
	if !a.trayReady.Load() {
		log.Printf("[app] tray is unavailable; allowing normal window close")
		return false
	}
	hideNativeWindow(ctx)
	return true
}

// ShowWindow brings the primary window back from the tray.
func (a *App) ShowWindow() {
	if a == nil || a.ctx == nil {
		return
	}
	showNativeWindow(a.ctx)
}

// Quit allows the close event and ends the application process.
func (a *App) Quit() {
	if a == nil {
		return
	}
	a.quitOnce.Do(func() {
		a.allowQuit.Store(true)
		if a.ctx != nil {
			quitNativeApplication(a.ctx)
		}
	})
}

// NativeNotificationSender delivers scheduler items through the Wails runtime.
type NativeNotificationSender struct{}

// NewNativeNotificationSender creates the adapter used by the core scheduler.
func NewNativeNotificationSender() notifications.Sender {
	return NativeNotificationSender{}
}

// Send shows a native system notification for a scheduled plugin reminder.
func (NativeNotificationSender) Send(ctx context.Context, item notifications.Item) error {
	return sendNativeNotification(ctx, runtime.NotificationOptions{
		ID:    "verstak:" + item.PluginID + ":" + item.ID,
		Title: item.Title,
		Body:  item.Body,
	})
}

func (a *App) findPlugin(pluginID string) (*plugin.Plugin, error) {
	for i := range a.plugins {
		if a.plugins[i].Manifest.ID == pluginID {
			return &a.plugins[i], nil
		}
	}
	return nil, fmt.Errorf("plugin %q not found", pluginID)
}

func (a *App) requirePluginAccess(pluginID, permission string) (*plugin.Plugin, error) {
	p, err := a.findPlugin(pluginID)
	if err != nil {
		return nil, err
	}
	if !p.Enabled || (p.Status != plugin.StatusLoaded && p.Status != plugin.StatusDegraded) {
		return nil, fmt.Errorf("plugin %q is not enabled and loaded: status=%s enabled=%v", pluginID, p.Status, p.Enabled)
	}
	if permission != "" && !hasString(p.Manifest.Permissions, permission) {
		return nil, fmt.Errorf("plugin %q lacks required permission %q", pluginID, permission)
	}
	return p, nil
}

func (a *App) requirePluginCapabilityAccess(pluginID, capabilityName string) (*plugin.Plugin, error) {
	p, err := a.requirePluginAccess(pluginID, "")
	if err != nil {
		return nil, err
	}
	if !hasString(p.Manifest.Requires, capabilityName) && !hasString(p.Manifest.OptionalRequires, capabilityName) {
		return nil, fmt.Errorf("plugin %q does not declare capability dependency %q", pluginID, capabilityName)
	}
	return p, nil
}

func hasString(items []string, want string) bool {
	for _, item := range items {
		if item == want {
			return true
		}
	}
	return false
}

func (a *App) requireVault() error {
	if a.vault == nil || a.vault.GetVaultStatus() != vault.StatusOpen {
		return fmt.Errorf("vault not open")
	}
	return nil
}

func (a *App) vaultPath() string {
	if a.vault == nil {
		return ""
	}
	return a.vault.GetVaultPath()
}
