package api

import (
	"fmt"
	"sync"
	"testing"

	"github.com/verstak/verstak-desktop/internal/core/events"
	"github.com/verstak/verstak-desktop/internal/core/plugin"
)

func TestUpsertPluginSettingRecordRequiresLoadedPluginAndPermission(t *testing.T) {
	app := newBridgeTestApp(t)
	for _, pluginID := range []string{"missing.plugin", "disabled.plugin", "no.storage"} {
		if err := app.UpsertPluginSettingRecord(pluginID, "entries", "id", map[string]interface{}{"id": "one"}, 100); err == "" {
			t.Fatalf("accepted unauthorized plugin %s", pluginID)
		}
	}
	if err := app.UpsertPluginSettingRecord("bridge.plugin", "entries", "id", map[string]interface{}{"id": "one", "text": "saved"}, 100); err != "" {
		t.Fatal(err)
	}
	settings, err := app.ReadPluginSettings("bridge.plugin")
	if err != "" {
		t.Fatal(err)
	}
	if len(settings["entries"].([]interface{})) != 1 {
		t.Fatalf("records = %#v", settings)
	}
	if err := app.UpsertPluginSettingRecord("bridge.plugin", "entries", "id", map[string]interface{}{"id": "two"}, 0); err == "" {
		t.Fatal("expected validation failure")
	}
}

func TestManualRecordsAndReceiverCapturesDoNotOverwriteEachOther(t *testing.T) {
	app := newBridgeTestApp(t)
	app.plugins = append(app.plugins, plugin.Plugin{
		Manifest: plugin.Manifest{ID: browserInboxPluginID, Permissions: []string{"storage.namespace"}},
		Status:   plugin.StatusLoaded, Enabled: true,
	})
	app.ensureBrowserInboxSubscriptions()
	var writers sync.WaitGroup
	for index := 0; index < 40; index++ {
		writers.Add(1)
		go func(index int) {
			defer writers.Done()
			capture := map[string]interface{}{"captureId": fmt.Sprintf("capture-%d", index), "title": "Concurrent material"}
			if index%2 == 0 {
				if err := app.UpsertPluginSettingRecord(browserInboxPluginID, browserInboxGlobalKey, "captureId", capture, 100); err != "" {
					t.Error(err)
				}
			} else if err := app.recordBrowserCapture(events.Event{Name: "browser.capture.link", Payload: capture}); err != nil {
				t.Error(err)
			}
		}(index)
	}
	writers.Wait()
	settings, err := app.ReadPluginSettings(browserInboxPluginID)
	if err != "" {
		t.Fatal(err)
	}
	records := settings[browserInboxGlobalKey].([]interface{})
	if len(records) != 40 {
		t.Fatalf("got %d records, want all 40", len(records))
	}
	seen := make(map[string]bool)
	for _, record := range records {
		seen[record.(map[string]interface{})["captureId"].(string)] = true
	}
	if len(seen) != 40 {
		t.Fatalf("got %d distinct records", len(seen))
	}
}
