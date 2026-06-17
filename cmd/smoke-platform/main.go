// Smoke-platform validates that the platform-test plugin is discovered correctly
// by the Verstak desktop runtime. This runs headless — no Wails GUI needed.
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

	if *testWorkspace {
		runWorkspaceTest(root)
		return
	}

	if *testContributions {
		runContributionsTest(root)
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

	// Register workspace capability (core service — always present when vault is open)
	if err := reg.Register("verstak-desktop", []string{"verstak/core/workspace/v1"}); err != nil {
		fmt.Printf("  ❌ register workspace capability: %v\n", err)
		allGood = false
	} else {
		fmt.Printf("  ✅ registered workspace capability\n")
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
	if totalCaps >= 9 {
		fmt.Printf("  ✅ total capabilities >= 9 (%d)\n", totalCaps)
	} else {
		fmt.Printf("  ❌ total capabilities < 9 (got %d, expected >= 9)\n", totalCaps)
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

	// Initialize vault
	v := vault.NewVault(nil)
	if err := v.CreateVault(vaultPath); err != nil {
		fmt.Printf("  ❌ create vault: %v\n", err)
		exitCode = 1
		return
	}
	fmt.Printf("  ✅ vault created\n")

	// Open the vault at the path returned by CreateVault (path/VerstakVault)
	openedPath := v.GetVaultPath()
	if err := v.OpenVault(openedPath); err != nil {
		fmt.Printf("  ❌ open vault: %v\n", err)
		exitCode = 1
		return
	}
	fmt.Printf("  ✅ vault opened at %s\n", openedPath)

	// Initialize plugin state
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

	// Check plugins.json
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

	// ── Summary ──
	fmt.Printf("\n=== summary ===\n")
	fmt.Printf("✅ enable/disable test passed\n")
}

// runContributionsTest tests the contribution registry lifecycle:
// 1. Creates a vault, discovers platform-test
// 2. Registers capabilities + contributions
// 3. Verifies contributions appear by name
// 4. Disables plugin → unregisters contributions → verifies gone
// 5. Re-enables → contributions return
// 6. Checks no duplicates after reload
func runContributionsTest(root string) {
	exitCode := 0
	defer func() {
		os.Exit(exitCode)
	}()

	fmt.Printf("=== smoke-platform: contribution registry test ===\n\n")

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

	// Initialize plugin state
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

	// Register capabilities
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
	for _, cap := range target.Manifest.Provides {
		_ = reg.Register(target.Manifest.ID, []string{cap})
	}
	totalCaps := len(reg.List())
	fmt.Printf("  ✅ registered %d capabilities (core + plugin)\n", totalCaps)

	// ── 1. Register contributions from platform-test ──
	fmt.Printf("\n[register contributions]\n")
	contribReg := contribution.NewRegistry()
	if target.Manifest.Contributes == nil {
		fmt.Printf("  ❌ platform-test has no contributions in manifest\n")
		exitCode = 1
		return
	}
	contribReg.Register(target.Manifest.ID, target.Manifest.Contributes)
	fmt.Printf("  ✅ contributions registered for %s\n", target.Manifest.ID)

	// ── 2. Verify contributions appear by name ──
	fmt.Printf("\n[verify contributions]\n")
	allGood := true

	// SidebarItems
	sidebarItems := contribReg.SidebarItems()
	sidebarNames := make([]string, len(sidebarItems))
	for i, item := range sidebarItems {
		sidebarNames[i] = item.Item.Title
	}
	fmt.Printf("  sidebarItems (%d): %v\n", len(sidebarItems), sidebarNames)
	if len(sidebarItems) == 0 {
		fmt.Printf("  ❌ no sidebarItems registered\n")
		allGood = false
	}

	// Views
	views := contribReg.Views()
	viewNames := make([]string, len(views))
	for i, v := range views {
		viewNames[i] = v.Item.Title
	}
	fmt.Printf("  views (%d): %v\n", len(views), viewNames)
	if len(views) == 0 {
		fmt.Printf("  ❌ no views registered\n")
		allGood = false
	}

	// SettingsPanels
	settingsPanels := contribReg.SettingsPanels()
	settingsNames := make([]string, len(settingsPanels))
	for i, s := range settingsPanels {
		settingsNames[i] = s.Item.Title
	}
	fmt.Printf("  settingsPanels (%d): %v\n", len(settingsPanels), settingsNames)
	if len(settingsPanels) == 0 {
		fmt.Printf("  ❌ no settingsPanels registered\n")
		allGood = false
	}

	// Commands
	commands := contribReg.Commands()
	cmdNames := make([]string, len(commands))
	for i, c := range commands {
		cmdNames[i] = c.Item.Title
	}
	fmt.Printf("  commands (%d): %v\n", len(commands), cmdNames)
	if len(commands) == 0 {
		fmt.Printf("  ❌ no commands registered\n")
		allGood = false
	}

	if !allGood {
		fmt.Printf("  ❌ some contribution types are missing\n")
		exitCode = 1
		return
	}
	fmt.Printf("  ✅ all contribution types present\n")

	// ── 3. Disable plugin → contributions unregistered ──
	fmt.Printf("\n[disable plugin → unregister contributions]\n")
	if err := psm.DisablePlugin("verstak.platform-test"); err != nil {
		fmt.Printf("  ❌ disable: %v\n", err)
		exitCode = 1
		return
	}
	fmt.Printf("  ✅ disabled platform-test\n")

	contribReg.Unregister("verstak.platform-test")
	remainingViews := len(contribReg.Views())
	remainingSidebar := len(contribReg.SidebarItems())
	remainingCommands := len(contribReg.Commands())
	remainingSettings := len(contribReg.SettingsPanels())
	fmt.Printf("  remaining: views=%d sidebar=%d commands=%d settings=%d\n",
		remainingViews, remainingSidebar, remainingCommands, remainingSettings)

	if remainingViews+remainingSidebar+remainingCommands+remainingSettings != 0 {
		fmt.Printf("  ❌ some contributions not unregistered\n")
		exitCode = 1
		return
	}
	fmt.Printf("  ✅ all contributions unregistered\n")

	// ── 4. Re-enable → contributions return ──
	fmt.Printf("\n[re-enable plugin → register contributions]\n")
	if err := psm.EnablePlugin("verstak.platform-test"); err != nil {
		fmt.Printf("  ❌ enable: %v\n", err)
		exitCode = 1
		return
	}
	fmt.Printf("  ✅ enabled platform-test\n")

	contribReg.Register(target.Manifest.ID, target.Manifest.Contributes)
	views2 := contribReg.Views()
	sidebar2 := contribReg.SidebarItems()
	commands2 := contribReg.Commands()
	settings2 := contribReg.SettingsPanels()

	fmt.Printf("  after re-register: views=%d sidebar=%d commands=%d settings=%d\n",
		len(views2), len(sidebar2), len(commands2), len(settings2))

	if len(views2) == 0 || len(sidebar2) == 0 || len(commands2) == 0 || len(settings2) == 0 {
		fmt.Printf("  ❌ contributions did not return after re-enable\n")
		exitCode = 1
		return
	}
	fmt.Printf("  ✅ contributions returned after re-register\n")

	// ── 5. Re-register (simulate reload) → no duplicates ──
	fmt.Printf("\n[re-register (reload) → no duplicates]\n")
	contribReg.Register(target.Manifest.ID, target.Manifest.Contributes)
	views3 := contribReg.Views()
	sidebar3 := contribReg.SidebarItems()
	commands3 := contribReg.Commands()
	settings3 := contribReg.SettingsPanels()

	fmt.Printf("  after reload: views=%d sidebar=%d commands=%d settings=%d\n",
		len(views3), len(sidebar3), len(commands3), len(settings3))

	if len(views3) != len(views2) {
		fmt.Printf("  ❌ duplicate views after reload: before=%d, after=%d\n", len(views2), len(views3))
		allGood = false
	}
	if len(sidebar3) != len(sidebar2) {
		fmt.Printf("  ❌ duplicate sidebarItems after reload: before=%d, after=%d\n", len(sidebar2), len(sidebar3))
		allGood = false
	}
	if len(commands3) != len(commands2) {
		fmt.Printf("  ❌ duplicate commands after reload: before=%d, after=%d\n", len(commands2), len(commands3))
		allGood = false
	}
	if len(settings3) != len(settings2) {
		fmt.Printf("  ❌ duplicate settingsPanels after reload: before=%d, after=%d\n", len(settings2), len(settings3))
		allGood = false
	}

	if !allGood {
		fmt.Printf("  ❌ some duplicates detected\n")
		exitCode = 1
		return
	}
	fmt.Printf("  ✅ no duplicates after reload\n")

	// ── Summary ──
	fmt.Printf("\n=== summary ===\n")
	fmt.Printf("✅ contribution registry test passed\n")
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
	// Create: root → folder1 → folder2 → case (4 levels)
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
	if len(tree.Nodes) != 7 { // root + case + folder + nested + folder1 + folder2 + deepCase
		fmt.Printf("  ❌ expected 7 nodes, got %d\n", len(tree.Nodes))
		exitCode = 1
		return
	}
	fmt.Printf("  ✅ tree has 7 nodes (4 levels deep)\n")

	// Verify deep case parent chain
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
