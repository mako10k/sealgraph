package format4extract

import (
	"bytes"
	"compress/zlib"
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mako10k/sealgraph/internal/canonical"
	canonicalv5 "github.com/mako10k/sealgraph/internal/canonical/v5"
	"github.com/mako10k/sealgraph/internal/domain"
	"github.com/mako10k/sealgraph/internal/migration"
	"github.com/mako10k/sealgraph/internal/store/native"
)

func TestExtractProducesCanonicalDocumentWithoutSourceMutation(t *testing.T) {
	dir, contentID, sealID, excludedID := makeFormat4Root(t)
	before := snapshotTree(t, filepath.Join(dir, ".sealgraph"))
	result, err := Extract(context.Background(), dir)
	if err != nil {
		t.Fatal(err)
	}
	after := snapshotTree(t, filepath.Join(dir, ".sealgraph"))
	if !mapsEqual(before, after) {
		t.Fatal("read-only extraction changed source tree")
	}
	if len(result.Warnings) != 0 || !bytes.HasSuffix(result.Document, []byte("\n")) {
		t.Fatalf("warnings=%v document=%q", result.Warnings, result.Document)
	}
	decoded, err := migration.DecodeUniversalBlobV1(result.Document)
	if err != nil {
		t.Fatal(err)
	}
	if len(decoded.Objects) != 1 || !decoded.Objects[0].ID.Equal(contentID) || len(decoded.Seals) != 1 || !decoded.Seals[0].ID.Equal(sealID) || len(decoded.ExcludedObjects) != 1 || !decoded.ExcludedObjects[0].Equal(excludedID) {
		t.Fatalf("unexpected migration inventory: %+v", decoded)
	}
	if _, err := migration.ProjectUniversalBlobV1(decoded); err != nil {
		t.Fatalf("importer projection rejected extractor output: %v", err)
	}
	again, err := Extract(context.Background(), dir)
	if err != nil || !bytes.Equal(again.Document, result.Document) {
		t.Fatalf("deterministic extraction err=%v equal=%t", err, bytes.Equal(again.Document, result.Document))
	}
}

