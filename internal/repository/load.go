package repository

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/mako10k/sealgraph/internal/domain"
	"github.com/mako10k/sealgraph/internal/migration"
)

type UniversalLoadResult struct {
	Receipt  []byte
	Warnings []string
}

func LoadUniversalBlobV1(ctx context.Context, workDir string, input []byte) (UniversalLoadResult, error) {
	target := filepath.Join(workDir, ".sealgraph")
	if err := preflightUniversalLoad(workDir, target); err != nil {
		return UniversalLoadResult{}, err
	}
	dump, err := migration.DecodeUniversalBlobV1(input)
	if err != nil {
		return UniversalLoadResult{}, fmt.Errorf("PRE_PUBLICATION_FAILURE: %w", err)
	}
	projection, err := projectUniversalDump(dump)
	if err != nil {
		return UniversalLoadResult{}, fmt.Errorf("PRE_PUBLICATION_FAILURE: %w", err)
	}
	staging, err := os.MkdirTemp(workDir, ".sealgraph-load-")
	if err != nil {
		return UniversalLoadResult{}, fmt.Errorf("PRE_PUBLICATION_FAILURE: create load staging directory: %w", err)
	}
	if err := constructUniversalStaging(ctx, staging, dump, projection); err != nil {
		return UniversalLoadResult{}, fmt.Errorf("PRE_PUBLICATION_FAILURE: staging retained at %s: %w", staging, err)
	}
	repo := newRepository(staging)
	if _, err := repo.Fsck(ctx); err != nil {
		return UniversalLoadResult{}, fmt.Errorf("PRE_PUBLICATION_FAILURE: staged format-5 fsck failed; staging retained at %s: %w", staging, err)
	}
	digest, err := repositoryDigest(ctx, repo)
	if err != nil {
		return UniversalLoadResult{}, fmt.Errorf("PRE_PUBLICATION_FAILURE: staged repository digest failed; staging retained at %s: %w", staging, err)
	}
	receipt, err := encodeUniversalLoadReceipt(input, dump, projection, digest)
	if err != nil {
		return UniversalLoadResult{}, fmt.Errorf("PRE_PUBLICATION_FAILURE: encode receipt; staging retained at %s: %w", staging, err)
	}
	receiptValue, err := decodeCanonicalUniversalLoadReceipt(receipt)
	if err != nil {
		return UniversalLoadResult{}, fmt.Errorf("PRE_PUBLICATION_FAILURE: canonical receipt validation failed; staging retained at %s: %w", staging, err)
	}
	if err := validateReceiptAgainstLoadInputs(input, dump, projection, digest, receiptValue); err != nil {
		return UniversalLoadResult{}, fmt.Errorf("PRE_PUBLICATION_FAILURE: receipt/source projection validation failed; staging retained at %s: %w", staging, err)
	}
	if err := validateReceiptAgainstRepository(ctx, repo, receiptValue); err != nil {
		return UniversalLoadResult{}, fmt.Errorf("PRE_PUBLICATION_FAILURE: receipt/repository validation failed; staging retained at %s: %w", staging, err)
	}
	receiptPath := migrationReceiptPath(staging, input)
	if err := writeSyncedFile(receiptPath, receipt, 0o600); err != nil {
		return UniversalLoadResult{}, fmt.Errorf("PRE_PUBLICATION_FAILURE: write durable receipt; staging retained at %s: %w", staging, err)
	}
	if err := syncStagingTree(staging); err != nil {
		return UniversalLoadResult{}, fmt.Errorf("PRE_PUBLICATION_FAILURE: synchronize staged tree; staging retained at %s: %w", staging, err)
	}
	if err := verifyUniversalLoadModes(staging); err != nil {
		return UniversalLoadResult{}, fmt.Errorf("PRE_PUBLICATION_FAILURE: verify staged creation modes; staging retained at %s: %w", staging, err)
	}
	if err := renameNoReplace(staging, target); err != nil {
		return UniversalLoadResult{}, fmt.Errorf("PRE_PUBLICATION_FAILURE: atomic no-replace publication failed; staging retained at %s: %w", staging, err)
	}
	if err := syncDirectoryForLoad(workDir); err != nil {
		return UniversalLoadResult{}, fmt.Errorf("LOAD_PUBLISHED_DURABILITY_UNCERTAIN: target may be visible at %s; do not retry or delete automatically: %w", target, err)
	}
	if err := readBackUniversalLoad(ctx, workDir, receiptPathForTarget(target, input), input, dump, projection, receipt, digest); err != nil {
		return UniversalLoadResult{}, fmt.Errorf("LOAD_PUBLISHED_READBACK_FAILED: durable target %s did not pass complete readback; do not retry or repair automatically: %w", target, err)
	}
	return UniversalLoadResult{Receipt: receipt, Warnings: universalMigrationWarnings(projection.Semantic)}, nil
}

