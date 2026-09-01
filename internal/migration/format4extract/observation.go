// Package format4extract implements the only read-only format-4 repository
// reader shipped in the format-5 binary. It exposes extraction, not a live
// repository handle or any source mutation operation.
package format4extract

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"reflect"

	"github.com/mako10k/sealgraph/internal/domain"
	"github.com/mako10k/sealgraph/internal/fsread"
)

const format4Config = "repository_format = 4\nobject_format = sha256\nref_format = manifest-v1\n"

type observedEntry struct {
	Kind   string
	Length int
	Digest [sha256.Size]byte
}

type sourceSnapshot struct {
	config      []byte
	refs        map[string][]byte
	objects     map[string]observedEntry
	objectBytes map[string][]byte
	candidates  map[string]observedEntry
}

func captureSource(ctx context.Context, workDir string) (sourceSnapshot, error) {
	repositoryDir := filepath.Join(workDir, ".sealgraph")
	if err := requireRealDirectory(repositoryDir, "format-4 repository"); err != nil {
		return sourceSnapshot{}, err
	}
	config, err := readRegular(filepath.Join(repositoryDir, "config"), "format-4 config")
	if err != nil {
		return sourceSnapshot{}, err
	}
	if !bytes.Equal(config, []byte(format4Config)) {
		return sourceSnapshot{}, fmt.Errorf("source config is not exact format 4")
	}
	for _, relative := range []string{"objects", "refs", filepath.Join("refs", "seals"), "index", "locks"} {
		if err := requireRealDirectory(filepath.Join(repositoryDir, relative), relative); err != nil {
			return sourceSnapshot{}, err
		}
	}
	if err := requireOnlySealsRefNamespace(filepath.Join(repositoryDir, "refs")); err != nil {
		return sourceSnapshot{}, err
	}
	refs, err := captureREFs(ctx, filepath.Join(repositoryDir, "refs", "seals"))
	if err != nil {
		return sourceSnapshot{}, err
	}
	objects, objectBytes, err := captureObjects(ctx, filepath.Join(repositoryDir, "objects"))
	if err != nil {
		return sourceSnapshot{}, err
	}
	candidates, err := captureCandidates(ctx, filepath.Join(repositoryDir, "index"))
	if err != nil {
		return sourceSnapshot{}, err
	}
	return sourceSnapshot{config: config, refs: refs, objects: objects, objectBytes: objectBytes, candidates: candidates}, nil
}

func (snapshot sourceSnapshot) sameObservation(other sourceSnapshot) bool {
	return bytes.Equal(snapshot.config, other.config) &&
		reflect.DeepEqual(snapshot.refs, other.refs) &&
		reflect.DeepEqual(snapshot.objects, other.objects) &&
		reflect.DeepEqual(snapshot.candidates, other.candidates)
}

func requireRealDirectory(path, label string) error {
	directory, err := fsread.OpenDirectoryNoFollow(path)
	if err != nil {
		return fmt.Errorf("inspect %s: %w", label, err)
	}
	defer directory.Close()
	info, err := directory.Stat()
	if err != nil {
		return fmt.Errorf("inspect %s: %w", label, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("%s is not a real directory", label)
	}
	return nil
}

func readRegular(path, label string) ([]byte, error) {
	file, err := fsread.OpenFileNoFollow(path)
	if err != nil {
		return nil, fmt.Errorf("inspect %s: %w", label, err)
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("inspect %s: %w", label, err)
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("%s is not a regular non-symlink file", label)
	}
	data, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", label, err)
	}
	return data, nil
}

func requireOnlySealsRefNamespace(path string) error {
	entries, err := os.ReadDir(path)
	if err != nil {
		return fmt.Errorf("list refs namespace: %w", err)
	}
	for _, entry := range entries {
		if entry.Name() != "seals" || entry.Type()&os.ModeSymlink != 0 || !entry.IsDir() {
			return fmt.Errorf("unexpected format-4 refs entry %q; expected only real directory seals", entry.Name())
		}
	}
	return nil
}