func TestExtractRejectsCandidateAndChangedObservationWithoutDocument(t *testing.T) {
	dir, _, _, _ := makeFormat4Root(t)
	candidatePath := filepath.Join(dir, ".sealgraph", "index", "ROOT", ".candidate")
	if err := os.MkdirAll(filepath.Dir(candidatePath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(candidatePath, []byte("mutable"), 0o600); err != nil {
		t.Fatal(err)
	}
	if result, err := Extract(context.Background(), dir); err == nil || !strings.Contains(err.Error(), "FORMAT4_CANDIDATE_STATE_PRESENT") || len(result.Document) != 0 {
		t.Fatalf("candidate result=%q err=%v", result.Document, err)
	}
	if err := os.Remove(candidatePath); err != nil {
		t.Fatal(err)
	}
	result, err := extract(context.Background(), dir, func() {
		if _, writeErr := native.NewObjectStore(filepath.Join(dir, ".sealgraph")).WriteBlob(context.Background(), []byte("concurrent")); writeErr != nil {
			t.Fatal(writeErr)
		}
	})
	if err == nil || !strings.Contains(err.Error(), "MIGRATION_SOURCE_CHANGED") || len(result.Document) != 0 {
		t.Fatalf("change result=%q err=%v", result.Document, err)
	}
}

func TestExtractDocumentsChangeAndRestoreObservationBoundary(t *testing.T) {
	dir, _, _, _ := makeFormat4Root(t)
	configPath := filepath.Join(dir, ".sealgraph", "config")
	baseline, err := Extract(context.Background(), dir)
	if err != nil {
		t.Fatal(err)
	}
	result, err := extract(context.Background(), dir, func() {
		if writeErr := os.WriteFile(configPath, []byte("transient invalid config"), 0o644); writeErr != nil {
			t.Fatal(writeErr)
		}
		if writeErr := os.WriteFile(configPath, []byte(format4Config), 0o644); writeErr != nil {
			t.Fatal(writeErr)
		}
	})
	if err != nil || !bytes.Equal(result.Document, baseline.Document) {
		t.Fatalf("change-and-restore boundary err=%v equal=%t", err, bytes.Equal(result.Document, baseline.Document))
	}
}

func TestExtractRejectsUnrecognizedCandidateNamespaceShapes(t *testing.T) {
	tests := []struct {
		name     string
		relative string
	}{
		{name: "track at namespace root", relative: ".track"},
		{name: "track below invalid REF", relative: filepath.Join("bad@ref", ".track")},
		{name: "unknown file below valid REF", relative: filepath.Join("ROOT", "unexpected")},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			dir, _, _, _ := makeFormat4Root(t)
			candidatePath := filepath.Join(dir, ".sealgraph", "index", test.relative)
			if err := os.MkdirAll(filepath.Dir(candidatePath), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(candidatePath, []byte("unexpected"), 0o600); err != nil {
				t.Fatal(err)
			}
			result, err := Extract(context.Background(), dir)
			if err == nil || !strings.Contains(err.Error(), "FORMAT4_CANDIDATE_STATE_PRESENT") || len(result.Document) != 0 {
				t.Fatalf("result=%q err=%v", result.Document, err)
			}
		})
	}
}

func TestExtractRejectsMalformedPhysicalObjectLayout(t *testing.T) {
	dir, _, _, _ := makeFormat4Root(t)
	if err := os.Mkdir(filepath.Join(dir, ".sealgraph", "objects", "not-hex"), 0o755); err != nil {
		t.Fatal(err)
	}
	if result, err := Extract(context.Background(), dir); err == nil || !strings.Contains(err.Error(), "unexpected object directory") || len(result.Document) != 0 {
		t.Fatalf("result=%q err=%v", result.Document, err)
	}
}

func TestExtractRejectsMalformedREFDirectoryLayout(t *testing.T) {
	dir, _, _, _ := makeFormat4Root(t)
	if err := os.Mkdir(filepath.Join(dir, ".sealgraph", "refs", "seals", "bad@ref"), 0o755); err != nil {
		t.Fatal(err)
	}
	if result, err := Extract(context.Background(), dir); err == nil || !strings.Contains(err.Error(), "invalid stored REF directory") || len(result.Document) != 0 {
		t.Fatalf("result=%q err=%v", result.Document, err)
	}
}

func TestExtractRejectsReservedFormat4TagName(t *testing.T) {
	dir, _, sealID, _ := makeFormat4Root(t)
	manifestPath := filepath.Join(dir, ".sealgraph", "refs", "seals", "ROOT", ".ref")
	manifest := fmt.Sprintf(`{"schema":"sealgraph/ref/v1","head":"%s","tags":[{"name":"abcd","target":"%s"}]}`, sealID, sealID)
	if err := os.WriteFile(manifestPath, []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}
	result, err := Extract(context.Background(), dir)
	if err == nil || !strings.Contains(err.Error(), "TAGNAME cannot be") || len(result.Document) != 0 {
		t.Fatalf("result=%q err=%v", result.Document, err)
	}
}

func TestExtractRejectsNonCanonicalLooseObjectHeaders(t *testing.T) {
	tests := []struct {
		name     string
		envelope []byte
	}{
		{name: "plus sign", envelope: []byte("blob +3\x00abc")},
		{name: "negative zero", envelope: []byte("blob -0\x00")},
		{name: "leading zero", envelope: []byte("blob 03\x00abc")},
		{name: "trailing space", envelope: []byte("blob 3 \x00abc")},
		{name: "extra separator", envelope: []byte("blob  3\x00abc")},
		{name: "non decimal", envelope: []byte("blob three\x00abc")},
		{name: "overflow", envelope: []byte("blob 999999999999999999999999999999999999\x00abc")},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			dir, _, _, _ := makeFormat4Root(t)
			writeRawLooseObject(t, filepath.Join(dir, ".sealgraph"), test.envelope)
			result, err := Extract(context.Background(), dir)
			if err == nil || !strings.Contains(err.Error(), "invalid object header") || len(result.Document) != 0 {
				t.Fatalf("result=%q err=%v", result.Document, err)
			}
		})
	}
}

