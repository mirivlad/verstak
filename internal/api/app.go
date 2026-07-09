// Package api provides Wails-bound methods for the frontend.
package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"strings"
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
	"github.com/verstak/verstak-desktop/internal/core/permissions"
	"github.com/verstak/verstak-desktop/internal/core/plugin"
	"github.com/verstak/verstak-desktop/internal/core/pluginstate"
	coresecrets "github.com/verstak/verstak-desktop/internal/core/secrets"
	"github.com/verstak/verstak-desktop/internal/core/storage"
	syncsvc "github.com/verstak/verstak-desktop/internal/core/sync"
	"github.com/verstak/verstak-desktop/internal/core/vault"
	coreworkbench "github.com/verstak/verstak-desktop/internal/core/workbench"
	"github.com/verstak/verstak-desktop/internal/core/workspace"
	"github.com/verstak/verstak-desktop/internal/shell/debug"
)

var newSyncClient = syncsvc.NewClient
var emitFrontendEvent = runtime.EventsEmit

const pluginEventRuntimeName = "verstak:plugin-event"
const activityGlobalKey = "events:global"
const activityWorkspacePrefix = "events:workspace:"
const maxActivityEvents = 250
const workspaceCreatedEventName = "workspace.created"
const workspaceRenamedEventName = "workspace.renamed"
const workspaceTrashedEventName = "workspace.trashed"
const workspaceSelectedEventName = "workspace.selected"

// App is the main application struct exposed to the Wails frontend.
type App struct {
	ctx             context.Context
	capRegistry     *capability.Registry
	contribRegistry *contribution.Registry
	permRegistry    *permissions.Registry
	eventBus        *events.Bus
	plugins         []plugin.Plugin
	vault           *vault.Vault
	storage         *storage.Storage
	files           *corefiles.Service
	externalOpen    externalOpenService
	appSettings     *appsettings.Manager
	pluginState     *pluginstate.Manager
	workbench       *coreworkbench.Router
	workspace       *workspace.Manager
	syncSvc         *syncsvc.Service
	browserReceiver *browserreceiver.Receiver
	secretsSession  *coresecrets.VaultSession
	fileWatcher     *filewatcher.Service
	debug           bool
	activityEvents  map[string]bool
}

type externalOpenService interface {
	OpenPath(path string) error
	ShowInFolder(path string, isDir bool) error
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
	workspaceMgr *workspace.Manager,
	syncService *syncsvc.Service,
	browserReceiverService *browserreceiver.Receiver,
	debugEnabled bool,
) *App {
	app := &App{
		capRegistry:     capReg,
		contribRegistry: contribReg,
		permRegistry:    permReg,
		eventBus:        bus,
		plugins:         plugins,
		vault:           vaultService,
		storage:         storageService,
		files:           filesService,
		externalOpen:    externalopen.NewService(),
		appSettings:     appSettingsMgr,
		pluginState:     pluginStateMgr,
		workbench:       coreworkbench.NewRouter(workbenchPrefsFromSettings(appSettingsMgr)),
		workspace:       workspaceMgr,
		syncSvc:         syncService,
		browserReceiver: browserReceiverService,
		fileWatcher:     filewatcher.NewService(bus, 0),
		debug:           debugEnabled,
		activityEvents:  make(map[string]bool),
	}
	if app.syncSvc == nil {
		app.rebindSyncService()
	}
	app.ensureActivityProviderSubscriptions()
	app.startFileWatcherForOpenVault()
	return app
}

func workbenchPrefsFromSettings(m *appsettings.Manager) coreworkbench.Preferences {
	if m == nil {
		return coreworkbench.Preferences{}
	}
	cfg := m.Get()
	return coreworkbench.Preferences{
		DefaultTextEditorProvider:          cfg.Workbench.DefaultTextEditorProvider,
		DefaultMarkdownEditorProvider:      cfg.Workbench.DefaultMarkdownEditorProvider,
		DefaultNotesMarkdownEditorProvider: cfg.Workbench.DefaultNotesMarkdownEditorProvider,
	}
}

func appSettingsWorkbenchPrefs(p coreworkbench.Preferences) appsettings.WorkbenchPreferences {
	return appsettings.WorkbenchPreferences{
		DefaultTextEditorProvider:          p.DefaultTextEditorProvider,
		DefaultMarkdownEditorProvider:      p.DefaultMarkdownEditorProvider,
		DefaultNotesMarkdownEditorProvider: p.DefaultNotesMarkdownEditorProvider,
	}
}

func (a *App) ensureWorkbench() *coreworkbench.Router {
	if a.workbench == nil {
		a.workbench = coreworkbench.NewRouter(workbenchPrefsFromSettings(a.appSettings))
	}
	return a.workbench
}

