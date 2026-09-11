package v6

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sort"
	"strconv"
	"unicode/utf8"

	"github.com/mako10k/sealgraph/internal/canonical"
	domainv5 "github.com/mako10k/sealgraph/internal/domain/v5"
)

const (
	maxMetadataEntries = 64
	maxNamespaceBytes  = 255
	maxSchemaBytes     = 1024
	maxValueDepth      = 16
	maxValueNodes      = 4096
	maxStringBytes     = 4096
	maxMetadataBytes   = 65536
)

type metadataBudget struct{ nodes int }
type objectMember struct {
	key   string
	value []byte
}

func normalizeMetadata(input []domainv5.MetadataEntry) ([]domainv5.MetadataEntry, []byte, error) {
	if len(input) > maxMetadataEntries {
		return nil, nil, fmt.Errorf("Cause Link has %d metadata entries; maximum is %d", len(input), maxMetadataEntries)
	}
	entries := make([]domainv5.MetadataEntry, len(input))
	budget := &metadataBudget{}
	rawBytes := 0
	for i, entry := range input {
		rawBytes += len(entry.Value)
		if rawBytes > maxMetadataBytes {
			return nil, nil, fmt.Errorf("metadata value input exceeds the maximum canonical array size %d before parsing", maxMetadataBytes)
		}
		if !utf8.ValidString(entry.Namespace) || len([]byte(entry.Namespace)) < 1 || len([]byte(entry.Namespace)) > maxNamespaceBytes {
			return nil, nil, fmt.Errorf("metadata namespace must be valid UTF-8 from 1 through %d bytes", maxNamespaceBytes)
		}
		if entry.Schema != nil && (!utf8.ValidString(*entry.Schema) || len([]byte(*entry.Schema)) < 1 || len([]byte(*entry.Schema)) > maxSchemaBytes) {
			return nil, nil, fmt.Errorf("metadata schema must be null or valid UTF-8 from 1 through %d bytes", maxSchemaBytes)
		}
		value, err := canonicalMetadataValue(entry.Value, budget)
		if err != nil {
			return nil, nil, fmt.Errorf("metadata namespace %q value: %w", entry.Namespace, err)
		}
		entries[i] = domainv5.MetadataEntry{Namespace: entry.Namespace, Schema: cloneString(entry.Schema), Value: json.RawMessage(value)}
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Namespace < entries[j].Namespace })
	for i := 1; i < len(entries); i++ {
		if entries[i-1].Namespace == entries[i].Namespace {
			return nil, nil, fmt.Errorf("duplicate metadata namespace %q", entries[i].Namespace)
		}
	}
	encoded, err := appendMetadata(nil, entries)
	if err != nil {
		return nil, nil, err
	}
	if len(encoded) > maxMetadataBytes {
		return nil, nil, fmt.Errorf("canonical metadata array has %d bytes; maximum is %d", len(encoded), maxMetadataBytes)
	}
	return entries, encoded, nil
}

// NormalizeMetadata validates and canonicalizes one complete metadata array.
// It is the shared writer boundary for repository-level Candidate mutations.
func NormalizeMetadata(input []domainv5.MetadataEntry) ([]domainv5.MetadataEntry, error) {
	entries, _, err := normalizeMetadata(input)
	return entries, err
}

func cloneString(value *string) *string {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}

func canonicalMetadataValue(raw []byte, budget *metadataBudget) ([]byte, error) {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	value, err := parseMetadataValue(decoder, 1, budget)
	if err != nil {
		return nil, err
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return nil, fmt.Errorf("trailing JSON value")
	}
	return value, nil
}