func universalMigrationWarnings(value migration.UniversalSemanticProjection) []string {
	return migration.SemanticWarnings(value)
}

func preflightUniversalLoad(workDir, target string) error {
	if info, err := os.Lstat(target); err == nil {
		return fmt.Errorf("PRE_PUBLICATION_FAILURE: %s already exists as %s; load requires an absent target and never merges or replaces", target, info.Mode().Type())
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("PRE_PUBLICATION_FAILURE: inspect load target %s: %w", target, err)
	}
	entries, err := os.ReadDir(workDir)
	if err != nil {
		return fmt.Errorf("PRE_PUBLICATION_FAILURE: inspect load parent: %w", err)
	}
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".sealgraph-load-") {
			return fmt.Errorf("PRE_PUBLICATION_FAILURE: prior staging path %s requires explicit inspection; it was not adopted or deleted", filepath.Join(workDir, entry.Name()))
		}
	}
	return nil
}

func constructUniversalStaging(ctx context.Context, staging string, dump migration.UniversalBlobV1, projection migrationProjection) error {
	for _, relative := range []string{"objects", filepath.Join("refs", "seals"), "index", "locks", filepath.Join("logs", "migration")} {
		if err := os.MkdirAll(filepath.Join(staging, relative), 0o755); err != nil {
			return err
		}
	}
	if err := writeSyncedFile(filepath.Join(staging, "config"), []byte(configBytes), 0o644); err != nil {
		return err
	}
	repo := newRepository(staging)
	for _, object := range dump.Objects {
		id, err := repo.objects.WriteBlob(ctx, object.Data)
		if err != nil || !id.Equal(object.ID) {
			return fmt.Errorf("write unchanged Blob %s: id=%s err=%w", object.ID, id, err)
		}
	}
	for _, seal := range projection.Seals {
		for _, object := range []struct {
			label string
			id    domain.ObjectID
			data  []byte
		}{{"Material", seal.MaterialID, seal.MaterialBytes}, {"Provenance", seal.ProvenanceID, seal.ProvenanceBytes}, {"Seal", seal.NewID, seal.SealBytes}} {
			id, err := repo.objects.WriteBlob(ctx, object.data)
			if err != nil || !id.Equal(object.id) {
				return fmt.Errorf("write projected %s %s: id=%s err=%w", object.label, object.id, id, err)
			}
		}
	}
	for _, ref := range dump.REFs {
		head := projection.OldToNew[ref.Head.String()]
		if err := repo.refs.Update(ctx, ref.Name, nil, &head); err != nil {
			return fmt.Errorf("write migrated REF %s: %w", ref.Name, err)
		}
	}
	for _, tag := range dump.Tags {
		target := projection.OldToNew[tag.Target.String()]
		head, err := repo.refs.Resolve(ctx, tag.REF)
		if err != nil {
			return err
		}
		if err := repo.tags.Create(ctx, tag.REF, tag.Name, target, head); err != nil {
			return fmt.Errorf("write migrated tag %s@%s: %w", tag.REF, tag.Name, err)
		}
	}
	return resetLoadRuntimeDirectories(staging)
}

func resetLoadRuntimeDirectories(staging string) error {
	for _, relative := range []string{"index", "locks"} {
		path := filepath.Join(staging, relative)
		if err := os.RemoveAll(path); err != nil {
			return err
		}
		if err := os.Mkdir(path, 0o755); err != nil {
			return err
		}
	}
	return nil
}

func migrationReceiptPath(repositoryDir string, input []byte) string {
	digest := sha256.Sum256(input)
	return filepath.Join(repositoryDir, "logs", "migration", fmt.Sprintf("%x.receipt", digest))
}
func receiptPathForTarget(target string, input []byte) string {
	return migrationReceiptPath(target, input)
}

func writeSyncedFile(path string, data []byte, mode os.FileMode) error {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, mode)
	if err != nil {
		return err
	}
	if err = file.Chmod(mode); err == nil {
		_, err = file.Write(data)
	}
	if err == nil {
		err = file.Sync()
	}
	if err == nil {
		var info os.FileInfo
		info, err = file.Stat()
		if err == nil && (!info.Mode().IsRegular() || info.Mode().Perm() != mode.Perm()) {
			err = fmt.Errorf("created file kind/mode verification failed: mode=%v", info.Mode())
		}
	}
	if closeErr := file.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return err
	}
	return nil
}

