package api

import (
	"encoding/json"
	"fmt"

	"github.com/verstak/verstak-desktop/internal/core/appsettings"
	"github.com/verstak/verstak-desktop/internal/core/contribution"
	"github.com/verstak/verstak-desktop/internal/core/plugin"
	coreworkbench "github.com/verstak/verstak-desktop/internal/core/workbench"
)

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
