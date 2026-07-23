package workspacetree

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestPlaceNodeBeforeAfterPersistsMixedOrder(t *testing.T) {
	vault := t.TempDir()
	svc := NewService(vault, nil)
	if err := svc.Initialize(); err != nil {
		t.Fatal(err)
	}
	alpha, _ := svc.CreateFolder("", "Alpha", noopRefresh)
	beta, _ := svc.CreateFolder("", "Beta", noopRefresh)
	deal, _ := svc.CreateWorkspace("", "Deal", "", noopRefresh)

	if _, err := svc.PlaceNode(PlacementRequest{
		SourceKey: "workspace:" + deal.ID,
		TargetKey: "folder:" + alpha.ID,
		Position:  "before",
	}, noopRefresh); err != nil {
		t.Fatal(err)
	}
	if got := nodeKeys(svc.GetTree().Roots); !reflect.DeepEqual(got, []string{
		"workspace:" + deal.ID,
		"folder:" + alpha.ID,
		"folder:" + beta.ID,
	}) {
		t.Fatalf("after before placement = %#v", got)
	}

	if _, err := svc.PlaceNode(PlacementRequest{
		SourceKey: "folder:" + beta.ID,
		TargetKey: "workspace:" + deal.ID,
		Position:  "after",
	}, noopRefresh); err != nil {
		t.Fatal(err)
	}
	want := []string{
		"workspace:" + deal.ID,
		"folder:" + beta.ID,
		"folder:" + alpha.ID,
	}
	if got := nodeKeys(svc.GetTree().Roots); !reflect.DeepEqual(got, want) {
		t.Fatalf("after placement = %#v, want %#v", got, want)
	}

	restarted := NewService(vault, nil)
	if err := restarted.Initialize(); err != nil {
		t.Fatal(err)
	}
	if got := nodeKeys(restarted.GetTree().Roots); !reflect.DeepEqual(got, want) {
		t.Fatalf("after restart = %#v, want %#v", got, want)
	}
}

func TestPlaceNodeInsideAndBackToRootUpdatesFilesystem(t *testing.T) {
	vault := t.TempDir()
	svc := NewService(vault, nil)
	if err := svc.Initialize(); err != nil {
		t.Fatal(err)
	}
	parent, _ := svc.CreateFolder("", "Parent", noopRefresh)
	deal, _ := svc.CreateWorkspace("", "Deal", "", noopRefresh)

	if _, err := svc.PlaceNode(PlacementRequest{
		SourceKey: "workspace:" + deal.ID,
		TargetKey: "folder:" + parent.ID,
		Position:  "inside",
	}, noopRefresh); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(vault, "Parent", "Deal")); err != nil {
		t.Fatalf("Deal was not moved inside folder: %v", err)
	}
	if got := nodeKeys(svc.GetTree().Roots[0].Children); !reflect.DeepEqual(got, []string{"workspace:" + deal.ID}) {
		t.Fatalf("inside children = %#v", got)
	}

	if _, err := svc.PlaceNode(PlacementRequest{
		SourceKey: "workspace:" + deal.ID,
		Position:  "root",
	}, noopRefresh); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(vault, "Deal")); err != nil {
		t.Fatalf("Deal was not moved to root: %v", err)
	}
	if got := nodeKeys(svc.GetTree().Roots); !reflect.DeepEqual(got, []string{
		"folder:" + parent.ID,
		"workspace:" + deal.ID,
	}) {
		t.Fatalf("root keys = %#v", got)
	}
}

func TestPlaceNodeOrderOnlyDoesNotRenameFilesystemEntry(t *testing.T) {
	vault := t.TempDir()
	svc := NewService(vault, nil)
	if err := svc.Initialize(); err != nil {
		t.Fatal(err)
	}
	first, _ := svc.CreateWorkspace("", "First", "", noopRefresh)
	second, _ := svc.CreateWorkspace("", "Second", "", noopRefresh)

	if _, err := svc.PlaceNode(PlacementRequest{
		SourceKey: "workspace:" + second.ID,
		TargetKey: "workspace:" + first.ID,
		Position:  "before",
	}, noopRefresh); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"First", "Second"} {
		if _, err := os.Stat(filepath.Join(vault, path)); err != nil {
			t.Fatalf("%s changed during order-only placement: %v", path, err)
		}
	}
}

