// Package filewatcher provides a lightweight live vault change watcher.
package filewatcher

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/verstak/verstak-desktop/internal/core/events"
)

const defaultInterval = 750 * time.Millisecond

type entryKind string

const (
	entryFile    entryKind = "file"
	entryFolder  entryKind = "folder"
	entrySymlink entryKind = "symlink"
	entryUnknown entryKind = "unknown"
)

type snapshotEntry struct {
	kind    entryKind
	size    int64
	modTime time.Time
}

// Service polls an open vault and publishes file.changed events for external changes.
type Service struct {
	bus      *events.Bus
	interval time.Duration

	mu       sync.Mutex
	root     string
	cancel   chan struct{}
	done     chan struct{}
	current  map[string]snapshotEntry
	onChange func()
}

// SetOnChange installs a lightweight notification used by core services that
// need a debounced reconciliation after the watcher has observed a change.
func (s *Service) SetOnChange(callback func()) {
	if s == nil {
		return
	}
	s.mu.Lock()
	s.onChange = callback
	s.mu.Unlock()
}

// NewService creates a watcher. The interval parameter is mainly for tests.
func NewService(bus *events.Bus, interval time.Duration) *Service {
	if interval <= 0 {
		interval = defaultInterval
	}
	return &Service{bus: bus, interval: interval}
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
		return
	}
	s.mu.Lock()
	prev := s.current
	s.current = next
	callback := s.onChange
	s.mu.Unlock()
	changed := false
	for path, entry := range next {
		old, ok := prev[path]
		if !ok {
			s.publish(path, "external.create", entry.kind)
			changed = true
			continue
		}
		if entry.kind == entryFile && (entry.size != old.size || !entry.modTime.Equal(old.modTime)) {
			s.publish(path, "external.update", entry.kind)
			changed = true
		}
	}
	for path, entry := range prev {
		if _, ok := next[path]; !ok {
			s.publish(path, "external.delete", entry.kind)
			changed = true
		}
	}
	if changed && callback != nil {
		callback()
	}
}

func (s *Service) publish(path, operation string, kind entryKind) {
	if s.bus == nil {
		return
	}
	s.bus.Publish(events.Event{
		Name:      "file.changed",
		Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
		Payload: map[string]interface{}{
			"path":              path,
			"title":             path,
			"operation":         operation,
			"type":              string(kind),
			"workspaceRootPath": workspaceRoot(path),
			"external":          true,
		},
	})
}

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
		if isReserved(rel) {
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

func kindFromInfo(info fs.FileInfo) entryKind {
	if info.Mode()&os.ModeSymlink != 0 {
		return entrySymlink
	}
	if info.IsDir() {
		return entryFolder
	}
	if info.Mode().IsRegular() {
		return entryFile
	}
	return entryUnknown
}

func isReserved(rel string) bool {
	for _, segment := range strings.Split(filepath.ToSlash(rel), "/") {
		if strings.EqualFold(segment, ".verstak") {
			return true
		}
	}
	return false
}

func workspaceRoot(path string) string {
	path = strings.Trim(strings.TrimSpace(filepath.ToSlash(path)), "/")
	if path == "" {
		return ""
	}
	if idx := strings.Index(path, "/"); idx >= 0 {
		return path[:idx]
	}
	return path
}
