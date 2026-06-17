package contribution

import (
	"testing"

	"github.com/verstak/verstak-desktop/internal/core/plugin"
)

// TestRegister_AddsContributions registers sidebar, view, command, settings contributions
// for plugin "test.plugin" and verifies they appear via Views(), Commands(),
// SettingsPanels(), SidebarItems().
func TestRegister_AddsContributions(t *testing.T) {
	r := NewRegistry()

	contribs := &plugin.Contributions{
		Views: []plugin.ContributionView{
			{ID: "test.view1", Title: "View 1", Component: "TestComponent"},
		},
		Commands: []plugin.ContributionCommand{
			{ID: "test.cmd1", Title: "Command 1"},
		},
		SettingsPanels: []plugin.ContributionSettingsPanel{
			{ID: "test.settings1", Title: "Settings 1", Component: "SettingsComponent"},
		},
		SidebarItems: []plugin.ContributionSidebarItem{
			{ID: "test.sidebar1", Title: "Sidebar 1", Icon: "icon", View: "test.view1", Position: 1},
		},
	}

	r.Register("test.plugin", contribs)

	// Verify counts
	if got := len(r.Views()); got != 1 {
		t.Errorf("Views(): got %d, want 1", got)
	}
	if got := len(r.Commands()); got != 1 {
		t.Errorf("Commands(): got %d, want 1", got)
	}
	if got := len(r.SettingsPanels()); got != 1 {
		t.Errorf("SettingsPanels(): got %d, want 1", got)
	}
	if got := len(r.SidebarItems()); got != 1 {
		t.Errorf("SidebarItems(): got %d, want 1", got)
	}

	// Verify the PluginID is set correctly
	if r.Views()[0].PluginID != "test.plugin" {
		t.Errorf("Views()[0].PluginID = %q, want %q", r.Views()[0].PluginID, "test.plugin")
	}
	if r.Commands()[0].PluginID != "test.plugin" {
		t.Errorf("Commands()[0].PluginID = %q, want %q", r.Commands()[0].PluginID, "test.plugin")
	}
	if r.SettingsPanels()[0].PluginID != "test.plugin" {
		t.Errorf("SettingsPanels()[0].PluginID = %q, want %q", r.SettingsPanels()[0].PluginID, "test.plugin")
	}
	if r.SidebarItems()[0].PluginID != "test.plugin" {
		t.Errorf("SidebarItems()[0].PluginID = %q, want %q", r.SidebarItems()[0].PluginID, "test.plugin")
	}

	// Verify item data is preserved
	if r.Views()[0].Item.Title != "View 1" {
		t.Errorf("Views()[0].Item.Title = %q, want %q", r.Views()[0].Item.Title, "View 1")
	}
	if r.Commands()[0].Item.Title != "Command 1" {
		t.Errorf("Commands()[0].Item.Title = %q, want %q", r.Commands()[0].Item.Title, "Command 1")
	}
	if r.SettingsPanels()[0].Item.Title != "Settings 1" {
		t.Errorf("SettingsPanels()[0].Item.Title = %q, want %q", r.SettingsPanels()[0].Item.Title, "Settings 1")
	}
	if r.SidebarItems()[0].Item.Title != "Sidebar 1" {
		t.Errorf("SidebarItems()[0].Item.Title = %q, want %q", r.SidebarItems()[0].Item.Title, "Sidebar 1")
	}
}

