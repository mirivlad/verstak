package plugin

import (
	"fmt"
	"strconv"
	"strings"
)

// HostAPIVersion is the plugin API this build of Verstak implements. A
// manifest's apiVersion names the API the plugin was written against.
const HostAPIVersion = "0.1.0"

// CheckAPICompatibility reports why a plugin written against pluginAPI cannot
// run on hostAPI, or "" when it can.
//
// The rule follows semantic versioning: the major version must match, and
// while the API is still 0.x a minor bump may break, so the minor must match
// too. Within a compatible line a plugin may target an older API than the
// host, never a newer one — the host would lack what the plugin calls.
func CheckAPICompatibility(pluginAPI, hostAPI string) string {
	plugin, err := parseAPIVersion(pluginAPI)
	if err != nil {
		return fmt.Sprintf("apiVersion %q is not a version: %v", pluginAPI, err)
	}
	host, err := parseAPIVersion(hostAPI)
	if err != nil {
		return fmt.Sprintf("host API version %q is not a version: %v", hostAPI, err)
	}
	incompatible := fmt.Sprintf("plugin targets API %s; this Verstak provides API %s", pluginAPI, hostAPI)
	if plugin[0] != host[0] {
		return incompatible
	}
	if host[0] == 0 && plugin[1] != host[1] {
		return incompatible
	}
	for i := 1; i < 3; i++ {
		if plugin[i] != host[i] {
			if plugin[i] > host[i] {
				return incompatible
			}
			break
		}
	}
	return ""
}

func parseAPIVersion(value string) ([3]int, error) {
	var out [3]int
	parts := strings.Split(strings.TrimPrefix(strings.TrimSpace(value), "v"), ".")
	if len(parts) != 3 {
		return out, fmt.Errorf("want MAJOR.MINOR.PATCH")
	}
	for i, part := range parts {
		n, err := strconv.Atoi(part)
		if err != nil || n < 0 {
			return out, fmt.Errorf("part %q is not a non-negative integer", part)
		}
		out[i] = n
	}
	return out, nil
}
