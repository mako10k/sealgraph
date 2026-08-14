package repository

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/mako10k/sealgraph/internal/domain"
	"github.com/mako10k/sealgraph/internal/store"
)

var ErrCandidateNotFound = errors.New("candidate not found")

type candidateStore struct{ root string }

func (s candidateStore) Load(ref string) (domain.Candidate, error) {
	if err := domain.ValidateREF(ref); err != nil {
		return domain.Candidate{}, err
	}
	if err := s.checkPrefixConflict(ref); err != nil {
		return domain.Candidate{}, err
	}
	data, err := os.ReadFile(s.path(ref))
	if errors.Is(err, os.ErrNotExist) {
		return domain.Candidate{}, fmt.Errorf("%w: %s", ErrCandidateNotFound, ref)
	}
	if err != nil {
		if pathIsDirectory(s.path(ref)) || isNotDirectoryError(err) {
			return domain.Candidate{}, fmt.Errorf("%w: candidate %s collides with a hierarchical candidate", store.ErrPrefixConflict, ref)
		}
		return domain.Candidate{}, fmt.Errorf("read candidate %s: %w", ref, err)
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var candidate domain.Candidate
	if err := decoder.Decode(&candidate); err != nil {
		return domain.Candidate{}, fmt.Errorf("candidate %s is corrupt: %w; recreate it explicitly with add", ref, err)
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return domain.Candidate{}, fmt.Errorf("candidate %s has trailing data; recreate it explicitly with add", ref)
	}
	if candidate.REF != ref {
		return domain.Candidate{}, fmt.Errorf("candidate path %s contains REF %s; recreate it explicitly", ref, candidate.REF)
	}
	if err := domain.ValidateCandidate(candidate); err != nil {
		return domain.Candidate{}, fmt.Errorf("candidate %s is invalid: %w", ref, err)
	}
	return candidate, nil
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
	if err := s.checkPrefixConflict(candidate.REF); err != nil {
		return err
	}
	path := s.path(candidate.REF)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		if isNotDirectoryError(err) {
			return fmt.Errorf("%w: existing candidate is a prefix of %s", store.ErrPrefixConflict, candidate.REF)
		}
		return fmt.Errorf("create candidate directory: %w", err)
	}
	data, err := json.MarshalIndent(candidate, "", "  ")
	if err != nil {
		return fmt.Errorf("encode candidate %s: %w", candidate.REF, err)
	}
	data = append(data, '\n')
	temp, err := os.CreateTemp(filepath.Dir(path), ".tmp-candidate-")
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
	if err := os.Rename(tempPath, path); err != nil {
		return fmt.Errorf("publish candidate %s atomically: %w", candidate.REF, err)
	}
	return nil
}

func (s candidateStore) Remove(ref string) error {
	path := s.path(ref)
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("remove sealed candidate %s: %w", ref, err)
	}
	for parent := filepath.Dir(path); parent != s.root; parent = filepath.Dir(parent) {
		if err := os.Remove(parent); err != nil {
			break
		}
	}
	return nil
}

func (s candidateStore) List() ([]string, error) {
	var refs []string
	err := filepath.WalkDir(s.root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path == s.root || entry.IsDir() {
			return nil
		}
		if entry.Type()&os.ModeSymlink != 0 || !entry.Type().IsRegular() {
			return fmt.Errorf("unexpected non-regular candidate entry %s", path)
		}
		relative, err := filepath.Rel(s.root, path)
		if err != nil {
			return err
		}
		ref := filepath.ToSlash(relative)
		if err := domain.ValidateREF(ref); err != nil {
			return fmt.Errorf("invalid candidate path %q: %w", ref, err)
		}
		refs = append(refs, ref)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("list candidates: %w", err)
	}
	sort.Strings(refs)
	return refs, nil
}

func (s candidateStore) checkPrefixConflict(ref string) error {
	components := strings.Split(ref, "/")
	path := s.root
	for i, component := range components {
		path = filepath.Join(path, component)
		info, err := os.Lstat(path)
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("inspect candidate namespace: %w", err)
		}
		last := i == len(components)-1
		if !last && !info.IsDir() {
			return fmt.Errorf("%w: existing candidate %q is a prefix of %q", store.ErrPrefixConflict, strings.Join(components[:i+1], "/"), ref)
		}
		if last && info.IsDir() {
			return fmt.Errorf("%w: candidate %q is a prefix of an existing hierarchical candidate", store.ErrPrefixConflict, ref)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("candidate namespace contains a symbolic link at %s", path)
		}
	}
	return nil
}

func (s candidateStore) path(ref string) string {
	return filepath.Join(s.root, filepath.FromSlash(ref))
}
func pathIsDirectory(path string) bool { info, err := os.Stat(path); return err == nil && info.IsDir() }
func isNotDirectoryError(err error) bool {
	return strings.Contains(strings.ToLower(err.Error()), "not a directory")
}
