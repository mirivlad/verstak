// Package workspace provides the semantic workspace lifecycle service.
//
// A workspace is any folder under the vault root containing a
// .verstak/workspace.json identity marker. The filesystem is the source
// of truth for workspace existence and listing. Workspaces may be nested
// inside ordinary folders; a workspace folder may not contain another
// workspace.
//
// Metadata under .verstak stores UI state, creation snapshots, and
// folder-level metadata (icon, color, order).
package workspace

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/google/uuid"
)

// NodeType is retained for compatibility with the current shell API.
type NodeType string

const (
	TypeSpace  NodeType = "space"
	TypeCase   NodeType = "case"
	TypeFolder NodeType = "folder"
)

// NodeStatus is retained for compatibility with the current shell API.
type NodeStatus string

const (
	StatusActive   NodeStatus = "active"
	StatusSleeping NodeStatus = "sleeping"
	StatusArchived NodeStatus = "archived"
)

// Workspace is a workspace folder anywhere under the vault root.
type Workspace struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Path     string `json:"path"`
	RootPath string `json:"rootPath"` // deprecated, equals Path
}

// FolderMetadata stores user-visible folder attributes (icon, color, order).
// Workspace identity markers are stored under .verstak/workspace.json.
type FolderMetadata struct {
	Icon      string `json:"icon,omitempty"`
	Color     string `json:"color,omitempty"`
	Order     int    `json:"order,omitempty"`
	UpdatedAt string `json:"updatedAt,omitempty"`
}

// TemplateSnapshot is copied into workspace metadata when a template is applied.
type TemplateSnapshot struct {
	TemplateID      string   `json:"templateId"`
	TemplateName    string   `json:"templateName"`
	TemplateVersion int      `json:"templateVersion"`
	AppliedAt       string   `json:"appliedAt"`
	WorkspaceTools  []string `json:"workspaceTools,omitempty"`
}

// WorkspaceTemplate describes a selectable built-in template without exposing
// its filesystem implementation details.
type WorkspaceTemplate struct {
	ID             string   `json:"id"`
	Name           string   `json:"name"`
	Description    string   `json:"description"`
	Version        int      `json:"version"`
	WorkspaceTools []string `json:"workspaceTools"`
}

// Metadata stores semantic workspace metadata that is not the source of truth
// for whether the workspace exists.
type Metadata struct {
	WorkspaceID         string            `json:"workspaceId,omitempty"`
	WorkspaceName       string            `json:"workspaceName"`
	WorkspacePath       string            `json:"workspacePath,omitempty"`
	CreatedFromTemplate *TemplateSnapshot `json:"createdFromTemplate,omitempty"`
	Features            map[string]bool   `json:"features,omitempty"`
	Folders             map[string]string `json:"folders,omitempty"`
	WorkspaceTools      []string          `json:"workspaceTools,omitempty"`
	UpdatedAt           string            `json:"updatedAt,omitempty"`
}

// FolderNode represents a virtual folder grouping workspaces in the tree.
type FolderNode struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Icon      string `json:"icon,omitempty"`
	Color     string `json:"color,omitempty"`
	Order     int    `json:"order,omitempty"`
	Path      string `json:"path"`
	HasMarker bool   `json:"hasMarker"` // true if this is itself a workspace
}

// TreeNode is a unified tree node: workspace or plain folder.
type TreeNode struct {
	ID        string     `json:"id"`
	ParentID  string     `json:"parentId,omitempty"`
	Type      NodeType   `json:"type"`
	Title     string     `json:"title"`
	Name      string     `json:"name,omitempty"`
	Path      string     `json:"path"`
	RootPath  string     `json:"rootPath,omitempty"`
	Icon      string     `json:"icon,omitempty"`
	Color     string     `json:"color,omitempty"`
	Status    NodeStatus `json:"status,omitempty"`
	Order     int        `json:"order"`
	CreatedAt string     `json:"createdAt,omitempty"`
	UpdatedAt string     `json:"updatedAt,omitempty"`
}

const workspaceIdentityRelativePath = ".verstak/workspace.json"

type workspaceIdentityMarker struct {
	WorkspaceID string `json:"workspaceId"`
}

// MetadataPatch updates metadata fields without replacing unspecified fields.
type MetadataPatch struct {
	Features map[string]bool   `json:"features,omitempty"`
	Folders  map[string]string `json:"folders,omitempty"`
}

// TrashResult describes a workspace moved into the internal trash area.
type TrashResult struct {
	WorkspaceID  string `json:"workspaceId"`
	OriginalPath string `json:"originalPath"`
	TrashPath    string `json:"trashPath"`
	TrashID      string `json:"trashId"`
	DeletedAt    string `json:"deletedAt"`
}

// WorkspaceIdentity identifies an existing workspace independently of its path.
type WorkspaceIdentity struct {
	WorkspaceID string `json:"workspaceId"`
	RootPath    string `json:"rootPath"`
	State       string `json:"state"`
}

// WorkspaceNode is a compatibility shell view of a workspace.
type WorkspaceNode struct {
	ID        string     `json:"id"`
	ParentID  string     `json:"parentId,omitempty"`
	Type      NodeType   `json:"type"`
	Title     string     `json:"title"`
	Name      string     `json:"name,omitempty"`
	RootPath  string     `json:"rootPath,omitempty"`
	Path      string     `json:"-"`
	Status    NodeStatus `json:"status"`
	Tags      []string   `json:"tags,omitempty"`
	Order     int        `json:"order"`
	CreatedAt string     `json:"createdAt,omitempty"`
	UpdatedAt string     `json:"updatedAt,omitempty"`
}

// WorkspaceTree is a compatibility flat list, derived from vault folders.
type WorkspaceTree struct {
	SchemaVersion int             `json:"schemaVersion"`
	Nodes         []WorkspaceNode `json:"nodes"`
	CurrentNodeID string          `json:"currentNodeId,omitempty"`
	UpdatedAt     string          `json:"updatedAt"`
}

type templateDefinition struct {
	ID             string
	Name           string
	Description    string
	Version        int
	Features       map[string]bool
	Folders        map[string]string
	Files          map[string]string
	WorkspaceTools []string
	Selectable     bool
	Order          int
}

