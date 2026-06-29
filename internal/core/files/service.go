package files

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/fs"
	"mime"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"

	"github.com/verstak/verstak-desktop/internal/core/vault"
)

type Service struct {
	vault *vault.Vault
}

func NewService(v *vault.Vault) *Service {
	return &Service{vault: v}
}

func (s *Service) ListVaultFiles(relativeDir string) ([]FileEntry, error) {
	root, err := s.vaultRoot()
	if err != nil {
		return nil, err
	}
	rel, err := NormalizeRelativeDir(relativeDir)
	if err != nil {
		return nil, err
	}
	full, err := s.resolve(root, rel)
	if err != nil {
		return nil, err
	}
	if err := rejectSymlinkPath(root, rel, true); err != nil {
		return nil, err
	}
	info, err := os.Stat(full)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("not-found: %s", rel)
		}
		return nil, err
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("not-directory: %s", rel)
	}

	dirEntries, err := os.ReadDir(full)
	if err != nil {
		return nil, err
	}
	entries := make([]FileEntry, 0, len(dirEntries))
	for _, dirEntry := range dirEntries {
		childRel := joinRel(rel, dirEntry.Name())
		if IsReservedPathNoNormalize(childRel) {
			continue
		}
		info, err := dirEntry.Info()
		if err != nil {
			continue
		}
		entries = append(entries, makeEntry(childRel, info))
	}
	return entries, nil
}

func (s *Service) GetVaultFileMetadata(relativePath string) (FileMetadata, error) {
	root, rel, full, err := s.resolveFile(relativePath)
	if err != nil {
		return FileMetadata{}, err
	}
	if err := rejectSymlinkPath(root, rel, false); err != nil {
		return FileMetadata{}, err
	}
	info, err := os.Lstat(full)
	if err != nil {
		if os.IsNotExist(err) {
			return FileMetadata{}, fmt.Errorf("not-found: %s", rel)
		}
		return FileMetadata{}, err
	}
	return makeMetadata(rel, info), nil
}

func (s *Service) ResolveExternalOpenTarget(relativePath string) (ExternalOpenTarget, error) {
	root, rel, full, err := s.resolveFile(relativePath)
	if err != nil {
		return ExternalOpenTarget{}, err
	}
	if err := rejectSymlinkPath(root, rel, true); err != nil {
		return ExternalOpenTarget{}, err
	}
	info, err := os.Lstat(full)
	if err != nil {
		if os.IsNotExist(err) {
			return ExternalOpenTarget{}, fmt.Errorf("not-found: %s", rel)
		}
		return ExternalOpenTarget{}, err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return ExternalOpenTarget{}, fmt.Errorf("symlink-not-allowed: %s", rel)
	}
	return ExternalOpenTarget{
		RelativePath: rel,
		AbsolutePath: full,
		Metadata:     makeMetadata(rel, info),
	}, nil
}

func (s *Service) ReadVaultTextFile(relativePath string) (string, error) {
	root, rel, full, err := s.resolveFile(relativePath)
	if err != nil {
		return "", err
	}
	if err := rejectSymlinkPath(root, rel, true); err != nil {
		return "", err
	}
	info, err := os.Lstat(full)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("not-found: %s", rel)
		}
		return "", err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return "", fmt.Errorf("symlink-not-allowed: %s", rel)
	}
	if !info.Mode().IsRegular() {
		return "", fmt.Errorf("not-regular-file: %s", rel)
	}
	if info.Size() > MaxTextFileBytes {
		return "", fmt.Errorf("file-too-large: %s", rel)
	}
	data, err := os.ReadFile(full)
	if err != nil {
		return "", err
	}
	if !utf8.Valid(data) {
		return "", fmt.Errorf("not-text-file: %s", rel)
	}
	return string(data), nil
}

func (s *Service) ReadVaultFileBytes(relativePath string) (FileBytes, error) {
	root, rel, full, err := s.resolveFile(relativePath)
	if err != nil {
		return FileBytes{}, err
	}
	if err := rejectSymlinkPath(root, rel, true); err != nil {
		return FileBytes{}, err
	}
	info, err := os.Lstat(full)
	if err != nil {
		if os.IsNotExist(err) {
			return FileBytes{}, fmt.Errorf("not-found: %s", rel)
		}
		return FileBytes{}, err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return FileBytes{}, fmt.Errorf("symlink-not-allowed: %s", rel)
	}
	if !info.Mode().IsRegular() {
		return FileBytes{}, fmt.Errorf("not-regular-file: %s", rel)
	}
	if info.Size() > MaxBinaryReadBytes {
		return FileBytes{}, fmt.Errorf("file-too-large: %s", rel)
	}
	data, err := os.ReadFile(full)
	if err != nil {
		return FileBytes{}, err
	}
	return FileBytes{
		RelativePath: rel,
		Size:         int64(len(data)),
		MimeHint:     mime.TypeByExtension(filepath.Ext(info.Name())),
		DataBase64:   base64.StdEncoding.EncodeToString(data),
	}, nil
}