// Startup is called when the app starts. Sets the Wails context for dialogs.
func (a *App) Startup(ctx context.Context) {
	a.ctx = ctx
	a.ensureActivityProviderSubscriptions()
	log.Printf("[api] App.Startup: initialized with %d plugins", len(a.plugins))
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

func (a *App) ensureActivityProviderSubscriptions() {
	if a.eventBus == nil || a.contribRegistry == nil {
		return
	}
	if a.activityEvents == nil {
		a.activityEvents = make(map[string]bool)
	}
	for _, provider := range a.contribRegistry.ActivityProviders() {
		for _, eventName := range provider.Item.Events {
			eventName = strings.TrimSpace(eventName)
			if eventName == "" || a.activityEvents[eventName] {
				continue
			}
			a.activityEvents[eventName] = true
			a.eventBus.Subscribe(eventName, func(event events.Event) {
				a.recordActivityProviderEvent(event)
			})
		}
	}
}

func (a *App) recordActivityProviderEvent(event events.Event) {
	if a.storage == nil || a.contribRegistry == nil {
		return
	}
	for _, provider := range a.contribRegistry.ActivityProviders() {
		if !hasString(provider.Item.Events, event.Name) {
			continue
		}
		if _, err := a.requirePluginAccess(provider.PluginID, "storage.namespace"); err != nil {
			continue
		}
		if err := a.appendActivityEvent(provider.PluginID, activityFromEvent(event)); err != nil {
			log.Printf("[api] activity provider %s failed to record %s: %v", provider.PluginID, event.Name, err)
		}
	}
}

func (a *App) appendActivityEvent(pluginID string, activity map[string]interface{}) error {
	settings, err := a.storage.ReadPluginSettings(pluginID)
	if err != nil {
		return err
	}
	key := activityGlobalKey
	if workspace, _ := activity["workspaceRootPath"].(string); strings.TrimSpace(workspace) != "" {
		key = activityWorkspacePrefix + url.QueryEscape(strings.TrimSpace(workspace))
		key = strings.ReplaceAll(key, "+", "%20")
	}
	eventsList := []interface{}{activity}
	if existing, ok := settings[key].([]interface{}); ok {
		eventsList = append(eventsList, existing...)
	} else if existingMaps, ok := settings[key].([]map[string]interface{}); ok {
		for _, item := range existingMaps {
			eventsList = append(eventsList, item)
		}
	}
	if len(eventsList) > maxActivityEvents {
		eventsList = eventsList[:maxActivityEvents]
	}
	settings[key] = eventsList
	return a.storage.WritePluginSettings(pluginID, settings)
}

func activityFromEvent(event events.Event) map[string]interface{} {
	payload := eventPayloadMap(event.Payload)
	delete(payload, "fileDataBase64")
	delete(payload, "dataBase64")
	now := time.Now().UTC().Format(time.RFC3339Nano)
	occurredAt := firstPayloadText(payload, "occurredAt", "capturedAt")
	if occurredAt == "" {
		occurredAt = event.Timestamp
	}
	if occurredAt == "" {
		occurredAt = now
	}
	workspaceRoot := firstPayloadText(payload, "workspaceRootPath", "workspaceName", "workspaceNodeId")
	if workspaceRoot == "" {
		workspaceRoot = workspaceRootFromRelativePath(firstPayloadText(payload, "path"))
	}
	return map[string]interface{}{
		"activityId":        fmt.Sprintf("activity-%d", time.Now().UnixNano()),
		"type":              event.Name,
		"title":             activityTitle(event.Name, payload),
		"summary":           activitySummary(event.Name, payload),
		"occurredAt":        occurredAt,
		"receivedAt":        now,
		"sourcePluginId":    firstPayloadText(payload, "pluginId", "sourcePluginId"),
		"workspaceRootPath": workspaceRoot,
		"payload":           payload,
	}
}

func eventPayloadMap(payload interface{}) map[string]interface{} {
	switch value := payload.(type) {
	case map[string]interface{}:
		result := make(map[string]interface{}, len(value))
		for key, item := range value {
			result[key] = item
		}
		return result
	case map[string]string:
		result := make(map[string]interface{}, len(value))
		for key, item := range value {
			result[key] = item
		}
		return result
	default:
		return map[string]interface{}{}
	}
}

func firstPayloadText(payload map[string]interface{}, keys ...string) string {
	for _, key := range keys {
		value := strings.TrimSpace(fmt.Sprint(payload[key]))
		if value != "" && value != "<nil>" {
			return value
		}
	}
	return ""
}

func activityTitle(eventName string, payload map[string]interface{}) string {
	if title := firstPayloadText(payload, "title", "name", "path", "url", "captureId"); title != "" {
		return title
	}
	return eventName
}

func activitySummary(eventName string, payload map[string]interface{}) string {
	if summary := firstPayloadText(payload, "text", "summary", "description", "path", "url", "domain"); summary != "" {
		return summary
	}
	return eventName
}

// ─── Plugin Manager API ─────────────────────────────────────

// GetPlugins returns all discovered plugins.
func (a *App) GetPlugins() []plugin.Plugin {
	if a.debug {
		debug.Logf("[api] GetPlugins: returning %d plugins", len(a.plugins))
		for i, p := range a.plugins {
			debug.Logf("[api]   plugin[%d]: id=%s status=%s enabled=%v root=%s", i, p.Manifest.ID, p.Status, p.Enabled, p.RootPath)
		}
	}
	return a.plugins
}

// GetCapabilities returns all registered capabilities.
func (a *App) GetCapabilities() []capability.Entry {
	entries := a.capRegistry.List()
	if a.debug {
		debug.Logf("[api] GetCapabilities: returning %d entries", len(entries))
	}
	return entries
}

// GetPermissions returns all known permissions.
func (a *App) GetPermissions() []permissions.Entry {
	entries := a.permRegistry.List()
	if a.debug {
		debug.Logf("[api] GetPermissions: returning %d entries", len(entries))
	}
	return entries
}

// ─── Flat contribution types for frontend ─────────────────

// FlatSidebarItem is a flattened sidebar item for the frontend.
type FlatSidebarItem struct {
	PluginID string `json:"pluginId"`
	ID       string `json:"id"`
	Title    string `json:"title"`
	Icon     string `json:"icon,omitempty"`
	View     string `json:"view"`
	Position int    `json:"position,omitempty"`
}

// FlatView is a flattened view contribution for the frontend.
type FlatView struct {
	PluginID  string `json:"pluginId"`
	ID        string `json:"id"`
	Title     string `json:"title"`
	Icon      string `json:"icon,omitempty"`
	Component string `json:"component"`
}

// FlatSettingsPanel is a flattened settings panel for the frontend.
type FlatSettingsPanel struct {
	PluginID  string `json:"pluginId"`
	ID        string `json:"id"`
	Title     string `json:"title"`
	Icon      string `json:"icon,omitempty"`
	Component string `json:"component"`
}

// FlatCommand is a flattened command contribution for the frontend.
type FlatCommand struct {
	PluginID string `json:"pluginId"`
	ID       string `json:"id"`
	Title    string `json:"title"`
	Icon     string `json:"icon,omitempty"`
	Handler  string `json:"handler,omitempty"`
}

type FlatSearchProvider struct {
	PluginID string `json:"pluginId"`
	ID       string `json:"id"`
	Label    string `json:"label"`
	Handler  string `json:"handler"`
}

type FlatStatusBarItem struct {
	PluginID string `json:"pluginId"`
	ID       string `json:"id"`
	Label    string `json:"label"`
	Position string `json:"position,omitempty"`
	Handler  string `json:"handler,omitempty"`
}

type FlatOpenProviderSupport struct {
	Kind       string   `json:"kind"`
	Mime       []string `json:"mime,omitempty"`
	Extensions []string `json:"extensions,omitempty"`
	Contexts   []string `json:"contexts,omitempty"`
	Modes      []string `json:"modes,omitempty"`
}

type FlatOpenProvider struct {
	PluginID  string                    `json:"pluginId"`
	ID        string                    `json:"id"`
	Title     string                    `json:"title"`
	Priority  int                       `json:"priority,omitempty"`
	Component string                    `json:"component"`
	Supports  []FlatOpenProviderSupport `json:"supports"`
}

type FlatWorkspaceItem struct {
	PluginID  string `json:"pluginId"`
	ID        string `json:"id"`
	Title     string `json:"title"`
	Icon      string `json:"icon,omitempty"`
	Component string `json:"component"`
}

type FlatAction struct {
	PluginID   string `json:"pluginId"`
	ID         string `json:"id"`
	Label      string `json:"label"`
	Icon       string `json:"icon,omitempty"`
	Capability string `json:"capability,omitempty"`
	Handler    string `json:"handler,omitempty"`
}

type FlatContextMenuEntry struct {
	PluginID   string `json:"pluginId"`
	ID         string `json:"id"`
	Label      string `json:"label"`
	Context    string `json:"context"`
	Group      string `json:"group,omitempty"`
	Capability string `json:"capability,omitempty"`
	Handler    string `json:"handler,omitempty"`
}

// ContributionSummary aggregates all contribution types for the frontend.
type ContributionSummary struct {
	Views              []FlatView             `json:"views"`
	Commands           []FlatCommand          `json:"commands"`
	SearchProviders    []FlatSearchProvider   `json:"searchProviders"`
	SettingsPanels     []FlatSettingsPanel    `json:"settingsPanels"`
	SidebarItems       []FlatSidebarItem      `json:"sidebarItems"`
	StatusBarItems     []FlatStatusBarItem    `json:"statusBarItems"`
	OpenProviders      []FlatOpenProvider     `json:"openProviders"`
	WorkspaceItems     []FlatWorkspaceItem    `json:"workspaceItems"`
	FileActions        []FlatAction           `json:"fileActions"`
	NoteActions        []FlatAction           `json:"noteActions"`
	ContextMenuEntries []FlatContextMenuEntry `json:"contextMenuEntries"`
}

// buildContributionSummary creates a ContributionSummary from the registry.
func buildContributionSummary(r *contribution.Registry) ContributionSummary {
	if r == nil {
		return ContributionSummary{}
	}
	regViews := r.Views()
	regCmds := r.Commands()
	regSearchProviders := r.SearchProviders()
	regPanels := r.SettingsPanels()
	regSidebar := r.SidebarItems()
	regStatusBar := r.StatusBarItems()
	regOpenProviders := r.OpenProviders()
	regWorkspaceItems := r.WorkspaceItems()
	regFileActions := r.FileActions()
	regNoteActions := r.NoteActions()
	regContextMenus := r.ContextMenus()

	views := make([]FlatView, len(regViews))
	for i, v := range regViews {
		views[i] = FlatView{PluginID: v.PluginID, ID: v.Item.ID, Title: v.Item.Title, Icon: v.Item.Icon, Component: v.Item.Component}
	}
	cmds := make([]FlatCommand, len(regCmds))
	for i, v := range regCmds {
		cmds[i] = FlatCommand{PluginID: v.PluginID, ID: v.Item.ID, Title: v.Item.Title, Icon: v.Item.Icon, Handler: v.Item.Handler}
	}
	searchProviders := make([]FlatSearchProvider, len(regSearchProviders))
	for i, v := range regSearchProviders {
		searchProviders[i] = FlatSearchProvider{PluginID: v.PluginID, ID: v.Item.ID, Label: v.Item.Label, Handler: v.Item.Handler}
	}
	panels := make([]FlatSettingsPanel, len(regPanels))
	for i, v := range regPanels {
		panels[i] = FlatSettingsPanel{PluginID: v.PluginID, ID: v.Item.ID, Title: v.Item.Title, Icon: v.Item.Icon, Component: v.Item.Component}
	}
	sidebar := make([]FlatSidebarItem, len(regSidebar))
	for i, v := range regSidebar {
		sidebar[i] = FlatSidebarItem{PluginID: v.PluginID, ID: v.Item.ID, Title: v.Item.Title, Icon: v.Item.Icon, View: v.Item.View, Position: v.Item.Position}
	}
	statusBarItems := make([]FlatStatusBarItem, len(regStatusBar))
	for i, v := range regStatusBar {
		statusBarItems[i] = FlatStatusBarItem{PluginID: v.PluginID, ID: v.Item.ID, Label: v.Item.Label, Position: v.Item.Position, Handler: v.Item.Handler}
	}
	openProviders := make([]FlatOpenProvider, len(regOpenProviders))
	for i, v := range regOpenProviders {
		supports := make([]FlatOpenProviderSupport, len(v.Item.Supports))
		for j, s := range v.Item.Supports {
			supports[j] = FlatOpenProviderSupport{Kind: s.Kind, Mime: s.Mime, Extensions: s.Extensions, Contexts: s.Contexts, Modes: s.Modes}
		}
		openProviders[i] = FlatOpenProvider{
			PluginID:  v.PluginID,
			ID:        v.Item.ID,
			Title:     v.Item.Title,
			Priority:  v.Item.Priority,
			Component: v.Item.Component,
			Supports:  supports,
		}
	}
	workspaceItems := make([]FlatWorkspaceItem, len(regWorkspaceItems))
	for i, v := range regWorkspaceItems {
		workspaceItems[i] = FlatWorkspaceItem{PluginID: v.PluginID, ID: v.Item.ID, Title: v.Item.Title, Icon: v.Item.Icon, Component: v.Item.Component}
	}
	fileActions := make([]FlatAction, len(regFileActions))
	for i, v := range regFileActions {
		fileActions[i] = FlatAction{PluginID: v.PluginID, ID: v.Item.ID, Label: v.Item.Label, Icon: v.Item.Icon, Capability: v.Item.Capability, Handler: v.Item.Handler}
	}
	noteActions := make([]FlatAction, len(regNoteActions))
	for i, v := range regNoteActions {
		noteActions[i] = FlatAction{PluginID: v.PluginID, ID: v.Item.ID, Label: v.Item.Label, Icon: v.Item.Icon, Capability: v.Item.Capability, Handler: v.Item.Handler}
	}
	contextMenus := make([]FlatContextMenuEntry, len(regContextMenus))
	for i, v := range regContextMenus {
		contextMenus[i] = FlatContextMenuEntry{PluginID: v.PluginID, ID: v.Item.ID, Label: v.Item.Label, Context: v.Item.Context, Group: v.Item.Group, Capability: v.Item.Capability, Handler: v.Item.Handler}
	}
	return ContributionSummary{Views: views, Commands: cmds, SearchProviders: searchProviders, SettingsPanels: panels, SidebarItems: sidebar, StatusBarItems: statusBarItems, OpenProviders: openProviders, WorkspaceItems: workspaceItems, FileActions: fileActions, NoteActions: noteActions, ContextMenuEntries: contextMenus}
}

// GetContributions returns all registered contributions flattened for the frontend.
func (a *App) GetContributions() ContributionSummary {
	if a.contribRegistry == nil {
		if a.debug {
			debug.Logf("[api] GetContributions: contribRegistry is nil")
		}
		return ContributionSummary{}
	}
	summary := buildContributionSummary(a.contribRegistry)
	if a.debug {
		debug.Logf("[api] GetContributions: returning views=%d commands=%d searchProviders=%d sidebar=%d statusBar=%d settings=%d openProviders=%d fileActions=%d noteActions=%d contextMenuEntries=%d",
			len(summary.Views), len(summary.Commands), len(summary.SearchProviders), len(summary.SidebarItems), len(summary.StatusBarItems), len(summary.SettingsPanels), len(summary.OpenProviders), len(summary.FileActions), len(summary.NoteActions), len(summary.ContextMenuEntries))
	}
	return summary
}

// ReloadPlugins re-discovers plugins from disk and returns a summary.
func (a *App) ReloadPlugins() (int, string) {
	discoveryDirs := plugin.DefaultDiscoveryDirs()
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
		"verstak/core/files/v1",
		"verstak/core/workbench/v1",
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

	// Re-register workspace capability if workspace is initialized
	if a.workspace != nil && a.workspace.IsInitialized() {
		if err := a.capRegistry.Register("verstak-desktop", []string{"verstak/core/workspace/v1"}); err != nil {
			log.Printf("[api] ReloadPlugins: failed to re-register workspace capability: %v", err)
		}
	}

	plugins, errs := plugin.DiscoverPlugins(discoveryDirs)

	// Plugin lifecycle: register capabilities + contributions
	for i := range plugins {
		p := &plugins[i]

		// Skip disabled plugins
		if a.pluginState != nil && a.pluginState.IsDisabled(p.Manifest.ID) {
			log.Printf("[plugin] %s: disabled in vault plugin state — skipping", p.Manifest.ID)
			p.Status = plugin.StatusDisabled
			p.Enabled = false
			continue
		}

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

		// Register contributions (unregister first to prevent duplicates)
		if p.Manifest.Contributes != nil {
			a.contribRegistry.Unregister(p.Manifest.ID)
			a.contribRegistry.Register(p.Manifest.ID, p.Manifest.Contributes)
		}

		// Record as desired plugin in vault state (only if vault is open)
		if a.pluginState != nil && a.vault != nil && a.vault.GetVaultStatus() == vault.StatusOpen {
			source := p.Manifest.Source
			if source == "" {
				source = "unknown"
			}
			if err := a.pluginState.RecordDesiredPlugin(p.Manifest.ID, p.Manifest.Version, source); err != nil {
				log.Printf("[plugin] %s: failed to record desired: %v", p.Manifest.ID, err)
			}
		}
	}

	a.plugins = plugins
	a.ensureActivityProviderSubscriptions()

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

	if a.debug {
		debug.Logf("[api] GetVaultStatus: status=%s path=%s vaultId=%s", status, path, vaultID)
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
	if err := a.vault.CreateVault(path); err != nil {
		return err
	}
	a.rebindSyncService()
	a.secretsSession = nil
	a.startFileWatcherForOpenVault()
	return nil
}

// OpenVault opens an existing vault at the given path.
func (a *App) OpenVault(path string) error {
	if a.vault == nil {
		return fmt.Errorf("vault service not initialized")
	}
	if err := a.vault.OpenVault(path); err != nil {
		return err
	}
	a.rebindSyncService()
	a.secretsSession = nil
	a.startFileWatcherForOpenVault()
	return nil
}

// CloseVault closes the current vault.
func (a *App) CloseVault() error {
	if a.vault == nil {
		return fmt.Errorf("vault service not initialized")
	}
	if a.fileWatcher != nil {
		a.fileWatcher.Stop()
	}
	a.vault.CloseVault()
	a.syncSvc = nil
	a.secretsSession = nil
	return nil
}

// ─── Storage API ────────────────────────────────────────────

// ReadPluginSettings returns all settings for a plugin.
func (a *App) ReadPluginSettings(pluginID string) (map[string]interface{}, string) {
	if _, err := a.requirePluginAccess(pluginID, "storage.namespace"); err != nil {
		return make(map[string]interface{}), err.Error()
	}
	if a.storage == nil {
		return make(map[string]interface{}), "storage not initialized"
	}
	data, err := a.storage.ReadPluginSettings(pluginID)
	if err != nil {
		log.Printf("[api] ReadPluginSettings(%s): %v", pluginID, err)
		return make(map[string]interface{}), err.Error()
	}
	return data, ""
}

// WritePluginSettings writes all settings for a plugin.
func (a *App) WritePluginSettings(pluginID string, data map[string]interface{}) string {
	if _, err := a.requirePluginAccess(pluginID, "storage.namespace"); err != nil {
		return err.Error()
	}
	if a.storage == nil {
		return "storage not initialized"
	}
	if err := a.storage.WritePluginSettings(pluginID, data); err != nil {
		log.Printf("[api] WritePluginSettings(%s): %v", pluginID, err)
		return err.Error()
	}
	return ""
}

// ReadPluginSetting returns a single setting value.
func (a *App) ReadPluginSetting(pluginID, key string) interface{} {
	if _, err := a.requirePluginAccess(pluginID, "storage.namespace"); err != nil {
		log.Printf("[api] ReadPluginSetting(%s, %s): %v", pluginID, key, err)
		return nil
	}
	if a.storage == nil {
		return nil
	}
	val, err := a.storage.ReadPluginSetting(pluginID, key)
	if err != nil {
		log.Printf("[api] ReadPluginSetting(%s, %s): %v", pluginID, key, err)
		return nil
	}
	return val
}

// WritePluginSetting writes a single setting value.
func (a *App) WritePluginSetting(pluginID, key string, value interface{}) string {
	if _, err := a.requirePluginAccess(pluginID, "storage.namespace"); err != nil {
		return err.Error()
	}
	if a.storage == nil {
		return "storage not initialized"
	}
	if err := a.storage.WritePluginSetting(pluginID, key, value); err != nil {
		log.Printf("[api] WritePluginSetting(%s, %s): %v", pluginID, key, err)
		return err.Error()
	}
	return ""
}

// ReadPluginDataJSON reads a named JSON data file for a plugin.
func (a *App) ReadPluginDataJSON(pluginID, name string) map[string]interface{} {
	if _, err := a.requirePluginAccess(pluginID, "storage.namespace"); err != nil {
		log.Printf("[api] ReadPluginDataJSON(%s, %s): %v", pluginID, name, err)
		return make(map[string]interface{})
	}
	if a.storage == nil {
		return make(map[string]interface{})
	}
	data, err := a.storage.ReadPluginDataJSON(pluginID, name)
	if err != nil {
		log.Printf("[api] ReadPluginDataJSON(%s, %s): %v", pluginID, name, err)
		return make(map[string]interface{})
	}
	return data
}

// WritePluginDataJSON writes a named JSON data file for a plugin.
func (a *App) WritePluginDataJSON(pluginID, name string, data map[string]interface{}) string {
	if _, err := a.requirePluginAccess(pluginID, "storage.namespace"); err != nil {
		return err.Error()
	}
	if a.storage == nil {
		return "storage not initialized"
	}
	if err := a.storage.WritePluginDataJSON(pluginID, name, data); err != nil {
		log.Printf("[api] WritePluginDataJSON(%s, %s): %v", pluginID, name, err)
		return err.Error()
	}
	return ""
}

// ListVaultFiles lists a vault-relative directory for a plugin with files.read.
func (a *App) ListVaultFiles(pluginID, relativeDir string) ([]corefiles.FileEntry, string) {
	if _, err := a.requirePluginAccess(pluginID, "files.read"); err != nil {
		return nil, err.Error()
	}
	if a.files == nil {
		return nil, "files service not initialized"
	}
	entries, err := a.files.ListVaultFiles(relativeDir)
	if err != nil {
		return nil, err.Error()
	}
	return entries, ""
}

// GetVaultFileMetadata returns metadata for a vault-relative path for a plugin with files.read.
func (a *App) GetVaultFileMetadata(pluginID, relativePath string) (corefiles.FileMetadata, string) {
	if _, err := a.requirePluginAccess(pluginID, "files.read"); err != nil {
		return corefiles.FileMetadata{}, err.Error()
	}
	if a.files == nil {
		return corefiles.FileMetadata{}, "files service not initialized"
	}
	meta, err := a.files.GetVaultFileMetadata(relativePath)
	if err != nil {
		return corefiles.FileMetadata{}, err.Error()
	}
	return meta, ""
}

// ReadVaultTextFile reads a UTF-8 text file for a plugin with files.read.
func (a *App) ReadVaultTextFile(pluginID, relativePath string) (string, string) {
	if _, err := a.requirePluginAccess(pluginID, "files.read"); err != nil {
		return "", err.Error()
	}
	if a.files == nil {
		return "", "files service not initialized"
	}
	text, err := a.files.ReadVaultTextFile(relativePath)
	if err != nil {
		return "", err.Error()
	}
	return text, ""
}

// ReadVaultFileBytes reads a bounded regular file as base64 for a plugin with files.read.
func (a *App) ReadVaultFileBytes(pluginID, relativePath string) (corefiles.FileBytes, string) {
	if _, err := a.requirePluginAccess(pluginID, "files.read"); err != nil {
		return corefiles.FileBytes{}, err.Error()
	}
	if a.files == nil {
		return corefiles.FileBytes{}, "files service not initialized"
	}
	data, err := a.files.ReadVaultFileBytes(relativePath)
	if err != nil {
		return corefiles.FileBytes{}, err.Error()
	}
	return data, ""
}

// WriteVaultTextFile atomically writes a UTF-8 text file for a plugin with files.write.
func (a *App) WriteVaultTextFile(pluginID, relativePath string, content string, options corefiles.WriteOptions) string {
	if _, err := a.requirePluginAccess(pluginID, "files.write"); err != nil {
		return err.Error()
	}
	if a.files == nil {
		return "files service not initialized"
	}
	opType := syncsvc.OpUpdate
	if _, err := a.files.GetVaultFileMetadata(relativePath); err != nil {
		if isSyncNotFound(err) {
			opType = syncsvc.OpCreate
		} else {
			return err.Error()
		}
	}
	if err := a.files.WriteVaultTextFile(relativePath, content, options); err != nil {
		return err.Error()
	}
	if err := a.recordFileSyncOp(syncsvc.EntityFile, relativePath, opType, map[string]string{
		"path":    relativePath,
		"content": content,
	}); err != nil {
		return err.Error()
	}
	a.publishFileActivity("file.changed", pluginID, relativePath, map[string]interface{}{
		"operation": opType,
	})
	return ""
}

// WriteVaultFileBytes atomically writes a bounded base64 file for a plugin with files.write.
func (a *App) WriteVaultFileBytes(pluginID, relativePath string, dataBase64 string, options corefiles.WriteOptions) string {
	if _, err := a.requirePluginAccess(pluginID, "files.write"); err != nil {
		return err.Error()
	}
	if a.files == nil {
		return "files service not initialized"
	}
	opType := syncsvc.OpUpdate
	if _, err := a.files.GetVaultFileMetadata(relativePath); err != nil {
		if isSyncNotFound(err) {
			opType = syncsvc.OpCreate
		} else {
			return err.Error()
		}
	}
	if err := a.files.WriteVaultFileBytes(relativePath, dataBase64, options); err != nil {
		return err.Error()
	}
	if err := a.recordFileSyncOp(syncsvc.EntityFile, relativePath, opType, map[string]string{
		"path":       relativePath,
		"dataBase64": dataBase64,
	}); err != nil {
		return err.Error()
	}
	a.publishFileActivity("file.changed", pluginID, relativePath, map[string]interface{}{
		"operation": opType,
	})
	return ""
}

// CreateVaultFolder creates a vault-relative folder for a plugin with files.write.
func (a *App) CreateVaultFolder(pluginID, relativePath string) string {
	if _, err := a.requirePluginAccess(pluginID, "files.write"); err != nil {
		return err.Error()
	}
	if a.files == nil {
		return "files service not initialized"
	}
	if err := a.files.CreateVaultFolder(relativePath); err != nil {
		return err.Error()
	}
	if err := a.recordFileSyncOp(syncsvc.EntityFolder, relativePath, syncsvc.OpCreate, map[string]string{
		"path": relativePath,
	}); err != nil {
		return err.Error()
	}
	a.publishFileActivity("file.changed", pluginID, relativePath, map[string]interface{}{
		"operation": syncsvc.OpCreate,
		"type":      string(corefiles.FileTypeFolder),
	})
	return ""
}

// MoveVaultPath moves a vault-relative file or folder for a plugin with files.write.
func (a *App) MoveVaultPath(pluginID, fromRelativePath string, toRelativePath string, options corefiles.MoveOptions) string {
	if _, err := a.requirePluginAccess(pluginID, "files.write"); err != nil {
		return err.Error()
	}
	if a.files == nil {
		return "files service not initialized"
	}
	meta, err := a.files.GetVaultFileMetadata(fromRelativePath)
	if err != nil {
		return err.Error()
	}
	if err := a.files.MoveVaultPath(fromRelativePath, toRelativePath, options); err != nil {
		return err.Error()
	}
	if err := a.recordFileSyncOp(syncEntityTypeForFileType(meta.Type), fromRelativePath, syncsvc.OpMove, map[string]string{
		"fromPath": fromRelativePath,
		"toPath":   toRelativePath,
	}); err != nil {
		return err.Error()
	}
	a.publishFileActivity("file.changed", pluginID, toRelativePath, map[string]interface{}{
		"operation": syncsvc.OpMove,
		"fromPath":  fromRelativePath,
		"type":      string(meta.Type),
	})
	return ""
}

// TrashVaultPath moves a vault-relative file or folder to internal trash for a plugin with files.delete.
func (a *App) TrashVaultPath(pluginID, relativePath string) (corefiles.TrashResult, string) {
	if _, err := a.requirePluginAccess(pluginID, "files.delete"); err != nil {
		return corefiles.TrashResult{}, err.Error()
	}
	if a.files == nil {
		return corefiles.TrashResult{}, "files service not initialized"
	}
	meta, err := a.files.GetVaultFileMetadata(relativePath)
	if err != nil {
		return corefiles.TrashResult{}, err.Error()
	}
	result, err := a.files.TrashVaultPath(relativePath)
	if err != nil {
		return corefiles.TrashResult{}, err.Error()
	}
	if err := a.recordFileSyncOp(syncEntityTypeForFileType(meta.Type), relativePath, syncsvc.OpDelete, map[string]string{
		"path": relativePath,
	}); err != nil {
		return corefiles.TrashResult{}, err.Error()
	}
	a.publishFileActivity("file.changed", pluginID, relativePath, map[string]interface{}{
		"operation": syncsvc.OpDelete,
		"type":      string(meta.Type),
	})
	return result, ""
}

// ListVaultTrash returns trash metadata entries for a plugin with files.delete.
func (a *App) ListVaultTrash(pluginID string) ([]corefiles.TrashEntry, string) {
	if _, err := a.requirePluginAccess(pluginID, "files.delete"); err != nil {
		return nil, err.Error()
	}
	if a.files == nil {
		return nil, "files service not initialized"
	}
	entries, err := a.files.ListTrashEntries()
	if err != nil {
		return nil, err.Error()
	}
	return entries, ""
}

// RestoreVaultTrash restores a file or folder from internal trash for a plugin with files.delete and files.write.
func (a *App) RestoreVaultTrash(pluginID, trashID string, options corefiles.RestoreOptions) (string, string) {
	if _, err := a.requirePluginAccess(pluginID, "files.delete"); err != nil {
		return "", err.Error()
	}
	if _, err := a.requirePluginAccess(pluginID, "files.write"); err != nil {
		return "", err.Error()
	}
	if a.files == nil {
		return "", "files service not initialized"
	}
	entries, err := a.files.ListTrashEntries()
	if err != nil {
		return "", err.Error()
	}
	var entry corefiles.TrashEntry
	for _, candidate := range entries {
		if candidate.TrashID == trashID {
			entry = candidate
			break
		}
	}
	if entry.TrashID == "" {
		return "", "not-found: trash entry " + trashID
	}
	restoredPath, err := a.files.RestoreTrashEntry(trashID, options)
	if err != nil {
		return "", err.Error()
	}
	if err := a.recordFileSyncOp(syncEntityTypeForFileType(entry.OriginalType), restoredPath, syncsvc.OpCreate, map[string]string{
		"path": restoredPath,
	}); err != nil {
		return "", err.Error()
	}
	a.publishFileActivity("file.changed", pluginID, restoredPath, map[string]interface{}{
		"operation": syncsvc.OpCreate,
		"type":      string(entry.OriginalType),
		"restored":  true,
		"trashId":   trashID,
	})
	return restoredPath, ""
}

// OpenVaultPathExternal opens a vault-relative file or folder in the OS default app.
func (a *App) OpenVaultPathExternal(pluginID, relativePath string) string {
	if _, err := a.requirePluginAccess(pluginID, "files.openExternal"); err != nil {
		return err.Error()
	}
	if a.files == nil {
		return "files service not initialized"
	}
	target, err := a.files.ResolveExternalOpenTarget(relativePath)
	if err != nil {
		return err.Error()
	}
	if err := a.externalOpenService().OpenPath(target.AbsolutePath); err != nil {
		return err.Error()
	}
	return ""
}

// ShowVaultPathInFolder reveals a vault-relative file or folder in the OS file manager.
func (a *App) ShowVaultPathInFolder(pluginID, relativePath string) string {
	if _, err := a.requirePluginAccess(pluginID, "files.openExternal"); err != nil {
		return err.Error()
	}
	if a.files == nil {
		return "files service not initialized"
	}
	target, err := a.files.ResolveExternalOpenTarget(relativePath)
	if err != nil {
		return err.Error()
	}
	isDir := target.Metadata.Type == corefiles.FileTypeFolder
	if err := a.externalOpenService().ShowInFolder(target.AbsolutePath, isDir); err != nil {
		return err.Error()
	}
	return ""
}

func (a *App) externalOpenService() externalOpenService {
	if a.externalOpen != nil {
		return a.externalOpen
	}
	return externalopen.NewService()
}

func (a *App) recordFileSyncOp(entityType, entityID, opType string, payload interface{}) error {
	if a.syncSvc == nil {
		return nil
	}
	return a.syncSvc.RecordOp(entityType, entityID, opType, payload)
}

func (a *App) publishFileActivity(eventName, pluginID, relativePath string, extra map[string]interface{}) {
	if a.eventBus == nil {
		return
	}
	path := strings.TrimSpace(filepath.ToSlash(relativePath))
	payload := map[string]interface{}{
		"path":              path,
		"title":             path,
		"workspaceRootPath": workspaceRootFromRelativePath(path),
		"pluginId":          pluginID,
	}
	for key, value := range extra {
		payload[key] = value
	}
	a.eventBus.Publish(events.Event{
		Name:      eventName,
		Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
		Payload:   payload,
	})
}

func workspaceRootFromRelativePath(relativePath string) string {
	path := strings.Trim(strings.TrimSpace(filepath.ToSlash(relativePath)), "/")
	if path == "" {
		return ""
	}
	if idx := strings.Index(path, "/"); idx >= 0 {
		return path[:idx]
	}
	return path
}

func (a *App) publishWorkspaceLifecycleEvent(eventName string, payload map[string]interface{}) {
	if a.eventBus == nil {
		return
	}
	if payload == nil {
		payload = map[string]interface{}{}
	}
	workspaceRoot := strings.TrimSpace(fmt.Sprint(payload["workspaceRootPath"]))
	if workspaceRoot == "" || workspaceRoot == "<nil>" {
		workspaceRoot = strings.TrimSpace(fmt.Sprint(payload["workspaceName"]))
	}
	if workspaceRoot != "" && workspaceRoot != "<nil>" {
		payload["workspaceRootPath"] = workspaceRoot
		if _, ok := payload["workspaceName"]; !ok {
			payload["workspaceName"] = workspaceRoot
		}
		if _, ok := payload["title"]; !ok {
			payload["title"] = workspaceRoot
		}
	}
	a.eventBus.Publish(events.Event{
		Name:      eventName,
		Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
		Payload:   payload,
	})
}

func syncEntityTypeForFileType(fileType corefiles.FileType) string {
	if fileType == corefiles.FileTypeFolder {
		return syncsvc.EntityFolder
	}
	return syncsvc.EntityFile
}

func (a *App) activeOpenProviders() []contribution.ContributionOpenProvider {
	if a.contribRegistry == nil {
		return nil
	}
	providers := a.contribRegistry.OpenProviders()
	active := make([]contribution.ContributionOpenProvider, 0, len(providers))
	for _, provider := range providers {
		p, err := a.findPlugin(provider.PluginID)
		if err != nil {
			continue
		}
		if !p.Enabled || (p.Status != plugin.StatusLoaded && p.Status != plugin.StatusDegraded) {
			continue
		}
		active = append(active, provider)
	}
	return active
}

func decodeOpenResourceRequest(raw map[string]interface{}) (coreworkbench.OpenResourceRequest, error) {
	data, err := json.Marshal(raw)
	if err != nil {
		return coreworkbench.OpenResourceRequest{}, err
	}
	var request coreworkbench.OpenResourceRequest
	if err := json.Unmarshal(data, &request); err != nil {
		return coreworkbench.OpenResourceRequest{}, err
	}
	if request.Kind == "" {
		return request, fmt.Errorf("resource kind is empty")
	}
	if request.Path == "" {
		return request, fmt.Errorf("resource path is empty")
	}
	return request, nil
}

func (a *App) OpenWorkbenchResource(pluginID string, rawRequest map[string]interface{}) (coreworkbench.OpenResourceResult, string) {
	if _, err := a.requirePluginAccess(pluginID, "workbench.open"); err != nil {
		return coreworkbench.OpenResourceResult{}, err.Error()
	}
	request, err := decodeOpenResourceRequest(rawRequest)
	if err != nil {
		return coreworkbench.OpenResourceResult{}, err.Error()
	}
	if request.Context.SourcePluginID == "" {
		request.Context.SourcePluginID = pluginID
	}
	result, err := a.ensureWorkbench().OpenResource(request, a.activeOpenProviders())
	if err != nil {
		return coreworkbench.OpenResourceResult{}, err.Error()
	}
	return result, ""
}

func (a *App) EditWorkbenchResource(pluginID string, rawRequest map[string]interface{}) (coreworkbench.OpenResourceResult, string) {
	if rawRequest == nil {
		rawRequest = map[string]interface{}{}
	}
	rawRequest["mode"] = "edit"
	return a.OpenWorkbenchResource(pluginID, rawRequest)
}

// ─── Secrets API ────────────────────────────────────────────

func (a *App) requirePluginSecretsAccess(pluginID string, write bool) error {
	if _, err := a.requirePluginAccess(pluginID, "secrets.read"); err != nil {
		return err
	}
	if write {
		if _, err := a.requirePluginAccess(pluginID, "secrets.write"); err != nil {
			return err
		}
	}
	return nil
}

func (a *App) ensureSecretsSession() (*coresecrets.VaultSession, error) {
	if err := a.requireVault(); err != nil {
		return nil, err
	}
	if a.secretsSession == nil {
		a.secretsSession = coresecrets.NewVaultSession(filepath.Join(a.vaultPath(), ".verstak", "secrets"))
	}
	return a.secretsSession, nil
}

func (a *App) requireUnlockedSecretStore() (*coresecrets.Store, error) {
	session, err := a.ensureSecretsSession()
	if err != nil {
		return nil, err
	}
	return session.Store()
}

func (a *App) PluginSecretsStatus(pluginID string) (map[string]interface{}, string) {
	if err := a.requirePluginSecretsAccess(pluginID, false); err != nil {
		return nil, err.Error()
	}
	session, err := a.ensureSecretsSession()
	if err != nil {
		return nil, err.Error()
	}
	initialized, err := session.Initialized()
	if err != nil {
		return nil, err.Error()
	}
	return map[string]interface{}{
		"initialized": initialized,
		"unlocked":    session.Unlocked(),
	}, ""
}

func (a *App) PluginSecretsUnlock(pluginID, masterPassword string) string {
	if err := a.requirePluginSecretsAccess(pluginID, false); err != nil {
		return err.Error()
	}
	session, err := a.ensureSecretsSession()
	if err != nil {
		return err.Error()
	}
	if _, err := session.Unlock(masterPassword); err != nil {
		return err.Error()
	}
	return ""
}

func (a *App) PluginSecretsList(pluginID string) ([]map[string]interface{}, string) {
	if err := a.requirePluginSecretsAccess(pluginID, false); err != nil {
		return nil, err.Error()
	}
	store, err := a.requireUnlockedSecretStore()
	if err != nil {
		return nil, err.Error()
	}
	records, err := store.ListRecords()
	if err != nil {
		return nil, err.Error()
	}
	result := make([]map[string]interface{}, 0, len(records))
	for _, record := range records {
		result = append(result, secretRecordMap(record, false))
	}
	return result, ""
}

func (a *App) PluginSecretsRead(pluginID, secretID string) (map[string]interface{}, string) {
	if err := a.requirePluginSecretsAccess(pluginID, false); err != nil {
		return nil, err.Error()
	}
	store, err := a.requireUnlockedSecretStore()
	if err != nil {
		return nil, err.Error()
	}
	record, err := store.ReadRecord(secretID)
	if err != nil {
		return nil, err.Error()
	}
	return secretRecordMap(record, true), ""
}

func (a *App) PluginSecretsWrite(pluginID string, rawRecord map[string]interface{}) (map[string]interface{}, string) {
	if err := a.requirePluginSecretsAccess(pluginID, true); err != nil {
		return nil, err.Error()
	}
	store, err := a.requireUnlockedSecretStore()
	if err != nil {
		return nil, err.Error()
	}
	record, err := decodeSecretRecord(rawRecord)
	if err != nil {
		return nil, err.Error()
	}
	if err := store.WriteRecord(record); err != nil {
		return nil, err.Error()
	}
	written, err := store.ReadRecord(record.ID)
	if err != nil {
		return nil, err.Error()
	}
	return secretRecordMap(written, false), ""
}

func (a *App) PluginSecretsDelete(pluginID, secretID string) string {
	if err := a.requirePluginSecretsAccess(pluginID, true); err != nil {
		return err.Error()
	}
	store, err := a.requireUnlockedSecretStore()
	if err != nil {
		return err.Error()
	}
	if err := store.Delete(secretID); err != nil {
		return err.Error()
	}
	return ""
}

func (a *App) PluginSecretsCopyLink(pluginID, secretID string) (string, string) {
	if err := a.requirePluginSecretsAccess(pluginID, false); err != nil {
		return "", err.Error()
	}
	store, err := a.requireUnlockedSecretStore()
	if err != nil {
		return "", err.Error()
	}
	record, err := store.ReadRecord(secretID)
	if err != nil {
		return "", err.Error()
	}
	title := strings.TrimSpace(record.Title)
	if title == "" {
		title = record.ID
	}
	return fmt.Sprintf("[%s](verstak-secret://%s)", title, url.PathEscape(record.ID)), ""
}

func decodeSecretRecord(raw map[string]interface{}) (coresecrets.SecretRecord, error) {
	data, err := json.Marshal(raw)
	if err != nil {
		return coresecrets.SecretRecord{}, err
	}
	var record coresecrets.SecretRecord
	if err := json.Unmarshal(data, &record); err != nil {
		return coresecrets.SecretRecord{}, err
	}
	if strings.TrimSpace(record.ID) == "" {
		record.ID = generatedSecretID(record.Title)
	}
	return record, nil
}

func generatedSecretID(title string) string {
	base := strings.ToLower(strings.TrimSpace(title))
	base = strings.Map(func(r rune) rune {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == '.' || r == '_' || r == '-' {
			return r
		}
		if r == ' ' || r == '\t' || r == '\n' || r == '\r' {
			return '-'
		}
		return -1
	}, base)
	base = strings.Trim(base, ".-_")
	if base == "" {
		base = "secret"
	}
	return fmt.Sprintf("%s-%d", base, time.Now().UnixNano())
}

func secretRecordMap(record coresecrets.SecretRecord, includeValue bool) map[string]interface{} {
	result := map[string]interface{}{
		"id":        record.ID,
		"title":     record.Title,
		"scope":     map[string]interface{}{"kind": record.Scope.Kind, "workspaceRootPath": record.Scope.WorkspaceRootPath},
		"username":  record.Username,
		"updatedAt": record.UpdatedAt,
	}
	if includeValue {
		result["value"] = record.Value
	}
	return result
}

func (a *App) GetWorkbenchOpenedResources() []coreworkbench.OpenedResource {
	return a.ensureWorkbench().OpenedResources()
}

func (a *App) GetWorkbenchPreferences() coreworkbench.Preferences {
	return a.ensureWorkbench().Preferences()
}

func (a *App) UpdateWorkbenchPreferences(preferences coreworkbench.Preferences) string {
	a.ensureWorkbench().SetPreferences(preferences)
	if a.appSettings == nil {
		return ""
	}
	if err := a.appSettings.Update(&appsettings.Config{Workbench: appSettingsWorkbenchPrefs(preferences)}); err != nil {
		return err.Error()
	}
	return ""
}

// ListPluginCapabilities returns the current capability registry for an enabled plugin.
func (a *App) ListPluginCapabilities(pluginID string) ([]capability.Entry, string) {
	if _, err := a.requirePluginCapabilityAccess(pluginID, "verstak/core/capability-registry/v1"); err != nil {
		return nil, err.Error()
	}
	if a.capRegistry == nil {
		return nil, "capability registry not initialized"
	}
	return a.capRegistry.List(), ""
}

// GetPluginCapability returns a single capability lookup for an enabled plugin.
func (a *App) GetPluginCapability(pluginID, capabilityName string) (map[string]interface{}, string) {
	if _, err := a.requirePluginCapabilityAccess(pluginID, "verstak/core/capability-registry/v1"); err != nil {
		return map[string]interface{}{"available": false}, err.Error()
	}
	if a.capRegistry == nil {
		return map[string]interface{}{"available": false}, "capability registry not initialized"
	}
	entry := a.capRegistry.Get(capabilityName)
	if entry == nil {
		return map[string]interface{}{"available": false, "name": capabilityName}, ""
	}
	return map[string]interface{}{
		"available": true,
		"name":      entry.Name,
		"pluginId":  entry.PluginID,
		"status":    entry.Status,
	}, ""
}

// ExecutePluginCommand validates that a command is declared by the plugin.
// Actual handler execution is intentionally deferred until sidecar/RPC exists.
func (a *App) ExecutePluginCommand(pluginID, commandID string, args map[string]interface{}) (map[string]interface{}, string) {
	if _, err := a.requirePluginAccess(pluginID, "commands.register"); err != nil {
		return nil, err.Error()
	}
	if a.contribRegistry == nil {
		return nil, "contribution registry not initialized"
	}
	for _, command := range a.contribRegistry.Commands() {
		if command.PluginID == pluginID && command.Item.ID == commandID {
			return map[string]interface{}{
				"status":    "declared",
				"pluginId":  pluginID,
				"commandId": commandID,
				"handler":   command.Item.Handler,
				"args":      args,
			}, ""
		}
	}
	return nil, fmt.Sprintf("command %q is not declared by plugin %q", commandID, pluginID)
}

// PublishPluginEvent validates publish permission and emits to the in-process bus.
func (a *App) PublishPluginEvent(pluginID, eventName string, payload map[string]interface{}) string {
	if _, err := a.requirePluginAccess(pluginID, "events.publish"); err != nil {
		return err.Error()
	}
	if eventName == "" {
		return "event name is empty"
	}
	if payload == nil {
		payload = make(map[string]interface{})
	}
	payload["pluginId"] = pluginID
	if a.eventBus != nil {
		a.eventBus.Publish(events.Event{
			Name:      eventName,
			Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
			Payload:   payload,
		})
	}
	return ""
}

// SubscribePluginEvent validates subscribe permission and bridges backend events
// into the bundled frontend plugin host.
func (a *App) SubscribePluginEvent(pluginID, eventName string) string {
	if _, err := a.requirePluginAccess(pluginID, "events.subscribe"); err != nil {
		return err.Error()
	}
	if eventName == "" {
		return "event name is empty"
	}
	if a.eventBus != nil {
		a.eventBus.Subscribe(eventName, func(event events.Event) {
			emitFrontendEvent(a.ctx, pluginEventRuntimeName, map[string]interface{}{
				"name":      event.Name,
				"timestamp": event.Timestamp,
				"payload":   event.Payload,
			})
		})
	}
	return ""
}

// ─── App Settings API ──────────────────────────────────────

// GetAppSettings returns the current app settings.
func (a *App) GetAppSettings() map[string]interface{} {
	if a.appSettings == nil {
		return map[string]interface{}{"status": "not initialized"}
	}
	cfg := a.appSettings.Get()
	return map[string]interface{}{
		"schemaVersion":    cfg.SchemaVersion,
		"currentVaultPath": cfg.CurrentVaultPath,
		"recentVaults":     cfg.RecentVaults,
		"theme":            cfg.Theme,
		"devMode":          cfg.DevMode,
		"userPluginsDir":   cfg.UserPluginsDir,
		"lastOpenedAt":     cfg.LastOpenedAt,
	}
}

// UpdateAppSettings patches and saves app settings.
func (a *App) UpdateAppSettings(patch map[string]interface{}) string {
	if a.appSettings == nil {
		return "app settings not initialized"
	}

	cfg := &appsettings.Config{}
	if v, ok := patch["theme"].(string); ok && v != "" {
		cfg.Theme = v
	}
	if v, ok := patch["devMode"].(bool); ok {
		cfg.DevMode = v
	}
	if v, ok := patch["userPluginsDir"].(string); ok && v != "" {
		cfg.UserPluginsDir = v
	}

	if err := a.appSettings.Update(cfg); err != nil {
		return err.Error()
	}
	return ""
}

// SetCurrentVault sets the current vault path in app settings and re-opens the vault.
// Loads workspace and registers vault + workspace capabilities.
func (a *App) SetCurrentVault(path string) string {
	if a.appSettings == nil {
		return "app settings not initialized"
	}
	if a.vault == nil {
		return "vault service not initialized"
	}
	// Try to open the vault first
	if err := a.vault.OpenVault(path); err != nil {
		return fmt.Sprintf("failed to open vault: %v", err)
	}
	a.secretsSession = nil
	// Save the actual vault path (normalized by OpenVault, includes VerstakVault/)
	vaultPath := a.vault.GetVaultPath()
	a.rebindSyncService()
	if err := a.appSettings.SetCurrentVault(vaultPath); err != nil {
		return fmt.Sprintf("failed to save app settings: %v", err)
	}
	// Load plugin state for the vault
	if a.pluginState != nil {
		if err := a.pluginState.Load(); err != nil {
			log.Printf("[api] SetCurrentVault: warning loading plugin state: %v", err)
		}
	}
	// Load workspace for the vault. This also handles first-run startup,
	// where no workspace manager exists until a vault is selected.
	a.workspace = workspace.NewManager(vaultPath)
	if err := a.workspace.Load(); err != nil {
		log.Printf("[api] SetCurrentVault: warning loading workspace: %v", err)
	}
	// Register vault capability
	if err := a.capRegistry.Register("verstak-desktop", []string{"verstak/core/vault/v1"}); err != nil {
		log.Printf("[api] SetCurrentVault: failed to register vault capability: %v", err)
	}
	// Register workspace capability
	if a.workspace != nil && a.workspace.IsInitialized() {
		if err := a.capRegistry.Register("verstak-desktop", []string{"verstak/core/workspace/v1"}); err != nil {
			log.Printf("[api] SetCurrentVault: failed to register workspace capability: %v", err)
		}
	}
	a.startFileWatcherForOpenVault()
	return ""
}

func (a *App) rebindSyncService() {
	if a == nil || a.vault == nil || a.vault.GetVaultStatus() != vault.StatusOpen {
		if a != nil {
			a.syncSvc = nil
		}
		return
	}
	a.syncSvc = syncsvc.NewService(a.vault.GetVaultPath(), "")
}

func (a *App) startFileWatcherForOpenVault() {
	if a == nil || a.vault == nil || a.eventBus == nil {
		return
	}
	if a.vault.GetVaultStatus() != vault.StatusOpen {
		return
	}
	if a.fileWatcher == nil {
		a.fileWatcher = filewatcher.NewService(a.eventBus, 0)
	}
	if err := a.fileWatcher.Start(a.vault.GetVaultPath()); err != nil {
		log.Printf("[api] file watcher start failed: %v", err)
	}
}

// ─── Workspace API ─────────────────────────────────────────

// ListWorkspaces returns top-level physical workspace folders.
func (a *App) ListWorkspaces() ([]workspace.Workspace, string) {
	if a.workspace == nil {
		return nil, "workspace not initialized"
	}
	workspaces, err := a.workspace.ListWorkspaces()
	if err != nil {
		return nil, err.Error()
	}
	return workspaces, ""
}

// CreateWorkspace creates a top-level physical workspace folder.
func (a *App) CreateWorkspace(name, templateID string) (workspace.Workspace, string) {
	if a.workspace == nil {
		return workspace.Workspace{}, "workspace not initialized"
	}
	ws, err := a.workspace.CreateWorkspace(name, templateID)
	if err != nil {
		return workspace.Workspace{}, err.Error()
	}
	a.publishWorkspaceLifecycleEvent(workspaceCreatedEventName, map[string]interface{}{
		"operation":         "create",
		"workspaceRootPath": ws.RootPath,
		"workspaceName":     ws.Name,
		"templateId":        templateID,
	})
	return ws, ""
}

// RenameWorkspace physically renames a top-level workspace folder.
func (a *App) RenameWorkspace(oldName, newName string) string {
	if a.workspace == nil {
		return "workspace not initialized"
	}
	if err := a.workspace.RenameWorkspace(oldName, newName); err != nil {
		return err.Error()
	}
	a.publishWorkspaceLifecycleEvent(workspaceRenamedEventName, map[string]interface{}{
		"operation":                 "rename",
		"workspaceRootPath":         newName,
		"workspaceName":             newName,
		"previousWorkspaceRootPath": oldName,
		"previousWorkspaceName":     oldName,
	})
	return ""
}

// TrashWorkspace moves a top-level workspace folder to internal trash.
func (a *App) TrashWorkspace(name string) (workspace.TrashResult, string) {
	if a.workspace == nil {
		return workspace.TrashResult{}, "workspace not initialized"
	}
	result, err := a.workspace.TrashWorkspace(name)
	if err != nil {
		return workspace.TrashResult{}, err.Error()
	}
	a.publishWorkspaceLifecycleEvent(workspaceTrashedEventName, map[string]interface{}{
		"operation":         "trash",
		"workspaceRootPath": name,
		"workspaceName":     name,
		"trashId":           result.TrashID,
		"trashPath":         result.TrashPath,
		"deletedAt":         result.DeletedAt,
	})
	return result, ""
}

// GetWorkspaceMetadata returns metadata or a generic fallback for a workspace.
func (a *App) GetWorkspaceMetadata(name string) (workspace.Metadata, string) {
	if a.workspace == nil {
		return workspace.Metadata{}, "workspace not initialized"
	}
	meta, err := a.workspace.GetWorkspaceMetadata(name)
	if err != nil {
		return workspace.Metadata{}, err.Error()
	}
	return meta, ""
}

// UpdateWorkspaceMetadata merges metadata for an existing workspace.
func (a *App) UpdateWorkspaceMetadata(name string, patch workspace.MetadataPatch) (workspace.Metadata, string) {
	if a.workspace == nil {
		return workspace.Metadata{}, "workspace not initialized"
	}
	meta, err := a.workspace.UpdateWorkspaceMetadata(name, patch)
	if err != nil {
		return workspace.Metadata{}, err.Error()
	}
	return meta, ""
}

// GetCurrentWorkspace returns the currently selected top-level workspace.
func (a *App) GetCurrentWorkspace() map[string]interface{} {
	if a.workspace == nil {
		return map[string]interface{}{"status": "not initialized"}
	}
	node, err := a.workspace.GetCurrentNode()
	if err != nil {
		return map[string]interface{}{"error": err.Error()}
	}
	return map[string]interface{}{
		"name":     node.Name,
		"rootPath": node.RootPath,
	}
}

// SetCurrentWorkspace stores the selected top-level workspace name as UI state.
func (a *App) SetCurrentWorkspace(name string) string {
	if a.workspace == nil {
		return "workspace not initialized"
	}
	if err := a.workspace.SetCurrentNode(name); err != nil {
		return err.Error()
	}
	a.publishWorkspaceLifecycleEvent(workspaceSelectedEventName, map[string]interface{}{
		"operation":         "select",
		"workspaceRootPath": name,
		"workspaceName":     name,
	})
	return ""
}

// Deprecated: compatibility wrapper over the flat top-level folder workspace
// model. Prefer ListWorkspaces.
func (a *App) GetWorkspaceTree() map[string]interface{} {
	if a.workspace == nil || !a.workspace.IsInitialized() {
		return map[string]interface{}{"status": "not initialized"}
	}
	tree := a.workspace.GetTree()
	return map[string]interface{}{
		"schemaVersion": tree.SchemaVersion,
		"nodes":         tree.Nodes,
		"currentNodeId": tree.CurrentNodeID,
		"updatedAt":     tree.UpdatedAt,
	}
}

// Deprecated: compatibility wrapper over the flat top-level folder workspace
// model. Prefer CreateWorkspace.
func (a *App) CreateWorkspaceNode(parentID, nodeType, title string) map[string]interface{} {
	if a.workspace == nil {
		return map[string]interface{}{"error": "workspace not initialized"}
	}
	node, err := a.workspace.CreateNode(parentID, workspace.NodeType(nodeType), title)
	if err != nil {
		return map[string]interface{}{"error": err.Error()}
	}
	return map[string]interface{}{
		"id":        node.ID,
		"parentId":  node.ParentID,
		"type":      string(node.Type),
		"title":     node.Title,
		"name":      node.Name,
		"rootPath":  node.RootPath,
		"status":    string(node.Status),
		"order":     node.Order,
		"createdAt": node.CreatedAt,
		"updatedAt": node.UpdatedAt,
	}
}

// Deprecated: compatibility wrapper over the flat top-level folder workspace
// model. Prefer RenameWorkspace.
func (a *App) RenameWorkspaceNode(id, title string) string {
	if a.workspace == nil {
		return "workspace not initialized"
	}
	if err := a.workspace.RenameNode(id, title); err != nil {
		return err.Error()
	}
	return ""
}

// Deprecated: compatibility wrapper retained only to reject old nested tree
// moves. The corrected workspace model is top-level folders only.
func (a *App) MoveWorkspaceNode(id, newParentID string) string {
	if a.workspace == nil {
		return "workspace not initialized"
	}
	if err := a.workspace.MoveNode(id, newParentID); err != nil {
		return err.Error()
	}
	return ""
}

// Deprecated: compatibility wrapper over the flat top-level folder workspace
// model. Prefer TrashWorkspace.
func (a *App) ArchiveWorkspaceNode(id string) string {
	if a.workspace == nil {
		return "workspace not initialized"
	}
	if err := a.workspace.ArchiveNode(id); err != nil {
		return err.Error()
	}
	return ""
}

// Deprecated: compatibility wrapper over the flat top-level folder workspace
// model. Prefer GetCurrentWorkspace.
func (a *App) GetCurrentWorkspaceNode() map[string]interface{} {
	if a.workspace == nil {
		return map[string]interface{}{"status": "not initialized"}
	}
	node, err := a.workspace.GetCurrentNode()
	if err != nil {
		return map[string]interface{}{"error": err.Error()}
	}
	return map[string]interface{}{
		"id":       node.ID,
		"type":     string(node.Type),
		"title":    node.Title,
		"name":     node.Name,
		"rootPath": node.RootPath,
		"status":   string(node.Status),
	}
}

// Deprecated: compatibility wrapper over the flat top-level folder workspace
// model. Prefer SetCurrentWorkspace.
func (a *App) SetCurrentWorkspaceNode(id string) string {
	if a.workspace == nil {
		return "workspace not initialized"
	}
	if err := a.workspace.SetCurrentNode(id); err != nil {
		return err.Error()
	}
	return ""
}

// ─── Vault Plugin State API ────────────────────────────────

// GetVaultPluginState returns the current vault plugin state.
func (a *App) GetVaultPluginState() map[string]interface{} {
	if a.pluginState == nil {
		return map[string]interface{}{"status": "not initialized"}
	}
	state := a.pluginState.Get()
	return map[string]interface{}{
		"schemaVersion":   state.SchemaVersion,
		"enabledPlugins":  state.EnabledPlugins,
		"disabledPlugins": state.DisabledPlugins,
		"desiredPlugins":  state.DesiredPlugins,
		"updatedAt":       state.UpdatedAt,
	}
}

// EnablePlugin enables a plugin in the vault.
func (a *App) EnablePlugin(pluginID string) string {
	if a.pluginState == nil {
		return "plugin state not initialized"
	}
	if err := a.pluginState.EnablePlugin(pluginID); err != nil {
		return err.Error()
	}
	return ""
}

// DisablePlugin disables a plugin in the vault.
func (a *App) DisablePlugin(pluginID string) string {
	if a.pluginState == nil {
		return "plugin state not initialized"
	}
	if err := a.pluginState.DisablePlugin(pluginID); err != nil {
		return err.Error()
	}
	return ""
}

// RecordDesiredPlugin records a plugin as desired for this vault.
func (a *App) RecordDesiredPlugin(pluginID, version, source string) string {
	if a.pluginState == nil {
		return "plugin state not initialized"
	}
	if err := a.pluginState.RecordDesiredPlugin(pluginID, version, source); err != nil {
		return err.Error()
	}
	return ""
}

// WriteFrontendLog writes a frontend debug message to the backend debug log.
func (a *App) WriteFrontendLog(component, message string) {
	if a.debug {
		debug.Logf("[frontend][%s] %s", component, message)
	}
}

// ─── Dialog API ─────────────────────────────────────────────

// SelectDirectory opens a native directory picker dialog.
// Returns the selected path or empty string if cancelled.
func (a *App) SelectDirectory() string {
	home, _ := os.UserHomeDir()

	selected, err := runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title:            "Select Vault Directory",
		DefaultDirectory: home,
	})
	if err != nil {
		log.Printf("[api] SelectDirectory: %v", err)
		return ""
	}
	return selected
}

