package tray

import "github.com/getlantern/systray"

type systrayBackend struct{}

type systrayMenuItem struct {
	item *systray.MenuItem
}

// NewNativeBackend creates the cross-platform native tray backend.
func NewNativeBackend() Backend {
	return systrayBackend{}
}

func (systrayBackend) Register(onReady func(), onExit func()) {
	systray.Register(onReady, onExit)
}

func (systrayBackend) SetIcon(icon []byte) {
	systray.SetIcon(icon)
}

func (systrayBackend) SetTooltip(tooltip string) {
	systray.SetTooltip(tooltip)
}

func (systrayBackend) AddMenuItem(title, tooltip string) MenuItem {
	return systrayMenuItem{item: systray.AddMenuItem(title, tooltip)}
}

func (systrayBackend) Quit() {
	systray.Quit()
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
