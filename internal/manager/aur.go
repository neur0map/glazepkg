package manager

import (
	"bufio"
	"os/exec"
	"strings"
	"time"

	"github.com/neur0map/glazepkg/internal/model"
)

type AUR struct{}

func (a *AUR) Name() model.Source { return model.SourceAUR }

func (a *AUR) Available() bool { return commandExists("pacman") }

func (a *AUR) Scan() ([]model.Package, error) {
	// Foreign packages = AUR or manually built
	out, err := exec.Command("pacman", "-Qm").Output()
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
			Source:      model.SourceAUR,
			InstalledAt: time.Now(),
		})
	}
	return pkgs, nil
}

func (a *AUR) CheckUpdates(pkgs []model.Package) map[string]string {
	// Check if an AUR helper is available (yay, paru)
	var cmd *exec.Cmd
	if commandExists("yay") {
		cmd = exec.Command("yay", "-Qum")
	} else if commandExists("paru") {
		cmd = exec.Command("paru", "-Qum")
	} else {
		return nil
	}

	out, err := cmd.Output()
	if err != nil || len(out) == 0 {
		return nil
	}

	updates := make(map[string]string)
	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	for scanner.Scan() {
		// Format: "name old_ver -> new_ver" or "name new_ver"
		fields := strings.Fields(scanner.Text())
		if len(fields) >= 4 {
			updates[fields[0]] = fields[3]
		} else if len(fields) >= 2 {
			updates[fields[0]] = fields[1]
		}
	}
	return updates
}

func (a *AUR) ListDependencies(pkgs []model.Package) map[string][]string {
	deps := make(map[string][]string, len(pkgs))
	for _, pkg := range pkgs {
		out, err := exec.Command("pacman", "-Qi", pkg.Name).Output()
		if err != nil {
			continue
		}
		scanner := bufio.NewScanner(strings.NewReader(string(out)))
		for scanner.Scan() {
			key, val, ok := parseField(scanner.Text())
			if !ok {
				continue
			}
			if key == "Depends On" {
				if val != "None" {
					deps[pkg.Name] = strings.Fields(val)
				} else {
					deps[pkg.Name] = nil
				}
				break
			}
		}
	}
	return deps
}

func (a *AUR) Describe(pkgs []model.Package) map[string]string {
	descs := make(map[string]string)
	for _, pkg := range pkgs {
		detail, err := QueryDetail(pkg.Name)
		if err == nil && detail.Description != "" {
			descs[pkg.Name] = detail.Description
		}
	}
	return descs
}
