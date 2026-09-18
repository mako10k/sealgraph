package repository

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mako10k/sealgraph/internal/domain"
	"github.com/mako10k/sealgraph/internal/migration"
)

func TestNativeSnapshotRestoresFullSourceAndOpaqueOrphan(t *testing.T) {
	document, sealID, sourceID, orphanID := nativeRoundTripFixture(t)
	target := t.TempDir()
	receipt, err := LoadNativeSnapshotV1(context.Background(), target, document)
	if err != nil {
		t.Fatal(err)
	}
	loaded, err := OpenStandalone(target)
	if err != nil {
		t.Fatal(err)
	}
	assertNativeLoadReceipt(t, receipt, document)
	assertNativeRestored(t, loaded, document, sealID, sourceID, orphanID)
}

func nativeRoundTripFixture(t *testing.T) ([]byte, domain.ObjectID, domain.ObjectID, domain.ObjectID) {
	t.Helper()
	ctx := context.Background()
	source := openFormat7Fixture(t)
	sealID, sourceID := format7SealFixture(t, source, []byte("abcXYZtail"), 3)
	if err := source.refs.Update(ctx, "root", nil, &sealID); err != nil {
		t.Fatal(err)
	}
	if err := source.tags.Create(ctx, "root", "checkpoint", sealID, sealID); err != nil {
		t.Fatal(err)
	}
	if _, err := source.Add(ctx, AddOptions{REF: "draft", Content: []byte("draft"), Root: true, RootSet: true, ClearCauseLinks: true}); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(source.workDir, "original.txt")
	if err := os.WriteFile(path, []byte("abcXYZtail"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := source.TraceSourceBind(ctx, "source-A", "original.txt"); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	orphanID := putFormat7Record(t, source, []byte("opaque orphan"))
	document, err := source.DumpNativeSnapshotV1(ctx)
	if err != nil {
		t.Fatal(err)
	}
	snapshot, err := migration.DecodeNativeSnapshotV1(document)
	if err != nil || len(snapshot.REFs) != 1 || len(snapshot.Candidates) != 1 {
		t.Fatalf("snapshot=%+v err=%v", snapshot, err)
	}
	return document, sealID, sourceID, orphanID
}

func assertNativeLoadReceipt(t *testing.T, receipt, document []byte) {
	t.Helper()
	var result nativeLoadReceipt
	snapshot, err := migration.DecodeNativeSnapshotV1(document)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(bytes.TrimSpace(receipt), &result); err != nil || result.Schema != "sealgraph/native-load/v1" || result.Result != "LOADED" || result.RepositoryFormat != 7 || result.REFs != 1 || result.Candidates != 1 || result.Blobs != len(snapshot.Blobs) {
		t.Fatalf("receipt=%s err=%v", receipt, err)
	}
}

func assertNativeRestored(t *testing.T, loaded *Repository, document []byte, sealID, sourceID, orphanID domain.ObjectID) {
	t.Helper()
	ctx := context.Background()
	if _, err := loaded.Fsck(ctx); err != nil {
		t.Fatal(err)
	}
	bindings, err := loaded.TraceSourceList()
	if err != nil || len(bindings) != 0 {
		t.Fatalf("local bindings transported: %+v err=%v", bindings, err)
	}
	restored, err := loaded.objects.ReadObject(ctx, sourceID)
	if err != nil || !bytes.Equal(restored.Data, []byte("abcXYZtail")) {
		t.Fatalf("full source=%q err=%v", restored.Data, err)
	}
	if _, err := loaded.objects.ReadObject(ctx, orphanID); err != nil {
		t.Fatalf("opaque orphan was not retained: %v", err)
	}
	resolved, err := loaded.LoadSeal(ctx, sealID)
	if err != nil || resolved.Provenance.Origin == nil {
		t.Fatalf("restored Seal=%+v err=%v", resolved, err)
	}
	redump, err := loaded.DumpNativeSnapshotV1(ctx)
	if err != nil || !bytes.Equal(redump, document) {
		t.Fatalf("native document changed after load: %v", err)
	}
}

func TestNativeSnapshotRejectsMissingClosureBeforePublication(t *testing.T) {
	ctx := context.Background()
	source := openFormat7Fixture(t)
	sealID, sourceID := format7SealFixture(t, source, []byte("abcXYZtail"), 3)
	if err := source.refs.Update(ctx, "root", nil, &sealID); err != nil {
		t.Fatal(err)
	}
	document, err := source.DumpNativeSnapshotV1(ctx)
	if err != nil {
		t.Fatal(err)
	}
	snapshot, err := migration.DecodeNativeSnapshotV1(document)
	if err != nil {
		t.Fatal(err)
	}
	filtered := snapshot.Blobs[:0]
	for _, blob := range snapshot.Blobs {
		if !blob.ID.Equal(sourceID) {
			filtered = append(filtered, blob)
		}
	}
	snapshot.Blobs = filtered
	broken, err := migration.EncodeNativeSnapshotV1(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	target := t.TempDir()
	if _, err := LoadNativeSnapshotV1(ctx, target, broken); err == nil || !strings.Contains(err.Error(), "PRE_PUBLICATION_FAILURE") {
		t.Fatalf("missing closure error=%v", err)
	}
	if _, err := os.Lstat(filepath.Join(target, ".sealgraph")); !os.IsNotExist(err) {
		t.Fatalf("invalid input published destination: %v", err)
	}
}

func TestNativeSnapshotRequiresAbsentTargetAndFormat7Source(t *testing.T) {
	ctx := context.Background()
	source := openFormat7Fixture(t)
	document, err := source.DumpNativeSnapshotV1(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := LoadNativeSnapshotV1(ctx, source.workDir, document); err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("existing destination error=%v", err)
	}
	legacy := t.TempDir()
	if _, err := InitStandalone(legacy); err != nil {
		t.Fatal(err)
	}
	old, err := OpenStandalone(legacy)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := old.DumpNativeSnapshotV1(ctx); err == nil || !strings.Contains(err.Error(), "format 7") {
		t.Fatalf("legacy dump error=%v", err)
	}
}

func TestNativeSnapshotRejectsChangedInputBeforePublication(t *testing.T) {
	ctx := context.Background()
	source := openFormat7Fixture(t)
	document, err := source.DumpNativeSnapshotV1(ctx)
	if err != nil {
		t.Fatal(err)
	}
	target := t.TempDir()
	checked := false
	_, err = LoadNativeSnapshotV1WithPrePublishCheck(ctx, target, document, func() error {
		checked = true
		return os.ErrInvalid
	})
	if !checked || err == nil || !strings.Contains(err.Error(), "PRE_PUBLICATION_FAILURE") {
		t.Fatalf("input prepublication check: checked=%t err=%v", checked, err)
	}
	if _, err := os.Lstat(filepath.Join(target, ".sealgraph")); !os.IsNotExist(err) {
		t.Fatalf("changed input published destination: %v", err)
	}
}
