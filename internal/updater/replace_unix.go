//go:build !windows

package updater

import "os"

// renameFile is os.Rename in production; tests override it to simulate the
// cross-filesystem failure that triggers the rename-aside path.
var renameFile = os.Rename

// replaceBinary overwrites dest with src.
//
// Same-filesystem fast path: a straight rename. The running gpk process
// keeps the old inode open via its existing file descriptor, so the rename
// is safe even while gpk is still running.
//
// Cross-filesystem path: when src lives on a different mount than dest
// (typical: /tmp is tmpfs, dest is /usr/bin on disk), os.Rename returns
// EXDEV. The fallback in moveFile then tries os.OpenFile(dest, O_TRUNC),
// which Linux refuses with ETXTBSY ("text file busy") because dest is the
// currently-running executable.
//
// To get past ETXTBSY we do the same rename-out-of-the-way dance as
// Windows: move dest to "<dest>.old" first (which works for a running
// binary because rename keeps the inode), then copy src into the now-vacant
// dest path. The .old binary can be unlinked immediately afterwards on
// Unix; the kernel preserves the inode for the running process until exit.
func replaceBinary(src, dest string) error {
	if err := renameFile(src, dest); err == nil {
		return nil
	}

	oldPath := dest + ".old"
	_ = os.Remove(oldPath) // best-effort cleanup of a stale .old from a prior failed update

	if err := os.Rename(dest, oldPath); err != nil {
		return err
	}
	if err := moveFile(src, dest); err != nil {
		_ = os.Rename(oldPath, dest) // roll back so the user isn't left with no binary at dest
		return err
	}
	_ = os.Remove(oldPath) // safe: Unix lets you unlink a running binary; the inode lives on
	return nil
}
