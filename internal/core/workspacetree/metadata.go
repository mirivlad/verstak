package workspacetree

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
)

const DealMetadataSchemaVersion = 2

// TemplateProvenance records the recipe applied at Deal creation. It is
// historical provenance only and never controls the Deal runtime.
type TemplateProvenance struct {
	TemplateID      string `json:"templateId"`
	TemplateName    string `json:"templateName,omitempty"`
	TemplateVersion int    `json:"templateVersion,omitempty"`
	AppliedAt       string `json:"appliedAt"`
	RecipeDigest    string `json:"recipeDigest,omitempty"`
}

// DealMetadata is the canonical UUID-keyed metadata record for a Deal.
type DealMetadata struct {
	SchemaVersion       int                        `json:"schemaVersion"`
	WorkspaceID         string                     `json:"workspaceId"`
	WorkspaceName       string                     `json:"workspaceName"`
	WorkspaceTools      []string                   `json:"workspaceTools"`
	ToolConfig          map[string]json.RawMessage `json:"toolConfig"`
	CreatedFromTemplate *TemplateProvenance        `json:"createdFromTemplate,omitempty"`
	UpdatedAt           string                     `json:"updatedAt"`
}

type legacyDealMetadata struct {
	WorkspaceID         string              `json:"workspaceId"`
	WorkspaceName       string              `json:"workspaceName"`
	WorkspaceTools      []string            `json:"workspaceTools"`
	Features            map[string]bool     `json:"features"`
	CreatedFromTemplate *TemplateProvenance `json:"createdFromTemplate"`
	UpdatedAt           string              `json:"updatedAt"`
}

func (s *Service) metadataVaultDir() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.vaultDir
}

func canonicalDealMetadataPath(vaultDir, workspaceID string) string {
	return filepath.Join(vaultDir, ".verstak", "workspaces", "uuid-"+workspaceID+".json")
}

func legacyDealMetadataPath(vaultDir, rootPath string) string {
	encoded := base64.RawURLEncoding.EncodeToString([]byte(filepath.ToSlash(rootPath)))
	return filepath.Join(vaultDir, ".verstak", "workspaces", encoded+".json")
}

// ReadDealMetadata reads the canonical UUID-keyed metadata record. Path-keyed
// files are one-shot migration input and are never part of the runtime path.
func (s *Service) ReadDealMetadata(workspaceID, rootPath string) (DealMetadata, error) {
	if _, err := uuid.Parse(workspaceID); err != nil {
		return DealMetadata{}, fmt.Errorf("invalid workspace identity: %w", err)
	}
	vaultDir := s.metadataVaultDir()
	canonicalPath := canonicalDealMetadataPath(vaultDir, workspaceID)
	data, err := os.ReadFile(canonicalPath)
	if err == nil {
		return decodeDealMetadata(data, workspaceID, filepath.Base(filepath.FromSlash(rootPath)))
	}
	if !errors.Is(err, os.ErrNotExist) {
		return DealMetadata{}, fmt.Errorf("read Deal metadata: %w", err)
	}
	return DealMetadata{}, fmt.Errorf("Deal metadata not found for %s: %w", workspaceID, os.ErrNotExist)
}

// WriteDealMetadata validates and atomically writes canonical UUID metadata.
func (s *Service) WriteDealMetadata(metadata DealMetadata) error {
	data, normalized, err := marshalDealMetadata(metadata)
	if err != nil {
		return err
	}
	dir := filepath.Dir(canonicalDealMetadataPath(s.metadataVaultDir(), normalized.WorkspaceID))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create Deal metadata registry: %w", err)
	}
	tmp, err := os.CreateTemp(dir, ".deal-metadata-*.tmp")
	if err != nil {
		return fmt.Errorf("create Deal metadata temporary file: %w", err)
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)
	if err := tmp.Chmod(0o600); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("set Deal metadata permissions: %w", err)
	}
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("write Deal metadata: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("sync Deal metadata: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close Deal metadata: %w", err)
	}
	path := canonicalDealMetadataPath(s.metadataVaultDir(), normalized.WorkspaceID)
	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("publish Deal metadata: %w", err)
	}
	return nil
}

