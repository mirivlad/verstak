package workspacetree

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/google/uuid"
)

// ── Create Folder ────────────────────────────────────────────────────────────

// CreateFolder creates a new organizational folder under parentFolderID (empty = root).
func (s *Service) CreateFolder(parentFolderID, name string, refreshBaseline func() error) (ScannedFolder, error) {
	name = strings.TrimSpace(name)
	if err := validateEntityName(name); err != nil {
		return ScannedFolder{}, err
	}

	s.mu.Lock()
	vaultDir := s.vaultDir
	s.mu.Unlock()

	// Resolve parent path.
	parentPath := ""
	if parentFolderID != "" {
		f, ok := s.GetFolderByID(parentFolderID)
		if !ok {
			return ScannedFolder{}, fmt.Errorf("parent folder not found: %s", parentFolderID)
		}
		parentPath = f.Path
	}

	childRel := joinRelPath(parentPath, name)
	childAbs := filepath.Join(vaultDir, filepath.FromSlash(childRel))

	// Check collision.
	if _, err := os.Lstat(childAbs); err == nil {
		return ScannedFolder{}, fmt.Errorf("conflict: %s already exists", childRel)
	}

	// Create staging directory in parent.
	stagingName := "." + name + ".staging." + uuid.NewString()[:8]
	stagingRel := joinRelPath(parentPath, stagingName)
	stagingAbs := filepath.Join(vaultDir, filepath.FromSlash(stagingRel))

	s.BeginInternalMutation()
	defer func() {
		// Cleanup staging on failure.
		if _, err := os.Lstat(stagingAbs); err == nil {
			_ = os.RemoveAll(stagingAbs)
		}
	}()

	if err := os.MkdirAll(stagingAbs, 0o755); err != nil {
		return ScannedFolder{}, err
	}

	newID := uuid.NewString()
	if err := WriteFolderMarker(stagingAbs, newID); err != nil {
		return ScannedFolder{}, err
	}

	// Atomic rename.
	if err := os.Rename(stagingAbs, childAbs); err != nil {
		return ScannedFolder{}, err
	}
	// Staging is gone now — disable cleanup.
	_ = os.RemoveAll(stagingAbs) // no-op if rename succeeded

	// Refresh.
	if err := s.EndInternalMutationAndRefreshBaseline(refreshBaseline); err != nil {
		return ScannedFolder{}, err
	}

	f, _ := s.GetFolderByID(newID)
	return f, nil
}

// ── Create Workspace ─────────────────────────────────────────────────────────

// CreateWorkspace creates a new workspace under parentFolderID (empty = root).
func (s *Service) CreateWorkspace(parentFolderID, name, templateID string, refreshBaseline func() error) (ScannedWorkspace, error) {
	return s.createWorkspace(parentFolderID, name, templateID, resolveTemplateTools(templateID), refreshBaseline)
}

// CreateWorkspaceWithTools creates a workspace with an explicit definitive tool set.
func (s *Service) CreateWorkspaceWithTools(parentFolderID, name, templateID string, workspaceTools []string, refreshBaseline func() error) (ScannedWorkspace, error) {
	return s.createWorkspace(parentFolderID, name, templateID, normalizeWorkspaceTools(workspaceTools), refreshBaseline)
}

// UpdateWorkspaceTools replaces the definitive workspace tool set without deleting
// provider data. Folders required by newly enabled tools are created if missing.
func (s *Service) UpdateWorkspaceTools(workspaceID string, workspaceTools []string, refreshBaseline func() error) error {
	ws, ok := s.GetWorkspaceByID(workspaceID)
	if !ok {
		return fmt.Errorf("workspace not found: %s", workspaceID)
	}
	tools := normalizeWorkspaceTools(workspaceTools)

	s.mu.Lock()
	vaultDir := s.vaultDir
	s.mu.Unlock()
	workspaceDir := filepath.Join(vaultDir, filepath.FromSlash(ws.RootPath))
	if err := applyWorkspaceToolFolders(workspaceDir, tools); err != nil {
		return err
	}

	metadata, err := s.ReadDealMetadata(workspaceID, ws.RootPath)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return err
		}
		metadata = DealMetadata{
			SchemaVersion: DealMetadataSchemaVersion,
			WorkspaceID:   workspaceID,
			WorkspaceName: ws.Name,
			ToolConfig:    map[string]json.RawMessage{},
		}
	}
	metadata.WorkspaceName = ws.Name
	metadata.WorkspaceTools = tools
	metadata.UpdatedAt = ""
	if err := s.WriteDealMetadata(metadata); err != nil {
		return err
	}
	if refreshBaseline != nil {
		return refreshBaseline()
	}
	return nil
}

