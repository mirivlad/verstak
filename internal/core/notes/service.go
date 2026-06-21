package notes

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/verstak/verstak-desktop/internal/core/files"
)

// Service provides note operations within a vault.
// It reuses the files.Service for actual file I/O to keep a single
// source of truth for vault file access.
type Service struct {
	files *files.Service
}

// NoteInfo describes a discovered note file.
type NoteInfo struct {
	Title      string `json:"title"`
	Filename   string `json:"filename"`
	Path       string `json:"path"`       // vault-relative path
	ParentPath string `json:"parentPath"` // parent of the Notes/ folder
	IsOverview bool   `json:"isOverview"`
}

// NewService creates a new notes service backed by the given file service.
func NewService(filesSvc *files.Service) *Service {
	return &Service{files: filesSvc}
}

// EnsureOverview creates Notes/Overview.md under the given parent path
// if it doesn't exist. Returns the vault-relative path of the overview file.
// parent is a vault-relative directory path (e.g. "Workspace" or "Clients/Acme").
func (s *Service) EnsureOverview(parent string) (string, error) {
	overviewRel := OverviewPath(parent)

	// Check if overview already exists
	_, err := s.files.GetVaultFileMetadata(overviewRel)
	if err == nil {
		return overviewRel, nil
	}

	// Ensure Notes folder exists
	notesRel := NotesPath(parent)
	if err := s.files.CreateVaultFolder(notesRel); err != nil {
		if !isConflictError(err) {
			return "", fmt.Errorf("create notes folder: %w", err)
		}
	}

	// Use parent basename as default title
	parentName := parent
	if idx := strings.LastIndex(parent, "/"); idx >= 0 {
		parentName = parent[idx+1:]
	}
	if parentName == "" {
		parentName = "Overview"
	}
	defaultContent := "# " + parentName + "\n"

	if err := s.files.WriteVaultTextFile(overviewRel, defaultContent, files.WriteOptions{
		CreateIfMissing: true,
		Overwrite:       false,
	}); err != nil {
		return "", fmt.Errorf("create overview: %w", err)
	}

	return overviewRel, nil
}

// CreateNote creates a new markdown note under the given parent's Notes/ folder.
// title is the human-readable note title. The filename is derived via
// NormalizeTitleToFilename.
// Returns the vault-relative path of the new note, or an error if:
//   - title is invalid
//   - the filename already exists (conflict)
//   - parent path is unsafe
func (s *Service) CreateNote(parent, title string, content string) (string, error) {
	if err := ValidateNoteTitle(title); err != nil {
		return "", err
	}

	filename, err := NormalizeTitleToFilename(title)
	if err != nil {
		return "", err
	}

	notesRel := NotesPath(parent)
	noteRel := notesRel + "/" + filename

	// Ensure Notes folder exists
	if err := s.files.CreateVaultFolder(notesRel); err != nil {
		if !isConflictError(err) {
			return "", fmt.Errorf("create notes folder: %w", err)
		}
	}

	// Check for conflict: file must not already exist
	if _, err := s.files.GetVaultFileMetadata(noteRel); err == nil {
		return "", &ConflictError{
			Path:     noteRel,
			Title:    title,
			Filename: filename,
		}
	}

	if content == "" {
		content = "# " + title + "\n"
	}

	if err := s.files.WriteVaultTextFile(noteRel, content, files.WriteOptions{
		CreateIfMissing: true,
		Overwrite:       false,
	}); err != nil {
		return "", fmt.Errorf("create note: %w", err)
	}

	return noteRel, nil
}

