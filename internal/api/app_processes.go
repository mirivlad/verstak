package api

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	goruntime "runtime"
	"strconv"
	"strings"

	"github.com/verstak/verstak-desktop/internal/core/processes"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

func (a *App) processOwner(pluginID string) string { return a.vaultPath() + "\x00" + pluginID }

// processAccessLocked checks Go state on EVERY call. The shared frontend and
// a plugin-supplied identity are not a sandbox or an authenticated caller.
func (a *App) processAccessLocked(pluginID string, request processes.Request) (processes.Request, string, error) {
	permission := "process.spawn"
	if request.Kind == "url" {
		permission = "external.openURL"
	}
	if _, err := a.requirePluginAccess(pluginID, permission); err != nil {
		return processes.Request{}, "", err
	}
	if _, err := a.requirePluginCapabilityAccess(pluginID, processes.Capability); err != nil {
		return processes.Request{}, "", err
	}
	if err := a.requireVault(); err != nil {
		return processes.Request{}, "", err
	}
	r, err := processes.Normalize(request)
	if err != nil {
		return processes.Request{}, "", err
	}
	if a.processes == nil {
		dir, err := os.UserConfigDir()
		if goruntime.GOOS == "windows" {
			dir, err = os.UserCacheDir()
		}
		if err != nil {
			return processes.Request{}, "", errors.New("approval_store_unavailable")
		}
		a.processes = processes.New(filepath.Join(dir, "verstak", "process-approvals.json"))
	}
	return r, a.processOwner(pluginID), nil
}

func (a *App) PluginProcessApproved(pluginID string, request processes.Request) (bool, error) {
	a.processMu.Lock()
	defer a.processMu.Unlock()
	r, owner, err := a.processAccessLocked(pluginID, request)
	if err != nil {
		return false, err
	}
	return a.processes.Approved(owner, r)
}

// PluginProcessReview grants only the exact request shown by the HOST in a
// native confirmation. No boolean from a plugin grants access.
func (a *App) PluginProcessReview(pluginID string, request processes.Request) (bool, error) {
	return a.confirmProcessRequest(pluginID, request, true)
}

// PluginProcessConfirm is an optional user confirmation, not a permission grant.
func (a *App) PluginProcessConfirm(pluginID string, request processes.Request) (bool, error) {
	return a.confirmProcessRequest(pluginID, request, false)
}

func (a *App) confirmProcessRequest(pluginID string, request processes.Request, remember bool) (bool, error) {
	a.processReviewMu.Lock()
	defer a.processReviewMu.Unlock()
	a.processMu.Lock()
	r, owner, err := a.processAccessLocked(pluginID, request)
	epoch := a.processEpoch
	a.processMu.Unlock()
	if err != nil {
		return false, err
	}
	title, message, allow, deny := "Run this action?", "Review the program and arguments before running.\n\nPlugin: ", "Yes", "No"
	if a.appSettings != nil && a.appSettings.Get().Language == "ru" {
		title, message, allow, deny = "Запустить действие?", "Проверьте программу и аргументы перед запуском.\n\nПлагин: ", "Yes", "No"
	}
	confirm := a.confirmProcess
	if confirm == nil {
		confirm = runtime.MessageDialog
	}
	answer, err := confirm(a.ctx, runtime.MessageDialogOptions{Type: runtime.QuestionDialog, Title: title, Message: message + pluginID + "\n\n" + processConfirmationPreview(r, a.appSettings != nil && a.appSettings.Get().Language == "ru"), Buttons: []string{deny, allow}, DefaultButton: deny, CancelButton: deny})
	if err != nil {
		return false, errors.New("review_failed")
	}
	if answer != allow {
		return false, nil
	}
	a.processMu.Lock()
	defer a.processMu.Unlock()
	current, currentOwner, err := a.processAccessLocked(pluginID, request)
	if err != nil {
		return false, err
	}
	if epoch != a.processEpoch || owner != currentOwner || processes.Preview(current) != processes.Preview(r) {
		return false, errors.New("review_expired")
	}
	if remember {
		if err := a.processes.SetApproval(owner, r, true); err != nil {
			return false, err
		}
	}
	return true, nil
}

func (a *App) PluginProcessRevoke(pluginID string, request processes.Request) error {
	a.processMu.Lock()
	defer a.processMu.Unlock()
	r, owner, err := a.processAccessLocked(pluginID, request)
	if err != nil {
		return err
	}
	return a.processes.SetApproval(owner, r, false)
}

func (a *App) PluginProcessRun(pluginID, operationID string, request processes.Request) (processes.Result, error) {
	a.processMu.Lock()
	r, owner, err := a.processAccessLocked(pluginID, request)
	if err != nil {
		a.processMu.Unlock()
		return processes.Result{}, err
	}
	if r.Kind == "url" {
		defer a.processMu.Unlock()
		if err := a.externalOpenService().OpenPath(r.URL); err != nil {
			return processes.Result{}, errors.New("open_failed")
		}
		return processes.Result{Started: true, ExitCode: -1}, nil
	}
	ctx := a.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	run, err := a.processes.Start(ctx, owner, operationID, r)
	a.processMu.Unlock()
	if err != nil {
		return processes.Result{}, err
	}
	return run.Wait(), nil
}

func (a *App) PluginProcessCancel(pluginID, operationID string) error {
	a.processMu.Lock()
	defer a.processMu.Unlock()
	if _, err := a.requirePluginAccess(pluginID, "process.spawn"); err != nil {
		return err
	}
	if _, err := a.requirePluginCapabilityAccess(pluginID, processes.Capability); err != nil {
		return err
	}
	if err := a.requireVault(); err != nil {
		return err
	}
	if a.processes != nil {
		a.processes.Cancel(a.processOwner(pluginID), operationID)
	}
	return nil
}
func (a *App) stopPluginProcesses(owner string) {
	a.processMu.Lock()
	defer a.processMu.Unlock()
	a.processEpoch++
	if a.processes != nil {
		a.processes.Stop(owner)
	}
}

// Keep the user-facing confirmation focused on what will actually execute.
func processConfirmationPreview(r processes.Request, russian bool) string {
	if r.Kind == "url" {
		return "URL: " + r.URL
	}
	labels := []string{"Program", "Arguments", "Working directory"}
	if russian {
		labels = []string{"Программа", "Аргументы", "Рабочий каталог"}
	}
	args := make([]string, len(r.Args))
	for i, arg := range r.Args {
		args[i] = strconv.Quote(arg)
	}
	return labels[0] + ": " + r.Executable + "\n" + labels[1] + ": " + strings.Join(args, " ") + "\n" + labels[2] + ": " + r.Cwd
}
