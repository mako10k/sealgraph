package repository

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	canonicalv7 "github.com/mako10k/sealgraph/internal/canonical/v7"
	"github.com/mako10k/sealgraph/internal/domain"
	"github.com/mako10k/sealgraph/internal/workfile"
)

const traceCorrespondenceSchema = "sealgraph/trace-correspondence/v1"

var ErrTraceCorrespondenceNotFound = errors.New("trace correspondence not found")

type TraceCorrespondenceRange struct {
	Start  uint64 `json:"start"`
	Length uint64 `json:"length"`
}

type TraceCorrespondence struct {
	Schema         string
	SourceSnapshot domain.ObjectID
	SourceStart    uint64
	Length         uint64
	CurrentBlob    domain.ObjectID
	CurrentRanges  []TraceCorrespondenceRange
	Deleted        bool
	Reason         string
	DeclaredAt     string
}

type TraceCorrespondencePutOptions struct {
	SourceSnapshot domain.ObjectID
	SourceStart    uint64
	Length         uint64
	CurrentBlob    domain.ObjectID
	CurrentRanges  []TraceCorrespondenceRange
	Deleted        bool
	Reason         string
	DeclaredAt     string
}

type TraceCorrespondenceMutation struct {
	ID      string
	Record  TraceCorrespondence
	Changed bool
}

type TraceCorrespondenceRecord struct {
	ID     string
	Record TraceCorrespondence
}

type traceCorrespondenceWire struct {
	Schema         string                     `json:"schema"`
	SourceSnapshot string                     `json:"source_snapshot"`
	SourceStart    uint64                     `json:"source_start"`
	Length         uint64                     `json:"length"`
	CurrentBlob    string                     `json:"current_blob"`
	CurrentRanges  []TraceCorrespondenceRange `json:"current_ranges"`
	Deleted        bool                       `json:"deleted"`
	Reason         string                     `json:"reason"`
	DeclaredAt     string                     `json:"declared_at"`
}

func traceCorrespondenceRoot(repositoryDir string) string {
	return filepath.Join(repositoryDir, "local", "trace-correspondences")
}

func traceCorrespondenceID(record []byte) string {
	digest := sha256.Sum256(record)
	return hex.EncodeToString(digest[:])
}

func correspondenceWire(record TraceCorrespondence) traceCorrespondenceWire {
	ranges := append([]TraceCorrespondenceRange(nil), record.CurrentRanges...)
	return traceCorrespondenceWire{Schema: traceCorrespondenceSchema, SourceSnapshot: record.SourceSnapshot.String(), SourceStart: record.SourceStart, Length: record.Length, CurrentBlob: record.CurrentBlob.String(), CurrentRanges: ranges, Deleted: record.Deleted, Reason: record.Reason, DeclaredAt: record.DeclaredAt}
}

func correspondenceRecord(w traceCorrespondenceWire) (TraceCorrespondence, error) {
	source, err := parseCorrespondenceID(w.SourceSnapshot, "source_snapshot")
	if err != nil {
		return TraceCorrespondence{}, err
	}
	current, err := parseCorrespondenceID(w.CurrentBlob, "current_blob")
	if err != nil {
		return TraceCorrespondence{}, err
	}
	return TraceCorrespondence{Schema: w.Schema, SourceSnapshot: source, SourceStart: w.SourceStart, Length: w.Length, CurrentBlob: current, CurrentRanges: append([]TraceCorrespondenceRange(nil), w.CurrentRanges...), Deleted: w.Deleted, Reason: w.Reason, DeclaredAt: w.DeclaredAt}, nil
}

func parseCorrespondenceID(value, label string) (domain.ObjectID, error) {
	if len(value) != 64 || strings.Trim(value, "0123456789abcdef") != "" {
		return domain.ObjectID{}, fmt.Errorf("%s is not a full lowercase object ID", label)
	}
	return domain.ObjectID{Hex: value}, nil
}

func encodeTraceCorrespondence(record TraceCorrespondence) ([]byte, error) {
	if record.Schema != traceCorrespondenceSchema {
		return nil, fmt.Errorf("unsupported schema %q", record.Schema)
	}
	if err := record.SourceSnapshot.ValidateNative(); err != nil {
		return nil, err
	}
	if err := record.CurrentBlob.ValidateNative(); err != nil {
		return nil, err
	}
	if record.Length == 0 {
		return nil, errors.New("correspondence length must be positive")
	}
	if _, err := time.Parse("2006-01-02T15:04:05Z", record.DeclaredAt); err != nil {
		return nil, errors.New("declared_at must be an explicit UTC timestamp")
	}
	if len(record.DeclaredAt) != len("2006-01-02T15:04:05Z") {
		return nil, errors.New("declared_at must use YYYY-MM-DDTHH:MM:SSZ exactly")
	}
	if strings.TrimSpace(record.Reason) == "" || !utf8.ValidString(record.Reason) {
		return nil, errors.New("correspondence reason is empty")
	}
	if record.Deleted != (len(record.CurrentRanges) == 0) {
		return nil, errors.New("deleted and current_ranges disagree")
	}
	if !record.Deleted {
		for _, r := range record.CurrentRanges {
			if r.Length == 0 {
				return nil, errors.New("current range length must be positive")
			}
		}
	}
	return json.Marshal(correspondenceWire(record))
}

