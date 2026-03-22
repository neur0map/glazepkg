package manager

import (
	"bufio"
	"os/exec"
	"strings"
	"time"

	"github.com/neur0map/glazepkg/internal/model"
)

type Apt struct{}

func (a *Apt) Name() model.Source { return model.SourceApt }

func (a *Apt) Available() bool { return commandExists("dpkg-query") }

func (a *Apt) Scan() ([]model.Package, error) {
	// Get explicitly installed packages only
	manualOut, err := exec.Command("apt-mark", "showmanual").Output()
	if err != nil {
		// Fallback to all packages if apt-mark fails
		return a.scanAll()
	}

	manual := make(map[string]bool)
	scanner := bufio.NewScanner(strings.NewReader(string(manualOut)))
	for scanner.Scan() {
		name := strings.TrimSpace(scanner.Text())
		if name != "" {
			manual[name] = true
		}
	}

	out, err := exec.Command("dpkg-query", "-W", "-f=${Package} ${Version}\n").Output()
	if err != nil {
		return nil, err
	}

	var pkgs []model.Package
	scanner = bufio.NewScanner(strings.NewReader(string(out)))
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 2 {
			continue
		}
		if !manual[fields[0]] {
			continue
		}
		pkgs = append(pkgs, model.Package{
			Name:        fields[0],
			Version:     fields[1],
			Source:      model.SourceApt,
			InstalledAt: time.Now(),
		})
	}
	return pkgs, nil
}

func (a *Apt) scanAll() ([]model.Package, error) {
	out, err := exec.Command("dpkg-query", "-W", "-f=${Package} ${Version}\n").Output()
	if err != nil {
		return nil, err
	}

	var pkgs []model.Package
	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 2 {
			continue
		}
		pkgs = append(pkgs, model.Package{
			Name:        fields[0],
			Version:     fields[1],
			Source:      model.SourceApt,
			InstalledAt: time.Now(),
		})
	}
	return pkgs, nil
}

func (a *Apt) CheckUpdates(pkgs []model.Package) map[string]string {
	out, err := exec.Command("apt", "list", "--upgradable").Output()
	if err != nil || len(out) == 0 {
		return nil
	}

	updates := make(map[string]string)
	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "Listing") {
			continue
		}
		// Format: "name/source version arch [upgradable from: old_ver]"
		parts := strings.SplitN(line, "/", 2)
		if len(parts) < 2 {
			continue
		}
		name := parts[0]
		fields := strings.Fields(parts[1])
		if len(fields) >= 2 {
			updates[name] = fields[1]
		}
	}
	return updates
}

func (a *Apt) ListDependencies(pkgs []model.Package) map[string][]string {
	deps := make(map[string][]string, len(pkgs))
	for _, p := range pkgs {
		out, err := exec.Command("apt-cache", "depends", "--no-recommends",
			"--no-suggests", "--no-conflicts", "--no-breaks",
			"--no-replaces", "--no-enhances", p.Name).Output()
		if err != nil {
			continue
		}
		var pkgDeps []string
		scanner := bufio.NewScanner(strings.NewReader(string(out)))
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			var dep string
			switch {
			case strings.HasPrefix(line, "Depends:"):
				dep = strings.TrimSpace(strings.TrimPrefix(line, "Depends:"))
			case strings.HasPrefix(line, "PreDepends:"):
				dep = strings.TrimSpace(strings.TrimPrefix(line, "PreDepends:"))
			default:
				continue
			}
			dep = strings.Trim(dep, "<>")
			if dep != "" {
				pkgDeps = append(pkgDeps, dep)
			}
		}
		deps[p.Name] = pkgDeps
	}
	return deps
}

func (a *Apt) Describe(pkgs []model.Package) map[string]string {
	descs := make(map[string]string)
	for _, p := range pkgs {
		out, err := exec.Command("apt-cache", "show", p.Name).Output()
		if err != nil {
			continue
		}
		scanner := bufio.NewScanner(strings.NewReader(string(out)))
		for scanner.Scan() {
			line := scanner.Text()
			if strings.HasPrefix(line, "Description:") {
				descs[p.Name] = strings.TrimSpace(strings.TrimPrefix(line, "Description:"))
				break
			}
		}
	}
	return descs
}

func (a *Apt) UpgradePackage(name string) error {
	cmd := exec.Command("apt", "install", "--only-upgrade", "-y", name)
	return runPrivilegedCommand(cmd)
}
