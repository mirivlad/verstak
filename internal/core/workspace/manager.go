// Package workspace provides the core workspace/cases service for Verstak.
// It manages a tree of workspaces, cases, and folders inside a vault.
//
// This is NOT notes/files/editor — it is the foundational layer that
// organizes work into a hierarchy. Plugins later reference workspace
// nodes via stable IDs.
package workspace

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// NodeType represents the type of a workspace node.
type NodeType string

const (
	TypeSpace  NodeType = "space"
	TypeCase   NodeType = "case"
	TypeFolder NodeType = "folder"
)

// NodeStatus represents the lifecycle status of a node.
type NodeStatus string

const (
	StatusActive   NodeStatus = "active"
	StatusSleeping NodeStatus = "sleeping"
	StatusArchived NodeStatus = "archived"
)

// WorkspaceNode is a single item in the workspace tree.
type WorkspaceNode struct {
	ID        string     `json:"id"`
	ParentID  string     `json:"parentId,omitempty"`
	Type      NodeType   `json:"type"`
	Title     string     `json:"title"`
	Status    NodeStatus `json:"status"`
	Tags      []string   `json:"tags,omitempty"`
	Order     int        `json:"order"`
	CreatedAt string     `json:"createdAt"`
	UpdatedAt string     `json:"updatedAt"`
}

// WorkspaceTree holds the full node tree and current selection.
type WorkspaceTree struct {
	SchemaVersion int             `json:"schemaVersion"`
	Nodes         []WorkspaceNode `json:"nodes"`
	CurrentNodeID string          `json:"currentNodeId,omitempty"`
	UpdatedAt     string          `json:"updatedAt"`
}

// Manager provides workspace operations.
type Manager struct {
	mu       sync.RWMutex
	tree     *WorkspaceTree
	vaultDir string
}

// NewManager creates a workspace manager for the given vault directory.
func NewManager(vaultDir string) *Manager {
	return &Manager{
		vaultDir: vaultDir,
	}
}

// workspaceFilePath returns the path to workspace.json inside the vault.
func (m *Manager) workspaceFilePath() string {
	return filepath.Join(m.vaultDir, ".verstak", "workspace.json")
}

// Load reads the workspace tree from disk.
// If no file exists, creates a default tree with a root node.
func (m *Manager) Load() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	path := m.workspaceFilePath()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			m.tree = m.defaultTree()
			return m.saveLocked()
		}
		return fmt.Errorf("failed to read workspace.json: %w", err)
	}

	var tree WorkspaceTree
	if err := json.Unmarshal(data, &tree); err != nil {
		// Corrupt: backup and create defaults
		backupPath := path + ".corrupt." + time.Now().Format("20060102-150405")
		os.WriteFile(backupPath, data, 0o600)
		m.tree = m.defaultTree()
		if saveErr := m.saveLocked(); saveErr != nil {
			return fmt.Errorf("corrupt workspace.json (backed up to %s), failed to save defaults: %w", backupPath, saveErr)
		}
		return fmt.Errorf("corrupt workspace.json (backed up to %s), defaults created", backupPath)
	}

	if tree.SchemaVersion != 1 {
		tree.SchemaVersion = 1
	}
	if tree.Nodes == nil {
		tree.Nodes = []WorkspaceNode{}
	}

	m.tree = &tree
	return nil
}

// saveLocked writes the workspace tree to disk atomically.
// Must be called with m.mu held (write lock).
func (m *Manager) saveLocked() error {
	if m.tree == nil {
		return fmt.Errorf("workspace tree is nil")
	}

	m.tree.UpdatedAt = time.Now().UTC().Format(time.RFC3339Nano)

	data, err := json.MarshalIndent(m.tree, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal workspace tree: %w", err)
	}

	path := m.workspaceFilePath()
	tmpPath := path + ".tmp"

	if err := os.WriteFile(tmpPath, data, 0o600); err != nil {
		return fmt.Errorf("failed to write workspace.json.tmp: %w", err)
	}

	if err := os.Rename(tmpPath, path); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("failed to rename workspace.json: %w", err)
	}

	return nil
}

