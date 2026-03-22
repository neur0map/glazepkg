package manager

// Upgrader is implemented by managers that can install updates for a single package.
type Upgrader interface {
	// UpgradePackage runs the manager's single-package upgrade command.
	UpgradePackage(name string) error
}
