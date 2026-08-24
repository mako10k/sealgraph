//go:build linux

package cli

import (
	"syscall"
	"unsafe"
)

type terminalWindowSize struct {
	rows, columns  uint16
	xpixel, ypixel uint16
}

func terminalColumns(fd uintptr) int {
	var size terminalWindowSize
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, fd, uintptr(syscall.TIOCGWINSZ), uintptr(unsafe.Pointer(&size)))
	if errno != 0 || size.columns == 0 {
		return 0
	}
	return int(size.columns)
}
