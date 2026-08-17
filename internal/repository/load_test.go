package repository

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mako10k/sealgraph/internal/domain"
	"github.com/mako10k/sealgraph/internal/migration"
)

func format3Record(t *testing.T, payload migration.Format3SealPayload) migration.SealRecord {
	t.Helper()
	encoded, err := migration.EncodeFormat3Seal(payload)
	if err != nil {
		t.Fatal(err)
	}
	return migration.SealRecord{OldSealID: domain.ComputeNativeBlobID(encoded), Payload: payload}
}

func loadFixture(t *testing.T) migration.LogicalDumpV1 {
	t.Helper()
	rootData := []byte{'r', 0, '\r', '\n', 0xff}
	childData := []byte("child")
	rootContent := domain.ComputeNativeBlobID(rootData)
	childContent := domain.ComputeNativeBlobID(childData)
	root := format3Record(t, migration.Format3SealPayload{
		Schema: migration.Format3SealSchema, REF: "ROOT",
		Content:     domain.ContentRef{Store: domain.NativeStore, Type: domain.BlobType, ID: rootContent},
		Attachments: []domain.Attachment{}, Links: []migration.Format3Link{}, Root: true,
	})
	child := format3Record(t, migration.Format3SealPayload{
		Schema: migration.Format3SealSchema, REF: "CHILD",
		Content:     domain.ContentRef{Store: domain.NativeStore, Type: domain.BlobType, ID: childContent},
		Attachments: []domain.Attachment{},
		Links:       []migration.Format3Link{{TargetREF: "ROOT", TargetSeal: root.OldSealID, Message: "basis"}},
	})
	return migration.LogicalDumpV1{
		Objects: []migration.ObjectRecord{{ID: childContent, Data: childData}, {ID: rootContent, Data: rootData}},
		Seals:   []migration.SealRecord{root, child},
		REFs:    []migration.RefRecord{{Name: "CHILD", Head: child.OldSealID}, {Name: "ROOT", Head: root.OldSealID}},
		Tags:    []migration.TagRecord{}, ExcludedObjects: []domain.ObjectID{},
	}
}

func TestLoadLogicalV1PublishesCompleteFormat4RepositoryAndReceipt(t *testing.T) {
	workDir := t.TempDir()
	dumpBytes, err := migration.EncodeLogicalDumpV1(loadFixture(t))
	if err != nil {
		t.Fatal(err)
	}
	receipt, err := LoadLogicalV1(context.Background(), workDir, dumpBytes)
	if err != nil {
		t.Fatal(err)
	}
	var decoded struct {
		Schema                    string `json:"schema"`
		SourceDumpSHA256          string `json:"source_dump_sha256"`
		OldToNewSeals             []any  `json:"old_to_new_seals"`
		Refs                      []any  `json:"refs"`
		Tags                      []any  `json:"tags"`
		PublishedRepositoryFormat int    `json:"published_repository_format"`
	}
	if err := json.Unmarshal(receipt, &decoded); err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(dumpBytes)
	if decoded.Schema != logicalLoadReceiptV1Schema || decoded.SourceDumpSHA256 != fmt.Sprintf("%x", digest) || len(decoded.OldToNewSeals) != 2 || len(decoded.Refs) != 2 || len(decoded.Tags) != 0 || decoded.PublishedRepositoryFormat != 4 {
		t.Fatalf("receipt = %s", receipt)
	}
	repo, err := OpenStandalone(workDir)
	if err != nil {
		t.Fatal(err)
	}
	child, err := repo.Show(context.Background(), "CHILD")
	if err != nil {
		t.Fatal(err)
	}
	if string(child.Content) != "child" || len(child.Payload.Links) != 1 || child.Payload.Links[0].Message != "basis" {
		t.Fatalf("loaded child = %+v content=%q", child.Payload, child.Content)
	}
	root, err := repo.Show(context.Background(), "ROOT")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(root.Content, []byte{'r', 0, '\r', '\n', 0xff}) {
		t.Fatalf("binary root content changed: %v", root.Content)
	}
	for _, relative := range []string{"index", "locks"} {
		entries, err := os.ReadDir(filepath.Join(workDir, ".sealgraph", relative))
		if err != nil || len(entries) != 0 {
			t.Fatalf("runtime/deferred directory %s entries=%v err=%v", relative, entries, err)
		}
	}
}

