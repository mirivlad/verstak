package importservice

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type directorySource struct {
	root             string
	rootMeta         string
	items            []sourceEntry
	byID             map[string]sourceEntry
	total            int64
	fingerprintValue string
	limits           limits
}

func newDirectorySource(selectedPath string, limits limits) (*directorySource, error) {
	absPath, err := filepath.Abs(selectedPath)
	if err != nil {
		return nil, sourceError("invalid-source", "could not resolve selected directory")
	}
	rootInfo, err := os.Lstat(absPath)
	if err != nil {
		return nil, sourceError("source-unavailable", "selected directory is unavailable")
	}
	if rootInfo.Mode()&os.ModeSymlink != 0 || !rootInfo.IsDir() {
		return nil, sourceError("unsupported-source-entry", "selected path is not a regular directory")
	}

	source := &directorySource{root: absPath, limits: limits}
	if err := source.index(rootInfo); err != nil {
		return nil, err
	}
	return source, nil
}

func (source *directorySource) index(rootInfo os.FileInfo) error {
	items := make([]sourceEntry, 0)
	seen := make(map[string]string)
	var total int64
	err := filepath.WalkDir(source.root, func(currentPath string, dirEntry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return sourceError("source-unavailable", "could not index selected directory")
		}
		if currentPath == source.root {
			return nil
		}
		info, err := os.Lstat(currentPath)
		if err != nil {
			return sourceError("source-unavailable", "could not index selected directory")
		}
		if info.Mode()&os.ModeSymlink != 0 || (!info.IsDir() && !info.Mode().IsRegular()) {
			return sourceError("unsupported-source-entry", "links and special files are not supported")
		}
		relativePath, err := filepath.Rel(source.root, currentPath)
		if err != nil {
			return sourceError("unsafe-source-path", "entry escaped selected directory")
		}
		normalizedPath, err := normalizeSourcePath(filepath.ToSlash(relativePath))
		if err != nil {
			return err
		}
		kind := "file"
		if info.IsDir() {
			kind = "directory"
		}
		entry := sourceEntry{Entry: Entry{
			ID: entryID(normalizedPath), Path: normalizedPath, Kind: kind,
			Size: info.Size(), ModifiedAt: formatModifiedAt(info.ModTime().UnixNano()), MediaHint: mediaHint(normalizedPath),
		}, rawName: filepath.ToSlash(relativePath), modifiedNanos: info.ModTime().UnixNano()}
		if kind == "directory" {
			entry.Size = 0
			entry.MediaHint = ""
		}
		return checkAndAppend(&items, seen, entry, &total, source.limits)
	})
	if err != nil {
		return err
	}
	sortEntries(items)
	rootMeta := fmt.Sprintf("%d:%d", rootInfo.Size(), rootInfo.ModTime().UnixNano())
	source.items = items
	source.total = total
	source.rootMeta = rootMeta
	source.fingerprintValue = fingerprintEntries("directory", rootMeta, items)
	source.byID = make(map[string]sourceEntry, len(items))
	for _, entry := range items {
		source.byID[entry.ID] = entry
	}
	return nil
}

func (source *directorySource) kind() string           { return "directory" }
func (source *directorySource) displayPath() string    { return source.root }
func (source *directorySource) displayName() string    { return filepath.Base(source.root) }
func (source *directorySource) entries() []sourceEntry { return source.items }
func (source *directorySource) totalBytes() int64      { return source.total }
func (source *directorySource) fingerprint() string    { return source.fingerprintValue }
func (source *directorySource) close() error           { return nil }

func (source *directorySource) verify() error {
	current, err := newDirectorySource(source.root, source.limits)
	if err != nil {
		return err
	}
	if current.fingerprint() != source.fingerprint() {
		return sourceError("source-changed", "directory inventory changed")
	}
	return nil
}

func (source *directorySource) open(entryIDValue string) (io.ReadCloser, error) {
	entry, ok := source.byID[entryIDValue]
	if !ok {
		return nil, sourceError("source-entry-not-found", "%s", entryIDValue)
	}
	if entry.Kind != "file" {
		return nil, sourceError("not-regular-entry", "entry is not a regular file")
	}
	if err := ensureDirectoryPathUnchanged(source.root, entry); err != nil {
		return nil, err
	}
	file, err := secureOpenDirectoryFile(source.root, entry.rawName)
	if err != nil {
		return nil, sourceError("source-unavailable", "entry is unavailable")
	}
	info, err := file.Stat()
	if err != nil {
		file.Close()
		return nil, sourceError("source-unavailable", "entry is unavailable")
	}
	if !info.Mode().IsRegular() || info.Size() != entry.Size || info.ModTime().UnixNano() != entry.modifiedNanos {
		file.Close()
		return nil, sourceError("source-changed", "entry metadata changed")
	}
	return file, nil
}

func ensureDirectoryPathUnchanged(root string, entry sourceEntry) error {
	current := root
	parts := strings.Split(entry.rawName, "/")
	for _, part := range parts {
		current = filepath.Join(current, part)
		info, err := os.Lstat(current)
		if err != nil {
			return sourceError("source-changed", "entry metadata changed")
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return sourceError("source-changed", "entry metadata changed")
		}
	}
	info, err := os.Lstat(current)
	if err != nil || !info.Mode().IsRegular() || info.Size() != entry.Size || info.ModTime().UnixNano() != entry.modifiedNanos {
		return sourceError("source-changed", "entry metadata changed")
	}
	return nil
}
