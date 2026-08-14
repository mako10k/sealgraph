package repository

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInitIsIndependentOfGitRepositoryPresence(t *testing.T) {
	plain := t.TempDir()
	insideGit := t.TempDir()
	if err := os.Mkdir(filepath.Join(insideGit, ".git"), 0o700); err != nil {
		t.Fatal(err)
	}
	if created, err := InitStandalone(plain); err != nil || !created {
		t.Fatalf("plain init created=%t err=%v", created, err)
	}
	if created, err := InitStandalone(insideGit); err != nil || !created {
		t.Fatalf("inside Git init created=%t err=%v", created, err)
	}
	for _, relative := range []string{"config", "objects", filepath.Join("refs", "seals"), filepath.Join("refs", "tags"), "index", "locks"} {
		plainInfo, plainErr := os.Stat(filepath.Join(plain, ".sealgraph", relative))
		gitInfo, gitErr := os.Stat(filepath.Join(insideGit, ".sealgraph", relative))
		if plainErr != nil || gitErr != nil || plainInfo.IsDir() != gitInfo.IsDir() {
			t.Fatalf("layout differs at %s: plain=%v/%v git=%v/%v", relative, plainInfo, plainErr, gitInfo, gitErr)
		}
	}
	if created, err := InitStandalone(insideGit); err != nil || created {
		t.Fatalf("idempotent init created=%t err=%v", created, err)
	}
}

func TestInitDoesNotReadDotGit(t *testing.T) {
	dir := t.TempDir()
	// A self-referential symlink makes any attempted traversal fail with ELOOP.
	if err := os.Symlink(".git", filepath.Join(dir, ".git")); err != nil {
		t.Fatal(err)
	}
	if created, err := InitStandalone(dir); err != nil || !created {
		t.Fatalf("init with unreadable .git created=%t err=%v", created, err)
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

func TestInitRejectsFormatOneWithoutMigration(t *testing.T) {
	dir := t.TempDir()
	if _, err := InitStandalone(dir); err != nil {
		t.Fatal(err)
	}
	config := filepath.Join(dir, ".sealgraph", "config")
	if err := os.WriteFile(config, []byte("repository_format = 1\nobject_format = sha256\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := InitStandalone(dir); err == nil || !strings.Contains(err.Error(), "unsupported or malformed config") {
		t.Fatalf("format-1 init error = %v", err)
	}
}

func TestExplicitInitBootstrapsOnlyMissingRuntimeDirectories(t *testing.T) {
	dir := t.TempDir()
	if created, err := InitStandalone(dir); err != nil || !created {
		t.Fatalf("initial init created=%t err=%v", created, err)
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
	if created, err := InitStandalone(dir); err != nil || created {
		t.Fatalf("bootstrap init created=%t err=%v", created, err)
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
