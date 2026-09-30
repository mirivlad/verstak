package api

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"github.com/verstak/verstak-desktop/internal/core/filewatcher"
	"github.com/verstak/verstak-desktop/internal/core/vault"
	"github.com/verstak/verstak-desktop/internal/core/workspacetree"
	"github.com/verstak/verstak-desktop/internal/shell/debug"
)

// GetVaultStatus returns the current vault status, path, and vault ID.
func (a *App) GetVaultStatus() map[string]string {
	status := "not-created"
	path := ""
	vaultID := ""

	if a.vault != nil {
		status = string(a.vault.GetVaultStatus())
		path = a.vault.GetVaultPath()
		meta := a.vault.GetVaultMeta()
		if meta != nil {
			vaultID = meta.VaultID
		}
	}

	if a.debug {
		debug.Logf("[api] GetVaultStatus: status=%s path=%s vaultId=%s", status, path, vaultID)
	}

	return map[string]string{
		"status":  status,
		"path":    path,
		"vaultId": vaultID,
	}
}

// CreateVault creates a new vault at the given path.
func (a *App) CreateVault(path string) error {
	if a.vault == nil {
		return fmt.Errorf("vault service not initialized")
	}
	a.closeImportService()
	if err := a.vault.CreateVault(path); err != nil {
		return err
	}
	a.rebindSyncService()
	if err := a.rebindImportService(); err != nil {
		return fmt.Errorf("recover import transactions: %w", err)
	}
	a.secretsSession = nil
	a.startFileWatcherForOpenVault()
	return nil
}

// OpenVault opens an existing vault at the given path.
func (a *App) OpenVault(path string) error {
	if a.vault == nil {
		return fmt.Errorf("vault service not initialized")
	}
	a.closeImportService()
	if err := a.vault.OpenVault(path); err != nil {
		return err
	}
	a.rebindSyncService()
	if err := a.rebindImportService(); err != nil {
		return fmt.Errorf("recover import transactions: %w", err)
	}
	a.secretsSession = nil
	a.startFileWatcherForOpenVault()
	return nil
}

// CloseVault closes the current vault.
func (a *App) CloseVault() error {
	if a.vault == nil {
		return fmt.Errorf("vault service not initialized")
	}
	if a.fileWatcher != nil {
		a.fileWatcher.Stop()
	}
	a.closeImportService()
	a.stopScheduledSnapshotScan()
	a.vault.CloseVault()
	a.syncSvc = nil
	a.secretsSession = nil
	return nil
}

// SetCurrentVault sets the current vault path in app settings and re-opens the vault.
// Loads workspace and registers vault + workspace capabilities.
func (a *App) SetCurrentVault(path string) string {
	if a.appSettings == nil {
		return "app settings not initialized"
	}
	if a.vault == nil {
		return "vault service not initialized"
	}
	a.closeImportService()
	// Try to open the vault first
	if err := a.vault.OpenVault(path); err != nil {
		return fmt.Sprintf("failed to open vault: %v", err)
	}
	a.secretsSession = nil
	// Save the actual vault path (normalized by OpenVault, includes VerstakVault/)
	vaultPath := a.vault.GetVaultPath()
	a.rebindSyncService()
	if err := a.rebindImportService(); err != nil {
		return fmt.Sprintf("failed to recover import transactions: %v", err)
	}
	if err := a.appSettings.SetCurrentVault(vaultPath); err != nil {
		return fmt.Sprintf("failed to save app settings: %v", err)
	}
	// Load plugin state for the vault
	if a.pluginState != nil {
		if err := a.pluginState.Load(); err != nil {
			log.Printf("[api] SetCurrentVault: warning loading plugin state: %v", err)
		}
	}
	// Stop old treeV2 before creating a new one (vault switch).
	if a.treeV2 != nil {
		a.treeV2.Stop()
	}
	// Initialize V2 workspace tree.
	a.treeV2 = workspacetree.NewService(vaultPath, a.eventBus)
	if err := a.treeV2.Initialize(); err != nil {
		log.Printf("[api] SetCurrentVault: warning initializing workspace tree v2: %v", err)
	}
	if err := a.runDealMigration(context.Background()); err != nil {
		a.treeV2.Stop()
		a.treeV2 = nil
		return fmt.Sprintf("failed to migrate legacy Deal data: %v", err)
	}
	a.treeV2.StartRescanLoop()
	// Register vault capability
	if err := a.capRegistry.Register("verstak-desktop", []string{"verstak/core/vault/v1"}); err != nil {
		log.Printf("[api] SetCurrentVault: failed to register vault capability: %v", err)
	}
	// Register workspace capability
	if a.treeV2 != nil {
		if err := a.capRegistry.Register("verstak-desktop", []string{"verstak/core/workspace/v1"}); err != nil {
			log.Printf("[api] SetCurrentVault: failed to register workspace capability: %v", err)
		}
	}
	a.startFileWatcherForOpenVault()
	return ""
}

