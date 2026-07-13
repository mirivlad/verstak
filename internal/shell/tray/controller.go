// Package tray owns the native tray menu wiring for the desktop shell.
package tray

import "sync"

// MenuItem exposes click events from a native tray menu item.
type MenuItem interface {
	Clicked() <-chan struct{}
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

// Controller initializes one native tray and routes its menu actions.
type Controller struct {
	backend Backend
	icon    []byte
	start   sync.Once
}

// New creates a tray controller for one application process.
func New(backend Backend, icon []byte) *Controller {
	return &Controller{
		backend: backend,
		icon:    append([]byte(nil), icon...),
	}
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
			show := c.backend.AddMenuItem("Show Verstak", "Show the Verstak window")
			quit := c.backend.AddMenuItem("Quit", "Quit Verstak")
			if actions.Show != nil && show != nil {
				go routeClicks(show.Clicked(), actions.Show)
			}
			if actions.Quit != nil && quit != nil {
				go routeClicks(quit.Clicked(), actions.Quit)
			}
		}, nil)
	})
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
