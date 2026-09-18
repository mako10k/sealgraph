// Package v7 implements the strict canonical format-7 typed records.
package v7

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"unicode/utf8"

	"github.com/mako10k/sealgraph/internal/canonical"
	canonicalv6 "github.com/mako10k/sealgraph/internal/canonical/v6"
	"github.com/mako10k/sealgraph/internal/domain"
	domainv5 "github.com/mako10k/sealgraph/internal/domain/v5"
	domainv7 "github.com/mako10k/sealgraph/internal/domain/v7"
)

func EncodeSourceSnapshot(v domainv7.SourceSnapshot) ([]byte, error) {
	if v.Schema != domainv7.SourceSnapshotSchema {
		return nil, fmt.Errorf("source snapshot schema is %q; expected %q", v.Schema, domainv7.SourceSnapshotSchema)
	}
	if !utf8.ValidString(v.SourceKey) || v.SourceKey == "" {
		return nil, errors.New("source snapshot source_key must be non-empty valid UTF-8")
	}
	if err := v.Content.ValidateNative(); err != nil {
		return nil, fmt.Errorf("invalid source snapshot content: %w", err)
	}
	b := []byte(`{"schema":`)
	b, _ = canonical.AppendString(b, v.Schema)
	b = append(b, `,"source_key":`...)
	b, _ = canonical.AppendString(b, v.SourceKey)
	b = append(b, `,"content":`...)
	b, _ = canonical.AppendNativeObjectID(b, v.Content)
	return append(b, '}'), nil
}

func DecodeSourceSnapshot(data []byte) (domainv7.SourceSnapshot, error) {
	return decodeCanonical(data, "source snapshot", parseSourceSnapshot, EncodeSourceSnapshot)
}

func EncodeOriginMap(v domainv7.OriginMap) ([]byte, error) {
	if v.Schema != domainv7.OriginMapSchema {
		return nil, fmt.Errorf("origin map schema is %q; expected %q", v.Schema, domainv7.OriginMapSchema)
	}
	if err := v.Content.ValidateNative(); err != nil {
		return nil, fmt.Errorf("invalid origin map content: %w", err)
	}
	if err := validateRuns(v.Runs); err != nil {
		return nil, err
	}
	b := []byte(`{"schema":`)
	b, _ = canonical.AppendString(b, v.Schema)
	b = append(b, `,"content":`...)
	b, _ = canonical.AppendNativeObjectID(b, v.Content)
	b = append(b, `,"runs":[`...)
	for i, r := range v.Runs {
		if i > 0 {
			b = append(b, ',')
		}
		b = append(b, `{"kind":`...)
		b, _ = canonical.AppendString(b, r.Kind)
		b = append(b, `,"length":`...)
		b = strconv.AppendUint(b, r.Length, 10)
		if r.Kind == "external" {
			b = append(b, `,"snapshot":`...)
			b, _ = canonical.AppendNativeObjectID(b, r.Snapshot)
			b = append(b, `,"source_start":`...)
			b = strconv.AppendUint(b, r.SourceStart, 10)
		}
		b = append(b, '}')
	}
	b = append(b, "]}"...)
	return b, nil
}

func DecodeOriginMap(data []byte) (domainv7.OriginMap, error) {
	return decodeCanonical(data, "origin map", parseOriginMap, EncodeOriginMap)
}

func EncodeSeal(v domainv7.Seal) ([]byte, error) {
	if v.Schema != domainv7.SealSchema {
		return nil, fmt.Errorf("seal schema is %q; expected %q", v.Schema, domainv7.SealSchema)
	}
	if err := v.Material.ValidateNative(); err != nil {
		return nil, fmt.Errorf("invalid material ID: %w", err)
	}
	if err := v.Provenance.ValidateNative(); err != nil {
		return nil, fmt.Errorf("invalid provenance ID: %w", err)
	}
	b := []byte(`{"schema":`)
	b, _ = canonical.AppendString(b, v.Schema)
	b = append(b, `,"material":`...)
	b, _ = canonical.AppendNativeObjectID(b, v.Material)
	b = append(b, `,"provenance":`...)
	b, _ = canonical.AppendNativeObjectID(b, v.Provenance)
	return append(b, '}'), nil
}
func DecodeSeal(data []byte) (domainv7.Seal, error) {
	return decodeCanonical(data, "seal", parseSeal, EncodeSeal)
}

