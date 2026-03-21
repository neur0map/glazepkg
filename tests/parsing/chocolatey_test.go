package parsing

import (
	"strings"
	"testing"
)

func TestChocolateyListParsing(t *testing.T) {
	// choco list output (v2 uses | separator, v1 uses space)
	outputV1 := `Chocolatey v1.4.0
chocolatey 1.4.0
git 2.44.0
nodejs 20.11.0
2 packages installed.
`
	type pkg struct{ name, version string }
	var pkgs []pkg
	for _, line := range strings.Split(outputV1, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "Chocolatey") || strings.HasSuffix(line, "installed.") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) >= 2 {
			pkgs = append(pkgs, pkg{fields[0], fields[1]})
		}
	}
	if len(pkgs) != 3 {
		t.Fatalf("expected 3 packages, got %d", len(pkgs))
	}
	if pkgs[1].name != "git" || pkgs[1].version != "2.44.0" {
		t.Errorf("pkg 1: %+v", pkgs[1])
	}
}

func TestChocolateyOutdatedParsing(t *testing.T) {
	output := `Outdated Packages
 Output is package name | current version | available version | pinned?

chocolatey|1.4.0|2.0.0|false
git|2.44.0|2.45.0|false

Chocolatey has determined 2 package(s) are outdated.
`
	updates := make(map[string]string)
	for _, line := range strings.Split(output, "\n") {
		parts := strings.SplitN(line, "|", 4)
		if len(parts) < 3 {
			continue
		}
		name := strings.TrimSpace(parts[0])
		latest := strings.TrimSpace(parts[2])
		if name != "" && latest != "" {
			updates[name] = latest
		}
	}
	if updates["git"] != "2.45.0" {
		t.Errorf("git: got %q", updates["git"])
	}
	if updates["chocolatey"] != "2.0.0" {
		t.Errorf("chocolatey: got %q", updates["chocolatey"])
	}
}
