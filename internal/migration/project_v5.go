package migration

import (
	"fmt"
	"sort"

	canonicalv5 "github.com/mako10k/sealgraph/internal/canonical/v5"
	"github.com/mako10k/sealgraph/internal/domain"
	domainv5 "github.com/mako10k/sealgraph/internal/domain/v5"
)

func SemanticWarnings(value UniversalSemanticProjection) []string {
	warnings := []string{}
	if len(value.Unobserved) > 0 {
		warnings = append(warnings, fmt.Sprintf("SEMANTIC_CHANGE_UNOBSERVED_PARENT_DROPPED count=%d", len(value.Unobserved)))
	}
	if len(value.Collapsed) > 0 {
		warnings = append(warnings, fmt.Sprintf("SEMANTIC_CHANGE_COLLAPSED_REVISION_DROPPED count=%d", len(value.Collapsed)))
	}
	if len(value.Merged) > 0 {
		warnings = append(warnings, fmt.Sprintf("SEMANTIC_CHANGE_MERGED_CAUSE_LINKS count=%d", len(value.Merged)))
	}
	return warnings
}

type ProjectedSeal struct {
	OldID, NewID, MaterialID, ProvenanceID    domain.ObjectID
	MaterialBytes, ProvenanceBytes, SealBytes []byte
}

type Projection struct {
	Seals      []ProjectedSeal
	OldToNew   map[string]domain.ObjectID
	Semantic   UniversalSemanticProjection
	TypedRoles map[string]map[string]bool
}

func ProjectUniversalBlobV1(dump UniversalBlobV1) (Projection, error) {
	result, err := ComputeUniversalBlobV1Projection(dump)
	if err != nil {
		return Projection{}, err
	}
	if !equalMigrationSemantic(result.Semantic, dump.Projection) {
		return Projection{}, fmt.Errorf("MIGRATION_SEMANTIC_PROJECTION_MISMATCH: extractor projection does not match importer recomputation")
	}
	return result, nil
}

// ComputeUniversalBlobV1Projection derives the exact format-5 identities and
// semantic classifications from a validated, dependency-first format-4
// migration inventory. The extractor uses it before the document contains its
// projection; the importer calls ProjectUniversalBlobV1 to verify the embedded
// projection against an independent recomputation.
func ComputeUniversalBlobV1Projection(dump UniversalBlobV1) (Projection, error) {
	result := Projection{
		OldToNew: make(map[string]domain.ObjectID, len(dump.Seals)),
		TypedRoles: map[string]map[string]bool{
			"material": {}, "provenance": {}, "seal": {},
		},
	}
	oldSeals := make(map[string]UniversalSealRecord, len(dump.Seals))
	for _, object := range dump.Objects {
		if _, err := canonicalv5.DecodeMaterial(object.Data); err == nil {
			result.TypedRoles["material"][object.ID.String()] = true
		}
		if _, err := canonicalv5.DecodeProvenance(object.Data); err == nil {
			result.TypedRoles["provenance"][object.ID.String()] = true
		}
		if _, err := canonicalv5.DecodeSeal(object.Data); err == nil {
			result.TypedRoles["seal"][object.ID.String()] = true
		}
	}
	for _, record := range dump.Seals {
		oldSeals[record.ID.String()] = record
	}
	for _, record := range dump.Seals {
		projected, semantic, err := projectOneSeal(record, oldSeals, result.OldToNew)
		if err != nil {
			return Projection{}, err
		}
		result.Seals = append(result.Seals, projected)
		result.OldToNew[record.ID.String()] = projected.NewID
		result.Semantic.Materialized = append(result.Semantic.Materialized, semantic.Materialized...)
		result.Semantic.Collapsed = append(result.Semantic.Collapsed, semantic.Collapsed...)
		result.Semantic.Merged = append(result.Semantic.Merged, semantic.Merged...)
		result.TypedRoles["material"][projected.MaterialID.String()] = true
		result.TypedRoles["provenance"][projected.ProvenanceID.String()] = true
		result.TypedRoles["seal"][projected.NewID.String()] = true
	}
	result.Semantic.Unobserved = unobservedParents(dump.Seals)
	normalizeMigrationSemantic(&result.Semantic)
	if err := validateProjectedGraph(result.Seals); err != nil {
		return Projection{}, fmt.Errorf("FORMAT4_COLLAPSE_GRAPH_CYCLE_UNSUPPORTED: %w", err)
	}
	return result, nil
}

type oneSealSemantic struct {
	Materialized []ParentAssertion
	Collapsed    []ParentAssertion
	Merged       []MergedCauseLink
}

type causeGroup struct {
	Target     domain.ObjectID
	OldTargets []domain.ObjectID
	Previous   []domain.ObjectID
	Messages   []string
}

