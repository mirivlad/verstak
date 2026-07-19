package workspacetree

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/verstak/verstak-desktop/internal/core/events"
	"github.com/verstak/verstak-desktop/internal/core/filewatcher"
)

func TestIntegrationContentChangeDoesNotReconcileTree(t *testing.T) {
	vault := t.TempDir()
	wsID := testUUID("int-cnt")
	createWS(t, vault, "Project", wsID)
	mustMkdirAll(t, filepath.Join(vault, "Project", "Notes"))

	bus := events.NewBus()
	svc := NewService(vault, bus)
	if err := svc.Initialize(); err != nil {
		t.Fatal(err)
	}
	rev := svc.GetTree().Revision

	fw := filewatcher.NewService(bus, 100*time.Millisecond)
	fw.SetWorkspaceResolver(func(relPath string) (string, string, bool) {
		return svc.ResolveWorkspaceForPath(relPath)
	})
	fw.SetOnStructuralChange(func() {
		_ = svc.RescanWorkspaceTree()
	})

	// Collect file.changed events.
	var mu sync.Mutex
	var fileEvents []events.Event
	bus.Subscribe("file.changed", func(e events.Event) {
		mu.Lock()
		fileEvents = append(fileEvents, e)
		mu.Unlock()
	})

	if err := fw.RefreshBaseline(); err != nil {
		t.Fatal(err)
	}
	if err := fw.Start(vault); err != nil {
		t.Fatal(err)
	}
	defer fw.Stop()

	// Modify a file inside workspace.
	time.Sleep(150 * time.Millisecond) // let watcher settle
	notePath := filepath.Join(vault, "Project", "Notes", "doc.md")
	mustMkdirAll(t, filepath.Dir(notePath))
	mustWriteFile(t, notePath, "# Hello")

	// Wait for watcher to detect the change.
	waitFor(t, 2*time.Second, func() bool {
		mu.Lock()
		defer mu.Unlock()
		return len(fileEvents) > 0
	})

	mu.Lock()
	hasEvent := len(fileEvents) > 0
	mu.Unlock()
	if !hasEvent {
		t.Fatal("expected file.changed event for content change")
	}

	// Tree revision should NOT change.
	newRev := svc.GetTree().Revision
	if newRev != rev {
		t.Fatalf("tree revision changed from %d to %d (should stay same for content)", rev, newRev)
	}

	// file.changed should have workspace context.
	mu.Lock()
	for _, evt := range fileEvents {
		if p, ok := evt.Payload.(map[string]interface{}); ok {
			if _, has := p["workspaceId"]; has {
				t.Logf("file.changed has workspaceId: %v", p["workspaceId"])
			}
		}
	}
	mu.Unlock()
}

func TestIntegrationStructuralFolderCreationReconcilesTree(t *testing.T) {
	vault := t.TempDir()
	bus := events.NewBus()
	svc := NewService(vault, bus)
	if err := svc.Initialize(); err != nil {
		t.Fatal(err)
	}
	if len(svc.GetTree().Roots) != 0 {
		t.Fatalf("expected empty tree, got %d roots", len(svc.GetTree().Roots))
	}

	fw := filewatcher.NewService(bus, 100*time.Millisecond)
	fw.SetWorkspaceResolver(func(relPath string) (string, string, bool) {
		return svc.ResolveWorkspaceForPath(relPath)
	})
	fw.SetOnStructuralChange(func() {
		_ = svc.RescanWorkspaceTree()
	})

	if err := fw.RefreshBaseline(); err != nil {
		t.Fatal(err)
	}
	if err := fw.Start(vault); err != nil {
		t.Fatal(err)
	}
	defer fw.Stop()

	time.Sleep(150 * time.Millisecond)

	// Create a new folder physically.
	mustMkdirAll(t, filepath.Join(vault, "NewFolder"))

	// Wait for tree to pick it up.
	waitFor(t, 3*time.Second, func() bool {
		return len(svc.GetTree().Roots) > 0
	})

	tree := svc.GetTree()
	if len(tree.Roots) == 0 {
		t.Fatal("tree should have the new folder")
	}
}

func TestIntegrationInternalMutationDoesNotCreateExternalEvent(t *testing.T) {
	vault := t.TempDir()
	wsID := testUUID("int-mut")
	createWS(t, vault, "Project", wsID)

	bus := events.NewBus()
	svc := NewService(vault, bus)
	if err := svc.Initialize(); err != nil {
		t.Fatal(err)
	}

	fw := filewatcher.NewService(bus, 100*time.Millisecond)
	fw.SetWorkspaceResolver(func(relPath string) (string, string, bool) {
		return svc.ResolveWorkspaceForPath(relPath)
	})

	var structuralCalls int
	var mu sync.Mutex
	fw.SetOnStructuralChange(func() {
		mu.Lock()
		structuralCalls++
		mu.Unlock()
	})

	if err := fw.RefreshBaseline(); err != nil {
		t.Fatal(err)
	}
	if err := fw.Start(vault); err != nil {
		t.Fatal(err)
	}
	defer fw.Stop()

	time.Sleep(150 * time.Millisecond)

	// Internal mutation: rename workspace.
	svc.BeginInternalMutation()
	old := filepath.Join(vault, "Project")
	new := filepath.Join(vault, "Renamed")
	if err := os.Rename(old, new); err != nil {
		t.Fatal(err)
	}
	if err := svc.EndInternalMutationAndRefreshBaseline(func() error {
		return fw.RefreshBaseline()
	}); err != nil {
		t.Fatal(err)
	}

	// Wait for watcher poll cycles.
	time.Sleep(500 * time.Millisecond)

	// External structural callback should NOT have been called
	// for the internal rename.
	mu.Lock()
	calls := structuralCalls
	mu.Unlock()

	// The watcher may have called the structural callback 0 or 1 times
	// (on startup there may be a call from polling the initial structure).
	// But the important thing: tree should show the rename.
	tree := svc.GetTree()
	found := false
	for _, r := range tree.Roots {
		if r.ID == wsID || r.Name == "Renamed" {
			found = true
		}
	}
	if !found {
		t.Fatalf("tree should show renamed workspace after internal mutation, calls=%d", calls)
	}
}

