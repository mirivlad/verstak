package dealmigration

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestJournalDataTransformMovesLegacyWorklogIntoDealMarkdown(t *testing.T) {
	vault := t.TempDir()
	workspaceID := "0181c5b6-7a13-7c45-9e0e-07e0d3119da3"
	writeMigrationJSON(t, vault, "Clients/Acme/.verstak/workspace.json", map[string]any{
		"schemaVersion": 1,
		"workspaceId":   workspaceID,
	})
	writeMigrationJSON(t, vault, ".verstak/plugin-settings/verstak.journal/settings.json", map[string]any{
		"worklog:workspace:Clients%2FAcme": []any{map[string]any{
			"entryId":      "legacy-journal-1",
			"date":         "2026-08-20",
			"title":        "Preserve worklog entry",
			"summary":      "Notes written before the migration.",
			"minutes":      45,
			"billable":     true,
			"sourceTodoId": "todo-1",
		}},
	})

	if err := NewDealOnlyRunner(vault).Run(context.Background()); err != nil {
		t.Fatal(err)
	}

	markdown, err := os.ReadFile(filepath.Join(vault, "Clients", "Acme", "Журнал", "2026-08.md"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(markdown)
	for _, want := range []string{"### Preserve worklog entry", "Notes written before the migration.", `"entryId":"legacy-journal-1"`, `"sourceTodoId":"todo-1"`} {
		if !strings.Contains(text, want) {
			t.Fatalf("migrated journal markdown missing %q:\n%s", want, text)
		}
	}
	settings := readMigrationJSON(t, vault, ".verstak/plugin-settings/verstak.journal/settings.json")
	if _, exists := settings["worklog:workspace:Clients%2FAcme"]; exists {
		t.Fatalf("legacy Journal runtime key survived migration: %#v", settings)
	}
}
