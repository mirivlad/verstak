package files

import (
	"strings"
	"testing"
)

func TestNormalizeRelativeDirAllowsRootAndPreservesCase(t *testing.T) {
	got, err := NormalizeRelativeDir("")
	if err != nil {
		t.Fatalf("NormalizeRelativeDir root: %v", err)
	}
	if got != "" {
		t.Fatalf("root dir = %q, want empty", got)
	}

	got, err = NormalizeRelativeDir("Notes/Overview.md")
	if err != nil {
		t.Fatalf("NormalizeRelativeDir preserves case: %v", err)
	}
	if got != "Notes/Overview.md" {
		t.Fatalf("path = %q, want Notes/Overview.md", got)
	}
}

func TestNormalizeRelativeFileRejectsUnsafePaths(t *testing.T) {
	cases := []string{
		"",
		"/etc/passwd",
		"C:\\Users\\file.txt",
		"C:/Windows/system.ini",
		`\\server\share`,
		"//server/share",
		`..\secret`,
		`folder\..\secret`,
		"../outside.txt",
		"folder/../../outside.txt",
		`folder\sub/../../secret`,
		`folder\sub`,
		"bad\x00name.txt",
		".verstak",
		".verstak/",
		".verstak/vault.json",
		"./.verstak",
		".verstak/trash",
		"Workspace/.verstak/workspace.json",
		"folder/../.verstak",
		".Verstak",
	}

	for _, input := range cases {
		t.Run(input, func(t *testing.T) {
			_, err := NormalizeRelativeFile(input)
			if err == nil {
				t.Fatalf("NormalizeRelativeFile(%q): expected error", input)
			}
		})
	}
}

func TestReservedPathPolicy(t *testing.T) {
	if !IsReservedPath(".verstak") {
		t.Fatal(".verstak should be reserved")
	}
	if !IsReservedPath(".verstak/trash/file.txt") {
		t.Fatal(".verstak/trash/file.txt should be reserved")
	}
	if !IsReservedPath(".Verstak/trash/file.txt") {
		t.Fatal(".Verstak/trash/file.txt should be reserved by case-insensitive policy")
	}
	if IsReservedPath("Notes/.verstak.md") {
		t.Fatal("Notes/.verstak.md should not be reserved")
	}
	if !IsReservedPath("Workspace/.verstak/workspace.json") {
		t.Fatal("nested .verstak should be reserved")
	}
}

func TestNormalizeRelativeFileAcceptsOnlySlashSeparatedRelativePaths(t *testing.T) {
	got, err := NormalizeRelativeFile("Notes/Overview.md")
	if err != nil {
		t.Fatalf("NormalizeRelativeFile valid slash path: %v", err)
	}
	if got != "Notes/Overview.md" {
		t.Fatalf("path = %q, want Notes/Overview.md", got)
	}
}

func TestPathPolicyErrorsAreReadable(t *testing.T) {
	_, err := NormalizeRelativeFile("../outside.txt")
	if err == nil {
		t.Fatal("expected traversal error")
	}
	if !strings.Contains(err.Error(), "path-traversal") {
		t.Fatalf("error = %q, want path-traversal", err.Error())
	}
}