// Save persists the current tree to disk.
func (m *Manager) Save() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.saveLocked()
}

// defaultTree creates a fresh workspace tree with a single root node.
func (m *Manager) defaultTree() *WorkspaceTree {
	now := time.Now().UTC().Format(time.RFC3339Nano)
	root := WorkspaceNode{
		ID:        uuid.New().String(),
		Type:      TypeSpace,
		Title:     "My Workspace",
		Status:    StatusActive,
		Order:     0,
		CreatedAt: now,
		UpdatedAt: now,
	}
	return &WorkspaceTree{
		SchemaVersion: 1,
		Nodes:         []WorkspaceNode{root},
		CurrentNodeID: root.ID,
		UpdatedAt:     now,
	}
}

// GetTree returns a copy of the full tree.
func (m *Manager) GetTree() WorkspaceTree {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.tree == nil {
		return WorkspaceTree{SchemaVersion: 1}
	}
	return *m.tree
}

// GetNode returns a node by ID.
func (m *Manager) GetNode(id string) (WorkspaceNode, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.tree == nil {
		return WorkspaceNode{}, fmt.Errorf("workspace not initialized")
	}
	for _, n := range m.tree.Nodes {
		if n.ID == id {
			return n, nil
		}
	}
	return WorkspaceNode{}, fmt.Errorf("node not found: %s", id)
}

// ListChildren returns direct children of a parent node, sorted by order.
func (m *Manager) ListChildren(parentID string) []WorkspaceNode {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.tree == nil {
		return nil
	}
	var children []WorkspaceNode
	for _, n := range m.tree.Nodes {
		if n.ParentID == parentID {
			children = append(children, n)
		}
	}
	sort.Slice(children, func(i, j int) bool {
		return children[i].Order < children[j].Order
	})
	return children
}

