// Package filewatcher provides a lightweight live vault change watcher.
//
// It classifies filesystem events as content (inside a workspace) or
// structural (marker files, directory creation outside workspaces) and
// notifies core services accordingly.
package filewatcher

import (
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/verstak/verstak-desktop/internal/core/events"
)

const defaultInterval = 750 * time.Millisecond

// ── Types ────────────────────────────────────────────────────────────────────

// EntryKind classifies a filesystem entry.
type EntryKind string

const (
	EntryFile    EntryKind = "file"
	EntryFolder  EntryKind = "folder"
	EntrySymlink EntryKind = "symlink"
	EntryUnknown EntryKind = "unknown"
)

// ChangeClass separates content from structural changes.
type ChangeClass string

const (
	ChangeContent    ChangeClass = "content"
	ChangeStructural ChangeClass = "structural"
)

// Change describes a single filesystem change.
type Change struct {
	Path      string
	Operation string // "external.create" | "external.update" | "external.delete"
	Kind      EntryKind
	Class     ChangeClass
}

// ResolveWorkspaceFunc resolves a relative vault path to its workspace.
// Returns workspaceID, workspaceRootPath, found.
type ResolveWorkspaceFunc func(relPath string) (workspaceID, workspaceRootPath string, found bool)

type snapshotEntry struct {
	kind    EntryKind
	size    int64
	modTime time.Time
}

// ── Service ──────────────────────────────────────────────────────────────────

// Service polls an open vault and publishes events for external changes.
type Service struct {
	bus      *events.Bus
	interval time.Duration

	mu      sync.Mutex
	root    string
	cancel  chan struct{}
	done    chan struct{}
	current map[string]snapshotEntry

	// Callbacks.
	onChange           func()               // content changes (sync scan, etc.)
	onStructuralChange func()               // triggers tree reconciliation
	resolveWorkspace   ResolveWorkspaceFunc // resolves path → workspace

	// scan error tracking.
	scanErrors int
	lastErr    error
	dirty      bool // structural dirty flag after scan error
}

// NewService creates a watcher. interval=0 uses default.
func NewService(bus *events.Bus, interval time.Duration) *Service {
	if interval <= 0 {
		interval = defaultInterval
	}
	return &Service{bus: bus, interval: interval}
}

// SetOnStructuralChange sets the callback for structural changes.
func (s *Service) SetOnStructuralChange(callback func()) {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.onStructuralChange = callback
}

// SetOnChange sets the content change callback.
func (s *Service) SetOnChange(callback func()) {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.onChange = callback
}

// SetWorkspaceResolver sets the function used to resolve a path to its workspace.
func (s *Service) SetWorkspaceResolver(fn ResolveWorkspaceFunc) {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.resolveWorkspace = fn
}

// Start begins watching root. Any previous watch is stopped first.
func (s *Service) Start(root string) error {
	if s == nil {
		return fmt.Errorf("file watcher is nil")
	}
	if root == "" {
		return fmt.Errorf("file watcher root is empty")
	}
	root, err := filepath.Abs(root)
	if err != nil {
		return err
	}
	info, err := os.Stat(root)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("file watcher root is not a directory: %s", root)
	}

	initial, err := scan(root)
	if err != nil {
		return err
	}

	s.Stop()

	s.mu.Lock()
	s.root = root
	s.current = initial
	s.cancel = make(chan struct{})
	s.done = make(chan struct{})
	s.scanErrors = 0
	s.dirty = false
	cancel := s.cancel
	done := s.done
	s.mu.Unlock()

	go s.loop(root, cancel, done)
	return nil
}

// Stop stops the active watcher.
func (s *Service) Stop() {
	if s == nil {
		return
	}
	s.mu.Lock()
	cancel := s.cancel
	done := s.done
	if cancel == nil {
		s.mu.Unlock()
		return
	}
	s.cancel = nil
	s.done = nil
	s.root = ""
	s.current = nil
	close(cancel)
	s.mu.Unlock()
	<-done
}

