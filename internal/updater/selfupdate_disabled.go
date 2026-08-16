//go:build noselfupdate

package updater

import "errors"

// DisabledByCompilation reports whether this build can replace its own binary.
const DisabledByCompilation = true

// ErrDisabled is returned by Update in builds made with the noselfupdate tag.
// Distributions that manage gpk themselves build with it so gpk never fights
// their package database.
var ErrDisabled = errors.New("this build of gpk cannot update itself")

func Update(_ string) (string, error) {
	return "", ErrDisabled
}