// TraceCorrespondenceCanonicalJSON returns the exact compact record bytes used
// for the declaration ID. It never appends a line feed.
func TraceCorrespondenceCanonicalJSON(record TraceCorrespondence) ([]byte, error) {
	return encodeTraceCorrespondence(record)
}

func decodeTraceCorrespondence(data []byte) (TraceCorrespondence, error) {
	var wire traceCorrespondenceWire
	if err := decodeTraceCorrespondenceWire(data, &wire); err != nil {
		return TraceCorrespondence{}, err
	}
	if wire.Schema != traceCorrespondenceSchema {
		return TraceCorrespondence{}, fmt.Errorf("unsupported schema %q", wire.Schema)
	}
	record, err := correspondenceRecord(wire)
	if err != nil {
		return TraceCorrespondence{}, err
	}
	canonical, err := encodeTraceCorrespondence(record)
	if err != nil {
		return TraceCorrespondence{}, err
	}
	if !bytes.Equal(data, canonical) {
		return TraceCorrespondence{}, errors.New("non-canonical trace correspondence representation")
	}
	return record, nil
}

// ParseTraceCorrespondenceInput parses the strict user declaration document.
// It validates schema, scalar/range shape, and canonical field semantics; the
// repository method additionally validates source/current bytes and identity.
func ParseTraceCorrespondenceInput(data []byte) (TraceCorrespondencePutOptions, error) {
	var wire traceCorrespondenceWire
	if err := decodeTraceCorrespondenceWire(data, &wire); err != nil {
		return TraceCorrespondencePutOptions{}, err
	}
	if wire.Schema != traceCorrespondenceSchema {
		return TraceCorrespondencePutOptions{}, fmt.Errorf("unsupported schema %q", wire.Schema)
	}
	record, err := correspondenceRecord(wire)
	if err != nil {
		return TraceCorrespondencePutOptions{}, err
	}
	if _, err := encodeTraceCorrespondence(record); err != nil {
		return TraceCorrespondencePutOptions{}, err
	}
	return TraceCorrespondencePutOptions{SourceSnapshot: record.SourceSnapshot, SourceStart: record.SourceStart, Length: record.Length, CurrentBlob: record.CurrentBlob, CurrentRanges: record.CurrentRanges, Deleted: record.Deleted, Reason: record.Reason, DeclaredAt: record.DeclaredAt}, nil
}

func decodeTraceCorrespondenceWire(data []byte, target *traceCorrespondenceWire) error {
	var fields map[string]json.RawMessage
	if err := decodeUniqueObject(data, &fields); err != nil {
		return err
	}
	allowed := map[string]bool{"schema": true, "source_snapshot": true, "source_start": true, "length": true, "current_blob": true, "current_ranges": true, "deleted": true, "reason": true, "declared_at": true}
	for key := range fields {
		if !allowed[key] {
			return fmt.Errorf("unknown member %q", key)
		}
	}
	if len(fields) != 9 {
		return errors.New("trace correspondence requires all record members")
	}
	rawRanges, ok := fields["current_ranges"]
	if !ok {
		return errors.New("current_ranges member is required")
	}
	var rangeValues []json.RawMessage
	if err := json.Unmarshal(rawRanges, &rangeValues); err != nil {
		return fmt.Errorf("current_ranges is invalid: %w", err)
	}
	ranges := make([]TraceCorrespondenceRange, len(rangeValues))
	for i, raw := range rangeValues {
		var values map[string]json.RawMessage
		if err := decodeUniqueObject(raw, &values); err != nil {
			return fmt.Errorf("current_ranges[%d]: %w", i, err)
		}
		for key := range values {
			if key != "start" && key != "length" {
				return fmt.Errorf("current_ranges[%d]: unknown member %q", i, key)
			}
		}
		if len(values) != 2 {
			return fmt.Errorf("current_ranges[%d]: start and length are required", i)
		}
		if err := json.Unmarshal(values["start"], &ranges[i].Start); err != nil {
			return fmt.Errorf("current_ranges[%d].start: %w", i, err)
		}
		if err := json.Unmarshal(values["length"], &ranges[i].Length); err != nil {
			return fmt.Errorf("current_ranges[%d].length: %w", i, err)
		}
	}
	delete(fields, "current_ranges")
	encoded, err := json.Marshal(fields)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(encoded, target); err != nil {
		return err
	}
	target.CurrentRanges = ranges
	return nil
}

