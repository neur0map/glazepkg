package manager

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/neur0map/glazepkg/internal/model"
)

func writeDesktop(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("[Desktop Entry]\n"+body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestScanAppsJoinsEntriesToOwners(t *testing.T) {
	root := t.TempDir()
	home := filepath.Join(root, "home")
	realUser := filepath.Join(root, "dotfiles", "applications")
	dataHome := filepath.Join(home, ".local", "share")
	sys := filepath.Join(root, "usr", "share", "applications")
	fpExport := filepath.Join(root, "var", "lib", "flatpak", "exports", "share", "applications")
	t.Setenv("XDG_DATA_HOME", dataHome)
	old := systemDesktopDirs
	systemDesktopDirs = nil
	defer func() { systemDesktopDirs = old }()

	// The user's applications dir is a symlink, the way dotfile managers do it.
	if err := os.MkdirAll(dataHome, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(realUser, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(realUser, filepath.Join(dataHome, "applications")); err != nil {
		t.Fatal(err)
	}
	userDir := filepath.Join(dataHome, "applications")

	writeDesktop(t, filepath.Join(sys, "firefox.desktop"), "Type=Application\nName=Firefox\nName[de]=Feuerfuchs\nGenericName=Web Browser\nIcon=firefox\nCategories=Network;WebBrowser;\nExec=firefox %u\n")
	writeDesktop(t, filepath.Join(sys, "kitty.desktop"), "Type=Application\nName=kitty\nExec=kitty\n")
	writeDesktop(t, filepath.Join(sys, "kitty-open.desktop"), "Type=Application\nName=kitty URL Launcher\nNoDisplay=true\nExec=kitty +open %U\n")
	writeDesktop(t, filepath.Join(sys, "htop.desktop"), "Type=Application\nName=htop\nTerminal=true\nExec=htop\n")
	writeDesktop(t, filepath.Join(sys, "kde4", "okular.desktop"), "Type=Application\nName=Okular\nExec=okular\n")
	writeDesktop(t, filepath.Join(sys, "site.desktop"), "Type=Link\nName=Website\nURL=https://example.org\n")
	writeDesktop(t, filepath.Join(sys, "gone.desktop"), "Type=Application\nName=Gone\nTryExec=definitely-missing-binary\nExec=gone\n")
	writeDesktop(t, filepath.Join(sys, "hiddenbyuser.desktop"), "Type=Application\nName=Hidden by user\nExec=x\n")
	writeDesktop(t, filepath.Join(sys, "stray.desktop"), "Type=Application\nName=Stray\nExec=stray\n")

	// User copy of a packaged entry, a user-hidden one, and two hand installs.
	writeDesktop(t, filepath.Join(realUser, "kitty.desktop"), "Type=Application\nName=kitty (mine)\nExec=kitty --single-instance\n")
	writeDesktop(t, filepath.Join(realUser, "hiddenbyuser.desktop"), "Type=Application\nName=Hidden by user\nHidden=true\n")
	writeDesktop(t, filepath.Join(realUser, "dev.zed.Zed.desktop"), "Type=Application\nName=Zed\nExec="+filepath.Join(home, ".local", "zed.app", "bin", "zed")+" %U\n")
	writeDesktop(t, filepath.Join(realUser, "helper-links.desktop"), "Type=Application\nName=Helper links\nNoDisplay=true\nExec=helper %u\n")

	writeDesktop(t, filepath.Join(fpExport, "org.gimp.GIMP.desktop"), "Type=Application\nName=GNU Image Manipulation Program\nX-Flatpak=org.gimp.GIMP\nExec=/usr/bin/flatpak run org.gimp.GIMP\n")

	var asked []string
	pr := appsProbe{
		home: home,
		dirs: []appDir{
			{path: userDir, location: AppLocationUser},
			{path: sys, location: AppLocationSystem},
			{path: fpExport, location: AppLocationFlatpak},
		},
		owners: func(paths []string) map[string]appOwner {
			asked = append(asked, paths...)
			return map[string]appOwner{
				filepath.Join(sys, "firefox.desktop"):        {source: model.SourcePacman, name: "firefox", version: "158.0.1-1", size: 316628008},
				filepath.Join(sys, "kitty.desktop"):          {source: model.SourcePacman, name: "kitty", version: "0.49.2-1", size: 80425779},
				filepath.Join(sys, "kitty-open.desktop"):     {source: model.SourcePacman, name: "kitty", version: "0.49.2-1", size: 80425779},
				filepath.Join(sys, "htop.desktop"):           {source: model.SourceAUR, name: "htop-git", version: "3-1", size: 400000},
				filepath.Join(sys, "kde4", "okular.desktop"): {source: model.SourcePacman, name: "okular", version: "1-1", size: 10},
			}
		},
		flatpaks: func() map[string]model.Package {
			return map[string]model.Package{"org.gimp.GIMP": {Name: "org.gimp.GIMP", Version: "3.0.4", SizeBytes: 450000000}}
		},
		lookPath: func(s string) (string, error) { return "", errors.New("not found") },
	}
	apps := scanApps(pr)

	byID := map[string]DesktopApp{}
	for _, a := range apps {
		if _, dup := byID[a.DesktopID]; dup {
			t.Errorf("duplicate desktop id %q", a.DesktopID)
		}
		byID[a.DesktopID] = a
	}
	for _, id := range []string{"site", "gone", "hiddenbyuser"} {
		if _, ok := byID[id]; ok {
			t.Errorf("%s should not be listed", id)
		}
	}
	if len(apps) != 9 {
		t.Errorf("got %d apps, want 9: %+v", len(apps), apps)
	}

	ff := byID["firefox"]
	if ff.Name != "Firefox" || ff.GenericName != "Web Browser" || ff.Icon != "firefox" || ff.Package != "firefox" ||
		ff.Source != model.SourcePacman || ff.Version != "158.0.1-1" || ff.SizeBytes != 316628008 || ff.Size == "" ||
		ff.Location != AppLocationSystem || len(ff.Categories) != 2 || ff.Categories[1] != "WebBrowser" || ff.Exec != "firefox %u" {
		t.Errorf("firefox = %+v", ff)
	}

	// A user copy keeps its own name and file but belongs to the package.
	k := byID["kitty"]
	if k.Name != "kitty (mine)" || k.Location != AppLocationUser || k.Package != "kitty" || k.Source != model.SourcePacman ||
		k.DesktopFile != filepath.Join(userDir, "kitty.desktop") {
		t.Errorf("kitty override = %+v", k)
	}
	if ko := byID["kitty-open"]; !ko.NoDisplay || ko.Package != "kitty" {
		t.Errorf("kitty-open = %+v", ko)
	}
	if h := byID["htop"]; !h.Terminal || h.Source != model.SourceAUR || h.Package != "htop-git" {
		t.Errorf("htop = %+v", h)
	}
	if o := byID["kde4-okular"]; o.Package != "okular" {
		t.Errorf("subdir entry = %+v", o)
	}

	gimp := byID["org.gimp.GIMP"]
	if gimp.Source != model.SourceFlatpak || gimp.Package != "org.gimp.GIMP" || gimp.Version != "3.0.4" ||
		gimp.SizeBytes != 450000000 || gimp.Location != AppLocationFlatpak {
		t.Errorf("gimp = %+v", gimp)
	}

	zed := byID["dev.zed.Zed"]
	if zed.Source != model.SourceLocal || zed.Package != "Zed" || zed.Location != AppLocationUser || zed.SizeBytes <= 0 {
		t.Errorf("zed = %+v", zed)
	}
	// Unowned and hidden from menus: listed, but `gpk remove -m local` can't
	// name it, so no package.
	if hl := byID["helper-links"]; hl.Source != model.SourceLocal || hl.Package != "" || !hl.NoDisplay {
		t.Errorf("helper-links = %+v", hl)
	}
	// Unowned system entry outside the local manager's dirs: local, no package.
	if s := byID["stray"]; s.Source != model.SourceLocal || s.Package != "" || s.Location != AppLocationSystem {
		t.Errorf("stray = %+v", s)
	}

	// Ownership is asked once, for system files only (plus the system file a
	// user copy shadows).
	want := map[string]bool{filepath.Join(sys, "kitty.desktop"): true, filepath.Join(sys, "firefox.desktop"): true}
	for _, p := range asked {
		if filepath.Dir(p) == userDir || filepath.Dir(p) == fpExport {
			t.Errorf("asked ownership of non-system path %s", p)
		}
		delete(want, p)
	}
	if len(want) != 0 {
		t.Errorf("never asked about %v", want)
	}

	if apps[0].Name > apps[len(apps)-1].Name && apps[0].Name != "firefox" {
		t.Errorf("apps not sorted by name: first %q last %q", apps[0].Name, apps[len(apps)-1].Name)
	}
}

func TestApplicationDirsOrderAndLocations(t *testing.T) {
	home := t.TempDir()
	t.Setenv("XDG_DATA_HOME", filepath.Join(home, ".local", "share"))
	t.Setenv("XDG_DATA_DIRS", "/usr/local/share:/usr/share:/var/lib/flatpak/exports/share:relative/ignored")
	dirs := applicationDirs(home)
	want := []appDir{
		{filepath.Join(home, ".local", "share", "applications"), AppLocationUser},
		{"/usr/local/share/applications", AppLocationSystem},
		{"/usr/share/applications", AppLocationSystem},
		{"/var/lib/flatpak/exports/share/applications", AppLocationFlatpak},
		{filepath.Join(home, ".local", "share", "flatpak", "exports", "share", "applications"), AppLocationFlatpak},
	}
	if len(dirs) != len(want) {
		t.Fatalf("dirs = %+v", dirs)
	}
	for i := range want {
		if dirs[i] != want[i] {
			t.Errorf("dir %d = %+v, want %+v", i, dirs[i], want[i])
		}
	}
}

func TestParsePacmanOwners(t *testing.T) {
	out := "/usr/share/applications/firefox.desktop is owned by firefox 158.0.1-1\n" +
		"/usr/share/applications/yay.desktop is owned by yay 13.0.1-1\n"
	o := ParsePacmanOwners(out)
	if len(o) != 2 {
		t.Fatalf("got %v", o)
	}
	ff := o["/usr/share/applications/firefox.desktop"]
	if ff.name != "firefox" || ff.version != "158.0.1-1" || ff.source != model.SourcePacman {
		t.Errorf("firefox = %+v", ff)
	}
}

func TestParseRpmOwners(t *testing.T) {
	paths := []string{"/usr/share/applications/firefox.desktop", "/usr/share/applications/stray.desktop", "/usr/share/applications/gimp.desktop"}
	out := "firefox\t132.0-1.fc41\t250000000\n" +
		"file /usr/share/applications/stray.desktop is not owned by any package\n" +
		"gimp\t2.10.38-1.fc41\t120000000\n"
	o, ok := ParseRpmOwners(paths, out)
	if !ok || len(o) != 2 {
		t.Fatalf("ok=%v owners=%v", ok, o)
	}
	if g := o[paths[2]]; g.name != "gimp" || g.version != "2.10.38-1.fc41" || g.size != 120000000 || g.source != model.SourceDnf {
		t.Errorf("gimp = %+v", g)
	}
	// Two owners for one file shift the lines: the batch is rejected.
	if _, ok := ParseRpmOwners(paths[:2], "a\t1-1\t1\nb\t1-1\t1\nc\t1-1\t1\n"); ok {
		t.Error("mismatched line count should not be accepted")
	}
}

func TestParseXbpsOwners(t *testing.T) {
	out := "firefox-132.0_1: /usr/share/applications/firefox.desktop (regular file)\n" +
		"foo-1.2_3: /usr/share/applications/foo.desktop -> /opt/foo/foo.desktop (link)\n"
	o := ParseXbpsOwners(out)
	if f := o["/usr/share/applications/firefox.desktop"]; f.name != "firefox" || f.version != "132.0_1" || f.source != model.SourceXbps {
		t.Errorf("firefox = %+v", f)
	}
	if f := o["/usr/share/applications/foo.desktop"]; f.name != "foo" || f.version != "1.2_3" {
		t.Errorf("foo link = %+v (all %v)", f, o)
	}
}

func TestParseDpkgOwnersAndSizes(t *testing.T) {
	o := ParseDpkgOwners("firefox-esr:amd64: /usr/share/applications/firefox-esr.desktop\n" +
		"libreoffice-calc, libreoffice-common: /usr/share/applications/calc.desktop\n" +
		"diversion by foo from: /usr/share/applications/x.desktop\n")
	if len(o) != 2 || o["/usr/share/applications/firefox-esr.desktop"].name != "firefox-esr" ||
		o["/usr/share/applications/calc.desktop"].name != "libreoffice-calc" {
		t.Errorf("owners = %+v", o)
	}
	s := ParseDpkgSizes("firefox-esr\t128.5.0esr-1\t250000\n")
	if s["firefox-esr"].size != 250000*1024 || s["firefox-esr"].version != "128.5.0esr-1" {
		t.Errorf("sizes = %+v", s)
	}
}

func TestFlatpakIDFromPath(t *testing.T) {
	cases := map[string]string{
		"/var/lib/flatpak/app/org.gimp.GIMP/current/active/export/share/applications/org.gimp.GIMP.desktop":            "org.gimp.GIMP",
		"/home/u/.local/share/flatpak/app/com.spotify.Client/x86_64/stable/active/export/share/applications/a.desktop": "com.spotify.Client",
		"/somewhere/else/org.foo.Bar.desktop": "org.foo.Bar",
	}
	for in, want := range cases {
		if got := flatpakIDFromPath(in); got != want {
			t.Errorf("flatpakIDFromPath(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestAppsCacheHonoursFingerprint(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_DATA_HOME", filepath.Join(home, "data"))
	t.Setenv("XDG_DATA_DIRS", filepath.Join(home, "sys"))
	sys := filepath.Join(home, "sys", "applications")
	writeDesktop(t, filepath.Join(sys, "a.desktop"), "Type=Application\nName=A\nExec=a\n")

	dirs := applicationDirs(home)
	fp := appsFingerprint(dirs, home)
	saveAppsCache(appsCache{Version: appsCacheVersion, Timestamp: time.Now(), Fingerprint: fp,
		Apps: []DesktopApp{{DesktopID: "cached", Categories: []string{}}}})
	if got := LoadApps(false); len(got) != 1 || got[0].DesktopID != "cached" {
		t.Fatalf("fresh cache not used: %+v", got)
	}

	// A new entry changes the dir's mtime, so the cache no longer applies.
	writeDesktop(t, filepath.Join(sys, "b.desktop"), "Type=Application\nName=B\nExec=b\n")
	future := time.Now().Add(2 * time.Second)
	_ = os.Chtimes(sys, future, future)
	got := LoadApps(false)
	ids := map[string]bool{}
	for _, a := range got {
		ids[a.DesktopID] = true
	}
	if ids["cached"] || !ids["a"] || !ids["b"] {
		t.Errorf("stale cache served after a change: %+v", got)
	}

	DeleteAppsCache()
	if _, err := os.Stat(appsCachePath()); !os.IsNotExist(err) {
		t.Errorf("cache file still present: %v", err)
	}
}
