package repository

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	canonicalv5 "github.com/mako10k/sealgraph/internal/canonical/v5"
	canonicalv6 "github.com/mako10k/sealgraph/internal/canonical/v6"
	domainv5 "github.com/mako10k/sealgraph/internal/domain/v5"
	domainv6 "github.com/mako10k/sealgraph/internal/domain/v6"
)

func TestFormat6ConfigOnlyMigrationAndSuccessorWrite(t *testing.T) {
	fixture := prepareFormat6MigrationFixture(t)
	verifyFormat6Migration(t, fixture)
	verifyFormat6SuccessorWrite(t, fixture)
}

type format6MigrationFixture struct {
	ctx      context.Context
	root     string
	first    SealResult
	before   format6MigrationObservation
	migrated *Repository
}

func prepareFormat6MigrationFixture(t *testing.T) format6MigrationFixture {
	t.Helper()
	ctx := context.Background()
	root := t.TempDir()
	if _, err := InitStandalone(root); err != nil {
		t.Fatal(err)
	}
	repo, err := OpenStandalone(root)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := repo.Add(ctx, AddOptions{REF: "root", Content: []byte("v1"), Root: true, RootSet: true, ClearCauseLinks: true}); err != nil {
		t.Fatal(err)
	}
	first, err := repo.Seal(ctx, "root")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := repo.Add(ctx, AddOptions{REF: "root", Content: []byte("v2")}); err != nil {
		t.Fatal(err)
	}
	before, err := repo.captureFormat6MigrationObservation(ctx)
	if err != nil {
		t.Fatal(err)
	}
	result, err := MigrateRepository5To6(ctx, root)
	if err != nil {
		t.Fatal(err)
	}
	if result.RetainedSealsV5 != 1 || result.RetainedProvenancesV1 != 1 || result.RetainedCandidatesV5 != 1 {
		t.Fatalf("migration result=%+v", result)
	}
	migrated, err := OpenStandalone(root)
	if err != nil {
		t.Fatal(err)
	}
	if migrated.format != 6 {
		t.Fatalf("format=%d", migrated.format)
	}
	return format6MigrationFixture{ctx: ctx, root: root, first: first, before: before, migrated: migrated}
}

