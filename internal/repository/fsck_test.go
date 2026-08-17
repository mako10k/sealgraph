package repository

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mako10k/sealgraph/internal/canonical"
	"github.com/mako10k/sealgraph/internal/domain"
)

func TestFsckInventoriesCleanAndUnreferencedObjectsWithoutMutation(t *testing.T) {
	_, repo := newFormat4Repository(t)
	sealed := sealRoot(t, repo, "root", []byte("root"))
	unreferenced, err := repo.objects.WriteBlob(context.Background(), []byte("unreferenced"))
	if err != nil {
		t.Fatal(err)
	}
	report, err := repo.Fsck(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if report.REFs != 1 || report.ActiveSeals != 1 || report.Seals != 1 || report.MaterialObjects != 2 {
		t.Fatalf("unexpected report: %+v", report)
	}
	if len(report.HistoricalOrDetachedSeals) != 0 || len(report.UnreferencedObjects) != 1 || !report.UnreferencedObjects[0].Equal(unreferenced) {
		t.Fatalf("unexpected inventory classification: %+v", report)
	}
	if shown, err := repo.Show(context.Background(), "root"); err != nil || !shown.ID.Equal(sealed.ID) {
		t.Fatalf("fsck changed repository: show=%+v err=%v", shown, err)
	}
}

func TestFsckRejectsCorruptAndMissingObjects(t *testing.T) {
	t.Run("corrupt", func(t *testing.T) {
		_, repo := newFormat4Repository(t)
		sealed := sealRoot(t, repo, "root", []byte("root"))
		path := filepath.Join(repo.dir, "objects", sealed.ID.Hex[:2], sealed.ID.Hex[2:])
		if err := os.Chmod(path, 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("corrupt"), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := repo.Fsck(context.Background()); err == nil || !strings.Contains(err.Error(), sealed.ID.String()) {
			t.Fatalf("corrupt fsck error=%v", err)
		}
	})
	t.Run("missing material", func(t *testing.T) {
		_, repo := newFormat4Repository(t)
		sealed := sealRoot(t, repo, "root", []byte("root"))
		id := sealed.Payload.Content.ID
		if err := os.Remove(filepath.Join(repo.dir, "objects", id.Hex[:2], id.Hex[2:])); err != nil {
			t.Fatal(err)
		}
		if _, err := repo.Fsck(context.Background()); err == nil || !strings.Contains(err.Error(), "missing") {
			t.Fatalf("missing fsck error=%v", err)
		}
	})
}

func TestFsckReportsDetachedCanonicalSeal(t *testing.T) {
	_, repo := newFormat4Repository(t)
	sealRoot(t, repo, "root", []byte("root"))
	content, err := repo.objects.WriteBlob(context.Background(), []byte("detached"))
	if err != nil {
		t.Fatal(err)
	}
	payload := domain.SealPayload{Schema: domain.SealSchema, Content: domain.ContentRef{Store: domain.NativeStore, Type: domain.BlobType, ID: content}, Attachments: []domain.Attachment{}, Links: []domain.Link{}, Root: true}
	encoded, err := canonical.EncodeSeal(payload)
	if err != nil {
		t.Fatal(err)
	}
	detached, err := repo.objects.WriteBlob(context.Background(), encoded)
	if err != nil {
		t.Fatal(err)
	}
	report, err := repo.Fsck(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(report.HistoricalOrDetachedSeals) != 1 || !report.HistoricalOrDetachedSeals[0].Equal(detached) {
		t.Fatalf("detached inventory=%+v", report.HistoricalOrDetachedSeals)
	}
}
