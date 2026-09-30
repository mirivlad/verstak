package api

import (
	"context"
	"fmt"
	"os"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"github.com/verstak/verstak-desktop/internal/core/importservice"
	"github.com/verstak/verstak-desktop/internal/core/vault"
)

func (a *App) requireImportAccess(pluginID string, apply bool) error {
	p, err := a.requirePluginCapabilityAccess(pluginID, "verstak/core/import/v1")
	if err != nil {
		return err
	}
	if !hasString(p.Manifest.Permissions, "imports.readExternal") {
		return fmt.Errorf("plugin %q lacks required permission %q", pluginID, "imports.readExternal")
	}
	if apply && !hasString(p.Manifest.Permissions, "imports.apply") {
		return fmt.Errorf("plugin %q lacks required permission %q", pluginID, "imports.apply")
	}
	return nil
}

func (a *App) rebindImportService() error {
	if a == nil {
		return nil
	}
	a.closeImportService()
	if a.vault == nil || a.vault.GetVaultStatus() != vault.StatusOpen {
		return nil
	}
	service := importservice.New(a.vault.GetVaultPath(), importservice.Options{
		OnProgress: a.emitImportProgress,
		Refresh:    a.refreshImportedTree,
	})
	if err := service.Recover(); err != nil {
		return err
	}
	a.importsMu.Lock()
	a.imports = service
	a.importsMu.Unlock()
	return nil
}

func (a *App) currentImportService() *importservice.Service {
	if a == nil {
		return nil
	}
	a.importsMu.RLock()
	defer a.importsMu.RUnlock()
	return a.imports
}

func (a *App) closeImportService() {
	if a == nil {
		return
	}
	a.importsMu.Lock()
	service := a.imports
	a.imports = nil
	a.importsMu.Unlock()
	if service != nil {
		service.CloseAll()
	}
}

func (a *App) refreshImportedTree() error {
	if a.treeV2 != nil {
		if err := a.treeV2.RescanWorkspaceTree(); err != nil {
			return err
		}
	}
	if a.fileWatcher != nil {
		return a.fileWatcher.RefreshBaseline()
	}
	return nil
}

func (a *App) emitImportProgress(pluginID string, progress importservice.Progress) {
	if a == nil || a.ctx == nil {
		return
	}
	emitFrontendEvent(a.ctx, "verstak:import-progress", map[string]any{
		"pluginId":     pluginID,
		"sourceHandle": progress.SourceHandle,
		"phase":        progress.Phase,
		"completed":    progress.Completed,
		"total":        progress.Total,
		"cancellable":  progress.Cancellable,
		"message":      progress.Message,
	})
}

func (a *App) PluginSelectImportDirectory(pluginID string) (importservice.SourceSession, string) {
	if err := a.requireImportAccess(pluginID, false); err != nil {
		return importservice.SourceSession{}, err.Error()
	}
	service := a.currentImportService()
	if service == nil {
		return importservice.SourceSession{}, "import service not initialized"
	}
	dialog := a.selectImportDirectory
	if dialog == nil {
		dialog = runtime.OpenDirectoryDialog
	}
	home, _ := os.UserHomeDir()
	selected, err := dialog(a.ctx, runtime.OpenDialogOptions{Title: "Выберите папку с данными для импорта", DefaultDirectory: home})
	if err != nil {
		return importservice.SourceSession{}, err.Error()
	}
	if selected == "" {
		return importservice.SourceSession{}, ""
	}
	session, err := service.OpenDirectory(pluginID, selected)
	if err != nil {
		return importservice.SourceSession{}, err.Error()
	}
	return session, ""
}

func (a *App) PluginSelectImportArchive(pluginID string) (importservice.SourceSession, string) {
	if err := a.requireImportAccess(pluginID, false); err != nil {
		return importservice.SourceSession{}, err.Error()
	}
	service := a.currentImportService()
	if service == nil {
		return importservice.SourceSession{}, "import service not initialized"
	}
	dialog := a.selectImportArchive
	if dialog == nil {
		dialog = runtime.OpenFileDialog
	}
	home, _ := os.UserHomeDir()
	selected, err := dialog(a.ctx, runtime.OpenDialogOptions{
		Title:            "Выберите архив для импорта",
		DefaultDirectory: home,
		Filters:          []runtime.FileFilter{{DisplayName: "Архивы ZIP и TAR", Pattern: "*.zip;*.tar;*.tar.gz;*.tgz"}},
	})
	if err != nil {
		return importservice.SourceSession{}, err.Error()
	}
	if selected == "" {
		return importservice.SourceSession{}, ""
	}
	session, err := service.OpenArchive(pluginID, selected)
	if err != nil {
		return importservice.SourceSession{}, err.Error()
	}
	return session, ""
}

func (a *App) PluginListImportEntries(pluginID, sourceHandle, cursor string) (importservice.EntryPage, string) {
	if err := a.requireImportAccess(pluginID, false); err != nil {
		return importservice.EntryPage{}, err.Error()
	}
	service := a.currentImportService()
	if service == nil {
		return importservice.EntryPage{}, "import service not initialized"
	}
	page, err := service.ListEntries(pluginID, sourceHandle, cursor)
	if err != nil {
		return importservice.EntryPage{}, err.Error()
	}
	return page, ""
}

func (a *App) PluginReadImportText(pluginID, sourceHandle, entryID string) (string, string) {
	if err := a.requireImportAccess(pluginID, false); err != nil {
		return "", err.Error()
	}
	service := a.currentImportService()
	if service == nil {
		return "", "import service not initialized"
	}
	text, err := service.ReadText(pluginID, sourceHandle, entryID)
	if err != nil {
		return "", err.Error()
	}
	return text, ""
}

func (a *App) PluginApplyImportPlan(pluginID, sourceHandle string, plan importservice.Plan) (importservice.ApplyResult, string) {
	if err := a.requireImportAccess(pluginID, true); err != nil {
		return importservice.ApplyResult{}, err.Error()
	}
	service := a.currentImportService()
	if service == nil {
		return importservice.ApplyResult{}, "import service not initialized"
	}
	ctx := a.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	result, applyErr := service.ApplyPlan(ctx, pluginID, sourceHandle, plan)
	if applyErr != nil {
		return importservice.ApplyResult{}, applyErr.Error()
	}
	a.scheduleSnapshotScan()
	return result, ""
}

func (a *App) PluginCancelImport(pluginID, sourceHandle string) string {
	if err := a.requireImportAccess(pluginID, false); err != nil {
		return err.Error()
	}
	service := a.currentImportService()
	if service == nil {
		return "import service not initialized"
	}
	if err := service.Cancel(pluginID, sourceHandle); err != nil {
		return err.Error()
	}
	return ""
}

func (a *App) PluginCloseImportSource(pluginID, sourceHandle string) string {
	if err := a.requireImportAccess(pluginID, false); err != nil {
		return err.Error()
	}
	service := a.currentImportService()
	if service == nil {
		return "import service not initialized"
	}
	if err := service.Close(pluginID, sourceHandle); err != nil {
		return err.Error()
	}
	return ""
}
