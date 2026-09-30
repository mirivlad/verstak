package api

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/verstak/verstak-desktop/internal/core/capability"
	"github.com/verstak/verstak-desktop/internal/core/contribution"
	"github.com/verstak/verstak-desktop/internal/core/events"
	"github.com/verstak/verstak-desktop/internal/core/notifications"
	"github.com/verstak/verstak-desktop/internal/core/permissions"
	"github.com/verstak/verstak-desktop/internal/core/plugin"
	"github.com/verstak/verstak-desktop/internal/core/vault"
	"github.com/verstak/verstak-desktop/internal/shell/debug"
)

const pluginEventRuntimeName = "verstak:plugin-event"

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

// FlatWorklogProvider is a plugin that can propose journal entries.
type FlatWorklogProvider struct {
	PluginID string `json:"pluginId"`
	ID       string `json:"id"`
	Label    string `json:"label"`
	Handler  string `json:"handler"`
}

// FlatOverviewProvider is a normalized Overview provider contribution.
type FlatOverviewProvider struct {
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
	Order     int    `json:"order,omitempty"`
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
	WorklogProviders   []FlatWorklogProvider  `json:"worklogProviders"`
	OverviewProviders  []FlatOverviewProvider `json:"overviewProviders"`
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
	regWorklogProviders := r.WorklogProviders()
	regOverviewProviders := r.OverviewProviders()
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
	worklogProviders := make([]FlatWorklogProvider, len(regWorklogProviders))
	overviewProviders := make([]FlatOverviewProvider, len(regOverviewProviders))
	for i, v := range regWorklogProviders {
		worklogProviders[i] = FlatWorklogProvider{PluginID: v.PluginID, ID: v.Item.ID, Label: v.Item.Label, Handler: v.Item.Handler}
	}
	for i, v := range regOverviewProviders {
		overviewProviders[i] = FlatOverviewProvider{PluginID: v.PluginID, ID: v.Item.ID, Label: v.Item.Label, Handler: v.Item.Handler}
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
		workspaceItems[i] = FlatWorkspaceItem{PluginID: v.PluginID, ID: v.Item.ID, Title: v.Item.Title, Icon: v.Item.Icon, Order: v.Item.Order, Component: v.Item.Component}
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
	return ContributionSummary{Views: views, Commands: cmds, SearchProviders: searchProviders, WorklogProviders: worklogProviders, OverviewProviders: overviewProviders, SettingsPanels: panels, SidebarItems: sidebar, StatusBarItems: statusBarItems, OpenProviders: openProviders, WorkspaceItems: workspaceItems, FileActions: fileActions, NoteActions: noteActions, ContextMenuEntries: contextMenus}
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
	if service := a.currentImportService(); service != nil {
		service.CloseAll()
	}
	discoveryDirs := plugin.DefaultDiscoveryDirs()
	log.Printf("[api] ReloadPlugins: scanning dirs: %v", discoveryDirs)

	// Remove entries for plugins that may no longer be discovered.
	if a.contribRegistry != nil {
		for _, existing := range a.plugins {
			a.contribRegistry.Unregister(existing.Manifest.ID)
		}
	}

	// Unregister all non-core capabilities
	a.capRegistry.UnregisterAll()

	// Re-register the same core capabilities as initial startup. Keeping this
	// list in the capability package prevents a plugin reload from dropping a
	// required capability such as native notifications.
	if err := a.capRegistry.Register(capability.CorePluginID, capability.CorePlatformCapabilities()); err != nil {
		log.Printf("[api] ReloadPlugins: failed to re-register core capabilities: %v", err)
	}

	// Re-register vault capability if vault is open
	if a.vault != nil && a.vault.GetVaultStatus() == vault.StatusOpen {
		if err := a.capRegistry.Register(capability.CorePluginID, []string{"verstak/core/vault/v1"}); err != nil {
			log.Printf("[api] ReloadPlugins: failed to re-register vault capability: %v", err)
		}
	}

	// Re-register workspace capability if the canonical Deal tree is initialized.
	if a.treeV2 != nil {
		if err := a.capRegistry.Register(capability.CorePluginID, []string{"verstak/core/workspace/v1"}); err != nil {
			log.Printf("[api] ReloadPlugins: failed to re-register workspace capability: %v", err)
		}
	}

	plugins, errs := plugin.DiscoverPlugins(discoveryDirs)

	plugin.ResolveLifecycle(plugins, a.capRegistry, func(pluginID string) bool {
		return a.pluginState != nil && a.pluginState.IsDisabled(pluginID)
	})

	// Register contributions for plugins with a resolved lifecycle.
	for i := range plugins {
		p := &plugins[i]

		if p.Status != plugin.StatusLoaded && p.Status != plugin.StatusDegraded {
			if p.Error != "" {
				log.Printf("[plugin] %s: status=%s: %s", p.Manifest.ID, p.Status, p.Error)
			}
			continue
		}

		// Register contributions after old discovery entries were removed.
		if p.Manifest.Contributes != nil {
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
	a.ensureBrowserInboxSubscriptions()

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
	before := a.snapshotPluginRecords(pluginID)
	if err := a.storage.WritePluginSettings(pluginID, data); err != nil {
		log.Printf("[api] WritePluginSettings(%s): %v", pluginID, err)
		return err.Error()
	}
	a.recordPluginRecordChanges(pluginID, before)
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
	before := a.snapshotPluginRecords(pluginID)
	if err := a.storage.WritePluginSetting(pluginID, key, value); err != nil {
		log.Printf("[api] WritePluginSetting(%s, %s): %v", pluginID, key, err)
		return err.Error()
	}
	a.recordPluginRecordChanges(pluginID, before)
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

// ReadPluginDataNDJSON reads append-only plugin data without exposing the
// underlying vault path to plugin frontends.
func (a *App) ReadPluginDataNDJSON(pluginID, name string) []map[string]interface{} {
	if _, err := a.requirePluginAccess(pluginID, "storage.namespace"); err != nil {
		log.Printf("[api] ReadPluginDataNDJSON(%s, %s): %v", pluginID, name, err)
		return []map[string]interface{}{}
	}
	if a.storage == nil {
		return []map[string]interface{}{}
	}
	data, err := a.storage.ReadPluginDataNDJSON(pluginID, name)
	if err != nil {
		log.Printf("[api] ReadPluginDataNDJSON(%s, %s): %v", pluginID, name, err)
		return []map[string]interface{}{}
	}
	return data
}

// WritePluginDataNDJSON replaces append-only data after an explicit user
// action, such as clearing activity history.
func (a *App) WritePluginDataNDJSON(pluginID, name string, data []map[string]interface{}) string {
	if _, err := a.requirePluginAccess(pluginID, "storage.namespace"); err != nil {
		return err.Error()
	}
	if a.storage == nil {
		return "storage not initialized"
	}
	before := a.snapshotPluginRecords(pluginID)
	if err := a.storage.WritePluginDataNDJSON(pluginID, name, data); err != nil {
		log.Printf("[api] WritePluginDataNDJSON(%s, %s): %v", pluginID, name, err)
		return err.Error()
	}
	a.recordPluginRecordChanges(pluginID, before)
	return ""
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

// ReplacePluginNotifications replaces one plugin's desired native notification
// schedules. Plugins must declare both the capability and permission.
func (a *App) ReplacePluginNotifications(pluginID string, requests []notifications.Request) string {
	if _, err := a.requirePluginAccess(pluginID, "notifications.schedule"); err != nil {
		return err.Error()
	}
	if _, err := a.requirePluginCapabilityAccess(pluginID, "verstak/core/notifications/v1"); err != nil {
		return err.Error()
	}
	if a.notifications == nil {
		return "notification scheduler not initialized"
	}
	if err := a.notifications.Replace(pluginID, requests); err != nil {
		log.Printf("[api] ReplacePluginNotifications(%s): %v", pluginID, err)
		return err.Error()
	}
	return ""
}

// ClearPluginNotifications removes every native notification schedule owned by
// one plugin.
func (a *App) ClearPluginNotifications(pluginID string) string {
	if _, err := a.requirePluginAccess(pluginID, "notifications.schedule"); err != nil {
		return err.Error()
	}
	if _, err := a.requirePluginCapabilityAccess(pluginID, "verstak/core/notifications/v1"); err != nil {
		return err.Error()
	}
	if a.notifications == nil {
		return "notification scheduler not initialized"
	}
	if err := a.notifications.Clear(pluginID); err != nil {
		log.Printf("[api] ClearPluginNotifications(%s): %v", pluginID, err)
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

// ExecutePluginCommand validates that a callable handler is declared by the
// plugin. Provider handlers are deliberately not command-palette entries: the
// shell invokes them for integrations such as search, Overview, and Journal.
// Actual handler execution is intentionally deferred until sidecar/RPC exists.
func (a *App) ExecutePluginCommand(pluginID, commandID string, args map[string]interface{}) (map[string]interface{}, string) {
	if _, err := a.requirePluginAccess(pluginID, "commands.register"); err != nil {
		return nil, err.Error()
	}
	if a.contribRegistry == nil {
		return nil, "contribution registry not initialized"
	}
	if handler, ok := a.pluginHandler(pluginID, commandID); ok {
		return map[string]interface{}{
			"status":    "declared",
			"pluginId":  pluginID,
			"commandId": commandID,
			"handler":   handler,
			"args":      args,
		}, ""
	}
	return nil, fmt.Sprintf("command %q is not declared by plugin %q", commandID, pluginID)
}

func (a *App) pluginHandler(pluginID, handlerID string) (string, bool) {
	for _, command := range a.contribRegistry.Commands() {
		if command.PluginID == pluginID && command.Item.ID == handlerID {
			return command.Item.Handler, true
		}
	}
	for _, provider := range a.contribRegistry.SearchProviders() {
		if provider.PluginID == pluginID && provider.Item.Handler == handlerID {
			return provider.Item.Handler, true
		}
	}
	for _, provider := range a.contribRegistry.WorklogProviders() {
		if provider.PluginID == pluginID && provider.Item.Handler == handlerID {
			return provider.Item.Handler, true
		}
	}
	for _, provider := range a.contribRegistry.OverviewProviders() {
		if provider.PluginID == pluginID && provider.Item.Handler == handlerID {
			return provider.Item.Handler, true
		}
	}
	// Capability operations may be intentionally headless. Their handler is
	// still registered through commands.register, but they must not acquire a
	// command-palette contribution merely to be callable by another plugin.
	if loaded, err := a.findPlugin(pluginID); err == nil {
		for _, operations := range loaded.Manifest.CapabilityOperations {
			for _, operationHandler := range operations {
				if operationHandler == handlerID {
					return operationHandler, true
				}
			}
		}
	}
	return "", false
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
	if service := a.currentImportService(); service != nil {
		service.ClosePlugin(pluginID)
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
			"pluginId":     p.Manifest.ID,
			"name":         p.Manifest.Name,
			"icon":         p.Manifest.Icon,
			"version":      p.Manifest.Version,
			"entry":        p.Manifest.Frontend.Entry,
			"style":        p.Manifest.Frontend.Style,
			"localization": p.Manifest.Localization,
			"rootPath":     p.RootPath,
		}
	}
	return map[string]interface{}{"status": "not-found"}
}

// GetPluginLocalization reads a locale catalog declared by the plugin manifest.
func (a *App) GetPluginLocalization(pluginID, locale string) (map[string]string, string) {
	var selected *plugin.Plugin
	for i := range a.plugins {
		if a.plugins[i].Manifest.ID == pluginID {
			selected = &a.plugins[i]
			break
		}
	}
	if selected == nil {
		return nil, "plugin not found"
	}
	localization := selected.Manifest.Localization
	if localization == nil {
		return nil, "plugin does not declare localization"
	}
	catalogPath, ok := localization.Locales[locale]
	if !ok {
		return nil, fmt.Sprintf("locale %q is not declared", locale)
	}
	if err := plugin.ValidateLocalizationPath(catalogPath); err != nil {
		return nil, err.Error()
	}

	absRoot, err := filepath.Abs(selected.RootPath)
	if err != nil {
		return nil, fmt.Sprintf("resolve plugin root: %v", err)
	}
	absPath, err := filepath.Abs(filepath.Join(absRoot, filepath.FromSlash(catalogPath)))
	if err != nil {
		return nil, fmt.Sprintf("resolve catalog path: %v", err)
	}
	if !pathInsideRoot(absRoot, absPath) {
		return nil, "catalog path escapes plugin root"
	}
	resolvedPath, err := filepath.EvalSymlinks(absPath)
	if err != nil {
		return nil, fmt.Sprintf("failed to resolve catalog: %v", err)
	}
	// Resolve the root the same way before comparing: a root reached through a
	// symlink or a Windows 8.3 short name otherwise never contains anything.
	resolvedRoot, err := filepath.EvalSymlinks(absRoot)
	if err != nil {
		return nil, fmt.Sprintf("resolve plugin root: %v", err)
	}
	if !pathInsideRoot(resolvedRoot, resolvedPath) {
		return nil, "catalog path escapes plugin root"
	}

	data, err := os.ReadFile(resolvedPath)
	if err != nil {
		return nil, fmt.Sprintf("failed to read catalog: %v", err)
	}
	catalog := map[string]string{}
	if err := json.Unmarshal(data, &catalog); err != nil {
		return nil, fmt.Sprintf("failed to parse catalog: %v", err)
	}
	return catalog, ""
}

func pathInsideRoot(root, candidate string) bool {
	relative, err := filepath.Rel(root, candidate)
	if err != nil {
		return false
	}
	return relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator))
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