func parseMetadataValue(decoder *json.Decoder, depth int, budget *metadataBudget) ([]byte, error) {
	if depth > maxValueDepth {
		return nil, fmt.Errorf("metadata value exceeds maximum depth %d", maxValueDepth)
	}
	budget.nodes++
	if budget.nodes > maxValueNodes {
		return nil, fmt.Errorf("metadata values exceed maximum %d nodes per Cause Link", maxValueNodes)
	}
	token, err := decoder.Token()
	if err != nil {
		return nil, fmt.Errorf("decode canonical JSON value: %w", err)
	}
	switch value := token.(type) {
	case nil:
		return []byte("null"), nil
	case bool:
		if value {
			return []byte("true"), nil
		}
		return []byte("false"), nil
	case string:
		if !utf8.ValidString(value) || len([]byte(value)) > maxStringBytes {
			return nil, fmt.Errorf("metadata string must be valid UTF-8 and at most %d bytes", maxStringBytes)
		}
		return canonical.AppendString(nil, value)
	case json.Number:
		text := value.String()
		if !canonicalInteger(text) {
			return nil, fmt.Errorf("metadata number %q is not a shortest signed 64-bit integer", text)
		}
		if _, err := strconv.ParseInt(text, 10, 64); err != nil {
			return nil, fmt.Errorf("metadata integer %q is outside signed 64-bit range", text)
		}
		return []byte(text), nil
	case json.Delim:
		switch value {
		case '[':
			return parseMetadataArray(decoder, depth, budget)
		case '{':
			return parseMetadataObject(decoder, depth, budget)
		}
	}
	return nil, fmt.Errorf("unsupported metadata JSON token %v", token)
}

func canonicalInteger(text string) bool {
	if text == "0" {
		return true
	}
	if text == "" || text == "-0" {
		return false
	}
	start := 0
	if text[0] == '-' {
		start = 1
	}
	if start == len(text) || text[start] == '0' {
		return false
	}
	for _, character := range text[start:] {
		if character < '0' || character > '9' {
			return false
		}
	}
	return true
}

func parseMetadataArray(decoder *json.Decoder, depth int, budget *metadataBudget) ([]byte, error) {
	result := []byte{'['}
	first := true
	for decoder.More() {
		value, err := parseMetadataValue(decoder, depth+1, budget)
		if err != nil {
			return nil, err
		}
		if !first {
			result = append(result, ',')
		}
		first = false
		result = append(result, value...)
	}
	if token, err := decoder.Token(); err != nil || token != json.Delim(']') {
		return nil, fmt.Errorf("metadata array is not closed")
	}
	return append(result, ']'), nil
}

func parseMetadataObject(decoder *json.Decoder, depth int, budget *metadataBudget) ([]byte, error) {
	members := []objectMember{}
	seen := map[string]bool{}
	for decoder.More() {
		token, err := decoder.Token()
		if err != nil {
			return nil, err
		}
		key, ok := token.(string)
		if !ok || !utf8.ValidString(key) || len([]byte(key)) > maxStringBytes {
			return nil, fmt.Errorf("metadata object key must be valid UTF-8 and at most %d bytes", maxStringBytes)
		}
		if seen[key] {
			return nil, fmt.Errorf("duplicate metadata object key %q", key)
		}
		seen[key] = true
		value, err := parseMetadataValue(decoder, depth+1, budget)
		if err != nil {
			return nil, err
		}
		members = append(members, objectMember{key: key, value: value})
	}
	if token, err := decoder.Token(); err != nil || token != json.Delim('}') {
		return nil, fmt.Errorf("metadata object is not closed")
	}
	sort.Slice(members, func(i, j int) bool { return members[i].key < members[j].key })
	result := []byte{'{'}
	for i, member := range members {
		if i > 0 {
			result = append(result, ',')
		}
		result, _ = canonical.AppendString(result, member.key)
		result = append(result, ':')
		result = append(result, member.value...)
	}
	return append(result, '}'), nil
}

func appendMetadata(result []byte, entries []domainv5.MetadataEntry) ([]byte, error) {
	result = append(result, '[')
	for i, entry := range entries {
		if i > 0 {
			result = append(result, ',')
		}
		result = append(result, `{"namespace":`...)
		result, _ = canonical.AppendString(result, entry.Namespace)
		result = append(result, `,"schema":`...)
		if entry.Schema == nil {
			result = append(result, "null"...)
		} else {
			result, _ = canonical.AppendString(result, *entry.Schema)
		}
		result = append(result, `,"value":`...)
		result = append(result, entry.Value...)
		result = append(result, '}')
	}
	return append(result, ']'), nil
}