var builtInTemplates = map[string]templateDefinition{
	"default": {
		ID:          "default",
		Name:        "General",
		Description: "Everyday workspace with notes, files, journal, activity, and browser captures.",
		Version:     2,
		Features: map[string]bool{
			"files":         true,
			"notes":         true,
			"secrets":       false,
			"activity":      true,
			"journal":       true,
			"browser-inbox": true,
		},
		Folders: map[string]string{
			"notes": "Notes",
			"files": "Files",
		},
		Files:          map[string]string{},
		WorkspaceTools: []string{"verstak.notes", "verstak.files", "verstak.journal", "verstak.activity", "verstak.browser-inbox"},
		Selectable:     true,
		Order:          10,
	},
	"project": {
		ID:          "project",
		Name:        "Project",
		Description: "Project planning with todos, journal, activity, and browser captures.",
		Version:     1,
		Features: map[string]bool{
			"files":         true,
			"notes":         true,
			"todo":          true,
			"journal":       true,
			"activity":      true,
			"browser-inbox": true,
		},
		Folders: map[string]string{
			"notes": "Notes",
			"files": "Files",
		},
		Files:          map[string]string{},
		WorkspaceTools: []string{"verstak.notes", "verstak.files", "verstak.todo", "verstak.journal", "verstak.activity", "verstak.browser-inbox"},
		Selectable:     true,
		Order:          20,
	},
	"writing": {
		ID:          "writing",
		Name:        "Writing",
		Description: "Focused notes, files, and journal workspace for documentation and writing.",
		Version:     1,
		Features: map[string]bool{
			"files":   true,
			"notes":   true,
			"journal": true,
		},
		Folders: map[string]string{
			"notes": "Notes",
			"files": "Files",
		},
		Files:          map[string]string{},
		WorkspaceTools: []string{"verstak.notes", "verstak.files", "verstak.journal"},
		Selectable:     true,
		Order:          30,
	},
	"admin": {
		ID:          "admin",
		Name:        "Admin",
		Description: "Infrastructure workspace with secrets, todos, and journal.",
		Version:     1,
		Features: map[string]bool{
			"files":   true,
			"notes":   true,
			"secrets": true,
			"todo":    true,
			"journal": true,
		},
		Folders: map[string]string{
			"notes":   "Notes",
			"files":   "Files",
			"secrets": "Secrets",
		},
		Files:          map[string]string{},
		WorkspaceTools: []string{"verstak.notes", "verstak.files", "verstak.secrets", "verstak.todo", "verstak.journal"},
		Selectable:     true,
		Order:          40,
	},
	"minimal": {
		ID:          "minimal",
		Name:        "Minimal",
		Description: "Only notes and files for a lightweight workspace.",
		Version:     1,
		Features: map[string]bool{
			"files": true,
			"notes": true,
		},
		Folders: map[string]string{
			"notes": "Notes",
			"files": "Files",
		},
		Files:          map[string]string{},
		WorkspaceTools: []string{"verstak.notes", "verstak.files"},
		Selectable:     true,
		Order:          50,
	},
	"client-project": {
		ID:          "client-project",
		Name:        "Client Project",
		Description: "Legacy client project template retained for existing integrations.",
		Version:     1,
		Features: map[string]bool{
			"files":    true,
			"notes":    true,
			"secrets":  true,
			"activity": false,
		},
		Folders: map[string]string{
			"notes":   "Notes",
			"files":   "Files",
			"secrets": "Secrets",
		},
		Files:          map[string]string{},
		WorkspaceTools: []string{"verstak.notes", "verstak.files", "verstak.secrets"},
		Selectable:     false,
		Order:          0,
	},
}

const folderMetadataRelativePath = ".verstak/folder-metadata.json"

// Manager provides workspace operations for one vault.
type Manager struct {
	mu                   sync.RWMutex
	vaultDir             string
	initialized          bool
	currentWorkspacePath string
}

// NewManager creates a workspace manager for the given vault directory.
func NewManager(vaultDir string) *Manager {
	return &Manager{vaultDir: vaultDir}
}

// Load initializes the manager without creating or migrating workspace folders.
func (m *Manager) Load() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.initialized = true
	m.currentWorkspacePath = m.readSelectedWorkspaceLocked()
	return nil
}

// IsInitialized returns true after Load has been called.
func (m *Manager) IsInitialized() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.initialized
}

// ListWorkspaces returns workspace folders from the vault, discovered
// recursively by scanning for .verstak/workspace.json markers.
func (m *Manager) ListWorkspaces() ([]Workspace, error) {
	result := make([]Workspace, 0)
	if err := m.walkForWorkspaces(m.vaultDir, "", &result); err != nil {
		return nil, err
	}
	sort.Slice(result, func(i, j int) bool {
		return strings.ToLower(result[i].Path) < strings.ToLower(result[j].Path)
	})
	return result, nil
}

func (m *Manager) walkForWorkspaces(base string, relPrefix string, result *[]Workspace) error {
	entries, err := os.ReadDir(base)
	if err != nil {
		return nil // directory removed, skip
	}
	for _, entry := range entries {
		name := entry.Name()
		if isReservedWorkspaceName(name) {
			continue
		}
		if entry.Type()&os.ModeSymlink != 0 {
			continue
		}
		if !entry.IsDir() {
			continue
		}
		relPath := joinWorkspacePath(relPrefix, name)
		full := filepath.Join(base, name)
		markerPath := filepath.Join(full, workspaceIdentityRelativePath)
		if _, err := os.Stat(markerPath); err == nil {
			workspaceID, err := ensureWorkspaceIdentity(full)
			if err != nil {
				return err
			}
			*result = append(*result, Workspace{ID: workspaceID, Name: name, Path: relPath, RootPath: relPath})
			// Stop recursion: workspace folders cannot contain other workspaces
			continue
		} else if !os.IsNotExist(err) {
			// Permission error or similar — skip this folder
			continue
		}
		// No marker: recurse into plain folder
		if err := m.walkForWorkspaces(full, relPath, result); err != nil {
			return err
		}
	}
	return nil
}

// ListWorkspaceTemplates returns selectable built-ins in their presentation order.
func (m *Manager) ListWorkspaceTemplates() []WorkspaceTemplate {
	templates := make([]WorkspaceTemplate, 0, len(builtInTemplates))
	for _, template := range builtInTemplates {
		if !template.Selectable {
			continue
		}
		templates = append(templates, WorkspaceTemplate{
			ID:             template.ID,
			Name:           template.Name,
			Description:    template.Description,
			Version:        template.Version,
			WorkspaceTools: cloneStringSlice(template.WorkspaceTools),
		})
	}
	sort.SliceStable(templates, func(i, j int) bool {
		return builtInTemplates[templates[i].ID].Order < builtInTemplates[templates[j].ID].Order
	})
	return templates
}

// GetWorkspaceIdentity resolves the durable identity for an existing workspace.
func (m *Manager) GetWorkspaceIdentity(path string) (WorkspaceIdentity, error) {
	path = strings.TrimSpace(path)
	if err := validateWorkspacePath(path); err != nil {
		return WorkspaceIdentity{}, err
	}
	full := filepath.Join(m.vaultDir, filepath.FromSlash(path))
	if err := ensureExistingWorkspaceDir(full, path); err != nil {
		return WorkspaceIdentity{}, err
	}
	workspaceID, err := ensureWorkspaceIdentity(full)
	if err != nil {
		return WorkspaceIdentity{}, err
	}
	return WorkspaceIdentity{WorkspaceID: workspaceID, RootPath: path, State: "active"}, nil
}

// ListWorkspaceIdentities returns durable identities and flags copied markers.
func (m *Manager) ListWorkspaceIdentities() ([]WorkspaceIdentity, error) {
	workspaces, err := m.ListWorkspaces()
	if err != nil {
		return nil, err
	}
	counts := make(map[string]int, len(workspaces))
	for _, ws := range workspaces {
		counts[ws.ID]++
	}
	identities := make([]WorkspaceIdentity, 0, len(workspaces))
	for _, ws := range workspaces {
		state := "active"
		if counts[ws.ID] > 1 {
			state = "duplicate"
		}
		identities = append(identities, WorkspaceIdentity{
			WorkspaceID: ws.ID,
			RootPath:    ws.RootPath,
			State:       state,
		})
	}
	return identities, nil
}