func captureREFs(ctx context.Context, root string) (map[string][]byte, error) {
	result := make(map[string][]byte)
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("REF namespace contains symbolic link %s", path)
		}
		if entry.IsDir() {
			if path == root {
				return nil
			}
			refPath, err := filepath.Rel(root, path)
			if err != nil {
				return err
			}
			if err := domain.ValidateREF(filepath.ToSlash(refPath)); err != nil {
				return fmt.Errorf("invalid stored REF directory %q: %w", filepath.ToSlash(refPath), err)
			}
			return nil
		}
		if entry.Name() != ".ref" || !entry.Type().IsRegular() {
			return fmt.Errorf("unexpected REF entry %s; expected only regular .ref manifests", path)
		}
		refPath, err := filepath.Rel(root, filepath.Dir(path))
		if err != nil {
			return err
		}
		ref := filepath.ToSlash(refPath)
		data, err := readRegular(path, "REF "+ref+" manifest")
		if err != nil {
			return err
		}
		result[ref] = data
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("capture REF manifests: %w", err)
	}
	return result, nil
}

func captureObjects(ctx context.Context, root string) (map[string]observedEntry, map[string][]byte, error) {
	entries := make(map[string]observedEntry)
	contents := make(map[string][]byte)
	err := walkNamespace(ctx, root, func(filesystemPath, key string, entry os.DirEntry) error {
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("object namespace contains symbolic link %s", key)
		}
		if entry.IsDir() {
			if err := validateObjectDirectoryPath(key); err != nil {
				return err
			}
			entries[key] = observedEntry{Kind: "DIRECTORY"}
			return nil
		}
		if !entry.Type().IsRegular() {
			return fmt.Errorf("object namespace contains special entry %s", key)
		}
		if _, err := objectIDFromPath(key); err != nil {
			return err
		}
		data, err := readRegular(filesystemPath, "object "+key)
		if err != nil {
			return err
		}
		entries[key] = observedEntry{Kind: "REGULAR", Length: len(data), Digest: sha256.Sum256(data)}
		contents[key] = data
		return nil
	})
	if err != nil {
		return nil, nil, fmt.Errorf("capture loose objects: %w", err)
	}
	return entries, contents, nil
}

func captureCandidates(ctx context.Context, root string) (map[string]observedEntry, error) {
	result := make(map[string]observedEntry)
	err := walkNamespace(ctx, root, func(filesystemPath, key string, entry os.DirEntry) error {
		if entry.Type()&os.ModeSymlink != 0 {
			result[key] = observedEntry{Kind: "SYMLINK"}
			return nil
		}
		if entry.IsDir() {
			if err := domain.ValidateREF(key); err != nil {
				result[key] = observedEntry{Kind: "UNRECOGNIZED_DIRECTORY"}
			}
			return nil
		}
		if !entry.Type().IsRegular() {
			result[key] = observedEntry{Kind: "SPECIAL"}
			return nil
		}
		if entry.Name() == ".track" {
			parent := path.Dir(key)
			if parent == "." {
				result[key] = observedEntry{Kind: "UNRECOGNIZED_TRACK"}
				return nil
			}
			if err := domain.ValidateREF(parent); err != nil {
				result[key] = observedEntry{Kind: "UNRECOGNIZED_TRACK"}
			}
			return nil
		}
		data, err := readRegular(filesystemPath, "Candidate namespace entry "+key)
		if err != nil {
			return err
		}
		result[key] = observedEntry{Kind: "REGULAR", Length: len(data), Digest: sha256.Sum256(data)}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("capture Candidate namespace: %w", err)
	}
	return result, nil
}

func walkNamespace(ctx context.Context, root string, visit func(path, key string, entry os.DirEntry) error) error {
	return filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		if path == root {
			return nil
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		key := filepath.ToSlash(relative)
		return visit(path, key, entry)
	})
}
