//go:build !windows
// +build !windows

package manager

import (
	"fmt"
	"os"
	"os/exec"
)

// runPrivilegedCommand runs cmd, automatically prefixing with sudo for non-root users on Unix.
func runPrivilegedCommand(cmd *exec.Cmd) error {
	if os.Geteuid() == 0 {
		return cmd.Run()
	}
	if commandExists("sudo") {
		sudoArgs := append([]string{cmd.Path}, cmd.Args[1:]...)
		sudoCmd := exec.Command("sudo", sudoArgs...)
		sudoCmd.Dir = cmd.Dir
		sudoCmd.Env = cmd.Env
		sudoCmd.Stdin = cmd.Stdin
		sudoCmd.Stdout = cmd.Stdout
		sudoCmd.Stderr = cmd.Stderr
		return sudoCmd.Run()
	}

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%w (requires root privileges; run gpk with sudo)", err)
	}
	return nil
}
