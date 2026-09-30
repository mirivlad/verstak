package api

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/verstak/verstak-desktop/internal/core/events"
	corefiles "github.com/verstak/verstak-desktop/internal/core/files"
	"github.com/verstak/verstak-desktop/internal/core/storage"
	"github.com/verstak/verstak-desktop/internal/core/vault"
)

const activityPluginID = "verstak.activity"
const activityRawDataName = "activity-events"
const activitySessionHandlingKey = "activity-session-handling-v2"
const activitySessionHandledEvent = "activity.session.handled"
const maxActivityRawEvents = 10000
const maxActivityRawBytes = 8 * 1024 * 1024
const activityRetention = 60 * 24 * time.Hour

func (a *App) activityAvailable() bool {
	if a == nil || a.storage == nil || a.vault == nil || a.vault.GetVaultStatus() != vault.StatusOpen {
		return false
	}
	_, err := a.requirePluginAccess(activityPluginID, "storage.namespace")
	return err == nil
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
	if !a.activityEvents[activitySessionHandledEvent] {
		a.activityEvents[activitySessionHandledEvent] = true
		a.eventBus.Subscribe(activitySessionHandledEvent, func(event events.Event) {
			if err := a.recordActivitySessionHandled(event); err != nil {
				log.Printf("[api] activity session handling update failed: %v", err)
			}
		})
	}
}

func (a *App) recordActivitySessionHandled(event events.Event) error {
	if !a.activityAvailable() {
		return fmt.Errorf("activity storage unavailable")
	}
	payload := eventPayloadMap(event.Payload)
	if firstPayloadText(payload, "pluginId") != "verstak.journal" {
		return fmt.Errorf("activity session handling source is not authorized")
	}
	sessionID := firstPayloadText(payload, "sessionId")
	handledThrough := firstPayloadText(payload, "handledThrough")
	status := firstPayloadText(payload, "status")
	if sessionID == "" || handledThrough == "" || (status != "accepted" && status != "dismissed") {
		return fmt.Errorf("activity session handling payload is invalid")
	}
	if _, err := time.Parse(time.RFC3339, handledThrough); err != nil {
		return fmt.Errorf("activity session handledThrough is invalid")
	}
	return a.storage.UpdatePluginSettings(activityPluginID, func(settings map[string]interface{}) error {
		handled, _ := settings[activitySessionHandlingKey].(map[string]interface{})
		if handled == nil {
			handled = make(map[string]interface{})
		}
		handled[sessionID] = map[string]interface{}{
			"status":         status,
			"handledThrough": handledThrough,
			"handledAt":      time.Now().UTC().Format(time.RFC3339Nano),
		}
		settings[activitySessionHandlingKey] = handled
		return nil
	})
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
		if err := a.appendActivityEvent(provider.PluginID, a.activityFromEvent(event)); err != nil {
			log.Printf("[api] activity provider %s failed to record %s: %v", provider.PluginID, event.Name, err)
		}
	}
}

func (a *App) appendActivityEvent(pluginID string, activity map[string]interface{}) error {
	_, err := a.storage.AppendPluginDataNDJSON(pluginID, activityRawDataName, []map[string]interface{}{activity}, storage.NDJSONRetention{
		TimestampField: "occurredAt",
		MaxAge:         activityRetention,
		MaxEntries:     maxActivityRawEvents,
		MaxBytes:       maxActivityRawBytes,
	})
	return err
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

func (a *App) activityFromEvent(event events.Event) map[string]interface{} {
	activity := activityFromEvent(event)
	workspaceRoot := firstPayloadText(activity, "workspaceRootPath")
	if workspaceRoot == "" || a == nil || a.treeV2 == nil {
		activity["sessionScope"] = map[string]interface{}{"kind": "unassigned"}
		return activity
	}
	identity, ok := a.treeV2.ResolveWorkspace(workspaceRoot)
	if !ok {
		activity["sessionScope"] = map[string]interface{}{"kind": "unassigned"}
		return activity
	}
	activity["workspaceId"] = identity.ID
	activity["workspaceRootPath"] = identity.RootPath
	activity["sessionScope"] = map[string]interface{}{"kind": "workspace", "workspaceId": identity.ID}
	return activity
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

// writeActivityPayload describes a write to whoever records activity. A plugin
// saving its own records asked for the write, so it says so, and the record is
// kept without being mistaken for the user's work.
func writeActivityPayload(opType string, options corefiles.WriteOptions) map[string]interface{} {
	payload := map[string]interface{}{"operation": opType}
	if options.Service {
		payload["service"] = true
	}
	return payload
}
