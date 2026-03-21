package parsing

import (
	"testing"

	"github.com/neur0map/glazepkg/internal/manager"
)

func TestParseSizeString(t *testing.T) {
	tests := []struct {
		input string
		want  int64
	}{
		{"1.5 MiB", 1572864},
		{"234 KiB", 239616},
		{"2.0 GiB", 2147483648},
		{"512 B", 512},
		{"100 MB", 104857600},
		{"1 KB", 1024},
		{"", 0},
		{"garbage", 0},
	}
	for _, tt := range tests {
		got := manager.ParseSizeString(tt.input)
		if got != tt.want {
			t.Errorf("ParseSizeString(%q) = %d, want %d", tt.input, got, tt.want)
		}
	}
}

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		input int64
		want  string
	}{
		{0, "0 B"},
		{512, "512 B"},
		{1024, "1 KiB"},
		{1048576, "1.0 MiB"},
		{1073741824, "1.0 GiB"},
	}
	for _, tt := range tests {
		got := manager.FormatBytes(tt.input)
		if got != tt.want {
			t.Errorf("FormatBytes(%d) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
