package tray

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"

	"fyne.io/systray"
)

var (
	runWithExternalLoop = systray.RunWithExternalLoop
	setOnTapped         = systray.SetOnTapped
)

type systrayBackend struct {
	mu            sync.Mutex
	end           func()
	iconPath      string
	started       bool
	stopRequested bool
}

type systrayMenuItem struct {
	item *systray.MenuItem
}

// NewNativeBackend creates the cross-platform tray backend. It uses
// RunWithExternalLoop so the native tray message loop runs alongside Wails.
func NewNativeBackend() Backend {
	return &systrayBackend{}
}

func (b *systrayBackend) Start(callbacks BackendCallbacks) (err error) {
	if b == nil {
		return errBackendUnavailable
	}
	b.mu.Lock()
	if b.started {
		b.mu.Unlock()
		return errors.New("native tray backend was already started")
	}
	b.started = true
	b.mu.Unlock()
	defer func() {
		if recovered := recover(); recovered != nil {
			err = fmt.Errorf("native tray startup panic: %v", recovered)
		}
	}()
	start, end := runWithExternalLoop(callbacks.Ready, callbacks.Exit)
	if callbacks.LeftClick != nil {
		setOnTapped(callbacks.LeftClick)
	}
	// The Ready callback is asynchronous on Windows and can synchronously
	// decide that startup failed. Start the native loop before publishing the
	// end function so a concurrent Stop is never left with a half-started loop.
	start()
	b.mu.Lock()
	stopRequested := b.stopRequested
	if !stopRequested {
		b.end = end
	}
	b.mu.Unlock()
	if stopRequested {
		end()
	}
	return nil
}

func (b *systrayBackend) SetIcon(icon []byte) error {
	if b == nil || len(icon) == 0 {
		return errors.New("tray icon is empty")
	}
	file, err := os.CreateTemp("", "verstak-tray-*"+IconFileExtension())
	if err != nil {
		return fmt.Errorf("create tray icon file: %w", err)
	}
	path := file.Name()
	if _, err := file.Write(icon); err != nil {
		file.Close()
		os.Remove(path)
		return fmt.Errorf("write tray icon file: %w", err)
	}
	if err := file.Close(); err != nil {
		os.Remove(path)
		return fmt.Errorf("close tray icon file: %w", err)
	}
	if err := systray.SetIconFromFilePath(path); err != nil {
		os.Remove(path)
		return fmt.Errorf("set native tray icon: %w", err)
	}
	b.mu.Lock()
	previous := b.iconPath
	b.iconPath = path
	b.mu.Unlock()
	if previous != "" && previous != path {
		_ = os.Remove(previous)
	}
	return nil
}

func (b *systrayBackend) SetTooltip(tooltip string) error {
	if strings.TrimSpace(tooltip) == "" {
		return errors.New("tray tooltip is empty")
	}
	systray.SetTooltip(tooltip)
	return nil
}

func (b *systrayBackend) AddMenuItem(title, tooltip string) (MenuItem, error) {
	if strings.TrimSpace(title) == "" {
		return nil, errors.New("tray menu title is empty")
	}
	item := systray.AddMenuItem(title, tooltip)
	if item == nil {
		return nil, errors.New("native tray menu item is nil")
	}
	return systrayMenuItem{item: item}, nil
}

func (b *systrayBackend) Stop() {
	if b == nil {
		return
	}
	b.mu.Lock()
	end, iconPath := b.end, b.iconPath
	b.end = nil
	b.iconPath = ""
	b.stopRequested = true
	b.mu.Unlock()
	if end != nil {
		end()
	}
	if iconPath != "" {
		_ = os.Remove(iconPath)
	}
}

func (item systrayMenuItem) Clicked() <-chan struct{} {
	if item.item == nil {
		return nil
	}
	return item.item.ClickedCh
}

func (item systrayMenuItem) SetTitle(title string) {
	if item.item != nil {
		item.item.SetTitle(title)
	}
}

func (item systrayMenuItem) SetTooltip(tooltip string) {
	if item.item != nil {
		item.item.SetTooltip(tooltip)
	}
}
