package api

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	corefiles "github.com/verstak/verstak-desktop/internal/core/files"
	syncsvc "github.com/verstak/verstak-desktop/internal/core/sync"
)

// captureTransferProgress collects the progress events a bulk transfer emits,
// so the tests can check that the interface is actually told what is happening
// rather than left to guess.
func captureTransferProgress(t *testing.T, app *App) *[]map[string]interface{} {
	t.Helper()
	app.ctx = context.Background()
	seen := make([]map[string]interface{}, 0)
	original := emitFrontendEvent
	emitFrontendEvent = func(_ context.Context, eventName string, data ...interface{}) {
		if eventName != "verstak:files-transfer-progress" || len(data) != 1 {
			return
		}
		if payload, ok := data[0].(map[string]interface{}); ok {
			seen = append(seen, payload)
		}
	}
	t.Cleanup(func() { emitFrontendEvent = original })
	return &seen
}

func seedTransferSource(t *testing.T, root string, count int) []corefiles.PathTransfer {
	t.Helper()
	for _, dir := range []string{"From", "To"} {
		if err := os.MkdirAll(filepath.Join(root, dir), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	transfers := make([]corefiles.PathTransfer, 0, count)
	for i := 0; i < count; i++ {
		name := "item-" + string(rune('a'+i)) + ".txt"
		if err := os.WriteFile(filepath.Join(root, "From", name), []byte(name), 0o600); err != nil {
			t.Fatal(err)
		}
		transfers = append(transfers, corefiles.PathTransfer{From: "From/" + name, To: "To/" + name})
	}
	return transfers
}

func TestMoveVaultPathsMovesEveryItemAndRecordsOneScan(t *testing.T) {
	app, root := newFilesTestApp(t, []string{"files.read", "files.write"})
	progress := captureTransferProgress(t, app)
	app.syncSvc = syncsvc.NewService(root, "device-a")
	if _, err := app.syncSvc.ScanAndRecord(); err != nil {
		t.Fatal(err)
	}
	transfers := seedTransferSource(t, root, 5)

	outcome, errStr := app.MoveVaultPaths("files.plugin", "t1", transfers, corefiles.MoveOptions{})
	if errStr != "" {
		t.Fatalf("MoveVaultPaths: %s", errStr)
	}
	if outcome.Succeeded != 5 || outcome.Failed != 0 || outcome.Cancelled {
		t.Fatalf("outcome = %+v", outcome)
	}
	for _, transfer := range transfers {
		if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(transfer.To))); err != nil {
			t.Fatalf("destination missing: %v", err)
		}
		if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(transfer.From))); !os.IsNotExist(err) {
			t.Fatalf("source still present: %s", transfer.From)
		}
	}

	// The batch must leave the snapshot agreeing with the vault, or the next
	// scan produces a second round of operations for the same files.
	before, err := app.syncSvc.GetUnpushedOps()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := app.syncSvc.ScanAndRecord(); err != nil {
		t.Fatal(err)
	}
	after, err := app.syncSvc.GetUnpushedOps()
	if err != nil {
		t.Fatal(err)
	}
	if len(after) != len(before) {
		t.Fatalf("a full scan after the batch recorded %d more operations; the batch left the snapshot stale", len(after)-len(before))
	}
	if len(before) == 0 {
		t.Fatal("the batch recorded no sync operations at all")
	}

	if len(*progress) != len(transfers) {
		t.Fatalf("progress was reported %d times for %d items", len(*progress), len(transfers))
	}
	last := (*progress)[len(*progress)-1]
	if last["completed"] != len(transfers) || last["total"] != len(transfers) {
		t.Fatalf("final progress event = %+v", last)
	}
}

