package plugin

import (
	"os"
	"path/filepath"
	"strings"
)

// DirResolveOptions makes plugin directory resolution testable.
type DirResolveOptions struct {
	EnvPluginDir   string
	CWD            string
	ExecutablePath string
	UserConfigDir  string
	HomeDir        string
}

// ResolveDiscoveryDirs returns plugin discovery directories in priority order:
// explicit env override, dev ./plugins, packaged binary-adjacent plugins, user plugins.
func ResolveDiscoveryDirs(opts DirResolveOptions) []string {
	var dirs []string
	add := func(path string) {
		if path == "" {
			return
		}
		cleaned := filepath.Clean(path)
		for _, existing := range dirs {
			if existing == cleaned {
				return
			}
		}
		dirs = append(dirs, cleaned)
	}

	if opts.EnvPluginDir != "" {
		for _, path := range filepath.SplitList(opts.EnvPluginDir) {
			add(path)
		}
	}

	if opts.CWD != "" {
		add(filepath.Join(opts.CWD, "plugins"))
	}

	if opts.ExecutablePath != "" {
		add(filepath.Join(filepath.Dir(opts.ExecutablePath), "plugins"))
	}

	if opts.UserConfigDir != "" {
		add(filepath.Join(opts.UserConfigDir, "verstak", "plugins"))
	} else if opts.HomeDir != "" {
		add(filepath.Join(opts.HomeDir, ".config", "verstak", "plugins"))
	}

	return dirs
}

// DefaultDiscoveryDirs resolves discovery directories from the current process.
func DefaultDiscoveryDirs() []string {
	cwd, _ := os.Getwd()
	exe, _ := os.Executable()
	userConfig, _ := os.UserConfigDir()
	home, _ := os.UserHomeDir()
	return ResolveDiscoveryDirs(DirResolveOptions{
		EnvPluginDir:   strings.TrimSpace(os.Getenv("VERSTAK_PLUGIN_DIR")),
		CWD:            cwd,
		ExecutablePath: exe,
		UserConfigDir:  userConfig,
		HomeDir:        home,
	})
}
