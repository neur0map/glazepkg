//go:build !windows

package updater

// replaceBinary overwrites dest with src. On Unix the running process keeps
// the old inode open via its existing file descriptor, so a straight atomic
// rename is safe even while gpk is still running.
func replaceBinary(src, dest string) error {
	return moveFile(src, dest)
}
