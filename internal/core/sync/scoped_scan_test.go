package sync

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// The scoped scanner exists to make a file operation cost O(what changed)
// instead of O(vault). That is only worth having if its result is
// indistinguishable from the full scan it replaces: the snapshot feeds the
// operation log, and a snapshot that disagrees with the log makes the next full
// scan invent duplicate or contradictory operations.
//
// These tests hold that line by running both paths over the same change and
// comparing what they produce.

func seedScopedVault(t *testing.T, root string) {
	t.Helper()
	for _, dir := range []string{"Clients/Deal-A/Notes", "Clients/Deal-A/Files", "Clients/Deal-B/Notes", "Loose"} {
		if err := os.MkdirAll(filepath.Join(root, filepath.FromSlash(dir)), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	files := map[string]string{
		"Clients/Deal-A/Notes/one.md":  "one",
		"Clients/Deal-A/Notes/two.md":  "two",
		"Clients/Deal-A/Files/doc.txt": "doc",
		"Clients/Deal-B/Notes/b.md":    "b",
		"Loose/loose.md":               "loose",
	}
	for rel, content := range files {
		if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(rel)), []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
}

// comparableOps strips the fields that are unique per recording — identifiers
// and timestamps — so two runs of the same change can be compared on what they
// actually assert about the vault.
func comparableOps(t *testing.T, ops []Op) []string {
	t.Helper()
	var result []string
	for _, op := range ops {
		payload := op.PayloadJSON
		var decoded interface{}
		if payload != "" {
			if err := json.Unmarshal([]byte(payload), &decoded); err != nil {
				t.Fatalf("decode payload %q: %v", payload, err)
			}
			normalized, err := json.Marshal(decoded)
			if err != nil {
				t.Fatal(err)
			}
			payload = string(normalized)
		}
		result = append(result, op.EntityType+" "+op.EntityID+" "+op.OpType+" "+payload)
	}
	return result
}

// withoutTimestamps removes the fields that record when a scan happened rather
// than what it found. The two runs being compared operate on separately created
// vaults, so their wall-clock stamps legitimately differ; everything the
// scanner actually reasons about — path, type, size, content hash — must not.
func withoutTimestamps(snapshot Snapshot) Snapshot {
	snapshot.ScannedAt = ""
	entries := make(map[string]SnapshotEntry, len(snapshot.Entries))
	for path, entry := range snapshot.Entries {
		entry.ModifiedAt = ""
		entries[path] = entry
	}
	snapshot.Entries = entries
	return snapshot
}

// runChange applies mutate to a fresh vault and returns the snapshot and
// operations recorded, either by scanning everything or only the given paths.
func runChange(t *testing.T, scoped []string, mutate func(root string)) (Snapshot, []string) {
	t.Helper()
	root := t.TempDir()
	seedScopedVault(t, root)
	service := NewService(root, "device-a")
	if _, err := service.ScanAndRecord(); err != nil {
		t.Fatalf("baseline scan: %v", err)
	}
	before, err := service.GetUnpushedOps()
	if err != nil {
		t.Fatal(err)
	}

	mutate(root)

	if scoped == nil {
		if _, err := service.ScanAndRecord(); err != nil {
			t.Fatalf("full scan: %v", err)
		}
	} else if _, err := service.ScanPathsAndRecord(scoped); err != nil {
		t.Fatalf("scoped scan: %v", err)
	}

	after, err := service.GetUnpushedOps()
	if err != nil {
		t.Fatal(err)
	}
	snapshot, err := service.LoadSnapshot()
	if err != nil {
		t.Fatal(err)
	}
	recorded := comparableOps(t, after[len(before):])

	// The sharpest check available: whatever the scan just wrote, a full scan
	// immediately afterwards must find nothing left to say. A snapshot that
	// disagrees with the filesystem shows up here as leftover operations, and
	// duplicated operations show up here as a repeat.
	if _, err := service.ScanAndRecord(); err != nil {
		t.Fatalf("verification scan: %v", err)
	}
	settled, err := service.GetUnpushedOps()
	if err != nil {
		t.Fatal(err)
	}
	if leftover := comparableOps(t, settled[len(after):]); len(leftover) != 0 {
		t.Errorf("snapshot did not agree with the vault; a following full scan still recorded %v", leftover)
	}

	return withoutTimestamps(snapshot), recorded
}

func TestScopedScanMatchesFullScan(t *testing.T) {
	cases := []struct {
		name   string
		scoped []string
		mutate func(root string)
	}{
		{
			name:   "file created",
			scoped: []string{"Clients/Deal-A/Notes"},
			mutate: func(root string) {
				write(at(root, "Clients/Deal-A/Notes/three.md"), "three")
			},
		},
		{
			name:   "file updated",
			scoped: []string{"Clients/Deal-A/Notes/one.md"},
			mutate: func(root string) {
				write(at(root, "Clients/Deal-A/Notes/one.md"), "one changed")
			},
		},
		{
			name:   "file deleted",
			scoped: []string{"Clients/Deal-A/Notes/two.md"},
			mutate: func(root string) {
				if err := os.Remove(at(root, "Clients/Deal-A/Notes/two.md")); err != nil {
					panic(err)
				}
			},
		},
		{
			name:   "file moved between deals",
			scoped: []string{"Clients/Deal-A/Notes/one.md", "Clients/Deal-B/Notes/one.md"},
			mutate: func(root string) {
				if err := os.Rename(at(root, "Clients/Deal-A/Notes/one.md"), at(root, "Clients/Deal-B/Notes/one.md")); err != nil {
					panic(err)
				}
			},
		},
		{
			name:   "folder created with contents",
			scoped: []string{"Clients/Deal-A/Files/Sub"},
			mutate: func(root string) {
				if err := os.MkdirAll(at(root, "Clients/Deal-A/Files/Sub/Deeper"), 0o755); err != nil {
					panic(err)
				}
				write(at(root, "Clients/Deal-A/Files/Sub/a.txt"), "a")
				write(at(root, "Clients/Deal-A/Files/Sub/Deeper/b.txt"), "b")
			},
		},
		{
			name:   "whole folder deleted",
			scoped: []string{"Clients/Deal-B"},
			mutate: func(root string) {
				if err := os.RemoveAll(at(root, "Clients/Deal-B")); err != nil {
					panic(err)
				}
			},
		},
		{
			// The scope names only the files. The directory holding them is new
			// too, and has to be recorded even though nothing pointed at it.
			name:   "files pasted into a folder created by the same operation",
			scoped: []string{"Clients/Deal-B/Notes/New/a.md", "Clients/Deal-B/Notes/New/b.md"},
			mutate: func(root string) {
				write(at(root, "Clients/Deal-B/Notes/New/a.md"), "a")
				write(at(root, "Clients/Deal-B/Notes/New/b.md"), "b")
			},
		},
		{
			name:   "many files pasted at once",
			scoped: []string{"Clients/Deal-B/Notes"},
			mutate: func(root string) {
				for _, name := range []string{"p1.md", "p2.md", "p3.md", "p4.md"} {
					write(at(root, "Clients/Deal-B/Notes/"+name), "pasted "+name)
				}
			},
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			fullSnapshot, fullOps := runChange(t, nil, testCase.mutate)
			scopedSnapshot, scopedOps := runChange(t, testCase.scoped, testCase.mutate)

			if !reflect.DeepEqual(fullOps, scopedOps) {
				t.Errorf("operations differ\nfull scan:   %v\nscoped scan: %v", fullOps, scopedOps)
			}
			if !reflect.DeepEqual(fullSnapshot.Entries, scopedSnapshot.Entries) {
				for path, entry := range fullSnapshot.Entries {
					if scoped, ok := scopedSnapshot.Entries[path]; !ok {
						t.Errorf("scoped scan missing entry %q", path)
					} else if scoped != entry {
						t.Errorf("entry %q: full %+v, scoped %+v", path, entry, scoped)
					}
				}
				for path := range scopedSnapshot.Entries {
					if _, ok := fullSnapshot.Entries[path]; !ok {
						t.Errorf("scoped scan invented entry %q", path)
					}
				}
			}
			if !reflect.DeepEqual(fullSnapshot.Folders, scopedSnapshot.Folders) {
				t.Errorf("folders differ\nfull:   %+v\nscoped: %+v", fullSnapshot.Folders, scopedSnapshot.Folders)
			}
			if !reflect.DeepEqual(fullSnapshot.Workspaces, scopedSnapshot.Workspaces) {
				t.Errorf("workspaces differ\nfull:   %+v\nscoped: %+v", fullSnapshot.Workspaces, scopedSnapshot.Workspaces)
			}
		})
	}
}

// A scoped scan is told what changed. If the caller is wrong about that, the
// change outside the scope must simply wait for the next full scan rather than
// corrupt the snapshot — the following full scan has to still report it.
func TestScopedScanLeavesUnrelatedChangesForTheNextFullScan(t *testing.T) {
	root := t.TempDir()
	seedScopedVault(t, root)
	service := NewService(root, "device-a")
	if _, err := service.ScanAndRecord(); err != nil {
		t.Fatal(err)
	}

	write(filepath.Join(root, filepath.FromSlash("Loose/loose.md")), "changed outside the scope")
	write(filepath.Join(root, filepath.FromSlash("Clients/Deal-A/Notes/one.md")), "changed inside the scope")

	if _, err := service.ScanPathsAndRecord([]string{"Clients/Deal-A/Notes/one.md"}); err != nil {
		t.Fatal(err)
	}
	ops, err := service.GetUnpushedOps()
	if err != nil {
		t.Fatal(err)
	}
	for _, op := range ops {
		if op.EntityID == "Loose/loose.md" {
			t.Fatalf("scoped scan recorded a change it was not asked to look at: %+v", op)
		}
	}

	if _, err := service.ScanAndRecord(); err != nil {
		t.Fatal(err)
	}
	ops, err = service.GetUnpushedOps()
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, op := range ops {
		if op.EntityID == "Loose/loose.md" && op.OpType == OpUpdate {
			found = true
		}
	}
	if !found {
		t.Fatalf("the change outside the scope was lost; full scan ops = %v", comparableOps(t, ops))
	}
}

// The stat cache must never hide a rewrite that keeps the same length and lands
// in the same filesystem timestamp tick — the exact shape of an edit that
// changes a word for another of equal size.
func TestScanNoticesSameLengthRewriteWithinOneTimestampTick(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "note.md")
	write(target, "aaaaaaaaaaaaaaaa")

	service := NewService(root, "device-a")
	if _, err := service.ScanAndRecord(); err != nil {
		t.Fatal(err)
	}
	if _, err := service.ScanAndRecord(); err != nil {
		t.Fatal(err)
	}
	before, err := service.GetUnpushedOps()
	if err != nil {
		t.Fatal(err)
	}

	// Written immediately, so the filesystem is likely to stamp it with the
	// same coarse modification time it gave the original.
	write(target, "bbbbbbbbbbbbbbbb")

	if _, err := service.ScanAndRecord(); err != nil {
		t.Fatal(err)
	}
	after, err := service.GetUnpushedOps()
	if err != nil {
		t.Fatal(err)
	}
	recorded := comparableOps(t, after[len(before):])
	if len(recorded) != 1 {
		t.Fatalf("same-length rewrite produced %v, want exactly one update", recorded)
	}
}

