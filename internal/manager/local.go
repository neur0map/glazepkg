package manager

import (
	"bufio"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/neur0map/glazepkg/internal/model"
)

// Local inventories applications installed outside any package manager: GUI
// apps that drop an XDG .desktop entry (Zed, Discord, Termius, …) and
// standalone CLI binaries dropped into a user bin dir by a curl|sh installer
// (Claude Code, omp, …). It can uninstall them by deleting the files they own.
//
// Linux only for now. Detection deliberately ignores anything a system package
// manager owns and any interpreter (#!) script, so language-tool entry points
// (pip/pipx/uv) and shell-script swarms are never misreported as apps.
type Local struct{}

func (l *Local) Name() model.Source { return model.SourceLocal }

func (l *Local) Available() bool { return runtime.GOOS == "linux" }

func (l *Local) Scan() ([]model.Package, error) {
	apps := discoverLocalApps()
	pkgs := make([]model.Package, 0, len(apps))
	for _, a := range apps {
		size := ""
		if a.sizeBytes > 0 {
			size = FormatBytes(a.sizeBytes)
		}
		pkgs = append(pkgs, model.Package{
			Name:        a.name,
			Version:     a.version,
			Source:      model.SourceLocal,
			Description: a.desc,
			Location:    a.primary,
			Size:        size,
			SizeBytes:   a.sizeBytes,
			Scope:       a.scope,
			InstalledAt: a.installedAt,
		})
	}
	return pkgs, nil
}

// RemoveCmd re-discovers local apps and returns a command that deletes every
// file the named app owns. Re-discovery keeps the manager stateless, matching
// the rest of the package. Paths under $HOME run unprivileged; if any owned
// path lives outside it, the whole removal is wrapped in sudo.
func (l *Local) RemoveCmd(name string) *exec.Cmd {
	for _, a := range discoverLocalApps() {
		if a.name != name {
			continue
		}
		paths := safeRemovablePaths(a.paths)
		if len(paths) == 0 {
			break
		}
		args := append([]string{"-rf", "--"}, paths...)
		if a.privileged {
			return privilegedCmd("rm", args...)
		}
		return exec.Command("rm", args...)
	}
	// The app vanished between scan and removal; fail loudly rather than
	// silently removing nothing (or, worse, the wrong thing).
	return exec.Command("false")
}

// localApp is the internal, richer view of a detected app. Scan maps the subset
// it needs onto model.Package; RemoveCmd consumes paths/privileged directly.
type localApp struct {
	name        string
	version     string
	desc        string
	primary     string   // representative path, shown as Location
	paths       []string // every path removal should delete
	roots       []string // self-contained install dirs this app claims
	sizeBytes   int64
	scope       string // "user" or "system"
	privileged  bool   // any owned path lives outside $HOME
	installedAt time.Time
}

func discoverLocalApps() []localApp {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return nil
	}

	apps := scanDesktopApps(home)

	// Binaries that resolve into an app dir already reported via its .desktop
	// entry (e.g. ~/.local/bin/zed inside ~/.local/zed.app) must not appear a
	// second time, so collect every claimed install root first.
	var claimed []string
	for _, a := range apps {
		claimed = append(claimed, a.roots...)
	}
	apps = append(apps, scanLocalBinaries(home, claimed)...)

	sort.Slice(apps, func(i, j int) bool {
		return strings.ToLower(apps[i].name) < strings.ToLower(apps[j].name)
	})
	return apps
}

// --- desktop entries (GUI apps) ---

func scanDesktopApps(home string) []localApp {
	var apps []localApp
	seen := map[string]bool{} // a user entry shadows a same-named system one
	for _, dir := range desktopAppDirs(home) {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".desktop") || seen[e.Name()] {
				continue
			}
			if app, ok := parseDesktopApp(filepath.Join(dir, e.Name()), home); ok {
				seen[e.Name()] = true
				apps = append(apps, app)
			}
		}
	}
	return apps
}