// A single bad item must not cost the user the rest of the paste.
func TestBulkTransferReportsFailuresWithoutAbandoningTheRest(t *testing.T) {
	app, root := newFilesTestApp(t, []string{"files.read", "files.write"})
	captureTransferProgress(t, app)
	transfers := seedTransferSource(t, root, 3)
	transfers = append([]corefiles.PathTransfer{{From: "From/missing.txt", To: "To/missing.txt"}}, transfers...)

	outcome, errStr := app.MoveVaultPaths("files.plugin", "t2", transfers, corefiles.MoveOptions{})
	if errStr != "" {
		t.Fatalf("MoveVaultPaths: %s", errStr)
	}
	if outcome.Succeeded != 3 || outcome.Failed != 1 {
		t.Fatalf("outcome = %+v", outcome)
	}
	if outcome.Results[0].Error == "" {
		t.Fatalf("the missing source was reported as a success: %+v", outcome.Results[0])
	}
	for _, result := range outcome.Results[1:] {
		if result.Error != "" {
			t.Fatalf("a good item failed after a bad one: %+v", result)
		}
	}
}

func TestCancelVaultTransferStopsTheBatchAndMarksTheRestSkipped(t *testing.T) {
	app, root := newFilesTestApp(t, []string{"files.read", "files.write"})
	captureTransferProgress(t, app)
	transfers := seedTransferSource(t, root, 4)

	// Cancelled before it starts: the whole batch must be skipped, and nothing
	// on disk may move.
	if errStr := app.CancelVaultTransfer("files.plugin", "t3"); errStr != "" {
		t.Fatalf("CancelVaultTransfer: %s", errStr)
	}
	outcome, errStr := app.MoveVaultPaths("files.plugin", "t3", transfers, corefiles.MoveOptions{})
	if errStr != "" {
		t.Fatalf("MoveVaultPaths: %s", errStr)
	}
	if !outcome.Cancelled || outcome.Succeeded != 0 {
		t.Fatalf("outcome = %+v", outcome)
	}
	if len(outcome.Results) != len(transfers) {
		t.Fatalf("every item should be accounted for, got %d of %d", len(outcome.Results), len(transfers))
	}
	for _, result := range outcome.Results {
		if !result.Skipped {
			t.Fatalf("item not marked skipped: %+v", result)
		}
	}
	for _, transfer := range transfers {
		if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(transfer.From))); err != nil {
			t.Fatalf("a cancelled transfer moved %s anyway", transfer.From)
		}
	}

	// The cancellation must not persist and poison the next transfer reusing
	// that id.
	outcome, errStr = app.MoveVaultPaths("files.plugin", "t3", transfers, corefiles.MoveOptions{})
	if errStr != "" {
		t.Fatalf("second MoveVaultPaths: %s", errStr)
	}
	if outcome.Cancelled || outcome.Succeeded != len(transfers) {
		t.Fatalf("a stale cancellation blocked a later transfer: %+v", outcome)
	}
}

func TestCopyVaultPathsLeavesSourcesInPlace(t *testing.T) {
	app, root := newFilesTestApp(t, []string{"files.read", "files.write"})
	captureTransferProgress(t, app)
	transfers := seedTransferSource(t, root, 3)

	outcome, errStr := app.CopyVaultPaths("files.plugin", "t4", transfers, corefiles.CopyOptions{})
	if errStr != "" {
		t.Fatalf("CopyVaultPaths: %s", errStr)
	}
	if outcome.Succeeded != 3 {
		t.Fatalf("outcome = %+v", outcome)
	}
	for _, transfer := range transfers {
		for _, path := range []string{transfer.From, transfer.To} {
			if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(path))); err != nil {
				t.Fatalf("expected %s to exist: %v", path, err)
			}
		}
	}
}

func TestBulkTransferRequiresPermission(t *testing.T) {
	app, root := newFilesTestApp(t, []string{"files.read"})
	captureTransferProgress(t, app)
	transfers := seedTransferSource(t, root, 1)

	if _, errStr := app.MoveVaultPaths("files.plugin", "t5", transfers, corefiles.MoveOptions{}); errStr == "" {
		t.Fatal("a plugin without files.write was allowed to move files in bulk")
	}
	if _, errStr := app.CopyVaultPaths("files.plugin", "t6", transfers, corefiles.CopyOptions{}); errStr == "" {
		t.Fatal("a plugin without files.write was allowed to copy files in bulk")
	}
	if errStr := app.CancelVaultTransfer("files.plugin", "t5"); errStr == "" {
		t.Fatal("a plugin without files.write was allowed to cancel a transfer")
	}
}
