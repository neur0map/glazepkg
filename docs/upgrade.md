# Upgrade system

Pressing `u` wires a manager-aware upgrade path into the UI.

1. `handleListKey` catches the `u` key while the package list is focused and calls `Model.UpgradeSelectedPackage()`.
2. `UpgradeSelectedPackage` reuses `selectedPackage()` to read the current entry, looks up the responsible manager via `manager.BySource`, and verifies the manager is available.
3. The manager is type asserted to the optional `manager.Upgrader` interface; only managers that implement it can receive commands.
4. If the manager is privileged (apt, dnf, pacman, snap, apk, or XBPS), `UpgradeSelectedPackage` saves an `upgradeRequest` and sets `confirmingUpgrade`, which renders a centered overlay describing the command and waits for `y`/`Enter` or `n`/`Esc` so that `u` never runs a privileged upgrade on a single keypress.
5. On confirmation (or immediately for non-privileged managers) the UI schedules `runUpgradeRequest`, which executes `UpgradePackage(name string)` and emits an `upgradeResultMsg`.
6. `Update` turns that message into user feedback: success becomes `Package upgraded successfully`, failures show the native error, and missing `Upgrader` implementations fall back to `manager.ErrUpgradeNotSupported`.
7. Because `manager.BySource` always returns the active manager for the current platform, `u` works the same way on macOS, Linux, and Windows without platform-specific hacks.

### Confirmation for privileged managers

The overlay reminds the user which package and manager are involved, notes that the command requires elevated privileges, and repeats the key hints (`y`/`Enter` to proceed, `n`/`Esc` to cancel). This guard prevents accidentally running a privileged command from the default `u` key binding. Privileged managers include apt, dnf, pacman, snap, apk, and XBPS because their native upgrade commands run as root.

### Supporting a new manager

Implement the standard `manager.Manager` methods (`Name`, `Available`, `Scan`). If your tool can upgrade single packages, implement `manager.Upgrader` by adding:

```go
func (m *YourManager) UpgradePackage(name string) error {
	return exec.Command("your-tool", "upgrade", name).Run()
}
```

Either the UI will run that command directly, or it will show the confirmation overlay if the manager is privileged. If your manager cannot upgrade packages individually, simply omit this method and `gpk` will automatically show `manager.ErrUpgradeNotSupported` instead of running a command.

### Manager coverage

In addition to the existing managers, the following now ship single-package upgrade commands:

- `gem update <name>`
- `flatpak update <name>`
- `pipx upgrade <name>`
- `opam upgrade --yes <name>`
- `apk add --upgrade <name>`
- `xbps-install -S --yes <name>`
- `conda/mamba update --yes <name>`
- `luarocks upgrade <name>`

Each command runs inside the same goroutine as other package manager upgrades so the user interface always reports completion, success, or failure.