// desktopAppDirs lists the application dirs that hold user- or locally-installed
// entries. /usr/share/applications is excluded on purpose: it is the system
// package manager's territory.
func desktopAppDirs(home string) []string {
	dataHome := os.Getenv("XDG_DATA_HOME")
	if dataHome == "" {
		dataHome = filepath.Join(home, ".local", "share")
	}
	dirs := []string{filepath.Join(dataHome, "applications")}
	return append(dirs, systemDesktopDirs...)
}

func parseDesktopApp(path, home string) (localApp, bool) {
	entry := parseDesktopEntry(path)
	if entry == nil || !strings.EqualFold(entry["Type"], "Application") {
		return localApp{}, false
	}
	if isTrue(entry["NoDisplay"]) || isTrue(entry["Hidden"]) {
		return localApp{}, false
	}
	// A .desktop sitting in a system dir might still belong to a package.
	if !underHome(path, home) && isPackageOwned(path) {
		return localApp{}, false
	}

	app := localApp{
		name:        firstNonEmpty(entry["Name"], strings.TrimSuffix(filepath.Base(path), ".desktop")),
		desc:        firstNonEmpty(entry["Comment"], entry["GenericName"]),
		primary:     path,
		paths:       []string{path},
		scope:       scopeFor(path, home),
		installedAt: fileModTime(path),
	}

	// Resolve the install root from the launcher the entry points at.
	for _, key := range []string{"TryExec", "Exec"} {
		prog := desktopProgram(entry[key])
		if prog == "" {
			continue
		}
		real := resolveProgram(prog)
		if real == "" {
			continue
		}
		if root, ok := ownedRoot(real, home); ok {
			app.roots = appendUnique(app.roots, root)
			app.paths = appendUnique(app.paths, root)
		}
		break
	}

	// An app-specific icon stored as a loose file (not inside the install root).
	if icon := entry["Icon"]; filepath.IsAbs(icon) &&
		underHome(icon, home) && isRegularFile(icon) && !underAnyRoot(icon, app.roots) {
		app.paths = appendUnique(app.paths, icon)
	}

	// CLI launchers in bin dirs that point into the install root.
	for _, root := range app.roots {
		for _, b := range binDirs(home) {
			for _, link := range launchersInto(b, root) {
				app.paths = appendUnique(app.paths, link)
			}
		}
	}

	app.version = versionFromPaths(app.roots, app.primary)
	app.sizeBytes = totalSize(app.paths)
	app.privileged = anyOutsideHome(app.paths, home)
	if app.privileged {
		app.scope = "system"
	}
	return app, true
}

// --- standalone binaries (CLI apps) ---

func scanLocalBinaries(home string, claimedRoots []string) []localApp {
	var apps []localApp
	seen := map[string]bool{} // dedup by resolved target across bin dirs
	for _, dir := range binDirs(home) {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			path := filepath.Join(dir, e.Name())
			real := resolveSymlink(path)

			// Skip launchers living inside an app already reported via .desktop.
			if underAnyRoot(real, claimedRoots) || underAnyRoot(path, claimedRoots) {
				continue
			}
			// Native executables only; #! scripts belong to a language manager.
			if !isNativeBinary(real) {
				continue
			}
			// A system-path binary a package owns is not a "local" install.
			if !underHome(path, home) && isPackageOwned(path) {
				continue
			}
			if seen[real] {
				continue
			}
			seen[real] = true

			app := localApp{
				name:        e.Name(),
				desc:        binaryDesc(path, real),
				primary:     path,
				paths:       []string{path},
				scope:       scopeFor(path, home),
				installedAt: fileModTime(real),
			}
			if root, ok := ownedRoot(real, home); ok {
				app.roots = appendUnique(app.roots, root)
				app.paths = appendUnique(app.paths, root)
			} else if real != "" && real != path {
				app.paths = appendUnique(app.paths, real)
			}
			app.version = versionFromPaths(app.roots, real)
			app.sizeBytes = totalSize(app.paths)
			app.privileged = anyOutsideHome(app.paths, home)
			if app.privileged {
				app.scope = "system"
			}
			apps = append(apps, app)
		}
	}
	return apps
}

