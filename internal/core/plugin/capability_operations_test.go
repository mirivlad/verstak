package plugin

import (
	"strings"
	"testing"
)

func TestValidateManifestCapabilityOperations(t *testing.T) {
	valid := &Manifest{
		SchemaVersion: 1,
		ID:            "provider.plugin",
		Name:          "Provider",
		Version:       "1.0.0",
		APIVersion:    "1.0",
		Provides:      []string{"example/cap/v1"},
		Permissions:   []string{"commands.register"},
		CapabilityOperations: map[string]map[string]string{
			"example/cap/v1": {"list": "provider.list"},
		},
		Contributes: &Contributions{Commands: []ContributionCommand{{
			ID: "provider.list", Title: "List", Handler: "provider.list",
		}}},
	}
	if errs := ValidateManifest(valid); len(errs) != 0 {
		t.Fatalf("valid capability operations errors = %v", errs)
	}

	invalidCapability := *valid
	invalidCapability.CapabilityOperations = map[string]map[string]string{"other/cap/v1": {"list": "provider.list"}}
	if got := strings.Join(ValidateManifest(&invalidCapability), "\n"); !strings.Contains(got, "declared in provides") {
		t.Fatalf("missing provided-capability error: %q", got)
	}

	invalidCommand := *valid
	invalidCommand.CapabilityOperations = map[string]map[string]string{"example/cap/v1": {"list": "provider.missing"}}
	if got := strings.Join(ValidateManifest(&invalidCommand), "\n"); !strings.Contains(got, "undeclared command") {
		t.Fatalf("missing command error: %q", got)
	}
}
