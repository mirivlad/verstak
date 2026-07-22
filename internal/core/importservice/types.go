package importservice

import (
	"fmt"
	"io"
	"time"
)

const (
	MaxEntries              = 250_000
	MaxTotalBytes     int64 = 20 << 30
	MaxEntryBytes     int64 = 2 << 30
	MaxTextBytes      int64 = 16 << 20
	MaxExpansionRatio int64 = 1_000
	DefaultPageSize         = 500
	DefaultSessionTTL       = 30 * time.Minute
)

type Options struct {
	MaxEntries     int
	MaxTotalBytes  int64
	MaxEntryBytes  int64
	MaxTextBytes   int64
	ExpansionRatio int64
	PageSize       int
	SessionTTL     time.Duration
	Now            func() time.Time
}

type SourceSession struct {
	SourceHandle string `json:"sourceHandle"`
	Kind         string `json:"kind"`
	DisplayPath  string `json:"displayPath"`
	DisplayName  string `json:"displayName"`
	Fingerprint  string `json:"fingerprint"`
	EntryCount   int    `json:"entryCount"`
	TotalBytes   int64  `json:"totalBytes"`
}

type Entry struct {
	ID         string `json:"id"`
	Path       string `json:"path"`
	Kind       string `json:"kind"`
	Size       int64  `json:"size"`
	ModifiedAt string `json:"modifiedAt"`
	MediaHint  string `json:"mediaHint"`
}

type EntryPage struct {
	Entries     []Entry `json:"entries"`
	NextCursor  string  `json:"nextCursor"`
	Fingerprint string  `json:"fingerprint"`
}

type sourceEntry struct {
	Entry
	rawName        string
	modifiedNanos  int64
	compressedSize int64
	checksum       uint32
}

type indexedSource interface {
	kind() string
	displayPath() string
	displayName() string
	entries() []sourceEntry
	totalBytes() int64
	fingerprint() string
	open(entryID string) (io.ReadCloser, error)
	verify() error
	close() error
}

type limits struct {
	maxEntries     int
	maxTotalBytes  int64
	maxEntryBytes  int64
	maxTextBytes   int64
	expansionRatio int64
	pageSize       int
	sessionTTL     time.Duration
}

func resolveOptions(options Options) (limits, func() time.Time) {
	resolved := limits{
		maxEntries: MaxEntries, maxTotalBytes: MaxTotalBytes, maxEntryBytes: MaxEntryBytes,
		maxTextBytes: MaxTextBytes, expansionRatio: MaxExpansionRatio,
		pageSize: DefaultPageSize, sessionTTL: DefaultSessionTTL,
	}
	if options.MaxEntries > 0 {
		resolved.maxEntries = options.MaxEntries
	}
	if options.MaxTotalBytes > 0 {
		resolved.maxTotalBytes = options.MaxTotalBytes
	}
	if options.MaxEntryBytes > 0 {
		resolved.maxEntryBytes = options.MaxEntryBytes
	}
	if options.MaxTextBytes > 0 {
		resolved.maxTextBytes = options.MaxTextBytes
	}
	if options.ExpansionRatio > 0 {
		resolved.expansionRatio = options.ExpansionRatio
	}
	if options.PageSize > 0 {
		resolved.pageSize = options.PageSize
	}
	if options.SessionTTL > 0 {
		resolved.sessionTTL = options.SessionTTL
	}
	now := options.Now
	if now == nil {
		now = time.Now
	}
	return resolved, now
}

func sourceError(code, format string, args ...any) error {
	if format == "" {
		return fmt.Errorf("%s", code)
	}
	return fmt.Errorf("%s: %s", code, fmt.Sprintf(format, args...))
}

func formatModifiedAt(nanos int64) string {
	if nanos <= 0 {
		return ""
	}
	return time.Unix(0, nanos).UTC().Format(time.RFC3339Nano)
}
