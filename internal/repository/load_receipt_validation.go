package repository

import (
	"context"
	"crypto/sha256"
	"fmt"
	"reflect"
	"sort"

	"github.com/mako10k/sealgraph/internal/domain"
	"github.com/mako10k/sealgraph/internal/migration"
)

func validateReceiptAgainstLoadInputs(input []byte, dump migration.UniversalBlobV1, projection migrationProjection, repositoryDigest string, receipt universalLoadReceiptValue) error {
	expected := expectedUniversalLoadReceiptValue(input, dump, projection, repositoryDigest)
	if !reflect.DeepEqual(receipt, expected) {
		return fmt.Errorf("receipt does not exactly match the validated migration document, semantic projection, and repository digest")
	}
	return nil
}

func expectedUniversalLoadReceiptValue(input []byte, dump migration.UniversalBlobV1, projection migrationProjection, repositoryDigest string) universalLoadReceiptValue {
	sourceDigest := sha256.Sum256(input)
	result := universalLoadReceiptValue{
		Schema: universalLoadReceiptV1Schema, SourceDigest: fmt.Sprintf("%x", sourceDigest),
		ExcludedState: append([]string{}, dump.ExcludedState...), PublishedFormat: 5, RepositoryDigest: repositoryDigest,
		SealMappings: []receiptSealMapping{}, CollapseGroups: []receiptCollapseGroup{}, TypedObjects: []receiptTypedObject{},
	}
	seals := append([]projectedSeal{}, projection.Seals...)
	sort.Slice(seals, func(i, j int) bool { return seals[i].OldID.String() < seals[j].OldID.String() })
	for _, seal := range seals {
		result.SealMappings = append(result.SealMappings, receiptSealMapping{
			OldSeal: seal.OldID.String(), NewSeal: seal.NewID.String(), Material: seal.MaterialID.String(), Provenance: seal.ProvenanceID.String(),
		})
	}
	for _, group := range migrationCollapseGroups(projection.Seals) {
		result.CollapseGroups = append(result.CollapseGroups, receiptCollapseGroup{NewSeal: group.New.String(), OldSeals: receiptIDStrings(group.Old)})
	}
	for _, object := range migrationTypedObjects(projection.TypedRoles) {
		result.TypedObjects = append(result.TypedObjects, receiptTypedObject{Kind: object.Kind, ID: object.ID.String()})
	}
	unchanged := make([]domain.ObjectID, 0, len(dump.Objects))
	for _, object := range dump.Objects {
		unchanged = append(unchanged, object.ID)
	}
	sortMigrationIDs(unchanged)
	result.UnchangedObjects = receiptIDStrings(unchanged)
	result.REFs = expectedReceiptREFs(dump.REFs, projection.OldToNew)
	result.Tags = expectedReceiptTags(dump.Tags, projection.OldToNew)
	result.SemanticProjection = expectedReceiptSemantic(projection.Semantic, projection.OldToNew)
	result.ExcludedObjects = receiptIDStrings(dump.ExcludedObjects)
	return result
}

func expectedReceiptREFs(refs []migration.RefRecord, mapping map[string]domain.ObjectID) []receiptREFMapping {
	result := make([]receiptREFMapping, 0, len(refs))
	for _, ref := range refs {
		result = append(result, receiptREFMapping{Name: ref.Name, OldHead: ref.Head.String(), NewHead: mapping[ref.Head.String()].String()})
	}
	return result
}

func expectedReceiptTags(tags []migration.TagRecord, mapping map[string]domain.ObjectID) []receiptTagMapping {
	result := make([]receiptTagMapping, 0, len(tags))
	for _, tag := range tags {
		result = append(result, receiptTagMapping{REF: tag.REF, Name: tag.Name, OldTarget: tag.Target.String(), NewTarget: mapping[tag.Target.String()].String()})
	}
	return result
}

