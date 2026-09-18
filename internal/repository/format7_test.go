package repository

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	canonicalv5 "github.com/mako10k/sealgraph/internal/canonical/v5"
	canonicalv7 "github.com/mako10k/sealgraph/internal/canonical/v7"
	"github.com/mako10k/sealgraph/internal/domain"
	domainv5 "github.com/mako10k/sealgraph/internal/domain/v5"
	domainv7 "github.com/mako10k/sealgraph/internal/domain/v7"
)

func openFormat7Fixture(t *testing.T) *Repository {
	t.Helper()
	root := t.TempDir()
	if _, err := InitStandalone(root); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".sealgraph", "config"), []byte(format7ConfigBytes), 0o644); err != nil {
		t.Fatal(err)
	}
	repo, err := OpenStandalone(root)
	if err != nil || repo.Format() != 7 {
		t.Fatalf("open format 7: repo=%v err=%v", repo, err)
	}
	return repo
}

func putFormat7Record(t *testing.T, repo *Repository, data []byte) domain.ObjectID {
	t.Helper()
	id, err := repo.objects.WriteBlob(context.Background(), data)
	if err != nil {
		t.Fatal(err)
	}
	return id
}

func putFormat7Material(t *testing.T, repo *Repository, contentID domain.ObjectID) domain.ObjectID {
	t.Helper()
	data, err := canonicalv5.EncodeMaterial(domainv5.Material{Schema: domainv5.MaterialSchema, Content: contentID, Attachments: []domainv5.Attachment{}})
	if err != nil {
		t.Fatal(err)
	}
	return putFormat7Record(t, repo, data)
}

func putFormat7Seal(t *testing.T, repo *Repository, materialID domain.ObjectID, origin *domain.ObjectID) domain.ObjectID {
	t.Helper()
	p, err := canonicalv7.EncodeProvenance(domainv7.Provenance{Schema: domainv7.ProvenanceSchema, Root: true, CauseLinks: []domainv7.CauseLink{}, Origin: origin})
	if err != nil {
		t.Fatal(err)
	}
	pID := putFormat7Record(t, repo, p)
	s, err := canonicalv7.EncodeSeal(domainv7.Seal{Schema: domainv7.SealSchema, Material: materialID, Provenance: pID})
	if err != nil {
		t.Fatal(err)
	}
	return putFormat7Record(t, repo, s)
}

func format7SealFixture(t *testing.T, repo *Repository, source []byte, start uint64) (domain.ObjectID, domain.ObjectID) {
	t.Helper()
	content := []byte("XYZ")
	contentID := putFormat7Record(t, repo, content)
	sourceID := putFormat7Record(t, repo, source)
	snapshotBytes, err := canonicalv7.EncodeSourceSnapshot(domainv7.SourceSnapshot{Schema: domainv7.SourceSnapshotSchema, SourceKey: "source-A", Content: sourceID})
	if err != nil {
		t.Fatal(err)
	}
	snapshotID := putFormat7Record(t, repo, snapshotBytes)
	mapBytes, err := canonicalv7.EncodeOriginMap(domainv7.OriginMap{Schema: domainv7.OriginMapSchema, Content: contentID, Runs: []domainv7.OriginRun{{Kind: "external", Length: 3, Snapshot: snapshotID, SourceStart: start}}})
	if err != nil {
		t.Fatal(err)
	}
	mapID := putFormat7Record(t, repo, mapBytes)
	materialID := putFormat7Material(t, repo, contentID)
	return putFormat7Seal(t, repo, materialID, &mapID), sourceID
}

func TestFormat7ReadsFullSourceClosure(t *testing.T) {
	repo := openFormat7Fixture(t)
	sealID, sourceID := format7SealFixture(t, repo, []byte("abcXYZtail"), 3)
	ctx := context.Background()
	if err := repo.refs.Update(ctx, "root", nil, &sealID); err != nil {
		t.Fatal(err)
	}
	resolved, err := repo.LoadSeal(ctx, sealID)
	if err != nil || resolved.Provenance.Origin == nil || resolved.ContentBytes != 3 {
		t.Fatalf("resolved=%+v err=%v", resolved, err)
	}
	report, err := repo.Fsck(ctx)
	if err != nil || report.SealsV7 != 1 || report.ProvenancesV3 != 1 {
		t.Fatalf("fsck=%+v err=%v", report, err)
	}
	for _, id := range report.UnreferencedBlobs {
		if id.Equal(sourceID) {
			t.Fatal("full source Blob was outside typed closure")
		}
	}
}

