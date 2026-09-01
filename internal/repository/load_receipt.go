package repository

import (
	"context"
	"crypto/sha256"
	"fmt"
	"sort"

	"github.com/mako10k/sealgraph/internal/canonical"
	"github.com/mako10k/sealgraph/internal/domain"
	"github.com/mako10k/sealgraph/internal/migration"
)

const universalLoadReceiptV1Schema = "sealgraph/universal-blob-load-receipt/v1"

type repositoryRefDigest struct {
	Name string
	Head domain.ObjectID
}
type repositoryTagDigest struct {
	REF, Name string
	Target    domain.ObjectID
}

func repositoryDigest(ctx context.Context, repo *Repository) (string, error) {
	physical, err := capturePhysicalRepositoryObservation(repo.dir)
	if err != nil {
		return "", fmt.Errorf("capture repository digest observation: %w", err)
	}
	if err := validateFsckPhysicalRepository(physical); err != nil {
		return "", fmt.Errorf("validate repository digest physical observation: %w", err)
	}
	observation, err := repo.observeHeads(ctx, "repository digest")
	if err != nil {
		return "", err
	}
	if err := validateFsckREFObservationPhysical(observation, physical); err != nil {
		return "", err
	}
	inventory, err := fsckInventoryFromPhysical(ctx, physical)
	if err != nil {
		return "", err
	}
	objectIDs := make([]domain.ObjectID, 0, len(inventory.objects))
	for _, object := range inventory.objects {
		objectIDs = append(objectIDs, object.ID)
	}
	sort.Slice(objectIDs, func(i, j int) bool { return objectIDs[i].String() < objectIDs[j].String() })
	refRecords := make([]repositoryRefDigest, 0, len(observation.names))
	tagRecords := []repositoryTagDigest{}
	for _, ref := range observation.names {
		refRecords = append(refRecords, repositoryRefDigest{Name: ref, Head: observation.heads[ref]})
	}
	for _, tag := range fsckObservedTags(observation) {
		tagRecords = append(tagRecords, repositoryTagDigest{REF: tag.REF, Name: tag.Name, Target: tag.Seal})
	}
	sort.Slice(tagRecords, func(i, j int) bool {
		if tagRecords[i].REF != tagRecords[j].REF {
			return tagRecords[i].REF < tagRecords[j].REF
		}
		return tagRecords[i].Name < tagRecords[j].Name
	})
	var configDigest [sha256.Size]byte
	for _, entry := range physical.Entries {
		if entry.Path == "CONFIG" {
			configDigest = entry.SHA256
			break
		}
	}
	b := []byte(`{"schema":"sealgraph/repository-observation/v1","format":5,"object_format":"sha256","config_sha256":`)
	b, _ = canonical.AppendString(b, fmt.Sprintf("%x", configDigest))
	b = append(b, `,"objects":[`...)
	for i, id := range objectIDs {
		if i > 0 {
			b = append(b, ',')
		}
		b, _ = canonical.AppendString(b, id.String())
	}
	b = append(b, `],"refs":[`...)
	b = appendRepositoryDigestREFs(b, refRecords)
	b = append(b, `],"tags":[`...)
	b = appendRepositoryDigestTags(b, tagRecords)
	b = append(b, ']', '}')
	if err := repo.revalidateHeads(ctx, observation, "repository digest"); err != nil {
		return "", err
	}
	finalPhysical, err := capturePhysicalRepositoryObservation(repo.dir)
	if err != nil || !equalPhysicalRepositoryObservations(physical, finalPhysical) {
		return "", fmt.Errorf("repository changed or became unreadable while deriving repository digest; rerun from a stable repository")
	}
	digest := sha256.Sum256(b)
	return fmt.Sprintf("%x", digest), nil
}

type collapseGroup struct {
	New domain.ObjectID
	Old []domain.ObjectID
}
type typedObject struct {
	Kind string
	ID   domain.ObjectID
}

