package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/verstak/verstak-desktop/internal/core/appsettings"
	"github.com/verstak/verstak-desktop/internal/core/updates"
)

func newUpdatesTestApp(t *testing.T, tag string) (*App, *atomic.Int32) {
	t.Helper()
	var hits atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hits.Add(1)
		_, _ = w.Write([]byte(`{"tag_name":"` + tag + `","html_url":"https://github.com/mirivlad/verstak/releases/tag/` + tag + `"}`))
	}))
	t.Cleanup(server.Close)
	app := &App{updateChecker: updates.Checker{Endpoint: server.URL}}
	app.appSettings = appsettings.NewManager(filepath.Join(t.TempDir(), "config.json"))
	return app, &hits
}

// The startup check reaches the network, so with the setting at its default
// it must not happen at all.
func TestStartupUpdateCheckStaysOfflineByDefault(t *testing.T) {
	app, hits := newUpdatesTestApp(t, "v99.0.0")
	originalEmit := emitFrontendEvent
	emitFrontendEvent = func(context.Context, string, ...interface{}) { t.Error("no event expected") }
	t.Cleanup(func() { emitFrontendEvent = originalEmit })

	app.checkForUpdatesAtStartup(context.Background())
	time.Sleep(100 * time.Millisecond)

	if hits.Load() != 0 {
		t.Fatalf("startup contacted the release server %d time(s) with the check turned off", hits.Load())
	}
}

func TestStartupUpdateCheckAnnouncesANewerReleaseWhenEnabled(t *testing.T) {
	app, hits := newUpdatesTestApp(t, "v99.0.0")
	if errText := app.UpdateAppSettings(map[string]interface{}{"checkForUpdates": true}); errText != "" {
		t.Fatal(errText)
	}
	if app.GetAppSettings()["checkForUpdates"] != true {
		t.Fatal("the setting must read back as enabled")
	}
	announced := make(chan map[string]interface{}, 1)
	originalEmit := emitFrontendEvent
	emitFrontendEvent = func(_ context.Context, name string, data ...interface{}) {
		if name == updateAvailableEvent && len(data) == 1 {
			announced <- data[0].(map[string]interface{})
		}
	}
	t.Cleanup(func() { emitFrontendEvent = originalEmit })

	app.checkForUpdatesAtStartup(context.Background())

	select {
	case result := <-announced:
		if result["latest"] != "v99.0.0" || result["newer"] != true {
			t.Fatalf("unexpected announcement %#v", result)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("no update announcement")
	}
	if hits.Load() != 1 {
		t.Fatalf("expected one request, got %d", hits.Load())
	}
}

func TestOpenUpdatePageUsesOnlyTheCheckedReleaseURL(t *testing.T) {
	app, _ := newUpdatesTestApp(t, "v99.0.0")
	var opened string
	app.externalOpen = newTestExternalOpenService(func(path string) error { opened = path; return nil })

	if errText := app.OpenUpdatePage(); errText == "" {
		t.Fatal("opening a release page before any check must be refused")
	}
	if _, errText := app.CheckForUpdates(); errText != "" {
		t.Fatal(errText)
	}
	if errText := app.OpenUpdatePage(); errText != "" {
		t.Fatal(errText)
	}
	if opened != "https://github.com/mirivlad/verstak/releases/tag/v99.0.0" {
		t.Fatalf("opened %q", opened)
	}
}

func TestUpdateCheckSettingRejectsNonBoolean(t *testing.T) {
	app, _ := newUpdatesTestApp(t, "v99.0.0")
	if errText := app.UpdateAppSettings(map[string]interface{}{"checkForUpdates": "yes"}); errText == "" {
		t.Fatal("a non-boolean value must be rejected")
	}
}
