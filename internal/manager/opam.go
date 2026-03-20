package manager

import (
	"bufio"
	"os/exec"
	"strings"
	"time"

	"github.com/neur0map/glazepkg/internal/model"
)

type Opam struct{}

func (o *Opam) Name() model.Source { return model.SourceOpam }

func (o *Opam) Available() bool { return commandExists("opam") }

func (o *Opam) Scan() ([]model.Package, error) {
	// --columns=package outputs "name.version" per line, --short suppresses headers
	out, err := exec.Command("opam", "list", "--installed", "--columns=package", "--short", "--color=never").Output()
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
		// Format: "name.version" — split on the first dot after the package name.
		// Package names can contain hyphens but not dots, so first dot starts the version.
		dotIdx := strings.Index(line, ".")
		if dotIdx < 0 {
			// Package with no version (e.g. base packages)
			pkgs = append(pkgs, model.Package{
				Name:        line,
				Version:     "base",
				Source:      model.SourceOpam,
				InstalledAt: time.Now(),
			})
			continue
		}
		name := line[:dotIdx]
		version := line[dotIdx+1:]

		pkgs = append(pkgs, model.Package{
			Name:        name,
			Version:     version,
			Source:      model.SourceOpam,
			InstalledAt: time.Now(),
		})
	}
	return pkgs, nil
}

func (o *Opam) CheckUpdates(pkgs []model.Package) map[string]string {
	out, err := exec.Command("opam", "upgrade", "--dry-run", "--color=never").Output()
	if err != nil && len(out) == 0 {
		return nil
	}

	// Look for lines like: "  - upgrade name   old_ver to new_ver"
	updates := make(map[string]string)
	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if !strings.HasPrefix(line, "- upgrade") {
			continue
		}
		// "- upgrade name   old_ver to new_ver"
		fields := strings.Fields(line)
		// fields: ["-", "upgrade", "name", "old_ver", "to", "new_ver"]
		if len(fields) >= 6 && fields[4] == "to" {
			updates[fields[2]] = fields[5]
		}
	}
	return updates
}

func (o *Opam) Describe(pkgs []model.Package) map[string]string {
	descs := make(map[string]string)
	for _, pkg := range pkgs {
		out, err := exec.Command("opam", "show", pkg.Name, "-f", "synopsis", "--color=never").Output()
		if err != nil {
			continue
		}
		desc := strings.TrimSpace(string(out))
		if desc != "" {
			descs[pkg.Name] = desc
		}
	}
	return descs
}
