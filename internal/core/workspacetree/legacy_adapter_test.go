package workspacetree

import "strings"

// These helpers exist only in the test build so lifecycle tests can focus on
// tree operations without making the removed template-ID API part of runtime.
func (s *Service) CreateWorkspace(parentFolderID, name, templateID string, refreshBaseline func() error) (ScannedWorkspace, error) {
	if strings.TrimSpace(templateID) == "" {
		templateID = "test-direct"
	}
	return s.CreateWorkspaceFromRecipe(parentFolderID, name, DealRecipeSnapshot{Provenance: RecipeProvenance{TemplateID: templateID}}, refreshBaseline)
}

func (s *Service) CreateWorkspaceWithTools(parentFolderID, name, templateID string, workspaceTools []string, refreshBaseline func() error) (ScannedWorkspace, error) {
	if strings.TrimSpace(templateID) == "" {
		templateID = "test-direct"
	}
	return s.CreateWorkspaceFromRecipe(parentFolderID, name, DealRecipeSnapshot{WorkspaceTools: workspaceTools, Provenance: RecipeProvenance{TemplateID: templateID}}, refreshBaseline)
}
