package cli

import (
	"strings"
	"testing"
)

func TestCLIFormat7InspectionSchemasAndOriginMembers(t *testing.T) {
	dir := t.TempDir()
	mustRunCLI(t, dir, "init")
	mustRunCLI(t, dir, "add", "root", "--root", "--clear-cause-links", "--content", "root")
	mustRunCLI(t, dir, "seal", "root")
	mustRunCLI(t, dir, "add", "dependent", "--non-root", "--target", "root", "--no-previous", "--content", "dependent")
	mustRunCLI(t, dir, "migrate", "repository", "--from", "5", "--to", "7", "--format", "json")

	commands := [][]string{
		{"show", "root"}, {"candidate", "show", "dependent"},
		{"candidate", "compare", "dependent"}, {"graph"}, {"impact", "root"},
		{"log", "root"}, {"linklog", "root"}, {"compare", "root", "root"}, {"fsck"},
	}
	names := []string{"show", "candidate-show", "candidate-compare", "graph", "impact", "log", "linklog", "compare", "fsck"}
	for i, command := range commands {
		args := append(append([]string{}, command...), "--format", "json")
		code, output, stderr := runCLI(t, dir, nil, args...)
		if code != 0 || stderr != "" || decodeCLIJSON(t, output)["schema"] != "sealgraph/"+names[i]+"/v4" {
			t.Fatalf("%v code=%d output=%q stderr=%q", args, code, output, stderr)
		}
	}

	code, output, stderr := runCLI(t, dir, nil, "show", "root", "--format", "json")
	if code != 0 || stderr != "" {
		t.Fatalf("show code=%d output=%q stderr=%q", code, output, stderr)
	}
	if !strings.Contains(output, `"cause_links":[],"origin_map_id":null`) {
		t.Fatalf("show lacks nullable origin suffix: %s", output)
	}
	if strings.Index(output, `"cause_links"`) > strings.Index(output, `"origin_map_id"`) {
		t.Fatalf("origin_map_id is not appended after cause_links: %s", output)
	}

	code, output, stderr = runCLI(t, dir, nil, "candidate", "show", "dependent", "--format", "json")
	if code != 0 || stderr != "" || !strings.Contains(output, `"cause_links":[`) || !strings.Contains(output, `,"origin_map_id":null`) {
		t.Fatalf("candidate origin suffix code=%d output=%q stderr=%q", code, output, stderr)
	}
	code, output, stderr = runCLI(t, dir, nil, "candidate", "compare", "dependent", "--format", "json")
	if code != 0 || stderr != "" || !strings.Contains(output, `"origin":{"changed":false,"before":null,"after":null}`) {
		t.Fatalf("candidate null origin change code=%d output=%q stderr=%q", code, output, stderr)
	}

	code, output, stderr = runCLI(t, dir, nil, "compare", "root", "root", "--format", "json")
	if code != 0 || stderr != "" || !strings.Contains(output, `"cause_links":{"changed":false`) || !strings.Contains(output, `,"origin":{"changed":false,"before":null,"after":null}`) {
		t.Fatalf("compare origin change code=%d output=%q stderr=%q", code, output, stderr)
	}
}

func TestCLIFormat7LeavesFormat5And6InspectionSchemasUnchanged(t *testing.T) {
	dir := t.TempDir()
	mustRunCLI(t, dir, "init")
	mustRunCLI(t, dir, "add", "root", "--root", "--clear-cause-links", "--content", "root")
	mustRunCLI(t, dir, "seal", "root")
	code, output, stderr := runCLI(t, dir, nil, "show", "root", "--format", "json")
	if code != 0 || stderr != "" || decodeCLIJSON(t, output)["schema"] != "sealgraph/show/v2" {
		t.Fatalf("format5 code=%d output=%q stderr=%q", code, output, stderr)
	}
	mustRunCLI(t, dir, "migrate", "repository", "--from", "5", "--to", "6", "--format", "json")
	code, output, stderr = runCLI(t, dir, nil, "show", "root", "--format", "json")
	if code != 0 || stderr != "" || decodeCLIJSON(t, output)["schema"] != "sealgraph/show/v3" {
		t.Fatalf("format6 code=%d output=%q stderr=%q", code, output, stderr)
	}
}

func TestCLIFormat7InspectionCarriesOriginMapID(t *testing.T) {
	dir := traceCLIFixture(t)
	writeTraceFixture(t, dir)
	mustRunCLI(t, dir, "add", "root", "--root", "--clear-cause-links", "--content", "XYZ-UV")
	if code, output, stderr := runCLI(t, dir, nil, "trace", "set", "root", "--recipe", "recipe.json", "--format", "json"); code != 0 || output == "" || stderr == "" {
		t.Fatalf("trace set code=%d output=%q stderr=%q", code, output, stderr)
	}
	code, output, stderr := runCLI(t, dir, nil, "candidate", "show", "root", "--format", "json")
	if code != 0 || stderr != "" || !strings.Contains(output, `"origin_map_id":"`) {
		t.Fatalf("candidate origin code=%d output=%q stderr=%q", code, output, stderr)
	}
	code, output, stderr = runCLI(t, dir, nil, "candidate", "compare", "root", "--format", "json")
	if code != 0 || stderr != "" || !strings.Contains(output, `"origin":{"changed":true,"before":null,"after":"`) {
		t.Fatalf("candidate new origin change code=%d output=%q stderr=%q", code, output, stderr)
	}
	mustRunCLI(t, dir, "seal", "root")
	code, output, stderr = runCLI(t, dir, nil, "show", "root", "--format", "json")
	if code != 0 || stderr != "" || !strings.Contains(output, `"origin_map_id":"`) {
		t.Fatalf("seal origin code=%d output=%q stderr=%q", code, output, stderr)
	}
}
