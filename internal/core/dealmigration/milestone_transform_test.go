package dealmigration

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestDealOnlyMigrationCopiesLegacyMilestonesIntoIndependentProvider(t *testing.T) {
	vault := t.TempDir()
	workspaceID := "0181c5b6-7a13-7c45-9e0e-07e0d3119da3"
	writeMigrationJSON(t, vault, ".verstak/plugin-settings/verstak.projects/settings.json", map[string]any{
		"projects:global": []any{map[string]any{
			"id":          "project-1",
			"name":        "Legacy launch",
			"workspaceId": workspaceID,
			"milestones": []any{map[string]any{
				"id":          "milestone-1",
				"title":       "Beta",
				"status":      "done",
				"dueAt":       "2026-09-01",
				"createdAt":   "2026-08-01T00:00:00Z",
				"completedAt": "2026-08-02T00:00:00Z",
			}},
		}},
	})

	if err := NewDealOnlyRunner(vault).Run(context.Background()); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(vault, ".verstak", "plugin-data", "verstak.milestones", "milestones.ndjson"))
	if err != nil {
		t.Fatalf("read migrated milestones: %v", err)
	}
	var milestone map[string]any
	if err := json.Unmarshal(bytes.TrimSpace(data), &milestone); err != nil {
		t.Fatal(err)
	}
	if milestone["id"] != "milestone-1" || milestone["workspaceId"] != workspaceID || milestone["title"] != "Beta" || milestone["status"] != "done" || milestone["dueAt"] != "2026-09-01" {
		t.Fatalf("migrated milestone = %#v", milestone)
	}
	if _, ok := milestone["projectId"]; ok {
		t.Fatalf("migrated milestone retained Project scope: %#v", milestone)
	}
	provenance, _ := milestone["migrationProvenance"].(map[string]any)
	if provenance["legacyProjectId"] != "project-1" || provenance["legacyProjectName"] != "Legacy launch" {
		t.Fatalf("migrated milestone provenance = %#v", provenance)
	}
}

func TestDealOnlyMigrationPreservesLegacyMilestoneWhenIDAlreadyExists(t *testing.T) {
	vault := t.TempDir()
	workspaceID := "0181c5b6-7a13-7c45-9e0e-07e0d3119da3"
	writeMigrationJSON(t, vault, ".verstak/plugin-data/verstak.milestones/milestones.ndjson", []any{map[string]any{
		"id":          "milestone-1",
		"workspaceId": workspaceID,
		"title":       "Current milestone",
		"status":      "open",
	}})
	writeMigrationJSON(t, vault, ".verstak/plugin-settings/verstak.projects/settings.json", map[string]any{
		"projects:global": []any{map[string]any{
			"id":          "project-1",
			"name":        "Legacy launch",
			"workspaceId": workspaceID,
			"milestones":  []any{map[string]any{"id": "milestone-1", "title": "Legacy milestone"}},
		}},
	})

	if err := NewDealOnlyRunner(vault).Run(context.Background()); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(vault, ".verstak", "plugin-data", "verstak.milestones", "milestones.ndjson"))
	if err != nil {
		t.Fatal(err)
	}
	var records []map[string]any
	for _, line := range bytes.Split(bytes.TrimSpace(data), []byte{'\n'}) {
		var record map[string]any
		if err := json.Unmarshal(line, &record); err != nil {
			t.Fatal(err)
		}
		records = append(records, record)
	}
	if len(records) != 2 {
		t.Fatalf("record count = %d, want both current and legacy milestones", len(records))
	}
	var legacy map[string]any
	for _, record := range records {
		if record["title"] == "Legacy milestone" {
			legacy = record
		}
	}
	if legacy == nil || legacy["id"] == "milestone-1" {
		t.Fatalf("legacy milestone did not receive a distinct retained record: %#v", legacy)
	}
}