func normalizeWorkspaceTools(workspaceTools []string) []string {
	tools := make([]string, 0, len(workspaceTools))
	seen := make(map[string]bool, len(workspaceTools))
	for _, toolID := range workspaceTools {
		toolID = strings.TrimSpace(toolID)
		if toolID == "" || seen[toolID] {
			continue
		}
		seen[toolID] = true
		tools = append(tools, toolID)
	}
	return tools
}

func (s *Service) createWorkspace(parentFolderID, name, templateID string, workspaceTools []string, refreshBaseline func() error) (ScannedWorkspace, error) {
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

	stagingName := "." + name + ".staging." + uuid.NewString()[:8]
	stagingRel := joinRelPath(parentPath, stagingName)

	stagingAbs := filepath.Join(vaultDir, filepath.FromSlash(stagingRel))

	s.BeginInternalMutation()
	defer func() {
		_ = os.RemoveAll(stagingAbs)
	}()

	if err := os.MkdirAll(stagingAbs, 0o755); err != nil {
		return ScannedWorkspace{}, err
	}

	wsID := uuid.NewString()
	if err := WriteWorkspaceMarker(stagingAbs, wsID); err != nil {
		return ScannedWorkspace{}, err
	}

	// Apply template.
	if templateID != "" {
		if err := applyWorkspaceTemplate(stagingAbs, templateID); err != nil {
			return ScannedWorkspace{}, err
		}
	}
	if err := applyWorkspaceToolFolders(stagingAbs, workspaceTools); err != nil {
		return ScannedWorkspace{}, err
	}

	metadata := newDealMetadata(wsID, name, templateID, workspaceTools)
	if err := s.WriteDealMetadata(metadata); err != nil {
		return ScannedWorkspace{}, err
	}
	metadataPath := canonicalDealMetadataPath(vaultDir, wsID)
	metadataPublished := false
	defer func() {
		if metadataPublished {
			return
		}
		_ = os.Remove(metadataPath)
	}()

	// Atomic rename.
	if err := os.Rename(stagingAbs, childAbs); err != nil {
		return ScannedWorkspace{}, err
	}
	_ = os.RemoveAll(stagingAbs)
	metadataPublished = true

	if err := s.EndInternalMutationAndRefreshBaseline(refreshBaseline); err != nil {
		return ScannedWorkspace{}, err
	}

	// Select the new workspace.
	if err := s.SetCurrentWorkspaceID(wsID); err != nil {
		return ScannedWorkspace{}, err
	}

	ws, _ := s.GetWorkspaceByID(wsID)
	return ws, nil
}

// ── Rename ───────────────────────────────────────────────────────────────────

// RenameFolder renames a folder, preserving its UUID.
func (s *Service) RenameFolder(folderID, newName string, refreshBaseline func() error) (ScannedFolder, error) {
	newName = strings.TrimSpace(newName)
	if err := validateEntityName(newName); err != nil {
		return ScannedFolder{}, err
	}

	s.mu.Lock()
	vaultDir := s.vaultDir
	s.mu.Unlock()

	f, ok := s.GetFolderByID(folderID)
	if !ok {
		return ScannedFolder{}, fmt.Errorf("folder not found: %s", folderID)
	}

	oldAbs := filepath.Join(vaultDir, filepath.FromSlash(f.Path))
	parent := parentPath(f.Path)
	newRel := joinRelPath(parent, newName)
	newAbs := filepath.Join(vaultDir, filepath.FromSlash(newRel))

	if newRel == f.Path {
		return f, nil // no-op
	}
	if _, err := os.Lstat(newAbs); err == nil {
		return ScannedFolder{}, fmt.Errorf("conflict: %s already exists", newRel)
	}

	s.BeginInternalMutation()
	if strings.EqualFold(f.Name, newName) && f.Name != newName {
		// Case-only rename on case-insensitive FS: use intermediate name.
		tmp := newRel + ".case-rename." + uuid.NewString()[:8]
		tmpAbs := filepath.Join(vaultDir, filepath.FromSlash(tmp))
		if err := os.Rename(oldAbs, tmpAbs); err != nil {
			return ScannedFolder{}, err
		}
		if err := os.Rename(tmpAbs, newAbs); err != nil {
			_ = os.Rename(tmpAbs, oldAbs)
			return ScannedFolder{}, err
		}
	} else {
		if err := os.Rename(oldAbs, newAbs); err != nil {
			return ScannedFolder{}, err
		}
	}

	if err := s.EndInternalMutationAndRefreshBaseline(refreshBaseline); err != nil {
		return ScannedFolder{}, err
	}

	updated, _ := s.GetFolderByID(folderID)
	return updated, nil
}

