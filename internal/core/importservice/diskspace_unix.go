//go:build unix

package importservice

import "golang.org/x/sys/unix"

func availableDiskSpace(path string) (uint64, error) {
	var status unix.Statfs_t
	if err := unix.Statfs(path, &status); err != nil {
		return 0, err
	}
	return status.Bavail * uint64(status.Bsize), nil
}
