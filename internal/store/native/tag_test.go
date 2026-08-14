package native

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
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

func TestTagStoreIsImmutableIdempotentAndScoped(t *testing.T) {
	repositoryDir := t.TempDir()
	for _, relative := range []string{filepath.Join("refs", "tags"), "locks"} {
		if err := os.MkdirAll(filepath.Join(repositoryDir, relative), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	tags := NewTagStore(repositoryDir)
	one := ObjectID([]byte("one"))
	two := ObjectID([]byte("two"))
	ctx := context.Background()
	if err := tags.Create(ctx, "design/api", "reviewed/1.0", one); err != nil {
		t.Fatal(err)
	}
	if err := tags.Create(ctx, "design/api", "reviewed/1.0", one); err != nil {
		t.Fatalf("idempotent create: %v", err)
	}
	if err := tags.Create(ctx, "design/api", "reviewed/1.0", two); !errors.Is(err, store.ErrTagConflict) {
		t.Fatalf("retarget error = %v, want immutable conflict", err)
	}
	resolved, err := tags.Resolve(ctx, "design/api", "reviewed/1.0")
	if err != nil || !resolved.Equal(one) {
		t.Fatalf("resolved = %s, %v, want %s", resolved, err, one)
	}
	if err := tags.Create(ctx, "other", "reviewed/1.0", two); err != nil {
		t.Fatal(err)
	}
	listed, err := tags.List(ctx, "design/api")
	if err != nil || len(listed) != 1 || listed[0].Name != "reviewed/1.0" || !listed[0].Seal.Equal(one) {
		t.Fatalf("listed = %+v, %v", listed, err)
	}
}

func TestTagStoreRejectsPrefixREFTagNamespaceConflict(t *testing.T) {
	repositoryDir := t.TempDir()
	for _, relative := range []string{filepath.Join("refs", "tags"), "locks"} {
		if err := os.MkdirAll(filepath.Join(repositoryDir, relative), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	tags := NewTagStore(repositoryDir)
	id := ObjectID([]byte("seal"))
	ctx := context.Background()
	if err := tags.Create(ctx, "design/api", "reviewed", id); err != nil {
		t.Fatal(err)
	}
	if err := tags.Create(ctx, "design", "api", id); err == nil || !strings.Contains(err.Error(), "conflicts") {
		t.Fatalf("prefix tag conflict error = %v", err)
	}
}
