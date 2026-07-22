package importservice

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"strings"
)

type archiveSource struct {
	archivePath      string
	archiveKind      string
	outerSize        int64
	outerModified    int64
	items            []sourceEntry
	byID             map[string]sourceEntry
	total            int64
	fingerprintValue string
	limits           limits
}

func newArchiveSource(selectedPath string, limits limits) (*archiveSource, error) {
	absPath, err := filepath.Abs(selectedPath)
	if err != nil {
		return nil, sourceError("invalid-source", "could not resolve selected archive")
	}
	info, err := os.Lstat(absPath)
	if err != nil {
		return nil, sourceError("source-unavailable", "selected archive is unavailable")
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return nil, sourceError("unsupported-source-entry", "archive is not a regular file")
	}
	lower := strings.ToLower(absPath)
	kind := ""
	switch {
	case strings.HasSuffix(lower, ".zip"):
		kind = "zip"
	case strings.HasSuffix(lower, ".tar.gz"), strings.HasSuffix(lower, ".tgz"):
		kind = "tar.gz"
	case strings.HasSuffix(lower, ".tar"):
		kind = "tar"
	default:
		return nil, sourceError("unsupported-archive", "supported formats: .zip, .tar, .tar.gz, .tgz")
	}
	source := &archiveSource{archivePath: absPath, archiveKind: kind, outerSize: info.Size(), outerModified: info.ModTime().UnixNano(), limits: limits}
	if err := source.index(); err != nil {
		return nil, err
	}
	return source, nil
}

func (source *archiveSource) index() error {
	var items []sourceEntry
	var total int64
	var err error
	if source.archiveKind == "zip" {
		items, total, err = source.indexZIP()
	} else {
		items, total, err = source.indexTAR()
	}
	if err != nil {
		return err
	}
	if source.archiveKind == "tar.gz" && exceedsExpansionRatio(total, source.outerSize, source.limits.expansionRatio) {
		return sourceError("source-limit-exceeded", "archive expansion ratio exceeds %d:1", source.limits.expansionRatio)
	}
	sortEntries(items)
	source.items = items
	source.total = total
	source.byID = make(map[string]sourceEntry, len(items))
	for _, entry := range items {
		source.byID[entry.ID] = entry
	}
	meta := fmt.Sprintf("%s:%d:%d", source.archiveKind, source.outerSize, source.outerModified)
	source.fingerprintValue = fingerprintEntries("archive", meta, items)
	return nil
}

func (source *archiveSource) indexZIP() ([]sourceEntry, int64, error) {
	reader, err := zip.OpenReader(source.archivePath)
	if err != nil {
		return nil, 0, sourceError("invalid-archive", "could not read ZIP archive")
	}
	defer reader.Close()
	items := make([]sourceEntry, 0, len(reader.File))
	seen := make(map[string]string)
	var total int64
	for _, file := range reader.File {
		if isArchiveRootEntry(file.Name) {
			continue
		}
		normalizedPath, err := normalizeSourcePath(file.Name)
		if err != nil {
			return nil, 0, err
		}
		mode := file.Mode()
		kind := "file"
		if file.FileInfo().IsDir() {
			kind = "directory"
		}
		if kind == "file" && !mode.IsRegular() {
			return nil, 0, sourceError("unsupported-source-entry", "links and special entries are not supported")
		}
		if kind == "directory" && !mode.IsDir() {
			return nil, 0, sourceError("unsupported-source-entry", "links and special entries are not supported")
		}
		if file.UncompressedSize64 > uint64(source.limits.maxEntryBytes) || file.CompressedSize64 > math.MaxInt64 {
			return nil, 0, sourceError("source-limit-exceeded", "entry size is outside supported bounds")
		}
		size := int64(file.UncompressedSize64)
		compressed := int64(file.CompressedSize64)
		if kind == "file" && exceedsExpansionRatio(size, compressed, source.limits.expansionRatio) {
			return nil, 0, sourceError("source-limit-exceeded", "archive expansion ratio exceeds %d:1", source.limits.expansionRatio)
		}
		modified := file.Modified.UnixNano()
		entry := sourceEntry{Entry: Entry{ID: entryID(normalizedPath), Path: normalizedPath, Kind: kind, Size: size,
			ModifiedAt: formatModifiedAt(modified), MediaHint: mediaHint(normalizedPath)}, rawName: file.Name,
			modifiedNanos: modified, compressedSize: compressed, checksum: file.CRC32}
		if kind == "directory" {
			entry.Size = 0
			entry.MediaHint = ""
		}
		if err := checkAndAppend(&items, seen, entry, &total, source.limits); err != nil {
			return nil, 0, err
		}
	}
	return items, total, nil
}

