package repository

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	canonicalv5 "github.com/mako10k/sealgraph/internal/canonical/v5"
	canonicalv6 "github.com/mako10k/sealgraph/internal/canonical/v6"
	"github.com/mako10k/sealgraph/internal/domain"
	domainv5 "github.com/mako10k/sealgraph/internal/domain/v5"
	"github.com/mako10k/sealgraph/internal/store"
	"github.com/mako10k/sealgraph/internal/store/native"
)

type FsckReport struct {
	Blobs, Seals, Materials, Provenances, REFs, Tags, ActiveSeals int
	SealsV5, SealsV6, ProvenancesV1, ProvenancesV2                int
	CandidatesV5, CandidatesV6                                    int
	HistoricalOrDetachedSeals                                     []domain.ObjectID
	UnreferencedBlobs                                             []domain.ObjectID
}

type fsckInventory struct {
	objects              []store.Object
	seals                map[string]domainv5.Seal
	materials            map[string]domainv5.Material
	provenances          map[string]domainv5.Provenance
	sealGeneration       map[string]int
	provenanceGeneration map[string]int
}
type fsckTag struct {
	REF, Name string
	Seal      domain.ObjectID
}

func (r *Repository) Fsck(ctx context.Context) (FsckReport, error) {
	physical, err := capturePhysicalRepositoryObservation(r.dir)
	if err != nil {
		return FsckReport{}, fmt.Errorf("capture physical repository observation for fsck: %w", err)
	}
	if err := validateFsckPhysicalRepository(physical); err != nil {
		return FsckReport{}, fmt.Errorf("validate captured physical repository for fsck: %w", err)
	}
	observation, err := r.observeHeads(ctx, "fsck")
	if err != nil {
		return FsckReport{}, err
	}
	if err := validateFsckREFObservationPhysical(observation, physical); err != nil {
		return FsckReport{}, err
	}
	inventory, err := fsckInventoryFromPhysicalFormat(ctx, physical, r.format)
	if err != nil {
		return FsckReport{}, err
	}
	if err := validateFsckObjectInventoryPhysical(inventory.objects, physical); err != nil {
		return FsckReport{}, err
	}
	tags := fsckObservedTags(observation)
	if err := validateFsckTargets(observation, tags, inventory.seals); err != nil {
		return FsckReport{}, err
	}
	if err := validateFsckTypedReferences(inventory); err != nil {
		return FsckReport{}, err
	}
	resolvedSeals := fsckResolvedSeals(inventory)
	allHeads := make(map[string]domain.ObjectID, len(inventory.seals))
	for id := range inventory.seals {
		allHeads["fsck/"+id] = domain.ObjectID{Hex: id}
	}
	if _, err := r.buildObservedGraph(ctx, allHeads, resolvedSeals); err != nil {
		return FsckReport{}, fmt.Errorf("validate complete repository graph: %w", err)
	}
	activeGraph, err := r.buildObservedGraph(ctx, observation.heads, resolvedSeals)
	if err != nil {
		return FsckReport{}, fmt.Errorf("validate active repository graph: %w", err)
	}
	referenced := fsckReferencedClosure(inventory)
	candidatesV5, candidatesV6, err := r.fsckCandidateGenerations(ctx)
	if err != nil {
		return FsckReport{}, err
	}
	report := buildFsckReport(inventory, observation, tags, activeGraph, referenced, candidatesV5, candidatesV6)
	if err := r.validateFsckFinalObservation(ctx, observation, physical); err != nil {
		return FsckReport{}, err
	}
	return report, nil
}

