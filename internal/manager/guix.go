package manager

import (
	"bufio"
	"os/exec"
	"strings"
	"time"

	"github.com/neur0map/glazepkg/internal/model"
)

type Guix struct{}

func (g *Guix) Name() model.Source { return model.SourceGuix }

func (g *Guix) Available() bool { return commandExists("guix") }

func (g *Guix) Scan() ([]model.Package, error) {
	// guix package -I outputs tab-separated: name\tversion\toutput\tstore-path
	out, err := exec.Command("guix", "package", "-I").Output()
	if err != nil {
		return nil, err
	}

	var pkgs []model.Package
	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		fields := strings.Split(line, "\t")
		if len(fields) < 2 {
			continue
		}
		name := strings.TrimSpace(fields[0])
		version := strings.TrimSpace(fields[1])
		if name == "" {
			continue
		}

		pkgs = append(pkgs, model.Package{
			Name:        name,
			Version:     version,
			Source:      model.SourceGuix,
			InstalledAt: time.Now(),
		})
	}
	return pkgs, nil
}

func (g *Guix) RemoveCmd(name string) *exec.Cmd {
	return exec.Command("guix", "package", "-r", name)
}

func (g *Guix) CheckUpdates(pkgs []model.Package) map[string]string {
	// guix upgrade -n (dry-run) shows "   name old → new"
	out, err := exec.Command("guix", "upgrade", "-n").Output()
	if err != nil && len(out) == 0 {
		return nil
	}

	updates := make(map[string]string)
	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	for scanner.Scan() {
		line := scanner.Text()
		// Lines like: "   emacs 27.2 → 28.1"
		if !strings.Contains(line, "→") {
			continue
		}
		fields := strings.Fields(line)
		// fields: ["name", "old", "→", "new"]
		arrowIdx := -1
		for i, f := range fields {
			if f == "→" {
				arrowIdx = i
				break
			}
		}
		if arrowIdx >= 1 && arrowIdx+1 < len(fields) {
			updates[fields[0]] = fields[arrowIdx+1]
		}
	}
	return updates
}

func (g *Guix) Describe(pkgs []model.Package) map[string]string {
	descs := make(map[string]string)
	for _, pkg := range pkgs {
		out, err := exec.Command("guix", "show", pkg.Name).Output()
		if err != nil {
			continue
		}
		// Recutils format: "synopsis: short description"
		scanner := bufio.NewScanner(strings.NewReader(string(out)))
		for scanner.Scan() {
			line := scanner.Text()
			if strings.HasPrefix(line, "synopsis: ") {
				desc := strings.TrimPrefix(line, "synopsis: ")
				if desc != "" {
					descs[pkg.Name] = desc
				}
				break
			}
		}
	}
	return descs
}

func (g *Guix) ListDependencies(pkgs []model.Package) map[string][]string {
	deps := make(map[string][]string, len(pkgs))
	for _, pkg := range pkgs {
		out, err := exec.Command("guix", "show", pkg.Name).Output()
		if err != nil {
			continue
		}
		// Parse "dependencies: dep1 dep2 ..." and continuation "+ dep3" lines
		scanner := bufio.NewScanner(strings.NewReader(string(out)))
		var pkgDeps []string
		inDeps := false
		for scanner.Scan() {
			line := scanner.Text()
			if strings.HasPrefix(line, "dependencies: ") {
				inDeps = true
				rest := strings.TrimPrefix(line, "dependencies: ")
				for _, d := range strings.Fields(rest) {
					if d != "" {
						pkgDeps = append(pkgDeps, d)
					}
				}
				continue
			}
			if inDeps && strings.HasPrefix(line, "+ ") {
				for _, d := range strings.Fields(strings.TrimPrefix(line, "+ ")) {
					if d != "" {
						pkgDeps = append(pkgDeps, d)
					}
				}
				continue
			}
			if inDeps {
				break
			}
		}
		if len(pkgDeps) > 0 {
			deps[pkg.Name] = pkgDeps
		}
	}
	return deps
}
