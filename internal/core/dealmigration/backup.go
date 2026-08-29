package dealmigration

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
)

const migrationID = "deal-only-v1"

var ErrUnsafeInput = errors.New("unsafe migration input")

type manifestRoot struct {
	Path   string `json:"path"`
	Exists bool   `json:"exists"`
}

type manifestEntry struct {
	Path   string      `json:"path"`
	Kind   string      `json:"kind"`
	Mode   fs.FileMode `json:"mode"`
	Bytes  int64       `json:"bytes,omitempty"`
	SHA256 string      `json:"sha256,omitempty"`
}

// Manifest identifies every copied input byte. Its digest is the source
// digest recorded in the migration ledger.
type Manifest struct {
	SchemaVersion int             `json:"schemaVersion"`
	MigrationID   string          `json:"migrationId"`
	CreatedAt     string          `json:"createdAt"`
	Roots         []manifestRoot  `json:"roots"`
	Entries       []manifestEntry `json:"entries"`
	Digest        string          `json:"digest"`
}

type backup struct {
	Path     string
	Manifest Manifest
}

func (r *Runner) createBackup() (backup, error) {
	backupRoot := filepath.Join(r.vaultDir, ".verstak", "migrations", migrationID, "backups")
	if err := os.MkdirAll(backupRoot, 0o755); err != nil {
		return backup{}, fmt.Errorf("create migration backup root: %w", err)
	}
	name := time.Now().UTC().Format("20060102T150405.000000000Z") + "-" + uuid.NewString()
	finalPath := filepath.Join(backupRoot, name)
	stagingPath := finalPath + ".staging"
	if err := os.Mkdir(stagingPath, 0o700); err != nil {
		return backup{}, fmt.Errorf("create migration backup staging directory: %w", err)
	}
	defer os.RemoveAll(stagingPath)

	manifest, err := snapshotRoots(r.vaultDir, r.roots, filepath.Join(stagingPath, "files"), true)
	if err != nil {
		return backup{}, err
	}
	manifest.CreatedAt = time.Now().UTC().Format(time.RFC3339Nano)
	manifest.Digest, err = manifestDigest(manifest)
	if err != nil {
		return backup{}, err
	}
	data, err := json.Marshal(manifest)
	if err != nil {
		return backup{}, fmt.Errorf("marshal migration backup manifest: %w", err)
	}
	if err := os.WriteFile(filepath.Join(stagingPath, "manifest.json"), data, 0o600); err != nil {
		return backup{}, fmt.Errorf("write migration backup manifest: %w", err)
	}
	if err := os.Rename(stagingPath, finalPath); err != nil {
		return backup{}, fmt.Errorf("publish migration backup: %w", err)
	}
	return backup{Path: finalPath, Manifest: manifest}, nil
}

func readBackup(path string) (backup, error) {
	data, err := os.ReadFile(filepath.Join(path, "manifest.json"))
	if err != nil {
		return backup{}, fmt.Errorf("read migration backup manifest: %w", err)
	}
	var manifest Manifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return backup{}, fmt.Errorf("decode migration backup manifest: %w", err)
	}
	if manifest.SchemaVersion != 1 || manifest.MigrationID != migrationID {
		return backup{}, fmt.Errorf("unsupported migration backup manifest")
	}
	digest, err := manifestDigest(manifest)
	if err != nil {
		return backup{}, err
	}
	if digest != manifest.Digest {
		return backup{}, fmt.Errorf("migration backup manifest digest mismatch")
	}
	return backup{Path: path, Manifest: manifest}, nil
}

func (r *Runner) restoreBackup(value backup) error {
	for _, root := range value.Manifest.Roots {
		path, err := resolveInputRoot(r.vaultDir, root.Path)
		if err != nil {
			return err
		}
		if err := os.RemoveAll(path); err != nil {
			return fmt.Errorf("remove mutated migration root %s: %w", root.Path, err)
		}
	}
	for _, entry := range value.Manifest.Entries {
		target, err := resolveInputRoot(r.vaultDir, entry.Path)
		if err != nil {
			return err
		}
		switch entry.Kind {
		case "dir":
			if err := os.MkdirAll(target, entry.Mode.Perm()); err != nil {
				return fmt.Errorf("restore migration directory %s: %w", entry.Path, err)
			}
		case "file":
			source := filepath.Join(value.Path, "files", filepath.FromSlash(entry.Path))
			data, err := os.ReadFile(source)
			if err != nil {
				return fmt.Errorf("read migration backup file %s: %w", entry.Path, err)
			}
			sum := sha256.Sum256(data)
			if hex.EncodeToString(sum[:]) != entry.SHA256 || int64(len(data)) != entry.Bytes {
				return fmt.Errorf("migration backup file integrity mismatch: %s", entry.Path)
			}
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			if err := os.WriteFile(target, data, entry.Mode.Perm()); err != nil {
				return fmt.Errorf("restore migration file %s: %w", entry.Path, err)
			}
		default:
			return fmt.Errorf("unsupported migration backup entry kind %q", entry.Kind)
		}
	}
	return nil
}

