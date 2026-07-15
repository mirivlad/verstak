//go:build windows

package tray

import _ "embed"

//go:embed verstak.ico
var defaultIcon []byte

// DefaultIcon returns the multi-resolution Windows ICO embedded in the binary.
func DefaultIcon() []byte {
	return append([]byte(nil), defaultIcon...)
}

// IconFileExtension is used when the backend materializes the embedded icon.
func IconFileExtension() string { return ".ico" }