// RenameWorkspace renames a workspace, preserving its UUID.
func (s *Service) RenameWorkspace(workspaceID, newName string, refreshBaseline func() error) (ScannedWorkspace, error) {
	newName = strings.TrimSpace(newName)
	if err := validateEntityName(newName); err != nil {
		return ScannedWorkspace{}, err
	}

	s.mu.Lock()
	vaultDir := s.vaultDir
	s.mu.Unlock()

	ws, ok := s.GetWorkspaceByID(workspaceID)
	if !ok {
		return ScannedWorkspace{}, fmt.Errorf("workspace not found: %s", workspaceID)
	}

	oldAbs := filepath.Join(vaultDir, filepath.FromSlash(ws.RootPath))
	parent := parentPath(ws.RootPath)
	newRel := joinRelPath(parent, newName)
	newAbs := filepath.Join(vaultDir, filepath.FromSlash(newRel))

	if newRel == ws.RootPath {
		return ws, nil
	}
	if _, err := os.Lstat(newAbs); err == nil {
		return ScannedWorkspace{}, fmt.Errorf("conflict: %s already exists", newRel)
	}

	s.BeginInternalMutation()
	if strings.EqualFold(ws.Name, newName) && ws.Name != newName {
		tmp := newRel + ".case-rename." + uuid.NewString()[:8]
		tmpAbs := filepath.Join(vaultDir, filepath.FromSlash(tmp))
		if err := os.Rename(oldAbs, tmpAbs); err != nil {
			return ScannedWorkspace{}, err
		}
		if err := os.Rename(tmpAbs, newAbs); err != nil {
			_ = os.Rename(tmpAbs, oldAbs)
			return ScannedWorkspace{}, err
		}
	} else {
		if err := os.Rename(oldAbs, newAbs); err != nil {
			return ScannedWorkspace{}, err
		}
	}

	if err := s.EndInternalMutationAndRefreshBaseline(refreshBaseline); err != nil {
		return ScannedWorkspace{}, err
	}

	updated, _ := s.GetWorkspaceByID(workspaceID)
	metadata, err := s.ReadDealMetadata(workspaceID, updated.RootPath)
	if err != nil {
		return ScannedWorkspace{}, err
	}
	metadata.WorkspaceName = updated.Name
	metadata.UpdatedAt = ""
	if err := s.WriteDealMetadata(metadata); err != nil {
		return ScannedWorkspace{}, err
	}
	return updated, nil
}

// ── Move ─────────────────────────────────────────────────────────────────────

// MoveFolder moves a folder to a new parent.
func (s *Service) MoveFolder(folderID, targetParentFolderID string, refreshBaseline func() error) (ScannedFolder, error) {
	s.mu.Lock()
	vaultDir := s.vaultDir
	s.mu.Unlock()

	f, ok := s.GetFolderByID(folderID)
	if !ok {
		return ScannedFolder{}, fmt.Errorf("folder not found: %s", folderID)
	}

	// Resolve target parent path.
	targetParentPath := ""
	if targetParentFolderID != "" {
		tf, ok := s.GetFolderByID(targetParentFolderID)
		if !ok {
			return ScannedFolder{}, fmt.Errorf("target parent folder not found: %s", targetParentFolderID)
		}
		targetParentPath = tf.Path
	}

	// Reject moving into itself or descendant.
	if targetParentPath == f.Path || isPathPrefix(f.Path, targetParentPath) {
		return ScannedFolder{}, fmt.Errorf("cannot move folder into itself or descendant")
	}

	newRel := joinRelPath(targetParentPath, f.Name)
	newAbs := filepath.Join(vaultDir, filepath.FromSlash(newRel))

	if newRel == f.Path {
		return f, nil
	}
	if _, err := os.Lstat(newAbs); err == nil {
		return ScannedFolder{}, fmt.Errorf("conflict: %s already exists", newRel)
	}

	s.BeginInternalMutation()
	if err := os.Rename(
		filepath.Join(vaultDir, filepath.FromSlash(f.Path)),
		newAbs,
	); err != nil {
		return ScannedFolder{}, err
	}

	if err := s.EndInternalMutationAndRefreshBaseline(refreshBaseline); err != nil {
		return ScannedFolder{}, err
	}

	updated, _ := s.GetFolderByID(folderID)
	return updated, nil
}

