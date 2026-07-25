package buildinfo

import "testing"

func TestResolveFallsBackToDev(t *testing.T) {
	version, commit, buildDate = "", "", ""
	got := resolve()
	if got.Version != "dev" {
		t.Fatalf("version without ldflags = %q, want %q", got.Version, "dev")
	}
	if got.Display == "" {
		t.Fatal("display string must never be empty")
	}
}

func TestResolveUsesLdflagValues(t *testing.T) {
	version, commit, buildDate = "v0.1.0-beta.3", "abcdef1234567890", "2026-07-26T00:00:00Z"
	t.Cleanup(func() { version, commit, buildDate = "", "", "" })

	got := resolve()
	if got.Version != "v0.1.0-beta.3" {
		t.Fatalf("version = %q", got.Version)
	}
	if got.Commit != "abcdef1" {
		t.Fatalf("commit = %q, want it shortened to 7 characters", got.Commit)
	}
	if got.Display != "v0.1.0-beta.3 (abcdef1)" {
		t.Fatalf("display = %q", got.Display)
	}
}
