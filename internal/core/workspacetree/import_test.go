package workspacetree

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestPrepareImportedWorkspace(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "staged-workspace")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}

	prepared, err := PrepareImportedWorkspace(dir, "Сайт", "default")
	if err != nil {
		t.Fatal(err)
	}
	if prepared.ID == "" {
		t.Fatal("workspace ID is empty")
	}
	if _, err := os.Stat(filepath.Join(dir, ".verstak", "workspace.json")); err != nil {
		t.Fatalf("workspace marker: %v", err)
	}

	var metadata map[string]any
	if err := json.Unmarshal(prepared.RegistryJSON, &metadata); err != nil {
		t.Fatalf("registry JSON: %v", err)
	}
	if metadata["workspaceId"] != prepared.ID || metadata["workspaceName"] != "Сайт" {
		t.Fatalf("metadata = %#v", metadata)
	}
}

func TestPrepareImportedWorkspaceTreatsTemplateAsProvenanceOnly(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "staged-workspace")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	prepared, err := PrepareImportedWorkspace(dir, "Сайт", "admin")
	if err != nil {
		t.Fatal(err)
	}
	var metadata map[string]any
	if err := json.Unmarshal(prepared.RegistryJSON, &metadata); err != nil {
		t.Fatal(err)
	}
	provenance, _ := metadata["createdFromTemplate"].(map[string]any)
	if provenance["templateId"] != "admin" {
		t.Fatalf("provenance = %#v", provenance)
	}
}
