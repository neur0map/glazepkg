//go:build windows
// +build windows

package manager

import "os/exec"

// runPrivilegedCommand is a no-op on Windows; commands must already have the right privileges.
func runPrivilegedCommand(cmd *exec.Cmd) error {
	return cmd.Run()
}