func verifyFormat6Migration(t *testing.T, fixture format6MigrationFixture) {
	t.Helper()
	ctx, root, first, before, migrated := fixture.ctx, fixture.root, fixture.first, fixture.before, fixture.migrated
	after, err := migrated.captureFormat6MigrationObservation(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if !equalFormat6MigrationObservations(before, after) {
		t.Fatal("migration changed retained state")
	}
	historical, err := migrated.LoadSeal(ctx, first.ID)
	if err != nil || historical.Seal.Schema != "sealgraph/seal/v5" || historical.Provenance.Schema != "sealgraph/provenance/v1" {
		t.Fatalf("historical=%+v err=%v", historical, err)
	}
	inspection, err := migrated.InspectCandidate(ctx, "root")
	if err != nil || inspection.Candidate.Schema != "sealgraph/candidate/v5" {
		t.Fatalf("candidate=%+v err=%v", inspection.Candidate, err)
	}
	stillV5, err := os.ReadFile(filepath.Join(root, ".sealgraph", "index", "root", ".candidate"))
	if err != nil || string(stillV5) != string(before.candidates["root"]) {
		t.Fatalf("inspection rewrote historical Candidate: err=%v", err)
	}
}

func verifyFormat6SuccessorWrite(t *testing.T, fixture format6MigrationFixture) {
	t.Helper()
	ctx, root, migrated := fixture.ctx, fixture.root, fixture.migrated
	if _, err := migrated.Add(ctx, AddOptions{REF: "root", Content: []byte("v3")}); err != nil {
		t.Fatal(err)
	}
	candidateBytes, err := os.ReadFile(filepath.Join(root, ".sealgraph", "index", "root", ".candidate"))
	if err != nil {
		t.Fatal(err)
	}
	if candidate, err := canonicalCandidateV6(candidateBytes); err != nil || candidate.Schema != domainv6.CandidateSchema {
		t.Fatalf("v6 candidate err=%v", err)
	}
	second, err := migrated.Seal(ctx, "root")
	if err != nil {
		t.Fatal(err)
	}
	if second.Resolved.Seal.Schema != domainv6.SealSchema || second.Resolved.Provenance.Schema != domainv6.ProvenanceSchema {
		t.Fatalf("successor=%+v", second.Resolved)
	}
	report, err := migrated.Fsck(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if report.SealsV5 != 1 || report.SealsV6 != 1 || report.ProvenancesV1 != 1 || report.ProvenancesV2 != 1 || report.Seals != 2 || report.Provenances != 2 {
		t.Fatalf("fsck=%+v", report)
	}
}

func TestFormat6RepositoryStoresMetadataAndRejectsCrossPairing(t *testing.T) {
	ctx, root, repo, rootSeal := prepareRootAndDependent(t, nil)
	dependent, err := repo.Seal(ctx, "dependent")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := MigrateRepository5To6(ctx, root); err != nil {
		t.Fatal(err)
	}
	migrated, _ := OpenStandalone(root)
	observation, graph, err := migrated.buildObservation(ctx, "metadata test")
	if err != nil {
		t.Fatal(err)
	}
	edit, err := migrated.candidateForObservedEdit(ctx, "dependent", observation, graph)
	if err != nil {
		t.Fatal(err)
	}
	edit.Candidate.CauseLinks[0].Metadata = []domainv5.MetadataEntry{{Namespace: "example.test/context", Value: json.RawMessage(`{"role":"input"}`)}}
	if err := migrated.candidates.SaveIfUnchanged(edit.Candidate, edit.Bytes, edit.Exists); err != nil {
		t.Fatal(err)
	}
	published, err := migrated.Seal(ctx, "dependent")
	if err != nil {
		t.Fatal(err)
	}
	if published.ID.Equal(dependent.ID) || len(published.Resolved.Provenance.CauseLinks[0].Metadata) != 1 {
		t.Fatalf("metadata publication=%+v", published)
	}
	if !published.Resolved.Provenance.CauseLinks[0].TargetSeal.Equal(rootSeal.ID) {
		t.Fatal("metadata changed structural Cause target")
	}
	if _, err := migrated.Fsck(ctx); err != nil {
		t.Fatal(err)
	}

	contentID, err := migrated.objects.WriteBlob(ctx, []byte("cross-pair"))
	if err != nil {
		t.Fatal(err)
	}
	material := domainv5.Material{Schema: domainv5.MaterialSchema, Content: contentID, Attachments: []domainv5.Attachment{}}
	materialBytes, _ := canonicalv6.EncodeMaterial(material)
	materialID, _ := migrated.objects.WriteBlob(ctx, materialBytes)
	provenance := domainv5.Provenance{Schema: domainv5.ProvenanceSchema, Root: true, CauseLinks: []domainv5.CauseLink{}}
	provenanceBytes, _ := canonicalv5ForTest(provenance)
	provenanceID, _ := migrated.objects.WriteBlob(ctx, provenanceBytes)
	crossSealBytes, _ := canonicalv6.EncodeSeal(domainv6.Seal{Schema: domainv6.SealSchema, Material: materialID, Provenance: provenanceID})
	crossID, _ := migrated.objects.WriteBlob(ctx, crossSealBytes)
	if err := migrated.refs.Update(ctx, "cross", nil, &crossID); err != nil {
		t.Fatal(err)
	}
	if _, err := migrated.Fsck(ctx); err == nil {
		t.Fatal("fsck accepted Seal v6 paired with Provenance v1")
	}
}

func TestLinkMetadataMutationCanonicalizesIsIdempotentAndRemoves(t *testing.T) {
	ctx, root, repo, rootSeal := prepareLinkMetadataFixture(t)
	schema := "example.test/v1"
	set, err := repo.SetLinkMetadata(ctx, "dependent", "root", "example.test/context", &schema, json.RawMessage(` { "z" : 2, "a" : [true, null] } `))
	if err != nil {
		t.Fatal(err)
	}
	if set.Action != "SET" || set.Before != nil || set.After == nil || string(set.After.Value) != `{"a":[true,null],"z":2}` || set.Candidate.Schema != domainv6.CandidateSchema {
		t.Fatalf("set=%+v", set)
	}
	path := filepath.Join(root, ".sealgraph", "index", "dependent", ".candidate")
	first, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	again, err := repo.SetLinkMetadata(ctx, "dependent", "@"+rootSeal.ID.String(), "example.test/context", &schema, json.RawMessage(`{"a":[true,null],"z":2}`))
	if err != nil || again.Before == nil {
		t.Fatalf("idempotent=%+v err=%v", again, err)
	}
	second, _ := os.ReadFile(path)
	if string(first) != string(second) {
		t.Fatal("same-entry set changed Candidate bytes")
	}
	removed, err := repo.RemoveLinkMetadata(ctx, "dependent", "root", "example.test/context")
	if err != nil || removed.Before == nil || removed.After != nil || len(removed.Candidate.CauseLinks[0].Metadata) != 0 || removed.Candidate.CauseLinks[0].Messages[0] != "kept" {
		t.Fatalf("removed=%+v err=%v", removed, err)
	}
	if _, err := repo.RemoveLinkMetadata(ctx, "dependent", "root", "example.test/context"); err == nil {
		t.Fatal("remove accepted an absent namespace")
	}
}

func prepareLinkMetadataFixture(t *testing.T) (context.Context, string, *Repository, SealResult) {
	t.Helper()
	ctx, root, repo, rootSeal := prepareRootAndDependent(t, []string{"kept"})
	if _, err := MigrateRepository5To6(ctx, root); err != nil {
		t.Fatal(err)
	}
	repo, _ = OpenStandalone(root)
	return ctx, root, repo, rootSeal
}

func prepareRootAndDependent(t *testing.T, messages []string) (context.Context, string, *Repository, SealResult) {
	t.Helper()
	ctx := context.Background()
	root := t.TempDir()
	if _, err := InitStandalone(root); err != nil {
		t.Fatal(err)
	}
	repo, _ := OpenStandalone(root)
	if _, err := repo.Add(ctx, AddOptions{REF: "root", Content: []byte("root"), Root: true, RootSet: true, ClearCauseLinks: true}); err != nil {
		t.Fatal(err)
	}
	rootSeal, err := repo.Seal(ctx, "root")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := repo.Add(ctx, AddOptions{REF: "dependent", Content: []byte("dependent"), RootSet: true, Cause: &CauseInput{Target: "root", Messages: messages}}); err != nil {
		t.Fatal(err)
	}
	return ctx, root, repo, rootSeal
}

func TestFormat6MixedGenerationGraphHistoryAndStaleness(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	if _, err := InitStandalone(root); err != nil {
		t.Fatal(err)
	}
	repo, _ := OpenStandalone(root)
	if _, err := repo.Add(ctx, AddOptions{REF: "root", Content: []byte("root-v1"), Root: true, RootSet: true, ClearCauseLinks: true}); err != nil {
		t.Fatal(err)
	}
	rootV1, err := repo.Seal(ctx, "root")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := repo.Add(ctx, AddOptions{REF: "dependent", Content: []byte("dependent"), RootSet: true, Cause: &CauseInput{Target: "root"}}); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.Seal(ctx, "dependent"); err != nil {
		t.Fatal(err)
	}
	if _, err := MigrateRepository5To6(ctx, root); err != nil {
		t.Fatal(err)
	}
	migrated, _ := OpenStandalone(root)
	if _, err := migrated.Add(ctx, AddOptions{REF: "root", Content: []byte("root-v2")}); err != nil {
		t.Fatal(err)
	}
	rootV2, err := migrated.Seal(ctx, "root")
	if err != nil {
		t.Fatal(err)
	}
	previous := "@" + rootV1.ID.String()
	if _, err := migrated.Add(ctx, AddOptions{REF: "observer", Content: []byte("observer"), RootSet: true, Cause: &CauseInput{Target: "root", Previous: []string{previous}}}); err != nil {
		t.Fatal(err)
	}
	if _, err := migrated.Seal(ctx, "observer"); err != nil {
		t.Fatal(err)
	}
	log, err := migrated.Log(ctx, "root", false, 0)
	if err != nil || len(log.Entries) != 2 || !log.Entries[0].Resolved.ID.Equal(rootV2.ID) || !log.Entries[1].Resolved.ID.Equal(rootV1.ID) {
		t.Fatalf("mixed log=%+v err=%v", log, err)
	}
	stale, _, err := migrated.Stale(ctx, false, false)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, status := range stale {
		if status.REF == "dependent" {
			found = true
		}
	}
	if !found {
		t.Fatalf("historical dependent was not stale: %+v", stale)
	}
}

func TestFormat6MigrationObservationChangeLeavesConfig5(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	if _, err := InitStandalone(root); err != nil {
		t.Fatal(err)
	}
	repo, _ := OpenStandalone(root)
	if _, err := repo.Add(ctx, AddOptions{REF: "root", Content: []byte("before"), Root: true, RootSet: true, ClearCauseLinks: true}); err != nil {
		t.Fatal(err)
	}
	before, err := repo.captureFormat6MigrationObservation(ctx)
	if err != nil {
		t.Fatal(err)
	}
	temp, err := writeFormat6ConfigTemp(repo.dir)
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(temp)
	if _, err := repo.Add(ctx, AddOptions{REF: "root", Content: []byte("after")}); err != nil {
		t.Fatal(err)
	}
	if err := repo.commitFormat6Config(ctx, temp, before); err == nil {
		t.Fatal("migration committed across changed observation")
	}
	config, err := os.ReadFile(filepath.Join(root, ".sealgraph", "config"))
	if err != nil || string(config) != configBytes {
		t.Fatalf("config changed: %q err=%v", config, err)
	}
}

func TestFormat6DirectlyPublishesHistoricalCandidateAsV6(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	if _, err := InitStandalone(root); err != nil {
		t.Fatal(err)
	}
	repo, _ := OpenStandalone(root)
	if _, err := repo.Add(ctx, AddOptions{REF: "root", Content: []byte("candidate-v5"), Root: true, RootSet: true, ClearCauseLinks: true}); err != nil {
		t.Fatal(err)
	}
	candidatePath := filepath.Join(root, ".sealgraph", "index", "root", ".candidate")
	before, err := os.ReadFile(candidatePath)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := MigrateRepository5To6(ctx, root); err != nil {
		t.Fatal(err)
	}
	afterMigration, err := os.ReadFile(candidatePath)
	if err != nil || string(afterMigration) != string(before) {
		t.Fatalf("migration rewrote Candidate: err=%v", err)
	}
	migrated, _ := OpenStandalone(root)
	published, err := migrated.Seal(ctx, "root")
	if err != nil {
		t.Fatal(err)
	}
	if published.Resolved.Seal.Schema != domainv6.SealSchema || published.Resolved.Provenance.Schema != domainv6.ProvenanceSchema {
		t.Fatalf("published=%+v", published)
	}
	if _, err := os.Lstat(candidatePath); !os.IsNotExist(err) {
		t.Fatalf("historical Candidate was not consumed: %v", err)
	}
}

func canonicalCandidateV6(data []byte) (domainv6.Candidate, error) {
	return canonicalv6.DecodeCandidate(data)
}

func canonicalv5ForTest(value domainv5.Provenance) ([]byte, error) {
	return canonicalv5.EncodeProvenance(value)
}
