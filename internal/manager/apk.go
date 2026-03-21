package manager

import (
	"bufio"
	"os/exec"
	"strings"
	"time"

	"github.com/neur0map/glazepkg/internal/model"
)

type Apk struct{}

func (a *Apk) Name() model.Source { return model.SourceApk }

func (a *Apk) Available() bool { return commandExists("apk") }

func (a *Apk) Scan() ([]model.Package, error) {
	// -vv gives "name-version description" per line
	out, err := exec.Command("apk", "info", "-vv").Output()
	if err != nil {
		return nil, err
	}

	var pkgs []model.Package
	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		// Format: "name-version - description" or "name-version description"
		// Split on " - " first (common format)
		var nameVer, desc string
		if sepIdx := strings.Index(line, " - "); sepIdx >= 0 {
			nameVer = line[:sepIdx]
			desc = line[sepIdx+3:]
		} else {
			fields := strings.Fields(line)
			if len(fields) < 1 {
				continue
			}
			nameVer = fields[0]
			if len(fields) > 1 {
				desc = strings.Join(fields[1:], " ")
			}
		}

		name, version := splitApkNameVersion(nameVer)
		if name == "" {
			continue
		}

		pkgs = append(pkgs, model.Package{
			Name:        name,
			Version:     version,
			Description: desc,
			Source:      model.SourceApk,
			InstalledAt: time.Now(),
		})
	}
	return pkgs, nil
}

func (a *Apk) CheckUpdates(pkgs []model.Package) map[string]string {
	out, err := exec.Command("apk", "upgrade", "-s").Output()
	if err != nil && len(out) == 0 {
		return nil
	}

	// Lines like: "(1/N) Upgrading name (old -> new)"
	updates := make(map[string]string)
	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.Contains(line, "Upgrading") {
			continue
		}
		// Extract "name (old -> new)"
		upgIdx := strings.Index(line, "Upgrading ")
		if upgIdx < 0 {
			continue
		}
		rest := line[upgIdx+len("Upgrading "):]
		parenIdx := strings.Index(rest, "(")
		if parenIdx < 0 {
			continue
		}
		name := strings.TrimSpace(rest[:parenIdx])
		inner := strings.TrimSuffix(strings.TrimSpace(rest[parenIdx+1:]), ")")
		parts := strings.Split(inner, " -> ")
		if len(parts) == 2 {
			updates[name] = strings.TrimSpace(parts[1])
		}
	}
	return updates
}

func (a *Apk) Describe(pkgs []model.Package) map[string]string {
	descs := make(map[string]string)
	for _, pkg := range pkgs {
		out, err := exec.Command("apk", "info", "-d", pkg.Name).Output()
		if err != nil {
			continue
		}
		// Output: "name-version description:\n<description text>"
		scanner := bufio.NewScanner(strings.NewReader(string(out)))
		foundHeader := false
		for scanner.Scan() {
			line := scanner.Text()
			if strings.HasSuffix(line, "description:") {
				foundHeader = true
				continue
			}
			if foundHeader {
				desc := strings.TrimSpace(line)
				if desc != "" {
					descs[pkg.Name] = desc
				}
				break
			}
		}
	}
	return descs
}

// splitApkNameVersion splits "name-version-rN" by finding the version boundary.
// Alpine package names can contain hyphens, so we look for the last hyphen
// that is followed by a digit.
func splitApkNameVersion(s string) (string, string) {
	for i := len(s) - 1; i > 0; i-- {
		if s[i] == '-' && i+1 < len(s) && s[i+1] >= '0' && s[i+1] <= '9' {
			return s[:i], s[i+1:]
		}
	}
	return s, ""
}
