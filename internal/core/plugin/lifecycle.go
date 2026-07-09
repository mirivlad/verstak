package plugin

import (
	"fmt"
	"strings"

	"github.com/verstak/verstak-desktop/internal/core/capability"
)

// ResolveLifecycle registers only viable plugin capabilities and assigns final
// lifecycle statuses after all required dependencies have had a chance to load.
func ResolveLifecycle(plugins []Plugin, registry *capability.Registry, isDisabled func(string) bool) {
	if registry == nil {
		for i := range plugins {
			plugins[i].Status = StatusFailed
			plugins[i].Error = "capability registry is unavailable"
		}
		return
	}

	pending := make(map[int]struct{}, len(plugins))
	for i := range plugins {
		p := &plugins[i]
		p.Error = ""
		if !p.Enabled || (isDisabled != nil && isDisabled(p.Manifest.ID)) {
			p.Enabled = false
			p.Status = StatusDisabled
			continue
		}
		pending[i] = struct{}{}
	}

	for len(pending) > 0 {
		progressed := false
		for i := range plugins {
			if _, ok := pending[i]; !ok {
				continue
			}
			p := &plugins[i]
			if len(registry.CheckRequired(p.Manifest.Requires)) > 0 {
				continue
			}
			if err := registry.Register(p.Manifest.ID, p.Manifest.Provides); err != nil {
				p.Status = StatusFailed
				p.Error = err.Error()
			} else {
				p.Status = StatusLoaded
			}
			delete(pending, i)
			progressed = true
		}
		if progressed {
			continue
		}

		for i := range pending {
			p := &plugins[i]
			missing := registry.CheckRequired(p.Manifest.Requires)
			p.Status = StatusMissingRequiredCapability
			p.Error = fmt.Sprintf("missing required: %s", strings.Join(missing, ", "))
		}
		break
	}

	for i := range plugins {
		p := &plugins[i]
		if p.Status != StatusLoaded {
			continue
		}
		if missing := registry.CheckRequired(p.Manifest.OptionalRequires); len(missing) > 0 {
			p.Status = StatusDegraded
			p.Error = fmt.Sprintf("missing optional: %s", strings.Join(missing, ", "))
		}
	}
}