func exceedsExpansionRatio(uncompressed, compressed, ratio int64) bool {
	if uncompressed <= 0 {
		return false
	}
	if compressed <= 0 {
		return true
	}
	if compressed > math.MaxInt64/ratio {
		return false
	}
	return uncompressed > compressed*ratio
}

func (source *archiveSource) indexTAR() ([]sourceEntry, int64, error) {
	reader, closeReader, err := source.openTARReader()
	if err != nil {
		return nil, 0, err
	}
	defer closeReader()
	items := make([]sourceEntry, 0)
	seen := make(map[string]string)
	var total int64
	for {
		header, err := reader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, 0, sourceError("invalid-archive", "could not read TAR archive")
		}
		if isArchiveRootEntry(header.Name) {
			continue
		}
		normalizedPath, err := normalizeSourcePath(header.Name)
		if err != nil {
			return nil, 0, err
		}
		kind := ""
		switch header.Typeflag {
		case tar.TypeReg, tar.TypeRegA:
			kind = "file"
		case tar.TypeDir:
			kind = "directory"
		default:
			return nil, 0, sourceError("unsupported-source-entry", "links and special entries are not supported")
		}
		entry := sourceEntry{Entry: Entry{ID: entryID(normalizedPath), Path: normalizedPath, Kind: kind, Size: header.Size,
			ModifiedAt: formatModifiedAt(header.ModTime.UnixNano()), MediaHint: mediaHint(normalizedPath)}, rawName: header.Name,
			modifiedNanos: header.ModTime.UnixNano()}
		if kind == "directory" {
			entry.Size = 0
			entry.MediaHint = ""
		}
		if err := checkAndAppend(&items, seen, entry, &total, source.limits); err != nil {
			return nil, 0, err
		}
	}
	return items, total, nil
}

func (source *archiveSource) kind() string           { return "archive" }
func (source *archiveSource) displayPath() string    { return source.archivePath }
func (source *archiveSource) displayName() string    { return filepath.Base(source.archivePath) }
func (source *archiveSource) entries() []sourceEntry { return source.items }
func (source *archiveSource) totalBytes() int64      { return source.total }
func (source *archiveSource) fingerprint() string    { return source.fingerprintValue }
func (source *archiveSource) close() error           { return nil }

func (source *archiveSource) verifyOuter() error {
	info, err := os.Lstat(source.archivePath)
	if err != nil || !info.Mode().IsRegular() || info.Size() != source.outerSize || info.ModTime().UnixNano() != source.outerModified {
		return sourceError("source-changed", "archive changed")
	}
	return nil
}

func (source *archiveSource) verify() error {
	if err := source.verifyOuter(); err != nil {
		return err
	}
	current, err := newArchiveSource(source.archivePath, source.limits)
	if err != nil {
		return err
	}
	if current.fingerprint() != source.fingerprint() {
		return sourceError("source-changed", "archive inventory changed")
	}
	return nil
}

func (source *archiveSource) open(entryIDValue string) (io.ReadCloser, error) {
	entry, ok := source.byID[entryIDValue]
	if !ok {
		return nil, sourceError("source-entry-not-found", "%s", entryIDValue)
	}
	if entry.Kind != "file" {
		return nil, sourceError("not-regular-entry", "entry is not a regular file")
	}
	if err := source.verifyOuter(); err != nil {
		return nil, err
	}
	if source.archiveKind == "zip" {
		return source.openZIPEntry(entry)
	}
	return source.openTAREntry(entry)
}

