package manager

import "os/exec"

// SystemUpgrader is implemented by managers that can update every installed
// package in one step (e.g. pacman -Syu). Distinct from Upgrader, which
// targets a single package. The command must be non-interactive since the TUI
// captures its output; the modal confirmation stands in for the manager's
// own prompt.
type SystemUpgrader interface {
	SystemUpgradeCmd() *exec.Cmd
}
