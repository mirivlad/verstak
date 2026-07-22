package importservice

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/verstak/verstak-desktop/internal/core/workspacetree"
)

const (
	importRootName      = "Импортировано"
	freeSpaceSafetyRoom = int64(64 << 20)
)

func (service *Service) ApplyPlan(parentContext context.Context, pluginID, handle string, plan Plan) (ApplyResult, error) {
	service.applyMu.Lock()
	defer service.applyMu.Unlock()

	session, err := service.get(pluginID, handle)
	if err != nil {
		return ApplyResult{}, err
	}
	ctx, cancel := context.WithCancel(parentContext)
	defer cancel()
	service.setApplyState(handle, cancel, "validating")
	defer service.clearApplyState(handle)
	service.emitProgress(pluginID, Progress{SourceHandle: handle, Phase: "validating", Total: len(plan.Nodes), Cancellable: true, Message: "Проверка плана импорта"})
	if err := checkImportCancellation(ctx); err != nil {
		return ApplyResult{}, err
	}
	if err := session.source.verify(); err != nil {
		return ApplyResult{}, err
	}
	if err := checkImportCancellation(ctx); err != nil {
		return ApplyResult{}, err
	}
	validated, err := validatePlan(plan, handle, session.source, service.limits.maxTextBytes)
	if err != nil {
		return ApplyResult{}, err
	}
	available, err := availableDiskSpace(service.vaultDir)
	if err != nil {
		return ApplyResult{}, sourceError("disk-space-unavailable", "could not inspect destination")
	}
	required := validated.requiredBytes
	if required > int64(^uint64(0)>>1)-freeSpaceSafetyRoom || available <= uint64(required+freeSpaceSafetyRoom) {
		return ApplyResult{}, sourceError("insufficient-disk-space", "not enough space for import")
	}

	if err := checkImportCancellation(ctx); err != nil {
		return ApplyResult{}, err
	}
	if err := service.recoverLocked(); err != nil {
		return ApplyResult{}, err
	}

	importRootExists, err := compatibleImportRoot(service.vaultDir)
	if err != nil {
		return ApplyResult{}, err
	}
	runName, err := availableRunName(service.vaultDir, validated.runName, importRootExists)
	if err != nil {
		return ApplyResult{}, err
	}
	validated.result.RunPath = filepath.ToSlash(filepath.Join(importRootName, runName))

	txnDir := filepath.Join(service.vaultDir, ".verstak", "import-staging", handle)
	if err := os.MkdirAll(filepath.Dir(txnDir), 0o700); err != nil {
		return ApplyResult{}, err
	}
	if err := os.Mkdir(txnDir, 0o700); err != nil {
		return ApplyResult{}, sourceError("import-transaction-exists", "staging session already exists")
	}
	cleanupTransaction := true
	defer func() {
		if cleanupTransaction {
			_ = os.RemoveAll(txnDir)
		}
	}()

	treeDir := filepath.Join(txnDir, "tree")
	registryDir := filepath.Join(txnDir, "registry")
	if err := os.MkdirAll(registryDir, 0o700); err != nil {
		return ApplyResult{}, err
	}
	var runDir string
	var publishSource string
	journal := transactionJournal{Version: 1, TransactionID: uuid.NewString(), Status: transactionStaged, CreatedImportRoot: !importRootExists}
	if importRootExists {
		runDir = filepath.Join(treeDir, runName)
		publishSource = runDir
		journal.PublishedRoot = filepath.ToSlash(filepath.Join(importRootName, runName))
	} else {
		stagedImportRoot := filepath.Join(treeDir, importRootName)
		if err := os.MkdirAll(stagedImportRoot, 0o755); err != nil {
			return ApplyResult{}, err
		}
		if err := workspacetree.WriteFolderMarker(stagedImportRoot, uuid.NewString()); err != nil {
			return ApplyResult{}, err
		}
		runDir = filepath.Join(stagedImportRoot, runName)
		publishSource = stagedImportRoot
		journal.PublishedRoot = importRootName
	}
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		return ApplyResult{}, err
	}
	if err := workspacetree.WriteFolderMarker(runDir, uuid.NewString()); err != nil {
		return ApplyResult{}, err
	}
	if err := writeImportOwnershipMarker(publishSource, journal.TransactionID); err != nil {
		return ApplyResult{}, err
	}

	service.setApplyPhase(handle, "staging")
	service.emitProgress(pluginID, Progress{SourceHandle: handle, Phase: "staging", Total: len(validated.nodes), Cancellable: true, Message: "Подготовка данных"})
	registryPromotions := make([]registryPromotion, 0, validated.result.Workspaces)
	for index, node := range validated.nodes {
		if err := checkImportCancellation(ctx); err != nil {
			return ApplyResult{}, err
		}
		target := filepath.Join(runDir, filepath.FromSlash(node.relPath))
		switch node.Kind {
		case "folder":
			if err := os.MkdirAll(target, 0o755); err != nil {
				return ApplyResult{}, err
			}
			if err := workspacetree.WriteFolderMarker(target, uuid.NewString()); err != nil {
				return ApplyResult{}, err
			}
		case "workspace":
			if err := os.MkdirAll(target, 0o755); err != nil {
				return ApplyResult{}, err
			}
			prepared, err := workspacetree.PrepareImportedWorkspace(target, node.Name, node.TemplateID)
			if err != nil {
				return ApplyResult{}, err
			}
			stagedRegistry := filepath.Join(registryDir, "uuid-"+prepared.ID+".json")
			if err := os.WriteFile(stagedRegistry, prepared.RegistryJSON, 0o600); err != nil {
				return ApplyResult{}, err
			}
			targetRegistryRel := filepath.ToSlash(filepath.Join(".verstak", "workspaces", "uuid-"+prepared.ID+".json"))
			registryPromotions = append(registryPromotions, registryPromotion{source: stagedRegistry, target: filepath.Join(service.vaultDir, filepath.FromSlash(targetRegistryRel))})
			journal.RegistryPaths = append(journal.RegistryPaths, targetRegistryRel)
		case "note":
			if err := writePlannedNote(target, node.Text, node.ModifiedAt); err != nil {
				return ApplyResult{}, err
			}
		case "file":
			if err := copyPlannedFile(session.source, *node.entry, target, node.ModifiedAt); err != nil {
				return ApplyResult{}, err
			}
		case "skip":
		}
		service.emitProgress(pluginID, Progress{SourceHandle: handle, Phase: "staging", Completed: index + 1, Total: len(validated.nodes), Cancellable: true, Message: "Подготовка данных"})
	}
	if err := checkImportCancellation(ctx); err != nil {
		return ApplyResult{}, err
	}
	for _, promotion := range registryPromotions {
		if _, err := os.Lstat(promotion.target); err == nil {
			return ApplyResult{}, sourceError("import-registry-conflict", "workspace metadata destination already exists")
		} else if !os.IsNotExist(err) {
			return ApplyResult{}, err
		}
	}
	if err := writeTransactionJournal(txnDir, journal); err != nil {
		return ApplyResult{}, err
	}

	service.setApplyPhase(handle, "publishing")
	service.emitProgress(pluginID, Progress{SourceHandle: handle, Phase: "publishing", Completed: len(validated.nodes), Total: len(validated.nodes), Cancellable: false, Message: "Публикация импорта"})
	journal.Status = transactionPublishing
	if err := writeTransactionJournal(txnDir, journal); err != nil {
		return ApplyResult{}, err
	}
	publishedTarget := filepath.Join(service.vaultDir, filepath.FromSlash(journal.PublishedRoot))
	if err := os.MkdirAll(filepath.Dir(publishedTarget), 0o755); err != nil {
		return ApplyResult{}, err
	}
	if _, err := os.Lstat(publishedTarget); err == nil {
		return ApplyResult{}, sourceError("import-target-conflict", "destination appeared during import")
	} else if !os.IsNotExist(err) {
		return ApplyResult{}, err
	}
	if err := os.Rename(publishSource, publishedTarget); err != nil {
		return ApplyResult{}, err
	}
	published := true
	promoted := make([]string, 0, len(registryPromotions))
	rollback := func() {
		for _, target := range promoted {
			_ = os.Remove(target)
		}
		if published {
			_ = os.RemoveAll(publishedTarget)
		}
	}
	for _, promotion := range registryPromotions {
		if err := service.promoteRegistry(promotion.source, promotion.target); err != nil {
			rollback()
			return ApplyResult{}, sourceError("import-publication-failed", "could not publish workspace metadata")
		}
		promoted = append(promoted, promotion.target)
	}
	journal.Status = transactionCommitted
	if err := writeTransactionJournal(txnDir, journal); err != nil {
		rollback()
		return ApplyResult{}, err
	}
	if err := os.Remove(filepath.Join(publishedTarget, ".verstak", "import-transaction.json")); err != nil && !os.IsNotExist(err) {
		validated.result.Warnings = append(validated.result.Warnings, "Импорт завершён, но временную метку транзакции не удалось удалить")
	}

	service.setApplyPhase(handle, "refreshing")
	service.emitProgress(pluginID, Progress{SourceHandle: handle, Phase: "refreshing", Completed: len(validated.nodes), Total: len(validated.nodes), Cancellable: false, Message: "Обновление дерева"})
	if service.refresh != nil {
		if err := service.refresh(); err != nil {
			validated.result.Warnings = append(validated.result.Warnings, "Импорт завершён, но дерево Верстака не удалось обновить автоматически")
		}
	}
	if err := os.RemoveAll(txnDir); err == nil {
		cleanupTransaction = false
	}
	return validated.result, nil
}