func binDirs(home string) []string {
	dirs := []string{
		filepath.Join(home, ".local", "bin"),
		filepath.Join(home, "bin"),
	}
	return append(dirs, systemBinDirs...)
}

// launchersInto returns the entries in binDir whose target resolves inside root
// — the CLI launchers belonging to an app installed under root.
func launchersInto(binDir, root string) []string {
	entries, err := os.ReadDir(binDir)
	if err != nil {
		return nil
	}
	var out []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		p := filepath.Join(binDir, e.Name())
		if underAnyRoot(resolveSymlink(p), []string{root}) {
			out = append(out, p)
		}
	}
	return out
}

// --- install-root resolution ---

// appContainers are directories whose immediate children are each one app's
// self-contained install dir. Used to map a launcher path back to the dir that
// should be removed wholesale.
func appContainers(home string) []string {
	dirs := []string{
		filepath.Join(home, ".local", "share", "ryoku-apps"),
		filepath.Join(home, ".local", "share"),
		filepath.Join(home, ".local", "lib"),
		filepath.Join(home, ".local"),
		filepath.Join(home, "Applications"),
		filepath.Join(home, "apps"),
	}
	return append(dirs, systemContainers...)
}

// systemDesktopDirs, systemBinDirs and systemContainers are the machine-wide
// locations scanned alongside the per-user ones. They are package vars so tests
// can scope discovery to a temp HOME.
var (
	systemDesktopDirs = []string{"/usr/local/share/applications"}
	systemBinDirs     = []string{"/usr/local/bin"}
	systemContainers  = []string{"/opt", "/usr/local"}
)

// sharedSubdirs are the segments directly under a container that are NOT a
// single app (a bare bin/, share/, …), so a binary sitting in one owns no
// install dir of its own.
var sharedSubdirs = map[string]bool{
	"bin": true, "sbin": true, "lib": true, "lib64": true, "libexec": true,
	"share": true, "applications": true, "icons": true, "fonts": true,
	"man": true, "include": true, "etc": true, "games": true, "src": true,
}

// ownedRoot returns the self-contained install directory p belongs to, e.g.
// ~/.local/zed.app for ~/.local/zed.app/bin/zed. It picks the deepest known
// container p sits under, then the single directory beneath it. ok is false
// when p sits directly in a shared dir (a bare bin/), meaning the binary owns
// no directory of its own.
func ownedRoot(p, home string) (string, bool) {
	p = filepath.Clean(p)
	best := ""
	for _, c := range appContainers(home) {
		c = filepath.Clean(c)
		if p != c && strings.HasPrefix(p, c+string(os.PathSeparator)) && len(c) > len(best) {
			best = c
		}
	}
	if best == "" {
		return "", false
	}
	rel := strings.TrimPrefix(p, best+string(os.PathSeparator))
	seg := rel
	if i := strings.IndexByte(rel, os.PathSeparator); i >= 0 {
		seg = rel[:i]
	}
	if seg == "" || sharedSubdirs[strings.ToLower(seg)] {
		return "", false
	}
	return filepath.Join(best, seg), true
}

// --- desktop file parsing ---

// parseDesktopEntry returns the key/values of the [Desktop Entry] group, taking
// the first value for each key and ignoring locale-suffixed keys (Name[fr]).
func parseDesktopEntry(path string) map[string]string {
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()

	m := map[string]string{}
	inEntry := false
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "[") {
			inEntry = line == "[Desktop Entry]"
			continue
		}
		if !inEntry {
			continue
		}
		i := strings.IndexByte(line, '=')
		if i <= 0 {
			continue
		}
		key := strings.TrimSpace(line[:i])
		if strings.IndexByte(key, '[') >= 0 {
			continue // locale-specific override
		}
		if _, exists := m[key]; !exists {
			m[key] = strings.TrimSpace(line[i+1:])
		}
	}
	if len(m) == 0 {
		return nil
	}
	return m
}

