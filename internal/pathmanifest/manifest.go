// Package pathmanifest builds deterministic claims over explicit file paths.
package pathmanifest

import (
	"crypto/sha256"
	"fmt"
	"sort"
	"strconv"
	"unicode/utf8"

	"github.com/mako10k/sealgraph/internal/canonical"
	"github.com/mako10k/sealgraph/internal/workfile"
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
		if err := workfile.ValidatePath(path); err != nil {
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

func readEntry(workDir, path string) (Entry, error) {
	data, err := workfile.ReadStable(workDir, path)
	if err != nil {
		return Entry{}, err
	}
	digest := sha256.Sum256(data)
	return Entry{Path: path, Bytes: int64(len(data)), SHA256: fmt.Sprintf("%x", digest)}, nil
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
