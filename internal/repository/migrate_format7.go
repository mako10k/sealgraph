package repository

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
)

// Format7MigrationResult contains only inventory counted after publication.
type Format7MigrationResult struct {
	RetainedSealsV5      int
	RetainedSealsV6      int
	RetainedCandidatesV5 int
	RetainedCandidatesV6 int
}

// MigrateRepositoryTo7 validates one exact historical repository and changes
// only its config. A committed error must be reconciled by readback, not retry.
func MigrateRepositoryTo7(ctx context.Context, workDir string, fromFormat int) (Format7MigrationResult, error) {
	if fromFormat != 5 && fromFormat != 6 {
		return Format7MigrationResult{}, fmt.Errorf("format-7 migration requires explicit source format 5 or 6")
	}
	dir := filepath.Join(workDir, ".sealgraph")
	actual, err := repositoryFormat(dir)
	if err != nil {
		return Format7MigrationResult{}, fmt.Errorf("inspect migration source: %w", err)
	}
	if actual != fromFormat {
		return Format7MigrationResult{}, fmt.Errorf("repository migration requires exact format %d; found format %d", fromFormat, actual)
	}
	repo := newRepositoryFormat(dir, fromFormat)
	return withMutation(ctx, repo.writer, "migrate repository to format 7", func() (Format7MigrationResult, error) {
		return repo.migrateFormat7Locked(ctx, workDir, fromFormat)
	})
}

func (r *Repository) migrateFormat7Locked(ctx context.Context, workDir string, fromFormat int) (Format7MigrationResult, error) {
	if current, err := repositoryFormat(r.dir); err != nil || current != fromFormat {
		return Format7MigrationResult{}, fmt.Errorf("migration source config changed before validation: format=%d err=%v", current, err)
	}
	beforeReport, err := r.Fsck(ctx)
	if err != nil {
		return Format7MigrationResult{}, fmt.Errorf("format-%d fsck before migration: %w", fromFormat, err)
	}
	beforePhysical, err := capturePhysicalRepositoryObservation(r.dir)
	if err != nil {
		return Format7MigrationResult{}, fmt.Errorf("capture migration source: %w", err)
	}
	if err := validateFsckPhysicalRepository(beforePhysical); err != nil {
		return Format7MigrationResult{}, fmt.Errorf("validate captured migration source: %w", err)
	}
	beforeInventory, err := r.captureFormat6MigrationObservation(ctx)
	if err != nil {
		return Format7MigrationResult{}, err
	}
	if err := r.validateMigrationCandidates(ctx, beforeInventory.candidates); err != nil {
		return Format7MigrationResult{}, err
	}
	staging, err := os.MkdirTemp(workDir, ".sealgraph-migrate7-")
	if err != nil {
		return Format7MigrationResult{}, fmt.Errorf("prepare format-7 staging: %w", err)
	}
	defer os.RemoveAll(staging)
	if err := stageFormat7Migration(staging, beforePhysical, beforeInventory.candidates); err != nil {
		return Format7MigrationResult{}, fmt.Errorf("prepare format-7 staging: %w", err)
	}
	if err := validateFormat7MigrationStaging(ctx, staging, beforePhysical, beforeInventory, beforeReport); err != nil {
		return Format7MigrationResult{}, fmt.Errorf("validate format-7 staging: %w", err)
	}
	temp, err := writeFormat7ConfigTemp(r.dir)
	if err != nil {
		return Format7MigrationResult{}, fmt.Errorf("prepare exact format-7 config: %w", err)
	}
	defer os.Remove(temp)
	if err := r.revalidateFormat7MigrationSource(ctx, beforePhysical, beforeInventory); err != nil {
		return Format7MigrationResult{}, err
	}
	if err := commitFormat7Config(r.dir, temp, fromFormat); err != nil {
		return Format7MigrationResult{}, err
	}
	return readBackFormat7Migration(ctx, workDir, beforePhysical, beforeInventory, beforeReport)
}

