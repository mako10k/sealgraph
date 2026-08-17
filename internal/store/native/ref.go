package native

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/mako10k/sealgraph/internal/domain"
	"github.com/mako10k/sealgraph/internal/store"
)

type RefStore struct {
	refsDir  string
	locksDir string
}

func NewRefStore(repositoryDir string) *RefStore {
	return &RefStore{
		refsDir:  filepath.Join(repositoryDir, "refs", "seals"),
		locksDir: filepath.Join(repositoryDir, "locks", "refs"),
	}
}

func (s *RefStore) Resolve(ctx context.Context, ref string) (domain.ObjectID, error) {
	if err := ctx.Err(); err != nil {
		return domain.ObjectID{}, err
	}
	if err := domain.ValidateREF(ref); err != nil {
		return domain.ObjectID{}, err
	}
	if err := s.checkPrefixConflict(ref); err != nil {
		return domain.ObjectID{}, err
	}
	data, err := os.ReadFile(s.refPath(ref))
	if errors.Is(err, os.ErrNotExist) {
		return domain.ObjectID{}, fmt.Errorf("%w: %s", store.ErrRefNotFound, ref)
	}
	if err != nil {
		if pathIsDirectory(s.refPath(ref)) || isNotDirectory(err) {
			return domain.ObjectID{}, fmt.Errorf("%w: %s collides with an existing REF prefix", store.ErrPrefixConflict, ref)
		}
		return domain.ObjectID{}, fmt.Errorf("read REF %s: %w", ref, err)
	}
	if len(data) == 0 || data[len(data)-1] != '\n' || bytesCount(data, '\n') != 1 {
		return domain.ObjectID{}, fmt.Errorf("REF %s file is corrupt; expected one full object ID followed by LF", ref)
	}
	id, err := domain.ParseObjectID(string(data[:len(data)-1]))
	if err != nil {
		return domain.ObjectID{}, fmt.Errorf("REF %s file is corrupt: %w", ref, err)
	}
	return id, nil
}

func (s *RefStore) Update(ctx context.Context, ref string, oldID, newID *domain.ObjectID) error {
	if err := validateRefUpdate(ctx, ref, oldID, newID); err != nil {
		return err
	}
	if err := s.checkPrefixConflict(ref); err != nil {
		return err
	}
	release, err := s.acquireLock(ref)
	if err != nil {
		return err
	}
	defer release()
	if err := s.checkCAS(ctx, ref, oldID); err != nil {
		return err
	}
	return s.writeREF(ref, *newID)
}

func validateRefUpdate(ctx context.Context, ref string, oldID, newID *domain.ObjectID) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := domain.ValidateREF(ref); err != nil {
		return err
	}
	if newID == nil {
		return errors.New("new REF object ID is required")
	}
	if err := newID.ValidateNative(); err != nil {
		return fmt.Errorf("invalid new REF value: %w", err)
	}
	if oldID != nil {
		if err := oldID.ValidateNative(); err != nil {
			return fmt.Errorf("invalid expected REF value: %w", err)
		}
	}
	return nil
}

func (s *RefStore) acquireLock(ref string) (func(), error) {
	lockPath := s.lockPath(ref)
	if err := os.MkdirAll(filepath.Dir(lockPath), 0o755); err != nil {
		return nil, fmt.Errorf("create REF lock directory: %w", err)
	}
	lock, err := os.OpenFile(lockPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		if errors.Is(err, os.ErrExist) {
			return nil, fmt.Errorf("REF %s is locked; retry after the active operation completes: %w", ref, err)
		}
		return nil, fmt.Errorf("lock REF %s: %w", ref, err)
	}
	if err := lock.Close(); err != nil {
		_ = os.Remove(lockPath)
		return nil, fmt.Errorf("close REF %s lock: %w", ref, err)
	}
	return func() { _ = os.Remove(lockPath) }, nil
}