func EncodeMaterial(v domainv7.Material) ([]byte, error) { return encodeMaterial(v) }
func DecodeMaterial(data []byte) (domainv7.Material, error) {
	return decodeCanonical(data, "material", parseMaterial, EncodeMaterial)
}

func EncodeProvenance(v domainv7.Provenance) ([]byte, error) {
	n, err := normalizeProvenance(v)
	if err != nil {
		return nil, err
	}
	b := []byte(`{"schema":`)
	b, _ = canonical.AppendString(b, n.Schema)
	b = append(b, `,"root":`...)
	b = canonical.AppendBool(b, n.Root)
	b = append(b, `,"draft":`...)
	b = canonical.AppendBool(b, n.Draft)
	b = append(b, `,"cause_links":`...)
	b, err = appendCauseLinks(b, n.CauseLinks)
	if err != nil {
		return nil, err
	}
	b = append(b, `,"origin":`...)
	b = appendOptionalID(b, n.Origin)
	return append(b, '}'), nil
}
func DecodeProvenance(data []byte) (domainv7.Provenance, error) {
	return decodeCanonical(data, "provenance", parseProvenance, EncodeProvenance)
}

func EncodeCandidate(v domainv7.Candidate) ([]byte, error) {
	n, err := normalizeCandidate(v)
	if err != nil {
		return nil, err
	}
	b := []byte(`{"schema":`)
	b, _ = canonical.AppendString(b, n.Schema)
	b = append(b, `,"ref":`...)
	b, _ = canonical.AppendString(b, n.REF)
	b = append(b, `,"expected_ref_head":`...)
	b = appendOptionalID(b, n.ExpectedREFHead)
	b = append(b, `,"content":`...)
	b, _ = canonical.AppendNativeObjectID(b, n.Content)
	b = append(b, `,"attachments":`...)
	b, err = appendAttachments(b, n.Attachments)
	if err != nil {
		return nil, err
	}
	b = append(b, `,"root":`...)
	b = canonical.AppendBool(b, n.Root)
	b = append(b, `,"draft":`...)
	b = canonical.AppendBool(b, n.Draft)
	b = append(b, `,"cause_links":`...)
	b, err = appendCauseLinks(b, n.CauseLinks)
	if err != nil {
		return nil, err
	}
	b = append(b, `,"origin":`...)
	b = appendOptionalID(b, n.Origin)
	return append(b, '}'), nil
}
func DecodeCandidate(data []byte) (domainv7.Candidate, error) {
	return decodeCanonical(data, "candidate", parseCandidate, EncodeCandidate)
}

func encodeMaterial(v domainv7.Material) ([]byte, error) {
	n, err := domainv5.NormalizeMaterial(v)
	if err != nil {
		return nil, err
	}
	b := []byte(`{"schema":`)
	b, _ = canonical.AppendString(b, n.Schema)
	b = append(b, `,"content":`...)
	b, _ = canonical.AppendNativeObjectID(b, n.Content)
	b = append(b, `,"attachments":`...)
	b, err = appendAttachments(b, n.Attachments)
	if err != nil {
		return nil, err
	}
	return append(b, '}'), nil
}

