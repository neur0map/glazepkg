package parsing

import (
	"strings"
	"testing"
)

func TestMacPortsListParsing(t *testing.T) {
	output := `  autoconf @2.71_0 (active)
  curl @7.88.1_0+ssl (active)
  git @2.39.2_0+credential_osxkeychain+diff_highlight (active)
  python311 @3.11.2_0 (active)`

	type pkg struct{ name, version string }
	var pkgs []pkg
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		atIdx := strings.Index(line, "@")
		if atIdx < 0 {
			continue
		}
		name := strings.TrimSpace(line[:atIdx])
		rest := line[atIdx+1:]
		version := strings.Fields(rest)[0]
		if plusIdx := strings.Index(version, "+"); plusIdx >= 0 {
			version = version[:plusIdx]
		}
		pkgs = append(pkgs, pkg{name, version})
	}

	if len(pkgs) != 4 {
		t.Fatalf("expected 4 ports, got %d", len(pkgs))
	}
	if pkgs[0].name != "autoconf" || pkgs[0].version != "2.71_0" {
		t.Errorf("pkg 0: %+v", pkgs[0])
	}
	if pkgs[1].name != "curl" || pkgs[1].version != "7.88.1_0" {
		t.Errorf("pkg 1: %+v", pkgs[1])
	}
}

func TestMacPortsOutdatedParsing(t *testing.T) {
	output := `gnupg                          1.4.16_0 < 1.4.18_0
gpgme                          1.5.0_0  < 1.5.1_0`

	updates := make(map[string]string)
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.Fields(line)
		ltIdx := -1
		for i, p := range parts {
			if p == "<" {
				ltIdx = i
				break
			}
		}
		if ltIdx < 0 || ltIdx+1 >= len(parts) {
			continue
		}
		updates[parts[0]] = parts[ltIdx+1]
	}
	if updates["gnupg"] != "1.4.18_0" {
		t.Errorf("gnupg: got %q", updates["gnupg"])
	}
	if updates["gpgme"] != "1.5.1_0" {
		t.Errorf("gpgme: got %q", updates["gpgme"])
	}
}
