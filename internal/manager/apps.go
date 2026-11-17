package manager

import (
	"bufio"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/neur0map/glazepkg/internal/model"
)

// DesktopApp is one launchable XDG desktop entry joined to the package that
// installed it. Several entries may share one package (an office suite ships
// a launcher per program).
type DesktopApp struct {
	DesktopID   string       `json:"desktop_id"`
	Name        string       `json:"name"`
	GenericName string       `json:"generic_name,omitempty"`
	Comment     string       `json:"comment,omitempty"`
	Icon        string       `json:"icon,omitempty"`
	Categories  []string     `json:"categories"`
	Exec        string       `json:"exec,omitempty"`
	NoDisplay   bool         `json:"no_display"`
	Terminal    bool         `json:"terminal"`
	DesktopFile string       `json:"desktop_file"`
	Package     string       `json:"package,omitempty"`
	Source      model.Source `json:"source"`
	Version     string       `json:"version,omitempty"`
	Size        string       `json:"size,omitempty"`
	SizeBytes   int64        `json:"size_bytes,omitempty"`
	Location    string       `json:"location"` // "system", "user" or "flatpak"
}

// Entry locations.
const (
	AppLocationSystem  = "system"
	AppLocationUser    = "user"
	AppLocationFlatpak = "flatpak"
)

// appDir is one applications directory and the location its entries report.
type appDir struct {
	path     string
	location string
}

// appOwner is the package a desktop file belongs to.
type appOwner struct {
	source  model.Source
	name    string
	version string
	size    int64
}

// appsProbe holds the system queries ScanApps makes, so tests can swap in
// fixtures for the package databases.
type appsProbe struct {
	home     string
	dirs     []appDir
	owners   func(paths []string) map[string]appOwner // system package ownership
	flatpaks func() map[string]model.Package          // installed flatpak apps by id
	lookPath func(string) (string, error)
}

// ScanApps lists every launchable desktop entry visible to the user (their own
// entries, the system dirs in $XDG_DATA_DIRS and flatpak exports), each with
// the owning package, its version and installed size.
func ScanApps() []DesktopApp {
	home, _ := os.UserHomeDir()
	return scanApps(appsProbe{
		home:     home,
		dirs:     applicationDirs(home),
		owners:   systemOwners,
		flatpaks: flatpakApps,
		lookPath: exec.LookPath,
	})
}

// applicationDirs returns the applications dirs in XDG precedence order: the
// user's data home first, then $XDG_DATA_DIRS, then any flatpak export dir
// $XDG_DATA_DIRS didn't already name.
func applicationDirs(home string) []appDir {
	dataHome := os.Getenv("XDG_DATA_HOME")
	if dataHome == "" && home != "" {
		dataHome = filepath.Join(home, ".local", "share")
	}
	dataDirs := os.Getenv("XDG_DATA_DIRS")
	if dataDirs == "" {
		dataDirs = "/usr/local/share:/usr/share"
	}
	var bases []string
	if dataHome != "" {
		bases = append(bases, dataHome)
	}
	bases = append(bases, filepath.SplitList(dataDirs)...)
	if home != "" {
		bases = append(bases, filepath.Join(home, ".local", "share", "flatpak", "exports", "share"))
	}
	bases = append(bases, "/var/lib/flatpak/exports/share")

	var dirs []appDir
	seen := map[string]bool{}
	for _, b := range bases {
		if b == "" || !filepath.IsAbs(b) {
			continue
		}
		p := filepath.Join(filepath.Clean(b), "applications")
		if seen[p] {
			continue
		}
		seen[p] = true
		dirs = append(dirs, appDir{path: p, location: dirLocation(p, dataHome, home)})
	}
	return dirs
}

func dirLocation(dir, dataHome, home string) string {
	if strings.Contains(dir, "/flatpak/exports/") {
		return AppLocationFlatpak
	}
	if (dataHome != "" && underHome(dir, dataHome)) || underHome(dir, home) {
		return AppLocationUser
	}
	return AppLocationSystem
}

