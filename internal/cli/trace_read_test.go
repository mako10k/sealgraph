package cli

// R: verify source-reading selection, exact bytes, bounded context and explicit
// uncertainty without confusing current file observations with saved evidence.

import (
	"bytes"
	"encoding/base64"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func traceReadArgs(view string) []string {
	return []string{"trace", "read", "--ref", "root", "--baseline", "candidate", "--run-index", "0", "--view", view}
}

func traceReadJSON(t *testing.T, dir, view string, extra ...string) map[string]any {
	t.Helper()
	args := append(traceReadArgs(view), extra...)
	args = append(args, "--format", "json")
	code, out, stderr := runCLI(t, dir, nil, args...)
	if code != 0 || stderr != "" {
		t.Fatalf("trace read: %d %q %q", code, out, stderr)
	}
	return decodeCLIJSON(t, out)
}

func readExcerpt(t *testing.T, doc map[string]any) []byte {
	t.Helper()
	value, err := base64.StdEncoding.DecodeString(doc["excerpt_bytes"].(string))
	if err != nil {
		t.Fatal(err)
	}
	return value
}

func TestTraceReadSavedContextAndCurrentRepresentative(t *testing.T) {
	source := "first\nbefore\nquoted\nafter\nlast\n"
	dir := traceOccurrencesFixture(t, source, "quoted", 13)
	saved := traceReadJSON(t, dir, "snapshot", "--context-lines", "1")
	if saved["schema"] != "sealgraph/trace-read/v1" || saved["state"] != "SAVED_RANGE" || saved["selected_start"] != float64(13) {
		t.Fatalf("saved=%v", saved)
	}
	if string(readExcerpt(t, saved)) != "before\nquoted\nafter\n" {
		t.Fatalf("excerpt=%q", readExcerpt(t, saved))
	}
	if err := os.WriteFile(filepath.Join(dir, "source.txt"), []byte("quoted\nquoted\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	current := traceReadJSON(t, dir, "current")
	if current["state"] != "PRESENT" || current["selected_start"] != float64(0) {
		t.Fatalf("current=%v", current)
	}
	if !strings.Contains(current["limitation"].(string), "not proof of historical identity") {
		t.Fatalf("missing limitation: %v", current)
	}
	savedAgain := traceReadJSON(t, dir, "snapshot")
	if !bytes.Equal(readExcerpt(t, savedAgain), []byte(source)) {
		t.Fatalf("saved source changed: %v", savedAgain)
	}
}

func TestTraceReadExactRawBytesAndUnavailableCurrent(t *testing.T) {
	source := "prefix\x00\r\nquote\r\nsuffix"
	dir := traceOccurrencesFixture(t, source, "quote", 9)
	code, out, stderr := runCLI(t, dir, nil, append(traceReadArgs("snapshot"), "--raw-source")...)
	if code != 0 || stderr != "" || out != source {
		t.Fatalf("raw: %d %q %q", code, out, stderr)
	}
	if err := os.Remove(filepath.Join(dir, "source.txt")); err != nil {
		t.Fatal(err)
	}
	doc := traceReadJSON(t, dir, "current")
	if doc["state"] != "INCOMPLETE" || doc["reason"] != "SOURCE_READ_FAILED" || doc["selected_start"] != nil {
		t.Fatalf("current=%v", doc)
	}
	code, out, stderr = runCLI(t, dir, nil, append(traceReadArgs("current"), "--raw-source")...)
	if code == 0 || out != "" || !strings.Contains(stderr, "unconfirmed") {
		t.Fatalf("missing current: %d %q %q", code, out, stderr)
	}
	doc = traceReadJSON(t, dir, "snapshot")
	if doc["state"] != "SAVED_RANGE" {
		t.Fatalf("snapshot unavailable: %v", doc)
	}
}

func TestTraceReadAbsentStillAllowsExplicitFullCurrent(t *testing.T) {
	dir := traceOccurrencesFixture(t, "before quote after", "quote", 7)
	if err := os.WriteFile(filepath.Join(dir, "source.txt"), []byte("other bytes"), 0o600); err != nil {
		t.Fatal(err)
	}
	doc := traceReadJSON(t, dir, "current")
	if doc["state"] != "ABSENT_EXACT" || doc["selected_start"] != nil || len(readExcerpt(t, doc)) != 0 {
		t.Fatalf("absent=%v", doc)
	}
	code, out, stderr := runCLI(t, dir, nil, append(traceReadArgs("current"), "--raw-source")...)
	if code != 0 || out != "other bytes" || stderr != "" {
		t.Fatalf("current raw: %d %q %q", code, out, stderr)
	}
}

func TestTraceReadBoundsAndFullSourceRecovery(t *testing.T) {
	source := strings.Repeat("p", 1024*1024) + "quoted" + strings.Repeat("s", 1024*1024)
	dir := traceOccurrencesFixture(t, source, "quoted", 1024*1024)
	doc := traceReadJSON(t, dir, "snapshot", "--max-bytes", "32")
	excerpt := readExcerpt(t, doc)
	if len(excerpt) > 32 || !bytes.Contains(excerpt, []byte("quoted")) || doc["truncated"] != true || doc["quote_truncated"] != false {
		t.Fatalf("bounded=%v", doc)
	}
	code, out, stderr := runCLI(t, dir, nil, append(traceReadArgs("snapshot"), "--raw-source")...)
	if code != 0 || out != source || stderr != "" {
		t.Fatalf("raw recovery code=%d bytes=%d stderr=%q", code, len(out), stderr)
	}
}

func TestTraceReadHeadAndImmutableDoNotFallBackToCandidate(t *testing.T) {
	dir := traceOccurrencesFixture(t, "whole quote original", "quote", 6)
	code, out, stderr := runCLI(t, dir, nil, "trace", "read", "--ref", "root", "--baseline", "head", "--run-index", "0", "--view", "snapshot")
	if code == 0 || out != "" || stderr == "" {
		t.Fatalf("missing head: %d %q %q", code, out, stderr)
	}
	mustRunCLI(t, dir, "seal", "root")
	head := traceReadArgs("snapshot")
	head[5] = "head"
	code, out, stderr = runCLI(t, dir, nil, append(head, "--format", "json")...)
	if code != 0 || stderr != "" {
		t.Fatalf("head: %d %q %q", code, out, stderr)
	}
	doc := decodeCLIJSON(t, out)
	baseline := doc["selection"].(map[string]any)["baseline"].(map[string]any)
	id := baseline["seal_id"].(string)
	code, out, stderr = runCLI(t, dir, nil, "trace", "read", "--seal", "@"+id, "--run-index", "0", "--view", "snapshot", "--raw-source")
	if code != 0 || out != "whole quote original" || stderr != "" {
		t.Fatalf("immutable: %d %q %q", code, out, stderr)
	}
}

func TestTraceReadArgumentAndFormatBoundaries(t *testing.T) {
	for _, extra := range [][]string{{"--format", "json", "--raw-source"}, {"--context-lines", "-1"}, {"--context-lines", "1001"}, {"--max-bytes", "0"}, {"--max-bytes", "1048577"}, {"--context-lines", "1", "--context-lines", "2"}, {"--limit", "1"}} {
		code, out, stderr := runCLI(t, t.TempDir(), nil, append(traceReadArgs("snapshot"), extra...)...)
		if code != 2 || out != "" || stderr == "" {
			t.Fatalf("%v: %d %q %q", extra, code, out, stderr)
		}
	}
	code, out, stderr := runCLI(t, t.TempDir(), nil, traceReadArgs("both")...)
	if code != 2 || out != "" || !strings.Contains(stderr, "snapshot or current") {
		t.Fatalf("both: %d %q %q", code, out, stderr)
	}
}

func TestTraceReadUnboundAndReadOnlyInventory(t *testing.T) {
	dir := traceOccurrencesFixture(t, "before quote after", "quote", 7)
	capture := func(args ...string) string {
		t.Helper()
		code, out, stderr := runCLI(t, dir, nil, args...)
		if code != 0 {
			t.Fatalf("capture %v: %d %q", args, code, stderr)
		}
		return out
	}
	beforeDump := capture("dump", "--format", "native-blobs-v1")
	beforeCandidate := capture("candidate", "show", "root", "--format", "json")
	traceReadJSON(t, dir, "snapshot")
	traceReadJSON(t, dir, "current")
	if capture("dump", "--format", "native-blobs-v1") != beforeDump || capture("candidate", "show", "root", "--format", "json") != beforeCandidate {
		t.Fatal("reading source changed canonical inventory or Candidate")
	}
	mustRunCLI(t, dir, "trace", "source", "unbind", "source-A", "--from", "source.txt")
	doc := traceReadJSON(t, dir, "current")
	if doc["state"] != "INCOMPLETE" || doc["reason"] != "SOURCE_UNBOUND" || doc["source_byte_length"] != nil {
		t.Fatalf("unbound=%v", doc)
	}
}