// RepairWorkspaceIdentity keeps the identity on one copied workspace and regenerates the other.
func (m *Manager) RepairWorkspaceIdentity(keepPath, regeneratePath string) error {
	keepPath = strings.TrimSpace(keepPath)
	regeneratePath = strings.TrimSpace(regeneratePath)
	if err := validateWorkspacePath(keepPath); err != nil {
		return err
	}
	if err := validateWorkspacePath(regeneratePath); err != nil {
		return err
	}
	if keepPath == regeneratePath {
		return fmt.Errorf("workspace identity repair requires two workspaces")
	}
	keepFull := filepath.Join(m.vaultDir, filepath.FromSlash(keepPath))
	regenerateFull := filepath.Join(m.vaultDir, filepath.FromSlash(regeneratePath))
	if err := ensureExistingWorkspaceDir(keepFull, keepPath); err != nil {
		return err
	}
	if err := ensureExistingWorkspaceDir(regenerateFull, regeneratePath); err != nil {
		return err
	}
	keepID, err := ensureWorkspaceIdentity(keepFull)
	if err != nil {
		return err
	}
	regenerateID, err := ensureWorkspaceIdentity(regenerateFull)
	if err != nil {
		return err
	}
	if keepID != regenerateID {
		return fmt.Errorf("workspaces do not share an identity")
	}
	_, err = writeWorkspaceIdentity(regenerateFull)
	return err
}

