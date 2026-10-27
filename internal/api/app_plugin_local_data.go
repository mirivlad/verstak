package api

import "errors"

// Local JSON uses the same namespace access checks as vault data, but lives
// outside the vault. It grants no process permissions or host approvals.
func (a *App) ReadPluginLocalDataJSON(pluginID, name string) (map[string]interface{}, error) {
	if _, err := a.requirePluginAccess(pluginID, "storage.namespace"); err != nil {
		return nil, err
	}
	if a.storage == nil {
		return nil, errors.New("storage not initialized")
	}
	return a.storage.ReadPluginLocalJSON(pluginID, name)
}

func (a *App) WritePluginLocalDataJSON(pluginID, name string, data map[string]interface{}) string {
	if _, err := a.requirePluginAccess(pluginID, "storage.namespace"); err != nil {
		return err.Error()
	}
	if a.storage == nil {
		return "storage not initialized"
	}
	if err := a.storage.WritePluginLocalJSON(pluginID, name, data); err != nil {
		return err.Error()
	}
	return ""
}
