package repository

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func newTraceSourceBindingTestRepo(t *testing.T) (*Repository, string) {
	t.Helper()
	root := t.TempDir()
	if _, err := InitStandalone(root); err != nil {
		t.Fatal(err)
	}
	repo, err := OpenStandalone(root)
	if err != nil {
		t.Fatal(err)
	}
	return repo, root
}

func writeTraceSourceTestFile(t *testing.T, root, name, content string) string {
	t.Helper()
	path := filepath.Join(root, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return strings.ReplaceAll(filepath.Join(name), string(filepath.Separator), "/")
}

func TestTraceSourceBindingLifecycleAndCanonicalStorage(t *testing.T) {
	repo, root := newTraceSourceBindingTestRepo(t)
	firstPath := writeTraceSourceTestFile(t, root, "docs/first.md", "first")
	secondPath := writeTraceSourceTestFile(t, root, "docs/second.md", "second")
	ctx := context.Background()

	binding, err := repo.TraceSourceBind(ctx, "manual-A", firstPath)
	if err != nil || binding != (TraceSourceBinding{SourceKey: "manual-A", Path: firstPath}) {
		t.Fatalf("bind=%+v err=%v", binding, err)
	}
	file := filepath.Join(root, ".sealgraph", "local", "trace-sources", traceSourceBindingFilename("manual-A"))
	data, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	want := `{"schema":"sealgraph/trace-source-binding/v1","source_key":"manual-A","path":"docs/first.md"}` + "\n"
	if string(data) != want {
		t.Fatalf("stored bytes=%q want %q", data, want)
	}

	unchanged, err := repo.TraceSourceBind(ctx, "manual-A", firstPath)
	if err != nil || unchanged != binding {
		t.Fatalf("idempotent bind=%+v err=%v", unchanged, err)
	}
	if _, err := repo.TraceSourceBind(ctx, "manual-A", secondPath); err == nil {
		t.Fatal("bind with a different path succeeded")
	}
	if got, err := repo.TraceSourceShow("manual-A"); err != nil || got != binding {
		t.Fatalf("show=%+v err=%v", got, err)
	}

	if _, err := repo.TraceSourceRebind(ctx, "manual-A", secondPath, secondPath); err == nil {
		t.Fatal("rebind with an incorrect expected old path succeeded")
	}
	updated, err := repo.TraceSourceRebind(ctx, "manual-A", firstPath, secondPath)
	if err != nil || updated.Path != secondPath {
		t.Fatalf("rebind=%+v err=%v", updated, err)
	}
	if _, err := repo.TraceSourceUnbind(ctx, "manual-A", firstPath); err == nil {
		t.Fatal("unbind with an incorrect expected old path succeeded")
	}
	removed, err := repo.TraceSourceUnbind(ctx, "manual-A", secondPath)
	if err != nil || removed != updated {
		t.Fatalf("unbind=%+v err=%v", removed, err)
	}
	if _, err := repo.TraceSourceShow("manual-A"); !errors.Is(err, ErrTraceSourceNotFound) {
		t.Fatalf("show after unbind err=%v", err)
	}
}

func TestTraceSourceBindingListHashAndCanonicalValidation(t *testing.T) {
	repo, root := newTraceSourceBindingTestRepo(t)
	path := writeTraceSourceTestFile(t, root, "source.txt", "source")
	ctx := context.Background()
	if _, err := repo.TraceSourceBind(ctx, "z", path); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.TraceSourceBind(ctx, "a", path); err != nil {
		t.Fatal(err)
	}
	list, err := repo.TraceSourceList()
	if err != nil || len(list) != 2 || list[0].SourceKey != "a" || list[1].SourceKey != "z" {
		t.Fatalf("list=%+v err=%v", list, err)
	}

	bad := filepath.Join(root, ".sealgraph", "local", "trace-sources", traceSourceBindingFilename("a"))
	if err := os.WriteFile(bad, []byte(`{"schema":"sealgraph/trace-source-binding/v1","source_key":"z","path":"source.txt"}`+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.TraceSourceShow("a"); err == nil {
		t.Fatal("source_key hash mismatch was accepted")
	}
	if _, err := repo.TraceSourceList(); err == nil {
		t.Fatal("corrupt binding was listed")
	}
}

func TestTraceSourceBindingRejectsUnsafePathsAndMissingKeys(t *testing.T) {
	repo, root := newTraceSourceBindingTestRepo(t)
	path := writeTraceSourceTestFile(t, root, "source.txt", "source")
	ctx := context.Background()
	for _, key := range []string{"", string([]byte{0xff})} {
		if _, err := repo.TraceSourceBind(ctx, key, path); err == nil {
			t.Fatalf("invalid key %q was accepted", key)
		}
	}
	for _, unsafe := range []string{"../source.txt", ".sealgraph/config", "/tmp/source.txt", "source\\file.txt"} {
		if _, err := repo.TraceSourceBind(ctx, "key-"+unsafe, unsafe); err == nil {
			t.Fatalf("unsafe path %q was accepted", unsafe)
		}
	}
}