func expectedReceiptSemantic(value migration.UniversalSemanticProjection, mapping map[string]domain.ObjectID) receiptSemanticProjection {
	result := receiptSemanticProjection{
		Materialized: []receiptMaterialized{}, Unobserved: []receiptUnobserved{},
		Collapsed: []receiptCollapsed{}, Merged: []receiptMerged{},
	}
	for _, item := range value.Materialized {
		result.Materialized = append(result.Materialized, receiptMaterialized{
			OldObserver: item.Observer.String(), OldChild: item.Child.String(), OldParent: item.Parent.String(),
			NewObserver: mapping[item.Observer.String()].String(), NewTarget: mapping[item.Child.String()].String(), NewPrevious: mapping[item.Parent.String()].String(),
		})
	}
	for _, item := range value.Unobserved {
		result.Unobserved = append(result.Unobserved, receiptUnobserved{
			OldChild: item.Child.String(), OldParent: item.Parent.String(), NewChild: mapping[item.Child.String()].String(), NewParent: mapping[item.Parent.String()].String(),
		})
	}
	for _, item := range value.Collapsed {
		result.Collapsed = append(result.Collapsed, receiptCollapsed{
			OldObserver: item.Observer.String(), OldChild: item.Child.String(), OldParent: item.Parent.String(),
			NewObserver: mapping[item.Observer.String()].String(), NewCollapsedSeal: mapping[item.Child.String()].String(),
		})
	}
	for _, item := range value.Merged {
		result.Merged = append(result.Merged, receiptMerged{
			OldObserver: item.Observer.String(), OldTargets: receiptIDStrings(item.OldTargets),
			NewObserver: mapping[item.Observer.String()].String(), NewTarget: item.NewTarget.String(),
		})
	}
	return result
}

func receiptIDStrings(ids []domain.ObjectID) []string {
	result := make([]string, len(ids))
	for index, id := range ids {
		result[index] = id.String()
	}
	return result
}

type universalLoadReceiptValue struct {
	Schema             string                    `json:"schema"`
	SourceDigest       string                    `json:"source_document_sha256"`
	SealMappings       []receiptSealMapping      `json:"seal_mappings"`
	CollapseGroups     []receiptCollapseGroup    `json:"collapse_groups"`
	TypedObjects       []receiptTypedObject      `json:"typed_objects"`
	UnchangedObjects   []string                  `json:"unchanged_objects"`
	REFs               []receiptREFMapping       `json:"refs"`
	Tags               []receiptTagMapping       `json:"tags"`
	SemanticProjection receiptSemanticProjection `json:"semantic_projection"`
	ExcludedObjects    []string                  `json:"excluded_objects"`
	ExcludedState      []string                  `json:"excluded_state"`
	PublishedFormat    int                       `json:"published_format"`
	RepositoryDigest   string                    `json:"repository_digest"`
}

type receiptSealMapping struct {
	OldSeal    string `json:"old_seal"`
	NewSeal    string `json:"new_seal"`
	Material   string `json:"material"`
	Provenance string `json:"provenance"`
}

type receiptCollapseGroup struct {
	NewSeal  string   `json:"new_seal"`
	OldSeals []string `json:"old_seals"`
}
type receiptTypedObject struct {
	Kind string `json:"kind"`
	ID   string `json:"id"`
}
type receiptREFMapping struct {
	Name    string `json:"name"`
	OldHead string `json:"old_head"`
	NewHead string `json:"new_head"`
}
type receiptTagMapping struct {
	REF       string `json:"ref"`
	Name      string `json:"name"`
	OldTarget string `json:"old_target"`
	NewTarget string `json:"new_target"`
}
type receiptSemanticProjection struct {
	Materialized []receiptMaterialized `json:"materialized_parent_assertions"`
	Unobserved   []receiptUnobserved   `json:"unobserved_parent_assertions"`
	Collapsed    []receiptCollapsed    `json:"collapsed_revision_assertions"`
	Merged       []receiptMerged       `json:"merged_cause_links"`
}
type receiptMaterialized struct {
	OldObserver string `json:"old_observer"`
	OldChild    string `json:"old_child"`
	OldParent   string `json:"old_parent"`
	NewObserver string `json:"new_observer"`
	NewTarget   string `json:"new_target"`
	NewPrevious string `json:"new_previous"`
}
type receiptUnobserved struct {
	OldChild  string `json:"old_child"`
	OldParent string `json:"old_parent"`
	NewChild  string `json:"new_child"`
	NewParent string `json:"new_parent"`
}
type receiptCollapsed struct {
	OldObserver      string `json:"old_observer"`
	OldChild         string `json:"old_child"`
	OldParent        string `json:"old_parent"`
	NewObserver      string `json:"new_observer"`
	NewCollapsedSeal string `json:"new_collapsed_seal"`
}
type receiptMerged struct {
	OldObserver string   `json:"old_observer"`
	OldTargets  []string `json:"old_targets"`
	NewObserver string   `json:"new_observer"`
	NewTarget   string   `json:"new_target"`
}

