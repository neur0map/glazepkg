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