func TestScanScopeRefusesPathsItCannotScopeBySafely(t *testing.T) {
	cases := []struct {
		name  string
		paths []string
		want  scanScope
	}{
		{name: "empty", paths: nil, want: nil},
		{name: "vault root", paths: []string{"/"}, want: nil},
		{name: "escaping", paths: []string{"Deal/../.."}, want: nil},
		{name: "only excluded", paths: []string{".verstak/sync/ops.json"}, want: nil},
		{
			// A leading slash means the vault root, so this scopes to a path
			// inside the vault that does not exist. The walk is rooted at the
			// vault and simply finds nothing — it cannot reach the real file.
			name:  "leading slash is vault-relative",
			paths: []string{"/etc/passwd"},
			want:  scanScope{"etc/passwd"},
		},
		{name: "normalised", paths: []string{"/Deal/Notes/", "Deal/Notes"}, want: scanScope{"Deal/Notes"}},
		{name: "nested collapsed", paths: []string{"Deal", "Deal/Notes/a.md"}, want: scanScope{"Deal"}},
		{name: "siblings kept", paths: []string{"B/two.md", "A/one.md"}, want: scanScope{"A/one.md", "B/two.md"}},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			if got := newScanScope(testCase.paths); !reflect.DeepEqual(got, testCase.want) {
				t.Fatalf("newScanScope(%v) = %v, want %v", testCase.paths, got, testCase.want)
			}
		})
	}
}

func at(root, rel string) string { return filepath.Join(root, filepath.FromSlash(rel)) }

func write(path, content string) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		panic(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		panic(err)
	}
}
