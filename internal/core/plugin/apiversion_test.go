package plugin

import (
	"strings"
	"testing"

	"github.com/verstak/verstak-desktop/internal/core/capability"
)

func TestCheckAPICompatibility(t *testing.T) {
	cases := []struct {
		plugin, host string
		ok           bool
	}{
		{"0.1.0", "0.1.0", true},
		{"0.1.0", "0.1.3", true},  // older patch on the same 0.x line
		{"0.1.4", "0.1.3", false}, // newer than the host
		{"0.2.0", "0.1.0", false}, // 0.x minor bumps may break
		{"0.1.0", "0.2.0", false},
		{"1.2.0", "1.4.1", true}, // older minor within a stable major
		{"1.5.0", "1.4.1", false},
		{"2.0.0", "1.4.1", false},
		{"1.0", "0.1.0", false}, // not MAJOR.MINOR.PATCH
		{"", "0.1.0", false},
		{"v0.1.0", "0.1.0", true},
	}
	for _, c := range cases {
		reason := CheckAPICompatibility(c.plugin, c.host)
		if (reason == "") != c.ok {
			t.Errorf("CheckAPICompatibility(%q, %q) = %q, want compatible=%v", c.plugin, c.host, reason, c.ok)
		}
	}
}

// A plugin built against another plugin API must be reported as incompatible
// and must not register what it provides: a dependant then says the capability
// is missing instead of calling into an API that is not there.
func TestResolveLifecycleMarksIncompatiblePluginsAndDoesNotRegisterThem(t *testing.T) {
	reg := capability.NewRegistry()
	plugins := []Plugin{
		{
			Manifest: Manifest{
				ID:          "future.plugin",
				APIVersion:  "0.9.0",
				Provides:    []string{"future.capability"},
				Permissions: []string{"vault.read"},
			},
			Enabled: true,
		},
		{
			Manifest: Manifest{
				ID:          "dependant.plugin",
				APIVersion:  HostAPIVersion,
				Provides:    []string{"dependant.capability"},
				Requires:    []string{"future.capability"},
				Permissions: []string{"vault.read"},
			},
			Enabled: true,
		},
	}

	ResolveLifecycle(plugins, reg, nil)

	if plugins[0].Status != StatusIncompatible {
		t.Fatalf("future.plugin status = %q, want %q", plugins[0].Status, StatusIncompatible)
	}
	if !strings.Contains(plugins[0].Error, "0.9.0") || !strings.Contains(plugins[0].Error, HostAPIVersion) {
		t.Errorf("future.plugin error %q should name both API versions", plugins[0].Error)
	}
	if missing := reg.CheckRequired([]string{"future.capability"}); len(missing) == 0 {
		t.Error("an incompatible plugin must not register its capabilities")
	}
	if plugins[1].Status != StatusMissingRequiredCapability {
		t.Errorf("dependant.plugin status = %q, want %q", plugins[1].Status, StatusMissingRequiredCapability)
	}
}

// Every official plugin must load on this host; a manifest bump that outruns
// HostAPIVersion should fail here rather than on a user's machine.
func TestShippedPluginManifestsTargetHostAPI(t *testing.T) {
	// The sibling source repository is the source of truth; ./plugins is a
	// gitignored install copy and would make this test machine-dependent.
	plugins, _ := DiscoverPlugins([]string{"../../../../verstak-official-plugins/plugins"})
	if len(plugins) == 0 {
		t.Skip("cannot judge: verstak-official-plugins is not checked out beside verstak-desktop")
	}
	for _, p := range plugins {
		if reason := CheckAPICompatibility(p.Manifest.APIVersion, HostAPIVersion); reason != "" {
			t.Errorf("%s: %s", p.Manifest.ID, reason)
		}
	}
}
