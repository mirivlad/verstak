package dealmigration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
)

const (
	legacyProjectsPluginID = "verstak.projects"
	projectsSettingsKey    = "projects:global"
)

// NewProviderDataTransform returns the one-shot provider-content conversion.
// It reads legacy Projects only as a migration lookup table; no migrated
// provider record retains a Project scope and runtime code never consumes it.
func NewProviderDataTransform() Transform {
	return FuncTransform("provider-data-to-deal-scope", migrateProviderData, verifyProviderData)
}

func migrateProviderData(ctx context.Context, vault string) error {
	projectToWorkspace, err := legacyProjectWorkspaceIDs(vault)
	if err != nil {
		return err
	}
	rootToWorkspace, err := currentWorkspaceRootIDs(vault)
	if err != nil {
		return err
	}
	for _, root := range []string{
		filepath.Join(vault, ".verstak", "plugin-settings"),
		filepath.Join(vault, ".verstak", "plugin-data"),
	} {
		if err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				if os.IsNotExist(walkErr) {
					return nil
				}
				return walkErr
			}
			if err := ctx.Err(); err != nil {
				return err
			}
			if entry.IsDir() || filepath.Base(path) != "settings.json" && filepath.Ext(path) != ".ndjson" {
				return nil
			}
			rel, err := filepath.Rel(root, path)
			if err != nil {
				return err
			}
			pluginID := strings.Split(filepath.ToSlash(rel), "/")[0]
			if pluginID == legacyProjectsPluginID {
				return nil
			}
			return migrateProviderFile(path, projectToWorkspace, rootToWorkspace)
		}); err != nil {
			return fmt.Errorf("migrate provider data: %w", err)
		}
	}
	return nil
}

