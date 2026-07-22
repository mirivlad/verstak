package importservice

import "testing"

func TestNormalizeSourcePathRejectsUnsafeNames(t *testing.T) {
	unsafe := []string{
		"", ".", "../escape.txt", "safe/../../escape.txt", "/absolute.txt",
		`C:/drive.txt`, `C:\drive.txt`, `\\server\share\file.txt`, "safe\\file.txt",
		"safe/\x00file.txt", "NUL", "safe/CON.txt", "safe/trailing. ",
	}
	for _, input := range unsafe {
		t.Run(input, func(t *testing.T) {
			if _, err := normalizeSourcePath(input); err == nil {
				t.Fatalf("normalizeSourcePath(%q) succeeded", input)
			}
		})
	}
}

func TestNormalizeSourcePathUsesSlashPaths(t *testing.T) {
	got, err := normalizeSourcePath("pages/project/start.txt")
	if err != nil {
		t.Fatal(err)
	}
	if got != "pages/project/start.txt" {
		t.Fatalf("path=%q", got)
	}
}

func TestNormalizeSourcePathAcceptsSafeTarDotPrefix(t *testing.T) {
	got, err := normalizeSourcePath("./wiki/data/pages/start.txt")
	if err != nil {
		t.Fatal(err)
	}
	if got != "wiki/data/pages/start.txt" {
		t.Fatalf("path=%q", got)
	}
}

func TestCollisionKeyIsUnicodeAndCaseInsensitive(t *testing.T) {
	if collisionKey("Notes/CAFÉ.md") != collisionKey("notes/cafe\u0301.md") {
		t.Fatal("expected equivalent collision keys")
	}
}