type zipEntryReadCloser struct {
	io.ReadCloser
	archive *zip.ReadCloser
}

func (reader *zipEntryReadCloser) Close() error {
	entryErr := reader.ReadCloser.Close()
	archiveErr := reader.archive.Close()
	if entryErr != nil {
		return entryErr
	}
	return archiveErr
}

func (source *archiveSource) openZIPEntry(entry sourceEntry) (io.ReadCloser, error) {
	reader, err := zip.OpenReader(source.archivePath)
	if err != nil {
		return nil, sourceError("invalid-archive", "could not read ZIP archive")
	}
	for _, file := range reader.File {
		if file.Name != entry.rawName {
			continue
		}
		if int64(file.UncompressedSize64) != entry.Size || file.CRC32 != entry.checksum || !file.Mode().IsRegular() {
			reader.Close()
			return nil, sourceError("source-changed", "archive entry metadata changed")
		}
		opened, err := file.Open()
		if err != nil {
			reader.Close()
			return nil, sourceError("invalid-archive", "could not read ZIP entry")
		}
		return &zipEntryReadCloser{ReadCloser: opened, archive: reader}, nil
	}
	reader.Close()
	return nil, sourceError("source-changed", "archive entry metadata changed")
}

type multiReadCloser struct {
	io.Reader
	closers []io.Closer
}

func (reader *multiReadCloser) Close() error {
	var first error
	for _, closer := range reader.closers {
		if err := closer.Close(); err != nil && first == nil {
			first = err
		}
	}
	return first
}

func (source *archiveSource) openTARReader() (*tar.Reader, func() error, error) {
	file, err := os.Open(source.archivePath)
	if err != nil {
		return nil, nil, sourceError("source-unavailable", "selected archive is unavailable")
	}
	if source.archiveKind != "tar.gz" {
		return tar.NewReader(file), file.Close, nil
	}
	gz, err := gzip.NewReader(file)
	if err != nil {
		file.Close()
		return nil, nil, sourceError("invalid-archive", "could not read GZIP stream")
	}
	return tar.NewReader(gz), func() error {
		gzErr := gz.Close()
		fileErr := file.Close()
		if gzErr != nil {
			return gzErr
		}
		return fileErr
	}, nil
}

func (source *archiveSource) openTAREntry(entry sourceEntry) (io.ReadCloser, error) {
	file, err := os.Open(source.archivePath)
	if err != nil {
		return nil, sourceError("source-unavailable", "selected archive is unavailable")
	}
	var input io.Reader = file
	closers := []io.Closer{file}
	if source.archiveKind == "tar.gz" {
		gz, err := gzip.NewReader(file)
		if err != nil {
			file.Close()
			return nil, sourceError("invalid-archive", "could not read GZIP stream")
		}
		input = gz
		closers = []io.Closer{gz, file}
	}
	reader := tar.NewReader(input)
	for {
		header, nextErr := reader.Next()
		if nextErr == io.EOF {
			break
		}
		if nextErr != nil {
			(&multiReadCloser{closers: closers}).Close()
			return nil, sourceError("invalid-archive", "could not read TAR archive")
		}
		if header.Name != entry.rawName {
			continue
		}
		if header.Size != entry.Size || (header.Typeflag != tar.TypeReg && header.Typeflag != tar.TypeRegA) {
			(&multiReadCloser{closers: closers}).Close()
			return nil, sourceError("source-changed", "archive entry metadata changed")
		}
		return &multiReadCloser{Reader: io.LimitReader(reader, entry.Size), closers: closers}, nil
	}
	(&multiReadCloser{closers: closers}).Close()
	return nil, sourceError("source-changed", "archive entry metadata changed")
}
