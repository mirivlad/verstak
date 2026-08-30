// Package dealmigration provides the one-shot Deal-only migration transaction.
// It is intentionally dormant until the complete target schema bundle is
// available; no production startup path invokes Runner.Run in this foundation
// phase.
package dealmigration

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/verstak/verstak-desktop/internal/core/workspacetree"
)

type State string

const (
	StatePrepared State = "prepared"
	StateApplied  State = "applied"
	StateVerified State = "verified"
)

// Ledger is the resumable record for exactly one Deal-only migration.
type Ledger struct {
	SchemaVersion int    `json:"schemaVersion"`
	MigrationID   string `json:"migrationId"`
	State         State  `json:"state"`
	BackupPath    string `json:"backupPath"`
	SourceDigest  string `json:"sourceDigest"`
	TargetDigest  string `json:"targetDigest,omitempty"`
	PreparedAt    string `json:"preparedAt"`
	AppliedAt     string `json:"appliedAt,omitempty"`
	VerifiedAt    string `json:"verifiedAt,omitempty"`
	LastError     string `json:"lastError,omitempty"`
}

// Transform is a target-schema migration step. The framework deliberately has
// no provider knowledge; target plugins register their transforms only when
// their complete schemas are available.
type Transform interface {
	Name() string
	Apply(context.Context, string) error
	Verify(context.Context, string) error
}

type funcTransform struct {
	name   string
	apply  func(context.Context, string) error
	verify func(context.Context, string) error
}

func (t funcTransform) Name() string                                  { return t.name }
func (t funcTransform) Apply(ctx context.Context, vault string) error { return t.apply(ctx, vault) }
func (t funcTransform) Verify(ctx context.Context, vault string) error {
	if t.verify == nil {
		return nil
	}
	return t.verify(ctx, vault)
}

// FuncTransform builds a small transform for tests and package-local wiring.
func FuncTransform(name string, apply func(context.Context, string) error, verify func(context.Context, string) error) Transform {
	return funcTransform{name: name, apply: apply, verify: verify}
}

type Option func(*Runner)

func WithTransform(transform Transform) Option {
	return func(r *Runner) { r.transforms = append(r.transforms, transform) }
}

func WithInjectedFailure(point string) Option {
	return func(r *Runner) { r.injectedFailure = point }
}

// Runner is inert until a caller explicitly invokes Run.
type Runner struct {
	vaultDir        string
	roots           []string
	transforms      []Transform
	injectedFailure string
}

func NewRunner(vaultDir string, options ...Option) *Runner {
	runner := &Runner{
		vaultDir: filepath.Clean(vaultDir),
		roots: []string{
			".verstak/workspaces",
			".verstak/plugin-settings",
			".verstak/plugin-data",
		},
	}
	for _, option := range options {
		option(runner)
	}
	return runner
}

// NewDealOnlyRunner assembles the complete one-shot migration. Calling it is
// harmless; migration still requires an explicit Run invocation.
func NewDealOnlyRunner(vaultDir string) *Runner {
	return NewRunner(vaultDir,
		WithTransform(NewDealMetadataTransform()),
		WithTransform(NewProviderDataTransform()),
		WithTransform(NewProjectMetaTransform()),
		WithTransform(NewMilestoneDataTransform()),
	)
}

// NeedsMigration reports whether this vault still has the retired Project
// scope or an interrupted one-shot migration. A verified ledger short-circuits
// every legacy read, so normal runtime never consults legacy Projects again.
func (r *Runner) NeedsMigration(ctx context.Context) (bool, error) {
	if err := r.Preflight(ctx); err != nil {
		return false, err
	}
	ledger, err := r.ReadLedger()
	if err == nil {
		return ledger.State != StateVerified, nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return false, err
	}
	legacy, err := hasLegacyDealScope(ctx, r.vaultDir)
	if err != nil || legacy {
		return legacy, err
	}
	return hasMissingCanonicalDealMetadata(r.vaultDir)
}

func hasMissingCanonicalDealMetadata(vault string) (bool, error) {
	rootToWorkspace, err := currentWorkspaceRootIDs(vault)
	if err != nil {
		return false, err
	}
	service := workspacetree.NewService(vault, nil)
	for rootPath, workspaceID := range rootToWorkspace {
		if _, err := service.ReadDealMetadata(workspaceID, rootPath); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return true, nil
			}
			return false, err
		}
	}
	return false, nil
}