func buildFsckReport(inventory fsckInventory, observation headObservation, tags []fsckTag, activeGraph *observedGraph, referenced map[string]bool, candidatesV5, candidatesV6 int) FsckReport {
	report := FsckReport{Blobs: len(inventory.objects), Seals: len(inventory.seals), Materials: len(inventory.materials), Provenances: len(inventory.provenances), REFs: len(observation.names), Tags: len(tags), ActiveSeals: len(activeGraph.active), HistoricalOrDetachedSeals: []domain.ObjectID{}, UnreferencedBlobs: []domain.ObjectID{}, CandidatesV5: candidatesV5, CandidatesV6: candidatesV6}
	for _, generation := range inventory.sealGeneration {
		if generation == 5 {
			report.SealsV5++
		} else {
			report.SealsV6++
		}
	}
	for _, generation := range inventory.provenanceGeneration {
		if generation == 1 {
			report.ProvenancesV1++
		} else {
			report.ProvenancesV2++
		}
	}
	for id := range inventory.seals {
		if !activeGraph.active[id] {
			report.HistoricalOrDetachedSeals = append(report.HistoricalOrDetachedSeals, domain.ObjectID{Hex: id})
		}
	}
	for _, object := range inventory.objects {
		if !referenced[object.ID.String()] {
			report.UnreferencedBlobs = append(report.UnreferencedBlobs, object.ID)
		}
	}
	sortIDs(report.HistoricalOrDetachedSeals)
	sortIDs(report.UnreferencedBlobs)
	return report
}

func validateFsckPhysicalRepository(observation physicalRepositoryObservation) error {
	entries := make(map[string]physicalEntryObservation, len(observation.Entries))
	for _, entry := range observation.Entries {
		if _, duplicate := entries[entry.Path]; duplicate {
			return fmt.Errorf("physical observation contains duplicate path %q", entry.Path)
		}
		entries[entry.Path] = entry
	}
	if err := validateFsckFixedPhysicalEntries(entries); err != nil {
		return err
	}
	return validateFsckPhysicalNamespaceEntries(observation.Entries)
}

func validateFsckFixedPhysicalEntries(entries map[string]physicalEntryObservation) error {
	if err := requireFsckDirectory(entries, "ROOT"); err != nil {
		return err
	}
	config, ok := entries["CONFIG"]
	if !ok || !config.Present || !config.Mode.IsRegular() {
		return fmt.Errorf("CONFIG is absent or not a regular file")
	}
	expected := configBytes
	if string(config.Data) == format6ConfigBytes {
		expected = format6ConfigBytes
	}
	expectedConfigDigest := sha256.Sum256([]byte(expected))
	if config.Size != int64(len(expected)) || config.SHA256 != expectedConfigDigest {
		return fmt.Errorf("CONFIG bytes are not an exact supported config")
	}
	if err := requireFsckDirectory(entries, "objects"); err != nil {
		return err
	}
	if err := requireFsckDirectory(entries, "refs"); err != nil {
		return err
	}
	if err := requireFsckDirectory(entries, "refs/seals"); err != nil {
		return err
	}
	return nil
}

func validateFsckPhysicalNamespaceEntries(entries []physicalEntryObservation) error {
	for _, entry := range entries {
		path := entry.Path
		switch {
		case path == "ROOT" || path == "CONFIG" || path == "objects" || path == "refs" || path == "refs/seals":
			continue
		case strings.HasPrefix(path, "objects/"):
			if err := validateFsckObjectPath(path, entry); err != nil {
				return err
			}
		case strings.HasPrefix(path, "refs/seals/"):
			if err := validateFsckREFPath(path, entry); err != nil {
				return err
			}
		default:
			return fmt.Errorf("unexpected canonical namespace entry %q", path)
		}
	}
	return nil
}

func requireFsckDirectory(entries map[string]physicalEntryObservation, path string) error {
	entry, ok := entries[path]
	if !ok || !entry.Present || !entry.Mode.IsDir() || entry.Mode&os.ModeSymlink != 0 {
		return fmt.Errorf("%s is absent or not a real directory", path)
	}
	return nil
}

func validateFsckObjectPath(path string, entry physicalEntryObservation) error {
	parts := strings.Split(strings.TrimPrefix(path, "objects/"), "/")
	if len(parts) == 1 && len(parts[0]) == 2 && isFsckLowerHex(parts[0]) && entry.Mode.IsDir() {
		return nil
	}
	if len(parts) == 2 && len(parts[0]) == 2 && len(parts[1]) == 62 && isFsckLowerHex(parts[0]+parts[1]) && entry.Mode.IsRegular() {
		return nil
	}
	return fmt.Errorf("invalid physical object namespace entry %q", path)
}

