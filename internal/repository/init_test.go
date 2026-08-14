package repository

import (
	"os"
	"path/filepath"
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
	for _, relative := range []string{"config", "objects", filepath.Join("refs", "seals"), "index", "locks"} {
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
