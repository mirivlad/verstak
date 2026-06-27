package externalopen

import (
	"os/exec"
	"path/filepath"
	"runtime"
)

type Runner func(name string, args ...string) error

type Service struct {
	goos   string
	runner Runner
}

func NewService() *Service {
	return NewServiceFor(runtime.GOOS, func(name string, args ...string) error {
		return exec.Command(name, args...).Start()
	})
}

func NewServiceFor(goos string, runner Runner) *Service {
	return &Service{goos: goos, runner: runner}
}

func (s *Service) OpenPath(path string) error {
	name, args := s.openCommand(path)
	return s.runner(name, args...)
}

func (s *Service) ShowInFolder(path string, isDir bool) error {
	name, args := s.showCommand(path, isDir)
	return s.runner(name, args...)
}

func (s *Service) openCommand(path string) (string, []string) {
	switch s.goos {
	case "darwin":
		return "open", []string{path}
	case "windows":
		return "rundll32", []string{"url.dll,FileProtocolHandler", path}
	default:
		return "xdg-open", []string{path}
	}
}

func (s *Service) showCommand(path string, isDir bool) (string, []string) {
	switch s.goos {
	case "darwin":
		return "open", []string{"-R", path}
	case "windows":
		if isDir {
			return "explorer", []string{path}
		}
		return "explorer", []string{"/select," + path}
	default:
		if isDir {
			return "xdg-open", []string{path}
		}
		return "xdg-open", []string{filepath.Dir(path)}
	}
}
