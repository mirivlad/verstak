package capability

import "testing"

func TestRegisterDoesNotPartiallyRegisterCapabilitiesOnConflict(t *testing.T) {
	registry := NewRegistry()
	if err := registry.Register("existing.plugin", []string{"shared.capability"}); err != nil {
		t.Fatalf("register existing capability: %v", err)
	}

	if err := registry.Register("failed.plugin", []string{"new.capability", "shared.capability"}); err == nil {
		t.Fatal("Register returned nil for a duplicate capability")
	}
	if registry.Has("new.capability") {
		t.Fatal("Register leaked a capability from the failed registration")
	}
	entry := registry.Get("shared.capability")
	if entry == nil || entry.PluginID != "existing.plugin" {
		t.Fatalf("shared capability entry = %#v, want existing.plugin", entry)
	}
}