// CreateWorkspace creates a workspace folder at path and applies a template once.
// path is a vault-relative slash path; parent folders are created automatically
// as plain folders (no marker).
func (m *Manager) CreateWorkspace(path, templateID string) (Workspace, error) {
	path = strings.TrimSpace(path)
	if err := validateWorkspacePath(path); err != nil {
		return Workspace{}, err
	}
	if templateID == "" {
		templateID = "default"
	}
	template, ok := builtInTemplates[templateID]
	if !ok {
		return Workspace{}, fmt.Errorf("template-not-found: %s", templateID)
	}

	name := filepath.Base(filepath.FromSlash(path))
	parentPath := filepath.Dir(filepath.FromSlash(path))
	if parentPath != "." {
		parentFull := filepath.Join(m.vaultDir, parentPath)
		if _, err := os.Stat(parentFull); os.IsNotExist(err) {
			return Workspace{}, fmt.Errorf("parent-not-found: %s", filepath.ToSlash(parentPath))
		}
		// Check parent is not a workspace
		if _, err := os.Stat(filepath.Join(parentFull, workspaceIdentityRelativePath)); err == nil {
			return Workspace{}, fmt.Errorf("parent-is-workspace: %s", filepath.ToSlash(parentPath))
		}
	}

	full := filepath.Join(m.vaultDir, filepath.FromSlash(path))
	if _, err := os.Lstat(full); err == nil {
		return Workspace{}, fmt.Errorf("conflict: %s", path)
	} else if !os.IsNotExist(err) {
		return Workspace{}, err
	}

	// Create parent plain folders
	plainParent := filepath.Join(m.vaultDir, parentPath)
	if parentPath != "." {
		if err := os.MkdirAll(plainParent, 0o755); err != nil {
			return Workspace{}, err
		}
	}

	if err := os.Mkdir(full, 0o755); err != nil {
		return Workspace{}, err
	}
	created := true
	defer func() {
		if created {
			_ = os.RemoveAll(full)
			_ = cleanupEmptyParents(m.vaultDir, parentPath)
		}
	}()

	workspaceID, err := ensureWorkspaceIdentity(full)
	if err != nil {
		return Workspace{}, err
	}
	if err := applyTemplate(full, template); err != nil {
		return Workspace{}, err
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	meta := Metadata{
		WorkspaceID:   workspaceID,
		WorkspaceName: name,
		WorkspacePath: path,
		CreatedFromTemplate: &TemplateSnapshot{
			TemplateID:      template.ID,
			TemplateName:    template.Name,
			TemplateVersion: template.Version,
			AppliedAt:       now,
			WorkspaceTools:  cloneStringSlice(template.WorkspaceTools),
		},
		Features:       cloneBoolMap(template.Features),
		Folders:        cloneStringMap(template.Folders),
		WorkspaceTools: cloneStringSlice(template.WorkspaceTools),
		UpdatedAt:      now,
	}
	if err := m.writeMetadata(path, meta); err != nil {
		return Workspace{}, err
	}

	created = false
	return Workspace{ID: workspaceID, Name: name, Path: path, RootPath: path}, nil
}

// RenameWorkspace physically renames/moves a workspace folder.
func (m *Manager) RenameWorkspace(oldPath, newPath string) error {
	oldPath = strings.TrimSpace(oldPath)
	newPath = strings.TrimSpace(newPath)
	if err := validateWorkspacePath(oldPath); err != nil {
		return err
	}
	if err := validateWorkspacePath(newPath); err != nil {
		return err
	}
	oldFull := filepath.Join(m.vaultDir, filepath.FromSlash(oldPath))
	newFull := filepath.Join(m.vaultDir, filepath.FromSlash(newPath))

	if err := ensureExistingWorkspaceDir(oldFull, oldPath); err != nil {
		return err
	}

	// Check new path parent exists
	newParentDir := filepath.Dir(newFull)
	if _, err := os.Stat(newParentDir); os.IsNotExist(err) {
		return fmt.Errorf("parent-not-found: %s", filepath.Dir(filepath.ToSlash(newPath)))
	}

	if _, err := os.Lstat(newFull); err == nil {
		return fmt.Errorf("conflict: %s", newPath)
	} else if !os.IsNotExist(err) {
		return err
	}

	// Check not moving workspace into another workspace
	newParentVaultRel := filepath.Dir(filepath.FromSlash(newPath))
	if newParentVaultRel != "." {
		newParentFull := filepath.Join(m.vaultDir, filepath.FromSlash(newParentVaultRel))
		if _, err := os.Stat(filepath.Join(newParentFull, workspaceIdentityRelativePath)); err == nil {
			return fmt.Errorf("parent-is-workspace: %s", newParentVaultRel)
		}
	}

	// Create intermediate plain folders if needed
	if err := os.MkdirAll(newParentDir, 0o755); err != nil {
		return err
	}

	if err := os.Rename(oldFull, newFull); err != nil {
		return err
	}
	renamed := true
	defer func() {
		if renamed {
			_ = os.Rename(newFull, oldFull)
			_ = cleanupEmptyParents(m.vaultDir, filepath.Dir(filepath.FromSlash(oldPath)))
		}
	}()

	// Move metadata
	oldMetaPath := m.metadataPath(oldPath)
	if data, err := os.ReadFile(oldMetaPath); err == nil {
		var meta Metadata
		if err := json.Unmarshal(data, &meta); err != nil {
			return err
		}
		meta.WorkspaceName = filepath.Base(filepath.FromSlash(newPath))
		meta.WorkspacePath = newPath
		meta.UpdatedAt = time.Now().UTC().Format(time.RFC3339Nano)
		if err := m.writeMetadata(newPath, meta); err != nil {
			return err
		}
		if err := os.Remove(oldMetaPath); err != nil && !os.IsNotExist(err) {
			return err
		}
	} else if !os.IsNotExist(err) {
		return err
	}

	// Move folder metadata if new parent is different
	oldFolderRel := filepath.Dir(filepath.FromSlash(oldPath))
	newFolderRel := filepath.Dir(filepath.FromSlash(newPath))
	if oldFolderRel != newFolderRel {
		_ = os.Remove(oldFolderMetadataPath(m.vaultDir, oldFolderRel))
	}

	// Cleanup empty parent folders
	_ = cleanupEmptyParents(m.vaultDir, filepath.Dir(filepath.FromSlash(oldPath)))

	m.mu.Lock()
	if m.currentWorkspacePath == oldPath {
		m.currentWorkspacePath = newPath
		_ = m.writeUIStateLocked()
	}
	m.mu.Unlock()

	renamed = false
	return nil
}

// TrashWorkspace moves the whole workspace folder to internal trash.
func (m *Manager) TrashWorkspace(path string) (TrashResult, error) {
	path = strings.TrimSpace(path)
	if err := validateWorkspacePath(path); err != nil {
		return TrashResult{}, err
	}
	full := filepath.Join(m.vaultDir, filepath.FromSlash(path))
	if err := ensureExistingWorkspaceDir(full, path); err != nil {
		return TrashResult{}, err
	}
	workspaceID, err := ensureWorkspaceIdentity(full)
	if err != nil {
		return TrashResult{}, err
	}

	deletedAt := time.Now().UTC().Format(time.RFC3339Nano)
	trashID := time.Now().UTC().Format("20060102T150405.000000000Z") + "-" + uuid.NewString()
	name := filepath.Base(filepath.FromSlash(path))
	trashRel := filepath.ToSlash(filepath.Join(".verstak", "trash", "workspaces", trashID, name))
	trashFull := filepath.Join(m.vaultDir, filepath.FromSlash(trashRel))
	if err := os.MkdirAll(filepath.Dir(trashFull), 0o755); err != nil {
		return TrashResult{}, err
	}
	if err := os.Rename(full, trashFull); err != nil {
		return TrashResult{}, err
	}

	// Cleanup empty parent folders
	_ = cleanupEmptyParents(m.vaultDir, filepath.Dir(filepath.FromSlash(path)))

	result := TrashResult{WorkspaceID: workspaceID, OriginalPath: path, TrashPath: trashRel, TrashID: trashID, DeletedAt: deletedAt}
	trashMeta := map[string]string{
		"originalPath": path,
		"trashPath":    trashRel,
		"trashId":      trashID,
		"deletedAt":    deletedAt,
		"originalType": "folder",
		"basename":     name,
		"type":         "workspace",
	}
	data, err := json.MarshalIndent(trashMeta, "", "  ")
	if err != nil {
		return TrashResult{}, err
	}
	trashDir := filepath.Join(m.vaultDir, ".verstak", "trash", "workspaces", trashID)
	if err := os.WriteFile(filepath.Join(trashDir, "metadata.json"), data, 0o644); err != nil {
		return TrashResult{}, err
	}
	if err := moveIfExists(m.metadataPath(path), filepath.Join(trashDir, "workspace.metadata.json")); err != nil {
		return TrashResult{}, err
	}

	m.mu.Lock()
	if m.currentWorkspacePath == path {
		m.currentWorkspacePath = ""
		_ = m.writeUIStateLocked()
	}
	m.mu.Unlock()

	return result, nil
}

// RestoreWorkspaceTrash restores a trashed workspace under targetPath without changing its identity.
func (m *Manager) RestoreWorkspaceTrash(trashID, targetPath string) (Workspace, error) {
	trashID = strings.TrimSpace(trashID)
	targetPath = strings.TrimSpace(targetPath)
	if err := validateWorkspaceTrashID(trashID); err != nil {
		return Workspace{}, err
	}
	if err := validateWorkspacePath(targetPath); err != nil {
		return Workspace{}, err
	}
	trashDir := filepath.Join(m.vaultDir, ".verstak", "trash", "workspaces", trashID)
	data, err := os.ReadFile(filepath.Join(trashDir, "metadata.json"))
	if err != nil {
		return Workspace{}, err
	}
	var trashMeta struct {
		OriginalPath string `json:"originalPath"`
	}
	if err := json.Unmarshal(data, &trashMeta); err != nil {
		return Workspace{}, err
	}
	if err := validateWorkspacePath(trashMeta.OriginalPath); err != nil {
		return Workspace{}, err
	}
	payloadPath := filepath.Join(trashDir, filepath.Base(filepath.FromSlash(trashMeta.OriginalPath)))
	if err := ensureExistingWorkspaceDir(payloadPath, trashMeta.OriginalPath); err != nil {
		return Workspace{}, err
	}
	targetFull := filepath.Join(m.vaultDir, filepath.FromSlash(targetPath))
	if _, err := os.Lstat(targetFull); err == nil {
		return Workspace{}, fmt.Errorf("conflict: %s", targetPath)
	} else if !os.IsNotExist(err) {
		return Workspace{}, err
	}

	// Create parent folders
	parentDir := filepath.Dir(targetFull)
	if err := os.MkdirAll(parentDir, 0o755); err != nil {
		return Workspace{}, err
	}

	if err := os.Rename(payloadPath, targetFull); err != nil {
		return Workspace{}, err
	}
	restored := true
	defer func() {
		if restored {
			_ = os.Rename(targetFull, payloadPath)
			_ = cleanupEmptyParents(m.vaultDir, filepath.Dir(filepath.FromSlash(targetPath)))
		}
	}()
	if err := moveIfExists(filepath.Join(trashDir, "workspace.metadata.json"), m.metadataPath(targetPath)); err != nil {
		return Workspace{}, err
	}
	workspaceID, err := ensureWorkspaceIdentity(targetFull)
	if err != nil {
		return Workspace{}, err
	}
	if err := os.RemoveAll(trashDir); err != nil {
		return Workspace{}, err
	}
	name := filepath.Base(filepath.FromSlash(targetPath))
	restored = false
	return Workspace{ID: workspaceID, Name: name, Path: targetPath, RootPath: targetPath}, nil
}

// GetWorkspaceTrashIdentity reads a trashed workspace identity without restoring it.
func (m *Manager) GetWorkspaceTrashIdentity(trashID string) (WorkspaceIdentity, error) {
	trashID = strings.TrimSpace(trashID)
	if err := validateWorkspaceTrashID(trashID); err != nil {
		return WorkspaceIdentity{}, err
	}
	trashDir := filepath.Join(m.vaultDir, ".verstak", "trash", "workspaces", trashID)
	data, err := os.ReadFile(filepath.Join(trashDir, "metadata.json"))
	if err != nil {
		return WorkspaceIdentity{}, err
	}
	var trashMeta struct {
		OriginalPath string `json:"originalPath"`
	}
	if err := json.Unmarshal(data, &trashMeta); err != nil {
		return WorkspaceIdentity{}, err
	}
	if err := validateWorkspacePath(trashMeta.OriginalPath); err != nil {
		return WorkspaceIdentity{}, err
	}
	workspaceID, err := readWorkspaceIdentity(filepath.Join(trashDir, filepath.Base(filepath.FromSlash(trashMeta.OriginalPath))))
	if err != nil {
		return WorkspaceIdentity{}, err
	}
	return WorkspaceIdentity{WorkspaceID: workspaceID, RootPath: trashMeta.OriginalPath, State: "trashed"}, nil
}

// PurgeWorkspaceTrash permanently removes a workspace trash entry.
func (m *Manager) PurgeWorkspaceTrash(trashID string) error {
	trashID = strings.TrimSpace(trashID)
	if err := validateWorkspaceTrashID(trashID); err != nil {
		return err
	}
	trashDir := filepath.Join(m.vaultDir, ".verstak", "trash", "workspaces", trashID)
	if _, err := os.Stat(filepath.Join(trashDir, "metadata.json")); err != nil {
		return err
	}
	return os.RemoveAll(trashDir)
}

// CreateWorkspaceFromSync creates a workspace from the core-only sync contract.
func (m *Manager) CreateWorkspaceFromSync(path, workspaceID string, meta Metadata) (Workspace, error) {
	path = strings.TrimSpace(path)
	if err := validateWorkspacePath(path); err != nil {
		return Workspace{}, err
	}
	if _, err := uuid.Parse(workspaceID); err != nil {
		return Workspace{}, fmt.Errorf("invalid workspace identity: %w", err)
	}
	if existing, found, err := m.findActiveWorkspaceByID(workspaceID); err != nil {
		return Workspace{}, err
	} else if found {
		if existing.Path == path {
			return existing, nil
		}
		return Workspace{}, fmt.Errorf("conflict: workspace identity %s already belongs to %s", workspaceID, existing.Path)
	}

	full := filepath.Join(m.vaultDir, filepath.FromSlash(path))
	if _, err := os.Lstat(full); err == nil {
		return Workspace{}, fmt.Errorf("conflict: %s", path)
	} else if !os.IsNotExist(err) {
		return Workspace{}, err
	}

	parentDir := filepath.Dir(full)
	if err := os.MkdirAll(parentDir, 0o755); err != nil {
		return Workspace{}, err
	}

	if err := os.Mkdir(full, 0o755); err != nil {
		return Workspace{}, err
	}
	created := true
	defer func() {
		if created {
			_ = os.RemoveAll(full)
			_ = cleanupEmptyParents(m.vaultDir, filepath.Dir(filepath.FromSlash(path)))
		}
	}()
	if err := writeWorkspaceIdentityValue(full, workspaceID); err != nil {
		return Workspace{}, err
	}
	meta.WorkspaceID = workspaceID
	meta.WorkspaceName = filepath.Base(filepath.FromSlash(path))
	meta.WorkspacePath = path
	if meta.Features == nil {
		meta.Features = map[string]bool{"files": true}
	}
	if meta.Folders == nil {
		meta.Folders = defaultFolders()
	}
	if meta.UpdatedAt == "" {
		meta.UpdatedAt = time.Now().UTC().Format(time.RFC3339Nano)
	}
	for _, folder := range meta.Folders {
		if strings.TrimSpace(folder) == "" {
			continue
		}
		if err := validateWorkspaceFolderPath(folder); err != nil {
			return Workspace{}, err
		}
		if err := os.MkdirAll(filepath.Join(full, folder), 0o755); err != nil {
			return Workspace{}, err
		}
	}
	if err := m.writeMetadata(path, meta); err != nil {
		return Workspace{}, err
	}
	name := filepath.Base(filepath.FromSlash(path))
	created = false
	return Workspace{ID: workspaceID, Name: name, Path: path, RootPath: path}, nil
}

// RenameWorkspaceFromSync applies an idempotent rename only to the durable identity.
func (m *Manager) RenameWorkspaceFromSync(workspaceID, oldPath, newPath string) error {
	newPath = strings.TrimSpace(newPath)
	if err := validateWorkspacePath(newPath); err != nil {
		return err
	}
	workspace, found, err := m.findActiveWorkspaceByID(workspaceID)
	if err != nil {
		return err
	}
	if !found {
		return fmt.Errorf("not-found: workspace identity %s", workspaceID)
	}
	if workspace.Path == newPath {
		return nil
	}
	if strings.TrimSpace(oldPath) != "" && workspace.Path != oldPath {
		return fmt.Errorf("conflict: workspace identity %s is at %s, expected %s", workspaceID, workspace.Path, oldPath)
	}
	return m.RenameWorkspace(workspace.Path, newPath)
}

// TrashWorkspaceFromSync moves the identified active workspace into local trash.
func (m *Manager) TrashWorkspaceFromSync(workspaceID, path string) (TrashResult, error) {
	workspace, found, err := m.findActiveWorkspaceByID(workspaceID)
	if err != nil {
		return TrashResult{}, err
	}
	if !found {
		if _, trashed, err := m.findTrashedWorkspaceByID(workspaceID); err != nil {
			return TrashResult{}, err
		} else if trashed {
			return TrashResult{WorkspaceID: workspaceID}, nil
		}
		return TrashResult{}, fmt.Errorf("not-found: workspace identity %s", workspaceID)
	}
	if strings.TrimSpace(path) != "" && workspace.Path != path {
		return TrashResult{}, fmt.Errorf("conflict: workspace identity %s is at %s, expected %s", workspaceID, workspace.Path, path)
	}
	return m.TrashWorkspace(workspace.Path)
}

// RestoreWorkspaceFromSync restores a workspace by durable identity.
func (m *Manager) RestoreWorkspaceFromSync(workspaceID, targetPath string) (Workspace, error) {
	targetPath = strings.TrimSpace(targetPath)
	if err := validateWorkspacePath(targetPath); err != nil {
		return Workspace{}, err
	}
	if active, found, err := m.findActiveWorkspaceByID(workspaceID); err != nil {
		return Workspace{}, err
	} else if found {
		if active.Path == targetPath {
			return active, nil
		}
		return Workspace{}, fmt.Errorf("conflict: workspace identity %s is already active at %s", workspaceID, active.Path)
	}
	trashID, found, err := m.findTrashedWorkspaceByID(workspaceID)
	if err != nil {
		return Workspace{}, err
	}
	if !found {
		return Workspace{}, fmt.Errorf("not-found: trashed workspace identity %s", workspaceID)
	}
	return m.RestoreWorkspaceTrash(trashID, targetPath)
}

// GetWorkspaceMetadata returns stored metadata or safe generic metadata.
func (m *Manager) GetWorkspaceMetadata(path string) (Metadata, error) {
	path = strings.TrimSpace(path)
	if err := validateWorkspacePath(path); err != nil {
		return Metadata{}, err
	}
	full := filepath.Join(m.vaultDir, filepath.FromSlash(path))
	if err := ensureExistingWorkspaceDir(full, path); err != nil {
		return Metadata{}, err
	}

	data, err := os.ReadFile(m.metadataPath(path))
	if err != nil {
		if os.IsNotExist(err) {
			return genericMetadata(path), nil
		}
		return Metadata{}, err
	}
	var meta Metadata
	if err := json.Unmarshal(data, &meta); err != nil {
		return Metadata{}, err
	}
	meta.WorkspaceName = filepath.Base(filepath.FromSlash(path))
	meta.WorkspacePath = path
	if meta.Features == nil {
		meta.Features = map[string]bool{"files": true}
	}
	if !hasAnyTrueFeature(meta.Features) {
		meta.Features["files"] = true
	}
	if meta.Folders == nil {
		meta.Folders = defaultFolders()
	}
	return meta, nil
}

// UpdateWorkspaceMetadata merges UI/semantic metadata fields for an existing workspace.
func (m *Manager) UpdateWorkspaceMetadata(path string, patch MetadataPatch) (Metadata, error) {
	meta, err := m.GetWorkspaceMetadata(path)
	if err != nil {
		return Metadata{}, err
	}
	if meta.Features == nil {
		meta.Features = map[string]bool{}
	}
	for k, v := range patch.Features {
		meta.Features[k] = v
	}
	if meta.Folders == nil {
		meta.Folders = map[string]string{}
	}
	for k, v := range patch.Folders {
		meta.Folders[k] = v
	}
	meta.UpdatedAt = time.Now().UTC().Format(time.RFC3339Nano)
	if err := m.writeMetadata(path, meta); err != nil {
		return Metadata{}, err
	}
	return meta, nil
}

// ─── Folder Metadata API ────────────────────────────────────

// GetFolderMetadata returns stored metadata for a plain folder (not a workspace).
func (m *Manager) GetFolderMetadata(path string) (FolderMetadata, error) {
	path = strings.TrimSpace(path)
	if path == "" || path == "." {
		return FolderMetadata{}, nil
	}
	data, err := os.ReadFile(folderMetadataPath(m.vaultDir, path))
	if os.IsNotExist(err) {
		return FolderMetadata{}, nil
	}
	if err != nil {
		return FolderMetadata{}, err
	}
	var meta FolderMetadata
	if err := json.Unmarshal(data, &meta); err != nil {
		return FolderMetadata{}, err
	}
	return meta, nil
}

// SetFolderMetadata updates metadata for a plain folder (creates the folder if needed).
func (m *Manager) SetFolderMetadata(path string, meta FolderMetadata) error {
	path = strings.TrimSpace(path)
	if path == "" || path == "." {
		return fmt.Errorf("cannot set metadata on vault root")
	}
	full := filepath.Join(m.vaultDir, filepath.FromSlash(path))
	if _, err := os.Stat(full); os.IsNotExist(err) {
		if err := os.MkdirAll(full, 0o755); err != nil {
			return err
		}
	}
	meta.UpdatedAt = time.Now().UTC().Format(time.RFC3339Nano)
	data, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return err
	}
	metaPath := folderMetadataPath(m.vaultDir, path)
	if err := os.MkdirAll(filepath.Dir(metaPath), 0o755); err != nil {
		return err
	}
	tmp := metaPath + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, metaPath)
}

