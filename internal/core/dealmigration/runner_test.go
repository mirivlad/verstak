package dealmigration

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

func copyMigrationFixture(t *testing.T) string {
	t.Helper()
	source := filepath.Join("testdata", "v016-vault")
	vault := t.TempDir()
	if err := filepath.WalkDir(source, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(source, path)
		if err != nil || rel == "." {
			return err
		}
		target := filepath.Join(vault, rel)
		if entry.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0o600)
	}); err != nil {
		t.Fatal(err)
	}
	return vault
}

func hashMigrationInputs(t *testing.T, vault string) string {
	t.Helper()
	paths := []string{}
	for _, root := range []string{".verstak/workspaces", ".verstak/plugin-data", ".verstak/plugin-settings"} {
		abs := filepath.Join(vault, filepath.FromSlash(root))
		_ = filepath.WalkDir(abs, func(path string, entry fs.DirEntry, err error) error {
			if err != nil || entry.IsDir() {
				return err
			}
			rel, err := filepath.Rel(vault, path)
			if err != nil {
				return err
			}
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			sum := sha256.Sum256(data)
			paths = append(paths, filepath.ToSlash(rel)+":"+hex.EncodeToString(sum[:]))
			return nil
		})
	}
	sort.Strings(paths)
	return strings.Join(paths, "\n")
}

func TestRunnerRestoresExactBytesWhenApplyFails(t *testing.T) {
	vault := copyMigrationFixture(t)
	before := hashMigrationInputs(t, vault)
	runner := NewRunner(vault,
		WithTransform(FuncTransform("write-provider", func(_ context.Context, vault string) error {
			if err := os.WriteFile(filepath.Join(vault, ".verstak", "plugin-data", "verstak.todo", "items.ndjson"), []byte(`{"taskId":"rewritten"}\n`), 0o600); err != nil {
				return err
			}
			return os.WriteFile(filepath.Join(vault, ".verstak", "plugin-data", "verstak.todo", "new-items.ndjson"), []byte(`{"taskId":"new"}\n`), 0o600)
		}, nil)),
		WithInjectedFailure("after-provider-write"),
	)
	if err := runner.Run(context.Background()); err == nil {
		t.Fatal("Run unexpectedly succeeded")
	}
	if after := hashMigrationInputs(t, vault); after != before {
		t.Fatalf("rollback changed source bytes\nbefore=%s\nafter=%s", before, after)
	}
	ledger, err := runner.ReadLedger()
	if err != nil {
		t.Fatal(err)
	}
	if ledger.State != StatePrepared || ledger.LastError == "" {
		t.Fatalf("ledger after rollback = %#v", ledger)
	}
}

func TestRunnerVerifiedMigrationIsIdempotent(t *testing.T) {
	vault := copyMigrationFixture(t)
	calls := 0
	runner := NewRunner(vault, WithTransform(FuncTransform("count", func(context.Context, string) error {
		calls++
		return nil
	}, func(context.Context, string) error { return nil })))
	if err := runner.Run(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := runner.Run(context.Background()); err != nil {
		t.Fatal(err)
	}
	if calls != 1 {
		t.Fatalf("transform calls = %d, want 1", calls)
	}
	ledger, err := runner.ReadLedger()
	if err != nil {
		t.Fatal(err)
	}
	if ledger.State != StateVerified || ledger.SourceDigest == "" || ledger.TargetDigest == "" {
		t.Fatalf("verified ledger = %#v", ledger)
	}
}

func TestRunnerPreflightRejectsSymlinkedMigrationInput(t *testing.T) {
	vault := copyMigrationFixture(t)
	target := filepath.Join(vault, ".verstak", "plugin-data", "verstak.todo", "items.ndjson")
	link := filepath.Join(vault, ".verstak", "plugin-data", "linked.ndjson")
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	runner := NewRunner(vault)
	err := runner.Preflight(context.Background())
	if err == nil || !errors.Is(err, ErrUnsafeInput) {
		t.Fatalf("Preflight error = %v", err)
	}
}
