package workspacetree

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
)

const OrderVersion = 1

type OrderState struct {
	Version  int                 `json:"version"`
	Children map[string][]string `json:"children"`
}

func OrderMetadataPath(vaultDir string) string {
	return filepath.Join(vaultDir, ".verstak", "workspace-tree", "order.json")
}

func ReadOrderState(vaultDir string) (OrderState, error) {
	data, err := os.ReadFile(OrderMetadataPath(vaultDir))
	if err != nil {
		if os.IsNotExist(err) {
			return emptyOrderState(), nil
		}
		return OrderState{}, fmt.Errorf("read workspace tree order: %w", err)
	}
	return ParseOrderState(data)
}

func ParseOrderState(data []byte) (OrderState, error) {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var state OrderState
	if err := decoder.Decode(&state); err != nil {
		return OrderState{}, fmt.Errorf("parse workspace tree order: %w", err)
	}
	if decoder.More() {
		return OrderState{}, fmt.Errorf("parse workspace tree order: trailing JSON value")
	}
	return normalizeOrderState(state)
}

func MarshalOrderState(state OrderState) ([]byte, error) {
	normalized, err := normalizeOrderState(state)
	if err != nil {
		return nil, err
	}
	data, err := json.MarshalIndent(normalized, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal workspace tree order: %w", err)
	}
	return append(data, '\n'), nil
}

func WriteOrderState(vaultDir string, state OrderState) error {
	data, err := MarshalOrderState(state)
	if err != nil {
		return err
	}
	path := OrderMetadataPath(vaultDir)
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("create workspace tree metadata directory: %w", err)
	}
	file, err := os.CreateTemp(dir, ".order-*.tmp")
	if err != nil {
		return fmt.Errorf("create workspace tree order temporary file: %w", err)
	}
	tempPath := file.Name()
	defer os.Remove(tempPath)
	if err := file.Chmod(0o600); err != nil {
		_ = file.Close()
		return fmt.Errorf("set workspace tree order permissions: %w", err)
	}
	if _, err := file.Write(data); err != nil {
		_ = file.Close()
		return fmt.Errorf("write workspace tree order: %w", err)
	}
	if err := file.Sync(); err != nil {
		_ = file.Close()
		return fmt.Errorf("sync workspace tree order: %w", err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("close workspace tree order: %w", err)
	}
	if err := os.Rename(tempPath, path); err != nil {
		return fmt.Errorf("replace workspace tree order: %w", err)
	}
	return nil
}

// ApplyOrderState replaces synchronized vault order metadata and immediately
// rebuilds the public tree from the backend's current filesystem topology.
func (s *Service) ApplyOrderState(state OrderState) error {
	if err := WriteOrderState(s.vaultDir, state); err != nil {
		return err
	}
	return s.fullReconcile()
}

func emptyOrderState() OrderState {
	return OrderState{
		Version:  OrderVersion,
		Children: make(map[string][]string),
	}
}

func normalizeOrderState(state OrderState) (OrderState, error) {
	if state.Version != OrderVersion {
		return OrderState{}, fmt.Errorf("unsupported workspace tree order version: %d", state.Version)
	}
	if state.Children == nil {
		state.Children = make(map[string][]string)
	}
	seen := make(map[string]bool)
	for parent, children := range state.Children {
		if parent != "root" {
			if _, err := uuid.Parse(parent); err != nil {
				return OrderState{}, fmt.Errorf("invalid workspace tree order parent: %s", parent)
			}
		}
		for _, key := range children {
			if !validOrderNodeKey(key) {
				return OrderState{}, fmt.Errorf("invalid workspace tree node key: %s", key)
			}
			if seen[key] {
				return OrderState{}, fmt.Errorf("duplicate workspace tree node key: %s", key)
			}
			seen[key] = true
		}
		state.Children[parent] = append([]string(nil), children...)
	}
	return state, nil
}

func validOrderNodeKey(key string) bool {
	prefix, id, ok := strings.Cut(key, ":")
	if !ok || (prefix != "folder" && prefix != "workspace") {
		return false
	}
	_, err := uuid.Parse(id)
	return err == nil
}