// MoveWorkspace moves a workspace to a new parent folder.
func (s *Service) MoveWorkspace(workspaceID, targetParentFolderID string, refreshBaseline func() error) (ScannedWorkspace, error) {
	s.mu.Lock()
	vaultDir := s.vaultDir
	s.mu.Unlock()

	ws, ok := s.GetWorkspaceByID(workspaceID)
	if !ok {
		return ScannedWorkspace{}, fmt.Errorf("workspace not found: %s", workspaceID)
	}

	targetParentPath := ""
	if targetParentFolderID != "" {
		tf, ok := s.GetFolderByID(targetParentFolderID)
		if !ok {
			return ScannedWorkspace{}, fmt.Errorf("target parent folder not found: %s", targetParentFolderID)
		}
		targetParentPath = tf.Path
	}

	newRel := joinRelPath(targetParentPath, ws.Name)
	newAbs := filepath.Join(vaultDir, filepath.FromSlash(newRel))

	if newRel == ws.RootPath {
		return ws, nil
	}
	if _, err := os.Lstat(newAbs); err == nil {
		return ScannedWorkspace{}, fmt.Errorf("conflict: %s already exists", newRel)
	}

	s.BeginInternalMutation()
	if err := os.Rename(
		filepath.Join(vaultDir, filepath.FromSlash(ws.RootPath)),
		newAbs,
	); err != nil {
		return ScannedWorkspace{}, err
	}

	if err := s.EndInternalMutationAndRefreshBaseline(refreshBaseline); err != nil {
		return ScannedWorkspace{}, err
	}

	updated, _ := s.GetWorkspaceByID(workspaceID)
	return updated, nil
}

// ── Validation ───────────────────────────────────────────────────────────────

func validateEntityName(name string) error {
	if name == "" {
		return fmt.Errorf("name cannot be empty")
	}
	if strings.Contains(name, "/") || strings.Contains(name, "\\") {
		return fmt.Errorf("name cannot contain path separators")
	}
	if name == "." || name == ".." {
		return fmt.Errorf("invalid name: %s", name)
	}
	if strings.HasPrefix(name, ".") {
		return fmt.Errorf("name cannot start with dot")
	}
	for _, r := range name {
		if r < 32 {
			return fmt.Errorf("name contains control character")
		}
	}
	return nil
}

// applyWorkspaceTemplate creates default folders for a new workspace.
func applyWorkspaceTemplate(workspaceDir, templateID string) error {
	// Default template: Notes + Files.
	folders := []string{"Notes", "Files"}
	for _, folder := range folders {
		if err := os.MkdirAll(filepath.Join(workspaceDir, folder), 0o755); err != nil {
			return err
		}
	}
	return nil
}

func applyWorkspaceToolFolders(workspaceDir string, workspaceTools []string) error {
	folders := toolsToFolders(workspaceTools)
	for _, folder := range folders {
		if err := os.MkdirAll(filepath.Join(workspaceDir, folder), 0o755); err != nil {
			return err
		}
	}
	return nil
}

