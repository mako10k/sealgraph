// Package native implements standalone loose object and REF storage.
package native

import (
	"bytes"
	"compress/zlib"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/mako10k/sealgraph/internal/domain"
	"github.com/mako10k/sealgraph/internal/store"
)

type ObjectStore struct {
	objectsDir string
}

func NewObjectStore(repositoryDir string) *ObjectStore {
	return &ObjectStore{objectsDir: filepath.Join(repositoryDir, "objects")}
}

func ObjectID(data []byte) domain.ObjectID {
	envelope := envelope(data)
	digest := sha256.Sum256(envelope)
	return domain.ObjectID{Hex: fmt.Sprintf("%x", digest)}
}

func (s *ObjectStore) WriteBlob(ctx context.Context, data []byte) (domain.ObjectID, error) {
	if err := ctx.Err(); err != nil {
		return domain.ObjectID{}, err
	}
	id := ObjectID(data)
	path := s.objectPath(id)
	if err := s.validateObjectPath(path, false); err != nil {
		return domain.ObjectID{}, err
	}
	if _, err := os.Lstat(path); err == nil {
		if _, readErr := s.ReadObject(ctx, id); readErr != nil {
			return domain.ObjectID{}, fmt.Errorf("object %s already exists but is corrupt; restore or remove it explicitly after inspection: %w", id, readErr)
		}
		return id, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return domain.ObjectID{}, fmt.Errorf("inspect object %s: %w", id, err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return domain.ObjectID{}, fmt.Errorf("create object directory: %w", err)
	}
	temp, err := os.CreateTemp(filepath.Dir(path), ".tmp-object-")
	if err != nil {
		return domain.ObjectID{}, fmt.Errorf("create object temporary file: %w", err)
	}
	tempPath := temp.Name()
	defer os.Remove(tempPath)
	zw := zlib.NewWriter(temp)
	if _, err = zw.Write(envelope(data)); err == nil {
		err = zw.Close()
	}
	if err == nil {
		err = temp.Sync()
	}
	closeErr := temp.Close()
	if err == nil {
		err = closeErr
	}
	if err != nil {
		return domain.ObjectID{}, fmt.Errorf("write object %s: %w", id, err)
	}
	if err := os.Chmod(tempPath, 0o444); err != nil {
		return domain.ObjectID{}, fmt.Errorf("make object %s immutable: %w", id, err)
	}
	if err := os.Link(tempPath, path); err != nil {
		if !errors.Is(err, os.ErrExist) {
			return domain.ObjectID{}, fmt.Errorf("publish object %s atomically: %w", id, err)
		}
		if _, readErr := s.ReadObject(ctx, id); readErr != nil {
			return domain.ObjectID{}, fmt.Errorf("concurrent object %s is corrupt; inspect it explicitly: %w", id, readErr)
		}
	}
	return id, nil
}

func (s *ObjectStore) ReadObject(ctx context.Context, id domain.ObjectID) (store.Object, error) {
	if err := ctx.Err(); err != nil {
		return store.Object{}, err
	}
	if err := id.ValidateNative(); err != nil {
		return store.Object{}, err
	}
	path := s.objectPath(id)
	if err := s.validateObjectPath(path, true); err != nil {
		return store.Object{}, err
	}
	compressed, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return store.Object{}, fmt.Errorf("%w: %s", store.ErrObjectNotFound, id)
	}
	if err != nil {
		return store.Object{}, fmt.Errorf("read object %s: %w", id, err)
	}
	source := bytes.NewReader(compressed)
	zr, err := zlib.NewReader(source)
	if err != nil {
		return store.Object{}, fmt.Errorf("object %s has invalid zlib envelope: %w", id, err)
	}
	uncompressed, readErr := io.ReadAll(zr)
	closeErr := zr.Close()
	if readErr != nil {
		return store.Object{}, fmt.Errorf("decompress object %s: %w", id, readErr)
	}
	if closeErr != nil {
		return store.Object{}, fmt.Errorf("verify object %s compression checksum: %w", id, closeErr)
	}
	if source.Len() != 0 {
		return store.Object{}, fmt.Errorf("object %s has %d trailing compressed bytes", id, source.Len())
	}
	digest := sha256.Sum256(uncompressed)
	actual := fmt.Sprintf("%x", digest)
	if actual != id.Hex {
		return store.Object{}, fmt.Errorf("object %s hash mismatch: content hashes to %s; restore the expected immutable object explicitly", id, actual)
	}
	typeName, data, err := parseEnvelope(uncompressed)
	if err != nil {
		return store.Object{}, fmt.Errorf("object %s is corrupt: %w", id, err)
	}
	return store.Object{ID: id, Type: typeName, Data: data}, nil
}