func TestLoadLogicalV1ReportsManyToOneCollapse(t *testing.T) {
	data := []byte("same")
	contentID := domain.ComputeNativeBlobID(data)
	first := format3Record(t, migration.Format3SealPayload{
		Schema: migration.Format3SealSchema, REF: "A", Content: domain.ContentRef{Store: domain.NativeStore, Type: domain.BlobType, ID: contentID}, Attachments: []domain.Attachment{}, Links: []migration.Format3Link{}, Root: true,
	})
	second := format3Record(t, migration.Format3SealPayload{
		Schema: migration.Format3SealSchema, REF: "B", Content: domain.ContentRef{Store: domain.NativeStore, Type: domain.BlobType, ID: contentID}, Attachments: []domain.Attachment{}, Links: []migration.Format3Link{}, Root: true,
	})
	seals := []migration.SealRecord{first, second}
	if seals[0].OldSealID.String() > seals[1].OldSealID.String() {
		seals[0], seals[1] = seals[1], seals[0]
	}
	dump := migration.LogicalDumpV1{
		Objects: []migration.ObjectRecord{{ID: contentID, Data: data}}, Seals: seals,
		REFs: []migration.RefRecord{{Name: "A", Head: first.OldSealID}, {Name: "B", Head: second.OldSealID}}, Tags: []migration.TagRecord{}, ExcludedObjects: []domain.ObjectID{},
	}
	input, err := migration.EncodeLogicalDumpV1(dump)
	if err != nil {
		t.Fatal(err)
	}
	workDir := t.TempDir()
	receipt, err := LoadLogicalV1(context.Background(), workDir, input)
	if err != nil {
		t.Fatal(err)
	}
	var decoded struct {
		Collapsed []struct {
			New string   `json:"new_seal_id"`
			Old []string `json:"old_seal_ids"`
		} `json:"collapsed_seals"`
	}
	if err := json.Unmarshal(receipt, &decoded); err != nil {
		t.Fatal(err)
	}
	if len(decoded.Collapsed) != 1 || len(decoded.Collapsed[0].Old) != 2 {
		t.Fatalf("collapse receipt = %s", receipt)
	}
	repo, _ := OpenStandalone(workDir)
	a, _ := repo.ResolveSelector(context.Background(), "A")
	b, _ := repo.ResolveSelector(context.Background(), "B")
	if !a.ID.Equal(b.ID) || a.ID.String() != decoded.Collapsed[0].New {
		t.Fatalf("collapsed heads = %s %s receipt=%s", a.ID, b.ID, receipt)
	}
}

func TestLoadPreservesTagsAndRejectsExistingTargetWithoutMutation(t *testing.T) {
	fixture := loadFixture(t)
	fixture.Tags = []migration.TagRecord{{REF: "ROOT", Name: "accepted", Target: fixture.REFs[1].Head}}
	input, err := migration.EncodeLogicalDumpV1(fixture)
	if err != nil {
		t.Fatal(err)
	}
	workDir := t.TempDir()
	receipt, err := LoadLogicalV1(context.Background(), workDir, input)
	if err != nil || !bytes.Contains(receipt, []byte(`"tags":[{"ref":"ROOT","name":"accepted"`)) {
		t.Fatalf("tag-bearing load receipt=%s err=%v", receipt, err)
	}
	repo, err := OpenStandalone(workDir)
	if err != nil {
		t.Fatal(err)
	}
	resolved, err := repo.ResolveSelector(context.Background(), "ROOT@accepted")
	if err != nil {
		t.Fatal(err)
	}
	root, _ := repo.ResolveSelector(context.Background(), "ROOT")
	if !resolved.ID.Equal(root.ID) {
		t.Fatalf("loaded tag target=%s root=%s", resolved.ID, root.ID)
	}

	existingDir := t.TempDir()
	if created, err := InitStandalone(existingDir); err != nil || !created {
		t.Fatal(err)
	}
	before, _ := os.ReadFile(filepath.Join(existingDir, ".sealgraph", "config"))
	tagless, _ := migration.EncodeLogicalDumpV1(loadFixture(t))
	if _, err := LoadLogicalV1(context.Background(), existingDir, tagless); err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("existing target load error = %v", err)
	}
	after, _ := os.ReadFile(filepath.Join(existingDir, ".sealgraph", "config"))
	if !bytes.Equal(before, after) {
		t.Fatal("existing repository changed")
	}
}

func TestLoadRejectsNoncanonicalInputBeforeTargetPublication(t *testing.T) {
	input, _ := migration.EncodeLogicalDumpV1(loadFixture(t))
	input = bytes.TrimSuffix(input, []byte{'\n'})
	workDir := t.TempDir()
	if _, err := LoadLogicalV1(context.Background(), workDir, input); err == nil {
		t.Fatal("noncanonical dump was accepted")
	}
	if _, err := os.Lstat(filepath.Join(workDir, ".sealgraph")); !os.IsNotExist(err) {
		t.Fatalf("failed load published target: %v", err)
	}
	entries, err := os.ReadDir(workDir)
	if err != nil || len(entries) != 0 {
		t.Fatalf("failed load retained staging: %v err=%v", entries, err)
	}
}