// SelectVaultForOpen opens a directory picker for opening an existing vault.
func (a *App) SelectVaultForOpen() string {
	home, _ := os.UserHomeDir()

	selected, err := runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title:            "Open Existing Vault",
		DefaultDirectory: home,
	})
	if err != nil {
		log.Printf("[api] SelectVaultForOpen: %v", err)
		return ""
	}
	return selected
}

// ─── Plugin Frontend Asset API ───────────────────────────

// GetPluginFrontendInfo returns frontend metadata for a plugin.
// Returns empty map if plugin has no frontend bundle or is not found.
func (a *App) GetPluginFrontendInfo(pluginID string) map[string]interface{} {
	for _, p := range a.plugins {
		if p.Manifest.ID != pluginID {
			continue
		}
		if p.Manifest.Frontend == nil {
			return map[string]interface{}{"status": "no-frontend"}
		}
		return map[string]interface{}{
			"pluginId": p.Manifest.ID,
			"name":     p.Manifest.Name,
			"icon":     p.Manifest.Icon,
			"version":  p.Manifest.Version,
			"entry":    p.Manifest.Frontend.Entry,
			"style":    p.Manifest.Frontend.Style,
			"rootPath": p.RootPath,
		}
	}
	return map[string]interface{}{"status": "not-found"}
}