// ListFolderPaths returns paths of all folders that have metadata entries.
func (m *Manager) ListFolderPaths() ([]string, error) {
	metaDir := filepath.Join(m.vaultDir, ".verstak", "folder-metadata")
	entries, err := os.ReadDir(metaDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	paths := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		encoded := strings.TrimSuffix(entry.Name(), ".json")
		decoded, err := base64.RawURLEncoding.DecodeString(encoded)
		if err != nil {
			continue
		}
		folderPath := string(decoded)
		if folderPath == "" || folderPath == "." {
			continue
		}
		paths = append(paths, folderPath)
	}
	sort.Strings(paths)
	return paths, nil
}

// ─── Tree API ───────────────────────────────────────────────

// GetTree returns a flat tree with nested path-derived ParentID.
func (m *Manager) GetTree() WorkspaceTree {
	workspaces, err := m.ListWorkspaces()
	if err != nil {
		return WorkspaceTree{SchemaVersion: 2, Nodes: []WorkspaceNode{}}
	}

	m.mu.RLock()
	current := m.currentWorkspacePath
	m.mu.RUnlock()
	if current == "" || !workspaceExistsByPath(workspaces, current) {
		if len(workspaces) > 0 {
			current = workspaces[0].Path
		}
	}

	// Build nodes with ParentID derived from parent folder path
	folderMetas := make(map[string]FolderMetadata)
	nodes := make([]WorkspaceNode, 0, len(workspaces)*2)

	// First pass: collect folder metadata for parent chains
	folderSeen := make(map[string]bool)
	for _, ws := range workspaces {
		parent := filepath.Dir(filepath.FromSlash(ws.Path))
		for parent != "." && !folderSeen[parent] {
			folderSeen[parent] = true
			meta, _ := m.GetFolderMetadata(parent)
			folderMetas[parent] = meta
			nodes = append(nodes, WorkspaceNode{
				ID:        parent,
				ParentID:  filepath.Dir(filepath.FromSlash(parent)),
				Type:      TypeFolder,
				Title:     filepath.Base(filepath.FromSlash(parent)),
				Path:      parent,
				Status:    StatusActive,
				Order:     meta.Order,
				UpdatedAt: meta.UpdatedAt,
			})
			parent = filepath.Dir(filepath.FromSlash(parent))
		}
	}

	// Sort folders by order/name
	sort.SliceStable(nodes, func(i, j int) bool {
		oi := nodes[i].Order
		oj := nodes[j].Order
		if oi != oj {
			return oi < oj
		}
		return strings.ToLower(nodes[i].Title) < strings.ToLower(nodes[j].Title)
	})

	idx := 0
	seen := make(map[string]bool)
	for i := range nodes {
		if seen[nodes[i].ID] {
			continue
		}
		seen[nodes[i].ID] = true
		nodes[idx] = nodes[i]
		idx++
	}
	folders := nodes[:idx]

	now := time.Now().UTC().Format(time.RFC3339Nano)
	wsNodes := make([]WorkspaceNode, len(workspaces))
	for i, ws := range workspaces {
		parentID := filepath.Dir(filepath.FromSlash(ws.Path))
		if parentID == "." {
			parentID = ""
		}
		wsNodes[i] = WorkspaceNode{
			ID:        ws.Path,
			ParentID:  parentID,
			Type:      TypeSpace,
			Title:     ws.Name,
			Name:      ws.Name,
			RootPath:  ws.Path,
			Path:      ws.Path,
			Status:    StatusActive,
			Order:     i + len(folders),
			CreatedAt: now,
			UpdatedAt: now,
		}
	}

	allNodes := append(folders, wsNodes...)

	return WorkspaceTree{SchemaVersion: 2, Nodes: allNodes, CurrentNodeID: current, UpdatedAt: now}
}

