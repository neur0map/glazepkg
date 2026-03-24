package manager

import (
	"bufio"
	"os/exec"
	"strings"
	"time"

	"github.com/neur0map/glazepkg/internal/model"
)

type Pacman struct{}

func (p *Pacman) Name() model.Source { return model.SourcePacman }

func (p *Pacman) Available() bool { return commandExists("pacman") }

func (p *Pacman) Scan() ([]model.Package, error) {
	// Get explicitly installed native packages (excludes AUR/foreign)
	out, err := exec.Command("pacman", "-Qen").Output()
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
			Source:      model.SourcePacman,
			InstalledAt: time.Now(),
		})
	}
	return pkgs, nil
}

func (p *Pacman) Describe(pkgs []model.Package) map[string]string {
	descs := make(map[string]string)
	for _, pkg := range pkgs {
		detail, err := QueryDetail(pkg.Name)
		if err == nil && detail.Description != "" {
			descs[pkg.Name] = detail.Description
		}
	}
	return descs
}

// QueryDetail fetches extended info for a single pacman package.
func QueryDetail(name string) (model.Package, error) {
	out, err := exec.Command("pacman", "-Qi", name).Output()
	if err != nil {
		return model.Package{}, err
	}

	pkg := model.Package{Name: name, Source: model.SourcePacman, InstalledAt: time.Now()}
	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	for scanner.Scan() {
		line := scanner.Text()
		key, val, ok := parseField(line)
		if !ok {
			continue
		}
		switch key {
		case "Version":
			pkg.Version = val
		case "Description":
			pkg.Description = val
		case "Installed Size":
			pkg.Size = val
			pkg.SizeBytes = ParseSizeString(val)
		case "Repository":
			pkg.Repository = val
		case "Depends On":
			if val != "None" {
				pkg.DependsOn = strings.Fields(val)
			}
		case "Required By":
			if val != "None" {
				pkg.RequiredBy = strings.Fields(val)
			}
		case "Install Date":
			if t, err := time.Parse("Mon 02 Jan 2006 03:04:05 PM MST", val); err == nil {
				pkg.InstalledAt = t
			}
		}
	}
	return pkg, nil
}

func (p *Pacman) ListDependencies(pkgs []model.Package) map[string][]string {
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

func (p *Pacman) CheckUpdates(pkgs []model.Package) map[string]string {
	// checkupdates (from pacman-contrib) is preferred as it doesn't need root
	// and doesn't modify the sync database. Falls back to pacman -Qu.
	var out []byte
	var err error
	if commandExists("checkupdates") {
		out, err = exec.Command("checkupdates").Output()
	} else {
		out, err = exec.Command("pacman", "-Qu").Output()
	}
	if err != nil || len(out) == 0 {
		return nil
	}

	updates := make(map[string]string)
	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	for scanner.Scan() {
		// Format: "name old_ver -> new_ver"
		fields := strings.Fields(scanner.Text())
		if len(fields) >= 4 {
			updates[fields[0]] = fields[3]
		} else if len(fields) >= 2 {
			updates[fields[0]] = fields[1]
		}
	}
	return updates
}

func (p *Pacman) UpgradeCmd(name string) *exec.Cmd {
	return privilegedCmd("pacman", "-S", name)
}

func (p *Pacman) RemoveCmd(name string) *exec.Cmd {
	return privilegedCmd("pacman", "-R", name)
}

func (p *Pacman) RemoveCmdWithDeps(name string) *exec.Cmd {
	return privilegedCmd("pacman", "-Rns", name)
}

func (p *Pacman) Search(query string) ([]model.Package, error) {
	// Run: pacman -Ss query
	// Output format pairs of lines:
	//   repo/name version [optional info]
	//       description text
	out, err := exec.Command("pacman", "-Ss", query).Output()
	if err != nil {
		return nil, err
	}
	var pkgs []model.Package
	lines := strings.Split(string(out), "\n")
	for i := 0; i < len(lines)-1; i += 2 {
		header := strings.TrimSpace(lines[i])
		desc := strings.TrimSpace(lines[i+1])
		if header == "" {
			continue
		}
		// Parse "repo/name version [installed]"
		parts := strings.Fields(header)
		if len(parts) < 2 {
			continue
		}
		nameWithRepo := parts[0]
		version := parts[1]
		// Extract repo prefix to distinguish AUR from native pacman results
		source := model.SourcePacman
		if idx := strings.Index(nameWithRepo, "/"); idx >= 0 {
			repo := nameWithRepo[:idx]
			nameWithRepo = nameWithRepo[idx+1:]
			if repo == "aur" {
				source = model.SourceAUR
			}
		}
		pkgs = append(pkgs, model.Package{
			Name:        nameWithRepo,
			Version:     version,
			Source:      source,
			Description: desc,
		})
	}
	return pkgs, nil
}

func (p *Pacman) InstallCmd(name string) *exec.Cmd {
	return privilegedCmd("pacman", "-S", name)
}

func parseField(line string) (key, val string, ok bool) {
	idx := strings.Index(line, ":")
	if idx < 0 {
		return "", "", false
	}
	return strings.TrimSpace(line[:idx]), strings.TrimSpace(line[idx+1:]), true
}
