package workspacetree

import (
	"encoding/json"
	"os"
	"path/filepath"
)

const (
	snapshotSchemaVersion = 1
	snapshotRelPath        = ".verstak/cache/workspace-tree.json"
)

// ReadSnapshot reads the last known semantic snapshot. Returns nil if none exists.
func ReadSnapshot(vaultDir string) (*SemanticSnapshot, error) {
	path := filepath.Join(vaultDir, filepath.FromSlash(snapshotRelPath))
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var snap SemanticSnapshot
	if err := json.Unmarshal(data, &snap); err != nil {
		// Corrupted cache is not fatal; treat as missing.
		return nil, nil
	}
	if snap.SchemaVersion != snapshotSchemaVersion {
		return nil, nil
	}
	if snap.Folders == nil {
		snap.Folders = make(map[string]SnapshotEntry)
	}
	if snap.Workspaces == nil {
		snap.Workspaces = make(map[string]SnapshotEntry)
	}
	return &snap, nil
}

// WriteSnapshot atomically writes the semantic snapshot.
func WriteSnapshot(vaultDir string, snap *SemanticSnapshot) error {
	if snap.SchemaVersion == 0 {
		snap.SchemaVersion = snapshotSchemaVersion
	}
	if snap.Folders == nil {
		snap.Folders = make(map[string]SnapshotEntry)
	}
	if snap.Workspaces == nil {
		snap.Workspaces = make(map[string]SnapshotEntry)
	}

	data, err := json.MarshalIndent(snap, "", "  ")
	if err != nil {
		return err
	}

	path := filepath.Join(vaultDir, filepath.FromSlash(snapshotRelPath))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}

// RemoveSnapshot deletes the cached semantic snapshot.
func RemoveSnapshot(vaultDir string) error {
	path := filepath.Join(vaultDir, filepath.FromSlash(snapshotRelPath))
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// NewSnapshotFromScan creates a fresh SemanticSnapshot from a ScanResult.
func NewSnapshotFromScan(scan *ScanResult) SemanticSnapshot {
	snap := SemanticSnapshot{
		SchemaVersion: snapshotSchemaVersion,
		Folders:       make(map[string]SnapshotEntry, len(scan.Folders)),
		Workspaces:    make(map[string]SnapshotEntry, len(scan.Workspaces)),
	}
	for id, f := range scan.Folders {
		snap.Folders[id] = SnapshotEntry{Path: f.Path}
	}
	for id, w := range scan.Workspaces {
		snap.Workspaces[id] = SnapshotEntry{Path: w.RootPath}
	}
	return snap
}