func validateUniversalLoadReceiptValue(receipt universalLoadReceiptValue) error {
	if receipt.Schema != universalLoadReceiptV1Schema || receipt.PublishedFormat != 5 {
		return fmt.Errorf("unsupported receipt schema=%q published_format=%d", receipt.Schema, receipt.PublishedFormat)
	}
	if err := parseObservationDigest(receipt.SourceDigest); err != nil {
		return fmt.Errorf("invalid source_document_sha256: %w", err)
	}
	if err := parseObservationDigest(receipt.RepositoryDigest); err != nil {
		return fmt.Errorf("invalid repository_digest: %w", err)
	}
	mappings, err := validateReceiptMappings(receipt.SealMappings)
	if err != nil {
		return err
	}
	if err := validateReceiptCollapseGroups(receipt.CollapseGroups, mappings); err != nil {
		return err
	}
	if err := validateReceiptTypedObjects(receipt.TypedObjects, receipt.SealMappings); err != nil {
		return err
	}
	if err := validateSortedReceiptIDs("unchanged_objects", receipt.UnchangedObjects, 0); err != nil {
		return err
	}
	if err := validateReceiptREFsAndTags(receipt.REFs, receipt.Tags, mappings); err != nil {
		return err
	}
	if err := validateReceiptSemantic(receipt.SemanticProjection, mappings); err != nil {
		return err
	}
	if err := validateSortedReceiptIDs("excluded_objects", receipt.ExcludedObjects, 0); err != nil {
		return err
	}
	if !equalStrings(receipt.ExcludedState, migration.UniversalExcludedState) {
		return fmt.Errorf("excluded_state is not the required universal-blob-v1 constant")
	}
	return validateReceiptSourceInventories(receipt, mappings)
}

func validateReceiptMappings(values []receiptSealMapping) (map[string]receiptSealMapping, error) {
	result := make(map[string]receiptSealMapping, len(values))
	last := ""
	for _, value := range values {
		for name, id := range map[string]string{"old_seal": value.OldSeal, "new_seal": value.NewSeal, "material": value.Material, "provenance": value.Provenance} {
			if err := validateReceiptID(name, id); err != nil {
				return nil, err
			}
		}
		if value.OldSeal <= last {
			return nil, fmt.Errorf("seal_mappings is not sorted and unique by old_seal")
		}
		last = value.OldSeal
		result[value.OldSeal] = value
	}
	return result, nil
}

func validateReceiptCollapseGroups(values []receiptCollapseGroup, mappings map[string]receiptSealMapping) error {
	expected := make(map[string][]string)
	for old, mapping := range mappings {
		expected[mapping.NewSeal] = append(expected[mapping.NewSeal], old)
	}
	for key := range expected {
		sort.Strings(expected[key])
		if len(expected[key]) < 2 {
			delete(expected, key)
		}
	}
	last := ""
	for _, value := range values {
		if err := validateReceiptID("collapse_groups.new_seal", value.NewSeal); err != nil {
			return err
		}
		if value.NewSeal <= last {
			return fmt.Errorf("collapse_groups is not sorted and unique by new_seal")
		}
		last = value.NewSeal
		if err := validateSortedReceiptIDs("collapse_groups.old_seals", value.OldSeals, 2); err != nil {
			return err
		}
		if !equalStrings(value.OldSeals, expected[value.NewSeal]) {
			return fmt.Errorf("collapse group %s does not match seal_mappings", value.NewSeal)
		}
		delete(expected, value.NewSeal)
	}
	if len(expected) != 0 {
		return fmt.Errorf("collapse_groups omits a many-old-to-one-new mapping")
	}
	return nil
}