func TestPlaceNodeRejectsInvalidRequests(t *testing.T) {
	vault := t.TempDir()
	svc := NewService(vault, nil)
	if err := svc.Initialize(); err != nil {
		t.Fatal(err)
	}
	folder, _ := svc.CreateFolder("", "Folder", noopRefresh)
	deal, _ := svc.CreateWorkspace("", "Deal", "", noopRefresh)
	tests := map[string]PlacementRequest{
		"malformed source": {SourceKey: "deal:" + deal.ID, TargetKey: "folder:" + folder.ID, Position: "before"},
		"missing source":   {SourceKey: "workspace:99999999-9999-9999-9999-999999999999", TargetKey: "folder:" + folder.ID, Position: "before"},
		"missing target":   {SourceKey: "workspace:" + deal.ID, TargetKey: "folder:99999999-9999-9999-9999-999999999999", Position: "before"},
		"unsupported":      {SourceKey: "workspace:" + deal.ID, TargetKey: "folder:" + folder.ID, Position: "near"},
		"self":             {SourceKey: "workspace:" + deal.ID, TargetKey: "workspace:" + deal.ID, Position: "before"},
		"inside deal":      {SourceKey: "folder:" + folder.ID, TargetKey: "workspace:" + deal.ID, Position: "inside"},
		"root target":      {SourceKey: "workspace:" + deal.ID, TargetKey: "folder:" + folder.ID, Position: "root"},
	}
	for name, request := range tests {
		t.Run(name, func(t *testing.T) {
			if _, err := svc.PlaceNode(request, noopRefresh); err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}

func TestPlaceNodeRejectsFolderIntoDescendant(t *testing.T) {
	vault := t.TempDir()
	svc := NewService(vault, nil)
	if err := svc.Initialize(); err != nil {
		t.Fatal(err)
	}
	parent, _ := svc.CreateFolder("", "Parent", noopRefresh)
	child, _ := svc.CreateFolder(parent.ID, "Child", noopRefresh)

	_, err := svc.PlaceNode(PlacementRequest{
		SourceKey: "folder:" + parent.ID,
		TargetKey: "folder:" + child.ID,
		Position:  "inside",
	}, noopRefresh)
	if err == nil || !strings.Contains(err.Error(), "descendant") {
		t.Fatalf("error = %v, want descendant rejection", err)
	}
}

func TestPlaceNodeRejectsDestinationPathConflict(t *testing.T) {
	vault := t.TempDir()
	svc := NewService(vault, nil)
	if err := svc.Initialize(); err != nil {
		t.Fatal(err)
	}
	source, _ := svc.CreateFolder("", "Same", noopRefresh)
	target, _ := svc.CreateFolder("", "Target", noopRefresh)
	if _, err := svc.CreateFolder(target.ID, "Same", noopRefresh); err != nil {
		t.Fatal(err)
	}

	_, err := svc.PlaceNode(PlacementRequest{
		SourceKey: "folder:" + source.ID,
		TargetKey: "folder:" + target.ID,
		Position:  "inside",
	}, noopRefresh)
	if err == nil || !strings.Contains(err.Error(), "conflict") {
		t.Fatalf("error = %v, want path conflict", err)
	}
	if _, statErr := os.Stat(filepath.Join(vault, "Same")); statErr != nil {
		t.Fatalf("source moved despite conflict: %v", statErr)
	}
}

func TestPlaceNodeReportsOrderWriteFailure(t *testing.T) {
	vault := t.TempDir()
	svc := NewService(vault, nil)
	if err := svc.Initialize(); err != nil {
		t.Fatal(err)
	}
	first, _ := svc.CreateWorkspace("", "First", "", noopRefresh)
	second, _ := svc.CreateWorkspace("", "Second", "", noopRefresh)
	metadataDir := filepath.Join(vault, ".verstak", "workspace-tree")
	if err := os.WriteFile(metadataDir, []byte("not a directory"), 0o600); err != nil {
		t.Fatal(err)
	}

	_, err := svc.PlaceNode(PlacementRequest{
		SourceKey: "workspace:" + second.ID,
		TargetKey: "workspace:" + first.ID,
		Position:  "before",
	}, noopRefresh)
	if err == nil {
		t.Fatal("expected order write error")
	}
}
