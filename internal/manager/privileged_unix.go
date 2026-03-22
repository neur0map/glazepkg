//go:build !windows

package manager

import (
	"os"
	"os/exec"
)

// privilegedCmd wraps args with sudo -S when the current user is not root.
// The -S flag makes sudo read the password from stdin, which allows the TUI
// to pipe it from the confirmation modal's password field.
func privilegedCmd(name string, args ...string) *exec.Cmd {
	if os.Geteuid() == 0 {
		return exec.Command(name, args...)
	}
	sudoArgs := append([]string{"-S", name}, args...)
	return exec.Command("sudo", sudoArgs...)
}
