package plugin

import (
	"path/filepath"
	"reflect"
	"testing"
)

func TestResolveDiscoveryDirs_EnvCwdBinaryUserDedup(t *testing.T) {
	root := t.TempDir()
	envDir := filepath.Join(root, "env-plugins")
	cwdDir := filepath.Join(root, "repo", "plugins")
	binaryDir := filepath.Join(root, "app", "plugins")
	userConfigDir := filepath.Join(root, "config")

	got := ResolveDiscoveryDirs(DirResolveOptions{
		EnvPluginDir:   envDir + string(filepath.ListSeparator) + cwdDir,
		CWD:            filepath.Join(root, "repo"),
		ExecutablePath: filepath.Join(root, "app", "verstak-desktop"),
		UserConfigDir:  userConfigDir,
	})

	want := []string{
		envDir,
		cwdDir,
		binaryDir,
		filepath.Join(userConfigDir, "verstak", "plugins"),
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ResolveDiscoveryDirs() = %#v, want %#v", got, want)
	}
}

func TestResolveDiscoveryDirs_UsesCwdWhenExecutablePathMissing(t *testing.T) {
	root := t.TempDir()
	got := ResolveDiscoveryDirs(DirResolveOptions{
		CWD:     root,
		HomeDir: filepath.Join(root, "home"),
	})

	wantFirst := filepath.Join(root, "plugins")
	if got[0] != wantFirst {
		t.Fatalf("first plugin dir = %q, want cwd plugins %q", got[0], wantFirst)
	}
}

func TestResolveDiscoveryDirs_FallsBackToHomeConfigDir(t *testing.T) {
	root := t.TempDir()
	got := ResolveDiscoveryDirs(DirResolveOptions{
		HomeDir: filepath.Join(root, "home"),
	})

	want := []string{filepath.Join(root, "home", ".config", "verstak", "plugins")}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ResolveDiscoveryDirs() = %#v, want %#v", got, want)
	}
}

func TestResolveDiscoveryDirs_NormalizesAndDeduplicatesPaths(t *testing.T) {
	root := t.TempDir()
	cwd := filepath.Join(root, "repo")
	got := ResolveDiscoveryDirs(DirResolveOptions{
		EnvPluginDir: filepath.Join(cwd, ".", "plugins") + string(filepath.ListSeparator) + filepath.Join(cwd, "plugins"),
		CWD:          cwd,
	})

	want := []string{filepath.Join(cwd, "plugins")}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ResolveDiscoveryDirs() = %#v, want %#v", got, want)
	}
}
