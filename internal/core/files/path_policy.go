package files

import (
	"fmt"
	"path"
	"path/filepath"
	"strings"
	"unicode"
)

func NormalizeRelativeDir(relativeDir string) (string, error) {
	return normalizeRelativePath(relativeDir, true)
}

func NormalizeRelativeFile(relativePath string) (string, error) {
	return normalizeRelativePath(relativePath, false)
}

func IsReservedPath(relativePath string) bool {
	normalized := strings.ReplaceAll(relativePath, "\\", "/")
	cleaned := path.Clean(normalized)
	if cleaned == "." {
		cleaned = ""
	}
	if cleaned == "" {
		return false
	}
	return containsReservedSegment(cleaned)
}

func normalizeRelativePath(input string, allowRoot bool) (string, error) {
	if strings.Contains(input, "\x00") {
		return "", fmt.Errorf("invalid-path: null-byte")
	}
	if strings.Contains(input, "\\") {
		return "", fmt.Errorf("invalid-path: backslash not allowed")
	}
	if looksAbsolute(input) {
		return "", fmt.Errorf("invalid-path: absolute path rejected")
	}

	normalized := input
	for _, part := range strings.Split(normalized, "/") {
		if part == ".." {
			return "", fmt.Errorf("invalid-path: path-traversal")
		}
	}

	cleaned := path.Clean(normalized)
	if cleaned == "." {
		cleaned = ""
	}
	if cleaned == "" && !allowRoot {
		return "", fmt.Errorf("invalid-path: empty path")
	}
	if cleaned == ".." || strings.HasPrefix(cleaned, "../") {
		return "", fmt.Errorf("invalid-path: path-traversal")
	}
	if IsReservedPathNoNormalize(cleaned) {
		return "", fmt.Errorf("reserved-path: .verstak is internal")
	}
	return cleaned, nil
}

func IsReservedPathNoNormalize(cleaned string) bool {
	if cleaned == "" {
		return false
	}
	return containsReservedSegment(cleaned)
}

func containsReservedSegment(cleaned string) bool {
	for _, segment := range strings.Split(cleaned, "/") {
		if strings.EqualFold(segment, ".verstak") {
			return true
		}
	}
	return false
}

func looksAbsolute(input string) bool {
	if input == "" {
		return false
	}
	if filepath.IsAbs(input) || strings.HasPrefix(input, "/") || strings.HasPrefix(input, "\\\\") || strings.HasPrefix(input, "\\") {
		return true
	}
	if len(input) >= 2 && input[1] == ':' && unicode.IsLetter(rune(input[0])) {
		return true
	}
	return false
}
