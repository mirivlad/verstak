// Package buildinfo reports which build of Verstak is running.
//
// Nothing in the application used to say this, so a tester had no way to tell
// a freshly installed package from an older one that was still on PATH — a
// fix that did not appear and a fix that was never installed look identical.
package buildinfo

import (
	"runtime/debug"
	"strings"
	"sync"
)

// Set through -ldflags at build time, for example:
//
//	go build -ldflags "-X .../buildinfo.version=v0.1.0 -X .../buildinfo.commit=abc1234"
var (
	version   = ""
	commit    = ""
	buildDate = ""
)

// Info describes the running build.
type Info struct {
	// Version is a release tag such as "v0.1.0-beta.3", or "dev" when the
	// binary was built without one.
	Version string `json:"version"`
	// Commit is the short revision the binary was built from, when known.
	Commit string `json:"commit"`
	// BuildDate is an RFC 3339 timestamp, when known.
	BuildDate string `json:"buildDate"`
	// Display is the single string the interface shows.
	Display string `json:"display"`
}

var (
	once   sync.Once
	cached Info
)

// Get returns the running build's identity.
func Get() Info {
	once.Do(func() { cached = resolve() })
	return cached
}

func resolve() Info {
	info := Info{
		Version:   strings.TrimSpace(version),
		Commit:    shortCommit(strings.TrimSpace(commit)),
		BuildDate: strings.TrimSpace(buildDate),
	}

	// A build made without ldflags — `go build`, `wails dev` — can still say
	// something useful: the Go toolchain records the VCS revision.
	if info.Commit == "" || info.BuildDate == "" {
		if readable, ok := debug.ReadBuildInfo(); ok {
			for _, setting := range readable.Settings {
				switch setting.Key {
				case "vcs.revision":
					if info.Commit == "" {
						info.Commit = shortCommit(setting.Value)
					}
				case "vcs.time":
					if info.BuildDate == "" {
						info.BuildDate = setting.Value
					}
				case "vcs.modified":
					if setting.Value == "true" {
						info.Commit += "+dirty"
					}
				}
			}
		}
	}

	if info.Version == "" {
		info.Version = "dev"
	}

	info.Display = info.Version
	if info.Commit != "" {
		info.Display += " (" + info.Commit + ")"
	}
	return info
}

func shortCommit(value string) string {
	if len(value) > 7 && !strings.Contains(value, "+") {
		return value[:7]
	}
	return value
}
