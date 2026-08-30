// Package gitservice owns device-local Git checkout registration.
package gitservice

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/verstak/verstak-desktop/internal/core/workspacetree"
)

const checkoutRegistryVersion = 1

// CheckoutRegistration binds one device-local checkout name to a Deal UUID.
// It contains no remote URL or credential material.
type CheckoutRegistration struct {
	WorkspaceID   string
	WorkspaceRoot string
	RepositoryID  string
	CheckoutName  string
}

type checkoutRegistry struct {
	Version      int               `json:"version"`
	WorkspaceID  string            `json:"workspaceId"`
	Repositories map[string]string `json:"repositories"`
}

// RegisterCheckout records a Core-managed checkout under
// <Deal>/Repositories/<checkoutName>. The registry is local metadata under
// the Deal's .verstak directory and is therefore never ordinary Sync content.
func RegisterCheckout(vaultRoot string, registration CheckoutRegistration) (string, error) {
	workspaceRoot, err := cleanRelativePath(registration.WorkspaceRoot)
	if err != nil {
		return "", fmt.Errorf("workspace root: %w", err)
	}
	if _, err := uuid.Parse(registration.WorkspaceID); err != nil {
		return "", fmt.Errorf("workspace ID: %w", err)
	}
	if !safeSegment(registration.RepositoryID) {
		return "", fmt.Errorf("invalid repository ID")
	}
	if !safeSegment(registration.CheckoutName) {
		return "", fmt.Errorf("invalid checkout name")
	}

	metadataDir := filepath.Join(vaultRoot, filepath.FromSlash(workspaceRoot), ".verstak")
	marker, err := readWorkspaceMarker(filepath.Join(metadataDir, "workspace.json"))
	if err != nil {
		return "", err
	}
	if marker.WorkspaceID != registration.WorkspaceID {
		return "", fmt.Errorf("workspace marker does not match Deal UUID")
	}

	registryPath := filepath.Join(metadataDir, "git-checkouts.json")
	registry, err := readRegistry(registryPath)
	if err != nil {
		return "", err
	}
	if registry.WorkspaceID != "" && registry.WorkspaceID != registration.WorkspaceID {
		return "", fmt.Errorf("Git checkout registry belongs to another Deal")
	}
	registry.Version = checkoutRegistryVersion
	registry.WorkspaceID = registration.WorkspaceID
	if registry.Repositories == nil {
		registry.Repositories = make(map[string]string)
	}

	// CheckoutName is user-facing filesystem state while RepositoryID is the
	// stable descriptor identity. Older Git plugin builds used repo-<uuid> as
	// both, which leaked implementation detail into Files. If the descriptor
	// starts asking for a better checkout name, move the existing managed tree
	// before updating the local registry. No syncable descriptor identity changes.
	if previous := strings.TrimSpace(registry.Repositories[registration.RepositoryID]); previous != "" && previous != registration.CheckoutName {
		repositoriesRoot := filepath.Join(vaultRoot, filepath.FromSlash(workspaceRoot), "Repositories")
		oldPath := filepath.Join(repositoriesRoot, previous)
		newPath := filepath.Join(repositoriesRoot, registration.CheckoutName)
		if _, err := os.Stat(oldPath); err == nil {
			if _, targetErr := os.Stat(newPath); targetErr == nil {
				return "", fmt.Errorf("cannot rename managed checkout: target already exists")
			} else if !os.IsNotExist(targetErr) {
				return "", targetErr
			}
			if err := os.Rename(oldPath, newPath); err != nil {
				return "", fmt.Errorf("rename managed checkout: %w", err)
			}
		} else if !os.IsNotExist(err) {
			return "", err
		}
	}

	registry.Repositories[registration.RepositoryID] = registration.CheckoutName
	if err := writeRegistry(registryPath, registry); err != nil {
		return "", err
	}
	return workspaceRoot + "/Repositories/" + registration.CheckoutName, nil
}