func normalizeProvenance(v domainv7.Provenance) (domainv7.Provenance, error) {
	if v.Schema != domainv7.ProvenanceSchema {
		return domainv7.Provenance{}, fmt.Errorf("provenance schema is %q; expected %q", v.Schema, domainv7.ProvenanceSchema)
	}
	legacy := domainv5.Provenance{Schema: "sealgraph/provenance/v2", Root: v.Root, Draft: v.Draft, CauseLinks: v.CauseLinks}
	n, err := canonicalv6.NormalizeProvenance(legacy)
	if err != nil {
		return domainv7.Provenance{}, err
	}
	v.CauseLinks = n.CauseLinks
	if v.Origin != nil {
		if err := v.Origin.ValidateNative(); err != nil {
			return domainv7.Provenance{}, fmt.Errorf("invalid origin map ID: %w", err)
		}
	}
	return v, nil
}
func normalizeCandidate(v domainv7.Candidate) (domainv7.Candidate, error) {
	if v.Schema != domainv7.CandidateSchema {
		return domainv7.Candidate{}, fmt.Errorf("candidate schema is %q; expected %q", v.Schema, domainv7.CandidateSchema)
	}
	legacy := domainv5.Candidate{Schema: "sealgraph/candidate/v6", REF: v.REF, ExpectedREFHead: v.ExpectedREFHead, Content: v.Content, Attachments: v.Attachments, Root: v.Root, Draft: v.Draft, CauseLinks: v.CauseLinks}
	n, err := canonicalv6.NormalizeCandidate(legacy)
	if err != nil {
		return domainv7.Candidate{}, err
	}
	v.Attachments, v.CauseLinks = n.Attachments, n.CauseLinks
	if v.Origin != nil {
		if err := v.Origin.ValidateNative(); err != nil {
			return domainv7.Candidate{}, fmt.Errorf("invalid origin map ID: %w", err)
		}
	}
	return v, nil
}

func appendOptionalID(b []byte, id *domain.ObjectID) []byte {
	if id == nil {
		return append(b, "null"...)
	}
	x, _ := canonical.AppendNativeObjectID(b, *id)
	return x
}
func appendAttachments(b []byte, a []domainv5.Attachment) ([]byte, error) {
	b = append(b, '[')
	for i, x := range a {
		if i > 0 {
			b = append(b, ',')
		}
		b = append(b, `{"name":`...)
		var err error
		b, err = canonical.AppendString(b, x.Name)
		if err != nil {
			return nil, err
		}
		b = append(b, `,"media_type":`...)
		b, err = canonical.AppendString(b, x.MediaType)
		if err != nil {
			return nil, err
		}
		b = append(b, `,"blob":`...)
		b, err = canonical.AppendNativeObjectID(b, x.Blob)
		if err != nil {
			return nil, err
		}
		b = append(b, '}')
	}
	return append(b, ']'), nil
}
func appendCauseLinks(b []byte, links []domainv5.CauseLink) ([]byte, error) {
	return canonicalv6.AppendCauseLinks(b, links)
}

func validateRuns(runs []domainv7.OriginRun) error {
	total := uint64(0)
	for i, r := range runs {
		if r.Kind != "external" && r.Kind != "untraced" {
			return fmt.Errorf("origin run %d has invalid kind %q", i, r.Kind)
		}
		if r.Length == 0 {
			return fmt.Errorf("origin run %d length must be positive", i)
		}
		if ^uint64(0)-total < r.Length {
			return errors.New("origin run lengths overflow uint64")
		}
		total += r.Length
		if r.Kind == "external" {
			if ^uint64(0)-r.SourceStart < r.Length {
				return fmt.Errorf("origin run %d source range overflows uint64", i)
			}
			if err := r.Snapshot.ValidateNative(); err != nil {
				return fmt.Errorf("origin run %d snapshot: %w", i, err)
			}
			if i > 0 {
				p := runs[i-1]
				if p.Kind == "external" && p.Snapshot.Equal(r.Snapshot) && ^uint64(0)-p.SourceStart >= p.Length && p.SourceStart+p.Length == r.SourceStart {
					return errors.New("adjacent external runs must be merged")
				}
			}
		} else if r.Snapshot.Hex != "" || r.SourceStart != 0 {
			return fmt.Errorf("untraced run %d has external fields", i)
		}
		if i > 0 && runs[i-1].Kind == "untraced" && r.Kind == "untraced" {
			return errors.New("adjacent untraced runs must be merged")
		}
	}
	return nil
}

