// Package contribution provides a registry for plugin contribution points.
package contribution

import (
	"sort"
	"sync"

	"github.com/verstak/verstak-desktop/internal/core/plugin"
)

// Registry tracks all contributions registered by plugins.
type Registry struct {
	mu sync.RWMutex

	views             []ContributionView
	commands          []ContributionCommand
	settingsPanels    []ContributionSettingsPanel
	sidebarItems      []ContributionSidebarItem
	fileActions       []ContributionAction
	noteActions       []ContributionAction
	contextMenus      []ContributionContextMenuEntry
	searchProviders   []ContributionSearchProvider
	activityProviders []ContributionActivityProvider
	statusBarItems    []ContributionStatusBarItem
}

// ContributionPointType defines the type of contribution point.
type ContributionPointType string

const (
	PointViews           ContributionPointType = "views"
	PointCommands        ContributionPointType = "commands"
	PointSettingsPanels  ContributionPointType = "settingsPanels"
	PointSidebarItems    ContributionPointType = "sidebarItems"
	PointFileActions     ContributionPointType = "fileActions"
	PointNoteActions     ContributionPointType = "noteActions"
	PointContextMenus    ContributionPointType = "contextMenus"
	PointSearchProviders ContributionPointType = "searchProviders"
	PointActivity        ContributionPointType = "activityProviders"
	PointStatusBar       ContributionPointType = "statusBarItems"
)

// ListByPoint returns all contributions for a given point type.
func (r *Registry) ListByPoint(point ContributionPointType) []interface{} {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []interface{}
	switch point {
	case PointViews:
		for _, v := range r.views {
			result = append(result, v)
		}
	case PointCommands:
		for _, v := range r.commands {
			result = append(result, v)
		}
	case PointSettingsPanels:
		for _, v := range r.settingsPanels {
			result = append(result, v)
		}
	case PointSidebarItems:
		for _, v := range r.sidebarItems {
			result = append(result, v)
		}
	case PointFileActions:
		for _, v := range r.fileActions {
			result = append(result, v)
		}
	case PointNoteActions:
		for _, v := range r.noteActions {
			result = append(result, v)
		}
	case PointContextMenus:
		for _, v := range r.contextMenus {
			result = append(result, v)
		}
	case PointSearchProviders:
		for _, v := range r.searchProviders {
			result = append(result, v)
		}
	case PointActivity:
		for _, v := range r.activityProviders {
			result = append(result, v)
		}
	case PointStatusBar:
		for _, v := range r.statusBarItems {
			result = append(result, v)
		}
	}
	return result
}

type ContributionView struct {
	PluginID string                  `json:"pluginId"`
	Item     plugin.ContributionView `json:"item"`
}

type ContributionCommand struct {
	PluginID string                     `json:"pluginId"`
	Item     plugin.ContributionCommand `json:"item"`
}

type ContributionSettingsPanel struct {
	PluginID string                           `json:"pluginId"`
	Item     plugin.ContributionSettingsPanel `json:"item"`
}

type ContributionSidebarItem struct {
	PluginID string                         `json:"pluginId"`
	Item     plugin.ContributionSidebarItem `json:"item"`
}

type ContributionAction struct {
	PluginID string                    `json:"pluginId"`
	Item     plugin.ContributionAction `json:"item"`
}

type ContributionContextMenuEntry struct {
	PluginID string                              `json:"pluginId"`
	Item     plugin.ContributionContextMenuEntry `json:"item"`
}

type ContributionSearchProvider struct {
	PluginID string                            `json:"pluginId"`
	Item     plugin.ContributionSearchProvider `json:"item"`
}

type ContributionActivityProvider struct {
	PluginID string                              `json:"pluginId"`
	Item     plugin.ContributionActivityProvider `json:"item"`
}

type ContributionStatusBarItem struct {
	PluginID string                           `json:"pluginId"`
	Item     plugin.ContributionStatusBarItem `json:"item"`
}