func marshalDealMetadata(metadata DealMetadata) ([]byte, DealMetadata, error) {
	metadata.SchemaVersion = DealMetadataSchemaVersion
	metadata.WorkspaceID = strings.TrimSpace(metadata.WorkspaceID)
	metadata.WorkspaceName = strings.TrimSpace(metadata.WorkspaceName)
	metadata.WorkspaceTools = normalizeWorkspaceTools(metadata.WorkspaceTools)
	if metadata.ToolConfig == nil {
		metadata.ToolConfig = map[string]json.RawMessage{}
	}
	if metadata.UpdatedAt == "" {
		metadata.UpdatedAt = time.Now().UTC().Format(time.RFC3339Nano)
	}
	if err := validateDealMetadata(metadata, metadata.WorkspaceID); err != nil {
		return nil, DealMetadata{}, err
	}
	data, err := json.Marshal(metadata)
	if err != nil {
		return nil, DealMetadata{}, fmt.Errorf("marshal Deal metadata: %w", err)
	}
	return data, metadata, nil
}

// marshalWorkspaceMetadataV2 is retained until import and Templates both send
// complete recipe snapshots. Unlike the old implementation, it emits the
// canonical typed schema and no feature/folder compatibility maps.
func marshalWorkspaceMetadataV2(workspaceID, workspaceName, templateID string, workspaceTools []string) ([]byte, error) {
	metadata := newDealMetadata(workspaceID, workspaceName, templateID, workspaceTools)
	data, _, err := marshalDealMetadata(metadata)
	return data, err
}

func newDealMetadata(workspaceID, workspaceName, templateID string, workspaceTools []string) DealMetadata {
	now := time.Now().UTC().Format(time.RFC3339Nano)
	metadata := DealMetadata{
		SchemaVersion:  DealMetadataSchemaVersion,
		WorkspaceID:    workspaceID,
		WorkspaceName:  workspaceName,
		WorkspaceTools: workspaceTools,
		ToolConfig:     map[string]json.RawMessage{},
		UpdatedAt:      now,
	}
	if templateID != "" {
		metadata.CreatedFromTemplate = &TemplateProvenance{
			TemplateID: templateID,
			AppliedAt:  now,
		}
	}
	return metadata
}

// MigrateLegacyDealMetadata materializes one compatible legacy record as
// canonical v2 metadata while retaining the source bytes for the one-shot
// migration backup.
func (s *Service) MigrateLegacyDealMetadata(workspaceID, rootPath string) (DealMetadata, bool, error) {
	if _, err := uuid.Parse(workspaceID); err != nil {
		return DealMetadata{}, false, fmt.Errorf("invalid workspace identity: %w", err)
	}
	vaultDir := s.metadataVaultDir()
	canonicalPath := canonicalDealMetadataPath(vaultDir, workspaceID)
	if data, err := os.ReadFile(canonicalPath); err == nil {
		var envelope struct {
			SchemaVersion int `json:"schemaVersion"`
		}
		if err := json.Unmarshal(data, &envelope); err != nil {
			return DealMetadata{}, false, fmt.Errorf("decode Deal metadata version: %w", err)
		}
		metadata, err := decodeDealMetadata(data, workspaceID, filepath.Base(filepath.FromSlash(rootPath)))
		if err != nil {
			return DealMetadata{}, false, err
		}
		if envelope.SchemaVersion == DealMetadataSchemaVersion {
			return metadata, false, nil
		}
		if err := s.WriteDealMetadata(metadata); err != nil {
			return DealMetadata{}, false, err
		}
		return metadata, true, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return DealMetadata{}, false, fmt.Errorf("read Deal metadata: %w", err)
	}

	if strings.TrimSpace(rootPath) == "" {
		return DealMetadata{}, false, fmt.Errorf("legacy Deal path is required")
	}
	data, err := os.ReadFile(legacyDealMetadataPath(vaultDir, rootPath))
	if err != nil {
		return DealMetadata{}, false, fmt.Errorf("read legacy Deal metadata: %w", err)
	}
	metadata, err := decodeDealMetadata(data, workspaceID, filepath.Base(filepath.FromSlash(rootPath)))
	if err != nil {
		return DealMetadata{}, false, err
	}
	if err := s.WriteDealMetadata(metadata); err != nil {
		return DealMetadata{}, false, err
	}
	return metadata, true, nil
}

