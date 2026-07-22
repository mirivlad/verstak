package importservice

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidatePlanRejectsUntrustedGraphsAndPaths(t *testing.T) {
	tests := []struct {
		name  string
		nodes []PlanNode
		code  string
	}{
		{name: "orphan", nodes: []PlanNode{{ID: "note", ParentID: "missing", Kind: "note", Name: "N", TargetSubpath: "N.md", Text: "x"}}, code: "invalid-plan-parent"},
		{name: "cycle", nodes: []PlanNode{{ID: "a", ParentID: "b", Kind: "folder", Name: "A"}, {ID: "b", ParentID: "a", Kind: "folder", Name: "B"}}, code: "invalid-plan-cycle"},
		{name: "traversal", nodes: []PlanNode{{ID: "ws", Kind: "workspace", Name: "W", TemplateID: "default"}, {ID: "note", ParentID: "ws", Kind: "note", Name: "N", TargetSubpath: "../N.md", Text: "x"}}, code: "invalid-target-path"},
		{name: "reserved", nodes: []PlanNode{{ID: "ws", Kind: "workspace", Name: ".verstak", TemplateID: "default"}}, code: "invalid-plan-name"},
		{name: "wrong parent kind", nodes: []PlanNode{{ID: "folder", Kind: "folder", Name: "F"}, {ID: "note", ParentID: "folder", Kind: "note", Name: "N", TargetSubpath: "N.md", Text: "x"}}, code: "invalid-plan-parent"},
		{name: "duplicate folded path", nodes: []PlanNode{{ID: "a", Kind: "workspace", Name: "Дело", TemplateID: "default"}, {ID: "b", Kind: "workspace", Name: "дело", TemplateID: "default"}}, code: "duplicate-target-path"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			vault, source, session, service := newApplyTestService(t)
			_ = vault
			_ = source
			plan := Plan{SchemaVersion: 1, SourceHandle: session.SourceHandle, SourceFingerprint: session.Fingerprint, RunName: "Obsidian — 2026-07-23 12-30-00", Nodes: test.nodes}
			_, err := service.ApplyPlan(context.Background(), "verstak.import", session.SourceHandle, plan)
			if err == nil || !strings.Contains(err.Error(), test.code) {
				t.Fatalf("expected %s, got %v", test.code, err)
			}
		})
	}
}

func TestValidatePlanRejectsWrongFingerprintAndMissingSourceEntry(t *testing.T) {
	_, _, session, service := newApplyTestService(t)
	plan := Plan{SchemaVersion: 1, SourceHandle: session.SourceHandle, SourceFingerprint: "wrong", RunName: "Import", Nodes: nil}
	if _, err := service.ApplyPlan(context.Background(), "verstak.import", session.SourceHandle, plan); err == nil || !strings.Contains(err.Error(), "source-fingerprint-mismatch") {
		t.Fatalf("wrong fingerprint: %v", err)
	}

	plan.SourceFingerprint = session.Fingerprint
	plan.Nodes = []PlanNode{{ID: "ws", Kind: "workspace", Name: "W", TemplateID: "default"}, {ID: "file", ParentID: "ws", Kind: "file", Name: "x", TargetSubpath: "x.bin", SourceEntryID: "missing"}}
	if _, err := service.ApplyPlan(context.Background(), "verstak.import", session.SourceHandle, plan); err == nil || !strings.Contains(err.Error(), "source-entry-not-found") {
		t.Fatalf("missing source entry: %v", err)
	}
}

func newApplyTestService(t *testing.T) (string, string, SourceSession, *Service) {
	t.Helper()
	vault := t.TempDir()
	if err := os.MkdirAll(filepath.Join(vault, ".verstak", "workspaces"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(vault, ".verstak", "vault.json"), []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}
	source := t.TempDir()
	if err := os.WriteFile(filepath.Join(source, "logo.png"), []byte{1, 2, 3, 4}, 0o600); err != nil {
		t.Fatal(err)
	}
	service := New(vault, Options{})
	session, err := service.OpenDirectory("verstak.import", source)
	if err != nil {
		t.Fatal(err)
	}
	return vault, source, session, service
}
