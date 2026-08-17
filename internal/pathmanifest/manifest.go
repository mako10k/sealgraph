// Package pathmanifest builds deterministic claims over explicit file paths.
package pathmanifest

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/mako10k/sealgraph/internal/canonical"
)

const (
	Schema             = "sealgraph/path-manifest/v1"
	Claim              = "path-digest-only"
	DigestAlgorithm    = "sha256"
	AggregateAlgorithm = "sha256-canonical-entries-v1"
)

type Entry struct {
	Path   string
	Bytes  int64
	SHA256 string
}

// Build reads only the explicitly named files below workDir and returns one
// canonical manifest document including its trailing LF.
func Build(workDir, source string, paths []string) ([]byte, error) {
	if source == "" {
		return nil, fmt.Errorf("source identity is empty")
	}
	if !utf8.ValidString(source) {
		return nil, fmt.Errorf("source identity is not valid UTF-8")
	}
	normalized, err := normalizePaths(paths)
	if err != nil {
		return nil, err
	}
	entries := make([]Entry, 0, len(normalized))
	for _, path := range normalized {
		entry, err := readEntry(workDir, path)
		if err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}
	return encode(source, entries)
}

func normalizePaths(paths []string) ([]string, error) {
	if len(paths) == 0 {
		return nil, fmt.Errorf("at least one file path is required")
	}
	normalized := append([]string(nil), paths...)
	for _, path := range normalized {
		if err := validatePath(path); err != nil {
			return nil, err
		}
	}
	sort.Strings(normalized)
	for i := 1; i < len(normalized); i++ {
		if normalized[i-1] == normalized[i] {
			return nil, fmt.Errorf("duplicate file path %q", normalized[i])
		}
	}
	return normalized, nil
}

func validatePath(path string) error {
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
		return fmt.Errorf("file path %q contains a backslash; use slash-separated semantic paths", path)
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

func readEntry(workDir, path string) (Entry, error) {
	fullPath, initial, err := inspectPath(workDir, path)
	if err != nil {
		return Entry{}, err
	}
	file, err := os.Open(fullPath)
	if err != nil {
		return Entry{}, fmt.Errorf("open file %q: %w", path, err)
	}
	data, readErr := readStableFile(file, initial)
	closeErr := file.Close()
	if readErr != nil {
		return Entry{}, fmt.Errorf("read file %q: %w", path, readErr)
	}
	if closeErr != nil {
		return Entry{}, fmt.Errorf("close file %q: %w", path, closeErr)
	}
	digest := sha256.Sum256(data)
	return Entry{Path: path, Bytes: int64(len(data)), SHA256: fmt.Sprintf("%x", digest)}, nil
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
		return nil, fmt.Errorf("file changed between inspection and open")
	}
	data, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}
	final, err := file.Stat()
	if err != nil {
		return nil, err
	}
	if !os.SameFile(opened, final) || opened.Size() != final.Size() || !opened.ModTime().Equal(final.ModTime()) || opened.Mode() != final.Mode() {
		return nil, fmt.Errorf("file changed while it was read; retry after the input is stable")
	}
	return data, nil
}

func encode(source string, entries []Entry) ([]byte, error) {
	entryBytes, err := encodeEntries(entries)
	if err != nil {
		return nil, err
	}
	aggregate := sha256.Sum256(entryBytes)
	b := make([]byte, 0, len(entryBytes)+256)
	b = append(b, `{"schema":`...)
	b, err = canonical.AppendString(b, Schema)
	if err != nil {
		return nil, err
	}
	b = append(b, `,"claim":`...)
	b, _ = canonical.AppendString(b, Claim)
	b = append(b, `,"source":`...)
	b, err = canonical.AppendString(b, source)
	if err != nil {
		return nil, err
	}
	b = append(b, `,"digest_algorithm":`...)
	b, _ = canonical.AppendString(b, DigestAlgorithm)
	b = append(b, `,"aggregate_algorithm":`...)
	b, _ = canonical.AppendString(b, AggregateAlgorithm)
	b = append(b, `,"entries":`...)
	b = append(b, entryBytes...)
	b = append(b, `,"aggregate_sha256":`...)
	b, _ = canonical.AppendString(b, fmt.Sprintf("%x", aggregate))
	b = append(b, '}', '\n')
	return b, nil
}

func encodeEntries(entries []Entry) ([]byte, error) {
	b := make([]byte, 0, len(entries)*128)
	b = append(b, '[')
	for i, entry := range entries {
		if i > 0 {
			b = append(b, ',')
		}
		b = append(b, `{"path":`...)
		var err error
		b, err = canonical.AppendString(b, entry.Path)
		if err != nil {
			return nil, err
		}
		b = append(b, `,"bytes":`...)
		b = strconv.AppendInt(b, entry.Bytes, 10)
		b = append(b, `,"sha256":`...)
		b, err = canonical.AppendString(b, entry.SHA256)
		if err != nil {
			return nil, err
		}
		b = append(b, '}')
	}
	return append(b, ']'), nil
}
