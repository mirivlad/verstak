package api

import (
	"context"
	"log"
	"time"

	"github.com/verstak/verstak-desktop/internal/core/buildinfo"
	"github.com/verstak/verstak-desktop/internal/core/updates"
)

const updateAvailableEvent = "verstak:update-available"

// CheckForUpdates asks the release server whether a newer Verstak exists. It
// is only ever called because the user clicked, or because the user turned on
// the startup check; nothing is downloaded either way.
func (a *App) CheckForUpdates() (map[string]interface{}, string) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	result, err := a.updateChecker.Check(ctx, buildinfo.Get().Version)
	if err != nil {
		return nil, err.Error()
	}
	a.updateMu.Lock()
	a.lastUpdate = &result
	a.updateMu.Unlock()
	return updateResultMap(result), ""
}

// OpenUpdatePage opens the release page of the last check that found a newer
// release. The URL comes from that check, never from the frontend.
func (a *App) OpenUpdatePage() string {
	a.updateMu.Lock()
	last := a.lastUpdate
	a.updateMu.Unlock()
	if last == nil || !last.Newer {
		return "no newer release has been found"
	}
	if err := a.externalOpenService().OpenPath(last.URL); err != nil {
		return err.Error()
	}
	return ""
}

// checkForUpdatesAtStartup runs the check once, in the background, when the
// user has asked for it, and tells the shell only when there is something new.
func (a *App) checkForUpdatesAtStartup(ctx context.Context) {
	if a.appSettings == nil || !a.appSettings.Get().CheckForUpdates {
		return
	}
	go func() {
		result, errText := a.CheckForUpdates()
		if errText != "" {
			log.Printf("[updates] startup check failed: %s", errText)
			return
		}
		if result["newer"] == true {
			emitFrontendEvent(ctx, updateAvailableEvent, result)
		}
	}()
}

func updateResultMap(result updates.Result) map[string]interface{} {
	return map[string]interface{}{
		"current":   result.Current,
		"latest":    result.Latest,
		"url":       result.URL,
		"newer":     result.Newer,
		"checkedAt": result.CheckedAt,
	}
}
