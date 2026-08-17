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

func TestCLIFormat3DumpIsAbsentAndTagsMoveWithREF(t *testing.T) {
	dir := t.TempDir()
	if code, _, stderr := runCLI(t, dir, nil, "init"); code != 0 {
		t.Fatal(stderr)
	}
	if code, stdout, stderr := runCLI(t, dir, nil, "dump", "--format", "logical-v1"); code != 2 || stdout != "" || !strings.Contains(stderr, `unknown command "dump"`) {
		t.Fatalf("dump code=%d stdout=%s stderr=%s", code, stdout, stderr)
	}
	for _, args := range [][]string{{"graph"}, {"stale", "--scan"}} {
		if code, _, stderr := runCLI(t, dir, nil, args...); code != 0 || stderr != "" {
			t.Fatalf("%v code=%d stderr=%s", args, code, stderr)
		}
	}
	mustRunCLI(t, dir, "add", "root", "--root", "--content", "root")
	rootID := mustSealCLI(t, dir, "root")
	if stdout := mustRunCLI(t, dir, "tag", "root", "reviewed/1.0"); !strings.Contains(stdout, rootID) {
		t.Fatalf("tag create stdout=%s", stdout)
	}
	if code, stdout, stderr := runCLI(t, dir, nil, "tag", "@"+rootID[:12], "global"); code != 2 || stdout != "" || !strings.Contains(stderr, "no REF scope") {
		t.Fatalf("unscoped tag code=%d stdout=%s stderr=%s", code, stdout, stderr)
	}
	if stdout := mustRunCLI(t, dir, "tag", "root"); !strings.Contains(stdout, `TAG root "reviewed/1.0" `+rootID) {
		t.Fatalf("tag list stdout=%s", stdout)
	}
	if stdout := mustRunCLI(t, dir, "mv", "root", "archive/root"); !strings.Contains(stdout, "tags=1") {
		t.Fatalf("mv stdout=%s", stdout)
	}
	if code, stdout, stderr := runCLI(t, dir, nil, "show", "root@reviewed/1.0"); code != 1 || stdout != "" || !strings.Contains(stderr, "REF not found") {
		t.Fatalf("old tag scope code=%d stdout=%s stderr=%s", code, stdout, stderr)
	}
	if code, stdout, stderr := runCLI(t, dir, nil, "show", "archive/root@reviewed/1.0"); code != 0 || !strings.Contains(stdout, rootID) || stderr != "" {
		t.Fatalf("moved tag scope code=%d stdout=%s stderr=%s", code, stdout, stderr)
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

func mustRunCLI(t *testing.T, dir string, args ...string) string {
	t.Helper()
	code, stdout, stderr := runCLI(t, dir, nil, args...)
	if code != 0 || stderr != "" {
		t.Fatalf("%v code=%d stdout=%q stderr=%q", args, code, stdout, stderr)
	}
	return stdout
}

func mustSealCLI(t *testing.T, dir, ref string) string {
	t.Helper()
	stdout := mustRunCLI(t, dir, "seal", ref)
	return strings.TrimSpace(strings.TrimPrefix(stdout, "SEALED "+ref+" "))
}

type cliRevisionFixture struct {
	dir          string
	root1, root2 string
}

func newCLIRevisionFixture(t *testing.T) cliRevisionFixture {
	t.Helper()
	dir := t.TempDir()
	mustRunCLI(t, dir, "init")
	mustRunCLI(t, dir, "add", "root", "--root", "--content", "root-v1")
	root1 := mustSealCLI(t, dir, "root")
	mustRunCLI(t, dir, "add", "middle", "--content", "middle", "--depend-on", "root")
	mustSealCLI(t, dir, "middle")
	mustRunCLI(t, dir, "add", "leaf", "--content", "leaf", "--depend-on", "middle")
	mustSealCLI(t, dir, "leaf")
	mustRunCLI(t, dir, "add", "root", "--root", "--content", "root-v2")
	root2 := mustSealCLI(t, dir, "root")
	return cliRevisionFixture{dir: dir, root1: root1, root2: root2}
}

func TestCLIRevisionGraphDeriveStaleHistoryAndImpact(t *testing.T) {
	fixture := newCLIRevisionFixture(t)
	verifyCLIStaleAndImpact(t, fixture)
	verifyCLIDeriveAndHistory(t, fixture)
}

func verifyCLIStaleAndImpact(t *testing.T, fixture cliRevisionFixture) {
	t.Helper()
	if stdout := mustRunCLI(t, fixture.dir, "stale", "--frontier", "--refs-only", "--scan"); stdout != "middle\n" {
		t.Fatalf("stale stdout=%q", stdout)
	}
	stdout := mustRunCLI(t, fixture.dir, "impact", "--all-paths", "--max-paths", "1", "@"+fixture.root2)
	if !strings.Contains(stdout, "SOURCE "+fixture.root2) || !strings.Contains(stdout, "refs=leaf") {
		t.Fatalf("impact stdout=%q", stdout)
	}
	if code, stdout, _ := runCLI(t, fixture.dir, nil, "impact", "--max-paths", "1", "@"+fixture.root2); code != 2 || stdout != "" {
		t.Fatalf("invalid impact code=%d stdout=%q", code, stdout)
	}
}

func verifyCLIDeriveAndHistory(t *testing.T, fixture cliRevisionFixture) {
	t.Helper()
	stdout := mustRunCLI(t, fixture.dir, "derive", "preserved", "--from", "@"+fixture.root1)
	if !strings.Contains(stdout, "parent="+fixture.root1) {
		t.Fatalf("derive stdout=%q", stdout)
	}
	mustSealCLI(t, fixture.dir, "preserved")
	stdout = mustRunCLI(t, fixture.dir, "diff", "root")
	if !strings.Contains(stdout, "FROM "+fixture.root1) || !strings.Contains(stdout, "TO "+fixture.root2) {
		t.Fatalf("diff stdout=%q", stdout)
	}
	stdout = mustRunCLI(t, fixture.dir, "log", "root")
	if strings.Count(stdout, "SEAL ") != 2 {
		t.Fatalf("log stdout=%q", stdout)
	}
}
