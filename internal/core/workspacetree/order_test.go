package workspacetree

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const (
	testFolderID     = "11111111-1111-1111-1111-111111111111"
	testWorkspaceID  = "22222222-2222-2222-2222-222222222222"
	testWorkspaceID2 = "33333333-3333-3333-3333-333333333333"
)

func TestOrderMetadataPathIsInternalVaultMetadata(t *testing.T) {
	vault := t.TempDir()
	got := OrderMetadataPath(vault)
	want := filepath.Join(vault, ".verstak", "workspace-tree", "order.json")
	if got != want {
		t.Fatalf("OrderMetadataPath() = %q, want %q", got, want)
	}
}

func TestOrderStateMissingFileReturnsEmptyVersionOne(t *testing.T) {
	state, err := ReadOrderState(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if state.Version != OrderVersion {
		t.Fatalf("Version = %d, want %d", state.Version, OrderVersion)
	}
	if state.Children == nil || len(state.Children) != 0 {
		t.Fatalf("Children = %#v, want initialized empty map", state.Children)
	}
}

func TestOrderStateWriteReadRoundTrip(t *testing.T) {
	vault := t.TempDir()
	want := OrderState{
		Version: OrderVersion,
		Children: map[string][]string{
			"root":       {"workspace:" + testWorkspaceID, "folder:" + testFolderID},
			testFolderID: {"workspace:" + testWorkspaceID2},
		},
	}
	if err := WriteOrderState(vault, want); err != nil {
		t.Fatal(err)
	}
	got, err := ReadOrderState(vault)
	if err != nil {
		t.Fatal(err)
	}
	if got.Version != want.Version {
		t.Fatalf("Version = %d, want %d", got.Version, want.Version)
	}
	if strings.Join(got.Children["root"], ",") != strings.Join(want.Children["root"], ",") {
		t.Fatalf("root order = %#v, want %#v", got.Children["root"], want.Children["root"])
	}
	info, err := os.Stat(OrderMetadataPath(vault))
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("mode = %o, want 600", info.Mode().Perm())
	}
}

func TestOrderStateMarshalSortsParentsAndPreservesSiblingOrder(t *testing.T) {
	state := OrderState{
		Version: OrderVersion,
		Children: map[string][]string{
			testFolderID: {"workspace:" + testWorkspaceID2},
			"root":       {"workspace:" + testWorkspaceID, "folder:" + testFolderID},
		},
	}
	data, err := MarshalOrderState(state)
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	if strings.Index(text, `"`+testFolderID+`"`) > strings.Index(text, `"root"`) {
		t.Fatalf("parent keys are not sorted: %s", text)
	}
	rootIndex := strings.Index(text, `"root"`)
	rootText := text[rootIndex:]
	if strings.Index(rootText, `"workspace:`+testWorkspaceID+`"`) > strings.Index(rootText, `"folder:`+testFolderID+`"`) {
		t.Fatalf("root sibling order changed: %s", text)
	}
}

func TestOrderStateRejectsInvalidDocuments(t *testing.T) {
	tests := map[string]string{
		"malformed json": `{`,
		"wrong version":  `{"version":2,"children":{}}`,
		"invalid parent": `{"version":1,"children":{"not-a-uuid":[]}}`,
		"invalid key":    `{"version":1,"children":{"root":["deal:not-a-uuid"]}}`,
		"duplicate key":  `{"version":1,"children":{"root":["folder:` + testFolderID + `"],"` + testFolderID + `":["folder:` + testFolderID + `"]}}`,
	}
	for name, document := range tests {
		t.Run(name, func(t *testing.T) {
			if _, err := ParseOrderState([]byte(document)); err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}

func TestOrderStateAllowsSyntacticallyValidMissingIDs(t *testing.T) {
	document := `{"version":1,"children":{"root":["workspace:` + testWorkspaceID + `"]}}`
	state, err := ParseOrderState([]byte(document))
	if err != nil {
		t.Fatal(err)
	}
	if got := state.Children["root"]; len(got) != 1 || got[0] != "workspace:"+testWorkspaceID {
		t.Fatalf("root order = %#v", got)
	}
}

func TestApplyOrderStateWritesAndReconcilesTree(t *testing.T) {
	vault := t.TempDir()
	svc := NewService(vault, nil)
	if err := svc.Initialize(); err != nil {
		t.Fatal(err)
	}
	folder, _ := svc.CreateFolder("", "Folder", noopRefresh)
	deal, _ := svc.CreateWorkspace("", "Deal", "", noopRefresh)
	state := OrderState{
		Version: OrderVersion,
		Children: map[string][]string{
			"root": {"workspace:" + deal.ID, "folder:" + folder.ID},
		},
	}

	if err := svc.ApplyOrderState(state); err != nil {
		t.Fatal(err)
	}
	if got := nodeKeys(svc.GetTree().Roots); strings.Join(got, ",") != strings.Join(state.Children["root"], ",") {
		t.Fatalf("tree order = %#v, want %#v", got, state.Children["root"])
	}
	persisted, err := ReadOrderState(vault)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(persisted.Children["root"], ",") != strings.Join(state.Children["root"], ",") {
		t.Fatalf("persisted order = %#v", persisted.Children["root"])
	}
}
