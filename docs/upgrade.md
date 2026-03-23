# Upgrade system

Pressing `u` in the detail view wires a manager-aware upgrade path through
the UI to the correct native command for each package manager.

## Flow

1. `handleListKey` catches the `u` key while the package list is focused and
   calls `Model.UpgradeSelectedPackage()`.
2. `UpgradeSelectedPackage` reads the current entry via `selectedPackage()`,
   looks up the responsible manager via `manager.BySource`, and verifies the
   manager is available on the current platform.
3. The manager is type-asserted to the optional `manager.Upgrader` interface;
   only managers that implement it can receive upgrade commands.
4. If the manager is privileged (apt, dnf, pacman, snap, apk, XBPS, or
   Chocolatey), `UpgradeSelectedPackage` saves an `upgradeRequest` and sets
   `confirmingUpgrade`, rendering a centred overlay that shows the exact
   command, warns about elevated privileges, and waits for `y`/`Enter` or
   `n`/`Esc`. This guard prevents accidentally running a privileged upgrade
   on a single keypress.
5. On confirmation (or immediately for non-privileged managers), the UI
   schedules `runUpgradeRequest`.
6. `runUpgradeRequest` runs inside a goroutine. **Before executing the
   command**, it calls `PrepareUpgrade` if the manager implements the optional
   `manager.PreUpgrader` interface (see below). Then it runs
   `cmd.CombinedOutput()` and emits an `upgradeResultMsg`.
7. `Update` turns the message into user feedback: a green `DONE` badge on
   success, a red `FAIL` badge with the native error text on failure.
8. Because `manager.BySource` always returns the active manager for the
   current platform, `u` works identically on macOS, Linux, and Windows.

## Interfaces

### `manager.Upgrader`

```go
type Upgrader interface {
    UpgradeCmd(name string) *exec.Cmd
}
```

Return the `*exec.Cmd` that upgrades a single package. The UI executes this
command, captures combined stdout/stderr, and reports the result. The command
must not read from stdin (password injection for `sudo -S` is handled by the
UI layer, not the manager).

### `manager.PreUpgrader`

```go
type PreUpgrader interface {
    PrepareUpgrade(name string) error
}
```

An optional interface for managers that need to run cleanup or preparation
*before* the upgrade command starts. Implementations must treat all errors as
non-fatal: if preparation is not possible, return `nil` so the upgrade command
runs and surfaces the real error to the user.

`PrepareUpgrade` is called **inside the upgrade goroutine**, after elevation
has been verified and before `cmd.CombinedOutput()`. This ensures every run
starts from a clean state without requiring manual intervention.

#### Why it exists — the Chocolatey `.chocolateyPending` bug

Chocolatey writes a lock sentinel
`C:\ProgramData\chocolatey\lib\<package>\.chocolateyPending` at the start of
every upgrade and removes it on successful completion. If any previous upgrade
was interrupted — by a crash, forced kill, power loss, or a failed run — the
file is left on disk.

On the **next** upgrade attempt, Chocolatey tries to recreate the sentinel.
Because the stale copy was created by a different process, its Windows ACL can
deny write access even to the current elevated session, producing:

```
Access to the path 'C:\ProgramData\chocolatey\lib\<pkg>\.chocolateyPending'
is denied.
```

This error blocks every subsequent upgrade of that package — including
`choco upgrade chocolatey` (self-upgrade) — until the marker is removed.

**Fix:** `Chocolatey.PrepareUpgrade` deletes the stale `.chocolateyPending`
file before each run via `os.Remove`. If the file does not exist
(`os.IsNotExist`), nothing happens. If deletion fails for any other reason,
the method returns `nil` so the upgrade command still runs and Chocolatey
surfaces the authoritative error. The file holds no package data — it is
purely a lock sentinel — so deletion is always safe.

The fix is **permanent and automatic**: it runs on every upgrade, before every
command, with no manual cleanup required after any failure.

## Confirmation overlay for privileged managers

The overlay shows:

- Which package and manager are involved
- The exact command that will run
- An "elevated terminal required" warning (Windows) or a password prompt
  (`sudo -S`, Linux/macOS)
- `Yes` / `No` buttons navigated with arrow keys or Tab

Privileged managers: **apt**, **dnf**, **pacman**, **snap**, **apk**,
**XBPS**, **Chocolatey**.

## Windows elevation (Chocolatey, winget)

On Windows, `privilegedCmd` resolves the correct execution strategy at runtime:

| Condition | Strategy |
|-----------|----------|
| Process is already elevated (running as Administrator) | Run directly — no wrapper needed |
| [gsudo](https://github.com/gerardog/gsudo) is on `PATH` | Wrap with `gsudo --wait` so stdout/stderr flow normally |
| Neither | Tag command with `GLAZEPKG_NEEDS_ELEVATION=1`; UI fails immediately with a clear, actionable message |

To avoid seeing the elevation error:

```powershell
# Option 1 — always launch gpk from an elevated terminal
# (right-click the terminal → "Run as Administrator")

# Option 2 — install gsudo so gpk can elevate on demand
choco install gsudo
```

With gsudo installed, gpk prompts for credentials in the same terminal window
and runs the upgrade elevated — no session restart needed.

## Supporting a new manager

### Basic upgrades

Implement `manager.Upgrader`:

```go
func (m *MyManager) UpgradeCmd(name string) *exec.Cmd {
    return exec.Command("mytool", "upgrade", name, "--yes")
}
```

### With pre-upgrade preparation

If your manager needs cleanup before the upgrade (lock-file removal, cache
invalidation, etc.), also implement `manager.PreUpgrader`:

```go
func (m *MyManager) PrepareUpgrade(name string) error {
    // Remove stale lock, clean temp dir, etc.
    // Always return nil — let the command surface real errors.
    _ = os.Remove(lockPath(name))
    return nil
}
```

### Privileged managers

If the upgrade command requires root/Administrator, add your source to
`isPrivilegedSource` in `internal/ui/app.go` so the confirmation overlay is
shown before the command runs.

## Manager upgrade commands

| Manager | Command | Notes |
|---------|---------|-------|
| apt | `apt-get install --only-upgrade <n>` | privileged; `sudo -S` |
| dnf | `dnf upgrade -y <n>` | privileged; `sudo -S` |
| pacman | `pacman -S --noconfirm <n>` | privileged; `sudo -S` |
| snap | `snap refresh <n>` | privileged; `sudo -S` |
| apk | `apk add --upgrade <n>` | privileged; `sudo -S` |
| XBPS | `xbps-install -S --yes <n>` | privileged; `sudo -S` |
| brew | `brew upgrade <n>` | |
| pip | `pip install --upgrade <n>` | |
| pipx | `pipx upgrade <n>` | |
| cargo | `cargo install <n>` | |
| npm | `npm install -g <n>` | |
| pnpm | `pnpm update -g <n>` | |
| gem | `gem update <n>` | |
| flatpak | `flatpak update -y <n>` | |
| opam | `opam upgrade --yes <n>` | |
| conda | `conda update --yes <n>` | |
| luarocks | `luarocks upgrade <n>` | |
| winget | `winget upgrade --id <n> -e` | |
| chocolatey | `choco upgrade <n> --yes --no-progress` | privileged; stale `.chocolateyPending` auto-removed via `PrepareUpgrade` |
| scoop | `scoop update <n>` | |
