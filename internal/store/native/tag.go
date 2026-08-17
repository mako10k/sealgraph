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

type TagStore struct {
	tagsDir  string
	locksDir string
}

func NewTagStore(repositoryDir string) *TagStore {
	return &TagStore{
		tagsDir:  filepath.Join(repositoryDir, "refs", "tags"),
		locksDir: filepath.Join(repositoryDir, "locks", "tags"),
	}
}

func EncodeTagName(name string) (string, error) {
	if err := domain.ValidateTagName(name); err != nil {
		return "", err
	}
	const hex = "0123456789ABCDEF"
	var encoded strings.Builder
	for _, b := range []byte(name) {
		if (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') ||
			(b >= '0' && b <= '9') || b == '-' || b == '_' {
			encoded.WriteByte(b)
			continue
		}
		encoded.WriteByte('%')
		encoded.WriteByte(hex[b>>4])
		encoded.WriteByte(hex[b&0x0f])
	}
	return encoded.String(), nil
}

func DecodeTagName(encoded string) (string, error) {
	data := make([]byte, 0, len(encoded))
	for i := 0; i < len(encoded); {
		b := encoded[i]
		if (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') ||
			(b >= '0' && b <= '9') || b == '-' || b == '_' {
			data = append(data, b)
			i++
			continue
		}
		if b != '%' || i+2 >= len(encoded) {
			return "", fmt.Errorf("invalid encoded TAGNAME %q", encoded)
		}
		hi, okHi := upperHex(encoded[i+1])
		lo, okLo := upperHex(encoded[i+2])
		if !okHi || !okLo {
			return "", fmt.Errorf("invalid encoded TAGNAME %q", encoded)
		}
		data = append(data, hi<<4|lo)
		i += 3
	}
	name := string(data)
	canonical, err := EncodeTagName(name)
	if err != nil || canonical != encoded {
		return "", fmt.Errorf("encoded TAGNAME %q is not canonical", encoded)
	}
	return name, nil
}

func upperHex(b byte) (byte, bool) {
	switch {
	case b >= '0' && b <= '9':
		return b - '0', true
	case b >= 'A' && b <= 'F':
		return b - 'A' + 10, true
	default:
		return 0, false
	}
}

func (s *TagStore) Resolve(ctx context.Context, ref, name string) (domain.ObjectID, error) {
	if err := ctx.Err(); err != nil {
		return domain.ObjectID{}, err
	}
	encoded, err := EncodeTagName(name)
	if err != nil {
		return domain.ObjectID{}, err
	}
	if err := domain.ValidateREF(ref); err != nil {
		return domain.ObjectID{}, err
	}
	data, err := os.ReadFile(s.path(ref, encoded))
	if errors.Is(err, os.ErrNotExist) {
		return domain.ObjectID{}, fmt.Errorf("%w: %s@%s", store.ErrTagNotFound, ref, name)
	}
	if err != nil {
		return domain.ObjectID{}, fmt.Errorf("read tag %s@%s: %w", ref, name, err)
	}
	if len(data) != 65 || data[64] != '\n' {
		return domain.ObjectID{}, fmt.Errorf("tag %s@%s is corrupt; expected one full object ID followed by LF", ref, name)
	}
	id, err := domain.ParseObjectID(string(data[:64]))
	if err != nil {
		return domain.ObjectID{}, fmt.Errorf("tag %s@%s is corrupt: %w", ref, name, err)
	}
	return id, nil
}

// Create writes an immutable tag. The caller validates seal ownership before
// this storage operation.
func (s *TagStore) Create(ctx context.Context, ref, name string, id domain.ObjectID) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := domain.ValidateREF(ref); err != nil {
		return err
	}
	encoded, err := EncodeTagName(name)
	if err != nil {
		return err
	}
	if err := id.ValidateNative(); err != nil {
		return err
	}
	path := s.path(ref, encoded)
	if err := s.validateNamespace(path); err != nil {
		return err
	}
	lockPath := s.lockPath(ref, encoded)
	if err := os.MkdirAll(filepath.Dir(lockPath), 0o755); err != nil {
		return fmt.Errorf("create tag lock directory: %w", err)
	}
	lock, err := os.OpenFile(lockPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return fmt.Errorf("lock tag %s@%s: %w", ref, name, err)
	}
	lock.Close()
	defer os.Remove(lockPath)

	if existing, resolveErr := s.Resolve(ctx, ref, name); resolveErr == nil {
		if existing.Equal(id) {
			return nil
		}
		return fmt.Errorf("%w: tag %s@%s already targets %s and cannot move to %s", store.ErrTagConflict, ref, name, existing, id)
	} else if !errors.Is(resolveErr, store.ErrTagNotFound) {
		return resolveErr
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create tag directory: %w", err)
	}
	temp, err := os.CreateTemp(filepath.Dir(path), ".tmp-tag-")
	if err != nil {
		return fmt.Errorf("create temporary tag: %w", err)
	}
	tempPath := temp.Name()
	defer os.Remove(tempPath)
	if _, err = fmt.Fprintln(temp, id); err == nil {
		err = temp.Sync()
	}
	closeErr := temp.Close()
	if err == nil {
		err = closeErr
	}
	if err != nil {
		return fmt.Errorf("write tag %s@%s: %w", ref, name, err)
	}
	if err := os.Chmod(tempPath, 0o444); err != nil {
		return fmt.Errorf("make tag %s@%s immutable: %w", ref, name, err)
	}
	if err := os.Link(tempPath, path); err != nil {
		if errors.Is(err, os.ErrExist) {
			return fmt.Errorf("%w: tag %s@%s appeared concurrently; retry to inspect its target", store.ErrTagConflict, ref, name)
		}
		return fmt.Errorf("publish tag %s@%s: %w", ref, name, err)
	}
	return nil
}

func (s *TagStore) List(ctx context.Context, ref string) ([]store.Tag, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := domain.ValidateREF(ref); err != nil {
		return nil, err
	}
	dir := filepath.Join(s.tagsDir, filepath.FromSlash(ref))
	entries, err := os.ReadDir(dir)
	if errors.Is(err, os.ErrNotExist) {
		return []store.Tag{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("list tags for %s: %w", ref, err)
	}
	result := make([]store.Tag, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue // namespace for a hierarchical child REF
		}
		if entry.Type()&os.ModeSymlink != 0 || !entry.Type().IsRegular() {
			return nil, fmt.Errorf("tag entry %s is not a regular file", entry.Name())
		}
		name, err := DecodeTagName(entry.Name())
		if err != nil {
			return nil, err
		}
		id, err := s.Resolve(ctx, ref, name)
		if err != nil {
			return nil, err
		}
		result = append(result, store.Tag{Name: name, Seal: id})
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Name < result[j].Name })
	return result, nil
}

