// Package notes provides the Notes layout service, title-to-filename normalization,
// and note CRUD operations for Verstak vaults.
//
// Canonical layout:
//
//	<parent>/Notes/          — notes folder for a project/workspace
//	<parent>/Notes/Overview.md — overview note
//	<parent>/Notes/<title>.md  — individual notes
//
// The invariant is: Note title is the source of truth. The filename is derived
// from the title via normalization, never stored independently.
package notes

import (
	"path"
	"strings"
)

// CanonicalLayout contains the canonical names for the notes layout.
// All code should use these constants; never hardcode "Notes" or "Overview.md".
const (
	CanonicalFolder   = "Notes"       // canonical notes folder name (always title-case)
	CanonicalOverview = "Overview.md" // canonical overview filename

	// NoteExtension is the file extension for notes.
	NoteExtension = ".md"
)

// NotesPath returns the canonical notes folder path relative to parent.
// parent is a vault-relative directory path (e.g. "Workspace" or "Clients/Acme").
func NotesPath(parent string) string {
	parent = strings.TrimSpace(parent)
	if parent == "" {
		return CanonicalFolder
	}
	return parent + "/" + CanonicalFolder
}

// OverviewPath returns the canonical overview file path relative to parent.
func OverviewPath(parent string) string {
	return NotesPath(parent) + "/" + CanonicalOverview
}

// IsInsideNotes checks whether the given vault-relative path is inside
// a canonical Notes folder. It checks any segment named "Notes", not just the first.
func IsInsideNotes(relativePath string) bool {
	if relativePath == "" {
		return false
	}
	cleaned := strings.TrimSpace(relativePath)
	cleaned = strings.TrimPrefix(cleaned, "./")
	cleaned = strings.TrimPrefix(cleaned, "/")
	parts := strings.Split(cleaned, "/")
	for _, part := range parts {
		if part == CanonicalFolder {
			return true
		}
	}
	return false
}

// IsOverview checks whether the given vault-relative path is the canonical
// Overview.md inside a Notes folder.
func IsOverview(relativePath string) bool {
	if relativePath == "" {
		return false
	}
	cleaned := strings.TrimSpace(relativePath)
	if !strings.HasSuffix(cleaned, "/"+CanonicalOverview) &&
		cleaned != CanonicalOverview {
		return false
	}
	notesParent := strings.TrimSuffix(cleaned, "/"+CanonicalOverview)
	if notesParent == "" {
		return true // just "Notes/Overview.md"
	}
	return IsInsideNotes(notesParent)
}

// ParentFromNotePath extracts the notes parent (the directory containing
// the Notes/ folder) from a note's vault-relative path.
// For example: "Workspace/Notes/MyNote.md" -> "Workspace"
// For example: "Notes/MyNote.md" -> ""
func ParentFromNotePath(notePath string) string {
	notePath = strings.TrimSpace(notePath)
	parts := strings.Split(notePath, "/")
	for i, part := range parts {
		if part == CanonicalFolder {
			if i == 0 {
				return ""
			}
			return path.Join(parts[:i]...)
		}
	}
	return ""
}
