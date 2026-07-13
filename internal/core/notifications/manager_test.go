package notifications

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"
)

type testVault struct {
	path string
}

func (v testVault) WithOpenPath(operation func(string) error) error {
	return operation(v.path)
}

type fakeSender struct {
	err   error
	items []Item
}

func (s *fakeSender) Send(_ context.Context, item Item) error {
	s.items = append(s.items, item)
	return s.err
}

func testManager(t *testing.T, sender *fakeSender, now time.Time) *Manager {
	t.Helper()
	return New(testVault{path: t.TempDir()}, sender, func() time.Time { return now })
}

func request(id, dueAt string) Request {
	return Request{ID: id, DueAt: dueAt, Title: "Reminder " + id, Body: "Todo " + id}
}

func TestReplaceRemovesStalePluginSchedules(t *testing.T) {
	manager := testManager(t, &fakeSender{}, time.Date(2026, 7, 14, 9, 0, 0, 0, time.UTC))
	if err := manager.Replace("verstak.todo", []Request{
		request("first", "2026-07-14T10:00:00Z"),
		request("second", "2026-07-14T11:00:00Z"),
	}); err != nil {
		t.Fatalf("initial replace: %v", err)
	}
	if err := manager.Replace("verstak.todo", []Request{request("second", "2026-07-14T11:00:00Z")}); err != nil {
		t.Fatalf("stale-item replace: %v", err)
	}

	items := manager.Items()
	if len(items) != 1 || items[0].PluginID != "verstak.todo" || items[0].ID != "second" {
		t.Fatalf("items = %#v, want only second Todo reminder", items)
	}
}

func TestReplaceReschedulesDeliveredItem(t *testing.T) {
	now := time.Date(2026, 7, 14, 12, 0, 0, 0, time.UTC)
	sender := &fakeSender{}
	manager := testManager(t, sender, now)
	if err := manager.Replace("verstak.todo", []Request{request("todo-1", "2026-07-14T11:00:00Z")}); err != nil {
		t.Fatalf("initial replace: %v", err)
	}
	manager.Tick(context.Background())
	if len(sender.items) != 1 || manager.Items()[0].SentForDueAt == "" {
		t.Fatalf("first delivery = %#v, state = %#v", sender.items, manager.Items())
	}

	if err := manager.Replace("verstak.todo", []Request{request("todo-1", "2026-07-14T13:00:00Z")}); err != nil {
		t.Fatalf("reschedule: %v", err)
	}
	if got := manager.Items()[0].SentForDueAt; got != "" {
		t.Fatalf("rescheduled item kept old acknowledgment %q", got)
	}
}

func TestTickRetriesFailedSendAndAcknowledgesOneDelivery(t *testing.T) {
	now := time.Date(2026, 7, 14, 12, 0, 0, 0, time.UTC)
	sender := &fakeSender{err: errors.New("notification unavailable")}
	manager := testManager(t, sender, now)
	if err := manager.Replace("verstak.todo", []Request{request("todo-1", "2026-07-14T11:00:00Z")}); err != nil {
		t.Fatalf("replace: %v", err)
	}

	manager.Tick(context.Background())
	if len(sender.items) != 1 || manager.Items()[0].SentForDueAt != "" {
		t.Fatalf("failed send was acknowledged: sent=%d items=%#v", len(sender.items), manager.Items())
	}

	sender.err = nil
	manager.Tick(context.Background())
	manager.Tick(context.Background())
	if len(sender.items) != 2 {
		t.Fatalf("send attempts = %d, want 2", len(sender.items))
	}
	if got := manager.Items()[0].SentForDueAt; got != "2026-07-14T11:00:00Z" {
		t.Fatalf("sent acknowledgment = %q", got)
	}
}

func TestPersistentSchedulesReloadAndDeliverOverdueOnlyOnce(t *testing.T) {
	path := t.TempDir()
	now := time.Date(2026, 7, 14, 12, 0, 0, 0, time.UTC)
	first := New(testVault{path: path}, &fakeSender{}, func() time.Time { return now })
	if err := first.Replace("verstak.todo", []Request{request("todo-1", "2026-07-14T11:00:00Z")}); err != nil {
		t.Fatalf("replace: %v", err)
	}
	if got, want := filepath.Join(path, ".verstak", "notifications", "schedules.json"), first.Path(); got != want {
		t.Fatalf("schedule path = %q, want %q", got, want)
	}

	sender := &fakeSender{}
	restarted := New(testVault{path: path}, sender, func() time.Time { return now })
	restarted.Tick(context.Background())
	restarted.Tick(context.Background())
	if len(sender.items) != 1 || sender.items[0].ID != "todo-1" {
		t.Fatalf("overdue delivery = %#v", sender.items)
	}
}

func TestReplaceRejectsInvalidRequests(t *testing.T) {
	manager := testManager(t, &fakeSender{}, time.Now().UTC())
	for _, requests := range [][]Request{
		{{ID: "", DueAt: "2026-07-14T10:00:00Z", Title: "title"}},
		{{ID: "same", DueAt: "2026-07-14T10:00:00Z", Title: "title"}, {ID: "same", DueAt: "2026-07-14T11:00:00Z", Title: "title"}},
		{{ID: "bad-date", DueAt: "not-a-date", Title: "title"}},
	} {
		if err := manager.Replace("verstak.todo", requests); err == nil {
			t.Fatalf("Replace(%#v) succeeded", requests)
		}
	}
}
