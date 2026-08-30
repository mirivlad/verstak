package dealmigration

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	milestonesPluginID = "verstak.milestones"
	milestonesDataName = "milestones"
)

// NewMilestoneDataTransform transfers legacy embedded Project milestones to
// the independent Milestones provider. Legacy Projects remain untouched in
// the migration backup and are never consulted by the new runtime.
func NewMilestoneDataTransform() Transform {
	return FuncTransform("legacy-milestones-to-deal-provider", migrateLegacyMilestones, verifyLegacyMilestones)
}

func migrateLegacyMilestones(ctx context.Context, vault string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	projects, err := legacyProjects(vault)
	if err != nil {
		return err
	}
	path := filepath.Join(vault, ".verstak", "plugin-data", milestonesPluginID, milestonesDataName+".ndjson")
	data, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	existing := []any{}
	if err == nil {
		value, err := decodeProviderValue(data, true)
		if err != nil {
			return fmt.Errorf("decode milestone records: %w", err)
		}
		var ok bool
		existing, ok = value.([]any)
		if !ok {
			return fmt.Errorf("milestone records must be an array")
		}
	}
	seen := make(map[string]bool, len(existing))
	for _, record := range existing {
		if item, ok := record.(map[string]any); ok && stringValue(item["id"]) != "" {
			seen[stringValue(item["id"])] = true
		}
	}
	changed := false
	for _, project := range projects {
		if err := ctx.Err(); err != nil {
			return err
		}
		workspaceID := stringValue(project["workspaceId"])
		if !isUUID(workspaceID) {
			continue
		}
		legacyProjectID := stringValue(project["id"])
		legacyProjectName := stringValue(project["name"])
		milestones, _ := project["milestones"].([]any)
		for index, raw := range milestones {
			item, ok := raw.(map[string]any)
			if !ok {
				continue
			}
			title := strings.TrimSpace(stringValue(item["title"]))
			if title == "" {
				continue
			}
			id := uniqueMigratedMilestoneID(strings.TrimSpace(stringValue(item["id"])), legacyProjectID, index, seen)
			status := strings.TrimSpace(strings.ToLower(stringValue(item["status"])))
			if status != "done" && status != "cancelled" {
				status = "open"
			}
			createdAt := stringValue(item["createdAt"])
			if createdAt == "" {
				createdAt = stringValue(project["createdAt"])
			}
			record := map[string]any{
				"id":          id,
				"workspaceId": workspaceID,
				"title":       title,
				"status":      status,
				"dueAt":       stringValue(item["dueAt"]),
				"createdAt":   createdAt,
				"updatedAt":   stringValue(item["updatedAt"]),
				"completedAt": stringValue(item["completedAt"]),
				"migrationProvenance": map[string]any{
					"legacyProjectId":   legacyProjectID,
					"legacyProjectName": legacyProjectName,
				},
			}
			existing = append(existing, record)
			seen[id] = true
			changed = true
		}
	}
	if !changed {
		return nil
	}
	encoded, err := encodeProviderValue(existing, true)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	return writeFileAtomically(path, encoded)
}

func uniqueMigratedMilestoneID(preferred, legacyProjectID string, index int, seen map[string]bool) string {
	if preferred == "" {
		preferred = fmt.Sprintf("legacy-milestone:%s:%d", legacyProjectID, index)
	}
	if !seen[preferred] {
		return preferred
	}
	base := fmt.Sprintf("legacy-milestone:%s:%d", legacyProjectID, index)
	if !seen[base] {
		return base
	}
	for suffix := 2; ; suffix++ {
		candidate := fmt.Sprintf("%s:%d", base, suffix)
		if !seen[candidate] {
			return candidate
		}
	}
}

func legacyProjects(vault string) ([]map[string]any, error) {
	path := filepath.Join(vault, ".verstak", "plugin-settings", legacyProjectsPluginID, "settings.json")
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var settings map[string]any
	if err := json.Unmarshal(data, &settings); err != nil {
		return nil, fmt.Errorf("decode legacy Projects settings: %w", err)
	}
	rawProjects, _ := settings[projectsSettingsKey].([]any)
	projects := make([]map[string]any, 0, len(rawProjects))
	for _, raw := range rawProjects {
		if project, ok := raw.(map[string]any); ok {
			projects = append(projects, project)
		}
	}
	return projects, nil
}

func verifyLegacyMilestones(ctx context.Context, vault string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	path := filepath.Join(vault, ".verstak", "plugin-data", milestonesPluginID, milestonesDataName+".ndjson")
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	value, err := decodeProviderValue(data, true)
	if err != nil {
		return err
	}
	for _, raw := range value.([]any) {
		record, ok := raw.(map[string]any)
		if !ok || !isUUID(stringValue(record["workspaceId"])) || strings.TrimSpace(stringValue(record["title"])) == "" {
			return fmt.Errorf("invalid migrated milestone record")
		}
		if _, hasProjectID := record["projectId"]; hasProjectID {
			return fmt.Errorf("migrated milestone retains projectId")
		}
	}
	return nil
}
