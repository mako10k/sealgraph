package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func nativeCLISnapshot(t *testing.T) []byte {
	t.Helper()
	source := traceCLIFixture(t)
	mustRunCLI(t, source, "add", "root", "--root", "--clear-cause-links", "--content", "native")
	code, snapshot, stderr := runCLI(t, source, nil, "dump", "--format", "native-blobs-v1")
	if code != 0 || !strings.Contains(stderr, "opaque orphan Blobs") {
		t.Fatalf("dump code=%d stderr=%q", code, stderr)
	}
	return []byte(snapshot)
}

func TestCLINativeLoadPublishesReceiptFromNamedFile(t *testing.T) {
	snapshot := nativeCLISnapshot(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "snapshot.json")
	if err := os.WriteFile(path, snapshot, 0o644); err != nil {
		t.Fatal(err)
	}
	code, output, stderr := runCLI(t, dir, nil, "load", "--format", "native-blobs-v1", "--file", "snapshot.json", "--max-input-bytes", "10485760")
	if code != 0 || stderr != "" {
		t.Fatalf("load code=%d output=%q stderr=%q", code, output, stderr)
	}
	var receipt map[string]any
	if err := json.Unmarshal([]byte(output), &receipt); err != nil {
		t.Fatal(err)
	}
	if receipt["schema"] != "sealgraph/native-load/v1" || receipt["result"] != "LOADED" || receipt["repository_format"] != float64(7) {
		t.Fatalf("receipt=%s", output)
	}
	if !strings.HasSuffix(output, "\n") || strings.Contains(output, " ") {
		t.Fatalf("receipt is not compact JSON+LF: %q", output)
	}
}

func TestCLINativeLoadRejectsInputOverBoundBeforePublication(t *testing.T) {
	snapshot := nativeCLISnapshot(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "snapshot.json")
	if err := os.WriteFile(path, snapshot, 0o644); err != nil {
		t.Fatal(err)
	}
	code, stdout, stderr := runCLI(t, dir, nil, "load", "--format", "native-blobs-v1", "--file", "snapshot.json", "--max-input-bytes", "1")
	if code != 1 || stdout != "" || !strings.Contains(stderr, "exceeds --max-input-bytes") {
		t.Fatalf("code=%d stdout=%q stderr=%q", code, stdout, stderr)
	}
	if _, err := os.Lstat(filepath.Join(dir, ".sealgraph")); !os.IsNotExist(err) {
		t.Fatalf("target was published after bounded input rejection: %v", err)
	}
}

func TestCLINativeLoadOutputFailureReportsCommittedState(t *testing.T) {
	snapshot := nativeCLISnapshot(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "snapshot.json")
	if err := os.WriteFile(path, snapshot, 0o644); err != nil {
		t.Fatal(err)
	}
	var stderr bytes.Buffer
	code := runStandaloneAtWithInput(dir, []string{"load", "--format", "native-blobs-v1", "--file", "snapshot.json", "--max-input-bytes", "10485760"}, bytes.NewReader(nil), errorWriter{}, &stderr)
	if code != 3 || !strings.Contains(stderr.String(), "LOAD_COMMITTED_OUTPUT_UNDELIVERED") || !strings.Contains(stderr.String(), "do not retry load") {
		t.Fatalf("code=%d stderr=%q", code, stderr.String())
	}
	if _, err := os.Stat(filepath.Join(dir, ".sealgraph", "config")); err != nil {
		t.Fatalf("committed target missing after output failure: %v", err)
	}
}
