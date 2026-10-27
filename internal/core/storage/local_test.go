package storage

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLocalDataStaysOutsideVaultAndIsScoped(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", home)
	t.Setenv("APPDATA", home)
	t.Setenv("LOCALAPPDATA", home)
	s, _ := newTestStorage(t)
	if err := s.WritePluginLocalJSON("actions", "trust", map[string]interface{}{"one": "hash"}); err != nil {
		t.Fatal(err)
	}
	for _, p := range []string{"actions", "other"} {
		got, err := s.ReadPluginLocalJSON(p, "trust")
		if err != nil {
			t.Fatal(err)
		}
		if (got["one"] == "hash") != (p == "actions") {
			t.Fatal("namespace leak", p, got)
		}
	}
	other, _ := newTestStorage(t)
	got, err := other.ReadPluginLocalJSON("actions", "trust")
	if err != nil || len(got) != 0 {
		t.Fatal("trust crossed vaults", got, err)
	}
	localPath, err := localDataPath(s.vault.GetVaultPath(), "actions", "trust")
	if err != nil {
		t.Fatal(err)
	}
	relative, err := filepath.Rel(s.vault.GetVaultPath(), localPath)
	if err != nil || relative == "." || (len(relative) < 2 || relative[:2] != "..") {
		t.Fatal("local data inside vault", localPath)
	}
	// A second device with the same vault path still has its own configuration.
	otherDevice := t.TempDir()
	t.Setenv("HOME", otherDevice)
	t.Setenv("XDG_CONFIG_HOME", otherDevice)
	t.Setenv("APPDATA", otherDevice)
	t.Setenv("LOCALAPPDATA", otherDevice)
	got, err = s.ReadPluginLocalJSON("actions", "trust")
	if err != nil || len(got) != 0 {
		t.Fatal("trust crossed devices", got, err)
	}
}

func TestLocalDataRejectsInvalidPathsAndCorruption(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", home)
	t.Setenv("APPDATA", home)
	t.Setenv("LOCALAPPDATA", home)
	s, _ := newTestStorage(t)
	for _, name := range []string{"", "..", "../outside", "/absolute"} {
		if err := s.WritePluginLocalJSON("actions", name, map[string]interface{}{}); err == nil {
			t.Fatal("accepted", name)
		}
	}
	path, err := localDataPath(s.vault.GetVaultPath(), "actions", "trust")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("broken"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := s.ReadPluginLocalJSON("actions", "trust"); err == nil {
		t.Fatal("corruption accepted")
	}
}
