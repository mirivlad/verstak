package api

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/verstak/verstak-desktop/internal/core/plugin"
	"github.com/verstak/verstak-desktop/internal/core/pluginstate"
	"github.com/verstak/verstak-desktop/internal/core/processes"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

func processBridge(t *testing.T) (*App, processes.Request) {
	t.Helper()
	a := newBridgeTestApp(t)
	a.pluginState = pluginstate.NewManager(a.vault)
	if err := a.pluginState.Load(); err != nil {
		t.Fatal(err)
	}
	a.plugins[0].Manifest.Requires = append(a.plugins[0].Manifest.Requires, processes.Capability)
	a.plugins[0].Manifest.Permissions = append(a.plugins[0].Manifest.Permissions, "process.spawn", "external.openURL")
	a.processes = processes.New(filepath.Join(t.TempDir(), "approvals.json"))
	return a, processes.Request{Kind: "url", URL: "https://example.org/actions-test"}
}
func TestProcessBridgeOptionalConfirmationDoesNotGrantAccess(t *testing.T) {
	a, r := processBridge(t)
	opens := 0
	a.externalOpen = newTestExternalOpenService(func(string) error { opens++; return nil })
	a.confirmProcess = func(context.Context, runtime.MessageDialogOptions) (string, error) {
		t.Fatal("local launch opened a dialog")
		return "No", nil
	}
	if _, err := a.PluginProcessRun("bridge.plugin", "one", r); err != nil || opens != 1 {
		t.Fatal("authorized first launch blocked", err)
	}
	seen := false
	a.confirmProcess = func(_ context.Context, options runtime.MessageDialogOptions) (string, error) {
		seen = strings.Contains(options.Message, r.URL) && options.DefaultButton == "No"
		return "No", nil
	}
	if allowed, err := a.PluginProcessConfirm("bridge.plugin", r); err != nil || allowed || !seen {
		t.Fatal("cancel did not fail closed", err)
	}
	a.confirmProcess = func(_ context.Context, options runtime.MessageDialogOptions) (string, error) { return "Yes", nil }
	if accepted, err := a.PluginProcessConfirm("bridge.plugin", r); err != nil || !accepted {
		t.Fatal(err)
	}
	if granted, err := a.PluginProcessApproved("bridge.plugin", r); err != nil || granted {
		t.Fatal("confirmation granted a legacy approval", err)
	}
	if allowed, err := a.PluginProcessReview("bridge.plugin", r); err != nil || !allowed {
		t.Fatal("review failed", err)
	}
	if allowed, err := a.PluginProcessApproved("bridge.plugin", r); err != nil || !allowed {
		t.Fatal("review not saved", err)
	}
	if err := a.PluginProcessRevoke("bridge.plugin", r); err != nil {
		t.Fatal(err)
	}
	if allowed, _ := a.PluginProcessApproved("bridge.plugin", r); allowed {
		t.Fatal("revoke failed")
	}
}
func TestProcessBridgeChecksPermissionsAndCurrentStateEveryCall(t *testing.T) {
	a, r := processBridge(t)
	for _, id := range []string{"missing.plugin", "disabled.plugin", "no.storage"} {
		if _, err := a.PluginProcessApproved(id, r); err == nil {
			t.Fatalf("%s reached approval store", id)
		}
		if _, err := a.PluginProcessConfirm(id, r); err == nil {
			t.Fatalf("%s reached review", id)
		}
		if _, err := a.PluginProcessRun(id, "one", r); err == nil {
			t.Fatalf("%s reached launch", id)
		}
		if err := a.PluginProcessCancel(id, "one"); err == nil {
			t.Fatalf("%s reached cancellation", id)
		}
	}
	a.confirmProcess = func(_ context.Context, _ runtime.MessageDialogOptions) (string, error) { return "Yes", nil }
	if ok, err := a.PluginProcessReview("bridge.plugin", r); err != nil || !ok {
		t.Fatal(err)
	}
	if err := a.DisablePlugin("bridge.plugin"); err != "" {
		t.Fatal(err)
	}
	if _, err := a.PluginProcessRun("bridge.plugin", "one", r); err == nil {
		t.Fatal("stale loaded state bypassed disable")
	}
	if err := a.EnablePlugin("bridge.plugin"); err != "" {
		t.Fatal(err)
	}
	if ok, err := a.PluginProcessApproved("bridge.plugin", r); err != nil || !ok {
		t.Fatal("enable lost approval", err)
	}
	a.plugins[0].Manifest.Requires = nil
	if _, err := a.PluginProcessRun("bridge.plugin", "no-capability", r); err == nil {
		t.Fatal("missing capability bypassed")
	}
	a.plugins[0].Manifest.Requires = []string{processes.Capability}
	a.plugins[0].Manifest.Permissions = nil
	if _, err := a.PluginProcessApproved("bridge.plugin", r); err == nil {
		t.Fatal("permission removal bypassed")
	}
}
func TestNativeReviewExpiresOnStateChange(t *testing.T) {
	a, r := processBridge(t)
	a.confirmProcess = func(_ context.Context, _ runtime.MessageDialogOptions) (string, error) {
		a.stopPluginProcesses("")
		return "Yes", nil
	}
	if ok, err := a.PluginProcessReview("bridge.plugin", r); err == nil || ok {
		t.Fatal("stale review was accepted")
	}
}

