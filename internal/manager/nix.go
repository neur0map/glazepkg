package manager

import (
	"bufio"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/neur0map/glazepkg/internal/model"
)

type Nix struct{}

func (n *Nix) Name() model.Source { return model.SourceNix }

func (n *Nix) Available() bool {
	return commandExists("nix-env") || commandExists("nix")
}

func (n *Nix) Scan() ([]model.Package, error) {
	seen := make(map[string]bool)
	var pkgs []model.Package

	// 1. New CLI: nix profile list --json (nix 2.4+)
	if commandExists("nix") {
		if p := n.scanNixProfile(seen); len(p) > 0 {
			pkgs = append(pkgs, p...)
		}
	}

	// 2. Legacy: nix-env -q (user environment)
	if commandExists("nix-env") {
		if p := n.scanNixEnv(seen); len(p) > 0 {
			pkgs = append(pkgs, p...)
		}
	}

	// 3. NixOS system packages (from /run/current-system)
	if p := n.scanNixOSSystem(seen); len(p) > 0 {
		pkgs = append(pkgs, p...)
	}

	return pkgs, nil
}

// scanNixProfile lists packages installed via `nix profile install`.
func (n *Nix) scanNixProfile(seen map[string]bool) []model.Package {
	out, err := exec.Command("nix", "profile", "list", "--json").Output()
	if err != nil || len(out) == 0 {
		return nil
	}

	// JSON structure: {"elements": {"name": {"storePaths": [...], ...}, ...}}
	var result struct {
		Elements map[string]struct {
			StorePaths []string `json:"storePaths"`
		} `json:"elements"`
	}
	if err := json.Unmarshal(out, &result); err != nil {
		return nil
	}

	var pkgs []model.Package
	for _, elem := range result.Elements {
		for _, sp := range elem.StorePaths {
			name, version := parseNixStorePath(sp)
			if name == "" || seen[name] {
				continue
			}
			seen[name] = true
			pkgs = append(pkgs, model.Package{
				Name:        name,
				Version:     version,
				Source:      model.SourceNix,
				InstalledAt: time.Now(),
			})
		}
	}
	return pkgs
}

// scanNixEnv lists packages installed via `nix-env -i`.
func (n *Nix) scanNixEnv(seen map[string]bool) []model.Package {
	out, err := exec.Command("nix-env", "-q").Output()
	if err != nil {
		return nil
	}

	var pkgs []model.Package
	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		name, version := splitNixNameVersion(line)
		if name == "" || seen[name] {
			continue
		}
		seen[name] = true
		pkgs = append(pkgs, model.Package{
			Name:        name,
			Version:     version,
			Source:      model.SourceNix,
			InstalledAt: time.Now(),
		})
	}
	return pkgs
}

// scanNixOSSystem lists packages in the NixOS system profile.
func (n *Nix) scanNixOSSystem(seen map[string]bool) []model.Package {
	swDir := "/run/current-system/sw/bin"
	if _, err := os.Stat(swDir); err != nil {
		return nil // not NixOS
	}

	// Read the system manifest to get installed packages
	manifestPath := "/nix/var/nix/profiles/system/manifest.nix"
	// Newer NixOS uses a different path
	if _, err := os.Stat(manifestPath); err != nil {
		manifestPath = "/run/current-system/sw/manifest.nix"
		if _, err := os.Stat(manifestPath); err != nil {
			return nil
		}
	}

	// Alternatively, list store paths in the system profile
	out, err := exec.Command("nix-store", "-q", "--references", "/run/current-system/sw").Output()
	if err != nil {
		return nil
	}

	var pkgs []model.Package
	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		name, version := parseNixStorePath(line)
		if name == "" || seen[name] {
			continue
		}
		seen[name] = true
		pkgs = append(pkgs, model.Package{
			Name:        name,
			Version:     version,
			Source:      model.SourceNix,
			InstalledAt: time.Now(),
		})
	}
	return pkgs
}

func (n *Nix) RemoveCmd(name string) *exec.Cmd {
	return exec.Command("nix-env", "-e", name)
}

