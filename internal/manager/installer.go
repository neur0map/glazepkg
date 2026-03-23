package manager

import "os/exec"

// Installer is implemented by managers that can install a new package.
type Installer interface {
	InstallCmd(name string) *exec.Cmd
}
