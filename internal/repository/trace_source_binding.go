package repository

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/mako10k/sealgraph/internal/workfile"
)

const traceSourceBindingSchema = "sealgraph/trace-source-binding/v1"

var ErrTraceSourceNotFound = errors.New("trace source binding not found")

// TraceSourceBinding associates one source_key with a machine-local input path.
// It is deliberately separate from the REF-scoped SourceBinding store.
type TraceSourceBinding struct {
	SourceKey string
	Path      string
}

type traceSourceBindingWire struct {
	Schema    string `json:"schema"`
	SourceKey string `json:"source_key"`
	Path      string `json:"path"`
}

func traceSourceBindingRoot(repositoryDir string) string {
	return filepath.Join(repositoryDir, "local", "trace-sources")
}

func validateTraceSourceKey(key string) error {
	if key == "" {
		return errors.New("trace source key is empty")
	}
	if !utf8.ValidString(key) {
		return errors.New("trace source key is not valid UTF-8")
	}
	return nil
}

func validateTraceSourcePath(path string) error {
	if err := workfile.ValidatePath(path); err != nil {
		return err
	}
	for _, component := range strings.Split(path, "/") {
		if component == ".sealgraph" {
			return fmt.Errorf("file path %q enters the .sealgraph directory", path)
		}
	}
	return nil
}

func traceSourceBindingFilename(key string) string {
	digest := sha256.Sum256([]byte(key))
	return hex.EncodeToString(digest[:]) + ".json"
}

func encodeTraceSourceBinding(binding TraceSourceBinding) ([]byte, error) {
	if err := validateTraceSourceKey(binding.SourceKey); err != nil {
		return nil, err
	}
	if err := validateTraceSourcePath(binding.Path); err != nil {
		return nil, err
	}
	data, err := json.Marshal(traceSourceBindingWire{
		Schema: traceSourceBindingSchema, SourceKey: binding.SourceKey, Path: binding.Path,
	})
	if err != nil {
		return nil, err
	}
	return append(data, '\n'), nil
}

func decodeTraceSourceBinding(data []byte, expectedKey, expectedFilename string) (TraceSourceBinding, error) {
	var wire traceSourceBindingWire
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&wire); err != nil {
		return TraceSourceBinding{}, err
	}
	binding := TraceSourceBinding{SourceKey: wire.SourceKey, Path: wire.Path}
	if wire.Schema != traceSourceBindingSchema {
		return TraceSourceBinding{}, fmt.Errorf("unsupported schema %q", wire.Schema)
	}
	if err := validateTraceSourceKey(binding.SourceKey); err != nil {
		return TraceSourceBinding{}, err
	}
	if expectedKey != "" && binding.SourceKey != expectedKey {
		return TraceSourceBinding{}, fmt.Errorf("trace source filename contains source_key %q, expected %q", binding.SourceKey, expectedKey)
	}
	if expectedFilename != "" && traceSourceBindingFilename(binding.SourceKey) != expectedFilename {
		return TraceSourceBinding{}, errors.New("trace source filename hash does not match source_key")
	}
	canonical, err := encodeTraceSourceBinding(binding)
	if err != nil {
		return TraceSourceBinding{}, err
	}
	if !bytes.Equal(data, canonical) {
		return TraceSourceBinding{}, errors.New("non-deterministic trace source binding representation")
	}
	return binding, nil
}

func ensureTraceSourceBindingRoot(root string) error {
	parent := filepath.Dir(root)
	for _, path := range []string{parent, root} {
		info, err := os.Lstat(path)
		if errors.Is(err, os.ErrNotExist) {
			if err := os.MkdirAll(path, 0o700); err != nil {
				return err
			}
			info, err = os.Lstat(path)
		}
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
			return fmt.Errorf("trace source store path %s is not a regular directory", path)
		}
	}
	return nil
}

func inspectTraceSourceBindingRoot(root string) (bool, error) {
	parent := filepath.Dir(root)
	for _, path := range []string{parent, root} {
		info, err := os.Lstat(path)
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		if err != nil {
			return false, err
		}
		if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
			return false, fmt.Errorf("trace source store path %s is not a regular directory", path)
		}
	}
	return true, nil
}

