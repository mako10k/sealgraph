package format4extract

import (
	"context"
	"fmt"
	"sort"

	"github.com/mako10k/sealgraph/internal/canonical"
	"github.com/mako10k/sealgraph/internal/domain"
	"github.com/mako10k/sealgraph/internal/migration"
	"github.com/mako10k/sealgraph/internal/store/refmanifest"
)

type Result struct {
	Document []byte
	Warnings []string
}

func Extract(ctx context.Context, workDir string) (Result, error) {
	return extract(ctx, workDir, nil)
}

func extract(ctx context.Context, workDir string, afterFirstCapture func()) (Result, error) {
	first, err := captureSource(ctx, workDir)
	if err != nil {
		return Result{}, fmt.Errorf("capture format-4 source: %w", err)
	}
	if len(first.candidates) != 0 {
		return Result{}, fmt.Errorf("FORMAT4_CANDIDATE_STATE_PRESENT: seal or discard every format-4 Candidate before extraction")
	}
	document, semantic, err := buildDocument(ctx, first)
	if err != nil {
		return Result{}, err
	}
	if afterFirstCapture != nil {
		afterFirstCapture()
	}
	second, err := captureSource(ctx, workDir)
	if err != nil {
		return Result{}, fmt.Errorf("MIGRATION_SOURCE_CHANGED: second source capture failed: %w", err)
	}
	if !first.sameObservation(second) {
		return Result{}, fmt.Errorf("MIGRATION_SOURCE_CHANGED: format-4 config, REF manifests, loose objects, or Candidate namespace changed during extraction; retry from a stable retained source")
	}
	return Result{Document: document, Warnings: migration.SemanticWarnings(semantic)}, nil
}

func buildDocument(ctx context.Context, snapshot sourceSnapshot) ([]byte, migration.UniversalSemanticProjection, error) {
	allObjects, err := decodeObjectInventory(ctx, snapshot.objectBytes)
	if err != nil {
		return nil, migration.UniversalSemanticProjection{}, err
	}
	refs, tags, roots, err := decodeREFInventory(snapshot.refs)
	if err != nil {
		return nil, migration.UniversalSemanticProjection{}, err
	}
	closure := newSealClosure(ctx, allObjects)
	for _, root := range roots {
		if err := closure.add(root); err != nil {
			return nil, migration.UniversalSemanticProjection{}, err
		}
	}
	value, err := closure.document(allObjects, refs, tags)
	if err != nil {
		return nil, migration.UniversalSemanticProjection{}, err
	}
	if err := migration.CanonicalizeUniversalBlobV1(&value); err != nil {
		return nil, migration.UniversalSemanticProjection{}, fmt.Errorf("order format-4 migration inventory: %w", err)
	}
	projection, err := migration.ComputeUniversalBlobV1Projection(value)
	if err != nil {
		return nil, migration.UniversalSemanticProjection{}, err
	}
	value.Projection = projection.Semantic
	if err := migration.CanonicalizeUniversalBlobV1(&value); err != nil {
		return nil, migration.UniversalSemanticProjection{}, err
	}
	document, err := migration.EncodeUniversalBlobV1(value)
	if err != nil {
		return nil, migration.UniversalSemanticProjection{}, err
	}
	return document, projection.Semantic, nil
}

