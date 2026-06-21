package notes

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/verstak/verstak-desktop/internal/core/files"
	"github.com/verstak/verstak-desktop/internal/core/vault"
)

// testHarness creates a temporary vault + notes service for testing.
type testHarness struct {
	t       *testing.T
	vault   *vault.Vault
	files   *files.Service
	notes   *Service
	tmpDir  string
	vaultPath string
}

func newTestHarness(t *testing.T) *testHarness {
	t.Helper()
	tmpDir, err := os.MkdirTemp("", "verstak-notes-test-*")
	if err != nil {
		t.Fatalf("mkdir temp: %v", err)
	}
	t.Cleanup(func() { os.RemoveAll(tmpDir) })

	v := vault.NewVault(nil)
	if err := v.CreateVault(tmpDir); err != nil {
		t.Fatalf("create vault: %v", err)
	}

	vaultPath := v.GetVaultPath()

	f := files.NewService(v)
	n := NewService(f)

	return &testHarness{
		t:         t,
		vault:     v,
		files:     f,
		notes:     n,
		tmpDir:    tmpDir,
		vaultPath: vaultPath,
	}
}

func TestLayoutConstants(t *testing.T) {
	if CanonicalFolder != "Notes" {
		t.Fatalf("CanonicalFolder = %q, want Notes", CanonicalFolder)
	}
	if CanonicalOverview != "Overview.md" {
		t.Fatalf("CanonicalOverview = %q, want Overview.md", CanonicalOverview)
	}
	if NoteExtension != ".md" {
		t.Fatalf("NoteExtension = %q, want .md", NoteExtension)
	}
}

func TestNotesPath(t *testing.T) {
	tests := []struct {
		parent string
		want   string
	}{
		{"", "Notes"},
		{"Workspace", "Workspace/Notes"},
		{"Workspace/Project", "Workspace/Project/Notes"},
	}
	for _, tt := range tests {
		got := NotesPath(tt.parent)
		if got != tt.want {
			t.Errorf("NotesPath(%q) = %q, want %q", tt.parent, got, tt.want)
		}
	}
}

func TestOverviewPath(t *testing.T) {
	tests := []struct {
		parent string
		want   string
	}{
		{"", "Notes/Overview.md"},
		{"Workspace", "Workspace/Notes/Overview.md"},
	}
	for _, tt := range tests {
		got := OverviewPath(tt.parent)
		if got != tt.want {
			t.Errorf("OverviewPath(%q) = %q, want %q", tt.parent, got, tt.want)
		}
	}
}

func TestIsInsideNotes(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{"Notes/Overview.md", true},
		{"Workspace/Notes/MyNote.md", true},
		{"Workspace/Notes/Sub/File.md", true},
		{"Workspace/Files/readme.txt", false},
		{"", false},
		{"Notes", true}, // just the folder itself
	}
	for _, tt := range tests {
		got := IsInsideNotes(tt.path)
		if got != tt.want {
			t.Errorf("IsInsideNotes(%q) = %v, want %v", tt.path, got, tt.want)
		}
	}
}

func TestIsOverview(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{"Notes/Overview.md", true},
		{"Workspace/Notes/Overview.md", true},
		{"Workspace/Notes/MyNote.md", false},
		{"readme.md", false},
		{"", false},
	}
	for _, tt := range tests {
		got := IsOverview(tt.path)
		if got != tt.want {
			t.Errorf("IsOverview(%q) = %v, want %v", tt.path, got, tt.want)
		}
	}
}

func TestParentFromNotePath(t *testing.T) {
	tests := []struct {
		path string
		want string
	}{
		{"Workspace/Notes/MyNote.md", "Workspace"},
		{"Notes/MyNote.md", ""},
		{"A/B/Notes/MyNote.md", "A/B"},
	}
	for _, tt := range tests {
		got := ParentFromNotePath(tt.path)
		if got != tt.want {
			t.Errorf("ParentFromNotePath(%q) = %q, want %q", tt.path, got, tt.want)
		}
	}
}

func TestNormalizeTitleToFilename(t *testing.T) {
	tests := []struct {
		title string
		want  string
	}{
		{"My Note", "My_Note.md"},
		{"Hello World", "Hello_World.md"},
		{"  Trimmed  ", "Trimmed.md"},
		{"en–dash—em", "en_dash_em.md"}, // en-dash and em-dash → hyphen → underscore (collapsed)
		{"special:chars<>", "specialchars.md"}, // :<> are illegal, removed entirely
		{"dots.and.dashes", "dots_and_dashes.md"}, // dots collapsed to underscore
		{"UPPERCASE", "UPPERCASE.md"},
		{"русский язык", "русский_язык.md"},                     // Cyrillic preserved
		{"  leading/trailing  ", "leadingtrailing.md"}, // / is illegal, removed
		{"notes release 2.0", "notes_release_2_0.md"}, // dots collapsed
		{"already.md", "already.md"},                   // already has .md extension
		{"ALREADY.MD", "ALREADY.md"},                   // normalized to lowercase .md
		{"emoji_😊_test", "emoji_test.md"},              // emoji dropped (non-printable non-letter)
	}
	for _, tt := range tests {
		got, err := NormalizeTitleToFilename(tt.title)
		if err != nil {
			t.Errorf("NormalizeTitleToFilename(%q) unexpected error: %v", tt.title, err)
			continue
		}
		if got != tt.want {
			t.Errorf("NormalizeTitleToFilename(%q) = %q, want %q", tt.title, got, tt.want)
		}
	}
}

