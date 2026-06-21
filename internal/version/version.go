// Package version compares package versions written in the many shapes the
// supported managers use — semver, pacman's epoch:ver-rel, apt's ver-rev with
// tildes, pip's post/dev suffixes, leading-v Go tags. It follows Debian's
// well-tested algorithm, which orders all of these sensibly.
package version

import "strings"

// Compare returns -1, 0, or 1 as a is older than, equal to, or newer than b.
func Compare(a, b string) int {
	a, b = strings.TrimSpace(a), strings.TrimSpace(b)
	a, b = trimLeadingV(a), trimLeadingV(b)
	if a == b {
		return 0
	}

	ae, ar := splitEpoch(a)
	be, br := splitEpoch(b)
	if c := verrevcmp(ae, be); c != 0 {
		return c
	}

	aUp, aRev := splitRevision(ar)
	bUp, bRev := splitRevision(br)
	if c := verrevcmp(aUp, bUp); c != 0 {
		return c
	}
	return verrevcmp(aRev, bRev)
}

// Less reports whether a is older than b, for sort interfaces.
func Less(a, b string) bool { return Compare(a, b) < 0 }

func trimLeadingV(s string) string {
	if len(s) >= 2 && (s[0] == 'v' || s[0] == 'V') && s[1] >= '0' && s[1] <= '9' {
		return s[1:]
	}
	return s
}

// splitEpoch separates a leading "N:" epoch from the rest. A missing epoch is
// "0" so unversioned-epoch strings compare below explicit ones.
func splitEpoch(s string) (epoch, rest string) {
	if i := strings.IndexByte(s, ':'); i > 0 && allDigits(s[:i]) {
		return s[:i], s[i+1:]
	}
	return "0", s
}

// splitRevision separates the upstream version from the trailing revision at the
// last '-' (pacman pkgrel, apt debian revision).
func splitRevision(s string) (upstream, revision string) {
	if i := strings.LastIndexByte(s, '-'); i >= 0 {
		return s[:i], s[i+1:]
	}
	return s, ""
}

func allDigits(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}

func isDigit(c byte) bool { return c >= '0' && c <= '9' }
func isAlpha(c byte) bool { return c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' }

// order ranks a byte for the non-digit comparison: '~' sorts before everything
// (even the empty string), letters keep their natural order, and other
// punctuation sorts after letters.
func order(c byte) int {
	switch {
	case isDigit(c):
		return 0
	case isAlpha(c):
		return int(c)
	case c == '~':
		return -1
	default:
		return int(c) + 256
	}
}

// verrevcmp is Debian's core version-part comparison: alternating runs of
// non-digits (compared by order) and digits (compared numerically).
func verrevcmp(a, b string) int {
	i, j := 0, 0
	for i < len(a) || j < len(b) {
		first := 0
		for (i < len(a) && !isDigit(a[i])) || (j < len(b) && !isDigit(b[j])) {
			ac, bc := 0, 0
			if i < len(a) {
				ac = order(a[i])
			}
			if j < len(b) {
				bc = order(b[j])
			}
			if ac != bc {
				if ac < bc {
					return -1
				}
				return 1
			}
			i++
			j++
		}
		for i < len(a) && a[i] == '0' {
			i++
		}
		for j < len(b) && b[j] == '0' {
			j++
		}
		for i < len(a) && isDigit(a[i]) && j < len(b) && isDigit(b[j]) {
			if first == 0 {
				first = int(a[i]) - int(b[j])
			}
			i++
			j++
		}
		if i < len(a) && isDigit(a[i]) {
			return 1
		}
		if j < len(b) && isDigit(b[j]) {
			return -1
		}
		if first != 0 {
			if first < 0 {
				return -1
			}
			return 1
		}
	}
	return 0
}
