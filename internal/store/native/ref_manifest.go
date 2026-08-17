package native

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

const refManifestSchema = "sealgraph/ref/v1"

type refManifest struct {
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

func encodeRefManifest(input refManifest) ([]byte, error) {
	manifest, err := normalizeRefManifest(input)
	if err != nil {
		return nil, err
	}
	b := make([]byte, 0, 128+len(manifest.Tags)*96)
	b = append(b, `{"schema":`...)
	b, _ = canonical.AppendString(b, refManifestSchema)
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

func decodeRefManifest(data []byte) (refManifest, error) {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var wire refManifestWire
	if err := decoder.Decode(&wire); err != nil {
		return refManifest{}, err
	}
	if err := requireJSONEOF(decoder); err != nil {
		return refManifest{}, err
	}
	manifest, err := manifestFromWire(wire)
	if err != nil {
		return refManifest{}, err
	}
	canonicalBytes, err := encodeRefManifest(manifest)
	if err != nil {
		return refManifest{}, err
	}
	if !bytes.Equal(data, canonicalBytes) {
		return refManifest{}, fmt.Errorf("manifest bytes are not canonical")
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

func manifestFromWire(wire refManifestWire) (refManifest, error) {
	if wire.Schema != refManifestSchema {
		return refManifest{}, fmt.Errorf("schema is %q, expected %q", wire.Schema, refManifestSchema)
	}
	head, err := domain.ParseObjectID(wire.Head)
	if err != nil {
		return refManifest{}, fmt.Errorf("invalid head: %w", err)
	}
	tags := make([]store.Tag, len(wire.Tags))
	for i, item := range wire.Tags {
		target, err := domain.ParseObjectID(item.Target)
		if err != nil {
			return refManifest{}, fmt.Errorf("tag %q target is invalid: %w", item.Name, err)
		}
		tags[i] = store.Tag{Name: item.Name, Seal: target}
	}
	return normalizeRefManifest(refManifest{Schema: wire.Schema, Head: head, Tags: tags})
}

func normalizeRefManifest(manifest refManifest) (refManifest, error) {
	if manifest.Schema != refManifestSchema {
		return refManifest{}, fmt.Errorf("schema is %q, expected %q", manifest.Schema, refManifestSchema)
	}
	if err := manifest.Head.ValidateNative(); err != nil {
		return refManifest{}, fmt.Errorf("invalid head: %w", err)
	}
	tags := append([]store.Tag(nil), manifest.Tags...)
	for _, tag := range tags {
		if err := domain.ValidateTagName(tag.Name); err != nil {
			return refManifest{}, err
		}
		if err := tag.Seal.ValidateNative(); err != nil {
			return refManifest{}, fmt.Errorf("tag %q target is invalid: %w", tag.Name, err)
		}
	}
	sort.Slice(tags, func(i, j int) bool { return tags[i].Name < tags[j].Name })
	for i := 1; i < len(tags); i++ {
		if tags[i-1].Name == tags[i].Name {
			return refManifest{}, fmt.Errorf("duplicate tag name %q", tags[i].Name)
		}
	}
	return refManifest{Schema: refManifestSchema, Head: manifest.Head, Tags: tags}, nil
}
