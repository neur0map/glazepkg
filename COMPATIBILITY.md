# Manager Compatibility

Reference for which package managers support which headless write operations (`gpk install` / `remove` / `upgrade`), and what command each one constructs. This is the source of truth for `gpk`'s cross-OS coverage claims.

> **Regenerating this doc:** the matrix is derived from the manager interfaces in `internal/manager/`. To re-run the audit and see fresh output:
> ```bash
> go test ./tests/integration/... -run TestManagerCapabilityMatrix -v
> go test ./tests/integration/... -run TestManagerCommandSamples -v
> ```

## How to read the matrix

Each column reports whether the manager implements the matching Go interface in `internal/manager/`:

| Column | Interface | What it controls |
|---|---|---|
| `Inst` | `Installer` | `gpk install <pkg>` |
| `Inst+y` | `NonInteractiveInstaller` | `gpk install <pkg> --yes` skips the manager's own prompt |
| `Up` | `Upgrader` | `gpk upgrade <pkg>` |
| `Up+y` | `NonInteractiveUpgrader` | `gpk upgrade <pkg> --yes` |
| `Rm` | `Remover` | `gpk remove <pkg>` |
| `Rm+y` | `NonInteractiveRemover` | `gpk remove <pkg> --yes` |
| `Deep` | `DeepRemover` | `gpk remove <pkg> --with-deps` |
| `Deep+y` | `NonInteractiveDeepRemover` | `gpk remove <pkg> --with-deps --yes` |

A `✓` means the interface is implemented; `─` means it isn't, and `gpk` falls back gracefully (interactive command for `+y`, error message for unsupported ops).

## TUI vs headless coverage

The TUI (`gpk` with no args) and the headless commands share the **same** manager interfaces — there's no manager that the TUI supports but headless doesn't, or vice versa.

| Operation | TUI surface | Headless surface | Interface both call |
|---|---|---|---|
| Install | Search panel (`i`) → result row → install confirmation | `gpk install <pkg>` | `manager.Installer.InstallCmd` |
| Upgrade | Detail view (`Enter`) → `u` | `gpk upgrade <pkg>` | `manager.Upgrader.UpgradeCmd` |
| Remove | Detail view (`Enter`) → `x` | `gpk remove <pkg>` | `manager.Remover.RemoveCmd` |
| Remove + deps | Remove modal → "package + orphaned deps" mode | `gpk remove <pkg> --with-deps` | `manager.DeepRemover.RemoveCmdWithDeps` |

So the `Inst` / `Up` / `Rm` / `Deep` columns in the matrix describe **both modes** at once. If a manager has `─` in the `Rm` column, neither `x` in the TUI nor `gpk remove` from the shell works for it.

**The `+y` columns are the one place TUI and headless differ.** They report whether the manager exposes a non-interactive variant (`pacman --noconfirm`, `apt -y`, etc.). The TUI doesn't use these — its modal owns confirmation, and the password field captures sudo input via `-S`. The headless `--yes` flag uses the `*Yes` interfaces when available; for managers without them, `--yes` is a safe no-op because either (a) the interactive command already includes its own non-interactive flag, or (b) the manager doesn't prompt by default. The next section unpacks this.

## Capability matrix

