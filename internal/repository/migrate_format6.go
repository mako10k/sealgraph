package repository

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

// Format6MigrationResult is the read-back inventory of a committed config-only
// migration. Public CLI receipt encoding belongs to the following CLI slice.
type Format6MigrationResult struct {
	RetainedSealsV5       int
	RetainedProvenancesV1 int
	RetainedCandidatesV5  int
}

type format6MigrationObservation struct {
	refs       map[string][]byte
	candidates map[string][]byte
	objects    []string
}

// MigrateRepository5To6 performs ADR 0030's atomic config-only transaction.
// It never rewrites objects, REF manifests, or Candidate files.
func MigrateRepository5To6(ctx context.Context, workDir string) (Format6MigrationResult, error) {
	repositoryDir := filepath.Join(workDir, ".sealgraph")
	format, err := repositoryFormat(repositoryDir)
	if err != nil {
		return Format6MigrationResult{}, fmt.Errorf("inspect migration source: %w", err)
	}
	if format != 5 {
		return Format6MigrationResult{}, fmt.Errorf("repository migration requires exact format 5; found format %d", format)
	}
	repo := newRepositoryFormat(repositoryDir, 5)
	return withMutation(ctx, repo.writer, "migrate repository format 5 to 6", func() (Format6MigrationResult, error) {
		return repo.migrateFormat6Locked(ctx, workDir)
	})
}

func (r *Repository) migrateFormat6Locked(ctx context.Context, workDir string) (Format6MigrationResult, error) {
	if current, err := repositoryFormat(r.dir); err != nil || current != 5 {
		return Format6MigrationResult{}, fmt.Errorf("migration source config changed before validation: format=%d err=%v", current, err)
	}
	if _, err := r.Fsck(ctx); err != nil {
		return Format6MigrationResult{}, fmt.Errorf("format-5 fsck before migration: %w", err)
	}
	before, err := r.captureFormat6MigrationObservation(ctx)
	if err != nil {
		return Format6MigrationResult{}, err
	}
	if err := r.validateMigrationCandidates(ctx, before.candidates); err != nil {
		return Format6MigrationResult{}, err
	}
	temp, err := writeFormat6ConfigTemp(r.dir)
	if err != nil {
		return Format6MigrationResult{}, fmt.Errorf("prepare exact format-6 config: %w", err)
	}
	defer os.Remove(temp)
	if err := r.commitFormat6Config(ctx, temp, before); err != nil {
		return Format6MigrationResult{}, err
	}
	return readBackFormat6Migration(ctx, workDir, before)
}

func (r *Repository) commitFormat6Config(ctx context.Context, temp string, before format6MigrationObservation) error {
	after, err := r.captureFormat6MigrationObservation(ctx)
	if err != nil || !equalFormat6MigrationObservations(before, after) {
		return fmt.Errorf("migration source changed or became unreadable before config commit; no config was changed: %v", err)
	}
	configPath := filepath.Join(r.dir, "config")
	current, err := observePhysicalPath(configPath, "CONFIG", false)
	if err != nil || !current.Mode.IsRegular() || !bytes.Equal(current.Data, []byte(configBytes)) {
		return fmt.Errorf("migration source config changed before atomic replacement; no config was changed")
	}
	if err := os.Rename(temp, configPath); err != nil {
		return fmt.Errorf("commit format-6 config atomically: %w", err)
	}
	if err := syncCommittedFormat6Config(configPath); err != nil {
		return fmt.Errorf("MIGRATION_COMMITTED_DURABILITY_UNCERTAIN: config may already be format 6; do not retry automatically: %w", err)
	}
	if err := syncDirectoryForLoad(r.dir); err != nil {
		return fmt.Errorf("MIGRATION_COMMITTED_DURABILITY_UNCERTAIN: config may already be format 6; do not retry automatically: %w", err)
	}
	return nil
}

func syncCommittedFormat6Config(path string) error {
	file, err := openFileNoFollow(path)
	if err != nil {
		return err
	}
	info, statErr := file.Stat()
	if statErr == nil && !info.Mode().IsRegular() {
		statErr = fmt.Errorf("committed config is not a regular file")
	}
	if statErr == nil {
		statErr = file.Sync()
	}
	if closeErr := file.Close(); statErr == nil {
		statErr = closeErr
	}
	return statErr
}

