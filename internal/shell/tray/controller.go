// Package tray owns the native tray menu wiring for the desktop shell.
package tray

import (
	"errors"
	"log"
	"strings"
	"sync"
	"sync/atomic"
)

var errBackendUnavailable = errors.New("tray backend is unavailable")

// MenuItem exposes click events from a native tray menu item.
type MenuItem interface {
	Clicked() <-chan struct{}
	SetTitle(title string)
	SetTooltip(tooltip string)
}

// BackendCallbacks are invoked by the native tray implementation.
// A backend keeps the platform-native secondary-click handler enabled so the
// operating system opens the menu on a right click.
type BackendCallbacks struct {
	Ready     func()
	Exit      func()
	LeftClick func()
}

// Backend is the platform tray implementation.
type Backend interface {
	Start(callbacks BackendCallbacks) error
	SetIcon(icon []byte) error
	SetTooltip(tooltip string) error
	AddMenuItem(title, tooltip string) (MenuItem, error)
	Stop()
}

// Actions are executed from the native tray menu.
type Actions struct {
	Show func()
	Quit func()
}

// Labels contains the user-visible text for one tray menu locale.
type Labels struct {
	ShowTitle   string
	ShowTooltip string
	QuitTitle   string
	QuitTooltip string
}

func englishLabels() Labels {
	return Labels{
		ShowTitle:   "Show Verstak",
		ShowTooltip: "Show the Verstak window",
		QuitTitle:   "Quit",
		QuitTooltip: "Quit Verstak",
	}
}

func russianLabels() Labels {
	return Labels{
		ShowTitle:   "Показать Верстак",
		ShowTooltip: "Показать окно Верстака",
		QuitTitle:   "Выйти",
		QuitTooltip: "Завершить Верстак",
	}
}

// LabelsForPreference resolves the saved language preference for native tray UI.
// System locale values are intentionally passed in by the shell so this package
// stays independent of process environment and is deterministic in tests.
func LabelsForPreference(preference string, systemLocales ...string) Labels {
	if strings.EqualFold(strings.TrimSpace(preference), "ru") {
		return russianLabels()
	}
	if strings.EqualFold(strings.TrimSpace(preference), "system") {
		for _, locale := range systemLocales {
			locale = strings.ToLower(strings.TrimSpace(locale))
			if locale == "" {
				continue
			}
			if strings.HasPrefix(locale, "ru") {
				return russianLabels()
			}
			return englishLabels()
		}
	}
	return englishLabels()
}

// Controller initializes one native tray and routes its menu actions.
type Controller struct {
	backend Backend
	icon    []byte

	startOnce sync.Once
	readyOnce sync.Once
	stopOnce  sync.Once
	started   atomic.Bool
	ready     atomic.Bool

	mu           sync.RWMutex
	labels       Labels
	actions      Actions
	show         MenuItem
	quit         MenuItem
	startErr     error
	readyChanged func(bool)
}

// New creates a tray controller for one application process.
func New(backend Backend, icon []byte) *Controller {
	return &Controller{
		backend: backend,
		icon:    append([]byte(nil), icon...),
		labels:  englishLabels(),
	}
}

// SetReadyChangedHandler receives readiness changes after native initialization
// has either completed or ended. It lets the Wails close policy avoid hiding a
// window before the tray can bring it back.
func (c *Controller) SetReadyChangedHandler(handler func(bool)) {
	if c == nil {
		return
	}
	c.mu.Lock()
	c.readyChanged = handler
	c.mu.Unlock()
}

// Ready reports whether the tray has completed icon and menu initialization.
func (c *Controller) Ready() bool {
	return c != nil && c.ready.Load()
}

// SetLabels updates the current and future native tray menu labels.
func (c *Controller) SetLabels(labels Labels) {
	if c == nil {
		return
	}
	c.mu.Lock()
	c.labels = labels
	show, quit := c.show, c.quit
	c.mu.Unlock()
	applyLabels(show, quit, labels)
}

