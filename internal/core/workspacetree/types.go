// Package workspacetree provides the v2 workspace tree service.
//
// The filesystem is the source of truth. Workspaces (Дела) are leaf nodes
// marked by .verstak/workspace.json. Organizational folders are marked by
// .verstak/folder.json. The scanner is read-only; markers are written by
// reconciliation.
package workspacetree

import "encoding/json"

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

// ── Reconciliation types ─────────────────────────────────────────────────────

// ScanResult is the output of a full read-only filesystem scan.
type ScanResult struct {
	Folders    map[string]ScannedFolder    // keyed by UUID
	Workspaces map[string]ScannedWorkspace // keyed by UUID
	Warnings   []TreeDiagnostic
}

// ReconEvent describes a semantic change detected during reconciliation.
type ReconEvent struct {
	Name    string                 `json:"name"`
	Payload map[string]interface{} `json:"payload"`
}

// ReconResult is the output of reconciliation.
type ReconResult struct {
	Snapshot SemanticSnapshot
	Events   []ReconEvent
	Warnings []TreeDiagnostic
}

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