func validateReceiptTypedObjects(values []receiptTypedObject, mappings []receiptSealMapping) error {
	expected := make(map[string]bool)
	for _, mapping := range mappings {
		expected["material\x00"+mapping.Material] = true
		expected["provenance\x00"+mapping.Provenance] = true
		expected["seal\x00"+mapping.NewSeal] = true
	}
	last := ""
	for _, value := range values {
		if value.Kind != "material" && value.Kind != "provenance" && value.Kind != "seal" {
			return fmt.Errorf("typed_objects kind %q is invalid", value.Kind)
		}
		if err := validateReceiptID("typed_objects.id", value.ID); err != nil {
			return err
		}
		key := value.Kind + "\x00" + value.ID
		if key <= last {
			return fmt.Errorf("typed_objects is not sorted and unique by kind and id")
		}
		last = key
		delete(expected, key)
	}
	if len(expected) != 0 {
		return fmt.Errorf("typed_objects omits a role named by seal_mappings")
	}
	return nil
}

func validateReceiptREFsAndTags(refs []receiptREFMapping, tags []receiptTagMapping, mappings map[string]receiptSealMapping) error {
	refSet := make(map[string]bool, len(refs))
	last := ""
	for _, value := range refs {
		if err := domain.ValidateREF(value.Name); err != nil {
			return err
		}
		if value.Name <= last {
			return fmt.Errorf("receipt refs is not sorted and unique")
		}
		last, refSet[value.Name] = value.Name, true
		mapping, ok := mappings[value.OldHead]
		if err := validateReceiptID("refs.old_head", value.OldHead); err != nil {
			return err
		}
		if err := validateReceiptID("refs.new_head", value.NewHead); err != nil {
			return err
		}
		if !ok || mapping.NewSeal != value.NewHead {
			return fmt.Errorf("REF %s mapping does not match seal_mappings", value.Name)
		}
	}
	last = ""
	for _, value := range tags {
		if !refSet[value.REF] {
			return fmt.Errorf("receipt tag %s@%s has no REF mapping", value.REF, value.Name)
		}
		if err := domain.ValidateTagName(value.Name); err != nil {
			return err
		}
		key := value.REF + "\x00" + value.Name
		if key <= last {
			return fmt.Errorf("receipt tags is not sorted and unique")
		}
		last = key
		mapping, ok := mappings[value.OldTarget]
		if err := validateReceiptID("tags.old_target", value.OldTarget); err != nil {
			return err
		}
		if err := validateReceiptID("tags.new_target", value.NewTarget); err != nil {
			return err
		}
		if !ok || mapping.NewSeal != value.NewTarget {
			return fmt.Errorf("tag %s@%s mapping does not match seal_mappings", value.REF, value.Name)
		}
	}
	return nil
}

func validateReceiptSemantic(value receiptSemanticProjection, mappings map[string]receiptSealMapping) error {
	if err := validateMaterializedReceipt(value.Materialized, mappings); err != nil {
		return err
	}
	if err := validateUnobservedReceipt(value.Unobserved, mappings); err != nil {
		return err
	}
	if err := validateCollapsedReceipt(value.Collapsed, mappings); err != nil {
		return err
	}
	return validateMergedReceipt(value.Merged, mappings)
}

func validateMaterializedReceipt(values []receiptMaterialized, mappings map[string]receiptSealMapping) error {
	last := ""
	for _, value := range values {
		ids := []string{value.OldObserver, value.OldChild, value.OldParent, value.NewObserver, value.NewTarget, value.NewPrevious}
		if err := validateReceiptIDs("materialized_parent_assertions", ids); err != nil {
			return err
		}
		key := value.OldObserver + "\x00" + value.OldChild + "\x00" + value.OldParent
		if key <= last {
			return fmt.Errorf("materialized_parent_assertions is not sorted and unique")
		}
		last = key
		if !receiptMappingEquals(mappings, value.OldObserver, value.NewObserver) || !receiptMappingEquals(mappings, value.OldChild, value.NewTarget) || !receiptMappingEquals(mappings, value.OldParent, value.NewPrevious) {
			return fmt.Errorf("materialized parent assertion does not match seal_mappings")
		}
	}
	return nil
}

