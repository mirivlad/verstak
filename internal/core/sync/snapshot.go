package sync

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/google/uuid"
	corefiles "github.com/verstak/verstak-desktop/internal/core/files"
)

const snapshotVersion = 2

// maxOperationFileBytes is an explicit desktop-side safety ceiling. Binary
// content is streamed through the Blob API rather than embedded in operations,
// so it is intentionally higher than the plugin Files API read limit.
const maxOperationFileBytes int64 = 256 * 1024 * 1024

// BlobCachePath is core-private durable staging for a local operation's
// immutable binary content. It remains excluded from ordinary file sync.
func BlobCachePath(vaultRoot, hash string) string {
	return filepath.Join(vaultRoot, ".verstak", "sync", "blobs", hash)
}

// Snapshot is the durable local view of the synchronizable part of a vault.
type Snapshot struct {
	Version               int                          `json:"version"`
	Entries               map[string]SnapshotEntry     `json:"entries"`
	Workspaces            map[string]WorkspaceSnapshot `json:"workspaces,omitempty"`
	TrashedWorkspaces     map[string]WorkspaceSnapshot `json:"trashedWorkspaces,omitempty"`
	Folders               map[string]FolderSnapshot    `json:"folders,omitempty"`
	TrashedFolders        map[string]FolderSnapshot    `json:"trashedFolders,omitempty"`
	WorkspacesInitialized bool                         `json:"workspacesInitialized,omitempty"`
	Unresolved            map[string]string            `json:"unresolved,omitempty"`
}

// SnapshotEntry stores only stable filesystem facts needed for reconciliation.
type SnapshotEntry struct {
	Path       string `json:"path"`
	Type       string `json:"type"`
	Size       int64  `json:"size"`
	ModifiedAt string `json:"modifiedAt"`
	Hash       string `json:"hash,omitempty"`
}

// WorkspaceSnapshot keeps the core-owned identity and creation metadata of a
// workspace. The marker itself remains excluded from normal file sync.
type WorkspaceSnapshot struct {
	WorkspaceID string                   `json:"workspaceId"`
	Path        string                   `json:"path"`
	Metadata    json.RawMessage          `json:"metadata,omitempty"`
	Entries     map[string]SnapshotEntry `json:"entries,omitempty"`
}

// FolderSnapshot records an organizational folder identity with enough
// information to restore its full subtree on a clean device.
type FolderSnapshot struct {
	FolderID     string                   `json:"folderId"`
	Path         string                   `json:"path"`
	Name         string                   `json:"name,omitempty"`
	Entries      map[string]SnapshotEntry `json:"entries,omitempty"`
	FolderIDs    []string                 `json:"folderIds,omitempty"`
	WorkspaceIDs []string                 `json:"workspaceIds,omitempty"`
}

type scanJournal struct {
	Snapshot Snapshot `json:"snapshot"`
	Ops      []Op     `json:"ops"`
}

type scannedVault struct {
	Entries    map[string]SnapshotEntry
	Workspaces map[string]WorkspaceSnapshot
	Folders    map[string]FolderSnapshot
	Unresolved map[string]string
}

// LoadSnapshot returns the current durable scanner snapshot. A missing
// snapshot is represented by an empty snapshot, which is useful to callers
// that only need to inspect it.
func (s *Service) LoadSnapshot() (Snapshot, error) {
	snapshot, exists, err := s.loadSnapshot()
	if err != nil {
		return Snapshot{}, err
	}
	if !exists {
		return newSnapshot(), nil
	}
	return snapshot, nil
}

// ScanAndRecord scans the whole synchronizable vault and records exactly the
// detected local changes. On its first run it writes a baseline and deliberately
// produces no operations; bootstrap decides what can safely be published.
func (s *Service) ScanAndRecord() ([]string, error) {
	if err := s.recoverScanJournal(); err != nil {
		return nil, err
	}
	previous, exists, err := s.loadSnapshot()
	if err != nil {
		return nil, err
	}
	current, warnings, err := scanVault(s.vaultRoot, previous)
	if err != nil {
		return nil, err
	}
	next := snapshotFromScan(current, previous, exists)
	if !exists {
		if err := s.saveSnapshot(next); err != nil {
			return nil, err
		}
		return warnings, nil
	}

	ops, next, err := diffSnapshots(previous, next, s.deviceID, s.vaultRoot)
	if err != nil {
		return nil, err
	}
	if len(ops) == 0 {
		if err := s.saveSnapshot(next); err != nil {
			return nil, err
		}
		return warnings, nil
	}
	if err := s.commitScanTransaction(next, ops); err != nil {
		return nil, err
	}
	return warnings, nil
}

