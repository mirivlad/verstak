// Package workspace provides the semantic workspace lifecycle service.
//
// A workspace is a top-level physical folder directly under the vault root.
// The filesystem is the source of truth for workspace existence and listing.
// Metadata under .verstak stores UI state and creation snapshots only.
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

// Workspace is a physical top-level workspace folder.
type Workspace struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	RootPath string `json:"rootPath"`
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
	CreatedFromTemplate *TemplateSnapshot `json:"createdFromTemplate,omitempty"`
	Features            map[string]bool   `json:"features,omitempty"`
	Folders             map[string]string `json:"folders,omitempty"`
	WorkspaceTools      []string          `json:"workspaceTools,omitempty"`
	UpdatedAt           string            `json:"updatedAt,omitempty"`
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

// WorkspaceIdentity identifies an existing top-level workspace independently of its path.
type WorkspaceIdentity struct {
	WorkspaceID string `json:"workspaceId"`
	RootPath    string `json:"rootPath"`
	State       string `json:"state"`
}

// WorkspaceNode is a compatibility shell view of a top-level workspace.
// Path is deliberately not serialized; workspaceRootPath is derived from Name/ID.
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

// WorkspaceTree is a compatibility flat list, derived from top-level folders.
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

// Manager provides workspace operations for one vault.
type Manager struct {
	mu                   sync.RWMutex
	vaultDir             string
	initialized          bool
	currentWorkspaceName string
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
	m.currentWorkspaceName = m.readSelectedWorkspaceLocked()
	return nil
}

// IsInitialized returns true after Load has been called.
func (m *Manager) IsInitialized() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.initialized
}

// ListWorkspaces returns top-level physical workspace folders from the vault.
func (m *Manager) ListWorkspaces() ([]Workspace, error) {
	entries, err := os.ReadDir(m.vaultDir)
	if err != nil {
		return nil, err
	}

	workspaces := make([]Workspace, 0, len(entries))
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
		// Skip organizational folders (identified by .verstak/folder.json).
		// ensureWorkspaceIdentity would otherwise write a workspace.json into them.
		fullPath := filepath.Join(m.vaultDir, name)
		if _, err := os.Stat(filepath.Join(fullPath, ".verstak", "folder.json")); err == nil {
			continue
		}
		workspaceID, err := ensureWorkspaceIdentity(fullPath)
		if err != nil {
			return nil, err
		}
		workspaces = append(workspaces, Workspace{ID: workspaceID, Name: name, RootPath: name})
	}
	sort.Slice(workspaces, func(i, j int) bool {
		return strings.ToLower(workspaces[i].Name) < strings.ToLower(workspaces[j].Name)
	})
	return workspaces, nil
}

// GetWorkspaceIdentity resolves the durable identity for an existing workspace.
func (m *Manager) GetWorkspaceIdentity(name string) (WorkspaceIdentity, error) {
	name = strings.TrimSpace(name)
	if err := validateWorkspaceName(name); err != nil {
		return WorkspaceIdentity{}, err
	}
	full := filepath.Join(m.vaultDir, name)
	if err := ensureExistingWorkspaceDir(full, name); err != nil {
		return WorkspaceIdentity{}, err
	}
	workspaceID, err := ensureWorkspaceIdentity(full)
	if err != nil {
		return WorkspaceIdentity{}, err
	}
	return WorkspaceIdentity{WorkspaceID: workspaceID, RootPath: name, State: "active"}, nil
}

// ListWorkspaceIdentities returns durable identities and flags copied markers.
func (m *Manager) ListWorkspaceIdentities() ([]WorkspaceIdentity, error) {
	workspaces, err := m.ListWorkspaces()
	if err != nil {
		return nil, err
	}
	counts := make(map[string]int, len(workspaces))
	for _, workspace := range workspaces {
		counts[workspace.ID]++
	}
	identities := make([]WorkspaceIdentity, 0, len(workspaces))
	for _, workspace := range workspaces {
		state := "active"
		if counts[workspace.ID] > 1 {
			state = "duplicate"
		}
		identities = append(identities, WorkspaceIdentity{
			WorkspaceID: workspace.ID,
			RootPath:    workspace.RootPath,
			State:       state,
		})
	}
	return identities, nil
}

