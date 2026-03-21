# Contributing to GlazePKG

Thanks for wanting to help! Here's how to get started.

## Adding a new package manager

Each manager is a single Go file in `internal/manager/`. Look at any existing one (e.g., `snap.go` or `gem.go`) for the pattern.

### 1. Create the manager file

Create `internal/manager/yourpkg.go`:

```go
package manager

import (
	"bufio"
	"os/exec"
	"strings"
	"time"

	"github.com/neur0map/glazepkg/internal/model"
)

type YourPkg struct{}

func (y *YourPkg) Name() model.Source { return model.SourceYourPkg }

func (y *YourPkg) Available() bool { return commandExists("yourpkg") }

func (y *YourPkg) Scan() ([]model.Package, error) {
	out, err := exec.Command("yourpkg", "list").Output()
	if err != nil {
		return nil, err
	}

	var pkgs []model.Package
	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	for scanner.Scan() {
		line := scanner.Text()
		// parse name and version from the line
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		pkgs = append(pkgs, model.Package{
			Name:        fields[0],
			Version:     fields[1],
			Source:      model.SourceYourPkg,
			InstalledAt: time.Now(),
		})
	}
	return pkgs, nil
}
```

### 2. Optional interfaces

You can also implement any of these:

```go
// Update detection
func (y *YourPkg) CheckUpdates(pkgs []model.Package) map[string]string {
	// return map of name → latest version
}

// Package descriptions
func (y *YourPkg) Describe(pkgs []model.Package) map[string]string {
	// return map of name → description
}

// Dependency info (shown when pressing d in detail view)
func (y *YourPkg) ListDependencies(pkgs []model.Package) map[string][]string {
	// return map of name → list of dependency names
}
```

### 3. Register it

Four files need a one-line addition each:

**`internal/model/package.go`** — add a source constant:
```go
SourceYourPkg Source = "yourpkg"
```

**`internal/manager/manager.go`** — add to the `All()` slice:
```go
&YourPkg{},
```

**`internal/ui/tabs.go`** — add to the sources list:
```go
{model.SourceYourPkg, "yourpkg"},
```

**`internal/ui/theme.go`** — pick a badge color from the palette (`ColorBlue`, `ColorGreen`, `ColorRed`, `ColorCyan`, `ColorPurple`, `ColorOrange`, `ColorYellow`):
```go
model.SourceYourPkg: ColorCyan,
```

### 4. Add a parsing test

Create `tests/parsing/yourpkg_test.go` with mock CLI output so CI can verify your parser without the tool installed:

```go
package parsing

import (
	"strings"
	"testing"
)

func TestYourPkgListParsing(t *testing.T) {
	// Paste real output from `yourpkg list` here
	output := `package-a 1.0.0
package-b 2.3.1`

	type pkg struct{ name, version string }
	var pkgs []pkg
	for _, line := range strings.Split(output, "\n") {
		fields := strings.Fields(line)
		if len(fields) >= 2 {
			pkgs = append(pkgs, pkg{fields[0], fields[1]})
		}
	}

	if len(pkgs) != 2 {
		t.Fatalf("expected 2 packages, got %d", len(pkgs))
	}
	if pkgs[0].name != "package-a" || pkgs[0].version != "1.0.0" {
		t.Errorf("pkg 0: %+v", pkgs[0])
	}
}
```

## Building and testing

```bash
go build ./cmd/gpk
go vet ./...
go test ./...
```

## Pull requests

- Keep PRs focused on one thing
- Add tests for any new parsing logic
- Make sure `go vet ./...` passes
- If you're adding a package manager you can't test, note that in the PR

## AI-assisted contributions

If you used Claude, Copilot, ChatGPT, or any other coding agent to help write your code, mention it in the PR description. Just a short note like "used Claude for the initial scaffold" is fine. We want to know what was human-reviewed vs generated.

Do not include `Co-Authored-By` lines from AI tools in your commits. Keep commit authorship to actual humans.
