package repository

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"

	"github.com/mako10k/sealgraph/internal/domain"
	"github.com/mako10k/sealgraph/internal/workfile"
)

const (
	sourceFile   = ".track"
	sourceSchema = "sealgraph/source-binding/v1"
)

var ErrSourceNotFound = errors.New("local source binding not found")

type SourceBinding struct {
	REF  string
	Path string
}

type LocalSourceAddOptions struct {
	REF               string
	Path              string
	BindSource        bool
	PreserveSemantics bool
	Dependencies      []Dependency
	Parent            string
	Root              bool
	RootSet           bool
	Draft             bool
	DraftSet          bool
}

type LocalSourceAddResult struct {
	Candidate     domain.Candidate
	SourceMode    string
	SourcePath    string
	SourceBinding string
}

type sourceWire struct {
	Schema string `json:"schema"`
	REF    string `json:"ref"`
	Path   string `json:"path"`
}

type sourceStore struct{ candidates candidateStore }

func (s sourceStore) load(ref string) (SourceBinding, []byte, error) {
	if err := domain.ValidateREF(ref); err != nil {
		return SourceBinding{}, nil, err
	}
	if err := s.candidates.inspectDirectory(ref); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return SourceBinding{}, nil, fmt.Errorf("%w: %s", ErrSourceNotFound, ref)
		}
		return SourceBinding{}, nil, err
	}
	path := filepath.Join(s.candidates.directory(ref), sourceFile)
	info, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		return SourceBinding{}, nil, fmt.Errorf("%w: %s", ErrSourceNotFound, ref)
	}
	if err != nil {
		return SourceBinding{}, nil, fmt.Errorf("inspect local source for %s: %w", ref, err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return SourceBinding{}, nil, fmt.Errorf("local source for %s is not a regular non-symlink file", ref)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return SourceBinding{}, nil, fmt.Errorf("read local source for %s: %w", ref, err)
	}
	binding, err := decodeSource(data)
	if err != nil {
		return SourceBinding{}, nil, fmt.Errorf("local source for %s is corrupt: %w", ref, err)
	}
	if binding.REF != ref {
		return SourceBinding{}, nil, fmt.Errorf("local source path %s contains REF %s", ref, binding.REF)
	}
	return binding, data, nil
}

func (s sourceStore) create(binding SourceBinding) error {
	if err := s.validateCreate(binding); err != nil {
		return err
	}
	current, _, err := s.load(binding.REF)
	if err == nil && current.Path == binding.Path {
		return nil
	}
	return s.save(binding)
}

func (s sourceStore) validateCreate(binding SourceBinding) error {
	current, _, err := s.load(binding.REF)
	if err == nil {
		if current.Path == binding.Path {
			return nil
		}
		return fmt.Errorf("local source for %s is already %q; use source rebind with the observed old path", binding.REF, current.Path)
	}
	if !errors.Is(err, ErrSourceNotFound) {
		return err
	}
	return nil
}

func (s sourceStore) replace(ref, oldPath, newPath string) error {
	current, _, err := s.load(ref)
	if err != nil {
		return err
	}
	if current.Path != oldPath {
		return fmt.Errorf("local source for %s is %q, not expected %q; inspect it with 'sealgraph source show %s'", ref, current.Path, oldPath, ref)
	}
	if oldPath == newPath {
		return nil
	}
	return s.save(SourceBinding{REF: ref, Path: newPath})
}

func (s sourceStore) save(binding SourceBinding) error {
	data, err := encodeSource(binding)
	if err != nil {
		return err
	}
	if err := s.candidates.ensureDirectory(binding.REF); err != nil {
		return fmt.Errorf("prepare local source directory for %s: %w", binding.REF, err)
	}
	dir := s.candidates.directory(binding.REF)
	temp, err := os.CreateTemp(dir, ".tmp-source-")
	if err != nil {
		return fmt.Errorf("create local source temporary file: %w", err)
	}
	tempPath := temp.Name()
	defer os.Remove(tempPath)
	if _, err = temp.Write(data); err == nil {
		err = temp.Sync()
	}
	closeErr := temp.Close()
	if err == nil {
		err = closeErr
	}
	if err != nil {
		return fmt.Errorf("write local source for %s: %w", binding.REF, err)
	}
	if err := os.Rename(tempPath, filepath.Join(dir, sourceFile)); err != nil {
		return fmt.Errorf("publish local source for %s atomically: %w", binding.REF, err)
	}
	return nil
}