func readStableTraceSourceFile(path string) ([]byte, os.FileInfo, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, nil, err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return nil, nil, errors.New("trace source binding is not a regular non-symlink file")
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, nil, err
	}
	opened, statErr := file.Stat()
	if statErr == nil && (!os.SameFile(info, opened) || !opened.Mode().IsRegular()) {
		statErr = errors.New("trace source binding changed before read")
	}
	data, readErr := readTraceSourceBytes(file)
	closeErr := file.Close()
	if statErr != nil {
		return nil, nil, statErr
	}
	if readErr != nil {
		return nil, nil, readErr
	}
	if closeErr != nil {
		return nil, nil, closeErr
	}
	final, err := os.Lstat(path)
	if err != nil {
		return nil, nil, err
	}
	if err := validateTraceSourceFileState(opened, final); err != nil {
		return nil, nil, err
	}
	return data, final, nil
}

func readTraceSourceBytes(file *os.File) ([]byte, error) {
	data, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}
	if _, err := file.Seek(0, 0); err != nil {
		return nil, err
	}
	verification, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}
	if !bytes.Equal(data, verification) {
		return nil, errors.New("trace source binding bytes changed during read")
	}
	return data, nil
}

func validateTraceSourceFileState(opened, final os.FileInfo) error {
	if final.Mode()&os.ModeSymlink != 0 || !final.Mode().IsRegular() || !os.SameFile(opened, final) || final.Size() != opened.Size() || !final.ModTime().Equal(opened.ModTime()) || final.Mode() != opened.Mode() {
		return errors.New("trace source binding changed during read")
	}
	return nil
}

func (r *Repository) traceSourceLoad(key string) (TraceSourceBinding, []byte, string, error) {
	if err := validateTraceSourceKey(key); err != nil {
		return TraceSourceBinding{}, nil, "", err
	}
	root := traceSourceBindingRoot(r.dir)
	exists, err := inspectTraceSourceBindingRoot(root)
	if err != nil {
		return TraceSourceBinding{}, nil, "", err
	}
	if !exists {
		return TraceSourceBinding{}, nil, "", fmt.Errorf("%w: %s", ErrTraceSourceNotFound, key)
	}
	filename := traceSourceBindingFilename(key)
	path := filepath.Join(root, filename)
	data, _, err := readStableTraceSourceFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return TraceSourceBinding{}, nil, "", fmt.Errorf("%w: %s", ErrTraceSourceNotFound, key)
	}
	if err != nil {
		return TraceSourceBinding{}, nil, "", fmt.Errorf("read trace source binding for %s: %w", key, err)
	}
	binding, err := decodeTraceSourceBinding(data, key, filename)
	if err != nil {
		return TraceSourceBinding{}, nil, "", fmt.Errorf("trace source binding for %s is corrupt: %w", key, err)
	}
	return binding, data, path, nil
}

func (r *Repository) traceSourceSave(binding TraceSourceBinding, expected []byte) error {
	data, err := encodeTraceSourceBinding(binding)
	if err != nil {
		return err
	}
	root := traceSourceBindingRoot(r.dir)
	if err := ensureTraceSourceBindingRoot(root); err != nil {
		return err
	}
	path := filepath.Join(root, traceSourceBindingFilename(binding.SourceKey))
	if expected != nil {
		current, _, err := readStableTraceSourceFile(path)
		if err != nil {
			return fmt.Errorf("re-read trace source binding for %s: %w", binding.SourceKey, err)
		}
		if !bytes.Equal(current, expected) {
			return fmt.Errorf("trace source binding for %s changed after inspection; no binding was replaced", binding.SourceKey)
		}
	}
	temp, err := os.CreateTemp(root, ".tmp-trace-source-")
	if err != nil {
		return err
	}
	tempPath := temp.Name()
	defer os.Remove(tempPath)
	if err = temp.Chmod(0o600); err == nil {
		_, err = temp.Write(data)
	}
	if err == nil {
		err = temp.Sync()
	}
	closeErr := temp.Close()
	if err == nil {
		err = closeErr
	}
	if err != nil {
		return err
	}
	if err := os.Rename(tempPath, path); err != nil {
		return fmt.Errorf("publish trace source binding for %s atomically: %w", binding.SourceKey, err)
	}
	return nil
}

func (r *Repository) TraceSourceShow(key string) (TraceSourceBinding, error) {
	binding, _, _, err := r.traceSourceLoad(key)
	return binding, err
}

func (r *Repository) TraceSourceList() ([]TraceSourceBinding, error) {
	root := traceSourceBindingRoot(r.dir)
	exists, err := inspectTraceSourceBindingRoot(root)
	if err != nil {
		return nil, err
	}
	if !exists {
		return []TraceSourceBinding{}, nil
	}
	var result []TraceSourceBinding
	err = filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("trace source store contains symbolic link %s", path)
		}
		if entry.IsDir() {
			return nil
		}
		if !entry.Type().IsRegular() || filepath.Ext(entry.Name()) != ".json" {
			return fmt.Errorf("unexpected trace source store entry %s", path)
		}
		key := strings.TrimSuffix(entry.Name(), ".json")
		var binding TraceSourceBinding
		var err error
		binding, _, _, err = r.traceSourceLoadByFilename(key)
		if err != nil {
			return err
		}
		result = append(result, binding)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("list trace source bindings: %w", err)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].SourceKey < result[j].SourceKey })
	return result, nil
}