func projectOneSeal(record UniversalSealRecord, oldSeals map[string]UniversalSealRecord, oldToNew map[string]domain.ObjectID) (ProjectedSeal, oneSealSemantic, error) {
	material := domainv5.Material{Schema: domainv5.MaterialSchema, Content: record.Payload.Content.ID}
	for _, attachment := range record.Payload.Attachments {
		material.Attachments = append(material.Attachments, domainv5.Attachment{Name: attachment.Name, MediaType: attachment.MediaType, Blob: attachment.Blob.ID})
	}
	materialBytes, err := canonicalv5.EncodeMaterial(material)
	if err != nil {
		return ProjectedSeal{}, oneSealSemantic{}, fmt.Errorf("project format-4 Seal %s Material: %w", record.ID, err)
	}
	materialID := domain.ComputeNativeBlobID(materialBytes)
	links, semantic, err := projectCauseLinks(record, oldSeals, oldToNew)
	if err != nil {
		return ProjectedSeal{}, oneSealSemantic{}, err
	}
	provenanceBytes, err := canonicalv5.EncodeProvenance(domainv5.Provenance{Schema: domainv5.ProvenanceSchema, Root: record.Payload.Root, Draft: record.Payload.Draft, CauseLinks: links})
	if err != nil {
		return ProjectedSeal{}, oneSealSemantic{}, fmt.Errorf("project format-4 Seal %s Provenance: %w", record.ID, err)
	}
	provenanceID := domain.ComputeNativeBlobID(provenanceBytes)
	sealBytes, err := canonicalv5.EncodeSeal(domainv5.Seal{Schema: domainv5.SealSchema, Material: materialID, Provenance: provenanceID})
	if err != nil {
		return ProjectedSeal{}, oneSealSemantic{}, fmt.Errorf("project format-4 Seal %s: %w", record.ID, err)
	}
	return ProjectedSeal{OldID: record.ID, NewID: domain.ComputeNativeBlobID(sealBytes), MaterialID: materialID, ProvenanceID: provenanceID, MaterialBytes: materialBytes, ProvenanceBytes: provenanceBytes, SealBytes: sealBytes}, semantic, nil
}