func TestRenameNoReplacePreservesConcurrentDestination(t *testing.T) {
	parent := t.TempDir()
	source := filepath.Join(parent, "source")
	destination := filepath.Join(parent, "destination")
	if err := os.Mkdir(source, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(destination, 0o755); err != nil {
		t.Fatal(err)
	}
	marker := filepath.Join(destination, "marker")
	if err := os.WriteFile(marker, []byte("keep"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := renameNoReplace(source, destination); err == nil {
		t.Fatal("no-replace publication replaced an existing destination")
	}
	if data, err := os.ReadFile(marker); err != nil || string(data) != "keep" {
		t.Fatalf("destination marker data=%q err=%v", data, err)
	}
	if info, err := os.Stat(source); err != nil || !info.IsDir() {
		t.Fatalf("source was consumed on failed publication: %v %v", info, err)
	}
}

func TestLoadReportsAbandonedStagingWithoutAdoptingOrDeletingIt(t *testing.T) {
	workDir := t.TempDir()
	staging := filepath.Join(workDir, ".sealgraph-load-evidence")
	if err := os.Mkdir(staging, 0o755); err != nil {
		t.Fatal(err)
	}
	marker := filepath.Join(staging, "inspect-me")
	if err := os.WriteFile(marker, []byte("evidence"), 0o600); err != nil {
		t.Fatal(err)
	}
	input, _ := migration.EncodeLogicalDumpV1(loadFixture(t))
	if _, err := LoadLogicalV1(context.Background(), workDir, input); err == nil || !strings.Contains(err.Error(), "inspect and remove it explicitly") {
		t.Fatalf("abandoned staging error = %v", err)
	}
	if data, err := os.ReadFile(marker); err != nil || string(data) != "evidence" {
		t.Fatalf("abandoned staging was changed: data=%q err=%v", data, err)
	}
	if _, err := os.Lstat(filepath.Join(workDir, ".sealgraph")); !os.IsNotExist(err) {
		t.Fatalf("abandoned staging was adopted: %v", err)
	}
}

func TestLoadRejectsCauseTargetsThatCollapseToFormat4Duplicate(t *testing.T) {
	shared := []byte("shared")
	childData := []byte("child")
	sharedID := domain.ComputeNativeBlobID(shared)
	childID := domain.ComputeNativeBlobID(childData)
	a := format3Record(t, migration.Format3SealPayload{
		Schema: migration.Format3SealSchema, REF: "A", Content: domain.ContentRef{Store: domain.NativeStore, Type: domain.BlobType, ID: sharedID}, Attachments: []domain.Attachment{}, Links: []migration.Format3Link{}, Root: true,
	})
	b := format3Record(t, migration.Format3SealPayload{
		Schema: migration.Format3SealSchema, REF: "B", Content: domain.ContentRef{Store: domain.NativeStore, Type: domain.BlobType, ID: sharedID}, Attachments: []domain.Attachment{}, Links: []migration.Format3Link{}, Root: true,
	})
	roots := []migration.SealRecord{a, b}
	if roots[0].OldSealID.String() > roots[1].OldSealID.String() {
		roots[0], roots[1] = roots[1], roots[0]
	}
	child := format3Record(t, migration.Format3SealPayload{
		Schema: migration.Format3SealSchema, REF: "C", Content: domain.ContentRef{Store: domain.NativeStore, Type: domain.BlobType, ID: childID}, Attachments: []domain.Attachment{},
		Links: []migration.Format3Link{{TargetREF: "A", TargetSeal: a.OldSealID}, {TargetREF: "B", TargetSeal: b.OldSealID}},
	})
	objects := []migration.ObjectRecord{{ID: sharedID, Data: shared}, {ID: childID, Data: childData}}
	if objects[0].ID.String() > objects[1].ID.String() {
		objects[0], objects[1] = objects[1], objects[0]
	}
	dump := migration.LogicalDumpV1{
		Objects: objects, Seals: append(roots, child),
		REFs: []migration.RefRecord{{Name: "A", Head: a.OldSealID}, {Name: "B", Head: b.OldSealID}, {Name: "C", Head: child.OldSealID}},
		Tags: []migration.TagRecord{}, ExcludedObjects: []domain.ObjectID{},
	}
	input, err := migration.EncodeLogicalDumpV1(dump)
	if err != nil {
		t.Fatal(err)
	}
	workDir := t.TempDir()
	if _, err := LoadLogicalV1(context.Background(), workDir, input); err == nil || !strings.Contains(err.Error(), "duplicate dependency seal") {
		t.Fatalf("collapsed duplicate Cause error = %v", err)
	}
	if _, err := os.Lstat(filepath.Join(workDir, ".sealgraph")); !os.IsNotExist(err) {
		t.Fatalf("failed collapsed load published target: %v", err)
	}
}