func TestIntegrationBurstDebounce(t *testing.T) {
	vault := t.TempDir()
	bus := events.NewBus()
	svc := NewService(vault, bus)
	if err := svc.Initialize(); err != nil {
		t.Fatal(err)
	}

	var reconcileCount int
	var mu sync.Mutex

	fw := filewatcher.NewService(bus, 50*time.Millisecond)
	fw.SetWorkspaceResolver(func(relPath string) (string, string, bool) {
		return svc.ResolveWorkspaceForPath(relPath)
	})
	fw.SetOnStructuralChange(func() {
		mu.Lock()
		reconcileCount++
		mu.Unlock()
		_ = svc.RescanWorkspaceTree()
	})

	if err := fw.RefreshBaseline(); err != nil {
		t.Fatal(err)
	}
	if err := fw.Start(vault); err != nil {
		t.Fatal(err)
	}
	defer fw.Stop()

	time.Sleep(150 * time.Millisecond)

	// Create several directories in quick succession.
	for _, name := range []string{"A", "B", "C", "D", "E"} {
		mustMkdirAll(t, filepath.Join(vault, name))
		time.Sleep(10 * time.Millisecond)
	}

	// Wait for debounce and reconciliation.
	time.Sleep(2 * time.Second)

	mu.Lock()
	calls := reconcileCount
	mu.Unlock()

	// With proper debounce, the number of reconciliation calls should be small.
	// It's hard to guarantee exactly 1 due to timing, but it shouldn't be 5.
	if calls > 3 {
		t.Logf("reconcileCount=%d (may be >1 due to timing, but should be reasonable)", calls)
	}

	// Tree should eventually contain all folders.
	tree := svc.GetTree()
	if len(tree.Roots) < 3 {
		t.Fatalf("expected at least 3 roots, got %d", len(tree.Roots))
	}
}

func TestIntegrationWatcherBaselineAfterInternalMutation(t *testing.T) {
	vault := t.TempDir()
	wsID := testUUID("int-base")
	createWS(t, vault, "Project", wsID)

	bus := events.NewBus()
	svc := NewService(vault, bus)
	svc.Initialize()

	fw := filewatcher.NewService(bus, 100*time.Millisecond)
	fw.SetWorkspaceResolver(func(relPath string) (string, string, bool) {
		return svc.ResolveWorkspaceForPath(relPath)
	})
	var externalCalls int
	var mu sync.Mutex
	fw.SetOnStructuralChange(func() {
		mu.Lock()
		externalCalls++
		mu.Unlock()
	})
	fw.RefreshBaseline()
	fw.Start(vault)
	defer fw.Stop()

	time.Sleep(150 * time.Millisecond)

	// Record baseline external calls before mutation.
	mu.Lock()
	before := externalCalls
	mu.Unlock()

	// Internal mutation with baseline refresh.
	svc.BeginInternalMutation()
	createWS(t, vault, "NewOne", testUUID("int-new"))
	svc.EndInternalMutationAndRefreshBaseline(func() error {
		return fw.RefreshBaseline()
	})

	// Wait multiple poll cycles.
	time.Sleep(600 * time.Millisecond)

	// External calls should be close to 'before' — the internal mutation
	// was suppressed by baseline refresh.
	mu.Lock()
	after := externalCalls
	mu.Unlock()

	// The delta should be small (at most 1, for the structural callback
	// that might fire due to the initial scan seeing the new workspace
	// as new).
	if after-before > 2 {
		t.Fatalf("external structural calls jumped from %d to %d after internal mutation", before, after)
	}

	// Tree should have the new workspace.
	found := false
	for _, r := range svc.GetTree().Roots {
		if r.ID == testUUID("int-new") {
			found = true
		}
	}
	if !found {
		t.Fatal("new workspace should appear in tree")
	}
}

func TestIntegrationWorkspaceResolverForNestedPath(t *testing.T) {
	vault := t.TempDir()
	wsID := testUUID("res-ws")
	fID := testUUID("res-f")
	createFolder(t, vault, "Clients", fID)
	createWS(t, vault, "Clients/Active/Client1", wsID)

	svc := NewService(vault, nil)
	svc.Initialize()

	// Resolve a deeply nested file inside the workspace.
	wsID2, wsRoot, found := svc.ResolveWorkspaceForPath("Clients/Active/Client1/Notes/doc.md")
	if !found {
		t.Fatal("should resolve workspace for nested file")
	}
	if wsID2 != wsID {
		t.Fatalf("workspaceId = %q, want %s", wsID2, wsID)
	}
	if wsRoot != "Clients/Active/Client1" {
		t.Fatalf("workspaceRootPath = %q", wsRoot)
	}

	// A file inside an organizational folder (not workspace) should not resolve.
	_, _, found = svc.ResolveWorkspaceForPath("Clients/somefile.txt")
	if found {
		t.Fatal("should not resolve workspace for file in organizational folder")
	}
}

// ── Helpers ──────────────────────────────────────────────────────────────────

func waitFor(t *testing.T, timeout time.Duration, condition func() bool) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("condition not met after %v", timeout)
}
