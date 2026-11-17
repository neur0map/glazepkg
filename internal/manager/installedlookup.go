package manager

import (
	"bufio"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/neur0map/glazepkg/internal/model"
)

// InstalledLooker answers "is this name installed at all?" for a system
// manager whose Scan lists only what the user asked for (pacman's explicit
// packages, dnf's user-installed set, xbps' manual set). Remove and installed
// use it when a manager is named, so a package that came in as a dependency
// can still be found; list keeps the narrower scan. Each returned package
// carries RequiredBy: the installed packages that depend on it.
type InstalledLooker interface {
	LookupInstalled(names []string) map[string]model.Package
}

// --- pacman / aur ---

// pacmanLookup queries every name with `pacman -Qi`, keeping native packages
// (foreign=false) or foreign ones (foreign=true), so pacman and aur never both
// claim a name.
func pacmanLookup(names []string, foreign bool) map[string]model.Package {
	if len(names) == 0 {
		return nil
	}
	// Both commands exit non-zero when any name is missing but print the rest.
	out, _ := exec.Command("pacman", append([]string{"-Qi"}, names...)...).Output()
	fo, _ := exec.Command("pacman", append([]string{"-Qmq"}, names...)...).Output()
	isForeign := map[string]bool{}
	for _, n := range strings.Fields(string(fo)) {
		isForeign[n] = true
	}
	src := model.SourcePacman
	if foreign {
		src = model.SourceAUR
	}
	found := map[string]model.Package{}
	for _, p := range ParsePacmanQiPackages(string(out)) {
		if isForeign[p.Name] != foreign {
			continue
		}
		p.Source = src
		found[p.Name] = p
	}
	return found
}

func (p *Pacman) LookupInstalled(names []string) map[string]model.Package {
	return pacmanLookup(names, false)
}

func (a *AUR) LookupInstalled(names []string) map[string]model.Package {
	return pacmanLookup(names, true)
}

// ParsePacmanQiPackages reads `pacman -Qi` output for any number of packages:
// name, version, installed size and Required By. A field value pacman wraps
// onto indented continuation lines is joined back together.
func ParsePacmanQiPackages(out string) []model.Package {
	var pkgs []model.Package
	var cur *model.Package
	lastKey := ""
	flush := func() {
		if cur != nil && cur.Name != "" {
			pkgs = append(pkgs, *cur)
		}
		cur, lastKey = nil, ""
	}
	set := func(key, val string) {
		switch key {
		case "Name":
			cur.Name = val
		case "Version":
			cur.Version = val
		case "Description":
			cur.Description = val
		case "Installed Size":
			cur.SizeBytes = ParseSizeString(val)
			if cur.SizeBytes > 0 {
				cur.Size = FormatBytes(cur.SizeBytes)
			}
		case "Required By":
			if val != "None" {
				cur.RequiredBy = append(cur.RequiredBy, strings.Fields(val)...)
			}
		}
	}
	sc := bufio.NewScanner(strings.NewReader(out))
	for sc.Scan() {
		line := sc.Text()
		if strings.TrimSpace(line) == "" {
			flush()
			continue
		}
		if cur == nil {
			cur = &model.Package{Source: model.SourcePacman, InstalledAt: time.Now()}
		}
		if (line[0] == ' ' || line[0] == '\t') && !strings.Contains(line, " : ") {
			if lastKey == "Required By" {
				set(lastKey, strings.TrimSpace(line))
			}
			continue
		}
		key, val, ok := strings.Cut(line, " : ")
		if !ok {
			continue
		}
		lastKey = strings.TrimSpace(key)
		set(lastKey, strings.TrimSpace(val))
	}
	flush()
	return pkgs
}

// --- dnf (rpm database) ---

func (d *Dnf) LookupInstalled(names []string) map[string]model.Package {
	if len(names) == 0 || !commandExists("rpm") {
		return nil
	}
	out, _ := exec.Command("rpm", append([]string{"-q", "--qf", `%{NAME}\t%{VERSION}-%{RELEASE}\t%{SIZE}\n`}, names...)...).Output()
	found := ParseRpmQueryPackages(string(out))
	for name, p := range found {
		p.RequiredBy = rpmRequiredBy(name)
		found[name] = p
	}
	return found
}