func validateUnobservedReceipt(values []receiptUnobserved, mappings map[string]receiptSealMapping) error {
	last := ""
	for _, value := range values {
		if err := validateReceiptIDs("unobserved_parent_assertions", []string{value.OldChild, value.OldParent, value.NewChild, value.NewParent}); err != nil {
			return err
		}
		key := value.OldChild + "\x00" + value.OldParent
		if key <= last {
			return fmt.Errorf("unobserved_parent_assertions is not sorted and unique")
		}
		last = key
		if !receiptMappingEquals(mappings, value.OldChild, value.NewChild) || !receiptMappingEquals(mappings, value.OldParent, value.NewParent) {
			return fmt.Errorf("unobserved parent assertion does not match seal_mappings")
		}
	}
	return nil
}

func validateCollapsedReceipt(values []receiptCollapsed, mappings map[string]receiptSealMapping) error {
	last := ""
	for _, value := range values {
		if err := validateReceiptIDs("collapsed_revision_assertions", []string{value.OldObserver, value.OldChild, value.OldParent, value.NewObserver, value.NewCollapsedSeal}); err != nil {
			return err
		}
		key := value.OldObserver + "\x00" + value.OldChild + "\x00" + value.OldParent
		if key <= last {
			return fmt.Errorf("collapsed_revision_assertions is not sorted and unique")
		}
		last = key
		if !receiptMappingEquals(mappings, value.OldObserver, value.NewObserver) || !receiptMappingEquals(mappings, value.OldChild, value.NewCollapsedSeal) || !receiptMappingEquals(mappings, value.OldParent, value.NewCollapsedSeal) {
			return fmt.Errorf("collapsed revision assertion does not match seal_mappings")
		}
	}
	return nil
}

func validateMergedReceipt(values []receiptMerged, mappings map[string]receiptSealMapping) error {
	last := ""
	for _, value := range values {
		if err := validateReceiptIDs("merged_cause_links", append([]string{value.OldObserver, value.NewObserver, value.NewTarget}, value.OldTargets...)); err != nil {
			return err
		}
		if err := validateSortedReceiptIDs("merged_cause_links.old_targets", value.OldTargets, 2); err != nil {
			return err
		}
		key := value.OldObserver + "\x00" + value.NewTarget + "\x00"
		for _, old := range value.OldTargets {
			key += old + "\x00"
			if !receiptMappingEquals(mappings, old, value.NewTarget) {
				return fmt.Errorf("merged Cause target does not match seal_mappings")
			}
		}
		if key <= last {
			return fmt.Errorf("merged_cause_links is not sorted and unique")
		}
		last = key
		if !receiptMappingEquals(mappings, value.OldObserver, value.NewObserver) {
			return fmt.Errorf("merged Cause observer does not match seal_mappings")
		}
	}
	return nil
}

func validateReceiptSourceInventories(receipt universalLoadReceiptValue, mappings map[string]receiptSealMapping) error {
	unchanged := make(map[string]bool, len(receipt.UnchangedObjects))
	for _, id := range receipt.UnchangedObjects {
		unchanged[id] = true
	}
	for _, id := range receipt.ExcludedObjects {
		if unchanged[id] {
			return fmt.Errorf("excluded_objects overlaps unchanged object %s", id)
		}
		if _, ok := mappings[id]; ok {
			return fmt.Errorf("excluded_objects overlaps old Seal %s", id)
		}
	}
	return nil
}

func validateReceiptAgainstRepository(ctx context.Context, repo *Repository, receipt universalLoadReceiptValue) error {
	for _, mapping := range receipt.SealMappings {
		id, _ := domain.ParseObjectID(mapping.NewSeal)
		resolved, err := repo.LoadSeal(ctx, id)
		if err != nil {
			return err
		}
		if resolved.Seal.Material.String() != mapping.Material || resolved.Seal.Provenance.String() != mapping.Provenance {
			return fmt.Errorf("new Seal %s does not join receipt Material/Provenance", mapping.NewSeal)
		}
	}
	if err := validateReceiptTypedRoleInventory(ctx, repo, receipt.TypedObjects); err != nil {
		return err
	}
	if err := validateReceiptObjectInventory(ctx, repo, receipt); err != nil {
		return err
	}
	return validateReceiptManifestInventory(ctx, repo, receipt)
}