func validateFsckREFPath(path string, entry physicalEntryObservation) error {
	relative := strings.TrimPrefix(path, "refs/seals/")
	if entry.Mode.IsDir() {
		if err := domain.ValidateREF(relative); err != nil {
			return fmt.Errorf("invalid physical REF directory %q: %w", path, err)
		}
		return nil
	}
	if !entry.Mode.IsRegular() || filepath.Base(relative) != ".ref" {
		return fmt.Errorf("invalid physical REF namespace entry %q", path)
	}
	ref := filepath.ToSlash(filepath.Dir(relative))
	if err := domain.ValidateREF(ref); err != nil {
		return fmt.Errorf("invalid physical REF manifest path %q: %w", path, err)
	}
	return nil
}

func isFsckLowerHex(value string) bool {
	for _, character := range value {
		if (character < '0' || character > '9') && (character < 'a' || character > 'f') {
			return false
		}
	}
	return true
}

func validateFsckREFObservationPhysical(observation headObservation, physical physicalRepositoryObservation) error {
	expected := make(map[string][]byte, len(observation.names))
	for _, ref := range observation.names {
		expected["refs/seals/"+ref+"/.ref"] = observation.manifests[ref]
	}
	seen := 0
	for _, entry := range physical.Entries {
		if !strings.HasPrefix(entry.Path, "refs/seals/") || !entry.Mode.IsRegular() {
			continue
		}
		manifest, ok := expected[entry.Path]
		if !ok {
			return fmt.Errorf("physical REF manifest %q is absent from logical REF observation", entry.Path)
		}
		digest := sha256.Sum256(manifest)
		if entry.Size != int64(len(manifest)) || entry.SHA256 != digest {
			return fmt.Errorf("logical REF manifest %q differs from captured physical bytes", entry.Path)
		}
		seen++
	}
	if seen != len(expected) {
		return fmt.Errorf("logical REF observation has %d manifests; captured physical namespace has %d", len(expected), seen)
	}
	return nil
}

func validateFsckObjectInventoryPhysical(objects []store.Object, physical physicalRepositoryObservation) error {
	expected := make(map[string]bool, len(objects))
	for _, object := range objects {
		id := object.ID.String()
		expected["objects/"+id[:2]+"/"+id[2:]] = true
	}
	seen := 0
	for _, entry := range physical.Entries {
		if !strings.HasPrefix(entry.Path, "objects/") || !entry.Mode.IsRegular() {
			continue
		}
		if !expected[entry.Path] {
			return fmt.Errorf("captured physical object %q is absent from validated object inventory", entry.Path)
		}
		seen++
	}
	if seen != len(expected) {
		return fmt.Errorf("validated object inventory has %d Blobs; captured physical namespace has %d", len(expected), seen)
	}
	return nil
}

func validateFsckTypedReferences(inventory fsckInventory) error {
	objects := make(map[string]bool, len(inventory.objects))
	for _, object := range inventory.objects {
		objects[object.ID.String()] = true
	}
	for id, seal := range inventory.seals {
		if _, ok := inventory.materials[seal.Material.String()]; !ok {
			return fmt.Errorf("Seal %s references Blob %s that is not a canonical Material", id, seal.Material)
		}
		if _, ok := inventory.provenances[seal.Provenance.String()]; !ok {
			return fmt.Errorf("Seal %s references Blob %s that is not canonical Provenance", id, seal.Provenance)
		}
		sealGeneration := inventory.sealGeneration[id]
		provenanceGeneration := inventory.provenanceGeneration[seal.Provenance.String()]
		if (sealGeneration == 5 && provenanceGeneration != 1) || (sealGeneration == 6 && provenanceGeneration != 2) {
			return fmt.Errorf("Seal %s generation v%d cross-pairs with Provenance generation v%d", id, sealGeneration, provenanceGeneration)
		}
	}
	for id, material := range inventory.materials {
		if !objects[material.Content.String()] {
			return fmt.Errorf("Material %s references missing content Blob %s", id, material.Content)
		}
		for _, attachment := range material.Attachments {
			if !objects[attachment.Blob.String()] {
				return fmt.Errorf("Material %s attachment %q references missing Blob %s", id, attachment.Name, attachment.Blob)
			}
		}
	}
	for id, provenance := range inventory.provenances {
		for _, link := range provenance.CauseLinks {
			if _, ok := inventory.seals[link.TargetSeal.String()]; !ok {
				return fmt.Errorf("Provenance %s Cause target %s is not a canonical Seal", id, link.TargetSeal)
			}
			for _, previous := range link.PreviousRevisionSealOfTargetSeal {
				if _, ok := inventory.seals[previous.String()]; !ok {
					return fmt.Errorf("Provenance %s previous revision %s is not a canonical Seal", id, previous)
				}
			}
		}
	}
	return nil
}

