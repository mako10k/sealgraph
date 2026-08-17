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

func TestPrefixREFsCoexistInEitherCreationOrder(t *testing.T) {
	ctx := context.Background()
	id := ObjectID([]byte("seal"))

	t.Run("leaf first", func(t *testing.T) {
		refs := newRefStoreForTest(t)
		if err := refs.Update(ctx, "design", nil, &id); err != nil {
			t.Fatal(err)
		}
		if err := refs.Update(ctx, "design/api", nil, &id); err != nil {
			t.Fatal(err)
		}
		if got, err := refs.List(ctx); err != nil || len(got) != 2 || got[0] != "design" || got[1] != "design/api" {
			t.Fatalf("REFs = %v err=%v", got, err)
		}
	})

	t.Run("hierarchy first", func(t *testing.T) {
		refs := newRefStoreForTest(t)
		if err := refs.Update(ctx, "design/api", nil, &id); err != nil {
			t.Fatal(err)
		}
		if err := refs.Update(ctx, "design", nil, &id); err != nil {
			t.Fatal(err)
		}
	})
}

func TestRefManifestIsCanonicalAndMoveIsAtomicNoReplace(t *testing.T) {
	ctx := context.Background()
	refs := newRefStoreForTest(t)
	one := ObjectID([]byte("one"))
	two := ObjectID([]byte("two"))
	if err := refs.Update(ctx, "design", nil, &one); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(refs.manifestPath("design"))
	if err != nil {
		t.Fatal(err)
	}
	want := `{"schema":"sealgraph/ref/v1","head":"` + one.String() + `","tags":[]}`
	if string(data) != want {
		t.Fatalf("manifest = %s, want %s", data, want)
	}
	if err := refs.Move(ctx, "design", "design/api"); err != nil {
		t.Fatal(err)
	}
	if _, err := refs.Resolve(ctx, "design"); !errors.Is(err, store.ErrRefNotFound) {
		t.Fatalf("old REF resolve error = %v", err)
	}
	if got, err := refs.Resolve(ctx, "design/api"); err != nil || !got.Equal(one) {
		t.Fatalf("moved REF = %s err=%v", got, err)
	}
	if err := refs.Update(ctx, "occupied", nil, &two); err != nil {
		t.Fatal(err)
	}
	if err := refs.Move(ctx, "design/api", "occupied"); err == nil {
		t.Fatal("move replaced an existing destination")
	}
	if got, err := refs.Resolve(ctx, "occupied"); err != nil || !got.Equal(two) {
		t.Fatalf("occupied destination changed to %s err=%v", got, err)
	}
	if got, err := refs.Resolve(ctx, "design/api"); err != nil || !got.Equal(one) {
		t.Fatalf("failed move consumed source %s err=%v", got, err)
	}
}

func TestRefManifestCanonicalDecoderRejectsVariants(t *testing.T) {
	id := ObjectID([]byte("head"))
	canonical := `{"schema":"sealgraph/ref/v1","head":"` + id.String() + `","tags":[]}`
	if _, err := decodeRefManifest([]byte(canonical)); err != nil {
		t.Fatalf("canonical manifest: %v", err)
	}
	variants := []string{
		canonical + "\n",
		`{"head":"` + id.String() + `","schema":"sealgraph/ref/v1","tags":[]}`,
		`{"schema":"sealgraph/ref/v1","head":"` + id.String() + `","tags":[],"extra":true}`,
		`{"schema":"sealgraph/ref/v1","head":"` + id.String() + `","tags":[{"name":"release","target":"` + id.String() + `"},{"name":"release","target":"` + id.String() + `"}]}`,
	}
	for _, variant := range variants {
		if _, err := decodeRefManifest([]byte(variant)); err == nil {
			t.Fatalf("noncanonical manifest accepted: %s", variant)
		}
	}
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
