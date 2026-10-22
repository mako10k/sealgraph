package repository

// R: verify historical review admission, immutable readback and observation boundaries.

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	domainv5 "github.com/mako10k/sealgraph/internal/domain/v5"
	"github.com/mako10k/sealgraph/internal/store"
	"strings"
	"testing"

	"github.com/mako10k/sealgraph/internal/domain"
	"github.com/mako10k/sealgraph/internal/historicalreview"
)

func historicalFixture(t *testing.T, transitive bool) (*Repository, SealResult, SealResult) {
	t.Helper()
	ctx := context.Background()
	repo := newV5TestRepository(t)
	old := sealRootV5(t, repo, "premise", "old version")
	if transitive {
		if _, err := repo.Add(ctx, AddOptions{REF: "middle", Content: []byte("middle relies on old"), RootSet: true, Cause: cause("premise")}); err != nil {
			t.Fatal(err)
		}
		if _, err := repo.Seal(ctx, "middle"); err != nil {
			t.Fatal(err)
		}
	}
	next := sealRootV5(t, repo, "premise", "changed version")
	if _, err := repo.Add(ctx, AddOptions{REF: "revision-observer", Content: []byte("explicit revision"), RootSet: true, Cause: cause("premise", "@"+old.ID.String())}); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.Seal(ctx, "revision-observer"); err != nil {
		t.Fatal(err)
	}
	target := "@" + old.ID.String()
	if transitive {
		target = "middle"
	}
	if _, err := repo.Add(ctx, AddOptions{REF: "dependent", Content: []byte("decided reliance"), RootSet: true, Cause: cause(target)}); err != nil {
		t.Fatal(err)
	}
	return repo, old, next
}

func reviewRecordFor(t *testing.T, repo *Repository, ref string) historicalreview.Record {
	t.Helper()
	facts, err := repo.PrepareHistoricalReview(context.Background(), ref)
	if err != nil {
		t.Fatal(err)
	}
	record := historicalreview.Record{Schema: historicalreview.Schema, REF: ref, SubjectSeal: facts.SubjectSeal, OtherHeads: facts.OtherHeads, Entries: []historicalreview.Entry{}}
	for _, target := range facts.Targets {
		record.Entries = append(record.Entries, historicalreview.Entry{TargetSeal: target.TargetSeal, ComparedSeals: target.ComparedSeals, Decision: "retain", Difference: "New premise changes scope; old clause remains the selected contract.", Impact: "Dependent content intentionally implements that exact old clause.", Reason: "Current publication is bounded to the previously selected contract."})
	}
	return record
}

func storeReview(t *testing.T, repo *Repository, record historicalreview.Record) {
	t.Helper()
	data, err := historicalreview.Encode(record)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := repo.SetHistoricalReview(context.Background(), record.REF, data); err != nil {
		t.Fatal(err)
	}
}

