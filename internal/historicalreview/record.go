// Package historicalreview owns the portable executor claim used for explicit
// historical-Cause admission. R: validate and encode one version-bound review
// record without inferring a judgment, approval, or current graph fact.
package historicalreview

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/mako10k/sealgraph/internal/domain"
)

const (
	Schema         = "sealgraph/historical-review/v1"
	AttachmentName = "sealgraph-historical-review.json"
	MediaType      = "application/vnd.sealgraph.historical-review+json"
	MaxBytes       = 1 << 20
)

type Head struct {
	REF  string          `json:"ref"`
	Seal domain.ObjectID `json:"seal"`
}
type Entry struct {
	TargetSeal    domain.ObjectID   `json:"target_seal"`
	ComparedSeals []domain.ObjectID `json:"compared_seals"`
	Decision      string            `json:"decision"`
	Difference    string            `json:"difference"`
	Impact        string            `json:"impact"`
	Reason        string            `json:"reason"`
}
type Record struct {
	Schema      string          `json:"schema"`
	REF         string          `json:"ref"`
	SubjectSeal domain.ObjectID `json:"subject_seal"`
	OtherHeads  []Head          `json:"other_heads"`
	Entries     []Entry         `json:"entries"`
}

// Decode accepts human-formatted input but rejects ambiguous JSON and emits
// exactly one normalized representation for the contained claim.
func Decode(data []byte) (Record, []byte, error) {
	if len(data) > MaxBytes || !utf8.Valid(data) {
		return Record{}, nil, fmt.Errorf("review must be UTF-8 and at most %d bytes", MaxBytes)
	}
	d := json.NewDecoder(bytes.NewReader(data))
	if err := uniqueValue(d, 0); err != nil {
		return Record{}, nil, err
	}
	if _, err := d.Token(); err != io.EOF {
		return Record{}, nil, fmt.Errorf("trailing review JSON")
	}
	if err := exactRecordMembers(data); err != nil {
		return Record{}, nil, err
	}
	d = json.NewDecoder(bytes.NewReader(data))
	d.DisallowUnknownFields()
	var record Record
	if err := d.Decode(&record); err != nil {
		return Record{}, nil, fmt.Errorf("decode review: %w", err)
	}
	encoded, err := Encode(record)
	if err != nil {
		return Record{}, nil, err
	}
	// Return independent, normalized arrays as well as canonical bytes.
	if err := json.Unmarshal(encoded, &record); err != nil {
		return Record{}, nil, err
	}
	return record, encoded, nil
}

func uniqueValue(d *json.Decoder, depth int) error {
	if depth > 12 {
		return fmt.Errorf("review JSON nesting exceeds 12")
	}
	token, err := d.Token()
	if err != nil {
		return err
	}
	delim, ok := token.(json.Delim)
	if !ok {
		return nil
	}
	seen := map[string]bool{}
	for d.More() {
		if delim == '{' {
			key, err := d.Token()
			if err != nil {
				return err
			}
			name, ok := key.(string)
			if !ok || seen[name] {
				return fmt.Errorf("duplicate or invalid review key %q", name)
			}
			seen[name] = true
		}
		if err := uniqueValue(d, depth+1); err != nil {
			return err
		}
	}
	_, err = d.Token()
	return err
}