// RefreshBaseline rescans the current root and replaces the low-level
// snapshot without publishing any events. Safe when watcher is not running.
func (s *Service) RefreshBaseline() error {
	if s == nil {
		return nil
	}
	s.mu.Lock()
	root := s.root
	s.mu.Unlock()
	if root == "" {
		return nil
	}
	fresh, err := scan(root)
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	// Only replace current if root hasn't changed (e.g. watcher was stopped).
	if s.root == root {
		s.current = fresh
		s.scanErrors = 0
		s.dirty = false
	}
	return nil
}

// Root returns the current watch root.
func (s *Service) Root() string {
	if s == nil {
		return ""
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.root
}

// ── Poll loop ────────────────────────────────────────────────────────────────

func (s *Service) loop(root string, cancel <-chan struct{}, done chan<- struct{}) {
	defer close(done)
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			s.poll(root)
		case <-cancel:
			return
		}
	}
}

func (s *Service) poll(root string) {
	next, err := scan(root)
	if err != nil {
		s.mu.Lock()
		s.scanErrors++
		s.lastErr = err
		s.dirty = true
		log.Printf("[filewatcher] scan error: %v (count=%d)", err, s.scanErrors)
		s.mu.Unlock()
		return
	}

	s.mu.Lock()
	prev := s.current
	s.current = next
	resolve := s.resolveWorkspace
	structuralCB := s.onStructuralChange
	contentCB := s.onChange
	// Clear error state on successful scan.
	if s.dirty {
		s.dirty = false
		s.scanErrors = 0
	}
	s.mu.Unlock()

	structural := false
	content := false

	// Detect created and updated entries.
	for path, entry := range next {
		old, existed := prev[path]
		if !existed {
			cls := s.classify(path, entry.kind, resolve)
			if cls == ChangeStructural {
				structural = true
				s.publishStructuralMarker(path, "external.create")
			} else {
				content = true
				s.publishFileChange(path, "external.create", entry.kind, resolve)
			}
			continue
		}
		if entry.kind == EntryFile && (entry.size != old.size || !entry.modTime.Equal(old.modTime)) {
			cls := s.classify(path, entry.kind, resolve)
			if cls == ChangeStructural {
				structural = true
				s.publishStructuralMarker(path, "external.update")
			} else {
				content = true
				s.publishFileChange(path, "external.update", entry.kind, resolve)
			}
		}
	}

	// Detect deleted entries.
	for path, entry := range prev {
		if _, exists := next[path]; !exists {
			cls := s.classify(path, entry.kind, resolve)
			if cls == ChangeStructural {
				structural = true
				s.publishStructuralMarker(path, "external.delete")
			} else {
				content = true
				s.publishFileChange(path, "external.delete", entry.kind, resolve)
			}
		}
	}

	if structural && structuralCB != nil {
		structuralCB()
	}
	if content && contentCB != nil {
		contentCB()
	}

	// If we had errors and this scan succeeded, trigger reconciliation
	// to catch up on any missed structural changes.
	s.mu.Lock()
	wasDirty := s.dirty
	s.mu.Unlock()
	if wasDirty && structuralCB != nil {
		structuralCB()
	}
}

// ── Classification ───────────────────────────────────────────────────────────

func (s *Service) classify(relPath string, kind EntryKind, resolve ResolveWorkspaceFunc) ChangeClass {
	relPath = filepath.ToSlash(relPath)
	// Marker files are always structural.
	if isMarkerPath(relPath) {
		return ChangeStructural
	}
	// Directory creation/deletion at vault level or inside folders (not workspaces)
	// is structural.
	if kind == EntryFolder {
		if resolve != nil {
			_, _, inWS := resolve(relPath)
			if inWS {
				return ChangeContent
			}
		}
		// If no resolver or outside workspace — structural.
		if resolve == nil {
			// Without resolver, check if this is a top-level directory.
			// Top-level dirs are structural, nested content can't be determined.
			if !strings.Contains(relPath, "/") {
				return ChangeStructural
			}
			return ChangeContent
		}
		return ChangeStructural
	}
	// Files inside workspaces are content.
	if resolve != nil {
		_, _, inWS := resolve(relPath)
		if inWS {
			return ChangeContent
		}
	}
	// Files at vault level or in organizational folders.
	return ChangeContent
}

// ── Publishing ───────────────────────────────────────────────────────────────

