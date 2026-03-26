package manager

import (
	"bufio"
	"os/exec"
	"strings"
	"time"

	"github.com/neur0map/glazepkg/internal/model"
)

type Mas struct{}

func (m *Mas) Name() model.Source { return model.SourceMas }

func (m *Mas) Available() bool { return commandExists("mas") }

func (m *Mas) Scan() ([]model.Package, error) {
	out, err := exec.Command("mas", "list").Output()
	if err != nil {
		return nil, err
	}

	// Output: "123456789  App Name  (1.2.3)"
	var pkgs []model.Package
	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		// Find version in trailing parentheses
		parenIdx := strings.LastIndex(line, "(")
		if parenIdx < 0 {
			continue
		}
		version := strings.TrimSuffix(strings.TrimSpace(line[parenIdx+1:]), ")")

		// Everything before the paren is "ID  Name"
		prefix := strings.TrimSpace(line[:parenIdx])
		fields := strings.Fields(prefix)
		if len(fields) < 2 {
			continue
		}
		// First field is the numeric ID, rest is the app name
		name := strings.Join(fields[1:], " ")

		pkgs = append(pkgs, model.Package{
			Name:        name,
			Version:     version,
			Source:      model.SourceMas,
			InstalledAt: time.Now(),
		})
	}
	return pkgs, nil
}

func (m *Mas) UpgradeCmd(name string) *exec.Cmd {
	return exec.Command("mas", "upgrade")
}

func (m *Mas) Search(query string) ([]model.Package, error) {
	out, err := exec.Command("mas", "search", query).Output()
	if err != nil || len(out) == 0 {
		return nil, nil
	}
	var pkgs []model.Package
	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		// Format: "  123456789  App Name          (1.2.3)"
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		id := fields[0]
		// Find version in parentheses at end
		version := ""
		if idx := strings.LastIndex(line, "("); idx >= 0 {
			version = strings.TrimSuffix(strings.TrimSpace(line[idx+1:]), ")")
		}
		// Name is everything between ID and version parens
		nameEnd := strings.LastIndex(line, "(")
		if nameEnd < 0 {
			nameEnd = len(line)
		}
		idEnd := strings.Index(line, id) + len(id)
		name := strings.TrimSpace(line[idEnd:nameEnd])
		if name == "" {
			name = id
		}
		pkgs = append(pkgs, model.Package{Name: name, Version: version, Source: model.SourceMas, Description: "App Store ID: " + id})
	}
	return pkgs, nil
}

func (m *Mas) InstallCmd(name string) *exec.Cmd {
	return exec.Command("mas", "install", name)
}

func (m *Mas) CheckUpdates(pkgs []model.Package) map[string]string {
	out, err := exec.Command("mas", "outdated").Output()
	if err != nil || len(out) == 0 {
		return nil
	}

	// Output: "123456789  App Name  (1.2.3 -> 1.3.0)"
	updates := make(map[string]string)
	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	for scanner.Scan() {
		line := scanner.Text()
		parenIdx := strings.LastIndex(line, "(")
		if parenIdx < 0 {
			continue
		}
		prefix := strings.TrimSpace(line[:parenIdx])
		fields := strings.Fields(prefix)
		if len(fields) < 2 {
			continue
		}
		name := strings.Join(fields[1:], " ")

		inner := strings.TrimSuffix(strings.TrimSpace(line[parenIdx+1:]), ")")
		parts := strings.Split(inner, " -> ")
		if len(parts) == 2 {
			updates[name] = strings.TrimSpace(parts[1])
		}
	}
	return updates
}