func readBackFormat6Migration(ctx context.Context, workDir string, before format6MigrationObservation) (Format6MigrationResult, error) {
	migrated, err := OpenStandalone(workDir)
	if err != nil {
		return Format6MigrationResult{}, fmt.Errorf("MIGRATION_COMMITTED_READBACK_FAILED: reopen format-6 repository; do not retry automatically: %w", err)
	}
	report, err := migrated.Fsck(ctx)
	if err != nil {
		return Format6MigrationResult{}, fmt.Errorf("MIGRATION_COMMITTED_READBACK_FAILED: format-6 fsck; do not retry automatically: %w", err)
	}
	readback, err := migrated.captureFormat6MigrationObservation(ctx)
	if err != nil || !equalFormat6MigrationObservations(before, readback) {
		return Format6MigrationResult{}, fmt.Errorf("MIGRATION_COMMITTED_READBACK_FAILED: retained repository state differs after config transition; do not retry automatically: %v", err)
	}
	if err := migrated.validateMigrationCandidates(ctx, readback.candidates); err != nil {
		return Format6MigrationResult{}, fmt.Errorf("MIGRATION_COMMITTED_READBACK_FAILED: %w; do not retry automatically", err)
	}
	if report.SealsV6 != 0 || report.ProvenancesV2 != 0 || report.CandidatesV6 != 0 {
		return Format6MigrationResult{}, fmt.Errorf("MIGRATION_COMMITTED_READBACK_FAILED: config-only migration unexpectedly observed successor records; do not retry automatically")
	}
	return Format6MigrationResult{RetainedSealsV5: report.SealsV5, RetainedProvenancesV1: report.ProvenancesV1, RetainedCandidatesV5: report.CandidatesV5}, nil
}

func writeFormat6ConfigTemp(repositoryDir string) (string, error) {
	file, err := os.CreateTemp(repositoryDir, ".tmp-config-")
	if err != nil {
		return "", err
	}
	path := file.Name()
	writeErr := file.Chmod(0o644)
	if writeErr == nil {
		_, writeErr = file.Write([]byte(format6ConfigBytes))
	}
	if writeErr == nil {
		writeErr = file.Sync()
	}
	if closeErr := file.Close(); writeErr == nil {
		writeErr = closeErr
	}
	if writeErr != nil {
		os.Remove(path)
		return "", writeErr
	}
	return path, nil
}

func (r *Repository) captureFormat6MigrationObservation(ctx context.Context) (format6MigrationObservation, error) {
	heads, err := r.observeHeads(ctx, "format-6 migration")
	if err != nil {
		return format6MigrationObservation{}, err
	}
	refs := make(map[string][]byte, len(heads.manifests))
	for name, data := range heads.manifests {
		refs[name] = append([]byte(nil), data...)
	}
	names, err := r.candidates.List()
	if err != nil {
		return format6MigrationObservation{}, err
	}
	candidates := make(map[string][]byte, len(names))
	for _, name := range names {
		snapshot, err := r.candidates.LoadSnapshot(name)
		if err != nil {
			return format6MigrationObservation{}, err
		}
		candidates[name] = append([]byte(nil), snapshot.Bytes...)
	}
	objects, err := r.captureObjectInventory(ctx, "format-6 migration")
	if err != nil {
		return format6MigrationObservation{}, err
	}
	return format6MigrationObservation{refs: refs, candidates: candidates, objects: objects}, nil
}

func (r *Repository) validateMigrationCandidates(ctx context.Context, expected map[string][]byte) error {
	observation, _, err := r.buildObservation(ctx, "Candidate migration validation")
	if err != nil {
		return err
	}
	names := make([]string, 0, len(expected))
	for name := range expected {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		snapshot, err := r.candidates.LoadSnapshot(name)
		if err != nil || !bytes.Equal(snapshot.Bytes, expected[name]) {
			return fmt.Errorf("Candidate %s changed or became unreadable during migration validation: %v", name, err)
		}
		if r.format == 5 && snapshot.Candidate.Schema != "sealgraph/candidate/v5" {
			return fmt.Errorf("Candidate %s is not exact Candidate v5", name)
		}
		if _, err := r.InspectCandidate(ctx, name); err != nil {
			return fmt.Errorf("validate Candidate %s closure for migration: %w", name, err)
		}
		if err := r.validateCandidateMutation(ctx, snapshot.Candidate, observation); err != nil {
			return fmt.Errorf("validate Candidate %s graph for migration: %w", name, err)
		}
	}
	return nil
}

func equalFormat6MigrationObservations(left, right format6MigrationObservation) bool {
	if len(left.refs) != len(right.refs) || len(left.candidates) != len(right.candidates) || len(left.objects) != len(right.objects) {
		return false
	}
	for name, data := range left.refs {
		if !bytes.Equal(data, right.refs[name]) {
			return false
		}
	}
	for name, data := range left.candidates {
		if !bytes.Equal(data, right.candidates[name]) {
			return false
		}
	}
	for i := range left.objects {
		if left.objects[i] != right.objects[i] {
			return false
		}
	}
	return true
}