// desktopFile is one .desktop file found while walking the applications dirs.
type desktopFile struct {
	id    string
	path  string
	dir   appDir
	top   bool // directly in dir, not in a subdirectory
	entry map[string]string
}

// walkDesktopFiles returns every .desktop file in dirs keyed by desktop id,
// each id's files in precedence order (the first one wins).
func walkDesktopFiles(dirs []appDir) (map[string][]desktopFile, []string) {
	byID := map[string][]desktopFile{}
	var order []string
	walked := map[string]bool{}
	for _, d := range dirs {
		// WalkDir doesn't descend into a symlinked root, and dotfile managers
		// often symlink ~/.local/share/applications; walk the real dir and
		// report paths under the name the user knows. Two names for one real
		// dir are walked once.
		root, err := filepath.EvalSymlinks(d.path)
		if err != nil || walked[root] {
			continue
		}
		walked[root] = true
		_ = filepath.WalkDir(root, func(p string, de os.DirEntry, err error) error {
			if err != nil {
				if p == root {
					return filepath.SkipDir
				}
				return nil
			}
			if de.IsDir() || !strings.HasSuffix(de.Name(), ".desktop") {
				return nil
			}
			rel, err := filepath.Rel(root, p)
			if err != nil {
				return nil
			}
			id := strings.ReplaceAll(strings.TrimSuffix(rel, ".desktop"), string(os.PathSeparator), "-")
			if _, ok := byID[id]; !ok {
				order = append(order, id)
			}
			byID[id] = append(byID[id], desktopFile{
				id: id, path: filepath.Join(d.path, rel), dir: d,
				top: !strings.ContainsRune(rel, os.PathSeparator),
			})
			return nil
		})
	}
	return byID, order
}

