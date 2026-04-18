//go:build windows

package updater

import (
	"fmt"
	"os"

	"golang.org/x/sys/windows"
)

// replaceBinary performs the "rename running exe" dance required to swap a
// running gpk.exe on Windows:
//
//  1. Windows refuses to overwrite or delete an executable that is currently
//     running, but it will happily *rename* it. Move the current binary to
//     "<dest>.old" to free its path.
//  2. Move the new binary into the now-vacant destination path.
//  3. Try to remove the old binary immediately. If Windows still has it
//     locked (the current process's image), schedule it for deletion on the
//     next reboot via MoveFileEx(..., MOVEFILE_DELAY_UNTIL_REBOOT) so we
//     don't leave stale ".old" files piling up.
//
// Any write step that fails with a permission error propagates up so the
// caller (wrapReplaceErr) can render the "re-run the installer as admin"
// guidance — the dance still needs write access to the install directory.
func replaceBinary(src, dest string) error {
	oldPath := dest + ".old"
	_ = os.Remove(oldPath) // best-effort cleanup of a stale .old from a prior update

	if err := os.Rename(dest, oldPath); err != nil {
		return fmt.Errorf("rename current binary: %w", err)
	}
	if err := moveFile(src, dest); err != nil {
		_ = os.Rename(oldPath, dest) // roll back so the user isn't left with nothing at dest
		return fmt.Errorf("install new binary: %w", err)
	}
	if err := os.Remove(oldPath); err != nil {
		scheduleDeleteOnReboot(oldPath)
	}
	return nil
}

func scheduleDeleteOnReboot(path string) {
	p, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return
	}
	_ = windows.MoveFileEx(p, nil, windows.MOVEFILE_DELAY_UNTIL_REBOOT)
}