type fields map[string]json.RawMessage

func object(data []byte, allowed ...string) (fields, error) {
	set := map[string]bool{}
	for _, k := range allowed {
		set[k] = true
	}
	d := json.NewDecoder(bytes.NewReader(data))
	t, err := d.Token()
	if err != nil || t != json.Delim('{') {
		return nil, errors.New("expected JSON object")
	}
	out := fields{}
	for d.More() {
		tok, err := d.Token()
		if err != nil {
			return nil, err
		}
		k, ok := tok.(string)
		if !ok || !set[k] {
			return nil, fmt.Errorf("unknown object member %q", tok)
		}
		if _, ok := out[k]; ok {
			return nil, fmt.Errorf("duplicate object member %q", k)
		}
		var raw json.RawMessage
		if err := d.Decode(&raw); err != nil {
			return nil, err
		}
		out[k] = raw
	}
	if tok, err := d.Token(); err != nil || tok != json.Delim('}') {
		return nil, errors.New("object is not closed")
	}
	var extra any
	if err := d.Decode(&extra); !errors.Is(err, io.EOF) {
		return nil, errors.New("trailing JSON value")
	}
	return out, nil
}
func required(f fields, key string) (json.RawMessage, error) {
	v, ok := f[key]
	if !ok {
		return nil, fmt.Errorf("missing required member %q", key)
	}
	return v, nil
}
func str(raw json.RawMessage, key string) (string, error) {
	var v string
	if err := json.Unmarshal(raw, &v); err != nil || !utf8.ValidString(v) {
		return "", fmt.Errorf("member %q must be valid UTF-8 string", key)
	}
	return v, nil
}
func boolean(raw json.RawMessage, key string) (bool, error) {
	var v bool
	if err := json.Unmarshal(raw, &v); err != nil || string(raw) != "true" && string(raw) != "false" {
		return false, fmt.Errorf("member %q must be boolean", key)
	}
	return v, nil
}
func oid(raw json.RawMessage, key string) (domain.ObjectID, error) {
	var v domain.ObjectID
	if err := json.Unmarshal(raw, &v); err != nil {
		return domain.ObjectID{}, fmt.Errorf("member %q: %w", key, err)
	}
	return v, nil
}
func optOID(raw json.RawMessage, key string) (*domain.ObjectID, error) {
	if string(raw) == "null" {
		return nil, nil
	}
	v, err := oid(raw, key)
	return &v, err
}
func uint64raw(raw json.RawMessage, key string) (uint64, error) {
	s := string(raw)
	if s == "" || (len(s) > 1 && s[0] == '0') {
		return 0, fmt.Errorf("member %q must be shortest unsigned integer", key)
	}
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0, fmt.Errorf("member %q must be unsigned integer", key)
		}
	}
	v, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("member %q overflows uint64", key)
	}
	return v, nil
}

