# Contributing to GlazePKG

Thanks for wanting to help! Here's how to get started.

## Adding a new package manager

Each manager is a single Go file in `internal/manager/`. Look at any existing one (e.g., `snap.go` or `gem.go`) for the pattern.

You need to implement:

```go
type YourManager struct{}

func (m *YourManager) Name() model.Source      { return model.SourceYourManager }
func (m *YourManager) Available() bool         { return commandExists("your-tool") }
func (m *YourManager) Scan() ([]model.Package, error) { /* ... */ }
```

If the tool supports single-package upgrades, implement the optional `manager.Upgrader` interface:

```go
func (m *YourManager) UpgradePackage(name string) error {
	return exec.Command("your-tool", "upgrade", name).Run()
}
```

If it can't upgrade individual packages, just omit this method and the UI will surface `manager.ErrUpgradeNotSupported`.

Optional interfaces:
- `manager.Upgrader` — implements `UpgradePackage(name string)` to run a single-package upgrade command
- `CheckUpdates(pkgs []model.Package) map[string]string` — update detection
- `Describe(pkgs []model.Package) map[string]string` — package descriptions
- `ListDependencies(pkgs []model.Package) map[string][]string` — dependency info


Then register it in:
1. `internal/model/package.go` — add `SourceYourManager` constant
2. `internal/manager/manager.go` — add to `All()`
3. `internal/ui/tabs.go` — add to the sources list
4. `internal/ui/theme.go` — pick a badge color

## Adding tests

Put parsing tests in `tests/parsing/` with mock CLI output. This lets CI verify your parser without the actual tool installed.

## Building and testing

```bash
go build ./cmd/gpk
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
