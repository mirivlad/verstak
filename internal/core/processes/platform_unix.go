//go:build linux || darwin

package processes

import (
	"os"
	"os/exec"
	"syscall"
)

func configureProcess(cmd *exec.Cmd) { cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true} }
func watchProcess(cmd *exec.Cmd) (func(), func(), error) {
	return func() { _ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL) }, func() {}, nil
}
func replaceApprovalFile(from, to string) error { return os.Rename(from, to) }