func (s *Service) WriteVaultTextFile(relativePath string, content string, options WriteOptions) error {
	return s.writeVaultFileData(relativePath, []byte(content), options)
}

func (s *Service) WriteVaultFileBytes(relativePath string, dataBase64 string, options WriteOptions) error {
	data, err := base64.StdEncoding.DecodeString(dataBase64)
	if err != nil {
		return fmt.Errorf("invalid-base64: %w", err)
	}
	if int64(len(data)) > MaxBinaryReadBytes {
		return fmt.Errorf("file-too-large: %s", relativePath)
	}
	return s.writeVaultFileData(relativePath, data, options)
}

func (s *Service) writeVaultFileData(relativePath string, data []byte, options WriteOptions) error {
	root, rel, full, err := s.resolveFile(relativePath)
	if err != nil {
		return err
	}
	if err := rejectSymlinkPath(root, rel, true); err != nil {
		return err
	}

	parent := filepath.Dir(full)
	if info, err := os.Stat(parent); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("parent-not-found: %s", pathDir(rel))
		}
		return err
	} else if !info.IsDir() {
		return fmt.Errorf("parent-not-directory: %s", pathDir(rel))
	}

	existing, err := os.Lstat(full)
	if err == nil {
		if existing.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("symlink-not-allowed: %s", rel)
		}
		if !existing.Mode().IsRegular() {
			return fmt.Errorf("not-regular-file: %s", rel)
		}
		if !options.Overwrite {
			return fmt.Errorf("conflict: %s", rel)
		}
	} else if os.IsNotExist(err) {
		if !options.CreateIfMissing {
			return fmt.Errorf("not-found: %s", rel)
		}
	} else {
		return err
	}

	tmp, err := os.CreateTemp(parent, ".verstak-write-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	cleanup := true
	defer func() {
		if cleanup {
			_ = os.Remove(tmpName)
		}
	}()
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpName, full); err != nil {
		return err
	}
	cleanup = false
	return nil
}

func (s *Service) CreateVaultFolder(relativePath string) error {
	root, rel, full, err := s.resolveFile(relativePath)
	if err != nil {
		return err
	}
	if _, err := os.Lstat(full); err == nil {
		return fmt.Errorf("conflict: %s", rel)
	} else if !os.IsNotExist(err) {
		return err
	}
	if err := rejectSymlinkPath(root, rel, true); err != nil {
		return err
	}
	parent := filepath.Dir(full)
	if info, err := os.Stat(parent); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("parent-not-found: %s", pathDir(rel))
		}
		return err
	} else if !info.IsDir() {
		return fmt.Errorf("parent-not-directory: %s", pathDir(rel))
	}
	return os.Mkdir(full, 0o755)
}

func (s *Service) MoveVaultPath(fromRelativePath string, toRelativePath string, options MoveOptions) error {
	root, fromRel, fromFull, err := s.resolveFile(fromRelativePath)
	if err != nil {
		return err
	}
	_, toRel, toFull, err := s.resolveFile(toRelativePath)
	if err != nil {
		return err
	}
	if fromRel == "" || toRel == "" {
		return fmt.Errorf("invalid-path: cannot move root")
	}
	if err := rejectSymlinkPath(root, fromRel, true); err != nil {
		return err
	}
	if err := rejectSymlinkPath(root, toRel, false); err != nil {
		return err
	}
	fromInfo, err := os.Lstat(fromFull)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("not-found: %s", fromRel)
		}
		return err
	}
	if fromInfo.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("symlink-not-allowed: %s", fromRel)
	}
	if fromInfo.IsDir() && (toRel == fromRel || strings.HasPrefix(toRel, fromRel+"/")) {
		return fmt.Errorf("move-into-self: %s -> %s", fromRel, toRel)
	}
	parent := filepath.Dir(toFull)
	if info, err := os.Stat(parent); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("parent-not-found: %s", pathDir(toRel))
		}
		return err
	} else if !info.IsDir() {
		return fmt.Errorf("parent-not-directory: %s", pathDir(toRel))
	}
	if _, err := os.Lstat(toFull); err == nil && !options.Overwrite {
		return fmt.Errorf("conflict: %s", toRel)
	} else if err != nil && !os.IsNotExist(err) {
		return err
	}
	return os.Rename(fromFull, toFull)
}

