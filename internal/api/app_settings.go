package api

import (
	"fmt"

	"github.com/verstak/verstak-desktop/internal/core/appsettings"
	"github.com/verstak/verstak-desktop/internal/core/buildinfo"
)

// GetBuildInfo reports which build of Verstak is running.
//
// Surfaced in the status bar so a tester can tell at a glance whether the
// package they just installed is the one they are looking at.
func (a *App) GetBuildInfo() buildinfo.Info {
	return buildinfo.Get()
}

// GetAppSettings returns the current app settings.
func (a *App) GetAppSettings() map[string]interface{} {
	if a.appSettings == nil {
		return map[string]interface{}{"status": "not initialized"}
	}
	cfg := a.appSettings.Get()
	return map[string]interface{}{
		"schemaVersion":     cfg.SchemaVersion,
		"currentVaultPath":  cfg.CurrentVaultPath,
		"recentVaults":      cfg.RecentVaults,
		"theme":             cfg.Theme,
		"language":          cfg.Language,
		"devMode":           cfg.DevMode,
		"debug":             a.debug,
		"userPluginsDir":    cfg.UserPluginsDir,
		"sidebarWidth":      cfg.SidebarWidth,
		"expandedFolderIds": cfg.ExpandedFolderIDs,
		"settingsSection":   cfg.SettingsSection,
		"lastOpenedAt":      cfg.LastOpenedAt,
	}
}

// UpdateAppSettings patches and saves app settings.
func (a *App) UpdateAppSettings(patch map[string]interface{}) string {
	if a.appSettings == nil {
		return "app settings not initialized"
	}

	cfg := &appsettings.Config{}
	hasConfigPatch := false
	var sidebarWidth *int
	var expandedFolderIDs *[]string
	var settingsSection *string
	if value, exists := patch["sidebarWidth"]; exists {
		width, ok := appSettingInt(value)
		if !ok {
			return "sidebarWidth must be an integer"
		}
		if width < appsettings.MinSidebarWidth || width > appsettings.MaxSidebarWidth {
			return fmt.Sprintf("sidebarWidth must be between %d and %d", appsettings.MinSidebarWidth, appsettings.MaxSidebarWidth)
		}
		sidebarWidth = &width
	}
	if value, exists := patch["expandedFolderIds"]; exists {
		ids, ok := appSettingStringSlice(value)
		if !ok {
			return "expandedFolderIds must be an array of strings"
		}
		expandedFolderIDs = &ids
	}
	if value, exists := patch["settingsSection"]; exists {
		section, ok := value.(string)
		if !ok {
			return "settingsSection must be a string"
		}
		settingsSection = &section
	}
	if v, ok := patch["theme"].(string); ok && v != "" {
		cfg.Theme = v
		hasConfigPatch = true
	}
	if v, ok := patch["devMode"].(bool); ok {
		cfg.DevMode = v
		hasConfigPatch = true
	}
	if v, ok := patch["userPluginsDir"].(string); ok && v != "" {
		cfg.UserPluginsDir = v
		hasConfigPatch = true
	}

	if hasConfigPatch {
		if err := a.appSettings.Update(cfg); err != nil {
			return err.Error()
		}
	}
	if value, exists := patch["language"]; exists {
		language, ok := value.(string)
		if !ok {
			return "language must be a string"
		}
		if err := a.appSettings.UpdateLanguage(language); err != nil {
			return err.Error()
		}
	}
	if sidebarWidth != nil || expandedFolderIDs != nil || settingsSection != nil {
		if err := a.appSettings.UpdateUIState(sidebarWidth, expandedFolderIDs, settingsSection); err != nil {
			return err.Error()
		}
	}
	return ""
}

func appSettingInt(value interface{}) (int, bool) {
	switch typed := value.(type) {
	case int:
		return typed, true
	case float64:
		integer := int(typed)
		return integer, float64(integer) == typed
	default:
		return 0, false
	}
}

func appSettingStringSlice(value interface{}) ([]string, bool) {
	switch typed := value.(type) {
	case []string:
		return append([]string(nil), typed...), true
	case []interface{}:
		result := make([]string, 0, len(typed))
		for _, item := range typed {
			text, ok := item.(string)
			if !ok {
				return nil, false
			}
			result = append(result, text)
		}
		return result, true
	default:
		return nil, false
	}
}
