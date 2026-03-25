package manager

import (
	"bufio"
	"encoding/json"
	"errors"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/neur0map/glazepkg/internal/model"
)

// Winget manages packages via the Windows Package Manager (winget).
type Winget struct{}

func (w *Winget) Name() model.Source { return model.SourceWinget }

func (w *Winget) Available() bool {
	return runtime.GOOS == "windows" && commandExists("winget")
}

// wingetEntry is a single package in winget JSON output.
type wingetEntry struct {
	Name    string `json:"Name"`
	Id      string `json:"Id"`
	Version string `json:"Version"`
	Source  string `json:"Source"`
}

// Scan lists all installed packages.
// Tries JSON output first (winget 1.5+), then falls back to fixed-width text parsing.
func (w *Winget) Scan() ([]model.Package, error) {
	out, err := exec.Command("winget", "list",
		"--accept-source-agreements", // without this winget blocks on an interactive EULA prompt
		"--output", "json",
	).Output()
	if err == nil && len(out) > 0 {
		if pkgs, err := w.parseJSON(out); err == nil && len(pkgs) > 0 {
			return pkgs, nil
		}
		// JSON unrecognized — try parsing the same output as text
		if pkgs := w.parseTextOutput(string(out)); len(pkgs) > 0 {
			return pkgs, nil
		}
	}
	// No output at all from JSON attempt — run plain text mode
	return w.parseText()
}

// parseJSON attempts to unmarshal winget JSON output, trying multiple schema shapes
// since the JSON format changed across winget versions and both are in the wild.
func (w *Winget) parseJSON(data []byte) ([]model.Package, error) {
	// Schema A: flat array
	var flat []wingetEntry
	if err := json.Unmarshal(data, &flat); err == nil && len(flat) > 0 {
		return w.toPackages(flat), nil
	}
	// Schema B: {"Sources": [{"Packages": [...]}]}
	var schemaB struct {
		Sources []struct {
			Packages []wingetEntry `json:"Packages"`
		} `json:"Sources"`
	}
	if err := json.Unmarshal(data, &schemaB); err == nil {
		var all []wingetEntry
		for _, s := range schemaB.Sources {
			all = append(all, s.Packages...)
		}
		if len(all) > 0 {
			return w.toPackages(all), nil
		}
	}
	return nil, errors.New("unrecognized winget JSON schema")
}

func (w *Winget) toPackages(entries []wingetEntry) []model.Package {
	pkgs := make([]model.Package, 0, len(entries))
	for _, e := range entries {
		if e.Name == "" {
			continue
		}
		v := e.Version
		if v == "" {
			v = "unknown"
		}
		pkgs = append(pkgs, model.Package{
			Name:        e.Name,
			Version:     v,
			Source:      model.SourceWinget,
			Repository:  e.Source,
			InstalledAt: time.Now(),
		})
	}
	return pkgs
}

// parseText runs `winget list` and delegates to parseTextOutput.
func (w *Winget) parseText() ([]model.Package, error) {
	out, err := exec.Command("winget", "list", "--accept-source-agreements").Output()
	if err != nil {
		return nil, err
	}
	return w.parseTextOutput(string(out)), nil
}

// parseTextOutput parses fixed-width `winget list` text output.
// The separator line of dashes is used to derive column start positions.
//
//	Name                  Id                    Version    Available  Source
//	--------------------  --------------------  ---------  ---------  ------
//	Git                   Git.Git               2.44.0     2.45.0     winget
func (w *Winget) parseTextOutput(s string) []model.Package {
	var colStarts []int
	var pkgs []model.Package
	scanner := bufio.NewScanner(strings.NewReader(s))
	for scanner.Scan() {
		line := scanner.Text()
		if colStarts == nil {
			if wingetIsSep(line) {
				colStarts = wingetColumns(line)
			}
			continue
		}
		if strings.TrimSpace(line) == "" {
			continue
		}
		fields := wingetExtract(line, colStarts)
		// Columns: Name, Id, Version, [Available], [Source]
		if len(fields) < 3 || fields[0] == "" {
			continue
		}
		v := fields[2]
		if v == "" {
			v = "unknown"
		}
		pkgs = append(pkgs, model.Package{
			Name:        fields[0],
			Version:     v,
			Source:      model.SourceWinget,
			InstalledAt: time.Now(),
		})
	}
	return pkgs
}

// CheckUpdates runs `winget upgrade` and returns available version per package name.
func (w *Winget) CheckUpdates(_ []model.Package) map[string]string {
	out, err := exec.Command("winget", "upgrade",
		"--include-unknown",          // show packages whose current version winget doesn't recognise
		"--accept-source-agreements", // same as Scan: prevents interactive blocking
	).Output()
	if err != nil || len(out) == 0 {
		return nil
	}
	return w.parseUpgradeOutput(string(out))
}

