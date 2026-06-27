# Manager Compatibility

What `gpk` can do per package manager, and the OS each manager runs on. This is the source-of-truth reference for both the TUI and headless CLI.

> **Regenerating the matrix:** the table is derived from the Go interfaces in `internal/manager/`. To re-run the audit:
> ```bash
> go test ./tests/integration/... -run TestManagerCapabilityMatrix -v
> go test ./tests/integration/... -run TestManagerCommandSamples -v
> ```

## What gpk can do

Every operation `gpk` exposes maps to one Go interface in `internal/manager/`. A manager supports the operation iff it implements that interface. **TUI and headless use the same interfaces** — anything you can do in the TUI you can do from `gpk <subcommand>`, and vice versa.

| Operation | TUI | Headless | Interface | Coverage |
|---|---|---|---|---|
| List installed packages | the package table | `gpk list` | `Manager.Scan` | **43 / 43** |
| Package details | `Enter` → detail view | `gpk info <pkg>` | `Manager.Scan` (base) + `Describer` (description) | base 43 / 43; rich desc 34 / 43 |
| Dependency tree | `d` in detail | _(detail-only)_ | `DependencyLister` | 23 / 43 |
| Which manager has it | source pill in row | `gpk source-of <pkg>` | `Manager.Scan` | 43 / 43 |
| Update detection (`↑`) | indicator in row | `gpk outdated` / `--count` / `--exit-code` | `UpdateChecker` | 29 / 43 |
| Install search (catalog) | `i` → search overlay | `gpk install <pkg>` (resolution step) | `Searcher` | 28 / 43 |
| Install | search → install confirm | `gpk install <pkg>` | `Installer` | 35 / 43 |
| Install non-interactive | _(uses modal)_ | `gpk install --yes` | `NonInteractiveInstaller` *(see note)* | 5 / 43 explicit; the rest are no-ops |
| Upgrade single | `u` in detail | `gpk upgrade <pkg>` | `Upgrader` | 36 / 43 |
| Upgrade non-interactive | _(uses modal)_ | `gpk upgrade --yes` | `NonInteractiveUpgrader` *(see note)* | 5 / 43 explicit |
| Remove | `x` in detail | `gpk remove <pkg>` | `Remover` | 36 / 43 |
| Remove non-interactive | _(uses modal)_ | `gpk remove --yes` | `NonInteractiveRemover` *(see note)* | 5 / 43 explicit |
| Remove + deps | remove modal "deep" option | `gpk remove --with-deps` | `DeepRemover` | 4 / 43 (pacman, apt, dnf, xbps) |
| Snapshots / diff / export | `s` / `d` / `e` | _(TUI-only in Phase 1)_ | filesystem (no manager interface) | universal |
| Fuzzy filter | `/` over the table | _(implicit via `gpk list` + pipe)_ | client-side | universal |
| Theme picker | `t` | _(N/A)_ | reads `~/.config/glazepkg/themes/*.toml` | universal |
| User notes (custom descriptions) | `e` in detail | _(TUI-only in Phase 1)_ | persists to `~/.local/share/glazepkg/notes.json` | universal |

**Note on `--yes`:** the headless `--yes` flag works correctly for **all 38 manager-capable tools** even though only 5 implement the explicit `NonInteractive*` interfaces. The other 33 either don't prompt by default (brew, cargo, npm, etc.) or already include their non-interactive flag in the regular command (apt has `-y` baked in, winget has `--disable-interactivity`, etc.). The 5 with explicit `*Yes` variants are the ones whose standard command does prompt: **pacman, aur, apt, dnf, chocolatey**.

## Capability matrix

How to read this table: each `✓` means the manager implements the matching Go interface. `─` means it doesn't, and `gpk` either falls back gracefully (interactive command for `+y`, less rich output for read interfaces) or returns "not supported" with a clear error.