func scanApps(pr appsProbe) []DesktopApp {
	byID, order := walkDesktopFiles(pr.dirs)

	// Pick the winning file per id and drop what isn't a launchable application.
	type pick struct {
		win    desktopFile
		system string // a shadowed system file of the same id, for ownership
	}
	var picks []pick
	var systemPaths []string
	for _, id := range order {
		files := byID[id]
		win := files[0]
		win.entry = parseDesktopEntry(win.path)
		e := win.entry
		if e == nil || !strings.EqualFold(e["Type"], "Application") || isTrue(e["Hidden"]) {
			continue
		}
		if te := strings.TrimSpace(e["TryExec"]); te != "" && pr.lookPath != nil {
			if _, err := pr.lookPath(te); err != nil {
				continue
			}
		}
		p := pick{win: win}
		switch win.dir.location {
		case AppLocationSystem:
			systemPaths = append(systemPaths, win.path)
		case AppLocationUser:
			// A user copy of a packaged entry (to tweak Exec or hide it) still
			// belongs to the package that ships the original.
			for _, f := range files[1:] {
				if f.dir.location == AppLocationSystem {
					p.system = f.path
					systemPaths = append(systemPaths, f.path)
					break
				}
			}
		}
		picks = append(picks, p)
	}

	var (
		owners   map[string]appOwner
		flatpaks map[string]model.Package
		wg       sync.WaitGroup
	)
	if pr.owners != nil && len(systemPaths) > 0 {
		wg.Add(1)
		go func() { defer wg.Done(); owners = pr.owners(systemPaths) }()
	}
	if pr.flatpaks != nil {
		wg.Add(1)
		go func() { defer wg.Done(); flatpaks = pr.flatpaks() }()
	}
	wg.Wait()

	localDirs := map[string]bool{}
	for _, d := range desktopAppDirs(pr.home) {
		localDirs[filepath.Clean(d)] = true
	}

	apps := make([]DesktopApp, 0, len(picks))
	for _, p := range picks {
		e := p.win.entry
		app := DesktopApp{
			DesktopID:   p.win.id,
			Name:        firstNonEmpty(e["Name"], p.win.id),
			GenericName: e["GenericName"],
			Comment:     e["Comment"],
			Icon:        e["Icon"],
			Categories:  splitDesktopList(e["Categories"]),
			Exec:        e["Exec"],
			NoDisplay:   isTrue(e["NoDisplay"]),
			Terminal:    isTrue(e["Terminal"]),
			DesktopFile: p.win.path,
			Location:    p.win.dir.location,
		}

		switch {
		case e["X-Flatpak"] != "" || p.win.dir.location == AppLocationFlatpak:
			id := firstNonEmpty(e["X-Flatpak"], flatpakIDFromPath(resolveSymlink(p.win.path)))
			app.Source = model.SourceFlatpak
			app.Package = id
			if fp, ok := flatpaks[id]; ok {
				app.Version = fp.Version
				app.SizeBytes = fp.SizeBytes
			}
		case e["X-SnapInstanceName"] != "":
			app.Source = model.SourceSnap
			app.Package = e["X-SnapInstanceName"]
		default:
			o, ok := owners[p.win.path]
			if !ok && p.system != "" {
				o, ok = owners[p.system]
			}
			if ok {
				app.Source = o.source
				app.Package = o.name
				app.Version = o.version
				app.SizeBytes = o.size
				break
			}
			// Nothing owns it: an app installed by hand, sized the way the
			// local manager measures it.
			la := localAppFromEntry(p.win.path, pr.home, e)
			app.Source = model.SourceLocal
			app.Version = la.version
			app.SizeBytes = la.sizeBytes
			// Only entries `gpk list -m local` reports can be removed by name.
			if p.win.top && !app.NoDisplay && localDirs[filepath.Clean(p.win.dir.path)] {
				app.Package = la.name
			}
		}
		if app.SizeBytes > 0 {
			app.Size = FormatBytes(app.SizeBytes)
		}
		apps = append(apps, app)
	}
	sort.SliceStable(apps, func(i, j int) bool {
		ni, nj := strings.ToLower(apps[i].Name), strings.ToLower(apps[j].Name)
		if ni != nj {
			return ni < nj
		}
		return apps[i].DesktopID < apps[j].DesktopID
	})
	return apps
}

// splitDesktopList splits a desktop-entry list value ("Network;WebBrowser;").
// Always returns a non-nil slice so JSON shows [] rather than null.
func splitDesktopList(v string) []string {
	out := []string{}
	for _, s := range strings.Split(v, ";") {
		if s = strings.TrimSpace(s); s != "" {
			out = append(out, s)
		}
	}
	return out
}

// flatpakIDFromPath reads the app id from a path inside a flatpak install,
// e.g. /var/lib/flatpak/app/org.gimp.GIMP/current/active/export/share/....
func flatpakIDFromPath(p string) string {
	parts := strings.Split(filepath.ToSlash(p), "/")
	for i := range len(parts) - 1 {
		if parts[i] == "app" && i > 0 && parts[i-1] == "flatpak" {
			return parts[i+1]
		}
	}
	return strings.TrimSuffix(filepath.Base(p), ".desktop")
}

// flatpakApps returns the installed flatpak apps keyed by app id.
func flatpakApps() map[string]model.Package {
	if !commandExists("flatpak") {
		return nil
	}
	pkgs, err := (&Flatpak{}).Scan()
	if err != nil {
		return nil
	}
	m := make(map[string]model.Package, len(pkgs))
	for _, p := range pkgs {
		if _, dup := m[p.Name]; !dup {
			m[p.Name] = p
		}
	}
	return m
}

// --- system package ownership ---

// systemOwners maps each path to the system package that owns it, using the
// host's package database in one batched query.
func systemOwners(paths []string) map[string]appOwner {
	switch {
	case commandExists("pacman"):
		return pacmanOwners(paths)
	case commandExists("dnf") && commandExists("rpm"):
		return rpmOwners(paths)
	case commandExists("xbps-query"):
		return xbpsOwners(paths)
	case commandExists("dpkg-query"):
		return dpkgOwners(paths)
	}
	return nil
}