func snapshotRoots(vaultDir string, roots []string, destination string, copyBytes bool) (Manifest, error) {
	manifest := Manifest{SchemaVersion: 1, MigrationID: migrationID, Roots: make([]manifestRoot, 0, len(roots))}
	for _, root := range roots {
		path, err := resolveInputRoot(vaultDir, root)
		if err != nil {
			return Manifest{}, err
		}
		info, err := os.Lstat(path)
		if errors.Is(err, os.ErrNotExist) {
			manifest.Roots = append(manifest.Roots, manifestRoot{Path: root, Exists: false})
			continue
		}
		if err != nil {
			return Manifest{}, fmt.Errorf("inspect migration input %s: %w", root, err)
		}
		if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return Manifest{}, fmt.Errorf("%w: migration root %s must be a real directory", ErrUnsafeInput, root)
		}
		manifest.Roots = append(manifest.Roots, manifestRoot{Path: root, Exists: true})
		err = filepath.WalkDir(path, func(current string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if entry.Type()&os.ModeSymlink != 0 {
				return fmt.Errorf("%w: symbolic link at %s", ErrUnsafeInput, current)
			}
			info, err := entry.Info()
			if err != nil {
				return err
			}
			if info.Mode()&os.ModeSymlink != 0 {
				return fmt.Errorf("%w: symbolic link at %s", ErrUnsafeInput, current)
			}
			rel, err := filepath.Rel(vaultDir, current)
			if err != nil {
				return err
			}
			rel = filepath.ToSlash(rel)
			if info.IsDir() {
				manifest.Entries = append(manifest.Entries, manifestEntry{Path: rel, Kind: "dir", Mode: info.Mode()})
				if copyBytes {
					return os.MkdirAll(filepath.Join(destination, filepath.FromSlash(rel)), info.Mode().Perm())
				}
				return nil
			}
			if !info.Mode().IsRegular() {
				return fmt.Errorf("%w: unsupported migration input type at %s", ErrUnsafeInput, current)
			}
			data, err := os.ReadFile(current)
			if err != nil {
				return err
			}
			sum := sha256.Sum256(data)
			manifest.Entries = append(manifest.Entries, manifestEntry{Path: rel, Kind: "file", Mode: info.Mode(), Bytes: int64(len(data)), SHA256: hex.EncodeToString(sum[:])})
			if !copyBytes {
				return nil
			}
			target := filepath.Join(destination, filepath.FromSlash(rel))
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			return os.WriteFile(target, data, info.Mode().Perm())
		})
		if err != nil {
			return Manifest{}, err
		}
	}
	sort.Slice(manifest.Roots, func(i, j int) bool { return manifest.Roots[i].Path < manifest.Roots[j].Path })
	sort.Slice(manifest.Entries, func(i, j int) bool { return manifest.Entries[i].Path < manifest.Entries[j].Path })
	return manifest, nil
}

func manifestDigest(manifest Manifest) (string, error) {
	copy := manifest
	copy.Digest = ""
	copy.CreatedAt = ""
	data, err := json.Marshal(copy)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}

func resolveInputRoot(vaultDir, root string) (string, error) {
	root = filepath.ToSlash(strings.TrimSpace(root))
	if root == ".verstak" || !strings.HasPrefix(root, ".verstak/") || strings.HasPrefix(root, ".verstak/migrations/") {
		return "", fmt.Errorf("%w: root %q is outside migration metadata inputs", ErrUnsafeInput, root)
	}
	cleaned := filepath.ToSlash(filepath.Clean(filepath.FromSlash(root)))
	if cleaned != root || strings.Contains(cleaned, "..") {
		return "", fmt.Errorf("%w: invalid migration root %q", ErrUnsafeInput, root)
	}
	vaultDir = filepath.Clean(vaultDir)
	if !filepath.IsAbs(vaultDir) {
		return "", fmt.Errorf("%w: vault path must be absolute", ErrUnsafeInput)
	}
	path := filepath.Join(vaultDir, filepath.FromSlash(cleaned))
	rel, err := filepath.Rel(vaultDir, path)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("%w: migration root escapes vault", ErrUnsafeInput)
	}
	return path, nil
}