Column legend:
- **avail** — does the binary respond on this system right now? (always `no` on systems where the manager doesn't exist)
- **Scan** — `Manager.Scan` (list installed packages). Universal.
- **Desc** — `Describer.Describe` (per-package descriptions for `gpk info`, list-row hints)
- **Deps** — `DependencyLister.ListDependencies` (`gpk info`'s dependency tree, TUI's `d` overlay)
- **Updates** — `UpdateChecker.CheckUpdates` (powers `gpk outdated`, the `↑` indicator)
- **Search** — `Searcher.Search` (lets `gpk install <pkg>` resolve which manager has the package; also the TUI's `i` overlay)
- **Inst / Up / Rm / Deep** — `Installer.InstallCmd` / `Upgrader.UpgradeCmd` / `Remover.RemoveCmd` / `DeepRemover.RemoveCmdWithDeps`
- **+y** variants — `NonInteractive*` siblings used by `gpk --yes`

| Manager | Platform | avail | Scan | Desc | Deps | Updates | Search | Inst | Inst+y | Up | Up+y | Rm | Rm+y | Deep | Deep+y |
|---|---|:-:|:-:|:-:|:-:|:-:|:-:|:-:|:-:|:-:|:-:|:-:|:-:|:-:|:-:|
| **pacman** | Arch | YES | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ |
| **aur** | Arch (via yay) | YES | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ─ | ─ |
| **apt** | Debian / Ubuntu | no | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ |
| **dnf** | Fedora / RHEL | no | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ |
| **brew** | macOS / Linux | no | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ─ | ✓ | ─ | ✓ | ─ | ─ | ─ |
| **brew-cask** | macOS | no | ✓ | ✓ | ─ | ✓ | ✓ | ✓ | ─ | ✓ | ─ | ✓ | ─ | ─ | ─ |
| **snap** | Linux | no | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ─ | ✓ | ─ | ✓ | ─ | ─ | ─ |
| **flatpak** | Linux | no | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ─ | ✓ | ─ | ✓ | ─ | ─ | ─ |
| **apk** | Alpine | no | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ─ | ✓ | ─ | ✓ | ─ | ─ | ─ |
| **xbps** | Void Linux | no | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ─ | ✓ | ─ | ✓ | ─ | ✓ | ─ |
| **portage** | Gentoo | no | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ─ | ✓ | ─ | ✓ | ─ | ─ | ─ |
| **guix** | GNU Guix | no | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ─ | ✓ | ─ | ✓ | ─ | ─ | ─ |
| **nix** | NixOS / cross | no | ✓ | ✓ | ✓ | ─ | ✓ | ✓ | ─ | ✓ | ─ | ✓ | ─ | ─ | ─ |
| **macports** | macOS | no | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ─ | ✓ | ─ | ✓ | ─ | ─ | ─ |
| **pkgsrc** | NetBSD | no | ✓ | ✓ | ✓ | ─ | ─ | ─ | ─ | ─ | ─ | ✓ | ─ | ─ | ─ |
| **pkg** | FreeBSD | no | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ─ | ✓ | ─ | ✓ | ─ | ─ | ─ |
| **mas** | macOS | no | ✓ | ✓ | ─ | ✓ | ✓ | ✓ | ─ | ✓ | ─ | ─ | ─ | ─ | ─ |
| **winget** | Windows | no | ✓ | ✓ | ─ | ✓ | ✓ | ✓ | ─ | ✓ | ─ | ✓ | ─ | ─ | ─ |
| **chocolatey** | Windows | no | ✓ | ✓ | ─ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ─ | ─ |
| **scoop** | Windows | no | ✓ | ✓ | ─ | ✓ | ✓ | ✓ | ─ | ✓ | ─ | ✓ | ─ | ─ | ─ |
| **nuget** | Windows / cross | no | ✓ | ─ | ─ | ─ | ─ | ✓ | ─ | ✓ | ─ | ✓ | ─ | ─ | ─ |
| **powershell** | Windows / cross | no | ✓ | ✓ | ─ | ─ | ✓ | ✓ | ─ | ✓ | ─ | ✓ | ─ | ─ | ─ |
| **windows-updates** | Windows | no | ✓ | ─ | ─ | ─ | ─ | ─ | ─ | ─ | ─ | ─ | ─ | ─ | ─ |
| **pip** | Cross (Python) | no | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ─ | ✓ | ─ | ✓ | ─ | ─ | ─ |
| **pipx** | Cross (Python) | no | ✓ | ✓ | ─ | ─ | ─ | ✓ | ─ | ✓ | ─ | ✓ | ─ | ─ | ─ |
| **uv** | Cross (Python) | YES | ✓ | ✓ | ─ | ✓ | ─ | ✓ | ─ | ✓ | ─ | ✓ | ─ | ─ | ─ |
| **conda** | Cross | no | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ─ | ✓ | ─ | ✓ | ─ | ─ | ─ |
| **cargo** | Cross (Rust) | YES | ✓ | ✓ | ─ | ─ | ✓ | ✓ | ─ | ✓ | ─ | ✓ | ─ | ─ | ─ |
| **go** | Cross (Go) | no | ✓ | ✓ | ─ | ─ | ─ | ✓ | ─ | ✓ | ─ | ✓ | ─ | ─ | ─ |
| **npm** | Cross (Node) | YES | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ─ | ✓ | ─ | ✓ | ─ | ─ | ─ |
| **pnpm** | Cross (Node) | no | ✓ | ✓ | ✓ | ✓ | ─ | ✓ | ─ | ✓ | ─ | ✓ | ─ | ─ | ─ |
| **bun** | Cross (Node) | no | ✓ | ✓ | ─ | ─ | ─ | ✓ | ─ | ✓ | ─ | ✓ | ─ | ─ | ─ |
| **gem** | Cross (Ruby) | YES | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ─ | ✓ | ─ | ✓ | ─ | ─ | ─ |
| **opam** | Cross (OCaml) | no | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ─ | ✓ | ─ | ✓ | ─ | ─ | ─ |
| **luarocks** | Cross (Lua) | YES | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ─ | ✓ | ─ | ✓ | ─ | ─ | ─ |
| **composer** | Cross (PHP) | no | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ─ | ✓ | ─ | ✓ | ─ | ─ | ─ |
| **maven** | Cross (Java) | no | ✓ | ─ | ─ | ✓ | ─ | ─ | ─ | ✓ | ─ | ─ | ─ | ─ | ─ |
| **am** | Linux | no | ✓ | ─ | ─ | ─ | ─ | ✓ | ─ | ✓ | ─ | ✓ | ─ | ─ | ─ |
| **gvm** | Cross (Go) | no | ✓ | ─ | ─ | ─ | ─ | ─ | ─ | ─ | ─ | ─ | ─ | ─ | ─ |
| **mise** | Cross | no | ✓ | ─ | ─ | ✓ | ─ | ✓ | ─ | ✓ | ─ | ✓ | ─ | ─ | ─ |
| **quicklisp** | Cross (Lisp) | no | ✓ | ─ | ─ | ─ | ─ | ─ | ─ | ─ | ─ | ─ | ─ | ─ | ─ |
| **softwareupdate** | macOS | no | ✓ | ─ | ─ | ─ | ─ | ─ | ─ | ─ | ─ | ─ | ─ | ─ | ─ |
| **local** | Linux | YES | ✓ | ─ | ─ | ─ | ─ | ─ | ─ | ─ | ─ | ✓ | ─ | ─ | ─ |

## Read-side notes

**Scan is universal.** Every manager `gpk` knows about lists installed packages, so `gpk list` and the TUI always work. Even `windows-updates` and `maven` — which have no install/remove semantics — implement `Scan` (windows-updates returns pending Windows updates; maven returns artifacts cached in `~/.m2/repository`).

**`Desc` (`Describer`)** — when missing, `gpk info` and the TUI's detail view still show name/version/source/size, just without a description string. Missing: `nuget`, `windows-updates`, `maven`.

**`Deps` (`DependencyLister`)** — when missing, `gpk info` and the TUI's `d` overlay show "(no dependency info)". This is most useful for system managers (pacman, apt, dnf, brew); cross-platform language managers like `cargo`, `go`, `bun`, `uv`, etc. often don't expose deps in a useful single-package way.

**`Updates` (`UpdateChecker`)** — when missing, `gpk outdated` skips that manager silently, and the TUI's `↑` indicator never lights up for its packages. Notably missing: `pipx`, `cargo`, `go`, `bun`, `nix`, `nuget`, `powershell`, `pkgsrc`. Reason varies — most language toolchains don't ship a "what's outdated" subcommand.

**`Search` (`Searcher`)** — needed for `gpk install <pkg>` to resolve which manager has the package when `--manager` isn't passed. When missing, you must pass `--manager` explicitly (`gpk install foo --manager npm`). Missing managers default to "requires `--manager`": `pipx`, `go`, `bun`, `pnpm`, `nix`, `pkgsrc`, `nuget`, `powershell`, `windows-updates`, `maven`, `uv`.

## Write-side notes

**`Inst` / `Up` / `Rm` are nearly universal.** Of the 37 managers, the only ones missing write operations are:
- `pkgsrc`: install/upgrade not exposed (NetBSD's pkgsrc is bootstrap-driven from source). Remove via `pkg_delete` works.
- `mas`: remove not exposed (Apple's Mac App Store CLI doesn't allow uninstall).
- `maven`: only upgrade is meaningful (`mvn dependency:resolve -U`); Maven's local cache isn't a user-managed package set.
- `windows-updates`: pure status reader. Triggering installs is owned by the Settings app.

**`Deep` (`DeepRemover`) is rare.** Only 4 managers: pacman, apt, dnf, xbps. These tools have a dedicated "remove this package AND its orphaned dependencies" command (`pacman -Rns`, `apt-get autoremove`, `dnf autoremove`, `xbps-remove -R`). Others can be approximated by `gpk remove pkg && gpk autoremove`, but autoremove isn't exposed as a separate gpk command yet.

**Non-interactive (`+y`) coverage is intentional.** Only the 5 managers whose default command prompts the user for confirmation have `*Yes` variants: pacman, aur, apt, dnf, chocolatey. Every other manager either doesn't prompt at all or bakes the non-interactive flag into its standard command — so `--yes` from gpk gives the user a noop on those (the underlying call was already silent) without needing a separate code path.

## TUI vs headless

Both modes call **the same** Go interfaces. There is no manager that works in the TUI but not headless, or vice versa. The only differences are surface:

- **Confirmation**: the TUI shows a modal with a password field (uses sudo `-S` to pipe). Headless prompts on the terminal and uses sudo's normal `tty` prompt; `--yes` skips entirely.
- **Batch operations**: the TUI's `m` multi-select with `Space` lets you select packages across managers and run a single command per privilege group. Headless takes multiple package names per command (`gpk install a b c`) but doesn't yet have a `gpk upgrade --all`.
- **Snapshots / diff / export**: Phase 1 TUI-only. A future phase may add headless equivalents.

## Sample constructed commands

The exact `*exec.Cmd` arguments `gpk` builds for a hypothetical package `DUMMY`. Generated by `TestManagerCommandSamples`; rerun the test to refresh.

### Arch Linux

```
=== pacman ===
  install:        sudo pacman -S DUMMY
  install --yes:  sudo pacman -S --noconfirm DUMMY
  upgrade:        sudo pacman -S DUMMY
  upgrade --yes:  sudo pacman -S --noconfirm DUMMY
  remove:         sudo pacman -R DUMMY
  remove --yes:   sudo pacman -R --noconfirm DUMMY
  remove deps:    sudo pacman -Rns DUMMY
  remove deps -y: sudo pacman -Rns --noconfirm DUMMY

=== aur ===
  install:        yay -S DUMMY
  install --yes:  yay -S --noconfirm DUMMY
  upgrade:        yay -S DUMMY
  upgrade --yes:  yay -S --noconfirm DUMMY
  remove:         sudo pacman -R DUMMY
  remove --yes:   sudo pacman -R --noconfirm DUMMY
```

AUR remove delegates to pacman because `yay -R` is a passthrough; using pacman directly is more honest.

### Debian / Ubuntu / Fedora

```
=== apt ===
  install:        sudo apt-get install -y DUMMY
  upgrade:        sudo apt-get install --only-upgrade -y DUMMY
  remove:         sudo apt-get remove -y DUMMY
  remove deps:    sudo apt-get autoremove -y DUMMY

=== dnf ===
  install:        sudo dnf install -y DUMMY
  upgrade:        sudo dnf upgrade -y DUMMY
  remove:         sudo dnf remove -y DUMMY
  remove deps:    sudo dnf autoremove -y DUMMY
```

`-y` is baked into the interactive form on these — `--yes` produces an identical command.

### macOS

```
=== brew ===
  install:        brew install DUMMY
  upgrade:        brew upgrade DUMMY
  remove:         brew uninstall DUMMY

=== brew-cask ===
  install:        brew install --cask DUMMY
  upgrade:        brew upgrade --cask DUMMY
  remove:         brew uninstall --cask DUMMY

=== mas ===
  install:        mas install DUMMY
  upgrade:        mas upgrade
  remove:         (not supported)

=== macports ===
  install:        sudo port install DUMMY
  upgrade:        sudo port upgrade DUMMY
  remove:         sudo port uninstall DUMMY
```

### Windows

```
=== winget ===
  install:        winget install --id DUMMY --exact --accept-source-agreements --accept-package-agreements --disable-interactivity
  upgrade:        winget upgrade --id DUMMY --exact --accept-source-agreements --accept-package-agreements --disable-interactivity
  remove:         winget uninstall --id DUMMY --exact --disable-interactivity

=== chocolatey ===
  install:        choco install DUMMY --yes
  upgrade:        choco upgrade DUMMY --yes
  remove:         choco uninstall DUMMY --yes

=== scoop ===
  install:        scoop install DUMMY
  upgrade:        scoop update DUMMY
  remove:         scoop uninstall DUMMY

=== powershell ===
  install:        Install-Module -Name DUMMY -Force
  upgrade:        Update-Module -Name DUMMY -Force
  remove:         Uninstall-Module -Name DUMMY -Force

=== windows-updates ===
  install:        (not supported)
  upgrade:        (not supported)
  remove:         (not supported)
```

### Linux distro-specifics

```
=== snap ===
  install:        sudo snap install DUMMY
  upgrade:        sudo snap refresh DUMMY
  remove:         sudo snap remove DUMMY

=== flatpak ===
  install:        flatpak install -y DUMMY
  upgrade:        flatpak update -y DUMMY
  remove:         flatpak uninstall -y DUMMY

=== apk ===
  install:        sudo apk add DUMMY
  upgrade:        sudo apk add --upgrade DUMMY
  remove:         sudo apk del DUMMY

=== xbps ===
  install:        sudo xbps-install -y DUMMY
  upgrade:        sudo xbps-install -y DUMMY
  remove:         sudo xbps-remove -y DUMMY
  remove deps:    sudo xbps-remove -Ry DUMMY

=== portage ===
  install:        sudo emerge DUMMY
  upgrade:        sudo emerge --update DUMMY
  remove:         sudo emerge --unmerge DUMMY

=== guix ===
  install:        guix install DUMMY
  upgrade:        guix upgrade DUMMY
  remove:         guix remove DUMMY

=== nix ===
  install:        nix profile install nixpkgs#DUMMY
  upgrade:        nix profile upgrade DUMMY
  remove:         nix profile remove DUMMY

=== pkg ===
  install:        sudo pkg install -y DUMMY
  upgrade:        sudo pkg upgrade -y DUMMY
  remove:         sudo pkg delete -y DUMMY

=== pkgsrc ===
  install:        (not supported; pkgsrc is bootstrap-driven)
  upgrade:        (not supported)
  remove:         sudo pkg_delete DUMMY
```

### Cross-platform language tools

```
=== pip ===
  install:        pip install DUMMY
  upgrade:        pip install --upgrade DUMMY
  remove:         pip uninstall -y DUMMY

=== pipx ===
  install:        pipx install DUMMY
  upgrade:        pipx upgrade DUMMY
  remove:         pipx uninstall DUMMY

=== uv ===
  install:        uv tool install DUMMY
  upgrade:        uv tool upgrade DUMMY
  remove:         uv tool uninstall DUMMY

=== conda ===
  install:        conda install -y DUMMY
  upgrade:        conda update -y DUMMY
  remove:         conda remove -y DUMMY

=== cargo ===
  install:        cargo install DUMMY
  upgrade:        cargo install DUMMY
  remove:         cargo uninstall DUMMY

=== go ===
  install:        go install DUMMY@latest
  upgrade:        go install DUMMY@latest
  remove:         rm ~/go/bin/DUMMY

=== npm ===
  install:        npm install -g DUMMY
  upgrade:        npm install -g DUMMY@latest
  remove:         npm uninstall -g DUMMY

=== pnpm ===
  install:        pnpm add -g DUMMY
  upgrade:        pnpm update -g DUMMY
  remove:         pnpm remove -g DUMMY

=== bun ===
  install:        bun add -g DUMMY
  upgrade:        bun update -g DUMMY
  remove:         bun remove -g DUMMY

=== gem ===
  install:        gem install DUMMY
  upgrade:        gem update DUMMY
  remove:         gem uninstall DUMMY

=== opam ===
  install:        opam install -y DUMMY
  upgrade:        opam upgrade -y DUMMY
  remove:         opam remove -y DUMMY

=== luarocks ===
  install:        luarocks install DUMMY
  upgrade:        luarocks install DUMMY
  remove:         luarocks remove DUMMY

=== composer ===
  install:        composer global require DUMMY
  upgrade:        composer global update DUMMY
  remove:         composer global remove DUMMY

=== maven ===
  install:        (not supported)
  upgrade:        mvn dependency:resolve -U
  remove:         (not supported)

=== nuget ===
  install:        nuget install DUMMY
  upgrade:        nuget update DUMMY
  remove:         (filesystem delete in ~/.nuget/packages/)
```

`go remove` is the only filesystem-level operation in the matrix — `go` itself has no uninstall command, so gpk runs `rm $GOPATH/bin/<name>`. Safe because the file was created by `go install` in the first place.

## Deliberate gaps

Five spots where a manager intentionally doesn't implement an interface. These aren't gpk bugs:

| Manager | Missing | Why |
|---|---|---|
| `pkgsrc` | Install, Upgrade, Search, Updates | NetBSD's pkgsrc is bootstrap-driven; there's no single-command install. Use `make install` from the source tree. |
| `mas` | Remove, Deps | Mac App Store CLI deliberately doesn't expose uninstall (Apple restriction). Apps that come from `mas` don't have a meaningful dependency graph. |
| `maven` | Install, Remove, Search, Deps, Desc | Maven's local cache (`~/.m2/repository`) is build-output, not a user-managed package set. There's no install/remove concept; the only write op is "refresh the cache" via `mvn dependency:resolve -U`. |
| `windows-updates` | Almost everything | Status-only manager — counts pending Windows updates. Triggering the install is owned by the Settings app or `UsoClient`. |
| `nuget` | Desc, Deps, Updates, Search | NuGet's surface here is the global package cache. Per-package introspection beyond install/upgrade/remove is package-source-dependent and not surfaced by the `nuget` CLI in a portable way. |

## Verifying locally

Reproduce the matrix and per-manager commands on your machine:

```bash
go test ./tests/integration/... -run TestManagerCapabilityMatrix -v
go test ./tests/integration/... -run TestManagerCommandSamples -v
```

For a single manager's commands:

```bash
go test ./tests/integration/... -run 'TestManagerCommandSamples/pacman$' -v
```

## Adding support to a new manager

A new manager is one Go file in `internal/manager/`. The minimum is `Manager.Scan`; everything else is opt-in:

```go
package manager

import "os/exec"

type Mytool struct{}

func (m *Mytool) Name() model.Source { return model.SourceMytool }
func (m *Mytool) Available() bool    { return commandExists("mytool") }
func (m *Mytool) Scan() ([]model.Package, error) { /* parse `mytool list` */ }

// Opt-in capabilities — add the methods you want gpk to support:
func (m *Mytool) Describe(pkgs []model.Package) map[string]string { ... }
func (m *Mytool) ListDependencies(pkgs []model.Package) map[string][]string { ... }
func (m *Mytool) CheckUpdates(pkgs []model.Package) map[string]string { ... }
func (m *Mytool) Search(query string) ([]model.Package, error) { ... }
func (m *Mytool) InstallCmd(name string) *exec.Cmd { ... }
func (m *Mytool) UpgradeCmd(name string) *exec.Cmd { ... }
func (m *Mytool) RemoveCmd(name string) *exec.Cmd { ... }

// And if the tool prompts and has a flag to skip:
func (m *Mytool) InstallCmdYes(name string) *exec.Cmd { ... }
func (m *Mytool) UpgradeCmdYes(name string) *exec.Cmd { ... }
func (m *Mytool) RemoveCmdYes(name string) *exec.Cmd { ... }
```

Then register: add `&Mytool{}` to `manager.All()` in `manager.go`. The audit test will pick it up automatically on next run.

Full guide in [CONTRIBUTING.md](CONTRIBUTING.md).
