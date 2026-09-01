// Package fsread provides read-only filesystem handles that reject symbolic
// links in every path component.
package fsread

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
)

func OpenFileNoFollow(path string) (*os.File, error) {
	return openPathNoFollow(path, false)
}

func OpenDirectoryNoFollow(path string) (*os.File, error) {
	return openPathNoFollow(path, true)
}

func openPathNoFollow(path string, directory bool) (*os.File, error) {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	absolute = filepath.Clean(absolute)
	directoryFlags := syscall.O_RDONLY | syscall.O_DIRECTORY | syscall.O_NOFOLLOW | syscall.O_CLOEXEC
	fd, err := syscall.Open(string(filepath.Separator), directoryFlags, 0)
	if err != nil {
		return nil, fmt.Errorf("open filesystem root without following symbolic links: %w", err)
	}
	components := strings.Split(strings.TrimPrefix(absolute, string(filepath.Separator)), string(filepath.Separator))
	for index, component := range components {
		if component == "" {
			continue
		}
		flags := directoryFlags
		if index == len(components)-1 && !directory {
			flags = syscall.O_RDONLY | syscall.O_NOFOLLOW | syscall.O_CLOEXEC
		}
		next, openErr := syscall.Openat(fd, component, flags, 0)
		_ = syscall.Close(fd)
		if openErr != nil {
			return nil, fmt.Errorf("open path %s without following symbolic links: %w", absolute, openErr)
		}
		fd = next
	}
	file := os.NewFile(uintptr(fd), absolute)
	if file == nil {
		_ = syscall.Close(fd)
		return nil, fmt.Errorf("adopt no-follow handle for %s", absolute)
	}
	return file, nil
}
