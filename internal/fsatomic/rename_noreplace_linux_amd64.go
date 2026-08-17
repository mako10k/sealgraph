//go:build linux && amd64

package fsatomic

import (
	"fmt"
	"syscall"
	"unsafe"
)

const (
	sysRenameat2        = 316
	atFDCWD             = -100
	renameNoReplaceFlag = 1
)

// RenameNoReplace atomically renames oldPath without replacing newPath.
func RenameNoReplace(oldPath, newPath string) error {
	oldPointer, err := syscall.BytePtrFromString(oldPath)
	if err != nil {
		return err
	}
	newPointer, err := syscall.BytePtrFromString(newPath)
	if err != nil {
		return err
	}
	dirfd := atFDCWD
	_, _, errno := syscall.Syscall6(
		sysRenameat2,
		uintptr(dirfd), uintptr(unsafe.Pointer(oldPointer)),
		uintptr(dirfd), uintptr(unsafe.Pointer(newPointer)),
		renameNoReplaceFlag, 0,
	)
	if errno != 0 {
		return fmt.Errorf("renameat2(RENAME_NOREPLACE): %w", errno)
	}
	return nil
}