func decodeObjectInventory(ctx context.Context, physical map[string][]byte) (map[string][]byte, error) {
	paths := make([]string, 0, len(physical))
	for path := range physical {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	result := make(map[string][]byte, len(paths))
	for _, path := range paths {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		id, err := objectIDFromPath(path)
		if err != nil {
			return nil, err
		}
		data, err := decodeLooseObject(id, physical[path])
		if err != nil {
			return nil, err
		}
		result[id.String()] = data
	}
	return result, nil
}

func decodeREFInventory(raw map[string][]byte) ([]migration.RefRecord, []migration.TagRecord, []domain.ObjectID, error) {
	names := make([]string, 0, len(raw))
	for name := range raw {
		names = append(names, name)
	}
	sort.Strings(names)
	refs := make([]migration.RefRecord, 0, len(names))
	tags := []migration.TagRecord{}
	roots := []domain.ObjectID{}
	for _, name := range names {
		if err := domain.ValidateREF(name); err != nil {
			return nil, nil, nil, fmt.Errorf("invalid stored REF path %q: %w", name, err)
		}
		manifest, err := refmanifest.Decode(raw[name])
		if err != nil {
			return nil, nil, nil, fmt.Errorf("REF %s manifest is corrupt: %w", name, err)
		}
		refs = append(refs, migration.RefRecord{Name: name, Head: manifest.Head})
		roots = append(roots, manifest.Head)
		for _, tag := range manifest.Tags {
			tags = append(tags, migration.TagRecord{REF: name, Name: tag.Name, Target: tag.Seal})
			roots = append(roots, tag.Seal)
		}
	}
	sort.Slice(roots, func(i, j int) bool { return roots[i].String() < roots[j].String() })
	return refs, tags, roots, nil
}

type sealClosure struct {
	ctx        context.Context
	objects    map[string][]byte
	state      map[string]uint8
	seals      map[string]migration.UniversalSealRecord
	referenced map[string]bool
}

func newSealClosure(ctx context.Context, objects map[string][]byte) *sealClosure {
	return &sealClosure{ctx: ctx, objects: objects, state: make(map[string]uint8), seals: make(map[string]migration.UniversalSealRecord), referenced: make(map[string]bool)}
}

func (closure *sealClosure) add(id domain.ObjectID) error {
	if err := closure.ctx.Err(); err != nil {
		return err
	}
	if closure.state[id.String()] == 1 {
		return fmt.Errorf("format-4 migration Seal graph contains a cycle at %s", id)
	}
	if closure.state[id.String()] == 2 {
		return nil
	}
	data, ok := closure.objects[id.String()]
	if !ok {
		return fmt.Errorf("format-4 Seal %s is absent from the loose object inventory", id)
	}
	payload, err := canonical.DecodeSeal(data)
	if err != nil {
		return fmt.Errorf("format-4 graph target %s is not a canonical Seal: %w", id, err)
	}
	closure.state[id.String()] = 1
	if payload.ParentRevision != nil {
		if err := closure.add(*payload.ParentRevision); err != nil {
			return err
		}
	}
	for _, link := range payload.Links {
		if err := closure.add(link.TargetSeal); err != nil {
			return err
		}
	}
	closure.referenced[payload.Content.ID.String()] = true
	for _, attachment := range payload.Attachments {
		closure.referenced[attachment.Blob.ID.String()] = true
	}
	closure.seals[id.String()] = migration.UniversalSealRecord{ID: id, Payload: payload, PayloadBytes: data}
	closure.state[id.String()] = 2
	return nil
}

func (closure *sealClosure) document(objects map[string][]byte, refs []migration.RefRecord, tags []migration.TagRecord) (migration.UniversalBlobV1, error) {
	value := migration.UniversalBlobV1{REFs: refs, Tags: tags, ExcludedState: append([]string{}, migration.UniversalExcludedState...)}
	for _, record := range closure.seals {
		value.Seals = append(value.Seals, record)
	}
	for id := range closure.referenced {
		data, ok := objects[id]
		if !ok {
			return migration.UniversalBlobV1{}, fmt.Errorf("referenced format-4 Blob %s is absent from the loose object inventory", id)
		}
		value.Objects = append(value.Objects, migration.ObjectRecord{ID: domain.ObjectID{Hex: id}, Data: data})
	}
	for id := range objects {
		if _, seal := closure.seals[id]; seal || closure.referenced[id] {
			continue
		}
		value.ExcludedObjects = append(value.ExcludedObjects, domain.ObjectID{Hex: id})
	}
	return value, nil
}
