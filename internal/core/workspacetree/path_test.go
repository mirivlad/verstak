package workspacetree

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestResolveInsideVaultRejectsTraversal(t *testing.T) {
	vault := t.TempDir()
	cases := []string{"../escape", "..", "a/../../b", "/abs/path", "C:\\windows"}
	for _, c := range cases {
		if _, err := ResolveInsideVault(vault, c); err == nil {
			t.Errorf("should reject: %s", c)
		}
	}
}

func TestResolveInsideVaultRejectsNullByte(t *testing.T) {
	vault := t.TempDir()
	if _, err := ResolveInsideVault(vault, "a\x00b"); err == nil {
		t.Fatal("should reject null byte")
	}
}

func TestResolveInsideVaultAcceptsValidPath(t *testing.T) {
	vault := t.TempDir()
	mustMkdirAll(t, filepath.Join(vault, "a", "b"))
	target, err := ResolveInsideVault(vault, "a/b")
	if err != nil {
		t.Fatal(err)
	}
	if target != filepath.Join(vault, "a", "b") {
		t.Fatalf("target = %q", target)
	}
}

func TestResolveInsideVaultRejectsSymlinkParent(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink")
	}
	vault := t.TempDir()
	outside := filepath.Join(t.TempDir(), "outside")
	mustMkdirAll(t, outside)
	symlinkPath := filepath.Join(vault, "link")
	if err := os.Symlink(outside, symlinkPath); err != nil {
		t.Fatal(err)
	}
	mustMkdirAll(t, filepath.Join(outside, "child"))
	if _, err := ResolveInsideVault(vault, "link/child"); err == nil {
		t.Fatal("should reject symlink parent")
	}
}

func TestResolveInsideVaultEmptyPath(t *testing.T) {
	vault := t.TempDir()
	target, err := ResolveInsideVault(vault, "")
	if err != nil {
		t.Fatal(err)
	}
	if target != vault {
		t.Fatalf("target = %q", target)
	}
}

func TestResolveInsideVaultWindowsDrivePath(t *testing.T) {
	vault := t.TempDir()
	if _, err := ResolveInsideVault(vault, "C:\\\\windows"); err == nil {
		t.Fatal("should reject Windows drive path")
	}
}
