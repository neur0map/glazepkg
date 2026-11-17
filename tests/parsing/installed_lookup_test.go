package parsing

import (
	"strings"
	"testing"

	"github.com/neur0map/glazepkg/internal/manager"
)

func TestPacmanQiPackages(t *testing.T) {
	// pacman -Qi libxkbcommon ripgrep (Required By wrapped by a terminal)
	output := `Name            : libxkbcommon
Version         : 1.11.0-1
Description     : Keymap handling library for toolkits and window systems
Required By     : gtk4  kitty  qt6-base
                  sway  wlroots0.19
Installed Size  : 1080.07 KiB

Name            : ripgrep
Version         : 15.2.0-1
Required By     : None
Installed Size  : 5.31 MiB
`
	pkgs := manager.ParsePacmanQiPackages(output)
	if len(pkgs) != 2 {
		t.Fatalf("got %d packages", len(pkgs))
	}
	x := pkgs[0]
	if x.Name != "libxkbcommon" || x.Version != "1.11.0-1" || strings.Join(x.RequiredBy, " ") != "gtk4 kitty qt6-base sway wlroots0.19" || x.SizeBytes == 0 {
		t.Errorf("libxkbcommon = %+v", x)
	}
	if r := pkgs[1]; r.Name != "ripgrep" || len(r.RequiredBy) != 0 {
		t.Errorf("ripgrep = %+v", r)
	}
}

func TestRpmQueryPackages(t *testing.T) {
	// rpm -q --qf '%{NAME}\t%{VERSION}-%{RELEASE}\t%{SIZE}\n' libxkbcommon nothere
	got := manager.ParseRpmQueryPackages("libxkbcommon\t1.7.0-1.fc41\t400000\npackage nothere is not installed\n")
	if len(got) != 1 || got["libxkbcommon"].Version != "1.7.0-1.fc41" || got["libxkbcommon"].SizeBytes != 400000 {
		t.Errorf("got %+v", got)
	}
	req := manager.ParseRpmWhatRequires("gtk4\nkitty\ngtk4\nlibxkbcommon\n", "libxkbcommon")
	if strings.Join(req, ",") != "gtk4,kitty" {
		t.Errorf("whatrequires = %v", req)
	}
	if none := manager.ParseRpmWhatRequires("no package requires ripgrep\n", "ripgrep"); len(none) != 0 {
		t.Errorf("no-requirer message parsed as packages: %v", none)
	}
}

// A library is required through its soname, never by package name, so
// required_by has to go through what the package provides.
func TestRpmRequiredByViaProvides(t *testing.T) {
	// rpm -q --provides libxkbcommon (Fedora 41)
	provides := `libxkbcommon = 1.7.0-1.fc41
libxkbcommon(x86-64) = 1.7.0-1.fc41
libxkbcommon.so.0()(64bit)
libxkbcommon.so.0(V_0.5.0)(64bit)
libxkbcommon.so.0(V_0.6.0)(64bit)
pkgconfig(xkbcommon) = 1.7.0
`
	caps := manager.ParseRpmProvides(provides)
	want := "libxkbcommon,libxkbcommon(x86-64),libxkbcommon.so.0()(64bit),libxkbcommon.so.0(V_0.5.0)(64bit),libxkbcommon.so.0(V_0.6.0)(64bit),pkgconfig(xkbcommon)"
	if strings.Join(caps, ",") != want {
		t.Errorf("provides = %v", caps)
	}
	if got := manager.ParseRpmProvides("package nothere is not installed\n"); len(got) != 0 {
		t.Errorf("not-installed line parsed as a capability: %v", got)
	}

	// rpm -q --whatrequires --qf '%{NAME}\n' libxkbcommon <caps above>: one
	// block per capability, the same package repeated across sonames, and
	// the devel package's pkgconfig requirement.
	whatrequires := `no package requires libxkbcommon
libxkbcommon-devel
gtk4
kitty
libxkbcommon-x11
mutter
gtk4
kitty
libxkbcommon-x11
mutter
gtk4
mutter
libxkbcommon-devel
`
	req := manager.ParseRpmWhatRequires(whatrequires, "libxkbcommon")
	if strings.Join(req, ",") != "libxkbcommon-devel,gtk4,kitty,libxkbcommon-x11,mutter" {
		t.Errorf("required_by = %v", req)
	}
}

func TestXbpsRevdeps(t *testing.T) {
	// xbps-query -X libxkbcommon
	got := manager.ParseXbpsRevdeps("gtk4-4.16.12_1\nkitty-0.39.1_1\n")
	if strings.Join(got, ",") != "gtk4,kitty" {
		t.Errorf("got %v", got)
	}
}
