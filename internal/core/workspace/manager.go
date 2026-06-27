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
	Name     string `json:"name"`
	RootPath string `json:"rootPath"`
}

// TemplateSnapshot is copied into workspace metadata when a template is applied.
type TemplateSnapshot struct {
	TemplateID      string `json:"templateId"`
	TemplateName    string `json:"templateName"`
	TemplateVersion int    `json:"templateVersion"`
	AppliedAt       string `json:"appliedAt"`
}

// Metadata stores semantic workspace metadata that is not the source of truth
// for whether the workspace exists.
type Metadata struct {
	WorkspaceName       string            `json:"workspaceName"`
	CreatedFromTemplate *TemplateSnapshot `json:"createdFromTemplate,omitempty"`
	Features            map[string]bool   `json:"features,omitempty"`
	Folders             map[string]string `json:"folders,omitempty"`
	UpdatedAt           string            `json:"updatedAt,omitempty"`
}

// MetadataPatch updates metadata fields without replacing unspecified fields.
type MetadataPatch struct {
	Features map[string]bool   `json:"features,omitempty"`
	Folders  map[string]string `json:"folders,omitempty"`
}

// TrashResult describes a workspace moved into the internal trash area.
type TrashResult struct {
	OriginalPath string `json:"originalPath"`
	TrashPath    string `json:"trashPath"`
	TrashID      string `json:"trashId"`
	DeletedAt    string `json:"deletedAt"`
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
	ID       string
	Name     string
	Version  int
	Features map[string]bool
	Folders  map[string]string
	Files    map[string]string
}

var builtInTemplates = map[string]templateDefinition{
	"default": {
		ID:      "default",
		Name:    "Default Workspace",
		Version: 1,
		Features: map[string]bool{
			"files":    true,
			"notes":    true,
			"secrets":  false,
			"activity": false,
		},
		Folders: map[string]string{
			"notes": "Notes",
			"files": "Files",
		},
		Files: map[string]string{},
	},
	"client-project": {
		ID:      "client-project",
		Name:    "Client Project",
		Version: 1,
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
		Files: map[string]string{},
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
		workspaces = append(workspaces, Workspace{Name: name, RootPath: name})
	}
	sort.Slice(workspaces, func(i, j int) bool {
		return strings.ToLower(workspaces[i].Name) < strings.ToLower(workspaces[j].Name)
	})
	return workspaces, nil
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

	if err := applyTemplate(full, template); err != nil {
		return Workspace{}, err
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	meta := Metadata{
		WorkspaceName: name,
		CreatedFromTemplate: &TemplateSnapshot{
			TemplateID:      template.ID,
			TemplateName:    template.Name,
			TemplateVersion: template.Version,
			AppliedAt:       now,
		},
		Features:  cloneBoolMap(template.Features),
		Folders:   cloneStringMap(template.Folders),
		UpdatedAt: now,
	}
	if err := m.writeMetadata(name, meta); err != nil {
		return Workspace{}, err
	}

	created = false
	return Workspace{Name: name, RootPath: name}, nil
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

	result := TrashResult{OriginalPath: name, TrashPath: trashRel, TrashID: trashID, DeletedAt: deletedAt}
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

// GetWorkspaceMetadata returns stored metadata or safe generic metadata.
func (m *Manager) GetWorkspaceMetadata(name string) (Metadata, error) {
	name = strings.TrimSpace(name)
	if err := validateWorkspaceName(name); err != nil {
		return Metadata{}, err
	}
	full := filepath.Join(m.vaultDir, name)
	if err := ensureExistingWorkspaceDir(full, name); err != nil {
		return Metadata{}, err
	}

	data, err := os.ReadFile(m.metadataPath(name))
	if err != nil {
		if os.IsNotExist(err) {
			return genericMetadata(name), nil
		}
		return Metadata{}, err
	}
	var meta Metadata
	if err := json.Unmarshal(data, &meta); err != nil {
		return Metadata{}, err
	}
	// Workspace identity is the top-level folder name. Stored metadata may be
	// stale after manual edits or old dev snapshots, so normalize the returned
	// presentation name without writing back to disk.
	meta.WorkspaceName = name
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
