package repository

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/mako10k/sealgraph/internal/domain"
)

func TestDumpLogicalV1DeterministicCompleteAndReadOnly(t *testing.T) {
	repo := openTestRepository(t)
	ctx := context.Background()
	rootContent := []byte{'R', 0, '\r', '\n', 0xff}
	if _, err := repo.Add(ctx, AddOptions{REF: "ROOT", Content: rootContent, Root: true}); err != nil {
		t.Fatal(err)
	}
	root, err := repo.Seal(ctx, "ROOT")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := repo.Add(ctx, AddOptions{REF: "design/api", Content: []byte("design"), Dependencies: []Dependency{{REF: "ROOT"}}}); err != nil {
		t.Fatal(err)
	}
	attachmentData := []byte("attachment\x00bytes")
	attachmentID, err := repo.objects.WriteBlob(ctx, attachmentData)
	if err != nil {
		t.Fatal(err)
	}
	candidate, err := repo.candidates.Load("design/api")
	if err != nil {
		t.Fatal(err)
	}
	candidate.Attachments = []domain.Attachment{{
		Name: "evidence.bin", MediaType: "application/octet-stream",
		Blob: domain.ContentRef{Store: domain.NativeStore, Type: domain.BlobType, ID: attachmentID},
	}}
	if err := repo.candidates.Save(candidate); err != nil {
		t.Fatal(err)
	}
	child, err := repo.Seal(ctx, "design/api")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := repo.CreateTag(ctx, "ROOT", "", "accepted"); err != nil {
		t.Fatal(err)
	}
	orphan, err := repo.objects.WriteBlob(ctx, []byte("unreachable object"))
	if err != nil {
		t.Fatal(err)
	}

	before := snapshotDumpTree(t, repo.dir)
	first, err := repo.DumpLogicalV1(ctx)
	if err != nil {
		t.Fatal(err)
	}
	second, err := repo.DumpLogicalV1(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(first, second) {
		t.Fatal("unchanged repository produced different logical dump bytes")
	}
	if len(first) == 0 || first[len(first)-1] != '\n' || bytes.HasSuffix(first, []byte("\n\n")) {
		t.Fatalf("dump does not have exactly one trailing LF: %q", first)
	}
	after := snapshotDumpTree(t, repo.dir)
	if !reflect.DeepEqual(before, after) {
		t.Fatalf("dump mutated repository\nbefore=%v\nafter=%v", before, after)
	}

	var document struct {
		Schema  string `json:"schema"`
		Objects []struct {
			ID         string `json:"id"`
			DataBase64 string `json:"data_base64"`
		} `json:"objects"`
		Seals []struct {
			OldSealID string `json:"old_seal_id"`
		} `json:"seals"`
		REFs []struct {
			Name string `json:"name"`
			Head string `json:"head"`
		} `json:"refs"`
		Tags []struct {
			REF    string `json:"ref"`
			Name   string `json:"name"`
			Target string `json:"target"`
		} `json:"tags"`
		Excluded []string `json:"excluded_objects"`
	}
	if err := json.Unmarshal(first, &document); err != nil {
		t.Fatal(err)
	}
	if document.Schema != "sealgraph/logical-dump/v1" {
		t.Fatalf("schema = %q", document.Schema)
	}
	if len(document.Seals) != 2 || document.Seals[0].OldSealID != root.ID.String() || document.Seals[1].OldSealID != child.ID.String() {
		t.Fatalf("dependency-first seals = %+v", document.Seals)
	}
	if len(document.REFs) != 2 || document.REFs[0].Name != "ROOT" || document.REFs[0].Head != root.ID.String() || document.REFs[1].Name != "design/api" || document.REFs[1].Head != child.ID.String() {
		t.Fatalf("REF records = %+v", document.REFs)
	}
	if len(document.Tags) != 1 || document.Tags[0].REF != "ROOT" || document.Tags[0].Name != "accepted" || document.Tags[0].Target != root.ID.String() {
		t.Fatalf("tag records = %+v", document.Tags)
	}
	if len(document.Excluded) != 1 || document.Excluded[0] != orphan.String() {
		t.Fatalf("excluded objects = %v, want %s", document.Excluded, orphan)
	}
	encodedRoot := base64.StdEncoding.EncodeToString(rootContent)
	foundRootContent := false
	for _, object := range document.Objects {
		if object.ID == root.Payload.Content.ID.String() {
			foundRootContent = object.DataBase64 == encodedRoot
		}
	}
	if !foundRootContent {
		t.Fatalf("root binary content was not exported exactly: %+v", document.Objects)
	}
	foundAttachment := false
	for _, object := range document.Objects {
		if object.ID == attachmentID.String() {
			foundAttachment = object.DataBase64 == base64.StdEncoding.EncodeToString(attachmentData)
		}
	}
	if !foundAttachment {
		t.Fatalf("attachment bytes were not exported exactly: %+v", document.Objects)
	}
}

func TestDumpLogicalV1RejectsEveryCandidateWithoutReadingIt(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		repo := openTestRepository(t)
		if _, err := repo.Add(context.Background(), AddOptions{REF: "ROOT", Content: []byte("candidate"), Root: true}); err != nil {
			t.Fatal(err)
		}
		before := snapshotDumpTree(t, repo.dir)
		output, err := repo.DumpLogicalV1(context.Background())
		if err == nil || !strings.Contains(err.Error(), "working candidate ROOT blocks logical dump") || len(output) != 0 {
			t.Fatalf("candidate dump output=%q err=%v", output, err)
		}
		if after := snapshotDumpTree(t, repo.dir); !reflect.DeepEqual(before, after) {
			t.Fatal("candidate rejection mutated repository")
		}
	})

	t.Run("corrupt", func(t *testing.T) {
		repo := openTestRepository(t)
		path := filepath.Join(repo.dir, "index", "BROKEN")
		if err := os.WriteFile(path, []byte("not candidate JSON or secret-safe output"), 0o600); err != nil {
			t.Fatal(err)
		}
		output, err := repo.DumpLogicalV1(context.Background())
		if err == nil || !strings.Contains(err.Error(), "working candidate BROKEN blocks logical dump") || len(output) != 0 {
			t.Fatalf("corrupt candidate dump output=%q err=%v", output, err)
		}
		if strings.Contains(err.Error(), "not candidate JSON") {
			t.Fatalf("candidate bytes leaked in error: %v", err)
		}
	})
}

