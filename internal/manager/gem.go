package manager

import (
	"bufio"
	"os/exec"
	"strings"
	"time"

	"github.com/neur0map/glazepkg/internal/model"
)

type Gem struct{}

func (g *Gem) Name() model.Source { return model.SourceGem }

func (g *Gem) Available() bool { return commandExists("gem") }

func (g *Gem) Scan() ([]model.Package, error) {
	out, err := exec.Command("gem", "list", "--local").Output()
	if err != nil {
		return nil, err
	}

	// Output: "name (version1, version2, ...)"
	// Header: "*** LOCAL GEMS ***"
	var pkgs []model.Package
	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" || strings.HasPrefix(line, "***") {
			continue
		}
		parenIdx := strings.Index(line, "(")
		if parenIdx < 0 {
			continue
		}
		name := strings.TrimSpace(line[:parenIdx])
		verStr := strings.TrimSuffix(strings.TrimSpace(line[parenIdx+1:]), ")")

		// Take the first (latest) version; skip system gems with "default:"
		parts := strings.SplitN(verStr, ",", 2)
		version := strings.TrimSpace(parts[0])
		if strings.Contains(version, "default:") {
			continue
		}

		pkgs = append(pkgs, model.Package{
			Name:        name,
			Version:     version,
			Source:      model.SourceGem,
			InstalledAt: time.Now(),
		})
	}
	return pkgs, nil
}

func (g *Gem) CheckUpdates(pkgs []model.Package) map[string]string {
	out, err := exec.Command("gem", "outdated").Output()
	if err != nil || len(out) == 0 {
		return nil
	}

	// Output: "name (installed < latest)"
	updates := make(map[string]string)
	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	for scanner.Scan() {
		line := scanner.Text()
		parenIdx := strings.Index(line, "(")
		if parenIdx < 0 {
			continue
		}
		name := strings.TrimSpace(line[:parenIdx])
		inner := strings.TrimSuffix(strings.TrimSpace(line[parenIdx+1:]), ")")
		parts := strings.Split(inner, " < ")
		if len(parts) == 2 {
			updates[name] = strings.TrimSpace(parts[1])
		}
	}
	return updates
}

func (g *Gem) ListDependencies(pkgs []model.Package) map[string][]string {
	deps := make(map[string][]string, len(pkgs))
	for _, pkg := range pkgs {
		out, err := exec.Command("gem", "dependency", pkg.Name, "--pipe").Output()
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
			// Format: "dep_name --version '>= 0'"
			fields := strings.Fields(line)
			if len(fields) >= 1 {
				pkgDeps = append(pkgDeps, fields[0])
			}
		}
		deps[pkg.Name] = pkgDeps
	}
	return deps
}

func (g *Gem) Describe(pkgs []model.Package) map[string]string {
	descs := make(map[string]string)
	for _, pkg := range pkgs {
		out, err := exec.Command("gem", "info", pkg.Name).Output()
		if err != nil {
			continue
		}
		// Description is the last indented line(s) after metadata
		scanner := bufio.NewScanner(strings.NewReader(string(out)))
		var lastIndented string
		for scanner.Scan() {
			line := scanner.Text()
			if strings.HasPrefix(line, "    ") && !strings.Contains(line, ":") {
				lastIndented = strings.TrimSpace(line)
			}
		}
		if lastIndented != "" {
			descs[pkg.Name] = lastIndented
		}
	}
	return descs
}

func (g *Gem) UpgradeCmd(name string) *exec.Cmd {
	return exec.Command("gem", "update", name)
}

func (g *Gem) RemoveCmd(name string) *exec.Cmd {
	return exec.Command("gem", "uninstall", name, "-x")
}
