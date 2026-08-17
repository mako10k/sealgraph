package repository

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/mako10k/sealgraph/internal/canonical"
	"github.com/mako10k/sealgraph/internal/domain"
)

var (
	ErrCandidateNotFound = errors.New("candidate not found")
	ErrCandidateChanged  = errors.New("candidate changed after it was loaded")
)

const candidateFile = ".candidate"

type candidateStore struct{ root string }

type candidateSnapshot struct {
	Candidate domain.Candidate
	Bytes     []byte
}

func (s candidateStore) Load(ref string) (domain.Candidate, error) {
	snapshot, err := s.LoadSnapshot(ref)
	return snapshot.Candidate, err
}

func (s candidateStore) LoadSnapshot(ref string) (candidateSnapshot, error) {
	if err := domain.ValidateREF(ref); err != nil {
		return candidateSnapshot{}, err
	}
	if err := s.inspectDirectory(ref); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return candidateSnapshot{}, fmt.Errorf("%w: %s", ErrCandidateNotFound, ref)
		}
		return candidateSnapshot{}, err
	}
	path := s.path(ref)
	info, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		return candidateSnapshot{}, fmt.Errorf("%w: %s", ErrCandidateNotFound, ref)
	}
	if err != nil {
		return candidateSnapshot{}, fmt.Errorf("inspect candidate %s: %w", ref, err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return candidateSnapshot{}, fmt.Errorf("candidate %s is not a regular non-symlink file", ref)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return candidateSnapshot{}, fmt.Errorf("read candidate %s: %w", ref, err)
	}
	candidate, err := canonical.DecodeCandidate(data)
	if err != nil {
		return candidateSnapshot{}, fmt.Errorf("candidate %s is corrupt: %w; recreate it explicitly with add", ref, err)
	}
	if candidate.REF != ref {
		return candidateSnapshot{}, fmt.Errorf("candidate path %s contains REF %s; recreate it explicitly", ref, candidate.REF)
	}
	if err := domain.ValidateCandidate(candidate); err != nil {
		return candidateSnapshot{}, fmt.Errorf("candidate %s is invalid: %w", ref, err)
	}
	return candidateSnapshot{Candidate: candidate, Bytes: data}, nil
}

func (s candidateStore) Save(candidate domain.Candidate) error {
	links, err := domain.NormalizeLinks(candidate.Links)
	if err != nil {
		return err
	}
	attachments, err := domain.NormalizeAttachments(candidate.Attachments)
	if err != nil {
		return err
	}
	candidate.Links, candidate.Attachments = links, attachments
	if err := domain.ValidateCandidate(candidate); err != nil {
		return err
	}
	if err := s.ensureDirectory(candidate.REF); err != nil {
		return fmt.Errorf("prepare candidate %s directory: %w", candidate.REF, err)
	}
	data, err := canonical.EncodeCandidate(candidate)
	if err != nil {
		return fmt.Errorf("encode candidate %s: %w", candidate.REF, err)
	}
	dir := s.directory(candidate.REF)
	temp, err := os.CreateTemp(dir, ".tmp-candidate-")
	if err != nil {
		return fmt.Errorf("create candidate temporary file: %w", err)
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
		return fmt.Errorf("write candidate %s: %w", candidate.REF, err)
	}
	if err := os.Rename(tempPath, s.path(candidate.REF)); err != nil {
		return fmt.Errorf("publish candidate %s atomically: %w", candidate.REF, err)
	}
	return nil
}

