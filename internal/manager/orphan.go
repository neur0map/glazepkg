package manager

import (
	"os/exec"
	"strings"
)

// Orphaner is implemented by managers that can find and remove dependencies
// nothing needs anymore. Orphans returns the candidate names (best effort,
// possibly empty); RemoveOrphansCmd returns the command that removes them, or
// nil when there is nothing to do.
type Orphaner interface {
	Orphans() ([]string, error)
	RemoveOrphansCmd(orphans []string, assumeYes bool) *exec.Cmd
}

func splitNonEmptyLines(out []byte) []string {
	var names []string
	for _, line := range strings.Split(string(out), "\n") {
		if line = strings.TrimSpace(line); line != "" {
			names = append(names, line)
		}
	}
	return names
}

func (p *Pacman) Orphans() ([]string, error) {
	// pacman -Qdtq exits non-zero when there are no orphans; treat that as
	// an empty list rather than an error.
	out, err := exec.Command("pacman", "-Qdtq").Output()
	if err != nil {
		return nil, nil
	}
	return splitNonEmptyLines(out), nil
}

func (p *Pacman) RemoveOrphansCmd(orphans []string, yes bool) *exec.Cmd {
	if len(orphans) == 0 {
		return nil
	}
	args := []string{"-Rns"}
	if yes {
		args = append(args, "--noconfirm")
	}
	args = append(args, orphans...)
	return privilegedCmd("pacman", args...)
}

func (a *Apt) Orphans() ([]string, error) {
	out, err := exec.Command("apt-get", "-s", "autoremove").Output()
	if err != nil {
		return nil, nil
	}
	var names []string
	for _, line := range strings.Split(string(out), "\n") {
		if rest, ok := strings.CutPrefix(strings.TrimSpace(line), "Remv "); ok {
			if fields := strings.Fields(rest); len(fields) > 0 {
				names = append(names, fields[0])
			}
		}
	}
	return names, nil
}

func (a *Apt) RemoveOrphansCmd(_ []string, yes bool) *exec.Cmd {
	if yes {
		return privilegedCmd("apt-get", "autoremove", "-y")
	}
	return privilegedCmd("apt-get", "autoremove")
}

func (d *Dnf) Orphans() ([]string, error) {
	out, err := exec.Command("dnf", "repoquery", "--unneeded", "-q").Output()
	if err != nil {
		return nil, nil
	}
	return splitNonEmptyLines(out), nil
}

func (d *Dnf) RemoveOrphansCmd(_ []string, yes bool) *exec.Cmd {
	if yes {
		return privilegedCmd("dnf", "autoremove", "-y")
	}
	return privilegedCmd("dnf", "autoremove")
}

func (x *Xbps) Orphans() ([]string, error) {
	out, err := exec.Command("xbps-query", "-O").Output()
	if err != nil {
		return nil, nil
	}
	return splitNonEmptyLines(out), nil
}

func (x *Xbps) RemoveOrphansCmd(_ []string, yes bool) *exec.Cmd {
	if yes {
		return privilegedCmd("xbps-remove", "-oy")
	}
	return privilegedCmd("xbps-remove", "-o")
}
