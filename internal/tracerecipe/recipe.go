// Package tracerecipe parses the strict trace-recipe/v1 input document.
package tracerecipe

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"unicode/utf8"

	"github.com/mako10k/sealgraph/internal/domain"
)

const Schema = "sealgraph/trace-recipe/v1"

type Recipe struct {
	Schema  string
	Sources []Source
	Runs    []Run
}

type Source struct {
	Name       string
	SourceKey  string
	File       string
	SnapshotID domain.ObjectID
}

type Run struct {
	Kind        string
	Length      uint64
	Source      string
	SourceStart uint64
}

func Parse(data []byte) (Recipe, error) {
	if !utf8.Valid(data) {
		return Recipe{}, fmt.Errorf("recipe is not valid UTF-8")
	}
	root, err := object(data, "recipe")
	if err != nil {
		return Recipe{}, err
	}
	if err := exact(root, "schema", "sources", "runs"); err != nil {
		return Recipe{}, err
	}
	schema, err := requiredString(root, "schema")
	if err != nil {
		return Recipe{}, err
	}
	if schema != Schema {
		return Recipe{}, fmt.Errorf("recipe schema is %q; expected %q", schema, Schema)
	}
	sources, err := parseSources(root["sources"])
	if err != nil {
		return Recipe{}, err
	}
	runs, err := parseRuns(root["runs"])
	if err != nil {
		return Recipe{}, err
	}
	used := make(map[string]bool)
	for i, run := range runs {
		if run.Kind == "external" {
			if _, ok := findSource(sources, run.Source); !ok {
				return Recipe{}, fmt.Errorf("runs[%d].source %q does not name a source", i, run.Source)
			}
			used[run.Source] = true
		}
	}
	for _, source := range sources {
		if !used[source.Name] {
			return Recipe{}, fmt.Errorf("source %q is unused", source.Name)
		}
	}
	return Recipe{Schema: schema, Sources: sources, Runs: runs}, nil
}

func parseSources(raw []byte) ([]Source, error) {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 || trimmed[0] != '[' {
		return nil, fmt.Errorf("sources must be an array")
	}
	var values []json.RawMessage
	if err := json.Unmarshal(raw, &values); err != nil {
		return nil, fmt.Errorf("sources must be an array: %w", err)
	}
	sources := make([]Source, len(values))
	names := make(map[string]bool, len(values))
	for i, value := range values {
		obj, err := object(value, fmt.Sprintf("sources[%d]", i))
		if err != nil {
			return nil, err
		}
		if has(obj, "source_key") || has(obj, "file") {
			if err := exact(obj, "name", "source_key", "file"); err != nil {
				return nil, err
			}
			sources[i].SourceKey, err = requiredString(obj, "source_key")
			if err != nil {
				return nil, fmt.Errorf("sources[%d].source_key: %w", i, err)
			}
			if sources[i].SourceKey == "" {
				return nil, fmt.Errorf("sources[%d].source_key must be non-empty", i)
			}
			sources[i].File, err = requiredString(obj, "file")
			if err != nil {
				return nil, fmt.Errorf("sources[%d].file: %w", i, err)
			}
			if sources[i].File == "" {
				return nil, fmt.Errorf("sources[%d].file must be non-empty", i)
			}
		} else {
			if err := exact(obj, "name", "snapshot"); err != nil {
				return nil, err
			}
			text, e := requiredString(obj, "snapshot")
			if e != nil {
				return nil, fmt.Errorf("sources[%d].snapshot: %w", i, e)
			}
			sources[i].SnapshotID, e = domain.ParseObjectID(text)
			if e != nil {
				return nil, fmt.Errorf("sources[%d].snapshot: %w", i, e)
			}
		}
		sources[i].Name, err = requiredString(obj, "name")
		if err != nil {
			return nil, fmt.Errorf("sources[%d].name: %w", i, err)
		}
		if sources[i].Name == "" {
			return nil, fmt.Errorf("sources[%d].name must be non-empty", i)
		}
		if names[sources[i].Name] {
			return nil, fmt.Errorf("duplicate source name %q", sources[i].Name)
		}
		names[sources[i].Name] = true
	}
	return sources, nil
}

