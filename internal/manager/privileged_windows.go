//go:build windows

package manager

import "os/exec"

// privilegedCmd is a pass-through on Windows; the process must already have
// the required privileges (e.g. run from an elevated prompt).
func privilegedCmd(name string, args ...string) *exec.Cmd {
	return exec.Command(name, args...)
}