| Manager | Platform | Inst | Inst+y | Up | Up+y | Rm | Rm+y | Deep | Deep+y |
|---|---|:-:|:-:|:-:|:-:|:-:|:-:|:-:|:-:|
| **pacman** | Arch Linux | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ |
| **aur** | Arch Linux | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ─ | ─ |
| **apt** | Debian / Ubuntu | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ |
| **dnf** | Fedora / RHEL | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ |
| **chocolatey** | Windows | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ─ | ─ |
| **brew** | macOS / Linux | ✓ | ─ | ✓ | ─ | ✓ | ─ | ─ | ─ |
| **brew-cask** | macOS | ✓ | ─ | ✓ | ─ | ✓ | ─ | ─ | ─ |
| **snap** | Linux | ✓ | ─ | ✓ | ─ | ✓ | ─ | ─ | ─ |
| **flatpak** | Linux | ✓ | ─ | ✓ | ─ | ✓ | ─ | ─ | ─ |
| **apk** | Alpine | ✓ | ─ | ✓ | ─ | ✓ | ─ | ─ | ─ |
| **xbps** | Void Linux | ✓ | ─ | ✓ | ─ | ✓ | ─ | ✓ | ─ |
| **portage** | Gentoo | ✓ | ─ | ✓ | ─ | ✓ | ─ | ─ | ─ |
| **guix** | GNU Guix | ✓ | ─ | ✓ | ─ | ✓ | ─ | ─ | ─ |
| **nix** | NixOS / cross | ✓ | ─ | ✓ | ─ | ✓ | ─ | ─ | ─ |
| **macports** | macOS | ✓ | ─ | ✓ | ─ | ✓ | ─ | ─ | ─ |
| **pkgsrc** | NetBSD / cross | ─ | ─ | ─ | ─ | ✓ | ─ | ─ | ─ |
| **pkg** | FreeBSD | ✓ | ─ | ✓ | ─ | ✓ | ─ | ─ | ─ |
| **mas** | macOS | ✓ | ─ | ✓ | ─ | ─ | ─ | ─ | ─ |
| **winget** | Windows | ✓ | ─ | ✓ | ─ | ✓ | ─ | ─ | ─ |
| **scoop** | Windows | ✓ | ─ | ✓ | ─ | ✓ | ─ | ─ | ─ |
| **windows-updates** | Windows | ─ | ─ | ─ | ─ | ─ | ─ | ─ | ─ |
| **powershell** | Cross | ✓ | ─ | ✓ | ─ | ✓ | ─ | ─ | ─ |
| **nuget** | Cross | ✓ | ─ | ✓ | ─ | ✓ | ─ | ─ | ─ |
| **pip** | Cross (Python) | ✓ | ─ | ✓ | ─ | ✓ | ─ | ─ | ─ |
| **pipx** | Cross (Python) | ✓ | ─ | ✓ | ─ | ✓ | ─ | ─ | ─ |
| **uv** | Cross (Python) | ✓ | ─ | ✓ | ─ | ✓ | ─ | ─ | ─ |
| **conda** | Cross | ✓ | ─ | ✓ | ─ | ✓ | ─ | ─ | ─ |
| **npm** | Cross (Node) | ✓ | ─ | ✓ | ─ | ✓ | ─ | ─ | ─ |
| **pnpm** | Cross (Node) | ✓ | ─ | ✓ | ─ | ✓ | ─ | ─ | ─ |
| **bun** | Cross (Node) | ✓ | ─ | ✓ | ─ | ✓ | ─ | ─ | ─ |
| **cargo** | Cross (Rust) | ✓ | ─ | ✓ | ─ | ✓ | ─ | ─ | ─ |
| **go** | Cross (Go) | ✓ | ─ | ✓ | ─ | ✓ | ─ | ─ | ─ |
| **gem** | Cross (Ruby) | ✓ | ─ | ✓ | ─ | ✓ | ─ | ─ | ─ |
| **opam** | Cross (OCaml) | ✓ | ─ | ✓ | ─ | ✓ | ─ | ─ | ─ |
| **luarocks** | Cross (Lua) | ✓ | ─ | ✓ | ─ | ✓ | ─ | ─ | ─ |
| **composer** | Cross (PHP) | ✓ | ─ | ✓ | ─ | ✓ | ─ | ─ | ─ |
| **maven** | Cross (Java) | ─ | ─ | ✓ | ─ | ─ | ─ | ─ | ─ |

## How `--yes` works in practice

The Inst+y / Up+y / Rm+y columns measure whether a manager has a *dedicated* non-interactive variant. But headless `--yes` works correctly for almost every manager regardless, because either:

1. **The interactive command already includes a `-y` / `--yes` / equivalent flag.** apt, dnf, pip, flatpak, winget, etc. ship with the non-interactive flag baked into their interactive form, because those tools assume "if you're calling me programmatically, you don't want prompts." `--yes` on these is functionally a no-op — gpk's own y/N prompt is skipped, and the manager wasn't going to prompt anyway.
2. **The manager doesn't prompt to begin with.** brew, cargo, npm, go, pip uninstall (when given a package), pnpm, bun, gem, opam, composer, luarocks, mas, scoop, nuget — none of these ask "Proceed? [Y/n]" by default. `--yes` is a no-op for the manager's part.
3. **The manager prompts AND has no flag to skip.** This is the case Inst+y was added for. The five managers that genuinely needed an explicit `*Yes` variant: **pacman, aur, apt, dnf, chocolatey**. All five are wired.

