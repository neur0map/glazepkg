package manager

import (
	"bufio"
	"os/exec"
	"strings"
	"time"

	"github.com/neur0map/glazepkg/internal/model"
)

type Snap struct{}

func (s *Snap) Name() model.Source { return model.SourceSnap }

func (s *Snap) Available() bool { return commandExists("snap") }

func (s *Snap) Scan() ([]model.Package, error) {
	out, err := exec.Command("snap", "list").Output()
	if err != nil {
		return nil, err
	}

	var pkgs []model.Package
	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	first := true
	for scanner.Scan() {
		if first {
			first = false
			continue // skip header
		}
		fields := strings.Fields(scanner.Text())
		if len(fields) < 2 {
			continue
		}
		pkgs = append(pkgs, model.Package{
			Name:        fields[0],
			Version:     fields[1],
			Source:      model.SourceSnap,
			InstalledAt: time.Now(),
		})
	}
	return pkgs, nil
}

func (s *Snap) CheckUpdates(pkgs []model.Package) map[string]string {
	out, err := exec.Command("snap", "refresh", "--list").Output()
	if err != nil || len(out) == 0 {
		return nil
	}

	updates := make(map[string]string)
	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	first := true
	for scanner.Scan() {
		if first {
			first = false
			continue // skip header
		}
		fields := strings.Fields(scanner.Text())
		if len(fields) >= 2 {
			updates[fields[0]] = fields[1]
		}
	}
	return updates
}

func (s *Snap) ListDependencies(pkgs []model.Package) map[string][]string {
	deps := make(map[string][]string, len(pkgs))
	for _, p := range pkgs {
		out, err := exec.Command("snap", "connections", p.Name).Output()
		if err != nil {
			continue
		}
		var pkgDeps []string
		scanner := bufio.NewScanner(strings.NewReader(string(out)))
		first := true
		for scanner.Scan() {
			if first {
				first = false
				continue // skip header
			}
			fields := strings.Fields(scanner.Text())
			// Format: "Interface  Plug  Slot  Notes"
			if len(fields) >= 3 && fields[2] != "-" {
				// Slot provider is the dependency (e.g., "core:network")
				slotParts := strings.SplitN(fields[2], ":", 2)
				if len(slotParts) >= 1 && slotParts[0] != p.Name {
					pkgDeps = append(pkgDeps, slotParts[0])
				}
			}
		}
		deps[p.Name] = pkgDeps
	}
	return deps
}

func (s *Snap) Describe(pkgs []model.Package) map[string]string {
	descs := make(map[string]string)
	for _, p := range pkgs {
		out, err := exec.Command("snap", "info", p.Name).Output()
		if err != nil {
			continue
		}
		scanner := bufio.NewScanner(strings.NewReader(string(out)))
		for scanner.Scan() {
			line := scanner.Text()
			if strings.HasPrefix(line, "summary:") {
				descs[p.Name] = strings.TrimSpace(strings.TrimPrefix(line, "summary:"))
				break
			}
		}
	}
	return descs
}

func (s *Snap) UpgradeCmd(name string) *exec.Cmd {
	return privilegedCmd("snap", "refresh", name)
}