func (a *App) startFileWatcherForOpenVault() {
	if a == nil || a.vault == nil || a.eventBus == nil {
		return
	}
	if a.vault.GetVaultStatus() != vault.StatusOpen {
		return
	}
	if a.currentImportService() == nil {
		if err := a.rebindImportService(); err != nil {
			log.Printf("[api] import recovery failed: %v", err)
			return
		}
	}
	if a.fileWatcher == nil {
		a.fileWatcher = filewatcher.NewService(a.eventBus, 0)
	}

	// Initialize treeV2 if not already set (covers auto-open on startup).
	if a.treeV2 == nil {
		vaultPath := a.vault.GetVaultPath()
		a.treeV2 = workspacetree.NewService(vaultPath, a.eventBus)
		if err := a.treeV2.Initialize(); err != nil {
			log.Printf("[api] startFileWatcher: warning initializing tree v2: %v", err)
		}
		if err := a.runDealMigration(context.Background()); err != nil {
			log.Printf("[api] startFileWatcher: legacy Deal migration failed: %v", err)
			a.treeV2 = nil
			return
		}
		a.treeV2.StartRescanLoop()
	}

	// Wire structural change callback → tree reconciliation.
	if a.treeV2 != nil {
		a.fileWatcher.SetOnStructuralChange(func() {
			a.treeV2.OnFileChanged()
		})
		a.fileWatcher.SetWorkspaceResolver(func(relPath string) (string, string, bool) {
			return a.treeV2.ResolveWorkspaceForPath(relPath)
		})
		a.treeV2.SetWatcherBaselineRefresh(func() error {
			return a.fileWatcher.RefreshBaseline()
		})
	}

	// Keep sync scan callback for content changes.
	a.fileWatcher.SetOnChange(a.scheduleSnapshotScan)

	// Start performs its own initial scan to establish baseline.
	// Tree initialization (including marker adoption) is already complete.
	if err := a.fileWatcher.Start(a.vault.GetVaultPath()); err != nil {
		log.Printf("[api] file watcher start failed: %v", err)
	}
	if _, err := a.scanLocalChanges(); err != nil {
		log.Printf("[api] initial sync snapshot scan failed: %v", err)
	}
}

// SelectDirectory opens a native directory picker dialog.
// Returns the selected path or empty string if cancelled.
func (a *App) SelectDirectory() string {
	home, _ := os.UserHomeDir()

	selected, err := runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title:            "Select Vault Directory",
		DefaultDirectory: home,
	})
	if err != nil {
		log.Printf("[api] SelectDirectory: %v", err)
		return ""
	}
	return selected
}

// SelectVaultForOpen opens a directory picker for opening an existing vault.
func (a *App) SelectVaultForOpen() string {
	home, _ := os.UserHomeDir()

	selected, err := runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title:            "Open Existing Vault",
		DefaultDirectory: home,
	})
	if err != nil {
		log.Printf("[api] SelectVaultForOpen: %v", err)
		return ""
	}
	return selected
}
