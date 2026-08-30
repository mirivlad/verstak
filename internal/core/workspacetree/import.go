package workspacetree

import "github.com/google/uuid"

// PreparedImportedWorkspace contains the identity and registry record for a
// workspace assembled outside the vault and not published yet.
type PreparedImportedWorkspace struct {
	ID           string
	RegistryJSON []byte
}

// PrepareImportedWorkspace prepares a normal workspace inside a transaction's
// staging tree without touching the vault-wide workspace registry.
func PrepareImportedWorkspace(workspaceDir, name, templateID string) (PreparedImportedWorkspace, error) {
	if err := validateEntityName(name); err != nil {
		return PreparedImportedWorkspace{}, err
	}
	if templateID == "" {
		templateID = "import"
	}

	workspaceID := uuid.NewString()
	if err := WriteWorkspaceMarker(workspaceDir, workspaceID); err != nil {
		return PreparedImportedWorkspace{}, err
	}
	registryJSON, err := marshalWorkspaceMetadataV2(workspaceID, name, templateID, nil)
	if err != nil {
		return PreparedImportedWorkspace{}, err
	}
	return PreparedImportedWorkspace{ID: workspaceID, RegistryJSON: registryJSON}, nil
}
