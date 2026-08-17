package repository

import (
	"bytes"
	"context"
	"fmt"
	"sort"

	"github.com/mako10k/sealgraph/internal/canonical"
	"github.com/mako10k/sealgraph/internal/domain"
	"github.com/mako10k/sealgraph/internal/graph"
	"github.com/mako10k/sealgraph/internal/revision"
	"github.com/mako10k/sealgraph/internal/store"
)

type FsckReport struct {
	Objects, Seals, MaterialObjects, REFs, Tags, ActiveSeals int
	HistoricalOrDetachedSeals                                []domain.ObjectID
	UnreferencedObjects                                      []domain.ObjectID
}

type fsckObjectInventory struct {
	objects            []store.Object
	sealIDs            []domain.ObjectID
	sealSet            map[string]bool
	referencedMaterial map[string]bool
}

type fsckTag struct {
	REF, Name string
	Seal      domain.ObjectID
}

// Fsck validates a coherent complete canonical inventory without reading
// candidates or writing cache, repair, mode, object, or REF state.
func (r *Repository) Fsck(ctx context.Context) (FsckReport, error) {
	observation, err := r.observeHeads(ctx, "fsck")
	if err != nil {
		return FsckReport{}, err
	}
	inventory, err := r.fsckInventory(ctx)
	if err != nil {
		return FsckReport{}, err
	}
	tags, err := r.fsckTags(ctx, observation.names)
	if err != nil {
		return FsckReport{}, err
	}
	if err := validateFsckTargets(observation, tags, inventory.sealSet); err != nil {
		return FsckReport{}, err
	}
	active, err := r.validateFsckGraphs(ctx, observation, inventory.sealIDs)
	if err != nil {
		return FsckReport{}, err
	}
	report := buildFsckReport(observation, tags, inventory, active)
	if err := r.validateFsckFinalObservation(ctx, observation, tags, inventory.objects); err != nil {
		return FsckReport{}, err
	}
	return report, nil
}

func (r *Repository) fsckInventory(ctx context.Context) (fsckObjectInventory, error) {
	objects, err := r.objects.List(ctx)
	if err != nil {
		return fsckObjectInventory{}, fmt.Errorf("inventory immutable objects: %w", err)
	}
	result := fsckObjectInventory{objects: objects, sealIDs: []domain.ObjectID{}, sealSet: make(map[string]bool), referencedMaterial: make(map[string]bool)}
	objectSet := make(map[string]bool, len(objects))
	for _, object := range objects {
		objectSet[object.ID.String()] = true
		payload, decodeErr := canonical.DecodeSeal(object.Data)
		if decodeErr != nil {
			continue
		}
		result.sealIDs = append(result.sealIDs, object.ID)
		result.sealSet[object.ID.String()] = true
		result.referencedMaterial[payload.Content.ID.String()] = true
		for _, attachment := range payload.Attachments {
			result.referencedMaterial[attachment.Blob.ID.String()] = true
		}
	}
	for id := range result.referencedMaterial {
		if !objectSet[id] {
			return fsckObjectInventory{}, fmt.Errorf("seal material object %s is missing", id)
		}
	}
	return result, nil
}

func validateFsckTargets(observation headObservation, tags []fsckTag, sealSet map[string]bool) error {
	for _, ref := range observation.names {
		if !sealSet[observation.heads[ref].String()] {
			return fmt.Errorf("REF %s head %s is not a canonical Seal", ref, observation.heads[ref])
		}
	}
	for _, tag := range tags {
		if !sealSet[tag.Seal.String()] {
			return fmt.Errorf("tag %s@%s target %s is not a canonical Seal", tag.REF, tag.Name, tag.Seal)
		}
	}
	return nil
}

