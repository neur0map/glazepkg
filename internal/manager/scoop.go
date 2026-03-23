package manager

import (
	"bufio"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/neur0map/glazepkg/internal/model"
)

// Scoop manages packages via the Scoop command-line installer for Windows.
type Scoop struct{}

func (s *Scoop) Name() model.Source { return model.SourceScoop }

func (s *Scoop) Available() bool {
	return runtime.GOOS == "windows" && commandExists("scoop")
}

// Scan lists all installed Scoop packages.
// Modern Scoop (2022+) renders a tabular format with a separator line;
// older versions print "  name version [bucket]" per line — both are handled.
func (s *Scoop) Scan() ([]model.Package, error) {
	out, err := exec.Command("scoop", "list").Output()
	if err != nil {
		return nil, err
	}

	text := string(out)

	// Modern tabular format uses the same dash-separator layout as winget;
	// wingetIsSep/wingetColumns/wingetExtract are reused deliberately.
	// We check line-by-line with wingetIsSep rather than strings.Contains("----")
	// to avoid misidentifying a package name that contains four dashes as a separator.
	for _, line := range strings.Split(text, "\n") {
		if wingetIsSep(strings.TrimSpace(line)) {
			return s.parseTabular(text)
		}
	}
	return s.parseLegacy(text)
}

// parseTabular handles the modern scoop list format:
//
//	Name    Version   Source  Updated
//	----    -------   ------  -------
//	7zip    24.09     main    2024-01-01 12:00:00
func (s *Scoop) parseTabular(text string) ([]model.Package, error) {
	var colStarts []int
	var pkgs []model.Package

	scanner := bufio.NewScanner(strings.NewReader(text))
	for scanner.Scan() {
		line := scanner.Text()
		if colStarts == nil {
			if wingetIsSep(strings.TrimSpace(line)) {
				colStarts = wingetColumns(strings.TrimSpace(line))
				// scoop indents its table; offset positions to match actual byte positions in each data line.
				indent := len(line) - len(strings.TrimLeft(line, " "))
				for i := range colStarts {
					colStarts[i] += indent
				}
			}
			continue
		}
		if strings.TrimSpace(line) == "" {
			continue
		}

		fields := wingetExtract(line, colStarts)
		// Columns: Name, Version, Source, Updated
		if len(fields) < 2 || fields[0] == "" {
			continue
		}

		v := fields[1]
		if v == "" {
			v = "unknown"
		}

		pkg := model.Package{
			Name:        fields[0],
			Version:     v,
			Source:      model.SourceScoop,
			InstalledAt: time.Now(),
		}

		// Populate bucket as repository
		if len(fields) >= 3 && fields[2] != "" {
			pkg.Repository = fields[2]
		}

		// Parse install timestamp from the Updated column if present
		if len(fields) >= 4 && len(fields[3]) >= 10 {
			if t, err := time.ParseInLocation("2006-01-02 15:04:05", fields[3], time.Local); err == nil {
				pkg.InstalledAt = t
			} else if t, err := time.ParseInLocation("2006-01-02", fields[3][:10], time.Local); err == nil {
				pkg.InstalledAt = t
			}
		}

		pkgs = append(pkgs, pkg)
	}
	if err := scanner.Err(); err != nil {
		return pkgs, err
	}
	return pkgs, nil
}

// parseLegacy handles older scoop list output:
//
//	Installed apps:
//	  7zip 24.09 [main]
//	  git  2.44.0 [main]
func (s *Scoop) parseLegacy(text string) ([]model.Package, error) {
	var pkgs []model.Package
	scanner := bufio.NewScanner(strings.NewReader(text))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasSuffix(line, ":") {
			continue
		}
		// Strip trailing "[bucket]" annotation
		if idx := strings.LastIndex(line, "["); idx > 0 {
			line = strings.TrimSpace(line[:idx])
		}
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}
		pkgs = append(pkgs, model.Package{
			Name:        parts[0],
			Version:     parts[1],
			Source:      model.SourceScoop,
			InstalledAt: time.Now(),
		})
	}
	if err := scanner.Err(); err != nil {
		return pkgs, err
	}
	return pkgs, nil
}

// CheckUpdates runs `scoop status` to find packages with available updates.
func (s *Scoop) CheckUpdates(_ []model.Package) map[string]string {
	out, err := exec.Command("scoop", "status").Output()
	if err != nil || len(out) == 0 {
		return nil
	}
	return s.parseStatusOutput(string(out))
}

// parseStatusOutput parses `scoop status` text output into a name→latestVersion map.
// Columns: Name, Installed Version, Latest Version, [Missing Deps], [Info]
func (s *Scoop) parseStatusOutput(text string) map[string]string {
	updates := make(map[string]string)

	// "Everything is ok!" means nothing to update
	if strings.Contains(text, "Everything is ok") || strings.Contains(text, "scoop update") {
		return updates
	}

	var colStarts []int
	scanner := bufio.NewScanner(strings.NewReader(text))
	for scanner.Scan() {
		line := scanner.Text()
		if colStarts == nil {
			if wingetIsSep(strings.TrimSpace(line)) {
				colStarts = wingetColumns(strings.TrimSpace(line))
				indent := len(line) - len(strings.TrimLeft(line, " "))
				for i := range colStarts {
					colStarts[i] += indent
				}
			}
			continue
		}
		if strings.TrimSpace(line) == "" {
			continue
		}
		fields := wingetExtract(line, colStarts)
		if len(fields) < 3 || fields[0] == "" || fields[2] == "" {
			continue
		}
		updates[fields[0]] = fields[2]
	}
	return updates
}

func (s *Scoop) UpgradeCmd(name string) *exec.Cmd {
	return exec.Command("scoop", "update", name)
}

func (s *Scoop) RemoveCmd(name string) *exec.Cmd {
	return exec.Command("scoop", "uninstall", name)
}

func (s *Scoop) Search(query string) ([]model.Package, error) {
	// Run: scoop search query
	// Output varies but typically: "name (version)" lines under bucket headers
	out, err := exec.Command("scoop", "search", query).Output()
	if err != nil {
		return nil, nil
	}
	var pkgs []model.Package
	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "'") || strings.HasPrefix(line, "Results") || strings.HasPrefix(line, "-") {
			continue
		}
		// Format varies: "    name (version)"
		fields := strings.Fields(line)
		if len(fields) < 1 {
			continue
		}
		name := fields[0]
		version := ""
		if len(fields) >= 2 {
			v := fields[1]
			v = strings.Trim(v, "()")
			version = v
		}
		pkgs = append(pkgs, model.Package{
			Name:    name,
			Version: version,
			Source:  model.SourceScoop,
		})
	}
	return pkgs, nil
}

func (s *Scoop) InstallCmd(name string) *exec.Cmd {
	return exec.Command("scoop", "install", name)
}
