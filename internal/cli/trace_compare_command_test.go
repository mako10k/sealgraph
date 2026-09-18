package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestParseTraceCompareArgs(t *testing.T) {
	var stdout bytes.Buffer
	options, err := parseTraceCompareArgs([]string{"--ref", "root", "--max-graph-visits", "12", "--estimate", "--format", "json"}, &stdout)
	if err != nil || !options.ref.set || options.ref.value != "root" || options.seal.set || options.maxGraphVisits != 12 || !options.estimate || !options.output.JSON {
		t.Fatalf("options=%+v err=%v", options, err)
	}
	options, err = parseTraceCompareArgs([]string{"--seal", "@abcd", "--max-graph-visits", "1"}, &stdout)
	if err != nil || !options.seal.set || options.seal.value != "@abcd" || options.estimate {
		t.Fatalf("seal options=%+v err=%v", options, err)
	}
}

func TestParseTraceCompareArgsRejectsInvalidSelectionAndBudget(t *testing.T) {
	var stdout bytes.Buffer
	for _, args := range [][]string{
		{"--ref", "root"}, {"--ref", "root", "--seal", "@abcd", "--max-graph-visits", "1"},
		{"--seal", "root", "--max-graph-visits", "1"}, {"--ref", "root", "--max-graph-visits", "0"},
		{"--ref", "root", "--max-graph-visits", "nan"}, {"--ref", "root", "--max-graph-visits", "1", "extra"},
		{"--ref", "root", "--max-alignment-cells", "1", "--max-graph-visits", "1"},
	} {
		if _, err := parseTraceCompareArgs(args, &stdout); err == nil {
			t.Fatalf("accepted invalid args %q", strings.Join(args, " "))
		}
	}
}
