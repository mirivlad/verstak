package filewatcher

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/verstak/verstak-desktop/internal/core/events"
)

func TestServicePublishesExternalFileChanges(t *testing.T) {
	root := t.TempDir()
	bus := events.NewBus()
	received := make(chan events.Event, 4)
	bus.Subscribe("file.changed", func(event events.Event) {
		received <- event
	})

	service := NewService(bus, 10*time.Millisecond)
	if err := service.Start(root); err != nil {
		t.Fatalf("Start: %v", err)
	}
	t.Cleanup(service.Stop)

	if err := os.WriteFile(filepath.Join(root, "note.md"), []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}

	event := waitForEvent(t, received)
	payload, ok := event.Payload.(map[string]interface{})
	if !ok {
		t.Fatalf("payload type = %T, want map", event.Payload)
	}
	if event.Name != "file.changed" {
		t.Fatalf("event name = %q, want file.changed", event.Name)
	}
	if payload["path"] != "note.md" {
		t.Fatalf("path = %v, want note.md", payload["path"])
	}
	if payload["operation"] != "external.create" {
		t.Fatalf("operation = %v, want external.create", payload["operation"])
	}
	if payload["type"] != "file" {
		t.Fatalf("type = %v, want file", payload["type"])
	}
}

func TestServiceIgnoresReservedVerstakPaths(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".verstak"), 0o755); err != nil {
		t.Fatal(err)
	}
	bus := events.NewBus()
	received := make(chan events.Event, 4)
	bus.Subscribe("file.changed", func(event events.Event) {
		received <- event
	})

	service := NewService(bus, 10*time.Millisecond)
	if err := service.Start(root); err != nil {
		t.Fatalf("Start: %v", err)
	}
	t.Cleanup(service.Stop)

	if err := os.WriteFile(filepath.Join(root, ".verstak", "internal.json"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}

	select {
	case event := <-received:
		t.Fatalf("unexpected reserved-path event: %+v", event)
	case <-time.After(80 * time.Millisecond):
	}
}

func TestServiceCallsChangeCallbackForExternalChanges(t *testing.T) {
	root := t.TempDir()
	service := NewService(events.NewBus(), 10*time.Millisecond)
	changed := make(chan struct{}, 1)
	service.SetOnChange(func() { changed <- struct{}{} })
	if err := service.Start(root); err != nil {
		t.Fatalf("Start: %v", err)
	}
	t.Cleanup(service.Stop)

	if err := os.WriteFile(filepath.Join(root, "external.txt"), []byte("change"), 0o644); err != nil {
		t.Fatal(err)
	}
	select {
	case <-changed:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("timed out waiting for watcher callback")
	}
}

func waitForEvent(t *testing.T, eventCh <-chan events.Event) events.Event {
	t.Helper()
	select {
	case event := <-eventCh:
		return event
	case <-time.After(500 * time.Millisecond):
		t.Fatal("timed out waiting for file.changed")
		return events.Event{}
	}
}
