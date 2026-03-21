package parsing

import (
	"testing"

	"github.com/neur0map/glazepkg/internal/manager"
)

func TestSanitizeDesc(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{`<p align="center"><code>npm i -g foo</code></p>`, "npm i -g foo"},
		{`<strong>Hello</strong> world`, "Hello world"},
		{"plain text no html", "plain text no html"},
		{"  lots   of   spaces  ", "lots of spaces"},
		{"", ""},
	}
	for _, tt := range tests {
		got := manager.SanitizeDesc(tt.input)
		if got != tt.want {
			t.Errorf("SanitizeDesc(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
