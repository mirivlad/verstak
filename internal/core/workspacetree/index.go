package workspacetree

import (
	"sort"
	"strings"
)

// TreeBuilder builds a TreeNode tree from scan results.
type TreeBuilder struct {
	folders    map[string]ScannedFolder
	workspaces map[string]ScannedWorkspace
	order      OrderState
	// children maps parent folder ID → child folder IDs and workspace IDs.
	folderChildren    map[string][]ScannedFolder
	workspaceChildren map[string][]ScannedWorkspace
}

// BuildTree constructs a TreeSnapshot from a ScanResult.
func BuildTree(scan *ScanResult, currentWorkspaceID string, revision uint64, order OrderState) *TreeSnapshot {
	b := &TreeBuilder{
		folders:           scan.Folders,
		workspaces:        scan.Workspaces,
		order:             order,
		folderChildren:    make(map[string][]ScannedFolder),
		workspaceChildren: make(map[string][]ScannedWorkspace),
	}
	return b.build(currentWorkspaceID, revision, scan.Warnings)
}

func (b *TreeBuilder) build(currentWorkspaceID string, revision uint64, warnings []TreeDiagnostic) *TreeSnapshot {
	// Group children by parent.
	for _, f := range b.folders {
		pid := f.ParentID
		b.folderChildren[pid] = append(b.folderChildren[pid], f)
	}
	for _, ws := range b.workspaces {
		// A workspace's parent is the folder that physically contains it.
		pid := b.findParentFolder(ws.RootPath)
		b.workspaceChildren[pid] = append(b.workspaceChildren[pid], ws)
	}

	// Build root nodes: folders and workspaces with no parent folder.
	roots := b.buildChildren("")

	// Collect workspace IDs from all roots to check current.
	allIDs := collectWorkspaceIDs(b.workspaces)
	if currentWorkspaceID != "" && !allIDs[currentWorkspaceID] {
		currentWorkspaceID = ""
	}

	return &TreeSnapshot{
		Roots:              roots,
		CurrentWorkspaceID: currentWorkspaceID,
		Revision:           revision,
		Warnings:           warnings,
	}
}

// buildChildren builds TreeNode children for a given parent folder ID.
func (b *TreeBuilder) buildChildren(parentID string) []TreeNode {
	var nodes []TreeNode

	// Add folders.
	for _, f := range b.folderChildren[parentID] {
		node := TreeNode{
			Key:      "folder:" + f.ID,
			Kind:     "folder",
			ID:       f.ID,
			Name:     f.Name,
			Path:     f.Path,
			Children: b.buildChildren(f.ID),
		}
		nodes = append(nodes, node)
	}

	// Add workspaces.
	for _, ws := range b.workspaceChildren[parentID] {
		node := TreeNode{
			Key:  "workspace:" + ws.ID,
			Kind: "workspace",
			ID:   ws.ID,
			Name: ws.Name,
			Path: ws.RootPath,
		}
		nodes = append(nodes, node)
	}

	// Deterministic fallback: folders first, then workspaces, by folded name and
	// finally stable key so equal names do not inherit map iteration order.
	sort.SliceStable(nodes, func(i, j int) bool {
		if nodes[i].Kind != nodes[j].Kind {
			return nodes[i].Kind == "folder"
		}
		leftName := strings.ToLower(nodes[i].Name)
		rightName := strings.ToLower(nodes[j].Name)
		if leftName != rightName {
			return leftName < rightName
		}
		return nodes[i].Key < nodes[j].Key
	})

	parentKey := parentID
	if parentKey == "" {
		parentKey = "root"
	}
	stored := b.order.Children[parentKey]
	if len(stored) == 0 {
		return nodes
	}
	byKey := make(map[string]TreeNode, len(nodes))
	for _, node := range nodes {
		byKey[node.Key] = node
	}
	ordered := make([]TreeNode, 0, len(nodes))
	for _, key := range stored {
		if node, ok := byKey[key]; ok {
			ordered = append(ordered, node)
			delete(byKey, key)
		}
	}
	for _, node := range nodes {
		if _, ok := byKey[node.Key]; ok {
			ordered = append(ordered, node)
		}
	}
	return ordered
}

// findParentFolder finds the folder UUID that physically contains a workspace path.
func (b *TreeBuilder) findParentFolder(workspacePath string) string {
	parent := parentPath(workspacePath)
	if parent == "" {
		return ""
	}
	// Find a folder whose path matches the parent directory.
	for _, f := range b.folders {
		if f.Path == parent {
			return f.ID
		}
	}
	// If no folder marker, check if there's a folder that contains this path.
	// Walk up the path to find the nearest parent folder.
	for parent != "" {
		for _, f := range b.folders {
			if f.Path == parent {
				return f.ID
			}
		}
		parent = parentPath(parent)
	}
	return ""
}

func collectWorkspaceIDs(workspaces map[string]ScannedWorkspace) map[string]bool {
	ids := make(map[string]bool, len(workspaces))
	for id := range workspaces {
		ids[id] = true
	}
	return ids
}
