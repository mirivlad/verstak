package dealmigration

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func writeMigrationJSON(t *testing.T, vault, relative string, value any) {
	t.Helper()
	path := filepath.Join(vault, filepath.FromSlash(relative))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
}

func readMigrationJSON(t *testing.T, vault, relative string) map[string]any {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(vault, filepath.FromSlash(relative)))
	if err != nil {
		t.Fatal(err)
	}
	var value map[string]any
	if err := json.Unmarshal(data, &value); err != nil {
		t.Fatal(err)
	}
	return value
}

func TestProviderDataTransformMovesContentToDealUUIDAndDropsProjectScope(t *testing.T) {
	vault := t.TempDir()
	workspaceID := "0181c5b6-7a13-7c45-9e0e-07e0d3119da3"
	writeMigrationJSON(t, vault, ".verstak/plugin-settings/verstak.projects/settings.json", map[string]any{
		"projects:global": []any{map[string]any{
			"id":                "project-1",
			"workspaceId":       workspaceID,
			"workspaceRootPath": "Clients/Acme",
		}},
	})
	writeMigrationJSON(t, vault, ".verstak/plugin-settings/verstak.notes/settings.json", map[string]any{
		"notes:projectScopes": map[string]any{"project-1": "Clients/Acme/Notes"},
		"notes:global":        []any{map[string]any{"id": "note-1", "projectId": "project-1", "title": "Preserve me"}},
	})
	writeMigrationJSON(t, vault, ".verstak/plugin-settings/verstak.todo/settings.json", map[string]any{
		"todos:global": []any{
			map[string]any{"id": "todo-1", "projectId": "project-1", "title": "Keep task"},
			map[string]any{"id": "todo-2", "projectId": "unknown-project", "title": "Keep unassigned task"},
		},
	})
	writeMigrationJSON(t, vault, ".verstak/plugin-data/verstak.activity/activity-events.ndjson", []any{
		map[string]any{"activityId": "activity-1", "projectId": "project-1", "summary": "Keep activity"},
	})

	runner := NewDealOnlyRunner(vault)
	if err := runner.Run(context.Background()); err != nil {
		t.Fatal(err)
	}

	notes := readMigrationJSON(t, vault, ".verstak/plugin-settings/verstak.notes/settings.json")
	if _, ok := notes["notes:projectScopes"]; ok {
		t.Fatal("notes Project scope survived migration")
	}
	note := notes["notes:global"].([]any)[0].(map[string]any)
	if note["workspaceId"] != workspaceID || note["title"] != "Preserve me" {
		t.Fatalf("note content was not mapped to Deal UUID: %#v", note)
	}
	if _, ok := note["projectId"]; ok {
		t.Fatalf("note still has projectId: %#v", note)
	}
	todos := readMigrationJSON(t, vault, ".verstak/plugin-settings/verstak.todo/settings.json")
	todo := todos["todos:global"].([]any)[0].(map[string]any)
	if todo["workspaceId"] != workspaceID || todo["title"] != "Keep task" {
		t.Fatalf("todo content was not mapped to Deal UUID: %#v", todo)
	}
	if _, ok := todo["projectId"]; ok {
		t.Fatalf("todo still has projectId: %#v", todo)
	}
	unassigned := todos["todos:global"].([]any)[1].(map[string]any)
	if unassigned["title"] != "Keep unassigned task" {
		t.Fatalf("unmapped todo content was lost: %#v", unassigned)
	}
	if _, ok := unassigned["workspaceId"]; ok {
		t.Fatalf("unmapped todo was assigned to an arbitrary Deal: %#v", unassigned)
	}
	activityData, err := os.ReadFile(filepath.Join(vault, ".verstak/plugin-data/verstak.activity/activity-events.ndjson"))
	if err != nil {
		t.Fatal(err)
	}
	var activity map[string]any
	if err := json.Unmarshal(activityData, &activity); err != nil {
		t.Fatal(err)
	}
	if activity["workspaceId"] != workspaceID || activity["summary"] != "Keep activity" {
		t.Fatalf("activity content was not mapped to Deal UUID: %#v", activity)
	}
	if _, ok := activity["projectId"]; ok {
		t.Fatalf("activity still has projectId: %#v", activity)
	}

	projects := readMigrationJSON(t, vault, ".verstak/plugin-settings/verstak.projects/settings.json")
	if projects["projects:global"].([]any)[0].(map[string]any)["id"] != "project-1" {
		t.Fatalf("legacy Projects source was unexpectedly changed: %#v", projects)
	}
}