func TestExtractClassifiesCollapsedRevisionAndMergedCauseLinks(t *testing.T) {
	dir, contentID, firstID, _ := makeFormat4Root(t)
	root := filepath.Join(dir, ".sealgraph")
	objectStore := native.NewObjectStore(root)
	secondPayload := domain.SealPayload{Schema: domain.SealSchema, ParentRevision: &firstID, Content: domain.ContentRef{Store: domain.NativeStore, Type: domain.BlobType, ID: contentID}, Attachments: []domain.Attachment{}, Links: []domain.Link{}, Root: true}
	secondID := writeFormat4Seal(t, objectStore, secondPayload)
	observerContent, err := objectStore.WriteBlob(context.Background(), []byte("observer"))
	if err != nil {
		t.Fatal(err)
	}
	observerPayload := domain.SealPayload{Schema: domain.SealSchema, Content: domain.ContentRef{Store: domain.NativeStore, Type: domain.BlobType, ID: observerContent}, Attachments: []domain.Attachment{}, Links: []domain.Link{{TargetSeal: firstID, Message: "first"}, {TargetSeal: secondID, Message: "second"}}}
	observerID := writeFormat4Seal(t, objectStore, observerPayload)
	updateFormat4RevisionAndObserver(t, root, firstID, secondID, observerID)
	result, err := Extract(context.Background(), dir)
	if err != nil {
		t.Fatal(err)
	}
	wantWarnings := []string{"SEMANTIC_CHANGE_COLLAPSED_REVISION_DROPPED count=1", "SEMANTIC_CHANGE_MERGED_CAUSE_LINKS count=1"}
	if strings.Join(result.Warnings, "\n") != strings.Join(wantWarnings, "\n") {
		t.Fatalf("warnings=%v", result.Warnings)
	}
	document, err := migration.DecodeUniversalBlobV1(result.Document)
	if err != nil {
		t.Fatal(err)
	}
	if len(document.Projection.Collapsed) != 1 || len(document.Projection.Merged) != 1 || len(document.Projection.Merged[0].OldTargets) != 2 {
		t.Fatalf("projection=%+v", document.Projection)
	}
	projection, err := migration.ProjectUniversalBlobV1(document)
	if err != nil {
		t.Fatal(err)
	}
	mapped := projection.OldToNew
	if !mapped[firstID.String()].Equal(mapped[secondID.String()]) {
		t.Fatalf("expected old targets to collapse: first=%s second=%s", mapped[firstID.String()], mapped[secondID.String()])
	}
	foundMergedMessages := false
	for _, projected := range projection.Seals {
		if !projected.OldID.Equal(observerID) {
			continue
		}
		provenance, decodeErr := canonicalv5.DecodeProvenance(projected.ProvenanceBytes)
		if decodeErr != nil {
			t.Fatal(decodeErr)
		}
		foundMergedMessages = len(provenance.CauseLinks) == 1 &&
			len(provenance.CauseLinks[0].PreviousRevisionSealOfTargetSeal) == 0 &&
			strings.Join(provenance.CauseLinks[0].Messages, "\x00") == "first\x00second"
	}
	if !foundMergedMessages {
		t.Fatal("collapsed Cause targets did not preserve the exact sorted union of messages")
	}
}

