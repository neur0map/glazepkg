package parsing

import (
	"bufio"
	"strings"
	"testing"
)

func TestSnapListParsing(t *testing.T) {
	output := `Name       Version    Rev    Tracking         Publisher   Notes
core22     20240111   1122   latest/stable    canonical✓  base
firefox    122.0.1    3836   latest/stable    mozilla✓    -
gnome-42   0+git.abc  195    latest/stable    canonical✓  -
`
	type pkg struct{ name, version string }
	var pkgs []pkg
	scanner := bufio.NewScanner(strings.NewReader(output))
	first := true
	for scanner.Scan() {
		if first {
			first = false
			continue
		}
		fields := strings.Fields(scanner.Text())
		if len(fields) < 2 {
			continue
		}
		pkgs = append(pkgs, pkg{fields[0], fields[1]})
	}
	if len(pkgs) != 3 {
		t.Fatalf("expected 3 snaps, got %d", len(pkgs))
	}
	if pkgs[0].name != "core22" || pkgs[0].version != "20240111" {
		t.Errorf("pkg 0: %+v", pkgs[0])
	}
	if pkgs[1].name != "firefox" || pkgs[1].version != "122.0.1" {
		t.Errorf("pkg 1: %+v", pkgs[1])
	}
}

func TestSnapRefreshListParsing(t *testing.T) {
	output := `Name       Version    Rev    Size   Publisher   Notes
firefox    123.0      3900   250MB  mozilla✓    -
core22     20240201   1130   65MB   canonical✓  base
`
	updates := make(map[string]string)
	scanner := bufio.NewScanner(strings.NewReader(output))
	first := true
	for scanner.Scan() {
		if first {
			first = false
			continue
		}
		fields := strings.Fields(scanner.Text())
		if len(fields) >= 2 {
			updates[fields[0]] = fields[1]
		}
	}
	if updates["firefox"] != "123.0" {
		t.Errorf("firefox: got %q", updates["firefox"])
	}
	if updates["core22"] != "20240201" {
		t.Errorf("core22: got %q", updates["core22"])
	}
}
