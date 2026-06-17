// Package api provides Wails-bound methods for the frontend.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/verstak/verstak-desktop/internal/core/capability"
	"github.com/verstak/verstak-desktop/internal/core/contribution"
	"github.com/verstak/verstak-desktop/internal/core/plugin"
	"github.com/verstak/verstak-desktop/internal/core/pluginstate"
	"github.com/verstak/verstak-desktop/internal/core/vault"
	"github.com/verstak/verstak-desktop/internal/core/workspace"
)

func main() {
	testEnableDisable := flag.Bool("test-enable-disable", false, "Test enable/disable lifecycle")
	testWorkspace := flag.Bool("test-workspace", false, "Test workspace/cases lifecycle")
	testContributions := flag.Bool("test-contributions", false, "Test contribution registry lifecycle")
	flag.Parse()
	exitCode := 0
	defer func() {
		os.Exit(exitCode)
	}()

	root, _ := os.Getwd()
	pluginDir := filepath.Join(root, "plugins")

	if *testContributions {
		runContributionsTest(root)
		return
	}

	if *testWorkspace {
		runWorkspaceTest(root)
		return
	}

	if *testEnableDisable {
		runEnableDisableTest(root)
		return
	}

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

	if err := reg.Register("verstak-desktop", []string{"verstak/core/vault/v1"}); err != nil {
		fmt.Printf("  ❌ register vault capability: %v\n", err)
		allGood = false
	} else {
		fmt.Printf("  ✅ registered vault capability\n")
	}

	if err := reg.Register("verstak-desktop", []string{"verstak/core/workspace/v1"}); err != nil {
		fmt.Printf("  ❌ register workspace capability: %v\n", err)
		allGood = false
	} else {
		fmt.Printf("  ✅ registered workspace capability\n")
	}

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

	// ── 9. Verify workspace capability by name ──
	fmt.Printf("\n[workspace capability verification]\n")
	if reg.Has("verstak/core/workspace/v1") {
		fmt.Printf("  ✅ workspace capability present: verstak/core/workspace/v1\n")
	} else {
		fmt.Printf("  ❌ workspace capability MISSING: verstak/core/workspace/v1\n")
		allGood = false
	}

	// ── 10. Verify required capabilities resolved ──
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

	// ── 11. Check optional capabilities ──
	fmt.Printf("\n[optional capability resolution]\n")
	missingOptional := reg.CheckRequired(m.OptionalRequires)
	if len(missingOptional) > 0 {
		for _, miss := range missingOptional {
			fmt.Printf("  ⚠️  missing optional (degraded): %s\n", miss)
		}
	}

	// ── 12. Determine expected status ──
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

	// ── 13. Total capability count ──
	fmt.Printf("\n[capability count]\n")
	totalCaps := len(reg.List())
	fmt.Printf("  total capabilities: %d\n", totalCaps)
	if totalCaps >= 9 {
		fmt.Printf("  ✅ total capabilities >= 9 (%d)\n", totalCaps)
	} else {
		fmt.Printf("  ❌ total capabilities < 9 (got %d, expected >= 9)\n", totalCaps)
		allGood = false
	}

	// ── 14. Summary ──
	fmt.Printf("\n=== summary ===\n")
	if allGood {
		fmt.Printf("✅ smoke-platform passed\n")
	} else {
		fmt.Printf("❌ smoke-platform failed\n")
		exitCode = 1
	}
}