func decodeUniqueObject(data []byte, target *map[string]json.RawMessage) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	token, err := decoder.Token()
	if err != nil {
		return err
	}
	if token != json.Delim('{') {
		return errors.New("trace correspondence member must be a JSON object")
	}
	values := map[string]json.RawMessage{}
	for decoder.More() {
		keyToken, err := decoder.Token()
		if err != nil {
			return err
		}
		key, ok := keyToken.(string)
		if !ok {
			return errors.New("trace correspondence key is not a string")
		}
		if _, exists := values[key]; exists {
			return fmt.Errorf("duplicate member %q", key)
		}
		var raw json.RawMessage
		if err := decoder.Decode(&raw); err != nil {
			return err
		}
		values[key] = raw
	}
	if _, err := decoder.Token(); err != nil {
		return err
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return errors.New("trailing JSON data")
		}
		return err
	}
	*target = values
	return nil
}

func (r *Repository) validateTraceCorrespondence(ctx context.Context, record TraceCorrespondence) error {
	data, err := r.readRepositoryBlobID(ctx, record.SourceSnapshot, "trace correspondence source snapshot")
	if err != nil {
		return err
	}
	snapshot, err := canonicalv7.DecodeSourceSnapshot(data)
	if err != nil {
		return err
	}
	source, err := r.readRepositoryBlobID(ctx, snapshot.Content, "trace correspondence source bytes")
	if err != nil {
		return err
	}
	if record.SourceStart > uint64(len(source)) || record.Length > uint64(len(source))-record.SourceStart {
		return errors.New("correspondence source range is outside SourceSnapshot")
	}
	binding, err := r.TraceSourceShow(snapshot.SourceKey)
	if err != nil {
		return err
	}
	current, err := workfile.ReadStable(r.workDir, binding.Path)
	if err != nil {
		return err
	}
	currentID := domain.ComputeNativeBlobID(current)
	if !currentID.Equal(record.CurrentBlob) {
		return fmt.Errorf("current BlobID %s does not match observed binding %s", record.CurrentBlob, currentID)
	}
	for i, item := range record.CurrentRanges {
		if item.Start > uint64(len(current)) || item.Length > uint64(len(current))-item.Start {
			return fmt.Errorf("current range %d is outside observed bytes", i)
		}
		for j := 0; j < i; j++ {
			if rangesOverlap(item, record.CurrentRanges[j]) {
				return fmt.Errorf("current ranges %d and %d overlap", j, i)
			}
		}
	}
	return nil
}

func rangesOverlap(a, b TraceCorrespondenceRange) bool {
	return a.Start < b.Start+b.Length && b.Start < a.Start+a.Length
}

func ensureTraceCorrespondenceRoot(root string) error {
	if err := os.MkdirAll(filepath.Dir(root), 0o700); err != nil {
		return err
	}
	if err := os.MkdirAll(root, 0o700); err != nil {
		return err
	}
	for _, path := range []string{filepath.Dir(root), root} {
		info, err := os.Lstat(path)
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
			return fmt.Errorf("trace correspondence store path %s is not a directory", path)
		}
	}
	return nil
}

