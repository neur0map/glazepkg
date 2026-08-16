package cli

import (
	"fmt"
	"os"
	"testing"
)

// TestMain pins the config and data dirs to throwaway directories for the whole
// package. Several commands read the config (the theme, the install preference,
// the manager skip list), so without this a developer's own config.toml decides
// whether the tests pass. Individual tests still override either dir when the
// setting is what they are testing.
func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "gpk-cli-test-")
	if err != nil {
		fmt.Fprintln(os.Stderr, "cli tests: cannot create temp dir:", err)
		os.Exit(1)
	}
	os.Setenv("XDG_CONFIG_HOME", dir)
	os.Setenv("XDG_DATA_HOME", dir)
	code := m.Run()
	os.RemoveAll(dir)
	os.Exit(code)
}
