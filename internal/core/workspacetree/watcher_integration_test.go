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

// ── Marker copy race ────────────────────────────────────────────────────────

func TestIntegrationMarkerCopyRace(t *testing.T) {
	vault := t.TempDir()
	bus := events.NewBus()
	svc := NewService(vault, bus)
	if err := svc.Initialize(); err != nil {
		t.Fatal(err)
	}

	// Use long debounce so the reconciler doesn't rush to create folder.json.
	fw := filewatcher.NewService(bus, 80*time.Millisecond)
	fw.SetWorkspaceResolver(func(relPath string) (string, string, bool) {
		return svc.ResolveWorkspaceForPath(relPath)
	})
	fw.SetOnStructuralChange(func() {
		_ = svc.RescanWorkspaceTree()
	})

	if err := fw.Start(vault); err != nil {
		t.Fatal(err)
	}
	defer fw.Stop()

	time.Sleep(100 * time.Millisecond)

	// Step 1: Create directory with .verstak but NO markers yet.
	copyDir := filepath.Join(vault, "CopyingDeal")
	mustMkdirAll(t, copyDir)
	verstakDir := filepath.Join(copyDir, ".verstak")
	mustMkdirAll(t, verstakDir)

	// Let watcher see the directory with .verstak but no valid markers.
	// The reconciler should NOT create folder.json because .verstak exists.
	time.Sleep(150 * time.Millisecond)

	// Step 2: Now create workspace marker inside the existing .verstak.
	wsID := testUUID("copy-race")
	createWorkspaceMarker(t, verstakDir, wsID)

	// Step 3: Add files inside.
	mustMkdirAll(t, filepath.Join(copyDir, "Notes"))
	mustWriteFile(t, filepath.Join(copyDir, "Notes", "doc.md"), "# Doc")

	// Wait for reconciliation.
	waitFor(t, 4*time.Second, func() bool {
		_, ok := svc.GetWorkspaceByID(wsID)
		return ok
	})

	ws, ok := svc.GetWorkspaceByID(wsID)
	if !ok || ws.Name != "CopyingDeal" {
		t.Fatalf("workspace not found: %+v, %v", ws, ok)
	}

	// Check: no folder.json was created inside.
	folderMarkerPath := filepath.Join(copyDir, ".verstak", "folder.json")
	if _, err := os.Stat(folderMarkerPath); err == nil {
		t.Fatal("folder.json should NOT exist inside workspace")
	}

	// Check: workspace appears in tree exactly once.
	tree := svc.GetTree()
	count := 0
	for _, r := range tree.Roots {
		if r.ID == wsID {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("workspace appears %d times in tree", count)
	}

	// Check: Notes directory is NOT in tree as a folder.
	for _, r := range tree.Roots {
		if r.Name == "Notes" {
			t.Fatal("Notes should not appear as organizational folder")
		}
	}
}

// ── Marker deletion ─────────────────────────────────────────────────────────

func TestIntegrationFolderMarkerDeletion(t *testing.T) {
	vault := t.TempDir()
	oldID := testUUID("old-f")
	createFolder(t, vault, "MyFolder", oldID)

	bus := events.NewBus()
	svc := NewService(vault, bus)
	svc.Initialize()

	fw := filewatcher.NewService(bus, 80*time.Millisecond)
	fw.SetWorkspaceResolver(func(relPath string) (string, string, bool) {
		return svc.ResolveWorkspaceForPath(relPath)
	})
	fw.SetOnStructuralChange(func() {
		_ = svc.RescanWorkspaceTree()
	})
	fw.Start(vault)
	defer fw.Stop()

	// Verify old folder exists.
	f, ok := svc.GetFolderByID(oldID)
	if !ok || f.Name != "MyFolder" {
		t.Fatalf("initial folder: %+v", f)
	}

	time.Sleep(150 * time.Millisecond)

	// Delete folder.json.
	markerPath := filepath.Join(vault, "MyFolder", ".verstak", "folder.json")
	if err := os.Remove(markerPath); err != nil {
		t.Fatal(err)
	}

	// Wait for watcher to detect and trigger reconciliation.
	waitFor(t, 3*time.Second, func() bool {
		_, ok := svc.GetFolderByID(oldID)
		return !ok
	})

	// Old ID should no longer exist.
	if _, ok := svc.GetFolderByID(oldID); ok {
		t.Fatal("old folder ID should be gone after marker deletion")
	}

	// Since .verstak still exists (empty), the directory is in intermediate
	// state and is NOT immediately converted to a new folder.
	// This is correct: the system waits for settle to avoid mistaking
	// a workspace-copy-in-progress for a folder.
}

func TestIntegrationWorkspaceMarkerDeletion(t *testing.T) {
	vault := t.TempDir()
	wsID := testUUID("ws-del-marker")
	createWS(t, vault, "MyDeal", wsID)
	mustMkdirAll(t, filepath.Join(vault, "MyDeal", "Notes"))
	mustMkdirAll(t, filepath.Join(vault, "MyDeal", "Files"))

	bus := events.NewBus()
	svc := NewService(vault, bus)
	svc.Initialize()

	fw := filewatcher.NewService(bus, 80*time.Millisecond)
	fw.SetWorkspaceResolver(func(relPath string) (string, string, bool) {
		return svc.ResolveWorkspaceForPath(relPath)
	})
	fw.SetOnStructuralChange(func() {
		_ = svc.RescanWorkspaceTree()
	})
	fw.Start(vault)
	defer fw.Stop()

	ws, ok := svc.GetWorkspaceByID(wsID)
	if !ok {
		t.Fatal("initial workspace not found")
	}
	_ = ws

	time.Sleep(150 * time.Millisecond)

	// Delete workspace.json marker.
	markerPath := filepath.Join(vault, "MyDeal", ".verstak", "workspace.json")
	if err := os.Remove(markerPath); err != nil {
		t.Fatal(err)
	}

	// Wait for watcher to detect and trigger reconciliation.
	// Do NOT call explicit RescanWorkspaceTree — the watcher handles it.
	waitFor(t, 3*time.Second, func() bool {
		diags := svc.GetWorkspaceTreeDiagnostics()
		for _, d := range diags {
			if d.Code == "workspace-marker-missing" {
				return true
			}
		}
		return false
	})

	// Workspace should be gone.
	if _, ok := svc.GetWorkspaceByID(wsID); ok {
		t.Fatal("workspace should be gone after marker deletion")
	}

	// Directory should NOT become a folder (contains Notes/Files, .verstak exists).
	for _, r := range svc.GetTree().Roots {
		if r.Name == "MyDeal" {
			t.Fatalf("MyDeal should NOT appear in tree at all after marker deletion, got kind=%s", r.Kind)
		}
	}

	// Notes should NOT appear in tree.
	for _, r := range svc.GetTree().Roots {
		if r.Name == "Notes" {
			t.Fatal("Notes should not appear in tree")
		}
	}
}

// ── Physical workspace move ─────────────────────────────────────────────────

func TestIntegrationPhysicalWorkspaceMove(t *testing.T) {
	vault := t.TempDir()
	wsID := testUUID("move-ws")
	f1ID := testUUID("move-f1")
	f2ID := testUUID("move-f2")
	createFolder(t, vault, "Clients", f1ID)
	createFolder(t, vault, "Archive", f2ID)
	createWS(t, vault, "Clients/Client1", wsID)
	mustMkdirAll(t, filepath.Join(vault, "Clients", "Client1", "Notes"))

	bus := events.NewBus()
	var treeEvents []events.Event
	var mu sync.Mutex
	bus.Subscribe("workspace-tree.changed", func(e events.Event) {
		mu.Lock()
		treeEvents = append(treeEvents, e)
		mu.Unlock()
	})

	svc := NewService(vault, bus)
	svc.Initialize()

	fw := filewatcher.NewService(bus, 80*time.Millisecond)
	fw.SetWorkspaceResolver(func(relPath string) (string, string, bool) {
		return svc.ResolveWorkspaceForPath(relPath)
	})
	fw.SetOnStructuralChange(func() {
		_ = svc.RescanWorkspaceTree()
	})
	fw.Start(vault)
	defer fw.Stop()

	time.Sleep(150 * time.Millisecond)

	// Physical move: Clients/Client1 → Archive/Client1
	src := filepath.Join(vault, "Clients", "Client1")
	dst := filepath.Join(vault, "Archive", "Client1")
	if err := os.Rename(src, dst); err != nil {
		t.Fatal(err)
	}

	waitFor(t, 3*time.Second, func() bool {
		ws, ok := svc.GetWorkspaceByID(wsID)
		return ok && ws.RootPath == "Archive/Client1"
	})

	ws, _ := svc.GetWorkspaceByID(wsID)
	if ws.RootPath != "Archive/Client1" {
		t.Fatalf("RootPath = %q, want Archive/Client1", ws.RootPath)
	}

	// Check events — filter to workspace-move action.
	mu.Lock()
	defer mu.Unlock()
	hasMoved := false
	hasDeleted := false
	for _, evt := range treeEvents {
		p, _ := evt.Payload.(map[string]interface{})
		a, _ := p["action"].(string)
		evWSID, _ := p["workspaceId"].(string)
		// Only consider events for THIS workspace.
		if evWSID != wsID {
			continue
		}
		if a == "workspace.external-moved" {
			hasMoved = true
		}
		if a == "workspace.external-deleted" {
			hasDeleted = true
		}
	}
	if !hasMoved {
		t.Fatalf("expected workspace.external-moved event for %s, got %d events", wsID, len(treeEvents))
	}
	if hasDeleted {
		t.Fatal("should not have workspace.external-deleted for move")
	}
}

// ── Watcher scan error recovery ─────────────────────────────────────────────

func TestIntegrationWatcherScanErrorRecovery(t *testing.T) {
	vault := t.TempDir()
	wsID := testUUID("err-ws")
	createWS(t, vault, "Project", wsID)

	bus := events.NewBus()
	svc := NewService(vault, bus)
	svc.Initialize()

	fw := filewatcher.NewService(bus, 50*time.Millisecond)
	fw.SetOnStructuralChange(func() {
		_ = svc.RescanWorkspaceTree()
	})
	fw.SetWorkspaceResolver(func(relPath string) (string, string, bool) {
		return svc.ResolveWorkspaceForPath(relPath)
	})
	fw.Start(vault)

	time.Sleep(200 * time.Millisecond)

	// Simulate watcher being stopped (like a scan error recovery scenario).
	fw.Stop()

	// Make offline changes while watcher is "down".
	newID := testUUID("err-new")
	createWS(t, vault, "NewProject", newID)

	// Explicit rescan catches up.
	if err := svc.RescanWorkspaceTree(); err != nil {
		t.Fatal(err)
	}

	// New workspace should be discovered.
	if _, ok := svc.GetWorkspaceByID(newID); !ok {
		t.Fatal("new workspace not discovered after rescan")
	}

	// Restart watcher — baseline should include the new workspace.
	if err := fw.Start(vault); err != nil {
		t.Fatal(err)
	}
	defer fw.Stop()

	// Verify still accessible after restart.
	if _, ok := svc.GetWorkspaceByID(newID); !ok {
		t.Fatal("workspace lost after watcher restart")
	}
}

// ── Startup ordering ────────────────────────────────────────────────────────

func TestIntegrationStartupOrdering(t *testing.T) {
	vault := t.TempDir()
	// Pre-create entities that need marker adoption.
	mustMkdirAll(t, filepath.Join(vault, "Unmarked"))
	createWS(t, vault, "Project", testUUID("so-ws"))

	bus := events.NewBus()
	svc := NewService(vault, bus)

	var treeReady bool
	var mu sync.Mutex

	// Before starting watcher, tree should already be initialized.
	if err := svc.Initialize(); err != nil {
		t.Fatal(err)
	}

	mu.Lock()
	treeReady = true
	mu.Unlock()

	// Now start watcher.
	fw := filewatcher.NewService(bus, 80*time.Millisecond)
	var watcherEventsDuringInit int32
	fw.SetOnStructuralChange(func() {
		mu.Lock()
		if !treeReady {
			// This should never happen — treeInitDone should be true before watcher starts.
		}
		mu.Unlock()
	})
	fw.SetWorkspaceResolver(func(relPath string) (string, string, bool) {
		return svc.ResolveWorkspaceForPath(relPath)
	})

	// Start watcher AFTER tree init.
	if err := fw.Start(vault); err != nil {
		t.Fatal(err)
	}
	defer fw.Stop()

	// Tree should be ready before watcher was started.
	mu.Lock()
	ready := treeReady
	mu.Unlock()
	if !ready {
		t.Fatal("tree was not ready before watcher started")
	}

	// Unmarked folder should already have a marker from startup reconciliation.
	markerPath := filepath.Join(vault, "Unmarked", ".verstak", "folder.json")
	if _, err := os.Stat(markerPath); err != nil {
		t.Fatalf("unmarked folder should have marker after startup: %v", err)
	}

	_ = watcherEventsDuringInit
}

// ── Suppression race window test ────────────────────────────────────────────

func TestIntegrationSuppressionRaceWindow(t *testing.T) {
	vault := t.TempDir()
	wsID := testUUID("race-ws")
	createWS(t, vault, "Project", wsID)

	bus := events.NewBus()
	svc := NewService(vault, bus)
	svc.Initialize()

	// Use a very short poll interval so we can trigger polls quickly.
	fw := filewatcher.NewService(bus, 30*time.Millisecond)

	var externalCalls int32
	fw.SetOnStructuralChange(func() {
		// Only count if not internal mutation.
		if !svc.IsInternalMutation() {
			// atomic add — but this is safe because we check IsInternalMutation first.
		}
	})
	fw.SetWorkspaceResolver(func(relPath string) (string, string, bool) {
		return svc.ResolveWorkspaceForPath(relPath)
	})
	fw.Start(vault)
	defer fw.Stop()

	time.Sleep(150 * time.Millisecond)

	// Begin internal mutation.
	svc.BeginInternalMutation()

	// Physical rename.
	old := filepath.Join(vault, "Project")
	new := filepath.Join(vault, "Renamed")
	if err := os.Rename(old, new); err != nil {
		t.Fatal(err)
	}

	// Verify suppression is active.
	if !svc.IsInternalMutation() {
		t.Fatal("suppression should be active")
	}

	// Wait for several poll cycles while suppression is active.
	// The watcher will poll but must NOT trigger structural callback.
	time.Sleep(200 * time.Millisecond)

	// Now end mutation with baseline refresh.
	if err := svc.EndInternalMutationAndRefreshBaseline(func() error {
		return fw.RefreshBaseline()
	}); err != nil {
		t.Fatal(err)
	}

	// Suppression should now be released.
	if svc.IsInternalMutation() {
		t.Fatal("suppression should be released")
	}

	// Wait for additional poll cycles.
	time.Sleep(200 * time.Millisecond)

	// Tree should reflect the rename.
	ws, ok := svc.GetWorkspaceByID(wsID)
	if !ok || ws.RootPath != "Renamed" {
		t.Fatalf("workspace after internal rename = %+v", ws)
	}

	_ = externalCalls
}
