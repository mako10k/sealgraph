package migration

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sort"

	"github.com/mako10k/sealgraph/internal/canonical"
	"github.com/mako10k/sealgraph/internal/domain"
)

const UniversalBlobV1Schema = "sealgraph/universal-blob-migration/v1"

var UniversalExcludedState = []string{"source_bindings", "cache", "event_logs", "recovery_journal", "locks", "temporary_files"}

type ObjectRecord struct {
	ID   domain.ObjectID
	Data []byte
}
type RefRecord struct {
	Name string
	Head domain.ObjectID
}
type TagRecord struct {
	REF    string
	Name   string
	Target domain.ObjectID
}

type UniversalSealRecord struct {
	ID           domain.ObjectID
	Payload      domain.SealPayload
	PayloadBytes []byte
}

type ParentAssertion struct{ Observer, Child, Parent domain.ObjectID }
type UnobservedParent struct{ Child, Parent domain.ObjectID }
type MergedCauseLink struct {
	Observer   domain.ObjectID
	OldTargets []domain.ObjectID
	NewTarget  domain.ObjectID
}
type UniversalSemanticProjection struct {
	Materialized []ParentAssertion
	Unobserved   []UnobservedParent
	Collapsed    []ParentAssertion
	Merged       []MergedCauseLink
}
type UniversalBlobV1 struct {
	Objects         []ObjectRecord
	Seals           []UniversalSealRecord
	REFs            []RefRecord
	Tags            []TagRecord
	Projection      UniversalSemanticProjection
	ExcludedObjects []domain.ObjectID
	ExcludedState   []string
}

// CanonicalizeUniversalBlobV1 orders every set-valued field according to the
// migration document contract. It does not invent or verify semantic
// projection records; callers must compute those from the canonical Seal
// order.
func CanonicalizeUniversalBlobV1(value *UniversalBlobV1) error {
	sort.Slice(value.Objects, func(i, j int) bool { return value.Objects[i].ID.String() < value.Objects[j].ID.String() })
	sealSet := make(map[string]UniversalSealRecord, len(value.Seals))
	for _, record := range value.Seals {
		if _, exists := sealSet[record.ID.String()]; exists {
			return fmt.Errorf("duplicate format-4 Seal %s", record.ID)
		}
		sealSet[record.ID.String()] = record
	}
	order, err := topologicalSealOrder(sealSet)
	if err != nil {
		return err
	}
	ordered := make([]UniversalSealRecord, 0, len(order))
	for _, id := range order {
		ordered = append(ordered, sealSet[id.String()])
	}
	value.Seals = ordered
	sort.Slice(value.REFs, func(i, j int) bool { return value.REFs[i].Name < value.REFs[j].Name })
	sort.Slice(value.Tags, func(i, j int) bool {
		if value.Tags[i].REF != value.Tags[j].REF {
			return value.Tags[i].REF < value.Tags[j].REF
		}
		return value.Tags[i].Name < value.Tags[j].Name
	})
	sort.Slice(value.ExcludedObjects, func(i, j int) bool { return value.ExcludedObjects[i].String() < value.ExcludedObjects[j].String() })
	normalizeMigrationSemantic(&value.Projection)
	return nil
}

type universalObjectWire struct {
	ID          string `json:"id"`
	BytesBase64 string `json:"bytes_base64"`
}
type universalSealWire struct {
	ID            string `json:"id"`
	PayloadBase64 string `json:"payload_base64"`
}
type universalREFWire struct {
	Name string `json:"name"`
	Head string `json:"head"`
}
type universalTagWire struct {
	REF    string `json:"ref"`
	Name   string `json:"name"`
	Target string `json:"target"`
}
type universalParentWire struct {
	Observer string `json:"observer"`
	Child    string `json:"child"`
	Parent   string `json:"parent"`
}
type universalUnobservedWire struct {
	Child  string `json:"child"`
	Parent string `json:"parent"`
}
type universalMergedWire struct {
	Observer   string   `json:"observer"`
	OldTargets []string `json:"old_targets"`
	NewTarget  string   `json:"new_target"`
}
type universalProjectionWire struct {
	Materialized []universalParentWire     `json:"materialized_parent_assertions"`
	Unobserved   []universalUnobservedWire `json:"unobserved_parent_assertions"`
	Collapsed    []universalParentWire     `json:"collapsed_revision_assertions"`
	Merged       []universalMergedWire     `json:"merged_cause_links"`
}