var templateRegistry = map[string]templateDef{
	"default":        {ID: "default", Name: "General", Description: "Everyday workspace with notes, files, journal, activity, and browser captures.", Version: 2, Selectable: true, Order: 10, WorkspaceTools: []string{"verstak.notes", "verstak.files", "verstak.journal", "verstak.activity", "verstak.browser-inbox"}},
	"project":        {ID: "project", Name: "Project", Description: "Project planning with todos, journal, activity, and browser captures.", Version: 1, Selectable: true, Order: 20, WorkspaceTools: []string{"verstak.projects", "verstak.notes", "verstak.files", "verstak.todo", "verstak.journal", "verstak.activity", "verstak.browser-inbox"}},
	"writing":        {ID: "writing", Name: "Writing", Description: "Focused notes, files, and journal workspace for documentation and writing.", Version: 1, Selectable: true, Order: 30, WorkspaceTools: []string{"verstak.notes", "verstak.files", "verstak.journal"}},
	"admin":          {ID: "admin", Name: "Admin", Description: "Infrastructure workspace with secrets, todos, and journal.", Version: 1, Selectable: true, Order: 40, WorkspaceTools: []string{"verstak.notes", "verstak.files", "verstak.secrets", "verstak.todo", "verstak.journal"}},
	"minimal":        {ID: "minimal", Name: "Minimal", Description: "Only notes and files for a lightweight workspace.", Version: 1, Selectable: true, Order: 50, WorkspaceTools: []string{"verstak.notes", "verstak.files"}},
	"client-project": {ID: "client-project", Name: "Client Project", Description: "Legacy client project template retained for existing integrations.", Version: 1, Selectable: false, WorkspaceTools: []string{"verstak.notes", "verstak.files", "verstak.secrets"}},
}

// templateDefinition mirrors the built-in template structure from workspace package.
type templateDef struct {
	ID             string
	Name           string
	Description    string
	Version        int
	WorkspaceTools []string
	Selectable     bool
	Order          int
}

// DealTemplateSummary is the temporary read model for built-in recipes until
// the Templates plugin owns CRUD persistence.
type DealTemplateSummary struct {
	ID             string
	Name           string
	Description    string
	Version        int
	WorkspaceTools []string
}

// ListDealTemplates returns selectable recipes in stable UI order.
func ListDealTemplates() []DealTemplateSummary {
	definitions := make([]templateDef, 0, len(templateRegistry))
	for _, definition := range templateRegistry {
		if definition.Selectable {
			definitions = append(definitions, definition)
		}
	}
	sort.Slice(definitions, func(i, j int) bool { return definitions[i].Order < definitions[j].Order })
	result := make([]DealTemplateSummary, 0, len(definitions))
	for _, definition := range definitions {
		result = append(result, DealTemplateSummary{
			ID:             definition.ID,
			Name:           definition.Name,
			Description:    definition.Description,
			Version:        definition.Version,
			WorkspaceTools: append([]string(nil), definition.WorkspaceTools...),
		})
	}
	return result
}

// resolveTemplateTools returns the workspaceTools list for a template ID.
func resolveTemplateTools(templateID string) []string {
	if templateID == "" {
		return []string{"verstak.notes", "verstak.files"}
	}
	tmpl, ok := templateRegistry[templateID]
	if !ok || !tmpl.Selectable {
		return []string{"verstak.notes", "verstak.files"}
	}
	return append([]string(nil), tmpl.WorkspaceTools...)
}

// toolsToFeatures derives a features map from a workspaceTools list.
func toolsToFeatures(tools []string) map[string]bool {
	f := make(map[string]bool)
	for _, t := range tools {
		switch t {
		case "verstak.files":
			f["files"] = true
		case "verstak.projects":
			f["projects"] = true
		case "verstak.notes":
			f["notes"] = true
		case "verstak.journal":
			f["journal"] = true
		case "verstak.activity":
			f["activity"] = true
		case "verstak.browser-inbox":
			f["browser-inbox"] = true
		case "verstak.todo":
			f["todo"] = true
		case "verstak.secrets":
			f["secrets"] = true
		}
	}
	return f
}

func toolsToFolders(tools []string) map[string]string {
	folders := make(map[string]string)
	for _, toolID := range tools {
		switch toolID {
		case "verstak.notes":
			folders["notes"] = "Notes"
		case "verstak.files":
			folders["files"] = "Files"
		case "verstak.secrets":
			folders["secrets"] = "Secrets"
		}
	}
	return folders
}

// ── Set current workspace ────────────────────────────────────────────────────

// SetCurrentWorkspaceID sets the current workspace by UUID.
func (s *Service) SetCurrentWorkspaceID(id string) error {
	s.mu.Lock()
	if s.scan != nil {
		if _, ok := s.scan.Workspaces[id]; !ok && id != "" {
			s.mu.Unlock()
			return fmt.Errorf("workspace not found: %s", id)
		}
	}
	s.mu.Unlock()
	if err := s.writeWorkspaceUIState(id); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.currentWS = id
	if s.tree != nil {
		s.tree.CurrentWorkspaceID = id
	}
	return nil
}

// GetCurrentWorkspaceID returns the current workspace UUID.
func (s *Service) GetCurrentWorkspaceID() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.currentWS
}