// RepairWorkspaceIdentity keeps the identity on one copied workspace and regenerates the other.
func (m *Manager) RepairWorkspaceIdentity(keepName, regenerateName string) error {
	keepName = strings.TrimSpace(keepName)
	regenerateName = strings.TrimSpace(regenerateName)
	if err := validateWorkspaceName(keepName); err != nil {
		return err
	}
	if err := validateWorkspaceName(regenerateName); err != nil {
		return err
	}
	if keepName == regenerateName {
		return fmt.Errorf("workspace identity repair requires two workspaces")
	}
	keepPath := filepath.Join(m.vaultDir, keepName)
	regeneratePath := filepath.Join(m.vaultDir, regenerateName)
	if err := ensureExistingWorkspaceDir(keepPath, keepName); err != nil {
		return err
	}
	if err := ensureExistingWorkspaceDir(regeneratePath, regenerateName); err != nil {
		return err
	}
	keepID, err := ensureWorkspaceIdentity(keepPath)
	if err != nil {
		return err
	}
	regenerateID, err := ensureWorkspaceIdentity(regeneratePath)
	if err != nil {
		return err
	}
	if keepID != regenerateID {
		return fmt.Errorf("workspaces do not share an identity")
	}
	_, err = writeWorkspaceIdentity(regeneratePath)
	return err
}

