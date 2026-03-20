package manager

import (
	"bufio"
	"os/exec"
	"strings"
	"time"

	"github.com/neur0map/glazepkg/internal/model"
)

type Nix struct{}

func (n *Nix) Name() model.Source { return model.SourceNix }

func (n *Nix) Available() bool { return commandExists("nix-env") }

func (n *Nix) Scan() ([]model.Package, error) {
	// nix-env -q lists installed packages as "name-version" per line
	out, err := exec.Command("nix-env", "-q").Output()
	if err != nil {
		return nil, err
	}

	var pkgs []model.Package
	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		// Last hyphen separates name from version
		idx := strings.LastIndex(line, "-")
		if idx <= 0 {
			pkgs = append(pkgs, model.Package{
				Name:        line,
				Source:      model.SourceNix,
				InstalledAt: time.Now(),
			})
			continue
		}
		name := line[:idx]
		version := line[idx+1:]

		pkgs = append(pkgs, model.Package{
			Name:        name,
			Version:     version,
			Source:      model.SourceNix,
			InstalledAt: time.Now(),
		})
	}
	return pkgs, nil
}

func (n *Nix) Describe(pkgs []model.Package) map[string]string {
	descs := make(map[string]string)
	for _, pkg := range pkgs {
		out, err := exec.Command("nix-env", "-qa", "--description", pkg.Name).Output()
		if err != nil {
			continue
		}
		// Output may be "name-version  description"
		line := strings.TrimSpace(string(out))
		if line == "" {
			continue
		}
		// Find the description after the package name-version
		fields := strings.Fields(line)
		if len(fields) > 1 {
			// First field is name-version, rest is description
			descs[pkg.Name] = strings.Join(fields[1:], " ")
		}
	}
	return descs
}