// ResolvePrefix resolves a 4-to-64-character lower-case hex object name across
// the native loose ODB. It resolves names only; callers must still read and
// validate the matched object's semantic type.
func (s *ObjectStore) ResolvePrefix(ctx context.Context, prefix string) (domain.ObjectID, error) {
	if err := ctx.Err(); err != nil {
		return domain.ObjectID{}, err
	}
	if !domain.IsObjectPrefix(prefix) {
		return domain.ObjectID{}, fmt.Errorf("object prefix %q must be 4 to 64 lower-case hexadecimal characters", prefix)
	}
	entries, err := os.ReadDir(s.objectsDir)
	if err != nil {
		return domain.ObjectID{}, fmt.Errorf("read object store: %w", err)
	}
	var match string
	for _, fanout := range entries {
		if err := ctx.Err(); err != nil {
			return domain.ObjectID{}, err
		}
		name := fanout.Name()
		if len(name) != 2 || !isLowerHex(name) || !strings.HasPrefix(name, prefix[:min(len(prefix), 2)]) {
			continue
		}
		if fanout.Type()&os.ModeSymlink != 0 || !fanout.IsDir() {
			return domain.ObjectID{}, fmt.Errorf("object fanout %s is not a real directory", name)
		}
		files, err := os.ReadDir(filepath.Join(s.objectsDir, name))
		if err != nil {
			return domain.ObjectID{}, fmt.Errorf("read object fanout %s: %w", name, err)
		}
		for _, file := range files {
			full := name + file.Name()
			if len(full) != 64 || !isLowerHex(full) || !strings.HasPrefix(full, prefix) {
				continue
			}
			if file.Type()&os.ModeSymlink != 0 || !file.Type().IsRegular() {
				return domain.ObjectID{}, fmt.Errorf("object path %s is not a regular file", full)
			}
			if match != "" && match != full {
				return domain.ObjectID{}, fmt.Errorf("%w %q; use more hexadecimal characters", store.ErrAmbiguousObjectPrefix, prefix)
			}
			match = full
		}
	}
	if match == "" {
		return domain.ObjectID{}, fmt.Errorf("%w: prefix %s", store.ErrObjectNotFound, prefix)
	}
	return domain.ObjectID{Hex: match}, nil
}

func isLowerHex(text string) bool {
	for _, c := range text {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
			return false
		}
	}
	return true
}

func (s *ObjectStore) objectPath(id domain.ObjectID) string {
	return filepath.Join(s.objectsDir, id.Hex[:2], id.Hex[2:])
}

func (s *ObjectStore) validateObjectPath(path string, requireObject bool) error {
	rootInfo, err := os.Lstat(s.objectsDir)
	if err != nil {
		return fmt.Errorf("inspect object store: %w", err)
	}
	if !rootInfo.IsDir() || rootInfo.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("object store is not a real directory; inspect %s explicitly", s.objectsDir)
	}
	directory := filepath.Dir(path)
	directoryInfo, err := os.Lstat(directory)
	if err == nil {
		if !directoryInfo.IsDir() || directoryInfo.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("object fanout path %s is not a real directory", directory)
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("inspect object fanout path: %w", err)
	} else if requireObject {
		return fmt.Errorf("%w: object fanout directory does not exist", store.ErrObjectNotFound)
	}
	objectInfo, err := os.Lstat(path)
	if err == nil {
		if !objectInfo.Mode().IsRegular() || objectInfo.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("object path %s is not a regular immutable file", path)
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("inspect object path: %w", err)
	} else if requireObject {
		return fmt.Errorf("%w: object file does not exist", store.ErrObjectNotFound)
	}
	return nil
}

func envelope(data []byte) []byte {
	header := []byte("blob " + strconv.Itoa(len(data)) + "\x00")
	return append(header, data...)
}

func parseEnvelope(data []byte) (string, []byte, error) {
	nul := bytes.IndexByte(data, 0)
	if nul < 0 {
		return "", nil, errors.New("object envelope has no NUL terminator")
	}
	header := string(data[:nul])
	typeName, sizeText, ok := strings.Cut(header, " ")
	if !ok || typeName != domain.BlobType || sizeText == "" {
		return "", nil, fmt.Errorf("invalid object header %q; expected 'blob <size>'", header)
	}
	if len(sizeText) > 1 && sizeText[0] == '0' {
		return "", nil, fmt.Errorf("object size %q has a leading zero", sizeText)
	}
	size, err := strconv.Atoi(sizeText)
	if err != nil || size < 0 {
		return "", nil, fmt.Errorf("invalid object size %q", sizeText)
	}
	payload := data[nul+1:]
	if len(payload) != size {
		return "", nil, fmt.Errorf("object declares %d payload bytes but contains %d", size, len(payload))
	}
	return typeName, payload, nil
}

func (s *ObjectStore) PathForTesting(id domain.ObjectID) string {
	return s.objectPath(id)
}
