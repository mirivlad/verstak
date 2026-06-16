// Smoke-platform validates that the platform-test plugin is discovered correctly
// by the Verstak desktop runtime. This runs headless — no Wails GUI needed.
package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/verstak/verstak-desktop/internal/core/capability"
	"github.com/verstak/verstak-desktop/internal/core/plugin"
)

func main() {
	exitCode := 0
	defer func() {
		os.Exit(exitCode)
	}()

	root, _ := os.Getwd()
	pluginDir := filepath.Join(root, "plugins")

	fmt.Printf("=== smoke-platform: headless plugin verification ===\n\n")
	fmt.Printf("  plugin dir: %s\n", pluginDir)

	// ── 1. Discover plugins ──
	fmt.Printf("\n[discovery]\n")
	plugins, errs := plugin.DiscoverPlugins([]string{pluginDir})

	if len(errs) > 0 {
		for _, e := range errs {
			fmt.Printf("  ⚠️  discovery warning: %v\n", e)
		}
	}

	if len(plugins) == 0 {
		fmt.Printf("  ❌ no plugins discovered\n")
		exitCode = 1
		return
	}

	fmt.Printf("  ✅ discovered %d plugin(s)\n", len(plugins))

	// ── 2. Find platform-test ──
	fmt.Printf("\n[platform-test lookup]\n")
	var target *plugin.Plugin
	for i, p := range plugins {
		if p.Manifest.ID == "verstak.platform-test" {
			target = &plugins[i]
			break
		}
	}

	if target == nil {
		fmt.Printf("  ❌ platform-test (id=verstak.platform-test) not found among discovered plugins\n")
		exitCode = 1
		return
	}
	fmt.Printf("  ✅ found: %s\n", target.Manifest.ID)

	// ── 3. Validate manifest fields ──
	fmt.Printf("\n[manifest validation]\n")
	allGood := true
	m := &target.Manifest

	checks := []struct {
		name  string
		value string
	}{
		{"name", m.Name},
		{"version", m.Version},
		{"apiVersion", m.APIVersion},
	}
	for _, c := range checks {
		if c.value == "" {
			fmt.Printf("  ❌ manifest.%s is empty\n", c.name)
			allGood = false
		} else {
			fmt.Printf("  ✅ %s: %s\n", c.name, c.value)
		}
	}

	if m.SchemaVersion != 1 {
		fmt.Printf("  ❌ schemaVersion: expected 1, got %d\n", m.SchemaVersion)
		allGood = false
	} else {
		fmt.Printf("  ✅ schemaVersion: 1\n")
	}

	// ── 4. provides ──
	fmt.Printf("\n[provides]\n")
	if len(m.Provides) == 0 {
		fmt.Printf("  ❌ provides is empty\n")
		allGood = false
	} else {
		for _, p := range m.Provides {
			fmt.Printf("  ✅ provides: %s\n", p)
		}
	}

	// ── 5. requires / optionalRequires ──
	fmt.Printf("\n[requires]\n")
	if len(m.Requires) > 0 {
		for _, r := range m.Requires {
			fmt.Printf("  ✅ requires: %s\n", r)
		}
	} else {
		fmt.Printf("  ℹ️  requires: none\n")
	}

	fmt.Printf("\n[optionalRequires]\n")
	if len(m.OptionalRequires) > 0 {
		for _, r := range m.OptionalRequires {
			fmt.Printf("  ✅ optionalRequires: %s\n", r)
		}
	} else {
		fmt.Printf("  ℹ️  optionalRequires: none\n")
	}

	// ── 6. contributions ──
	fmt.Printf("\n[contributions]\n")
	if m.Contributes != nil {
		c := m.Contributes
		if len(c.Views) > 0 {
			for _, v := range c.Views {
				fmt.Printf("  ✅ view: %s (%s)\n", v.ID, v.Title)
			}
		}
		if len(c.Commands) > 0 {
			for _, cmd := range c.Commands {
				fmt.Printf("  ✅ command: %s (%s)\n", cmd.ID, cmd.Title)
			}
		}
		if len(c.SidebarItems) > 0 {
			for _, s := range c.SidebarItems {
				fmt.Printf("  ✅ sidebarItem: %s (%s)\n", s.ID, s.Title)
			}
		}
		if len(c.StatusBarItems) > 0 {
			for _, s := range c.StatusBarItems {
				fmt.Printf("  ✅ statusBarItem: %s (%s)\n", s.ID, s.Label)
			}
		}
		if len(c.Views)+len(c.Commands)+len(c.SidebarItems)+len(c.StatusBarItems) == 0 {
			fmt.Printf("  ℹ️  contributes: empty sections only\n")
		}
	} else {
		fmt.Printf("  ℹ️  contributes: none\n")
	}

	// ── 7. Capability registration (core + plugin) ──
	fmt.Printf("\n[capability registration]\n")
	reg := capability.NewRegistry()

	// Register core capabilities (same list as main.go)
	coreCaps := []string{
		"verstak/core/plugin-manager/v1",
		"verstak/core/capability-registry/v1",
		"verstak/core/contribution-registry/v1",
		"verstak/core/permissions/v1",
		"verstak/core/events/v1",
	}
	if err := reg.Register("verstak-desktop", coreCaps); err != nil {
		fmt.Printf("  ❌ register core capabilities: %v\n", err)
		allGood = false
	} else {
		fmt.Printf("  ✅ registered %d core capabilities\n", len(coreCaps))
	}

	// Register vault capability (core service)
	if err := reg.Register("verstak-desktop", []string{"verstak/core/vault/v1"}); err != nil {
		fmt.Printf("  ❌ register vault capability: %v\n", err)
		allGood = false
	} else {
		fmt.Printf("  ✅ registered vault capability\n")
	}

	// Register plugin capabilities
	for _, p := range m.Provides {
		if err := reg.Register(m.ID, []string{p}); err != nil {
			fmt.Printf("  ❌ register capability %s: %v\n", p, err)
			allGood = false
		} else {
			fmt.Printf("  ✅ registered plugin capability: %s\n", p)
		}
	}

	// ── 8. Verify core capabilities present ──
	fmt.Printf("\n[core capability verification]\n")
	for _, capName := range coreCaps {
		if reg.Has(capName) {
			fmt.Printf("  ✅ core capability present: %s\n", capName)
		} else {
			fmt.Printf("  ❌ core capability MISSING: %s\n", capName)
			allGood = false
		}
	}

	// ── 9. Verify required capabilities resolved ──
	fmt.Printf("\n[required capability resolution]\n")
	missingRequired := reg.CheckRequired(m.Requires)
	if len(missingRequired) > 0 {
		for _, miss := range missingRequired {
			fmt.Printf("  ❌ MISSING required: %s\n", miss)
		}
		allGood = false
	} else {
		fmt.Printf("  ✅ all required capabilities resolved\n")
	}

	// ── 10. Check optional capabilities ──
	fmt.Printf("\n[optional capability resolution]\n")
	missingOptional := reg.CheckRequired(m.OptionalRequires)
	if len(missingOptional) > 0 {
		for _, miss := range missingOptional {
			fmt.Printf("  ⚠️  missing optional (degraded): %s\n", miss)
		}
	}

	// ── 11. Determine expected status ──
	fmt.Printf("\n[plugin status]\n")
	expectedStatus := "loaded"
	if len(missingOptional) > 0 {
		expectedStatus = "degraded"
	}
	if len(missingRequired) > 0 {
		expectedStatus = "missing-required-capability"
	}
	fmt.Printf("  ℹ️  expected status: %s\n", expectedStatus)
	if expectedStatus == "degraded" {
		fmt.Printf("  ✅ degraded is correct (optional capabilities missing, required OK)\n")
	} else if expectedStatus == "loaded" {
		fmt.Printf("  ✅ loaded is correct (all capabilities resolved)\n")
	} else {
		fmt.Printf("  ❌ unexpected: required capabilities should be resolved\n")
		allGood = false
	}

	// ── 12. Total capability count ──
	fmt.Printf("\n[capability count]\n")
	totalCaps := len(reg.List())
	fmt.Printf("  total capabilities: %d\n", totalCaps)
	if totalCaps >= 8 {
		fmt.Printf("  ✅ total capabilities >= 8 (%d)\n", totalCaps)
	} else {
		fmt.Printf("  ❌ total capabilities < 8 (got %d, expected >= 8)\n", totalCaps)
		allGood = false
	}

	// ── 13. Summary ──
	fmt.Printf("\n=== summary ===\n")
	if allGood {
		fmt.Printf("✅ smoke-platform passed\n")
	} else {
		fmt.Printf("❌ smoke-platform failed\n")
		exitCode = 1
	}
}