func encodeUniversalLoadReceipt(input []byte, dump migration.UniversalBlobV1, projection migrationProjection, repositoryDigest string) ([]byte, error) {
	sourceDigest := sha256.Sum256(input)
	collapses := migrationCollapseGroups(projection.Seals)
	typed := migrationTypedObjects(projection.TypedRoles)
	b := []byte(`{"schema":`)
	b, _ = canonical.AppendString(b, universalLoadReceiptV1Schema)
	b = append(b, `,"source_document_sha256":`...)
	b, _ = canonical.AppendString(b, fmt.Sprintf("%x", sourceDigest))
	b = append(b, `,"seal_mappings":[`...)
	mappings := append([]projectedSeal{}, projection.Seals...)
	sort.Slice(mappings, func(i, j int) bool { return mappings[i].OldID.String() < mappings[j].OldID.String() })
	for i, item := range mappings {
		if i > 0 {
			b = append(b, ',')
		}
		b = append(b, `{"old_seal":`...)
		b, _ = canonical.AppendString(b, item.OldID.String())
		b = append(b, `,"new_seal":`...)
		b, _ = canonical.AppendString(b, item.NewID.String())
		b = append(b, `,"material":`...)
		b, _ = canonical.AppendString(b, item.MaterialID.String())
		b = append(b, `,"provenance":`...)
		b, _ = canonical.AppendString(b, item.ProvenanceID.String())
		b = append(b, '}')
	}
	b = append(b, `],"collapse_groups":[`...)
	for i, item := range collapses {
		if i > 0 {
			b = append(b, ',')
		}
		b = append(b, `{"new_seal":`...)
		b, _ = canonical.AppendString(b, item.New.String())
		b = append(b, `,"old_seals":[`...)
		b = appendIDs(b, item.Old)
		b = append(b, ']', '}')
	}
	b = append(b, `],"typed_objects":[`...)
	for i, item := range typed {
		if i > 0 {
			b = append(b, ',')
		}
		b = append(b, `{"kind":`...)
		b, _ = canonical.AppendString(b, item.Kind)
		b = append(b, `,"id":`...)
		b, _ = canonical.AppendString(b, item.ID.String())
		b = append(b, '}')
	}
	b = append(b, `],"unchanged_objects":[`...)
	unchanged := make([]domain.ObjectID, 0, len(dump.Objects))
	for _, object := range dump.Objects {
		unchanged = append(unchanged, object.ID)
	}
	sortMigrationIDs(unchanged)
	b = appendIDs(b, unchanged)
	b = append(b, `],"refs":[`...)
	b = appendReceiptREFs(b, dump.REFs, projection.OldToNew)
	b = append(b, `],"tags":[`...)
	b = appendReceiptTags(b, dump.Tags, projection.OldToNew)
	b = append(b, `],"semantic_projection":{"materialized_parent_assertions":[`...)
	b = appendMaterializedReceipt(b, projection.Semantic.Materialized, projection.OldToNew)
	b = append(b, `],"unobserved_parent_assertions":[`...)
	b = appendUnobservedReceipt(b, projection.Semantic.Unobserved, projection.OldToNew)
	b = append(b, `],"collapsed_revision_assertions":[`...)
	b = appendCollapsedReceipt(b, projection.Semantic.Collapsed, projection.OldToNew)
	b = append(b, `],"merged_cause_links":[`...)
	b = appendMergedReceipt(b, projection.Semantic.Merged, projection.OldToNew)
	b = append(b, `]},"excluded_objects":[`...)
	b = appendIDs(b, dump.ExcludedObjects)
	b = append(b, `],"excluded_state":[`...)
	for i, item := range dump.ExcludedState {
		if i > 0 {
			b = append(b, ',')
		}
		b, _ = canonical.AppendString(b, item)
	}
	b = append(b, `],"published_format":5,"repository_digest":`...)
	b, _ = canonical.AppendString(b, repositoryDigest)
	b = append(b, '}', '\n')
	return b, nil
}

func appendRepositoryDigestREFs(dst []byte, refs []repositoryRefDigest) []byte {
	for index, ref := range refs {
		dst = appendReceiptSeparator(dst, index)
		dst = appendRepositoryDigestREF(dst, ref)
	}
	return dst
}

func appendRepositoryDigestREF(dst []byte, ref repositoryRefDigest) []byte {
	dst = append(dst, `{"name":`...)
	dst, _ = canonical.AppendString(dst, ref.Name)
	dst = append(dst, `,"head":`...)
	dst, _ = canonical.AppendString(dst, ref.Head.String())
	return append(dst, '}')
}

func appendRepositoryDigestTags(dst []byte, tags []repositoryTagDigest) []byte {
	for index, tag := range tags {
		dst = appendReceiptSeparator(dst, index)
		dst = appendRepositoryDigestTag(dst, tag)
	}
	return dst
}

func appendRepositoryDigestTag(dst []byte, tag repositoryTagDigest) []byte {
	dst = append(dst, `{"ref":`...)
	dst, _ = canonical.AppendString(dst, tag.REF)
	dst = append(dst, `,"name":`...)
	dst, _ = canonical.AppendString(dst, tag.Name)
	dst = append(dst, `,"target":`...)
	dst, _ = canonical.AppendString(dst, tag.Target.String())
	return append(dst, '}')
}

func appendReceiptREFs(dst []byte, refs []migration.RefRecord, mapping map[string]domain.ObjectID) []byte {
	for index, ref := range refs {
		dst = appendReceiptSeparator(dst, index)
		dst = append(dst, `{"name":`...)
		dst, _ = canonical.AppendString(dst, ref.Name)
		dst = append(dst, `,"old_head":`...)
		dst, _ = canonical.AppendString(dst, ref.Head.String())
		dst = append(dst, `,"new_head":`...)
		dst, _ = canonical.AppendString(dst, mapping[ref.Head.String()].String())
		dst = append(dst, '}')
	}
	return dst
}