// runEnableDisableTest tests the enable/disable lifecycle with vault plugin state.
func runEnableDisableTest(root string) {
	exitCode := 0
	defer func() {
		os.Exit(exitCode)
	}()

	fmt.Printf("=== smoke-platform: enable/disable test ===\n\n")

	// Create a temp vault
	tmpDir, err := os.MkdirTemp("", "verstak-smoke-*")
	if err != nil {
		fmt.Printf("  ❌ failed to create temp dir: %v\n", err)
		exitCode = 1
		return
	}
	defer os.RemoveAll(tmpDir)

	vaultPath := filepath.Join(tmpDir, "testvault")
	fmt.Printf("  vault path: %s\n", vaultPath)

	v := vault.NewVault(nil)
	if err := v.CreateVault(vaultPath); err != nil {
		fmt.Printf("  ❌ create vault: %v\n", err)
		exitCode = 1
		return
	}
	fmt.Printf("  ✅ vault created\n")

	openedPath := v.GetVaultPath()
	if err := v.OpenVault(openedPath); err != nil {
		fmt.Printf("  ❌ open vault: %v\n", err)
		exitCode = 1
		return
	}
	fmt.Printf("  ✅ vault opened at %s\n", openedPath)

	psm := pluginstate.NewManager(v)
	if err := psm.Load(); err != nil {
		fmt.Printf("  ❌ load plugin state: %v\n", err)
		exitCode = 1
		return
	}
	fmt.Printf("  ✅ plugin state loaded\n")

	// Discover plugins
	pluginDir := filepath.Join(root, "plugins")
	plugins, _ := plugin.DiscoverPlugins([]string{pluginDir})
	if len(plugins) == 0 {
		fmt.Printf("  ❌ no plugins discovered\n")
		exitCode = 1
		return
	}
	fmt.Printf("  ✅ discovered %d plugin(s)\n", len(plugins))

	var target *plugin.Plugin
	for i, p := range plugins {
		if p.Manifest.ID == "verstak.platform-test" {
			target = &plugins[i]
			break
		}
	}
	if target == nil {
		fmt.Printf("  ❌ platform-test not found\n")
		exitCode = 1
		return
	}
	fmt.Printf("  ✅ platform-test found\n")

	// Register capabilities
	reg := capability.NewRegistry()
	coreCaps := []string{
		"verstak/core/plugin-manager/v1",
		"verstak/core/capability-registry/v1",
		"verstak/core/contribution-registry/v1",
		"verstak/core/permissions/v1",
		"verstak/core/events/v1",
	}
	reg.Register("verstak-desktop", coreCaps)
	reg.Register("verstak-desktop", []string{"verstak/core/vault/v1"})
	for _, cap := range target.Manifest.Provides {
		reg.Register(target.Manifest.ID, []string{cap})
	}
	totalCaps := len(reg.List())
	fmt.Printf("  ✅ registered %d capabilities (core + plugin)\n", totalCaps)

	// ── Test 1: Disable platform-test ──
	fmt.Printf("\n[disable]\n")
	if err := psm.DisablePlugin("verstak.platform-test"); err != nil {
		fmt.Printf("  ❌ disable: %v\n", err)
		exitCode = 1
		return
	}
	fmt.Printf("  ✅ disabled platform-test\n")

	if !psm.IsDisabled("verstak.platform-test") {
		fmt.Printf("  ❌ IsDisabled returned false after disable\n")
		exitCode = 1
		return
	}
	fmt.Printf("  ✅ IsDisabled: true\n")

	if psm.IsEnabled("verstak.platform-test") {
		fmt.Printf("  ❌ IsEnabled returned true after disable\n")
		exitCode = 1
		return
	}
	fmt.Printf("  ✅ IsEnabled: false\n")

	state := psm.Get()
	found := false
	for _, dp := range state.DesiredPlugins {
		if dp.ID == "verstak.platform-test" {
			found = true
			break
		}
	}
	if found {
		fmt.Printf("  ✅ platform-test in desiredPlugins\n")
	} else {
		fmt.Printf("  ℹ️  platform-test not in desiredPlugins (ok if not recorded)\n")
	}

	// ── Test 2: Enable platform-test ──
	fmt.Printf("\n[enable]\n")
	if err := psm.EnablePlugin("verstak.platform-test"); err != nil {
		fmt.Printf("  ❌ enable: %v\n", err)
		exitCode = 1
		return
	}
	fmt.Printf("  ✅ enabled platform-test\n")

	if !psm.IsEnabled("verstak.platform-test") {
		fmt.Printf("  ❌ IsEnabled returned false after enable\n")
		exitCode = 1
		return
	}
	fmt.Printf("  ✅ IsEnabled: true\n")

	if psm.IsDisabled("verstak.platform-test") {
		fmt.Printf("  ❌ IsDisabled returned true after enable\n")
		exitCode = 1
		return
	}
	fmt.Printf("  ✅ IsDisabled: false\n")

	// ── Test 3: Verify plugins.json on disk ──
	fmt.Printf("\n[plugins.json verification]\n")
	statePath := filepath.Join(v.GetVaultPath(), ".verstak", "plugins.json")
	data, err := os.ReadFile(statePath)
	if err != nil {
		fmt.Printf("  ❌ read plugins.json: %v\n", err)
		exitCode = 1
		return
	}
	fmt.Printf("  ✅ plugins.json exists on disk\n")
	fmt.Printf("  content:\n%s\n", string(data))

	fmt.Printf("\n=== summary ===\n")
	fmt.Printf("✅ enable/disable test passed\n")
}

