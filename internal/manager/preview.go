package manager

import "os/exec"

// Previewer is implemented by managers that can report the extra packages an
// install would pull in. gpk shows these in the install plan so you see the
// dependencies before committing.
type Previewer interface {
	InstallDeps(name string) ([]string, error)
}

func (p *Pacman) InstallDeps(name string) ([]string, error) {
	out, err := exec.Command("pacman", "-S", "--print", "--print-format", "%n", name).Output()
	if err != nil {
		return nil, err
	}
	return pacmanPrintDeps(out, name), nil
}

// pacmanPrintDeps keeps every printed target except the requested package,
// leaving just the dependencies the transaction adds.
func pacmanPrintDeps(out []byte, name string) []string {
	var deps []string
	for _, line := range splitNonEmptyLines(out) {
		if line != name {
			deps = append(deps, line)
		}
	}
	return deps
}