func decodeDealMetadata(data []byte, expectedWorkspaceID, fallbackName string) (DealMetadata, error) {
	var envelope struct {
		SchemaVersion int `json:"schemaVersion"`
	}
	if err := json.Unmarshal(data, &envelope); err != nil {
		return DealMetadata{}, fmt.Errorf("decode Deal metadata version: %w", err)
	}
	if envelope.SchemaVersion != 0 && envelope.SchemaVersion != DealMetadataSchemaVersion {
		return DealMetadata{}, fmt.Errorf("unsupported Deal metadata schema version %d", envelope.SchemaVersion)
	}

	var metadata DealMetadata
	if envelope.SchemaVersion == DealMetadataSchemaVersion {
		if err := json.Unmarshal(data, &metadata); err != nil {
			return DealMetadata{}, fmt.Errorf("decode Deal metadata: %w", err)
		}
	} else {
		var legacy legacyDealMetadata
		if err := json.Unmarshal(data, &legacy); err != nil {
			return DealMetadata{}, fmt.Errorf("decode legacy Deal metadata: %w", err)
		}
		workspaceTools := legacy.WorkspaceTools
		if workspaceTools == nil && legacy.CreatedFromTemplate != nil {
			workspaceTools = toolsFromLegacyFeatures(legacy.Features)
		}
		metadata = DealMetadata{
			SchemaVersion:       DealMetadataSchemaVersion,
			WorkspaceID:         legacy.WorkspaceID,
			WorkspaceName:       legacy.WorkspaceName,
			WorkspaceTools:      workspaceTools,
			ToolConfig:          map[string]json.RawMessage{},
			CreatedFromTemplate: legacy.CreatedFromTemplate,
			UpdatedAt:           legacy.UpdatedAt,
		}
	}
	if metadata.WorkspaceID == "" {
		metadata.WorkspaceID = expectedWorkspaceID
	}
	if metadata.WorkspaceName == "" {
		metadata.WorkspaceName = fallbackName
	}
	if metadata.UpdatedAt == "" {
		metadata.UpdatedAt = time.Now().UTC().Format(time.RFC3339Nano)
	}
	if metadata.CreatedFromTemplate != nil && metadata.CreatedFromTemplate.AppliedAt == "" {
		metadata.CreatedFromTemplate.AppliedAt = metadata.UpdatedAt
	}
	metadata.WorkspaceTools = normalizeWorkspaceTools(metadata.WorkspaceTools)
	if metadata.ToolConfig == nil {
		metadata.ToolConfig = map[string]json.RawMessage{}
	}
	if err := validateDealMetadata(metadata, expectedWorkspaceID); err != nil {
		return DealMetadata{}, err
	}
	return metadata, nil
}

func toolsFromLegacyFeatures(features map[string]bool) []string {
	tools := []string{"verstak.notes", "verstak.files"}
	for _, mapping := range []struct {
		feature  string
		pluginID string
	}{
		{feature: "projects", pluginID: "verstak.projects"},
		{feature: "journal", pluginID: "verstak.journal"},
		{feature: "activity", pluginID: "verstak.activity"},
		{feature: "browser-inbox", pluginID: "verstak.browser-inbox"},
		{feature: "todo", pluginID: "verstak.todo"},
		{feature: "secrets", pluginID: "verstak.secrets"},
	} {
		if features[mapping.feature] {
			tools = append(tools, mapping.pluginID)
		}
	}
	return tools
}

func validateDealMetadata(metadata DealMetadata, expectedWorkspaceID string) error {
	if metadata.SchemaVersion != DealMetadataSchemaVersion {
		return fmt.Errorf("unsupported Deal metadata schema version %d", metadata.SchemaVersion)
	}
	if _, err := uuid.Parse(metadata.WorkspaceID); err != nil {
		return fmt.Errorf("invalid workspace identity: %w", err)
	}
	if expectedWorkspaceID != "" && metadata.WorkspaceID != expectedWorkspaceID {
		return fmt.Errorf("workspace identity mismatch: expected %s, got %s", expectedWorkspaceID, metadata.WorkspaceID)
	}
	if strings.TrimSpace(metadata.WorkspaceName) == "" {
		return fmt.Errorf("workspace name is required")
	}
	if _, err := time.Parse(time.RFC3339Nano, metadata.UpdatedAt); err != nil {
		return fmt.Errorf("invalid Deal metadata updatedAt: %w", err)
	}
	if provenance := metadata.CreatedFromTemplate; provenance != nil {
		if strings.TrimSpace(provenance.TemplateID) == "" {
			return fmt.Errorf("template provenance ID is required")
		}
		if _, err := time.Parse(time.RFC3339Nano, provenance.AppliedAt); err != nil {
			return fmt.Errorf("invalid template provenance appliedAt: %w", err)
		}
	}
	for pluginID, config := range metadata.ToolConfig {
		if pluginID == "" || strings.TrimSpace(pluginID) != pluginID {
			return fmt.Errorf("invalid tool config namespace %q", pluginID)
		}
		if len(config) == 0 || !json.Valid(config) {
			return fmt.Errorf("invalid tool config for %s", pluginID)
		}
	}
	return nil
}