// IsManagedCheckoutPath reports whether relativePath belongs to the managed
// Git checkout root of a Deal. A directory named Repositories is not special
// by itself: the parent must be a marked Deal whose local registry matches the
// same UUID and lists at least one checkout.
func IsManagedCheckoutPath(vaultRoot, relativePath string) bool {
	rel, err := cleanRelativePath(relativePath)
	if err != nil {
		return false
	}
	segments := strings.Split(rel, "/")
	for index, segment := range segments {
		if segment != "Repositories" || index == 0 {
			continue
		}
		workspaceRoot := strings.Join(segments[:index], "/")
		metadataDir := filepath.Join(vaultRoot, filepath.FromSlash(workspaceRoot), ".verstak")
		marker, err := readWorkspaceMarker(filepath.Join(metadataDir, "workspace.json"))
		if err != nil || marker.WorkspaceID == "" {
			continue
		}
		registry, err := readRegistry(filepath.Join(metadataDir, "git-checkouts.json"))
		if err != nil || registry.Version != checkoutRegistryVersion || registry.WorkspaceID != marker.WorkspaceID || len(registry.Repositories) == 0 {
			continue
		}
		return true
	}
	return false
}

func cleanRelativePath(raw string) (string, error) {
	value := strings.Trim(strings.ReplaceAll(strings.TrimSpace(raw), "\\", "/"), "/")
	if value == "" {
		return "", fmt.Errorf("path is empty")
	}
	cleaned := filepath.ToSlash(filepath.Clean(value))
	if cleaned == "." || cleaned == ".." || strings.HasPrefix(cleaned, "../") || filepath.IsAbs(filepath.FromSlash(cleaned)) {
		return "", fmt.Errorf("path is unsafe")
	}
	return cleaned, nil
}

func safeSegment(value string) bool {
	value = strings.TrimSpace(value)
	return value != "" && value != "." && value != ".." && !strings.ContainsAny(value, "/\\")
}

func readWorkspaceMarker(path string) (workspacetree.WorkspaceMarker, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return workspacetree.WorkspaceMarker{}, fmt.Errorf("read Deal marker: %w", err)
	}
	var marker workspacetree.WorkspaceMarker
	if err := json.Unmarshal(data, &marker); err != nil {
		return workspacetree.WorkspaceMarker{}, fmt.Errorf("decode Deal marker: %w", err)
	}
	if marker.WorkspaceID == "" {
		return workspacetree.WorkspaceMarker{}, fmt.Errorf("Deal marker has no UUID")
	}
	return marker, nil
}

func readRegistry(path string) (checkoutRegistry, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return checkoutRegistry{}, nil
	}
	if err != nil {
		return checkoutRegistry{}, fmt.Errorf("read Git checkout registry: %w", err)
	}
	var registry checkoutRegistry
	if err := json.Unmarshal(data, &registry); err != nil {
		return checkoutRegistry{}, fmt.Errorf("decode Git checkout registry: %w", err)
	}
	if registry.Version != checkoutRegistryVersion {
		return checkoutRegistry{}, fmt.Errorf("unsupported Git checkout registry version")
	}
	return registry, nil
}

func writeRegistry(path string, registry checkoutRegistry) error {
	data, err := json.Marshal(registry)
	if err != nil {
		return fmt.Errorf("encode Git checkout registry: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("create Git checkout registry: %w", err)
	}
	temporary, err := os.CreateTemp(filepath.Dir(path), ".git-checkouts-*.tmp")
	if err != nil {
		return fmt.Errorf("create Git checkout registry temporary file: %w", err)
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)
	if err := temporary.Chmod(0o600); err != nil {
		_ = temporary.Close()
		return err
	}
	if _, err := temporary.Write(data); err != nil {
		_ = temporary.Close()
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	if err := os.Rename(temporaryPath, path); err != nil {
		return err
	}
	return nil
}
