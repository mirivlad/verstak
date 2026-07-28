// Package diagnostics writes down what a bug report needs and nothing a user
// would not want to send.
//
// What goes in: which build this is, what the machine is, which plugins loaded
// and which did not, whether sync is configured and when it last ran, and the
// tail of this run's log. What stays out: the contents of the vault, note and
// file names, secrets, sync tokens and API keys. The vault's own path is
// included -- it names a folder, and "which vault" is usually the question.
package diagnostics

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"
)

// PluginState is one plugin as the report should describe it.
type PluginState struct {
	ID      string
	Name    string
	Version string
	Source  string
	Status  string
	Enabled bool
	Error   string
}

// Input is everything the report is built from. The caller gathers it; this
// package decides what is safe to write and how it reads.
type Input struct {
	AppVersion  string
	Commit      string
	BuiltAt     string
	Language    string
	VaultPath   string
	VaultOpen   bool
	SyncServer  string
	SyncDevice  string
	LastSyncAt  string
	LastWarning string
	Plugins     []PluginState
	LogPath     string
	LogTail     string
}

// LogTailBytes bounds how much of the log goes into a report. Enough to cover
// startup and whatever went wrong after it, small enough to paste into a
// message.
const LogTailBytes = 64 * 1024

// ReadLogTail returns the last LogTailBytes of a log file, starting at a line
// boundary so the report does not open mid-word.
func ReadLogTail(path string) string {
	if strings.TrimSpace(path) == "" {
		return ""
	}
	file, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return ""
	}
	size := info.Size()
	offset := int64(0)
	if size > LogTailBytes {
		offset = size - LogTailBytes
	}
	if _, err := file.Seek(offset, 0); err != nil {
		return ""
	}
	data := make([]byte, size-offset)
	read, err := file.Read(data)
	if err != nil && read == 0 {
		return ""
	}
	text := string(data[:read])
	if offset > 0 {
		if cut := strings.IndexByte(text, '\n'); cut >= 0 {
			text = text[cut+1:]
		}
	}
	return text
}

// Render builds the report. It is plain text on purpose: a user has to be able
// to read what they are about to send.
func Render(input Input, now time.Time) string {
	var out strings.Builder
	line := func(format string, args ...interface{}) {
		fmt.Fprintf(&out, format+"\n", args...)
	}

	line("Verstak diagnostics")
	line("collected: %s", now.UTC().Format(time.RFC3339))
	line("")
	line("## Build")
	line("version:  %s", orUnknown(input.AppVersion))
	line("commit:   %s", orUnknown(input.Commit))
	line("built:    %s", orUnknown(input.BuiltAt))
	line("os:       %s/%s", runtime.GOOS, runtime.GOARCH)
	line("go:       %s", runtime.Version())
	line("language: %s", orUnknown(input.Language))
	line("")
	line("## Vault")
	line("path: %s", orNone(input.VaultPath))
	line("open: %v", input.VaultOpen)
	line("")
	line("## Sync")
	// The server address says whether sync is configured and where to look. The
	// device token and API key are what would let somebody else use it, and are
	// never written.
	line("server:       %s", orNone(input.SyncServer))
	line("device:       %s", orNone(input.SyncDevice))
	line("last sync:    %s", orNone(input.LastSyncAt))
	line("last warning: %s", orNone(input.LastWarning))
	line("")
	line("## Plugins (%d)", len(input.Plugins))
	plugins := append([]PluginState(nil), input.Plugins...)
	sort.Slice(plugins, func(left, right int) bool { return plugins[left].ID < plugins[right].ID })
	for _, item := range plugins {
		state := item.Status
		if !item.Enabled {
			state += ", disabled"
		}
		line("%-28s %-10s %-9s %s", item.ID, orUnknown(item.Version), orUnknown(item.Source), state)
		if strings.TrimSpace(item.Error) != "" {
			line("%-28s   error: %s", "", item.Error)
		}
	}
	line("")
	line("## Log")
	line("file: %s", orNone(input.LogPath))
	if strings.TrimSpace(input.LogTail) == "" {
		line("(no log for this session)")
		return out.String()
	}
	line("last %d bytes:", LogTailBytes)
	line("")
	out.WriteString(input.LogTail)
	if !strings.HasSuffix(input.LogTail, "\n") {
		out.WriteString("\n")
	}
	return out.String()
}

// Write puts a report next to the logs and returns its path.
func Write(dir string, report string, now time.Time) (string, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	path := filepath.Join(dir, fmt.Sprintf("verstak-diagnostics-%s.txt", now.Format("2006-01-02-150405")))
	if err := os.WriteFile(path, []byte(report), 0o600); err != nil {
		return "", err
	}
	return path, nil
}

func orUnknown(value string) string {
	if strings.TrimSpace(value) == "" {
		return "unknown"
	}
	return value
}

func orNone(value string) string {
	if strings.TrimSpace(value) == "" {
		return "—"
	}
	return value
}
