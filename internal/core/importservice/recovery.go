package importservice

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

type transactionStatus string

const (
	transactionStaged     transactionStatus = "staged"
	transactionPublishing transactionStatus = "publishing"
	transactionCommitted  transactionStatus = "committed"
)

type transactionJournal struct {
	Version           int               `json:"version"`
	TransactionID     string            `json:"transactionId"`
	Status            transactionStatus `json:"status"`
	PublishedRoot     string            `json:"publishedRoot"`
	CreatedImportRoot bool              `json:"createdImportRoot,omitempty"`
	RegistryPaths     []string          `json:"registryPaths,omitempty"`
}

func (service *Service) Recover() error {
	service.applyMu.Lock()
	defer service.applyMu.Unlock()
	return service.recoverLocked()
}

func (service *Service) recoverLocked() error {
	stagingRoot := filepath.Join(service.vaultDir, ".verstak", "import-staging")
	entries, err := os.ReadDir(stagingRoot)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	for _, entry := range entries {
		txnDir := filepath.Join(stagingRoot, entry.Name())
		if entry.Type()&os.ModeSymlink != 0 || !entry.IsDir() {
			if err := os.RemoveAll(txnDir); err != nil {
				return err
			}
			continue
		}
		data, err := os.ReadFile(filepath.Join(txnDir, "transaction.json"))
		if os.IsNotExist(err) {
			if err := os.RemoveAll(txnDir); err != nil {
				return err
			}
			continue
		}
		if err != nil {
			return err
		}
		var journal transactionJournal
		if err := json.Unmarshal(data, &journal); err != nil || journal.Version != 1 {
			return sourceError("invalid-import-journal", "staging journal is invalid")
		}
		if journal.Status == transactionPublishing {
			published, err := recoveryPublishedPath(service.vaultDir, journal)
			if err != nil {
				return err
			}
			if _, statErr := os.Lstat(published); statErr == nil {
				owned, ownershipErr := importPathOwnedByTransaction(published, journal.TransactionID)
				if ownershipErr != nil {
					return ownershipErr
				}
				if !owned {
					return sourceError("import-transaction-owner-mismatch", "published path is not owned by this transaction")
				}
				if err := os.RemoveAll(published); err != nil {
					return err
				}
			} else if !os.IsNotExist(statErr) {
				return statErr
			}
			for _, relative := range journal.RegistryPaths {
				registry, err := recoveryRegistryPath(service.vaultDir, relative)
				if err != nil {
					return err
				}
				if err := os.Remove(registry); err != nil && !os.IsNotExist(err) {
					return err
				}
			}
		}
		if err := os.RemoveAll(txnDir); err != nil {
			return err
		}
	}
	return nil
}

type importOwnershipMarker struct {
	TransactionID string `json:"transactionId"`
}

func writeImportOwnershipMarker(root, transactionID string) error {
	if transactionID == "" {
		return sourceError("invalid-import-journal", "transaction ID is required")
	}
	data, err := json.Marshal(importOwnershipMarker{TransactionID: transactionID})
	if err != nil {
		return err
	}
	markerDir := filepath.Join(root, ".verstak")
	if err := os.MkdirAll(markerDir, 0o755); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(markerDir, "import-transaction.json"), data, 0o600)
}

func importPathOwnedByTransaction(root, transactionID string) (bool, error) {
	if transactionID == "" {
		return false, nil
	}
	data, err := os.ReadFile(filepath.Join(root, ".verstak", "import-transaction.json"))
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	var marker importOwnershipMarker
	if err := json.Unmarshal(data, &marker); err != nil {
		return false, nil
	}
	return marker.TransactionID == transactionID, nil
}

func writeTransactionJournal(txnDir string, journal transactionJournal) error {
	data, err := json.Marshal(journal)
	if err != nil {
		return err
	}
	target := filepath.Join(txnDir, "transaction.json")
	temporary := target + ".tmp"
	file, err := os.OpenFile(temporary, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	if _, err := file.Write(data); err != nil {
		_ = file.Close()
		_ = os.Remove(temporary)
		return err
	}
	if err := file.Sync(); err != nil {
		_ = file.Close()
		_ = os.Remove(temporary)
		return err
	}
	if err := file.Close(); err != nil {
		_ = os.Remove(temporary)
		return err
	}
	if err := os.Rename(temporary, target); err != nil {
		_ = os.Remove(temporary)
		return err
	}
	return syncDirectory(txnDir)
}

func recoveryPublishedPath(vaultDir string, journal transactionJournal) (string, error) {
	normalized, err := normalizeJournalPath(journal.PublishedRoot)
	if err != nil {
		return "", err
	}
	if journal.CreatedImportRoot {
		if normalized != importRootName {
			return "", sourceError("invalid-import-journal", "created root does not match")
		}
	} else if !strings.HasPrefix(normalized, importRootName+"/") {
		return "", sourceError("invalid-import-journal", "published path is outside import root")
	}
	return filepath.Join(vaultDir, filepath.FromSlash(normalized)), nil
}

func recoveryRegistryPath(vaultDir, relative string) (string, error) {
	normalized, err := normalizeJournalPath(relative)
	if err != nil {
		return "", err
	}
	if !strings.HasPrefix(normalized, ".verstak/workspaces/uuid-") || pathDepth(normalized) != 3 || !strings.HasSuffix(normalized, ".json") {
		return "", sourceError("invalid-import-journal", "registry path is outside registry")
	}
	return filepath.Join(vaultDir, filepath.FromSlash(normalized)), nil
}

func normalizeJournalPath(value string) (string, error) {
	if value == "" || strings.ContainsRune(value, 0) || strings.Contains(value, `\`) || filepath.IsAbs(value) {
		return "", sourceError("invalid-import-journal", "unsafe journal path")
	}
	cleaned := filepath.ToSlash(filepath.Clean(filepath.FromSlash(value)))
	if cleaned != value || cleaned == "." || cleaned == ".." || strings.HasPrefix(cleaned, "../") {
		return "", sourceError("invalid-import-journal", "unsafe journal path")
	}
	return cleaned, nil
}

func pathDepth(value string) int {
	if value == "" {
		return 0
	}
	return len(strings.Split(value, "/"))
}