func pacmanOwners(paths []string) map[string]appOwner {
	// pacman exits non-zero when any path is unowned but prints the rest.
	out, _ := exec.Command("pacman", append([]string{"-Qo"}, paths...)...).Output()
	owners := ParsePacmanOwners(string(out))
	if len(owners) == 0 {
		return owners
	}
	foreign := map[string]bool{}
	if fo, err := exec.Command("pacman", "-Qmq").Output(); err == nil || len(fo) > 0 {
		for _, n := range strings.Fields(string(fo)) {
			foreign[n] = true
		}
	}
	var names []string
	seen := map[string]bool{}
	for _, o := range owners {
		if !seen[o.name] {
			seen[o.name] = true
			names = append(names, o.name)
		}
	}
	sizes := pacmanSizes(names)
	for p, o := range owners {
		if foreign[o.name] {
			o.source = model.SourceAUR
		}
		o.size = sizes[o.name]
		owners[p] = o
	}
	return owners
}

// ParsePacmanOwners reads `pacman -Qo <paths>` output,
// "/usr/share/applications/foo.desktop is owned by foo 1.2-1".
func ParsePacmanOwners(out string) map[string]appOwner {
	owners := map[string]appOwner{}
	sc := bufio.NewScanner(strings.NewReader(out))
	for sc.Scan() {
		path, rest, ok := strings.Cut(sc.Text(), " is owned by ")
		if !ok {
			continue
		}
		f := strings.Fields(rest)
		if len(f) < 1 {
			continue
		}
		o := appOwner{source: model.SourcePacman, name: f[0]}
		if len(f) >= 2 {
			o.version = f[1]
		}
		owners[strings.TrimSpace(path)] = o
	}
	return owners
}

const rpmOwnerQF = `%{NAME}\t%{VERSION}-%{RELEASE}\t%{SIZE}\n`

func rpmOwners(paths []string) map[string]appOwner {
	out, _ := exec.Command("rpm", append([]string{"-qf", "--qf", rpmOwnerQF}, paths...)...).Output()
	if owners, ok := ParseRpmOwners(paths, string(out)); ok {
		return owners
	}
	// A file owned by several packages prints several lines, so the batch no
	// longer lines up with its paths: ask per path instead.
	owners := map[string]appOwner{}
	for _, p := range paths {
		out, _ := exec.Command("rpm", "-qf", "--qf", rpmOwnerQF, p).Output()
		if o, ok := ParseRpmOwners([]string{p}, firstLine(string(out))); ok {
			for k, v := range o {
				owners[k] = v
			}
		}
	}
	return owners
}

// ParseRpmOwners reads `rpm -qf --qf '%{NAME}\t%{VERSION}-%{RELEASE}\t%{SIZE}\n'`
// output for paths, which rpm answers one line per path in order ("file ... is
// not owned by any package" for unowned ones). ok is false when the line count
// doesn't match the paths, meaning some file has more than one owner.
func ParseRpmOwners(paths []string, out string) (map[string]appOwner, bool) {
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	if out == "" {
		lines = nil
	}
	if len(lines) != len(paths) {
		return nil, false
	}
	owners := map[string]appOwner{}
	for i, line := range lines {
		f := strings.Split(line, "\t")
		if len(f) != 3 {
			continue // not owned
		}
		size, _ := strconv.ParseInt(strings.TrimSpace(f[2]), 10, 64)
		owners[paths[i]] = appOwner{source: model.SourceDnf, name: f[0], version: f[1], size: size}
	}
	return owners, true
}