func projectCauseLinks(observer UniversalSealRecord, oldSeals map[string]UniversalSealRecord, oldToNew map[string]domain.ObjectID) ([]domainv5.CauseLink, oneSealSemantic, error) {
	groups := make(map[string]*causeGroup)
	semantic := oneSealSemantic{}
	for _, oldLink := range observer.Payload.Links {
		target, ok := oldToNew[oldLink.TargetSeal.String()]
		if !ok {
			return nil, oneSealSemantic{}, fmt.Errorf("format-4 Seal %s Cause target %s has no earlier mapping", observer.ID, oldLink.TargetSeal)
		}
		group := groups[target.String()]
		if group == nil {
			group = &causeGroup{Target: target}
			groups[target.String()] = group
		}
		group.OldTargets = appendUniqueID(group.OldTargets, oldLink.TargetSeal)
		group.Messages = appendUniqueString(group.Messages, oldLink.Message)
		targetRecord := oldSeals[oldLink.TargetSeal.String()]
		if targetRecord.Payload.ParentRevision == nil {
			continue
		}
		parent := *targetRecord.Payload.ParentRevision
		mappedParent, ok := oldToNew[parent.String()]
		if !ok {
			return nil, oneSealSemantic{}, fmt.Errorf("format-4 target %s parent %s has no earlier mapping", oldLink.TargetSeal, parent)
		}
		classification := ParentAssertion{Observer: observer.ID, Child: oldLink.TargetSeal, Parent: parent}
		if mappedParent.Equal(target) {
			semantic.Collapsed = append(semantic.Collapsed, classification)
		} else {
			group.Previous = appendUniqueID(group.Previous, mappedParent)
			semantic.Materialized = append(semantic.Materialized, classification)
		}
	}
	keys := make([]string, 0, len(groups))
	for key := range groups {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	result := make([]domainv5.CauseLink, 0, len(keys))
	for _, key := range keys {
		group := groups[key]
		sortMigrationIDs(group.OldTargets)
		sortMigrationIDs(group.Previous)
		sort.Strings(group.Messages)
		result = append(result, domainv5.CauseLink{TargetSeal: group.Target, PreviousRevisionSealOfTargetSeal: group.Previous, Messages: group.Messages})
		if len(group.OldTargets) > 1 {
			semantic.Merged = append(semantic.Merged, MergedCauseLink{Observer: observer.ID, OldTargets: group.OldTargets, NewTarget: group.Target})
		}
	}
	return result, semantic, nil
}

func unobservedParents(seals []UniversalSealRecord) []UnobservedParent {
	observed := make(map[string]bool)
	for _, seal := range seals {
		for _, link := range seal.Payload.Links {
			observed[link.TargetSeal.String()] = true
		}
	}
	result := []UnobservedParent{}
	for _, seal := range seals {
		if seal.Payload.ParentRevision != nil && !observed[seal.ID.String()] {
			result = append(result, UnobservedParent{Child: seal.ID, Parent: *seal.Payload.ParentRevision})
		}
	}
	return result
}

func normalizeMigrationSemantic(value *UniversalSemanticProjection) {
	sort.Slice(value.Materialized, func(i, j int) bool {
		return parentAssertionKey(value.Materialized[i]) < parentAssertionKey(value.Materialized[j])
	})
	sort.Slice(value.Unobserved, func(i, j int) bool { return unobservedKey(value.Unobserved[i]) < unobservedKey(value.Unobserved[j]) })
	sort.Slice(value.Collapsed, func(i, j int) bool {
		return parentAssertionKey(value.Collapsed[i]) < parentAssertionKey(value.Collapsed[j])
	})
	sort.Slice(value.Merged, func(i, j int) bool { return mergedKey(value.Merged[i]) < mergedKey(value.Merged[j]) })
}

func equalMigrationSemantic(left, right UniversalSemanticProjection) bool {
	return keysEqual(parentAssertionKeys(left.Materialized), parentAssertionKeys(right.Materialized)) &&
		keysEqual(unobservedKeys(left.Unobserved), unobservedKeys(right.Unobserved)) &&
		keysEqual(parentAssertionKeys(left.Collapsed), parentAssertionKeys(right.Collapsed)) &&
		keysEqual(mergedKeys(left.Merged), mergedKeys(right.Merged))
}

func parentAssertionKey(value ParentAssertion) string {
	return value.Observer.String() + "\x00" + value.Child.String() + "\x00" + value.Parent.String()
}
func unobservedKey(value UnobservedParent) string {
	return value.Child.String() + "\x00" + value.Parent.String()
}
func mergedKey(value MergedCauseLink) string {
	result := value.Observer.String() + "\x00" + value.NewTarget.String()
	for _, id := range value.OldTargets {
		result += "\x00" + id.String()
	}
	return result
}
func parentAssertionKeys(values []ParentAssertion) []string {
	result := make([]string, len(values))
	for i, value := range values {
		result[i] = parentAssertionKey(value)
	}
	return result
}
func unobservedKeys(values []UnobservedParent) []string {
	result := make([]string, len(values))
	for i, value := range values {
		result[i] = unobservedKey(value)
	}
	return result
}
func mergedKeys(values []MergedCauseLink) []string {
	result := make([]string, len(values))
	for i, value := range values {
		result[i] = mergedKey(value)
	}
	return result
}
func keysEqual(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func appendUniqueString(values []string, value string) []string {
	for _, existing := range values {
		if existing == value {
			return values
		}
	}
	return append(values, value)
}

func appendUniqueID(values []domain.ObjectID, value domain.ObjectID) []domain.ObjectID {
	for _, existing := range values {
		if existing.Equal(value) {
			return values
		}
	}
	return append(values, value)
}

func sortMigrationIDs(values []domain.ObjectID) {
	sort.Slice(values, func(i, j int) bool { return values[i].String() < values[j].String() })
}

func validateProjectedGraph(seals []ProjectedSeal) error {
	dependencies := make(map[string][]domain.ObjectID)
	nodes := make(map[string]bool)
	for _, projected := range seals {
		nodes[projected.NewID.String()] = true
		provenance, err := canonicalv5.DecodeProvenance(projected.ProvenanceBytes)
		if err != nil {
			return err
		}
		for _, link := range provenance.CauseLinks {
			if link.TargetSeal.Equal(projected.NewID) {
				return fmt.Errorf("projected Cause self-edge at %s", projected.NewID)
			}
			dependencies[projected.NewID.String()] = appendUniqueID(dependencies[projected.NewID.String()], link.TargetSeal)
			for _, previous := range link.PreviousRevisionSealOfTargetSeal {
				if previous.Equal(link.TargetSeal) {
					return fmt.Errorf("projected revision self-edge at %s", link.TargetSeal)
				}
				dependencies[link.TargetSeal.String()] = appendUniqueID(dependencies[link.TargetSeal.String()], previous)
			}
		}
	}
	state := make(map[string]uint8)
	var visit func(string) error
	visit = func(id string) error {
		if state[id] == 1 {
			return fmt.Errorf("projected combined Cause/revision cycle reaches %s", id)
		}
		if state[id] == 2 {
			return nil
		}
		if !nodes[id] {
			return fmt.Errorf("projected graph references absent Seal %s", id)
		}
		state[id] = 1
		for _, next := range dependencies[id] {
			if err := visit(next.String()); err != nil {
				return err
			}
		}
		state[id] = 2
		return nil
	}
	ids := make([]string, 0, len(nodes))
	for id := range nodes {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		if err := visit(id); err != nil {
			return err
		}
	}
	return nil
}