// GetNode returns a compatibility node by workspace path.
func (m *Manager) GetNode(id string) (WorkspaceNode, error) {
	for _, node := range m.GetTree().Nodes {
		if node.ID == id {
			return node, nil
		}
	}
	return WorkspaceNode{}, fmt.Errorf("workspace not found: %s", id)
}

// ListChildren returns no children — compatibility, unused in new tree model.
func (m *Manager) ListChildren(parentID string) []WorkspaceNode {
	tree := m.GetTree()
	if parentID == "" {
		return tree.Nodes
	}
	result := make([]WorkspaceNode, 0)
	for _, n := range tree.Nodes {
		if n.ParentID == parentID {
			result = append(result, n)
		}
	}
	return result
}

// CreateNode is a compatibility wrapper for creating workspaces.
func (m *Manager) CreateNode(parentID string, nodeType NodeType, title string) (WorkspaceNode, error) {
	path := title
	if parentID != "" {
		path = filepath.ToSlash(filepath.Join(filepath.FromSlash(parentID), title))
	}
	ws, err := m.CreateWorkspace(path, "")
	if err != nil {
		return WorkspaceNode{}, err
	}
	return WorkspaceNode{ID: ws.Path, ParentID: parentID, Type: TypeSpace, Title: ws.Name, Name: ws.Name, RootPath: ws.Path, Order: 0}, nil
}

// RenameNode is a compatibility wrapper for physical workspace rename.
func (m *Manager) RenameNode(id, title string) error {
	newPath := title
	parentID := filepath.Dir(filepath.FromSlash(id))
	if parentID != "." {
		newPath = filepath.ToSlash(filepath.Join(filepath.FromSlash(parentID), title))
	}
	return m.RenameWorkspace(id, newPath)
}