// TestUnregister_RemovesOwnedContributions registers for two plugins, unregisters one,
// and verifies only that plugin's contributions are removed.
func TestUnregister_RemovesOwnedContributions(t *testing.T) {
	r := NewRegistry()

	contribA := &plugin.Contributions{
		Views: []plugin.ContributionView{
			{ID: "a.view1", Title: "A View", Component: "A"},
		},
		Commands: []plugin.ContributionCommand{
			{ID: "a.cmd1", Title: "A Command"},
		},
		SettingsPanels: []plugin.ContributionSettingsPanel{
			{ID: "a.settings1", Title: "A Settings", Component: "A"},
		},
		SidebarItems: []plugin.ContributionSidebarItem{
			{ID: "a.sidebar1", Title: "A Sidebar", View: "a.view1"},
		},
	}

	contribB := &plugin.Contributions{
		Views: []plugin.ContributionView{
			{ID: "b.view1", Title: "B View", Component: "B"},
		},
		Commands: []plugin.ContributionCommand{
			{ID: "b.cmd1", Title: "B Command"},
		},
		SettingsPanels: []plugin.ContributionSettingsPanel{
			{ID: "b.settings1", Title: "B Settings", Component: "B"},
		},
		SidebarItems: []plugin.ContributionSidebarItem{
			{ID: "b.sidebar1", Title: "B Sidebar", View: "b.view1"},
		},
	}

	r.Register("plugin.a", contribA)
	r.Register("plugin.b", contribB)

	// Unregister plugin.a
	r.Unregister("plugin.a")

	// Verify plugin.a contributions are removed
	if got := r.Views(); len(got) != 1 || got[0].PluginID != "plugin.b" {
		t.Errorf("Views: got %d items (first PluginID=%q), want 1 from plugin.b", len(got), safePluginIDView(got))
	}
	if got := r.Commands(); len(got) != 1 || got[0].PluginID != "plugin.b" {
		t.Errorf("Commands: got %d items (first PluginID=%q), want 1 from plugin.b", len(got), safePluginIDCmd(got))
	}
	if got := r.SettingsPanels(); len(got) != 1 || got[0].PluginID != "plugin.b" {
		t.Errorf("SettingsPanels: got %d items (first PluginID=%q), want 1 from plugin.b", len(got), safePluginIDSettings(got))
	}
	if got := r.SidebarItems(); len(got) != 1 || got[0].PluginID != "plugin.b" {
		t.Errorf("SidebarItems: got %d items (first PluginID=%q), want 1 from plugin.b", len(got), safePluginIDSidebar(got))
	}

	// Verify plugin.b data is intact
	if r.Views()[0].Item.ID != "b.view1" {
		t.Errorf("Remaining View ID: got %q, want %q", r.Views()[0].Item.ID, "b.view1")
	}
	if r.Commands()[0].Item.ID != "b.cmd1" {
		t.Errorf("Remaining Command ID: got %q, want %q", r.Commands()[0].Item.ID, "b.cmd1")
	}
	if r.SettingsPanels()[0].Item.ID != "b.settings1" {
		t.Errorf("Remaining SettingsPanel ID: got %q, want %q", r.SettingsPanels()[0].Item.ID, "b.settings1")
	}
	if r.SidebarItems()[0].Item.ID != "b.sidebar1" {
		t.Errorf("Remaining SidebarItem ID: got %q, want %q", r.SidebarItems()[0].Item.ID, "b.sidebar1")
	}
}

// safe helpers for error messages when slices are empty
func safePluginIDView(items []ContributionView) string {
	if len(items) == 0 {
		return "<empty>"
	}
	return items[0].PluginID
}
func safePluginIDCmd(items []ContributionCommand) string {
	if len(items) == 0 {
		return "<empty>"
	}
	return items[0].PluginID
}
func safePluginIDSettings(items []ContributionSettingsPanel) string {
	if len(items) == 0 {
		return "<empty>"
	}
	return items[0].PluginID
}
func safePluginIDSidebar(items []ContributionSidebarItem) string {
	if len(items) == 0 {
		return "<empty>"
	}
	return items[0].PluginID
}

// TestListByPoint registers various types and calls ListByPoint for each point type,
// verifying correct counts.
func TestListByPoint(t *testing.T) {
	r := NewRegistry()

	contrib := &plugin.Contributions{
		Views:              []plugin.ContributionView{{ID: "v1", Title: "V1", Component: "C"}},
		Commands:           []plugin.ContributionCommand{{ID: "c1", Title: "C1"}},
		SettingsPanels:     []plugin.ContributionSettingsPanel{{ID: "s1", Title: "S1", Component: "C"}},
		SidebarItems:       []plugin.ContributionSidebarItem{{ID: "si1", Title: "SI1", View: "v1"}},
		FileActions:        []plugin.ContributionAction{{ID: "fa1", Label: "FA1"}},
		NoteActions:        []plugin.ContributionAction{{ID: "na1", Label: "NA1"}},
		ContextMenuEntries: []plugin.ContributionContextMenuEntry{{ID: "cm1", Label: "CM1", Context: "file"}},
		SearchProviders:    []plugin.ContributionSearchProvider{{ID: "sp1", Label: "SP1", Handler: "h"}},
		ActivityProviders:  []plugin.ContributionActivityProvider{{ID: "ap1", Events: []string{"test"}, Handler: "h"}},
		StatusBarItems:     []plugin.ContributionStatusBarItem{{ID: "sb1", Label: "SB1"}},
	}

	r.Register("test.plugin", contrib)

	tests := []struct {
		point ContributionPointType
		want  int
	}{
		{PointViews, 1},
		{PointCommands, 1},
		{PointSettingsPanels, 1},
		{PointSidebarItems, 1},
		{PointFileActions, 1},
		{PointNoteActions, 1},
		{PointContextMenus, 1},
		{PointSearchProviders, 1},
		{PointActivity, 1},
		{PointStatusBar, 1},
	}

	for _, tt := range tests {
		got := r.ListByPoint(tt.point)
		if len(got) != tt.want {
			t.Errorf("ListByPoint(%q): got %d items, want %d", tt.point, len(got), tt.want)
		}
	}
}

