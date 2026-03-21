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
	out, err := exec.Command("pkg", "info").Output()
	if err != nil {
		return nil, err
	}

	// Output: "name-version    description"
	// The last hyphen separates name from version.
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
		name := nameVer[:idx]
		version := nameVer[idx+1:]

		desc := ""
		if len(fields) > 1 {
			desc = strings.Join(fields[1:], " ")
		}

		pkgs = append(pkgs, model.Package{
			Name:        name,
			Version:     version,
			Description: desc,
			Source:      model.SourcePkg,
			InstalledAt: time.Now(),
		})
	}
	return pkgs, nil
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