func hasLegacyDealScope(ctx context.Context, vault string) (bool, error) {
	projectsPath := filepath.Join(vault, ".verstak", "plugin-settings", legacyProjectsPluginID, "settings.json")
	if data, err := os.ReadFile(projectsPath); err == nil {
		var settings map[string]any
		if err := json.Unmarshal(data, &settings); err != nil {
			return false, fmt.Errorf("decode legacy Projects settings: %w", err)
		}
		if projects, _ := settings[projectsSettingsKey].([]any); len(projects) > 0 {
			return true, nil
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return false, err
	}
	for _, root := range []string{
		filepath.Join(vault, ".verstak", "plugin-settings"),
		filepath.Join(vault, ".verstak", "plugin-data"),
	} {
		found := false
		err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
			if walkErr != nil {
				if errors.Is(walkErr, os.ErrNotExist) {
					return nil
				}
				return walkErr
			}
			if err := ctx.Err(); err != nil {
				return err
			}
			if entry.IsDir() || (filepath.Base(path) != "settings.json" && filepath.Ext(path) != ".ndjson") {
				return nil
			}
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			value, err := decodeProviderValue(data, filepath.Ext(path) == ".ndjson")
			if err != nil {
				return fmt.Errorf("decode migration input %s: %w", filepath.ToSlash(path), err)
			}
			if containsProjectScope(value) {
				found = true
				return filepath.SkipAll
			}
			return nil
		})
		if err != nil {
			return false, err
		}
		if found {
			return true, nil
		}
	}
	return false, nil
}