type universalWire struct {
	Schema           string `json:"schema"`
	SourceRepository struct {
		Format       int    `json:"format"`
		ObjectFormat string `json:"object_format"`
	} `json:"source_repository"`
	Objects            []universalObjectWire   `json:"objects"`
	Seals              []universalSealWire     `json:"seals"`
	REFs               []universalREFWire      `json:"refs"`
	Tags               []universalTagWire      `json:"tags"`
	SemanticProjection universalProjectionWire `json:"semantic_projection"`
	ExcludedObjects    []string                `json:"excluded_objects"`
	ExcludedState      []string                `json:"excluded_state"`
}

func DecodeUniversalBlobV1(data []byte) (UniversalBlobV1, error) {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var wire universalWire
	if err := decoder.Decode(&wire); err != nil {
		return UniversalBlobV1{}, fmt.Errorf("decode universal-blob-v1: %w", err)
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return UniversalBlobV1{}, fmt.Errorf("decode universal-blob-v1: trailing JSON value")
	}
	value, err := universalFromWire(wire)
	if err != nil {
		return UniversalBlobV1{}, err
	}
	encoded, err := EncodeUniversalBlobV1(value)
	if err != nil {
		return UniversalBlobV1{}, err
	}
	if !bytes.Equal(data, encoded) {
		return UniversalBlobV1{}, fmt.Errorf("universal-blob-v1 document is not canonical")
	}
	return value, nil
}

func universalFromWire(wire universalWire) (UniversalBlobV1, error) {
	if wire.Schema != UniversalBlobV1Schema || wire.SourceRepository.Format != 4 || wire.SourceRepository.ObjectFormat != "sha256" {
		return UniversalBlobV1{}, fmt.Errorf("unsupported universal migration source: schema=%q format=%d object_format=%q", wire.Schema, wire.SourceRepository.Format, wire.SourceRepository.ObjectFormat)
	}
	objects, err := universalObjectsFromWire(wire.Objects)
	if err != nil {
		return UniversalBlobV1{}, err
	}
	seals, err := universalSealsFromWire(wire.Seals)
	if err != nil {
		return UniversalBlobV1{}, err
	}
	refs, err := universalREFsFromWire(wire.REFs)
	if err != nil {
		return UniversalBlobV1{}, err
	}
	tags, err := universalTagsFromWire(wire.Tags)
	if err != nil {
		return UniversalBlobV1{}, err
	}
	projection, err := universalProjectionFromWire(wire.SemanticProjection)
	if err != nil {
		return UniversalBlobV1{}, err
	}
	excluded, err := parseIDs(wire.ExcludedObjects)
	if err != nil {
		return UniversalBlobV1{}, err
	}
	return UniversalBlobV1{Objects: objects, Seals: seals, REFs: refs, Tags: tags, Projection: projection, ExcludedObjects: excluded, ExcludedState: append([]string{}, wire.ExcludedState...)}, nil
}

func universalObjectsFromWire(items []universalObjectWire) ([]ObjectRecord, error) {
	result := make([]ObjectRecord, 0, len(items))
	for _, item := range items {
		id, err := domain.ParseObjectID(item.ID)
		if err != nil {
			return nil, err
		}
		data, err := decodeStrictBase64(item.BytesBase64)
		if err != nil {
			return nil, fmt.Errorf("object %s bytes_base64: %w", id, err)
		}
		result = append(result, ObjectRecord{ID: id, Data: data})
	}
	return result, nil
}

func universalSealsFromWire(items []universalSealWire) ([]UniversalSealRecord, error) {
	result := make([]UniversalSealRecord, 0, len(items))
	for _, item := range items {
		id, err := domain.ParseObjectID(item.ID)
		if err != nil {
			return nil, err
		}
		payloadBytes, err := decodeStrictBase64(item.PayloadBase64)
		if err != nil {
			return nil, fmt.Errorf("seal %s payload_base64: %w", id, err)
		}
		payload, err := canonical.DecodeSeal(payloadBytes)
		if err != nil {
			return nil, fmt.Errorf("seal %s format-4 payload: %w", id, err)
		}
		result = append(result, UniversalSealRecord{ID: id, Payload: payload, PayloadBytes: payloadBytes})
	}
	return result, nil
}

