// Package api provides Wails-bound methods for the frontend.
package api

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"github.com/verstak/verstak-desktop/internal/core/appsettings"
	"github.com/verstak/verstak-desktop/internal/core/capability"
	"github.com/verstak/verstak-desktop/internal/core/contribution"
	"github.com/verstak/verstak-desktop/internal/core/events"
	"github.com/verstak/verstak-desktop/internal/core/permissions"
	"github.com/verstak/verstak-desktop/internal/core/plugin"
	"github.com/verstak/verstak-desktop/internal/core/pluginstate"
	"github.com/verstak/verstak-desktop/internal/core/storage"
	"github.com/verstak/verstak-desktop/internal/core/vault"
	"github.com/verstak/verstak-desktop/internal/core/workspace"
)

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
	appSettings     *appsettings.Manager
	pluginState     *pluginstate.Manager
	workspace       *workspace.Manager
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
	appSettingsMgr *appsettings.Manager,
	pluginStateMgr *pluginstate.Manager,
	workspaceMgr *workspace.Manager,
) *App {
	return &App{
		capRegistry:     capReg,
		contribRegistry: contribReg,
		permRegistry:    permReg,
		eventBus:        bus,
		plugins:         plugins,
		vault:           vaultService,
		storage:         storageService,
		appSettings:     appSettingsMgr,
		pluginState:     pluginStateMgr,
		workspace:       workspaceMgr,
	}
}

// Startup is called when the app starts. Sets the Wails context for dialogs.
func (a *App) Startup(ctx context.Context) {
	a.ctx = ctx
	log.Printf("[api] App.Startup: initialized with %d plugins", len(a.plugins))
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

// ContributionSummary aggregates all contribution types for the frontend.
type ContributionSummary struct {
	Views          []FlatView          `json:"views"`
	Commands       []FlatCommand       `json:"commands"`
	SettingsPanels []FlatSettingsPanel `json:"settingsPanels"`
	SidebarItems   []FlatSidebarItem   `json:"sidebarItems"`
}

// buildContributionSummary creates a ContributionSummary from the registry.
func buildContributionSummary(r *contribution.Registry) ContributionSummary {
	if r == nil {
		return ContributionSummary{}
	}
	regViews := r.Views()
	regCmds := r.Commands()
	regPanels := r.SettingsPanels()
	regSidebar := r.SidebarItems()

	views := make([]FlatView, len(regViews))
	for i, v := range regViews {
		views[i] = FlatView{PluginID: v.PluginID, ID: v.Item.ID, Title: v.Item.Title, Icon: v.Item.Icon, Component: v.Item.Component}
	}
	cmds := make([]FlatCommand, len(regCmds))
	for i, v := range regCmds {
		cmds[i] = FlatCommand{PluginID: v.PluginID, ID: v.Item.ID, Title: v.Item.Title, Icon: v.Item.Icon, Handler: v.Item.Handler}
	}
	panels := make([]FlatSettingsPanel, len(regPanels))
	for i, v := range regPanels {
		panels[i] = FlatSettingsPanel{PluginID: v.PluginID, ID: v.Item.ID, Title: v.Item.Title, Icon: v.Item.Icon, Component: v.Item.Component}
	}
	sidebar := make([]FlatSidebarItem, len(regSidebar))
	for i, v := range regSidebar {
		sidebar[i] = FlatSidebarItem{PluginID: v.PluginID, ID: v.Item.ID, Title: v.Item.Title, Icon: v.Item.Icon, View: v.Item.View, Position: v.Item.Position}
	}
	return ContributionSummary{Views: views, Commands: cmds, SettingsPanels: panels, SidebarItems: sidebar}
}

// GetContributions returns all registered contributions flattened for the frontend.
func (a *App) GetContributions() ContributionSummary {
	if a.contribRegistry == nil {
		return ContributionSummary{}
	}
	return buildContributionSummary(a.contribRegistry)
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

// ─── Storage API ────────────────────────────────────────────

// ReadPluginSettings returns all settings for a plugin.
func (a *App) ReadPluginSettings(pluginID string) map[string]interface{} {
	if a.storage == nil {
		return make(map[string]interface{})
	}
	data, err := a.storage.ReadPluginSettings(pluginID)
	if err != nil {
		log.Printf("[api] ReadPluginSettings(%s): %v", pluginID, err)
		return make(map[string]interface{})
	}
	return data
}

// WritePluginSettings writes all settings for a plugin.
func (a *App) WritePluginSettings(pluginID string, data map[string]interface{}) string {
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
	if a.storage == nil {
		return "storage not initialized"
	}
	if err := a.storage.WritePluginDataJSON(pluginID, name, data); err != nil {
		log.Printf("[api] WritePluginDataJSON(%s, %s): %v", pluginID, name, err)
		return err.Error()
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
	// Save the actual vault path (normalized by OpenVault, includes VerstakVault/)
	vaultPath := a.vault.GetVaultPath()
	if err := a.appSettings.SetCurrentVault(vaultPath); err != nil {
		return fmt.Sprintf("failed to save app settings: %v", err)
	}
	// Load plugin state for the vault
	if a.pluginState != nil {
		if err := a.pluginState.Load(); err != nil {
			log.Printf("[api] SetCurrentVault: warning loading plugin state: %v", err)
		}
	}
	// Load workspace for the vault
	if a.workspace != nil {
		// Replace workspace manager with one pointing to the new vault
		a.workspace = workspace.NewManager(vaultPath)
		if err := a.workspace.Load(); err != nil {
			log.Printf("[api] SetCurrentVault: warning loading workspace: %v", err)
		}
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
	return ""
}

// ─── Workspace API ─────────────────────────────────────────

// GetWorkspaceTree returns the full workspace tree.
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

// CreateWorkspaceNode creates a new workspace node.
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
		"status":    string(node.Status),
		"order":     node.Order,
		"createdAt": node.CreatedAt,
		"updatedAt": node.UpdatedAt,
	}
}

// RenameWorkspaceNode renames a workspace node.
func (a *App) RenameWorkspaceNode(id, title string) string {
	if a.workspace == nil {
		return "workspace not initialized"
	}
	if err := a.workspace.RenameNode(id, title); err != nil {
		return err.Error()
	}
	return ""
}

// MoveWorkspaceNode moves a node to a new parent.
func (a *App) MoveWorkspaceNode(id, newParentID string) string {
	if a.workspace == nil {
		return "workspace not initialized"
	}
	if err := a.workspace.MoveNode(id, newParentID); err != nil {
		return err.Error()
	}
	return ""
}

// ArchiveWorkspaceNode archives a workspace node.
func (a *App) ArchiveWorkspaceNode(id string) string {
	if a.workspace == nil {
		return "workspace not initialized"
	}
	if err := a.workspace.ArchiveNode(id); err != nil {
		return err.Error()
	}
	return ""
}

// GetCurrentWorkspaceNode returns the currently selected node.
func (a *App) GetCurrentWorkspaceNode() map[string]interface{} {
	if a.workspace == nil {
		return map[string]interface{}{"status": "not initialized"}
	}
	node, err := a.workspace.GetCurrentNode()
	if err != nil {
		return map[string]interface{}{"error": err.Error()}
	}
	return map[string]interface{}{
		"id":     node.ID,
		"type":   string(node.Type),
		"title":  node.Title,
		"status": string(node.Status),
	}
}

// SetCurrentWorkspaceNode sets the currently selected node.
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
