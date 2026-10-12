package api

import (
	"fmt"

	syncsvc "github.com/verstak/verstak-desktop/internal/core/sync"
)

// initialRemoteFileHistory records a server-side file lifetime beginning with
// a create operation. We can disregard that lifetime on a new device *only*
// when the server history subsequently deletes that same file. The ordinary
// initial reconciliation remains responsible for all still-live files and
// ambiguous histories; no local file may be overwritten to replay old moves.
type initialRemoteFileHistory struct {
	operations []string
	complete   bool
}

// obsoleteInitialFileOps inspects the complete server history, not the local
// filesystem. A create -> move -> delete chain describes a file that no longer
// exists remotely; replaying it against an independently populated local vault
// can move or destroy an unrelated file with the same name.
//
// Only discard fully traced file lifetimes. A missing create, an invalid move,
// or a destination collision fails closed and will be handled by the existing
// conflict-aware replay. Deletes without a traced create are not discarded.
func obsoleteInitialFileOps(ops []syncsvc.Op) map[string]bool {
	live := make(map[string]initialRemoteFileHistory)
	obsolete := make(map[string]bool)
	for _, op := range ops {
		if op.EntityType != syncsvc.EntityFile || op.OpID == "" {
			continue
		}
		payload, err := parseSyncFilePayload(op.PayloadJSON)
		if err != nil {
			continue
		}
		path := syncPayloadPath(op, payload)
		switch op.OpType {
		case syncsvc.OpCreate:
			if path == "" {
				continue
			}
			// A duplicate create leaves the old lifetime ambiguous.
			if _, occupied := live[path]; occupied {
				live[path] = initialRemoteFileHistory{}
				continue
			}
			live[path] = initialRemoteFileHistory{operations: []string{op.OpID}, complete: true}
		case syncsvc.OpUpdate:
			if path == "" {
				continue
			}
			history := live[path]
			history.operations = append(history.operations, op.OpID)
			live[path] = history
		case syncsvc.OpMove:
			fromPath := payload.FromPath
			if fromPath == "" {
				fromPath = op.EntityID
			}
			if fromPath == "" || payload.ToPath == "" || fromPath == payload.ToPath {
				continue
			}
			history, known := live[fromPath]
			if _, occupied := live[payload.ToPath]; occupied {
				// Cannot safely reason about a move over an existing target.
				live[fromPath] = initialRemoteFileHistory{}
				live[payload.ToPath] = initialRemoteFileHistory{}
				continue
			}
			delete(live, fromPath)
			history.operations = append(history.operations, op.OpID)
			history.complete = known && history.complete
			live[payload.ToPath] = history
		case syncsvc.OpDelete:
			if path == "" {
				continue
			}
			if history, known := live[path]; known && history.complete {
				for _, id := range history.operations {
					obsolete[id] = true
				}
				obsolete[op.OpID] = true
			}
			delete(live, path)
		}
	}
	return obsolete
}

// fetchInitialFileHistory is a read-only preflight. A bounded, paginated
// snapshot of the server log is needed to distinguish transient old operations
// from still-live remote files before touching any local data. If preflight
// fails, the caller must not proceed to reconciliation or push.
func fetchInitialFileHistory(client *syncsvc.Client) (map[string]bool, error) {
	const maxHistoryOps = 200000
	var operations []syncsvc.Op
	cursor := 0
	for {
		page, err := client.PullPage(cursor, 0)
		if err != nil {
			return nil, fmt.Errorf("inspect initial remote history: %w", err)
		}
		if len(operations)+len(page.Ops) > maxHistoryOps {
			return nil, fmt.Errorf("initial remote history exceeds %d operations; refusing unsafe replay", maxHistoryOps)
		}
		last := cursor
		for _, op := range page.Ops {
			if op.ServerSequence <= last || op.OpID == "" {
				return nil, fmt.Errorf("initial remote history is not strictly ordered at sequence %d", op.ServerSequence)
			}
			last = op.ServerSequence
		}
		if page.PageLastSequence != last {
			return nil, fmt.Errorf("initial remote history cursor mismatch: received %d, expected %d", page.PageLastSequence, last)
		}
		operations = append(operations, page.Ops...)
		if !page.HasMore {
			return obsoleteInitialFileOps(operations), nil
		}
		if last <= cursor {
			return nil, fmt.Errorf("initial remote history did not advance the cursor")
		}
		cursor = last
	}
}
