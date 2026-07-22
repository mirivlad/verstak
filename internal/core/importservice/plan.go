package importservice

import (
	"fmt"
	"math"
	"path"
	"sort"
	"strings"
	"time"
	"unicode"
)

type validatedNode struct {
	PlanNode
	relPath string
	entry   *sourceEntry
	depth   int
}

type validatedPlan struct {
	runName       string
	nodes         []validatedNode
	requiredBytes int64
	result        ApplyResult
}

func validatePlan(plan Plan, handle string, source indexedSource, maxTextBytes int64) (validatedPlan, error) {
	if plan.SchemaVersion != 1 {
		return validatedPlan{}, sourceError("invalid-plan-schema", "unsupported schema")
	}
	if plan.SourceHandle == "" || plan.SourceHandle != handle {
		return validatedPlan{}, sourceError("source-handle-mismatch", "plan belongs to another source")
	}
	if plan.SourceFingerprint == "" || plan.SourceFingerprint != source.fingerprint() {
		return validatedPlan{}, sourceError("source-fingerprint-mismatch", "source changed after analysis")
	}
	if err := validateStructuralName(plan.RunName); err != nil {
		return validatedPlan{}, sourceError("invalid-plan-name", "invalid run name")
	}
	if len(plan.Nodes) > MaxEntries {
		return validatedPlan{}, sourceError("invalid-plan-size", "too many plan nodes")
	}

	byID := make(map[string]PlanNode, len(plan.Nodes))
	for _, node := range plan.Nodes {
		if node.ID == "" {
			return validatedPlan{}, sourceError("invalid-plan-node", "node ID is required")
		}
		if _, exists := byID[node.ID]; exists {
			return validatedPlan{}, sourceError("invalid-plan-node", "duplicate node ID")
		}
		if strings.TrimSpace(node.Name) == "" || containsControl(node.Name) {
			return validatedPlan{}, sourceError("invalid-plan-name", "invalid node name")
		}
		switch node.Kind {
		case "folder", "workspace", "note", "file", "skip":
		default:
			return validatedPlan{}, sourceError("invalid-plan-kind", "unknown node kind")
		}
		byID[node.ID] = node
	}

	visiting := make(map[string]bool, len(byID))
	visited := make(map[string]bool, len(byID))
	var visit func(string) error
	visit = func(id string) error {
		if visited[id] {
			return nil
		}
		if visiting[id] {
			return sourceError("invalid-plan-cycle", "parent graph contains a cycle")
		}
		visiting[id] = true
		node := byID[id]
		if node.ParentID != "" {
			if _, ok := byID[node.ParentID]; !ok {
				return sourceError("invalid-plan-parent", "parent does not exist")
			}
			if err := visit(node.ParentID); err != nil {
				return err
			}
		}
		delete(visiting, id)
		visited[id] = true
		return nil
	}
	for id := range byID {
		if err := visit(id); err != nil {
			return validatedPlan{}, err
		}
	}

	entries := make(map[string]*sourceEntry)
	sourceEntries := source.entries()
	for index := range sourceEntries {
		entry := &sourceEntries[index]
		entries[entry.ID] = entry
	}
	resolved := make(map[string]validatedNode, len(byID))
	var resolve func(string) (validatedNode, error)
	resolve = func(id string) (validatedNode, error) {
		if node, ok := resolved[id]; ok {
			return node, nil
		}
		raw := byID[id]
		result := validatedNode{PlanNode: raw}
		var parent validatedNode
		var err error
		if raw.ParentID != "" {
			parent, err = resolve(raw.ParentID)
			if err != nil {
				return validatedNode{}, err
			}
			result.depth = parent.depth + 1
		}

		switch raw.Kind {
		case "folder":
			if raw.ParentID != "" && parent.Kind != "folder" {
				return validatedNode{}, sourceError("invalid-plan-parent", "folder parent must be a folder")
			}
			if err := validateStructuralName(raw.Name); err != nil {
				return validatedNode{}, sourceError("invalid-plan-name", "invalid folder name")
			}
			result.relPath = path.Join(parent.relPath, raw.Name)
		case "workspace":
			if raw.ParentID != "" && parent.Kind != "folder" {
				return validatedNode{}, sourceError("invalid-plan-parent", "workspace parent must be a folder")
			}
			if err := validateStructuralName(raw.Name); err != nil {
				return validatedNode{}, sourceError("invalid-plan-name", "invalid workspace name")
			}
			if raw.TemplateID != "default" {
				return validatedNode{}, sourceError("invalid-plan-template", "only the default template is supported")
			}
			result.relPath = path.Join(parent.relPath, raw.Name)
		case "note", "file":
			if raw.ParentID == "" || parent.Kind != "workspace" {
				return validatedNode{}, sourceError("invalid-plan-parent", "content parent must be a workspace")
			}
			target, normalizeErr := normalizeTargetSubpath(raw.TargetSubpath)
			if normalizeErr != nil {
				return validatedNode{}, normalizeErr
			}
			base := "Notes"
			if raw.Kind == "file" {
				base = "Files"
				entry, ok := entries[raw.SourceEntryID]
				if !ok || entry.Kind != "file" {
					return validatedNode{}, sourceError("source-entry-not-found", "file source entry is unavailable")
				}
				if raw.SourcePath != "" && raw.SourcePath != entry.Path {
					return validatedNode{}, sourceError("source-entry-mismatch", "file source path changed")
				}
				entryCopy := *entry
				result.entry = &entryCopy
			} else {
				if int64(len([]byte(raw.Text))) > maxTextBytes {
					return validatedNode{}, sourceError("text-entry-too-large", "generated note is too large")
				}
			}
			if raw.ModifiedAt != "" {
				if _, parseErr := time.Parse(time.RFC3339Nano, raw.ModifiedAt); parseErr != nil {
					return validatedNode{}, sourceError("invalid-plan-time", "invalid modification time")
				}
			}
			result.relPath = path.Join(parent.relPath, base, target)
		case "skip":
			// A skipped node remains in the reviewed plan for accounting only.
		}
		resolved[id] = result
		return result, nil
	}

	paths := make(map[string]string)
	ensureDirectory := func(relative string, exclusive bool) error {
		parts := strings.Split(relative, "/")
		for index := range parts {
			current := strings.Join(parts[:index+1], "/")
			key := collisionKey(current)
			if kind, ok := paths[key]; ok {
				if kind == "file" || (exclusive && index == len(parts)-1) {
					return sourceError("duplicate-target-path", "target paths collide")
				}
				continue
			}
			paths[key] = "directory"
		}
		return nil
	}
	ensureFile := func(relative string) error {
		if parent := path.Dir(relative); parent != "." {
			if err := ensureDirectory(parent, false); err != nil {
				return err
			}
		}
		key := collisionKey(relative)
		if _, exists := paths[key]; exists {
			return sourceError("duplicate-target-path", "target paths collide")
		}
		paths[key] = "file"
		return nil
	}

	validated := validatedPlan{runName: plan.RunName, result: ApplyResult{Warnings: []string{}}}
	for _, raw := range plan.Nodes {
		node, err := resolve(raw.ID)
		if err != nil {
			return validatedPlan{}, err
		}
		validated.nodes = append(validated.nodes, node)
	}
	sort.SliceStable(validated.nodes, func(i, j int) bool { return validated.nodes[i].depth < validated.nodes[j].depth })
	for _, node := range validated.nodes {
		switch node.Kind {
		case "folder":
			if err := ensureDirectory(node.relPath, true); err != nil {
				return validatedPlan{}, err
			}
			validated.result.Folders++
		case "workspace":
			if err := ensureDirectory(node.relPath, true); err != nil {
				return validatedPlan{}, err
			}
			if err := ensureDirectory(path.Join(node.relPath, "Notes"), true); err != nil {
				return validatedPlan{}, err
			}
			if err := ensureDirectory(path.Join(node.relPath, "Files"), true); err != nil {
				return validatedPlan{}, err
			}
			validated.result.Workspaces++
		case "note":
			if err := ensureFile(node.relPath); err != nil {
				return validatedPlan{}, err
			}
			if validated.requiredBytes > math.MaxInt64-int64(len([]byte(node.Text))) {
				return validatedPlan{}, sourceError("invalid-plan-size", "planned size overflow")
			}
			validated.requiredBytes += int64(len([]byte(node.Text)))
			validated.result.Notes++
		case "file":
			if err := ensureFile(node.relPath); err != nil {
				return validatedPlan{}, err
			}
			if node.entry.Size > math.MaxInt64-validated.requiredBytes {
				return validatedPlan{}, sourceError("invalid-plan-size", "planned size overflow")
			}
			validated.requiredBytes += node.entry.Size
			validated.result.Files++
		case "skip":
			validated.result.Skipped++
		}
	}
	return validated, nil
}

func validateStructuralName(name string) error {
	if name == "" || name != strings.TrimSpace(name) || name == "." || name == ".." || strings.HasPrefix(name, ".") || strings.ContainsAny(name, `/\`) || containsControl(name) || invalidPortableSegment(name) || strings.EqualFold(name, ".verstak") {
		return fmt.Errorf("invalid entity name")
	}
	return nil
}

func containsControl(value string) bool {
	for _, character := range value {
		if unicode.IsControl(character) {
			return true
		}
	}
	return false
}

func normalizeTargetSubpath(value string) (string, error) {
	normalized, err := normalizeSourcePath(value)
	if err != nil {
		return "", sourceError("invalid-target-path", "target path is unsafe")
	}
	for _, segment := range strings.Split(normalized, "/") {
		if strings.EqualFold(segment, ".verstak") {
			return "", sourceError("invalid-target-path", "reserved target path")
		}
	}
	return normalized, nil
}