// rpmRequiredBy returns the installed packages that require anything name
// provides. Dependents rarely require a package by name: a program linked to
// a library requires its soname ("libxkbcommon.so.0()(64bit)"), and others
// require pkgconfig(...) or a virtual capability. So every capability the
// package provides is asked for, plus the name itself, in one query.
func rpmRequiredBy(name string) []string {
	prov, _ := exec.Command("rpm", "-q", "--provides", name).Output()
	caps := ParseRpmProvides(string(prov))
	caps = append([]string{name}, caps...)
	req, _ := exec.Command("rpm", append([]string{"-q", "--whatrequires", "--qf", `%{NAME}\n`}, caps...)...).Output()
	return ParseRpmWhatRequires(string(req), name)
}

// ParseRpmProvides reads `rpm -q --provides <pkg>` output into capability
// names, dropping the version constraint ("libfoo(x86-64) = 1.2-1" ->
// "libfoo(x86-64)") and duplicates.
func ParseRpmProvides(out string) []string {
	var caps []string
	seen := map[string]bool{}
	sc := bufio.NewScanner(strings.NewReader(out))
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "package ") {
			continue // "package x is not installed"
		}
		c := line
		for _, op := range []string{" = ", " >= ", " <= ", " > ", " < "} {
			if i := strings.Index(c, op); i >= 0 {
				c = c[:i]
			}
		}
		c = strings.TrimSpace(c)
		if c != "" && !seen[c] {
			seen[c] = true
			caps = append(caps, c)
		}
	}
	return caps
}

// ParseRpmQueryPackages reads `rpm -q --qf '%{NAME}\t%{VERSION}-%{RELEASE}\t%{SIZE}\n'`
// output; "package x is not installed" lines are skipped. Several installed
// arches of one name are summed.
func ParseRpmQueryPackages(out string) map[string]model.Package {
	found := map[string]model.Package{}
	sc := bufio.NewScanner(strings.NewReader(out))
	for sc.Scan() {
		f := strings.Split(sc.Text(), "\t")
		if len(f) != 3 || f[0] == "" {
			continue
		}
		size, _ := strconv.ParseInt(strings.TrimSpace(f[2]), 10, 64)
		p := found[f[0]]
		p.Name, p.Version, p.Source = f[0], f[1], model.SourceDnf
		p.SizeBytes += size
		if p.SizeBytes > 0 {
			p.Size = FormatBytes(p.SizeBytes)
		}
		if p.InstalledAt.IsZero() {
			p.InstalledAt = time.Now()
		}
		found[f[0]] = p
	}
	return found
}

// ParseRpmWhatRequires reads `rpm -q --whatrequires --qf '%{NAME}\n' <caps...>`
// output: the requiring package names, each once, without self (a package
// often requires its own capabilities) and without rpm's "no package
// requires ..." lines for capabilities nothing needs.
func ParseRpmWhatRequires(out, self string) []string {
	var names []string
	seen := map[string]bool{self: true}
	sc := bufio.NewScanner(strings.NewReader(out))
	for sc.Scan() {
		n := strings.TrimSpace(sc.Text())
		if n == "" || strings.Contains(n, " ") || seen[n] {
			continue
		}
		seen[n] = true
		names = append(names, n)
	}
	return names
}

// --- xbps ---

func (x *Xbps) LookupInstalled(names []string) map[string]model.Package {
	found := map[string]model.Package{}
	var sizes map[string]int64
	for _, name := range names {
		out, err := exec.Command("xbps-query", "-p", "pkgver", name).Output()
		if err != nil {
			continue
		}
		pn, ver := SplitXbpsNameVersion(strings.TrimSpace(string(out)))
		if pn != name {
			continue
		}
		if sizes == nil {
			sizes = xbpsSizes()
		}
		p := model.Package{Name: name, Version: ver, Source: model.SourceXbps, InstalledAt: time.Now()}
		if b := sizes[name]; b > 0 {
			p.SizeBytes, p.Size = b, FormatBytes(b)
		}
		rev, _ := exec.Command("xbps-query", "-X", name).Output()
		p.RequiredBy = ParseXbpsRevdeps(string(rev))
		found[name] = p
	}
	return found
}

// ParseXbpsRevdeps reads `xbps-query -X <pkg>` output, one pkgver per line,
// into package names.
func ParseXbpsRevdeps(out string) []string {
	var names []string
	sc := bufio.NewScanner(strings.NewReader(out))
	for sc.Scan() {
		if n, _ := SplitXbpsNameVersion(strings.TrimSpace(sc.Text())); n != "" {
			names = append(names, n)
		}
	}
	return names
}