func (s *Service) TrashVaultPath(relativePath string) (TrashResult, error) {
	root, rel, full, err := s.resolveFile(relativePath)
	if err != nil {
		return TrashResult{}, err
	}
	if err := rejectSymlinkPath(root, rel, true); err != nil {
		return TrashResult{}, err
	}
	info, err := os.Lstat(full)
	if err != nil {
		if os.IsNotExist(err) {
			return TrashResult{}, fmt.Errorf("not-found: %s", rel)
		}
		return TrashResult{}, err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return TrashResult{}, fmt.Errorf("symlink-not-allowed: %s", rel)
	}

	deletedAt := time.Now().UTC().Format(time.RFC3339Nano)
	trashID := time.Now().UTC().Format("20060102T150405.000000000Z") + "-" + uuid.NewString()
	trashRel := filepath.ToSlash(filepath.Join(".verstak", "trash", "files", trashID, filepath.Base(rel)))
	trashFull := filepath.Join(root, filepath.FromSlash(trashRel))
	if err := os.MkdirAll(filepath.Dir(trashFull), 0o755); err != nil {
		return TrashResult{}, err
	}
	if err := os.Rename(full, trashFull); err != nil {
		return TrashResult{}, err
	}
	result := TrashResult{
		OriginalPath: rel,
		TrashPath:    trashRel,
		TrashID:      trashID,
		DeletedAt:    deletedAt,
	}
	meta := map[string]string{
		"originalPath": rel,
		"trashPath":    trashRel,
		"trashId":      trashID,
		"deletedAt":    deletedAt,
		"originalType": string(fileTypeFromInfo(info)),
		"basename":     filepath.Base(rel),
		"type":         string(fileTypeFromInfo(info)),
	}
	data, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return TrashResult{}, err
	}
	if err := os.WriteFile(filepath.Join(root, ".verstak", "trash", "files", trashID, "metadata.json"), data, 0o644); err != nil {
		return TrashResult{}, err
	}
	return result, nil
}

func (s *Service) ListTrashEntries() ([]TrashEntry, error) {
	root, err := s.vaultRoot()
	if err != nil {
		return nil, err
	}
	trashRoot := filepath.Join(root, ".verstak", "trash", "files")
	dirs, err := os.ReadDir(trashRoot)
	if err != nil {
		if os.IsNotExist(err) {
			return []TrashEntry{}, nil
		}
		return nil, err
	}

	entries := make([]TrashEntry, 0, len(dirs))
	for _, dir := range dirs {
		if !dir.IsDir() {
			continue
		}
		data, err := os.ReadFile(filepath.Join(trashRoot, dir.Name(), "metadata.json"))
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, err
		}
		var raw struct {
			OriginalPath string `json:"originalPath"`
			TrashPath    string `json:"trashPath"`
			TrashID      string `json:"trashId"`
			DeletedAt    string `json:"deletedAt"`
			OriginalType string `json:"originalType"`
			Basename     string `json:"basename"`
		}
		if err := json.Unmarshal(data, &raw); err != nil {
			continue
		}
		if raw.OriginalPath == "" || raw.TrashPath == "" || raw.TrashID == "" || raw.DeletedAt == "" {
			continue
		}
		entries = append(entries, TrashEntry{
			OriginalPath: raw.OriginalPath,
			TrashPath:    raw.TrashPath,
			TrashID:      raw.TrashID,
			DeletedAt:    raw.DeletedAt,
			OriginalType: FileType(raw.OriginalType),
			Basename:     raw.Basename,
		})
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].DeletedAt != entries[j].DeletedAt {
			return entries[i].DeletedAt > entries[j].DeletedAt
		}
		return entries[i].TrashID > entries[j].TrashID
	})
	return entries, nil
}