func (s *Service) publishFileChange(path, operation string, kind EntryKind, resolve ResolveWorkspaceFunc) {
	if s.bus == nil {
		return
	}
	payload := map[string]interface{}{
		"path":      path,
		"title":     path,
		"operation": operation,
		"type":      string(kind),
		"external":  true,
	}
	// Resolve workspace context.
	if resolve != nil {
		wsID, wsRootPath, found := resolve(path)
		if found {
			payload["workspaceId"] = wsID
			payload["workspaceRootPath"] = wsRootPath
		}
	} else {
		payload["workspaceRootPath"] = legacyWorkspaceRoot(path)
	}
	s.bus.Publish(events.Event{
		Name:      "file.changed",
		Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
		Payload:   payload,
	})
}

func (s *Service) publishStructuralMarker(path, operation string) {
	// Marker file changes are handled by the tree reconciler,
	// not published as user-facing file.changed events.
	// The OnStructuralChange callback triggers reconciliation.
	_ = path
	_ = operation
}

// ── Scan ─────────────────────────────────────────────────────────────────────

func scan(root string) (map[string]snapshotEntry, error) {
	out := make(map[string]snapshotEntry)
	err := filepath.WalkDir(root, func(path string, dirEntry fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if path == root {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return nil
		}
		rel = filepath.ToSlash(rel)
		// New policy: observe marker files, skip internal .verstak paths.
		if isIgnoredInternal(rel) {
			if dirEntry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		info, err := dirEntry.Info()
		if err != nil {
			return nil
		}
		out[rel] = snapshotEntry{
			kind:    kindFromInfo(info),
			size:    info.Size(),
			modTime: info.ModTime(),
		}
		return nil
	})
	return out, err
}

func kindFromInfo(info fs.FileInfo) EntryKind {
	if info.Mode()&os.ModeSymlink != 0 {
		return EntrySymlink
	}
	if info.IsDir() {
		return EntryFolder
	}
	if info.Mode().IsRegular() {
		return EntryFile
	}
	return EntryUnknown
}

// ── .verstak policy ──────────────────────────────────────────────────────────

// isMarkerPath returns true for structural marker files.
func isMarkerPath(rel string) bool {
	rel = filepath.ToSlash(rel)
	return strings.HasSuffix(rel, "/.verstak/folder.json") ||
		strings.HasSuffix(rel, "/.verstak/workspace.json")
}

// isIgnoredInternal returns true for vault-internal paths that should
// never appear in the scan.
func isIgnoredInternal(rel string) bool {
	rel = filepath.ToSlash(rel)
	segments := strings.Split(rel, "/")

	// Find .verstak segment position.
	vi := -1
	for i, seg := range segments {
		if strings.EqualFold(seg, ".verstak") {
			vi = i
			break
		}
	}
	if vi < 0 {
		return false // not in .verstak
	}
	// Exactly a marker file → NOT ignored (we want to observe it).
	if isMarkerPath(rel) {
		return false
	}
	// If this is the .verstak directory entry itself, keep it (to detect
	// creation/deletion) but we need to scan inside for markers.
	// The directory entry is the path up to .verstak, e.g. "Project/.verstak".
	// Check if there's a segment after .verstak.
	if vi == len(segments)-1 {
		// This is the .verstak directory entry — don't skip, we need to scan inside.
		return false
	}
	// Segment after .verstak.
	next := segments[vi+1]
	// Ignore internal directories.
	switch next {
	case "cache", "trash", "sync", "workspaces":
		return true
	}
	// Ignore temp/backup files.
	if strings.HasSuffix(next, ".tmp") || strings.HasSuffix(next, "~") || strings.HasPrefix(next, ".") && strings.HasSuffix(next, ".swp") {
		return true
	}
	// Allow other files inside .verstak (e.g. folder.json, workspace.json)
	// — they are already handled by isMarkerPath above, so any other
	// .verstak content is ignored.
	if vi == len(segments)-2 {
		// This is a file directly inside .verstak that isn't a marker.
		return true
	}
	return true
}

// ── Legacy helpers ───────────────────────────────────────────────────────────

func legacyWorkspaceRoot(path string) string {
	path = strings.Trim(strings.TrimSpace(filepath.ToSlash(path)), "/")
	if path == "" {
		return ""
	}
	if idx := strings.Index(path, "/"); idx >= 0 {
		return path[:idx]
	}
	return path
}