func (s candidateStore) RemoveIfUnchanged(ref string, expected []byte) error {
	if err := domain.ValidateREF(ref); err != nil {
		return err
	}
	if err := s.inspectDirectory(ref); err != nil {
		return fmt.Errorf("re-read sealed candidate %s: %w", ref, err)
	}
	path := s.path(ref)
	current, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("%w: candidate %s disappeared", ErrCandidateChanged, ref)
	}
	if err != nil {
		return fmt.Errorf("re-read sealed candidate %s: %w", ref, err)
	}
	if !bytes.Equal(current, expected) {
		return fmt.Errorf("%w: candidate %s no longer matches the sealed version", ErrCandidateChanged, ref)
	}
	if err := os.Remove(path); err != nil {
		return fmt.Errorf("remove sealed candidate %s: %w", ref, err)
	}
	s.removeEmptyParents(path)
	return nil
}

func (s candidateStore) Discard(ref string) error {
	if err := domain.ValidateREF(ref); err != nil {
		return err
	}
	if err := s.inspectDirectory(ref); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("%w: %s", ErrCandidateNotFound, ref)
		}
		return err
	}
	path := s.path(ref)
	info, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("%w: %s", ErrCandidateNotFound, ref)
	}
	if err != nil {
		return fmt.Errorf("inspect candidate %s for discard: %w", ref, err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return fmt.Errorf("candidate %s is not a regular non-symlink file; no state was removed", ref)
	}
	if err := os.Remove(path); err != nil {
		return fmt.Errorf("discard candidate %s: %w", ref, err)
	}
	s.removeEmptyParents(path)
	return nil
}

func (s candidateStore) removeEmptyParents(path string) {
	for parent := filepath.Dir(path); parent != s.root; parent = filepath.Dir(parent) {
		if err := os.Remove(parent); err != nil {
			break
		}
	}
}

func (s candidateStore) List() ([]string, error) {
	var refs []string
	err := filepath.WalkDir(s.root, func(path string, entry fs.DirEntry, walkErr error) error {
		return s.collect(path, entry, walkErr, &refs)
	})
	if err != nil {
		return nil, fmt.Errorf("list candidates: %w", err)
	}
	sort.Strings(refs)
	return refs, nil
}

func (s candidateStore) collect(path string, entry fs.DirEntry, walkErr error, refs *[]string) error {
	if walkErr != nil {
		return walkErr
	}
	if entry.Type()&os.ModeSymlink != 0 {
		return fmt.Errorf("candidate namespace contains a symbolic link at %s", path)
	}
	if entry.IsDir() {
		return nil
	}
	if entry.Name() != candidateFile || !entry.Type().IsRegular() {
		return fmt.Errorf("unexpected candidate entry %s; expected only %s files", path, candidateFile)
	}
	relative, err := filepath.Rel(s.root, filepath.Dir(path))
	if err != nil {
		return err
	}
	ref := filepath.ToSlash(relative)
	if err := domain.ValidateREF(ref); err != nil {
		return fmt.Errorf("invalid candidate path %q: %w", ref, err)
	}
	if _, err := s.LoadSnapshot(ref); err != nil {
		return err
	}
	*refs = append(*refs, ref)
	return nil
}

func (s candidateStore) inspectDirectory(ref string) error {
	current := s.root
	for _, component := range strings.Split(ref, "/") {
		current = filepath.Join(current, component)
		info, err := os.Lstat(current)
		if err != nil {
			return err
		}
		if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("candidate namespace component %s is not a real directory", current)
		}
	}
	return nil
}

func (s candidateStore) ensureDirectory(ref string) error {
	current := s.root
	for _, component := range strings.Split(ref, "/") {
		current = filepath.Join(current, component)
		info, err := os.Lstat(current)
		if errors.Is(err, os.ErrNotExist) {
			if err := os.Mkdir(current, 0o755); err != nil && !errors.Is(err, os.ErrExist) {
				return err
			}
			info, err = os.Lstat(current)
		}
		if err != nil {
			return err
		}
		if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("candidate namespace component %s is not a real directory", current)
		}
	}
	return nil
}

func (s candidateStore) directory(ref string) string {
	return filepath.Join(s.root, filepath.FromSlash(ref))
}

func (s candidateStore) path(ref string) string {
	return filepath.Join(s.directory(ref), candidateFile)
}
