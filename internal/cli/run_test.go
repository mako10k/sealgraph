package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/mako10k/sealgraph/internal/domain"
	"github.com/mako10k/sealgraph/internal/migration"
)

func runCLI(t *testing.T, dir string, stdin []byte, args ...string) (int, string, string) {
	t.Helper()
	var stdout, stderr bytes.Buffer
	code := runStandaloneAtWithInput(dir, args, bytes.NewReader(stdin), &stdout, &stderr)
	return code, stdout.String(), stderr.String()
}

func TestCLIFormat4RootAndSelectorShow(t *testing.T) {
	dir := t.TempDir()
	if code, _, stderr := runCLI(t, dir, nil, "init"); code != 0 {
		t.Fatalf("init code=%d stderr=%s", code, stderr)
	}
	if code, stdout, stderr := runCLI(t, dir, nil, "add", "root", "--root", "--content", "material"); code != 0 || !strings.Contains(stdout, "CANDIDATE root") {
		t.Fatalf("add code=%d stdout=%s stderr=%s", code, stdout, stderr)
	}
	code, stdout, stderr := runCLI(t, dir, nil, "candidate", "show", "root")
	if code != 0 || !strings.Contains(stdout, "PARENT_REVISION -") || !strings.Contains(stdout, "EXPECTED_REF_HEAD -") || !strings.Contains(stdout, "EXPECTED_HEAD_STATE EXPECTED_ABSENT") {
		t.Fatalf("candidate show code=%d stdout=%s stderr=%s", code, stdout, stderr)
	}
	code, stdout, stderr = runCLI(t, dir, nil, "seal", "root")
	if code != 0 || !strings.HasPrefix(stdout, "SEALED root ") {
		t.Fatalf("seal code=%d stdout=%s stderr=%s", code, stdout, stderr)
	}
	id := strings.TrimSpace(strings.TrimPrefix(stdout, "SEALED root "))
	code, stdout, stderr = runCLI(t, dir, nil, "show", "@"+id[:12])
	if code != 0 || !strings.Contains(stdout, "CURRENT_REFS root") || !strings.Contains(stdout, "PARENT_REVISION -") || strings.Contains(stdout, "target_ref") {
		t.Fatalf("show code=%d stdout=%s stderr=%s", code, stdout, stderr)
	}
}

func cliLogicalDump(t *testing.T) []byte {
	t.Helper()
	data := []byte("migrated")
	contentID := domain.ComputeNativeBlobID(data)
	payload := migration.Format3SealPayload{
		Schema: migration.Format3SealSchema, REF: "ROOT",
		Content:     domain.ContentRef{Store: domain.NativeStore, Type: domain.BlobType, ID: contentID},
		Attachments: []domain.Attachment{}, Links: []migration.Format3Link{}, Root: true,
	}
	sealBytes, err := migration.EncodeFormat3Seal(payload)
	if err != nil {
		t.Fatal(err)
	}
	oldID := domain.ComputeNativeBlobID(sealBytes)
	dump, err := migration.EncodeLogicalDumpV1(migration.LogicalDumpV1{
		Objects: []migration.ObjectRecord{{ID: contentID, Data: data}},
		Seals:   []migration.SealRecord{{OldSealID: oldID, Payload: payload}},
		REFs:    []migration.RefRecord{{Name: "ROOT", Head: oldID}},
		Tags:    []migration.TagRecord{}, ExcludedObjects: []domain.ObjectID{},
	})
	if err != nil {
		t.Fatal(err)
	}
	return dump
}

func TestCLILoadPublishesOnlyAfterCanonicalInput(t *testing.T) {
	dir := t.TempDir()
	input := cliLogicalDump(t)
	code, stdout, stderr := runCLI(t, dir, input, "load", "--format", "logical-v1")
	if code != 0 || !strings.Contains(stdout, `"schema":"sealgraph/logical-load-receipt/v1"`) || stderr != "" {
		t.Fatalf("load code=%d stdout=%s stderr=%s", code, stdout, stderr)
	}
	if code, stdout, stderr := runCLI(t, dir, nil, "show", "ROOT"); code != 0 || !strings.Contains(stdout, `CONTENT_PREVIEW "migrated"`) {
		t.Fatalf("show loaded code=%d stdout=%s stderr=%s", code, stdout, stderr)
	}
	other := t.TempDir()
	code, stdout, stderr = runCLI(t, other, bytes.TrimSuffix(input, []byte{'\n'}), "load", "--format", "logical-v1")
	if code != 1 || stdout != "" || !strings.Contains(stderr, "not canonical") {
		t.Fatalf("noncanonical load code=%d stdout=%s stderr=%s", code, stdout, stderr)
	}
}

func TestCLIFormat3DumpAndDeferredSurfacesFailExplicitly(t *testing.T) {
	dir := t.TempDir()
	if code, _, stderr := runCLI(t, dir, nil, "init"); code != 0 {
		t.Fatal(stderr)
	}
	if code, stdout, stderr := runCLI(t, dir, nil, "dump", "--format", "logical-v1"); code != 2 || stdout != "" || !strings.Contains(stderr, `unknown command "dump"`) {
		t.Fatalf("dump code=%d stdout=%s stderr=%s", code, stdout, stderr)
	}
	for _, args := range [][]string{{"graph"}, {"stale", "--scan"}, {"tag", "root"}} {
		code, stdout, stderr := runCLI(t, dir, nil, args...)
		if code != 1 || stdout != "" || (!strings.Contains(stderr, "not implemented") && !strings.Contains(stderr, "tag namespace")) {
			t.Fatalf("%v code=%d stdout=%s stderr=%s", args, code, stdout, stderr)
		}
	}
}

func TestCLILoadRequiresSingleKnownFormatAndAbsentTarget(t *testing.T) {
	input := cliLogicalDump(t)
	for _, args := range [][]string{{"load"}, {"load", "--format", "other"}, {"load", "--format", "logical-v1", "extra"}, {"load", "--format", "logical-v1", "--format", "logical-v1"}} {
		code, stdout, _ := runCLI(t, t.TempDir(), input, args...)
		if code != 2 || stdout != "" {
			t.Fatalf("%v code=%d stdout=%s", args, code, stdout)
		}
	}
	dir := t.TempDir()
	if code, _, stderr := runCLI(t, dir, nil, "init"); code != 0 {
		t.Fatal(stderr)
	}
	if code, stdout, stderr := runCLI(t, dir, input, "load", "--format", "logical-v1"); code != 1 || stdout != "" || !strings.Contains(stderr, "already exists") {
		t.Fatalf("existing load code=%d stdout=%s stderr=%s", code, stdout, stderr)
	}
}