func xbpsOwners(paths []string) map[string]appOwner {
	// One pattern per directory: xbps-query -o takes a single fnmatch pattern,
	// and "*" crosses "/" so subdirectories are covered too.
	dirs := map[string]bool{}
	var order []string
	for _, p := range paths {
		d := filepath.Dir(p)
		if !dirs[d] {
			dirs[d] = true
			order = append(order, d)
		}
	}
	want := map[string]bool{}
	for _, p := range paths {
		want[p] = true
	}
	owners := map[string]appOwner{}
	for _, d := range order {
		out, _ := exec.Command("xbps-query", "-o", d+"/*").Output()
		for p, o := range ParseXbpsOwners(string(out)) {
			if want[p] {
				owners[p] = o
			}
		}
	}
	if len(owners) == 0 {
		return owners
	}
	sizes := xbpsSizes()
	for p, o := range owners {
		o.size = sizes[o.name]
		owners[p] = o
	}
	return owners
}

// ParseXbpsOwners reads `xbps-query -o <pattern>` output,
// "foo-1.2_1: /usr/share/applications/foo.desktop (regular file)", where a
// link adds " -> target" before the type.
func ParseXbpsOwners(out string) map[string]appOwner {
	owners := map[string]appOwner{}
	sc := bufio.NewScanner(strings.NewReader(out))
	for sc.Scan() {
		pkgver, rest, ok := strings.Cut(sc.Text(), ": ")
		if !ok {
			continue
		}
		if i := strings.LastIndex(rest, " ("); i >= 0 && strings.HasSuffix(rest, ")") {
			rest = rest[:i]
		}
		if i := strings.Index(rest, " -> "); i >= 0 {
			rest = rest[:i]
		}
		name, version := SplitXbpsNameVersion(strings.TrimSpace(pkgver))
		if name == "" || rest == "" {
			continue
		}
		owners[strings.TrimSpace(rest)] = appOwner{source: model.SourceXbps, name: name, version: version}
	}
	return owners
}

func dpkgOwners(paths []string) map[string]appOwner {
	out, _ := exec.Command("dpkg-query", append([]string{"-S"}, paths...)...).Output()
	owners := ParseDpkgOwners(string(out))
	if len(owners) == 0 {
		return owners
	}
	var names []string
	seen := map[string]bool{}
	for _, o := range owners {
		if !seen[o.name] {
			seen[o.name] = true
			names = append(names, o.name)
		}
	}
	info, _ := exec.Command("dpkg-query", append([]string{"-W", "-f", `${Package}\t${Version}\t${Installed-Size}\n`}, names...)...).Output()
	meta := ParseDpkgSizes(string(info))
	for p, o := range owners {
		if m, ok := meta[o.name]; ok {
			o.version, o.size = m.version, m.size
		}
		owners[p] = o
	}
	return owners
}

// ParseDpkgOwners reads `dpkg-query -S <paths>` output, "pkg[:arch]: /path";
// the first package wins when several are listed ("a, b: /path").
func ParseDpkgOwners(out string) map[string]appOwner {
	owners := map[string]appOwner{}
	sc := bufio.NewScanner(strings.NewReader(out))
	for sc.Scan() {
		line := sc.Text()
		i := strings.LastIndex(line, ": ")
		if i < 0 || strings.HasPrefix(line, "diversion ") {
			continue
		}
		pkg := strings.TrimSpace(strings.Split(line[:i], ",")[0])
		if j := strings.IndexByte(pkg, ':'); j >= 0 {
			pkg = pkg[:j]
		}
		if pkg == "" {
			continue
		}
		owners[strings.TrimSpace(line[i+2:])] = appOwner{source: model.SourceApt, name: pkg}
	}
	return owners
}

// ParseDpkgSizes reads `dpkg-query -W -f '${Package}\t${Version}\t${Installed-Size}\n'`
// output; Installed-Size is in KiB.
func ParseDpkgSizes(out string) map[string]appOwner {
	m := map[string]appOwner{}
	sc := bufio.NewScanner(strings.NewReader(out))
	for sc.Scan() {
		f := strings.Split(sc.Text(), "\t")
		if len(f) != 3 {
			continue
		}
		kib, _ := strconv.ParseInt(strings.TrimSpace(f[2]), 10, 64)
		m[f[0]] = appOwner{source: model.SourceApt, name: f[0], version: f[1], size: kib * 1024}
	}
	return m
}

