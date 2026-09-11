// Package refmanifest implements the pure canonical REF manifest byte contract.
package refmanifest

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"sort"

	"github.com/mako10k/sealgraph/internal/canonical"
	"github.com/mako10k/sealgraph/internal/domain"
	"github.com/mako10k/sealgraph/internal/store"
)

const Schema = "sealgraph/ref/v1"

type Manifest struct {
	Schema string
	Head   domain.ObjectID
	Tags   []store.Tag
}

type refManifestWire struct {
	Schema string `json:"schema"`
	Head   string `json:"head"`
	Tags   []struct {
		Name   string `json:"name"`
		Target string `json:"target"`
	} `json:"tags"`
}

func Encode(input Manifest) ([]byte, error) {
	manifest, err := Normalize(input)
	if err != nil {
		return nil, err
	}
	b := make([]byte, 0, 128+len(manifest.Tags)*96)
	b = append(b, `{"schema":`...)
	b, _ = canonical.AppendString(b, Schema)
	b = append(b, `,"head":`...)
	b, _ = canonical.AppendString(b, manifest.Head.String())
	b = append(b, `,"tags":[`...)
	for i, tag := range manifest.Tags {
		if i > 0 {
			b = append(b, ',')
		}
		b = append(b, `{"name":`...)
		b, _ = canonical.AppendString(b, tag.Name)
		b = append(b, `,"target":`...)
		b, _ = canonical.AppendString(b, tag.Seal.String())
		b = append(b, '}')
	}
	b = append(b, ']', '}')
	return b, nil
}

func Decode(data []byte) (Manifest, error) {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var wire refManifestWire
	if err := decoder.Decode(&wire); err != nil {
		return Manifest{}, err
	}
	if err := requireJSONEOF(decoder); err != nil {
		return Manifest{}, err
	}
	manifest, err := manifestFromWire(wire)
	if err != nil {
		return Manifest{}, err
	}
	canonicalBytes, err := Encode(manifest)
	if err != nil {
		return Manifest{}, err
	}
	if !bytes.Equal(data, canonicalBytes) {
		return Manifest{}, fmt.Errorf("manifest bytes are not canonical")
	}
	return manifest, nil
}

func requireJSONEOF(decoder *json.Decoder) error {
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			return fmt.Errorf("manifest has trailing JSON value")
		}
		return fmt.Errorf("manifest has trailing data: %w", err)
	}
	return nil
}

func manifestFromWire(wire refManifestWire) (Manifest, error) {
	if wire.Schema != Schema {
		return Manifest{}, fmt.Errorf("schema is %q, expected %q", wire.Schema, Schema)
	}
	head, err := domain.ParseObjectID(wire.Head)
	if err != nil {
		return Manifest{}, fmt.Errorf("invalid head: %w", err)
	}
	tags := make([]store.Tag, len(wire.Tags))
	for i, item := range wire.Tags {
		target, err := domain.ParseObjectID(item.Target)
		if err != nil {
			return Manifest{}, fmt.Errorf("tag %q target is invalid: %w", item.Name, err)
		}
		tags[i] = store.Tag{Name: item.Name, Seal: target}
	}
	return Normalize(Manifest{Schema: wire.Schema, Head: head, Tags: tags})
}

func Normalize(manifest Manifest) (Manifest, error) {
	if manifest.Schema != Schema {
		return Manifest{}, fmt.Errorf("schema is %q, expected %q", manifest.Schema, Schema)
	}
	if err := manifest.Head.ValidateNative(); err != nil {
		return Manifest{}, fmt.Errorf("invalid head: %w", err)
	}
	tags := append([]store.Tag(nil), manifest.Tags...)
	for _, tag := range tags {
		if err := domain.ValidateTagName(tag.Name); err != nil {
			return Manifest{}, err
		}
		if err := tag.Seal.ValidateNative(); err != nil {
			return Manifest{}, fmt.Errorf("tag %q target is invalid: %w", tag.Name, err)
		}
	}
	sort.Slice(tags, func(i, j int) bool { return tags[i].Name < tags[j].Name })
	for i := 1; i < len(tags); i++ {
		if tags[i-1].Name == tags[i].Name {
			return Manifest{}, fmt.Errorf("duplicate tag name %q", tags[i].Name)
		}
	}
	return Manifest{Schema: Schema, Head: manifest.Head, Tags: tags}, nil
}