// RenameNote renames a note by changing its title. The filename is derived
// from the new title. The old note file is renamed to the new filename.
// If the new filename would conflict with an existing file, a ConflictError is returned.
//
// notePath is the current vault-relative path of the note.
func (s *Service) RenameNote(notePath, newTitle string) (string, error) {
	if err := ValidateNoteTitle(newTitle); err != nil {
		return "", err
	}

	// Check that the note exists
	oldMeta, err := s.files.GetVaultFileMetadata(notePath)
	if err != nil {
		return "", fmt.Errorf("note not found: %w", err)
	}
	if oldMeta.Type != files.FileTypeFile {
		return "", fmt.Errorf("not a file: %s", notePath)
	}

	newFilename, err := NormalizeTitleToFilename(newTitle)
	if err != nil {
		return "", err
	}

	oldDir := pathDir(notePath)
	oldName := filepath.Base(notePath)
	_ = oldName

	// Check if filename would actually change
	if strings.EqualFold(filepath.Base(notePath), newFilename) {
		// Same filename (case-insensitive). If exact case matches, no rename needed.
		if filepath.Base(notePath) == newFilename {
			return notePath, nil
		}
		// Only case differs — proceed (the OS rename will handle case change).
	}

	newPath := oldDir + "/" + newFilename

	// Prevent conflict: if the target path already exists and is not the source
	if newPath != notePath {
		if _, err := s.files.GetVaultFileMetadata(newPath); err == nil {
			return "", &ConflictError{
				Path:     newPath,
				Title:    newTitle,
				Filename: newFilename,
			}
		}
	}

	if err := s.files.MoveVaultPath(notePath, newPath, files.MoveOptions{
		Overwrite: false,
	}); err != nil {
		return "", fmt.Errorf("rename note: %w", err)
	}

	return newPath, nil
}

// ReadNote reads the content of a note file.
func (s *Service) ReadNote(notePath string) (string, error) {
	return s.files.ReadVaultTextFile(notePath)
}

// SaveNote writes content to a note file. Requires overwrite permission.
func (s *Service) SaveNote(notePath, content string) error {
	return s.files.WriteVaultTextFile(notePath, content, files.WriteOptions{
		CreateIfMissing: false,
		Overwrite:       true,
	})
}

// ListNotes returns all markdown notes in the given parent's Notes/ folder
// (non-recursive). Each NoteInfo has title derived from filename.
func (s *Service) ListNotes(parent string) ([]NoteInfo, error) {
	notesRel := NotesPath(parent)

	entries, err := s.files.ListVaultFiles(notesRel)
	if err != nil {
		if os.IsNotExist(err) || isNotFoundError(err) {
			return []NoteInfo{}, nil
		}
		return nil, err
	}

	var notes []NoteInfo
	for _, entry := range entries {
		if entry.Type != files.FileTypeFile {
			continue
		}
		ext := strings.ToLower(filepath.Ext(entry.Name))
		if ext != ".md" && ext != ".markdown" {
			continue
		}
		title := TitleFromFilename(entry.Name)
		notes = append(notes, NoteInfo{
			Title:      title,
			Filename:   entry.Name,
			Path:       entry.RelativePath,
			ParentPath: parent,
			IsOverview: strings.EqualFold(entry.Name, CanonicalOverview),
		})
	}

	sort.Slice(notes, func(i, j int) bool {
		// Overview always first
		if notes[i].IsOverview != notes[j].IsOverview {
			return notes[i].IsOverview
		}
		return strings.ToLower(notes[i].Title) < strings.ToLower(notes[j].Title)
	})

	return notes, nil
}

// SearchNotes performs a simple case-insensitive search across notes
// in the vault. It walks known Notes/ folders by scanning the vault root
// directory for workspace folders that contain Notes/ subdirectories.
//
// This is a minimal local search without an index. For a large vault,
// it should be replaced by a proper search plugin/indexer.
func (s *Service) SearchNotes(vaultRoot string, query string) ([]NoteInfo, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, nil
	}

	var results []NoteInfo
	seen := make(map[string]bool)

	vaultRoot = filepath.ToSlash(filepath.Clean(vaultRoot))
	rootDir := vaultRoot

	entries, err := os.ReadDir(rootDir)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		workspaceName := entry.Name()
		if strings.HasPrefix(workspaceName, ".") {
			continue
		}

		notesDir := filepath.Join(rootDir, workspaceName, CanonicalFolder)
		notesRel := filepath.ToSlash(filepath.Join(workspaceName, CanonicalFolder))

		notesEntries, err := os.ReadDir(notesDir)
		if err != nil {
			continue // no Notes folder in this workspace
		}

		for _, noteEntry := range notesEntries {
			if noteEntry.IsDir() {
				continue
			}
			ext := strings.ToLower(filepath.Ext(noteEntry.Name()))
			if ext != ".md" && ext != ".markdown" {
				continue
			}

			title := TitleFromFilename(noteEntry.Name())
			noteRel := notesRel + "/" + noteEntry.Name()

			if seen[noteRel] {
				continue
			}

			if matchSearch(title, query) || matchSearch(noteEntry.Name(), query) {
				results = append(results, NoteInfo{
					Title:      title,
					Filename:   noteEntry.Name(),
					Path:       noteRel,
					ParentPath: workspaceName,
					IsOverview: strings.EqualFold(noteEntry.Name(), CanonicalOverview),
				})
				seen[noteRel] = true
			}
		}
	}

	sort.Slice(results, func(i, j int) bool {
		return strings.ToLower(results[i].Title) < strings.ToLower(results[j].Title)
	})

	return results, nil
}