func TestSyncedActionsNeverTransferDeviceApprovalOrExecute(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", home)
	t.Setenv("APPDATA", home)
	t.Setenv("LOCALAPPDATA", home)
	source := newRecordSyncApp(t, "actions-source")
	target := newRecordSyncApp(t, "actions-target")
	for _, a := range []*App{source, target} {
		a.plugins[0].Manifest.ID = "verstak.actions"
		a.plugins[0].Manifest.Requires = []string{processes.Capability}
		a.plugins[0].Manifest.Permissions = []string{"storage.namespace", "external.openURL", "process.spawn"}
		a.plugins[0].Manifest.Sync.Records = []plugin.SyncRecordSet{{ID: "actions", Storage: "settings", Key: "actions", Identity: "id"}}
		a.processes = processes.New(filepath.Join(t.TempDir(), "local-approvals.json"))
	}
	if err := source.WritePluginLocalDataJSON("verstak.actions", "action-trust-v1", map[string]interface{}{"global": "source-local-hash"}); err != "" {
		t.Fatal(err)
	}
	r := processes.Request{Kind: "url", URL: "https://example.org/imported-action"}
	source.confirmProcess = func(_ context.Context, _ runtime.MessageDialogOptions) (string, error) { return "Yes", nil }
	if ok, err := source.PluginProcessReview("verstak.actions", r); err != nil || !ok {
		t.Fatal(err)
	}
	for _, record := range []map[string]interface{}{
		{"id": "global", "title": "Imported global", "workspaceId": "", "type": "url", "target": r.URL, "approved": true},
		{"id": "local", "title": "Imported local", "workspaceId": "11111111-1111-4111-8111-111111111111", "type": "application", "target": "printf"},
	} {
		if err := source.UpsertPluginSettingRecord("verstak.actions", "actions", "id", record, 10000); err != "" {
			t.Fatal(err)
		}
	}
	opens := 0
	target.externalOpen = newTestExternalOpenService(func(string) error { opens++; return nil })
	for _, op := range unpushedOps(t, source) {
		if err := target.applyRemotePluginRecordOp(op); err != nil {
			t.Fatal(err)
		}
	}
	records, err := target.ReadPluginSettings("verstak.actions")
	if err != "" || len(recordsFromSettingsValue(records["actions"])) != 2 {
		t.Fatal("definitions did not sync", err)
	}
	if opens != 0 {
		t.Fatal("sync executed URL handler")
	}
	if ok, err := target.PluginProcessApproved("verstak.actions", r); err != nil || ok {
		t.Fatal("source approval crossed devices", err)
	}
	local, localErr := target.ReadPluginLocalDataJSON("verstak.actions", "action-trust-v1")
	if localErr != nil || len(local) != 0 {
		t.Fatal("record data minted local trust", local, localErr)
	}
	if opens != 0 {
		t.Fatal("sync executed a process")
	}

}

func TestConfirmationPreviewShowsProgramArgumentsAndDirectory(t *testing.T) {
	r := processes.Request{Kind: "process", Executable: "/usr/bin/printf", Args: []string{"%s", "two words", "$(literal)", ""}, Cwd: "/tmp", Mode: "wait"}
	for _, ru := range []bool{false, true} {
		text := processConfirmationPreview(r, ru)
		for _, value := range []string{"/usr/bin/printf", `"%s"`, `"two words"`, `"$(literal)"`, `""`, "/tmp"} {
			if !strings.Contains(text, value) {
				t.Fatalf("confirmation omitted %q: %s", value, text)
			}
		}
	}
}
