package repository

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func traceOwnFixture(t *testing.T, current string, bind bool, path string) (*Repository, string) {
	t.Helper()
	repo := openFormat7Fixture(t)
	root := repo.workDir
	ctx := context.Background()
	if _, err := repo.Add(ctx, AddOptions{REF: "root", Content: []byte("uXYZ"), Root: true, RootSet: true, ClearCauseLinks: true}); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.TraceSet(ctx, TraceSetOptions{REF: "root", Runs: []TraceRunInput{{Kind: "untraced", Length: 1}, {Kind: "external", Length: 3, SourceName: "source", SourceStart: 3}}, Sources: []TraceSourceInput{{Name: "source", SourceKey: "source-A", Content: []byte("abcXYZtail")}}}); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, path), []byte(current), 0o600); err != nil {
		t.Fatal(err)
	}
	if bind {
		if _, err := repo.TraceSourceBind(ctx, "source-A", path); err != nil {
			t.Fatal(err)
		}
	}
	return repo, root
}

func TestTraceCompareOwnRequiresExplicitBaseline(t *testing.T) {
	repo := openFormat7Fixture(t)
	_, err := repo.TraceCompareOwn(context.Background(), TraceCompareOwnOptions{Baseline: TraceOwnBaseline{REF: "root"}})
	if err == nil {
		t.Fatal("missing baseline kind was accepted")
	}
}

func TestTraceCompareOwnRejectsSealWithoutExactID(t *testing.T) {
	repo := openFormat7Fixture(t)
	_, err := repo.TraceCompareOwn(context.Background(), TraceCompareOwnOptions{Baseline: TraceOwnBaseline{Kind: TraceOwnSeal, REF: "root"}})
	if err == nil {
		t.Fatal("Seal baseline without exact ID was accepted")
	}
}

func TestTraceCompareOwnPresentMovedAndRunIndex(t *testing.T) {
	repo, _ := traceOwnFixture(t, "ZZXYZ", true, "source.txt")
	got, err := repo.TraceCompareOwn(context.Background(), TraceCompareOwnOptions{Baseline: TraceOwnBaseline{Kind: TraceOwnCandidate, REF: "root"}})
	if err != nil || len(got.Runs) != 1 {
		t.Fatalf("result=%+v err=%v", got, err)
	}
	run := got.Runs[0]
	if run.RunIndex != 1 || run.Presence != "PRESENT" || run.SelectedMatchStart == nil || *run.SelectedMatchStart != 2 {
		t.Fatalf("run=%+v", run)
	}
}

func TestTraceCompareOwnAbsentAndHeadBaseline(t *testing.T) {
	repo, root := traceOwnFixture(t, "none", true, "source.txt")
	ctx := context.Background()
	got, err := repo.TraceCompareOwn(ctx, TraceCompareOwnOptions{Baseline: TraceOwnBaseline{Kind: TraceOwnCandidate, REF: "root"}})
	if err != nil || len(got.Runs) != 1 || got.Runs[0].Presence != "ABSENT_EXACT" {
		t.Fatalf("candidate result=%+v err=%v", got, err)
	}
	sealed, err := repo.Seal(ctx, "root")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "source.txt"), []byte("XYZ"), 0o600); err != nil {
		t.Fatal(err)
	}
	head, err := repo.TraceCompareOwn(ctx, TraceCompareOwnOptions{Baseline: TraceOwnBaseline{Kind: TraceOwnHead, REF: "root"}})
	if err != nil || head.SealID == nil || !head.SealID.Equal(sealed.ID) || len(head.Runs) != 1 || head.Runs[0].Presence != "PRESENT" {
		t.Fatalf("head result=%+v err=%v", head, err)
	}
	exact, err := repo.TraceCompareOwn(ctx, TraceCompareOwnOptions{Baseline: TraceOwnBaseline{Kind: TraceOwnSeal, SealID: &sealed.ID}})
	if err != nil || exact.SealID == nil || !exact.SealID.Equal(sealed.ID) || len(exact.Runs) != 1 {
		t.Fatalf("exact result=%+v err=%v", exact, err)
	}
}

func TestTraceCompareOwnReadFailuresRemainUndetermined(t *testing.T) {
	for _, tc := range []struct {
		name string
		bind bool
		path string
	}{
		{name: "missing binding", bind: false, path: "source.txt"},
		{name: "read failure", bind: true, path: "missing.txt"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repo, root := traceOwnFixture(t, "unused", tc.bind, tc.path)
			if tc.name == "read failure" {
				if err := os.Remove(filepath.Join(root, tc.path)); err != nil {
					t.Fatal(err)
				}
			}
			got, err := repo.TraceCompareOwn(context.Background(), TraceCompareOwnOptions{Baseline: TraceOwnBaseline{Kind: TraceOwnCandidate, REF: "root"}})
			if err != nil || len(got.Runs) != 1 || got.Runs[0].Presence != "UNDETERMINED" || got.Runs[0].SourceError == "" {
				t.Fatalf("result=%+v err=%v", got, err)
			}
		})
	}
}

func TestTraceCompareOwnCorruptBindingIsStructuralError(t *testing.T) {
	repo, root := traceOwnFixture(t, "XYZ", true, "source.txt")
	path := filepath.Join(root, ".sealgraph", "local", "trace-sources", traceSourceBindingFilename("source-A"))
	if err := os.WriteFile(path, []byte("{}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.TraceCompareOwn(context.Background(), TraceCompareOwnOptions{Baseline: TraceOwnBaseline{Kind: TraceOwnCandidate, REF: "root"}}); err == nil {
		t.Fatal("corrupt binding was converted to a run result")
	}
}
