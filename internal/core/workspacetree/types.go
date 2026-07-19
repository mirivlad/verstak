// Package workspacetree provides the v2 workspace tree service.
//
// The filesystem is the source of truth. Workspaces (Дела) are leaf nodes
// marked by .verstak/workspace.json. Organizational folders are marked by
// .verstak/folder.json. The scanner is read-only; markers are written by
// reconciliation.
package workspacetree

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/google/uuid"
)

// ── Marker formats ───────────────────────────────────────────────────────────

// WorkspaceMarker is stored at <deal>/.verstak/workspace.json.
type WorkspaceMarker struct {
	SchemaVersion int    `json:"schemaVersion"`
	WorkspaceID   string `json:"workspaceId"`
}

// FolderMarker is stored at <folder>/.verstak/folder.json.
type FolderMarker struct {
	SchemaVersion int    `json:"schemaVersion"`
	FolderID      string `json:"folderId"`
}

// ── Domain types ─────────────────────────────────────────────────────────────

// ScannedWorkspace is a read-only discovery result.
type ScannedWorkspace struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	RootPath string `json:"rootPath"` // relative to vault root, slash-separated
}

// ScannedFolder is a read-only discovery result.
type ScannedFolder struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Path     string `json:"path"`               // relative to vault root, slash-separated
	ParentID string `json:"parentId,omitempty"` // folder UUID or empty for root
}

// ── Tree output types ────────────────────────────────────────────────────────

// TreeNode is a single node in the public tree API.
type TreeNode struct {
	Key      string     `json:"key"`
	Kind     string     `json:"kind"` // "folder" | "workspace"
	ID       string     `json:"id"`
	Name     string     `json:"name"`
	Path     string     `json:"path"`
	Children []TreeNode `json:"children,omitempty"`
}

// TreeSnapshot is the full tree state returned to the frontend.
type TreeSnapshot struct {
	Roots              []TreeNode       `json:"roots"`
	CurrentWorkspaceID string           `json:"currentWorkspaceId,omitempty"`
	Revision           uint64           `json:"revision"`
	Warnings           []TreeDiagnostic `json:"warnings,omitempty"`
}

// TreeDiagnostic describes a non-fatal issue found during reconciliation.
type TreeDiagnostic struct {
	Level    string `json:"level"` // "warning" | "error"
	Code     string `json:"code"`
	Message  string `json:"message"`
	EntityID string `json:"entityId,omitempty"`
	Path     string `json:"path,omitempty"`
}

// ── Snapshot persistence types ───────────────────────────────────────────────

// SemanticSnapshot is the cached last known tree state stored at
// .verstak/cache/workspace-tree.json.
type SemanticSnapshot struct {
	SchemaVersion int                      `json:"schemaVersion"`
	Folders       map[string]SnapshotEntry `json:"folders"`
	Workspaces    map[string]SnapshotEntry `json:"workspaces"`
}

// SnapshotEntry records the last known path for a UUID.
type SnapshotEntry struct {
	Path string `json:"path"`
}

// ── Unmanaged directories ────────────────────────────────────────────────────

// UnmanagedDirectory is a directory without a marker, reported by the scanner.
// The scanner never writes markers; the reconciler may create folder.json.
type UnmanagedDirectory struct {
	Path   string `json:"path"`
	Name   string `json:"name"`
	Parent string `json:"parent,omitempty"` // parent path or empty for root
}

// ── Move / rename info ───────────────────────────────────────────────────────

// MoveInfo describes whether a path change is a rename, move, or both.
type MoveInfo struct {
	NameChanged   bool
	ParentChanged bool
	PreviousPath  string
	NewPath       string
}

// DetectMoveInfo compares previous and new paths.
func DetectMoveInfo(prevPath, newPath string) MoveInfo {
	prevParent := parentPath(prevPath)
	newParent := parentPath(newPath)
	prevName := filepath.Base(filepath.FromSlash(prevPath))
	newName := filepath.Base(filepath.FromSlash(newPath))
	return MoveInfo{
		NameChanged:   prevName != newName,
		ParentChanged: prevParent != newParent,
		PreviousPath:  prevPath,
		NewPath:       newPath,
	}
}

