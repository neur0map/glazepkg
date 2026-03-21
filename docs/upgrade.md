# Upgrade system

The new `u` keybinding wires a single, manager-aware upgrade flow into the heart of the UI:

1. `handleListKey` intercepts `u` while the list view is active, grabs the highlighted package, updates the status bar to `upgrading <name>...`, and calls `Model.UpgradeSelectedPackage()`.
2. `UpgradeSelectedPackage` is the central coordinator. It reuses `selectedPackage()` to read the current package, looks up the responsible manager through `manager.BySource`, and calls the manager’s `UpgradePackage(name string) error` implementation.
3. Each manager now implements `Manager.UpgradePackage`. Supported managers run the native single-package command (e.g., `apt install --only-upgrade`, `brew upgrade`, `winget upgrade`, `pip install --upgrade`, `npm update -g`, etc.). Managers that can’t do single-package upgrades simply return `manager.ErrUpgradeNotSupported`.
4. The goroutine returned by `UpgradeSelectedPackage` emits an `upgradeResultMsg`. `Update` translates that message into user feedback: success yields `Package upgraded successfully`, failures surface the raw error, and `ErrUpgradeNotSupported` renders the literal `"This package manager does not support upgrading a single package."` so the UI never falls back to a bulk upgrade.

### Supporting a new manager

To teach a manager about upgrades, implement:

```go
func (m *YourManager) UpgradePackage(name string) error {
    return exec.Command("your-tool", "upgrade", name).Run()
}
```

Return `manager.ErrUpgradeNotSupported` if the manager only supports global upgrades. Once your manager is registered in `manager.All()` and the source constant is added to `model`, the existing `u` flow automatically invokes your implementation.

The same `u` key binding is shared across Windows, macOS, and Linux; the code path selects the right command for the active manager, so users never leave the unified UI to do a single-package upgrade.
