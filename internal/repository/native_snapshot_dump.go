package repository

import (
	"bytes"
	"context"
	"fmt"
	"sort"

	"github.com/mako10k/sealgraph/internal/migration"
)

// DumpNativeSnapshotV1 captures every retained canonical Blob, REF manifest,
// and Candidate in a format-7 repository. Local source state is excluded.
func (r *Repository) DumpNativeSnapshotV1(ctx context.Context) ([]byte, error) {
	if r.format != 7 {
		return nil, fmt.Errorf("native-blobs-v1 dump requires repository format 7; found %d", r.format)
	}
	return withMutation(ctx, r.writer, "dump native snapshot", func() ([]byte, error) {
		if _, err := r.Fsck(ctx); err != nil {
			return nil, fmt.Errorf("validate repository before native dump: %w", err)
		}
		first, err := r.captureNativeSnapshot(ctx)
		if err != nil {
			return nil, err
		}
		encoded, err := migration.EncodeNativeSnapshotV1(first)
		if err != nil {
			return nil, fmt.Errorf("encode native snapshot: %w", err)
		}
		second, err := r.captureNativeSnapshot(ctx)
		if err != nil {
			return nil, err
		}
		verified, err := migration.EncodeNativeSnapshotV1(second)
		if err != nil || !bytes.Equal(encoded, verified) {
			return nil, fmt.Errorf("canonical repository changed while capturing native snapshot; rerun dump")
		}
		return encoded, nil
	})
}

func (r *Repository) captureNativeSnapshot(ctx context.Context) (migration.NativeSnapshotV1, error) {
	physical, err := capturePhysicalRepositoryObservation(r.dir)
	if err != nil {
		return migration.NativeSnapshotV1{}, err
	}
	if err := validateFsckPhysicalRepository(physical); err != nil {
		return migration.NativeSnapshotV1{}, err
	}
	inventory, err := fsckInventoryFromPhysicalFormat(ctx, physical, 7)
	if err != nil {
		return migration.NativeSnapshotV1{}, err
	}
	if err := validateFsckObjectInventoryPhysical(inventory.objects, physical); err != nil {
		return migration.NativeSnapshotV1{}, err
	}
	result := migration.NativeSnapshotV1{
		Blobs:      make([]migration.NativeBlobRecord, 0, len(inventory.objects)),
		REFs:       make([]migration.NativeRefRecord, 0),
		Candidates: make([]migration.NativeCandidateRecord, 0),
	}
	for _, object := range inventory.objects {
		result.Blobs = append(result.Blobs, migration.NativeBlobRecord{ID: object.ID, Data: object.Data})
	}
	observation, err := r.observeHeads(ctx, "native snapshot")
	if err != nil {
		return migration.NativeSnapshotV1{}, err
	}
	if err := validateFsckREFObservationPhysical(observation, physical); err != nil {
		return migration.NativeSnapshotV1{}, err
	}
	for _, name := range observation.names {
		result.REFs = append(result.REFs, migration.NativeRefRecord{REF: name, Manifest: observation.manifests[name]})
	}
	names, err := r.candidates.List()
	if err != nil {
		return migration.NativeSnapshotV1{}, err
	}
	sort.Strings(names)
	for _, name := range names {
		snapshot, err := r.candidates.LoadSnapshot(name)
		if err != nil {
			return migration.NativeSnapshotV1{}, err
		}
		result.Candidates = append(result.Candidates, migration.NativeCandidateRecord{REF: name, Candidate: snapshot.Bytes})
	}
	if err := r.revalidateHeads(ctx, observation, "native snapshot"); err != nil {
		return migration.NativeSnapshotV1{}, err
	}
	finalPhysical, err := capturePhysicalRepositoryObservation(r.dir)
	if err != nil || !equalPhysicalRepositoryObservations(physical, finalPhysical) {
		return migration.NativeSnapshotV1{}, fmt.Errorf("canonical repository changed while capturing native snapshot; rerun dump")
	}
	return result, nil
}
