package repository

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mako10k/sealgraph/internal/domain"
)

func correspondenceFixture(t *testing.T) (*Repository, StoredTraceSource, string) {
	t.Helper()
	repo := openFormat7Fixture(t)
	ctx := context.Background()
	if _, err := repo.Add(ctx, AddOptions{REF: "root", Content: []byte("abc"), Root: true, RootSet: true, ClearCauseLinks: true}); err != nil {
		t.Fatal(err)
	}
	result, err := repo.TraceSet(ctx, TraceSetOptions{REF: "root", Runs: []TraceRunInput{{Kind: "external", Length: 3, SourceName: "source", SourceStart: 0}}, Sources: []TraceSourceInput{{Name: "source", SourceKey: "source-A", Content: []byte("abc")}}})
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(repo.workDir, "source.txt")
	if err := os.WriteFile(path, []byte("abcXYZ"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.TraceSourceBind(ctx, "source-A", "source.txt"); err != nil {
		t.Fatal(err)
	}
	return repo, result.StoredSources[0], path
}

func TestTraceCorrespondencePutIsCanonicalIdempotentAndListsID(t *testing.T) {
	repo, source, _ := correspondenceFixture(t)
	current := domain.ComputeNativeBlobID([]byte("abcXYZ"))
	options := TraceCorrespondencePutOptions{SourceSnapshot: source.SnapshotID, SourceStart: 0, Length: 3, CurrentBlob: current, CurrentRanges: []TraceCorrespondenceRange{{Start: 0, Length: 3}, {Start: 3, Length: 3}}, Reason: "manual", DeclaredAt: "2026-09-18T08:00:00Z"}
	first, err := repo.TraceCorrespondencePut(context.Background(), options)
	if err != nil || !first.Changed || first.ID == "" {
		t.Fatalf("first=%+v err=%v", first, err)
	}
	second, err := repo.TraceCorrespondencePut(context.Background(), options)
	if err != nil || second.Changed || second.ID != first.ID {
		t.Fatalf("second=%+v err=%v", second, err)
	}
	got, err := repo.TraceCorrespondenceShow(first.ID)
	if err != nil || got.SourceSnapshot != source.SnapshotID || len(got.CurrentRanges) != 2 {
		t.Fatalf("show=%+v err=%v", got, err)
	}
	list, err := repo.TraceCorrespondenceList()
	if err != nil || len(list) != 1 || list[0].ID != first.ID || list[0].Record.Reason != "manual" {
		t.Fatalf("list=%+v err=%v", list, err)
	}
	removed, err := repo.TraceCorrespondenceRemove(context.Background(), first.ID)
	if err != nil || !removed {
		t.Fatalf("remove=%t err=%v", removed, err)
	}
	if _, err := repo.TraceCorrespondenceShow(first.ID); err == nil {
		t.Fatal("removed correspondence remained readable")
	}
}

func TestTraceCorrespondenceRejectsCurrentIdentityAndInvalidRanges(t *testing.T) {
	repo, source, _ := correspondenceFixture(t)
	base := TraceCorrespondencePutOptions{SourceSnapshot: source.SnapshotID, SourceStart: 0, Length: 3, CurrentBlob: domain.ComputeNativeBlobID([]byte("wrong")), CurrentRanges: []TraceCorrespondenceRange{{Start: 0, Length: 3}}, Reason: "manual", DeclaredAt: "2026-09-18T08:00:00Z"}
	if _, err := repo.TraceCorrespondencePut(context.Background(), base); err == nil || !strings.Contains(err.Error(), "does not match") {
		t.Fatalf("identity error=%v", err)
	}
	base.CurrentBlob = domain.ComputeNativeBlobID([]byte("abcXYZ"))
	base.CurrentRanges = []TraceCorrespondenceRange{{Start: 4, Length: 3}}
	if _, err := repo.TraceCorrespondencePut(context.Background(), base); err == nil || !strings.Contains(err.Error(), "outside") {
		t.Fatalf("range error=%v", err)
	}
	base.CurrentRanges = []TraceCorrespondenceRange{{Start: 0, Length: 3}, {Start: 2, Length: 2}}
	if _, err := repo.TraceCorrespondencePut(context.Background(), base); err == nil || !strings.Contains(err.Error(), "overlap") {
		t.Fatalf("overlap error=%v", err)
	}
}

func TestDecodeTraceCorrespondenceRejectsDuplicateAndNonCanonicalMembers(t *testing.T) {
	repo, source, _ := correspondenceFixture(t)
	current := domain.ComputeNativeBlobID([]byte("abcXYZ"))
	put, err := repo.TraceCorrespondencePut(context.Background(), TraceCorrespondencePutOptions{SourceSnapshot: source.SnapshotID, SourceStart: 0, Length: 3, CurrentBlob: current, CurrentRanges: []TraceCorrespondenceRange{{Start: 0, Length: 3}}, Reason: "manual", DeclaredAt: "2026-09-18T08:00:00Z"})
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(traceCorrespondenceRoot(repo.dir), put.ID+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := decodeTraceCorrespondence(append(append([]byte(nil), data[:len(data)-1]...), []byte(`,"schema":"sealgraph/trace-correspondence/v1"}`)...)); err == nil {
		t.Fatal("duplicate member accepted")
	}
	if err := os.WriteFile(path, append(data, '\n'), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.TraceCorrespondenceShow(put.ID); err == nil {
		t.Fatal("noncanonical LF accepted")
	}
}

func TestParseTraceCorrespondenceRejectsUnknownMembersRecursively(t *testing.T) {
	_, source, _ := correspondenceFixture(t)
	current := domain.ComputeNativeBlobID([]byte("abcXYZ"))
	valid := TraceCorrespondence{Schema: traceCorrespondenceSchema, SourceSnapshot: source.SnapshotID, SourceStart: 0, Length: 3, CurrentBlob: current, CurrentRanges: []TraceCorrespondenceRange{{Start: 0, Length: 3}}, Reason: "manual", DeclaredAt: "2026-09-18T08:00:00Z"}
	encoded, err := TraceCorrespondenceCanonicalJSON(valid)
	if err != nil {
		t.Fatal(err)
	}
	unknownTop := append(append([]byte(nil), encoded[:len(encoded)-1]...), []byte(`,"extra":true}`)...)
	if _, err := ParseTraceCorrespondenceInput(unknownTop); err == nil {
		t.Fatal("unknown top-level member accepted")
	}
	unknownRange := strings.Replace(string(encoded), `{"start":0,"length":3}`, `{"start":0,"length":3,"extra":true}`, 1)
	if _, err := ParseTraceCorrespondenceInput([]byte(unknownRange)); err == nil {
		t.Fatal("unknown range member accepted")
	}
	duplicateRange := strings.Replace(string(encoded), `{"start":0,"length":3}`, `{"start":0,"length":3,"start":0}`, 1)
	if _, err := ParseTraceCorrespondenceInput([]byte(duplicateRange)); err == nil {
		t.Fatal("duplicate range member accepted")
	}
	invalidTime := strings.Replace(string(encoded), "2026-09-18T08:00:00Z", "2026-09-18T08:00:00.123Z", 1)
	if _, err := ParseTraceCorrespondenceInput([]byte(invalidTime)); err == nil {
		t.Fatal("fractional declared_at accepted")
	}
}
