package manager

import (
	"bufio"
	"os/exec"
	"strings"
	"time"

	"github.com/neur0map/glazepkg/internal/model"
)

type FreeBSDPkg struct{}

func (f *FreeBSDPkg) Name() model.Source { return model.SourcePkg }

func (f *FreeBSDPkg) Available() bool {
	if !commandExists("pkg") {
		return false
	}
	// Avoid matching macOS /usr/sbin/pkg (package receipt tool).
	// FreeBSD's pkg supports "pkg info"; macOS's does not.
	err := exec.Command("pkg", "info", "-q").Run()
	return err == nil
}

func (f *FreeBSDPkg) Scan() ([]model.Package, error) {
	// Use pkg query to get only explicitly installed packages (not auto-installed deps).
	// %a=0 means non-automatic, %n=name, %v=version, %c=comment/description.
	out, err := exec.Command("pkg", "query", "-e", "%a = 0", "%n\t%v\t%c").Output()
	if err != nil {
		// Fall back to pkg info (shows all packages) if query fails
		return f.scanAll()
	}

	var pkgs []model.Package
	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	for scanner.Scan() {
		parts := strings.SplitN(scanner.Text(), "\t", 3)
		if len(parts) < 2 || parts[0] == "" {
			continue
		}
		desc := ""
		if len(parts) == 3 {
			desc = parts[2]
		}
		pkgs = append(pkgs, model.Package{
			Name:        parts[0],
			Version:     parts[1],
			Description: desc,
			Source:      model.SourcePkg,
			InstalledAt: time.Now(),
		})
	}
	return pkgs, nil
}

func (f *FreeBSDPkg) scanAll() ([]model.Package, error) {
	out, err := exec.Command("pkg", "info").Output()
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
		fields := strings.Fields(line)
		if len(fields) < 1 {
			continue
		}
		nameVer := fields[0]
		idx := strings.LastIndex(nameVer, "-")
		if idx <= 0 {
			continue
		}
		desc := ""
		if len(fields) > 1 {
			desc = strings.Join(fields[1:], " ")
		}
		pkgs = append(pkgs, model.Package{
			Name:        nameVer[:idx],
			Version:     nameVer[idx+1:],
			Description: desc,
			Source:      model.SourcePkg,
			InstalledAt: time.Now(),
		})
	}
	return pkgs, nil
}

func (f *FreeBSDPkg) RemoveCmd(name string) *exec.Cmd {
	return privilegedCmd("pkg", "delete", "-y", name)
}

func (f *FreeBSDPkg) CheckUpdates(pkgs []model.Package) map[string]string {
	out, err := exec.Command("pkg", "upgrade", "-n").Output()
	if err != nil && len(out) == 0 {
		return nil
	}

	// Look for lines like: "\tname: old_ver -> new_ver"
	updates := make(map[string]string)
	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "\t") {
			continue
		}
		line = strings.TrimSpace(line)
		// "name: old_ver -> new_ver"
		colonIdx := strings.Index(line, ":")
		if colonIdx < 0 {
			continue
		}
		name := line[:colonIdx]
		rest := strings.TrimSpace(line[colonIdx+1:])
		parts := strings.Split(rest, " -> ")
		if len(parts) == 2 {
			updates[name] = strings.TrimSpace(parts[1])
		}
	}
	return updates
}

func (f *FreeBSDPkg) ListDependencies(pkgs []model.Package) map[string][]string {
	deps := make(map[string][]string, len(pkgs))
	for _, pkg := range pkgs {
		out, err := exec.Command("pkg", "info", "-dq", pkg.Name).Output()
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
			// Lines are "dep-name-version"
			idx := strings.LastIndex(line, "-")
			if idx > 0 {
				pkgDeps = append(pkgDeps, line[:idx])
			}
		}
		deps[pkg.Name] = pkgDeps
	}
	return deps
}

func (f *FreeBSDPkg) Describe(pkgs []model.Package) map[string]string {
	// Descriptions are already populated during Scan.
	return nil
}
