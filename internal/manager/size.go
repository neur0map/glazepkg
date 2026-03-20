package manager

import (
	"fmt"
	"strconv"
	"strings"
)

// FormatBytes returns a human-readable size string.
func FormatBytes(b int64) string {
	switch {
	case b >= 1<<30:
		return fmt.Sprintf("%.1f GiB", float64(b)/float64(1<<30))
	case b >= 1<<20:
		return fmt.Sprintf("%.1f MiB", float64(b)/float64(1<<20))
	case b >= 1<<10:
		return fmt.Sprintf("%.0f KiB", float64(b)/float64(1<<10))
	default:
		return fmt.Sprintf("%d B", b)
	}
}

// ParseSizeString converts a human-readable size like "1.5 MiB" or "234 KiB"
// into bytes. Returns 0 if parsing fails.
func ParseSizeString(s string) int64 {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}

	parts := strings.Fields(s)
	if len(parts) < 2 {
		return 0
	}
	val, err := strconv.ParseFloat(parts[0], 64)
	if err != nil {
		return 0
	}

	unit := strings.ToLower(parts[1])
	switch unit {
	case "b", "bytes":
		return int64(val)
	case "kib", "kb", "k":
		return int64(val * float64(1<<10))
	case "mib", "mb", "m":
		return int64(val * float64(1<<20))
	case "gib", "gb", "g":
		return int64(val * float64(1<<30))
	case "tib", "tb", "t":
		return int64(val * float64(1<<40))
	default:
		return 0
	}
}
