package manager

import (
	"bufio"
	"os/exec"
	"strings"
	"time"

	"github.com/neur0map/glazepkg/internal/model"
)

type Dnf struct{}

func (d *Dnf) Name() model.Source { return model.SourceDnf }

func (d *Dnf) Available() bool { return commandExists("dnf") }

func (d *Dnf) Scan() ([]model.Package, error) {
	out, err := exec.Command("dnf", "list", "installed", "--quiet").Output()
	if err != nil {
		return nil, err
	}

	var pkgs []model.Package
	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" || strings.HasPrefix(line, "Installed") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		// Name might have .arch suffix — strip it
		name := fields[0]
		if idx := strings.LastIndex(name, "."); idx > 0 {
			name = name[:idx]
		}
		pkgs = append(pkgs, model.Package{
			Name:        name,
			Version:     fields[1],
			Source:      model.SourceDnf,
			InstalledAt: time.Now(),
		})
	}
	return pkgs, nil
}

func (d *Dnf) CheckUpdates(pkgs []model.Package) map[string]string {
	out, _ := exec.Command("dnf", "check-update", "--quiet").Output()
	if len(out) == 0 {
		return nil
	}

	updates := make(map[string]string)
	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 2 {
			continue
		}
		name := fields[0]
		if idx := strings.LastIndex(name, "."); idx > 0 {
			name = name[:idx]
		}
		updates[name] = fields[1]
	}
	return updates
}

func (d *Dnf) ListDependencies(pkgs []model.Package) map[string][]string {
	deps := make(map[string][]string, len(pkgs))
	for _, p := range pkgs {
		out, err := exec.Command("rpm", "-qR", p.Name).Output()
		if err != nil {
			continue
		}
		var pkgDeps []string
		scanner := bufio.NewScanner(strings.NewReader(string(out)))
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" {
				continue
			}
			// Skip non-package requirements (paths, libraries, rpmlib, config)
			if strings.HasPrefix(line, "/") || strings.Contains(line, "(") ||
				strings.HasPrefix(line, "rpmlib") || strings.HasPrefix(line, "config") {
				continue
			}
			name := strings.Fields(line)[0]
			if name != "" {
				pkgDeps = append(pkgDeps, name)
			}
		}
		deps[p.Name] = pkgDeps
	}
	return deps
}

func (d *Dnf) Describe(pkgs []model.Package) map[string]string {
	descs := make(map[string]string)
	for _, p := range pkgs {
		out, err := exec.Command("dnf", "info", p.Name).Output()
		if err != nil {
			continue
		}
		scanner := bufio.NewScanner(strings.NewReader(string(out)))
		for scanner.Scan() {
			line := scanner.Text()
			if strings.HasPrefix(line, "Summary") {
				parts := strings.SplitN(line, ":", 2)
				if len(parts) == 2 {
					descs[p.Name] = strings.TrimSpace(parts[1])
				}
				break
			}
		}
	}
	return descs
}
