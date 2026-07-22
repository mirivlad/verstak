package importservice

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestSessionOwnershipPaginationAndClose(t *testing.T) {
	root := t.TempDir()
	for i := 0; i < 3; i++ {
		name := filepath.Join(root, string(rune('a'+i))+".md")
		if err := os.WriteFile(name, []byte("text"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	service := New(t.TempDir(), Options{PageSize: 2})
	session, err := service.OpenDirectory("verstak.import", root)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.ListEntries("other.plugin", session.SourceHandle, ""); err == nil || !strings.Contains(err.Error(), "source-session-owner") {
		t.Fatalf("expected source-session-owner, got %v", err)
	}
	first, err := service.ListEntries("verstak.import", session.SourceHandle, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(first.Entries) != 2 || first.NextCursor != "2" {
		t.Fatalf("first=%+v", first)
	}
	second, err := service.ListEntries("verstak.import", session.SourceHandle, first.NextCursor)
	if err != nil {
		t.Fatal(err)
	}
	if len(second.Entries) != 1 || second.NextCursor != "" {
		t.Fatalf("second=%+v", second)
	}
	if err := service.Close("verstak.import", session.SourceHandle); err != nil {
		t.Fatal(err)
	}
	if err := service.Close("verstak.import", session.SourceHandle); err != nil {
		t.Fatal(err)
	}
	if _, err := service.ListEntries("verstak.import", session.SourceHandle, ""); err == nil || !strings.Contains(err.Error(), "source-session-not-found") {
		t.Fatalf("expected source-session-not-found, got %v", err)
	}
}

func TestReadTextRejectsBinaryAndConfiguredLimit(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "binary.bin"), []byte{'a', 0, 'b'}, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "large.txt"), []byte("12345"), 0o600); err != nil {
		t.Fatal(err)
	}
	service := New(t.TempDir(), Options{MaxTextBytes: 4})
	session, err := service.OpenDirectory("verstak.import", root)
	if err != nil {
		t.Fatal(err)
	}
	page, err := service.ListEntries("verstak.import", session.SourceHandle, "")
	if err != nil {
		t.Fatal(err)
	}
	ids := map[string]string{}
	for _, entry := range page.Entries {
		ids[entry.Path] = entry.ID
	}
	if _, err := service.ReadText("verstak.import", session.SourceHandle, ids["binary.bin"]); err == nil || !strings.Contains(err.Error(), "binary-entry") {
		t.Fatalf("expected binary-entry, got %v", err)
	}
	if _, err := service.ReadText("verstak.import", session.SourceHandle, ids["large.txt"]); err == nil || !strings.Contains(err.Error(), "text-entry-too-large") {
		t.Fatalf("expected text-entry-too-large, got %v", err)
	}
}

func TestCancelIsIdempotentAfterClose(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "page.md"), []byte("text"), 0o600); err != nil {
		t.Fatal(err)
	}
	service := New(t.TempDir(), Options{})
	session, err := service.OpenDirectory("verstak.import", root)
	if err != nil {
		t.Fatal(err)
	}
	if err := service.Cancel("verstak.import", session.SourceHandle); err != nil {
		t.Fatal(err)
	}
	if err := service.Close("verstak.import", session.SourceHandle); err != nil {
		t.Fatal(err)
	}
	if err := service.Cancel("verstak.import", session.SourceHandle); err != nil {
		t.Fatalf("second cancel: %v", err)
	}
}

func TestLifecycleCleanupKeepsCancellationIdempotent(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "page.md"), []byte("text"), 0o600); err != nil {
		t.Fatal(err)
	}
	service := New(t.TempDir(), Options{})
	first, err := service.OpenDirectory("verstak.import", root)
	if err != nil {
		t.Fatal(err)
	}
	second, err := service.OpenDirectory("other.plugin", root)
	if err != nil {
		t.Fatal(err)
	}
	service.ClosePlugin("verstak.import")
	if err := service.Cancel("verstak.import", first.SourceHandle); err != nil {
		t.Fatalf("cancel after ClosePlugin: %v", err)
	}
	service.CloseAll()
	if err := service.Cancel("other.plugin", second.SourceHandle); err != nil {
		t.Fatalf("cancel after CloseAll: %v", err)
	}
}

func TestSessionExpiresAfterConfiguredIdleTime(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "page.md"), []byte("text"), 0o600); err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 7, 23, 12, 0, 0, 0, time.UTC)
	service := New(t.TempDir(), Options{SessionTTL: time.Minute, Now: func() time.Time { return now }})
	session, err := service.OpenDirectory("verstak.import", root)
	if err != nil {
		t.Fatal(err)
	}
	now = now.Add(2 * time.Minute)
	if _, err := service.ListEntries("verstak.import", session.SourceHandle, ""); err == nil || !strings.Contains(err.Error(), "source-session-not-found") {
		t.Fatalf("expected expired session, got %v", err)
	}
}
