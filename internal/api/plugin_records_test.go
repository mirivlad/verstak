package api

import (
	"testing"

	"github.com/verstak/verstak-desktop/internal/core/events"
	"github.com/verstak/verstak-desktop/internal/core/plugin"
	"github.com/verstak/verstak-desktop/internal/core/pluginsync"
	"github.com/verstak/verstak-desktop/internal/core/storage"
	syncsvc "github.com/verstak/verstak-desktop/internal/core/sync"
	"github.com/verstak/verstak-desktop/internal/core/vault"
)

func newRecordSyncApp(t *testing.T, deviceID string) *App {
	t.Helper()
	v := vault.NewVault(nil)
	if err := v.CreateVault(t.TempDir()); err != nil {
		t.Fatalf("CreateVault: %v", err)
	}
	app := &App{
		vault:    v,
		storage:  storage.New(v),
		syncSvc:  syncsvc.NewService(v.GetVaultPath(), deviceID),
		eventBus: events.NewBus(),
		plugins: []plugin.Plugin{
			{
				Manifest: plugin.Manifest{
					ID:          "verstak.todo",
					Name:        "Todos",
					Version:     "1.0.0",
					Permissions: []string{"storage.namespace"},
					Sync: &plugin.SyncConfig{
						Records: []plugin.SyncRecordSet{
							{ID: "todos", Storage: "settings", Key: "todos:global", Identity: "id"},
						},
					},
				},
				Status:  plugin.StatusLoaded,
				Enabled: true,
			},
		},
	}
	return app
}

func todoRecords(titles ...string) []map[string]interface{} {
	records := make([]map[string]interface{}, 0, len(titles))
	for index, title := range titles {
		records = append(records, map[string]interface{}{
			"id":    string(rune('a' + index)),
			"title": title,
		})
	}
	return records
}

func writeTodos(t *testing.T, app *App, records []map[string]interface{}) {
	t.Helper()
	value := make([]interface{}, 0, len(records))
	for _, record := range records {
		value = append(value, record)
	}
	if errText := app.WritePluginSetting("verstak.todo", "todos:global", value); errText != "" {
		t.Fatalf("WritePluginSetting: %s", errText)
	}
}

func readTodos(t *testing.T, app *App) []map[string]interface{} {
	t.Helper()
	settings, errText := app.ReadPluginSettings("verstak.todo")
	if errText != "" {
		t.Fatalf("ReadPluginSettings: %s", errText)
	}
	return recordsFromSettingsValue(settings["todos:global"])
}

func unpushedOps(t *testing.T, app *App) []syncsvc.Op {
	t.Helper()
	ops, err := app.syncSvc.GetUnpushedOps()
	if err != nil {
		t.Fatalf("GetUnpushedOps: %v", err)
	}
	return ops
}

// A plugin saving a declared list records one operation per changed record, not
// one for the whole document. That is the difference between two devices
// keeping both their todos and one of them losing theirs.
func TestPluginSettingWriteRecordsOnePerChangedRecord(t *testing.T) {
	app := newRecordSyncApp(t, "device-1")

	writeTodos(t, app, todoRecords("buy milk", "call the client"))
	ops := unpushedOps(t, app)
	if len(ops) != 2 {
		t.Fatalf("ops after first write = %#v", ops)
	}
	for _, op := range ops {
		if op.EntityType != pluginsync.EntityType || op.OpType != syncsvc.OpCreate {
			t.Fatalf("unexpected op: %#v", op)
		}
	}

	// Changing one todo records one update, and saving the same list again
	// records nothing at all.
	writeTodos(t, app, []map[string]interface{}{
		{"id": "a", "title": "buy milk"},
		{"id": "b", "title": "call the client back"},
	})
	ops = unpushedOps(t, app)
	if len(ops) != 3 || ops[2].OpType != syncsvc.OpUpdate {
		t.Fatalf("ops after edit = %#v", ops)
	}
	writeTodos(t, app, []map[string]interface{}{
		{"id": "a", "title": "buy milk"},
		{"id": "b", "title": "call the client back"},
	})
	if ops := unpushedOps(t, app); len(ops) != 3 {
		t.Fatalf("saving an unchanged list recorded something: %#v", ops)
	}

	writeTodos(t, app, []map[string]interface{}{{"id": "a", "title": "buy milk"}})
	ops = unpushedOps(t, app)
	if len(ops) != 4 || ops[3].OpType != syncsvc.OpDelete {
		t.Fatalf("ops after delete = %#v", ops)
	}
	if _, document, recordID, ok := pluginsync.ParseEntityID(ops[3].EntityID); !ok || document != "todos" || recordID != "b" {
		t.Fatalf("delete op identity = %q", ops[3].EntityID)
	}
}

