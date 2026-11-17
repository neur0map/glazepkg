package manager

import (
	"bufio"
	"os/exec"
	"strconv"
	"strings"
	"unicode"

	"github.com/neur0map/glazepkg/internal/model"
)

// Installed sizes for the system managers. Each manager answers in one batched
// query for every package it reports rather than one subprocess per package,
// so filling sizes costs a scan a single extra command.

// applySizes fills Size and SizeBytes from a name -> bytes map. Packages the
// map doesn't know keep whatever they had.
func applySizes(pkgs []model.Package, sizes map[string]int64) {
	for i := range pkgs {
		if b, ok := sizes[pkgs[i].Name]; ok && b > 0 {
			pkgs[i].SizeBytes = b
			pkgs[i].Size = FormatBytes(b)
		}
	}
}

// pacmanSizes returns installed sizes for the named pacman packages: exact
// bytes from expac when it's installed, otherwise parsed from `pacman -Qi`.
func pacmanSizes(names []string) map[string]int64 {
	if len(names) == 0 {
		return nil
	}
	if commandExists("expac") {
		out, err := exec.Command("expac", append([]string{"-Q", `%n\t%m`}, names...)...).Output()
		if err == nil || len(out) > 0 {
			if sizes := ParseExpacSizes(string(out)); len(sizes) > 0 {
				return sizes
			}
		}
	}
	// pacman exits non-zero when any name is missing but still prints the rest.
	out, _ := exec.Command("pacman", append([]string{"-Qi"}, names...)...).Output()
	return ParsePacmanQiSizes(string(out))
}

// ParseExpacSizes reads `expac -Q '%n\t%m'` output: name, tab, bytes.
func ParseExpacSizes(out string) map[string]int64 {
	sizes := make(map[string]int64)
	sc := bufio.NewScanner(strings.NewReader(out))
	for sc.Scan() {
		name, val, ok := strings.Cut(sc.Text(), "\t")
		if !ok {
			continue
		}
		if b, err := strconv.ParseInt(strings.TrimSpace(val), 10, 64); err == nil && b >= 0 {
			sizes[strings.TrimSpace(name)] = b
		}
	}
	return sizes
}

// ParsePacmanQiSizes reads the Name and Installed Size fields from
// `pacman -Qi` output for any number of packages (blocks separated by blank
// lines). Needs the C locale, which the CLI sets.
func ParsePacmanQiSizes(out string) map[string]int64 {
	sizes := make(map[string]int64)
	name := ""
	sc := bufio.NewScanner(strings.NewReader(out))
	for sc.Scan() {
		line := sc.Text()
		if strings.TrimSpace(line) == "" {
			name = ""
			continue
		}
		key, val, ok := parseField(line)
		if !ok {
			continue
		}
		switch key {
		case "Name":
			name = val
		case "Installed Size":
			if name != "" {
				sizes[name] = ParseSizeString(val)
			}
		}
	}
	return sizes
}

// rpmSizes returns the installed size of every rpm package by name. Several
// installed arches of one name (multilib) are summed: both occupy disk.
func rpmSizes() map[string]int64 {
	out, err := exec.Command("rpm", "-qa", "--qf", `%{NAME}\t%{SIZE}\n`).Output()
	if err != nil && len(out) == 0 {
		return nil
	}
	return ParseRpmSizes(string(out))
}

// ParseRpmSizes reads `rpm -qa --qf '%{NAME}\t%{SIZE}\n'` output.
func ParseRpmSizes(out string) map[string]int64 {
	sizes := make(map[string]int64)
	sc := bufio.NewScanner(strings.NewReader(out))
	for sc.Scan() {
		name, val, ok := strings.Cut(sc.Text(), "\t")
		if !ok {
			continue
		}
		if b, err := strconv.ParseInt(strings.TrimSpace(val), 10, 64); err == nil && b >= 0 {
			sizes[strings.TrimSpace(name)] += b
		}
	}
	return sizes
}

// xbpsSizes returns the installed size of every installed xbps package. The
// property search prints "pkgver: <size>" for each package whose property
// matches the pattern, and an empty pattern matches every value.
func xbpsSizes() map[string]int64 {
	out, err := exec.Command("xbps-query", "-p", "installed_size", "-s", "").Output()
	if err != nil && len(out) == 0 {
		return nil
	}
	return ParseXbpsSizes(string(out))
}

// ParseXbpsSizes reads `xbps-query -p installed_size -s ”` output, lines of
// "name-version_rev: 1.5MB".
func ParseXbpsSizes(out string) map[string]int64 {
	sizes := make(map[string]int64)
	sc := bufio.NewScanner(strings.NewReader(out))
	for sc.Scan() {
		pkgver, val, ok := strings.Cut(sc.Text(), ": ")
		if !ok {
			continue
		}
		name, _ := SplitXbpsNameVersion(strings.TrimSpace(pkgver))
		if b := ParseXbpsSize(val); b > 0 && name != "" {
			sizes[name] = b
		}
	}
	return sizes
}

// ParseXbpsSize reads xbps' humanize_number form: binary multiples with no
// space and a "B" suffix ("512B", "1.5KB", "12MB", "1.2GB").
func ParseXbpsSize(s string) int64 {
	s = strings.TrimSpace(s)
	i := strings.IndexFunc(s, func(r rune) bool { return !(unicode.IsDigit(r) || r == '.') })
	if i <= 0 {
		return 0
	}
	val, err := strconv.ParseFloat(s[:i], 64)
	if err != nil {
		return 0
	}
	mult := map[string]float64{
		"B": 1, "KB": 1 << 10, "MB": 1 << 20, "GB": 1 << 30, "TB": 1 << 40, "PB": 1 << 50,
	}[strings.ToUpper(strings.TrimSpace(s[i:]))]
	if mult == 0 {
		return 0
	}
	return int64(val * mult)
}

// ParseFlatpakSize reads flatpak's GLib-formatted sizes: decimal (SI) units,
// with a regular or no-break space before the unit ("14.1\u00a0MB", "1.2 GB",
// "980 bytes").
func ParseFlatpakSize(s string) int64 {
	s = strings.TrimSpace(strings.ReplaceAll(s, "\u00a0", " "))
	parts := strings.Fields(s)
	if len(parts) != 2 {
		return 0
	}
	val, err := strconv.ParseFloat(strings.ReplaceAll(parts[0], ",", "."), 64)
	if err != nil {
		return 0
	}
	mult := map[string]float64{
		"byte": 1, "bytes": 1, "b": 1,
		"kb": 1e3, "mb": 1e6, "gb": 1e9, "tb": 1e12, "pb": 1e15,
		"kib": 1 << 10, "mib": 1 << 20, "gib": 1 << 30, "tib": 1 << 40,
	}[strings.ToLower(parts[1])]
	if mult == 0 {
		return 0
	}
	return int64(val * mult)
}