// TestRegister_DuplicatePrevention calls Register twice for the same plugin
// (simulating reload) and checks contributions appear only once (no duplicates).
// This is the KEY TEST for idempotent re-registration.
func TestRegister_DuplicatePrevention(t *testing.T) {
	r := NewRegistry()

	contrib := &plugin.Contributions{
		Views: []plugin.ContributionView{
			{ID: "test.view1", Title: "View 1", Component: "C"},
		},
		Commands: []plugin.ContributionCommand{
			{ID: "test.cmd1", Title: "Cmd 1"},
		},
		SettingsPanels: []plugin.ContributionSettingsPanel{
			{ID: "test.settings1", Title: "Settings 1", Component: "C"},
		},
		SidebarItems: []plugin.ContributionSidebarItem{
			{ID: "test.sidebar1", Title: "Sidebar 1", View: "test.view1"},
		},
	}

	// First registration
	r.Register("test.plugin", contrib)
	// Second registration — simulates plugin reload
	r.Register("test.plugin", contrib)

	// Each type should have only 1 entry (no duplicates)
	if got := len(r.Views()); got != 1 {
		t.Errorf("Views after double Register: got %d, want 1 (no duplicates)", got)
	}
	if got := len(r.Commands()); got != 1 {
		t.Errorf("Commands after double Register: got %d, want 1 (no duplicates)", got)
	}
	if got := len(r.SettingsPanels()); got != 1 {
		t.Errorf("SettingsPanels after double Register: got %d, want 1 (no duplicates)", got)
	}
	if got := len(r.SidebarItems()); got != 1 {
		t.Errorf("SidebarItems after double Register: got %d, want 1 (no duplicates)", got)
	}

	// Also verify the item data is preserved correctly
	if r.Views()[0].Item.ID != "test.view1" {
		t.Errorf("View ID after reload: got %q, want %q", r.Views()[0].Item.ID, "test.view1")
	}
	if r.Commands()[0].Item.ID != "test.cmd1" {
		t.Errorf("Command ID after reload: got %q, want %q", r.Commands()[0].Item.ID, "test.cmd1")
	}
	if r.SettingsPanels()[0].Item.ID != "test.settings1" {
		t.Errorf("SettingsPanel ID after reload: got %q, want %q", r.SettingsPanels()[0].Item.ID, "test.settings1")
	}
	if r.SidebarItems()[0].Item.ID != "test.sidebar1" {
		t.Errorf("SidebarItem ID after reload: got %q, want %q", r.SidebarItems()[0].Item.ID, "test.sidebar1")
	}
}

// TestUnregister_NoSideEffects verifies that unregistering a non-existent plugin
// doesn't crash or corrupt the registry.
func TestUnregister_NoSideEffects(t *testing.T) {
	r := NewRegistry()

	// Register a plugin
	contrib := &plugin.Contributions{
		Views: []plugin.ContributionView{
			{ID: "v1", Title: "V1", Component: "C"},
		},
	}
	r.Register("existing.plugin", contrib)

	// Unregister a plugin that was never registered — should not panic
	r.Unregister("nonexistent.plugin")

	// Existing plugin's contributions should still be intact
	if got := len(r.Views()); got != 1 {
		t.Errorf("Views after unregistering non-existent: got %d, want 1", got)
	}
	if r.Views()[0].PluginID != "existing.plugin" {
		t.Errorf("PluginID after unregistering non-existent: got %q, want %q", r.Views()[0].PluginID, "existing.plugin")
	}

	// Unregister with empty string — should not panic
	r.Unregister("")

	// Still intact
	if got := len(r.Views()); got != 1 {
		t.Errorf("Views after unregistering empty string: got %d, want 1", got)
	}
}
