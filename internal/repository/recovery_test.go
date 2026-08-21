package repository

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mako10k/sealgraph/internal/recovery"
	"github.com/mako10k/sealgraph/internal/store"
)

func TestRecoverInitialAndExistingSealWithoutDeletingSeals(t *testing.T) {
	_, repo := newFormat4Repository(t)
	ctx := context.Background()
	first := addAndSealRoot(t, repo, "root", "v1")
	inspection := requireRecoveryStatus(t, repo, first.OperationID, "RECOVERABLE")
	if inspection.Kind != "seal" {
		t.Fatalf("inspection=%+v", inspection)
	}
	if _, err := repo.Recover(ctx, first.OperationID); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.refs.Resolve(ctx, "root"); !errors.Is(err, store.ErrRefNotFound) {
		t.Fatalf("initial REF still exists: %v", err)
	}
	if _, err := repo.LoadSeal(ctx, first.ID); err != nil {
		t.Fatalf("recovered-away Seal became invalid: %v", err)
	}
	requireRecoveryStatus(t, repo, first.OperationID, "ALREADY_RECOVERED")

	if err := repo.refs.Update(ctx, "root", nil, &first.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.Add(ctx, AddOptions{REF: "root", Content: []byte("v2"), Root: true}); err != nil {
		t.Fatal(err)
	}
	second, err := repo.Seal(ctx, "root")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := repo.Recover(ctx, second.OperationID); err != nil {
		t.Fatal(err)
	}
	head, err := repo.refs.Resolve(ctx, "root")
	if err != nil || !head.Equal(first.ID) {
		t.Fatalf("restored head=%s err=%v want=%s", head, err, first.ID)
	}
	if _, err := repo.LoadSeal(ctx, second.ID); err != nil {
		t.Fatalf("second Seal was deleted: %v", err)
	}
}

func TestRecoverTagRestoresCompleteManifestAndMoveIsInverseRename(t *testing.T) {
	_, repo := newFormat4Repository(t)
	ctx := context.Background()
	sealed := addAndSealRoot(t, repo, "root", "root")
	tagged, err := repo.CreateTag(ctx, "root", "reviewed")
	if err != nil || tagged.OperationID == "" {
		t.Fatalf("tagged=%+v err=%v", tagged, err)
	}
	if _, err := repo.Recover(ctx, tagged.OperationID); err != nil {
		t.Fatal(err)
	}
	tags, err := repo.Tags(ctx, "root")
	if err != nil || len(tags) != 0 {
		t.Fatalf("tags=%+v err=%v", tags, err)
	}
	head, _ := repo.refs.Resolve(ctx, "root")
	if !head.Equal(sealed.ID) {
		t.Fatalf("tag recovery changed HEAD: %s", head)
	}

	moved, err := repo.MoveREF(ctx, "root", "archive/root")
	if err != nil || moved.OperationID == "" {
		t.Fatalf("moved=%+v err=%v", moved, err)
	}
	if _, err := repo.Recover(ctx, moved.OperationID); err != nil {
		t.Fatal(err)
	}
	head, err = repo.refs.Resolve(ctx, "root")
	if err != nil || !head.Equal(sealed.ID) {
		t.Fatalf("inverse move head=%s err=%v", head, err)
	}
	if _, err := repo.refs.Resolve(ctx, "archive/root"); !errors.Is(err, store.ErrRefNotFound) {
		t.Fatalf("move destination remains: %v", err)
	}
}

func TestRecoverRejectsInterveningManifestAndCorruptLogsDoNotBlockFsck(t *testing.T) {
	dir, repo := newFormat4Repository(t)
	ctx := context.Background()
	first := addAndSealRoot(t, repo, "root", "v1")
	if _, err := repo.CreateTag(ctx, "root", "later"); err != nil {
		t.Fatal(err)
	}
	requireRecoveryStatus(t, repo, first.OperationID, "INTERVENED")
	if _, err := repo.Recover(ctx, first.OperationID); err == nil || !strings.Contains(err.Error(), "INTERVENED") {
		t.Fatalf("intervening mutation recovery error=%v", err)
	}
	logs := filepath.Join(dir, ".sealgraph", "logs", "recovery")
	if err := os.WriteFile(filepath.Join(logs, strings.Repeat("a", 32)+".json"), []byte("corrupt\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("missing", filepath.Join(logs, strings.Repeat("b", 32)+".json")); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(logs, strings.Repeat("c", 32)+".json"), 0o700); err != nil {
		t.Fatal(err)
	}
	inspections, err := repo.RecoveryShow(ctx, "")
	if err != nil {
		t.Fatal(err)
	}
	foundCorrupt := false
	for _, inspection := range inspections {
		foundCorrupt = foundCorrupt || inspection.Status == "CORRUPT"
	}
	if !foundCorrupt {
		t.Fatalf("corrupt journal entry not reported: %+v", inspections)
	}
	if _, err := repo.Fsck(ctx); err != nil {
		t.Fatalf("corrupt local recovery record affected canonical fsck: %v", err)
	}
}

func TestRecoveryClassifiesPreparedCrashStatesFromCurrentBytes(t *testing.T) {
	_, repo := newFormat4Repository(t)
	ctx := context.Background()
	sealed := addAndSealRoot(t, repo, "root", "v1")
	committed, err := repo.recovery.Load(sealed.OperationID)
	if err != nil {
		t.Fatal(err)
	}
	applied, err := repo.recovery.Prepare("seal", committed.Transitions)
	if err != nil {
		t.Fatal(err)
	}
	requireRecoveryStatus(t, repo, applied.ID, "PREPARED_APPLIED_RECOVERABLE")

	refs, err := repo.recoveryRefs()
	if err != nil {
		t.Fatal(err)
	}
	before, err := refs.Snapshot(ctx, "root")
	if err != nil {
		t.Fatal(err)
	}
	after, err := refs.PreviewTag(ctx, "root", "planned", sealed.ID, sealed.ID)
	if err != nil {
		t.Fatal(err)
	}
	notApplied, err := repo.recovery.Prepare("tag", []recovery.Transition{{REF: "root", Before: before, After: after}})
	if err != nil {
		t.Fatal(err)
	}
	requireRecoveryStatus(t, repo, notApplied.ID, "PREPARED_NOT_APPLIED")
}

func TestDropREFIsExactBlockedAndRecoverableWithTags(t *testing.T) {
	_, repo := newFormat4Repository(t)
	ctx := context.Background()
	sealed := addAndSealRoot(t, repo, "root", "v1")
	if _, err := repo.CreateTag(ctx, "root", "reviewed"); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.Add(ctx, AddOptions{REF: "root", Content: []byte("pending"), Root: true}); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.DropREF(ctx, "root"); err == nil || !strings.Contains(err.Error(), "candidate") {
		t.Fatalf("candidate did not block drop: %v", err)
	}
	if err := repo.DiscardCandidate(ctx, "root"); err != nil {
		t.Fatal(err)
	}
	dropped, err := repo.DropREF(ctx, "root")
	if err != nil || dropped.Head != sealed.ID || dropped.Tags != 1 || dropped.OperationID == "" {
		t.Fatalf("dropped=%+v err=%v", dropped, err)
	}
	if _, err := repo.refs.Resolve(ctx, "root"); !errors.Is(err, store.ErrRefNotFound) {
		t.Fatalf("dropped REF resolves: %v", err)
	}
	requireRecoveryStatus(t, repo, dropped.OperationID, "RECOVERABLE")
	if _, err := repo.Recover(ctx, dropped.OperationID); err != nil {
		t.Fatal(err)
	}
	tags, err := repo.Tags(ctx, "root")
	if err != nil || len(tags) != 1 || tags[0].Name != "reviewed" {
		t.Fatalf("restored tags=%+v err=%v", tags, err)
	}
}

func addAndSealRoot(t *testing.T, repo *Repository, ref, content string) SealResult {
	t.Helper()
	ctx := context.Background()
	if _, err := repo.Add(ctx, AddOptions{REF: ref, Content: []byte(content), Root: true}); err != nil {
		t.Fatal(err)
	}
	result, err := repo.Seal(ctx, ref)
	if err != nil {
		t.Fatal(err)
	}
	if result.OperationID == "" {
		t.Fatal("seal omitted recovery operation ID")
	}
	return result
}

func requireRecoveryStatus(t *testing.T, repo *Repository, id, expected string) RecoveryInspection {
	t.Helper()
	inspections, err := repo.RecoveryShow(context.Background(), id)
	if err != nil || len(inspections) != 1 || inspections[0].Status != expected {
		t.Fatalf("recovery %s inspections=%+v err=%v", id, inspections, err)
	}
	return inspections[0]
}