// RecordBootstrapOps records creates for a pre-pull local snapshot. It is used
// only after a successful initial pull, so an empty new vault never turns into
// remote delete operations. Callers may pass the snapshot captured before the
// pull; remote-only entries added during reconciliation are therefore not
// reflected back to the server as local creates.
func (s *Service) RecordBootstrapOps(initial Snapshot) error {
	if err := s.recoverScanJournal(); err != nil {
		return err
	}
	current, exists, err := s.loadSnapshot()
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("bootstrap requires an initial snapshot")
	}
	ops, _, err := diffSnapshots(newSnapshot(), initial, s.deviceID, s.vaultRoot)
	if err != nil {
		return err
	}
	existing, err := s.GetUnpushedOps()
	if err != nil {
		return err
	}
	existingCreates := make(map[string]bool, len(existing))
	for _, op := range existing {
		if op.OpType == OpCreate {
			existingCreates[op.EntityType+"\x00"+op.EntityID] = true
		}
	}
	filtered := ops[:0]
	for _, op := range ops {
		if op.OpType == OpCreate && existingCreates[op.EntityType+"\x00"+op.EntityID] {
			continue
		}
		filtered = append(filtered, op)
	}
	ops = filtered
	if len(ops) == 0 {
		return nil
	}
	return s.commitScanTransaction(current, ops)
}

// RebaseSnapshot accepts filesystem changes that were applied from a remote
// operation without producing any outgoing operation for them.
func (s *Service) RebaseSnapshot() ([]string, error) {
	if err := s.recoverScanJournal(); err != nil {
		return nil, err
	}
	previous, exists, err := s.loadSnapshot()
	if err != nil {
		return nil, err
	}
	current, warnings, err := scanVault(s.vaultRoot, previous)
	if err != nil {
		return nil, err
	}
	next := snapshotFromScan(current, previous, exists)
	acceptWorkspaceLifecycle(previous, &next)
	if err := s.saveSnapshot(next); err != nil {
		return nil, err
	}
	return warnings, nil
}

func newSnapshot() Snapshot {
	return Snapshot{
		Version:               snapshotVersion,
		Entries:               make(map[string]SnapshotEntry),
		Workspaces:            make(map[string]WorkspaceSnapshot),
		TrashedWorkspaces:     make(map[string]WorkspaceSnapshot),
		Folders:               make(map[string]FolderSnapshot),
		TrashedFolders:        make(map[string]FolderSnapshot),
		WorkspacesInitialized: true,
		Unresolved:            make(map[string]string),
	}
}

func (s *Service) loadSnapshot() (Snapshot, bool, error) {
	data, err := os.ReadFile(s.snapshotPath())
	if err != nil {
		if os.IsNotExist(err) {
			return Snapshot{}, false, nil
		}
		return Snapshot{}, false, fmt.Errorf("read snapshot: %w", err)
	}
	var snapshot Snapshot
	if err := json.Unmarshal(data, &snapshot); err != nil {
		return Snapshot{}, false, fmt.Errorf("parse snapshot: %w", err)
	}
	if snapshot.Version != snapshotVersion {
		return Snapshot{}, false, fmt.Errorf("unsupported snapshot version: %d", snapshot.Version)
	}
	if snapshot.Entries == nil {
		snapshot.Entries = make(map[string]SnapshotEntry)
	}
	if snapshot.Workspaces == nil {
		snapshot.Workspaces = make(map[string]WorkspaceSnapshot)
	}
	if snapshot.TrashedWorkspaces == nil {
		snapshot.TrashedWorkspaces = make(map[string]WorkspaceSnapshot)
	}
	if snapshot.Folders == nil {
		snapshot.Folders = make(map[string]FolderSnapshot)
	}
	if snapshot.TrashedFolders == nil {
		snapshot.TrashedFolders = make(map[string]FolderSnapshot)
	}
	if snapshot.Unresolved == nil {
		snapshot.Unresolved = make(map[string]string)
	}
	return snapshot, true, nil
}

