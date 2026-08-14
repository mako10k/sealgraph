package native

import (
	"bytes"
	"context"
	"os"
	"os/exec"
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

func TestResolvePrefixRequiresRepositoryWideUniqueness(t *testing.T) {
	repositoryDir := t.TempDir()
	if err := os.Mkdir(filepath.Join(repositoryDir, "objects"), 0o755); err != nil {
		t.Fatal(err)
	}
	objects := NewObjectStore(repositoryDir)
	id, err := objects.WriteBlob(context.Background(), []byte("unique payload"))
	if err != nil {
		t.Fatal(err)
	}
	resolved, err := objects.ResolvePrefix(context.Background(), id.Hex[:12])
	if err != nil || !resolved.Equal(id) {
		t.Fatalf("resolve unique prefix = %s, %v, want %s", resolved, err, id)
	}

	fanout := filepath.Join(repositoryDir, "objects", "ab")
	if err := os.Mkdir(fanout, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"cd0" + strings.Repeat("0", 59), "cd1" + strings.Repeat("0", 59)} {
		if err := os.WriteFile(filepath.Join(fanout, name), []byte("fixture"), 0o444); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := objects.ResolvePrefix(context.Background(), "abcd"); err == nil || !strings.Contains(err.Error(), "ambiguous") {
		t.Fatalf("ambiguous prefix error = %v", err)
	}
}

func TestNativeLooseObjectsInteroperateWithGitSHA256(t *testing.T) {
	git, err := exec.LookPath("git")
	if err != nil {
		t.Skip("git is not installed")
	}
	gitDir := filepath.Join(t.TempDir(), "repo.git")
	initCommand := exec.Command(git, "init", "--bare", "--object-format=sha256", gitDir)
	if output, err := initCommand.CombinedOutput(); err != nil {
		t.Skipf("installed Git does not support SHA-256 repositories: %v: %s", err, output)
	}
	payload := []byte("Git ODB conformance\x00with binary\n")
	nativeID := ObjectID(payload)
	hashCommand := exec.Command(git, "--git-dir="+gitDir, "hash-object", "-w", "--stdin")
	hashCommand.Stdin = bytes.NewReader(payload)
	output, err := hashCommand.CombinedOutput()
	if err != nil {
		t.Fatalf("git hash-object: %v: %s", err, output)
	}
	if got := strings.TrimSpace(string(output)); got != nativeID.String() {
		t.Fatalf("git ID = %s, native ID = %s", got, nativeID)
	}
	gitObjects := NewObjectStore(gitDir)
	read, err := gitObjects.ReadObject(context.Background(), nativeID)
	if err != nil || !bytes.Equal(read.Data, payload) {
		t.Fatalf("read Git-produced object = %q, %v", read.Data, err)
	}

	nativeDir := t.TempDir()
	if err := os.Mkdir(filepath.Join(nativeDir, "objects"), 0o755); err != nil {
		t.Fatal(err)
	}
	nativeObjects := NewObjectStore(nativeDir)
	secondID, err := nativeObjects.WriteBlob(context.Background(), []byte("written by sealgraph"))
	if err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(gitDir, "objects", secondID.Hex[:2], secondID.Hex[2:])
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatal(err)
	}
	loose, err := os.ReadFile(nativeObjects.PathForTesting(secondID))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, loose, 0o444); err != nil {
		t.Fatal(err)
	}
	catCommand := exec.Command(git, "--git-dir="+gitDir, "cat-file", "blob", secondID.String())
	catOutput, err := catCommand.CombinedOutput()
	if err != nil || string(catOutput) != "written by sealgraph" {
		t.Fatalf("git cat-file native object = %q, %v", catOutput, err)
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
