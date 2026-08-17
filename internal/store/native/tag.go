package native

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/mako10k/sealgraph/internal/domain"
	"github.com/mako10k/sealgraph/internal/store"
)

type TagStore struct{ refs *RefStore }

func NewTagStore(repositoryDir string) *TagStore {
	return &TagStore{refs: NewRefStore(repositoryDir)}
}

func EncodeTagName(name string) (string, error) {
	if err := domain.ValidateTagName(name); err != nil {
		return "", err
	}
	const hex = "0123456789ABCDEF"
	var encoded strings.Builder
	for _, b := range []byte(name) {
		if isLiteralTagByte(b) {
			encoded.WriteByte(b)
			continue
		}
		encoded.WriteByte('%')
		encoded.WriteByte(hex[b>>4])
		encoded.WriteByte(hex[b&0x0f])
	}
	return encoded.String(), nil
}

func isLiteralTagByte(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') ||
		(b >= '0' && b <= '9') || b == '-' || b == '_'
}

func DecodeTagName(encoded string) (string, error) {
	data := make([]byte, 0, len(encoded))
	for i := 0; i < len(encoded); {
		if isLiteralTagByte(encoded[i]) {
			data = append(data, encoded[i])
			i++
			continue
		}
		decoded, next, err := decodeTagEscape(encoded, i)
		if err != nil {
			return "", err
		}
		data = append(data, decoded)
		i = next
	}
	name := string(data)
	canonical, err := EncodeTagName(name)
	if err != nil || canonical != encoded {
		return "", fmt.Errorf("encoded TAGNAME %q is not canonical", encoded)
	}
	return name, nil
}

func decodeTagEscape(encoded string, index int) (byte, int, error) {
	if encoded[index] != '%' || index+2 >= len(encoded) {
		return 0, 0, fmt.Errorf("invalid encoded TAGNAME %q", encoded)
	}
	hi, okHi := upperHex(encoded[index+1])
	lo, okLo := upperHex(encoded[index+2])
	if !okHi || !okLo {
		return 0, 0, fmt.Errorf("invalid encoded TAGNAME %q", encoded)
	}
	return hi<<4 | lo, index + 3, nil
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
	if err := domain.ValidateTagName(name); err != nil {
		return domain.ObjectID{}, err
	}
	manifest, err := s.refs.loadManifest(ctx, ref)
	if err != nil {
		return domain.ObjectID{}, err
	}
	index := sort.Search(len(manifest.Tags), func(i int) bool { return manifest.Tags[i].Name >= name })
	if index == len(manifest.Tags) || manifest.Tags[index].Name != name {
		return domain.ObjectID{}, fmt.Errorf("%w: %s@%s", store.ErrTagNotFound, ref, name)
	}
	return manifest.Tags[index].Seal, nil
}

// Create adds one immutable binding while requiring the caller's observed REF
// head to remain current. The caller validates that id decodes as a Seal.
func (s *TagStore) Create(ctx context.Context, ref, name string, id, expectedHead domain.ObjectID) error {
	if err := validateTagCreate(ctx, ref, name, id, expectedHead); err != nil {
		return err
	}
	release, err := s.refs.acquireLocks(ref)
	if err != nil {
		return err
	}
	defer release()
	manifest, err := s.refs.loadManifest(ctx, ref)
	if err != nil {
		return err
	}
	if !manifest.Head.Equal(expectedHead) {
		return fmt.Errorf("%w for %s tag creation: expected head %s, found %s; resolve the selector again", store.ErrCASMismatch, ref, expectedHead, manifest.Head)
	}
	for _, existing := range manifest.Tags {
		if existing.Name != name {
			continue
		}
		if existing.Seal.Equal(id) {
			return nil
		}
		return fmt.Errorf("%w: tag %s@%s already targets %s and cannot move to %s", store.ErrTagConflict, ref, name, existing.Seal, id)
	}
	manifest.Tags = append(manifest.Tags, store.Tag{Name: name, Seal: id})
	return s.refs.writeManifest(ref, manifest)
}

func validateTagCreate(ctx context.Context, ref, name string, id, expectedHead domain.ObjectID) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := domain.ValidateREF(ref); err != nil {
		return err
	}
	if err := domain.ValidateTagName(name); err != nil {
		return err
	}
	if err := id.ValidateNative(); err != nil {
		return fmt.Errorf("invalid tag target: %w", err)
	}
	if err := expectedHead.ValidateNative(); err != nil {
		return fmt.Errorf("invalid expected REF head: %w", err)
	}
	return nil
}

func (s *TagStore) List(ctx context.Context, ref string) ([]store.Tag, error) {
	manifest, err := s.refs.loadManifest(ctx, ref)
	if err != nil {
		return nil, err
	}
	return append([]store.Tag(nil), manifest.Tags...), nil
}
