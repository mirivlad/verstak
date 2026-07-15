package tray

import (
	"errors"
	"sync"
	"testing"
	"time"
)

type fakeMenuItem struct {
	clicked chan struct{}
	title   string
	tooltip string
}

func (i *fakeMenuItem) Clicked() <-chan struct{}  { return i.clicked }
func (i *fakeMenuItem) SetTitle(title string)     { i.title = title }
func (i *fakeMenuItem) SetTooltip(tooltip string) { i.tooltip = tooltip }

type fakeBackend struct {
	callbacks BackendCallbacks
	icon      []byte
	tooltip   string
	items     map[string]*fakeMenuItem
	startErr  error
	iconErr   error
	menuErr   error
	starts    int
	stops     int
}

func (b *fakeBackend) Start(callbacks BackendCallbacks) error {
	b.starts++
	b.callbacks = callbacks
	return b.startErr
}

func (b *fakeBackend) SetIcon(icon []byte) error {
	b.icon = append([]byte(nil), icon...)
	return b.iconErr
}

func (b *fakeBackend) SetTooltip(tooltip string) error {
	b.tooltip = tooltip
	return nil
}

func (b *fakeBackend) AddMenuItem(title, tooltip string) (MenuItem, error) {
	if b.menuErr != nil {
		return nil, b.menuErr
	}
	item := &fakeMenuItem{clicked: make(chan struct{}, 1), title: title, tooltip: tooltip}
	b.items[title] = item
	return item, nil
}

func (b *fakeBackend) Stop() { b.stops++ }

func (b *fakeBackend) ready() {
	if b.callbacks.Ready != nil {
		b.callbacks.Ready()
	}
}

func (b *fakeBackend) exited() {
	if b.callbacks.Exit != nil {
		b.callbacks.Exit()
	}
}

func (b *fakeBackend) leftClick() {
	if b.callbacks.LeftClick != nil {
		b.callbacks.LeftClick()
	}
}

func waitFor(t *testing.T, signal <-chan struct{}) {
	t.Helper()
	select {
	case <-signal:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for tray action")
	}
}

func TestControllerBecomesReadyOnlyAfterBackendCallback(t *testing.T) {
	backend := &fakeBackend{items: make(map[string]*fakeMenuItem)}
	controller := New(backend, []byte{1, 2, 3})

	if err := controller.Start(Actions{}); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	if controller.Ready() {
		t.Fatal("tray became ready before backend callback")
	}

	backend.ready()
	if !controller.Ready() {
		t.Fatal("tray did not become ready after backend callback")
	}
	if string(backend.icon) != string([]byte{1, 2, 3}) || backend.tooltip != "Verstak" {
		t.Fatalf("tray initialization = icon:%v tooltip:%q", backend.icon, backend.tooltip)
	}
}

func TestControllerRoutesLeftClickAndMenuActionsToShowAndQuit(t *testing.T) {
	backend := &fakeBackend{items: make(map[string]*fakeMenuItem)}
	showCalls := make(chan struct{}, 2)
	quitCalls := make(chan struct{}, 1)
	controller := New(backend, []byte{1})

	if err := controller.Start(Actions{
		Show: func() { showCalls <- struct{}{} },
		Quit: func() { quitCalls <- struct{}{} },
	}); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	backend.ready()

	backend.leftClick()
	waitFor(t, showCalls)
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

func TestControllerStopCallsBackendOnce(t *testing.T) {
	backend := &fakeBackend{items: make(map[string]*fakeMenuItem)}
	controller := New(backend, []byte{1})
	if err := controller.Start(Actions{}); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	controller.Stop()
	controller.Stop()

	if backend.stops != 1 {
		t.Fatalf("backend stop calls = %d, want 1", backend.stops)
	}
}

func TestControllerStartupAndSetupFailuresNeverBecomeReady(t *testing.T) {
	for name, backend := range map[string]*fakeBackend{
		"start": {items: make(map[string]*fakeMenuItem), startErr: errors.New("start failed")},
		"icon":  {items: make(map[string]*fakeMenuItem), iconErr: errors.New("icon failed")},
		"menu":  {items: make(map[string]*fakeMenuItem), menuErr: errors.New("menu failed")},
	} {
		t.Run(name, func(t *testing.T) {
			controller := New(backend, []byte{1})
			err := controller.Start(Actions{})
			if name == "start" && err == nil {
				t.Fatal("Start() error = nil, want startup failure")
			}
			if name != "start" && err != nil {
				t.Fatalf("Start() error = %v, want nil before ready callback", err)
			}
			backend.ready()
			if controller.Ready() {
				t.Fatal("failed tray became ready")
			}
		})
	}
}

func TestControllerExitRevokesReadinessAndKeepsLocalizedMenuItems(t *testing.T) {
	backend := &fakeBackend{items: make(map[string]*fakeMenuItem)}
	controller := New(backend, []byte{1})
	controller.SetLabels(LabelsForPreference("ru"))
	if err := controller.Start(Actions{}); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	backend.ready()

	show := backend.items["Показать Верстак"]
	quit := backend.items["Выйти"]
	if show == nil || quit == nil {
		t.Fatalf("Russian tray menu = %#v", backend.items)
	}
	controller.SetLabels(LabelsForPreference("en"))
	if show.title != "Show Verstak" || quit.title != "Quit" {
		t.Fatalf("English tray menu after update = show:%q quit:%q", show.title, quit.title)
	}
	backend.exited()
	if controller.Ready() {
		t.Fatal("tray stayed ready after backend exit")
	}
}

func TestLabelsForSystemRussianLocale(t *testing.T) {
	labels := LabelsForPreference("system", "", "ru_RU.UTF-8", "en_US.UTF-8")
	if labels.ShowTitle != "Показать Верстак" || labels.QuitTitle != "Выйти" {
		t.Fatalf("system Russian labels = %#v", labels)
	}
	labels = LabelsForPreference("system", "C.UTF-8", "", "ru_RU.UTF-8")
	if labels.ShowTitle != "Show Verstak" || labels.QuitTitle != "Quit" {
		t.Fatalf("higher-priority system locale must win, got %#v", labels)
	}
}

func TestControllerReadyNotificationIsSafeAcrossCallbacks(t *testing.T) {
	backend := &fakeBackend{items: make(map[string]*fakeMenuItem)}
	controller := New(backend, []byte{1})
	var changes []bool
	var mu sync.Mutex
	controller.SetReadyChangedHandler(func(ready bool) {
		mu.Lock()
		changes = append(changes, ready)
		mu.Unlock()
	})
	if err := controller.Start(Actions{}); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	backend.ready()
	backend.exited()

	mu.Lock()
	defer mu.Unlock()
	if len(changes) != 2 || !changes[0] || changes[1] {
		t.Fatalf("ready changes = %#v, want [true false]", changes)
	}
}
