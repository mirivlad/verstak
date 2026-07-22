package permissions

import "testing"

func TestImportPermissionsAreKnownAndDangerous(t *testing.T) {
	registry := NewRegistry()
	for _, name := range []string{"imports.readExternal", "imports.apply"} {
		entry, ok := registry.Get(name)
		if !ok {
			t.Fatalf("permission %q is not registered", name)
		}
		if !entry.Dangerous {
			t.Fatalf("permission %q must be dangerous", name)
		}
	}
}
