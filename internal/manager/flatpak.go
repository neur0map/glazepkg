package manager

import (
	"bufio"
	"os/exec"
	"strings"
	"time"

	"github.com/neur0map/glazepkg/internal/model"
)

type Flatpak struct{}

func (f *Flatpak) Name() model.Source { return model.SourceFlatpak }

func (f *Flatpak) Available() bool { return commandExists("flatpak") }

func (f *Flatpak) Scan() ([]model.Package, error) {
	out, err := exec.Command("flatpak", "list", "--app", "--columns=application,version").Output()
	if err != nil {
		return nil, err
	}

	var pkgs []model.Package
	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 1 {
			continue
		}
		name := fields[0]
		version := ""
		if len(fields) >= 2 {
			version = fields[1]
		}
		pkgs = append(pkgs, model.Package{
			Name:        name,
			Version:     version,
			Source:      model.SourceFlatpak,
			InstalledAt: time.Now(),
		})
	}
	return pkgs, nil
}

func (f *Flatpak) Describe(pkgs []model.Package) map[string]string {
	descs := make(map[string]string)
	for _, pkg := range pkgs {
		out, err := exec.Command("flatpak", "info", pkg.Name).Output()
		if err != nil {
			continue
		}
		scanner := bufio.NewScanner(strings.NewReader(string(out)))
		for scanner.Scan() {
			line := scanner.Text()
			key, val, ok := parseField(line)
			if ok && key == "Description" {
				descs[pkg.Name] = val
				break
			}
		}
	}
	return descs
}
