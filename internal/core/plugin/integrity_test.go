package plugin

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/verstak/verstak-desktop/internal/core/capability"
)

func writePackagedPlugin(t *testing.T, dir string, files map[string]string) {
	t.Helper()
	for name, body := range files {
		path := filepath.Join(dir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	var lines []string
	for name := range files {
		sum, err := hashFile(filepath.Join(dir, filepath.FromSlash(name)))
		if err != nil {
			t.Fatal(err)
		}
		lines = append(lines, fmt.Sprintf("%s  %s", sum, name))
	}
	if err := os.WriteFile(filepath.Join(dir, ChecksumsFile), []byte(strings.Join(lines, "\n")+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
}

const integrityManifest = `{"schemaVersion":1,"id":"verstak.sample","name":"Sample","version":"0.1.0","apiVersion":"0.1.0","provides":["sample"],"permissions":["storage.namespace"]}`

func TestVerifyPackageAcceptsAnUntouchedPackage(t *testing.T) {
	dir := t.TempDir()
	writePackagedPlugin(t, dir, map[string]string{
		"plugin.json":            integrityManifest,
		"frontend/dist/index.js": "console.log('sample');\n",
	})

	findings, err := VerifyPackage(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(findings) != 0 {
		t.Fatalf("findings = %#v", findings)
	}
}

// A plugin under development has no checksums, and must not be treated as
// damaged for it.
func TestVerifyPackageSaysNothingWithoutChecksums(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "plugin.json"), []byte(integrityManifest), 0o644); err != nil {
		t.Fatal(err)
	}
	findings, err := VerifyPackage(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(findings) != 0 {
		t.Fatalf("findings = %#v", findings)
	}
}

func TestVerifyPackageNamesChangedMissingAndExtraFiles(t *testing.T) {
	dir := t.TempDir()
	writePackagedPlugin(t, dir, map[string]string{
		"plugin.json":            integrityManifest,
		"frontend/dist/index.js": "console.log('sample');\n",
		"locales/en.json":        `{"a":"b"}`,
	})

	// The trap this exists for: an installed copy a version behind its source.
	if err := os.WriteFile(filepath.Join(dir, "frontend", "dist", "index.js"), []byte("console.log('stale');\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(dir, "locales", "en.json")); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "extra.js"), []byte("//\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	findings, err := VerifyPackage(dir)
	if err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(findings, "\n")
	for _, expected := range []string{
		"frontend/dist/index.js does not match the package",
		"locales/en.json is missing",
		"extra.js is not part of the package",
	} {
		if !strings.Contains(joined, expected) {
			t.Fatalf("findings do not report %q:\n%s", expected, joined)
		}
	}
}

func TestVerifyPackageRejectsAChecksumsFileItCannotRead(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "plugin.json"), []byte(integrityManifest), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ChecksumsFile), []byte("not a checksum line\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := VerifyPackage(dir); err == nil {
		t.Fatal("a checksums file that is not one was accepted")
	}
}

// A plugin whose files no longer match still runs -- refusing to start over a
// stale file costs more than the file does -- but the Plugin Manager says so,
// and lifecycle resolution must not wipe that on its way past.
func TestDiscoveredPluginKeepsItsIntegrityFindingThroughLifecycle(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "sample")
	writePackagedPlugin(t, dir, map[string]string{
		"plugin.json":            integrityManifest,
		"frontend/dist/index.js": "console.log('sample');\n",
	})
	if err := os.WriteFile(filepath.Join(dir, "frontend", "dist", "index.js"), []byte("console.log('stale');\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	plugins, errs := DiscoverPlugins([]string{root})
	if len(errs) != 0 || len(plugins) != 1 {
		t.Fatalf("plugins = %#v errs = %#v", plugins, errs)
	}
	if plugins[0].Integrity == "" {
		t.Fatal("discovery did not notice the changed file")
	}

	ResolveLifecycle(plugins, capability.NewRegistry(), nil)
	if plugins[0].Status != StatusDegraded {
		t.Fatalf("status = %s, want degraded", plugins[0].Status)
	}
	if !strings.Contains(plugins[0].Error, "does not match its checksums") {
		t.Fatalf("error = %q", plugins[0].Error)
	}
}