func stageFormat7Migration(staging string, physical physicalRepositoryObservation, candidates map[string][]byte) error {
	for _, relative := range []string{"objects", filepath.Join("refs", "seals"), "index", "locks"} {
		if err := os.MkdirAll(filepath.Join(staging, relative), 0o755); err != nil {
			return err
		}
	}
	if err := writeSyncedFile(filepath.Join(staging, "config"), []byte(format7ConfigBytes), 0o644); err != nil {
		return err
	}
	for _, entry := range physical.Entries {
		if entry.Path == "ROOT" || entry.Path == "CONFIG" {
			continue
		}
		path := filepath.Join(staging, filepath.FromSlash(entry.Path))
		if entry.Mode.IsDir() {
			if err := os.MkdirAll(path, 0o755); err != nil {
				return err
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return err
		}
		if err := writeSyncedFile(path, entry.Data, entry.Mode.Perm()); err != nil {
			return fmt.Errorf("stage %s: %w", entry.Path, err)
		}
	}
	staged := newRepositoryFormat(staging, 7)
	for name, data := range candidates {
		path := staged.candidates.path(name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return err
		}
		if err := writeSyncedFile(path, data, 0o600); err != nil {
			return fmt.Errorf("stage Candidate %s: %w", name, err)
		}
	}
	return syncStagingTree(staging)
}

func validateFormat7MigrationStaging(ctx context.Context, staging string, physical physicalRepositoryObservation, inventory format6MigrationObservation, report FsckReport) error {
	staged := newRepositoryFormat(staging, 7)
	stagedReport, err := staged.Fsck(ctx)
	if err != nil {
		return fmt.Errorf("staged fsck: %w", err)
	}
	if err := validateFormat7MigrationCounts(report, stagedReport); err != nil {
		return err
	}
	stagedPhysical, err := capturePhysicalRepositoryObservation(staging)
	if err != nil || !equalMigrationRetainedPhysical(physical, stagedPhysical) {
		return fmt.Errorf("staged canonical bytes differ from source: %v", err)
	}
	stagedInventory, err := staged.captureFormat6MigrationObservation(ctx)
	if err != nil || !equalFormat6MigrationObservations(inventory, stagedInventory) {
		return fmt.Errorf("staged REF, Candidate, or object inventory differs from source: %v", err)
	}
	return staged.validateMigrationCandidates(ctx, inventory.candidates)
}

func validateFormat7MigrationCounts(before, after FsckReport) error {
	if before.SealsV5 != after.SealsV5 || before.SealsV6 != after.SealsV6 ||
		before.CandidatesV5 != after.CandidatesV5 || before.CandidatesV6 != after.CandidatesV6 ||
		before.Seals != after.Seals || before.REFs != after.REFs || before.Tags != after.Tags ||
		before.Blobs != after.Blobs || after.SealsV7 != 0 || after.CandidatesV7 != 0 || after.ProvenancesV3 != 0 {
		return fmt.Errorf("config-only migration changed retained typed inventory")
	}
	return nil
}

func equalMigrationRetainedPhysical(source, target physicalRepositoryObservation) bool {
	if len(source.Entries) != len(target.Entries) {
		return false
	}
	for i, left := range source.Entries {
		right := target.Entries[i]
		if left.Path != right.Path || left.Present != right.Present {
			return false
		}
		if left.Path == "ROOT" || left.Path == "CONFIG" {
			continue
		}
		if left.Mode.IsDir() != right.Mode.IsDir() || left.Mode.IsRegular() != right.Mode.IsRegular() ||
			left.Size != right.Size || left.SHA256 != right.SHA256 {
			return false
		}
	}
	return true
}

func (r *Repository) revalidateFormat7MigrationSource(ctx context.Context, physical physicalRepositoryObservation, inventory format6MigrationObservation) error {
	currentPhysical, err := capturePhysicalRepositoryObservation(r.dir)
	if err != nil || !equalPhysicalRepositoryObservations(physical, currentPhysical) {
		return fmt.Errorf("migration source changed or became unreadable before config commit; no config was changed: %v", err)
	}
	currentInventory, err := r.captureFormat6MigrationObservation(ctx)
	if err != nil || !equalFormat6MigrationObservations(inventory, currentInventory) {
		return fmt.Errorf("migration source inventory changed before config commit; no config was changed: %v", err)
	}
	return nil
}

func writeFormat7ConfigTemp(dir string) (string, error) {
	file, err := os.CreateTemp(dir, ".tmp-config-")
	if err != nil {
		return "", err
	}
	path := file.Name()
	writeErr := file.Chmod(0o644)
	if writeErr == nil {
		_, writeErr = file.Write([]byte(format7ConfigBytes))
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

func commitFormat7Config(dir, temp string, fromFormat int) error {
	configPath := filepath.Join(dir, "config")
	current, err := observePhysicalPath(configPath, "CONFIG", false)
	if err != nil || !current.Mode.IsRegular() || !bytes.Equal(current.Data, []byte(migrationSourceConfig(fromFormat))) {
		return fmt.Errorf("migration source config changed before atomic replacement; no config was changed")
	}
	if err := os.Rename(temp, configPath); err != nil {
		return fmt.Errorf("commit format-7 config atomically: %w", err)
	}
	if err := syncCommittedFormat6Config(configPath); err != nil {
		return fmt.Errorf("MIGRATION_COMMITTED_DURABILITY_UNCERTAIN: config may already be format 7; do not retry automatically: %w", err)
	}
	if err := syncDirectoryForLoad(dir); err != nil {
		return fmt.Errorf("MIGRATION_COMMITTED_DURABILITY_UNCERTAIN: config may already be format 7; do not retry automatically: %w", err)
	}
	return nil
}

func migrationSourceConfig(format int) string {
	if format == 5 {
		return configBytes
	}
	return format6ConfigBytes
}

func readBackFormat7Migration(ctx context.Context, workDir string, physical physicalRepositoryObservation, inventory format6MigrationObservation, before FsckReport) (Format7MigrationResult, error) {
	migrated, err := OpenStandalone(workDir)
	if err != nil || migrated.Format() != 7 {
		return Format7MigrationResult{}, fmt.Errorf("MIGRATION_COMMITTED_READBACK_FAILED: reopen format-7 repository; do not retry automatically: %v", err)
	}
	after, err := migrated.Fsck(ctx)
	if err != nil {
		return Format7MigrationResult{}, fmt.Errorf("MIGRATION_COMMITTED_READBACK_FAILED: format-7 fsck; do not retry automatically: %w", err)
	}
	if err := validateFormat7MigrationCounts(before, after); err != nil {
		return Format7MigrationResult{}, fmt.Errorf("MIGRATION_COMMITTED_READBACK_FAILED: %w; do not retry automatically", err)
	}
	currentPhysical, err := capturePhysicalRepositoryObservation(migrated.dir)
	if err != nil || !equalMigrationRetainedPhysical(physical, currentPhysical) {
		return Format7MigrationResult{}, fmt.Errorf("MIGRATION_COMMITTED_READBACK_FAILED: retained canonical bytes differ; do not retry automatically: %v", err)
	}
	currentInventory, err := migrated.captureFormat6MigrationObservation(ctx)
	if err != nil || !equalFormat6MigrationObservations(inventory, currentInventory) {
		return Format7MigrationResult{}, fmt.Errorf("MIGRATION_COMMITTED_READBACK_FAILED: retained inventory differs; do not retry automatically: %v", err)
	}
	if err := migrated.validateMigrationCandidates(ctx, inventory.candidates); err != nil {
		return Format7MigrationResult{}, fmt.Errorf("MIGRATION_COMMITTED_READBACK_FAILED: %w; do not retry automatically", err)
	}
	return Format7MigrationResult{
		RetainedSealsV5: after.SealsV5, RetainedSealsV6: after.SealsV6,
		RetainedCandidatesV5: after.CandidatesV5, RetainedCandidatesV6: after.CandidatesV6,
	}, nil
}
