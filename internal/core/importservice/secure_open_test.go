package importservice

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSecureOpenRejectsSymlinkParent(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	if err := os.WriteFile(filepath.Join(outside, "page.md"), []byte("outside"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(root, "linked")); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	file, err := secureOpenDirectoryFile(root, "linked/page.md")
	if file != nil {
		file.Close()
	}
	if err == nil {
		t.Fatal("secure opener followed a symlink parent")
	}
}

func TestSecureOpenReadsRegularDescendant(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "pages"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "pages", "start.md"), []byte("start"), 0o600); err != nil {
		t.Fatal(err)
	}
	file, err := secureOpenDirectoryFile(root, "pages/start.md")
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	data := make([]byte, 5)
	if _, err := file.Read(data); err != nil {
		t.Fatal(err)
	}
	if string(data) != "start" {
		t.Fatalf("data=%q", data)
	}
}
