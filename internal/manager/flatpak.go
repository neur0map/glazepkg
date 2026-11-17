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
	out, err := exec.Command("flatpak", "list", "--app", "--columns=application,version,size").Output()
	if err != nil {
		// Flatpak releases without the size column: fall back to the plain list.
		if out, err = exec.Command("flatpak", "list", "--app", "--columns=application,version").Output(); err != nil {
			return nil, err
		}
	}
	return ParseFlatpakList(string(out)), nil
}

// ParseFlatpakList reads `flatpak list --app --columns=application,version,size`
// output. Columns are tab-separated and the version may be empty, so splitting
// on whitespace would shift the size into the version; output without tabs (a
// two-column list from an older flatpak) is split on whitespace.
func ParseFlatpakList(out string) []model.Package {
	var pkgs []model.Package
	scanner := bufio.NewScanner(strings.NewReader(out))
	for scanner.Scan() {
		line := scanner.Text()
		var fields []string
		if strings.Contains(line, "\t") {
			fields = strings.Split(line, "\t")
		} else {
			fields = strings.Fields(line)
		}
		if len(fields) < 1 || strings.TrimSpace(fields[0]) == "" {
			continue
		}
		pkg := model.Package{
			Name:        strings.TrimSpace(fields[0]),
			Source:      model.SourceFlatpak,
			InstalledAt: time.Now(),
		}
		if len(fields) >= 2 {
			pkg.Version = strings.TrimSpace(fields[1])
		}
		if len(fields) >= 3 {
			if b := ParseFlatpakSize(fields[2]); b > 0 {
				pkg.SizeBytes = b
				pkg.Size = FormatBytes(b)
			}
		}
		pkgs = append(pkgs, pkg)
	}
	return pkgs
}

func (f *Flatpak) CheckUpdates(pkgs []model.Package) map[string]string {
	out, err := exec.Command("flatpak", "remote-ls", "--updates", "--columns=application,version").Output()
	if err != nil || len(out) == 0 {
		return nil
	}

	updates := make(map[string]string)
	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) >= 2 {
			updates[fields[0]] = fields[1]
		}
	}
	return updates
}

func (f *Flatpak) ListDependencies(pkgs []model.Package) map[string][]string {
	deps := make(map[string][]string, len(pkgs))
	for _, pkg := range pkgs {
		out, err := exec.Command("flatpak", "info", pkg.Name).Output()
		if err != nil {
			continue
		}
		scanner := bufio.NewScanner(strings.NewReader(string(out)))
		for scanner.Scan() {
			key, val, ok := parseField(scanner.Text())
			if ok && key == "Runtime" && val != "" {
				deps[pkg.Name] = []string{val}
				break
			}
		}
	}
	return deps
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

func (f *Flatpak) UpgradeCmd(name string) *exec.Cmd {
	return exec.Command("flatpak", "update", "-y", name)
}

func (f *Flatpak) RemoveCmd(name string) *exec.Cmd {
	return exec.Command("flatpak", "uninstall", "-y", name)
}

func (f *Flatpak) Search(query string) ([]model.Package, error) {
	// Run: flatpak search query
	// Output is tab-separated: Name\tDescription\tApplication ID\tVersion\tBranch\tRemotes
	out, err := exec.Command("flatpak", "search", query).Output()
	if err != nil {
		return nil, nil
	}
	var pkgs []model.Package
	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	for scanner.Scan() {
		fields := strings.Split(scanner.Text(), "\t")
		if len(fields) < 4 {
			continue
		}
		name := strings.TrimSpace(fields[0])
		desc := strings.TrimSpace(fields[1])
		version := strings.TrimSpace(fields[3])
		pkgs = append(pkgs, model.Package{
			Name:        name,
			ID:          strings.TrimSpace(fields[2]),
			Version:     version,
			Source:      model.SourceFlatpak,
			Description: desc,
		})
	}
	return pkgs, nil
}

func (f *Flatpak) InstallCmd(name string) *exec.Cmd {
	return exec.Command("flatpak", "install", "-y", name)
}
