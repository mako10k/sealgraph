package native

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/mako10k/sealgraph/internal/store"
)

func TestTagNameEncodingIsInjectiveAndEscapesSlash(t *testing.T) {
	tests := map[string]string{
		"release-1":       "release-1",
		"release/1":       "release%2F1",
		"v1.0":            "v1%2E0",
		"レビュー":            "%E3%83%AC%E3%83%93%E3%83%A5%E3%83%BC",
		"literal%2Fvalue": "literal%252Fvalue",
	}
	for raw, want := range tests {
		encoded, err := EncodeTagName(raw)
		if err != nil {
			t.Fatalf("EncodeTagName(%q): %v", raw, err)
		}
		if encoded != want {
			t.Errorf("EncodeTagName(%q) = %q, want %q", raw, encoded, want)
		}
		decoded, err := DecodeTagName(encoded)
		if err != nil || decoded != raw {
			t.Errorf("DecodeTagName(%q) = %q, %v, want %q", encoded, decoded, err, raw)
		}
	}
}

func newTagStoresForTest(t *testing.T) (*RefStore, *TagStore) {
	t.Helper()
	repositoryDir := t.TempDir()
	for _, relative := range []string{filepath.Join("refs", "seals"), "locks"} {
		if err := os.MkdirAll(filepath.Join(repositoryDir, relative), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	return NewRefStore(repositoryDir), NewTagStore(repositoryDir)
}

func TestPrefixREFAndTagNamesCoexistInEitherCreationOrder(t *testing.T) {
	ctx := context.Background()
	id := ObjectID([]byte("target"))

	t.Run("tag first", func(t *testing.T) {
		refs, tags := newTagStoresForTest(t)
		if err := refs.Update(ctx, "design", nil, &id); err != nil {
			t.Fatal(err)
		}
		if err := tags.Create(ctx, "design", "api", id, id); err != nil {
			t.Fatal(err)
		}
		if err := refs.Update(ctx, "design/api", nil, &id); err != nil {
			t.Fatal(err)
		}
		if got, err := tags.Resolve(ctx, "design", "api"); err != nil || !got.Equal(id) {
			t.Fatalf("prefix tag = %s err=%v", got, err)
		}
	})

	t.Run("child first", func(t *testing.T) {
		refs, tags := newTagStoresForTest(t)
		if err := refs.Update(ctx, "design/api", nil, &id); err != nil {
			t.Fatal(err)
		}
		if err := refs.Update(ctx, "design", nil, &id); err != nil {
			t.Fatal(err)
		}
		if err := tags.Create(ctx, "design", "api", id, id); err != nil {
			t.Fatal(err)
		}
		if err := tags.Create(ctx, "design/api", "reviewed", id, id); err != nil {
			t.Fatal(err)
		}
		prefixTags, prefixErr := tags.List(ctx, "design")
		childTags, childErr := tags.List(ctx, "design/api")
		if prefixErr != nil || childErr != nil || len(prefixTags) != 1 || len(childTags) != 1 {
			t.Fatalf("prefix tags=%+v err=%v child tags=%+v err=%v", prefixTags, prefixErr, childTags, childErr)
		}
	})
}

func TestTagStoreIsImmutableIdempotentAndMovesWithREF(t *testing.T) {
	refs, tags := newTagStoresForTest(t)
	one := ObjectID([]byte("one"))
	two := ObjectID([]byte("two"))
	ctx := context.Background()
	if err := refs.Update(ctx, "design/api", nil, &one); err != nil {
		t.Fatal(err)
	}
	if err := tags.Create(ctx, "design/api", "reviewed/1.0", one, one); err != nil {
		t.Fatal(err)
	}
	if err := tags.Create(ctx, "design/api", "reviewed/1.0", one, one); err != nil {
		t.Fatalf("idempotent create: %v", err)
	}
	if err := tags.Create(ctx, "design/api", "reviewed/1.0", two, one); !errors.Is(err, store.ErrTagConflict) {
		t.Fatalf("retarget error = %v, want immutable conflict", err)
	}
	if err := refs.Update(ctx, "design/api", &one, &two); err != nil {
		t.Fatal(err)
	}
	if preserved, err := tags.Resolve(ctx, "design/api", "reviewed/1.0"); err != nil || !preserved.Equal(one) {
		t.Fatalf("HEAD update lost tag: target=%s err=%v", preserved, err)
	}
	if err := refs.Move(ctx, "design/api", "renamed/api"); err != nil {
		t.Fatal(err)
	}
	if _, err := tags.Resolve(ctx, "design/api", "reviewed/1.0"); !errors.Is(err, store.ErrRefNotFound) {
		t.Fatalf("old tag scope error = %v", err)
	}
	resolved, err := tags.Resolve(ctx, "renamed/api", "reviewed/1.0")
	if err != nil || !resolved.Equal(one) {
		t.Fatalf("moved tag = %s err=%v", resolved, err)
	}
	listed, err := tags.List(ctx, "renamed/api")
	if err != nil || len(listed) != 1 || listed[0].Name != "reviewed/1.0" || !listed[0].Seal.Equal(one) {
		t.Fatalf("listed = %+v, %v", listed, err)
	}
}

func TestRefManifestRejectsNoncanonicalBytesAndSymlink(t *testing.T) {
	ctx := context.Background()
	t.Run("trailing LF", func(t *testing.T) {
		refs, _ := newTagStoresForTest(t)
		id := ObjectID([]byte("head"))
		if err := refs.Update(ctx, "design", nil, &id); err != nil {
			t.Fatal(err)
		}
		path := refs.manifestPath("design")
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, append(data, '\n'), 0o600); err != nil {
			t.Fatal(err)
		}
		if _, err := refs.Resolve(ctx, "design"); err == nil {
			t.Fatal("noncanonical manifest was accepted")
		}
	})

	t.Run("symlink", func(t *testing.T) {
		refs, _ := newTagStoresForTest(t)
		dir := refs.refDirectory("design")
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink("outside", refs.manifestPath("design")); err != nil {
			t.Fatal(err)
		}
		if _, err := refs.Resolve(ctx, "design"); err == nil {
			t.Fatal("symbolic-link manifest was accepted")
		}
	})
}

func TestTagCreateRequiresUnchangedObservedHead(t *testing.T) {
	refs, tags := newTagStoresForTest(t)
	one := ObjectID([]byte("one"))
	two := ObjectID([]byte("two"))
	ctx := context.Background()
	if err := refs.Update(ctx, "design", nil, &one); err != nil {
		t.Fatal(err)
	}
	if err := refs.Update(ctx, "design", &one, &two); err != nil {
		t.Fatal(err)
	}
	if err := tags.Create(ctx, "design", "reviewed", one, one); !errors.Is(err, store.ErrCASMismatch) {
		t.Fatalf("tag stale-head error = %v", err)
	}
	if listed, err := tags.List(ctx, "design"); err != nil || len(listed) != 0 {
		t.Fatalf("stale tag creation changed manifest: %+v err=%v", listed, err)
	}
}