func (s sourceStore) remove(ref, expectedPath string) error {
	current, data, err := s.load(ref)
	if err != nil {
		return err
	}
	if current.Path != expectedPath {
		return fmt.Errorf("local source for %s is %q, not expected %q; no binding was removed", ref, current.Path, expectedPath)
	}
	path := filepath.Join(s.candidates.directory(ref), sourceFile)
	recheck, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("re-read local source for %s: %w", ref, err)
	}
	if !bytes.Equal(recheck, data) {
		return fmt.Errorf("local source for %s changed after inspection; no binding was removed", ref)
	}
	if err := os.Remove(path); err != nil {
		return fmt.Errorf("remove local source for %s: %w", ref, err)
	}
	s.candidates.removeEmptyParents(path)
	return nil
}

func (s sourceStore) list() ([]SourceBinding, error) {
	var result []SourceBinding
	err := filepath.WalkDir(s.candidates.root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("runtime index contains a symbolic link at %s", path)
		}
		if entry.IsDir() || entry.Name() == candidateFile {
			return nil
		}
		if entry.Name() != sourceFile || !entry.Type().IsRegular() {
			return fmt.Errorf("unexpected runtime index entry %s", path)
		}
		relative, err := filepath.Rel(s.candidates.root, filepath.Dir(path))
		if err != nil {
			return err
		}
		ref := filepath.ToSlash(relative)
		binding, _, err := s.load(ref)
		if err != nil {
			return err
		}
		result = append(result, binding)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("list local sources: %w", err)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].REF < result[j].REF })
	return result, nil
}

func encodeSource(binding SourceBinding) ([]byte, error) {
	if err := domain.ValidateREF(binding.REF); err != nil {
		return nil, err
	}
	if err := workfile.ValidatePath(binding.Path); err != nil {
		return nil, err
	}
	data, err := json.Marshal(sourceWire{Schema: sourceSchema, REF: binding.REF, Path: binding.Path})
	if err != nil {
		return nil, err
	}
	return append(data, '\n'), nil
}

func decodeSource(data []byte) (SourceBinding, error) {
	var wire sourceWire
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&wire); err != nil {
		return SourceBinding{}, err
	}
	if wire.Schema != sourceSchema {
		return SourceBinding{}, fmt.Errorf("unsupported schema %q", wire.Schema)
	}
	binding := SourceBinding{REF: wire.REF, Path: wire.Path}
	canonical, err := encodeSource(binding)
	if err != nil {
		return SourceBinding{}, err
	}
	if !bytes.Equal(data, canonical) {
		return SourceBinding{}, fmt.Errorf("non-deterministic local source representation")
	}
	return binding, nil
}

func (r *Repository) SourceShow(ref string) (SourceBinding, error) {
	binding, _, err := r.sources.load(ref)
	return binding, err
}

func (r *Repository) SourceList() ([]SourceBinding, error) { return r.sources.list() }

func (r *Repository) SourceBind(ctx context.Context, ref, path string) (SourceBinding, error) {
	return withMutation(ctx, r.writer, "bind local source", func() (SourceBinding, error) {
		if err := domain.ValidateREF(ref); err != nil {
			return SourceBinding{}, err
		}
		binding := SourceBinding{REF: ref, Path: path}
		if _, err := workfile.ReadStable(r.workDir, path); err != nil {
			return SourceBinding{}, err
		}
		if err := r.sources.create(binding); err != nil {
			return SourceBinding{}, err
		}
		return binding, nil
	})
}

func (r *Repository) SourceRebind(ctx context.Context, ref, oldPath, newPath string) (SourceBinding, error) {
	return withMutation(ctx, r.writer, "rebind local source", func() (SourceBinding, error) {
		if err := domain.ValidateREF(ref); err != nil {
			return SourceBinding{}, err
		}
		if err := workfile.ValidatePath(oldPath); err != nil {
			return SourceBinding{}, fmt.Errorf("invalid expected old source path: %w", err)
		}
		if _, err := workfile.ReadStable(r.workDir, newPath); err != nil {
			return SourceBinding{}, err
		}
		if err := r.sources.replace(ref, oldPath, newPath); err != nil {
			return SourceBinding{}, err
		}
		return SourceBinding{REF: ref, Path: newPath}, nil
	})
}