// Start connects the tray to Wails without taking over Wails' GUI event loop.
// Ready remains false until the backend callback and all required menu setup
// finish successfully.
func (c *Controller) Start(actions Actions) error {
	if c == nil || c.backend == nil {
		return errBackendUnavailable
	}
	c.startOnce.Do(func() {
		c.mu.Lock()
		c.actions = actions
		c.mu.Unlock()
		c.started.Store(true)
		log.Printf("[tray] starting native backend")
		err := c.backend.Start(BackendCallbacks{
			Ready:     c.onReady,
			Exit:      c.onExit,
			LeftClick: c.onLeftClick,
		})
		if err != nil {
			c.mu.Lock()
			c.startErr = err
			c.mu.Unlock()
			c.started.Store(false)
			c.setReady(false)
			log.Printf("[tray] backend start failed: %v; falling back to normal window close", err)
		}
	})
	c.mu.RLock()
	err := c.startErr
	c.mu.RUnlock()
	return err
}

func (c *Controller) onReady() {
	if c == nil {
		return
	}
	if !c.started.Load() {
		log.Printf("[tray] ready callback ignored after backend startup failed")
		return
	}
	c.readyOnce.Do(func() {
		log.Printf("[tray] native backend reported ready")
		if err := c.backend.SetIcon(c.icon); err != nil {
			c.fail("icon setup", err)
			return
		}
		if err := c.backend.SetTooltip("Verstak"); err != nil {
			c.fail("tooltip setup", err)
			return
		}
		c.mu.RLock()
		labels := c.labels
		actions := c.actions
		c.mu.RUnlock()
		show, err := c.backend.AddMenuItem(labels.ShowTitle, labels.ShowTooltip)
		if err != nil || show == nil {
			if err == nil {
				err = errors.New("show menu item is nil")
			}
			c.fail("show menu creation", err)
			return
		}
		quit, err := c.backend.AddMenuItem(labels.QuitTitle, labels.QuitTooltip)
		if err != nil || quit == nil {
			if err == nil {
				err = errors.New("quit menu item is nil")
			}
			c.fail("quit menu creation", err)
			return
		}
		c.mu.Lock()
		c.show, c.quit = show, quit
		labels = c.labels
		c.mu.Unlock()
		applyLabels(show, quit, labels)
		if actions.Show != nil {
			go routeClicks(show.Clicked(), func() {
				log.Printf("[tray] Show command")
				actions.Show()
			})
		}
		if actions.Quit != nil {
			go routeClicks(quit.Clicked(), func() {
				log.Printf("[tray] Quit command")
				actions.Quit()
			})
		}
		c.setReady(true)
		log.Printf("[tray] native tray is ready")
	})
}

func (c *Controller) onLeftClick() {
	if c == nil {
		return
	}
	log.Printf("[tray] left click")
	if !c.Ready() {
		log.Printf("[tray] left click ignored while tray is not ready")
		return
	}
	c.mu.RLock()
	show := c.actions.Show
	c.mu.RUnlock()
	if show != nil {
		show()
	}
}

func (c *Controller) onExit() {
	if c == nil {
		return
	}
	c.setReady(false)
	log.Printf("[tray] native message loop ended; falling back to normal window close")
}

func (c *Controller) fail(stage string, err error) {
	c.setReady(false)
	log.Printf("[tray] %s failed: %v; falling back to normal window close", stage, err)
	c.Stop()
}

func (c *Controller) setReady(ready bool) {
	if c == nil || c.ready.Swap(ready) == ready {
		return
	}
	c.mu.RLock()
	handler := c.readyChanged
	c.mu.RUnlock()
	if handler != nil {
		handler(ready)
	}
}

func applyLabels(show, quit MenuItem, labels Labels) {
	if show != nil {
		show.SetTitle(labels.ShowTitle)
		show.SetTooltip(labels.ShowTooltip)
	}
	if quit != nil {
		quit.SetTitle(labels.QuitTitle)
		quit.SetTooltip(labels.QuitTooltip)
	}
}

// Stop releases the native tray after Wails has begun application shutdown.
func (c *Controller) Stop() {
	if c == nil || c.backend == nil || !c.started.Load() {
		return
	}
	c.stopOnce.Do(func() {
		c.setReady(false)
		log.Printf("[tray] stopping native backend")
		c.backend.Stop()
	})
}

func routeClicks(clicked <-chan struct{}, action func()) {
	for range clicked {
		action()
	}
}