// NewRegistry creates a new contribution registry.
func NewRegistry() *Registry {
	return &Registry{}
}

// Register adds all contributions from a plugin.
// If the plugin already has registered contributions they are replaced
// (supports reload without duplicates).
func (r *Registry) Register(pluginID string, c *plugin.Contributions) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Remove existing contributions for this plugin to prevent duplicates on reload
	r.views = removeViews(r.views, pluginID)
	r.commands = removeCommands(r.commands, pluginID)
	r.settingsPanels = removeSettingsPanels(r.settingsPanels, pluginID)
	r.sidebarItems = removeSidebarItems(r.sidebarItems, pluginID)
	r.fileActions = removeActions(r.fileActions, pluginID)
	r.noteActions = removeActions(r.noteActions, pluginID)
	r.contextMenus = removeContextMenus(r.contextMenus, pluginID)
	r.searchProviders = removeSearchProviders(r.searchProviders, pluginID)
	r.activityProviders = removeActivityProviders(r.activityProviders, pluginID)
	r.statusBarItems = removeStatusBarItems(r.statusBarItems, pluginID)

	for _, item := range c.Views {
		r.views = append(r.views, ContributionView{PluginID: pluginID, Item: item})
	}
	for _, item := range c.Commands {
		r.commands = append(r.commands, ContributionCommand{PluginID: pluginID, Item: item})
	}
	for _, item := range c.SettingsPanels {
		r.settingsPanels = append(r.settingsPanels, ContributionSettingsPanel{PluginID: pluginID, Item: item})
	}
	for _, item := range c.SidebarItems {
		r.sidebarItems = append(r.sidebarItems, ContributionSidebarItem{PluginID: pluginID, Item: item})
	}
	for _, item := range c.FileActions {
		r.fileActions = append(r.fileActions, ContributionAction{PluginID: pluginID, Item: item})
	}
	for _, item := range c.NoteActions {
		r.noteActions = append(r.noteActions, ContributionAction{PluginID: pluginID, Item: item})
	}
	for _, item := range c.ContextMenuEntries {
		r.contextMenus = append(r.contextMenus, ContributionContextMenuEntry{PluginID: pluginID, Item: item})
	}
	for _, item := range c.SearchProviders {
		r.searchProviders = append(r.searchProviders, ContributionSearchProvider{PluginID: pluginID, Item: item})
	}
	for _, item := range c.ActivityProviders {
		r.activityProviders = append(r.activityProviders, ContributionActivityProvider{PluginID: pluginID, Item: item})
	}
	for _, item := range c.StatusBarItems {
		r.statusBarItems = append(r.statusBarItems, ContributionStatusBarItem{PluginID: pluginID, Item: item})
	}
}

// Unregister removes all contributions from a plugin.
func (r *Registry) Unregister(pluginID string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.views = removeViews(r.views, pluginID)
	r.commands = removeCommands(r.commands, pluginID)
	r.settingsPanels = removeSettingsPanels(r.settingsPanels, pluginID)
	r.sidebarItems = removeSidebarItems(r.sidebarItems, pluginID)
	r.fileActions = removeActions(r.fileActions, pluginID)
	r.noteActions = removeActions(r.noteActions, pluginID)
	r.contextMenus = removeContextMenus(r.contextMenus, pluginID)
	r.searchProviders = removeSearchProviders(r.searchProviders, pluginID)
	r.activityProviders = removeActivityProviders(r.activityProviders, pluginID)
	r.statusBarItems = removeStatusBarItems(r.statusBarItems, pluginID)
}

// Getters — sorted for deterministic display.

func (r *Registry) Views() []ContributionView {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]ContributionView, len(r.views))
	copy(result, r.views)
	sort.Slice(result, func(i, j int) bool { return result[i].Item.ID < result[j].Item.ID })
	return result
}