// GetPluginAssetContent reads a frontend asset file from a plugin directory.
// Security: validates that the assetPath is relative and does not escape the plugin root.
func (a *App) GetPluginAssetContent(pluginID, assetPath string) (string, string) {
	// Validate asset path — reject absolute paths and path traversal
	if strings.HasPrefix(assetPath, "/") || strings.HasPrefix(assetPath, "\\") {
		return "", "absolute paths not allowed"
	}
	if strings.Contains(assetPath, "..") {
		return "", "path traversal not allowed"
	}

	// Find the plugin
	var pluginRoot string
	found := false
	for _, p := range a.plugins {
		if p.Manifest.ID == pluginID && p.Manifest.Frontend != nil {
			pluginRoot = p.RootPath
			found = true
			break
		}
	}
	if !found {
		return "", "plugin not found or has no frontend"
	}

	// Resolve path relative to plugin root
	fullPath := filepath.Join(pluginRoot, assetPath)
	// Verify we haven't escaped plugin root
	absRoot, _ := filepath.Abs(pluginRoot)
	absPath, _ := filepath.Abs(fullPath)
	if !strings.HasPrefix(absPath, absRoot+string(filepath.Separator)) && absPath != absRoot {
		return "", "path escapes plugin root"
	}

	data, err := os.ReadFile(absPath)
	if err != nil {
		return "", fmt.Sprintf("failed to read asset: %v", err)
	}
	return string(data), ""
}