// runContributionsTest verifies the full manifest → discovery → registry pipeline,
// mirroring the real ReloadPlugins() flow in internal/api/app.go.
//
// Flow:
//  1. plugin.json on disk → DiscoverPlugins() → parsed manifest
//  2. Manifest contributions fragment displayed (proof it comes from real plugin.json)
//  3. Capability registration + resolution → determine plugin status
//  4. Contribution registration (gated by status: only loaded/degraded get contributions)
//  5. Verify all 4 contribution types present by name, data matches manifest
//  6. Disable plugin → Unregister → contributions gone
//  7. Re-enable → Unregister+Register (ReloadPlugins) → contributions return
//  8. Reload → Unregister+Register → no duplicates
func runContributionsTest(root string) {
	exitCode := 0
	defer func() {
		os.Exit(exitCode)
	}()

	fmt.Printf("=== smoke-platform: contribution registry test ===\n\n")

	// Create temp vault
	tmpDir, err := os.MkdirTemp("", "verstak-smoke-*")
	if err != nil {
		fmt.Printf("  ❌ failed to create temp dir: %v\n", err)
		exitCode = 1
		return
	}
	defer os.RemoveAll(tmpDir)

	vaultPath := filepath.Join(tmpDir, "testvault")
	fmt.Printf("  vault path: %s\n", vaultPath)

	v := vault.NewVault(nil)
	if err := v.CreateVault(vaultPath); err != nil {
		fmt.Printf("  ❌ create vault: %v\n", err)
		exitCode = 1
		return
	}
	fmt.Printf("  ✅ vault created\n")

	openedPath := v.GetVaultPath()
	if err := v.OpenVault(openedPath); err != nil {
		fmt.Printf("  ❌ open vault: %v\n", err)
		exitCode = 1
		return
	}
	fmt.Printf("  ✅ vault opened at %s\n", openedPath)

	psm := pluginstate.NewManager(v)
	if err := psm.Load(); err != nil {
		fmt.Printf("  ❌ load plugin state: %v\n", err)
		exitCode = 1
		return
	}
	fmt.Printf("  ✅ plugin state loaded\n")

	// ── Step 1: Discover plugins from disk (reads real plugin.json) ──
	pluginDir := filepath.Join(root, "plugins")
	plugins, _ := plugin.DiscoverPlugins([]string{pluginDir})
	if len(plugins) == 0 {
		fmt.Printf("  ❌ no plugins discovered\n")
		exitCode = 1
		return
	}
	fmt.Printf("  ✅ discovered %d plugin(s)\n", len(plugins))

	// Find platform-test
	var target *plugin.Plugin
	for i, p := range plugins {
		if p.Manifest.ID == "verstak.platform-test" {
			target = &plugins[i]
			break
		}
	}
	if target == nil {
		fmt.Printf("  ❌ platform-test not found\n")
		exitCode = 1
		return
	}
	fmt.Printf("  ✅ platform-test found: %s@%s\n", target.Manifest.ID, target.Manifest.Version)

	// ── Step 2: Show manifest contributions fragment (proof it comes from plugin.json) ──
	fmt.Printf("\n[manifest contributions from plugin.json]\n")
	fmt.Printf("  plugin: %s\n", target.Manifest.ID)
	if c := target.Manifest.Contributes; c != nil {
		fmt.Printf("  sidebarItems: %d\n", len(c.SidebarItems))
		for _, s := range c.SidebarItems {
			fmt.Printf("    - id=%q title=%q view=%q\n", s.ID, s.Title, s.View)
		}
		fmt.Printf("  views: %d\n", len(c.Views))
		for _, vw := range c.Views {
			fmt.Printf("    - id=%q title=%q component=%q\n", vw.ID, vw.Title, vw.Component)
		}
		fmt.Printf("  settingsPanels: %d\n", len(c.SettingsPanels))
		for _, s := range c.SettingsPanels {
			fmt.Printf("    - id=%q title=%q component=%q\n", s.ID, s.Title, s.Component)
		}
		fmt.Printf("  commands: %d\n", len(c.Commands))
		for _, cmd := range c.Commands {
			fmt.Printf("    - id=%q title=%q\n", cmd.ID, cmd.Title)
		}
	}

	// ── Step 3: Register capabilities (simulates main.go + ReloadPlugins) ──
	reg := capability.NewRegistry()
	coreCaps := []string{
		"verstak/core/plugin-manager/v1",
		"verstak/core/capability-registry/v1",
		"verstak/core/contribution-registry/v1",
		"verstak/core/permissions/v1",
		"verstak/core/events/v1",
	}
	_ = reg.Register("verstak-desktop", coreCaps)
	_ = reg.Register("verstak-desktop", []string{"verstak/core/vault/v1"})
	for _, capID := range target.Manifest.Provides {
		_ = reg.Register(target.Manifest.ID, []string{capID})
	}
	fmt.Printf("\n[capability registration]\n")
	fmt.Printf("  registered %d capabilities\n", len(reg.List()))

	// ── Step 4: ReloadPlugins flow with capability gating ──
	fmt.Printf("\n[register contributions (ReloadPlugins flow)]\n")
	contribReg := contribution.NewRegistry()

	// (a) Check disabled → skip
	if target.Status == plugin.StatusDisabled {
		fmt.Printf("  ❌ plugin is disabled — contributions should NOT be registered\n")
		exitCode = 1
		return
	}

	// (b) Capability resolution (same as ReloadPlugins)
	missingRequired := reg.CheckRequired(target.Manifest.Requires)
	missingOptional := reg.CheckRequired(target.Manifest.OptionalRequires)
	status := plugin.StatusLoaded
	if len(missingRequired) > 0 {
		status = plugin.StatusMissingRequiredCapability
	} else if len(missingOptional) > 0 {
		status = plugin.StatusDegraded
	}

	// (c) Only loaded/degraded get contributions
	if status == plugin.StatusLoaded || status == plugin.StatusDegraded {
		fmt.Printf("  plugin status=%s → contributions WILL be registered\n", status)
		contribReg.Register(target.Manifest.ID, target.Manifest.Contributes)
		fmt.Printf("  contributions registered for %s\n", target.Manifest.ID)
	} else {
		fmt.Printf("  ❌ plugin status=%s — contributions should NOT be registered\n", status)
		exitCode = 1
		return
	}

	// ── Step 5: Verify contributions by name, data matches manifest ──
	fmt.Printf("\n[verify contributions by name]\n")
	allGood := true

	sidebarItems := contribReg.SidebarItems()
	if len(sidebarItems) != 1 || sidebarItems[0].Item.Title != "Platform Test" {
		fmt.Printf("  ❌ sidebarItems mismatch: got %v\n", sidebarItems)
		allGood = false
	} else {
		fmt.Printf("  ✅ sidebarItem: plugin=%s id=%s title=%s\n",
			sidebarItems[0].PluginID, sidebarItems[0].Item.ID, sidebarItems[0].Item.Title)
	}

	views := contribReg.Views()
	if len(views) != 1 || views[0].Item.Title != "Platform Diagnostics" {
		fmt.Printf("  ❌ views mismatch: got %v\n", views)
		allGood = false
	} else {
		fmt.Printf("  ✅ view: plugin=%s id=%s title=%s component=%s\n",
			views[0].PluginID, views[0].Item.ID, views[0].Item.Title, views[0].Item.Component)
	}

	settingsPanels := contribReg.SettingsPanels()
	if len(settingsPanels) != 1 || settingsPanels[0].Item.Title != "Platform Test Settings" {
		fmt.Printf("  ❌ settingsPanels mismatch: got %v\n", settingsPanels)
		allGood = false
	} else {
		fmt.Printf("  ✅ settingsPanel: plugin=%s id=%s title=%s\n",
			settingsPanels[0].PluginID, settingsPanels[0].Item.ID, settingsPanels[0].Item.Title)
	}

	commands := contribReg.Commands()
	if len(commands) != 2 {
		fmt.Printf("  ❌ expected 2 commands, got %d\n", len(commands))
		allGood = false
	} else {
		for _, c := range commands {
			fmt.Printf("  command: plugin=%s id=%s title=%s\n", c.PluginID, c.Item.ID, c.Item.Title)
		}
	}

	if !allGood {
		fmt.Printf("  ❌ some contribution types missing or data mismatch\n")
		exitCode = 1
		return
	}
	fmt.Printf("  ✅ all 4 contribution types present, data matches manifest\n")

	// ── Step 6: Disable → Unregister → contributions removed ──
	fmt.Printf("\n[disable plugin → Unregister → contributions removed]\n")
	if err := psm.DisablePlugin("verstak.platform-test"); err != nil {
		fmt.Printf("  ❌ disable: %v\n", err)
		exitCode = 1
		return
	}
	fmt.Printf("  disabled platform-test\n")

	// ReloadPlugins: Unregister before next scan, disabled plugins don't get re-registered
	contribReg.Unregister("verstak.platform-test")
	remaining := len(contribReg.Views()) + len(contribReg.SidebarItems()) +
		len(contribReg.Commands()) + len(contribReg.SettingsPanels())
	fmt.Printf("  remaining contributions: %d\n", remaining)
	if remaining != 0 {
		fmt.Printf("  ❌ contributions not removed after disable\n")
		exitCode = 1
		return
	}
	fmt.Printf("  ✅ all contributions removed\n")

	// ── Step 7: Re-enable → contributions return via Unregister+Register ──
	fmt.Printf("\n[re-enable → Unregister+Register → contributions return]\n")
	if err := psm.EnablePlugin("verstak.platform-test"); err != nil {
		fmt.Printf("  ❌ enable: %v\n", err)
		exitCode = 1
		return
	}
	fmt.Printf("  enabled platform-test\n")

	// ReloadPlugins: Unregister before Register (prevents duplicates)
	contribReg.Unregister(target.Manifest.ID)
	contribReg.Register(target.Manifest.ID, target.Manifest.Contributes)
	count := len(contribReg.Views()) + len(contribReg.SidebarItems()) +
		len(contribReg.Commands()) + len(contribReg.SettingsPanels())
	fmt.Printf("  contributions after re-enable: %d\n", count)
	if count < 4 {
		fmt.Printf("  ❌ expected >= 4 contributions, got %d\n", count)
		exitCode = 1
		return
	}
	fmt.Printf("  ✅ contributions returned (count=%d)\n", count)

	// ── Step 8: Reload → Unregister+Register → no duplicates ──
	fmt.Printf("\n[reload → Unregister+Register → no duplicates]\n")
	beforeCount := count
	contribReg.Unregister(target.Manifest.ID)
	contribReg.Register(target.Manifest.ID, target.Manifest.Contributes)
	afterCount := len(contribReg.Views()) + len(contribReg.SidebarItems()) +
		len(contribReg.Commands()) + len(contribReg.SettingsPanels())
	fmt.Printf("  before reload: %d, after reload: %d\n", beforeCount, afterCount)
	if afterCount != beforeCount {
		fmt.Printf("  ❌ duplicate after reload: before=%d after=%d\n", beforeCount, afterCount)
		exitCode = 1
		return
	}
	fmt.Printf("  ✅ no duplicate contributions after reload (count=%d)\n", afterCount)

	fmt.Printf("\n=== summary ===\n")
	if exitCode == 0 {
		fmt.Printf("✅ contribution registry test passed\n")
	} else {
		fmt.Printf("❌ contribution registry test failed\n")
	}
}