func fsckReferencedClosure(inventory fsckInventory) map[string]bool {
	referenced := make(map[string]bool)
	queue := make([]string, 0, len(inventory.seals))
	for id := range inventory.seals {
		queue = append(queue, id)
	}
	for len(queue) > 0 {
		id := queue[len(queue)-1]
		queue = queue[:len(queue)-1]
		if referenced[id] {
			continue
		}
		referenced[id] = true
		if seal, ok := inventory.seals[id]; ok {
			queue = append(queue, seal.Material.String(), seal.Provenance.String())
		}
		if material, ok := inventory.materials[id]; ok {
			queue = append(queue, material.Content.String())
			for _, attachment := range material.Attachments {
				queue = append(queue, attachment.Blob.String())
			}
		}
		if provenance, ok := inventory.provenances[id]; ok {
			for _, link := range provenance.CauseLinks {
				queue = append(queue, link.TargetSeal.String())
				for _, previous := range link.PreviousRevisionSealOfTargetSeal {
					queue = append(queue, previous.String())
				}
			}
		}
	}
	return referenced
}

func (r *Repository) fsckInventory(ctx context.Context) (fsckInventory, error) {
	objects, err := r.objects.List(ctx)
	if err != nil {
		return fsckInventory{}, fmt.Errorf("inventory immutable Blobs: %w", err)
	}
	return classifyFsckObjects(objects, r.format), nil
}

func fsckInventoryFromPhysical(ctx context.Context, physical physicalRepositoryObservation) (fsckInventory, error) {
	return fsckInventoryFromPhysicalFormat(ctx, physical, 5)
}

func fsckInventoryFromPhysicalFormat(ctx context.Context, physical physicalRepositoryObservation, format int) (fsckInventory, error) {
	objects := []store.Object{}
	for _, entry := range physical.Entries {
		if err := ctx.Err(); err != nil {
			return fsckInventory{}, err
		}
		if !strings.HasPrefix(entry.Path, "objects/") || !entry.Mode.IsRegular() {
			continue
		}
		idText := strings.ReplaceAll(strings.TrimPrefix(entry.Path, "objects/"), "/", "")
		id, err := domain.ParseObjectID(idText)
		if err != nil {
			return fsckInventory{}, fmt.Errorf("invalid captured object path %q: %w", entry.Path, err)
		}
		if int64(len(entry.Data)) != entry.Size || sha256.Sum256(entry.Data) != entry.SHA256 {
			return fsckInventory{}, fmt.Errorf("captured object %s bytes do not match its physical observation tuple", id)
		}
		object, err := native.DecodeLooseObject(id, entry.Data)
		if err != nil {
			return fsckInventory{}, fmt.Errorf("validate captured object %s: %w", id, err)
		}
		objects = append(objects, object)
	}
	sort.Slice(objects, func(i, j int) bool { return objects[i].ID.String() < objects[j].ID.String() })
	return classifyFsckObjects(objects, format), nil
}

