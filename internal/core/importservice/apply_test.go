package importservice

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/verstak/verstak-desktop/internal/core/workspacetree"
)

func TestApplyPlanPublishesReviewedTree(t *testing.T) {
	vault, _, session, service := newApplyTestService(t)
	page, err := service.ListEntries("verstak.import", session.SourceHandle, "")
	if err != nil {
		t.Fatal(err)
	}
	var imageID string
	for _, entry := range page.Entries {
		if entry.Path == "logo.png" {
			imageID = entry.ID
		}
	}
	plan := Plan{SchemaVersion: 1, SourceHandle: session.SourceHandle, SourceFingerprint: session.Fingerprint, RunName: "DokuWiki — 2026-07-23 12-30-00", Nodes: []PlanNode{
		{ID: "folder", Kind: "folder", Name: "Проекты"},
		{ID: "deal", ParentID: "folder", Kind: "workspace", Name: "Сайт", TemplateID: "default"},
		{ID: "note", ParentID: "deal", Kind: "note", Name: "Старт", TargetSubpath: "Документы/Старт.md", Text: "# Старт\n"},
		{ID: "file", ParentID: "deal", Kind: "file", Name: "logo.png", TargetSubpath: "assets/logo.png", SourceEntryID: imageID},
	}}
	result, err := service.ApplyPlan(context.Background(), "verstak.import", session.SourceHandle, plan)
	if err != nil {
		t.Fatal(err)
	}
	if result.RunPath != "Импортировано/DokuWiki — 2026-07-23 12-30-00" {
		t.Fatalf("runPath=%q", result.RunPath)
	}
	assertImportFileText(t, filepath.Join(vault, filepath.FromSlash(result.RunPath), "Проекты", "Сайт", "Notes", "Документы", "Старт.md"), "# Старт\n")
	assertImportFileBytes(t, filepath.Join(vault, filepath.FromSlash(result.RunPath), "Проекты", "Сайт", "Files", "assets", "logo.png"), []byte{1, 2, 3, 4})
	if result.Folders != 1 || result.Workspaces != 1 || result.Notes != 1 || result.Files != 1 {
		t.Fatalf("result=%+v", result)
	}
	if _, err := os.Stat(filepath.Join(vault, "Импортировано", ".verstak", "import-transaction.json")); !os.IsNotExist(err) {
		t.Fatalf("transaction marker remains: %v", err)
	}
	entries, err := os.ReadDir(filepath.Join(vault, ".verstak", "workspaces"))
	if err != nil || len(entries) != 1 {
		t.Fatalf("registry entries=%d err=%v", len(entries), err)
	}
}

func TestApplyPlanSuffixesRunCollision(t *testing.T) {
	vault, _, session, service := newApplyTestService(t)
	imported := filepath.Join(vault, "Импортировано")
	if err := os.MkdirAll(filepath.Join(imported, "run"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := workspacetree.WriteFolderMarker(imported, "32800f1f-321c-4f93-b7f0-99031ae00c43"); err != nil {
		t.Fatal(err)
	}
	plan := Plan{SchemaVersion: 1, SourceHandle: session.SourceHandle, SourceFingerprint: session.Fingerprint, RunName: "Run"}
	result, err := service.ApplyPlan(context.Background(), "verstak.import", session.SourceHandle, plan)
	if err != nil {
		t.Fatal(err)
	}
	if result.RunPath != "Импортировано/Run (2)" {
		t.Fatalf("runPath=%q", result.RunPath)
	}
}

func TestApplyPlanCancellationBeforePublishLeavesNoTree(t *testing.T) {
	vault, _, session, service := newApplyTestService(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	plan := Plan{SchemaVersion: 1, SourceHandle: session.SourceHandle, SourceFingerprint: session.Fingerprint, RunName: "Cancelled"}
	if _, err := service.ApplyPlan(ctx, "verstak.import", session.SourceHandle, plan); err == nil || !strings.Contains(err.Error(), "import-cancelled") {
		t.Fatalf("expected cancellation, got %v", err)
	}
	if _, err := os.Stat(filepath.Join(vault, "Импортировано")); !os.IsNotExist(err) {
		t.Fatalf("published tree exists: %v", err)
	}
}

func TestApplyPlanCancelAPIStopsStaging(t *testing.T) {
	vault, _, session, service := newApplyTestService(t)
	service.onProgress = func(_ string, progress Progress) {
		if progress.Phase == "staging" {
			_ = service.Cancel("verstak.import", session.SourceHandle)
		}
	}
	plan := Plan{SchemaVersion: 1, SourceHandle: session.SourceHandle, SourceFingerprint: session.Fingerprint, RunName: "Cancelled", Nodes: []PlanNode{{ID: "folder", Kind: "folder", Name: "One"}}}
	if _, err := service.ApplyPlan(context.Background(), "verstak.import", session.SourceHandle, plan); err == nil || !strings.Contains(err.Error(), "import-cancelled") {
		t.Fatalf("expected cancellation, got %v", err)
	}
	if _, err := os.Stat(filepath.Join(vault, "Импортировано")); !os.IsNotExist(err) {
		t.Fatalf("published tree exists: %v", err)
	}
}

func TestApplyPlanRollsBackMetadataPromotionFailure(t *testing.T) {
	vault, _, session, service := newApplyTestService(t)
	service.promoteRegistry = func(_, _ string) error { return errors.New("injected failure") }
	plan := Plan{SchemaVersion: 1, SourceHandle: session.SourceHandle, SourceFingerprint: session.Fingerprint, RunName: "Broken", Nodes: []PlanNode{{ID: "workspace", Kind: "workspace", Name: "Deal", TemplateID: "default"}}}
	if _, err := service.ApplyPlan(context.Background(), "verstak.import", session.SourceHandle, plan); err == nil || !strings.Contains(err.Error(), "import-publication-failed") {
		t.Fatalf("expected publication failure, got %v", err)
	}
	if _, err := os.Stat(filepath.Join(vault, "Импортировано")); !os.IsNotExist(err) {
		t.Fatalf("published tree remains: %v", err)
	}
	registry, err := os.ReadDir(filepath.Join(vault, ".verstak", "workspaces"))
	if err != nil || len(registry) != 0 {
		t.Fatalf("registry=%v err=%v", registry, err)
	}
}

func TestApplyPlanRejectsOrdinaryImportRoot(t *testing.T) {
	vault, _, session, service := newApplyTestService(t)
	if err := os.Mkdir(filepath.Join(vault, "Импортировано"), 0o755); err != nil {
		t.Fatal(err)
	}
	plan := Plan{SchemaVersion: 1, SourceHandle: session.SourceHandle, SourceFingerprint: session.Fingerprint, RunName: "Run"}
	if _, err := service.ApplyPlan(context.Background(), "verstak.import", session.SourceHandle, plan); err == nil || !strings.Contains(err.Error(), "import-root-conflict") {
		t.Fatalf("expected import-root-conflict, got %v", err)
	}
}

func assertImportFileText(t *testing.T, path, want string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != want {
		t.Fatalf("%s=%q want %q", path, data, want)
	}
}

func assertImportFileBytes(t *testing.T, path string, want []byte) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != string(want) {
		t.Fatalf("%s=%v want %v", path, data, want)
	}
}