// ─── Browser Receiver API ──────────────────────────────────

func (a *App) requirePluginBrowserReceiverAccess(pluginID string) error {
	_, err := a.requirePluginAccess(pluginID, "browser.receiver.manage")
	return err
}

func (a *App) browserReceiverPairing() (map[string]string, error) {
	if a.browserReceiver == nil {
		return nil, fmt.Errorf("browser receiver is unavailable")
	}
	if a.appSettings == nil {
		return nil, fmt.Errorf("app settings not initialized")
	}
	token := strings.TrimSpace(a.appSettings.Get().BrowserReceiver.Token)
	if token == "" {
		return nil, fmt.Errorf("browser receiver token is unavailable")
	}
	return map[string]string{
		"receiverUrl":   browserreceiver.DefaultCaptureURL,
		"receiverToken": token,
	}, nil
}

// PluginBrowserReceiverPairing returns the local receiver settings for authorized plugins.
func (a *App) PluginBrowserReceiverPairing(pluginID string) (map[string]string, string) {
	if err := a.requirePluginBrowserReceiverAccess(pluginID); err != nil {
		return nil, err.Error()
	}
	pairing, err := a.browserReceiverPairing()
	if err != nil {
		return nil, err.Error()
	}
	return pairing, ""
}

// PluginRotateBrowserReceiverToken invalidates previous extension pairings.
func (a *App) PluginRotateBrowserReceiverToken(pluginID string) (map[string]string, string) {
	if err := a.requirePluginBrowserReceiverAccess(pluginID); err != nil {
		return nil, err.Error()
	}
	if _, err := a.browserReceiverPairing(); err != nil {
		return nil, err.Error()
	}
	token, err := a.appSettings.RotateBrowserReceiverToken()
	if err != nil {
		return nil, err.Error()
	}
	a.browserReceiver.SetReceiverToken(token)
	pairing, err := a.browserReceiverPairing()
	if err != nil {
		return nil, err.Error()
	}
	return pairing, ""
}