// MoveNode moves a workspace to another folder.
func (m *Manager) MoveNode(id, newParentID string) error {
	if err := validateWorkspacePath(id); err != nil {
		return err
	}
	name := filepath.Base(filepath.FromSlash(id))
	newPath := name
	if newParentID != "" {
		newPath = filepath.ToSlash(filepath.Join(filepath.FromSlash(newParentID), name))
	}
	if newPath == id {
		return nil
	}
	return m.RenameWorkspace(id, newPath)
}

// ArchiveNode is a compatibility wrapper for trashing a workspace.
func (m *Manager) ArchiveNode(id string) error {
	_, err := m.TrashWorkspace(id)
	return err
}

// SetCurrentNode stores UI selection only.
func (m *Manager) SetCurrentNode(id string) error {
	if err := validateWorkspacePath(id); err != nil {
		return err
	}
	workspaces, err := m.ListWorkspaces()
	if err != nil {
		return err
	}
	if !workspaceExistsByPath(workspaces, id) {
		return fmt.Errorf("workspace not found: %s", id)
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	m.currentWorkspacePath = id
	return m.writeUIStateLocked()
}

// GetCurrentNode returns the currently selected compatibility node.
func (m *Manager) GetCurrentNode() (WorkspaceNode, error) {
	tree := m.GetTree()
	if tree.CurrentNodeID == "" {
		return WorkspaceNode{}, fmt.Errorf("no current workspace")
	}
	for _, node := range tree.Nodes {
		if node.ID == tree.CurrentNodeID {
			return node, nil
		}
	}
	return WorkspaceNode{}, fmt.Errorf("current workspace not found: %s", tree.CurrentNodeID)
}

// Save persists UI state only; workspace existence is never persisted here.
func (m *Manager) Save() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.writeUIStateLocked()
}

// ─── Helpers ─────────────────────────────────────────────────

func joinWorkspacePath(prefix, name string) string {
	if prefix == "" {
		return name
	}
	return filepath.ToSlash(filepath.Join(filepath.FromSlash(prefix), name))
}

func folderMetadataPath(vaultDir, folderRel string) string {
	encoded := base64.RawURLEncoding.EncodeToString([]byte(folderRel))
	return filepath.Join(vaultDir, ".verstak", "folder-metadata", encoded+".json")
}

func oldFolderMetadataPath(vaultDir, folderRel string) string {
	encoded := base64.RawURLEncoding.EncodeToString([]byte(folderRel))
	return filepath.Join(vaultDir, ".verstak", "folder-metadata", encoded+".json")
}

func applyTemplate(workspaceDir string, template templateDefinition) error {
	for rel, content := range template.Files {
		if strings.Contains(rel, "\x00") || strings.Contains(rel, "\\") || strings.HasPrefix(rel, "/") {
			return fmt.Errorf("invalid-template-path: %s", rel)
		}
		parts := strings.Split(rel, "/")
		for _, part := range parts {
			if part == "" || part == "." || part == ".." {
				return fmt.Errorf("invalid-template-path: %s", rel)
			}
		}
		full := filepath.Join(workspaceDir, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			return err
		}
	}
	for _, folder := range template.Folders {
		if folder == "" {
			continue
		}
		if err := os.MkdirAll(filepath.Join(workspaceDir, folder), 0o755); err != nil {
			return err
		}
	}
	return nil
}

func writeWorkspaceIdentity(workspacePath string) (string, error) {
	workspaceID := uuid.NewString()
	if err := writeWorkspaceIdentityValue(workspacePath, workspaceID); err != nil {
		return "", err
	}
	return workspaceID, nil
}

func writeWorkspaceIdentityValue(workspacePath, workspaceID string) error {
	if _, err := uuid.Parse(workspaceID); err != nil {
		return fmt.Errorf("invalid workspace identity: %w", err)
	}
	data, err := json.Marshal(workspaceIdentityMarker{WorkspaceID: workspaceID})
	if err != nil {
		return err
	}
	markerPath := filepath.Join(workspacePath, filepath.FromSlash(workspaceIdentityRelativePath))
	if err := os.MkdirAll(filepath.Dir(markerPath), 0o755); err != nil {
		return err
	}
	tmpPath := markerPath + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0o644); err != nil {
		return err
	}
	if err := os.Rename(tmpPath, markerPath); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	return nil
}

func ensureWorkspaceIdentity(workspacePath string) (string, error) {
	workspaceID, err := readWorkspaceIdentity(workspacePath)
	if os.IsNotExist(err) {
		return writeWorkspaceIdentity(workspacePath)
	}
	if err != nil {
		return "", err
	}
	return workspaceID, nil
}

func readWorkspaceIdentity(workspacePath string) (string, error) {
	markerPath := filepath.Join(workspacePath, filepath.FromSlash(workspaceIdentityRelativePath))
	data, err := os.ReadFile(markerPath)
	if err != nil {
		return "", err
	}
	var marker workspaceIdentityMarker
	if err := json.Unmarshal(data, &marker); err != nil {
		return "", err
	}
	if _, err := uuid.Parse(marker.WorkspaceID); err != nil {
		return "", fmt.Errorf("invalid workspace identity: %w", err)
	}
	return marker.WorkspaceID, nil
}

func validateWorkspacePath(path string) error {
	if strings.TrimSpace(path) == "" {
		return fmt.Errorf("invalid-workspace-path: empty")
	}
	if strings.Contains(path, "\x00") {
		return fmt.Errorf("invalid-workspace-path: null-byte")
	}
	if strings.Contains(path, "\\") {
		return fmt.Errorf("invalid-workspace-path: backslash")
	}
	if filepath.IsAbs(path) || strings.HasPrefix(path, "/") {
		return fmt.Errorf("invalid-workspace-path: absolute path rejected")
	}
	cleaned := filepath.ToSlash(filepath.Clean(filepath.FromSlash(path)))
	if cleaned == "." || cleaned == ".." || strings.HasPrefix(cleaned, "../") {
		return fmt.Errorf("invalid-workspace-path: traversal")
	}
	for _, segment := range strings.Split(path, "/") {
		if segment == ".." {
			return fmt.Errorf("invalid-workspace-path: traversal segment")
		}
		if segment == "" || segment == "." {
			return fmt.Errorf("invalid-workspace-path: empty segment")
		}
		if strings.EqualFold(segment, ".verstak") || strings.EqualFold(segment, ".git") {
			return fmt.Errorf("reserved-name: %s", segment)
		}
		for _, r := range segment {
			if unicode.IsControl(r) {
				return fmt.Errorf("invalid-workspace-path: control character")
			}
		}
	}
	return nil
}