func (n *Nix) ListDependencies(pkgs []model.Package) map[string][]string {
	deps := make(map[string][]string, len(pkgs))
	for _, pkg := range pkgs {
		var storePath string

		// Try nix profile first
		if commandExists("nix") {
			out, _ := exec.Command("nix", "profile", "list", "--json").Output()
			if len(out) > 0 {
				var result struct {
					Elements map[string]struct {
						StorePaths []string `json:"storePaths"`
					} `json:"elements"`
				}
				if json.Unmarshal(out, &result) == nil {
					for _, elem := range result.Elements {
						for _, sp := range elem.StorePaths {
							n, _ := parseNixStorePath(sp)
							if n == pkg.Name {
								storePath = sp
								break
							}
						}
						if storePath != "" {
							break
						}
					}
				}
			}
		}

		// Fallback to nix-env
		if storePath == "" && commandExists("nix-env") {
			pathOut, err := exec.Command("nix-env", "-q", "--out-path", pkg.Name).Output()
			if err != nil {
				continue
			}
			fields := strings.Fields(strings.TrimSpace(string(pathOut)))
			if len(fields) >= 2 {
				storePath = fields[len(fields)-1]
			}
		}

		if storePath == "" {
			continue
		}

		out, err := exec.Command("nix-store", "-q", "--references", storePath).Output()
		if err != nil {
			continue
		}
		var pkgDeps []string
		scanner := bufio.NewScanner(strings.NewReader(string(out)))
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" || line == storePath {
				continue
			}
			name, _ := parseNixStorePath(line)
			if name != "" {
				pkgDeps = append(pkgDeps, name)
			}
		}
		deps[pkg.Name] = pkgDeps
	}
	return deps
}

func (n *Nix) Describe(pkgs []model.Package) map[string]string {
	descs := make(map[string]string)
	if !commandExists("nix-env") {
		return descs
	}
	for _, pkg := range pkgs {
		out, err := exec.Command("nix-env", "-qa", "--description", pkg.Name).Output()
		if err != nil {
			continue
		}
		line := strings.TrimSpace(string(out))
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) > 1 {
			descs[pkg.Name] = strings.Join(fields[1:], " ")
		}
	}
	return descs
}

func (n *Nix) UpgradeCmd(name string) *exec.Cmd {
	return exec.Command("nix-env", "--upgrade", name)
}

func (n *Nix) Search(query string) ([]model.Package, error) {
	out, err := exec.Command("nix", "search", "nixpkgs", query, "--json").Output()
	if err != nil || len(out) == 0 {
		return nil, nil
	}
	var results map[string]struct {
		PName       string `json:"pname"`
		Version     string `json:"version"`
		Description string `json:"description"`
	}
	if err := json.Unmarshal(out, &results); err != nil {
		return nil, nil
	}
	pkgs := make([]model.Package, 0, len(results))
	for _, r := range results {
		pkgs = append(pkgs, model.Package{Name: r.PName, Version: r.Version, Source: model.SourceNix, Description: r.Description})
	}
	return pkgs, nil
}

func (n *Nix) InstallCmd(name string) *exec.Cmd {
	return exec.Command("nix-env", "-iA", "nixpkgs."+name)
}

// parseNixStorePath extracts name and version from a store path like
// "/nix/store/hash-name-version".
func parseNixStorePath(storePath string) (string, string) {
	base := filepath.Base(storePath)
	// Strip the hash prefix "xxxxxxxx-"
	if idx := strings.Index(base, "-"); idx >= 0 {
		base = base[idx+1:]
	} else {
		return base, ""
	}
	return splitNixNameVersion(base)
}

// splitNixNameVersion splits "name-version" by the last hyphen before a digit.
func splitNixNameVersion(s string) (string, string) {
	for i := len(s) - 1; i > 0; i-- {
		if s[i] == '-' && i+1 < len(s) && s[i+1] >= '0' && s[i+1] <= '9' {
			return s[:i], s[i+1:]
		}
	}
	return s, ""
}