func (r *Repository) traceSourceLoadByFilename(filename string) (TraceSourceBinding, []byte, string, error) {
	root := traceSourceBindingRoot(r.dir)
	path := filepath.Join(root, filename+".json")
	data, _, err := readStableTraceSourceFile(path)
	if err != nil {
		return TraceSourceBinding{}, nil, "", err
	}
	binding, err := decodeTraceSourceBinding(data, "", filename+".json")
	if err != nil {
		return TraceSourceBinding{}, nil, "", fmt.Errorf("trace source binding %s is corrupt: %w", filename, err)
	}
	return binding, data, path, nil
}

func (r *Repository) TraceSourceBind(ctx context.Context, key, path string) (TraceSourceBinding, error) {
	return withMutation(ctx, r.writer, "bind trace source", func() (TraceSourceBinding, error) {
		if err := validateTraceSourceKey(key); err != nil {
			return TraceSourceBinding{}, err
		}
		if err := validateTraceSourcePath(path); err != nil {
			return TraceSourceBinding{}, err
		}
		current, _, _, err := r.traceSourceLoad(key)
		if err == nil {
			if current.Path == path {
				return current, nil
			}
			return TraceSourceBinding{}, fmt.Errorf("trace source %s is already bound to %q; use trace source rebind with the observed old path", key, current.Path)
		}
		if !errors.Is(err, ErrTraceSourceNotFound) {
			return TraceSourceBinding{}, err
		}
		binding := TraceSourceBinding{SourceKey: key, Path: path}
		if err := r.traceSourceSave(binding, nil); err != nil {
			return TraceSourceBinding{}, err
		}
		return binding, nil
	})
}

func (r *Repository) TraceSourceRebind(ctx context.Context, key, oldPath, newPath string) (TraceSourceBinding, error) {
	return withMutation(ctx, r.writer, "rebind trace source", func() (TraceSourceBinding, error) {
		if err := validateTraceSourceKey(key); err != nil {
			return TraceSourceBinding{}, err
		}
		if err := validateTraceSourcePath(oldPath); err != nil {
			return TraceSourceBinding{}, fmt.Errorf("invalid expected old trace source path: %w", err)
		}
		if err := validateTraceSourcePath(newPath); err != nil {
			return TraceSourceBinding{}, err
		}
		current, data, _, err := r.traceSourceLoad(key)
		if err != nil {
			return TraceSourceBinding{}, err
		}
		if current.Path != oldPath {
			return TraceSourceBinding{}, fmt.Errorf("trace source %s is %q, not expected %q", key, current.Path, oldPath)
		}
		if oldPath == newPath {
			return current, nil
		}
		binding := TraceSourceBinding{SourceKey: key, Path: newPath}
		if err := r.traceSourceSave(binding, data); err != nil {
			return TraceSourceBinding{}, err
		}
		return binding, nil
	})
}

func (r *Repository) TraceSourceUnbind(ctx context.Context, key, oldPath string) (TraceSourceBinding, error) {
	return withMutation(ctx, r.writer, "unbind trace source", func() (TraceSourceBinding, error) {
		if err := validateTraceSourceKey(key); err != nil {
			return TraceSourceBinding{}, err
		}
		if err := validateTraceSourcePath(oldPath); err != nil {
			return TraceSourceBinding{}, fmt.Errorf("invalid expected old trace source path: %w", err)
		}
		current, data, filePath, err := r.traceSourceLoad(key)
		if err != nil {
			return TraceSourceBinding{}, err
		}
		if current.Path != oldPath {
			return TraceSourceBinding{}, fmt.Errorf("trace source %s is %q, not expected %q; no binding was removed", key, current.Path, oldPath)
		}
		recheck, _, err := readStableTraceSourceFile(filePath)
		if err != nil {
			return TraceSourceBinding{}, fmt.Errorf("re-read trace source binding for %s: %w", key, err)
		}
		if !bytes.Equal(recheck, data) {
			return TraceSourceBinding{}, fmt.Errorf("trace source binding for %s changed after inspection; no binding was removed", key)
		}
		if err := os.Remove(filePath); err != nil {
			return TraceSourceBinding{}, err
		}
		return current, nil
	})
}