func runWorkspaceTest(root string) {
	exitCode := 0
	defer func() {
		os.Exit(exitCode)
	}()

	fmt.Printf("=== smoke-platform: workspace test ===\n\n")

	tmpDir, err := os.MkdirTemp("", "verstak-ws-smoke-*")
	if err != nil {
		fmt.Printf("  ❌ failed to create temp dir: %v\n", err)
		exitCode = 1
		return
	}
	defer os.RemoveAll(tmpDir)

	vaultPath := filepath.Join(tmpDir, "testvault")
	fmt.Printf("  vault path: %s\n", vaultPath)

	v := vault.NewVault(nil)
	if err := v.CreateVault(vaultPath); err != nil {
		fmt.Printf("  ❌ create vault: %v\n", err)
		exitCode = 1
		return
	}
	fmt.Printf("  ✅ vault created\n")

	openedPath := v.GetVaultPath()
	if err := v.OpenVault(openedPath); err != nil {
		fmt.Printf("  ❌ open vault: %v\n", err)
		exitCode = 1
		return
	}
	fmt.Printf("  ✅ vault opened at %s\n", openedPath)

	fmt.Printf("\n[workspace init]\n")
	ws := workspace.NewManager(openedPath)
	if err := ws.Load(); err != nil {
		fmt.Printf("  ❌ load workspace: %v\n", err)
		exitCode = 1
		return
	}
	fmt.Printf("  ✅ workspace loaded\n")

	tree := ws.GetTree()
	if len(tree.Nodes) != 1 {
		fmt.Printf("  ❌ expected 1 root node, got %d\n", len(tree.Nodes))
		exitCode = 1
		return
	}
	fmt.Printf("  ✅ root node exists (id=%s)\n", tree.Nodes[0].ID)
	rootID := tree.Nodes[0].ID

	fmt.Printf("\n[workspace capability]\n")
	reg := capability.NewRegistry()
	reg.Register("verstak-desktop", []string{
		"verstak/core/plugin-manager/v1",
		"verstak/core/capability-registry/v1",
		"verstak/core/contribution-registry/v1",
		"verstak/core/permissions/v1",
		"verstak/core/events/v1",
	})
	reg.Register("verstak-desktop", []string{"verstak/core/vault/v1"})
	reg.Register("verstak-desktop", []string{"verstak/core/workspace/v1"})
	if !reg.Has("verstak/core/workspace/v1") {
		fmt.Printf("  ❌ workspace capability not registered\n")
		exitCode = 1
		return
	}
	fmt.Printf("  ✅ workspace capability registered\n")
	totalCaps := len(reg.List())
	if totalCaps < 7 {
		fmt.Printf("  ❌ expected >= 7 capabilities, got %d\n", totalCaps)
		exitCode = 1
		return
	}
	fmt.Printf("  ✅ total capabilities >= 7 (%d)\n", totalCaps)

	fmt.Printf("\n[create case]\n")
	caseNode, err := ws.CreateNode(rootID, workspace.TypeCase, "Test Case")
	if err != nil {
		fmt.Printf("  ❌ create case: %v\n", err)
		exitCode = 1
		return
	}
	fmt.Printf("  ✅ case created: %s\n", caseNode.Title)

	fmt.Printf("\n[create folder]\n")
	folderNode, err := ws.CreateNode(rootID, workspace.TypeFolder, "Test Folder")
	if err != nil {
		fmt.Printf("  ❌ create folder: %v\n", err)
		exitCode = 1
		return
	}
	fmt.Printf("  ✅ folder created: %s\n", folderNode.Title)

	fmt.Printf("\n[create nested case]\n")
	nestedCase, err := ws.CreateNode(folderNode.ID, workspace.TypeCase, "Nested Case")
	if err != nil {
		fmt.Printf("  ❌ create nested case: %v\n", err)
		exitCode = 1
		return
	}
	fmt.Printf("  ✅ nested case created: %s\n", nestedCase.Title)

	fmt.Printf("\n[tree structure]\n")
	tree = ws.GetTree()
	if len(tree.Nodes) != 4 {
		fmt.Printf("  ❌ expected 4 nodes, got %d\n", len(tree.Nodes))
		exitCode = 1
		return
	}
	fmt.Printf("  ✅ tree has 4 nodes\n")
	children := ws.ListChildren(rootID)
	if len(children) != 2 {
		fmt.Printf("  ❌ expected 2 root children, got %d\n", len(children))
		exitCode = 1
		return
	}
	fmt.Printf("  ✅ root has 2 children\n")

	fmt.Printf("\n[rename]\n")
	if err := ws.RenameNode(caseNode.ID, "Renamed Case"); err != nil {
		fmt.Printf("  ❌ rename: %v\n", err)
		exitCode = 1
		return
	}
	renamed, _ := ws.GetNode(caseNode.ID)
	if renamed.Title != "Renamed Case" {
		fmt.Printf("  ❌ rename failed: got %q\n", renamed.Title)
		exitCode = 1
		return
	}
	fmt.Printf("  ✅ renamed to %q\n", renamed.Title)

	fmt.Printf("\n[set current node]\n")
	if err := ws.SetCurrentNode(caseNode.ID); err != nil {
		fmt.Printf("  ❌ set current: %v\n", err)
		exitCode = 1
		return
	}
	current, err := ws.GetCurrentNode()
	if err != nil {
		fmt.Printf("  ❌ get current: %v\n", err)
		exitCode = 1
		return
	}
	if current.ID != caseNode.ID {
		fmt.Printf("  ❌ current node mismatch\n")
		exitCode = 1
		return
	}
	fmt.Printf("  ✅ current node: %s\n", current.Title)

	fmt.Printf("\n[archive]\n")
	if err := ws.ArchiveNode(folderNode.ID); err != nil {
		fmt.Printf("  ❌ archive: %v\n", err)
		exitCode = 1
		return
	}
	archived, _ := ws.GetNode(folderNode.ID)
	if archived.Status != workspace.StatusArchived {
		fmt.Printf("  ❌ archive failed: status=%s\n", archived.Status)
		exitCode = 1
		return
	}
	fmt.Printf("  ✅ archived %s\n", archived.Title)

	fmt.Printf("\n[reopen persistence]\n")
	ws2 := workspace.NewManager(openedPath)
	if err := ws2.Load(); err != nil {
		fmt.Printf("  ❌ reopen: %v\n", err)
		exitCode = 1
		return
	}
	tree2 := ws2.GetTree()
	if len(tree2.Nodes) != 4 {
		fmt.Printf("  ❌ expected 4 nodes after reopen, got %d\n", len(tree2.Nodes))
		exitCode = 1
		return
	}
	fmt.Printf("  ✅ tree persisted: %d nodes\n", len(tree2.Nodes))
	current2, err := ws2.GetCurrentNode()
	if err != nil {
		fmt.Printf("  ❌ get current after reopen: %v\n", err)
		exitCode = 1
		return
	}
	if current2.ID != caseNode.ID {
		fmt.Printf("  ❌ current node not persisted\n")
		exitCode = 1
		return
	}
	fmt.Printf("  ✅ current node persisted\n")

	fmt.Printf("\n[workspace.json verification]\n")
	wsPath := filepath.Join(openedPath, ".verstak", "workspace.json")
	wsData, err := os.ReadFile(wsPath)
	if err != nil {
		fmt.Printf("  ❌ read workspace.json: %v\n", err)
		exitCode = 1
		return
	}
	fmt.Printf("  ✅ workspace.json exists on disk\n")
	fmt.Printf("  content:\n%s\n", string(wsData))

	// ── Test 4-level deep tree ──
	fmt.Printf("\n[4-level deep tree]\n")
	folder1, err := ws.CreateNode(rootID, workspace.TypeFolder, "Level 1 Folder")
	if err != nil {
		fmt.Printf("  ❌ create folder1: %v\n", err)
		exitCode = 1
		return
	}
	fmt.Printf("  ✅ created: %s\n", folder1.Title)

	folder2, err := ws.CreateNode(folder1.ID, workspace.TypeFolder, "Level 2 Folder")
	if err != nil {
		fmt.Printf("  ❌ create folder2: %v\n", err)
		exitCode = 1
		return
	}
	fmt.Printf("  ✅ created: %s\n", folder2.Title)

	deepCase, err := ws.CreateNode(folder2.ID, workspace.TypeCase, "Deep Case")
	if err != nil {
		fmt.Printf("  ❌ create deep case: %v\n", err)
		exitCode = 1
		return
	}
	fmt.Printf("  ✅ created: %s (depth 4)\n", deepCase.Title)

	tree = ws.GetTree()
	if len(tree.Nodes) != 7 {
		fmt.Printf("  ❌ expected 7 nodes, got %d\n", len(tree.Nodes))
		exitCode = 1
		return
	}
	fmt.Printf("  ✅ tree has 7 nodes (4 levels deep)\n")

	deepNode, _ := ws.GetNode(deepCase.ID)
	if deepNode.ParentID != folder2.ID {
		fmt.Printf("  ❌ deep case parent mismatch\n")
		exitCode = 1
		return
	}
	fmt.Printf("  ✅ deep case parent chain correct\n")

	fmt.Printf("\n=== summary ===\n")
	fmt.Printf("✅ workspace test passed\n")
}