func TestExtractComplexFixtureIsStableAndDependencyFirst(t *testing.T) {
	dir, _, firstID, _ := makeFormat4Root(t)
	root := filepath.Join(dir, ".sealgraph")
	objectStore := native.NewObjectStore(root)
	attachmentID, err := objectStore.WriteBlob(context.Background(), []byte{0, 1, 2, '\n', 0xff})
	if err != nil {
		t.Fatal(err)
	}
	leftContent, err := objectStore.WriteBlob(context.Background(), []byte("left\x00\n雪\u2028\U0001f642/\"quote\"\\literal"))
	if err != nil {
		t.Fatal(err)
	}
	rightContent, err := objectStore.WriteBlob(context.Background(), []byte("right\r\n<&>"))
	if err != nil {
		t.Fatal(err)
	}
	leftID := writeFormat4Seal(t, objectStore, domain.SealPayload{
		Schema:         domain.SealSchema,
		ParentRevision: &firstID,
		Content:        domain.ContentRef{Store: domain.NativeStore, Type: domain.BlobType, ID: leftContent},
		Attachments: []domain.Attachment{{
			Name:      "資料\n雪",
			MediaType: "application/octet-stream; x=\"雪\"",
			Blob:      domain.ContentRef{Store: domain.NativeStore, Type: domain.BlobType, ID: attachmentID},
		}},
		Links: []domain.Link{},
		Root:  true,
	})
	rightID := writeFormat4Seal(t, objectStore, domain.SealPayload{
		Schema:         domain.SealSchema,
		ParentRevision: &firstID,
		Content:        domain.ContentRef{Store: domain.NativeStore, Type: domain.BlobType, ID: rightContent},
		Attachments:    []domain.Attachment{},
		Links:          []domain.Link{},
		Root:           true,
	})
	refs := native.NewRefStore(root)
	if err := refs.Update(context.Background(), "LEFT", nil, &leftID); err != nil {
		t.Fatal(err)
	}
	if err := refs.Update(context.Background(), "RIGHT", nil, &rightID); err != nil {
		t.Fatal(err)
	}
	if err := native.NewTagStore(root).Create(context.Background(), "LEFT", "v 1/雪", firstID, leftID); err != nil {
		t.Fatal(err)
	}

	result, err := Extract(context.Background(), dir)
	if err != nil {
		t.Fatal(err)
	}
	document, err := migration.DecodeUniversalBlobV1(result.Document)
	if err != nil {
		t.Fatal(err)
	}
	if len(document.REFs) != 3 || len(document.Tags) != 1 || len(document.Objects) != 4 || len(document.Seals) != 3 || len(document.Projection.Unobserved) != 2 {
		t.Fatalf("unexpected complex inventory: refs=%d tags=%d objects=%d seals=%d projection=%+v", len(document.REFs), len(document.Tags), len(document.Objects), len(document.Seals), document.Projection)
	}
	positions := make(map[string]int, len(document.Seals))
	for index, seal := range document.Seals {
		positions[seal.ID.String()] = index
	}
	if positions[firstID.String()] >= positions[leftID.String()] || positions[firstID.String()] >= positions[rightID.String()] {
		t.Fatalf("Seal inventory is not dependency-first: %+v", positions)
	}
	wantWarnings := []string{"SEMANTIC_CHANGE_UNOBSERVED_PARENT_DROPPED count=2"}
	if strings.Join(result.Warnings, "\n") != strings.Join(wantWarnings, "\n") {
		t.Fatalf("warnings=%v", result.Warnings)
	}
	digest := fmt.Sprintf("%x", sha256.Sum256(result.Document))
	const wantDigest = "92934549feae313481d89937e47f31e7a9883dad0e3e8d2dcaff5de25e6b90b4"
	if digest != wantDigest {
		t.Fatalf("document digest=%s want=%s", digest, wantDigest)
	}
}

func TestExtractClassifiesMaterializedRevision(t *testing.T) {
	dir, firstContent, firstID, _ := makeFormat4Root(t)
	root := filepath.Join(dir, ".sealgraph")
	objectStore := native.NewObjectStore(root)
	secondContent, err := objectStore.WriteBlob(context.Background(), []byte("changed material"))
	if err != nil {
		t.Fatal(err)
	}
	secondID := writeFormat4Seal(t, objectStore, domain.SealPayload{Schema: domain.SealSchema, ParentRevision: &firstID, Content: domain.ContentRef{Store: domain.NativeStore, Type: domain.BlobType, ID: secondContent}, Attachments: []domain.Attachment{}, Links: []domain.Link{}, Root: true})
	observerID := writeFormat4Seal(t, objectStore, domain.SealPayload{Schema: domain.SealSchema, Content: domain.ContentRef{Store: domain.NativeStore, Type: domain.BlobType, ID: firstContent}, Attachments: []domain.Attachment{}, Links: []domain.Link{{TargetSeal: secondID, Message: "observed revision"}}})
	updateFormat4RevisionAndObserver(t, root, firstID, secondID, observerID)

	result, err := Extract(context.Background(), dir)
	if err != nil {
		t.Fatal(err)
	}
	document, err := migration.DecodeUniversalBlobV1(result.Document)
	if err != nil {
		t.Fatal(err)
	}
	if len(document.Projection.Materialized) != 1 || len(document.Projection.Collapsed) != 0 || len(document.Projection.Unobserved) != 0 || len(result.Warnings) != 0 {
		t.Fatalf("projection=%+v warnings=%v", document.Projection, result.Warnings)
	}
}

