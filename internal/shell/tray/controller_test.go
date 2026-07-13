package tray

import (
	"testing"
	"time"
)

type fakeMenuItem struct {
	clicked chan struct{}
}

func (i *fakeMenuItem) Clicked() <-chan struct{} {
	return i.clicked
}

type fakeBackend struct {
	icon        []byte
	tooltip     string
	items       map[string]*fakeMenuItem
	quitCalls   int
	registering bool
}

func (b *fakeBackend) Register(onReady func(), _ func()) {
	b.registering = true
	onReady()
}

func (b *fakeBackend) SetIcon(icon []byte) {
	b.icon = append([]byte(nil), icon...)
}

func (b *fakeBackend) SetTooltip(tooltip string) {
	b.tooltip = tooltip
}

func (b *fakeBackend) AddMenuItem(title, _ string) MenuItem {
	item := &fakeMenuItem{clicked: make(chan struct{}, 1)}
	b.items[title] = item
	return item
}

func (b *fakeBackend) Quit() {
	b.quitCalls++
}

func waitFor(t *testing.T, signal <-chan struct{}) {
	t.Helper()
	select {
	case <-signal:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for tray action")
	}
}

func TestControllerInitializesTrayAndRoutesMenuActions(t *testing.T) {
	backend := &fakeBackend{items: make(map[string]*fakeMenuItem)}
	showCalls := make(chan struct{}, 1)
	quitCalls := make(chan struct{}, 1)
	controller := New(backend, []byte{1, 2, 3})

	controller.Start(Actions{
		Show: func() { showCalls <- struct{}{} },
		Quit: func() { quitCalls <- struct{}{} },
	})

	if !backend.registering || string(backend.icon) != string([]byte{1, 2, 3}) || backend.tooltip != "Verstak" {
		t.Fatalf("tray initialization = registering:%t icon:%v tooltip:%q", backend.registering, backend.icon, backend.tooltip)
	}
	showItem := backend.items["Show Verstak"]
	quitItem := backend.items["Quit"]
	if showItem == nil || quitItem == nil {
		t.Fatalf("tray menu = %#v, want Show Verstak and Quit", backend.items)
	}

	showItem.clicked <- struct{}{}
	waitFor(t, showCalls)
	quitItem.clicked <- struct{}{}
	waitFor(t, quitCalls)
}

func TestControllerStopsNativeTrayBackend(t *testing.T) {
	backend := &fakeBackend{items: make(map[string]*fakeMenuItem)}
	controller := New(backend, []byte{1})

	controller.Stop()

	if backend.quitCalls != 1 {
		t.Fatalf("backend quit calls = %d, want 1", backend.quitCalls)
	}
}
