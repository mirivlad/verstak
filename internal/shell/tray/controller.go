// Package tray owns the native tray menu wiring for the desktop shell.
package tray

import (
	"strings"
	"sync"
)

// MenuItem exposes click events from a native tray menu item.
type MenuItem interface {
	Clicked() <-chan struct{}
	SetTitle(title string)
	SetTooltip(tooltip string)
}

// Backend is the platform tray implementation.
type Backend interface {
	Register(onReady func(), onExit func())
	SetIcon(icon []byte)
	SetTooltip(tooltip string)
	AddMenuItem(title, tooltip string) MenuItem
	Quit()
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
	start   sync.Once
	mu      sync.RWMutex
	labels  Labels
	show    MenuItem
	quit    MenuItem
}

// New creates a tray controller for one application process.
func New(backend Backend, icon []byte) *Controller {
	return &Controller{
		backend: backend,
		icon:    append([]byte(nil), icon...),
		labels:  englishLabels(),
	}
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

// Start registers the native tray without taking over the Wails event loop.
func (c *Controller) Start(actions Actions) {
	if c == nil || c.backend == nil {
		return
	}
	c.start.Do(func() {
		c.backend.Register(func() {
			c.backend.SetIcon(c.icon)
			c.backend.SetTooltip("Verstak")
			c.mu.RLock()
			labels := c.labels
			c.mu.RUnlock()
			show := c.backend.AddMenuItem(labels.ShowTitle, labels.ShowTooltip)
			quit := c.backend.AddMenuItem(labels.QuitTitle, labels.QuitTooltip)
			c.mu.Lock()
			c.show, c.quit = show, quit
			labels = c.labels
			c.mu.Unlock()
			applyLabels(show, quit, labels)
			if actions.Show != nil && show != nil {
				go routeClicks(show.Clicked(), actions.Show)
			}
			if actions.Quit != nil && quit != nil {
				go routeClicks(quit.Clicked(), actions.Quit)
			}
		}, nil)
	})
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
	if c == nil || c.backend == nil {
		return
	}
	c.backend.Quit()
}

func routeClicks(clicked <-chan struct{}, action func()) {
	for range clicked {
		action()
	}
}
