package dealmigration

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"
)

func writeCanonicalDealMetadata(t *testing.T, vault, workspaceID, name string) {
	t.Helper()
	writeMigrationJSON(t, vault, filepath.ToSlash(filepath.Join(".verstak", "workspaces", "uuid-"+workspaceID+".json")), map[string]any{
		"schemaVersion":  2,
		"workspaceId":    workspaceID,
		"workspaceName":  name,
		"workspaceTools": []any{"verstak.notes"},
		"toolConfig":     map[string]any{},
		"updatedAt":      "2026-08-30T00:00:00Z",
	})
}

func readCanonicalProjectMeta(t *testing.T, vault, workspaceID string) map[string]any {
	t.Helper()
	metadata := readMigrationJSON(t, vault, filepath.ToSlash(filepath.Join(".verstak", "workspaces", "uuid-"+workspaceID+".json")))
	config := metadata["toolConfig"].(map[string]any)[legacyProjectsPluginID]
	encoded, err := json.Marshal(config)
	if err != nil {
		t.Fatal(err)
	}
	var result map[string]any
	if err := json.Unmarshal(encoded, &result); err != nil {
		t.Fatal(err)
	}
	return result
}

func TestProjectMetaTransformMigratesOneLegacyProjectPerDeal(t *testing.T) {
	vault := t.TempDir()
	workspaceID := "0181c5b6-7a13-7c45-9e0e-07e0d3119da3"
	writeCanonicalDealMetadata(t, vault, workspaceID, "Acme")
	writeMigrationJSON(t, vault, ".verstak/plugin-settings/verstak.projects/settings.json", map[string]any{
		"projects:global": []any{map[string]any{
			"id":          workspaceID + ":project",
			"workspaceId": workspaceID,
			"name":        "Acme launch",
			"description": "Keep this context",
			"status":      "paused",
			"priority":    "high",
			"tags":        []any{"client", "Launch", "client"},
			"startDate":   "2026-08-01",
			"dueDate":     "2026-09-01",
			"updatedAt":   "2026-08-20T00:00:00Z",
		}},
	})

	if err := NewDealOnlyRunner(vault).Run(context.Background()); err != nil {
		t.Fatal(err)
	}
	meta := readCanonicalProjectMeta(t, vault, workspaceID)
	if meta["name"] != "Acme launch" || meta["description"] != "Keep this context" || meta["status"] != "paused" || meta["priority"] != "high" || meta["startDate"] != "2026-08-01" || meta["dueDate"] != "2026-09-01" {
		t.Fatalf("migrated Project Meta = %#v", meta)
	}
	tags := meta["tags"].([]any)
	if len(tags) != 2 || tags[0] != "client" || tags[1] != "Launch" {
		t.Fatalf("migrated tags = %#v", tags)
	}
	metadata := readMigrationJSON(t, vault, filepath.ToSlash(filepath.Join(".verstak", "workspaces", "uuid-"+workspaceID+".json")))
	tools := metadata["workspaceTools"].([]any)
	if len(tools) != 2 || tools[1] != legacyProjectsPluginID {
		t.Fatalf("Project Meta tool was not enabled: %#v", tools)
	}
}

func TestProjectMetaTransformDoesNotChooseFromMultipleLegacyProjects(t *testing.T) {
	vault := t.TempDir()
	workspaceID := "0181c5b6-7a13-7c45-9e0e-07e0d3119da3"
	writeCanonicalDealMetadata(t, vault, workspaceID, "Acme")
	writeMigrationJSON(t, vault, ".verstak/plugin-settings/verstak.projects/settings.json", map[string]any{
		"projects:global": []any{
			map[string]any{"id": "one", "workspaceId": workspaceID, "name": "First"},
			map[string]any{"id": "two", "workspaceId": workspaceID, "name": "Second"},
		},
	})

	if err := NewDealOnlyRunner(vault).Run(context.Background()); err != nil {
		t.Fatal(err)
	}
	metadata := readMigrationJSON(t, vault, filepath.ToSlash(filepath.Join(".verstak", "workspaces", "uuid-"+workspaceID+".json")))
	if _, ok := metadata["toolConfig"].(map[string]any)[legacyProjectsPluginID]; ok {
		t.Fatalf("multi-project Deal gained arbitrary Project Meta: %#v", metadata)
	}
}
