package api

import (
	"fmt"
	"log"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/verstak/verstak-desktop/internal/core/browserreceiver"
	"github.com/verstak/verstak-desktop/internal/core/events"
	"github.com/verstak/verstak-desktop/internal/core/storage"
	"github.com/verstak/verstak-desktop/internal/core/vault"
)

const browserInboxPluginID = "verstak.browser-inbox"
const browserInboxGlobalKey = "captures:global"
const browserInboxLegacyKey = "captures"
const browserInboxWorkspacePrefix = "captures:workspace:"
const browserInboxMutationEvent = "browser-inbox.storage.mutate"
const maxBrowserInboxCaptures = 100

func (a *App) ensureBrowserInboxSubscriptions() {
	if a.eventBus == nil || a.storage == nil {
		a.browserInboxEnabled.Store(false)
		return
	}
	if _, err := a.requirePluginAccess(browserInboxPluginID, "storage.namespace"); err != nil {
		a.browserInboxEnabled.Store(false)
		return
	}
	a.browserInboxEnabled.Store(true)
	if a.browserInboxEvents == nil {
		a.browserInboxEvents = make(map[string]bool)
	}
	for _, eventName := range []string{browserInboxMutationEvent, workspaceRenamedEventName, workspaceTrashedEventName, workspaceRestoredEventName, workspacePurgedEventName} {
		if a.browserInboxEvents[eventName] {
			continue
		}
		a.browserInboxEvents[eventName] = true
		a.eventBus.Subscribe(eventName, func(event events.Event) {
			if event.Name == browserInboxMutationEvent {
				if err := a.mutateBrowserInboxCapture(event); err != nil {
					log.Printf("[api] browser inbox mutation failed: %v", err)
				}
				return
			}
			if err := a.updateBrowserInboxWorkspaceLifecycle(event); err != nil {
				log.Printf("[api] browser inbox workspace lifecycle failed: %v", err)
			}
		})
	}
}

func (a *App) browserInboxAvailable() bool {
	if a == nil || a.storage == nil || a.vault == nil || !a.browserInboxEnabled.Load() || a.vault.GetVaultStatus() != vault.StatusOpen {
		return false
	}
	return true
}