func parseSourceSnapshot(data []byte) (domainv7.SourceSnapshot, error) {
	f, err := object(data, "schema", "source_key", "content")
	if err != nil {
		return domainv7.SourceSnapshot{}, err
	}
	s, err := strReq(f, "schema")
	if err != nil || s != domainv7.SourceSnapshotSchema {
		return domainv7.SourceSnapshot{}, fmt.Errorf("invalid source snapshot schema")
	}
	k, err := strReq(f, "source_key")
	if err != nil {
		return domainv7.SourceSnapshot{}, err
	}
	c, err := oidReq(f, "content")
	return domainv7.SourceSnapshot{Schema: s, SourceKey: k, Content: c}, err
}
func parseOriginMap(data []byte) (domainv7.OriginMap, error) {
	f, err := object(data, "schema", "content", "runs")
	if err != nil {
		return domainv7.OriginMap{}, err
	}
	s, err := strReq(f, "schema")
	if err != nil || s != domainv7.OriginMapSchema {
		return domainv7.OriginMap{}, errors.New("invalid origin map schema")
	}
	c, err := oidReq(f, "content")
	if err != nil {
		return domainv7.OriginMap{}, err
	}
	var raws []json.RawMessage
	if err := json.Unmarshal(f["runs"], &raws); err != nil {
		return domainv7.OriginMap{}, errors.New("runs must be array")
	}
	runs := make([]domainv7.OriginRun, len(raws))
	for i, raw := range raws {
		q, e := object(raw, "kind", "length", "snapshot", "source_start")
		if e != nil {
			return domainv7.OriginMap{}, e
		}
		k, e := strReq(q, "kind")
		if e != nil {
			return domainv7.OriginMap{}, e
		}
		l, e := uintReq(q, "length")
		if e != nil {
			return domainv7.OriginMap{}, e
		}
		runs[i] = domainv7.OriginRun{Kind: k, Length: l}
		if k == "external" {
			runs[i].Snapshot, e = oidReq(q, "snapshot")
			if e != nil {
				return domainv7.OriginMap{}, e
			}
			runs[i].SourceStart, e = uintReq(q, "source_start")
			if e != nil {
				return domainv7.OriginMap{}, e
			}
		} else if k == "untraced" {
			for _, x := range []string{"snapshot", "source_start"} {
				if _, ok := q[x]; ok {
					return domainv7.OriginMap{}, fmt.Errorf("untraced run contains %s", x)
				}
			}
		}
	}
	return domainv7.OriginMap{Schema: s, Content: c, Runs: runs}, validateRuns(runs)
}
func parseSeal(data []byte) (domainv7.Seal, error) {
	f, e := object(data, "schema", "material", "provenance")
	if e != nil {
		return domainv7.Seal{}, e
	}
	s, e := strReq(f, "schema")
	if e != nil || s != domainv7.SealSchema {
		return domainv7.Seal{}, errors.New("invalid seal schema")
	}
	m, e := oidReq(f, "material")
	if e != nil {
		return domainv7.Seal{}, e
	}
	p, e := oidReq(f, "provenance")
	return domainv7.Seal{Schema: s, Material: m, Provenance: p}, e
}
func parseMaterial(data []byte) (domainv7.Material, error) {
	var v domainv5.Material
	f, e := object(data, "schema", "content", "attachments")
	if e != nil {
		return v, e
	}
	v.Schema, e = strReq(f, "schema")
	if e != nil {
		return v, e
	}
	v.Content, e = oidReq(f, "content")
	if e != nil {
		return v, e
	}
	var raws []json.RawMessage
	if e = json.Unmarshal(f["attachments"], &raws); e != nil {
		return v, e
	}
	v.Attachments, e = parseAttachments(raws)
	return v, e
}
func parseAttachments(raws []json.RawMessage) ([]domainv5.Attachment, error) {
	r := make([]domainv5.Attachment, len(raws))
	for i, x := range raws {
		f, e := object(x, "name", "media_type", "blob")
		if e != nil {
			return nil, e
		}
		r[i].Name, e = strReq(f, "name")
		if e != nil {
			return nil, e
		}
		r[i].MediaType, e = strReq(f, "media_type")
		if e != nil {
			return nil, e
		}
		r[i].Blob, e = oidReq(f, "blob")
		if e != nil {
			return nil, e
		}
	}
	return r, nil
}
func parseLinks(raws []json.RawMessage) ([]domainv5.CauseLink, error) {
	r := make([]domainv5.CauseLink, len(raws))
	for i, x := range raws {
		if e := json.Unmarshal(x, &r[i]); e != nil {
			return nil, e
		}
	}
	return r, nil
}
func parseProvenance(data []byte) (domainv7.Provenance, error) {
	f, e := object(data, "schema", "root", "draft", "cause_links", "origin")
	if e != nil {
		return domainv7.Provenance{}, e
	}
	s, e := strReq(f, "schema")
	if e != nil || s != domainv7.ProvenanceSchema {
		return domainv7.Provenance{}, errors.New("invalid provenance schema")
	}
	root, e := boolReq(f, "root")
	if e != nil {
		return domainv7.Provenance{}, e
	}
	draft, e := boolReq(f, "draft")
	if e != nil {
		return domainv7.Provenance{}, e
	}
	var raws []json.RawMessage
	if e = json.Unmarshal(f["cause_links"], &raws); e != nil {
		return domainv7.Provenance{}, e
	}
	links, e := parseLinks(raws)
	if e != nil {
		return domainv7.Provenance{}, e
	}
	o, e := optReq(f, "origin")
	if e != nil {
		return domainv7.Provenance{}, e
	}
	return domainv7.Provenance{Schema: s, Root: root, Draft: draft, CauseLinks: links, Origin: o}, nil
}
func parseCandidate(data []byte) (domainv7.Candidate, error) {
	f, e := object(data, "schema", "ref", "expected_ref_head", "content", "attachments", "root", "draft", "cause_links", "origin")
	if e != nil {
		return domainv7.Candidate{}, e
	}
	s, e := strReq(f, "schema")
	if e != nil || s != domainv7.CandidateSchema {
		return domainv7.Candidate{}, errors.New("invalid candidate schema")
	}
	ref, e := strReq(f, "ref")
	if e != nil {
		return domainv7.Candidate{}, e
	}
	head, e := optReq(f, "expected_ref_head")
	if e != nil {
		return domainv7.Candidate{}, e
	}
	c, e := oidReq(f, "content")
	if e != nil {
		return domainv7.Candidate{}, e
	}
	var ar []json.RawMessage
	if e = json.Unmarshal(f["attachments"], &ar); e != nil {
		return domainv7.Candidate{}, e
	}
	a, e := parseAttachments(ar)
	if e != nil {
		return domainv7.Candidate{}, e
	}
	root, e := boolReq(f, "root")
	if e != nil {
		return domainv7.Candidate{}, e
	}
	draft, e := boolReq(f, "draft")
	if e != nil {
		return domainv7.Candidate{}, e
	}
	var lr []json.RawMessage
	if e = json.Unmarshal(f["cause_links"], &lr); e != nil {
		return domainv7.Candidate{}, e
	}
	links, e := parseLinks(lr)
	if e != nil {
		return domainv7.Candidate{}, e
	}
	o, e := optReq(f, "origin")
	return domainv7.Candidate{Schema: s, REF: ref, ExpectedREFHead: head, Content: c, Attachments: a, Root: root, Draft: draft, CauseLinks: links, Origin: o}, e
}

