package api

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
)

const dealCapabilityHostAPIVersion = "0.1.0"

var dealCapabilityOperations = map[string]map[string]struct{}{
	"verstak/notes/v2":    {"list": {}, "create": {}, "open": {}},
	"verstak/files/v2":    {"list": {}, "create": {}, "open": {}},
	"verstak/todo/v2":     {"list": {}, "create": {}, "setStatus": {}},
	"verstak/activity/v2": {"list": {}, "search": {}},
}

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

// ResolveDealCapabilityOperation validates the UUID-only Deal envelope before
// resolving a v2 provider command. It is intentionally separate from the v1
// resolver so the temporary provider migration does not widen the old contract.
func (a *App) ResolveDealCapabilityOperation(pluginID, capabilityName, operation string, request map[string]interface{}) (map[string]interface{}, string) {
	consumer, err := a.requirePluginAccess(pluginID, "")
	if err != nil {
		return nil, err.Error()
	}
	if consumer.Manifest.APIVersion != dealCapabilityHostAPIVersion {
		return nil, fmt.Sprintf("plugin %q targets host API %q; Deal capability host API is %q", pluginID, consumer.Manifest.APIVersion, dealCapabilityHostAPIVersion)
	}
	if errStr := validateDealCapabilityRequest(capabilityName, operation, request); errStr != "" {
		return nil, errStr
	}
	resolved, errStr := a.ResolvePluginCapabilityOperation(pluginID, capabilityName, operation)
	if errStr != "" {
		return nil, errStr
	}
	provider, err := a.requirePluginAccess(resolved["pluginId"].(string), "")
	if err != nil {
		return nil, err.Error()
	}
	if provider.Manifest.APIVersion != dealCapabilityHostAPIVersion {
		return nil, fmt.Sprintf("plugin %q targets host API %q; Deal capability host API is %q", provider.Manifest.ID, provider.Manifest.APIVersion, dealCapabilityHostAPIVersion)
	}
	return resolved, ""
}

func validateDealCapabilityRequest(capabilityName, operation string, request map[string]interface{}) string {
	operations, known := dealCapabilityOperations[capabilityName]
	if !known {
		return fmt.Sprintf("capability %q is not a Deal-scoped v2 provider", capabilityName)
	}
	if _, allowed := operations[operation]; !allowed {
		return fmt.Sprintf("capability %q does not define Deal operation %q", capabilityName, operation)
	}
	scope, ok := request["scope"].(map[string]interface{})
	if !ok {
		return "Deal operation requires scope"
	}
	if scope["kind"] != "deal" {
		return "DealScope.kind must be deal"
	}
	workspaceID, _ := scope["workspaceId"].(string)
	parsed, err := uuid.Parse(workspaceID)
	if err != nil || parsed == uuid.Nil {
		return "DealScope.workspaceId must be a UUID"
	}
	return ""
}
