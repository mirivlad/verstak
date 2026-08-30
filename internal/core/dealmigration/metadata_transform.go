package dealmigration

import (
	"context"
	"errors"
	"os"
	"path/filepath"

	"github.com/verstak/verstak-desktop/internal/core/workspacetree"
)

// NewDealMetadataTransform materializes one canonical UUID-keyed metadata
// record for every current Deal marker before the remaining transforms run.
// Path-keyed files remain untouched in the migration backup and never serve
// runtime reads after this step.
func NewDealMetadataTransform() Transform {
	return FuncTransform("deal-metadata-to-uuid-registry", migrateDealMetadata, verifyDealMetadata)
}

func migrateDealMetadata(ctx context.Context, vault string) error {
	rootToWorkspace, err := currentWorkspaceRootIDs(vault)
	if err != nil {
		return err
	}
	service := workspacetree.NewService(vault, nil)
	for rootPath, workspaceID := range rootToWorkspace {
		if err := ctx.Err(); err != nil {
			return err
		}
		_, _, err := service.MigrateLegacyDealMetadata(workspaceID, rootPath)
		if errors.Is(err, os.ErrNotExist) {
			err = service.WriteDealMetadata(workspacetree.DealMetadata{
				WorkspaceID:   workspaceID,
				WorkspaceName: filepath.Base(filepath.FromSlash(rootPath)),
			})
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func verifyDealMetadata(ctx context.Context, vault string) error {
	rootToWorkspace, err := currentWorkspaceRootIDs(vault)
	if err != nil {
		return err
	}
	service := workspacetree.NewService(vault, nil)
	for rootPath, workspaceID := range rootToWorkspace {
		if err := ctx.Err(); err != nil {
			return err
		}
		if _, err := service.ReadDealMetadata(workspaceID, rootPath); err != nil {
			return err
		}
	}
	return nil
}