// CreateNode creates a new node under the given parent.
func (m *Manager) CreateNode(parentID string, nodeType NodeType, title string) (WorkspaceNode, error) {
	if nodeType != TypeSpace && nodeType != TypeCase && nodeType != TypeFolder {
		return WorkspaceNode{}, fmt.Errorf("invalid node type: %s", nodeType)
	}
	if strings.TrimSpace(title) == "" {
		return WorkspaceNode{}, fmt.Errorf("title cannot be empty")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if m.tree == nil {
		return WorkspaceNode{}, fmt.Errorf("workspace not initialized")
	}

	// Validate parent exists (empty parentID means root-level)
	if parentID != "" {
		parentFound := false
		for _, n := range m.tree.Nodes {
			if n.ID == parentID {
				parentFound = true
				break
			}
		}
		if !parentFound {
			return WorkspaceNode{}, fmt.Errorf("parent node not found: %s", parentID)
		}
	}

	now := time.Now().UTC().Format(time.RFC3339Nano)

	// Calculate order: max existing sibling order + 1
	maxOrder := -1
	for _, n := range m.tree.Nodes {
		if n.ParentID == parentID && n.Order > maxOrder {
			maxOrder = n.Order
		}
	}

	node := WorkspaceNode{
		ID:        uuid.New().String(),
		ParentID:  parentID,
		Type:      nodeType,
		Title:     title,
		Status:    StatusActive,
		Order:     maxOrder + 1,
		CreatedAt: now,
		UpdatedAt: now,
	}

	m.tree.Nodes = append(m.tree.Nodes, node)
	if err := m.saveLocked(); err != nil {
		// Rollback: remove the node we just added
		m.tree.Nodes = m.tree.Nodes[:len(m.tree.Nodes)-1]
		return WorkspaceNode{}, fmt.Errorf("failed to save after create: %w", err)
	}

	return node, nil
}

// RenameNode updates a node's title.
func (m *Manager) RenameNode(id, title string) error {
	if strings.TrimSpace(title) == "" {
		return fmt.Errorf("title cannot be empty")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if m.tree == nil {
		return fmt.Errorf("workspace not initialized")
	}

	for i := range m.tree.Nodes {
		if m.tree.Nodes[i].ID == id {
			m.tree.Nodes[i].Title = title
			m.tree.Nodes[i].UpdatedAt = time.Now().UTC().Format(time.RFC3339Nano)
			return m.saveLocked()
		}
	}
	return fmt.Errorf("node not found: %s", id)
}

// MoveNode changes a node's parent and order.
func (m *Manager) MoveNode(id, newParentID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.tree == nil {
		return fmt.Errorf("workspace not initialized")
	}

	// Find the node
	nodeIdx := -1
	for i := range m.tree.Nodes {
		if m.tree.Nodes[i].ID == id {
			nodeIdx = i
			break
		}
	}
	if nodeIdx < 0 {
		return fmt.Errorf("node not found: %s", id)
	}

	// Cannot move to self
	if newParentID == id {
		return fmt.Errorf("cannot move node into itself")
	}

	// Cannot move to own descendant
	if m.isDescendant(id, newParentID) {
		return fmt.Errorf("cannot move node into its own descendant")
	}

	// Validate new parent exists (empty = root level)
	if newParentID != "" {
		parentFound := false
		for _, n := range m.tree.Nodes {
			if n.ID == newParentID {
				parentFound = true
				break
			}
		}
		if !parentFound {
			return fmt.Errorf("parent node not found: %s", newParentID)
		}
	}

	// Calculate new order
	maxOrder := -1
	for _, n := range m.tree.Nodes {
		if n.ParentID == newParentID && n.Order > maxOrder {
			maxOrder = n.Order
		}
	}

	m.tree.Nodes[nodeIdx].ParentID = newParentID
	m.tree.Nodes[nodeIdx].Order = maxOrder + 1
	m.tree.Nodes[nodeIdx].UpdatedAt = time.Now().UTC().Format(time.RFC3339Nano)

	return m.saveLocked()
}

// isDescendant checks if targetID is a descendant of ancestorID.
func (m *Manager) isDescendant(ancestorID, targetID string) bool {
	if targetID == "" {
		return false
	}
	// Build parent map
	parentMap := make(map[string]string)
	for _, n := range m.tree.Nodes {
		parentMap[n.ID] = n.ParentID
	}
	// Walk up from target
	current := targetID
	for current != "" {
		if current == ancestorID {
			return true
		}
		current = parentMap[current]
	}
	return false
}

// ArchiveNode sets a node's status to archived.
func (m *Manager) ArchiveNode(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.tree == nil {
		return fmt.Errorf("workspace not initialized")
	}

	for i := range m.tree.Nodes {
		if m.tree.Nodes[i].ID == id {
			m.tree.Nodes[i].Status = StatusArchived
			m.tree.Nodes[i].UpdatedAt = time.Now().UTC().Format(time.RFC3339Nano)
			return m.saveLocked()
		}
	}
	return fmt.Errorf("node not found: %s", id)
}

// SetCurrentNode sets the currently selected node.
func (m *Manager) SetCurrentNode(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.tree == nil {
		return fmt.Errorf("workspace not initialized")
	}

	// Validate node exists
	found := false
	for _, n := range m.tree.Nodes {
		if n.ID == id {
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("node not found: %s", id)
	}

	m.tree.CurrentNodeID = id
	m.tree.UpdatedAt = time.Now().UTC().Format(time.RFC3339Nano)
	return m.saveLocked()
}

// GetCurrentNode returns the currently selected node.
func (m *Manager) GetCurrentNode() (WorkspaceNode, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.tree == nil || m.tree.CurrentNodeID == "" {
		return WorkspaceNode{}, fmt.Errorf("no current node")
	}

	for _, n := range m.tree.Nodes {
		if n.ID == m.tree.CurrentNodeID {
			return n, nil
		}
	}
	return WorkspaceNode{}, fmt.Errorf("current node not found: %s", m.tree.CurrentNodeID)
}

// IsInitialized returns true if the workspace has been loaded.
func (m *Manager) IsInitialized() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.tree != nil
}
