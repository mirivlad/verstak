package debug

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPruneSessionsKeepsTheNewestAndLeavesCrashReports(t *testing.T) {
	dir := t.TempDir()
	for index := 0; index < keptSessions+5; index++ {
		name := fmt.Sprintf("verstak-2026-07-%02d-120000.log", index+1)
		if err := os.WriteFile(filepath.Join(dir, name), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	// A crash report is the file somebody will actually be asked for.
	crash := filepath.Join(dir, "crash-2026-07-01-120000.log")
	if err := os.WriteFile(crash, []byte("boom"), 0o644); err != nil {
		t.Fatal(err)
	}

	pruneSessions(dir)

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	sessions := 0
	crashKept := false
	newest := false
	for _, entry := range entries {
		switch {
		case strings.HasPrefix(entry.Name(), "verstak-"):
			sessions++
			if entry.Name() == fmt.Sprintf("verstak-2026-07-%02d-120000.log", keptSessions+5) {
				newest = true
			}
		case strings.HasPrefix(entry.Name(), "crash-"):
			crashKept = true
		}
	}
	// One short of the limit, because the run doing the pruning is about to
	// open its own.
	if sessions != keptSessions-1 {
		t.Fatalf("kept %d session logs, want %d", sessions, keptSessions-1)
	}
	if !newest {
		t.Fatal("the newest session log was deleted")
	}
	if !crashKept {
		t.Fatal("a crash report was deleted with the session logs")
	}
}

func TestWriteCrashReportRecordsTheReasonAndTheStack(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)

	path := WriteCrashReport("something went wrong", []byte("goroutine 1 [running]:\nmain.main()"))
	if path == "" {
		t.Fatal("no crash report was written")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	report := string(data)
	for _, expected := range []string{"something went wrong", "goroutine 1 [running]", "verstak crash"} {
		if !strings.Contains(report, expected) {
			t.Fatalf("crash report is missing %q:\n%s", expected, report)
		}
	}
	if filepath.Dir(path) != LogDir() {
		t.Fatalf("crash report written to %s, want a file in %s", path, LogDir())
	}
}
