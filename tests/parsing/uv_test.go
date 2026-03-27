package parsing

import (
	"bufio"
	"bytes"
	"strings"
	"testing"
)

// parseUvToolList mirrors the parsing logic in internal/manager/uv.go.
func parseUvToolList(out []byte) []struct{ name, version string } {
	type pkg = struct{ name, version string }
	var pkgs []pkg
	scanner := bufio.NewScanner(bytes.NewReader(out))
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" || strings.HasPrefix(line, "-") || strings.HasPrefix(line, " ") {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}
		ver := parts[1]
		if strings.HasPrefix(ver, "v") {
			ver = ver[1:]
		}
		if len(ver) == 0 || ver[0] < '0' || ver[0] > '9' {
			continue
		}
		pkgs = append(pkgs, pkg{parts[0], ver})
	}
	return pkgs
}

func TestUvToolListParsing(t *testing.T) {
	output := []byte(`posting v2.10.0
- posting
ruff v0.15.8
- ruff
black v24.4.2
- black
- blackd
`)
	pkgs := parseUvToolList(output)

	if len(pkgs) != 3 {
		t.Fatalf("expected 3 tools, got %d", len(pkgs))
	}

	expected := []struct{ name, version string }{
		{"posting", "2.10.0"},
		{"ruff", "0.15.8"},
		{"black", "24.4.2"},
	}
	for i, exp := range expected {
		if pkgs[i].name != exp.name || pkgs[i].version != exp.version {
			t.Errorf("pkg %d: got %+v, want %+v", i, pkgs[i], exp)
		}
	}
}

func TestUvToolListEmpty(t *testing.T) {
	output := []byte("No tools installed\n")
	pkgs := parseUvToolList(output)

	if len(pkgs) != 0 {
		t.Fatalf("expected 0 tools from empty output, got %d", len(pkgs))
	}
}

func TestUvToolListVersionWithoutPrefix(t *testing.T) {
	output := []byte("mytool 1.0.0\n- mytool\n")
	pkgs := parseUvToolList(output)

	if len(pkgs) != 1 {
		t.Fatalf("expected 1 tool, got %d", len(pkgs))
	}
	if pkgs[0].name != "mytool" || pkgs[0].version != "1.0.0" {
		t.Errorf("got %+v, want {mytool 1.0.0}", pkgs[0])
	}
}
