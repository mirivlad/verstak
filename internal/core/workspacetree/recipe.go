package workspacetree

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
)

// DealRecipeSnapshot is a complete, plugin-neutral creation input. Core only
// validates generic filesystem and JSON constraints; it does not know any
// template or plugin identity.
type DealRecipeSnapshot struct {
	WorkspaceTools []string                   `json:"workspaceTools"`
	InitialFolders []string                   `json:"initialFolders,omitempty"`
	InitialFiles   []DealRecipeFile           `json:"initialFiles,omitempty"`
	ToolConfig     map[string]json.RawMessage `json:"toolConfig,omitempty"`
	Provenance     RecipeProvenance           `json:"provenance"`
}

type DealRecipeFile struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}
type RecipeProvenance struct {
	TemplateID      string `json:"templateId"`
	TemplateName    string `json:"templateName,omitempty"`
	TemplateVersion int    `json:"templateVersion,omitempty"`
}

func (r DealRecipeSnapshot) validated() (DealRecipeSnapshot, string, error) {
	r.WorkspaceTools = normalizeWorkspaceTools(r.WorkspaceTools)
	if strings.TrimSpace(r.Provenance.TemplateID) == "" {
		return DealRecipeSnapshot{}, "", fmt.Errorf("recipe provenance template ID is required")
	}
	seen := map[string]bool{}
	for _, folder := range r.InitialFolders {
		if err := validateRecipePath(folder); err != nil {
			return DealRecipeSnapshot{}, "", fmt.Errorf("invalid recipe folder: %w", err)
		}
		if seen[folder] {
			return DealRecipeSnapshot{}, "", fmt.Errorf("duplicate recipe path: %s", folder)
		}
		seen[folder] = true
	}
	for _, file := range r.InitialFiles {
		if err := validateRecipePath(file.Path); err != nil {
			return DealRecipeSnapshot{}, "", fmt.Errorf("invalid recipe file: %w", err)
		}
		if seen[file.Path] {
			return DealRecipeSnapshot{}, "", fmt.Errorf("duplicate recipe path: %s", file.Path)
		}
		if len(file.Content) > 1<<20 {
			return DealRecipeSnapshot{}, "", fmt.Errorf("recipe file exceeds 1 MiB: %s", file.Path)
		}
		seen[file.Path] = true
	}
	if r.ToolConfig == nil {
		r.ToolConfig = map[string]json.RawMessage{}
	}
	for namespace, value := range r.ToolConfig {
		if strings.TrimSpace(namespace) == "" || !json.Valid(value) {
			return DealRecipeSnapshot{}, "", fmt.Errorf("invalid recipe tool config")
		}
	}
	data, err := json.Marshal(r)
	if err != nil {
		return DealRecipeSnapshot{}, "", err
	}
	sum := sha256.Sum256(data)
	return r, "sha256:" + hex.EncodeToString(sum[:]), nil
}

func validateRecipePath(value string) error {
	value = strings.TrimSpace(value)
	if value == "" || strings.HasPrefix(value, "/") || strings.Contains(value, "\\") || strings.Contains(value, "\x00") {
		return fmt.Errorf("path must be non-empty relative slash path")
	}
	clean := path.Clean(value)
	if clean != value || clean == "." || strings.HasPrefix(clean, "../") || clean == ".verstak" || strings.HasPrefix(clean, ".verstak/") {
		return fmt.Errorf("unsafe path %q", value)
	}
	return nil
}

// CreateWorkspaceFromRecipe stages all recipe content before atomically
// publishing the Deal and its UUID-keyed canonical metadata.
func (s *Service) CreateWorkspaceFromRecipe(parentFolderID, name string, recipe DealRecipeSnapshot, refreshBaseline func() error) (ScannedWorkspace, error) {
	recipe, digest, err := recipe.validated()
	if err != nil {
		return ScannedWorkspace{}, err
	}
	name = strings.TrimSpace(name)
	if err := validateEntityName(name); err != nil {
		return ScannedWorkspace{}, err
	}
	s.mu.Lock()
	vaultDir := s.vaultDir
	s.mu.Unlock()
	parentPath := ""
	if parentFolderID != "" {
		f, ok := s.GetFolderByID(parentFolderID)
		if !ok {
			return ScannedWorkspace{}, fmt.Errorf("parent folder not found: %s", parentFolderID)
		}
		parentPath = f.Path
	}
	childRel := joinRelPath(parentPath, name)
	childAbs := filepath.Join(vaultDir, filepath.FromSlash(childRel))
	if _, err := os.Lstat(childAbs); err == nil {
		return ScannedWorkspace{}, fmt.Errorf("conflict: %s already exists", childRel)
	}
	stagingAbs := filepath.Join(vaultDir, filepath.FromSlash(joinRelPath(parentPath, "."+name+".staging."+uuid.NewString()[:8])))
	s.BeginInternalMutation()
	defer os.RemoveAll(stagingAbs)
	if err := os.MkdirAll(stagingAbs, 0o755); err != nil {
		return ScannedWorkspace{}, err
	}
	wsID := uuid.NewString()
	if err := WriteWorkspaceMarker(stagingAbs, wsID); err != nil {
		return ScannedWorkspace{}, err
	}
	for _, folder := range recipe.InitialFolders {
		if err := os.MkdirAll(filepath.Join(stagingAbs, filepath.FromSlash(folder)), 0o755); err != nil {
			return ScannedWorkspace{}, err
		}
	}
	for _, file := range recipe.InitialFiles {
		target := filepath.Join(stagingAbs, filepath.FromSlash(file.Path))
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return ScannedWorkspace{}, err
		}
		if err := os.WriteFile(target, []byte(file.Content), 0o644); err != nil {
			return ScannedWorkspace{}, err
		}
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	metadata := DealMetadata{SchemaVersion: DealMetadataSchemaVersion, WorkspaceID: wsID, WorkspaceName: name, WorkspaceTools: recipe.WorkspaceTools, ToolConfig: recipe.ToolConfig, UpdatedAt: now, CreatedFromTemplate: &TemplateProvenance{TemplateID: recipe.Provenance.TemplateID, TemplateName: recipe.Provenance.TemplateName, TemplateVersion: recipe.Provenance.TemplateVersion, AppliedAt: now, RecipeDigest: digest}}
	if err := s.WriteDealMetadata(metadata); err != nil {
		return ScannedWorkspace{}, err
	}
	metadataPath := canonicalDealMetadataPath(vaultDir, wsID)
	published := false
	defer func() {
		if !published {
			_ = os.Remove(metadataPath)
		}
	}()
	if err := os.Rename(stagingAbs, childAbs); err != nil {
		return ScannedWorkspace{}, err
	}
	published = true
	if err := s.EndInternalMutationAndRefreshBaseline(refreshBaseline); err != nil {
		return ScannedWorkspace{}, err
	}
	if err := s.SetCurrentWorkspaceID(wsID); err != nil {
		return ScannedWorkspace{}, err
	}
	workspace, _ := s.GetWorkspaceByID(wsID)
	return workspace, nil
}