// desktopProgram extracts the executable path/name from an Exec/TryExec value,
// dropping surrounding quotes and field codes (%U, %f, …).
func desktopProgram(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	if s[0] == '"' {
		if j := strings.IndexByte(s[1:], '"'); j >= 0 {
			return s[1 : 1+j]
		}
	}
	tok := strings.Fields(s)[0]
	if strings.HasPrefix(tok, "%") {
		return ""
	}
	return tok
}

func resolveProgram(prog string) string {
	if prog == "" {
		return ""
	}
	if !filepath.IsAbs(prog) {
		lp, err := exec.LookPath(prog)
		if err != nil {
			return ""
		}
		prog = lp
	}
	return resolveSymlink(prog)
}

// --- binary classification ---

// isNativeBinary reports whether path is a native executable we treat as an app:
// an ELF file with an executable bit, never a #! interpreter script (those are
// pip/npm/etc. entry points or shell wrappers) and never a shared object.
func isNativeBinary(path string) bool {
	if path == "" {
		return false
	}
	fi, err := os.Stat(path)
	if err != nil || !fi.Mode().IsRegular() || fi.Mode().Perm()&0o111 == 0 {
		return false
	}
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()
	var hdr [4]byte
	n, _ := f.Read(hdr[:])
	if n >= 2 && hdr[0] == '#' && hdr[1] == '!' {
		return false
	}
	return n >= 4 && hdr[0] == 0x7f && hdr[1] == 'E' && hdr[2] == 'L' && hdr[3] == 'F'
}

// isPackageOwned reports whether a system package manager owns path. Only
// meaningful for system locations, so callers skip it (and its subprocess cost)
// for paths under $HOME, which a system package manager never owns.
func isPackageOwned(path string) bool {
	probes := []struct {
		bin  string
		args []string
	}{
		{"pacman", []string{"-Qo"}},
		{"dpkg", []string{"-S"}},
		{"rpm", []string{"-qf"}},
	}
	for _, pr := range probes {
		if !commandExists(pr.bin) {
			continue
		}
		// First available tool is authoritative for this system.
		return exec.Command(pr.bin, append(pr.args, path)...).Run() == nil
	}
	return false
}

// --- versions, sizes, paths ---

// versionFromPaths derives a version from install-dir or binary names, e.g.
// "discord-1.0.144" -> "1.0.144" or ".../versions/2.1.193" -> "2.1.193".
func versionFromPaths(roots []string, real string) string {
	var cands []string
	for _, r := range roots {
		cands = append(cands, filepath.Base(r))
	}
	if real != "" {
		cands = append(cands, filepath.Base(real), filepath.Base(filepath.Dir(real)))
	}
	for _, c := range cands {
		if v := amVersionFromString(c); v != "" {
			return v
		}
	}
	return ""
}

func totalSize(paths []string) int64 {
	var total int64
	seen := map[string]bool{}
	for _, p := range paths {
		if seen[p] {
			continue
		}
		seen[p] = true
		fi, err := os.Lstat(p)
		if err != nil {
			continue
		}
		if fi.IsDir() {
			total += dirSize(p)
		} else {
			total += fi.Size()
		}
	}
	return total
}

func dirSize(root string) int64 {
	var total int64
	_ = filepath.WalkDir(root, func(_ string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if info, err := d.Info(); err == nil && info.Mode().IsRegular() {
			total += info.Size()
		}
		return nil
	})
	return total
}

// --- removal safety ---

