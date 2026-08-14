package native

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/mako10k/sealgraph/internal/store"
)

func newRefStoreForTest(t *testing.T) *RefStore {
	t.Helper()
	dir := t.TempDir()
	for _, relative := range []string{filepath.Join("refs", "seals"), "locks"} {
		if err := os.MkdirAll(filepath.Join(dir, relative), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	return NewRefStore(dir)
}

func TestHierarchicalREFAndPrefixConflictsInEitherOrder(t *testing.T) {
	ctx := context.Background()
	id := ObjectID([]byte("seal"))

	t.Run("leaf first", func(t *testing.T) {
		refs := newRefStoreForTest(t)
		if err := refs.Update(ctx, "design", nil, &id); err != nil {
			t.Fatal(err)
		}
		if err := refs.Update(ctx, "design/api", nil, &id); !errors.Is(err, store.ErrPrefixConflict) {
			t.Fatalf("error = %v, want prefix conflict", err)
		}
	})

	t.Run("hierarchy first", func(t *testing.T) {
		refs := newRefStoreForTest(t)
		if err := refs.Update(ctx, "design/api", nil, &id); err != nil {
			t.Fatal(err)
		}
		if err := refs.Update(ctx, "design", nil, &id); !errors.Is(err, store.ErrPrefixConflict) {
			t.Fatalf("error = %v, want prefix conflict", err)
		}
	})
}

func TestRefUpdateCompareAndSwap(t *testing.T) {
	ctx := context.Background()
	refs := newRefStoreForTest(t)
	one := ObjectID([]byte("one"))
	two := ObjectID([]byte("two"))
	wrong := ObjectID([]byte("wrong"))
	if err := refs.Update(ctx, "design/api", nil, &one); err != nil {
		t.Fatal(err)
	}
	if err := refs.Update(ctx, "design/api", &wrong, &two); !errors.Is(err, store.ErrCASMismatch) {
		t.Fatalf("error = %v, want CAS mismatch", err)
	}
	current, err := refs.Resolve(ctx, "design/api")
	if err != nil {
		t.Fatal(err)
	}
	if !current.Equal(one) {
		t.Fatalf("failed CAS changed REF to %s, want %s", current, one)
	}
	if err := refs.Update(ctx, "design/api", &one, &two); err != nil {
		t.Fatal(err)
	}
	current, err = refs.Resolve(ctx, "design/api")
	if err != nil || !current.Equal(two) {
		t.Fatalf("successful CAS current=%s err=%v", current, err)
	}
}