// The point of the whole exercise: a todo added on each device ends up on both.
func TestPluginRecordOpsFromAnotherDeviceMergeRatherThanReplace(t *testing.T) {
	laptop := newRecordSyncApp(t, "laptop")
	desktop := newRecordSyncApp(t, "desktop")

	writeTodos(t, laptop, []map[string]interface{}{{"id": "shared", "title": "prepare the invoice"}})
	writeTodos(t, desktop, []map[string]interface{}{{"id": "shared", "title": "prepare the invoice"}})

	writeTodos(t, laptop, []map[string]interface{}{
		{"id": "shared", "title": "prepare the invoice"},
		{"id": "from-laptop", "title": "written on the laptop"},
	})
	writeTodos(t, desktop, []map[string]interface{}{
		{"id": "shared", "title": "prepare the invoice"},
		{"id": "from-desktop", "title": "written on the desktop"},
	})

	// The laptop's new todo reaches the desktop.
	laptopOps := unpushedOps(t, laptop)
	incoming := laptopOps[len(laptopOps)-1]
	if err := desktop.applyRemotePluginRecordOp(incoming); err != nil {
		t.Fatalf("applyRemotePluginRecordOp: %v", err)
	}

	titles := map[string]bool{}
	for _, record := range readTodos(t, desktop) {
		titles[record["title"].(string)] = true
	}
	for _, expected := range []string{"prepare the invoice", "written on the laptop", "written on the desktop"} {
		if !titles[expected] {
			t.Fatalf("%q is missing after the merge: %#v", expected, titles)
		}
	}
}

// Applying a pull must not push it straight back.
func TestApplyingARemoteRecordDoesNotRecordItAgain(t *testing.T) {
	laptop := newRecordSyncApp(t, "laptop")
	desktop := newRecordSyncApp(t, "desktop")

	writeTodos(t, laptop, []map[string]interface{}{{"id": "a", "title": "one"}})
	incoming := unpushedOps(t, laptop)[0]

	if err := desktop.applyRemotePluginRecordOp(incoming); err != nil {
		t.Fatalf("applyRemotePluginRecordOp: %v", err)
	}
	if ops := unpushedOps(t, desktop); len(ops) != 0 {
		t.Fatalf("applying a pulled record recorded new operations: %#v", ops)
	}
	if records := readTodos(t, desktop); len(records) != 1 || records[0]["title"] != "one" {
		t.Fatalf("records after apply = %#v", records)
	}

	// And applying it twice leaves the same result.
	if err := desktop.applyRemotePluginRecordOp(incoming); err != nil {
		t.Fatalf("second applyRemotePluginRecordOp: %v", err)
	}
	if records := readTodos(t, desktop); len(records) != 1 {
		t.Fatalf("re-applying duplicated the record: %#v", records)
	}
}

// A plugin that does not share a set gets nothing recorded for it, and an
// operation for a plugin this device does not have must not stop the pull.
func TestUndeclaredPluginDataStaysOnThisDevice(t *testing.T) {
	app := newRecordSyncApp(t, "device-1")
	app.plugins[0].Manifest.Sync = nil

	writeTodos(t, app, []map[string]interface{}{{"id": "a", "title": "one"}})
	if ops := unpushedOps(t, app); len(ops) != 0 {
		t.Fatalf("data nobody agreed to share was recorded: %#v", ops)
	}

	stranger := syncsvc.Op{
		EntityType:  pluginsync.EntityType,
		EntityID:    pluginsync.EntityID("someone.else", "things", "x"),
		OpType:      syncsvc.OpCreate,
		PayloadJSON: `{"pluginId":"someone.else","document":"things","recordId":"x","record":{"id":"x"}}`,
	}
	if err := app.applyRemotePluginRecordOp(stranger); err != nil {
		t.Fatalf("an operation for an absent plugin stopped the pull: %v", err)
	}
}

// A device that just learned about a record has to tell the plugin, or the list
// on screen stays as it was.
func TestApplyingARemoteRecordAnnouncesIt(t *testing.T) {
	app := newRecordSyncApp(t, "device-1")
	var seen []events.Event
	app.eventBus.Subscribe(pluginRecordsChangedEvent, func(event events.Event) {
		seen = append(seen, event)
	})

	incoming := syncsvc.Op{
		EntityType:  pluginsync.EntityType,
		EntityID:    pluginsync.EntityID("verstak.todo", "todos", "a"),
		OpType:      syncsvc.OpCreate,
		PayloadJSON: `{"pluginId":"verstak.todo","document":"todos","recordId":"a","record":{"id":"a","title":"one"}}`,
	}
	if err := app.applyRemotePluginRecordOp(incoming); err != nil {
		t.Fatalf("applyRemotePluginRecordOp: %v", err)
	}
	if len(seen) != 1 {
		t.Fatalf("events = %#v", seen)
	}
	payload, ok := seen[0].Payload.(map[string]interface{})
	if !ok || payload["pluginId"] != "verstak.todo" || payload["document"] != "todos" {
		t.Fatalf("event payload = %#v", seen[0].Payload)
	}
}
