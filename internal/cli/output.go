package cli

import (
	"encoding/json"
	"io"
	"os"
	"regexp"
	"time"

	"github.com/mattn/go-isatty"
	"github.com/neur0map/glazepkg/internal/model"
)

// SchemaVersion is bumped only on non-additive JSON changes.
// Additive fields (new keys with omitempty) keep schema=1.
const SchemaVersion = 1

// envelope is the top-level shape of every --json output.
type envelope struct {
	GpkVersion string      `json:"gpk_version"`
	Schema     int         `json:"schema"`
	Data       interface{} `json:"data"`
}

// writeEnvelope serializes the envelope to w and appends a trailing newline.
// Always uses compact (non-indented) JSON.
func writeEnvelope(w io.Writer, version string, data interface{}) error {
	env := envelope{
		GpkVersion: version,
		Schema:     SchemaVersion,
		Data:       data,
	}
	enc := json.NewEncoder(w)
	return enc.Encode(env)
}

// cliPackage is the JSON DTO for a package emitted by the cli. It differs
// from model.Package only by including LatestVersion (which model.Package
// hides with json:"-" so it doesn't leak into snapshot files).
type cliPackage struct {
	Name          string       `json:"name"`
	Version       string       `json:"version"`
	Source        model.Source `json:"source"`
	Description   string       `json:"description,omitempty"`
	Size          string       `json:"size,omitempty"`
	SizeBytes     int64        `json:"size_bytes,omitempty"`
	Repository    string       `json:"repository,omitempty"`
	DependsOn     []string     `json:"depends_on,omitempty"`
	RequiredBy    []string     `json:"required_by,omitempty"`
	InstalledAt   time.Time    `json:"installed_at"`
	LatestVersion string       `json:"latest_version,omitempty"`
}

func toCLIPackage(p model.Package) cliPackage {
	return cliPackage{
		Name:          p.Name,
		Version:       p.Version,
		Source:        p.Source,
		Description:   p.Description,
		Size:          p.Size,
		SizeBytes:     p.SizeBytes,
		Repository:    p.Repository,
		DependsOn:     p.DependsOn,
		RequiredBy:    p.RequiredBy,
		InstalledAt:   p.InstalledAt,
		LatestVersion: p.LatestVersion,
	}
}

func toCLIPackages(ps []model.Package) []cliPackage {
	out := make([]cliPackage, len(ps))
	for i, p := range ps {
		out[i] = toCLIPackage(p)
	}
	return out
}

// ansiRegex matches ANSI CSI escape sequences (color codes, cursor moves).
// Used by stripANSI for non-TTY human output.
var ansiRegex = regexp.MustCompile("\x1b\\[[0-9;]*[a-zA-Z]")

func stripANSI(s string) string {
	return ansiRegex.ReplaceAllString(s, "")
}

// isStdoutTTY reports whether os.Stdout is connected to a real terminal.
// Tests use a bytes.Buffer (not a TTY), so this always returns false in tests.
func isStdoutTTY() bool {
	return isatty.IsTerminal(os.Stdout.Fd())
}