func TestDumpLogicalV1RejectsChangedFinalObservation(t *testing.T) {
	repo := openTestRepository(t)
	sealRoot(t, repo, "ROOT", "root")
	output, err := repo.dumpLogicalV1(context.Background(), func() {
		path := filepath.Join(repo.dir, "index", "RACE")
		if err := os.WriteFile(path, []byte("{"), 0o600); err != nil {
			panic(err)
		}
	})
	if err == nil || len(output) != 0 || !strings.Contains(err.Error(), "working candidate RACE blocks logical dump") {
		t.Fatalf("changed observation output=%q err=%v", output, err)
	}
}

func TestDumpLogicalV1RejectsCorruptUnreachableObject(t *testing.T) {
	repo := openTestRepository(t)
	sealRoot(t, repo, "ROOT", "root")
	orphan, err := repo.objects.WriteBlob(context.Background(), []byte("orphan"))
	if err != nil {
		t.Fatal(err)
	}
	path := repo.objects.PathForTesting(orphan)
	if err := os.Chmod(path, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("corrupt"), 0o644); err != nil {
		t.Fatal(err)
	}
	output, err := repo.DumpLogicalV1(context.Background())
	if err == nil || len(output) != 0 || !strings.Contains(err.Error(), orphan.String()) {
		t.Fatalf("corrupt unreachable output=%q err=%v", output, err)
	}
}

func TestDumpLogicalV1RejectsOrphanPhysicalTagScope(t *testing.T) {
	repo := openTestRepository(t)
	root := sealRoot(t, repo, "ROOT", "root")
	path := filepath.Join(repo.dir, "refs", "tags", "GHOST", "accepted")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(root.ID.String()+"\n"), 0o444); err != nil {
		t.Fatal(err)
	}
	output, err := repo.DumpLogicalV1(context.Background())
	if err == nil || len(output) != 0 || !strings.Contains(err.Error(), "has no current REF scope") {
		t.Fatalf("orphan tag output=%q err=%v", output, err)
	}
}

func TestDependencyFirstSealOrderUsesOldIDForDiamondTie(t *testing.T) {
	a := strings.Repeat("a", 64)
	b := strings.Repeat("b", 64)
	c := strings.Repeat("c", 64)
	d := strings.Repeat("d", 64)
	seals := map[string]domain.SealPayload{
		a: {},
		b: {Links: []domain.Link{{TargetSeal: domain.ObjectID{Hex: a}}}},
		c: {Links: []domain.Link{{TargetSeal: domain.ObjectID{Hex: a}}}},
		d: {Links: []domain.Link{{TargetSeal: domain.ObjectID{Hex: b}}, {TargetSeal: domain.ObjectID{Hex: c}}}},
	}
	order, err := dependencyFirstSealOrder(seals)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{a, b, c, d}
	if !reflect.DeepEqual(order, want) {
		t.Fatalf("diamond order = %v, want %v", order, want)
	}
}

type dumpTreeEntry struct {
	Mode os.FileMode
	Data string
}

func snapshotDumpTree(t *testing.T, root string) map[string]dumpTreeEntry {
	t.Helper()
	result := make(map[string]dumpTreeEntry)
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if entry.IsDir() {
			result[relative] = dumpTreeEntry{Mode: info.Mode()}
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		result[relative] = dumpTreeEntry{Mode: info.Mode(), Data: fmt.Sprintf("%x", data)}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return result
}
