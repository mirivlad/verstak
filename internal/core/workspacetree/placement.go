package workspacetree

import (
	"fmt"
	"os"
	"path/filepath"
)

const (
	PlacementBefore = "before"
	PlacementAfter  = "after"
	PlacementInside = "inside"
	PlacementRoot   = "root"
)

type PlacementRequest struct {
	SourceKey string `json:"sourceKey"`
	TargetKey string `json:"targetKey"`
	Position  string `json:"position"`
}

type placementNode struct {
	Key      string
	Kind     string
	ID       string
	Name     string
	Path     string
	ParentID string
}

// PlaceNode validates and applies one stable-key tree placement. Parent
// resolution and filesystem movement remain entirely backend-owned.
func (s *Service) PlaceNode(request PlacementRequest, refreshBaseline func() error) (OrderState, error) {
	if !validOrderNodeKey(request.SourceKey) {
		return OrderState{}, fmt.Errorf("invalid source key: %s", request.SourceKey)
	}
	switch request.Position {
	case PlacementBefore, PlacementAfter, PlacementInside:
		if !validOrderNodeKey(request.TargetKey) {
			return OrderState{}, fmt.Errorf("invalid target key: %s", request.TargetKey)
		}
	case PlacementRoot:
		if request.TargetKey != "" {
			return OrderState{}, fmt.Errorf("root placement does not accept a target")
		}
	default:
		return OrderState{}, fmt.Errorf("unsupported placement position: %s", request.Position)
	}

	nodes, siblings := s.placementTopology()
	source, ok := nodes[request.SourceKey]
	if !ok {
		return OrderState{}, fmt.Errorf("source node not found: %s", request.SourceKey)
	}
	var target placementNode
	if request.Position != PlacementRoot {
		var targetOK bool
		target, targetOK = nodes[request.TargetKey]
		if !targetOK {
			return OrderState{}, fmt.Errorf("target node not found: %s", request.TargetKey)
		}
		if source.Key == target.Key {
			return OrderState{}, fmt.Errorf("cannot place a node relative to itself")
		}
	}

	destinationParentID := ""
	switch request.Position {
	case PlacementBefore, PlacementAfter:
		destinationParentID = target.ParentID
	case PlacementInside:
		if target.Kind != "folder" {
			return OrderState{}, fmt.Errorf("inside placement requires a folder target")
		}
		destinationParentID = target.ID
	}

	if source.Kind == "folder" && destinationParentID != "" {
		parent, ok := nodes["folder:"+destinationParentID]
		if !ok {
			return OrderState{}, fmt.Errorf("target parent folder not found: %s", destinationParentID)
		}
		if parent.Path == source.Path || isPathPrefix(source.Path, parent.Path) {
			return OrderState{}, fmt.Errorf("cannot move folder into itself or descendant")
		}
	}

	if source.ParentID != destinationParentID {
		if err := s.validatePlacementDestination(source, destinationParentID, nodes); err != nil {
			return OrderState{}, err
		}
		var err error
		if source.Kind == "folder" {
			_, err = s.MoveFolder(source.ID, destinationParentID, refreshBaseline)
		} else {
			_, err = s.MoveWorkspace(source.ID, destinationParentID, refreshBaseline)
		}
		if err != nil {
			return OrderState{}, err
		}
		nodes, siblings = s.placementTopology()
	}

	state, err := ReadOrderState(s.vaultDir)
	if err != nil {
		return OrderState{}, err
	}
	for parent, keys := range state.Children {
		state.Children[parent] = removePlacementKey(keys, source.Key)
	}

	oldParentKey := orderParentKey(source.ParentID)
	destinationParentKey := orderParentKey(destinationParentID)
	state.Children[oldParentKey] = append([]string(nil), siblings[source.ParentID]...)
	if oldParentKey != destinationParentKey {
		state.Children[destinationParentKey] = append([]string(nil), siblings[destinationParentID]...)
	}
	destination := removePlacementKey(state.Children[destinationParentKey], source.Key)
	insertAt := len(destination)
	if request.Position == PlacementBefore || request.Position == PlacementAfter {
		targetIndex := indexPlacementKey(destination, target.Key)
		if targetIndex < 0 {
			return OrderState{}, fmt.Errorf("target is not in resolved destination parent: %s", target.Key)
		}
		insertAt = targetIndex
		if request.Position == PlacementAfter {
			insertAt++
		}
	}
	destination = append(destination, "")
	copy(destination[insertAt+1:], destination[insertAt:])
	destination[insertAt] = source.Key
	state.Children[destinationParentKey] = destination

	if err := WriteOrderState(s.vaultDir, state); err != nil {
		return OrderState{}, err
	}
	if err := s.fullReconcile(); err != nil {
		return OrderState{}, err
	}
	return state, nil
}

func (s *Service) validatePlacementDestination(source placementNode, parentID string, nodes map[string]placementNode) error {
	parentPath := ""
	if parentID != "" {
		parent, ok := nodes["folder:"+parentID]
		if !ok {
			return fmt.Errorf("target parent folder not found: %s", parentID)
		}
		parentPath = parent.Path
	}
	newRel := joinRelPath(parentPath, source.Name)
	if newRel == source.Path {
		return nil
	}
	newAbs := filepath.Join(s.vaultDir, filepath.FromSlash(newRel))
	if _, err := os.Lstat(newAbs); err == nil {
		return fmt.Errorf("conflict: %s already exists", newRel)
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("validate placement destination: %w", err)
	}
	return nil
}

func (s *Service) placementTopology() (map[string]placementNode, map[string][]string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	nodes := make(map[string]placementNode)
	siblings := make(map[string][]string)
	if s.tree == nil {
		return nodes, siblings
	}
	var walk func([]TreeNode, string)
	walk = func(children []TreeNode, parentID string) {
		for _, node := range children {
			nodes[node.Key] = placementNode{
				Key:      node.Key,
				Kind:     node.Kind,
				ID:       node.ID,
				Name:     node.Name,
				Path:     node.Path,
				ParentID: parentID,
			}
			siblings[parentID] = append(siblings[parentID], node.Key)
			if node.Kind == "folder" {
				walk(node.Children, node.ID)
			}
		}
	}
	walk(s.tree.Roots, "")
	return nodes, siblings
}

func orderParentKey(parentID string) string {
	if parentID == "" {
		return "root"
	}
	return parentID
}

func removePlacementKey(keys []string, key string) []string {
	result := make([]string, 0, len(keys))
	for _, candidate := range keys {
		if candidate != key {
			result = append(result, candidate)
		}
	}
	return result
}

func indexPlacementKey(keys []string, key string) int {
	for i, candidate := range keys {
		if candidate == key {
			return i
		}
	}
	return -1
}