// safeRemovablePaths filters a removal set down to paths safe to rm -rf:
// absolute, deep enough, and not a home or system root. A defensive backstop in
// case discovery ever yields a dangerous path.
func safeRemovablePaths(paths []string) []string {
	home, _ := os.UserHomeDir()
	var out []string
	for _, p := range paths {
		if isSafeRemovePath(p, home) {
			out = appendUnique(out, p)
		}
	}
	return out
}

func isSafeRemovePath(p, home string) bool {
	if p == "" || !filepath.IsAbs(p) {
		return false
	}
	clean := filepath.Clean(p)
	if clean == "/" || clean == "." || depth(clean) < 2 {
		return false
	}
	for _, deny := range removeDenylist(home) {
		if clean == filepath.Clean(deny) {
			return false
		}
	}
	return true
}

func removeDenylist(home string) []string {
	d := []string{
		"/", "/usr", "/usr/local", "/usr/local/bin", "/usr/local/share",
		"/usr/local/share/applications", "/usr/bin", "/usr/sbin", "/bin",
		"/sbin", "/lib", "/lib64", "/opt", "/etc", "/var", "/var/opt",
		"/boot", "/dev", "/proc", "/sys", "/run", "/tmp", "/home", "/root", "/srv",
	}
	if home != "" {
		d = append(d,
			home,
			filepath.Join(home, ".local"),
			filepath.Join(home, ".local", "bin"),
			filepath.Join(home, ".local", "lib"),
			filepath.Join(home, ".local", "share"),
			filepath.Join(home, ".local", "share", "applications"),
			filepath.Join(home, ".local", "share", "icons"),
			filepath.Join(home, ".local", "share", "ryoku-apps"),
			filepath.Join(home, ".config"),
			filepath.Join(home, "bin"),
			filepath.Join(home, "Applications"),
			filepath.Join(home, "apps"),
		)
	}
	return d
}

func depth(p string) int {
	p = strings.Trim(filepath.Clean(p), string(os.PathSeparator))
	if p == "" {
		return 0
	}
	return strings.Count(p, string(os.PathSeparator)) + 1
}

// --- small helpers ---

func binaryDesc(path, real string) string {
	dir := tildeify(filepath.Dir(path))
	if real != "" && real != path {
		return "Standalone binary in " + dir + " \u2192 " + tildeify(real)
	}
	return "Standalone binary in " + dir
}

func scopeFor(path, home string) string {
	if underHome(path, home) {
		return "user"
	}
	return "system"
}

func underHome(p, home string) bool {
	if home == "" {
		return false
	}
	p, home = filepath.Clean(p), filepath.Clean(home)
	return p == home || strings.HasPrefix(p, home+string(os.PathSeparator))
}

func anyOutsideHome(paths []string, home string) bool {
	for _, p := range paths {
		if !underHome(p, home) {
			return true
		}
	}
	return false
}

func underAnyRoot(p string, roots []string) bool {
	p = filepath.Clean(p)
	for _, r := range roots {
		r = filepath.Clean(r)
		if p == r || strings.HasPrefix(p, r+string(os.PathSeparator)) {
			return true
		}
	}
	return false
}

func resolveSymlink(p string) string {
	if r, err := filepath.EvalSymlinks(p); err == nil {
		return r
	}
	return p
}

func isRegularFile(p string) bool {
	fi, err := os.Lstat(p)
	return err == nil && fi.Mode().IsRegular()
}

func fileModTime(p string) time.Time {
	if fi, err := os.Lstat(p); err == nil {
		return fi.ModTime()
	}
	return time.Time{}
}

func isTrue(s string) bool { return strings.EqualFold(strings.TrimSpace(s), "true") }

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

func appendUnique(list []string, v string) []string {
	for _, x := range list {
		if x == v {
			return list
		}
	}
	return append(list, v)
}

func tildeify(p string) string {
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		if p == home {
			return "~"
		}
		if strings.HasPrefix(p, home+string(os.PathSeparator)) {
			return "~" + p[len(home):]
		}
	}
	return p
}
