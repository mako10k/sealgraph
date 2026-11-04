package cli

// R: verify text inspection preserves exact selection and bounded, non-mutating output.

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCompareTextReadsChangesBeyondPreview(t *testing.T) {
	dir := t.TempDir()
	mustRunCLI(t, dir, "init")
	before := strings.Repeat("unchanged prefix\n", 40) + "old detail\n"
	mustRunCLI(t, dir, "add", "root", "--root", "--clear-cause-links", "--content", before)
	old := mustSealCLI(t, dir, "root")
	after := strings.Repeat("unchanged prefix\n", 40) + "new detail\n"
	mustRunCLI(t, dir, "add", "root", "--content", after)
	current := mustSealCLI(t, dir, "root")
	manifest := filepath.Join(dir, ".sealgraph", "refs", "seals", "root", ".ref")
	original, err := os.ReadFile(manifest)
	if err != nil {
		t.Fatal(err)
	}
	text := mustRunCLI(t, dir, "compare", "@"+old, "root", "--text")
	for _, want := range []string{"TEXT COMPARISON", `- "old detail\n"`, `+ "new detail\n"`, old, current, "40 unchanged lines", "--raw-content"} {
		if !strings.Contains(text, want) {
			t.Fatalf("missing %q in %s", want, text)
		}
	}
	unchanged, err := os.ReadFile(manifest)
	if err != nil || !bytes.Equal(original, unchanged) {
		t.Fatal("inspection changed REF", err)
	}
	if _, err := os.Stat(filepath.Join(dir, ".sealgraph", "index", "root", ".candidate")); !os.IsNotExist(err) {
		t.Fatalf("inspection created Candidate: %v", err)
	}
	normal := decodeCLIJSON(t, mustRunCLI(t, dir, "compare", "@"+old, "root", "--format", "json"))
	if normal["schema"] != "sealgraph/compare/v2" {
		t.Fatalf("schema changed: %v", normal["schema"])
	}
}

func TestCompareTextUsageAndRedirect(t *testing.T) {
	fixture := newCLIRevisionFixture(t)
	for _, suffix := range [][]string{{"--text", "--format", "json"}, {"--text", "--text"}, {"--text=true"}} {
		args := append([]string{"compare", "@" + fixture.root1, "root"}, suffix...)
		code, out, _ := runCLI(t, fixture.dir, nil, args...)
		if code != 2 || out != "" {
			t.Fatalf("usage accepted %v: %d %q", args, code, out)
		}
	}
	path := filepath.Join(t.TempDir(), "comparison.txt")
	code, data, stderr := runCLIToRegularFile(t, fixture.dir, path, "compare", "@"+fixture.root1, "root", "--text")
	if code != 0 || stderr != "" || !bytes.Contains(data, []byte("TEXT COMPARISON")) {
		t.Fatalf("redirect %d %q %q", code, data, stderr)
	}
}

func TestTextChangesBoundsAndBinary(t *testing.T) {
	tests := []struct {
		name        string
		left, right []byte
		want        string
	}{
		{"large", bytes.Repeat([]byte("a"), 2*1024*1024), bytes.Repeat([]byte("b"), 2*1024*1024), "OMITTED"},
		{"binary", []byte("before\x00\x1b"), []byte("after\x00\x1b"), "Binary or non-text"},
		{"invalid utf8", []byte{0xff}, []byte("after"), "Binary or non-text"},
		{"line budget", bytes.Repeat([]byte("a\n"), 4096), bytes.Repeat([]byte("b\n"), 4096), "LCS cell budget exceeded"},
		{"trailing newline", []byte("before"), []byte("before\n"), `+ "before\n"`},
		{"equal", []byte("same"), []byte("same"), "Content bytes unchanged"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var out bytes.Buffer
			printTextChanges(&out, tt.left, tt.right)
			if !strings.Contains(out.String(), tt.want) {
				t.Fatalf("missing %q", tt.want)
			}
			if out.Len() > compareTextLimit+4096 {
				t.Fatalf("unbounded output: %d", out.Len())
			}
			if bytes.Contains(out.Bytes(), []byte{0x1b}) {
				t.Fatal("unescaped control byte")
			}
		})
	}
}

func TestTextChangesUTF8Clipping(t *testing.T) {
	before := []byte(strings.Repeat("界", compareTextLimit))
	after := append([]byte("changed\n"), before...)
	var out bytes.Buffer
	printTextChanges(&out, before, after)
	if !strings.Contains(out.String(), "OMITTED") {
		t.Fatal("clipping not disclosed")
	}
}