func universalREFsFromWire(items []universalREFWire) ([]RefRecord, error) {
	result := make([]RefRecord, 0, len(items))
	for _, item := range items {
		head, err := domain.ParseObjectID(item.Head)
		if err != nil {
			return nil, err
		}
		result = append(result, RefRecord{Name: item.Name, Head: head})
	}
	return result, nil
}

func universalTagsFromWire(items []universalTagWire) ([]TagRecord, error) {
	result := make([]TagRecord, 0, len(items))
	for _, item := range items {
		target, err := domain.ParseObjectID(item.Target)
		if err != nil {
			return nil, err
		}
		result = append(result, TagRecord{REF: item.REF, Name: item.Name, Target: target})
	}
	return result, nil
}

func universalProjectionFromWire(wire universalProjectionWire) (UniversalSemanticProjection, error) {
	result := UniversalSemanticProjection{}
	for _, item := range wire.Materialized {
		record, err := parseParentAssertion(item.Observer, item.Child, item.Parent)
		if err != nil {
			return UniversalSemanticProjection{}, err
		}
		result.Materialized = append(result.Materialized, record)
	}
	for _, item := range wire.Unobserved {
		child, err := domain.ParseObjectID(item.Child)
		if err != nil {
			return UniversalSemanticProjection{}, err
		}
		parent, err := domain.ParseObjectID(item.Parent)
		if err != nil {
			return UniversalSemanticProjection{}, err
		}
		result.Unobserved = append(result.Unobserved, UnobservedParent{Child: child, Parent: parent})
	}
	for _, item := range wire.Collapsed {
		record, err := parseParentAssertion(item.Observer, item.Child, item.Parent)
		if err != nil {
			return UniversalSemanticProjection{}, err
		}
		result.Collapsed = append(result.Collapsed, record)
	}
	for _, item := range wire.Merged {
		observer, err := domain.ParseObjectID(item.Observer)
		if err != nil {
			return UniversalSemanticProjection{}, err
		}
		newTarget, err := domain.ParseObjectID(item.NewTarget)
		if err != nil {
			return UniversalSemanticProjection{}, err
		}
		oldTargets, err := parseIDs(item.OldTargets)
		if err != nil {
			return UniversalSemanticProjection{}, err
		}
		result.Merged = append(result.Merged, MergedCauseLink{Observer: observer, OldTargets: oldTargets, NewTarget: newTarget})
	}
	return result, nil
}

func parseParentAssertion(observerText, childText, parentText string) (ParentAssertion, error) {
	observer, err := domain.ParseObjectID(observerText)
	if err != nil {
		return ParentAssertion{}, err
	}
	child, err := domain.ParseObjectID(childText)
	if err != nil {
		return ParentAssertion{}, err
	}
	parent, err := domain.ParseObjectID(parentText)
	if err != nil {
		return ParentAssertion{}, err
	}
	return ParentAssertion{Observer: observer, Child: child, Parent: parent}, nil
}

