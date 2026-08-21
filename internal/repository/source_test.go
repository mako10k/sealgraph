package repository

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestLocalSourceLifecycleIsExplicitAndNonCanonical(t *testing.T) {
	dir, repo := newFormat4Repository(t)
	writeSourceFile(t, dir, "docs/one.md", "one")
	writeSourceFile(t, dir, "docs/two.md", "two")
	ctx := context.Background()

	if _, err := repo.SourceBind(ctx, "spec", "docs/one.md"); err != nil {
		t.Fatal(err)
	}
	binding, err := repo.SourceShow("spec")
	if err != nil || binding.Path != "docs/one.md" {
		t.Fatalf("binding=%+v err=%v", binding, err)
	}
	if _, err := repo.SourceBind(ctx, "spec", "docs/two.md"); err == nil {
		t.Fatal("bind silently retargeted an existing source")
	}
	if _, err := repo.SourceRebind(ctx, "spec", "wrong.md", "docs/two.md"); err == nil {
		t.Fatal("rebind ignored expected old path")
	}
	if _, err := repo.SourceRebind(ctx, "spec", "docs/one.md", "docs/two.md"); err != nil {
		t.Fatal(err)
	}
	bindings, err := repo.SourceList()
	if err != nil || len(bindings) != 1 || bindings[0].Path != "docs/two.md" {
		t.Fatalf("bindings=%+v err=%v", bindings, err)
	}
	if _, err := repo.SourceUnbind(ctx, "spec", "docs/one.md"); err == nil {
		t.Fatal("unbind ignored expected old path")
	}
	if _, err := repo.SourceUnbind(ctx, "spec", "docs/two.md"); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.SourceShow("spec"); !errors.Is(err, ErrSourceNotFound) {
		t.Fatalf("show after unbind error=%v", err)
	}
}

func TestContentlessAddPreservesSemanticsAndDoesNotSilentlyFallback(t *testing.T) {
	dir, repo := newFormat4Repository(t)
	writeSourceFile(t, dir, "spec.md", "v1")
	ctx := context.Background()
	first, err := repo.AddLocalSource(ctx, LocalSourceAddOptions{REF: "spec.md", BindSource: true, PreserveSemantics: true, Root: true, RootSet: true})
	if err != nil || !first.Candidate.Root || first.SourceMode != "initial-ref-path" || first.SourceBinding != "BOUND" {
		t.Fatalf("first=%+v err=%v", first, err)
	}
	if _, err := repo.Seal(ctx, "spec.md"); err != nil {
		t.Fatal(err)
	}
	writeSourceFile(t, dir, "spec.md", "v2")
	second, err := repo.AddLocalSource(ctx, LocalSourceAddOptions{REF: "spec.md", PreserveSemantics: true})
	if err != nil || !second.Candidate.Root || second.SourceMode != "bound-source" {
		t.Fatalf("second=%+v err=%v", second, err)
	}
	if _, err := repo.SourceUnbind(ctx, "spec.md", "spec.md"); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.AddLocalSource(ctx, LocalSourceAddOptions{REF: "spec.md", PreserveSemantics: true}); err == nil {
		t.Fatal("existing REF silently fell back to REF-as-path")
	}
}

func TestExplicitFileCannotDisagreeWithExistingBinding(t *testing.T) {
	dir, repo := newFormat4Repository(t)
	writeSourceFile(t, dir, "one.md", "one")
	writeSourceFile(t, dir, "two.md", "two")
	ctx := context.Background()
	if _, err := repo.SourceBind(ctx, "spec", "one.md"); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.AddLocalSource(ctx, LocalSourceAddOptions{REF: "spec", Path: "two.md", PreserveSemantics: false, Root: true, RootSet: true}); err == nil {
		t.Fatal("explicit file silently disagreed with existing binding")
	}
	if _, err := repo.candidates.Load("spec"); !errors.Is(err, ErrCandidateNotFound) {
		t.Fatalf("disagreement created candidate: %v", err)
	}
}

func TestStatusReportsWorkfileAgainstCandidateThenHead(t *testing.T) {
	dir, repo := newFormat4Repository(t)
	writeSourceFile(t, dir, "spec.md", "v1")
	ctx := context.Background()
	if _, err := repo.AddLocalSource(ctx, LocalSourceAddOptions{REF: "spec.md", BindSource: true, PreserveSemantics: true, Root: true, RootSet: true}); err != nil {
		t.Fatal(err)
	}
	statuses, err := repo.Status(ctx, "spec.md")
	if err != nil || statuses[0].Source == nil || statuses[0].Source.Relation != "WORKFILE_MATCHES_CANDIDATE" {
		t.Fatalf("candidate status=%+v err=%v", statuses, err)
	}
	if _, err := repo.Seal(ctx, "spec.md"); err != nil {
		t.Fatal(err)
	}
	statuses, err = repo.Status(ctx, "spec.md")
	if err != nil || statuses[0].Source.Relation != "WORKFILE_MATCHES_HEAD" {
		t.Fatalf("head status=%+v err=%v", statuses, err)
	}
	writeSourceFile(t, dir, "spec.md", "v2")
	statuses, err = repo.Status(ctx, "spec.md")
	if err != nil || statuses[0].Source.Relation != "WORKFILE_DIFFERS_FROM_HEAD" {
		t.Fatalf("modified status=%+v err=%v", statuses, err)
	}
}

func writeSourceFile(t *testing.T, root, relative, content string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(relative))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}