func (s *Service) saveSnapshot(snapshot Snapshot) error {
	data, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal snapshot: %w", err)
	}
	return atomicWriteFile(s.snapshotPath(), data, 0o600)
}

func (s *Service) loadScanJournal() (scanJournal, bool, error) {
	data, err := os.ReadFile(s.scanJournalPath())
	if err != nil {
		if os.IsNotExist(err) {
			return scanJournal{}, false, nil
		}
		return scanJournal{}, false, fmt.Errorf("read scan journal: %w", err)
	}
	var journal scanJournal
	if err := json.Unmarshal(data, &journal); err != nil {
		return scanJournal{}, false, fmt.Errorf("parse scan journal: %w", err)
	}
	return journal, true, nil
}

func (s *Service) saveScanJournal(journal scanJournal) error {
	data, err := json.MarshalIndent(journal, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal scan journal: %w", err)
	}
	return atomicWriteFile(s.scanJournalPath(), data, 0o600)
}

func (s *Service) recoverScanJournal() error {
	journal, exists, err := s.loadScanJournal()
	if err != nil || !exists {
		return err
	}
	if err := s.recordOps(journal.Ops); err != nil {
		return err
	}
	if err := s.saveSnapshot(journal.Snapshot); err != nil {
		return err
	}
	if err := os.Remove(s.scanJournalPath()); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func (s *Service) commitScanTransaction(snapshot Snapshot, ops []Op) error {
	journal := scanJournal{Snapshot: snapshot, Ops: ops}
	if err := s.saveScanJournal(journal); err != nil {
		return err
	}
	if err := s.recordOps(ops); err != nil {
		return err
	}
	if err := s.saveSnapshot(snapshot); err != nil {
		return err
	}
	if err := os.Remove(s.scanJournalPath()); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func scanVault(root string, previous Snapshot) (scannedVault, []string, error) {
	result := scannedVault{
		Entries:    make(map[string]SnapshotEntry),
		Workspaces: make(map[string]WorkspaceSnapshot),
		Folders:    make(map[string]FolderSnapshot),
		Unresolved: make(map[string]string),
	}
	var warnings []string
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if path == root {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if excludedFromSync(rel) {
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if entry.Type()&os.ModeSymlink != 0 {
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if info.IsDir() {
			result.Entries[rel] = SnapshotEntry{
				Path:       rel,
				Type:       EntityFolder,
				ModifiedAt: info.ModTime().UTC().Format(time.RFC3339Nano),
			}
			return nil
		}
		if !info.Mode().IsRegular() {
			return nil
		}
		if info.Size() > maxOperationFileBytes {
			message := fmt.Sprintf("file-too-large: %s (%d bytes exceeds %d bytes)", rel, info.Size(), maxOperationFileBytes)
			result.Unresolved[rel] = message
			warnings = append(warnings, message)
			return nil
		}
		hash, err := sha256File(path)
		if err != nil {
			return fmt.Errorf("hash %s: %w", rel, err)
		}
		result.Entries[rel] = SnapshotEntry{
			Path:       rel,
			Type:       EntityFile,
			Size:       info.Size(),
			ModifiedAt: info.ModTime().UTC().Format(time.RFC3339Nano),
			Hash:       hash,
		}
		return nil
	})
	if err != nil {
		return scannedVault{}, nil, err
	}
	workspaces, folders, workspaceWarnings, err := scanTreeSnapshots(root, previous.Workspaces, previous.Folders)
	if err != nil {
		return scannedVault{}, nil, err
	}
	for workspaceID, workspace := range workspaces {
		result.Workspaces[workspaceID] = workspace
	}
	for folderID, folder := range folders {
		result.Folders[folderID] = folder
	}
	for _, warning := range workspaceWarnings {
		warnings = append(warnings, warning)
		if strings.HasPrefix(warning, "duplicate-workspace-id: ") {
			path := strings.TrimPrefix(warning, "duplicate-workspace-id: ")
			result.Unresolved[path] = warning
			removeEntriesUnder(result.Entries, path)
		}
	}
	sort.Strings(warnings)
	return result, warnings, nil
}

func scanTreeSnapshots(root string, preferredWS map[string]WorkspaceSnapshot, preferredF map[string]FolderSnapshot) (map[string]WorkspaceSnapshot, map[string]FolderSnapshot, []string, error) {
	wsCandidates := make(map[string][]WorkspaceSnapshot)
	fCandidates := make(map[string][]FolderSnapshot)
	var warnings []string

	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return nil
		}
		if path == root {
			return nil
		}
		if !entry.IsDir() || entry.Type()&os.ModeSymlink != 0 {
			return nil
		}
		rel, _ := filepath.Rel(root, path)
		rel = filepath.ToSlash(rel)
		if excludedFromSync(rel) {
			return filepath.SkipDir
		}

		// Check for workspace marker.
		wsMarkerPath := filepath.Join(path, ".verstak", "workspace.json")
		if data, err := os.ReadFile(wsMarkerPath); err == nil {
			var marker struct{ WorkspaceID string `json:"workspaceId"` }
			if json.Unmarshal(data, &marker) != nil || !isValidUUID(marker.WorkspaceID) {
				warnings = append(warnings, "invalid-workspace-id: "+rel)
				return filepath.SkipDir
			}
			wsCandidates[marker.WorkspaceID] = append(wsCandidates[marker.WorkspaceID], WorkspaceSnapshot{
				WorkspaceID: marker.WorkspaceID,
				Path:        rel,
			})
			return filepath.SkipDir // Stop at workspace.
		}

		// Check for folder marker.
		fMarkerPath := filepath.Join(path, ".verstak", "folder.json")
		if data, err := os.ReadFile(fMarkerPath); err == nil {
			var marker struct{ FolderID string `json:"folderId"` }
			if json.Unmarshal(data, &marker) != nil || !isValidUUID(marker.FolderID) {
				warnings = append(warnings, "invalid-folder-id: "+rel)
				return nil
			}
			fCandidates[marker.FolderID] = append(fCandidates[marker.FolderID], FolderSnapshot{
				FolderID: marker.FolderID,
				Path:     rel,
				Name:     filepath.Base(filepath.FromSlash(rel)),
			})
		}
		return nil
	})
	if err != nil {
		return nil, nil, nil, err
	}

	// Resolve workspaces.
	workspaces := make(map[string]WorkspaceSnapshot, len(wsCandidates))
	for wsID, choices := range wsCandidates {
		sort.Slice(choices, func(i, j int) bool { return choices[i].Path < choices[j].Path })
		selected := choices[0]
		if old, ok := preferredWS[wsID]; ok {
			for _, c := range choices {
				if c.Path == old.Path {
					selected = c
					break
				}
			}
		}
		workspaces[wsID] = selected
		for _, c := range choices {
			if c.Path != selected.Path {
				warnings = append(warnings, "duplicate-workspace-id: "+c.Path)
			}
		}
	}

	// Resolve folders.
	folders := make(map[string]FolderSnapshot, len(fCandidates))
	for fID, choices := range fCandidates {
		sort.Slice(choices, func(i, j int) bool { return choices[i].Path < choices[j].Path })
		selected := choices[0]
		if old, ok := preferredF[fID]; ok {
			for _, c := range choices {
				if c.Path == old.Path {
					selected = c
					break
				}
			}
		}
		folders[fID] = selected
		for _, c := range choices {
			if c.Path != selected.Path {
				warnings = append(warnings, "duplicate-folder-id: "+c.Path)
			}
		}
	}

	return workspaces, folders, warnings, nil
}