func strReq(f fields, k string) (string, error) {
	r, e := required(f, k)
	if e != nil {
		return "", e
	}
	return str(r, k)
}
func oidReq(f fields, k string) (domain.ObjectID, error) {
	r, e := required(f, k)
	if e != nil {
		return domain.ObjectID{}, e
	}
	return oid(r, k)
}
func uintReq(f fields, k string) (uint64, error) {
	r, e := required(f, k)
	if e != nil {
		return 0, e
	}
	return uint64raw(r, k)
}
func boolReq(f fields, k string) (bool, error) {
	r, e := required(f, k)
	if e != nil {
		return false, e
	}
	return boolean(r, k)
}
func optReq(f fields, k string) (*domain.ObjectID, error) {
	r, e := required(f, k)
	if e != nil {
		return nil, e
	}
	return optOID(r, k)
}
func decodeCanonical[T any](data []byte, label string, parse func([]byte) (T, error), encode func(T) ([]byte, error)) (T, error) {
	var z T
	v, e := parse(data)
	if e != nil {
		return z, fmt.Errorf("decode %s: %w", label, e)
	}
	b, e := encode(v)
	if e != nil {
		return z, fmt.Errorf("validate %s: %w", label, e)
	}
	if !bytes.Equal(data, b) {
		return z, fmt.Errorf("%s is not canonical", label)
	}
	return v, nil
}
