package repository

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mako10k/sealgraph/internal/migration"
)

type nativeLoadReceipt struct {
	Schema           string `json:"schema"`
	Result           string `json:"result"`
	SnapshotSHA256   string `json:"snapshot_sha256"`
	RepositoryFormat int    `json:"repository_format"`
	Blobs            int    `json:"blobs"`
	REFs             int    `json:"refs"`
	Candidates       int    `json:"candidates"`
}

// LoadNativeSnapshotV1 validates an exact native document and publishes it to
// an absent target once. Any post-publication error retains the target for
// explicit readback; callers must never retry blindly.
func LoadNativeSnapshotV1(ctx context.Context, workDir string, input []byte) ([]byte, error) {
	return loadNativeSnapshotV1(ctx, workDir, input, nil)
}

// LoadNativeSnapshotV1WithPrePublishCheck lets the caller revalidate a named
// input file's exact bytes immediately before the absent-target publication.
func LoadNativeSnapshotV1WithPrePublishCheck(ctx context.Context, workDir string, input []byte, check func() error) ([]byte, error) {
	if check == nil {
		return nil, fmt.Errorf("PRE_PUBLICATION_FAILURE: native input revalidation is required")
	}
	return loadNativeSnapshotV1(ctx, workDir, input, check)
}

func loadNativeSnapshotV1(ctx context.Context, workDir string, input []byte, check func() error) ([]byte, error) {
	target := filepath.Join(workDir, ".sealgraph")
	if err := preflightUniversalLoad(workDir, target); err != nil {
		return nil, err
	}
	snapshot, err := migration.DecodeNativeSnapshotV1(input)
	if err != nil {
		return nil, fmt.Errorf("PRE_PUBLICATION_FAILURE: %w", err)
	}
	staging, err := os.MkdirTemp(workDir, ".sealgraph-load-native-")
	if err != nil {
		return nil, fmt.Errorf("PRE_PUBLICATION_FAILURE: create native load staging: %w", err)
	}
	if err := constructNativeStaging(ctx, staging, snapshot); err != nil {
		return nil, fmt.Errorf("PRE_PUBLICATION_FAILURE: native staging retained at %s: %w", staging, err)
	}
	staged := newRepositoryFormat(staging, 7)
	if _, err := staged.Fsck(ctx); err != nil {
		return nil, fmt.Errorf("PRE_PUBLICATION_FAILURE: staged native fsck failed; staging retained at %s: %w", staging, err)
	}
	if _, err := verifyNativeSnapshotEquals(ctx, staged, input); err != nil {
		return nil, fmt.Errorf("PRE_PUBLICATION_FAILURE: staged native inventory mismatch; staging retained at %s: %w", staging, err)
	}
	if err := syncStagingTree(staging); err != nil {
		return nil, fmt.Errorf("PRE_PUBLICATION_FAILURE: synchronize native staging retained at %s: %w", staging, err)
	}
	if err := verifyNativeLoadModes(staging); err != nil {
		return nil, fmt.Errorf("PRE_PUBLICATION_FAILURE: verify native staging retained at %s: %w", staging, err)
	}
	if check != nil {
		if err := check(); err != nil {
			return nil, fmt.Errorf("PRE_PUBLICATION_FAILURE: native input changed before publication; staging retained at %s: %w", staging, err)
		}
	}
	if err := renameNoReplace(staging, target); err != nil {
		return nil, fmt.Errorf("PRE_PUBLICATION_FAILURE: atomic no-replace native publication failed; staging retained at %s: %w", staging, err)
	}
	if err := syncDirectoryForLoad(workDir); err != nil {
		return nil, fmt.Errorf("LOAD_PUBLISHED_DURABILITY_UNCERTAIN: target may be visible at %s; do not retry or delete automatically: %w", target, err)
	}
	loaded, err := OpenStandalone(workDir)
	if err != nil {
		return nil, fmt.Errorf("LOAD_PUBLISHED_READBACK_FAILED: open durable target; do not retry: %w", err)
	}
	if _, err := loaded.Fsck(ctx); err != nil {
		return nil, fmt.Errorf("LOAD_PUBLISHED_READBACK_FAILED: native fsck; do not retry: %w", err)
	}
	readback, err := verifyNativeSnapshotEquals(ctx, loaded, input)
	if err != nil {
		return nil, fmt.Errorf("LOAD_PUBLISHED_READBACK_FAILED: exact native inventory; do not retry: %w", err)
	}
	return encodeNativeLoadReceipt(input, readback)
}