func isValidUUID(s string) bool {
	_, err := uuid.Parse(s)
	return err == nil
}

func readWorkspaceMetadataSnapshot(root, name string) (json.RawMessage, error) {
	path := filepath.Join(root, ".verstak", "workspaces", name, "metadata.json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read workspace metadata %s: %w", name, err)
	}
	if !json.Valid(data) {
		return nil, fmt.Errorf("invalid workspace metadata: %s", name)
	}
	return json.RawMessage(append([]byte(nil), data...)), nil
}

func excludedFromSync(rel string) bool {
	rel = filepath.ToSlash(rel)
	for _, segment := range strings.Split(rel, "/") {
		if strings.EqualFold(segment, ".verstak") {
			return true
		}
	}
	base := filepath.Base(rel)
	return strings.HasPrefix(base, ".verstak-write-") || strings.HasSuffix(base, ".tmp") || strings.HasSuffix(base, ".swp") || strings.HasSuffix(base, "~")
}

func sha256File(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func snapshotFromScan(current scannedVault, previous Snapshot, previousExists bool) Snapshot {
	next := newSnapshot()
	for path, entry := range current.Entries {
		next.Entries[path] = entry
	}
	for workspaceID, workspace := range current.Workspaces {
		next.Workspaces[workspaceID] = workspace
	}
	for folderID, folder := range current.Folders {
		next.Folders[folderID] = folder
	}
	for workspaceID, workspace := range previous.TrashedWorkspaces {
		next.TrashedWorkspaces[workspaceID] = workspace
	}
	for folderID, folder := range previous.TrashedFolders {
		next.TrashedFolders[folderID] = folder
	}
	for path, message := range current.Unresolved {
		next.Unresolved[path] = message
		copyEntriesUnder(next.Entries, previous.Entries, path)
	}
	if !previousExists {
		return next
	}
	for path, message := range previous.Unresolved {
		if _, supported := current.Entries[path]; supported {
			continue
		}
		if _, stillUnsupported := current.Unresolved[path]; stillUnsupported {
			continue
		}
		next.Unresolved[path] = "unresolved sync file disappeared before it could be synchronized: " + message
		copyEntriesUnder(next.Entries, previous.Entries, path)
	}
	return next
}

func diffSnapshots(previous, next Snapshot, deviceID, vaultRoot string) ([]Op, Snapshot, error) {
	folderOps, err := diffFolderSnapshots(&previous, &next, deviceID)
	if err != nil {
		return nil, next, err
	}
	workspaceOps, err := diffWorkspaceSnapshots(&previous, &next, deviceID)
	if err != nil {
		return nil, next, err
	}
	semanticOps := append(folderOps, workspaceOps...)
	var createsOrUpdates []Op
	var deletes []Op
	for path, entry := range next.Entries {
		old, existed := previous.Entries[path]
		if existed && entriesEqual(old, entry) {
			continue
		}
		if unresolvedPath(next.Unresolved, path) {
			continue
		}
		opType := OpCreate
		if existed {
			opType = OpUpdate
		}
		payload, err := payloadForEntry(vaultRoot, entry)
		if err != nil {
			return nil, next, err
		}
		createsOrUpdates = append(createsOrUpdates, newSnapshotOp(deviceID, entry.Type, path, opType, payload))
	}
	for path, old := range previous.Entries {
		if _, exists := next.Entries[path]; exists {
			continue
		}
		if unresolvedPath(previous.Unresolved, path) {
			continue
		}
		deletes = append(deletes, newSnapshotOp(deviceID, old.Type, path, OpDelete, map[string]string{"path": path}))
	}
	sort.Slice(createsOrUpdates, func(i, j int) bool {
		left, right := createsOrUpdates[i], createsOrUpdates[j]
		leftDepth, rightDepth := pathDepth(left.EntityID), pathDepth(right.EntityID)
		if leftDepth != rightDepth {
			return leftDepth < rightDepth
		}
		if left.EntityType != right.EntityType {
			return left.EntityType == EntityFolder
		}
		return left.EntityID < right.EntityID
	})
	sort.Slice(deletes, func(i, j int) bool {
		left, right := deletes[i], deletes[j]
		leftDepth, rightDepth := pathDepth(left.EntityID), pathDepth(right.EntityID)
		if leftDepth != rightDepth {
			return leftDepth > rightDepth
		}
		if left.EntityType != right.EntityType {
			return left.EntityType == EntityFile
		}
		return left.EntityID < right.EntityID
	})
	return append(semanticOps, append(createsOrUpdates, deletes...)...), next, nil
}

func diffFolderSnapshots(previous, next *Snapshot, deviceID string) ([]Op, error) {
	var ops []Op
	if !previous.WorkspacesInitialized {
		return ops, nil
	}
	// Detected deleted folders.
	for folderID, oldFolder := range previous.Folders {
		if _, active := next.Folders[folderID]; !active {
			// Moved to trash.
			next.TrashedFolders[folderID] = oldFolder
			ops = append(ops, newFolderSnapshotOp(deviceID, folderID, OpTrash, oldFolder, ""))
		}
	}
	// Detected new/restored folders.
	for folderID, currentFolder := range next.Folders {
		if oldFolder, wasActive := previous.Folders[folderID]; wasActive {
			if currentFolder.Path != oldFolder.Path {
				opType := OpMove
				if filepath.Base(filepath.FromSlash(currentFolder.Path)) != filepath.Base(filepath.FromSlash(oldFolder.Path)) {
					opType = OpRename
				}
				ops = append(ops, newFolderSnapshotOp(deviceID, folderID, opType, currentFolder, oldFolder.Path))
			}
			continue
		}
		if _, wasTrashed := previous.TrashedFolders[folderID]; wasTrashed {
			ops = append(ops, newFolderSnapshotOp(deviceID, folderID, OpRestore, currentFolder, ""))
			delete(next.TrashedFolders, folderID)
			continue
		}
		ops = append(ops, newFolderSnapshotOp(deviceID, folderID, OpCreate, currentFolder, ""))
	}
	// Permanently purged.
	for folderID := range previous.TrashedFolders {
		if _, stillTrashed := next.TrashedFolders[folderID]; !stillTrashed {
			ops = append(ops, newFolderSnapshotOp(deviceID, folderID, OpDelete, FolderSnapshot{FolderID: folderID}, ""))
		}
	}
	sort.Slice(ops, func(i, j int) bool {
		return pathDepth(ops[i].EntityID) < pathDepth(ops[j].EntityID)
	})
	return ops, nil
}

func newFolderSnapshotOp(deviceID, folderID, opType string, folder FolderSnapshot, previousPath string) Op {
	payload, _ := json.Marshal(map[string]interface{}{
		"folderId":     folderID,
		"path":         folder.Path,
		"previousPath": previousPath,
	})
	return newSnapshotOp(deviceID, EntityWorkspaceFolder, folderID, opType, json.RawMessage(payload))
}

func diffWorkspaceSnapshots(previous, next *Snapshot, deviceID string) ([]Op, error) {
	var ops []Op
	if !previous.WorkspacesInitialized {
		return ops, nil
	}
	for workspaceID, oldWorkspace := range previous.Workspaces {
		currentWorkspace, active := next.Workspaces[workspaceID]
		if active {
			if currentWorkspace.Path != oldWorkspace.Path {
				ops = append(ops, newWorkspaceSnapshotOp(deviceID, workspaceID, OpRename, currentWorkspace, oldWorkspace.Path))
				remapEntriesPrefix(previous.Entries, oldWorkspace.Path, currentWorkspace.Path)
			}
			continue
		}
		ops = append(ops, newWorkspaceSnapshotOp(deviceID, workspaceID, OpTrash, oldWorkspace, ""))
		oldWorkspace.Entries = entriesUnder(previous.Entries, oldWorkspace.Path)
		removeEntriesUnder(previous.Entries, oldWorkspace.Path)
		next.TrashedWorkspaces[workspaceID] = oldWorkspace
	}
	for workspaceID, currentWorkspace := range next.Workspaces {
		if _, alreadyActive := previous.Workspaces[workspaceID]; alreadyActive {
			continue
		}
		if trashedWorkspace, wasTrashed := previous.TrashedWorkspaces[workspaceID]; wasTrashed {
			ops = append(ops, newWorkspaceSnapshotOp(deviceID, workspaceID, OpRestore, currentWorkspace, ""))
			copyRemappedEntries(previous.Entries, trashedWorkspace.Entries, trashedWorkspace.Path, currentWorkspace.Path)
			delete(next.TrashedWorkspaces, workspaceID)
			continue
		}
		ops = append(ops, newWorkspaceSnapshotOp(deviceID, workspaceID, OpCreate, currentWorkspace, ""))
		delete(next.Entries, currentWorkspace.Path)
	}
	sort.Slice(ops, func(i, j int) bool {
		if ops[i].OpType != ops[j].OpType {
			return workspaceOpOrder(ops[i].OpType) < workspaceOpOrder(ops[j].OpType)
		}
		return ops[i].EntityID < ops[j].EntityID
	})
	return ops, nil
}

func workspaceOpOrder(opType string) int {
	switch opType {
	case OpCreate:
		return 0
	case OpRename:
		return 1
	case OpRestore:
		return 2
	case OpTrash:
		return 3
	default:
		return 4
	}
}

type snapshotWorkspacePayload struct {
	WorkspaceID  string          `json:"workspaceId"`
	Path         string          `json:"path"`
	PreviousPath string          `json:"previousPath,omitempty"`
	Name         string          `json:"name"`
	Metadata     json.RawMessage `json:"metadata,omitempty"`
}

func newWorkspaceSnapshotOp(deviceID, workspaceID, opType string, workspace WorkspaceSnapshot, previousPath string) Op {
	payload, _ := json.Marshal(snapshotWorkspacePayload{
		WorkspaceID:  workspaceID,
		Path:         workspace.Path,
		PreviousPath: previousPath,
		Name:         workspace.Path,
		Metadata:     workspace.Metadata,
	})
	return newSnapshotOp(deviceID, EntityWorkspace, workspaceID, opType, json.RawMessage(payload))
}

func acceptWorkspaceLifecycle(previous Snapshot, next *Snapshot) {
	for workspaceID, oldWorkspace := range previous.Workspaces {
		if _, stillActive := next.Workspaces[workspaceID]; !stillActive {
			oldWorkspace.Entries = entriesUnder(previous.Entries, oldWorkspace.Path)
			next.TrashedWorkspaces[workspaceID] = oldWorkspace
		}
	}
	for workspaceID := range next.Workspaces {
		delete(next.TrashedWorkspaces, workspaceID)
	}
	// Permanently purged.
	for workspaceID := range previous.TrashedWorkspaces {
		if _, stillTrashed := next.TrashedWorkspaces[workspaceID]; !stillTrashed {
			// Purge is not explicitly diffed here; handled by trashed list disappearing.
			_ = workspaceID
		}
	}
}

func unresolvedPath(unresolved map[string]string, path string) bool {
	for unresolvedPath := range unresolved {
		if path == unresolvedPath || strings.HasPrefix(path, unresolvedPath+"/") {
			return true
		}
	}
	return false
}

func copyEntriesUnder(destination, source map[string]SnapshotEntry, root string) {
	for path, entry := range source {
		if path == root || strings.HasPrefix(path, root+"/") {
			destination[path] = entry
		}
	}
}

func removeEntriesUnder(entries map[string]SnapshotEntry, root string) {
	for path := range entries {
		if path == root || strings.HasPrefix(path, root+"/") {
			delete(entries, path)
		}
	}
}

func remapEntriesPrefix(entries map[string]SnapshotEntry, oldPrefix, newPrefix string) {
	type remappedEntry struct {
		oldPath string
		entry   SnapshotEntry
	}
	var remapped []remappedEntry
	for path, entry := range entries {
		if path != oldPrefix && !strings.HasPrefix(path, oldPrefix+"/") {
			continue
		}
		suffix := strings.TrimPrefix(path, oldPrefix)
		entry.Path = newPrefix + suffix
		remapped = append(remapped, remappedEntry{oldPath: path, entry: entry})
	}
	for _, item := range remapped {
		delete(entries, item.oldPath)
		entries[item.entry.Path] = item.entry
	}
}

func entriesUnder(entries map[string]SnapshotEntry, root string) map[string]SnapshotEntry {
	result := make(map[string]SnapshotEntry)
	copyEntriesUnder(result, entries, root)
	return result
}

func copyRemappedEntries(destination, source map[string]SnapshotEntry, oldPrefix, newPrefix string) {
	for path, entry := range source {
		if path != oldPrefix && !strings.HasPrefix(path, oldPrefix+"/") {
			continue
		}
		suffix := strings.TrimPrefix(path, oldPrefix)
		entry.Path = newPrefix + suffix
		destination[entry.Path] = entry
	}
}

func entriesEqual(left, right SnapshotEntry) bool {
	return left.Type == right.Type && left.Size == right.Size && left.Hash == right.Hash
}

func payloadForEntry(vaultRoot string, entry SnapshotEntry) (map[string]interface{}, error) {
	payload := map[string]interface{}{"path": entry.Path, "contentHash": entry.Hash}
	if entry.Type == EntityFolder {
		return payload, nil
	}
	path := filepath.Join(vaultRoot, filepath.FromSlash(entry.Path))
	if hash, err := sha256File(path); err != nil {
		return nil, err
	} else if hash != entry.Hash {
		return nil, fmt.Errorf("file changed during scan: %s", entry.Path)
	}
	if entry.Size > corefiles.MaxTextFileBytes {
		if err := cacheBlob(path, BlobCachePath(vaultRoot, entry.Hash), entry.Hash); err != nil {
			return nil, err
		}
		payload["blob"] = map[string]interface{}{"sha256": entry.Hash, "size": entry.Size}
		return payload, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if isSyncText(data) {
		payload["content"] = string(data)
		return payload, nil
	}
	if err := cacheBlob(path, BlobCachePath(vaultRoot, entry.Hash), entry.Hash); err != nil {
		return nil, err
	}
	payload["blob"] = map[string]interface{}{"sha256": entry.Hash, "size": entry.Size}
	return payload, nil
}

func cacheBlob(source, destination, wantHash string) error {
	if info, err := os.Lstat(destination); err == nil {
		if !info.Mode().IsRegular() {
			return fmt.Errorf("blob cache target is not a regular file")
		}
		if hash, err := sha256File(destination); err == nil && hash == wantHash {
			return nil
		}
		if err := os.Remove(destination); err != nil {
			return err
		}
	} else if !os.IsNotExist(err) {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(destination), 0o750); err != nil {
		return err
	}
	in, err := os.Open(source)
	if err != nil {
		return err
	}
	defer in.Close()
	tmp, err := os.CreateTemp(filepath.Dir(destination), ".blob-*")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	cleanup := true
	defer func() {
		if cleanup {
			_ = os.Remove(tmpPath)
		}
	}()
	hash := sha256.New()
	if _, err := io.Copy(io.MultiWriter(tmp, hash), in); err != nil {
		_ = tmp.Close()
		return err
	}
	if actual := hex.EncodeToString(hash.Sum(nil)); actual != wantHash {
		_ = tmp.Close()
		return fmt.Errorf("file changed during blob staging")
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpPath, destination); err != nil {
		return err
	}
	cleanup = false
	return nil
}

func newSnapshotOp(deviceID, entityType, entityID, opType string, payload interface{}) Op {
	data, _ := json.Marshal(payload)
	id := uuid.NewString()
	return Op{
		ID:          id,
		OpID:        id,
		DeviceID:    deviceID,
		EntityType:  entityType,
		EntityID:    entityID,
		OpType:      opType,
		PayloadJSON: string(data),
		CreatedAt:   time.Now().UTC().Format(time.RFC3339Nano),
	}
}

func pathDepth(path string) int {
	if path == "" {
		return 0
	}
	return strings.Count(path, "/") + 1
}

func isSyncText(data []byte) bool {
	if !utf8.Valid(data) {
		return false
	}
	for _, r := range string(data) {
		if unicode.IsControl(r) && r != '\n' && r != '\r' && r != '\t' {
			return false
		}
	}
	return true
}
