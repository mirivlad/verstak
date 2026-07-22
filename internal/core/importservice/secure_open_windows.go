//go:build windows

package importservice

import (
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/sys/windows"
)

func secureOpenDirectoryFile(root, relativePath string) (*os.File, error) {
	rootFile, err := os.Open(root)
	if err != nil {
		return nil, sourceError("source-changed", "selected directory changed")
	}
	rootFinal, err := finalPathByHandle(rootFile)
	_ = rootFile.Close()
	if err != nil {
		return nil, sourceError("source-changed", "selected directory changed")
	}

	file, err := os.Open(filepath.Join(root, filepath.FromSlash(relativePath)))
	if err != nil {
		return nil, sourceError("source-changed", "entry path changed")
	}
	entryFinal, err := finalPathByHandle(file)
	if err != nil {
		_ = file.Close()
		return nil, sourceError("source-changed", "entry path changed")
	}
	rootPrefix := strings.TrimRight(strings.ToLower(rootFinal), `\/`) + `\`
	if !strings.HasPrefix(strings.ToLower(entryFinal), rootPrefix) {
		_ = file.Close()
		return nil, sourceError("source-changed", "entry escaped selected directory")
	}
	return file, nil
}

func finalPathByHandle(file *os.File) (string, error) {
	size := uint32(512)
	for {
		buffer := make([]uint16, size)
		length, err := windows.GetFinalPathNameByHandle(windows.Handle(file.Fd()), &buffer[0], size, 0)
		if err != nil {
			return "", err
		}
		if length < size {
			return windows.UTF16ToString(buffer[:length]), nil
		}
		size = length + 1
	}
}
