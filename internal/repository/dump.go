package repository

import (
	"bytes"
	"context"
	"fmt"
	"reflect"
	"sort"

	"github.com/mako10k/sealgraph/internal/canonical"
	"github.com/mako10k/sealgraph/internal/domain"
	"github.com/mako10k/sealgraph/internal/migration"
	"github.com/mako10k/sealgraph/internal/store"
)

type dumpObservation struct {
	Objects []store.Object
	REFs    []migration.RefRecord
	Tags    []migration.TagRecord
}

// DumpLogicalV1 returns a fully validated and buffered format-3 logical dump.
// It does not acquire the writer guard or change repository state.
func (r *Repository) DumpLogicalV1(ctx context.Context) ([]byte, error) {
	return r.dumpLogicalV1(ctx, nil)
}

func (r *Repository) dumpLogicalV1(ctx context.Context, beforeRevalidate func()) ([]byte, error) {
	before, err := r.captureDumpObservation(ctx)
	if err != nil {
		return nil, err
	}
	dump, err := buildLogicalDumpV1(before)
	if err != nil {
		return nil, err
	}
	encoded, err := migration.EncodeLogicalDumpV1(dump)
	if err != nil {
		return nil, fmt.Errorf("encode logical-v1 dump: %w", err)
	}
	if beforeRevalidate != nil {
		beforeRevalidate()
	}
	after, err := r.captureDumpObservation(ctx)
	if err != nil {
		return nil, err
	}
	if !reflect.DeepEqual(before, after) {
		return nil, fmt.Errorf("repository changed during logical dump; stop concurrent writes and rerun")
	}
	return encoded, nil
}

func (r *Repository) captureDumpObservation(ctx context.Context) (dumpObservation, error) {
	if err := validateCanonicalLayout(r.dir); err != nil {
		return dumpObservation{}, fmt.Errorf("validate format-3 repository for logical dump: %w", err)
	}
	candidates, err := r.candidates.List()
	if err != nil {
		return dumpObservation{}, fmt.Errorf("inventory working candidates: %w", err)
	}
	if len(candidates) != 0 {
		ref := candidates[0]
		return dumpObservation{}, fmt.Errorf("working candidate %s blocks logical dump; inspect it with 'sealgraph candidate show %s', then seal or discard it explicitly", ref, ref)
	}
	objects, err := r.objects.List(ctx)
	if err != nil {
		return dumpObservation{}, fmt.Errorf("inventory loose objects: %w", err)
	}
	refs, err := r.refs.List(ctx)
	if err != nil {
		return dumpObservation{}, err
	}
	refRecords := make([]migration.RefRecord, 0, len(refs))
	for _, ref := range refs {
		head, err := r.refs.Resolve(ctx, ref)
		if err != nil {
			return dumpObservation{}, fmt.Errorf("resolve dump REF %s: %w", ref, err)
		}
		refRecords = append(refRecords, migration.RefRecord{Name: ref, Head: head})
	}
	scopedTags, err := r.tags.ListAll(ctx, refs)
	if err != nil {
		return dumpObservation{}, err
	}
	tagRecords := make([]migration.TagRecord, 0, len(scopedTags))
	for _, tag := range scopedTags {
		tagRecords = append(tagRecords, migration.TagRecord{REF: tag.REF, Name: tag.Name, Target: tag.Seal})
	}
	return dumpObservation{Objects: objects, REFs: refRecords, Tags: tagRecords}, nil
}