func (r *Repository) SourceUnbind(ctx context.Context, ref, oldPath string) (SourceBinding, error) {
	return withMutation(ctx, r.writer, "unbind local source", func() (SourceBinding, error) {
		if err := domain.ValidateREF(ref); err != nil {
			return SourceBinding{}, err
		}
		if err := workfile.ValidatePath(oldPath); err != nil {
			return SourceBinding{}, fmt.Errorf("invalid expected old source path: %w", err)
		}
		binding, _, err := r.sources.load(ref)
		if err != nil {
			return SourceBinding{}, err
		}
		if err := r.sources.remove(ref, oldPath); err != nil {
			return SourceBinding{}, err
		}
		return binding, nil
	})
}

func (r *Repository) AddLocalSource(ctx context.Context, options LocalSourceAddOptions) (LocalSourceAddResult, error) {
	return withMutation(ctx, r.writer, "add local source candidate", func() (LocalSourceAddResult, error) {
		if err := domain.ValidateREF(options.REF); err != nil {
			return LocalSourceAddResult{}, err
		}
		path, mode, err := r.resolveLocalAddPath(ctx, options.REF, options.Path)
		if err != nil {
			return LocalSourceAddResult{}, err
		}
		binding := SourceBinding{REF: options.REF, Path: path}
		if options.BindSource {
			if err := r.sources.validateCreate(binding); err != nil {
				return LocalSourceAddResult{}, err
			}
		}
		content, err := workfile.ReadStable(r.workDir, path)
		if err != nil {
			return LocalSourceAddResult{}, err
		}
		candidate, err := r.addLocked(ctx, AddOptions{
			REF: options.REF, Content: content, Dependencies: options.Dependencies,
			Parent: options.Parent, Root: options.Root, Draft: options.Draft,
		}, options.PreserveSemantics, options.RootSet, options.DraftSet)
		if err != nil {
			return LocalSourceAddResult{}, err
		}
		bindingState := "NONE"
		if options.BindSource {
			if err := r.sources.create(binding); err != nil {
				return LocalSourceAddResult{Candidate: candidate, SourceMode: mode, SourcePath: path, SourceBinding: "NONE"}, fmt.Errorf("candidate %s was updated from %q, but its local source binding was not published: %w; inspect candidate and source state before retrying", options.REF, path, err)
			}
			bindingState = "BOUND"
		} else if existing, _, err := r.sources.load(options.REF); err == nil && existing.Path == path {
			bindingState = "BOUND"
		}
		return LocalSourceAddResult{Candidate: candidate, SourceMode: mode, SourcePath: path, SourceBinding: bindingState}, nil
	})
}

func (r *Repository) resolveLocalAddPath(ctx context.Context, ref, explicitPath string) (string, string, error) {
	if explicitPath != "" {
		if err := workfile.ValidatePath(explicitPath); err != nil {
			return "", "", err
		}
		binding, _, err := r.sources.load(ref)
		if err == nil && binding.Path != explicitPath {
			return "", "", fmt.Errorf("REF %s is bound to %q, not explicit source %q; use source rebind or source unbind before changing the file source", ref, binding.Path, explicitPath)
		}
		if err != nil && !errors.Is(err, ErrSourceNotFound) {
			return "", "", err
		}
		return explicitPath, "explicit-file", nil
	}
	binding, _, err := r.sources.load(ref)
	if err == nil {
		return binding.Path, "bound-source", nil
	}
	if !errors.Is(err, ErrSourceNotFound) {
		return "", "", err
	}
	if err := r.requireAbsentDestination(ctx, ref); err != nil {
		return "", "", fmt.Errorf("existing REF or candidate %s has no local source binding; use --content-file PATH or 'sealgraph source bind %s --file PATH': %w", ref, ref, err)
	}
	if err := workfile.ValidatePath(ref); err != nil {
		return "", "", fmt.Errorf("REF %s cannot be used as an initial source path: %w", ref, err)
	}
	return ref, "initial-ref-path", nil
}