// ListAll inventories every physical tag file and attributes it to exactly one
// current REF scope. It rejects orphan and ambiguous paths rather than
// interpreting the collision-prone format-3 tree heuristically.
func (s *TagStore) ListAll(ctx context.Context, refs []string) ([]store.ScopedTag, error) {
	refSet := make(map[string]struct{}, len(refs))
	for _, ref := range refs {
		if err := domain.ValidateREF(ref); err != nil {
			return nil, err
		}
		refSet[ref] = struct{}{}
	}
	result := make([]store.ScopedTag, 0)
	seen := make(map[string]struct{})
	err := filepath.WalkDir(s.tagsDir, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		if path == s.tagsDir {
			if entry.Type()&os.ModeSymlink != 0 || !entry.IsDir() {
				return fmt.Errorf("tag store is not a real directory")
			}
			return nil
		}
		if entry.IsDir() && entry.Type()&os.ModeSymlink == 0 {
			return nil
		}
		if entry.Type()&os.ModeSymlink != 0 || !entry.Type().IsRegular() {
			return fmt.Errorf("tag entry %s is not a regular file", path)
		}
		relative, err := filepath.Rel(s.tagsDir, path)
		if err != nil {
			return err
		}
		relative = filepath.ToSlash(relative)
		if _, conflicts := refSet[relative]; conflicts {
			return fmt.Errorf("tag path %q collides with a hierarchical REF scope", relative)
		}
		var scope, encoded string
		for _, ref := range refs {
			prefix := ref + "/"
			if !strings.HasPrefix(relative, prefix) {
				continue
			}
			candidate := strings.TrimPrefix(relative, prefix)
			if candidate != "" && !strings.Contains(candidate, "/") {
				if scope != "" {
					return fmt.Errorf("tag path %q is attributable to more than one REF", relative)
				}
				scope, encoded = ref, candidate
			}
		}
		if scope == "" {
			return fmt.Errorf("tag path %q has no current REF scope", relative)
		}
		name, err := DecodeTagName(encoded)
		if err != nil {
			return err
		}
		key := scope + "\x00" + name
		if _, exists := seen[key]; exists {
			return fmt.Errorf("duplicate logical tag %s@%s", scope, name)
		}
		id, err := s.Resolve(ctx, scope, name)
		if err != nil {
			return err
		}
		seen[key] = struct{}{}
		result = append(result, store.ScopedTag{REF: scope, Name: name, Seal: id})
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("inventory tags: %w", err)
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].REF != result[j].REF {
			return result[i].REF < result[j].REF
		}
		return result[i].Name < result[j].Name
	})
	return result, nil
}

func (s *TagStore) validateNamespace(path string) error {
	for current := filepath.Dir(path); current != s.tagsDir; current = filepath.Dir(current) {
		info, err := os.Lstat(current)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return err
		}
		if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("tag path conflicts with an existing REF/tag namespace at %s", current)
		}
	}
	if info, err := os.Lstat(path); err == nil && info.IsDir() {
		return fmt.Errorf("tag path conflicts with a hierarchical REF namespace at %s", path)
	} else if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

func (s *TagStore) path(ref, encoded string) string {
	return filepath.Join(s.tagsDir, filepath.FromSlash(ref), encoded)
}

func (s *TagStore) lockPath(ref, encoded string) string {
	return filepath.Join(s.locksDir, filepath.FromSlash(ref), encoded+".lock")
}