func (s *Service) RestoreTrashEntry(trashID string, options RestoreOptions) (string, error) {
	root, err := s.vaultRoot()
	if err != nil {
		return "", err
	}
	if err := validateTrashID(trashID); err != nil {
		return "", err
	}
	entry, err := readTrashEntry(root, trashID)
	if err != nil {
		return "", err
	}

	targetRel := entry.OriginalPath
	if options.TargetPath != "" {
		targetRel = options.TargetPath
	}
	targetRel, err = NormalizeRelativeFile(targetRel)
	if err != nil {
		return "", err
	}
	targetFull, err := s.resolve(root, targetRel)
	if err != nil {
		return "", err
	}
	if err := rejectSymlinkPath(root, targetRel, false); err != nil {
		return "", err
	}

	trashDir := filepath.Join(root, ".verstak", "trash", "files", trashID)
	payloadFull := filepath.Join(root, filepath.FromSlash(entry.TrashPath))
	if !isInsideDir(trashDir, payloadFull) {
		return "", fmt.Errorf("invalid-trash-entry: payload outside trash")
	}
	if info, err := os.Lstat(payloadFull); err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("not-found: trash payload %s", trashID)
		}
		return "", err
	} else if info.Mode()&os.ModeSymlink != 0 {
		return "", fmt.Errorf("symlink-not-allowed: %s", entry.TrashPath)
	}

	parent := filepath.Dir(targetFull)
	if info, err := os.Stat(parent); err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("parent-not-found: %s", pathDir(targetRel))
		}
		return "", err
	} else if !info.IsDir() {
		return "", fmt.Errorf("parent-not-directory: %s", pathDir(targetRel))
	}

	if existing, err := os.Lstat(targetFull); err == nil {
		if existing.Mode()&os.ModeSymlink != 0 {
			return "", fmt.Errorf("symlink-not-allowed: %s", targetRel)
		}
		if !options.Overwrite {
			return "", fmt.Errorf("conflict: %s", targetRel)
		}
		if err := os.RemoveAll(targetFull); err != nil {
			return "", err
		}
	} else if err != nil && !os.IsNotExist(err) {
		return "", err
	}

	if err := os.Rename(payloadFull, targetFull); err != nil {
		return "", err
	}
	if err := os.RemoveAll(trashDir); err != nil {
		return "", err
	}
	return targetRel, nil
}

