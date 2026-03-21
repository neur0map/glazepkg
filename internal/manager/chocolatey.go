package manager

import (
	"bufio"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/neur0map/glazepkg/internal/model"
)

// Chocolatey manages packages via the Chocolatey package manager for Windows.
type Chocolatey struct{}

func (c *Chocolatey) Name() model.Source { return model.SourceChocolatey }

func (c *Chocolatey) Available() bool {
	return runtime.GOOS == "windows" && commandExists("choco")
}

// Scan lists locally installed Chocolatey packages.
// Handles both Chocolatey v1 (requires --local-only flag) and v2+ (local by default).
func (c *Chocolatey) Scan() ([]model.Package, error) {
	if c.isV2() {
		return c.runList(false)
	}
	return c.runList(true)
}

// isV2 returns true if the installed Chocolatey is version 2 or later.
// v2 removed the --local-only flag and made local listing the default.
func (c *Chocolatey) isV2() bool {
	out, err := exec.Command("choco", "--version").Output()
	if err != nil {
		return false
	}
	return chocoIsV2OrLater(strings.TrimSpace(string(out)))
}

// chocoIsV2OrLater parses a Chocolatey version string and returns true for v2+.
// Extracted so tests can exercise the logic without invoking the choco binary.
func chocoIsV2OrLater(v string) bool {
	parts := strings.SplitN(v, ".", 2)
	if len(parts) == 0 {
		return false
	}
	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return false
	}
	return major >= 2
}

// runList runs `choco list` and delegates to parseListOutput.
func (c *Chocolatey) runList(withLocalOnly bool) ([]model.Package, error) {
	args := []string{"list"}
	if withLocalOnly {
		args = append(args, "--local-only")
	}
	out, err := exec.Command("choco", args...).Output()
	if err != nil {
		return nil, err
	}
	return c.parseListOutput(string(out)), nil
}

// parseListOutput parses `choco list` text output.
// Both v1 and v2 produce "PackageName Version" space-separated lines.
func (c *Chocolatey) parseListOutput(s string) []model.Package {
	var pkgs []model.Package
	scanner := bufio.NewScanner(strings.NewReader(s))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		// Skip Chocolatey header/footer/warning lines
		if strings.HasPrefix(line, "Chocolatey v") ||
			strings.Contains(line, "packages installed") ||
			strings.HasPrefix(line, "WARNING") ||
			strings.HasPrefix(line, "Validation") {
			continue
		}
		// Output format: "PackageName Version"
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}
		name, version := parts[0], parts[1]
		// choco always appends a summary line like "3 packages installed";
		// a non-digit in the version column reliably filters those out.
		if len(version) == 0 || version[0] < '0' || version[0] > '9' {
			continue
		}
		pkgs = append(pkgs, model.Package{
			Name:        name,
			Version:     version,
			Source:      model.SourceChocolatey,
			InstalledAt: time.Now(),
		})
	}
	return pkgs
}

// CheckUpdates runs `choco outdated` to find packages with available updates.
func (c *Chocolatey) CheckUpdates(_ []model.Package) map[string]string {
	out, err := exec.Command("choco", "outdated", "--ignore-unfound").Output()
	if err != nil || len(out) == 0 {
		return nil
	}
	return c.parseOutdatedOutput(string(out))
}

// parseOutdatedOutput parses `choco outdated` text output.
// Output format: "name|currentVersion|availableVersion|pinned"
func (c *Chocolatey) parseOutdatedOutput(s string) map[string]string {
	updates := make(map[string]string)
	scanner := bufio.NewScanner(strings.NewReader(s))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" ||
			strings.HasPrefix(line, "Chocolatey v") ||
			strings.HasPrefix(line, "Outdated") ||
			strings.HasPrefix(line, "Output is") {
			continue
		}
		parts := strings.Split(line, "|")
		if len(parts) < 3 {
			continue
		}
		name := strings.TrimSpace(parts[0])
		available := strings.TrimSpace(parts[2])
		if name != "" && available != "" {
			updates[name] = available
		}
	}
	return updates
}