func TestFormat7RejectsCopiedByteMismatch(t *testing.T) {
	repo := openFormat7Fixture(t)
	sealID, _ := format7SealFixture(t, repo, []byte("abcXYZtail"), 0)
	ctx := context.Background()
	if _, err := repo.LoadSeal(ctx, sealID); err == nil {
		t.Fatal("LoadSeal accepted mismatched copied bytes")
	}
	if _, err := repo.Fsck(ctx); err == nil {
		t.Fatal("fsck accepted mismatched copied bytes in detached Seal")
	}
}

func TestFormat7RejectsSourceRangeOutsideFullBlob(t *testing.T) {
	repo := openFormat7Fixture(t)
	sealID, _ := format7SealFixture(t, repo, []byte("abcXYZtail"), 9)
	if _, err := repo.LoadSeal(context.Background(), sealID); err == nil {
		t.Fatal("LoadSeal accepted range beyond full source Blob")
	}
}

func TestFormat7DistinguishesUntracedMapFromNull(t *testing.T) {
	repo := openFormat7Fixture(t)
	contentID := putFormat7Record(t, repo, []byte("XYZ"))
	mapBytes, err := canonicalv7.EncodeOriginMap(domainv7.OriginMap{Schema: domainv7.OriginMapSchema, Content: contentID, Runs: []domainv7.OriginRun{{Kind: "untraced", Length: 3}}})
	if err != nil {
		t.Fatal(err)
	}
	mapID := putFormat7Record(t, repo, mapBytes)
	materialID := putFormat7Material(t, repo, contentID)
	withMap := putFormat7Seal(t, repo, materialID, &mapID)
	withoutMap := putFormat7Seal(t, repo, materialID, nil)
	if withMap.Equal(withoutMap) {
		t.Fatal("distinct origin states have identical SealID")
	}
	ctx := context.Background()
	withResolved, withErr := repo.LoadSeal(ctx, withMap)
	withoutResolved, withoutErr := repo.LoadSeal(ctx, withoutMap)
	if withErr != nil || withoutErr != nil || withResolved.Provenance.Origin == nil || withoutResolved.Provenance.Origin != nil {
		t.Fatalf("with=%+v err=%v without=%+v err=%v", withResolved, withErr, withoutResolved, withoutErr)
	}
}

func TestFormat7EmptyContentHasEmptyRunCoverage(t *testing.T) {
	repo := openFormat7Fixture(t)
	contentID := putFormat7Record(t, repo, nil)
	mapBytes, err := canonicalv7.EncodeOriginMap(domainv7.OriginMap{Schema: domainv7.OriginMapSchema, Content: contentID, Runs: []domainv7.OriginRun{}})
	if err != nil {
		t.Fatal(err)
	}
	mapID := putFormat7Record(t, repo, mapBytes)
	materialID := putFormat7Material(t, repo, contentID)
	sealID := putFormat7Seal(t, repo, materialID, &mapID)
	resolved, err := repo.LoadSeal(context.Background(), sealID)
	if err != nil || resolved.ContentBytes != 0 || resolved.Provenance.Origin == nil {
		t.Fatalf("empty content closure=%+v err=%v", resolved, err)
	}
}

func TestFormat7ReadsHistoricalSealWithoutRewriting(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	if _, err := InitStandalone(root); err != nil {
		t.Fatal(err)
	}
	legacy, err := OpenStandalone(root)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := legacy.Add(ctx, AddOptions{REF: "legacy", Content: []byte("before"), Root: true, RootSet: true, ClearCauseLinks: true}); err != nil {
		t.Fatal(err)
	}
	sealed, err := legacy.Seal(ctx, "legacy")
	if err != nil {
		t.Fatal(err)
	}
	before, err := legacy.objects.ReadObject(ctx, sealed.ID)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".sealgraph", "config"), []byte(format7ConfigBytes), 0o644); err != nil {
		t.Fatal(err)
	}
	successor, err := OpenStandalone(root)
	if err != nil {
		t.Fatal(err)
	}
	resolved, err := successor.LoadSeal(ctx, sealed.ID)
	if err != nil || resolved.Seal.Schema != domainv5.SealSchema || resolved.Provenance.Origin != nil {
		t.Fatalf("historical read=%+v err=%v", resolved, err)
	}
	after, err := successor.objects.ReadObject(ctx, sealed.ID)
	if err != nil || string(after.Data) != string(before.Data) {
		t.Fatalf("historical Seal bytes changed: %v", err)
	}
	if _, err := successor.Fsck(ctx); err != nil {
		t.Fatal(err)
	}
}

