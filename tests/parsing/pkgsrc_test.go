package parsing

import (
	"testing"

	"github.com/neur0map/glazepkg/internal/manager"
)

func TestSplitPkgsrcNameVersion(t *testing.T) {
	tests := []struct {
		input       string
		wantName    string
		wantVersion string
	}{
		{"bash-5.2.15", "bash", "5.2.15"},
		{"mozilla-rootcerts-1.0.20230420nb1", "mozilla-rootcerts", "1.0.20230420nb1"},
		{"git-2.41.0nb1", "git", "2.41.0nb1"},
		{"pkg_install-20230420", "pkg_install", "20230420"},
		{"noversion", "noversion", ""},
	}
	for _, tt := range tests {
		name, ver := manager.SplitPkgsrcNameVersion(tt.input)
		if name != tt.wantName || ver != tt.wantVersion {
			t.Errorf("SplitPkgsrcNameVersion(%q) = (%q, %q), want (%q, %q)",
				tt.input, name, ver, tt.wantName, tt.wantVersion)
		}
	}
}