func parseRuns(raw []byte) ([]Run, error) {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 || trimmed[0] != '[' {
		return nil, fmt.Errorf("runs must be an array")
	}
	var values []json.RawMessage
	if err := json.Unmarshal(raw, &values); err != nil {
		return nil, fmt.Errorf("runs must be an array: %w", err)
	}
	runs := make([]Run, len(values))
	for i, value := range values {
		obj, err := object(value, fmt.Sprintf("runs[%d]", i))
		if err != nil {
			return nil, err
		}
		kind, err := requiredString(obj, "kind")
		if err != nil {
			return nil, fmt.Errorf("runs[%d].kind: %w", i, err)
		}
		runs[i].Kind = kind
		length, err := uintField(obj, "length")
		if err != nil {
			return nil, fmt.Errorf("runs[%d].length: %w", i, err)
		}
		if length == 0 {
			return nil, fmt.Errorf("runs[%d].length must be positive", i)
		}
		runs[i].Length = length
		switch kind {
		case "external":
			if err := exact(obj, "kind", "length", "source", "source_start"); err != nil {
				return nil, err
			}
			runs[i].Source, err = requiredString(obj, "source")
			if err != nil {
				return nil, fmt.Errorf("runs[%d].source: %w", i, err)
			}
			runs[i].SourceStart, err = uintField(obj, "source_start")
			if err != nil {
				return nil, fmt.Errorf("runs[%d].source_start: %w", i, err)
			}
		case "untraced":
			if err := exact(obj, "kind", "length"); err != nil {
				return nil, err
			}
		default:
			return nil, fmt.Errorf("runs[%d].kind %q is invalid", i, kind)
		}
	}
	return runs, nil
}

func object(data []byte, label string) (map[string]json.RawMessage, error) {
	dec := json.NewDecoder(bytes.NewReader(data))
	first, err := dec.Token()
	if err != nil {
		return nil, fmt.Errorf("%s: %w", label, err)
	}
	if delimiter, ok := first.(json.Delim); !ok || delimiter != '{' {
		return nil, fmt.Errorf("%s: must be an object", label)
	}
	result := make(map[string]json.RawMessage)
	for dec.More() {
		token, err := dec.Token()
		if err != nil {
			return nil, fmt.Errorf("%s: %w", label, err)
		}
		key, ok := token.(string)
		if !ok {
			return nil, fmt.Errorf("%s: object member name must be a string", label)
		}
		if _, exists := result[key]; exists {
			return nil, fmt.Errorf("%s: duplicate member %q", label, key)
		}
		var value json.RawMessage
		if err := dec.Decode(&value); err != nil {
			return nil, fmt.Errorf("%s member %q: %w", label, key, err)
		}
		result[key] = value
	}
	closing, err := dec.Token()
	if err != nil {
		return nil, fmt.Errorf("%s: %w", label, err)
	}
	if delimiter, ok := closing.(json.Delim); !ok || delimiter != '}' {
		return nil, fmt.Errorf("%s: object is not closed", label)
	}
	var extra any
	if err := dec.Decode(&extra); err != io.EOF {
		if err == nil {
			err = fmt.Errorf("trailing JSON value")
		}
		return nil, fmt.Errorf("%s: %w", label, err)
	}
	return result, nil
}

func exact(obj map[string]json.RawMessage, allowed ...string) error {
	set := make(map[string]bool, len(allowed))
	for _, key := range allowed {
		set[key] = true
	}
	for key := range obj {
		if !set[key] {
			return fmt.Errorf("unknown member %q", key)
		}
	}
	for _, key := range allowed {
		if _, ok := obj[key]; !ok {
			return fmt.Errorf("missing member %q", key)
		}
	}
	return nil
}

func has(obj map[string]json.RawMessage, key string) bool { _, ok := obj[key]; return ok }

func requiredString(obj map[string]json.RawMessage, key string) (string, error) {
	raw, ok := obj[key]
	if !ok {
		return "", fmt.Errorf("missing member %q", key)
	}
	var value string
	if err := json.Unmarshal(raw, &value); err != nil {
		return "", fmt.Errorf("must be a string")
	}
	return value, nil
}

func uintField(obj map[string]json.RawMessage, key string) (uint64, error) {
	raw, ok := obj[key]
	if !ok {
		return 0, fmt.Errorf("missing member %q", key)
	}
	text := string(raw)
	if text == "" || text[0] == '-' {
		return 0, fmt.Errorf("must be a non-negative integer")
	}
	value, err := strconv.ParseUint(text, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("must be a uint64 integer")
	}
	return value, nil
}

func findSource(sources []Source, name string) (Source, bool) {
	for _, source := range sources {
		if source.Name == name {
			return source, true
		}
	}
	return Source{}, false
}
