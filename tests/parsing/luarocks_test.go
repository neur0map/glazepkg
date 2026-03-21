package parsing

import (
	"strings"
	"testing"
)

func TestLuarocksListPorcelain(t *testing.T) {
	output := "luafilesystem\t1.8.0-1\tinstalled\t/usr/local/lib/luarocks/rocks-5.4\nluasocket\t3.1.0-1\tinstalled\t/usr/local/lib/luarocks/rocks-5.4\nluacheck\t1.1.2-1\tinstalled\t/usr/local/lib/luarocks/rocks-5.4"

	type pkg struct{ name, version string }
	var pkgs []pkg
	for _, line := range strings.Split(output, "\n") {
		if line == "" {
			continue
		}
		fields := strings.Split(line, "\t")
		if len(fields) < 3 || fields[2] != "installed" {
			continue
		}
		pkgs = append(pkgs, pkg{fields[0], fields[1]})
	}

	if len(pkgs) != 3 {
		t.Fatalf("expected 3 rocks, got %d", len(pkgs))
	}
	if pkgs[0].name != "luafilesystem" || pkgs[0].version != "1.8.0-1" {
		t.Errorf("pkg 0: %+v", pkgs[0])
	}
	if pkgs[2].name != "luacheck" || pkgs[2].version != "1.1.2-1" {
		t.Errorf("pkg 2: %+v", pkgs[2])
	}
}

func TestLuarocksOutdatedPorcelain(t *testing.T) {
	output := "luasocket\t3.0.0-1\t3.1.0-1\thttps://luarocks.org\nluacheck\t1.0.0-1\t1.1.2-1\thttps://luarocks.org"

	updates := make(map[string]string)
	for _, line := range strings.Split(output, "\n") {
		if line == "" {
			continue
		}
		fields := strings.Split(line, "\t")
		if len(fields) >= 3 {
			updates[fields[0]] = fields[2]
		}
	}

	if updates["luasocket"] != "3.1.0-1" {
		t.Errorf("luasocket: got %q", updates["luasocket"])
	}
	if updates["luacheck"] != "1.1.2-1" {
		t.Errorf("luacheck: got %q", updates["luacheck"])
	}
}
