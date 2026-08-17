package pathmanifest

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"unicode/utf8"
)

func TestBuildIsCanonicalAndInputOrderIndependent(t *testing.T) {
	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, "z.txt"), []byte("z\x00\r\n"))
	if err := os.Mkdir(filepath.Join(dir, "docs"), 0o755); err != nil {
		t.Fatal(err)
	}
	mustWrite(t, filepath.Join(dir, "docs", "a.txt"), []byte("alpha"))

	first, err := Build(dir, "git:012345", []string{"z.txt", "docs/a.txt"})
	if err != nil {
		t.Fatal(err)
	}
	second, err := Build(dir, "git:012345", []string{"docs/a.txt", "z.txt"})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(first, second) {
		t.Fatalf("input order changed manifest:\n%s\n%s", first, second)
	}
	if !utf8.Valid(first) || first[len(first)-1] != '\n' || bytes.Count(first, []byte{'\n'}) != 1 {
		t.Fatalf("manifest is not one canonical UTF-8 JSON line: %q", first)
	}
	for _, field := range []string{`"schema":"` + Schema + `"`, `"claim":"` + Claim + `"`, `"digest_algorithm":"sha256"`, `"aggregate_algorithm":"` + AggregateAlgorithm + `"`} {
		if !bytes.Contains(first, []byte(field)) {
			t.Fatalf("manifest missing %s: %s", field, first)
		}
	}
	if bytes.Index(first, []byte(`"path":"docs/a.txt"`)) > bytes.Index(first, []byte(`"path":"z.txt"`)) {
		t.Fatalf("entries are not bytewise path sorted: %s", first)
	}
	const fixtureSHA256 = "920797199f7bc62b012d56fbd31c96da4115fa3f3ac5ebb0181fdc66da9d0f14"
	if got := fmt.Sprintf("%x", sha256.Sum256(first)); got != fixtureSHA256 {
		t.Fatalf("canonical fixture sha256=%s, want %s", got, fixtureSHA256)
	}
}

func TestBuildIdentityChangesWithFilePathAndSource(t *testing.T) {
	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, "a"), []byte("same"))
	mustWrite(t, filepath.Join(dir, "b"), []byte("same"))
	base := mustBuild(t, dir, "source-1", "a")
	for name, changed := range map[string][]byte{
		"path":   mustBuild(t, dir, "source-1", "b"),
		"source": mustBuild(t, dir, "source-2", "a"),
	} {
		if bytes.Equal(base, changed) {
			t.Fatalf("%s did not change manifest identity", name)
		}
	}
	mustWrite(t, filepath.Join(dir, "a"), []byte("Same"))
	if changed := mustBuild(t, dir, "source-1", "a"); bytes.Equal(base, changed) {
		t.Fatal("one changed file byte did not change manifest identity")
	}
}

func TestBuildRejectsUnsafeAndAmbiguousPathsWithoutOutput(t *testing.T) {
	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, "regular"), []byte("ok"))
	if err := os.Mkdir(filepath.Join(dir, "directory"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("regular", filepath.Join(dir, "symlink")); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(dir, "real"), 0o755); err != nil {
		t.Fatal(err)
	}
	mustWrite(t, filepath.Join(dir, "real", "nested"), []byte("nested"))
	if err := os.Symlink("real", filepath.Join(dir, "linked-dir")); err != nil {
		t.Fatal(err)
	}
	if err := syscall.Mkfifo(filepath.Join(dir, "fifo"), 0o600); err != nil {
		t.Fatal(err)
	}
	listener, err := net.Listen("unix", filepath.Join(dir, "socket"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = listener.Close() })

	cases := map[string][]string{
		"empty":             {""},
		"absolute":          {filepath.Join(dir, "regular")},
		"dot":               {"./regular"},
		"dot-dot":           {"../regular"},
		"empty-component":   {"real//nested"},
		"backslash":         {`real\nested`},
		"missing":           {"missing"},
		"directory":         {"directory"},
		"symlink":           {"symlink"},
		"symlink-ancestor":  {"linked-dir/nested"},
		"fifo":              {"fifo"},
		"socket":            {"socket"},
		"duplicate":         {"regular", "regular"},
		"control-character": {"bad\npath"},
	}
	for name, paths := range cases {
		t.Run(name, func(t *testing.T) {
			if output, err := Build(dir, "source", paths); err == nil || output != nil {
				t.Fatalf("Build output=%q err=%v", output, err)
			}
		})
	}
}

func mustWrite(t *testing.T, path string, data []byte) {
	t.Helper()
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
}

func mustBuild(t *testing.T, dir, source string, paths ...string) []byte {
	t.Helper()
	output, err := Build(dir, source, paths)
	if err != nil {
		t.Fatal(err)
	}
	return output
}

func TestBuildDoesNotInspectUnselectedDotGit(t *testing.T) {
	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, "selected"), []byte("selected"))
	if err := os.Symlink(".git", filepath.Join(dir, ".git")); err != nil {
		t.Fatal(err)
	}
	output := mustBuild(t, dir, "explicit", "selected")
	if !strings.Contains(string(output), `"source":"explicit"`) {
		t.Fatalf("manifest source = %s", output)
	}
}
