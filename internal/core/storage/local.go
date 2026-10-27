package storage

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	goruntime "runtime"
)

// Local data is scoped to this vault path and plugin, outside the vault. It is
// never a sync record, export payload or import destination, nor a secret store.
func localDataPath(vaultPath, pluginID, name string) (string, error) {
	if err := validatePluginID(pluginID); err != nil {
		return "", err
	}
	if err := validateStorageName("local data", name); err != nil {
		return "", err
	}
	dir, err := os.UserConfigDir()
	if goruntime.GOOS == "windows" {
		dir, err = os.UserCacheDir()
	}
	if err != nil {
		return "", errors.New("local_data_unavailable")
	}
	owner := sha256.Sum256([]byte(vaultPath + "\x00" + pluginID))
	return filepath.Join(dir, "verstak", "plugin-local", hex.EncodeToString(owner[:]), name+".json"), nil
}

func (s *Storage) ReadPluginLocalJSON(pluginID, name string) (map[string]interface{}, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := map[string]interface{}{}
	err := s.withOpenVault(func(vaultPath string) error {
		path, err := localDataPath(vaultPath, pluginID, name)
		if err != nil {
			return err
		}
		data, err := os.ReadFile(path)
		if os.IsNotExist(err) {
			return nil
		}
		if err != nil || len(data) > 4*1024*1024 {
			return errors.New("local_data_unavailable")
		}
		if json.Unmarshal(data, &result) != nil || result == nil {
			return errors.New("local_data_invalid")
		}
		return nil
	})
	return result, err
}

func (s *Storage) WritePluginLocalJSON(pluginID, name string, value map[string]interface{}) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.withOpenVault(func(vaultPath string) error {
		path, err := localDataPath(vaultPath, pluginID, name)
		if err != nil {
			return err
		}
		data, err := json.Marshal(value)
		if err != nil || len(data) > 4*1024*1024 {
			return errors.New("invalid_local_data")
		}
		if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
			return errors.New("local_data_unavailable")
		}
		if err := atomicWrite(path, data); err != nil {
			return errors.New("local_data_unavailable")
		}
		return nil
	})
}