func TestNormalizeTitleToFilenameEmpty(t *testing.T) {
	_, err := NormalizeTitleToFilename("")
	if err == nil {
		t.Fatal("expected error for empty title")
	}
	_, err = NormalizeTitleToFilename("___")
	if err == nil {
		t.Fatal("expected error for title that normalizes to empty")
	}
}

func TestTitleFromFilename(t *testing.T) {
	tests := []struct {
		filename string
		want     string
	}{
		{"My_Note.md", "My Note"},
		{"Hello.md", "Hello"},
		{"UPPERCASE.MD", "UPPERCASE"},
	}
	for _, tt := range tests {
		got := TitleFromFilename(tt.filename)
		if got != tt.want {
			t.Errorf("TitleFromFilename(%q) = %q, want %q", tt.filename, got, tt.want)
		}
	}
}

func TestEnsureOverviewCreatesFile(t *testing.T) {
	h := newTestHarness(t)

	overviewPath, err := h.notes.EnsureOverview("Workspace")
	if err != nil {
		t.Fatalf("EnsureOverview: %v", err)
	}

	want := "Workspace/Notes/Overview.md"
	if overviewPath != want {
		t.Fatalf("overview path = %q, want %q", overviewPath, want)
	}

	// Verify file exists on disk
	fullPath := filepath.Join(h.vaultPath, filepath.FromSlash(overviewPath))
	if _, err := os.Stat(fullPath); err != nil {
		t.Fatalf("overview file not found: %v", err)
	}

	// Verify content — the workspace template creates "# Overview\n"
	content, err := h.notes.ReadNote(overviewPath)
	if err != nil {
		t.Fatalf("ReadNote: %v", err)
	}
	if content == "" {
		t.Fatal("overview content is empty")
	}
}

func TestEnsureOverviewIdempotent(t *testing.T) {
	h := newTestHarness(t)

	p1, err := h.notes.EnsureOverview("Workspace")
	if err != nil {
		t.Fatalf("first EnsureOverview: %v", err)
	}

	p2, err := h.notes.EnsureOverview("Workspace")
	if err != nil {
		t.Fatalf("second EnsureOverview: %v", err)
	}

	if p1 != p2 {
		t.Fatalf("paths differ: %q vs %q", p1, p2)
	}
}

func TestCreateNoteCreatesFile(t *testing.T) {
	h := newTestHarness(t)

	notePath, err := h.notes.CreateNote("Workspace", "My First Note", "")
	if err != nil {
		t.Fatalf("CreateNote: %v", err)
	}

	want := "Workspace/Notes/My_First_Note.md"
	if notePath != want {
		t.Fatalf("note path = %q, want %q", notePath, want)
	}

	// Verify file exists
	fullPath := filepath.Join(h.vaultPath, filepath.FromSlash(notePath))
	if _, err := os.Stat(fullPath); err != nil {
		t.Fatalf("note file not found: %v", err)
	}

	// Verify content has title
	content, err := h.notes.ReadNote(notePath)
	if err != nil {
		t.Fatalf("ReadNote: %v", err)
	}
	if !strings.Contains(content, "My First Note") {
		t.Fatalf("content should contain title, got: %q", content)
	}
}

func TestCreateNoteRejectsConflict(t *testing.T) {
	h := newTestHarness(t)

	_, err := h.notes.CreateNote("Workspace", "My Note", "")
	if err != nil {
		t.Fatalf("first CreateNote: %v", err)
	}

	_, err = h.notes.CreateNote("Workspace", "My Note", "")
	if err == nil {
		t.Fatal("expected conflict error for duplicate note")
	}
	var ce *ConflictError
	if !asConflictError(err, &ce) {
		t.Fatalf("expected ConflictError, got: %T %v", err, err)
	}
}

func TestCreateNoteRejectsEmptyTitle(t *testing.T) {
	h := newTestHarness(t)

	_, err := h.notes.CreateNote("Workspace", "", "")
	if err == nil {
		t.Fatal("expected error for empty title")
	}
}

