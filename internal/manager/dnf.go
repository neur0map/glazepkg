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

func (d *Dnf) UpgradeCmd(name string) *exec.Cmd {
	return privilegedCmd("dnf", "upgrade", "-y", name)
}

func (d *Dnf) RemoveCmd(name string) *exec.Cmd {
	return privilegedCmd("dnf", "remove", "-y", name)
}

func (d *Dnf) RemoveCmdWithDeps(name string) *exec.Cmd {
	return privilegedCmd("dnf", "autoremove", "-y", name)
}

func (d *Dnf) Search(query string) ([]model.Package, error) {
	// Run: dnf search query
	// Output has header lines then "name.arch : description" format
	out, err := exec.Command("dnf", "search", query).Output()
	if err != nil {
		// dnf search returns exit 1 when no results
		if len(out) == 0 {
			return nil, nil
		}
	}
	var pkgs []model.Package
	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "=") || strings.HasPrefix(line, "Last metadata") || line == "" {
			continue
		}
		parts := strings.SplitN(line, " : ", 2)
		if len(parts) < 2 {
			continue
		}
		nameArch := strings.TrimSpace(parts[0])
		desc := strings.TrimSpace(parts[1])
		// Strip .arch suffix (e.g., "curl.x86_64" -> "curl")
		if idx := strings.LastIndex(nameArch, "."); idx > 0 {
			nameArch = nameArch[:idx]
		}
		pkgs = append(pkgs, model.Package{
			Name:        nameArch,
			Source:      model.SourceDnf,
			Description: desc,
		})
	}
	return pkgs, nil
}

func (d *Dnf) InstallCmd(name string) *exec.Cmd {
	return privilegedCmd("dnf", "install", "-y", name)
}