func parseIDs(values []string) ([]domain.ObjectID, error) {
	ids := make([]domain.ObjectID, 0, len(values))
	for _, value := range values {
		id, err := domain.ParseObjectID(value)
		if err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, nil
}

func decodeStrictBase64(text string) ([]byte, error) {
	data, err := base64.StdEncoding.Strict().DecodeString(text)
	if err != nil {
		return nil, err
	}
	if base64.StdEncoding.EncodeToString(data) != text {
		return nil, fmt.Errorf("base64 is not canonical padded RFC 4648")
	}
	return data, nil
}

func validateUniversalBlob(value UniversalBlobV1) error {
	if !equalStrings(value.ExcludedState, UniversalExcludedState) {
		return fmt.Errorf("excluded_state is not the required universal-blob-v1 constant")
	}
	sealSet, err := validateUniversalSeals(value.Seals)
	if err != nil {
		return err
	}
	objects, err := validateUniversalObjects(value.Objects)
	if err != nil {
		return err
	}
	if err := validateUniversalReferencedObjects(value.Seals, objects); err != nil {
		return err
	}
	if err := validateRefsTags(value.REFs, value.Tags, sealSet); err != nil {
		return err
	}
	if err := validateProjectionOrder(value.Projection); err != nil {
		return err
	}
	return validateUniversalExcludedObjects(value.ExcludedObjects, objects, sealSet)
}

func validateUniversalSeals(seals []UniversalSealRecord) (map[string]UniversalSealRecord, error) {
	sealSet := make(map[string]UniversalSealRecord, len(seals))
	for _, record := range seals {
		if !domain.ComputeNativeBlobID(record.PayloadBytes).Equal(record.ID) {
			return nil, fmt.Errorf("format-4 Seal %s payload identity mismatch", record.ID)
		}
		if _, exists := sealSet[record.ID.String()]; exists {
			return nil, fmt.Errorf("duplicate format-4 Seal %s", record.ID)
		}
		sealSet[record.ID.String()] = record
	}
	expectedOrder, err := topologicalSealOrder(sealSet)
	if err != nil {
		return nil, err
	}
	for i, id := range expectedOrder {
		if !seals[i].ID.Equal(id) {
			return nil, fmt.Errorf("format-4 Seal array is not canonical dependency-first order at index %d", i)
		}
	}
	return sealSet, nil
}

func validateUniversalObjects(records []ObjectRecord) (map[string]bool, error) {
	objects := make(map[string]bool, len(records))
	lastObject := ""
	for _, object := range records {
		if object.ID.String() <= lastObject || !domain.ComputeNativeBlobID(object.Data).Equal(object.ID) {
			return nil, fmt.Errorf("objects are not sorted unique valid Blob identities at %s", object.ID)
		}
		lastObject = object.ID.String()
		objects[object.ID.String()] = true
	}
	return objects, nil
}

func validateUniversalReferencedObjects(seals []UniversalSealRecord, objects map[string]bool) error {
	referenced := make(map[string]bool)
	for _, seal := range seals {
		referenced[seal.Payload.Content.ID.String()] = true
		for _, attachment := range seal.Payload.Attachments {
			referenced[attachment.Blob.ID.String()] = true
		}
	}
	if len(objects) != len(referenced) {
		return fmt.Errorf("objects inventory has %d entries; exact referenced inventory has %d", len(objects), len(referenced))
	}
	for id := range referenced {
		if !objects[id] {
			return fmt.Errorf("referenced format-4 Blob %s is absent from objects", id)
		}
	}
	return nil
}

func validateUniversalExcludedObjects(excluded []domain.ObjectID, objects map[string]bool, seals map[string]UniversalSealRecord) error {
	lastExcluded := ""
	for _, id := range excluded {
		if id.String() <= lastExcluded || objects[id.String()] || seals[id.String()].ID.String() != "" {
			return fmt.Errorf("excluded_objects is not sorted, unique, and disjoint at %s", id)
		}
		lastExcluded = id.String()
	}
	return nil
}

func topologicalSealOrder(seals map[string]UniversalSealRecord) ([]domain.ObjectID, error) {
	remaining := make(map[string]int, len(seals))
	dependents := make(map[string][]string)
	for id, seal := range seals {
		dependencies := []domain.ObjectID{}
		if seal.Payload.ParentRevision != nil {
			dependencies = append(dependencies, *seal.Payload.ParentRevision)
		}
		for _, link := range seal.Payload.Links {
			dependencies = append(dependencies, link.TargetSeal)
		}
		seen := make(map[string]bool)
		for _, dependency := range dependencies {
			if _, ok := seals[dependency.String()]; !ok {
				return nil, fmt.Errorf("format-4 Seal %s references absent Seal %s", id, dependency)
			}
			if !seen[dependency.String()] {
				remaining[id]++
				dependents[dependency.String()] = append(dependents[dependency.String()], id)
				seen[dependency.String()] = true
			}
		}
	}
	ready := []string{}
	for id, count := range remaining {
		if count == 0 {
			ready = append(ready, id)
		}
	}
	for id := range seals {
		if _, ok := remaining[id]; !ok {
			ready = append(ready, id)
		}
	}
	order := make([]domain.ObjectID, 0, len(seals))
	for len(ready) > 0 {
		sort.Strings(ready)
		id := ready[0]
		ready = ready[1:]
		order = append(order, domain.ObjectID{Hex: id})
		for _, dependent := range dependents[id] {
			remaining[dependent]--
			if remaining[dependent] == 0 {
				ready = append(ready, dependent)
			}
		}
	}
	if len(order) != len(seals) {
		return nil, fmt.Errorf("format-4 migration Seal graph contains a cycle")
	}
	return order, nil
}

func validateRefsTags(refs []RefRecord, tags []TagRecord, seals map[string]UniversalSealRecord) error {
	refSet := make(map[string]bool)
	last := ""
	for _, ref := range refs {
		if err := domain.ValidateREF(ref.Name); err != nil {
			return err
		}
		if ref.Name <= last || seals[ref.Head.String()].ID.String() == "" {
			return fmt.Errorf("REF inventory is not sorted unique or targets an absent Seal at %s", ref.Name)
		}
		last, refSet[ref.Name] = ref.Name, true
	}
	last = ""
	for _, tag := range tags {
		key := tag.REF + "\x00" + tag.Name
		if !refSet[tag.REF] || key <= last || seals[tag.Target.String()].ID.String() == "" {
			return fmt.Errorf("tag inventory is not sorted unique or targets an absent scope/Seal at %s@%s", tag.REF, tag.Name)
		}
		if err := domain.ValidateTagName(tag.Name); err != nil {
			return err
		}
		last = key
	}
	return nil
}

func validateProjectionOrder(value UniversalSemanticProjection) error {
	last := ""
	for _, record := range value.Materialized {
		key := parentKey(record)
		if key <= last {
			return fmt.Errorf("materialized_parent_assertions is not sorted unique")
		}
		last = key
	}
	last = ""
	for _, record := range value.Unobserved {
		key := record.Child.String() + "\x00" + record.Parent.String()
		if key <= last {
			return fmt.Errorf("unobserved_parent_assertions is not sorted unique")
		}
		last = key
	}
	last = ""
	for _, record := range value.Collapsed {
		key := parentKey(record)
		if key <= last {
			return fmt.Errorf("collapsed_revision_assertions is not sorted unique")
		}
		last = key
	}
	last = ""
	for _, record := range value.Merged {
		if len(record.OldTargets) < 2 || !idsStrictlySorted(record.OldTargets) {
			return fmt.Errorf("merged Cause Link old_targets must be sorted unique with at least two IDs")
		}
		key := record.Observer.String() + "\x00" + record.NewTarget.String() + "\x00" + idsKey(record.OldTargets)
		if key <= last {
			return fmt.Errorf("merged_cause_links is not sorted unique")
		}
		last = key
	}
	return nil
}

func parentKey(value ParentAssertion) string {
	return value.Observer.String() + "\x00" + value.Child.String() + "\x00" + value.Parent.String()
}
func idsKey(values []domain.ObjectID) string {
	result := ""
	for _, id := range values {
		result += id.String() + "\x00"
	}
	return result
}
func idsStrictlySorted(values []domain.ObjectID) bool {
	for i := 1; i < len(values); i++ {
		if values[i-1].String() >= values[i].String() {
			return false
		}
	}
	return true
}
func equalStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}