// ─── Sync API ──────────────────────────────────────────────

func (a *App) requirePluginSyncAccess(pluginID string, remote bool) error {
	if _, err := a.requirePluginAccess(pluginID, "sync.participate"); err != nil {
		return err
	}
	if remote {
		if _, err := a.requirePluginAccess(pluginID, "network.remote"); err != nil {
			return err
		}
	}
	return nil
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

// SyncStatusDTO holds sync status information for the frontend.
type SyncStatusDTO struct {
	Configured   bool   `json:"configured"`
	ServerURL    string `json:"serverUrl"`
	DeviceID     string `json:"deviceId"`
	DeviceName   string `json:"deviceName"`
	Connected    bool   `json:"connected"`
	Revoked      bool   `json:"revoked"`
	TokenStored  bool   `json:"tokenStored"`
	UnpushedOps  int    `json:"unpushedOps"`
	LastSyncAt   string `json:"lastSyncAt"`
	SyncInterval int    `json:"syncInterval"`
	LastError    string `json:"lastError"`
	StatusLabel  string `json:"statusLabel"`
}

func (a *App) syncStatus() (*SyncStatusDTO, error) {
	if a.vault == nil || a.vault.GetVaultStatus() != vault.StatusOpen {
		return &SyncStatusDTO{}, nil
	}

	vaultPath := a.vaultPath()
	if a.syncSvc == nil {
		return &SyncStatusDTO{}, nil
	}

	serverURL, apiKey, _, lastSyncAt, err := a.syncSvc.GetState()
	if err != nil {
		return &SyncStatusDTO{}, nil
	}

	cfg := a.appSettings.Get()
	deviceToken := syncsvc.LoadDeviceToken(vaultPath)

	dto := &SyncStatusDTO{
		Configured:   serverURL != "" && (apiKey != "" || deviceToken != ""),
		ServerURL:    serverURL,
		LastSyncAt:   lastSyncAt,
		UnpushedOps:  0,
		TokenStored:  deviceToken != "",
		SyncInterval: cfg.Sync.SyncInterval,
		LastError:    cfg.Sync.LastError,
	}

	if deviceID := a.syncSvc.GetDeviceID(); deviceID != "" {
		dto.DeviceID = deviceID
	} else if cfg.Sync.DeviceID != "" {
		dto.DeviceID = cfg.Sync.DeviceID
	}

	unpushed, _ := a.syncSvc.GetUnpushedOps()
	dto.UnpushedOps = len(unpushed)

	if deviceToken != "" {
		client := newSyncClient(serverURL, "", "", vaultPath)
		client.DeviceToken = deviceToken
		if dto.DeviceID != "" {
			client.DeviceID = dto.DeviceID
		}
		if info, err := client.GetMe(); err == nil {
			if info.DeviceID != "" {
				_ = a.syncSvc.SetDeviceID(info.DeviceID)
			}
			dto.DeviceName = info.DeviceName
			dto.DeviceID = info.DeviceID
			dto.Connected = true
			if info.RevokedAt != "" {
				dto.Revoked = true
				dto.Connected = false
			}
		}
	}

	switch {
	case dto.Revoked:
		dto.StatusLabel = "revoked"
	case dto.LastError != "":
		dto.StatusLabel = "error"
	case dto.Connected:
		dto.StatusLabel = "connected"
	case dto.Configured:
		dto.StatusLabel = "disconnected"
	case dto.ServerURL != "":
		dto.StatusLabel = "disconnected"
	default:
		dto.StatusLabel = "disabled"
	}

	if cfg.Sync.LastSyncAt != lastSyncAt || cfg.Sync.LastStatus != dto.StatusLabel {
		cfg.Sync.LastSyncAt = lastSyncAt
		cfg.Sync.LastStatus = dto.StatusLabel
		_ = a.appSettings.UpdateSync(cfg.Sync)
	}

	return dto, nil
}

// PluginSyncStatus returns sync status for plugins with sync permission.
func (a *App) PluginSyncStatus(pluginID string) (*SyncStatusDTO, string) {
	if err := a.requirePluginSyncAccess(pluginID, false); err != nil {
		return nil, err.Error()
	}
	dto, err := a.syncStatus()
	if err != nil {
		return nil, err.Error()
	}
	return dto, ""
}

func (a *App) syncConfigure(serverURL, username, password string) error {
	if err := a.requireVault(); err != nil {
		return err
	}
	meta := a.vault.GetVaultMeta()
	if meta == nil || strings.TrimSpace(meta.VaultID) == "" {
		return fmt.Errorf("vault ID is unavailable")
	}
	vaultPath := a.vaultPath()
	hostname, _ := os.Hostname()
	if hostname == "" {
		hostname = "unknown"
	}
	client := newSyncClient(serverURL, "", "", vaultPath)
	deviceID, deviceToken, err := client.PairDevice(serverURL, username, password, hostname, "verstak-desktop/v2", meta.VaultID)
	if err != nil {
		return fmt.Errorf("pair: %w", err)
	}
	if err := syncsvc.SaveDeviceToken(vaultPath, deviceToken); err != nil {
		return fmt.Errorf("save token: %w", err)
	}
	a.syncSvc = syncsvc.NewService(vaultPath, deviceID)
	if err := a.syncSvc.SetState(serverURL, ""); err != nil {
		return err
	}

	cfg := a.appSettings.Get()
	cfg.Sync.Enabled = true
	cfg.Sync.ServerURL = serverURL
	cfg.Sync.DeviceID = deviceID
	cfg.Sync.DeviceName = hostname
	cfg.Sync.LastStatus = "connected"
	cfg.Sync.LastError = ""
	_ = a.appSettings.UpdateSync(cfg.Sync)

	return nil
}

// PluginSyncConfigure pairs the current vault with a sync server for a plugin.
func (a *App) PluginSyncConfigure(pluginID, serverURL, username, password string) string {
	if err := a.requirePluginSyncAccess(pluginID, true); err != nil {
		return err.Error()
	}
	if err := a.syncConfigure(serverURL, username, password); err != nil {
		return err.Error()
	}
	return ""
}

func (a *App) syncDisconnect() error {
	if err := a.requireVault(); err != nil {
		return err
	}
	vaultPath := a.vaultPath()
	deviceToken := syncsvc.LoadDeviceToken(vaultPath)
	cfg := a.appSettings.Get()
	serverURL := cfg.Sync.ServerURL
	if a.syncSvc != nil {
		if currentServerURL, _, _, _, err := a.syncSvc.GetState(); err == nil && currentServerURL != "" {
			serverURL = currentServerURL
		}
	}

	if deviceToken != "" && serverURL != "" {
		client := newSyncClient(serverURL, "", "", vaultPath)
		client.DeviceToken = deviceToken
		_ = client.RevokeCurrent()
	}
	_ = syncsvc.RemoveDeviceToken(vaultPath)

	cfg.Sync.Enabled = false
	cfg.Sync.ServerURL = ""
	cfg.Sync.DeviceID = ""
	cfg.Sync.DeviceName = ""
	cfg.Sync.LastStatus = "disabled"
	cfg.Sync.LastError = ""
	if err := a.appSettings.UpdateSync(cfg.Sync); err != nil {
		return err
	}
	if a.syncSvc == nil {
		return nil
	}
	return a.syncSvc.SetState("", "")
}

// PluginSyncDisconnect disconnects sync for a plugin with sync permission.
func (a *App) PluginSyncDisconnect(pluginID string) string {
	if err := a.requirePluginSyncAccess(pluginID, false); err != nil {
		return err.Error()
	}
	if err := a.syncDisconnect(); err != nil {
		return err.Error()
	}
	return ""
}

func (a *App) syncTestConnection(serverURL, username, password string) error {
	vaultPath := a.vaultPath()
	if vaultPath == "" {
		vaultPath = "/tmp"
	}
	client := newSyncClient(serverURL, "", "", vaultPath)
	return client.TestAuth(serverURL, username, password)
}

// PluginSyncTestConnection tests sync server credentials for a plugin.
func (a *App) PluginSyncTestConnection(pluginID, serverURL, username, password string) string {
	if err := a.requirePluginSyncAccess(pluginID, true); err != nil {
		return err.Error()
	}
	if err := a.syncTestConnection(serverURL, username, password); err != nil {
		return err.Error()
	}
	return ""
}

func (a *App) syncSetInterval(minutes int) error {
	if err := a.requireVault(); err != nil {
		return err
	}
	cfg := a.appSettings.Get()
	cfg.Sync.SyncInterval = minutes
	if cfg.Sync.DeviceID == "" && a.syncSvc != nil {
		cfg.Sync.DeviceID = a.syncSvc.GetDeviceID()
	}
	return a.appSettings.UpdateSync(cfg.Sync)
}

// PluginSyncSetInterval sets the sync interval for a plugin with sync permission.
func (a *App) PluginSyncSetInterval(pluginID string, minutes int) string {
	if err := a.requirePluginSyncAccess(pluginID, false); err != nil {
		return err.Error()
	}
	if err := a.syncSetInterval(minutes); err != nil {
		return err.Error()
	}
	return ""
}

func (a *App) syncResetKey() error {
	if err := a.requireVault(); err != nil {
		return err
	}
	vaultPath := a.vaultPath()
	deviceToken := syncsvc.LoadDeviceToken(vaultPath)
	if err := syncsvc.RemoveDeviceToken(vaultPath); err != nil && !os.IsNotExist(err) {
		return err
	}

	cfg := a.appSettings.Get()
	cfg.Sync.Enabled = false
	cfg.Sync.DeviceID = ""
	cfg.Sync.DeviceName = ""
	cfg.Sync.LastStatus = "disconnected"
	cfg.Sync.LastError = ""
	if a.syncSvc != nil {
		serverURL, _, _, _, err := a.syncSvc.GetState()
		if err != nil {
			return err
		}
		if serverURL == "" && deviceToken != "" {
			serverURL = cfg.Sync.ServerURL
		}
		if err := a.syncSvc.SetState(serverURL, ""); err != nil {
			return err
		}
	}
	return a.appSettings.UpdateSync(cfg.Sync)
}

// PluginSyncResetKey clears the stored sync device token for a plugin with sync permission.
func (a *App) PluginSyncResetKey(pluginID string) string {
	if err := a.requirePluginSyncAccess(pluginID, false); err != nil {
		return err.Error()
	}
	if err := a.syncResetKey(); err != nil {
		return err.Error()
	}
	return ""
}

func (a *App) syncNow() (map[string]interface{}, error) {
	if err := a.requireVault(); err != nil {
		return nil, err
	}
	vaultPath := a.vaultPath()
	if a.syncSvc == nil {
		a.rebindSyncService()
	}
	if a.syncSvc == nil {
		return nil, fmt.Errorf("sync service not initialized")
	}

	serverURL, apiKey, lastPullSeq, _, err := a.syncSvc.GetState()
	deviceToken := syncsvc.LoadDeviceToken(vaultPath)
	if err != nil || serverURL == "" || (apiKey == "" && deviceToken == "") {
		return nil, fmt.Errorf("sync not configured")
	}

	deviceID := a.syncSvc.GetDeviceID()
	cfg := a.appSettings.Get()
	if deviceID == "" && deviceToken == "" && cfg.Sync.DeviceID != "" {
		deviceID = cfg.Sync.DeviceID
	}

	client := newSyncClient(serverURL, apiKey, deviceID, vaultPath)
	client.DeviceToken = deviceToken
	if deviceID == "" && deviceToken != "" {
		info, err := client.GetMe()
		if err != nil {
			return nil, fmt.Errorf("sync identity: %w", err)
		}
		if info.DeviceID == "" {
			return nil, fmt.Errorf("sync identity: server returned an empty device ID")
		}
		if err := a.syncSvc.SetDeviceID(info.DeviceID); err != nil {
			return nil, fmt.Errorf("save sync identity: %w", err)
		}
		deviceID = info.DeviceID
		client.DeviceID = deviceID
	}

	unpushed, err := a.syncSvc.GetUnpushedOps()
	if err != nil {
		return nil, fmt.Errorf("get ops: %w", err)
	}
	for i := range unpushed {
		unpushed[i].LastSeenServerSeq = lastPullSeq
	}
	pushResult := &syncsvc.PushResponse{}
	if len(unpushed) > 0 {
		pushResult, err = client.Push(unpushed)
		if err != nil {
			_ = a.updateSyncError(fmt.Sprintf("push: %v", err))
			return nil, fmt.Errorf("push: %w", err)
		}
		if err := a.syncSvc.MarkPushed(pushResult.Accepted); err != nil {
			return nil, fmt.Errorf("mark pushed: %w", err)
		}
	}

	pullResult, err := client.Pull(lastPullSeq)
	if err != nil {
		_ = a.updateSyncError(fmt.Sprintf("pull: %v", err))
		return nil, fmt.Errorf("pull: %w", err)
	}

	var applyErrors []string
	for _, op := range pullResult.Ops {
		if err := a.applyRemoteOp(op); err != nil {
			applyErrors = append(applyErrors, fmt.Sprintf("%s/%s: %v", op.EntityType, op.OpID, err))
		}
		_ = a.syncSvc.RecordRemoteOp(op)
	}
	if len(pullResult.Ops) > 0 {
		opIDs := make([]string, len(pullResult.Ops))
		for i, op := range pullResult.Ops {
			opIDs[i] = op.OpID
		}
		_ = a.syncSvc.MarkApplied(opIDs)
	}

	if len(pushResult.Conflicts) > 0 {
		log.Printf("[sync] %d conflict(s) detected on push", len(pushResult.Conflicts))
		for _, c := range pushResult.Conflicts {
			log.Printf("[sync] conflict: op=%v entity=%v/%v",
				c["op_id"], c["entity_type"], c["entity_id"])
		}
	}

	if pullResult.ServerSequence > lastPullSeq {
		_ = a.syncSvc.SetLastPullSeq(pullResult.ServerSequence)
	}
	_ = a.syncSvc.SetLastSyncAt(time.Now().UTC().Format(time.RFC3339))

	now := time.Now().UTC().Format(time.RFC3339)
	a.updateSyncSuccess(now)

	result := map[string]interface{}{
		"pushed":         len(pushResult.Accepted),
		"pulled":         len(pullResult.Ops),
		"serverSequence": pullResult.ServerSequence,
	}
	if len(applyErrors) > 0 {
		result["applyErrors"] = applyErrors
	}
	if len(pushResult.Conflicts) > 0 {
		result["conflicts"] = pushResult.Conflicts
	}
	return result, nil
}

// PluginSyncNow triggers sync for a plugin with sync permission.
func (a *App) PluginSyncNow(pluginID string) (map[string]interface{}, string) {
	if err := a.requirePluginSyncAccess(pluginID, true); err != nil {
		return nil, err.Error()
	}
	result, err := a.syncNow()
	if err != nil {
		return nil, err.Error()
	}
	return result, ""
}

func (a *App) updateSyncError(errMsg string) error {
	cfg := a.appSettings.Get()
	cfg.Sync.LastError = errMsg
	cfg.Sync.LastStatus = "error"
	return a.appSettings.UpdateSync(cfg.Sync)
}

func (a *App) updateSyncSuccess(lastSyncAt string) error {
	cfg := a.appSettings.Get()
	cfg.Sync.LastError = ""
	cfg.Sync.LastStatus = "connected"
	cfg.Sync.LastSyncAt = lastSyncAt
	return a.appSettings.UpdateSync(cfg.Sync)
}

func (a *App) applyRemoteOp(op syncsvc.Op) error {
	if a.debug {
		log.Printf("[sync] applyRemoteOp: type=%s entity=%s/%s", op.OpType, op.EntityType, op.EntityID)
	}
	if op.DeviceID != "" && op.DeviceID == a.localSyncDeviceID() {
		return nil
	}
	if a.files == nil {
		return fmt.Errorf("files service not initialized")
	}

	payload, err := parseSyncFilePayload(op.PayloadJSON)
	if err != nil {
		return err
	}
	switch op.EntityType {
	case syncsvc.EntityFile:
		return a.applyRemoteFileOp(op, payload)
	case syncsvc.EntityFolder:
		return a.applyRemoteFolderOp(op, payload)
	default:
		return fmt.Errorf("unsupported sync entity type: %s", op.EntityType)
	}
}

type syncFilePayload struct {
	Path       string  `json:"path"`
	Content    string  `json:"content"`
	DataBase64 *string `json:"dataBase64"`
	FromPath   string  `json:"fromPath"`
	ToPath     string  `json:"toPath"`
}

func parseSyncFilePayload(payloadJSON string) (syncFilePayload, error) {
	if payloadJSON == "" {
		return syncFilePayload{}, nil
	}
	var payload syncFilePayload
	if err := json.Unmarshal([]byte(payloadJSON), &payload); err != nil {
		return syncFilePayload{}, fmt.Errorf("invalid sync payload: %w", err)
	}
	return payload, nil
}

func (a *App) applyRemoteFileOp(op syncsvc.Op, payload syncFilePayload) error {
	switch op.OpType {
	case syncsvc.OpCreate:
		path := syncPayloadPath(op, payload)
		if path == "" {
			return fmt.Errorf("missing file path")
		}
		if payload.DataBase64 != nil {
			return a.files.WriteVaultFileBytes(path, *payload.DataBase64, corefiles.WriteOptions{CreateIfMissing: true})
		}
		return a.files.WriteVaultTextFile(path, payload.Content, corefiles.WriteOptions{CreateIfMissing: true})
	case syncsvc.OpUpdate:
		path := syncPayloadPath(op, payload)
		if path == "" {
			return fmt.Errorf("missing file path")
		}
		if payload.DataBase64 != nil {
			return a.files.WriteVaultFileBytes(path, *payload.DataBase64, corefiles.WriteOptions{CreateIfMissing: true, Overwrite: true})
		}
		return a.files.WriteVaultTextFile(path, payload.Content, corefiles.WriteOptions{CreateIfMissing: true, Overwrite: true})
	case syncsvc.OpDelete:
		path := syncPayloadPath(op, payload)
		if path == "" {
			return fmt.Errorf("missing file path")
		}
		_, err := a.files.TrashVaultPath(path)
		if isSyncNotFound(err) {
			return nil
		}
		return err
	case syncsvc.OpMove:
		fromPath := payload.FromPath
		if fromPath == "" {
			fromPath = op.EntityID
		}
		if fromPath == "" || payload.ToPath == "" {
			return fmt.Errorf("missing file move path")
		}
		err := a.files.MoveVaultPath(fromPath, payload.ToPath, corefiles.MoveOptions{})
		if isSyncNotFound(err) {
			return nil
		}
		return err
	default:
		return fmt.Errorf("unsupported file sync op type: %s", op.OpType)
	}
}

func (a *App) applyRemoteFolderOp(op syncsvc.Op, payload syncFilePayload) error {
	switch op.OpType {
	case syncsvc.OpCreate:
		path := syncPayloadPath(op, payload)
		if path == "" {
			return fmt.Errorf("missing folder path")
		}
		err := a.files.CreateVaultFolder(path)
		if isSyncConflict(err) {
			return nil
		}
		return err
	case syncsvc.OpDelete:
		path := syncPayloadPath(op, payload)
		if path == "" {
			return fmt.Errorf("missing folder path")
		}
		_, err := a.files.TrashVaultPath(path)
		if isSyncNotFound(err) {
			return nil
		}
		return err
	case syncsvc.OpMove:
		fromPath := payload.FromPath
		if fromPath == "" {
			fromPath = op.EntityID
		}
		if fromPath == "" || payload.ToPath == "" {
			return fmt.Errorf("missing folder move path")
		}
		err := a.files.MoveVaultPath(fromPath, payload.ToPath, corefiles.MoveOptions{})
		if isSyncNotFound(err) {
			return nil
		}
		return err
	default:
		return fmt.Errorf("unsupported folder sync op type: %s", op.OpType)
	}
}

func syncPayloadPath(op syncsvc.Op, payload syncFilePayload) string {
	if payload.Path != "" {
		return payload.Path
	}
	return op.EntityID
}

func (a *App) localSyncDeviceID() string {
	if a.syncSvc != nil {
		if deviceID := a.syncSvc.GetDeviceID(); deviceID != "" {
			return deviceID
		}
	}
	if a.vault != nil && a.vault.GetVaultStatus() == vault.StatusOpen && syncsvc.LoadDeviceToken(a.vaultPath()) != "" {
		return ""
	}
	if a.appSettings != nil {
		if deviceID := a.appSettings.Get().Sync.DeviceID; deviceID != "" {
			return deviceID
		}
	}
	return ""
}

func isSyncNotFound(err error) bool {
	return err != nil && strings.Contains(err.Error(), "not-found")
}

func isSyncConflict(err error) bool {
	return err != nil && strings.Contains(err.Error(), "conflict")
}
