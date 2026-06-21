package notes

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"
)

// illegalFilenameChars matches characters that are unsafe or illegal in filenames
// across Linux, macOS, and Windows. We are strict to keep vault portable.
var illegalFilenameChars = regexp.MustCompile(`[<>:"/\\|?*\x00-\x1f\x7f]`)

// collapseWhitespace matches runs of whitespace.
var collapseWhitespace = regexp.MustCompile(`\s+`)

// typographicDashSet contains Unicode dash characters to normalize.
var typographicDashSet = []rune{0x2012, 0x2013, 0x2014, 0x2015, 0x2212}

// NormalizeTitleToFilename converts a note title to a safe filename.
//
// Rules:
//  1. Trim leading/trailing whitespace
//  2. Collapse internal whitespace runs → underscore
//  3. Typographic dashes (en dash, em dash, etc.) → ASCII hyphen
//  4. Remove/replace illegal filename characters
//  5. Preserve letters, digits, Unicode letters, `.`, `_`, `-`
//  6. Replace other characters with underscore
//  7. Ensure result is non-empty
//  8. Append `.md` extension
//
// Returns the normalized filename (with .md) or an error if the result is empty.
func NormalizeTitleToFilename(title string) (string, error) {
	s := strings.TrimSpace(title)

	// Strip any existing .md/.markdown extension for normalization, then re-add
	extStripped := false
	if strings.HasSuffix(strings.ToLower(s), ".markdown") && len(s) > 9 {
		s = s[:len(s)-9]
		extStripped = true
	} else if strings.HasSuffix(strings.ToLower(s), ".md") && len(s) > 3 {
		s = s[:len(s)-3]
		extStripped = true
	}
	if s == "" {
		return "", fmt.Errorf("title %q normalizes to an empty filename", title)
	}

	// Collapse whitespace runs → underscore
	s = collapseWhitespace.ReplaceAllString(s, "_")

	// Normalize dashes (typographic → ASCII hyphen)
	s = replaceTypographicDashes(s)

	// Remove illegal characters
	s = illegalFilenameChars.ReplaceAllString(s, "")

	// Replace any remaining unsafe characters (control chars, etc.)
	runes := make([]rune, 0, len(s))
	for _, r := range s {
		if r == '.' || r == '_' || r == '-' || unicode.IsLetter(r) || unicode.IsDigit(r) {
			runes = append(runes, r)
		} else if unicode.IsPrint(r) {
			runes = append(runes, '_')
		}
		// non-printable characters are dropped
	}
	s = string(runes)

	// Collapse multiple underscores/hyphens/dots (e.g. "foo___bar" → "foo_bar")
	s = collapseRepeatedUnderscores(s)

	// Trim leading/trailing dots, spaces, underscores, hyphens
	s = strings.Trim(s, "._- ")

	if s == "" {
		return "", fmt.Errorf("title %q normalizes to an empty filename", title)
	}

	// If the original title had .md/.markdown extension, preserve it exactly
	if extStripped {
		return s + NoteExtension, nil
	}
	return s + NoteExtension, nil
}

// replaceTypographicDashes replaces Unicode dash characters with ASCII hyphen.
func replaceTypographicDashes(s string) string {
	var result strings.Builder
	for _, r := range s {
		isDash := false
		for _, d := range typographicDashSet {
			if r == d {
				result.WriteRune('-')
				isDash = true
				break
			}
		}
		if !isDash {
			result.WriteRune(r)
		}
	}
	return result.String()
}

func collapseRepeatedUnderscores(s string) string {
	var result strings.Builder
	lastWasSep := false
	for _, r := range s {
		if r == '_' || r == '-' || r == '.' {
			if !lastWasSep {
				result.WriteRune('_')
				lastWasSep = true
			}
		} else {
			result.WriteRune(r)
			lastWasSep = false
		}
	}
	return result.String()
}

// TitleFromFilename extracts a human-readable title from a note filename.
// This is the inverse of NormalizeTitleToFilename (best-effort).
func TitleFromFilename(filename string) string {
	filename = strings.TrimSpace(filename)
	// Remove .md extension
	if strings.HasSuffix(strings.ToLower(filename), ".md") {
		filename = filename[:len(filename)-3]
	}
	// Replace underscores → spaces
	result := strings.ReplaceAll(filename, "_", " ")
	return strings.TrimSpace(result)
}

// ValidateNoteTitle checks that a title is valid for creating a note.
func ValidateNoteTitle(title string) error {
	title = strings.TrimSpace(title)
	if title == "" {
		return fmt.Errorf("note title must not be empty")
	}
	if len(title) > 500 {
		return fmt.Errorf("note title too long (%d characters, max 500)", len(title))
	}
	return nil
}