func constructNativeStaging(ctx context.Context, staging string, snapshot migration.NativeSnapshotV1) error {
	for _, relative := range []string{"objects", filepath.Join("refs", "seals"), "index", "locks"} {
		if err := os.MkdirAll(filepath.Join(staging, relative), 0o755); err != nil {
			return err
		}
	}
	if err := writeSyncedFile(filepath.Join(staging, "config"), []byte(format7ConfigBytes), 0o644); err != nil {
		return err
	}
	repo := newRepositoryFormat(staging, 7)
	for _, blob := range snapshot.Blobs {
		id, err := repo.objects.WriteBlob(ctx, blob.Data)
		if err != nil || !id.Equal(blob.ID) {
			return fmt.Errorf("stage Blob %s: id=%s err=%w", blob.ID, id, err)
		}
	}
	for _, ref := range snapshot.REFs {
		path := filepath.Join(staging, "refs", "seals", filepath.FromSlash(ref.REF), ".ref")
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return err
		}
		if err := writeSyncedFile(path, ref.Manifest, 0o600); err != nil {
			return fmt.Errorf("stage REF %s: %w", ref.REF, err)
		}
	}
	for _, candidate := range snapshot.Candidates {
		path := repo.candidates.path(candidate.REF)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return err
		}
		if err := writeSyncedFile(path, candidate.Candidate, 0o600); err != nil {
			return fmt.Errorf("stage Candidate %s: %w", candidate.REF, err)
		}
	}
	return nil
}

func verifyNativeSnapshotEquals(ctx context.Context, repo *Repository, input []byte) (migration.NativeSnapshotV1, error) {
	actual, err := repo.DumpNativeSnapshotV1(ctx)
	if err != nil {
		return migration.NativeSnapshotV1{}, err
	}
	if !bytes.Equal(actual, input) {
		return migration.NativeSnapshotV1{}, fmt.Errorf("native snapshot exact bytes differ from source document")
	}
	return migration.DecodeNativeSnapshotV1(actual)
}

func verifyNativeLoadModes(root string) error {
	return filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("native loader path %s is a symbolic link", relative)
		}
		if entry.IsDir() {
			return requireLoadMode(relative, info.Mode(), 0o755)
		}
		name := filepath.ToSlash(relative)
		expected := os.FileMode(0)
		switch {
		case name == "config":
			expected = 0o644
		case strings.HasPrefix(name, "objects/"):
			expected = 0o444
		case strings.HasPrefix(name, "refs/seals/") && filepath.Base(name) == ".ref":
			expected = 0o600
		case strings.HasPrefix(name, "index/") && filepath.Base(name) == candidateFile:
			expected = 0o600
		default:
			return fmt.Errorf("native loader path %s is unexpected", relative)
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("native loader path %s is not a regular file", relative)
		}
		return requireLoadMode(relative, info.Mode(), expected)
	})
}

func encodeNativeLoadReceipt(input []byte, snapshot migration.NativeSnapshotV1) ([]byte, error) {
	digest := sha256.Sum256(input)
	value := nativeLoadReceipt{
		Schema: "sealgraph/native-load/v1", Result: "LOADED", SnapshotSHA256: fmt.Sprintf("%x", digest),
		RepositoryFormat: 7, Blobs: len(snapshot.Blobs), REFs: len(snapshot.REFs), Candidates: len(snapshot.Candidates),
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	return append(encoded, '\n'), nil
}
