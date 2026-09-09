package repository

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestInitGitignorePolicy(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("Git is required to check ignore semantics")
	}
	dir := t.TempDir()
	if _, err := InitStandalone(dir); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("git", "init", "--quiet", dir)
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git init: %v: %s", err, output)
	}
	for path, ignored := range map[string]bool{
		"index/design/.candidate": true, "index/design/.track": true,
		"cache/graph": true, "locks/writer": true, "logs/recovery/record.json": true,
		"objects/ab/.tmp-object-123":         true,
		"refs/seals/design/.tmp-ref-123":     true,
		"refs/seals/design/api/.tmp-ref-123": true,
		".gitignore":                         false, "config": false,
		"objects/ab/" + strings.Repeat("c", 62): false,
		"refs/seals/design/.ref":                false,
		"refs/seals/index/.ref":                 false, "refs/seals/cache/.ref": false,
		"refs/seals/locks/.ref": false, "refs/seals/logs/.ref": false,
	} {
		t.Run(path, func(t *testing.T) {
			cmd := exec.Command("git", "-c", "core.excludesFile=/dev/null", "check-ignore", "--no-index", "--quiet", "--", ".sealgraph/"+path)
			cmd.Dir = dir
			output, err := cmd.CombinedOutput()
			var exitErr *exec.ExitError
			if err != nil && (!errors.As(err, &exitErr) || exitErr.ExitCode() != 1) {
				t.Fatalf("check-ignore: %v: %s", err, output)
			}
			if (err == nil) != ignored {
				t.Fatalf("ignored=%v, want %v", err == nil, ignored)
			}
		})
	}
}

func TestReinitPreservesGitignorePolicy(t *testing.T) {
	for _, absent := range []bool{false, true} {
		dir := t.TempDir()
		if _, err := InitStandalone(dir); err != nil {
			t.Fatal(err)
		}
		path := filepath.Join(dir, ".sealgraph", ".gitignore")
		custom := []byte("# custom policy\n/cache/\n")
		var err error
		if absent {
			err = os.Remove(path)
		} else {
			err = os.WriteFile(path, custom, 0o644)
		}
		if err != nil {
			t.Fatal(err)
		}
		if result, err := InitStandalone(dir); err != nil || result.Outcome != InitAlreadyComplete {
			t.Fatalf("reinit result=%+v err=%v", result, err)
		}
		got, err := os.ReadFile(path)
		if absent {
			if !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("missing gitignore was recreated: %v", err)
			}
		} else if err != nil || string(got) != string(custom) {
			t.Fatalf("custom gitignore changed: %q, %v", got, err)
		}
	}
}

func TestInitIsIndependentOfGitRepositoryPresence(t *testing.T) {
	plain := t.TempDir()
	insideGit := t.TempDir()
	if err := os.Mkdir(filepath.Join(insideGit, ".git"), 0o700); err != nil {
		t.Fatal(err)
	}
	if result, err := InitStandalone(plain); err != nil || result.Outcome != InitInitialized {
		t.Fatalf("plain init result=%+v err=%v", result, err)
	}
	if result, err := InitStandalone(insideGit); err != nil || result.Outcome != InitInitialized {
		t.Fatalf("inside Git init result=%+v err=%v", result, err)
	}
	for _, relative := range []string{"config", ".gitignore", "objects", filepath.Join("refs", "seals"), "index", "locks"} {
		plainInfo, plainErr := os.Stat(filepath.Join(plain, ".sealgraph", relative))
		gitInfo, gitErr := os.Stat(filepath.Join(insideGit, ".sealgraph", relative))
		if plainErr != nil || gitErr != nil || plainInfo.IsDir() != gitInfo.IsDir() {
			t.Fatalf("layout differs at %s: plain=%v/%v git=%v/%v", relative, plainInfo, plainErr, gitInfo, gitErr)
		}
	}
	if result, err := InitStandalone(insideGit); err != nil || result.Outcome != InitAlreadyComplete {
		t.Fatalf("idempotent init result=%+v err=%v", result, err)
	}
}

func TestInitCreatesExplicitFormat5Modes(t *testing.T) {
	dir := t.TempDir()
	if _, err := InitStandalone(dir); err != nil {
		t.Fatal(err)
	}
	configBytesOnDisk, err := os.ReadFile(filepath.Join(dir, ".sealgraph", "config"))
	if err != nil || string(configBytesOnDisk) != configBytes {
		t.Fatalf("config=%q err=%v", configBytesOnDisk, err)
	}
	for relative, expected := range map[string]os.FileMode{
		".": 0o755, "config": 0o644, ".gitignore": 0o644, "objects": 0o755,
		"refs": 0o755, filepath.Join("refs", "seals"): 0o755,
		"index": 0o755, "locks": 0o755,
	} {
		info, err := os.Lstat(filepath.Join(dir, ".sealgraph", relative))
		if err != nil || info.Mode().Perm() != expected {
			t.Fatalf("%s mode=%v expected=%04o err=%v", relative, initMode(info), expected, err)
		}
	}
}

