package workspacetree

import (
	"reflect"
	"testing"
)

func TestBuildTreeAppliesMixedStoredOrderAndDeterministicFallback(t *testing.T) {
	folderA := "11111111-1111-1111-1111-111111111111"
	folderB := "22222222-2222-2222-2222-222222222222"
	workspaceA := "33333333-3333-3333-3333-333333333333"
	workspaceB := "44444444-4444-4444-4444-444444444444"
	scan := &ScanResult{
		Folders: map[string]ScannedFolder{
			folderA: {ID: folderA, Name: "Alpha", Path: "Alpha"},
			folderB: {ID: folderB, Name: "Beta", Path: "Beta"},
		},
		Workspaces: map[string]ScannedWorkspace{
			workspaceA: {ID: workspaceA, Name: "Able", RootPath: "Able"},
			workspaceB: {ID: workspaceB, Name: "Zulu", RootPath: "Zulu"},
		},
	}
	order := OrderState{
		Version: OrderVersion,
		Children: map[string][]string{
			"root": {"workspace:" + workspaceB, "folder:" + folderB},
		},
	}

	tree := BuildTree(scan, "", 7, order)

	want := []string{
		"workspace:" + workspaceB,
		"folder:" + folderB,
		"folder:" + folderA,
		"workspace:" + workspaceA,
	}
	if got := nodeKeys(tree.Roots); !reflect.DeepEqual(got, want) {
		t.Fatalf("root keys = %#v, want %#v", got, want)
	}
}

func TestBuildTreeIgnoresStaleAndWrongParentKeys(t *testing.T) {
	parentID := "11111111-1111-1111-1111-111111111111"
	childID := "22222222-2222-2222-2222-222222222222"
	workspaceID := "33333333-3333-3333-3333-333333333333"
	missingID := "44444444-4444-4444-4444-444444444444"
	scan := &ScanResult{
		Folders: map[string]ScannedFolder{
			parentID: {ID: parentID, Name: "Parent", Path: "Parent"},
			childID:  {ID: childID, Name: "Child", Path: "Parent/Child", ParentID: parentID},
		},
		Workspaces: map[string]ScannedWorkspace{
			workspaceID: {ID: workspaceID, Name: "Deal", RootPath: "Parent/Deal"},
		},
	}
	order := OrderState{
		Version: OrderVersion,
		Children: map[string][]string{
			"root": {
				"folder:" + childID,
				"workspace:" + missingID,
				"folder:" + parentID,
			},
		},
	}

	tree := BuildTree(scan, "", 1, order)

	if got := nodeKeys(tree.Roots); !reflect.DeepEqual(got, []string{"folder:" + parentID}) {
		t.Fatalf("root keys = %#v", got)
	}
	if got := nodeKeys(tree.Roots[0].Children); !reflect.DeepEqual(got, []string{
		"folder:" + childID,
		"workspace:" + workspaceID,
	}) {
		t.Fatalf("child keys = %#v", got)
	}
}

func TestBuildTreeUsesStableKeyTieBreakerForEqualNames(t *testing.T) {
	firstID := "11111111-1111-1111-1111-111111111111"
	secondID := "22222222-2222-2222-2222-222222222222"
	scan := &ScanResult{
		Folders: map[string]ScannedFolder{
			secondID: {ID: secondID, Name: "same", Path: "same-2"},
			firstID:  {ID: firstID, Name: "Same", Path: "same-1"},
		},
		Workspaces: map[string]ScannedWorkspace{},
	}

	tree := BuildTree(scan, "", 1, emptyOrderState())

	if got := nodeKeys(tree.Roots); !reflect.DeepEqual(got, []string{
		"folder:" + firstID,
		"folder:" + secondID,
	}) {
		t.Fatalf("root keys = %#v", got)
	}
}

func nodeKeys(nodes []TreeNode) []string {
	keys := make([]string, len(nodes))
	for i, node := range nodes {
		keys[i] = node.Key
	}
	return keys
}
