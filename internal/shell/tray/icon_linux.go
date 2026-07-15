//go:build linux

package tray

import _ "embed"

//go:embed verstak.png
var defaultIcon []byte

// DefaultIcon returns the Linux PNG tray icon embedded in the binary.
func DefaultIcon() []byte {
	return append([]byte(nil), defaultIcon...)
}

// IconFileExtension is used when the backend materializes the embedded icon.
func IconFileExtension() string { return ".png" }