func classifyFsckObjects(objects []store.Object, format int) fsckInventory {
	result := fsckInventory{objects: objects, seals: make(map[string]domainv5.Seal), materials: make(map[string]domainv5.Material), provenances: make(map[string]domainv5.Provenance), sealGeneration: make(map[string]int), provenanceGeneration: make(map[string]int)}
	for _, object := range objects {
		if value, err := canonicalv5.DecodeSeal(object.Data); err == nil {
			result.seals[object.ID.String()] = value
			result.sealGeneration[object.ID.String()] = 5
		}
		if value, err := canonicalv5.DecodeMaterial(object.Data); err == nil {
			result.materials[object.ID.String()] = value
		}
		if value, err := canonicalv5.DecodeProvenance(object.Data); err == nil {
			result.provenances[object.ID.String()] = value
			result.provenanceGeneration[object.ID.String()] = 1
		}
		if format == 6 {
			if value, err := canonicalv6.DecodeSeal(object.Data); err == nil {
				result.seals[object.ID.String()] = value
				result.sealGeneration[object.ID.String()] = 6
			}
			if value, err := canonicalv6.DecodeProvenance(object.Data); err == nil {
				result.provenances[object.ID.String()] = value
				result.provenanceGeneration[object.ID.String()] = 2
			}
		}
	}
	return result
}

func (r *Repository) fsckCandidateGenerations(ctx context.Context) (int, int, error) {
	names, err := r.candidates.List()
	if err != nil {
		return 0, 0, fmt.Errorf("validate Candidate namespace for fsck: %w", err)
	}
	v5, v6 := 0, 0
	for _, name := range names {
		snapshot, err := r.candidates.LoadSnapshot(name)
		if err != nil {
			return 0, 0, err
		}
		if _, err := canonicalv6.DecodeCandidate(snapshot.Bytes); err == nil {
			v6++
		} else if _, err := canonicalv5.DecodeCandidate(snapshot.Bytes); err == nil {
			v5++
		} else {
			return 0, 0, fmt.Errorf("Candidate %s has unsupported generation", name)
		}
		if r.format == 6 {
			if _, err := r.InspectCandidate(ctx, name); err != nil {
				return 0, 0, fmt.Errorf("validate Candidate %s closure for fsck: %w", name, err)
			}
		}
	}
	return v5, v6, nil
}

func fsckResolvedSeals(inventory fsckInventory) map[string]domainv5.ResolvedSeal {
	objects := make(map[string]store.Object, len(inventory.objects))
	for _, object := range inventory.objects {
		objects[object.ID.String()] = object
	}
	result := make(map[string]domainv5.ResolvedSeal, len(inventory.seals))
	for id, seal := range inventory.seals {
		material := inventory.materials[seal.Material.String()]
		result[id] = domainv5.ResolvedSeal{
			ID: domain.ObjectID{Hex: id}, Seal: seal, Material: material,
			Provenance: inventory.provenances[seal.Provenance.String()], ContentBytes: len(objects[material.Content.String()].Data),
		}
	}
	return result
}

func validateFsckTargets(observation headObservation, tags []fsckTag, seals map[string]domainv5.Seal) error {
	for _, ref := range observation.names {
		if _, ok := seals[observation.heads[ref].String()]; !ok {
			return fmt.Errorf("REF %s head %s is not a supported canonical Seal", ref, observation.heads[ref])
		}
	}
	for _, tag := range tags {
		if _, ok := seals[tag.Seal.String()]; !ok {
			return fmt.Errorf("tag %s@%s target %s is not a supported canonical Seal", tag.REF, tag.Name, tag.Seal)
		}
	}
	return nil
}
func sortIDs(ids []domain.ObjectID) {
	sort.Slice(ids, func(i, j int) bool { return ids[i].String() < ids[j].String() })
}

func (r *Repository) validateFsckFinalObservation(ctx context.Context, observation headObservation, physical physicalRepositoryObservation) error {
	if err := r.revalidateHeads(ctx, observation, "fsck"); err != nil {
		return err
	}
	finalPhysical, err := capturePhysicalRepositoryObservation(r.dir)
	if err != nil || !equalPhysicalRepositoryObservations(physical, finalPhysical) {
		return fmt.Errorf("physical repository namespace changed or became unreadable while deriving fsck; rerun the command")
	}
	return nil
}

func fsckObservedTags(observation headObservation) []fsckTag {
	result := []fsckTag{}
	for _, ref := range observation.names {
		names := make([]string, 0, len(observation.tags[ref]))
		for name := range observation.tags[ref] {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			result = append(result, fsckTag{REF: ref, Name: name, Seal: observation.tags[ref][name]})
		}
	}
	return result
}
