package importservice

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDirectoryIndexesAndReadsBoundedText(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "pages"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "pages", "start.txt"), append([]byte{0xef, 0xbb, 0xbf}, []byte("hello")...), 0o600); err != nil {
		t.Fatal(err)
	}
	service := New(t.TempDir(), Options{})
	session, err := service.OpenDirectory("verstak.import", root)
	if err != nil {
		t.Fatal(err)
	}
	if session.Kind != "directory" || session.EntryCount != 2 {
		t.Fatalf("session=%+v", session)
	}
	page, err := service.ListEntries("verstak.import", session.SourceHandle, "")
	if err != nil {
		t.Fatal(err)
	}
	var fileID string
	for _, entry := range page.Entries {
		if entry.Path == "pages/start.txt" {
			fileID = entry.ID
		}
	}
	text, err := service.ReadText("verstak.import", session.SourceHandle, fileID)
	if err != nil {
		t.Fatal(err)
	}
	if text != "hello" {
		t.Fatalf("text=%q", text)
	}
}

func TestDirectoryDoesNotFollowSymlinks(t *testing.T) {
	root := t.TempDir()
	outside := filepath.Join(t.TempDir(), "secret.txt")
	if err := os.WriteFile(outside, []byte("secret"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(root, "linked.txt")); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	_, err := New(t.TempDir(), Options{}).OpenDirectory("verstak.import", root)
	if err == nil || !strings.Contains(err.Error(), "unsupported-source-entry") {
		t.Fatalf("expected unsupported-source-entry, got %v", err)
	}
}

func TestDirectoryReadRejectsSourceMutation(t *testing.T) {
	root := t.TempDir()
	filePath := filepath.Join(root, "page.md")
	if err := os.WriteFile(filePath, []byte("first"), 0o600); err != nil {
		t.Fatal(err)
	}
	service := New(t.TempDir(), Options{})
	session, err := service.OpenDirectory("verstak.import", root)
	if err != nil {
		t.Fatal(err)
	}
	page, err := service.ListEntries("verstak.import", session.SourceHandle, "")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filePath, []byte("changed-content"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err = service.ReadText("verstak.import", session.SourceHandle, page.Entries[0].ID)
	if err == nil || !strings.Contains(err.Error(), "source-changed") {
		t.Fatalf("expected source-changed, got %v", err)
	}
}

func TestDirectoryReadsUnicodeNameThroughNormalizedEntry(t *testing.T) {
	root := t.TempDir()
	decomposedName := "cafe\u0301.md"
	if err := os.WriteFile(filepath.Join(root, decomposedName), []byte("unicode"), 0o600); err != nil {
		t.Fatal(err)
	}
	service := New(t.TempDir(), Options{})
	session, err := service.OpenDirectory("verstak.import", root)
	if err != nil {
		t.Fatal(err)
	}
	page, err := service.ListEntries("verstak.import", session.SourceHandle, "")
	if err != nil {
		t.Fatal(err)
	}
	if page.Entries[0].Path != "café.md" {
		t.Fatalf("normalized path=%q", page.Entries[0].Path)
	}
	text, err := service.ReadText("verstak.import", session.SourceHandle, page.Entries[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	if text != "unicode" {
		t.Fatalf("text=%q", text)
	}
}
