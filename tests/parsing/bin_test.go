package parsing

import (
	"testing"

	"github.com/neur0map/glazepkg/internal/manager"
	"github.com/neur0map/glazepkg/internal/model"
)

// Real config.json written by bin v0.29.1 after installing jq and yq from
// GitHub into a throwaway download dir.
const binConfigJSON = `{
    "default_path": "/tmp/gpk-bin-test/binaries",
    "bins": {
        "/tmp/gpk-bin-test/binaries/jq": {
            "path": "/tmp/gpk-bin-test/binaries/jq",
            "remote_name": "jq-linux-amd64",
            "version": "jq-1.8.2",
            "hash": "b1c22172dd303f3be49e935aa56aa48a8b7a46e0bc838b4997d3bb451495870f",
            "url": "https://github.com/jqlang/jq",
            "provider": "github",
            "package_path": "",
            "selected_asset": "jq-linux-amd64",
            "pinned": false
        },
        "/tmp/gpk-bin-test/binaries/yq": {
            "path": "/tmp/gpk-bin-test/binaries/yq",
            "remote_name": "yq_linux_amd64",
            "version": "v4.53.3",
            "hash": "fa52a4e758c63d38299163fbdd1edfb4c4963247918bf9c1c5d31d84789eded4",
            "url": "https://github.com/mikefarah/yq",
            "provider": "github",
            "package_path": "",
            "selected_asset": "yq_linux_amd64",
            "pinned": false
        }
    }
}`

func TestBinConfigParsing(t *testing.T) {
	pkgs, err := manager.ParseBinConfig([]byte(binConfigJSON))
	if err != nil {
		t.Fatalf("ParseBinConfig error: %v", err)
	}
	if len(pkgs) != 2 {
		t.Fatalf("expected 2 packages, got %d", len(pkgs))
	}

	// Sorted by name: jq then yq.
	if pkgs[0].Name != "jq" || pkgs[0].Version != "jq-1.8.2" {
		t.Errorf("pkg 0: got %q %q", pkgs[0].Name, pkgs[0].Version)
	}
	if pkgs[0].Source != model.SourceBin {
		t.Errorf("pkg 0 source = %q, want bin", pkgs[0].Source)
	}
	if pkgs[0].Location != "/tmp/gpk-bin-test/binaries/jq" {
		t.Errorf("pkg 0 location = %q", pkgs[0].Location)
	}
	if pkgs[0].Repository != "https://github.com/jqlang/jq" {
		t.Errorf("pkg 0 repository = %q", pkgs[0].Repository)
	}

	if pkgs[1].Name != "yq" || pkgs[1].Version != "v4.53.3" {
		t.Errorf("pkg 1: got %q %q", pkgs[1].Name, pkgs[1].Version)
	}
	if pkgs[1].Location != "/tmp/gpk-bin-test/binaries/yq" {
		t.Errorf("pkg 1 location = %q", pkgs[1].Location)
	}
}

func TestBinOutdatedParsing(t *testing.T) {
	// Real `bin update --dry-run` stderr (bin v0.29.1) with jq's recorded
	// version rolled back to force an update. The bullet (\u2022) and the
	// failure glyph (\u2a2f) are wrapped in ANSI colour escapes.
	out := "\x1b[1;94m  \u2022 \x1b[m /tmp/gpk-bin-test/binaries/jq jq-1.6 -> jq-1.8.2 (https://github.com/jqlang/jq/releases/tag/jq-1.8.2)\n" +
		"\x1b[1;91m  \u2a2f \x1b[m command failed                                   \x1b[1;91merror\x1b[m=Updates found, exit (dry-run mode).\n"

	updates := manager.ParseBinOutdated([]byte(out))
	if len(updates) != 1 {
		t.Fatalf("expected 1 update, got %d: %v", len(updates), updates)
	}
	if updates["jq"] != "jq-1.8.2" {
		t.Errorf("jq latest = %q, want jq-1.8.2", updates["jq"])
	}
}
