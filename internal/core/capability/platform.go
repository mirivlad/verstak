package capability

const CorePluginID = "verstak-desktop"

var platformCapabilities = []string{
	"verstak/core/plugin-manager/v1",
	"verstak/core/capability-registry/v1",
	"verstak/core/contribution-registry/v1",
	"verstak/core/permissions/v1",
	"verstak/core/events/v1",
	"verstak/core/files/v1",
	"verstak/core/workbench/v1",
	"verstak/core/notifications/v1",
	"verstak/core/workspace/v1",
}

// CorePlatformCapabilities returns a copy of the capabilities registered by
// the desktop before dynamic plugins are resolved.
func CorePlatformCapabilities() []string {
	return append([]string(nil), platformCapabilities...)
}