func Encode(record Record) ([]byte, error) {
	if record.Schema != Schema {
		return nil, fmt.Errorf("review schema must be %s", Schema)
	}
	if err := domain.ValidateREF(record.REF); err != nil {
		return nil, err
	}
	if err := record.SubjectSeal.ValidateNative(); err != nil {
		return nil, err
	}
	if record.OtherHeads == nil || record.Entries == nil {
		return nil, fmt.Errorf("review other_heads and entries must be arrays")
	}
	record.OtherHeads = append([]Head{}, record.OtherHeads...)
	record.Entries = append([]Entry{}, record.Entries...)
	sort.Slice(record.OtherHeads, func(i, j int) bool { return record.OtherHeads[i].REF < record.OtherHeads[j].REF })
	for i, head := range record.OtherHeads {
		if err := domain.ValidateREF(head.REF); err != nil {
			return nil, err
		}
		if err := head.Seal.ValidateNative(); err != nil {
			return nil, err
		}
		if head.REF == record.REF || (i > 0 && record.OtherHeads[i-1].REF == head.REF) {
			return nil, fmt.Errorf("duplicate or subject REF in other_heads")
		}
	}
	sort.Slice(record.Entries, func(i, j int) bool {
		return record.Entries[i].TargetSeal.String() < record.Entries[j].TargetSeal.String()
	})
	for i := range record.Entries {
		entry, err := normalizeEntry(record.Entries[i])
		if err != nil {
			return nil, err
		}
		record.Entries[i] = entry
		if i > 0 && record.Entries[i-1].TargetSeal.Equal(entry.TargetSeal) {
			return nil, fmt.Errorf("duplicate review target %s", entry.TargetSeal)
		}
	}
	data, err := json.Marshal(record)
	if err == nil && len(data) > MaxBytes {
		err = fmt.Errorf("canonical review exceeds %d bytes", MaxBytes)
	}
	return data, err
}

func normalizeEntry(entry Entry) (Entry, error) {
	if err := entry.TargetSeal.ValidateNative(); err != nil {
		return Entry{}, err
	}
	if entry.ComparedSeals == nil {
		return Entry{}, fmt.Errorf("compared_seals must be an array")
	}
	entry.ComparedSeals = append([]domain.ObjectID{}, entry.ComparedSeals...)
	sort.Slice(entry.ComparedSeals, func(i, j int) bool { return entry.ComparedSeals[i].String() < entry.ComparedSeals[j].String() })
	for i, id := range entry.ComparedSeals {
		if err := id.ValidateNative(); err != nil {
			return Entry{}, err
		}
		if i > 0 && entry.ComparedSeals[i-1].Equal(id) {
			return Entry{}, fmt.Errorf("duplicate compared Seal %s", id)
		}
	}
	if entry.Decision != "retain" && entry.Decision != "unreviewed" {
		return Entry{}, fmt.Errorf("review decision must be retain or unreviewed")
	}
	for _, text := range []string{entry.Difference, entry.Impact, entry.Reason} {
		if !utf8.ValidString(text) {
			return Entry{}, fmt.Errorf("review text must be UTF-8")
		}
		if entry.Decision == "retain" && strings.TrimSpace(text) == "" {
			return Entry{}, fmt.Errorf("retain requires difference, impact, and reason for %s", entry.TargetSeal)
		}
	}
	return entry, nil
}

// exactRecordMembers rejects case aliases and omitted members that encoding/json
// otherwise accepts for Go structs. Claims must have one unambiguous shape.
func exactRecordMembers(data []byte) error {
	object, err := exactObject(data, "schema", "ref", "subject_seal", "other_heads", "entries")
	if err != nil {
		return err
	}
	for _, group := range []struct {
		field string
		keys  []string
	}{
		{"other_heads", []string{"ref", "seal"}},
		{"entries", []string{"target_seal", "compared_seals", "decision", "difference", "impact", "reason"}},
	} {
		var elements []json.RawMessage
		if err := json.Unmarshal(object[group.field], &elements); err != nil {
			return err
		}
		if elements == nil {
			return fmt.Errorf("review %s must be an array", group.field)
		}
		for _, element := range elements {
			if _, err := exactObject(element, group.keys...); err != nil {
				return err
			}
		}
	}
	return nil
}

func exactObject(data []byte, keys ...string) (map[string]json.RawMessage, error) {
	var object map[string]json.RawMessage
	if err := json.Unmarshal(data, &object); err != nil {
		return nil, err
	}
	if len(object) != len(keys) {
		return nil, fmt.Errorf("unknown or missing review object members")
	}
	for _, key := range keys {
		value, ok := object[key]
		if !ok {
			return nil, fmt.Errorf("missing exact review member %s", key)
		}
		if bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
			return nil, fmt.Errorf("review member %s cannot be null", key)
		}
	}
	return object, nil
}