// CreateWorkspace creates a top-level workspace folder and applies a template once.
func (m *Manager) CreateWorkspace(name, templateID string) (Workspace, error) {
	name = strings.TrimSpace(name)
	if err := validateWorkspaceName(name); err != nil {
		return Workspace{}, err
	}
	if templateID == "" {
		templateID = "default"
	}
	template, ok := builtInTemplates[templateID]
	if !ok {
		return Workspace{}, fmt.Errorf("template-not-found: %s", templateID)
	}

	full := filepath.Join(m.vaultDir, name)
	if _, err := os.Lstat(full); err == nil {
		return Workspace{}, fmt.Errorf("conflict: %s", name)
	} else if !os.IsNotExist(err) {
		return Workspace{}, err
	}
	if err := os.Mkdir(full, 0o755); err != nil {
		return Workspace{}, err
	}
	created := true
	defer func() {
		if created {
			_ = os.RemoveAll(full)
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
	if err := m.writeMetadata(name, meta); err != nil {
		return Workspace{}, err
	}

	created = false
	return Workspace{ID: workspaceID, Name: name, RootPath: name}, nil
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
	// Guard: refuse to write workspace identity into an organizational folder.
	if _, err := os.Stat(filepath.Join(workspacePath, ".verstak", "folder.json")); err == nil {
		return "", fmt.Errorf("directory is an organizational folder, not a workspace: %s", workspacePath)
	}
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
	if err := validateWorkspaceName(trashMeta.OriginalPath); err != nil {
		return WorkspaceIdentity{}, err
	}
	workspaceID, err := readWorkspaceIdentity(filepath.Join(trashDir, trashMeta.OriginalPath))
	if err != nil {
		return WorkspaceIdentity{}, err
	}
	return WorkspaceIdentity{WorkspaceID: workspaceID, RootPath: trashMeta.OriginalPath, State: "trashed"}, nil
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

// RenameWorkspace physically renames a top-level workspace folder and metadata key.
func (m *Manager) RenameWorkspace(oldName, newName string) error {
	oldName = strings.TrimSpace(oldName)
	newName = strings.TrimSpace(newName)
	if err := validateWorkspaceName(oldName); err != nil {
		return err
	}
	if err := validateWorkspaceName(newName); err != nil {
		return err
	}
	oldFull := filepath.Join(m.vaultDir, oldName)
	newFull := filepath.Join(m.vaultDir, newName)
	if err := ensureExistingWorkspaceDir(oldFull, oldName); err != nil {
		return err
	}
	if _, err := os.Lstat(newFull); err == nil {
		return fmt.Errorf("conflict: %s", newName)
	} else if !os.IsNotExist(err) {
		return err
	}

	if err := os.Rename(oldFull, newFull); err != nil {
		return err
	}
	renamedFolder := true
	defer func() {
		if renamedFolder {
			_ = os.Rename(newFull, oldFull)
		}
	}()

	oldMetaPath := m.metadataPath(oldName)
	if data, err := os.ReadFile(oldMetaPath); err == nil {
		var meta Metadata
		if err := json.Unmarshal(data, &meta); err != nil {
			return err
		}
		meta.WorkspaceName = newName
		meta.UpdatedAt = time.Now().UTC().Format(time.RFC3339Nano)
		if err := m.writeMetadata(newName, meta); err != nil {
			return err
		}
		if err := os.Remove(oldMetaPath); err != nil && !os.IsNotExist(err) {
			return err
		}
	} else if !os.IsNotExist(err) {
		return err
	}

	m.mu.Lock()
	if m.currentWorkspaceName == oldName {
		m.currentWorkspaceName = newName
		_ = m.writeUIStateLocked()
	}
	m.mu.Unlock()

	renamedFolder = false
	return nil
}

// TrashWorkspace moves the whole top-level workspace folder to internal trash.
func (m *Manager) TrashWorkspace(name string) (TrashResult, error) {
	name = strings.TrimSpace(name)
	if err := validateWorkspaceName(name); err != nil {
		return TrashResult{}, err
	}
	full := filepath.Join(m.vaultDir, name)
	if err := ensureExistingWorkspaceDir(full, name); err != nil {
		return TrashResult{}, err
	}
	workspaceID, err := ensureWorkspaceIdentity(full)
	if err != nil {
		return TrashResult{}, err
	}

	deletedAt := time.Now().UTC().Format(time.RFC3339Nano)
	trashID := time.Now().UTC().Format("20060102T150405.000000000Z") + "-" + uuid.NewString()
	trashRel := filepath.ToSlash(filepath.Join(".verstak", "trash", "workspaces", trashID, name))
	trashFull := filepath.Join(m.vaultDir, filepath.FromSlash(trashRel))
	if err := os.MkdirAll(filepath.Dir(trashFull), 0o755); err != nil {
		return TrashResult{}, err
	}
	if err := os.Rename(full, trashFull); err != nil {
		return TrashResult{}, err
	}

	result := TrashResult{WorkspaceID: workspaceID, OriginalPath: name, TrashPath: trashRel, TrashID: trashID, DeletedAt: deletedAt}
	trashMeta := map[string]string{
		"originalPath": name,
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
	if err := moveIfExists(m.metadataPath(name), filepath.Join(trashDir, "workspace.metadata.json")); err != nil {
		return TrashResult{}, err
	}

	m.mu.Lock()
	if m.currentWorkspaceName == name {
		m.currentWorkspaceName = ""
		_ = m.writeUIStateLocked()
	}
	m.mu.Unlock()

	return result, nil
}

// RestoreWorkspaceTrash restores a trashed workspace under targetName without changing its identity.
func (m *Manager) RestoreWorkspaceTrash(trashID, targetName string) (Workspace, error) {
	trashID = strings.TrimSpace(trashID)
	targetName = strings.TrimSpace(targetName)
	if err := validateWorkspaceTrashID(trashID); err != nil {
		return Workspace{}, err
	}
	if err := validateWorkspaceName(targetName); err != nil {
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
	if err := validateWorkspaceName(trashMeta.OriginalPath); err != nil {
		return Workspace{}, err
	}
	payloadPath := filepath.Join(trashDir, trashMeta.OriginalPath)
	if err := ensureExistingWorkspaceDir(payloadPath, trashMeta.OriginalPath); err != nil {
		return Workspace{}, err
	}
	targetPath := filepath.Join(m.vaultDir, targetName)
	if _, err := os.Lstat(targetPath); err == nil {
		return Workspace{}, fmt.Errorf("conflict: %s", targetName)
	} else if !os.IsNotExist(err) {
		return Workspace{}, err
	}
	if err := os.Rename(payloadPath, targetPath); err != nil {
		return Workspace{}, err
	}
	restored := true
	defer func() {
		if restored {
			_ = os.Rename(targetPath, payloadPath)
		}
	}()
	if err := moveIfExists(filepath.Join(trashDir, "workspace.metadata.json"), m.metadataPath(targetName)); err != nil {
		return Workspace{}, err
	}
	workspaceID, err := ensureWorkspaceIdentity(targetPath)
	if err != nil {
		return Workspace{}, err
	}
	if err := os.RemoveAll(trashDir); err != nil {
		return Workspace{}, err
	}
	restored = false
	return Workspace{ID: workspaceID, Name: targetName, RootPath: targetName}, nil
}

// CreateWorkspaceFromSync creates a workspace from the core-only sync contract.
// It deliberately does not apply a live template: child folders and files are
// represented by normal file operations, while the captured metadata preserves
// the historical template snapshot that matters for restoration.
func (m *Manager) CreateWorkspaceFromSync(name, workspaceID string, meta Metadata) (Workspace, error) {
	name = strings.TrimSpace(name)
	if err := validateWorkspaceName(name); err != nil {
		return Workspace{}, err
	}
	if _, err := uuid.Parse(workspaceID); err != nil {
		return Workspace{}, fmt.Errorf("invalid workspace identity: %w", err)
	}
	if existing, found, err := m.findActiveWorkspaceByID(workspaceID); err != nil {
		return Workspace{}, err
	} else if found {
		if existing.Name == name {
			return existing, nil
		}
		return Workspace{}, fmt.Errorf("conflict: workspace identity %s already belongs to %s", workspaceID, existing.Name)
	}

	full := filepath.Join(m.vaultDir, name)
	if _, err := os.Lstat(full); err == nil {
		return Workspace{}, fmt.Errorf("conflict: %s", name)
	} else if !os.IsNotExist(err) {
		return Workspace{}, err
	}
	if err := os.Mkdir(full, 0o755); err != nil {
		return Workspace{}, err
	}
	created := true
	defer func() {
		if created {
			_ = os.RemoveAll(full)
		}
	}()
	if err := writeWorkspaceIdentityValue(full, workspaceID); err != nil {
		return Workspace{}, err
	}
	meta.WorkspaceID = workspaceID
	meta.WorkspaceName = name
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
	if err := m.writeMetadata(name, meta); err != nil {
		return Workspace{}, err
	}
	created = false
	return Workspace{ID: workspaceID, Name: name, RootPath: name}, nil
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

// RenameWorkspaceFromSync applies an idempotent rename only to the durable
// identity named by the operation. A different local path is a visible conflict.
func (m *Manager) RenameWorkspaceFromSync(workspaceID, oldName, newName string) error {
	newName = strings.TrimSpace(newName)
	if err := validateWorkspaceName(newName); err != nil {
		return err
	}
	workspace, found, err := m.findActiveWorkspaceByID(workspaceID)
	if err != nil {
		return err
	}
	if !found {
		return fmt.Errorf("not-found: workspace identity %s", workspaceID)
	}
	if workspace.Name == newName {
		return nil
	}
	if strings.TrimSpace(oldName) != "" && workspace.Name != oldName {
		return fmt.Errorf("conflict: workspace identity %s is at %s, expected %s", workspaceID, workspace.Name, oldName)
	}
	return m.RenameWorkspace(workspace.Name, newName)
}

// TrashWorkspaceFromSync moves the identified active workspace into this
// device's local trash. Local trash IDs are intentionally not synchronized.
func (m *Manager) TrashWorkspaceFromSync(workspaceID, name string) (TrashResult, error) {
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
	if strings.TrimSpace(name) != "" && workspace.Name != name {
		return TrashResult{}, fmt.Errorf("conflict: workspace identity %s is at %s, expected %s", workspaceID, workspace.Name, name)
	}
	return m.TrashWorkspace(workspace.Name)
}

// RestoreWorkspaceFromSync restores a workspace by durable identity. It is
// idempotent at the requested target and rejects a workspace of another ID.
func (m *Manager) RestoreWorkspaceFromSync(workspaceID, targetName string) (Workspace, error) {
	targetName = strings.TrimSpace(targetName)
	if err := validateWorkspaceName(targetName); err != nil {
		return Workspace{}, err
	}
	if active, found, err := m.findActiveWorkspaceByID(workspaceID); err != nil {
		return Workspace{}, err
	} else if found {
		if active.Name == targetName {
			return active, nil
		}
		return Workspace{}, fmt.Errorf("conflict: workspace identity %s is already active at %s", workspaceID, active.Name)
	}
	trashID, found, err := m.findTrashedWorkspaceByID(workspaceID)
	if err != nil {
		return Workspace{}, err
	}
	if !found {
		return Workspace{}, fmt.Errorf("not-found: trashed workspace identity %s", workspaceID)
	}
	return m.RestoreWorkspaceTrash(trashID, targetName)
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
	for _, workspace := range workspaces {
		if workspace.ID != workspaceID {
			continue
		}
		if match.ID != "" {
			return Workspace{}, false, fmt.Errorf("conflict: duplicated workspace identity %s", workspaceID)
		}
		match = workspace
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

func validateWorkspaceTrashID(trashID string) error {
	if trashID == "" || strings.ContainsAny(trashID, `/\\`) || filepath.Clean(trashID) != trashID {
		return fmt.Errorf("invalid workspace trash ID")
	}
	return nil
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

// GetWorkspaceMetadata returns stored metadata or safe generic metadata.
// name may be a top-level workspace name or a relative path (e.g. "Folder/WorkspaceName").
func (m *Manager) GetWorkspaceMetadata(name string) (Metadata, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return Metadata{}, fmt.Errorf("invalid-workspace-name: empty")
	}
	if err := validateWorkspaceMetadataPath(name); err != nil {
		return Metadata{}, err
	}
	full := filepath.Join(m.vaultDir, filepath.FromSlash(name))
	if err := ensureExistingWorkspaceDir(full, name); err != nil {
		return Metadata{}, err
	}

	// Try name-keyed metadata first.
	data, err := os.ReadFile(m.metadataPath(name))
	if err == nil {
		var meta Metadata
		if err := json.Unmarshal(data, &meta); err != nil {
			return Metadata{}, err
		}
		meta.WorkspaceName = filepath.Base(filepath.FromSlash(name))
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
	if !os.IsNotExist(err) {
		return Metadata{}, err
	}

	// Name-keyed metadata not found — try UUID-keyed metadata via workspace marker.
	workspaceID, readErr := readWorkspaceIdentity(full)
	if readErr == nil && workspaceID != "" {
		if uuidMeta, uuidErr := m.GetWorkspaceMetadataByUUID(workspaceID); uuidErr == nil {
			uuidMeta.WorkspaceName = filepath.Base(filepath.FromSlash(name))
			return uuidMeta, nil
		}
	}

	return genericMetadata(filepath.Base(filepath.FromSlash(name))), nil
}

// UpdateWorkspaceMetadata merges UI/semantic metadata fields for an existing workspace.
func (m *Manager) UpdateWorkspaceMetadata(name string, patch MetadataPatch) (Metadata, error) {
	meta, err := m.GetWorkspaceMetadata(name)
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
	if err := m.writeMetadata(name, meta); err != nil {
		return Metadata{}, err
	}
	return meta, nil
}

// GetTree returns a compatibility flat tree derived from top-level folders.
func (m *Manager) GetTree() WorkspaceTree {
	workspaces, err := m.ListWorkspaces()
	if err != nil {
		return WorkspaceTree{SchemaVersion: 1, Nodes: []WorkspaceNode{}}
	}

	m.mu.RLock()
	current := m.currentWorkspaceName
	m.mu.RUnlock()
	if current == "" || !workspaceExists(workspaces, current) {
		if len(workspaces) > 0 {
			current = workspaces[0].Name
		}
	}

	nodes := make([]WorkspaceNode, 0, len(workspaces))
	now := time.Now().UTC().Format(time.RFC3339Nano)
	for i, ws := range workspaces {
		nodes = append(nodes, WorkspaceNode{
			ID:        ws.Name,
			Type:      TypeSpace,
			Title:     ws.Name,
			Name:      ws.Name,
			RootPath:  ws.RootPath,
			Status:    StatusActive,
			Order:     i,
			CreatedAt: now,
			UpdatedAt: now,
		})
	}
	return WorkspaceTree{SchemaVersion: 1, Nodes: nodes, CurrentNodeID: current, UpdatedAt: now}
}

// GetNode returns a compatibility node by workspace name.
func (m *Manager) GetNode(id string) (WorkspaceNode, error) {
	for _, node := range m.GetTree().Nodes {
		if node.ID == id {
			return node, nil
		}
	}
	return WorkspaceNode{}, fmt.Errorf("workspace not found: %s", id)
}

// ListChildren returns no children because workspaces are only top-level folders.
func (m *Manager) ListChildren(parentID string) []WorkspaceNode {
	if parentID != "" {
		return nil
	}
	return m.GetTree().Nodes
}

// CreateNode is a compatibility wrapper for creating top-level workspaces only.
func (m *Manager) CreateNode(parentID string, nodeType NodeType, title string) (WorkspaceNode, error) {
	if parentID != "" {
		return WorkspaceNode{}, fmt.Errorf("workspace folders are top-level only")
	}
	if nodeType != "" && nodeType != TypeSpace {
		return WorkspaceNode{}, fmt.Errorf("workspace folders are top-level only")
	}
	ws, err := m.CreateWorkspace(title, "")
	if err != nil {
		return WorkspaceNode{}, err
	}
	return WorkspaceNode{ID: ws.Name, Type: TypeSpace, Title: ws.Name, Name: ws.Name, RootPath: ws.RootPath, Status: StatusActive}, nil
}

// RenameNode is a compatibility wrapper for physical workspace rename.
func (m *Manager) RenameNode(id, title string) error {
	return m.RenameWorkspace(id, title)
}

// MoveNode is unsupported in the corrected workspace model.
func (m *Manager) MoveNode(id, newParentID string) error {
	return fmt.Errorf("workspace folders are top-level only")
}

// ArchiveNode is a compatibility wrapper for trashing a workspace.
func (m *Manager) ArchiveNode(id string) error {
	_, err := m.TrashWorkspace(id)
	return err
}

// SetCurrentNode stores UI selection only.
func (m *Manager) SetCurrentNode(id string) error {
	if err := validateWorkspaceName(id); err != nil {
		return err
	}
	workspaces, err := m.ListWorkspaces()
	if err != nil {
		return err
	}
	if !workspaceExists(workspaces, id) {
		return fmt.Errorf("workspace not found: %s", id)
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	m.currentWorkspaceName = id
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

func validateWorkspaceMetadataPath(name string) error {
	if name == "" {
		return fmt.Errorf("invalid-workspace-name: empty")
	}
	if strings.Contains(name, "\\x00") {
		return fmt.Errorf("invalid-workspace-name: null-byte")
	}
	if strings.Contains(name, "\\") {
		return fmt.Errorf("invalid-workspace-name: backslash not allowed")
	}
	if filepath.IsAbs(name) || strings.HasPrefix(name, "/") {
		return fmt.Errorf("invalid-workspace-name: absolute path rejected")
	}
	if len(name) >= 2 && name[1] == ':' && unicode.IsLetter(rune(name[0])) {
		return fmt.Errorf("invalid-workspace-name: absolute path rejected")
	}
	parts := strings.Split(name, "/")
	for _, part := range parts {
		if part == "" || part == "." || part == ".." {
			return fmt.Errorf("invalid-workspace-name: path traversal")
		}
		for _, r := range part {
			if unicode.IsControl(r) {
				return fmt.Errorf("invalid-workspace-name: control character")
			}
		}
		if isReservedWorkspaceName(part) {
			return fmt.Errorf("reserved-workspace-name: %s", part)
		}
	}
	return nil
}

// Deprecated: kept for top-level workspace name validation. Use validateWorkspaceMetadataPath.
func validateWorkspaceName(name string) error {
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

func ensureExistingWorkspaceDir(full, name string) error {
	info, err := os.Lstat(full)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("not-found: %s", name)
		}
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("symlink-not-allowed: %s", name)
	}
	if !info.IsDir() {
		return fmt.Errorf("not-directory: %s", name)
	}
	return nil
}

func (m *Manager) metadataPath(name string) string {
	encoded := base64.RawURLEncoding.EncodeToString([]byte(name))
	return filepath.Join(m.vaultDir, ".verstak", "workspaces", encoded+".json")
}

func (m *Manager) metadataPathForUUID(workspaceID string) string {
	return filepath.Join(m.vaultDir, ".verstak", "workspaces", "uuid-"+workspaceID+".json")
}

func (m *Manager) writeMetadata(name string, meta Metadata) error {
	if err := os.MkdirAll(filepath.Join(m.vaultDir, ".verstak", "workspaces"), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return err
	}
	path := m.metadataPath(name)
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	// Also write/update UUID-keyed metadata for tree-move resilience.
	if meta.WorkspaceID != "" {
		_ = m.writeMetadataForUUID(meta.WorkspaceID, meta)
	}
	return nil
}

func (m *Manager) writeMetadataForUUID(workspaceID string, meta Metadata) error {
	if _, err := uuid.Parse(workspaceID); err != nil {
		return fmt.Errorf("invalid workspace identity: %w", err)
	}
	if err := os.MkdirAll(filepath.Join(m.vaultDir, ".verstak", "workspaces"), 0o755); err != nil {
		return err
	}
	meta.WorkspaceID = workspaceID
	data, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return err
	}
	path := m.metadataPathForUUID(workspaceID)
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}

// GetWorkspaceMetadataByUUID returns metadata for a workspace keyed by its durable UUID.
// Falls back to generic metadata if no UUID-keyed file exists.
func (m *Manager) GetWorkspaceMetadataByUUID(workspaceID string) (Metadata, error) {
	if _, err := uuid.Parse(workspaceID); err != nil {
		return Metadata{}, fmt.Errorf("invalid workspace identity: %w", err)
	}
	data, err := os.ReadFile(m.metadataPathForUUID(workspaceID))
	if err != nil {
		if os.IsNotExist(err) {
			return Metadata{}, fmt.Errorf("not-found: workspace metadata for %s", workspaceID)
		}
		return Metadata{}, err
	}
	var meta Metadata
	if err := json.Unmarshal(data, &meta); err != nil {
		return Metadata{}, err
	}
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
		SelectedWorkspace: m.currentWorkspaceName,
		UpdatedAt:         time.Now().UTC().Format(time.RFC3339Nano),
	}
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	path := m.uiStatePath()
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}

func genericMetadata(name string) Metadata {
	return Metadata{
		WorkspaceName: name,
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

func workspaceExists(workspaces []Workspace, name string) bool {
	for _, ws := range workspaces {
		if ws.Name == name {
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