func buildLogicalDumpV1(observation dumpObservation) (migration.LogicalDumpV1, error) {
	objects := make(map[string]store.Object, len(observation.Objects))
	for _, object := range observation.Objects {
		objects[object.ID.String()] = object
	}

	seals := make(map[string]domain.SealPayload)
	var loadSeal func(domain.ObjectID, string, string) error
	loadSeal = func(id domain.ObjectID, expectedREF, relation string) error {
		if payload, exists := seals[id.String()]; exists {
			if payload.REF != expectedREF {
				return fmt.Errorf("%s %s belongs to REF %s, not %s", relation, id, payload.REF, expectedREF)
			}
			return nil
		}
		object, exists := objects[id.String()]
		if !exists {
			return fmt.Errorf("%s %s is missing from the loose object inventory", relation, id)
		}
		if object.Type != domain.BlobType {
			return fmt.Errorf("%s %s has type %s, expected blob", relation, id, object.Type)
		}
		payload, err := canonical.DecodeSeal(object.Data)
		if err != nil {
			return fmt.Errorf("%s %s is not a canonical format-3 Seal: %w", relation, id, err)
		}
		if payload.REF != expectedREF {
			return fmt.Errorf("%s %s belongs to REF %s, not %s", relation, id, payload.REF, expectedREF)
		}
		seals[id.String()] = payload
		if _, exists := objects[payload.Content.ID.String()]; !exists {
			return fmt.Errorf("seal %s content object %s is missing", id, payload.Content.ID)
		}
		for _, attachment := range payload.Attachments {
			if _, exists := objects[attachment.Blob.ID.String()]; !exists {
				return fmt.Errorf("seal %s attachment %q object %s is missing", id, attachment.Name, attachment.Blob.ID)
			}
		}
		if payload.Parent != nil {
			if err := loadSeal(*payload.Parent, payload.REF, "parent"); err != nil {
				return err
			}
		}
		for _, link := range payload.Links {
			if err := loadSeal(link.TargetSeal, link.TargetREF, "Cause target"); err != nil {
				return err
			}
		}
		return nil
	}

	for _, ref := range observation.REFs {
		if err := loadSeal(ref.Head, ref.Name, "REF head"); err != nil {
			return migration.LogicalDumpV1{}, fmt.Errorf("validate REF %s: %w", ref.Name, err)
		}
	}
	for _, tag := range observation.Tags {
		if err := loadSeal(tag.Target, tag.REF, "tag target"); err != nil {
			return migration.LogicalDumpV1{}, fmt.Errorf("validate tag %s@%s: %w", tag.REF, tag.Name, err)
		}
	}

	order, err := dependencyFirstSealOrder(seals)
	if err != nil {
		return migration.LogicalDumpV1{}, err
	}
	sealRecords := make([]migration.SealRecord, 0, len(order))
	materialIDs := make(map[string]struct{})
	for _, id := range order {
		payload := seals[id]
		sealRecords = append(sealRecords, migration.SealRecord{OldSealID: domain.ObjectID{Hex: id}, Payload: payload})
		materialIDs[payload.Content.ID.String()] = struct{}{}
		for _, attachment := range payload.Attachments {
			materialIDs[attachment.Blob.ID.String()] = struct{}{}
		}
	}
	materialOrder := make([]string, 0, len(materialIDs))
	for id := range materialIDs {
		materialOrder = append(materialOrder, id)
	}
	sort.Strings(materialOrder)
	objectRecords := make([]migration.ObjectRecord, 0, len(materialOrder))
	for _, id := range materialOrder {
		object := objects[id]
		objectRecords = append(objectRecords, migration.ObjectRecord{ID: object.ID, Data: bytes.Clone(object.Data)})
	}
	excluded := make([]domain.ObjectID, 0)
	for _, object := range observation.Objects {
		_, isSeal := seals[object.ID.String()]
		_, isMaterial := materialIDs[object.ID.String()]
		if !isSeal && !isMaterial {
			excluded = append(excluded, object.ID)
		}
	}
	return migration.LogicalDumpV1{
		Objects:         objectRecords,
		Seals:           sealRecords,
		REFs:            observation.REFs,
		Tags:            observation.Tags,
		ExcludedObjects: excluded,
	}, nil
}

func dependencyFirstSealOrder(seals map[string]domain.SealPayload) ([]string, error) {
	indegree := make(map[string]int, len(seals))
	dependents := make(map[string][]string)
	for id, payload := range seals {
		dependencies := make(map[string]struct{}, len(payload.Links)+1)
		if payload.Parent != nil {
			dependencies[payload.Parent.String()] = struct{}{}
		}
		for _, link := range payload.Links {
			dependencies[link.TargetSeal.String()] = struct{}{}
		}
		indegree[id] = len(dependencies)
		for dependency := range dependencies {
			if _, exists := seals[dependency]; !exists {
				return nil, fmt.Errorf("seal %s dependency %s is outside the exported graph", id, dependency)
			}
			dependents[dependency] = append(dependents[dependency], id)
		}
	}
	ready := make([]string, 0)
	for id, degree := range indegree {
		if degree == 0 {
			ready = append(ready, id)
		}
	}
	sort.Strings(ready)
	order := make([]string, 0, len(seals))
	for len(ready) != 0 {
		current := ready[0]
		ready = ready[1:]
		order = append(order, current)
		children := dependents[current]
		sort.Strings(children)
		for _, child := range children {
			indegree[child]--
			if indegree[child] == 0 {
				ready = append(ready, child)
			}
		}
		sort.Strings(ready)
	}
	if len(order) != len(seals) {
		return nil, fmt.Errorf("format-3 Seal graph has a combined parent/Cause rebuild cycle; repair provenance explicitly before dumping")
	}
	return order, nil
}