// parseUpgradeOutput parses `winget upgrade` text output into a name→latestVersion map.
func (w *Winget) parseUpgradeOutput(s string) map[string]string {
	var colStarts []int
	updates := make(map[string]string)
	scanner := bufio.NewScanner(strings.NewReader(s))
	for scanner.Scan() {
		line := scanner.Text()
		if colStarts == nil {
			if wingetIsSep(line) {
				colStarts = wingetColumns(line)
			}
			continue
		}
		if strings.TrimSpace(line) == "" {
			continue
		}
		fields := wingetExtract(line, colStarts)
		// Columns: Name, Id, Version, Available, Source
		if len(fields) < 4 || fields[0] == "" || fields[3] == "" {
			continue
		}
		updates[fields[0]] = fields[3]
	}
	return updates
}

// wingetIsSep returns true for winget column separator lines (only dashes and spaces).
func wingetIsSep(line string) bool {
	if line == "" {
		return false
	}
	hasDash := false
	for _, c := range line {
		switch c {
		case '-':
			hasDash = true
		case ' ':
		default:
			return false
		}
	}
	return hasDash
}

// wingetColumns derives column start byte positions from a separator line.
// Column widths vary with content, so positions can't be hardcoded.
// Uses byte iteration (not rune) to stay consistent with the byte-based slicing in wingetExtract.
func wingetColumns(sep string) []int {
	var starts []int
	prev := byte(' ')
	for i := 0; i < len(sep); i++ {
		c := sep[i]
		if c == '-' && prev == ' ' {
			starts = append(starts, i)
		}
		prev = c
	}
	return starts
}

// wingetExtract extracts trimmed field values from a fixed-width-column line.
func wingetExtract(line string, starts []int) []string {
	fields := make([]string, len(starts))
	for i, start := range starts {
		end := len(line)
		if i+1 < len(starts) && starts[i+1] < len(line) {
			end = starts[i+1]
		}
		if start < len(line) {
			fields[i] = strings.TrimSpace(line[start:end])
		}
	}
	return fields
}

func (w *Winget) UpgradeCmd(name string) *exec.Cmd {
	return exec.Command("winget", "upgrade",
		"--id", name,
		"--exact",
		"--accept-source-agreements",
		"--accept-package-agreements",
		"--disable-interactivity",
	)
}

func (w *Winget) RemoveCmd(name string) *exec.Cmd {
	return exec.Command("winget", "uninstall",
		"--id", name,
		"--exact",
		"--disable-interactivity",
	)
}

func (w *Winget) Search(query string) ([]model.Package, error) {
	// Run: winget search query --disable-interactivity --accept-source-agreements
	out, err := exec.Command("winget", "search", query,
		"--disable-interactivity",
		"--accept-source-agreements",
	).Output()
	if err != nil {
		return nil, nil
	}
	// Parse tabular output. Find header separator line (-----).
	lines := strings.Split(string(out), "\n")
	var sepIdx int
	var starts []int
	for i, line := range lines {
		if wingetIsSep(strings.TrimSpace(line)) {
			sepIdx = i
			// Find column positions from the separator line
			starts = wingetColumns(strings.TrimSpace(line))
			break
		}
	}
	if len(starts) == 0 || sepIdx == 0 {
		return nil, nil
	}
	var pkgs []model.Package
	for i := sepIdx + 1; i < len(lines); i++ {
		line := lines[i]
		if strings.TrimSpace(line) == "" {
			continue
		}
		fields := wingetExtract(line, starts)
		if len(fields) < 2 {
			continue
		}
		pkgs = append(pkgs, model.Package{
			Name:    strings.TrimSpace(fields[0]),
			Version: strings.TrimSpace(fields[1]),
			Source:  model.SourceWinget,
		})
	}
	return pkgs, nil
}

func (w *Winget) Describe(pkgs []model.Package) map[string]string {
	descs := make(map[string]string)
	for _, pkg := range pkgs {
		out, err := exec.Command("winget", "show", pkg.Name,
			"--accept-source-agreements",
			"--disable-interactivity",
		).Output()
		if err != nil {
			continue
		}
		// winget show output contains a line like:
		// Description: Some description here
		for _, line := range strings.Split(string(out), "\n") {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "Description:") {
				desc := strings.TrimSpace(strings.TrimPrefix(trimmed, "Description:"))
				if desc != "" {
					descs[pkg.Name] = desc
				}
				break
			}
		}
	}
	return descs
}

func (w *Winget) InstallCmd(name string) *exec.Cmd {
	return exec.Command("winget", "install",
		"--id", name,
		"--exact",
		"--accept-source-agreements",
		"--accept-package-agreements",
		"--disable-interactivity",
	)
}
