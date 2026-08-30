package workspacetree

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCreateWorkspaceFromRecipeUsesSnapshotWithoutTemplateRegistry(t *testing.T) {
	vault := t.TempDir()
	svc := NewService(vault, nil)
	if err := svc.Initialize(); err != nil {
		t.Fatal(err)
	}
	recipe := DealRecipeSnapshot{
		WorkspaceTools: []string{"third.party.research"},
		InitialFolders: []string{"Brief"},
		InitialFiles:   []DealRecipeFile{{Path: "Brief/README.md", Content: "# Brief\n"}},
		Provenance:     RecipeProvenance{TemplateID: "custom-research", TemplateName: "Research", TemplateVersion: 3},
	}
	workspace, err := svc.CreateWorkspaceFromRecipe("", "Research Deal", recipe, noopRefresh)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(vault, "Research Deal", "Brief", "README.md")); err != nil {
		t.Fatalf("recipe file missing: %v", err)
	}
	metadata, err := svc.ReadDealMetadata(workspace.ID, workspace.RootPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(metadata.WorkspaceTools) != 1 || metadata.WorkspaceTools[0] != "third.party.research" {
		t.Fatalf("workspace tools = %#v", metadata.WorkspaceTools)
	}
	if metadata.CreatedFromTemplate == nil || metadata.CreatedFromTemplate.TemplateID != "custom-research" || metadata.CreatedFromTemplate.RecipeDigest == "" {
		t.Fatalf("recipe provenance = %#v", metadata.CreatedFromTemplate)
	}
}