func TestHistoricalReviewPublicationKeepsStaleAndOldBytes(t *testing.T) {
	for _, transitive := range []bool{false, true} {
		t.Run(map[bool]string{false: "direct", true: "transitive"}[transitive], func(t *testing.T) {
			ctx := context.Background()
			repo, old, _ := historicalFixture(t, transitive)
			original, err := repo.objects.ReadObject(ctx, old.ID)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := repo.SealReviewed(ctx, "dependent"); err == nil || !strings.Contains(err.Error(), "REVIEW_REQUIRED") {
				t.Fatalf("missing record: %v", err)
			}
			record := reviewRecordFor(t, repo, "dependent")
			storeReview(t, repo, record)
			if _, err := repo.Seal(ctx, "dependent"); err == nil {
				t.Fatal("default strict admission was bypassed")
			}
			result, err := repo.SealReviewed(ctx, "dependent")
			if err != nil {
				t.Fatal(err)
			}
			if result.Resolved.Provenance.Draft {
				t.Fatal("historical reliance became draft")
			}
			inspection, err := repo.InspectHistoricalReview(ctx, "@"+result.ID.String(), false)
			if err != nil {
				t.Fatal(err)
			}
			assertHistoricalReviewReadback(t, inspection, result.ID, record.SubjectSeal)
			observation, graph, err := repo.buildObservation(ctx, "test")
			if err != nil {
				t.Fatal(err)
			}
			if graph.isActiveLeaf(old.ID) {
				t.Fatal("old target became current leaf")
			}
			if !observation.heads["dependent"].Equal(result.ID) {
				t.Fatal("wrong REF publication")
			}
			if graph.isActiveLeaf(old.ID) || len(result.Resolved.Material.Attachments) != 1 {
				t.Fatal("stale/record lost")
			}
			after, err := repo.objects.ReadObject(ctx, old.ID)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(original.Data, after.Data) {
				t.Fatal("historical object changed")
			}
			if _, err := repo.Fsck(ctx); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestHistoricalReviewRejectsUnreviewedAndChangedBindings(t *testing.T) {
	cases := []string{"unreviewed", "content", "cause", "heads", "coverage", "comparison"}
	for _, test := range cases {
		t.Run(test, func(t *testing.T) {
			ctx := context.Background()
			repo, _, next := historicalFixture(t, true)
			record := reviewRecordFor(t, repo, "dependent")
			if test == "unreviewed" {
				record.Entries[0].Decision = "unreviewed"
			}
			if test == "coverage" {
				record.Entries = record.Entries[:1]
			}
			if test == "comparison" {
				record.Entries[0].ComparedSeals = []domain.ObjectID{}
			}
			data, _ := historicalreview.Encode(record)
			before, _ := repo.candidates.LoadSnapshot("dependent")
			_, err := repo.SetHistoricalReview(ctx, "dependent", data)
			if test == "coverage" || test == "comparison" {
				if err == nil {
					t.Fatal("invalid coverage accepted")
				}
				after, _ := repo.candidates.LoadSnapshot("dependent")
				if !bytes.Equal(before.Bytes, after.Bytes) {
					t.Fatal("failed review changed Candidate")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			switch test {
			case "content":
				_, err = repo.Add(ctx, AddOptions{REF: "dependent", Content: []byte("changed dependent")})
			case "cause":
				_, err = repo.Link(ctx, "dependent", *cause("@" + next.ID.String()))
			case "heads":
				sealRootV5(t, repo, "unrelated", "a changed complete observation")
			}
			if err != nil {
				t.Fatal(err)
			}
			if _, err := repo.SealReviewed(ctx, "dependent"); err == nil {
				t.Fatal("invalid review admitted")
			}
			inspection, err := repo.InspectHistoricalReview(ctx, "dependent", true)
			if err != nil {
				t.Fatal(err)
			}
			if inspection.Applicable {
				t.Fatal("changed/unreviewed record displayed applicable")
			}
		})
	}
}

func TestHistoricalReviewDoesNotAdmitDraftClosure(t *testing.T) {
	ctx := context.Background()
	repo := newV5TestRepository(t)
	if _, err := repo.Add(ctx, AddOptions{REF: "provisional", Content: []byte("draft"), Root: true, RootSet: true, ClearCauseLinks: true, Draft: true, DraftSet: true}); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.Seal(ctx, "provisional"); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.Add(ctx, AddOptions{REF: "dependent", Content: []byte("final"), RootSet: true, Cause: cause("provisional")}); err != nil {
		t.Fatal(err)
	}
	record := reviewRecordFor(t, repo, "dependent")
	storeReview(t, repo, record)
	if _, err := repo.SealReviewed(ctx, "dependent"); err == nil || !strings.Contains(err.Error(), "draft Cause") {
		t.Fatalf("draft admission: %v", err)
	}
}

func TestHistoricalReviewCanonicalRecordDoesNotInventJudgments(t *testing.T) {
	repo, _, _ := historicalFixture(t, false)
	facts, err := repo.PrepareHistoricalReview(context.Background(), "dependent")
	if err != nil {
		t.Fatal(err)
	}
	data, _ := json.Marshal(facts)
	if strings.Contains(string(data), "\"decision\"") {
		t.Fatal("preparation fabricated judgment")
	}
}

func TestHistoricalReviewDetachedCauseAndFormatGenerations(t *testing.T) {
	for _, format := range []int{5, 6, 7} {
		t.Run(string(rune('0'+format)), func(t *testing.T) {
			ctx := context.Background()
			dir, repo := newTestRepositoryWithDir(t)
			if format == 6 {
				if _, err := MigrateRepository5To6(ctx, dir); err != nil {
					t.Fatal(err)
				}
			}
			if format == 7 {
				if _, err := MigrateRepositoryTo7(ctx, dir, 5); err != nil {
					t.Fatal(err)
				}
			}
			var err error
			repo, err = OpenStandalone(dir)
			if err != nil {
				t.Fatal(err)
			}
			old := sealRootV5(t, repo, "premise", "historical original")
			sealRootV5(t, repo, "premise", "new, without inferred revision assertion")
			if _, err := repo.Add(ctx, AddOptions{REF: "dependent", Content: []byte("retained detached premise"), RootSet: true, Cause: cause("@" + old.ID.String())}); err != nil {
				t.Fatal(err)
			}
			facts, err := repo.PrepareHistoricalReview(ctx, "dependent")
			if err != nil {
				t.Fatal(err)
			}
			if facts.Targets[0].RevisionState != HistoricalOrDetached || len(facts.Targets[0].ComparedSeals) != 0 {
				t.Fatalf("detached facts=%+v", facts)
			}
			record := reviewRecordFor(t, repo, "dependent")
			storeReview(t, repo, record)
			published, err := repo.SealReviewed(ctx, "dependent")
			if err != nil {
				t.Fatal(err)
			}
			if published.Resolved.Seal.Schema != "sealgraph/seal/v"+string(rune('0'+format)) {
				t.Fatalf("schema=%s", published.Resolved.Seal.Schema)
			}
			statuses, err := repo.Status(ctx, "dependent")
			if err != nil {
				t.Fatal(err)
			}
			if len(statuses) != 1 || !strings.Contains(strings.Join(statuses[0].Labels(), " "), "STALE_DIRECT") {
				t.Fatalf("stale=%+v", statuses)
			}
			if _, err := repo.Fsck(ctx); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestHistoricalReviewSubjectIncludesOriginAndOtherAttachments(t *testing.T) {
	ctx := context.Background()
	repo, _, _ := historicalFixture(t, false)
	record := reviewRecordFor(t, repo, "dependent")
	storeReview(t, repo, record)
	snapshot, err := repo.candidates.LoadSnapshot("dependent")
	if err != nil {
		t.Fatal(err)
	}
	blob, err := repo.objects.WriteBlob(ctx, []byte("supporting immutable evidence"))
	if err != nil {
		t.Fatal(err)
	}
	candidate := snapshot.Candidate
	candidate.Attachments = append(candidate.Attachments, domainv5.Attachment{Name: "other-evidence", MediaType: "text/plain", Blob: blob})
	if err := repo.candidates.SaveIfUnchanged(candidate, snapshot.Bytes, true); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.SealReviewed(ctx, "dependent"); err == nil || !strings.Contains(err.Error(), "REVIEW_SUBJECT_CHANGED") {
		t.Fatalf("other evidence binding=%v", err)
	}
	before, err := repo.prospectiveSeal(withoutHistoricalReview(candidate))
	if err != nil {
		t.Fatal(err)
	}
	repo.format = 7
	candidate.Origin = &blob
	withOrigin, err := repo.prospectiveSeal(withoutHistoricalReview(candidate))
	if err != nil {
		t.Fatal(err)
	}
	candidate.Origin = nil
	withoutOrigin, err := repo.prospectiveSeal(withoutHistoricalReview(candidate))
	if err != nil {
		t.Fatal(err)
	}
	if withOrigin.ID.Equal(withoutOrigin.ID) || before.ID.Equal(withoutOrigin.ID) {
		t.Fatal("origin/format not bound by subject identity")
	}
}

// A non-cooperative concurrent writer changes one exact manifest after review
// admission but before publication. The normal observation guard must still win.
type reviewChangingRefs struct {
	store.RecoveryRefStore
	calls         int
	ref           string
	before, after domain.ObjectID
}

func (s *reviewChangingRefs) List(ctx context.Context) ([]string, error) {
	s.calls++
	if s.calls == 2 {
		if err := s.Update(ctx, s.ref, &s.before, &s.after); err != nil {
			return nil, err
		}
	}
	return s.RecoveryRefStore.List(ctx)
}
func TestHistoricalReviewPublicationRejectsHeadRace(t *testing.T) {
	ctx := context.Background()
	repo, old, next := historicalFixture(t, false)
	storeReview(t, repo, reviewRecordFor(t, repo, "dependent"))
	snapshot, _ := repo.candidates.LoadSnapshot("dependent")
	repo.refs = &reviewChangingRefs{RecoveryRefStore: repo.refs.(store.RecoveryRefStore), ref: "premise", before: next.ID, after: old.ID}
	if _, err := repo.SealReviewed(ctx, "dependent"); err == nil || !strings.Contains(err.Error(), "changed") {
		t.Fatalf("race=%v", err)
	}
	if _, err := repo.refs.Resolve(ctx, "dependent"); !errors.Is(err, store.ErrRefNotFound) {
		t.Fatalf("dependent published despite race: %v", err)
	}
	after, _ := repo.candidates.LoadSnapshot("dependent")
	if !bytes.Equal(snapshot.Bytes, after.Bytes) {
		t.Fatal("race removed/replaced Candidate")
	}
}

func TestHistoricalReviewCannotBypassDestinationCAS(t *testing.T) {
	ctx := context.Background()
	repo, old, _ := historicalFixture(t, false)
	storeReview(t, repo, reviewRecordFor(t, repo, "dependent"))
	snapshot, _ := repo.candidates.LoadSnapshot("dependent")
	// This uncooperative writer publishes the previously absent destination.
	if err := repo.refs.Update(ctx, "dependent", nil, &old.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.SealReviewed(ctx, "dependent"); err == nil || !strings.Contains(err.Error(), "appeared") {
		t.Fatalf("destination CAS bypassed: %v", err)
	}
	head, err := repo.refs.Resolve(ctx, "dependent")
	if err != nil || !head.Equal(old.ID) {
		t.Fatalf("unexpected destination head %s: %v", head, err)
	}
	after, _ := repo.candidates.LoadSnapshot("dependent")
	if !bytes.Equal(snapshot.Bytes, after.Bytes) {
		t.Fatal("CAS failure changed Candidate")
	}
}

func TestHistoricalReviewCannotTransferToAnotherREF(t *testing.T) {
	ctx := context.Background()
	repo, _, _ := historicalFixture(t, false)
	record := reviewRecordFor(t, repo, "dependent")
	record.REF = "another-dependent"
	data, err := historicalreview.Encode(record)
	if err != nil {
		t.Fatal(err)
	}
	before, _ := repo.candidates.LoadSnapshot("dependent")
	if _, err := repo.SetHistoricalReview(ctx, "dependent", data); err == nil || !strings.Contains(err.Error(), "REVIEW_SUBJECT_CHANGED") {
		t.Fatalf("cross-REF review accepted: %v", err)
	}
	after, _ := repo.candidates.LoadSnapshot("dependent")
	if !bytes.Equal(before.Bytes, after.Bytes) {
		t.Fatal("failed cross-REF set changed Candidate")
	}
}

func assertHistoricalReviewReadback(t *testing.T, inspection HistoricalReviewInspection, selected, subject domain.ObjectID) {
	t.Helper()
	if !inspection.Applicable || inspection.Candidate {
		t.Fatalf("published review applicability=%+v", inspection)
	}
	if !inspection.SelectedSeal.Equal(selected) || !inspection.Record.SubjectSeal.Equal(subject) {
		t.Fatalf("published review identities=%+v", inspection)
	}
}
