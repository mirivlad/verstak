//go:build unix

package importservice

import (
	"os"
	"strings"

	"golang.org/x/sys/unix"
)

func secureOpenDirectoryFile(root, relativePath string) (*os.File, error) {
	parts := strings.Split(relativePath, "/")
	if len(parts) == 0 {
		return nil, sourceError("source-changed", "entry path is unavailable")
	}
	currentFD, err := unix.Open(root, unix.O_RDONLY|unix.O_CLOEXEC|unix.O_DIRECTORY|unix.O_NOFOLLOW, 0)
	if err != nil {
		return nil, sourceError("source-changed", "selected directory changed")
	}
	for index, part := range parts {
		flags := unix.O_RDONLY | unix.O_CLOEXEC | unix.O_NOFOLLOW
		if index < len(parts)-1 {
			flags |= unix.O_DIRECTORY
		}
		nextFD, openErr := unix.Openat(currentFD, part, flags, 0)
		_ = unix.Close(currentFD)
		if openErr != nil {
			return nil, sourceError("source-changed", "entry path changed")
		}
		currentFD = nextFD
	}
	file := os.NewFile(uintptr(currentFD), "source-entry")
	if file == nil {
		_ = unix.Close(currentFD)
		return nil, sourceError("source-unavailable", "could not open source entry")
	}
	return file, nil
}