func (s *RefStore) checkCAS(ctx context.Context, ref string, oldID *domain.ObjectID) error {
	current, resolveErr := s.Resolve(ctx, ref)
	if oldID == nil {
		if resolveErr == nil {
			return fmt.Errorf("%w for %s: expected no head, found %s; recreate the candidate from the current head", store.ErrCASMismatch, ref, current)
		}
		if !errors.Is(resolveErr, store.ErrRefNotFound) {
			return resolveErr
		}
		return nil
	}
	if resolveErr != nil {
		return fmt.Errorf("%w for %s: expected %s but current head cannot be read: %v", store.ErrCASMismatch, ref, oldID, resolveErr)
	}
	if !current.Equal(*oldID) {
		return fmt.Errorf("%w for %s: expected %s, found %s; recreate the candidate from the current head", store.ErrCASMismatch, ref, oldID, current)
	}
	return nil
}

func (s *RefStore) writeREF(ref string, newID domain.ObjectID) error {
	path := s.refPath(ref)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		if isNotDirectory(err) {
			return fmt.Errorf("%w: %s collides with an existing REF prefix", store.ErrPrefixConflict, ref)
		}
		return fmt.Errorf("create REF directory: %w", err)
	}
	temp, err := os.CreateTemp(filepath.Dir(path), ".tmp-ref-")
	if err != nil {
		return fmt.Errorf("create temporary REF %s: %w", ref, err)
	}
	tempPath := temp.Name()
	defer os.Remove(tempPath)
	if _, err = fmt.Fprintln(temp, newID.String()); err == nil {
		err = temp.Sync()
	}
	closeErr := temp.Close()
	if err == nil {
		err = closeErr
	}
	if err != nil {
		return fmt.Errorf("write temporary REF %s: %w", ref, err)
	}
	if err := os.Rename(tempPath, path); err != nil {
		return fmt.Errorf("publish REF %s atomically: %w", ref, err)
	}
	return nil
}

func (s *RefStore) List(ctx context.Context) ([]string, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	var refs []string
	err := filepath.WalkDir(s.refsDir, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path == s.refsDir || entry.IsDir() {
			return nil
		}
		if entry.Type()&os.ModeSymlink != 0 || !entry.Type().IsRegular() {
			return fmt.Errorf("unexpected non-regular REF entry %s", path)
		}
		relative, err := filepath.Rel(s.refsDir, path)
		if err != nil {
			return err
		}
		ref := filepath.ToSlash(relative)
		if err := domain.ValidateREF(ref); err != nil {
			return fmt.Errorf("invalid stored REF path %q: %w", ref, err)
		}
		refs = append(refs, ref)
		return nil
	})
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("list REFs: %w", err)
	}
	sort.Strings(refs)
	return refs, nil
}

func (s *RefStore) checkPrefixConflict(ref string) error {
	components := strings.Split(ref, "/")
	path := s.refsDir
	for i, component := range components {
		path = filepath.Join(path, component)
		info, err := os.Lstat(path)
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("inspect REF namespace %s: %w", ref, err)
		}
		last := i == len(components)-1
		if !last && !info.IsDir() {
			return fmt.Errorf("%w: existing REF %q is a prefix of %q", store.ErrPrefixConflict, strings.Join(components[:i+1], "/"), ref)
		}
		if last && info.IsDir() {
			return fmt.Errorf("%w: REF %q is a prefix of an existing hierarchical REF", store.ErrPrefixConflict, ref)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("REF namespace %s contains a symbolic link; inspect it explicitly", ref)
		}
	}
	return nil
}

func (s *RefStore) refPath(ref string) string {
	return filepath.Join(s.refsDir, filepath.FromSlash(ref))
}
func (s *RefStore) lockPath(ref string) string {
	return filepath.Join(s.locksDir, filepath.FromSlash(ref)+".lock")
}

func bytesCount(data []byte, target byte) int {
	count := 0
	for _, b := range data {
		if b == target {
			count++
		}
	}
	return count
}
func isNotDirectory(err error) bool {
	return strings.Contains(strings.ToLower(err.Error()), "not a directory")
}
func pathIsDirectory(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}
