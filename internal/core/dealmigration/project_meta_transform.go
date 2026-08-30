package dealmigration

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/verstak/verstak-desktop/internal/core/workspacetree"
)

// NewProjectMetaTransform converts one unambiguous legacy Project into the
// Project Meta namespace of its Deal. Several Projects for one Deal remain
// only in the migration backup: choosing one would silently discard meaning.
func NewProjectMetaTransform() Transform {
	return FuncTransform("legacy-project-meta-to-deal-config", migrateProjectMeta, verifyProjectMeta)
}

func migrateProjectMeta(ctx context.Context, vault string) error {
	projects, err := legacyProjects(vault)
	if err != nil {
		return err
	}
	byWorkspace := map[string][]map[string]any{}
	for _, project := range projects {
		if err := ctx.Err(); err != nil {
			return err
		}
		workspaceID := stringValue(project["workspaceId"])
		if isUUID(workspaceID) {
			byWorkspace[workspaceID] = append(byWorkspace[workspaceID], project)
		}
	}
	service := workspacetree.NewService(vault, nil)
	for workspaceID, candidates := range byWorkspace {
		if len(candidates) != 1 {
			continue
		}
		metadata, found, err := readProjectMetaTarget(service, workspaceID, candidates[0])
		if err != nil {
			return err
		}
		if !found {
			continue
		}
		if _, exists := metadata.ToolConfig[legacyProjectsPluginID]; exists {
			continue
		}
		config, err := json.Marshal(projectMetaConfig(candidates[0]))
		if err != nil {
			return fmt.Errorf("encode Project Meta for Deal %s: %w", workspaceID, err)
		}
		metadata.ToolConfig[legacyProjectsPluginID] = config
		metadata.WorkspaceTools = appendUniqueTool(metadata.WorkspaceTools, legacyProjectsPluginID)
		metadata.UpdatedAt = ""
		if err := service.WriteDealMetadata(metadata); err != nil {
			return err
		}
	}
	return nil
}

func readProjectMetaTarget(service *workspacetree.Service, workspaceID string, project map[string]any) (workspacetree.DealMetadata, bool, error) {
	metadata, err := service.ReadDealMetadata(workspaceID, strings.TrimSpace(stringValue(project["workspaceRootPath"])))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return workspacetree.DealMetadata{}, false, nil
		}
		return workspacetree.DealMetadata{}, false, fmt.Errorf("read canonical Deal metadata for Project Meta %s: %w", workspaceID, err)
	}
	if metadata.ToolConfig == nil {
		metadata.ToolConfig = map[string]json.RawMessage{}
	}
	return metadata, true, nil
}

func projectMetaConfig(project map[string]any) map[string]any {
	return map[string]any{
		"schemaVersion": 1,
		"name":          strings.TrimSpace(stringValue(project["name"])),
		"description":   strings.TrimSpace(stringValue(project["description"])),
		"status":        projectMetaStatus(stringValue(project["status"])),
		"priority":      projectMetaPriority(stringValue(project["priority"])),
		"tags":          projectMetaTags(project["tags"]),
		"startDate":     strings.TrimSpace(stringValue(project["startDate"])),
		"dueDate":       strings.TrimSpace(stringValue(project["dueDate"])),
		"updatedAt":     strings.TrimSpace(stringValue(project["updatedAt"])),
	}
}

func projectMetaStatus(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "active", "paused", "done", "archived":
		return strings.ToLower(strings.TrimSpace(value))
	default:
		return "active"
	}
}

func projectMetaPriority(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "low", "normal", "high":
		return strings.ToLower(strings.TrimSpace(value))
	default:
		return "normal"
	}
}

func projectMetaTags(value any) []string {
	values, _ := value.([]any)
	tags := make([]string, 0, len(values))
	seen := map[string]bool{}
	for _, raw := range values {
		tag := strings.TrimSpace(stringValue(raw))
		key := strings.ToLower(tag)
		if tag == "" || seen[key] || len(tags) == 20 {
			continue
		}
		seen[key] = true
		tags = append(tags, tag)
	}
	return tags
}

func appendUniqueTool(tools []string, toolID string) []string {
	for _, tool := range tools {
		if tool == toolID {
			return tools
		}
	}
	return append(tools, toolID)
}

func verifyProjectMeta(ctx context.Context, vault string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	// The provider and metadata transforms validate their respective durable
	// schemas. Legacy records are intentionally retained only in the backup.
	return nil
}
