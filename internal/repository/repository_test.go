package repository

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/mako10k/sealgraph/internal/canonical"
	"github.com/mako10k/sealgraph/internal/domain"
	"github.com/mako10k/sealgraph/internal/history"
)

var fixedTime = time.Date(2026, 8, 14, 0, 0, 0, 0, time.UTC)

func openTestRepository(t *testing.T) *Repository {
	t.Helper()
	dir := t.TempDir()
	if _, err := InitStandalone(dir); err != nil {
		t.Fatal(err)
	}
	repo, err := OpenStandalone(dir, func() time.Time { return fixedTime })
	if err != nil {
		t.Fatal(err)
	}
	return repo
}

func sealRoot(t *testing.T, repo *Repository, ref, content, message string) SealResult {
	t.Helper()
	ctx := context.Background()
	if _, err := repo.Add(ctx, AddOptions{REF: ref, Content: []byte(content), Root: true}); err != nil {
		t.Fatal(err)
	}
	sealed, err := repo.Seal(ctx, ref, message)
	if err != nil {
		t.Fatal(err)
	}
	return sealed
}

func sealDraftRoot(t *testing.T, repo *Repository) SealResult {
	t.Helper()
	ctx := context.Background()
	if _, err := repo.Add(ctx, AddOptions{REF: "ROOT", Content: []byte("provisional root"), Root: true, Draft: true}); err != nil {
		t.Fatal(err)
	}
	sealed, err := repo.Seal(ctx, "ROOT", "provisional root")
	if err != nil {
		t.Fatal(err)
	}
	if !sealed.Payload.Draft {
		t.Fatal("root seal is not draft")
	}
	return sealed
}

func TestRootCanSealAndNonRootNeedsDependency(t *testing.T) {
	repo := openTestRepository(t)
	root := sealRoot(t, repo, "requirements/ROOT-001", "external requirement", "initial root")
	if !root.Payload.Root || root.Payload.Parent != nil {
		t.Fatalf("root payload = %+v", root.Payload)
	}

	ctx := context.Background()
	if _, err := repo.Add(ctx, AddOptions{REF: "design/DESIGN-001", Content: []byte("design")}); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.Seal(ctx, "design/DESIGN-001", "reviewed"); err == nil || !strings.Contains(err.Error(), "requires at least one dependency") {
		t.Fatalf("non-root seal error = %v, want dependency rejection", err)
	}
}