func validateReceiptTypedRoleInventory(ctx context.Context, repo *Repository, receipt []receiptTypedObject) error {
	inventory, err := repo.fsckInventory(ctx)
	if err != nil {
		return err
	}
	actual := make(map[string]bool)
	for id := range inventory.materials {
		actual["material\x00"+id] = true
	}
	for id := range inventory.provenances {
		actual["provenance\x00"+id] = true
	}
	for id := range inventory.seals {
		actual["seal\x00"+id] = true
	}
	if len(actual) != len(receipt) {
		return fmt.Errorf("receipt has %d typed roles; repository has %d", len(receipt), len(actual))
	}
	for _, value := range receipt {
		if !actual[value.Kind+"\x00"+value.ID] {
			return fmt.Errorf("receipt typed role %s/%s is absent from repository", value.Kind, value.ID)
		}
	}
	return nil
}

func validateReceiptObjectInventory(ctx context.Context, repo *Repository, receipt universalLoadReceiptValue) error {
	expected := make(map[string]bool)
	for _, item := range receipt.TypedObjects {
		expected[item.ID] = true
	}
	for _, id := range receipt.UnchangedObjects {
		expected[id] = true
	}
	objects, err := repo.objects.List(ctx)
	if err != nil {
		return err
	}
	if len(objects) != len(expected) {
		return fmt.Errorf("receipt object inventory has %d IDs; repository has %d", len(expected), len(objects))
	}
	for _, object := range objects {
		if !expected[object.ID.String()] {
			return fmt.Errorf("repository Blob %s is absent from receipt inventories", object.ID)
		}
	}
	return nil
}

func validateReceiptManifestInventory(ctx context.Context, repo *Repository, receipt universalLoadReceiptValue) error {
	refs, err := repo.refs.List(ctx)
	if err != nil {
		return err
	}
	if len(refs) != len(receipt.REFs) {
		return fmt.Errorf("receipt has %d REFs; repository has %d", len(receipt.REFs), len(refs))
	}
	for index, ref := range refs {
		mapping := receipt.REFs[index]
		if ref != mapping.Name {
			return fmt.Errorf("receipt REF %q differs from repository REF %q", mapping.Name, ref)
		}
		head, err := repo.refs.Resolve(ctx, ref)
		if err != nil || head.String() != mapping.NewHead {
			return fmt.Errorf("receipt REF %s head differs from repository: head=%s err=%v", ref, head, err)
		}
	}
	actualTags := []repositoryTagDigest{}
	for _, ref := range refs {
		tags, err := repo.tags.List(ctx, ref)
		if err != nil {
			return err
		}
		for _, tag := range tags {
			actualTags = append(actualTags, repositoryTagDigest{REF: ref, Name: tag.Name, Target: tag.Seal})
		}
	}
	if len(actualTags) != len(receipt.Tags) {
		return fmt.Errorf("receipt has %d tags; repository has %d", len(receipt.Tags), len(actualTags))
	}
	for index, actual := range actualTags {
		expected := receipt.Tags[index]
		if actual.REF != expected.REF || actual.Name != expected.Name || actual.Target.String() != expected.NewTarget {
			return fmt.Errorf("receipt tag at index %d differs from repository", index)
		}
	}
	return nil
}

func validateSortedReceiptIDs(name string, values []string, minimum int) error {
	if len(values) < minimum {
		return fmt.Errorf("%s requires at least %d IDs", name, minimum)
	}
	last := ""
	for _, id := range values {
		if err := validateReceiptID(name, id); err != nil {
			return err
		}
		if id <= last {
			return fmt.Errorf("%s is not sorted and duplicate-free", name)
		}
		last = id
	}
	return nil
}

func validateReceiptIDs(name string, values []string) error {
	for _, id := range values {
		if err := validateReceiptID(name, id); err != nil {
			return err
		}
	}
	return nil
}

func validateReceiptID(name, value string) error {
	if _, err := domain.ParseObjectID(value); err != nil {
		return fmt.Errorf("%s has invalid ID %q: %w", name, value, err)
	}
	return nil
}

func receiptMappingEquals(mappings map[string]receiptSealMapping, old, new string) bool {
	mapping, ok := mappings[old]
	return ok && mapping.NewSeal == new
}