func (r *Repository) TraceCorrespondencePut(ctx context.Context, options TraceCorrespondencePutOptions) (TraceCorrespondenceMutation, error) {
	record := TraceCorrespondence{Schema: traceCorrespondenceSchema, SourceSnapshot: options.SourceSnapshot, SourceStart: options.SourceStart, Length: options.Length, CurrentBlob: options.CurrentBlob, CurrentRanges: append([]TraceCorrespondenceRange(nil), options.CurrentRanges...), Deleted: options.Deleted, Reason: options.Reason, DeclaredAt: options.DeclaredAt}
	if _, err := encodeTraceCorrespondence(record); err != nil {
		return TraceCorrespondenceMutation{}, err
	}
	if err := r.validateTraceCorrespondence(ctx, record); err != nil {
		return TraceCorrespondenceMutation{}, err
	}
	data, err := encodeTraceCorrespondence(record)
	if err != nil {
		return TraceCorrespondenceMutation{}, err
	}
	id := traceCorrespondenceID(data)
	return withMutation(ctx, r.writer, "put trace correspondence", func() (TraceCorrespondenceMutation, error) {
		root := traceCorrespondenceRoot(r.dir)
		if err := ensureTraceCorrespondenceRoot(root); err != nil {
			return TraceCorrespondenceMutation{}, err
		}
		path := filepath.Join(root, id+".json")
		if existing, _, err := readStableTraceSourceFile(path); err == nil {
			if !bytes.Equal(existing, data) {
				return TraceCorrespondenceMutation{}, errors.New("trace correspondence ID collision")
			}
			return TraceCorrespondenceMutation{ID: id, Record: record}, nil
		} else if !errors.Is(err, os.ErrNotExist) {
			return TraceCorrespondenceMutation{}, err
		}
		tmp, err := os.CreateTemp(root, ".tmp-trace-correspondence-")
		if err != nil {
			return TraceCorrespondenceMutation{}, err
		}
		tmpPath := tmp.Name()
		defer os.Remove(tmpPath)
		if err = tmp.Chmod(0o600); err == nil {
			_, err = tmp.Write(data)
		}
		if err == nil {
			err = tmp.Sync()
		}
		closeErr := tmp.Close()
		if err == nil {
			err = closeErr
		}
		if err != nil {
			return TraceCorrespondenceMutation{}, err
		}
		if err := os.Link(tmpPath, path); err != nil {
			if errors.Is(err, os.ErrExist) {
				existing, _, readErr := readStableTraceSourceFile(path)
				if readErr != nil {
					return TraceCorrespondenceMutation{}, readErr
				}
				if bytes.Equal(existing, data) {
					return TraceCorrespondenceMutation{ID: id, Record: record}, nil
				}
				return TraceCorrespondenceMutation{}, errors.New("trace correspondence ID collision")
			}
			return TraceCorrespondenceMutation{}, err
		}
		published, _, err := readStableTraceSourceFile(path)
		if err != nil {
			return TraceCorrespondenceMutation{}, err
		}
		if !bytes.Equal(published, data) {
			return TraceCorrespondenceMutation{}, errors.New("trace correspondence readback differs after publication")
		}
		return TraceCorrespondenceMutation{ID: id, Record: record, Changed: true}, nil
	})
}

func (r *Repository) TraceCorrespondenceShow(id string) (TraceCorrespondence, error) {
	if _, err := parseCorrespondenceID(id, "correspondence ID"); err != nil {
		return TraceCorrespondence{}, err
	}
	path := filepath.Join(traceCorrespondenceRoot(r.dir), id+".json")
	info, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		return TraceCorrespondence{}, fmt.Errorf("%w: %s", ErrTraceCorrespondenceNotFound, id)
	}
	if err != nil {
		return TraceCorrespondence{}, err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return TraceCorrespondence{}, errors.New("trace correspondence is not a regular non-symlink file")
	}
	data, _, err := readStableTraceSourceFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return TraceCorrespondence{}, fmt.Errorf("%w: %s", ErrTraceCorrespondenceNotFound, id)
	}
	if err != nil {
		return TraceCorrespondence{}, err
	}
	record, err := decodeTraceCorrespondence(data)
	if err != nil {
		return TraceCorrespondence{}, err
	}
	if traceCorrespondenceID(data) != id {
		return TraceCorrespondence{}, errors.New("trace correspondence filename does not match record ID")
	}
	return record, nil
}

func (r *Repository) TraceCorrespondenceList() ([]TraceCorrespondenceRecord, error) {
	root := traceCorrespondenceRoot(r.dir)
	if info, err := os.Lstat(root); errors.Is(err, os.ErrNotExist) {
		return []TraceCorrespondenceRecord{}, nil
	} else if err != nil {
		return nil, err
	} else if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return nil, errors.New("trace correspondence store is not a regular directory")
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}
	type item = TraceCorrespondenceRecord
	items := []item{}
	for _, entry := range entries {
		if entry.IsDir() || entry.Type()&os.ModeSymlink != 0 || filepath.Ext(entry.Name()) != ".json" {
			return nil, fmt.Errorf("unexpected trace correspondence entry %s", entry.Name())
		}
		id := strings.TrimSuffix(entry.Name(), ".json")
		record, err := r.TraceCorrespondenceShow(id)
		if err != nil {
			return nil, err
		}
		items = append(items, item{ID: id, Record: record})
	}
	sort.Slice(items, func(i, j int) bool { return items[i].ID < items[j].ID })
	result := make([]TraceCorrespondenceRecord, len(items))
	for i := range items {
		result[i] = items[i]
	}
	return result, nil
}

func (r *Repository) TraceCorrespondenceRemove(ctx context.Context, id string) (bool, error) {
	if _, err := parseCorrespondenceID(id, "correspondence ID"); err != nil {
		return false, err
	}
	return withMutation(ctx, r.writer, "remove trace correspondence", func() (bool, error) {
		_, err := r.TraceCorrespondenceShow(id)
		if err != nil {
			return false, err
		}
		if err := os.Remove(filepath.Join(traceCorrespondenceRoot(r.dir), id+".json")); err != nil {
			return false, err
		}
		return true, nil
	})
}