func (r *Registry) Commands() []ContributionCommand {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]ContributionCommand, len(r.commands))
	copy(result, r.commands)
	sort.Slice(result, func(i, j int) bool { return result[i].Item.ID < result[j].Item.ID })
	return result
}

func (r *Registry) SettingsPanels() []ContributionSettingsPanel {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]ContributionSettingsPanel, len(r.settingsPanels))
	copy(result, r.settingsPanels)
	sort.Slice(result, func(i, j int) bool { return result[i].Item.ID < result[j].Item.ID })
	return result
}

func (r *Registry) SidebarItems() []ContributionSidebarItem {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]ContributionSidebarItem, len(r.sidebarItems))
	copy(result, r.sidebarItems)
	sort.Slice(result, func(i, j int) bool { return result[i].Item.ID < result[j].Item.ID })
	return result
}

func (r *Registry) FileActions() []ContributionAction {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]ContributionAction, len(r.fileActions))
	copy(result, r.fileActions)
	sort.Slice(result, func(i, j int) bool { return result[i].Item.ID < result[j].Item.ID })
	return result
}

func (r *Registry) NoteActions() []ContributionAction {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]ContributionAction, len(r.noteActions))
	copy(result, r.noteActions)
	sort.Slice(result, func(i, j int) bool { return result[i].Item.ID < result[j].Item.ID })
	return result
}

func (r *Registry) SearchProviders() []ContributionSearchProvider {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]ContributionSearchProvider, len(r.searchProviders))
	copy(result, r.searchProviders)
	sort.Slice(result, func(i, j int) bool { return result[i].Item.ID < result[j].Item.ID })
	return result
}

// ─── Remove helpers ─────────────────────────────────────────

func removeViews(items []ContributionView, pluginID string) []ContributionView {
	var result []ContributionView
	for _, item := range items {
		if item.PluginID != pluginID {
			result = append(result, item)
		}
	}
	return result
}

func removeCommands(items []ContributionCommand, pluginID string) []ContributionCommand {
	var result []ContributionCommand
	for _, item := range items {
		if item.PluginID != pluginID {
			result = append(result, item)
		}
	}
	return result
}

func removeSettingsPanels(items []ContributionSettingsPanel, pluginID string) []ContributionSettingsPanel {
	var result []ContributionSettingsPanel
	for _, item := range items {
		if item.PluginID != pluginID {
			result = append(result, item)
		}
	}
	return result
}

func removeSidebarItems(items []ContributionSidebarItem, pluginID string) []ContributionSidebarItem {
	var result []ContributionSidebarItem
	for _, item := range items {
		if item.PluginID != pluginID {
			result = append(result, item)
		}
	}
	return result
}

func removeActions(items []ContributionAction, pluginID string) []ContributionAction {
	var result []ContributionAction
	for _, item := range items {
		if item.PluginID != pluginID {
			result = append(result, item)
		}
	}
	return result
}

func removeContextMenus(items []ContributionContextMenuEntry, pluginID string) []ContributionContextMenuEntry {
	var result []ContributionContextMenuEntry
	for _, item := range items {
		if item.PluginID != pluginID {
			result = append(result, item)
		}
	}
	return result
}

func removeSearchProviders(items []ContributionSearchProvider, pluginID string) []ContributionSearchProvider {
	var result []ContributionSearchProvider
	for _, item := range items {
		if item.PluginID != pluginID {
			result = append(result, item)
		}
	}
	return result
}

func removeActivityProviders(items []ContributionActivityProvider, pluginID string) []ContributionActivityProvider {
	var result []ContributionActivityProvider
	for _, item := range items {
		if item.PluginID != pluginID {
			result = append(result, item)
		}
	}
	return result
}

func removeStatusBarItems(items []ContributionStatusBarItem, pluginID string) []ContributionStatusBarItem {
	var result []ContributionStatusBarItem
	for _, item := range items {
		if item.PluginID != pluginID {
			result = append(result, item)
		}
	}
	return result
}