func TestRenameNoteRenamesFile(t *testing.T) {
	h := newTestHarness(t)

	oldPath, err := h.notes.CreateNote("Workspace", "Old Title", "")
	if err != nil {
		t.Fatalf("CreateNote: %v", err)
	}

	newPath, err := h.notes.RenameNote(oldPath, "New Title")
	if err != nil {
		t.Fatalf("RenameNote: %v", err)
	}

	want := "Workspace/Notes/New_Title.md"
	if newPath != want {
		t.Fatalf("new path = %q, want %q", newPath, want)
	}

	// Old file should not exist
	oldFull := filepath.Join(h.vaultPath, filepath.FromSlash(oldPath))
	if _, err := os.Stat(oldFull); !os.IsNotExist(err) {
		t.Fatalf("old file should not exist, stat error: %v", err)
	}

	// New file should exist
	newFull := filepath.Join(h.vaultPath, filepath.FromSlash(newPath))
	if _, err := os.Stat(newFull); err != nil {
		t.Fatalf("new file should exist: %v", err)
	}
}

func TestRenameNoteRejectsConflict(t *testing.T) {
	h := newTestHarness(t)

	_, err := h.notes.CreateNote("Workspace", "Note A", "")
	if err != nil {
		t.Fatalf("create Note A: %v", err)
	}
	_, err = h.notes.CreateNote("Workspace", "Note B", "")
	if err != nil {
		t.Fatalf("create Note B: %v", err)
	}

	// Rename Note A to "Note B" — should conflict
	_, err = h.notes.RenameNote("Workspace/Notes/Note_A.md", "Note B")
	if err == nil {
		t.Fatal("expected conflict error")
	}
	var ce *ConflictError
	if !asConflictError(err, &ce) {
		t.Fatalf("expected ConflictError, got: %T %v", err, err)
	}
}

func TestListNotes(t *testing.T) {
	h := newTestHarness(t)

	// Ensure overview
	h.notes.EnsureOverview("Workspace")

	// Create some notes
	h.notes.CreateNote("Workspace", "Alpha", "")
	h.notes.CreateNote("Workspace", "Beta", "")
	h.notes.CreateNote("Workspace", "Gamma", "")

	notes, err := h.notes.ListNotes("Workspace")
	if err != nil {
		t.Fatalf("ListNotes: %v", err)
	}

	if len(notes) != 4 {
		t.Fatalf("expected 4 notes, got %d", len(notes))
	}

	// Overview should be first
	if notes[0].IsOverview != true {
		t.Fatal("first note should be overview")
	}

	// The rest should be sorted alphabetically
	if notes[1].Title != "Alpha" {
		t.Fatalf("second note title = %q, want Alpha", notes[1].Title)
	}
}

func TestSaveAndReadNote(t *testing.T) {
	h := newTestHarness(t)

	path, err := h.notes.CreateNote("Workspace", "Test Note", "original content")
	if err != nil {
		t.Fatalf("CreateNote: %v", err)
	}

	content, err := h.notes.ReadNote(path)
	if err != nil {
		t.Fatalf("ReadNote: %v", err)
	}
	if content != "original content" {
		t.Fatalf("content = %q, want %q", content, "original content")
	}

	// Update content
	err = h.notes.SaveNote(path, "updated content")
	if err != nil {
		t.Fatalf("SaveNote: %v", err)
	}

	content, err = h.notes.ReadNote(path)
	if err != nil {
		t.Fatalf("ReadNote after save: %v", err)
	}
	if content != "updated content" {
		t.Fatalf("content after save = %q, want %q", content, "updated content")
	}
}

func TestSearchNotes(t *testing.T) {
	h := newTestHarness(t)

	h.notes.CreateNote("Workspace", "Meeting Notes", "")
	h.notes.CreateNote("Workspace", "Project Plan", "")
	h.notes.CreateNote("Workspace", "Ideas", "")

	results, err := h.notes.SearchNotes(h.vaultPath, "meeting")
	if err != nil {
		t.Fatalf("SearchNotes: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("expected 1 result for 'meeting', got %d", len(results))
	}
	if results[0].Title != "Meeting Notes" {
		t.Fatalf("title = %q, want 'Meeting Notes'", results[0].Title)
	}
}

func TestSearchNotesBySwappedLayout(t *testing.T) {
	h := newTestHarness(t)

	h.notes.CreateNote("Workspace", "Привет", "")

	// Search with English QWERTY equivalent of "привет" -> "ghbdtn"
	results, err := h.notes.SearchNotes(h.vaultPath, "ghbdtn")
	if err != nil {
		t.Fatalf("SearchNotes: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("expected 1 result for 'ghbdtn' (swapped layout), got %d", len(results))
	}
	if results[0].Title != "Привет" {
		t.Fatalf("title = %q, want 'Привет'", results[0].Title)
	}
}

func TestUnsafePathsRejected(t *testing.T) {
	h := newTestHarness(t)

	// Try to create a note with path traversal
	_, err := h.notes.CreateNote("../outside", "Note", "")
	if err == nil {
		t.Fatal("expected error for path traversal parent")
	}
}

// ─── helpers ──────────────────────────────────────────────────

func asConflictError(err error, target **ConflictError) bool {
	if err == nil {
		return false
	}
	ce, ok := err.(*ConflictError)
	if !ok {
		return false
	}
	if target != nil {
		*target = ce
	}
	return true
}
