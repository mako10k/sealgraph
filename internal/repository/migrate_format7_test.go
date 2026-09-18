package repository

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestMigrateRepositoryTo7RetainsFormat5And6State(t *testing.T) {
	for _, sourceFormat := range []int{5, 6} {
		t.Run(migrationSourceConfig(sourceFormat), func(t *testing.T) {
			ctx, root, repo := prepareFormat7MigrationFixture(t, sourceFormat)
			beforePhysical, err := capturePhysicalRepositoryObservation(repo.dir)
			if err != nil {
				t.Fatal(err)
			}
			beforeInventory, err := repo.captureFormat6MigrationObservation(ctx)
			if err != nil {
				t.Fatal(err)
			}
			beforeReport, err := repo.Fsck(ctx)
			if err != nil {
				t.Fatal(err)
			}
			result, err := MigrateRepositoryTo7(ctx, root, sourceFormat)
			if err != nil {
				t.Fatal(err)
			}
			migrated, err := OpenStandalone(root)
			if err != nil || migrated.Format() != 7 {
				t.Fatalf("open format 7: format=%v err=%v", migrated, err)
			}
			afterPhysical, err := capturePhysicalRepositoryObservation(migrated.dir)
			if err != nil || !equalMigrationRetainedPhysical(beforePhysical, afterPhysical) {
				t.Fatalf("retained canonical bytes changed: %v", err)
			}
			afterInventory, err := migrated.captureFormat6MigrationObservation(ctx)
			if err != nil || !equalFormat6MigrationObservations(beforeInventory, afterInventory) {
				t.Fatalf("retained inventory changed: %v", err)
			}
			if result.RetainedSealsV5 != beforeReport.SealsV5 || result.RetainedSealsV6 != beforeReport.SealsV6 ||
				result.RetainedCandidatesV5 != beforeReport.CandidatesV5 || result.RetainedCandidatesV6 != beforeReport.CandidatesV6 {
				t.Fatalf("readback counts=%+v before=%+v", result, beforeReport)
			}
			if _, err := migrated.Fsck(ctx); err != nil {
				t.Fatalf("format 7 fsck: %v", err)
			}
			if _, err := MigrateRepositoryTo7(ctx, root, sourceFormat); err == nil {
				t.Fatal("repeated migration accepted format 7")
			}
		})
	}
}

func prepareFormat7MigrationFixture(t *testing.T, sourceFormat int) (context.Context, string, *Repository) {
	t.Helper()
	ctx := context.Background()
	root := t.TempDir()
	if _, err := InitStandalone(root); err != nil {
		t.Fatal(err)
	}
	repo, err := OpenStandalone(root)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := repo.Add(ctx, AddOptions{REF: "root", Content: []byte("old"), Root: true, RootSet: true, ClearCauseLinks: true}); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.Seal(ctx, "root"); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.Add(ctx, AddOptions{REF: "root", Content: []byte("candidate-v5")}); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.objects.WriteBlob(ctx, []byte("unreferenced historical blob")); err != nil {
		t.Fatal(err)
	}
	if sourceFormat == 6 {
		if _, err := MigrateRepository5To6(ctx, root); err != nil {
			t.Fatal(err)
		}
		repo, err = OpenStandalone(root)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := repo.Add(ctx, AddOptions{REF: "root", Content: []byte("successor-v6")}); err != nil {
			t.Fatal(err)
		}
		if _, err := repo.Seal(ctx, "root"); err != nil {
			t.Fatal(err)
		}
		if _, err := repo.Add(ctx, AddOptions{REF: "root", Content: []byte("candidate-v6")}); err != nil {
			t.Fatal(err)
		}
	}
	return ctx, root, repo
}

func TestMigrateRepositoryTo7RejectsWrongSourceAndCorruptionBeforeCommit(t *testing.T) {
	ctx, root, repo := prepareFormat7MigrationFixture(t, 5)
	configPath := filepath.Join(root, ".sealgraph", "config")
	if _, err := MigrateRepositoryTo7(ctx, root, 6); err == nil {
		t.Fatal("accepted a mismatched source format")
	}
	if _, err := MigrateRepositoryTo7(ctx, root, 7); err == nil {
		t.Fatal("accepted a non-historical source format")
	}
	path := repo.candidates.path("root")
	if err := os.WriteFile(path, []byte("invalid candidate"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := MigrateRepositoryTo7(ctx, root, 5); err == nil {
		t.Fatal("accepted a corrupt source candidate")
	}
	config, err := os.ReadFile(configPath)
	if err != nil || string(config) != configBytes {
		t.Fatalf("precommit failure changed config: %q err=%v", config, err)
	}
}

func TestMigrateRepositoryTo7RetainsNewerShapedOrphan(t *testing.T) {
	for _, sourceFormat := range []int{5, 6} {
		t.Run(migrationSourceConfig(sourceFormat), func(t *testing.T) {
			ctx, root, repo := prepareFormat7MigrationFixture(t, sourceFormat)
			content, err := repo.objects.WriteBlob(ctx, []byte("orphan-content"))
			if err != nil {
				t.Fatal(err)
			}
			material := putFormat7Material(t, repo, content)
			orphan := putFormat7Seal(t, repo, material, nil)
			before, err := repo.Fsck(ctx)
			if err != nil || before.SealsV7 != 0 {
				t.Fatalf("source fsck=%+v err=%v", before, err)
			}
			physical, err := capturePhysicalRepositoryObservation(repo.dir)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := MigrateRepositoryTo7(ctx, root, sourceFormat); err != nil {
				t.Fatal(err)
			}
			migrated, err := OpenStandalone(root)
			if err != nil {
				t.Fatal(err)
			}
			after, err := migrated.Fsck(ctx)
			if err != nil || after.SealsV7 != 1 {
				t.Fatalf("migrated fsck=%+v err=%v", after, err)
			}
			if _, err := migrated.LoadSeal(ctx, orphan); err != nil {
				t.Fatalf("retained orphan cannot be read: %v", err)
			}
			readback, err := capturePhysicalRepositoryObservation(migrated.dir)
			if err != nil || !equalMigrationRetainedPhysical(physical, readback) {
				t.Fatalf("orphan bytes changed: %v", err)
			}
		})
	}
}
