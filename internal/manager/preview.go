package manager

import (
	"os/exec"
	"strings"
)

// InstallPreview describes what an install would actually do: the extra
// packages pulled in as dependencies and the transaction's download/installed
// sizes (0 when the manager can't report them).
type InstallPreview struct {
	Deps      []string
	Download  int64
	Installed int64
}

// Previewer is implemented by managers that can preview an install before it
// runs. gpk shows this in the install plan.
type Previewer interface {
	PreviewInstall(name string) (InstallPreview, error)
}

func (p *Pacman) PreviewInstall(name string) (InstallPreview, error) {
	out, err := exec.Command("pacman", "-S", "--print", "--print-format", "%n", name).Output()
	if err != nil {
		return InstallPreview{}, err
	}
	pv := InstallPreview{Deps: pacmanPrintDeps(out, name)}
	if targets := splitNonEmptyLines(out); len(targets) > 0 {
		args := append([]string{"-Si"}, targets...)
		if si, err := exec.Command("pacman", args...).Output(); err == nil {
			pv.Download, pv.Installed = sumPacmanSizes(si)
		}
	}
	return pv, nil
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

// sumPacmanSizes totals the Download/Installed Size fields across a multi-package
// `pacman -Si` dump.
func sumPacmanSizes(out []byte) (download, installed int64) {
	for _, line := range strings.Split(string(out), "\n") {
		key, val, ok := parseField(line)
		if !ok {
			continue
		}
		switch key {
		case "Download Size":
			download += ParseSizeString(val)
		case "Installed Size":
			installed += ParseSizeString(val)
		}
	}
	return download, installed
}