func TestDependencyResolutionHistoricalImmutabilityAndDirectStale(t *testing.T) {
	repo := openTestRepository(t)
	ctx := context.Background()
	rootV1 := sealRoot(t, repo, "requirements/ROOT-001", "requirement v1", "root v1")

	if _, err := repo.Add(ctx, AddOptions{
		REF: "design/api/DESIGN-001", Content: []byte("same design"),
		Dependencies: []Dependency{{REF: "requirements/ROOT-001"}},
	}); err != nil {
		t.Fatal(err)
	}
	designV1, err := repo.Seal(ctx, "design/api/DESIGN-001", "reviewed root v1")
	if err != nil {
		t.Fatal(err)
	}
	if got := designV1.Payload.Links[0].TargetSeal; !got.Equal(rootV1.ID) {
		t.Fatalf("plain --depend-on equivalent resolved %s, want concrete HEAD %s", got, rootV1.ID)
	}

	rootV2 := sealRoot(t, repo, "requirements/ROOT-001", "requirement v2", "root v2")
	if rootV2.ID.Equal(rootV1.ID) {
		t.Fatal("root supersession did not create a new identity")
	}

	oldDesign, err := repo.LoadSeal(ctx, designV1.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !oldDesign.Links[0].TargetSeal.Equal(rootV1.ID) {
		t.Fatalf("upstream supersession mutated old downstream link: %+v", oldDesign.Links[0])
	}

	statuses, err := repo.Status(ctx, "design/api/DESIGN-001")
	if err != nil {
		t.Fatal(err)
	}
	if len(statuses) != 1 || len(statuses[0].StaleDirect) != 1 || !contains(statuses[0].Labels(), "STALE_DIRECT") {
		t.Fatalf("status = %+v, want STALE_DIRECT", statuses)
	}

	if _, err := repo.Add(ctx, AddOptions{
		REF: "review/historical", Content: []byte("historical review"), Draft: true,
		Dependencies: []Dependency{{REF: "requirements/ROOT-001", Revision: rootV1.ID.String()}},
	}); err != nil {
		t.Fatal(err)
	}
	historical, err := repo.Seal(ctx, "review/historical", "draft against v1")
	if err != nil {
		t.Fatalf("explicit historical draft seal: %v", err)
	}
	if !historical.Payload.Links[0].TargetSeal.Equal(rootV1.ID) {
		t.Fatalf("historical link = %s, want %s", historical.Payload.Links[0].TargetSeal, rootV1.ID)
	}
	status, err := repo.Status(ctx, "review/historical")
	if err != nil {
		t.Fatal(err)
	}
	if !contains(status[0].Labels(), "DRAFT") || !contains(status[0].Labels(), "STALE_DIRECT") {
		t.Fatalf("historical draft status = %v", status[0].Labels())
	}
}

func TestNormalHistoricalSealIsRejectedAndRelinkChangesIdentity(t *testing.T) {
	repo := openTestRepository(t)
	ctx := context.Background()
	rootV1 := sealRoot(t, repo, "ROOT", "one", "v1")
	if _, err := repo.Add(ctx, AddOptions{REF: "CHILD", Content: []byte("unchanged"), Dependencies: []Dependency{{REF: "ROOT"}}}); err != nil {
		t.Fatal(err)
	}
	childV1, err := repo.Seal(ctx, "CHILD", "same review message")
	if err != nil {
		t.Fatal(err)
	}
	rootV2 := sealRoot(t, repo, "ROOT", "two", "v2")

	if _, err := repo.Add(ctx, AddOptions{REF: "NORMAL-HISTORY", Content: []byte("x"), Dependencies: []Dependency{{REF: "ROOT", Revision: rootV1.ID.String()}}}); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.Seal(ctx, "NORMAL-HISTORY", "not draft"); err == nil || !strings.Contains(err.Error(), "HEAD-consistent") {
		t.Fatalf("normal historical seal error = %v, want HEAD consistency rejection", err)
	}

	if _, err := repo.Link(ctx, "CHILD", []Dependency{{REF: "ROOT"}}); err != nil {
		t.Fatal(err)
	}
	childV2, err := repo.Seal(ctx, "CHILD", "same review message")
	if err != nil {
		t.Fatal(err)
	}
	if childV2.ID.Equal(childV1.ID) {
		t.Fatal("relink to a new upstream identity did not change downstream seal identity")
	}
	if !childV2.Payload.Links[0].TargetSeal.Equal(rootV2.ID) {
		t.Fatalf("relinked target = %s, want %s", childV2.Payload.Links[0].TargetSeal, rootV2.ID)
	}
	if childV2.Payload.Parent == nil || !childV2.Payload.Parent.Equal(childV1.ID) {
		t.Fatalf("parent = %v, want %s", childV2.Payload.Parent, childV1.ID)
	}
}

func TestNormalSealChecksCompleteDependencyClosure(t *testing.T) {
	repo := openTestRepository(t)
	ctx := context.Background()
	sealRoot(t, repo, "ROOT", "v1", "v1")
	if _, err := repo.Add(ctx, AddOptions{REF: "MIDDLE", Content: []byte("middle"), Dependencies: []Dependency{{REF: "ROOT"}}}); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.Seal(ctx, "MIDDLE", "middle v1"); err != nil {
		t.Fatal(err)
	}
	sealRoot(t, repo, "ROOT", "v2", "v2")
	if _, err := repo.Add(ctx, AddOptions{REF: "LEAF", Content: []byte("leaf"), Dependencies: []Dependency{{REF: "MIDDLE"}}}); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.Seal(ctx, "LEAF", "leaf"); err == nil || !strings.Contains(err.Error(), "HEAD-consistent") {
		t.Fatalf("closure-stale seal error = %v, want HEAD consistency rejection", err)
	}
}

func TestNormalSealRejectsDraftAnywhereInDependencyClosure(t *testing.T) {
	t.Run("direct draft", func(t *testing.T) {
		repo := openTestRepository(t)
		ctx := context.Background()
		draftRoot := sealDraftRoot(t, repo)

		if _, err := repo.Add(ctx, AddOptions{REF: "CHILD", Content: []byte("child"), Dependencies: []Dependency{{REF: "ROOT"}}}); err != nil {
			t.Fatal(err)
		}
		if _, err := repo.Seal(ctx, "CHILD", "normal child"); err == nil || !strings.Contains(err.Error(), "non-draft dependency closure") || !strings.Contains(err.Error(), "ROOT@") {
			t.Fatalf("normal seal error = %v, want direct draft rejection", err)
		}

		if _, err := repo.Add(ctx, AddOptions{REF: "CHILD", Content: []byte("child"), Draft: true}); err != nil {
			t.Fatal(err)
		}
		child, err := repo.Seal(ctx, "CHILD", "provisional child")
		if err != nil {
			t.Fatalf("draft child depending on draft root: %v", err)
		}
		if !child.Payload.Draft || !child.Payload.Links[0].TargetSeal.Equal(draftRoot.ID) {
			t.Fatalf("draft child = %+v", child.Payload)
		}
	})

	t.Run("transitive draft", func(t *testing.T) {
		repo := openTestRepository(t)
		ctx := context.Background()
		draftRoot := sealDraftRoot(t, repo)
		middleContent, err := repo.objects.WriteBlob(ctx, []byte("synthetic middle"))
		if err != nil {
			t.Fatal(err)
		}
		middle := writeTestSealPayload(t, repo, domain.SealPayload{
			Schema: domain.SealSchema, REF: "MIDDLE", Content: domain.ContentRef{Store: domain.NativeStore, Type: domain.BlobType, ID: middleContent},
			Attachments: []domain.Attachment{}, Links: []domain.Link{{TargetREF: "ROOT", TargetSeal: draftRoot.ID}},
			Message: "synthetic legacy normal seal", CreatedAt: "2026-08-14T00:00:01Z",
		})
		if err := repo.refs.Update(ctx, "MIDDLE", nil, &middle); err != nil {
			t.Fatal(err)
		}
		if _, err := repo.Add(ctx, AddOptions{REF: "LEAF", Content: []byte("leaf"), Dependencies: []Dependency{{REF: "MIDDLE"}}}); err != nil {
			t.Fatal(err)
		}
		if _, err := repo.Seal(ctx, "LEAF", "normal leaf"); err == nil || !strings.Contains(err.Error(), "ROOT@") || !strings.Contains(err.Error(), "is draft") {
			t.Fatalf("normal seal error = %v, want transitive draft rejection", err)
		}
	})
}

func TestWriterGuardSerializesSealAndPreservesLaterCandidate(t *testing.T) {
	repo := openTestRepository(t)
	ctx := context.Background()
	sealRoot(t, repo, "ROOT", "root", "root")
	if _, err := repo.Add(ctx, AddOptions{REF: "CHILD", Content: []byte("sealed version"), Dependencies: []Dependency{{REF: "ROOT"}}}); err != nil {
		t.Fatal(err)
	}

	workDir := filepath.Dir(repo.dir)
	clockEntered := make(chan struct{})
	releaseClock := make(chan struct{})
	var clockOnce sync.Once
	sealingRepo, err := OpenStandalone(workDir, func() time.Time {
		clockOnce.Do(func() {
			close(clockEntered)
			<-releaseClock
		})
		return fixedTime
	})
	if err != nil {
		t.Fatal(err)
	}
	editingRepo, err := OpenStandalone(workDir, func() time.Time { return fixedTime })
	if err != nil {
		t.Fatal(err)
	}

	type sealOutcome struct {
		result SealResult
		err    error
	}
	sealDone := make(chan sealOutcome, 1)
	go func() {
		result, err := sealingRepo.Seal(ctx, "CHILD", "seal first version")
		sealDone <- sealOutcome{result: result, err: err}
	}()
	select {
	case <-clockEntered:
	case <-time.After(2 * time.Second):
		close(releaseClock)
		t.Fatal("seal did not reach the guarded publication path")
	}

	type addOutcome struct {
		candidate domain.Candidate
		err       error
	}
	addDone := make(chan addOutcome, 1)
	go func() {
		candidate, err := editingRepo.Add(ctx, AddOptions{REF: "CHILD", Content: []byte("later candidate")})
		addDone <- addOutcome{candidate: candidate, err: err}
	}()
	select {
	case outcome := <-addDone:
		close(releaseClock)
		t.Fatalf("later add completed while seal held the writer guard: %+v", outcome)
	case <-time.After(100 * time.Millisecond):
	}
	close(releaseClock)

	sealed := <-sealDone
	if sealed.err != nil {
		t.Fatal(sealed.err)
	}
	added := <-addDone
	if added.err != nil {
		t.Fatal(added.err)
	}
	if added.candidate.Base == nil || !added.candidate.Base.Equal(sealed.result.ID) {
		t.Fatalf("later candidate base = %v, want published seal %s", added.candidate.Base, sealed.result.ID)
	}
	object, err := repo.objects.ReadObject(ctx, added.candidate.Content.ID)
	if err != nil {
		t.Fatal(err)
	}
	if string(object.Data) != "later candidate" {
		t.Fatalf("later candidate content = %q", object.Data)
	}
	snapshot, err := repo.candidates.LoadSnapshot("CHILD")
	if err != nil {
		t.Fatalf("later candidate was lost: %v", err)
	}
	if !snapshot.Candidate.Content.ID.Equal(added.candidate.Content.ID) {
		t.Fatalf("persisted candidate = %+v, want %+v", snapshot.Candidate, added.candidate)
	}
}

func TestCandidateCleanupRefusesChangedVersion(t *testing.T) {
	repo := openTestRepository(t)
	ctx := context.Background()
	if _, err := repo.Add(ctx, AddOptions{REF: "ROOT", Content: []byte("one"), Root: true}); err != nil {
		t.Fatal(err)
	}
	original, err := repo.candidates.LoadSnapshot("ROOT")
	if err != nil {
		t.Fatal(err)
	}
	changed := original.Candidate
	changed.Draft = true
	if err := repo.candidates.Save(changed); err != nil {
		t.Fatal(err)
	}
	if err := repo.candidates.RemoveIfUnchanged("ROOT", original.Bytes); !errors.Is(err, ErrCandidateChanged) {
		t.Fatalf("cleanup error = %v, want candidate changed", err)
	}
	remaining, err := repo.candidates.Load("ROOT")
	if err != nil || !remaining.Draft {
		t.Fatalf("changed candidate was not retained: %+v, %v", remaining, err)
	}
}

func TestWriterGuardSerializesAcrossProcesses(t *testing.T) {
	if os.Getenv("SEALGRAPH_WRITER_GUARD_HELPER") == "1" {
		runWriterGuardHelper(t)
		return
	}
	lockDir := filepath.Join(t.TempDir(), "locks")
	if err := os.Mkdir(lockDir, 0o755); err != nil {
		t.Fatal(err)
	}
	ready := filepath.Join(t.TempDir(), "ready")
	release := filepath.Join(t.TempDir(), "release")
	command := exec.Command(os.Args[0], "-test.run=^TestWriterGuardSerializesAcrossProcesses$")
	command.Env = append(os.Environ(),
		"SEALGRAPH_WRITER_GUARD_HELPER=1",
		"SEALGRAPH_WRITER_GUARD_DIR="+lockDir,
		"SEALGRAPH_WRITER_GUARD_READY="+ready,
		"SEALGRAPH_WRITER_GUARD_RELEASE="+release,
	)
	if err := command.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = command.Process.Kill() }()
	waitForTestPath(t, ready)

	waitCtx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	if unlock, err := acquireProcessWriter(waitCtx, lockDir); !errors.Is(err, context.DeadlineExceeded) {
		if err == nil {
			_ = unlock()
		}
		t.Fatalf("second process lock error = %v, want context deadline", err)
	}
	if err := os.WriteFile(release, []byte("release"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := command.Wait(); err != nil {
		t.Fatalf("writer guard helper: %v", err)
	}
	unlock, err := acquireProcessWriter(context.Background(), lockDir)
	if err != nil {
		t.Fatalf("lock was not released by helper process: %v", err)
	}
	if err := unlock(); err != nil {
		t.Fatal(err)
	}
}

func runWriterGuardHelper(t *testing.T) {
	unlock, err := acquireProcessWriter(context.Background(), os.Getenv("SEALGRAPH_WRITER_GUARD_DIR"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := unlock(); err != nil {
			t.Error(err)
		}
	}()
	if err := os.WriteFile(os.Getenv("SEALGRAPH_WRITER_GUARD_READY"), []byte("ready"), 0o600); err != nil {
		t.Fatal(err)
	}
	waitForTestPath(t, os.Getenv("SEALGRAPH_WRITER_GUARD_RELEASE"))
}

func waitForTestPath(t *testing.T, path string) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for {
		if _, err := os.Stat(path); err == nil {
			return
		} else if !errors.Is(err, os.ErrNotExist) {
			t.Fatal(err)
		}
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for %s", path)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func TestTransitiveStaleReverseImpactAndSequentialRepair(t *testing.T) {
	repo := openTestRepository(t)
	ctx := context.Background()
	rootV1 := sealRoot(t, repo, "ROOT", "root v1", "root v1")
	if _, err := repo.Add(ctx, AddOptions{REF: "MIDDLE", Content: []byte("middle"), Dependencies: []Dependency{{REF: "ROOT"}}}); err != nil {
		t.Fatal(err)
	}
	middleV1, err := repo.Seal(ctx, "MIDDLE", "middle v1")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := repo.Add(ctx, AddOptions{REF: "LEAF", Content: []byte("leaf"), Dependencies: []Dependency{{REF: "MIDDLE"}}}); err != nil {
		t.Fatal(err)
	}
	leafV1, err := repo.Seal(ctx, "LEAF", "leaf v1")
	if err != nil {
		t.Fatal(err)
	}
	rootV2 := sealRoot(t, repo, "ROOT", "root v2", "root v2")

	middleStatus, err := repo.Status(ctx, "MIDDLE")
	if err != nil {
		t.Fatal(err)
	}
	if !contains(middleStatus[0].Labels(), "STALE_DIRECT") || contains(middleStatus[0].Labels(), "STALE_TRANSITIVE") {
		t.Fatalf("middle labels = %v, want direct stale only", middleStatus[0].Labels())
	}
	leafStatus, err := repo.Status(ctx, "LEAF")
	if err != nil {
		t.Fatal(err)
	}
	if contains(leafStatus[0].Labels(), "STALE_DIRECT") || !contains(leafStatus[0].Labels(), "STALE_TRANSITIVE") {
		t.Fatalf("leaf labels = %v, want transitive stale only", leafStatus[0].Labels())
	}
	path := leafStatus[0].StaleTransitive[0]
	if len(path.Nodes) != 2 || path.Nodes[0].REF != "LEAF" || path.Nodes[1].REF != "MIDDLE" || !path.Link.TargetSeal.Equal(rootV1.ID) || !path.CurrentHead.Equal(rootV2.ID) {
		t.Fatalf("leaf stale path = %+v", path)
	}

	stale, err := repo.Stale(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(stale) != 2 || stale[0].REF != "LEAF" || stale[1].REF != "MIDDLE" {
		t.Fatalf("stale refs = %+v, want LEAF and MIDDLE", stale)
	}
	sourceHead, impacts, err := repo.Impact(ctx, "ROOT")
	if err != nil {
		t.Fatal(err)
	}
	if !sourceHead.Equal(rootV2.ID) || len(impacts) != 2 || impacts[0].REF != "LEAF" || impacts[0].Direct || impacts[1].REF != "MIDDLE" || !impacts[1].Direct {
		t.Fatalf("source=%s impacts=%+v", sourceHead, impacts)
	}

	if _, err := repo.Link(ctx, "MIDDLE", []Dependency{{REF: "ROOT"}}); err != nil {
		t.Fatal(err)
	}
	middleV2, err := repo.Seal(ctx, "MIDDLE", "middle reviewed against root v2")
	if err != nil {
		t.Fatal(err)
	}
	if middleV2.ID.Equal(middleV1.ID) {
		t.Fatal("middle relink did not change seal identity")
	}
	leafStatus, err = repo.Status(ctx, "LEAF")
	if err != nil {
		t.Fatal(err)
	}
	if !contains(leafStatus[0].Labels(), "STALE_DIRECT") || !contains(leafStatus[0].Labels(), "STALE_TRANSITIVE") {
		t.Fatalf("leaf after middle repair labels = %v, want direct and transitive stale until explicit leaf repair", leafStatus[0].Labels())
	}
	if _, err := repo.Link(ctx, "LEAF", []Dependency{{REF: "MIDDLE"}}); err != nil {
		t.Fatal(err)
	}
	leafV2, err := repo.Seal(ctx, "LEAF", "leaf reviewed against middle v2")
	if err != nil {
		t.Fatal(err)
	}
	if leafV2.ID.Equal(leafV1.ID) {
		t.Fatal("leaf relink did not change seal identity")
	}
	leafStatus, err = repo.Status(ctx, "LEAF")
	if err != nil {
		t.Fatal(err)
	}
	if labels := leafStatus[0].Labels(); len(labels) != 1 || labels[0] != "CLEAN" {
		t.Fatalf("leaf after explicit repair labels = %v, want CLEAN", labels)
	}
}

func TestShowExplicitHistoricalSeal(t *testing.T) {
	repo := openTestRepository(t)
	ctx := context.Background()
	v1 := sealRoot(t, repo, "requirements/ROOT", "v1 bytes", "v1")
	sealRoot(t, repo, "requirements/ROOT", "v2 bytes", "v2")
	shown, err := repo.Show(ctx, "requirements/ROOT", v1.ID.String())
	if err != nil {
		t.Fatal(err)
	}
	if string(shown.Content) != "v1 bytes" || !shown.ID.Equal(v1.ID) {
		t.Fatalf("historical show = %+v content=%q", shown, shown.Content)
	}
}

func TestUniquePrefixScopedTagAndLinkMessageResolveToConcreteSeal(t *testing.T) {
	repo := openTestRepository(t)
	ctx := context.Background()
	rootV1 := sealRoot(t, repo, "requirements/ROOT", "v1", "v1")
	if _, err := repo.CreateTag(ctx, "requirements/ROOT", rootV1.ID.Hex[:12], "baseline/1.0"); err != nil {
		t.Fatal(err)
	}
	rootV2 := sealRoot(t, repo, "requirements/ROOT", "v2", "v2")
	if _, err := repo.CreateTag(ctx, "requirements/ROOT", "", "current"); err != nil {
		t.Fatal(err)
	}

	for _, revision := range []string{rootV1.ID.Hex[:12], "baseline/1.0"} {
		resolved, err := repo.ResolveSealID(ctx, "requirements/ROOT", revision)
		if err != nil || !resolved.Equal(rootV1.ID) {
			t.Fatalf("resolve %q = %s, %v, want %s", revision, resolved, err, rootV1.ID)
		}
	}
	resolvedCurrent, err := repo.ResolveSealID(ctx, "requirements/ROOT", "current")
	if err != nil || !resolvedCurrent.Equal(rootV2.ID) {
		t.Fatalf("current tag = %s, %v, want %s", resolvedCurrent, err, rootV2.ID)
	}

	if _, err := repo.Add(ctx, AddOptions{
		REF: "design/api", Content: []byte("design"), Draft: true,
		Dependencies: []Dependency{{REF: "requirements/ROOT", Revision: "baseline/1.0", Message: "Reviewed against the baseline requirement"}},
	}); err != nil {
		t.Fatal(err)
	}
	design, err := repo.Seal(ctx, "design/api", "historical design review")
	if err != nil {
		t.Fatal(err)
	}
	link := design.Payload.Links[0]
	if !link.TargetSeal.Equal(rootV1.ID) || link.Message != "Reviewed against the baseline requirement" {
		t.Fatalf("persisted link = %+v", link)
	}
	if strings.Contains(link.TargetSeal.String(), "baseline") || len(link.TargetSeal.String()) != 64 {
		t.Fatalf("link did not persist a concrete full ID: %s", link.TargetSeal)
	}
	tags, err := repo.ListTags(ctx, "requirements/ROOT")
	if err != nil || len(tags) != 2 || tags[0].Name != "baseline/1.0" || tags[1].Name != "current" {
		t.Fatalf("tags = %+v, %v", tags, err)
	}
}

func TestLogLinkLogAndDiffAcrossRelink(t *testing.T) {
	repo := openTestRepository(t)
	ctx := context.Background()
	rootV1 := sealRoot(t, repo, "ROOT", "root v1", "root v1")
	if _, err := repo.Add(ctx, AddOptions{REF: "design/api", Content: []byte("unchanged design"), Dependencies: []Dependency{{REF: "ROOT"}}}); err != nil {
		t.Fatal(err)
	}
	designV1, err := repo.Seal(ctx, "design/api", "reviewed root v1")
	if err != nil {
		t.Fatal(err)
	}
	rootV2 := sealRoot(t, repo, "ROOT", "root v2", "root v2")
	if _, err := repo.Link(ctx, "design/api", []Dependency{{REF: "ROOT"}}); err != nil {
		t.Fatal(err)
	}
	designV2, err := repo.Seal(ctx, "design/api", "reviewed root v2")
	if err != nil {
		t.Fatal(err)
	}

	entries, err := repo.Log(ctx, "design/api")
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 || !entries[0].ID.Equal(designV2.ID) || !entries[1].ID.Equal(designV1.ID) {
		t.Fatalf("history = %+v", entries)
	}

	linkEntries, err := repo.LinkLog(ctx, "design/api", "ROOT")
	if err != nil {
		t.Fatal(err)
	}
	if len(linkEntries) != 2 || len(linkEntries[0].Changes) != 1 || linkEntries[0].Changes[0].Kind != history.LinkRepoint {
		t.Fatalf("new link history = %+v", linkEntries)
	}
	change := linkEntries[0].Changes[0]
	if !change.BeforeSeal.Equal(rootV1.ID) || !change.AfterSeal.Equal(rootV2.ID) {
		t.Fatalf("repoint = %+v", change)
	}
	if len(linkEntries[1].Changes) != 1 || linkEntries[1].Changes[0].Kind != history.LinkAdd {
		t.Fatalf("initial link history = %+v", linkEntries[1].Changes)
	}

	diff, err := repo.DiffCurrent(ctx, "design/api")
	if err != nil {
		t.Fatal(err)
	}
	if diff.Content.Changed || len(diff.Links) != 1 || diff.Links[0].Kind != history.LinkRepoint || !diff.Message.Changed || !diff.Parent.Changed {
		t.Fatalf("current diff = %+v", diff)
	}
	explicit, err := repo.DiffExact(ctx, "design/api", designV1.ID, designV2.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(explicit, diff) {
		t.Fatalf("explicit diff = %+v, current diff = %+v", explicit, diff)
	}
}

func TestCurrentDiffRejectsInitialSeal(t *testing.T) {
	repo := openTestRepository(t)
	sealRoot(t, repo, "ROOT", "root", "initial")
	if _, err := repo.DiffCurrent(context.Background(), "ROOT"); err == nil || !strings.Contains(err.Error(), "has no parent") {
		t.Fatalf("initial diff error = %v", err)
	}
}

func TestLogRejectsCorruptAndForeignParent(t *testing.T) {
	t.Run("corrupt parent", func(t *testing.T) {
		repo := openTestRepository(t)
		ctx := context.Background()
		root := sealRoot(t, repo, "ROOT", "root", "initial")
		badParent, err := repo.objects.WriteBlob(ctx, []byte("not a canonical seal"))
		if err != nil {
			t.Fatal(err)
		}
		head := writeTestSealPayload(t, repo, domain.SealPayload{
			Schema: domain.SealSchema, REF: "ROOT", Parent: &badParent, Content: root.Payload.Content,
			Attachments: []domain.Attachment{}, Links: []domain.Link{}, Message: "synthetic corrupt ancestry",
			Root: true, CreatedAt: "2026-08-14T00:00:01Z",
		})
		if err := repo.refs.Update(ctx, "ROOT", &root.ID, &head); err != nil {
			t.Fatal(err)
		}
		if _, err := repo.Log(ctx, "ROOT"); err == nil || !strings.Contains(err.Error(), "not a valid canonical seal") || !strings.Contains(err.Error(), "inspect the named seal objects") {
			t.Fatalf("corrupt parent error = %v", err)
		}
	})

	t.Run("foreign parent", func(t *testing.T) {
		repo := openTestRepository(t)
		ctx := context.Background()
		root := sealRoot(t, repo, "ROOT", "root", "initial")
		foreign := sealRoot(t, repo, "OTHER", "other", "other initial")
		head := writeTestSealPayload(t, repo, domain.SealPayload{
			Schema: domain.SealSchema, REF: "ROOT", Parent: &foreign.ID, Content: root.Payload.Content,
			Attachments: []domain.Attachment{}, Links: []domain.Link{}, Message: "synthetic foreign ancestry",
			Root: true, CreatedAt: "2026-08-14T00:00:01Z",
		})
		if err := repo.refs.Update(ctx, "ROOT", &root.ID, &head); err != nil {
			t.Fatal(err)
		}
		if _, err := repo.Log(ctx, "ROOT"); err == nil || !strings.Contains(err.Error(), "belongs to REF OTHER, not ROOT") {
			t.Fatalf("foreign parent error = %v", err)
		}
	})
}

func writeTestSealPayload(t *testing.T, repo *Repository, payload domain.SealPayload) domain.ObjectID {
	t.Helper()
	encoded, err := canonical.EncodeSeal(payload)
	if err != nil {
		t.Fatal(err)
	}
	id, err := repo.objects.WriteBlob(context.Background(), encoded)
	if err != nil {
		t.Fatal(err)
	}
	return id
}

func TestUnsealedDraftCandidateStatus(t *testing.T) {
	repo := openTestRepository(t)
	ctx := context.Background()
	sealRoot(t, repo, "ROOT", "root", "root")
	if _, err := repo.Add(ctx, AddOptions{REF: "REVIEW", Content: []byte("work"), Draft: true, Dependencies: []Dependency{{REF: "ROOT"}}}); err != nil {
		t.Fatal(err)
	}
	statuses, err := repo.Status(ctx, "REVIEW")
	if err != nil {
		t.Fatal(err)
	}
	labels := statuses[0].Labels()
	if !contains(labels, "UNSEALED") || !contains(labels, "DRAFT") {
		t.Fatalf("labels = %v, want UNSEALED and DRAFT", labels)
	}
}

func contains(values []string, wanted string) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}