func appendReceiptTags(dst []byte, tags []migration.TagRecord, mapping map[string]domain.ObjectID) []byte {
	for index, tag := range tags {
		dst = appendReceiptSeparator(dst, index)
		dst = append(dst, `{"ref":`...)
		dst, _ = canonical.AppendString(dst, tag.REF)
		dst = append(dst, `,"name":`...)
		dst, _ = canonical.AppendString(dst, tag.Name)
		dst = append(dst, `,"old_target":`...)
		dst, _ = canonical.AppendString(dst, tag.Target.String())
		dst = append(dst, `,"new_target":`...)
		dst, _ = canonical.AppendString(dst, mapping[tag.Target.String()].String())
		dst = append(dst, '}')
	}
	return dst
}

func appendReceiptSeparator(dst []byte, index int) []byte {
	if index != 0 {
		return append(dst, ',')
	}
	return dst
}

func migrationCollapseGroups(seals []projectedSeal) []collapseGroup {
	byNew := make(map[string][]domain.ObjectID)
	for _, item := range seals {
		byNew[item.NewID.String()] = append(byNew[item.NewID.String()], item.OldID)
	}
	result := []collapseGroup{}
	for text, olds := range byNew {
		if len(olds) > 1 {
			sortMigrationIDs(olds)
			result = append(result, collapseGroup{New: domain.ObjectID{Hex: text}, Old: olds})
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].New.String() < result[j].New.String() })
	return result
}
func migrationTypedObjects(roles map[string]map[string]bool) []typedObject {
	result := []typedObject{}
	for kind, ids := range roles {
		for text := range ids {
			result = append(result, typedObject{Kind: kind, ID: domain.ObjectID{Hex: text}})
		}
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Kind != result[j].Kind {
			return result[i].Kind < result[j].Kind
		}
		return result[i].ID.String() < result[j].ID.String()
	})
	return result
}
func appendIDs(b []byte, values []domain.ObjectID) []byte {
	for i, id := range values {
		if i > 0 {
			b = append(b, ',')
		}
		b, _ = canonical.AppendString(b, id.String())
	}
	return b
}

func appendMaterializedReceipt(b []byte, values []migration.ParentAssertion, mapping map[string]domain.ObjectID) []byte {
	for i, item := range values {
		if i > 0 {
			b = append(b, ',')
		}
		b = appendOldTriple(b, item)
		b = append(b[:len(b)-1], `,"new_observer":`...)
		b, _ = canonical.AppendString(b, mapping[item.Observer.String()].String())
		b = append(b, `,"new_target":`...)
		b, _ = canonical.AppendString(b, mapping[item.Child.String()].String())
		b = append(b, `,"new_previous":`...)
		b, _ = canonical.AppendString(b, mapping[item.Parent.String()].String())
		b = append(b, '}')
	}
	return b
}
func appendUnobservedReceipt(b []byte, values []migration.UnobservedParent, mapping map[string]domain.ObjectID) []byte {
	for i, item := range values {
		if i > 0 {
			b = append(b, ',')
		}
		b = append(b, `{"old_child":`...)
		b, _ = canonical.AppendString(b, item.Child.String())
		b = append(b, `,"old_parent":`...)
		b, _ = canonical.AppendString(b, item.Parent.String())
		b = append(b, `,"new_child":`...)
		b, _ = canonical.AppendString(b, mapping[item.Child.String()].String())
		b = append(b, `,"new_parent":`...)
		b, _ = canonical.AppendString(b, mapping[item.Parent.String()].String())
		b = append(b, '}')
	}
	return b
}
func appendCollapsedReceipt(b []byte, values []migration.ParentAssertion, mapping map[string]domain.ObjectID) []byte {
	for i, item := range values {
		if i > 0 {
			b = append(b, ',')
		}
		b = appendOldTriple(b, item)
		b = append(b[:len(b)-1], `,"new_observer":`...)
		b, _ = canonical.AppendString(b, mapping[item.Observer.String()].String())
		b = append(b, `,"new_collapsed_seal":`...)
		b, _ = canonical.AppendString(b, mapping[item.Child.String()].String())
		b = append(b, '}')
	}
	return b
}
func appendMergedReceipt(b []byte, values []migration.MergedCauseLink, mapping map[string]domain.ObjectID) []byte {
	for i, item := range values {
		if i > 0 {
			b = append(b, ',')
		}
		b = append(b, `{"old_observer":`...)
		b, _ = canonical.AppendString(b, item.Observer.String())
		b = append(b, `,"old_targets":[`...)
		b = appendIDs(b, item.OldTargets)
		b = append(b, `],"new_observer":`...)
		b, _ = canonical.AppendString(b, mapping[item.Observer.String()].String())
		b = append(b, `,"new_target":`...)
		b, _ = canonical.AppendString(b, item.NewTarget.String())
		b = append(b, '}')
	}
	return b
}
func appendOldTriple(b []byte, item migration.ParentAssertion) []byte {
	b = append(b, `{"old_observer":`...)
	b, _ = canonical.AppendString(b, item.Observer.String())
	b = append(b, `,"old_child":`...)
	b, _ = canonical.AppendString(b, item.Child.String())
	b = append(b, `,"old_parent":`...)
	b, _ = canonical.AppendString(b, item.Parent.String())
	return append(b, '}')
}