func TestExtractPreservesDraftAndMultiObserverAssertions(t *testing.T) {
	dir, firstContent, firstID, _ := makeFormat4Root(t)
	root := filepath.Join(dir, ".sealgraph")
	objectStore := native.NewObjectStore(root)
	secondContent, err := objectStore.WriteBlob(context.Background(), []byte("multi-observer target"))
	if err != nil {
		t.Fatal(err)
	}
	secondID := writeFormat4Seal(t, objectStore, domain.SealPayload{Schema: domain.SealSchema, ParentRevision: &firstID, Content: domain.ContentRef{Store: domain.NativeStore, Type: domain.BlobType, ID: secondContent}, Attachments: []domain.Attachment{}, Links: []domain.Link{}, Root: true})
	firstObserverID := writeFormat4Seal(t, objectStore, domain.SealPayload{Schema: domain.SealSchema, Content: domain.ContentRef{Store: domain.NativeStore, Type: domain.BlobType, ID: firstContent}, Attachments: []domain.Attachment{}, Links: []domain.Link{{TargetSeal: secondID, Message: ""}}})
	secondObserverContent, err := objectStore.WriteBlob(context.Background(), []byte("draft observer"))
	if err != nil {
		t.Fatal(err)
	}
	secondMessage := "observed\n雪\U0001f642\\quoted"
	secondObserverID := writeFormat4Seal(t, objectStore, domain.SealPayload{Schema: domain.SealSchema, Content: domain.ContentRef{Store: domain.NativeStore, Type: domain.BlobType, ID: secondObserverContent}, Attachments: []domain.Attachment{}, Links: []domain.Link{{TargetSeal: secondID, Message: secondMessage}}, Draft: true})
	refs := native.NewRefStore(root)
	if err := refs.Update(context.Background(), "ROOT", &firstID, &secondID); err != nil {
		t.Fatal(err)
	}
	if err := refs.Update(context.Background(), "OBSERVER-1", nil, &firstObserverID); err != nil {
		t.Fatal(err)
	}
	if err := refs.Update(context.Background(), "OBSERVER-2", nil, &secondObserverID); err != nil {
		t.Fatal(err)
	}

	result, err := Extract(context.Background(), dir)
	if err != nil {
		t.Fatal(err)
	}
	document, err := migration.DecodeUniversalBlobV1(result.Document)
	if err != nil {
		t.Fatal(err)
	}
	projection, err := migration.ProjectUniversalBlobV1(document)
	if err != nil {
		t.Fatal(err)
	}
	if len(document.Projection.Materialized) != 2 || len(document.Projection.Unobserved) != 0 || len(result.Warnings) != 0 {
		t.Fatalf("projection=%+v warnings=%v", document.Projection, result.Warnings)
	}
	foundDraft := false
	for _, projected := range projection.Seals {
		if !projected.OldID.Equal(secondObserverID) {
			continue
		}
		provenance, decodeErr := canonicalv5.DecodeProvenance(projected.ProvenanceBytes)
		if decodeErr != nil {
			t.Fatal(decodeErr)
		}
		foundDraft = provenance.Draft && len(provenance.CauseLinks) == 1 && len(provenance.CauseLinks[0].Messages) == 1 && provenance.CauseLinks[0].Messages[0] == secondMessage
	}
	if !foundDraft {
		t.Fatal("draft observer or its exact message was not preserved in projected Provenance")
	}
}

func updateFormat4RevisionAndObserver(t *testing.T, root string, firstID, secondID, observerID domain.ObjectID) {
	t.Helper()
	refs := native.NewRefStore(root)
	if err := refs.Update(context.Background(), "ROOT", &firstID, &secondID); err != nil {
		t.Fatal(err)
	}
	if err := refs.Update(context.Background(), "OBSERVER", nil, &observerID); err != nil {
		t.Fatal(err)
	}
}

