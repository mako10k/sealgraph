package native

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestObjectIDUsesGitBlobEnvelopeSHA256(t *testing.T) {
	id := ObjectID([]byte("hello"))
	const expected = "8aec4e4876f854f688d0ebfc8f37598f38e5fd6903cccc850ca36591175aeb60"
	if id.Hex != expected {
		t.Fatalf("ObjectID(hello) = %s, want %s", id.Hex, expected)
	}
}

func TestWriteReadBlobAndRejectHashMismatch(t *testing.T) {
	repositoryDir := t.TempDir()
	if err := os.Mkdir(filepath.Join(repositoryDir, "objects"), 0o755); err != nil {
		t.Fatal(err)
	}
	objects := NewObjectStore(repositoryDir)
	ctx := context.Background()
	one, err := objects.WriteBlob(ctx, []byte("one"))
	if err != nil {
		t.Fatal(err)
	}
	two, err := objects.WriteBlob(ctx, []byte("two"))
	if err != nil {
		t.Fatal(err)
	}
	read, err := objects.ReadObject(ctx, one)
	if err != nil {
		t.Fatal(err)
	}
	if string(read.Data) != "one" {
		t.Fatalf("read data = %q, want one", read.Data)
	}

	otherBytes, err := os.ReadFile(objects.PathForTesting(two))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(objects.PathForTesting(one), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(objects.PathForTesting(one), otherBytes, 0o444); err != nil {
		t.Fatal(err)
	}
	if _, err := objects.ReadObject(ctx, one); err == nil || !strings.Contains(err.Error(), "hash mismatch") {
		t.Fatalf("corrupt object read error = %v, want hash mismatch", err)
	}
	if _, err := objects.WriteBlob(ctx, []byte("one")); err == nil || !strings.Contains(err.Error(), "corrupt") {
		t.Fatalf("write over corrupt immutable object error = %v, want corruption error", err)
	}
}

func TestReadRejectsTrailingCompressedData(t *testing.T) {
	repositoryDir := t.TempDir()
	if err := os.Mkdir(filepath.Join(repositoryDir, "objects"), 0o755); err != nil {
		t.Fatal(err)
	}
	objects := NewObjectStore(repositoryDir)
	id, err := objects.WriteBlob(context.Background(), []byte("payload"))
	if err != nil {
		t.Fatal(err)
	}
	path := objects.PathForTesting(id)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, append(data, []byte("trailing")...), 0o444); err != nil {
		t.Fatal(err)
	}
	if _, err := objects.ReadObject(context.Background(), id); err == nil || !strings.Contains(err.Error(), "trailing") {
		t.Fatalf("trailing data error = %v, want rejection", err)
	}
}
