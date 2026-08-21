// Package workfile validates and reads explicit working-directory-relative files.
package workfile

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

// ValidatePath accepts only portable slash-separated relative paths.
func ValidatePath(path string) error {
	if path == "" {
		return fmt.Errorf("file path is empty")
	}
	if !utf8.ValidString(path) {
		return fmt.Errorf("file path is not valid UTF-8")
	}
	if filepath.IsAbs(path) || strings.HasPrefix(path, "/") {
		return fmt.Errorf("file path %q is absolute; use an explicit working-directory-relative path", path)
	}
	if strings.Contains(path, "\\") {
		return fmt.Errorf("file path %q contains a backslash; use slash-separated paths", path)
	}
	for _, r := range path {
		if r < 0x20 || r == 0x7f {
			return fmt.Errorf("file path contains an ASCII control or DEL byte")
		}
	}
	for _, component := range strings.Split(path, "/") {
		if component == "" || component == "." || component == ".." {
			return fmt.Errorf("file path %q contains an empty, dot, or dot-dot component", path)
		}
	}
	return nil
}

// ReadStable returns exact bytes after rejecting symlinks, non-regular files,
// and observable replacement or mutation during the read.
func ReadStable(workDir, path string) ([]byte, error) {
	if err := ValidatePath(path); err != nil {
		return nil, err
	}
	fullPath, initial, err := inspectPath(workDir, path)
	if err != nil {
		return nil, err
	}
	file, err := os.Open(fullPath)
	if err != nil {
		return nil, fmt.Errorf("open file %q: %w", path, err)
	}
	data, readErr := readStableFile(file, initial)
	closeErr := file.Close()
	if readErr != nil {
		return nil, fmt.Errorf("read file %q: %w", path, readErr)
	}
	if closeErr != nil {
		return nil, fmt.Errorf("close file %q: %w", path, closeErr)
	}
	return data, nil
}

func inspectPath(workDir, path string) (string, os.FileInfo, error) {
	current := workDir
	components := strings.Split(path, "/")
	for i, component := range components {
		current = filepath.Join(current, component)
		info, err := os.Lstat(current)
		if err != nil {
			return "", nil, fmt.Errorf("inspect file %q: %w", path, err)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return "", nil, fmt.Errorf("file path %q contains a symbolic link", path)
		}
		if i < len(components)-1 {
			if !info.IsDir() {
				return "", nil, fmt.Errorf("file path %q has a non-directory ancestor", path)
			}
			continue
		}
		if !info.Mode().IsRegular() {
			return "", nil, fmt.Errorf("file path %q is not a regular non-symlink file", path)
		}
		return current, info, nil
	}
	return "", nil, fmt.Errorf("file path %q has no terminal component", path)
}

func readStableFile(file *os.File, initial os.FileInfo) ([]byte, error) {
	opened, err := file.Stat()
	if err != nil {
		return nil, err
	}
	if !opened.Mode().IsRegular() || !os.SameFile(initial, opened) {
		return nil, fmt.Errorf("CHANGED_DURING_READ: file changed between inspection and open")
	}
	data, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return nil, err
	}
	verification, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}
	final, err := file.Stat()
	if err != nil {
		return nil, err
	}
	if !os.SameFile(opened, final) || opened.Size() != final.Size() || !opened.ModTime().Equal(final.ModTime()) || opened.Mode() != final.Mode() {
		return nil, fmt.Errorf("CHANGED_DURING_READ: file changed while it was read; retry after the input is stable")
	}
	if !bytes.Equal(data, verification) {
		return nil, fmt.Errorf("CHANGED_DURING_READ: exact bytes changed between verification reads; retry after the input is stable")
	}
	return data, nil
}
