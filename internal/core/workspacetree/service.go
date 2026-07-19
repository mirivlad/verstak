package workspacetree

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/verstak/verstak-desktop/internal/core/events"
)

const (
	defaultDebounceInterval = 1500 * time.Millisecond
	defaultRescanInterval   = 5 * time.Minute
)

// Service manages the v2 workspace tree lifecycle.
type Service struct {
	mu       sync.Mutex
	vaultDir string
	eventBus *events.Bus

	// Current tree state.
	snapshot  *SemanticSnapshot
	scan      *ScanResult
	tree      *TreeSnapshot
	revision  uint64
	currentWS string

	// Debounce / rescan.
	debounceInterval time.Duration
	rescanInterval   time.Duration
	debounceTimer    *time.Timer
	stopRescan       chan struct{}
	rescanDone       chan struct{}

	// Internal mutation suppression.
	internalMutations int32 // atomic counter
	refreshRequested  int32 // atomic flag
}

// NewService creates a new workspace tree service.
func NewService(vaultDir string, bus *events.Bus) *Service {
	return &Service{
		vaultDir:         vaultDir,
		eventBus:         bus,
		debounceInterval: defaultDebounceInterval,
		rescanInterval:   defaultRescanInterval,
	}
}

// Initialize performs the initial scan and reconciliation.
func (s *Service) Initialize() error {
	return s.fullReconcile()
}

// GetTree returns the current tree snapshot.
func (s *Service) GetTree() *TreeSnapshot {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.tree == nil {
		return &TreeSnapshot{Roots: []TreeNode{}}
	}
	return s.tree
}

// GetWorkspaceByID returns a workspace by its UUID.
func (s *Service) GetWorkspaceByID(id string) (ScannedWorkspace, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.scan == nil {
		return ScannedWorkspace{}, false
	}
	ws, ok := s.scan.Workspaces[id]
	return ws, ok
}

// GetFolderByID returns a folder by its UUID.
func (s *Service) GetFolderByID(id string) (ScannedFolder, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.scan == nil {
		return ScannedFolder{}, false
	}
	f, ok := s.scan.Folders[id]
	return f, ok
}

// GetWorkspaceTreeDiagnostics returns the current warnings.
func (s *Service) GetWorkspaceTreeDiagnostics() []TreeDiagnostic {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.tree == nil {
		return nil
	}
	return s.tree.Warnings
}

// RescanWorkspaceTree triggers an immediate full reconciliation.
func (s *Service) RescanWorkspaceTree() error {
	return s.fullReconcile()
}

// SetCurrentWorkspace updates the current workspace ID without a filesystem scan.
func (s *Service) SetCurrentWorkspace(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.currentWS = id
	if s.tree != nil {
		s.tree.CurrentWorkspaceID = id
	}
}

// OnFileChanged is called by the file watcher when any file changes.
// It triggers a debounced reconciliation.
func (s *Service) OnFileChanged() {
	if s.isInternalMutation() {
		return
	}
	s.scheduleDebounce()
}

// BeginInternalMutation marks the start of a core-initiated filesystem mutation.
func (s *Service) BeginInternalMutation() {
	atomic.AddInt32(&s.internalMutations, 1)
}

// EndInternalMutationAndRefreshBaseline marks the end of a core-initiated mutation
// and triggers an immediate rescan + baseline update without publishing duplicate events.
func (s *Service) EndInternalMutationAndRefreshBaseline() {
	atomic.AddInt32(&s.internalMutations, -1)
	// Perform a silent rescan to update the baseline.
	_ = s.fullReconcile()
}

// StartRescanLoop starts periodic background rescan.
func (s *Service) StartRescanLoop() {
	s.mu.Lock()
	if s.stopRescan != nil {
		s.mu.Unlock()
		return
	}
	s.stopRescan = make(chan struct{})
	s.rescanDone = make(chan struct{})
	stop := s.stopRescan
	done := s.rescanDone
	interval := s.rescanInterval
	s.mu.Unlock()

	go func() {
		defer close(done)
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				_ = s.fullReconcile()
			case <-stop:
				return
			}
		}
	}()
}

// Stop stops background operations.
func (s *Service) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.debounceTimer != nil {
		s.debounceTimer.Stop()
		s.debounceTimer = nil
	}
	if s.stopRescan != nil {
		close(s.stopRescan)
		<-s.rescanDone
		s.stopRescan = nil
		s.rescanDone = nil
	}
}

// ── Internal methods ─────────────────────────────────────────────────────────

func (s *Service) fullReconcile() error {
	// Read previous snapshot.
	prev, err := ReadSnapshot(s.vaultDir)
	if err != nil {
		return err
	}

	// Scan filesystem (read-only).
	scan, err := Scan(s.vaultDir, prev)
	if err != nil {
		return err
	}

	// Reconcile.
	result := Reconcile(s.vaultDir, prev, scan)

	// Build tree.
	s.revision++
	tree := BuildTree(scan, s.currentWS, s.revision)

	// Write new snapshot.
	if err := WriteSnapshot(s.vaultDir, &result.Snapshot); err != nil {
		return err
	}

	// Update state.
	s.mu.Lock()
	s.snapshot = &result.Snapshot
	s.scan = scan
	s.tree = tree
	s.mu.Unlock()

	// Publish events.
	for _, evt := range result.Events {
		s.publish(evt)
	}

	return nil
}

func (s *Service) scheduleDebounce() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.debounceTimer != nil {
		s.debounceTimer.Stop()
	}
	s.debounceTimer = time.AfterFunc(s.debounceInterval, func() {
		_ = s.fullReconcile()
	})
}

func (s *Service) isInternalMutation() bool {
	return atomic.LoadInt32(&s.internalMutations) > 0
}

func (s *Service) publish(event ReconEvent) {
	if s.eventBus == nil {
		return
	}
	s.eventBus.Publish(events.Event{
		Name:      event.Name,
		Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
		Payload:   event.Payload,
	})
}