type registryPromotion struct {
	source string
	target string
}

func compatibleImportRoot(vaultDir string) (bool, error) {
	root := filepath.Join(vaultDir, importRootName)
	info, err := os.Lstat(root)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return false, sourceError("import-root-conflict", "destination is not an organizational folder")
	}
	if _, err := os.Lstat(filepath.Join(root, ".verstak", "workspace.json")); err == nil {
		return false, sourceError("import-root-conflict", "destination is a workspace")
	}
	data, err := os.ReadFile(filepath.Join(root, ".verstak", "folder.json"))
	if err != nil {
		return false, sourceError("import-root-conflict", "destination is not managed by Verstak")
	}
	marker, err := workspacetree.ParseFolderMarker(data)
	if err != nil || marker.SchemaVersion != 1 {
		return false, sourceError("import-root-conflict", "destination marker is invalid")
	}
	if _, err := uuid.Parse(marker.FolderID); err != nil {
		return false, sourceError("import-root-conflict", "destination marker is invalid")
	}
	return true, nil
}

func availableRunName(vaultDir, requested string, importRootExists bool) (string, error) {
	if !importRootExists {
		return requested, nil
	}
	root := filepath.Join(vaultDir, importRootName)
	entries, err := os.ReadDir(root)
	if err != nil {
		return "", err
	}
	existing := make(map[string]bool, len(entries))
	for _, entry := range entries {
		existing[collisionKey(entry.Name())] = true
	}
	for suffix := 1; suffix < 10_000; suffix++ {
		candidate := requested
		if suffix > 1 {
			candidate = fmt.Sprintf("%s (%d)", requested, suffix)
		}
		if !existing[collisionKey(candidate)] {
			return candidate, nil
		}
	}
	return "", sourceError("import-target-conflict", "could not allocate a unique import name")
}