So while only 5 managers show `✓` in the `+y` columns, **all 36 manager-capable tools work non-interactively under `--yes`**. The two exceptions (`pkgsrc`'s install, `mas`'s remove, `maven`'s install/remove) aren't a `--yes` problem — those operations aren't exposed by the underlying tool at all.

## Sample constructed commands

Below are the exact `*exec.Cmd` arguments `gpk` builds for a hypothetical package `DUMMY`. Generated by `TestManagerCommandSamples`; rerun the test to refresh if you change a manager's command construction.

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

AUR remove delegates to pacman because `yay -R` is just a passthrough to pacman; using pacman directly is more honest.

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

The `-y` is baked into the interactive form; `--yes` produces an identical command (the explicit `+y` variants exist for symmetry / safety against future changes).

### macOS / Linux

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

brew doesn't prompt — `--yes` is a no-op. `mas` (Mac App Store) can't uninstall from CLI; Apple restriction.

### Windows

```
=== winget ===
  install:        winget install --id DUMMY --exact --accept-source-agreements --accept-package-agreements --disable-interactivity
  upgrade:        winget upgrade --id DUMMY --exact --accept-source-agreements --accept-package-agreements --disable-interactivity
  remove:         winget uninstall --id DUMMY --exact --disable-interactivity

=== scoop ===
  install:        scoop install DUMMY
  upgrade:        scoop update DUMMY
  remove:         scoop uninstall DUMMY

=== chocolatey ===
  install:        choco install DUMMY --yes
  upgrade:        choco upgrade DUMMY --yes
  remove:         choco uninstall DUMMY --yes

=== nuget ===
  install:        nuget install DUMMY
  upgrade:        nuget update DUMMY
  remove:         (filesystem delete in ~/.nuget/packages/)

=== powershell ===
  install:        Install-Module -Name DUMMY -Force
  upgrade:        Update-Module -Name DUMMY -Force
  remove:         Uninstall-Module -Name DUMMY -Force

=== windows-updates ===
  install:        (not supported)
  upgrade:        (not supported)
  remove:         (not supported)
```

`winget` builds in all four "yes" flags already. `windows-updates` is a status reader — write operations are owned by the Settings app.

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
```

`go remove` is the only filesystem-level operation in the matrix — `go` itself has no uninstall command, so gpk runs `rm $GOPATH/bin/<name>`. Safe because the file was created by `go install` in the first place.

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

## Deliberate gaps

Four managers have intentionally-missing write capabilities. They aren't bugs in gpk:

| Manager | Missing | Why |
|---|---|---|
| `pkgsrc` | Install, Upgrade | NetBSD's pkgsrc is bootstrap-driven; there's no single-command install. Use `make install` from the source tree. |
| `mas` | Remove | Mac App Store CLI deliberately doesn't expose uninstall (Apple restriction). Use the GUI Launchpad / Finder. |
| `maven` | Install, Remove | Maven's local cache (`~/.m2/repository`) is build-output, not a user-managed package set. There's no "uninstall" concept. |
| `windows-updates` | All | This is a status-only manager — counts pending Windows updates. Triggering the install is owned by the Settings app or `UsoClient`. |

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

## Adding non-interactive support to a new manager

If a manager's interactive command prompts and the underlying tool has a flag to skip the prompt, implement the matching `NonInteractive*` interface from `internal/manager/noninteractive.go`. Example for a fictional `xyz` manager:

```go
// internal/manager/xyz.go
func (x *XYZ) InstallCmdYes(name string) *exec.Cmd {
    return exec.Command("xyz", "install", "--no-confirm", name)
}
func (x *XYZ) UpgradeCmdYes(name string) *exec.Cmd { ... }
func (x *XYZ) RemoveCmdYes(name string) *exec.Cmd { ... }
```

`gpk` automatically picks up the `*Yes` variant when the user passes `--yes`. If the tool already runs non-interactively without a flag (most cross-platform language managers), skip this — `gpk`'s `--yes` is already a no-op for those.
