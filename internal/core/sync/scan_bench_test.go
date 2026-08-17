package sync

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// buildSyntheticVault writes a vault with deals Deals, each holding files
// Notes and files Files, so the total is deals*files*2 regular files. Sizes are
// small on purpose: the cost being measured is walking and stat-ing the tree,
// not reading content.
func buildSyntheticVault(tb testing.TB, root string, deals, files int) int {
	tb.Helper()
	total := 0
	for d := 0; d < deals; d++ {
		deal := filepath.Join(root, fmt.Sprintf("Deal-%03d", d))
		for _, section := range []string{"Notes", "Files"} {
			dir := filepath.Join(deal, section)
			if err := os.MkdirAll(dir, 0o755); err != nil {
				tb.Fatal(err)
			}
			for f := 0; f < files; f++ {
				name := filepath.Join(dir, fmt.Sprintf("item-%03d.md", f))
				if err := os.WriteFile(name, []byte("# item\n\nbody\n"), 0o600); err != nil {
					tb.Fatal(err)
				}
				total++
			}
		}
	}
	return total
}

// TestScanAndRecordCostMeasurement records what a single recorded file change
// costs today at two synthetic vault sizes.
//
// Every write, move, copy and delete in the Files API goes through
// recordFileSyncOp, which ignores its arguments and calls ScanAndRecord — a
// full walk of the vault. One file operation therefore costs O(vault), and
// pasting N files costs O(N x vault), serialised behind one mutex. This test
// records the numbers so the improvement remains visible in verbose test logs.
//
// Wall-clock ordering is intentionally not asserted. A single filesystem scan
// on a shared CI runner is dominated by cache state and scheduler noise; in
// practice the 800-file sample can occasionally complete faster than the
// 200-file sample. Performance comparisons belong in the benchmarks below,
// where Go can run enough iterations to produce a meaningful measurement.
func TestScanAndRecordCostMeasurement(t *testing.T) {
	if testing.Short() {
		t.Skip("timing measurement")
	}

	measure := func(deals, files int) (int, time.Duration) {
		root := t.TempDir()
		count := buildSyntheticVault(t, root, deals, files)
		service := NewService(root, "device-a")
		// First scan writes the baseline; the second is the steady-state cost
		// of recording one change.
		if _, err := service.ScanAndRecord(); err != nil {
			t.Fatal(err)
		}
		target := filepath.Join(root, "Deal-000", "Notes", "item-000.md")
		if err := os.WriteFile(target, []byte("# changed\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		start := time.Now()
		if _, err := service.ScanAndRecord(); err != nil {
			t.Fatal(err)
		}
		return count, time.Since(start)
	}

	smallCount, smallCost := measure(10, 10)
	largeCount, largeCost := measure(40, 10)

	t.Logf("one recorded change in a %d-file vault: %v", smallCount, smallCost)
	t.Logf("one recorded change in a %d-file vault: %v", largeCount, largeCost)
	t.Logf("observed cost delta across %d additional files: %v", largeCount-smallCount, largeCost-smallCost)
}

// BenchmarkRecordOneChange reports the per-operation cost directly, so the
// before and after of the sync recording change can be compared with
// `go test -bench RecordOneChange ./internal/core/sync`.
func BenchmarkRecordOneChange(b *testing.B) {
	root := b.TempDir()
	buildSyntheticVault(b, root, 40, 10)
	service := NewService(root, "device-a")
	if _, err := service.ScanAndRecord(); err != nil {
		b.Fatal(err)
	}
	target := filepath.Join(root, "Deal-000", "Notes", "item-000.md")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		if err := os.WriteFile(target, []byte(fmt.Sprintf("# change %d\n", i)), 0o600); err != nil {
			b.Fatal(err)
		}
		b.StartTimer()
		if _, err := service.ScanAndRecord(); err != nil {
			b.Fatal(err)
		}
	}
}

// TestRecordCostOnSyntheticVault measures one recorded file change against a
// vault on disk, at a size worth caring about.
//
// Generate one with scripts/make-synthetic-vault.sh and point this at it:
//
//	VERSTAK_BENCH_VAULT=/tmp/verstak-bench go test ./internal/core/sync \
//	    -run TestRecordCostOnSyntheticVault -v
//
// It writes a sync snapshot into the vault, so give it a throwaway one.
func TestRecordCostOnSyntheticVault(t *testing.T) {
	root := os.Getenv("VERSTAK_BENCH_VAULT")
	if root == "" {
		t.Skip("set VERSTAK_BENCH_VAULT to a throwaway vault")
	}

	var fileCount int
	if err := filepath.Walk(root, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			fileCount++
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}

	service := NewService(root, "device-bench")
	if _, err := service.ScanAndRecord(); err != nil {
		t.Fatal(err)
	}

	target := ""
	if err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil || target != "" {
			return err
		}
		if !info.IsDir() && filepath.Ext(path) == ".md" {
			target = path
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if target == "" {
		t.Skip("no markdown file in the vault to touch")
	}

	relative, err := filepath.Rel(root, target)
	if err != nil {
		t.Fatal(err)
	}
	relative = filepath.ToSlash(relative)

	measure := func(record func(round int) error) time.Duration {
		const rounds = 5
		var total time.Duration
		for i := 0; i < rounds; i++ {
			if err := os.WriteFile(target, []byte(fmt.Sprintf("# round %d\n", i)), 0o600); err != nil {
				t.Fatal(err)
			}
			start := time.Now()
			if err := record(i); err != nil {
				t.Fatal(err)
			}
			total += time.Since(start)
		}
		return total / rounds
	}

	full := measure(func(int) error {
		_, err := service.ScanAndRecord()
		return err
	})
	scoped := measure(func(int) error {
		_, err := service.ScanPathsAndRecord([]string{relative})
		return err
	})
	// What pasting into one destination folder now costs: the files are written
	// first, then a single scan covers all of them.
	batch := measure(func(int) error {
		_, err := service.ScanPathsAndRecord([]string{filepath.ToSlash(filepath.Dir(relative))})
		return err
	})

	t.Logf("vault: %s (%d files)", root, fileCount)
	t.Logf("one file change, full scan:     %v", full)
	t.Logf("one file change, scoped scan:   %v", scoped)
	t.Logf("pasting 200 files, one per scan: %v", scoped*200)
	t.Logf("pasting 200 files, one batch:    %v", batch)
}
