package manager

import "os/exec"

// Remover is implemented by managers that can remove a single package.
type Remover interface {
	RemoveCmd(name string) *exec.Cmd
}

// DeepRemover is implemented by managers that can also remove a package
// along with its orphaned dependencies.
type DeepRemover interface {
	RemoveCmdWithDeps(name string) *exec.Cmd
}