func TestExtractRejectsMissingGraphTargetWithoutDocument(t *testing.T) {
	dir, _, _, _ := makeFormat4Root(t)
	missing := domain.ObjectID{Hex: strings.Repeat("a", 64)}
	manifest, err := native.NewRefStore(filepath.Join(dir, ".sealgraph")).PreviewUpdate(context.Background(), "MISSING", nil, &missing)
	if err != nil {
		t.Fatal(err)
	}
	manifestPath := filepath.Join(dir, ".sealgraph", "refs", "seals", "MISSING", ".ref")
	if err := os.MkdirAll(filepath.Dir(manifestPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(manifestPath, manifest, 0o644); err != nil {
		t.Fatal(err)
	}
	result, err := Extract(context.Background(), dir)
	if err == nil || !strings.Contains(err.Error(), "absent from the loose object inventory") || len(result.Document) != 0 {
		t.Fatalf("result=%q err=%v", result.Document, err)
	}
}

func makeFormat4Root(t *testing.T) (string, domain.ObjectID, domain.ObjectID, domain.ObjectID) {
	t.Helper()
	dir := t.TempDir()
	root := filepath.Join(dir, ".sealgraph")
	for _, relative := range []string{"objects", filepath.Join("refs", "seals"), "index", "locks"} {
		if err := os.MkdirAll(filepath.Join(root, relative), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(root, "config"), []byte(format4Config), 0o644); err != nil {
		t.Fatal(err)
	}
	store := native.NewObjectStore(root)
	contentID, err := store.WriteBlob(context.Background(), []byte("format-4 material"))
	if err != nil {
		t.Fatal(err)
	}
	payload := domain.SealPayload{Schema: domain.SealSchema, Content: domain.ContentRef{Store: domain.NativeStore, Type: domain.BlobType, ID: contentID}, Attachments: []domain.Attachment{}, Links: []domain.Link{}, Root: true}
	sealID := writeFormat4Seal(t, store, payload)
	excludedID, err := store.WriteBlob(context.Background(), []byte("unreferenced format-4 object"))
	if err != nil {
		t.Fatal(err)
	}
	if err := native.NewRefStore(root).Update(context.Background(), "ROOT", nil, &sealID); err != nil {
		t.Fatal(err)
	}
	return dir, contentID, sealID, excludedID
}

func writeFormat4Seal(t *testing.T, store *native.ObjectStore, payload domain.SealPayload) domain.ObjectID {
	t.Helper()
	payloadBytes, err := canonical.EncodeSeal(payload)
	if err != nil {
		t.Fatal(err)
	}
	id, err := store.WriteBlob(context.Background(), payloadBytes)
	if err != nil {
		t.Fatal(err)
	}
	return id
}

func writeRawLooseObject(t *testing.T, root string, envelope []byte) domain.ObjectID {
	t.Helper()
	digest := sha256.Sum256(envelope)
	id := domain.ObjectID{Hex: fmt.Sprintf("%x", digest)}
	var compressed bytes.Buffer
	writer := zlib.NewWriter(&compressed)
	if _, err := writer.Write(envelope); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	objectPath := filepath.Join(root, "objects", id.String()[:2], id.String()[2:])
	if err := os.MkdirAll(filepath.Dir(objectPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(objectPath, compressed.Bytes(), 0o444); err != nil {
		t.Fatal(err)
	}
	return id
}

func snapshotTree(t *testing.T, root string) map[string]string {
	t.Helper()
	result := make(map[string]string)
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		info, err := os.Lstat(path)
		if err != nil {
			return err
		}
		value := fmt.Sprintf("%s:%o", info.Mode().Type(), info.Mode().Perm())
		if info.Mode().IsRegular() {
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			value += fmt.Sprintf(":%x", sha256.Sum256(data))
		}
		result[filepath.ToSlash(relative)] = value
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func mapsEqual(left, right map[string]string) bool {
	if len(left) != len(right) {
		return false
	}
	for key, value := range left {
		if right[key] != value {
			return false
		}
	}
	return true
}
