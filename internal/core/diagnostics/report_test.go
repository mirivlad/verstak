package diagnostics

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func sampleInput() Input {
	return Input{
		AppVersion: "v0.1.0-beta.1",
		Commit:     "abc1234",
		BuiltAt:    "2026-07-28T10:00:00Z",
		Language:   "ru",
		VaultPath:  "/home/someone/Vault",
		VaultOpen:  true,
		SyncServer: "https://sync.example.com",
		SyncDevice: "device-1",
		LastSyncAt: "2026-07-28T09:59:00Z",
		Plugins: []PluginState{
			{ID: "verstak.notes", Version: "0.1.0", Source: "official", Status: "loaded", Enabled: true},
			{ID: "verstak.files", Version: "0.1.0", Source: "official", Status: "failed", Enabled: true, Error: "manifest is not valid JSON"},
			{ID: "verstak.todo", Version: "0.1.0", Source: "official", Status: "loaded", Enabled: false},
		},
		LogPath: "/home/someone/.local/share/verstak/logs/verstak-2026-07-28.log",
		LogTail: "[main] started\n[plugin] verstak.files: failed\n",
	}
}

func TestRenderNamesTheBuildTheVaultAndEveryPlugin(t *testing.T) {
	report := Render(sampleInput(), time.Date(2026, 7, 28, 12, 0, 0, 0, time.UTC))

	for _, expected := range []string{
		"v0.1.0-beta.1",
		"abc1234",
		"/home/someone/Vault",
		"https://sync.example.com",
		"verstak.notes",
		"verstak.files",
		"manifest is not valid JSON",
		"[plugin] verstak.files: failed",
	} {
		if !strings.Contains(report, expected) {
			t.Fatalf("report is missing %q:\n%s", expected, report)
		}
	}
	// A plugin somebody switched off is exactly what a bug report needs to say.
	if !strings.Contains(report, "verstak.todo") || !strings.Contains(report, "disabled") {
		t.Fatalf("a disabled plugin is not described:\n%s", report)
	}
}

// The report is meant to be sent to somebody. What must never be in it is what
// would let them into the vault or the sync account.
func TestRenderCarriesNoSecrets(t *testing.T) {
	input := sampleInput()
	input.LogTail = "[sync] configured\n"
	report := Render(input, time.Now())

	for _, forbidden := range []string{"apiKey", "api_key", "token", "password", "secret"} {
		if strings.Contains(strings.ToLower(report), forbidden) {
			t.Fatalf("report mentions %q:\n%s", forbidden, report)
		}
	}
}

func TestReadLogTailReturnsTheEndOfTheFileAtALineBoundary(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "session.log")
	var builder strings.Builder
	for index := 0; builder.Len() < LogTailBytes*2; index++ {
		builder.WriteString("line ")
		builder.WriteString(strings.Repeat("x", 100))
		builder.WriteString("\n")
	}
	builder.WriteString("the last line\n")
	if err := os.WriteFile(path, []byte(builder.String()), 0o600); err != nil {
		t.Fatal(err)
	}

	tail := ReadLogTail(path)
	if len(tail) > LogTailBytes {
		t.Fatalf("tail is %d bytes, want at most %d", len(tail), LogTailBytes)
	}
	if !strings.HasSuffix(tail, "the last line\n") {
		t.Fatalf("tail does not end at the end of the log: %q", tail[max(0, len(tail)-40):])
	}
	if !strings.HasPrefix(tail, "line ") {
		t.Fatalf("tail starts mid-line: %q", tail[:40])
	}
	if ReadLogTail(filepath.Join(dir, "missing.log")) != "" {
		t.Fatal("a missing log produced something")
	}
}

func TestWritePutsTheReportWhereItCanBeFound(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "logs")
	path, err := Write(dir, "report body", time.Date(2026, 7, 28, 12, 30, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Dir(path) != dir {
		t.Fatalf("report written to %s, want a file in %s", path, dir)
	}
	data, err := os.ReadFile(path)
	if err != nil || string(data) != "report body" {
		t.Fatalf("read back %q err=%v", data, err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	// Somebody else's account on the same machine has no business reading it.
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("report mode = %v, want 0600", info.Mode().Perm())
	}
}

func max(left, right int) int {
	if left > right {
		return left
	}
	return right
}
