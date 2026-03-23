# Windows support

GlazePKG runs natively on Windows and supports winget, Chocolatey, Scoop,
NuGet, PowerShell modules, and Windows Update out of the box.

## Supported managers

| Manager | Notes |
|---------|-------|
| winget | Windows Package Manager; no elevation required |
| chocolatey | Requires elevated session or gsudo |
| scoop | Runs in user scope; no elevation required |
| nuget | Reads global package cache; no elevation required |
| powershell | PSModulePath enumeration; no elevation required |
| windows-updates | Pending system updates via WMI |

Managers that aren't installed are silently skipped.

## Upgrading packages on Windows

### Chocolatey

Chocolatey upgrades require Administrator privileges because they write to
`C:\ProgramData\chocolatey`. GlazePKG detects the elevation context at
runtime and takes the best available path:

| Situation | What happens |
|-----------|-------------|
| gpk is running in an elevated terminal | Upgrade runs directly — no wrapper |
| [gsudo](https://github.com/gerardog/gsudo) is on `PATH` | Upgrade is wrapped with `gsudo --wait`; output is captured normally |
| Neither | gpk shows an actionable error: "administrator privileges required" |

**Recommended setup:**

```powershell
choco install gsudo
```

With gsudo, you can run `gpk` from a normal terminal and it will prompt for
credentials inline when you trigger a Chocolatey upgrade. No need to keep a
dedicated elevated window open.

### Stale `.chocolateyPending` fix

Every Chocolatey upgrade creates
`C:\ProgramData\chocolatey\lib\<package>\.chocolateyPending` as an
in-progress lock and deletes it on completion. If an upgrade is interrupted
(crash, kill, power loss, or a prior access-denied failure), the file is
left behind. The next upgrade attempt fails with:

```
Access to the path '...\lib\<package>\.chocolateyPending' is denied.
```

**GlazePKG fixes this automatically.** Before every Chocolatey upgrade,
`Chocolatey.PrepareUpgrade` deletes any stale `.chocolateyPending` file via
`os.Remove`. The fix is:

- **Permanent** — runs before every upgrade, not just once
- **Safe** — the file holds no package data; it is a lock sentinel only
- **Non-fatal** — if deletion fails, the upgrade still runs so Chocolatey
  can report the authoritative error

You will never need to manually delete `.chocolateyPending` files or run
`choco upgrade` from the command line just to clear a stale lock.

### winget

winget does not require elevation for user-scope packages. The upgrade
command is:

```
winget upgrade --id <package> -e
```

### Scoop

Scoop installs to `%USERPROFILE%\scoop` by default and never requires
elevation. The upgrade command is:

```
scoop update <package>
```

## PATH setup

After installing via `go install`, make sure Go's bin directory is on your
PATH:

```powershell
# PowerShell — add to $PROFILE for persistence
$env:PATH += ";$env:USERPROFILE\go\bin"
```

Or use the pre-built binary from [releases](https://github.com/neur0map/glazepkg/releases)
and place it anywhere on your PATH (e.g. `C:\tools\gpk.exe`).

## Terminal recommendations

GlazePKG renders correctly in any terminal that supports 24-bit color and
UTF-8. Recommended options:

- [Windows Terminal](https://github.com/microsoft/terminal) — best overall
  experience; handles ANSI escape codes, Unicode, and resizing cleanly
- PowerShell 7+ inside Windows Terminal
- [Alacritty](https://github.com/alacritty/alacritty) — GPU-rendered,
  very fast

Avoid the legacy `cmd.exe` console host — it does not support 24-bit color
and will render the TUI with degraded color.

## Known limitations

- `windows-updates` uses WMI and may be slow to scan on some systems.
- Chocolatey v1 and v2 are both supported; the `--local-only` flag is
  automatically included for v1.
- NuGet reports packages from the global cache, not per-project installs.