func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i+1]
	}
	return s
}

// --- cache ---

// appsCacheTTL bounds how long a cached app list is trusted even when nothing
// it fingerprints changed: hand-installed apps can grow in place.
const appsCacheTTL = 24 * time.Hour

const appsCacheVersion = 1

type appsCache struct {
	Version     int          `json:"version"`
	Timestamp   time.Time    `json:"timestamp"`
	Fingerprint string       `json:"fingerprint"`
	Apps        []DesktopApp `json:"apps"`
}

func appsCachePath() string {
	return filepath.Join(filepath.Dir(scanCachePath()), "apps.json")
}

// appsFingerprint stats the applications dirs and the package databases. Any
// install, removal or upgrade touches at least one of them, so a matching
// fingerprint means the cached list still describes the system.
func appsFingerprint(dirs []appDir, home string) string {
	paths := []string{
		"/var/lib/pacman/local",
		"/var/lib/rpm/rpmdb.sqlite",
		"/usr/lib/sysimage/rpm/rpmdb.sqlite",
		"/var/db/xbps",
		"/var/lib/dpkg/status",
		"/var/lib/flatpak/.changed",
	}
	if home != "" {
		paths = append(paths, filepath.Join(home, ".local", "share", "flatpak", ".changed"))
	}
	for _, d := range dirs {
		paths = append(paths, d.path)
		// Subdirectories hold entries too (e.g. applications/kde4/).
		if ents, err := os.ReadDir(d.path); err == nil {
			for _, e := range ents {
				if e.IsDir() {
					paths = append(paths, filepath.Join(d.path, e.Name()))
				}
			}
		}
	}
	var b strings.Builder
	for _, p := range paths {
		b.WriteString(p)
		b.WriteByte('=')
		if fi, err := os.Stat(p); err == nil {
			b.WriteString(strconv.FormatInt(fi.ModTime().UnixNano(), 10))
		} else {
			b.WriteByte('-')
		}
		b.WriteByte(';')
	}
	return b.String()
}

// LoadApps returns the desktop app list, from the cache when it is fresh and
// the system hasn't changed since, otherwise from a live scan written back to
// the cache. noCache forces the live scan.
func LoadApps(noCache bool) []DesktopApp {
	home, _ := os.UserHomeDir()
	fp := appsFingerprint(applicationDirs(home), home)
	if !noCache {
		if data, err := os.ReadFile(appsCachePath()); err == nil {
			var c appsCache
			if json.Unmarshal(data, &c) == nil && c.Version == appsCacheVersion &&
				c.Fingerprint == fp && time.Since(c.Timestamp) <= appsCacheTTL && c.Apps != nil {
				return c.Apps
			}
		}
	}
	apps := ScanApps()
	saveAppsCache(appsCache{Version: appsCacheVersion, Timestamp: time.Now(), Fingerprint: fp, Apps: apps})
	return apps
}

func saveAppsCache(c appsCache) {
	data, err := json.Marshal(c)
	if err != nil {
		return
	}
	path := appsCachePath()
	_ = os.MkdirAll(filepath.Dir(path), 0o755)
	tmp, err := os.CreateTemp(filepath.Dir(path), "apps-*.json")
	if err != nil {
		return
	}
	name := tmp.Name()
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(name)
		return
	}
	if tmp.Close() != nil || os.Chmod(name, 0o644) != nil || os.Rename(name, path) != nil {
		os.Remove(name)
	}
}

// DeleteAppsCache drops the cached app list so the next `gpk apps` rescans.
func DeleteAppsCache() {
	_ = os.Remove(appsCachePath())
}