func TestFormat7ReadsFormat6AndHistoricalCandidate(t *testing.T) {
	fixture := prepareFormat6MigrationFixture(t)
	ctx := fixture.ctx
	if _, err := fixture.migrated.Add(ctx, AddOptions{REF: "successor", Content: []byte("format-6"), Root: true, RootSet: true, ClearCauseLinks: true}); err != nil {
		t.Fatal(err)
	}
	successorSeal, err := fixture.migrated.Seal(ctx, "successor")
	if err != nil {
		t.Fatal(err)
	}
	beforeCandidate, err := fixture.migrated.candidates.LoadSnapshot("root")
	if err != nil {
		t.Fatal(err)
	}
	beforeSeal, err := fixture.migrated.objects.ReadObject(ctx, fixture.first.ID)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(fixture.root, ".sealgraph", "config"), []byte(format7ConfigBytes), 0o644); err != nil {
		t.Fatal(err)
	}
	repo, err := OpenStandalone(fixture.root)
	if err != nil {
		t.Fatal(err)
	}
	inspection, err := repo.InspectCandidate(ctx, "root")
	if err != nil || inspection.Candidate.Schema != domainv5.CandidateSchema || inspection.Prospective.Seal.Schema != domainv7.SealSchema || inspection.Prospective.Provenance.Origin != nil {
		t.Fatalf("historical Candidate projection=%+v err=%v", inspection, err)
	}
	afterCandidate, err := repo.candidates.LoadSnapshot("root")
	if err != nil || string(afterCandidate.Bytes) != string(beforeCandidate.Bytes) {
		t.Fatalf("historical Candidate bytes changed: %v", err)
	}
	afterSeal, err := repo.objects.ReadObject(ctx, fixture.first.ID)
	if err != nil || string(afterSeal.Data) != string(beforeSeal.Data) {
		t.Fatalf("historical Seal bytes changed: %v", err)
	}
	resolvedV6, err := repo.LoadSeal(ctx, successorSeal.ID)
	if err != nil || resolvedV6.Seal.Schema != "sealgraph/seal/v6" || resolvedV6.Provenance.Origin != nil {
		t.Fatalf("format-6 Seal read=%+v err=%v", resolvedV6, err)
	}
	if _, err := repo.Fsck(ctx); err != nil {
		t.Fatal(err)
	}
}

func TestFormat7RejectsCrossGenerationPair(t *testing.T) {
	repo := openFormat7Fixture(t)
	ctx := context.Background()
	contentID := putFormat7Record(t, repo, []byte("content"))
	materialID := putFormat7Material(t, repo, contentID)
	legacyBytes, err := canonicalv5.EncodeProvenance(domainv5.Provenance{Schema: domainv5.ProvenanceSchema, Root: true, CauseLinks: []domainv5.CauseLink{}})
	if err != nil {
		t.Fatal(err)
	}
	legacyID := putFormat7Record(t, repo, legacyBytes)
	sealBytes, err := canonicalv7.EncodeSeal(domainv7.Seal{Schema: domainv7.SealSchema, Material: materialID, Provenance: legacyID})
	if err != nil {
		t.Fatal(err)
	}
	sealID := putFormat7Record(t, repo, sealBytes)
	if _, err := repo.LoadSeal(ctx, sealID); err == nil {
		t.Fatal("LoadSeal accepted v7 Seal paired with v1 Provenance")
	}
	if _, err := repo.Fsck(ctx); err == nil {
		t.Fatal("fsck accepted v7 Seal paired with v1 Provenance")
	}
}
