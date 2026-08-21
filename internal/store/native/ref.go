package native

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/mako10k/sealgraph/internal/domain"
	"github.com/mako10k/sealgraph/internal/fsatomic"
	"github.com/mako10k/sealgraph/internal/store"
)

const refManifestFile = ".ref"

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
	manifest, err := s.loadManifest(ctx, ref)
	if err != nil {
		return domain.ObjectID{}, err
	}
	return manifest.Head, nil
}

func (s *RefStore) Snapshot(ctx context.Context, ref string) ([]byte, error) {
	manifest, err := s.loadManifest(ctx, ref)
	if err != nil {
		if errors.Is(err, store.ErrRefNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return encodeRefManifest(manifest)
}

func (s *RefStore) PreviewUpdate(ctx context.Context, ref string, oldID, newID *domain.ObjectID) ([]byte, error) {
	if err := validateRefUpdate(ctx, ref, oldID, newID); err != nil {
		return nil, err
	}
	manifest, err := s.manifestForUpdate(ctx, ref, oldID)
	if err != nil {
		return nil, err
	}
	manifest.Head = *newID
	return encodeRefManifest(manifest)
}

func (s *RefStore) PreviewTag(ctx context.Context, ref, name string, id, expectedHead domain.ObjectID) ([]byte, error) {
	manifest, err := s.loadManifest(ctx, ref)
	if err != nil {
		return nil, err
	}
	if !manifest.Head.Equal(expectedHead) {
		return nil, fmt.Errorf("%w for %s tag preview", store.ErrCASMismatch, ref)
	}
	for _, existing := range manifest.Tags {
		if existing.Name == name {
			if existing.Seal.Equal(id) {
				return encodeRefManifest(manifest)
			}
			return nil, store.ErrTagConflict
		}
	}
	manifest.Tags = append(manifest.Tags, store.Tag{Name: name, Seal: id})
	return encodeRefManifest(manifest)
}

func (s *RefStore) ManifestTargets(data []byte) ([]domain.ObjectID, error) {
	manifest, err := decodeRefManifest(data)
	if err != nil {
		return nil, err
	}
	targets := []domain.ObjectID{manifest.Head}
	for _, tag := range manifest.Tags {
		targets = append(targets, tag.Seal)
	}
	return targets, nil
}

func (s *RefStore) Update(ctx context.Context, ref string, oldID, newID *domain.ObjectID) error {
	if err := validateRefUpdate(ctx, ref, oldID, newID); err != nil {
		return err
	}
	release, err := s.acquireLocks(ref)
	if err != nil {
		return err
	}
	defer release()

	manifest, err := s.manifestForUpdate(ctx, ref, oldID)
	if err != nil {
		return err
	}
	manifest.Head = *newID
	return s.writeManifest(ref, manifest)
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

func (s *RefStore) manifestForUpdate(ctx context.Context, ref string, oldID *domain.ObjectID) (refManifest, error) {
	current, err := s.loadManifest(ctx, ref)
	if oldID == nil {
		if err == nil {
			return refManifest{}, fmt.Errorf("%w for %s: expected no head, found %s; recreate the candidate from the current head", store.ErrCASMismatch, ref, current.Head)
		}
		if !errors.Is(err, store.ErrRefNotFound) {
			return refManifest{}, err
		}
		return refManifest{Schema: refManifestSchema, Tags: []store.Tag{}}, nil
	}
	if err != nil {
		return refManifest{}, fmt.Errorf("%w for %s: expected %s but current head cannot be read: %v", store.ErrCASMismatch, ref, oldID, err)
	}
	if !current.Head.Equal(*oldID) {
		return refManifest{}, fmt.Errorf("%w for %s: expected %s, found %s; recreate the candidate from the current head", store.ErrCASMismatch, ref, oldID, current.Head)
	}
	return current, nil
}

func (s *RefStore) Move(ctx context.Context, oldRef, newRef string) error {
	if err := validateRefMove(ctx, oldRef, newRef); err != nil {
		return err
	}
	release, err := s.acquireLocks(oldRef, newRef)
	if err != nil {
		return err
	}
	defer release()
	if _, err := s.loadManifest(ctx, oldRef); err != nil {
		return fmt.Errorf("move source REF %s: %w", oldRef, err)
	}
	if destination, err := s.loadManifest(ctx, newRef); err == nil {
		return fmt.Errorf("move destination REF %s already exists at %s", newRef, destination.Head)
	} else if !errors.Is(err, store.ErrRefNotFound) {
		return fmt.Errorf("inspect move destination REF %s: %w", newRef, err)
	}
	if err := s.ensureRefDirectory(newRef); err != nil {
		return err
	}
	oldPath := s.manifestPath(oldRef)
	newPath := s.manifestPath(newRef)
	if err := fsatomic.RenameNoReplace(oldPath, newPath); err != nil {
		return fmt.Errorf("move REF %s to %s atomically without replacement: %w", oldRef, newRef, err)
	}
	if err := syncMovedDirectories(filepath.Dir(oldPath), filepath.Dir(newPath)); err != nil {
		return fmt.Errorf("REF move %s to %s committed but directory durability failed: %w; inspect both names before any retry", oldRef, newRef, err)
	}
	return nil
}

func (s *RefStore) ReplaceExact(ctx context.Context, ref string, expected, replacement []byte) error {
	if err := domain.ValidateREF(ref); err != nil {
		return err
	}
	var replacementManifest refManifest
	if replacement != nil {
		decoded, err := decodeRefManifest(replacement)
		if err != nil {
			return fmt.Errorf("replacement REF %s manifest is invalid: %w", ref, err)
		}
		replacementManifest = decoded
	}
	release, err := s.acquireLocks(ref)
	if err != nil {
		return err
	}
	defer release()
	current, err := s.Snapshot(ctx, ref)
	if err != nil {
		return err
	}
	if !bytes.Equal(current, expected) {
		return fmt.Errorf("%w for exact REF %s manifest replacement", store.ErrCASMismatch, ref)
	}
	if replacement != nil {
		return s.writeManifest(ref, replacementManifest)
	}
	path := s.manifestPath(ref)
	if err := os.Remove(path); err != nil {
		return fmt.Errorf("remove exact REF %s manifest: %w", ref, err)
	}
	if err := syncDirectory(filepath.Dir(path)); err != nil {
		return fmt.Errorf("REF %s removal committed but directory durability failed: %w", ref, err)
	}
	return nil
}

func (s *RefStore) MoveExact(ctx context.Context, oldRef, newRef string, oldExpected, newExpected []byte) error {
	if err := validateRefMove(ctx, oldRef, newRef); err != nil {
		return err
	}
	release, err := s.acquireLocks(oldRef, newRef)
	if err != nil {
		return err
	}
	defer release()
	oldCurrent, err := s.Snapshot(ctx, oldRef)
	if err != nil {
		return err
	}
	newCurrent, err := s.Snapshot(ctx, newRef)
	if err != nil {
		return err
	}
	if !bytes.Equal(oldCurrent, oldExpected) || !bytes.Equal(newCurrent, newExpected) {
		return fmt.Errorf("%w for exact REF move %s to %s", store.ErrCASMismatch, oldRef, newRef)
	}
	if oldCurrent == nil || newCurrent != nil {
		return fmt.Errorf("exact REF move requires present source and absent destination")
	}
	if err := s.ensureRefDirectory(newRef); err != nil {
		return err
	}
	oldPath, newPath := s.manifestPath(oldRef), s.manifestPath(newRef)
	if err := fsatomic.RenameNoReplace(oldPath, newPath); err != nil {
		return err
	}
	return syncMovedDirectories(filepath.Dir(oldPath), filepath.Dir(newPath))
}

func validateRefMove(ctx context.Context, oldRef, newRef string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := domain.ValidateREF(oldRef); err != nil {
		return fmt.Errorf("invalid source REF: %w", err)
	}
	if err := domain.ValidateREF(newRef); err != nil {
		return fmt.Errorf("invalid destination REF: %w", err)
	}
	if oldRef == newRef {
		return fmt.Errorf("source and destination REF are both %s; choose a different absent destination", oldRef)
	}
	return nil
}

func (s *RefStore) List(ctx context.Context) ([]string, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	var refs []string
	err := filepath.WalkDir(s.refsDir, func(path string, entry fs.DirEntry, walkErr error) error {
		return s.collectREF(ctx, path, entry, walkErr, &refs)
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

func (s *RefStore) collectREF(ctx context.Context, path string, entry fs.DirEntry, walkErr error, refs *[]string) error {
	if walkErr != nil {
		return walkErr
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if entry.Type()&os.ModeSymlink != 0 {
		return fmt.Errorf("REF namespace contains a symbolic link at %s", path)
	}
	if entry.IsDir() {
		return nil
	}
	if entry.Name() != refManifestFile || !entry.Type().IsRegular() {
		return fmt.Errorf("unexpected canonical REF entry %s; expected only %s manifest files", path, refManifestFile)
	}
	relative, err := filepath.Rel(s.refsDir, filepath.Dir(path))
	if err != nil {
		return err
	}
	ref := filepath.ToSlash(relative)
	if err := domain.ValidateREF(ref); err != nil {
		return fmt.Errorf("invalid stored REF path %q: %w", ref, err)
	}
	if _, err := s.loadManifest(ctx, ref); err != nil {
		return err
	}
	*refs = append(*refs, ref)
	return nil
}

func (s *RefStore) loadManifest(ctx context.Context, ref string) (refManifest, error) {
	if err := ctx.Err(); err != nil {
		return refManifest{}, err
	}
	if err := domain.ValidateREF(ref); err != nil {
		return refManifest{}, err
	}
	if err := inspectDirectoryChain(s.refsDir, ref); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return refManifest{}, fmt.Errorf("%w: %s", store.ErrRefNotFound, ref)
		}
		return refManifest{}, err
	}
	path := s.manifestPath(ref)
	info, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		return refManifest{}, fmt.Errorf("%w: %s", store.ErrRefNotFound, ref)
	}
	if err != nil {
		return refManifest{}, fmt.Errorf("inspect REF %s manifest: %w", ref, err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return refManifest{}, fmt.Errorf("REF %s manifest is not a regular non-symlink file", ref)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return refManifest{}, fmt.Errorf("read REF %s manifest: %w", ref, err)
	}
	manifest, err := decodeRefManifest(data)
	if err != nil {
		return refManifest{}, fmt.Errorf("REF %s manifest is corrupt: %w", ref, err)
	}
	return manifest, nil
}

func (s *RefStore) writeManifest(ref string, manifest refManifest) error {
	data, err := encodeRefManifest(manifest)
	if err != nil {
		return fmt.Errorf("encode REF %s manifest: %w", ref, err)
	}
	if err := s.ensureRefDirectory(ref); err != nil {
		return err
	}
	dir := s.refDirectory(ref)
	temp, err := os.CreateTemp(dir, ".tmp-ref-")
	if err != nil {
		return fmt.Errorf("create temporary REF %s manifest: %w", ref, err)
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
		return fmt.Errorf("write temporary REF %s manifest: %w", ref, err)
	}
	if err := os.Rename(tempPath, s.manifestPath(ref)); err != nil {
		return fmt.Errorf("publish REF %s manifest atomically: %w", ref, err)
	}
	if err := syncDirectory(dir); err != nil {
		return fmt.Errorf("REF %s manifest was published but directory durability failed: %w; inspect the REF before retrying", ref, err)
	}
	return nil
}

func (s *RefStore) ensureRefDirectory(ref string) error {
	if err := ensureDirectoryChain(s.refsDir, ref); err != nil {
		return fmt.Errorf("prepare REF %s manifest directory: %w", ref, err)
	}
	if err := syncDirectoryChain(s.refsDir, s.refDirectory(ref)); err != nil {
		return fmt.Errorf("make REF %s manifest directories durable: %w", ref, err)
	}
	return nil
}

func (s *RefStore) acquireLocks(refs ...string) (func(), error) {
	ordered := append([]string(nil), refs...)
	sort.Strings(ordered)
	releases := make([]func(), 0, len(ordered))
	for i, ref := range ordered {
		if i > 0 && ref == ordered[i-1] {
			continue
		}
		release, err := s.acquireLock(ref)
		if err != nil {
			for j := len(releases) - 1; j >= 0; j-- {
				releases[j]()
			}
			return nil, err
		}
		releases = append(releases, release)
	}
	return func() {
		for i := len(releases) - 1; i >= 0; i-- {
			releases[i]()
		}
	}, nil
}

func (s *RefStore) acquireLock(ref string) (func(), error) {
	if err := ensureDirectoryChain(s.locksDir, ref); err != nil {
		return nil, fmt.Errorf("prepare REF lock directory: %w", err)
	}
	path := s.lockPath(ref)
	lock, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		if errors.Is(err, os.ErrExist) {
			return nil, fmt.Errorf("REF %s is locked; retry after the active operation completes: %w", ref, err)
		}
		return nil, fmt.Errorf("lock REF %s: %w", ref, err)
	}
	if err := lock.Close(); err != nil {
		_ = os.Remove(path)
		return nil, fmt.Errorf("close REF %s lock: %w", ref, err)
	}
	return func() { _ = os.Remove(path) }, nil
}

func (s *RefStore) refDirectory(ref string) string {
	return filepath.Join(s.refsDir, filepath.FromSlash(ref))
}

func (s *RefStore) manifestPath(ref string) string {
	return filepath.Join(s.refDirectory(ref), refManifestFile)
}

func (s *RefStore) lockPath(ref string) string {
	return filepath.Join(s.locksDir, filepath.FromSlash(ref), ".lock")
}

func inspectDirectoryChain(root, ref string) error {
	info, err := os.Lstat(root)
	if err != nil {
		return err
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("%s is not a real directory", root)
	}
	current := root
	for _, component := range splitREF(ref) {
		current = filepath.Join(current, component)
		info, err := os.Lstat(current)
		if err != nil {
			return err
		}
		if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("REF namespace component %s is not a real directory", current)
		}
	}
	return nil
}

func ensureDirectoryChain(root, ref string) error {
	if err := ensureRealDirectory(root); err != nil {
		return err
	}
	current := root
	for _, component := range splitREF(ref) {
		current = filepath.Join(current, component)
		if err := ensureRealDirectory(current); err != nil {
			return err
		}
	}
	return nil
}

func ensureRealDirectory(path string) error {
	info, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		if err := os.Mkdir(path, 0o755); err != nil && !errors.Is(err, os.ErrExist) {
			return err
		}
		info, err = os.Lstat(path)
	}
	if err != nil {
		return err
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("%s is not a real directory", path)
	}
	return nil
}

func splitREF(ref string) []string {
	return strings.Split(ref, "/")
}

func syncMovedDirectories(paths ...string) error {
	seen := make(map[string]struct{}, len(paths))
	for _, path := range paths {
		if _, ok := seen[path]; ok {
			continue
		}
		seen[path] = struct{}{}
		if err := syncDirectory(path); err != nil {
			return err
		}
	}
	return nil
}

func syncDirectoryChain(root, leaf string) error {
	for current := leaf; ; current = filepath.Dir(current) {
		if err := syncDirectory(current); err != nil {
			return err
		}
		if current == root {
			return nil
		}
		if parent := filepath.Dir(current); parent == current {
			return fmt.Errorf("directory %s is outside synchronization root %s", leaf, root)
		}
	}
}

func syncDirectory(path string) error {
	dir, err := os.Open(path)
	if err != nil {
		return err
	}
	defer dir.Close()
	return dir.Sync()
}