func readTrashEntry(root, trashID string) (TrashEntry, error) {
	if err := validateTrashID(trashID); err != nil {
		return TrashEntry{}, err
	}
	data, err := os.ReadFile(filepath.Join(root, ".verstak", "trash", "files", trashID, "metadata.json"))
	if err != nil {
		if os.IsNotExist(err) {
			return TrashEntry{}, fmt.Errorf("not-found: trash entry %s", trashID)
		}
		return TrashEntry{}, err
	}
	var raw struct {
		OriginalPath string `json:"originalPath"`
		TrashPath    string `json:"trashPath"`
		TrashID      string `json:"trashId"`
		DeletedAt    string `json:"deletedAt"`
		OriginalType string `json:"originalType"`
		Basename     string `json:"basename"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return TrashEntry{}, err
	}
	if raw.OriginalPath == "" || raw.TrashPath == "" || raw.TrashID == "" || raw.DeletedAt == "" {
		return TrashEntry{}, fmt.Errorf("invalid-trash-entry: missing metadata")
	}
	if raw.TrashID != trashID {
		return TrashEntry{}, fmt.Errorf("invalid-trash-entry: mismatched trash id")
	}
	return TrashEntry{
		OriginalPath: raw.OriginalPath,
		TrashPath:    raw.TrashPath,
		TrashID:      raw.TrashID,
		DeletedAt:    raw.DeletedAt,
		OriginalType: FileType(raw.OriginalType),
		Basename:     raw.Basename,
	}, nil
}

func validateTrashID(trashID string) error {
	if trashID == "" || trashID == "." || trashID == ".." || strings.ContainsAny(trashID, "/\\\x00") {
		return fmt.Errorf("invalid-trash-id")
	}
	return nil
}

func isInsideDir(parent, child string) bool {
	absParent, err := filepath.Abs(parent)
	if err != nil {
		return false
	}
	absChild, err := filepath.Abs(child)
	if err != nil {
		return false
	}
	rel, err := filepath.Rel(absParent, absChild)
	if err != nil {
		return false
	}
	return rel != "." && rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator)) && !filepath.IsAbs(rel)
}

func (s *Service) vaultRoot() (string, error) {
	if s == nil || s.vault == nil {
		return "", fmt.Errorf("vault-not-initialized")
	}
	if s.vault.GetVaultStatus() != vault.StatusOpen {
		return "", fmt.Errorf("vault-not-open")
	}
	root := s.vault.GetVaultPath()
	if root == "" {
		return "", fmt.Errorf("vault-not-open")
	}
	return root, nil
}

func (s *Service) resolveFile(relativePath string) (string, string, string, error) {
	root, err := s.vaultRoot()
	if err != nil {
		return "", "", "", err
	}
	rel, err := NormalizeRelativeFile(relativePath)
	if err != nil {
		return "", "", "", err
	}
	full, err := s.resolve(root, rel)
	return root, rel, full, err
}

func (s *Service) resolve(root, rel string) (string, error) {
	full := filepath.Join(root, filepath.FromSlash(rel))
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	absFull, err := filepath.Abs(full)
	if err != nil {
		return "", err
	}
	relToRoot, err := filepath.Rel(absRoot, absFull)
	if err != nil {
		return "", err
	}
	if relToRoot == ".." || strings.HasPrefix(relToRoot, ".."+string(os.PathSeparator)) || filepath.IsAbs(relToRoot) {
		return "", fmt.Errorf("invalid-path: path-traversal")
	}
	return absFull, nil
}

func rejectSymlinkPath(root, rel string, includeFinal bool) error {
	if rel == "" {
		return nil
	}
	parts := strings.Split(rel, "/")
	limit := len(parts)
	if !includeFinal {
		limit--
	}
	current := root
	for i := 0; i < limit; i++ {
		current = filepath.Join(current, filepath.FromSlash(parts[i]))
		info, err := os.Lstat(current)
		if err != nil {
			if os.IsNotExist(err) {
				return nil
			}
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("symlink-not-allowed: %s", strings.Join(parts[:i+1], "/"))
		}
	}
	return nil
}

func makeEntry(rel string, info fs.FileInfo) FileEntry {
	t := fileTypeFromInfo(info)
	return FileEntry{
		Name:         info.Name(),
		RelativePath: rel,
		Type:         t,
		Size:         sizeForType(t, info),
		ModifiedAt:   info.ModTime().UTC().Format(time.RFC3339Nano),
		Extension:    strings.TrimPrefix(filepath.Ext(info.Name()), "."),
		IsHidden:     strings.HasPrefix(info.Name(), "."),
		IsReserved:   IsReservedPathNoNormalize(rel),
		CanRead:      t == FileTypeFile || t == FileTypeFolder,
		CanWrite:     t == FileTypeFile || t == FileTypeFolder,
	}
}

func makeMetadata(rel string, info fs.FileInfo) FileMetadata {
	t := fileTypeFromInfo(info)
	ext := strings.TrimPrefix(filepath.Ext(info.Name()), ".")
	return FileMetadata{
		RelativePath: rel,
		Type:         t,
		Size:         sizeForType(t, info),
		ModifiedAt:   info.ModTime().UTC().Format(time.RFC3339Nano),
		Extension:    ext,
		MimeHint:     mime.TypeByExtension(filepath.Ext(info.Name())),
		IsText:       isTextExtension(ext),
		IsHidden:     strings.HasPrefix(info.Name(), "."),
		IsReserved:   IsReservedPathNoNormalize(rel),
		CanRead:      t == FileTypeFile || t == FileTypeFolder,
		CanWrite:     t == FileTypeFile || t == FileTypeFolder,
	}
}

func fileTypeFromInfo(info fs.FileInfo) FileType {
	if info.Mode()&os.ModeSymlink != 0 {
		return FileTypeSymlink
	}
	if info.IsDir() {
		return FileTypeFolder
	}
	if info.Mode().IsRegular() {
		return FileTypeFile
	}
	return FileTypeUnknown
}

func sizeForType(t FileType, info fs.FileInfo) int64 {
	if t == FileTypeFolder {
		return 0
	}
	return info.Size()
}

func isTextExtension(ext string) bool {
	switch strings.ToLower(ext) {
	case "txt", "md", "markdown", "json", "yaml", "yml", "toml", "csv", "log", "xml", "html", "css", "js", "ts", "svelte", "go":
		return true
	default:
		return false
	}
}

func joinRel(parent, name string) string {
	if parent == "" {
		return name
	}
	return parent + "/" + name
}

func pathDir(rel string) string {
	dir := pathDirSlash(rel)
	if dir == "." {
		return ""
	}
	return dir
}

func pathDirSlash(rel string) string {
	idx := strings.LastIndex(rel, "/")
	if idx < 0 {
		return "."
	}
	return rel[:idx]
}