func EncodeUniversalBlobV1(value UniversalBlobV1) ([]byte, error) {
	if err := validateUniversalBlob(value); err != nil {
		return nil, fmt.Errorf("encode universal-blob-v1: %w", err)
	}
	b := []byte(`{"schema":`)
	b, _ = canonical.AppendString(b, UniversalBlobV1Schema)
	b = append(b, `,"source_repository":{"format":4,"object_format":"sha256"},"objects":[`...)
	for i, object := range value.Objects {
		if i > 0 {
			b = append(b, ',')
		}
		b = append(b, `{"id":`...)
		b, _ = canonical.AppendString(b, object.ID.String())
		b = append(b, `,"bytes_base64":`...)
		b, _ = canonical.AppendString(b, base64.StdEncoding.EncodeToString(object.Data))
		b = append(b, '}')
	}
	b = append(b, `],"seals":[`...)
	for i, seal := range value.Seals {
		if i > 0 {
			b = append(b, ',')
		}
		b = append(b, `{"id":`...)
		b, _ = canonical.AppendString(b, seal.ID.String())
		b = append(b, `,"payload_base64":`...)
		b, _ = canonical.AppendString(b, base64.StdEncoding.EncodeToString(seal.PayloadBytes))
		b = append(b, '}')
	}
	b = append(b, `],"refs":[`...)
	b = appendUniversalREFRecords(b, value.REFs)
	b = append(b, `],"tags":[`...)
	b = appendUniversalTagRecords(b, value.Tags)
	b = append(b, `],"semantic_projection":{"materialized_parent_assertions":[`...)
	b = appendParentAssertions(b, value.Projection.Materialized)
	b = append(b, `],"unobserved_parent_assertions":[`...)
	for i, item := range value.Projection.Unobserved {
		if i > 0 {
			b = append(b, ',')
		}
		b = append(b, `{"child":`...)
		b, _ = canonical.AppendString(b, item.Child.String())
		b = append(b, `,"parent":`...)
		b, _ = canonical.AppendString(b, item.Parent.String())
		b = append(b, '}')
	}
	b = append(b, `],"collapsed_revision_assertions":[`...)
	b = appendParentAssertions(b, value.Projection.Collapsed)
	b = append(b, `],"merged_cause_links":[`...)
	for i, item := range value.Projection.Merged {
		if i > 0 {
			b = append(b, ',')
		}
		b = append(b, `{"observer":`...)
		b, _ = canonical.AppendString(b, item.Observer.String())
		b = append(b, `,"old_targets":[`...)
		for j, id := range item.OldTargets {
			if j > 0 {
				b = append(b, ',')
			}
			b, _ = canonical.AppendString(b, id.String())
		}
		b = append(b, `],"new_target":`...)
		b, _ = canonical.AppendString(b, item.NewTarget.String())
		b = append(b, '}')
	}
	b = append(b, `]},"excluded_objects":[`...)
	for i, id := range value.ExcludedObjects {
		if i > 0 {
			b = append(b, ',')
		}
		b, _ = canonical.AppendString(b, id.String())
	}
	b = append(b, `],"excluded_state":[`...)
	for i, item := range value.ExcludedState {
		if i > 0 {
			b = append(b, ',')
		}
		b, _ = canonical.AppendString(b, item)
	}
	b = append(b, ']', '}', '\n')
	return b, nil
}