// ── Reconciliation types ─────────────────────────────────────────────────────

// ScanResult is the output of a full read-only filesystem scan.
type ScanResult struct {
	Folders    map[string]ScannedFolder    // keyed by UUID
	Workspaces map[string]ScannedWorkspace // keyed by UUID
	Unmanaged  []UnmanagedDirectory
	Warnings   []TreeDiagnostic
}

// ReconEvent describes a semantic change detected during reconciliation.
type ReconEvent struct {
	Name    string                 `json:"name"`
	Payload map[string]interface{} `json:"payload"`
}

// ReconResult is the output of reconciliation.
type ReconResult struct {
	Snapshot      SemanticSnapshot
	Events        []ReconEvent
	Warnings      []TreeDiagnostic
	NewFolders    []ScannedFolder // folders created (with new markers) during reconciliation
	NewWorkspaces []ScannedWorkspace
}

// ── Change class for watcher ─────────────────────────────────────────────────

// ChangeClass classifies a filesystem event as content or structural.
type ChangeClass string

const (
	ChangeContent    ChangeClass = "content"
	ChangeStructural ChangeClass = "structural"
)

// ── Marker helpers ───────────────────────────────────────────────────────────

// ParseWorkspaceMarker decodes a workspace marker without validation.
func ParseWorkspaceMarker(data []byte) (WorkspaceMarker, error) {
	var m WorkspaceMarker
	if err := json.Unmarshal(data, &m); err != nil {
		return m, err
	}
	return m, nil
}

// ParseFolderMarker decodes a folder marker without validation.
func ParseFolderMarker(data []byte) (FolderMarker, error) {
	var m FolderMarker
	if err := json.Unmarshal(data, &m); err != nil {
		return m, err
	}
	return m, nil
}

// ── Marker I/O ───────────────────────────────────────────────────────────────

// WriteWorkspaceMarker atomically writes a workspace marker file.
func WriteWorkspaceMarker(dir string, id string) error {
	if _, err := uuid.Parse(id); err != nil {
		return fmt.Errorf("invalid workspace identity: %w", err)
	}
	marker := WorkspaceMarker{SchemaVersion: 1, WorkspaceID: id}
	data, err := json.Marshal(marker)
	if err != nil {
		return err
	}
	// Diagnostic: log every workspace marker write with caller context.
	absDir, _ := filepath.Abs(dir)
	fmt.Fprintf(os.Stderr, "[MARKER-DIAG] WriteWorkspaceMarker dir=%q id=%s\n", absDir, id)

	// Invariant: refuse to write workspace marker into an organizational folder.
	verstakDir := filepath.Join(dir, ".verstak")
	if _, err := os.Stat(filepath.Join(verstakDir, "folder.json")); err == nil {
		return fmt.Errorf("refusing to write workspace marker into directory with folder marker: %s", dir)
	}
	return atomicWrite(dir, "workspace.json", data)
}

// WriteFolderMarker atomically writes a folder marker file.
func WriteFolderMarker(dir string, id string) error {
	if _, err := uuid.Parse(id); err != nil {
		return fmt.Errorf("invalid folder identity: %w", err)
	}
	marker := FolderMarker{SchemaVersion: 1, FolderID: id}
	data, err := json.Marshal(marker)
	if err != nil {
		return err
	}
	return atomicWrite(dir, "folder.json", data)
}

func atomicWrite(parentDir, filename string, data []byte) error {
	verstakDir := filepath.Join(parentDir, ".verstak")
	if err := os.MkdirAll(verstakDir, 0o755); err != nil {
		return err
	}
	target := filepath.Join(verstakDir, filename)
	tmp := target + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	if err := os.Rename(tmp, target); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}
