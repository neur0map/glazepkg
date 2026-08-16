//go:build !noselfupdate

package updater

// DisabledByCompilation reports whether this build can replace its own binary.
const DisabledByCompilation = false

// Update downloads the latest release and replaces the running executable,
// returning the new version.
func Update(currentVersion string) (string, error) {
	return update(currentVersion)
}
