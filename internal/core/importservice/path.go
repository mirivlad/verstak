package importservice

import (
	"crypto/sha256"
	"encoding/hex"
	"mime"
	"path"
	"regexp"
	"strings"

	"golang.org/x/text/unicode/norm"
)

var drivePathPattern = regexp.MustCompile(`^[A-Za-z]:`)

func normalizeSourcePath(input string) (string, error) {
	if input == "" || strings.ContainsRune(input, '\x00') || strings.Contains(input, `\`) {
		return "", sourceError("unsafe-source-path", "invalid path")
	}
	trimmed := strings.TrimSuffix(input, "/")
	for strings.HasPrefix(trimmed, "./") {
		trimmed = strings.TrimPrefix(trimmed, "./")
	}
	if trimmed == "" || strings.HasPrefix(trimmed, "/") || drivePathPattern.MatchString(trimmed) {
		return "", sourceError("unsafe-source-path", "absolute or empty path")
	}
	cleaned := path.Clean(trimmed)
	if cleaned == "." || cleaned == ".." || strings.HasPrefix(cleaned, "../") || cleaned != trimmed {
		return "", sourceError("unsafe-source-path", "path traversal or non-canonical path")
	}
	for _, segment := range strings.Split(cleaned, "/") {
		if segment == "" || segment == "." || segment == ".." || invalidPortableSegment(segment) {
			return "", sourceError("unsafe-source-path", "invalid path segment")
		}
	}
	return norm.NFC.String(cleaned), nil
}

func isArchiveRootEntry(input string) bool {
	trimmed := strings.TrimSuffix(input, "/")
	for strings.HasPrefix(trimmed, "./") {
		trimmed = strings.TrimPrefix(trimmed, "./")
	}
	return trimmed == "."
}

func invalidPortableSegment(segment string) bool {
	trimmed := strings.TrimRight(segment, " .")
	if trimmed == "" || trimmed != segment {
		return true
	}
	stem := trimmed
	if dot := strings.IndexByte(stem, '.'); dot >= 0 {
		stem = stem[:dot]
	}
	switch strings.ToUpper(stem) {
	case "CON", "PRN", "AUX", "NUL", "COM1", "COM2", "COM3", "COM4", "COM5", "COM6", "COM7", "COM8", "COM9", "LPT1", "LPT2", "LPT3", "LPT4", "LPT5", "LPT6", "LPT7", "LPT8", "LPT9":
		return true
	}
	return false
}

func collisionKey(value string) string {
	return strings.ToLower(norm.NFC.String(value))
}

func entryID(normalizedPath string) string {
	sum := sha256.Sum256([]byte(normalizedPath))
	return hex.EncodeToString(sum[:])
}

func mediaHint(normalizedPath string) string {
	extension := strings.ToLower(path.Ext(normalizedPath))
	switch extension {
	case ".md", ".markdown":
		return "text/markdown"
	case ".txt":
		return "text/plain"
	}
	return mime.TypeByExtension(extension)
}