func (a *App) recordBrowserActivityBatch(event events.Event) error {
	if !a.activityAvailable() {
		return fmt.Errorf("activity storage unavailable")
	}
	payload := eventPayloadMap(event.Payload)
	batchID := firstPayloadText(payload, "batchId")
	if batchID == "" {
		return fmt.Errorf("batchId is empty")
	}
	entries, ok := payload["entries"].([]map[string]interface{})
	if !ok || len(entries) == 0 {
		return fmt.Errorf("activity batch entries are empty")
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	receivedAt := event.Timestamp
	if receivedAt == "" {
		receivedAt = now
	}
	records := make([]map[string]interface{}, 0, len(entries))
	for index, entry := range entries {
		hostname := firstPayloadText(entry, "hostname")
		endedAt := firstPayloadText(entry, "endedAt")
		if hostname == "" || endedAt == "" {
			return fmt.Errorf("activity batch entry %d is invalid", index)
		}
		durationSeconds, _ := entry["durationSeconds"].(int64)
		if durationSeconds == 0 {
			if number, ok := entry["durationSeconds"].(float64); ok {
				durationSeconds = int64(number)
			}
		}
		pageURL := firstPayloadText(entry, "url")
		// The address is what the user recognises; a batch from an older
		// extension has only the site, and then the site is the best title
		// there is.
		title := hostname
		if pageURL != "" {
			title = pageURL
		}
		record := map[string]interface{}{
			"activityId": fmt.Sprintf("browser-domain:%s:%d", batchID, index),
			"type":       "browser.activity.domain",
			"title":      title,
			// No summary: the duration is on the record as a number, and
			// whoever displays it says it in the reader's language. Writing
			// "18 min browser activity" here put untranslated English in front
			// of the user.
			"occurredAt":        endedAt,
			"receivedAt":        receivedAt,
			"sourcePluginId":    "verstak-browser-extension",
			"sourceBatchId":     batchID,
			"hostname":          hostname,
			"startedAt":         firstPayloadText(entry, "startedAt"),
			"endedAt":           endedAt,
			"durationSeconds":   durationSeconds,
			"workspaceRootPath": "",
			"payload": map[string]interface{}{
				"hostname":        hostname,
				"startedAt":       firstPayloadText(entry, "startedAt"),
				"endedAt":         endedAt,
				"durationSeconds": durationSeconds,
			},
		}
		if pageURL != "" {
			record["url"] = pageURL
			record["payload"].(map[string]interface{})["url"] = pageURL
		}
		records = append(records, record)
	}
	_, err := a.storage.AppendPluginDataNDJSON(activityPluginID, activityRawDataName, records, storage.NDJSONRetention{
		TimestampField:   "occurredAt",
		MaxAge:           activityRetention,
		MaxEntries:       maxActivityRawEvents,
		MaxBytes:         maxActivityRawBytes,
		DeduplicateField: "sourceBatchId",
		DeduplicateValue: batchID,
	})
	return err
}

func (a *App) recordBrowserCapture(event events.Event) error {
	if !a.browserInboxAvailable() {
		return fmt.Errorf("browser inbox unavailable")
	}
	capture := eventPayloadMap(event.Payload)
	captureID := firstPayloadText(capture, "captureId")
	if captureID == "" {
		return fmt.Errorf("captureId is empty")
	}
	if firstPayloadText(capture, "kind") == "" {
		capture["kind"] = strings.TrimPrefix(event.Name, "browser.capture.")
	}
	if firstPayloadText(capture, "capturedAt") == "" {
		capture["capturedAt"] = event.Timestamp
	}
	if firstPayloadText(capture, "globalState") == "" {
		capture["globalState"] = "inbox"
	}
	a.annotateBrowserCaptureWorkspace(capture)
	capture["receivedAt"] = time.Now().UTC().Format(time.RFC3339Nano)
	return a.updateBrowserInboxCaptures(func(captures []map[string]interface{}) []map[string]interface{} {
		for _, stored := range captures {
			if firstPayloadText(stored, "captureId") == captureID {
				return captures
			}
		}
		result := []map[string]interface{}{capture}
		result = append(result, captures...)
		return result
	})
}

func (a *App) mutateBrowserInboxCapture(event events.Event) error {
	payload := eventPayloadMap(event.Payload)
	if firstPayloadText(payload, "pluginId") != browserInboxPluginID {
		return fmt.Errorf("browser inbox mutation source is not authorized")
	}
	action := firstPayloadText(payload, "action")
	switch action {
	case "migrate", "assign", "archive", "restore", "delete", "processed":
	default:
		return fmt.Errorf("unsupported browser inbox mutation %q", action)
	}
	captureID := firstPayloadText(payload, "captureId")
	captureIDs := make(map[string]bool)
	if captureID != "" {
		captureIDs[captureID] = true
	}
	if items, ok := payload["captureIds"].([]interface{}); ok {
		for _, item := range items {
			if id, ok := item.(string); ok && strings.TrimSpace(id) != "" {
				captureIDs[strings.TrimSpace(id)] = true
			}
		}
	}
	if action != "migrate" && len(captureIDs) == 0 {
		return fmt.Errorf("captureId is empty")
	}
	return a.updateBrowserInboxCaptures(func(captures []map[string]interface{}) []map[string]interface{} {
		result := make([]map[string]interface{}, 0, len(captures))
		for _, capture := range captures {
			storedID := firstPayloadText(capture, "captureId")
			if !captureIDs[storedID] {
				result = append(result, capture)
				continue
			}
			switch action {
			case "delete":
				continue
			case "archive":
				capture["globalState"] = "archived"
			case "restore":
				capture["globalState"] = "inbox"
			case "assign":
				workspaceRoot := firstPayloadText(payload, "workspaceRootPath")
				capture["workspaceRootPath"] = workspaceRoot
				capture["workspaceName"] = workspaceRoot
				delete(capture, "workspaceId")
				delete(capture, "workspaceTrashId")
				a.annotateBrowserCaptureWorkspace(capture)
			case "processed":
				capture["processed"], _ = payload["processed"].(bool)
			}
			result = append(result, capture)
		}
		return result
	})
}

func (a *App) annotateBrowserCaptureWorkspace(capture map[string]interface{}) {
	if capture == nil {
		return
	}
	workspaceRoot := firstPayloadText(capture, "workspaceRootPath")
	if workspaceRoot == "" {
		capture["workspaceState"] = "unassigned"
		delete(capture, "workspaceId")
		delete(capture, "workspaceTrashId")
		return
	}
	if firstPayloadText(capture, "workspaceId") != "" {
		if firstPayloadText(capture, "workspaceState") == "" {
			capture["workspaceState"] = "active"
		}
		return
	}
	if a.treeV2 == nil {
		capture["workspaceState"] = "unavailable"
		return
	}
	identity, ok := a.treeV2.ResolveWorkspace(workspaceRoot)
	if !ok {
		capture["workspaceState"] = "unavailable"
		return
	}
	capture["workspaceId"] = identity.ID
	capture["workspaceRootPath"] = identity.RootPath
	capture["workspaceName"] = identity.Name
	capture["workspaceState"] = "active"
}

func (a *App) updateBrowserInboxWorkspaceLifecycle(event events.Event) error {
	payload := eventPayloadMap(event.Payload)
	workspaceID := firstPayloadText(payload, "workspaceId")
	if workspaceID == "" {
		return nil
	}
	return a.updateBrowserInboxCaptures(func(captures []map[string]interface{}) []map[string]interface{} {
		for _, capture := range captures {
			if firstPayloadText(capture, "workspaceId") != workspaceID {
				continue
			}
			switch event.Name {
			case workspaceRenamedEventName, workspaceRestoredEventName:
				capture["workspaceRootPath"] = firstPayloadText(payload, "workspaceRootPath")
				capture["workspaceName"] = firstPayloadText(payload, "workspaceName", "workspaceRootPath")
				capture["workspaceState"] = "active"
				delete(capture, "workspaceTrashId")
			case workspaceTrashedEventName:
				capture["workspaceState"] = "trashed"
				capture["workspaceTrashId"] = firstPayloadText(payload, "trashId")
			case workspacePurgedEventName:
				capture["workspaceState"] = "orphaned"
				delete(capture, "workspaceTrashId")
			}
		}
		return captures
	})
}

func (a *App) updateBrowserInboxCaptures(update func([]map[string]interface{}) []map[string]interface{}) error {
	if !a.browserInboxAvailable() {
		return fmt.Errorf("browser inbox unavailable")
	}
	return a.storage.UpdatePluginSettings(browserInboxPluginID, func(settings map[string]interface{}) error {
		captures, legacyKeys := browserInboxCaptures(settings)
		captures = update(captures)
		if len(captures) > maxBrowserInboxCaptures {
			captures = captures[:maxBrowserInboxCaptures]
		}
		stored := make([]interface{}, 0, len(captures))
		for _, capture := range captures {
			stored = append(stored, capture)
		}
		settings[browserInboxGlobalKey] = stored
		for _, key := range legacyKeys {
			settings[key] = []interface{}{}
		}
		return nil
	})
}

func browserInboxCaptures(settings map[string]interface{}) ([]map[string]interface{}, []string) {
	keys := []string{browserInboxGlobalKey, browserInboxLegacyKey}
	for key := range settings {
		if strings.HasPrefix(key, browserInboxWorkspacePrefix) {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys[2:])
	seen := make(map[string]bool)
	legacyKeys := make([]string, 0, len(keys)-1)
	var captures []map[string]interface{}
	for _, key := range keys {
		if key != browserInboxGlobalKey {
			legacyKeys = append(legacyKeys, key)
		}
		workspaceRoot := ""
		if strings.HasPrefix(key, browserInboxWorkspacePrefix) {
			workspaceRoot, _ = url.PathUnescape(strings.TrimPrefix(key, browserInboxWorkspacePrefix))
		}
		items, _ := settings[key].([]interface{})
		for _, item := range items {
			original, ok := item.(map[string]interface{})
			if !ok {
				continue
			}
			capture := eventPayloadMap(original)
			captureID := firstPayloadText(capture, "captureId")
			if captureID == "" || seen[captureID] {
				continue
			}
			seen[captureID] = true
			if firstPayloadText(capture, "globalState") == "" {
				capture["globalState"] = "inbox"
			}
			if firstPayloadText(capture, "workspaceRootPath") == "" && workspaceRoot != "" {
				capture["workspaceRootPath"] = workspaceRoot
				capture["workspaceName"] = workspaceRoot
			}
			captures = append(captures, capture)
		}
	}
	return captures, legacyKeys
}

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