func validateWorkspaceName(name string) error {
	// Legacy: validate a single segment name
	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("invalid-workspace-name: empty")
	}
	if strings.Contains(name, "\x00") {
		return fmt.Errorf("invalid-workspace-name: null-byte")
	}
	if strings.ContainsAny(name, `/\`) {
		return fmt.Errorf("invalid-workspace-name: path separators are not allowed")
	}
	if looksAbsoluteName(name) {
		return fmt.Errorf("invalid-workspace-name: absolute path rejected")
	}
	if name == "." || name == ".." || strings.Contains(name, "..") {
		return fmt.Errorf("invalid-workspace-name: path traversal")
	}
	for _, r := range name {
		if unicode.IsControl(r) {
			return fmt.Errorf("invalid-workspace-name: control character")
		}
	}
	if isReservedWorkspaceName(name) {
		return fmt.Errorf("reserved-workspace-name: %s", name)
	}
	return nil
}

func validateWorkspaceFolderPath(folder string) error {
	folder = strings.TrimSpace(folder)
	if folder == "" || filepath.IsAbs(folder) || strings.Contains(folder, `\`) {
		return fmt.Errorf("invalid workspace folder path")
	}
	cleaned := filepath.ToSlash(filepath.Clean(filepath.FromSlash(folder)))
	if cleaned == "." || cleaned == ".." || strings.HasPrefix(cleaned, "../") {
		return fmt.Errorf("invalid workspace folder path")
	}
	for _, segment := range strings.Split(cleaned, "/") {
		if strings.EqualFold(segment, ".verstak") {
			return fmt.Errorf("invalid workspace folder path")
		}
	}
	return nil
}

func validateWorkspaceTrashID(trashID string) error {
	if trashID == "" || strings.ContainsAny(trashID, `/\\`) || filepath.Clean(trashID) != trashID {
		return fmt.Errorf("invalid workspace trash ID")
	}
	return nil
}

func looksAbsoluteName(name string) bool {
	if filepath.IsAbs(name) || strings.HasPrefix(name, "/") || strings.HasPrefix(name, "\\") {
		return true
	}
	return len(name) >= 2 && name[1] == ':' && unicode.IsLetter(rune(name[0]))
}

func isReservedWorkspaceName(name string) bool {
	reserved := []string{".verstak", ".git"}
	for _, item := range reserved {
		if strings.EqualFold(name, item) {
			return true
		}
	}
	return false
}

func ensureExistingWorkspaceDir(full, path string) error {
	info, err := os.Lstat(full)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("not-found: %s", path)
		}
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("symlink-not-allowed: %s", path)
	}
	if !info.IsDir() {
		return fmt.Errorf("not-directory: %s", path)
	}
	return nil
}

func (m *Manager) metadataPath(path string) string {
	encoded := base64.RawURLEncoding.EncodeToString([]byte(path))
	return filepath.Join(m.vaultDir, ".verstak", "workspaces", encoded+".json")
}

func (m *Manager) writeMetadata(path string, meta Metadata) error {
	if err := os.MkdirAll(filepath.Join(m.vaultDir, ".verstak", "workspaces"), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return err
	}
	p := m.metadataPath(path)
	tmp := p + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	if err := os.Rename(tmp, p); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}

func (m *Manager) uiStatePath() string {
	return filepath.Join(m.vaultDir, ".verstak", "workspace-ui.json")
}

func (m *Manager) readSelectedWorkspaceLocked() string {
	data, err := os.ReadFile(m.uiStatePath())
	if err != nil {
		return ""
	}
	var state struct {
		SelectedWorkspace string `json:"selectedWorkspace"`
	}
	if err := json.Unmarshal(data, &state); err != nil {
		return ""
	}
	return state.SelectedWorkspace
}

func (m *Manager) writeUIStateLocked() error {
	if err := os.MkdirAll(filepath.Join(m.vaultDir, ".verstak"), 0o755); err != nil {
		return err
	}
	state := struct {
		SelectedWorkspace string `json:"selectedWorkspace,omitempty"`
		UpdatedAt         string `json:"updatedAt"`
	}{
		SelectedWorkspace: m.currentWorkspacePath,
		UpdatedAt:         time.Now().UTC().Format(time.RFC3339Nano),
	}
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	p := m.uiStatePath()
	tmp := p + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	if err := os.Rename(tmp, p); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}

func cleanupEmptyParents(vaultDir, relPath string) error {
	for relPath != "" && relPath != "." {
		full := filepath.Join(vaultDir, filepath.FromSlash(relPath))
		entries, err := os.ReadDir(full)
		if err != nil || len(entries) > 0 {
			return nil
		}
		if err := os.Remove(full); err != nil {
			return nil
		}
		relPath = filepath.Dir(filepath.FromSlash(relPath))
		if relPath == "." {
			relPath = ""
		}
	}
	return nil
}

func genericMetadata(path string) Metadata {
	name := filepath.Base(filepath.FromSlash(path))
	return Metadata{
		WorkspaceName: name,
		WorkspacePath: path,
		Features:      map[string]bool{"files": true},
		Folders:       defaultFolders(),
	}
}

func defaultFolders() map[string]string {
	return map[string]string{
		"notes": "Notes",
		"files": "Files",
	}
}

func (m *Manager) findActiveWorkspaceByID(workspaceID string) (Workspace, bool, error) {
	if _, err := uuid.Parse(workspaceID); err != nil {
		return Workspace{}, false, fmt.Errorf("invalid workspace identity: %w", err)
	}
	workspaces, err := m.ListWorkspaces()
	if err != nil {
		return Workspace{}, false, err
	}
	var match Workspace
	for _, ws := range workspaces {
		if ws.ID != workspaceID {
			continue
		}
		if match.ID != "" {
			return Workspace{}, false, fmt.Errorf("conflict: duplicated workspace identity %s", workspaceID)
		}
		match = ws
	}
	return match, match.ID != "", nil
}

func (m *Manager) findTrashedWorkspaceByID(workspaceID string) (string, bool, error) {
	if _, err := uuid.Parse(workspaceID); err != nil {
		return "", false, fmt.Errorf("invalid workspace identity: %w", err)
	}
	trashRoot := filepath.Join(m.vaultDir, ".verstak", "trash", "workspaces")
	entries, err := os.ReadDir(trashRoot)
	if err != nil {
		if os.IsNotExist(err) {
			return "", false, nil
		}
		return "", false, err
	}
	var match string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		identity, err := m.GetWorkspaceTrashIdentity(entry.Name())
		if err != nil {
			continue
		}
		if identity.WorkspaceID != workspaceID {
			continue
		}
		if match != "" {
			return "", false, fmt.Errorf("conflict: duplicated trashed workspace identity %s", workspaceID)
		}
		match = entry.Name()
	}
	return match, match != "", nil
}

func cloneBoolMap(src map[string]bool) map[string]bool {
	dst := make(map[string]bool, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

func cloneStringMap(src map[string]string) map[string]string {
	dst := make(map[string]string, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

func cloneStringSlice(src []string) []string {
	return append([]string(nil), src...)
}

func hasAnyTrueFeature(features map[string]bool) bool {
	for _, enabled := range features {
		if enabled {
			return true
		}
	}
	return false
}

func workspaceExistsByPath(workspaces []Workspace, path string) bool {
	for _, ws := range workspaces {
		if ws.Path == path {
			return true
		}
	}
	return false
}

func moveIfExists(from, to string) error {
	if _, err := os.Stat(from); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if err := os.MkdirAll(filepath.Dir(to), 0o755); err != nil {
		return err
	}
	return os.Rename(from, to)
}

// ClearTemplateRegistryForTest simulates templates disappearing after a workspace
// has already stored its creation snapshot.
func ClearTemplateRegistryForTest(t interface{ Cleanup(func()) }) {
	original := builtInTemplates
	builtInTemplates = map[string]templateDefinition{}
	t.Cleanup(func() {
		builtInTemplates = original
	})
}