func TestInitDoesNotReadDotGit(t *testing.T) {
	dir := t.TempDir()
	// A self-referential symlink makes any attempted traversal fail with ELOOP.
	if err := os.Symlink(".git", filepath.Join(dir, ".git")); err != nil {
		t.Fatal(err)
	}
	if result, err := InitStandalone(dir); err != nil || result.Outcome != InitInitialized {
		t.Fatalf("init with unreadable .git result=%+v err=%v", result, err)
	}
}

func TestInitRejectsUnsafeExistingRepositoryDirectory(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, ".sealgraph"), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := InitStandalone(dir); err == nil {
		t.Fatal("init accepted incomplete existing .sealgraph")
	}
}

func TestInitRejectsOlderFormatsWithoutMigration(t *testing.T) {
	dir := t.TempDir()
	if _, err := InitStandalone(dir); err != nil {
		t.Fatal(err)
	}
	config := filepath.Join(dir, ".sealgraph", "config")
	for _, format := range []string{"1", "2", "3"} {
		if err := os.WriteFile(config, []byte("repository_format = "+format+"\nobject_format = sha256\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := InitStandalone(dir); err == nil || !strings.Contains(err.Error(), "unsupported or malformed config") {
			t.Fatalf("format-%s init error = %v", format, err)
		}
	}
	if err := os.WriteFile(config, []byte("repository_format = 4\nobject_format = sha256\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := InitStandalone(dir); err == nil || !strings.Contains(err.Error(), "unsupported or malformed config") {
		t.Fatalf("interim format-4 config error = %v", err)
	}
}

func TestManifestFormatRejectsLegacyTagTree(t *testing.T) {
	dir := t.TempDir()
	if _, err := InitStandalone(dir); err != nil {
		t.Fatal(err)
	}
	legacy := filepath.Join(dir, ".sealgraph", "refs", "tags")
	if err := os.Mkdir(legacy, 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := OpenStandalone(dir); err == nil || !strings.Contains(err.Error(), "stores tags inside") {
		t.Fatalf("legacy tag tree open error = %v", err)
	}
}

func TestExplicitInitBootstrapsOnlyMissingRuntimeDirectories(t *testing.T) {
	dir := t.TempDir()
	if result, err := InitStandalone(dir); err != nil || result.Outcome != InitInitialized {
		t.Fatalf("initial init result=%+v err=%v", result, err)
	}
	repositoryDir := filepath.Join(dir, ".sealgraph")
	configBefore, err := os.ReadFile(filepath.Join(repositoryDir, "config"))
	if err != nil {
		t.Fatal(err)
	}
	for _, relative := range []string{"index", "locks"} {
		if err := os.Remove(filepath.Join(repositoryDir, relative)); err != nil {
			t.Fatal(err)
		}
	}
	if result, err := InitStandalone(dir); err != nil || result.Outcome != InitRuntimeBootstrapped || strings.Join(result.RuntimeDirectories, ",") != "index,locks" {
		t.Fatalf("bootstrap init result=%+v err=%v", result, err)
	}
	for _, relative := range []string{"index", "locks"} {
		info, err := os.Lstat(filepath.Join(repositoryDir, relative))
		if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			t.Fatalf("runtime directory %s info=%v err=%v", relative, info, err)
		}
	}
	configAfter, err := os.ReadFile(filepath.Join(repositoryDir, "config"))
	if err != nil {
		t.Fatal(err)
	}
	if string(configAfter) != string(configBefore) {
		t.Fatal("runtime bootstrap changed canonical config")
	}
}

func TestRuntimeBootstrapRejectsUnsafePathBeforeCreatingAnything(t *testing.T) {
	dir := t.TempDir()
	if _, err := InitStandalone(dir); err != nil {
		t.Fatal(err)
	}
	repositoryDir := filepath.Join(dir, ".sealgraph")
	if err := os.Remove(filepath.Join(repositoryDir, "index")); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(repositoryDir, "locks")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("objects", filepath.Join(repositoryDir, "locks")); err != nil {
		t.Fatal(err)
	}
	if _, err := InitStandalone(dir); err == nil {
		t.Fatal("init accepted a symbolic-link runtime path")
	}
	if _, err := os.Lstat(filepath.Join(repositoryDir, "index")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("failed bootstrap created index before rejecting locks: %v", err)
	}
}
