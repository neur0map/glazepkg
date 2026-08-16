package manager

import (
	"bufio"
	"bytes"
	"os/exec"
	"strings"

	"github.com/neur0map/glazepkg/internal/model"
)

// DotnetTool manages .NET global tools installed with `dotnet tool install -g`,
// mirroring pipx's user-wide model.
type DotnetTool struct{}

func (d *DotnetTool) Name() model.Source { return model.SourceDotnetTool }

func (d *DotnetTool) Available() bool { return commandExists("dotnet") }

func (d *DotnetTool) Scan() ([]model.Package, error) {
	// The table format is stable across every SDK; `--format json` only exists
	// on the .NET 9 SDK and later, so parsing the table keeps gpk working on
	// the .NET 6/7/8 SDKs too.
	out, err := exec.Command("dotnet", "tool", "list", "--global").Output()
	if err != nil {
		return nil, err
	}
	return ParseDotnetToolList(out), nil
}

// ParseDotnetToolList reads the `dotnet tool list --global` table: a header
// row, a dashed separator, then one space-padded row per tool
// ("<Package Id>  <Version>  <Commands...>"). Parsing starts after the dashed
// separator because the header is translated when DOTNET_CLI_UI_LANGUAGE is
// set, which the dotnet CLI honors even under gpk's LC_ALL=C. Package ids and
// versions never contain spaces, so field splitting is width-independent.
func ParseDotnetToolList(data []byte) []model.Package {
	var pkgs []model.Package
	pastSeparator := false
	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		if !pastSeparator {
			if isDashRule(line) {
				pastSeparator = true
			}
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		pkg := model.Package{
			Name:    fields[0],
			Version: fields[1],
			Source:  model.SourceDotnetTool,
			Scope:   "global",
		}
		if len(fields) > 2 {
			pkg.Description = strings.Join(fields[2:], " ")
		}
		pkgs = append(pkgs, pkg)
	}
	return pkgs
}

// isDashRule reports whether s is the table's separator row (a run of dashes).
func isDashRule(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r != '-' {
			return false
		}
	}
	return true
}

func (d *DotnetTool) InstallCmd(name string) *exec.Cmd {
	return exec.Command("dotnet", "tool", "install", "--global", name)
}

func (d *DotnetTool) UpgradeCmd(name string) *exec.Cmd {
	return exec.Command("dotnet", "tool", "update", "--global", name)
}

func (d *DotnetTool) RemoveCmd(name string) *exec.Cmd {
	return exec.Command("dotnet", "tool", "uninstall", "--global", name)
}

func (d *DotnetTool) Search(query string) ([]model.Package, error) {
	out, err := exec.Command("dotnet", "tool", "search", query).Output()
	if err != nil {
		return nil, err
	}
	return ParseDotnetToolSearch(out), nil
}

// ParseDotnetToolSearch reads the `dotnet tool search` table. Columns are
// Package ID, Latest Version, Authors, Downloads, Verified; the Authors column
// contains spaces, but the first two fields never do, so only the id and its
// latest version are read.
func ParseDotnetToolSearch(data []byte) []model.Package {
	var pkgs []model.Package
	pastSeparator := false
	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		if !pastSeparator {
			if isDashRule(line) {
				pastSeparator = true
			}
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		pkgs = append(pkgs, model.Package{
			Name:    fields[0],
			Version: fields[1],
			Source:  model.SourceDotnetTool,
		})
	}
	return pkgs
}

// CheckUpdates resolves each installed tool's latest version from nuget.org via
// `dotnet tool search`, matching the exact package id. The map is keyed by
// package name, which is what FetchUpdates looks up.
func (d *DotnetTool) CheckUpdates(pkgs []model.Package) map[string]string {
	updates := make(map[string]string)
	for _, p := range pkgs {
		out, err := exec.Command("dotnet", "tool", "search", p.Name).Output()
		if err != nil {
			continue
		}
		for _, hit := range ParseDotnetToolSearch(out) {
			if !strings.EqualFold(hit.Name, p.Name) {
				continue
			}
			if hit.Version != "" && hit.Version != p.Version {
				updates[p.Name] = hit.Version
			}
			break
		}
	}
	return updates
}