func writePlannedNote(target, text, modifiedAt string) error {
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}
	file, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	_, writeErr := io.WriteString(file, text)
	closeErr := file.Close()
	if writeErr != nil {
		return writeErr
	}
	if closeErr != nil {
		return closeErr
	}
	return applyModifiedTime(target, modifiedAt)
}

func copyPlannedFile(source indexedSource, entry sourceEntry, target, modifiedAt string) error {
	reader, err := source.open(entry.ID)
	if err != nil {
		return err
	}
	defer reader.Close()
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}
	file, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	written, copyErr := io.Copy(file, io.LimitReader(reader, entry.Size+1))
	closeErr := file.Close()
	if copyErr != nil {
		return copyErr
	}
	if closeErr != nil {
		return closeErr
	}
	if written != entry.Size {
		return sourceError("source-changed", "source entry size changed")
	}
	return applyModifiedTime(target, modifiedAt)
}

func applyModifiedTime(target, value string) error {
	if value == "" {
		return nil
	}
	modifiedAt, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return err
	}
	return os.Chtimes(target, modifiedAt, modifiedAt)
}

func promoteRegistryFile(sourcePath, targetPath string) error {
	data, err := os.ReadFile(sourcePath)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
		return err
	}
	target, err := os.OpenFile(targetPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	_, writeErr := target.Write(data)
	closeErr := target.Close()
	if writeErr != nil {
		_ = os.Remove(targetPath)
		return writeErr
	}
	if closeErr != nil {
		_ = os.Remove(targetPath)
		return closeErr
	}
	return nil
}

func checkImportCancellation(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return sourceError("import-cancelled", "import was cancelled")
		}
		return err
	}
	return nil
}

func (service *Service) setApplyState(handle string, cancel context.CancelFunc, phase string) {
	service.mu.Lock()
	defer service.mu.Unlock()
	if session := service.sessions[handle]; session != nil {
		session.cancel = cancel
		session.phase = phase
		if session.closeRequested {
			cancel()
		}
	}
}

func (service *Service) setApplyPhase(handle, phase string) {
	service.mu.Lock()
	defer service.mu.Unlock()
	if session := service.sessions[handle]; session != nil {
		session.phase = phase
	}
}

func (service *Service) clearApplyState(handle string) {
	service.mu.Lock()
	defer service.mu.Unlock()
	if session := service.sessions[handle]; session != nil {
		session.cancel = nil
		session.phase = ""
	}
}

func (service *Service) emitProgress(pluginID string, progress Progress) {
	if service.onProgress != nil {
		service.onProgress(pluginID, progress)
	}
}

func hasPathPrefix(value, prefix string) bool {
	return value == prefix || strings.HasPrefix(value, prefix+"/")
}