func appendUniversalREFRecords(dst []byte, refs []RefRecord) []byte {
	for index, ref := range refs {
		dst = appendRecordSeparator(dst, index)
		dst = append(dst, `{"name":`...)
		dst, _ = canonical.AppendString(dst, ref.Name)
		dst = append(dst, `,"head":`...)
		dst, _ = canonical.AppendString(dst, ref.Head.String())
		dst = append(dst, '}')
	}
	return dst
}

func appendUniversalTagRecords(dst []byte, tags []TagRecord) []byte {
	for index, tag := range tags {
		dst = appendRecordSeparator(dst, index)
		dst = append(dst, `{"ref":`...)
		dst, _ = canonical.AppendString(dst, tag.REF)
		dst = append(dst, `,"name":`...)
		dst, _ = canonical.AppendString(dst, tag.Name)
		dst = append(dst, `,"target":`...)
		dst, _ = canonical.AppendString(dst, tag.Target.String())
		dst = append(dst, '}')
	}
	return dst
}

func appendRecordSeparator(dst []byte, index int) []byte {
	if index > 0 {
		return append(dst, ',')
	}
	return dst
}

func appendParentAssertions(b []byte, values []ParentAssertion) []byte {
	for i, item := range values {
		if i > 0 {
			b = append(b, ',')
		}
		b = append(b, `{"observer":`...)
		b, _ = canonical.AppendString(b, item.Observer.String())
		b = append(b, `,"child":`...)
		b, _ = canonical.AppendString(b, item.Child.String())
		b = append(b, `,"parent":`...)
		b, _ = canonical.AppendString(b, item.Parent.String())
		b = append(b, '}')
	}
	return b
}
