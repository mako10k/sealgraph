// Package migration defines explicit, versioned repository conversion
// documents. It is not a compatibility reader for an older repository.
package migration

import (
	"encoding/base64"
	"fmt"

	"github.com/mako10k/sealgraph/internal/canonical"
	"github.com/mako10k/sealgraph/internal/domain"
)

const LogicalDumpV1Schema = "sealgraph/logical-dump/v1"

type ObjectRecord struct {
	ID   domain.ObjectID
	Data []byte
}

type SealRecord struct {
	OldSealID domain.ObjectID
	Payload   domain.SealPayload
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

type LogicalDumpV1 struct {
	Objects         []ObjectRecord
	Seals           []SealRecord
	REFs            []RefRecord
	Tags            []TagRecord
	ExcludedObjects []domain.ObjectID
}

// EncodeLogicalDumpV1 validates and emits the exact compact logical-v1 JSON
// envelope followed by one LF.
func EncodeLogicalDumpV1(dump LogicalDumpV1) ([]byte, error) {
	sealBytes, err := validateLogicalDumpV1(dump)
	if err != nil {
		return nil, err
	}

	b := make([]byte, 0, 1024)
	b = append(b, `{"schema":`...)
	b, _ = canonical.AppendString(b, LogicalDumpV1Schema)
	b = append(b, `,"source_repository":{"repository_format":3,"object_format":"sha256"},"objects":[`...)
	for i, object := range dump.Objects {
		if i > 0 {
			b = append(b, ',')
		}
		b = append(b, `{"id":`...)
		b, _ = canonical.AppendString(b, object.ID.String())
		b = append(b, `,"type":"blob","data_base64":`...)
		b, _ = canonical.AppendString(b, base64.StdEncoding.EncodeToString(object.Data))
		b = append(b, '}')
	}
	b = append(b, `],"seals":[`...)
	for i, seal := range dump.Seals {
		if i > 0 {
			b = append(b, ',')
		}
		b = append(b, `{"old_seal_id":`...)
		b, _ = canonical.AppendString(b, seal.OldSealID.String())
		b = append(b, `,"payload":`...)
		b = append(b, sealBytes[i]...)
		b = append(b, '}')
	}
	b = append(b, `],"refs":[`...)
	for i, ref := range dump.REFs {
		if i > 0 {
			b = append(b, ',')
		}
		b = append(b, `{"name":`...)
		b, _ = canonical.AppendString(b, ref.Name)
		b = append(b, `,"head":`...)
		b, _ = canonical.AppendString(b, ref.Head.String())
		b = append(b, '}')
	}
	b = append(b, `],"tags":[`...)
	for i, tag := range dump.Tags {
		if i > 0 {
			b = append(b, ',')
		}
		b = append(b, `{"ref":`...)
		b, _ = canonical.AppendString(b, tag.REF)
		b = append(b, `,"name":`...)
		b, _ = canonical.AppendString(b, tag.Name)
		b = append(b, `,"target":`...)
		b, _ = canonical.AppendString(b, tag.Target.String())
		b = append(b, '}')
	}
	b = append(b, `],"excluded_objects":[`...)
	for i, id := range dump.ExcludedObjects {
		if i > 0 {
			b = append(b, ',')
		}
		b, _ = canonical.AppendString(b, id.String())
	}
	b = append(b, ']', '}', '\n')
	return b, nil
}

func validateLogicalDumpV1(dump LogicalDumpV1) ([][]byte, error) {
	objectIDs := make(map[string]struct{}, len(dump.Objects))
	for i, object := range dump.Objects {
		if err := requireOrderedID(i, dump.Objects, object.ID); err != nil {
			return nil, fmt.Errorf("objects: %w", err)
		}
		if got := domain.ComputeNativeBlobID(object.Data); !got.Equal(object.ID) {
			return nil, fmt.Errorf("object %s payload hashes to %s", object.ID, got)
		}
		objectIDs[object.ID.String()] = struct{}{}
	}

	sealPositions := make(map[string]int, len(dump.Seals))
	sealPayloads := make(map[string]domain.SealPayload, len(dump.Seals))
	sealBytes := make([][]byte, len(dump.Seals))
	for i, seal := range dump.Seals {
		if err := seal.OldSealID.ValidateNative(); err != nil {
			return nil, fmt.Errorf("seal record %d old ID: %w", i, err)
		}
		if _, exists := sealPositions[seal.OldSealID.String()]; exists {
			return nil, fmt.Errorf("duplicate old SealID %s", seal.OldSealID)
		}
		encoded, err := canonical.EncodeSeal(seal.Payload)
		if err != nil {
			return nil, fmt.Errorf("encode old seal %s: %w", seal.OldSealID, err)
		}
		if got := domain.ComputeNativeBlobID(encoded); !got.Equal(seal.OldSealID) {
			return nil, fmt.Errorf("old seal %s canonical payload hashes to %s", seal.OldSealID, got)
		}
		sealPositions[seal.OldSealID.String()] = i
		sealPayloads[seal.OldSealID.String()] = seal.Payload
		sealBytes[i] = encoded
	}

	materialIDs := make(map[string]struct{})
	for i, seal := range dump.Seals {
		if seal.Payload.Parent != nil {
			if err := requireEarlierSeal(sealPositions, i, *seal.Payload.Parent, "parent"); err != nil {
				return nil, fmt.Errorf("seal %s: %w", seal.OldSealID, err)
			}
			if parent := sealPayloads[seal.Payload.Parent.String()]; parent.REF != seal.Payload.REF {
				return nil, fmt.Errorf("seal %s parent %s belongs to REF %s, not %s", seal.OldSealID, seal.Payload.Parent, parent.REF, seal.Payload.REF)
			}
		}
		for _, link := range seal.Payload.Links {
			if err := requireEarlierSeal(sealPositions, i, link.TargetSeal, "Cause target"); err != nil {
				return nil, fmt.Errorf("seal %s: %w", seal.OldSealID, err)
			}
			if target := sealPayloads[link.TargetSeal.String()]; target.REF != link.TargetREF {
				return nil, fmt.Errorf("seal %s Cause target %s belongs to REF %s, not %s", seal.OldSealID, link.TargetSeal, target.REF, link.TargetREF)
			}
		}
		materialIDs[seal.Payload.Content.ID.String()] = struct{}{}
		for _, attachment := range seal.Payload.Attachments {
			materialIDs[attachment.Blob.ID.String()] = struct{}{}
		}
	}
	for id := range materialIDs {
		if _, exists := objectIDs[id]; !exists {
			return nil, fmt.Errorf("referenced material object %s is absent from objects", id)
		}
	}
	for id := range objectIDs {
		if _, used := materialIDs[id]; !used {
			return nil, fmt.Errorf("object %s is not referenced material; list it as excluded instead", id)
		}
	}

	for i, ref := range dump.REFs {
		if err := domain.ValidateREF(ref.Name); err != nil {
			return nil, fmt.Errorf("REF record %d: %w", i, err)
		}
		if i > 0 && dump.REFs[i-1].Name >= ref.Name {
			return nil, fmt.Errorf("REF records are not in strict name order at %q", ref.Name)
		}
		head, exists := sealPayloads[ref.Head.String()]
		if !exists {
			return nil, fmt.Errorf("REF %s head %s is not an exported Seal", ref.Name, ref.Head)
		}
		if head.REF != ref.Name {
			return nil, fmt.Errorf("REF %s head %s belongs to REF %s", ref.Name, ref.Head, head.REF)
		}
	}
	for i, tag := range dump.Tags {
		if err := domain.ValidateREF(tag.REF); err != nil {
			return nil, fmt.Errorf("tag record %d REF: %w", i, err)
		}
		if err := domain.ValidateTagName(tag.Name); err != nil {
			return nil, fmt.Errorf("tag record %d name: %w", i, err)
		}
		if i > 0 {
			previous := dump.Tags[i-1]
			if previous.REF > tag.REF || previous.REF == tag.REF && previous.Name >= tag.Name {
				return nil, fmt.Errorf("tag records are not in strict (REF, name) order at %s@%s", tag.REF, tag.Name)
			}
		}
		target, exists := sealPayloads[tag.Target.String()]
		if !exists {
			return nil, fmt.Errorf("tag %s@%s target %s is not an exported Seal", tag.REF, tag.Name, tag.Target)
		}
		if target.REF != tag.REF {
			return nil, fmt.Errorf("tag %s@%s target %s belongs to REF %s", tag.REF, tag.Name, tag.Target, target.REF)
		}
	}

	for i, id := range dump.ExcludedObjects {
		if err := id.ValidateNative(); err != nil {
			return nil, fmt.Errorf("excluded object %d: %w", i, err)
		}
		if i > 0 && dump.ExcludedObjects[i-1].String() >= id.String() {
			return nil, fmt.Errorf("excluded objects are not in strict ID order at %s", id)
		}
		if _, exists := objectIDs[id.String()]; exists {
			return nil, fmt.Errorf("excluded object %s is also exported material", id)
		}
		if _, exists := sealPositions[id.String()]; exists {
			return nil, fmt.Errorf("excluded object %s is also an exported Seal", id)
		}
	}
	return sealBytes, nil
}

func requireOrderedID(i int, objects []ObjectRecord, id domain.ObjectID) error {
	if err := id.ValidateNative(); err != nil {
		return err
	}
	if i > 0 && objects[i-1].ID.String() >= id.String() {
		return fmt.Errorf("records are not in strict ID order at %s", id)
	}
	return nil
}

func requireEarlierSeal(positions map[string]int, current int, dependency domain.ObjectID, relation string) error {
	position, exists := positions[dependency.String()]
	if !exists {
		return fmt.Errorf("%s %s is absent from seals", relation, dependency)
	}
	if position >= current {
		return fmt.Errorf("%s %s does not precede its dependent", relation, dependency)
	}
	return nil
}
