package parsing

import (
	"bufio"
	"strings"
	"testing"
)

func parseGemList(output string) map[string]string {
	result := make(map[string]string)
	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" || strings.HasPrefix(line, "***") {
			continue
		}
		parenIdx := strings.Index(line, "(")
		if parenIdx < 0 {
			continue
		}
		name := strings.TrimSpace(line[:parenIdx])
		verStr := strings.TrimSuffix(strings.TrimSpace(line[parenIdx+1:]), ")")
		parts := strings.SplitN(verStr, ",", 2)
		version := strings.TrimSpace(parts[0])
		if strings.Contains(version, "default:") {
			continue
		}
		result[name] = version
	}
	return result
}

func TestGemListParsing(t *testing.T) {
	output := `*** LOCAL GEMS ***

bigdecimal (default: 3.1.1)
bundler (2.3.7, default: 2.3.3)
csv (default: 3.2.2)
rake (13.0.6)
`
	pkgs := parseGemList(output)
	// bigdecimal and csv are pure default gems — they should be skipped.
	// bundler has a user-installed version (2.3.7) as the first entry, so it stays.
	// rake is a normal gem, so it stays.
	if len(pkgs) != 2 {
		t.Fatalf("expected 2 gems, got %d: %v", len(pkgs), pkgs)
	}
	if _, ok := pkgs["bigdecimal"]; ok {
		t.Error("bigdecimal is a default system gem and should be skipped")
	}
	if _, ok := pkgs["csv"]; ok {
		t.Error("csv is a default system gem and should be skipped")
	}
	if pkgs["bundler"] != "2.3.7" {
		t.Errorf("bundler: got %q, want 2.3.7", pkgs["bundler"])
	}
	if pkgs["rake"] != "13.0.6" {
		t.Errorf("rake: got %q, want 13.0.6", pkgs["rake"])
	}
}

func TestGemOutdatedParsing(t *testing.T) {
	output := `abbrev (0.1.0 < 0.1.2)
bigdecimal (3.1.1 < 3.1.4)
`
	updates := make(map[string]string)
	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := scanner.Text()
		parenIdx := strings.Index(line, "(")
		if parenIdx < 0 {
			continue
		}
		name := strings.TrimSpace(line[:parenIdx])
		inner := strings.TrimSuffix(strings.TrimSpace(line[parenIdx+1:]), ")")
		parts := strings.Split(inner, " < ")
		if len(parts) == 2 {
			updates[name] = strings.TrimSpace(parts[1])
		}
	}
	if updates["abbrev"] != "0.1.2" {
		t.Errorf("abbrev: got %q, want 0.1.2", updates["abbrev"])
	}
	if updates["bigdecimal"] != "3.1.4" {
		t.Errorf("bigdecimal: got %q, want 3.1.4", updates["bigdecimal"])
	}
}
