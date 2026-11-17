package parsing

import (
	"testing"

	"github.com/neur0map/glazepkg/internal/manager"
)

// Installed-size parsing for the managers `gpk list --json` fills size_bytes
// for. Output is pasted from the real tools (C locale).

func TestPacmanQiSizes(t *testing.T) {
	// pacman -Qi firefox base
	output := `Name            : firefox
Version         : 158.0.1-1
Description     : Fast, Private & Safe Web Browser
Installed Size  : 301.96 MiB
Packager        : Jan Alexander Steffens (heftig) <heftig@archlinux.org>

Name            : base
Version         : 3-3
Installed Size  : 0.00 B

Name            : ripgrep
Version         : 15.2.0-1
Installed Size  : 5.31 MiB
`
	got := manager.ParsePacmanQiSizes(output)
	want := map[string]int64{"firefox": 316628008, "base": 0, "ripgrep": 5567938}
	for name, b := range want {
		if got[name] != b {
			t.Errorf("%s = %d, want %d", name, got[name], b)
		}
	}
}

func TestExpacSizes(t *testing.T) {
	// expac -Q '%n\t%m' firefox ripgrep
	got := manager.ParseExpacSizes("firefox\t316628008\nripgrep\t5568102\nbroken line\n")
	if len(got) != 2 || got["firefox"] != 316628008 || got["ripgrep"] != 5568102 {
		t.Errorf("got %v", got)
	}
}

func TestRpmSizes(t *testing.T) {
	// rpm -qa --qf '%{NAME}\t%{SIZE}\n' (glibc twice: x86_64 and i686)
	got := manager.ParseRpmSizes("firefox\t262144000\nglibc\t6000000\nglibc\t5000000\n")
	if got["firefox"] != 262144000 || got["glibc"] != 11000000 {
		t.Errorf("got %v", got)
	}
}

func TestXbpsSizes(t *testing.T) {
	// xbps-query -p installed_size -s ''
	output := `firefox-132.0_1: 245MB
ripgrep-14.1.1_1: 5242KB
xbps-0.59.2_1: 1.5MB
base-system-0.114_2: 0B
`
	got := manager.ParseXbpsSizes(output)
	if got["firefox"] != 245<<20 || got["ripgrep"] != 5242<<10 || got["xbps"] != 1572864 {
		t.Errorf("got %v", got)
	}
	if _, ok := got["base-system"]; ok {
		t.Errorf("zero size should be skipped: %v", got)
	}
	for in, want := range map[string]int64{"512B": 512, "1.2GB": 1288490188, "12MB": 12 << 20, "garbage": 0, "": 0} {
		if b := manager.ParseXbpsSize(in); b != want {
			t.Errorf("ParseXbpsSize(%q) = %d, want %d", in, b, want)
		}
	}
}

func TestFlatpakSizes(t *testing.T) {
	// flatpak list --app --columns=application,version,size: GLib decimal
	// units, a no-break space before the unit, and an empty version column.
	output := "com.spotify.Client\t1.2.95.453.g0eeebbed\t14.1\u00a0MB\n" +
		"org.gimp.GIMP\t3.0.4\t1.2\u00a0GB\n" +
		"io.example.NoVersion\t\t980\u00a0bytes\n"
	pkgs := manager.ParseFlatpakList(output)
	if len(pkgs) != 3 {
		t.Fatalf("got %d packages", len(pkgs))
	}
	if p := pkgs[0]; p.Name != "com.spotify.Client" || p.Version != "1.2.95.453.g0eeebbed" || p.SizeBytes != 14100000 || p.Size == "" {
		t.Errorf("spotify = %+v", p)
	}
	if p := pkgs[1]; p.SizeBytes != 1200000000 {
		t.Errorf("gimp = %+v", p)
	}
	if p := pkgs[2]; p.Version != "" || p.SizeBytes != 980 {
		t.Errorf("empty version column shifted fields: %+v", p)
	}

	// Two-column output from a flatpak without the size column.
	old := manager.ParseFlatpakList("org.mozilla.firefox  122.0.1\n")
	if len(old) != 1 || old[0].Version != "122.0.1" || old[0].SizeBytes != 0 {
		t.Errorf("old format = %+v", old)
	}

	for in, want := range map[string]int64{"345.6 kB": 345600, "2 MiB": 2 << 20, "1,5\u00a0GB": 1500000000, "n/a": 0} {
		if b := manager.ParseFlatpakSize(in); b != want {
			t.Errorf("ParseFlatpakSize(%q) = %d, want %d", in, b, want)
		}
	}
}