func (r *Repository) validateFsckGraphs(ctx context.Context, observation headObservation, sealIDs []domain.ObjectID) (*revision.Index, error) {
	allHeads := make([]revision.Head, 0, len(sealIDs))
	for _, id := range sealIDs {
		allHeads = append(allHeads, revision.Head{REF: "fsck/" + id.String(), Seal: id})
	}
	allRevisions, err := revision.Build(ctx, allHeads, revision.LoadSealFunc(r.LoadSeal))
	if err != nil {
		return nil, fmt.Errorf("validate every parent_revision chain: %w", err)
	}
	if _, err := graph.Build(ctx, sealIDs, allRevisions, graph.LoadSealFunc(r.LoadSeal)); err != nil {
		return nil, fmt.Errorf("validate every Cause graph: %w", err)
	}
	active, err := revision.Build(ctx, observation.revisionHeads(), revision.LoadSealFunc(r.LoadSeal))
	if err != nil {
		return nil, fmt.Errorf("validate active revision graph: %w", err)
	}
	return active, nil
}

func buildFsckReport(observation headObservation, tags []fsckTag, inventory fsckObjectInventory, active *revision.Index) FsckReport {
	report := FsckReport{Objects: len(inventory.objects), Seals: len(inventory.sealIDs), MaterialObjects: len(inventory.objects) - len(inventory.sealIDs), REFs: len(observation.names), Tags: len(tags), ActiveSeals: len(active.Nodes()), HistoricalOrDetachedSeals: []domain.ObjectID{}, UnreferencedObjects: []domain.ObjectID{}}
	for _, id := range inventory.sealIDs {
		if _, found := active.Node(id); !found {
			report.HistoricalOrDetachedSeals = append(report.HistoricalOrDetachedSeals, id)
		}
	}
	for _, object := range inventory.objects {
		if !inventory.sealSet[object.ID.String()] && !inventory.referencedMaterial[object.ID.String()] {
			report.UnreferencedObjects = append(report.UnreferencedObjects, object.ID)
		}
	}
	sort.Slice(report.HistoricalOrDetachedSeals, func(i, j int) bool {
		return report.HistoricalOrDetachedSeals[i].String() < report.HistoricalOrDetachedSeals[j].String()
	})
	sort.Slice(report.UnreferencedObjects, func(i, j int) bool {
		return report.UnreferencedObjects[i].String() < report.UnreferencedObjects[j].String()
	})
	return report
}

func (r *Repository) validateFsckFinalObservation(ctx context.Context, observation headObservation, tags []fsckTag, objects []store.Object) error {
	if err := r.revalidateHeads(ctx, observation, "fsck"); err != nil {
		return err
	}
	finalTags, err := r.fsckTags(ctx, observation.names)
	if err != nil || !equalFsckTags(tags, finalTags) {
		return fmt.Errorf("REF tags changed or became unreadable while deriving fsck; rerun the command")
	}
	finalObjects, err := r.objects.List(ctx)
	if err != nil || !equalFsckObjects(objects, finalObjects) {
		return fmt.Errorf("object inventory changed or became unreadable while deriving fsck; rerun the command")
	}
	return nil
}

func (r *Repository) fsckTags(ctx context.Context, refs []string) ([]fsckTag, error) {
	result := make([]fsckTag, 0)
	for _, ref := range refs {
		tags, err := r.tags.List(ctx, ref)
		if err != nil {
			return nil, fmt.Errorf("validate tags for REF %s: %w", ref, err)
		}
		for _, tag := range tags {
			result = append(result, fsckTag{REF: ref, Name: tag.Name, Seal: tag.Seal})
		}
	}
	return result, nil
}

func equalFsckTags(left, right []fsckTag) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i].REF != right[i].REF || left[i].Name != right[i].Name || !left[i].Seal.Equal(right[i].Seal) {
			return false
		}
	}
	return true
}

func equalFsckObjects(left, right []store.Object) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if !left[i].ID.Equal(right[i].ID) || left[i].Type != right[i].Type || !bytes.Equal(left[i].Data, right[i].Data) {
			return false
		}
	}
	return true
}