func (r *Runner) Preflight(context.Context) error {
	if !filepath.IsAbs(r.vaultDir) {
		return fmt.Errorf("%w: vault path must be absolute", ErrUnsafeInput)
	}
	info, err := os.Stat(r.vaultDir)
	if err != nil {
		return fmt.Errorf("inspect migration vault: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("%w: vault path is not a directory", ErrUnsafeInput)
	}
	seen := make(map[string]bool, len(r.transforms))
	for _, transform := range r.transforms {
		if transform == nil || strings.TrimSpace(transform.Name()) == "" {
			return fmt.Errorf("invalid migration transform")
		}
		if seen[transform.Name()] {
			return fmt.Errorf("duplicate migration transform %q", transform.Name())
		}
		seen[transform.Name()] = true
	}
	_, err = snapshotRoots(r.vaultDir, r.roots, "", false)
	return err
}

func (r *Runner) Prepare(ctx context.Context) (Ledger, error) {
	if err := r.Preflight(ctx); err != nil {
		return Ledger{}, err
	}
	if ledger, err := r.ReadLedger(); err == nil {
		switch ledger.State {
		case StatePrepared, StateApplied, StateVerified:
			return ledger, nil
		default:
			return Ledger{}, fmt.Errorf("unsupported migration ledger state %q", ledger.State)
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return Ledger{}, err
	}
	created, err := r.createBackup()
	if err != nil {
		return Ledger{}, err
	}
	ledger := Ledger{
		SchemaVersion: 1,
		MigrationID:   migrationID,
		State:         StatePrepared,
		BackupPath:    filepath.ToSlash(strings.TrimPrefix(created.Path, r.vaultDir+string(filepath.Separator))),
		SourceDigest:  created.Manifest.Digest,
		PreparedAt:    time.Now().UTC().Format(time.RFC3339Nano),
	}
	if err := r.writeLedger(ledger); err != nil {
		return Ledger{}, err
	}
	return ledger, nil
}

func (r *Runner) Apply(ctx context.Context) (Ledger, error) {
	ledger, err := r.Prepare(ctx)
	if err != nil {
		return Ledger{}, err
	}
	if ledger.State == StateVerified || ledger.State == StateApplied {
		return ledger, nil
	}
	backupPath, err := resolveLedgerBackup(r.vaultDir, ledger.BackupPath)
	if err != nil {
		return Ledger{}, err
	}
	value, err := readBackup(backupPath)
	if err != nil {
		return Ledger{}, err
	}
	for _, transform := range r.transforms {
		if err := ctx.Err(); err != nil {
			return r.rollback(ledger, value, err)
		}
		if err := transform.Apply(ctx, r.vaultDir); err != nil {
			return r.rollback(ledger, value, fmt.Errorf("apply %s: %w", transform.Name(), err))
		}
	}
	if r.injectedFailure == "after-provider-write" {
		return r.rollback(ledger, value, errors.New("injected failure after provider write"))
	}
	target, err := snapshotRoots(r.vaultDir, r.roots, "", false)
	if err != nil {
		return r.rollback(ledger, value, err)
	}
	ledger.State = StateApplied
	ledger.TargetDigest, err = manifestDigest(target)
	if err != nil {
		return r.rollback(ledger, value, err)
	}
	ledger.AppliedAt = time.Now().UTC().Format(time.RFC3339Nano)
	ledger.LastError = ""
	if err := r.writeLedger(ledger); err != nil {
		return Ledger{}, err
	}
	return ledger, nil
}

func (r *Runner) Verify(ctx context.Context) (Ledger, error) {
	ledger, err := r.ReadLedger()
	if err != nil {
		return Ledger{}, err
	}
	if ledger.State == StateVerified {
		return ledger, nil
	}
	if ledger.State != StateApplied {
		return Ledger{}, fmt.Errorf("migration must be applied before verification")
	}
	for _, transform := range r.transforms {
		if err := transform.Verify(ctx, r.vaultDir); err != nil {
			return Ledger{}, fmt.Errorf("verify %s: %w", transform.Name(), err)
		}
	}
	target, err := snapshotRoots(r.vaultDir, r.roots, "", false)
	if err != nil {
		return Ledger{}, err
	}
	targetDigest, err := manifestDigest(target)
	if err != nil {
		return Ledger{}, err
	}
	if targetDigest != ledger.TargetDigest {
		return Ledger{}, fmt.Errorf("migration target digest changed before verification")
	}
	ledger.State = StateVerified
	ledger.VerifiedAt = time.Now().UTC().Format(time.RFC3339Nano)
	ledger.LastError = ""
	if err := r.writeLedger(ledger); err != nil {
		return Ledger{}, err
	}
	return ledger, nil
}

func (r *Runner) Run(ctx context.Context) error {
	if err := r.Preflight(ctx); err != nil {
		return err
	}
	ledger, err := r.Prepare(ctx)
	if err != nil {
		return err
	}
	if ledger.State == StateVerified {
		return nil
	}
	if ledger.State == StatePrepared {
		if _, err := r.Apply(ctx); err != nil {
			return err
		}
	}
	_, err = r.Verify(ctx)
	return err
}

func (r *Runner) rollback(ledger Ledger, value backup, applyErr error) (Ledger, error) {
	if err := r.restoreBackup(value); err != nil {
		return Ledger{}, fmt.Errorf("%v; rollback failed: %w", applyErr, err)
	}
	ledger.State = StatePrepared
	ledger.TargetDigest = ""
	ledger.AppliedAt = ""
	ledger.LastError = applyErr.Error()
	if err := r.writeLedger(ledger); err != nil {
		return Ledger{}, err
	}
	return ledger, applyErr
}

func (r *Runner) ReadLedger() (Ledger, error) {
	data, err := os.ReadFile(r.ledgerPath())
	if err != nil {
		return Ledger{}, err
	}
	var ledger Ledger
	if err := json.Unmarshal(data, &ledger); err != nil {
		return Ledger{}, fmt.Errorf("decode migration ledger: %w", err)
	}
	if ledger.SchemaVersion != 1 || ledger.MigrationID != migrationID {
		return Ledger{}, fmt.Errorf("unsupported migration ledger")
	}
	return ledger, nil
}

func (r *Runner) writeLedger(ledger Ledger) error {
	dir := filepath.Dir(r.ledgerPath())
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	data, err := json.Marshal(ledger)
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".ledger-*.tmp")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)
	if err := tmp.Chmod(0o600); err != nil {
		_ = tmp.Close()
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpPath, r.ledgerPath())
}

func (r *Runner) ledgerPath() string {
	return filepath.Join(r.vaultDir, ".verstak", "migrations", migrationID, "ledger.json")
}

func resolveLedgerBackup(vaultDir, relativePath string) (string, error) {
	cleaned := filepath.ToSlash(filepath.Clean(filepath.FromSlash(relativePath)))
	prefix := ".verstak/migrations/" + migrationID + "/backups/"
	if !strings.HasPrefix(cleaned, prefix) || strings.Contains(strings.TrimPrefix(cleaned, prefix), "/") {
		return "", fmt.Errorf("%w: invalid migration backup path", ErrUnsafeInput)
	}
	return filepath.Join(vaultDir, filepath.FromSlash(cleaned)), nil
}