// currentWorkspaceRootIDs is migration-only lookup data from canonical Deal
// markers. Paths are retained as display data, never as runtime scope.
func currentWorkspaceRootIDs(vault string) (map[string]string, error) {
	mapping := map[string]string{}
	err := filepath.WalkDir(vault, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || entry.Name() != "workspace.json" || filepath.Base(filepath.Dir(path)) != ".verstak" {
			return nil
		}
		if entry.Type()&fs.ModeSymlink != 0 {
			return fmt.Errorf("reject symlink Deal marker %s", filepath.ToSlash(path))
		}
		workspaceDir := filepath.Dir(filepath.Dir(path))
		rel, err := filepath.Rel(vault, workspaceDir)
		if err != nil {
			return err
		}
		root := filepath.ToSlash(filepath.Clean(rel))
		if root == "." || strings.HasPrefix(root, ".verstak/") {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		var marker struct {
			WorkspaceID string `json:"workspaceId"`
		}
		if err := json.Unmarshal(data, &marker); err != nil {
			return fmt.Errorf("decode Deal marker %s: %w", filepath.ToSlash(rel), err)
		}
		if !isUUID(marker.WorkspaceID) {
			return fmt.Errorf("invalid Deal UUID in marker %s", filepath.ToSlash(rel))
		}
		mapping[root] = marker.WorkspaceID
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("scan Deal markers: %w", err)
	}
	return mapping, nil
}

func legacyProjectWorkspaceIDs(vault string) (map[string]string, error) {
	path := filepath.Join(vault, ".verstak", "plugin-settings", legacyProjectsPluginID, "settings.json")
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return map[string]string{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read legacy Projects map: %w", err)
	}
	var settings map[string]any
	if err := json.Unmarshal(data, &settings); err != nil {
		return nil, fmt.Errorf("decode legacy Projects map: %w", err)
	}
	projects, _ := settings[projectsSettingsKey].([]any)
	mapping := make(map[string]string, len(projects))
	for _, raw := range projects {
		project, _ := raw.(map[string]any)
		projectID := stringValue(project["id"])
		workspaceID := stringValue(project["workspaceId"])
		if projectID != "" && workspaceID != "" && isUUID(workspaceID) {
			mapping[projectID] = workspaceID
		}
	}
	return mapping, nil
}

func migrateProviderFile(path string, projectToWorkspace, rootToWorkspace map[string]string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	isNDJSON := filepath.Ext(path) == ".ndjson"
	value, err := decodeProviderValue(data, isNDJSON)
	if err != nil {
		return fmt.Errorf("decode %s: %w", path, err)
	}
	if settings, ok := value.(map[string]any); ok {
		delete(settings, "notes:projectScopes")
		delete(settings, "files:projectScopes")
	}
	migrateProviderValue(value, projectToWorkspace, rootToWorkspace)
	encoded, err := encodeProviderValue(value, isNDJSON)
	if err != nil {
		return fmt.Errorf("encode %s: %w", path, err)
	}
	if bytes.Equal(data, encoded) {
		return nil
	}
	return writeFileAtomically(path, encoded)
}

func decodeProviderValue(data []byte, ndjson bool) (any, error) {
	if !ndjson {
		var value any
		if err := json.Unmarshal(data, &value); err != nil {
			return nil, err
		}
		return value, nil
	}
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 {
		return []any{}, nil
	}
	if trimmed[0] == '[' {
		var records []any
		if err := json.Unmarshal(trimmed, &records); err != nil {
			return nil, err
		}
		return records, nil
	}
	records := []any{}
	for _, line := range bytes.Split(trimmed, []byte{'\n'}) {
		line = bytes.TrimSpace(line)
		if len(line) == 0 {
			continue
		}
		var record any
		if err := json.Unmarshal(line, &record); err != nil {
			return nil, err
		}
		records = append(records, record)
	}
	return records, nil
}

func encodeProviderValue(value any, ndjson bool) ([]byte, error) {
	if !ndjson {
		return json.MarshalIndent(value, "", "  ")
	}
	records, ok := value.([]any)
	if !ok {
		return nil, fmt.Errorf("NDJSON root must be an array")
	}
	var output bytes.Buffer
	for _, record := range records {
		data, err := json.Marshal(record)
		if err != nil {
			return nil, err
		}
		output.Write(data)
		output.WriteByte('\n')
	}
	return output.Bytes(), nil
}

func migrateProviderValue(value any, projectToWorkspace, rootToWorkspace map[string]string) {
	switch item := value.(type) {
	case []any:
		for _, child := range item {
			migrateProviderValue(child, projectToWorkspace, rootToWorkspace)
		}
	case map[string]any:
		projectID := stringValue(item["projectId"])
		delete(item, "projectId")
		if workspaceID := stringValue(item["workspaceId"]); workspaceID != "" && !isUUID(workspaceID) {
			delete(item, "workspaceId")
		}
		if workspaceID := projectToWorkspace[projectID]; workspaceID != "" && stringValue(item["workspaceId"]) == "" {
			item["workspaceId"] = workspaceID
		}
		if workspaceID := rootToWorkspace[stringValue(item["workspaceRootPath"])]; workspaceID != "" && stringValue(item["workspaceId"]) == "" {
			item["workspaceId"] = workspaceID
		}
		for _, child := range item {
			migrateProviderValue(child, projectToWorkspace, rootToWorkspace)
		}
	}
}

func verifyProviderData(ctx context.Context, vault string) error {
	for _, root := range []string{
		filepath.Join(vault, ".verstak", "plugin-settings"),
		filepath.Join(vault, ".verstak", "plugin-data"),
	} {
		if err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				if os.IsNotExist(walkErr) {
					return nil
				}
				return walkErr
			}
			if err := ctx.Err(); err != nil {
				return err
			}
			if entry.IsDir() || filepath.Base(path) != "settings.json" && filepath.Ext(path) != ".ndjson" {
				return nil
			}
			rel, err := filepath.Rel(root, path)
			if err != nil {
				return err
			}
			if strings.Split(filepath.ToSlash(rel), "/")[0] == legacyProjectsPluginID {
				return nil
			}
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			value, err := decodeProviderValue(data, filepath.Ext(path) == ".ndjson")
			if err != nil {
				return err
			}
			if containsProjectScope(value) {
				return fmt.Errorf("legacy Project scope remains in %s", filepath.ToSlash(rel))
			}
			if containsInvalidWorkspaceID(value) {
				return fmt.Errorf("invalid Deal UUID remains in %s", filepath.ToSlash(rel))
			}
			return nil
		}); err != nil {
			return err
		}
	}
	return nil
}

func containsProjectScope(value any) bool {
	switch item := value.(type) {
	case []any:
		for _, child := range item {
			if containsProjectScope(child) {
				return true
			}
		}
	case map[string]any:
		if _, ok := item["projectId"]; ok {
			return true
		}
		if _, ok := item["notes:projectScopes"]; ok {
			return true
		}
		if _, ok := item["files:projectScopes"]; ok {
			return true
		}
		for _, child := range item {
			if containsProjectScope(child) {
				return true
			}
		}
	}
	return false
}

func containsInvalidWorkspaceID(value any) bool {
	switch item := value.(type) {
	case []any:
		for _, child := range item {
			if containsInvalidWorkspaceID(child) {
				return true
			}
		}
	case map[string]any:
		if workspaceID := stringValue(item["workspaceId"]); workspaceID != "" && !isUUID(workspaceID) {
			return true
		}
		for _, child := range item {
			if containsInvalidWorkspaceID(child) {
				return true
			}
		}
	}
	return false
}

func isUUID(value string) bool {
	_, err := uuid.Parse(value)
	return err == nil
}

func stringValue(value any) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(fmt.Sprint(value))
}

func writeFileAtomically(path string, data []byte) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), ".deal-migration-*")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)
	if err := tmp.Chmod(0o600); err != nil {
		tmp.Close()
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpPath, path)
}
