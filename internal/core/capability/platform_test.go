package capability

import "testing"

func TestCorePlatformCapabilitiesIncludeImport(t *testing.T) {
	for _, name := range CorePlatformCapabilities() {
		if name == "verstak/core/import/v1" {
			return
		}
	}
	t.Fatal("verstak/core/import/v1 is not registered")
}
