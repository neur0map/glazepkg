package manager

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestLocalOwnedRoot(t *testing.T) {
	home := "/home/u"
	cases := []struct {
		in       string
		wantRoot string
		wantOK   bool
	}{
		{"/home/u/.local/zed.app/bin/zed", "/home/u/.local/zed.app", true},
		{"/home/u/.local/share/ryoku-apps/discord-1.0.144/Discord/discord", "/home/u/.local/share/ryoku-apps/discord-1.0.144", true},
		{"/home/u/.local/share/claude/versions/2.1.193", "/home/u/.local/share/claude", true},
		{"/home/u/.local/bin/omp", "", false}, // bare bin/, owns no dir
		{"/usr/local/bin/foo", "", false},     // bare bin/ in a system container
		{"/opt/vendorapp/bin/app", "/opt/vendorapp", true},
		{"/usr/bin/kitty", "", false}, // not under any container
	}
	for _, c := range cases {
		root, ok := ownedRoot(c.in, home)
		if root != c.wantRoot || ok != c.wantOK {
			t.Errorf("ownedRoot(%q) = (%q, %v), want (%q, %v)", c.in, root, ok, c.wantRoot, c.wantOK)
		}
	}
}

func TestLocalDesktopProgram(t *testing.T) {
	cases := []struct{ in, want string }{
		{"/home/u/.local/zed.app/bin/zed %U", "/home/u/.local/zed.app/bin/zed"},
		{`"/home/u/.local/bin/claude" --handle-uri %u`, "/home/u/.local/bin/claude"},
		{"kitty -e nvim %F", "kitty"},
		{"%U", ""},
		{"", ""},
		{`"/opt/with space/app" --x`, "/opt/with space/app"},
	}
	for _, c := range cases {
		if got := desktopProgram(c.in); got != c.want {
			t.Errorf("desktopProgram(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestLocalIsNativeBinary(t *testing.T) {
	dir := t.TempDir()
	write := func(name string, data []byte, mode os.FileMode) string {
		p := filepath.Join(dir, name)
		if err := os.WriteFile(p, data, mode); err != nil {
			t.Fatal(err)
		}
		return p
	}
	elf := []byte{0x7f, 'E', 'L', 'F', 2, 1, 1, 0}

	cases := []struct {
		path string
		want bool
	}{
		{write("elf", elf, 0o755), true},
		{write("elf-noexec", elf, 0o644), false},                            // not executable
		{write("script", []byte("#!/usr/bin/env python3\n"), 0o755), false}, // interpreter
		{write("text", []byte("hello world"), 0o755), false},                // not ELF
		{write("short", []byte("x"), 0o755), false},                         // too short
		{filepath.Join(dir, "missing"), false},                              // does not exist
	}
	for _, c := range cases {
		if got := isNativeBinary(c.path); got != c.want {
			t.Errorf("isNativeBinary(%q) = %v, want %v", c.path, got, c.want)
		}
	}
}

func TestLocalIsSafeRemovePath(t *testing.T) {
	home := "/home/u"
	cases := []struct {
		in   string
		want bool
	}{
		{"/home/u/.local/zed.app", true},
		{"/home/u/.local/share/applications/x.desktop", true},
		{"/opt/vendorapp", true},
		{"/home/u/.local", false},  // denylisted root
		{"/home/u", false},         // home itself
		{"/home/u/.config", false}, // protected config root
		{"/usr/local/bin", false},  // denylisted
		{"/opt", false},            // denylisted
		{"/", false},
		{"relative/path", false}, // not absolute
		{"", false},
	}
	for _, c := range cases {
		if got := isSafeRemovePath(c.in, home); got != c.want {
			t.Errorf("isSafeRemovePath(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestLocalVersionFromPaths(t *testing.T) {
	cases := []struct {
		roots []string
		real  string
		want  string
	}{
		{[]string{"discord-1.0.144"}, "", "1.0.144"},
		{[]string{"claude"}, "/x/.local/share/claude/versions/2.1.193", "2.1.193"},
		{[]string{"zed.app"}, "/x/.local/zed.app/bin/zed", ""},
		{nil, "/x/.local/bin/omp", ""},
	}
	for _, c := range cases {
		if got := versionFromPaths(c.roots, c.real); got != c.want {
			t.Errorf("versionFromPaths(%v, %q) = %q, want %q", c.roots, c.real, got, c.want)
		}
	}
}

func TestLocalParseDesktopEntry(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "x.desktop")
	content := `[Desktop Entry]
Type=Application
Name=Real Name
Name[fr]=Nom
Comment=A description
Exec=/opt/app/bin/app %U
NoDisplay=true

[Desktop Action New]
Name=Should be ignored
Exec=/opt/app/bin/app --new
`
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	m := parseDesktopEntry(p)
	if m["Name"] != "Real Name" {
		t.Errorf("Name = %q, want %q (locale key must not override)", m["Name"], "Real Name")
	}
	if m["Type"] != "Application" || m["Comment"] != "A description" {
		t.Errorf("unexpected entry: %+v", m)
	}
	if !isTrue(m["NoDisplay"]) {
		t.Errorf("NoDisplay = %q, want true", m["NoDisplay"])
	}
	if m["Exec"] != "/opt/app/bin/app %U" {
		t.Errorf("Exec from second group leaked: %q", m["Exec"])
	}
}

// TestLocalDiscoverAndRemove exercises the full pipeline against a temp HOME:
// a desktop app (with install dir, icon, and a bin launcher), a standalone
// binary, and several entries that must be filtered out.
func TestLocalDiscoverAndRemove(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink-based layout is POSIX-only")
	}

	// Canonicalize so EvalSymlinks results match HOME-derived prefixes.
	tmp := t.TempDir()
	home, err := filepath.EvalSymlinks(tmp)
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)
	t.Setenv("XDG_DATA_HOME", "")

	// Scope discovery to the temp HOME only.
	defer func(d, b, c []string) { systemDesktopDirs, systemBinDirs, systemContainers = d, b, c }(
		systemDesktopDirs, systemBinDirs, systemContainers)
	systemDesktopDirs, systemBinDirs, systemContainers = nil, nil, nil

	apps := filepath.Join(home, ".local", "share", "applications")
	icons := filepath.Join(home, ".local", "share", "icons")
	bin := filepath.Join(home, ".local", "bin")
	appRoot := filepath.Join(home, ".local", "myapp.app")
	for _, d := range []string{apps, icons, bin, filepath.Join(appRoot, "bin")} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}

	elf := []byte{0x7f, 'E', 'L', 'F', 2, 1, 1, 0}
	mustWrite := func(p string, data []byte, mode os.FileMode) {
		if err := os.WriteFile(p, data, mode); err != nil {
			t.Fatal(err)
		}
	}

	// Desktop app: install dir binary + icon + bin launcher.
	appBin := filepath.Join(appRoot, "bin", "myapp")
	mustWrite(appBin, elf, 0o755)
	iconPath := filepath.Join(icons, "myapp.png")
	mustWrite(iconPath, []byte("png"), 0o644)
	launcher := filepath.Join(bin, "myapp")
	if err := os.Symlink(appBin, launcher); err != nil {
		t.Fatal(err)
	}
	desktop := filepath.Join(apps, "myapp.desktop")
	mustWrite(desktop, []byte("[Desktop Entry]\nType=Application\nName=MyApp\nComment=Cool app\nExec="+appBin+" %U\nIcon="+iconPath+"\n"), 0o644)

	// A standalone curl-installed binary.
	toolPath := filepath.Join(bin, "tool")
	mustWrite(toolPath, elf, 0o755)

	// Things that must be filtered out.
	mustWrite(filepath.Join(apps, "handler.desktop"), []byte("[Desktop Entry]\nType=Application\nName=Handler\nNoDisplay=true\nExec=/usr/bin/true\n"), 0o644)
	mustWrite(filepath.Join(apps, "link.desktop"), []byte("[Desktop Entry]\nType=Link\nName=ALink\nURL=https://x\n"), 0o644)
	mustWrite(filepath.Join(bin, "pyscript"), []byte("#!/usr/bin/env python3\nprint(1)\n"), 0o755) // interpreter
	mustWrite(filepath.Join(bin, "notes.txt"), []byte("hi"), 0o644)                                // non-exec

	got := map[string]localApp{}
	for _, a := range discoverLocalApps() {
		got[a.name] = a
	}

	// Present.
	myapp, ok := got["MyApp"]
	if !ok {
		t.Fatalf("MyApp not detected; got %v", keys(got))
	}
	if myapp.desc != "Cool app" {
		t.Errorf("MyApp desc = %q, want %q", myapp.desc, "Cool app")
	}
	if !contains(myapp.roots, appRoot) {
		t.Errorf("MyApp roots = %v, want to include %q", myapp.roots, appRoot)
	}
	for _, want := range []string{desktop, appRoot, iconPath, launcher} {
		if !contains(myapp.paths, want) {
			t.Errorf("MyApp paths missing %q; have %v", want, myapp.paths)
		}
	}
	if myapp.privileged {
		t.Errorf("MyApp should not be privileged (all under HOME)")
	}
	if _, ok := got["tool"]; !ok {
		t.Errorf("standalone binary 'tool' not detected; got %v", keys(got))
	}

	// Absent.
	for _, bad := range []string{"myapp", "Handler", "ALink", "pyscript", "notes.txt", "tool.txt"} {
		if _, ok := got[bad]; ok {
			t.Errorf("%q should not be reported as a local app", bad)
		}
	}

	// Removal: a single rm covering exactly the owned paths, no sudo.
	l := &Local{}
	cmd := l.RemoveCmd("MyApp")
	if cmd.Args[0] != "rm" {
		t.Fatalf("RemoveCmd(MyApp) = %v, want rm (no sudo for HOME paths)", cmd.Args)
	}
	for _, want := range []string{desktop, appRoot, iconPath, launcher} {
		if !contains(cmd.Args, want) {
			t.Errorf("RemoveCmd(MyApp) missing %q; args=%v", want, cmd.Args)
		}
	}

	toolCmd := l.RemoveCmd("tool")
	if toolCmd.Args[0] != "rm" || !contains(toolCmd.Args, toolPath) {
		t.Errorf("RemoveCmd(tool) = %v, want rm including %q", toolCmd.Args, toolPath)
	}

	// Unknown app: must fail loudly, never rm anything.
	missing := l.RemoveCmd("does-not-exist")
	if !strings.HasSuffix(missing.Path, "false") && missing.Args[0] != "false" {
		t.Errorf("RemoveCmd(unknown) = %v, want a failing no-op command", missing.Args)
	}
}

func contains(s []string, v string) bool {
	for _, x := range s {
		if x == v {
			return true
		}
	}
	return false
}

func keys(m map[string]localApp) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