func syncStagingTree(root string) error {
	directories := []string{}
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("staging contains symbolic link %s", path)
		}
		if entry.IsDir() {
			directories = append(directories, path)
		}
		return nil
	})
	if err != nil {
		return err
	}
	sort.Slice(directories, func(i, j int) bool {
		left, right := strings.Count(directories[i], string(filepath.Separator)), strings.Count(directories[j], string(filepath.Separator))
		if left != right {
			return left > right
		}
		return directories[i] < directories[j]
	})
	for _, directory := range directories {
		if err := finalizeStagingDirectory(directory); err != nil {
			return err
		}
	}
	return nil
}

func finalizeStagingDirectory(path string) error {
	directory, err := openDirectoryNoFollow(path)
	if err != nil {
		return err
	}
	if err = directory.Chmod(0o755); err == nil {
		var info os.FileInfo
		info, err = directory.Stat()
		if err == nil && (!info.IsDir() || info.Mode().Perm() != 0o755) {
			err = fmt.Errorf("created directory kind/mode verification failed: mode=%v", info.Mode())
		}
	}
	if err == nil {
		err = directory.Sync()
	}
	if closeErr := directory.Close(); err == nil {
		err = closeErr
	}
	return err
}

func syncDirectoryForLoad(path string) error {
	directory, err := openDirectoryNoFollow(path)
	if err != nil {
		return err
	}
	info, err := directory.Stat()
	if err == nil && !info.IsDir() {
		err = fmt.Errorf("path is not a directory")
	}
	if err == nil {
		err = directory.Sync()
	}
	if closeErr := directory.Close(); err == nil {
		err = closeErr
	}
	return err
}

func verifyUniversalLoadModes(root string) error {
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
			return fmt.Errorf("loader path %s is a symbolic link", relative)
		}
		if entry.IsDir() {
			return requireLoadMode(relative, info.Mode(), 0o755)
		}
		expected, ok := expectedUniversalLoadFileMode(filepath.ToSlash(relative))
		if !ok || !info.Mode().IsRegular() {
			return fmt.Errorf("loader path %s is not an expected regular file", relative)
		}
		return requireLoadMode(relative, info.Mode(), expected)
	})
}

func expectedUniversalLoadFileMode(relative string) (os.FileMode, bool) {
	switch {
	case relative == "config":
		return 0o644, true
	case strings.HasPrefix(relative, "objects/"):
		return 0o444, true
	case strings.HasPrefix(relative, "refs/seals/") && filepath.Base(relative) == ".ref":
		return 0o600, true
	case strings.HasPrefix(relative, "logs/migration/") && strings.HasSuffix(relative, ".receipt"):
		return 0o600, true
	default:
		return 0, false
	}
}

func requireLoadMode(relative string, actual, expected os.FileMode) error {
	if actual.Perm() != expected.Perm() {
		return fmt.Errorf("loader path %s mode is %04o, expected %04o", relative, actual.Perm(), expected.Perm())
	}
	return nil
}

func readBackUniversalLoad(ctx context.Context, workDir, receiptPath string, input []byte, dump migration.UniversalBlobV1, projection migrationProjection, receipt []byte, expectedDigest string) error {
	repo, err := OpenStandalone(workDir)
	if err != nil {
		return err
	}
	if _, err := repo.Fsck(ctx); err != nil {
		return err
	}
	digest, err := repositoryDigest(ctx, repo)
	if err != nil {
		return err
	}
	if digest != expectedDigest {
		return fmt.Errorf("repository digest is %s, expected %s", digest, expectedDigest)
	}
	stored, err := os.ReadFile(receiptPath)
	if err != nil {
		return err
	}
	if !bytes.Equal(stored, receipt) {
		return fmt.Errorf("stored migration receipt differs from delivered bytes")
	}
	storedValue, err := decodeCanonicalUniversalLoadReceipt(stored)
	if err != nil {
		return fmt.Errorf("stored migration receipt is not canonical and valid: %w", err)
	}
	if err := validateReceiptAgainstLoadInputs(input, dump, projection, expectedDigest, storedValue); err != nil {
		return fmt.Errorf("stored migration receipt does not match load inputs: %w", err)
	}
	if err := validateReceiptAgainstRepository(ctx, repo, storedValue); err != nil {
		return fmt.Errorf("stored migration receipt does not match reopened repository: %w", err)
	}
	return verifyUniversalLoadModes(filepath.Join(workDir, ".sealgraph"))
}