// ConflictError is returned when a note filename conflicts with an existing file.
type ConflictError struct {
	Path     string
	Title    string
	Filename string
}

func (e *ConflictError) Error() string {
	return fmt.Sprintf("conflict: a note with filename %q already exists at %q", e.Filename, e.Path)
}

// ─── helpers ──────────────────────────────────────────────────

// pathDir returns the parent directory of a relative path, or "" if root.
func pathDir(rel string) string {
	idx := strings.LastIndex(rel, "/")
	if idx < 0 {
		return ""
	}
	return rel[:idx]
}

func isConflictError(err error) bool {
	if err == nil {
		return false
	}
	return strings.HasPrefix(err.Error(), "conflict:")
}

func isNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	return strings.HasPrefix(err.Error(), "not-found:")
}

// matchSearch performs case-insensitive substring match.
// It also attempts basic RU↔EN layout swap matching for the query.
func matchSearch(text, query string) bool {
	lower := strings.ToLower(text)
	q := strings.ToLower(query)
	if strings.Contains(lower, q) {
		return true
	}
	// Attempt swapped layout matching (RU↔EN)
	swapped := swapKeyboardLayout(q)
	if swapped != q && strings.Contains(lower, swapped) {
		return true
	}
	// Also try swapping the text and matching the original query
	swappedText := swapKeyboardLayout(lower)
	if swappedText != lower && strings.Contains(swappedText, q) {
		return true
	}
	return false
}

// swapKeyboardLayout performs simple RU↔EN character swap for common
// misplaced keyboard layout characters. This is a best-effort mapping
// for the most common mismatched characters in note titles.
func swapKeyboardLayout(s string) string {
	var swapped strings.Builder
	for _, r := range s {
		if en, ok := ruToEn[r]; ok {
			swapped.WriteRune(en)
		} else if ru, ok := enToRu[r]; ok {
			swapped.WriteRune(ru)
		} else {
			swapped.WriteRune(r)
		}
	}
	return swapped.String()
}

// ruToEn maps Russian Cyrillic characters to their English QWERTY counterparts.
var ruToEn = map[rune]rune{
	'а': 'f', 'б': ',', 'в': 'd', 'г': 'u', 'д': 'l', 'е': 't', 'ё': '`',
	'ж': ';', 'з': 'p', 'и': 'b', 'й': 'q', 'к': 'r', 'л': 'k', 'м': 'v',
	'н': 'y', 'о': 'j', 'п': 'g', 'р': 'h', 'с': 'c', 'т': 'n', 'у': 'e',
	'ф': 'a', 'х': '[', 'ц': 'w', 'ч': 'x', 'ш': 'i', 'щ': 'o', 'ъ': ']',
	'ы': 's', 'ь': 'm', 	'э': '\'', 'ю': '.', 'я': 'z',
	'А': 'F', 'Б': '<', 'В': 'D', 'Г': 'U', 'Д': 'L', 'Е': 'T', 'Ё': '~',
	'Ж': ':', 'З': 'P', 'И': 'B', 'Й': 'Q', 'К': 'R', 'Л': 'K', 'М': 'V',
	'Н': 'Y', 'О': 'J', 'П': 'G', 'Р': 'H', 'С': 'C', 'Т': 'N', 'У': 'E',
	'Ф': 'A', 'Х': '{', 'Ц': 'W', 'Ч': 'X', 'Ш': 'I', 'Щ': 'O', 'Ъ': '}',
	'Ы': 'S', 'Ь': 'M', 'Э': '"', 'Ю': '>', 'Я': 'Z',
}

// enToRu maps English QWERTY characters to Russian Cyrillic.
var enToRu map[rune]rune

func init() {
	enToRu = make(map[rune]rune, len(ruToEn))
	for ru, en := range ruToEn {
		enToRu[en] = ru
	}
}
