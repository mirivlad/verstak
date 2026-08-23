package api

import (
	"fmt"
	"strings"
)

// ResolvePluginCapabilityOperation resolves a declared capability operation to
// the provider command that implements it. Consumers can resolve only
// capabilities declared in requires or optionalRequires.
func (a *App) ResolvePluginCapabilityOperation(pluginID, capabilityName, operation string) (map[string]interface{}, string) {
	if _, err := a.requirePluginCapabilityAccess(pluginID, capabilityName); err != nil {
		return nil, err.Error()
	}
	if strings.TrimSpace(operation) == "" {
		return nil, "capability operation is empty"
	}
	if a.capRegistry == nil {
		return nil, "capability registry not initialized"
	}
	entry := a.capRegistry.Get(capabilityName)
	if entry == nil {
		return nil, fmt.Sprintf("capability %q is not available", capabilityName)
	}
	provider, err := a.requirePluginAccess(entry.PluginID, "")
	if err != nil {
		return nil, err.Error()
	}
	commandID := provider.Manifest.CapabilityOperations[capabilityName][operation]
	if commandID == "" {
		return nil, fmt.Sprintf("capability %q does not provide operation %q", capabilityName, operation)
	}
	return map[string]interface{}{
		"capability": capabilityName,
		"operation":  operation,
		"pluginId":   entry.PluginID,
		"commandId":  commandID,
	}, ""
}
