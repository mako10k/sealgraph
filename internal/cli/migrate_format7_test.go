package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestCLIMigrateRepositoryFormat7ReceiptAndExactPairs(t *testing.T) {
	dir := t.TempDir()
	mustRunCLI(t, dir, "init")
	mustRunCLI(t, dir, "add", "root", "--root", "--clear-cause-links", "--content", "root")
	mustRunCLI(t, dir, "seal", "root")

	code, output, stderr := runCLI(t, dir, nil, "migrate", "repository", "--from", "5", "--to", "7", "--format", "json")
	if code != 0 || stderr != "" {
		t.Fatalf("5->7 code=%d stderr=%q", code, stderr)
	}
	if want := `{"schema":"sealgraph/repository-migrate/v2","from_format":5,"to_format":7,"result":"MIGRATED","retained_seals_v5":1,"retained_seals_v6":0,"retained_candidates_v5":0,"retained_candidates_v6":0}` + "\n"; output != want {
		t.Fatalf("5->7 receipt=%q, want %q", output, want)
	}
	if code, _, stderr := runCLI(t, dir, nil, "migrate", "repository", "--from", "5", "--to", "7", "--format", "json"); code != 1 || !strings.Contains(stderr, "format 7") {
		t.Fatalf("second migration code=%d stderr=%q", code, stderr)
	}
	code, fsck, stderr := runCLI(t, dir, nil, "fsck", "--format", "json")
	if code != 0 || stderr != "" || !strings.Contains(fsck, `"schema":"sealgraph/fsck/v4"`) || !strings.Contains(fsck, `"origin_maps":0,"source_snapshots":0`) {
		t.Fatalf("format7 fsck code=%d output=%q stderr=%q", code, fsck, stderr)
	}
}

func TestCLIMigrateRepositoryFormat7From6AndCLIContract(t *testing.T) {
	dir := t.TempDir()
	mustRunCLI(t, dir, "init")
	mustRunCLI(t, dir, "add", "root", "--root", "--clear-cause-links", "--content", "root")
	mustRunCLI(t, dir, "seal", "root")
	mustRunCLI(t, dir, "migrate", "repository", "--from", "5", "--to", "6", "--format", "json")
	code, output, stderr := runCLI(t, dir, nil, "migrate", "repository", "--from", "6", "--to", "7", "--format", "json")
	if code != 0 || stderr != "" || !strings.Contains(output, `"schema":"sealgraph/repository-migrate/v2"`) || !strings.Contains(output, `"from_format":6`) {
		t.Fatalf("6->7 code=%d output=%q stderr=%q", code, output, stderr)
	}
	for _, args := range [][]string{
		{"migrate", "repository", "--from", "6", "--to", "6", "--format", "json"},
		{"migrate", "repository", "--from", "7", "--to", "7", "--format", "json"},
		{"migrate", "repository", "--from", "5", "--to", "5", "--format", "json"},
	} {
		dir := t.TempDir()
		mustRunCLI(t, dir, "init")
		if code, _, stderr := runCLI(t, dir, nil, args...); code != 2 || !strings.Contains(stderr, "requires exactly") {
			t.Fatalf("invalid pair %v code=%d stderr=%q", args, code, stderr)
		}
	}
}

func TestCLIMigrateRepositoryFormat7ReportsCommittedOutputFailure(t *testing.T) {
	dir := t.TempDir()
	mustRunCLI(t, dir, "init")
	mustRunCLI(t, dir, "add", "root", "--root", "--clear-cause-links", "--content", "root")
	var stderr bytes.Buffer
	code := runStandaloneAtWithInput(dir, []string{"migrate", "repository", "--from", "5", "--to", "7", "--format", "json"}, bytes.NewReader(nil), errorWriter{}, &stderr)
	if code != 3 || !strings.Contains(stderr.String(), "MIGRATION_COMMITTED_OUTPUT_UNDELIVERED") || !strings.Contains(stderr.String(), "already format 7") || !strings.Contains(stderr.String(), "do not retry") {
		t.Fatalf("code=%d stderr=%q", code, stderr.String())
	}
	code, fsck, fsckStderr := runCLI(t, dir, nil, "fsck", "--format", "json")
	if code != 0 || fsckStderr != "" {
		t.Fatalf("fsck code=%d output=%q stderr=%q", code, fsck, fsckStderr)
	}
	if value := decodeCLIJSON(t, fsck); value["schema"] != "sealgraph/fsck/v4" || value["seals_v5"] != float64(0) || value["seals_v6"] != float64(0) {
		t.Fatalf("fsck recovery inventory=%#v", value)
	}
}

func TestCLIMigrateFormat7HelpAndCompletion(t *testing.T) {
	if output := mustRunCLI(t, t.TempDir(), "help", "migrate", "repository"); !strings.Contains(output, "--from 5 --to 7") || !strings.Contains(output, "--from 6 --to 7") {
		t.Fatalf("help=%q", output)
	}
	if output := mustRunCLI(t, t.TempDir(), "__completion", "--bash", "migrate", "repository", "--from", ""); output != "__sealgraph_completion_mode=plain\n5\n6\n" {
		t.Fatalf("from completion=%q", output)
	}
	if output := mustRunCLI(t, t.TempDir(), "__completion", "--bash", "migrate", "repository", "--to", ""); output != "__sealgraph_completion_mode=plain\n6\n7\n" {
		t.Fatalf("to completion=%q", output)
	}
}
