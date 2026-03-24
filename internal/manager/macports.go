package manager

import (
	"bufio"
	"os/exec"
	"strings"
	"time"

	"github.com/neur0map/glazepkg/internal/model"
)

type MacPorts struct{}

func (m *MacPorts) Name() model.Source { return model.SourceMacPorts }

func (m *MacPorts) Available() bool { return commandExists("port") }

func (m *MacPorts) Scan() ([]model.Package, error) {
	// -q suppresses the header line
	out, err := exec.Command("port", "-q", "installed").Output()
	if err != nil {
		return nil, err
	}

	// Output format: "  name @version_revision+variants (active)"
	var pkgs []model.Package
	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		// Split on @ to get name and version
		atIdx := strings.Index(line, "@")
		if atIdx < 0 {
			continue
		}
		name := strings.TrimSpace(line[:atIdx])
		rest := line[atIdx+1:]

		// Version ends at first space (before variants or "(active)")
		version := strings.Fields(rest)[0]
		// Strip revision suffix (_0, _1, etc.) and variants (+foo)
		if plusIdx := strings.Index(version, "+"); plusIdx >= 0 {
			version = version[:plusIdx]
		}

		pkgs = append(pkgs, model.Package{
			Name:        name,
			Version:     version,
			Source:      model.SourceMacPorts,
			InstalledAt: time.Now(),
		})
	}
	return pkgs, nil
}

func (m *MacPorts) RemoveCmd(name string) *exec.Cmd {
	return privilegedCmd("port", "uninstall", name)
}

func (m *MacPorts) CheckUpdates(pkgs []model.Package) map[string]string {
	out, err := exec.Command("port", "-q", "outdated").Output()
	if err != nil || len(out) == 0 {
		return nil
	}

	// Output format: "name   installed_version < available_version"
	updates := make(map[string]string)
	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		parts := strings.Fields(line)
		// Expect: name installed_ver < available_ver
		ltIdx := -1
		for i, p := range parts {
			if p == "<" {
				ltIdx = i
				break
			}
		}
		if ltIdx < 0 || ltIdx+1 >= len(parts) || len(parts) < 1 {
			continue
		}
		updates[parts[0]] = parts[ltIdx+1]
	}
	return updates
}

func (m *MacPorts) ListDependencies(pkgs []model.Package) map[string][]string {
	deps := make(map[string][]string, len(pkgs))
	for _, pkg := range pkgs {
		out, err := exec.Command("port", "deps", pkg.Name).Output()
		if err != nil {
			continue
		}
		var pkgDeps []string
		scanner := bufio.NewScanner(strings.NewReader(string(out)))
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			// Lines like: "Build Dependencies: dep1, dep2"
			// "Library Dependencies: dep3"
			// "Runtime Dependencies: dep4, dep5"
			if idx := strings.Index(line, "Dependencies:"); idx >= 0 {
				rest := strings.TrimSpace(line[idx+len("Dependencies:"):])
				for _, d := range strings.Split(rest, ",") {
					d = strings.TrimSpace(d)
					if d != "" {
						pkgDeps = append(pkgDeps, d)
					}
				}
			}
		}
		deps[pkg.Name] = pkgDeps
	}
	return deps
}

func (m *MacPorts) Describe(pkgs []model.Package) map[string]string {
	descs := make(map[string]string)
	for _, pkg := range pkgs {
		// -q suppresses labels, --description gives the short description
		out, err := exec.Command("port", "-q", "info", "--description", pkg.Name).Output()
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
