package repository

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	canonicalv5 "github.com/mako10k/sealgraph/internal/canonical/v5"
	"github.com/mako10k/sealgraph/internal/domain"
	domainv5 "github.com/mako10k/sealgraph/internal/domain/v5"
)

func TestFsckClassifiesTypedAndUnreferencedBlobs(t *testing.T) {
	repo := newV5TestRepository(t)
	sealRootV5(t, repo, "premise", "content")
	extra, err := repo.objects.WriteBlob(context.Background(), []byte("unreferenced"))
	if err != nil {
		t.Fatal(err)
	}
	report, err := repo.Fsck(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if report.Blobs != 5 || report.Seals != 1 || report.Materials != 1 || report.Provenances != 1 || report.ActiveSeals != 1 {
		t.Fatalf("report = %+v", report)
	}
	if len(report.UnreferencedBlobs) != 1 || !report.UnreferencedBlobs[0].Equal(extra) {
		t.Fatalf("unreferenced = %+v", report.UnreferencedBlobs)
	}
}

func TestPhysicalFsckObservationTracksModeButAcceptsStableWritableState(t *testing.T) {
	repo := newV5TestRepository(t)
	sealRootV5(t, repo, "premise", "content")
	before, err := capturePhysicalRepositoryObservation(repo.dir)
	if err != nil {
		t.Fatal(err)
	}
	config := filepath.Join(repo.dir, "config")
	if err := os.Chmod(config, 0o600); err != nil {
		t.Fatal(err)
	}
	after, err := capturePhysicalRepositoryObservation(repo.dir)
	if err != nil {
		t.Fatal(err)
	}
	if equalPhysicalRepositoryObservations(before, after) {
		t.Fatal("physical observation ignored config mode change")
	}
	if _, err := repo.Fsck(context.Background()); err != nil {
		t.Fatalf("stable writable-mode fsck failed: %v", err)
	}
}

func TestFsckRejectsStableInvalidCapturedConfigAndREFLayout(t *testing.T) {
	tests := map[string]func(*testing.T, *Repository){
		"config bytes": func(t *testing.T, repo *Repository) {
			if err := os.WriteFile(filepath.Join(repo.dir, "config"), []byte("repository_format = 500\n"), 0o644); err != nil {
				t.Fatal(err)
			}
		},
		"unexpected refs entry": func(t *testing.T, repo *Repository) {
			if err := os.Mkdir(filepath.Join(repo.dir, "refs", "legacy"), 0o755); err != nil {
				t.Fatal(err)
			}
		},
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			repo := newV5TestRepository(t)
			sealRootV5(t, repo, "premise", "content")
			mutate(t, repo)
			if _, err := repo.Fsck(context.Background()); err == nil || !strings.Contains(err.Error(), "physical repository") {
				t.Fatalf("fsck error = %v", err)
			}
		})
	}
}

func TestFsckSemanticInventoryUsesCapturedObjectBytes(t *testing.T) {
	ctx := context.Background()
	repo := newV5TestRepository(t)
	sealed := sealRootV5(t, repo, "premise", "content")
	id := sealed.ID.String()
	path := filepath.Join(repo.dir, "objects", id[:2], id[2:])
	valid, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("captured corrupt envelope"), 0o644); err != nil {
		t.Fatal(err)
	}
	physical, err := capturePhysicalRepositoryObservation(repo.dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, valid, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, 0o444); err != nil {
		t.Fatal(err)
	}
	if _, err := fsckInventoryFromPhysical(ctx, physical); err == nil || !strings.Contains(err.Error(), "captured object") {
		t.Fatalf("captured-object validation error=%v", err)
	}
	if _, err := repo.Fsck(ctx); err != nil {
		t.Fatalf("restored live repository should be valid: %v", err)
	}
}

func TestPhysicalFileObservationDoesNotFollowSymlink(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "target")
	if err := os.WriteFile(target, []byte("bytes"), 0o600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "link")
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	if file, err := openFileNoFollow(link); err == nil {
		_ = file.Close()
		t.Fatal("physical file open followed a symbolic link")
	}
}

func TestFsckReportsHistoricalSealWithoutCallingItUnreferenced(t *testing.T) {
	repo := newV5TestRepository(t)
	first := sealRootV5(t, repo, "premise", "v1")
	sealRootV5(t, repo, "premise", "v2")
	report, err := repo.Fsck(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, id := range report.HistoricalOrDetachedSeals {
		if id.Equal(first.ID) {
			found = true
		}
	}
	if !found {
		t.Fatalf("historical seals = %+v", report.HistoricalOrDetachedSeals)
	}
	for _, id := range report.UnreferencedBlobs {
		if id.Equal(first.ID) {
			t.Fatal("historical Seal also reported as unreferenced Blob")
		}
	}
}

func TestFsckSealRootedClosureDoesNotReachThroughUnreferencedMaterial(t *testing.T) {
	repo := newV5TestRepository(t)
	sealRootV5(t, repo, "premise", "content")
	content, err := repo.objects.WriteBlob(context.Background(), []byte("detached content"))
	if err != nil {
		t.Fatal(err)
	}
	materialBytes, err := canonicalv5.EncodeMaterial(domainv5.Material{
		Schema: domainv5.MaterialSchema, Content: content, Attachments: []domainv5.Attachment{},
	})
	if err != nil {
		t.Fatal(err)
	}
	material, err := repo.objects.WriteBlob(context.Background(), materialBytes)
	if err != nil {
		t.Fatal(err)
	}
	report, err := repo.Fsck(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]bool{content.String(): true, material.String(): true}
	for _, id := range report.UnreferencedBlobs {
		delete(want, id.String())
	}
	if len(want) != 0 {
		t.Fatalf("Seal-rooted unreferenced closure omitted IDs: %v; report=%+v", want, report.UnreferencedBlobs)
	}
}

func TestFsckRejectsInvalidReferencesFromUnreferencedTypedBlobs(t *testing.T) {
	repo := newV5TestRepository(t)
	sealRootV5(t, repo, "premise", "content")
	missing := domain.ObjectID{Hex: strings.Repeat("a", 64)}
	materialBytes, err := canonicalv5.EncodeMaterial(domainv5.Material{
		Schema: domainv5.MaterialSchema, Content: missing, Attachments: []domainv5.Attachment{},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := repo.objects.WriteBlob(context.Background(), materialBytes); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.Fsck(context.Background()); err == nil || !strings.Contains(err.Error(), "references missing content Blob") {
		t.Fatalf("fsck error = %v", err)
	}
}
