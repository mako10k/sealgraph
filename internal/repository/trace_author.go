package repository

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"sort"

	canonicalv7 "github.com/mako10k/sealgraph/internal/canonical/v7"
	"github.com/mako10k/sealgraph/internal/domain"
	domainv5 "github.com/mako10k/sealgraph/internal/domain/v5"
	domainv7 "github.com/mako10k/sealgraph/internal/domain/v7"
	"github.com/mako10k/sealgraph/internal/store"
)

// TraceSourceInput is one recipe source whose exact bytes have already been
// read by the caller. A source is either new (SourceKey and Content) or an
// existing immutable SourceSnapshot (SnapshotID).
type TraceSourceInput struct {
	Name        string
	SourceKey   string
	Content     []byte
	DisplayPath string
	SnapshotID  *domain.ObjectID
}

// TraceRunInput is one content-ordered origin recipe run.
type TraceRunInput struct {
	Kind        string
	Length      uint64
	SourceName  string
	SourceStart uint64
}

// StoredTraceSource identifies a SourceSnapshot and its retained full source.
// DisplayPath is caller-owned presentation data and is never persisted.
type StoredTraceSource struct {
	Name        string
	SourceKey   string
	DisplayPath string
	BlobID      domain.ObjectID
	ByteCount   uint64
	SnapshotID  domain.ObjectID
	New         bool
}

type TraceSetOptions struct {
	REF         string
	Content     []byte
	ContentSet  bool
	Runs        []TraceRunInput
	Sources     []TraceSourceInput
	BeforeStore func([]StoredTraceSource) error
}

type TraceSetResult struct {
	Candidate             domainv5.Candidate
	OriginID              domain.ObjectID
	BeforeCandidateSHA256 [sha256.Size]byte
	AfterCandidateSHA256  [sha256.Size]byte
	StoredSources         []StoredTraceSource
}

type TraceClearResult struct {
	Candidate             domainv5.Candidate
	Changed               bool
	BeforeCandidateSHA256 [sha256.Size]byte
	AfterCandidateSHA256  [sha256.Size]byte
}

type LoadedTraceSource struct {
	SnapshotID domain.ObjectID
	Snapshot   domainv7.SourceSnapshot
	Content    []byte
}

type LoadedTraceOrigin struct {
	ID      domain.ObjectID
	Map     domainv7.OriginMap
	Sources []LoadedTraceSource
}

// CandidateExactDigest returns the SHA-256 of the exact persisted Candidate
// bytes. It never projects or re-encodes historical Candidate generations.
func (r *Repository) CandidateExactDigest(_ context.Context, ref string) ([sha256.Size]byte, error) {
	snapshot, err := r.candidates.LoadSnapshot(ref)
	if err != nil {
		return [sha256.Size]byte{}, err
	}
	return sha256.Sum256(snapshot.Bytes), nil
}

// CurrentREFHead returns nil when ref has no published head.
func (r *Repository) CurrentREFHead(ctx context.Context, ref string) (*domain.ObjectID, error) {
	if err := domain.ValidateREF(ref); err != nil {
		return nil, err
	}
	head, err := r.refs.Resolve(ctx, ref)
	if errors.Is(err, store.ErrRefNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &head, nil
}

func (r *Repository) TraceSet(ctx context.Context, options TraceSetOptions) (TraceSetResult, error) {
	if r.format != 7 {
		return TraceSetResult{}, fmt.Errorf("trace set requires repository format 7")
	}
	return withMutation(ctx, r.writer, "trace set Candidate", func() (TraceSetResult, error) {
		if err := domain.ValidateREF(options.REF); err != nil {
			return TraceSetResult{}, err
		}
		snapshot, err := r.candidates.LoadSnapshot(options.REF)
		if err != nil {
			if errors.Is(err, ErrCandidateNotFound) {
				return TraceSetResult{}, fmt.Errorf("REF %s has no working Candidate; run 'sealgraph add' first", options.REF)
			}
			return TraceSetResult{}, err
		}
		content := options.Content
		if !options.ContentSet {
			content, err = r.readRepositoryBlobID(ctx, snapshot.Candidate.Content, fmt.Sprintf("Candidate content for %s", options.REF))
			if err != nil {
				return TraceSetResult{}, err
			}
		}
		candidate := snapshot.Candidate
		candidate.Content = domain.ComputeNativeBlobID(content)
		originID, records, sources, err := r.prepareTraceOrigin(ctx, content, options.Runs, options.Sources)
		if err != nil {
			return TraceSetResult{}, err
		}
		candidate.Origin = &originID
		observation, _, err := r.buildObservation(ctx, "trace set Candidate")
		if err != nil {
			return TraceSetResult{}, err
		}
		if err := r.validateCandidateMutation(ctx, candidate, observation); err != nil {
			return TraceSetResult{}, err
		}
		if options.BeforeStore != nil {
			newSources := make([]StoredTraceSource, 0, len(sources))
			for _, source := range sources {
				if source.New {
					newSources = append(newSources, source)
				}
			}
			if err := options.BeforeStore(newSources); err != nil {
				return TraceSetResult{}, fmt.Errorf("trace set pre-store notice: %w", err)
			}
		}
		for _, id := range sortedTraceRecordIDs(records) {
			written, err := r.objects.WriteBlob(ctx, records[id.String()])
			if err != nil || !written.Equal(id) {
				return TraceSetResult{}, fmt.Errorf("store trace immutable Blob %s: id=%s err=%w", id, written, err)
			}
		}
		if err := r.revalidateHeads(ctx, observation, "trace set Candidate publication"); err != nil {
			return TraceSetResult{}, err
		}
		if err := r.candidates.SaveIfUnchanged(candidate, snapshot.Bytes, true); err != nil {
			return TraceSetResult{}, fmt.Errorf("save Candidate %s: %w", options.REF, err)
		}
		after, err := r.candidates.LoadSnapshot(options.REF)
		if err != nil {
			return TraceSetResult{}, fmt.Errorf("read back Candidate %s: %w", options.REF, err)
		}
		return TraceSetResult{Candidate: after.Candidate, OriginID: originID, BeforeCandidateSHA256: sha256.Sum256(snapshot.Bytes), AfterCandidateSHA256: sha256.Sum256(after.Bytes), StoredSources: sources}, nil
	})
}

func (r *Repository) TraceClear(ctx context.Context, ref string) (TraceClearResult, error) {
	if r.format != 7 {
		return TraceClearResult{}, fmt.Errorf("trace clear requires repository format 7")
	}
	return withMutation(ctx, r.writer, "trace clear Candidate", func() (TraceClearResult, error) {
		snapshot, err := r.candidates.LoadSnapshot(ref)
		if err != nil {
			return TraceClearResult{}, err
		}
		before := sha256.Sum256(snapshot.Bytes)
		if snapshot.Candidate.Origin == nil {
			return TraceClearResult{Candidate: snapshot.Candidate, BeforeCandidateSHA256: before, AfterCandidateSHA256: before}, nil
		}
		candidate := snapshot.Candidate
		candidate.Origin = nil
		observation, _, err := r.buildObservation(ctx, "trace clear Candidate")
		if err != nil {
			return TraceClearResult{}, err
		}
		if err := r.validateCandidateMutation(ctx, candidate, observation); err != nil {
			return TraceClearResult{}, err
		}
		if err := r.candidates.SaveIfUnchanged(candidate, snapshot.Bytes, true); err != nil {
			return TraceClearResult{}, fmt.Errorf("save Candidate %s: %w", ref, err)
		}
		after, err := r.candidates.LoadSnapshot(ref)
		if err != nil {
			return TraceClearResult{}, err
		}
		return TraceClearResult{Candidate: after.Candidate, Changed: true, BeforeCandidateSHA256: before, AfterCandidateSHA256: sha256.Sum256(after.Bytes)}, nil
	})
}

func (r *Repository) LoadOrigin(ctx context.Context, originID, contentID domain.ObjectID) (LoadedTraceOrigin, error) {
	content, err := r.readRepositoryBlobID(ctx, contentID, "origin content")
	if err != nil {
		return LoadedTraceOrigin{}, err
	}
	origin, snapshots, err := originClosure(contentID, content, originID, func(child domain.ObjectID) ([]byte, error) {
		return r.readRepositoryBlobID(ctx, child, "origin typed closure")
	})
	if err != nil {
		return LoadedTraceOrigin{}, err
	}
	ids := make([]string, 0, len(snapshots))
	for id := range snapshots {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	result := LoadedTraceOrigin{ID: originID, Map: origin, Sources: make([]LoadedTraceSource, 0, len(ids))}
	for _, text := range ids {
		snapshot := snapshots[text]
		source, err := r.readRepositoryBlobID(ctx, snapshot.Content, "origin source Blob")
		if err != nil {
			return LoadedTraceOrigin{}, err
		}
		result.Sources = append(result.Sources, LoadedTraceSource{SnapshotID: domain.ObjectID{Hex: text}, Snapshot: snapshot, Content: source})
	}
	return result, nil
}

func (r *Repository) prepareTraceOrigin(ctx context.Context, content []byte, inputs []TraceRunInput, sourceInputs []TraceSourceInput) (domain.ObjectID, map[string][]byte, []StoredTraceSource, error) {
	records := map[string][]byte{}
	contentID := domain.ComputeNativeBlobID(content)
	records[contentID.String()] = content
	byName, err := r.prepareTraceSources(ctx, sourceInputs, records)
	if err != nil {
		return domain.ObjectID{}, nil, nil, err
	}
	runs, err := traceRuns(inputs, byName)
	if err != nil {
		return domain.ObjectID{}, nil, nil, err
	}
	mapBytes, err := canonicalv7.EncodeOriginMap(domainv7.OriginMap{Schema: domainv7.OriginMapSchema, Content: contentID, Runs: runs})
	if err != nil {
		return domain.ObjectID{}, nil, nil, fmt.Errorf("canonicalize trace origin map: %w", err)
	}
	originID := domain.ComputeNativeBlobID(mapBytes)
	records[originID.String()] = mapBytes
	if _, _, err := originClosure(contentID, content, originID, func(id domain.ObjectID) ([]byte, error) {
		if data, ok := records[id.String()]; ok {
			return data, nil
		}
		return r.readRepositoryBlobID(ctx, id, "trace origin closure")
	}); err != nil {
		return domain.ObjectID{}, nil, nil, fmt.Errorf("validate trace origin closure: %w", err)
	}
	sources := make([]StoredTraceSource, 0, len(byName))
	for _, source := range byName {
		sources = append(sources, source)
	}
	sort.Slice(sources, func(i, j int) bool { return sources[i].Name < sources[j].Name })
	return originID, records, sources, nil
}

func (r *Repository) prepareTraceSources(ctx context.Context, inputs []TraceSourceInput, records map[string][]byte) (map[string]StoredTraceSource, error) {
	result := make(map[string]StoredTraceSource, len(inputs))
	for _, input := range inputs {
		if input.Name == "" {
			return nil, errors.New("trace source name is empty")
		}
		if _, exists := result[input.Name]; exists {
			return nil, fmt.Errorf("duplicate trace source name %q", input.Name)
		}
		source, err := r.prepareTraceSource(ctx, input, records)
		if err != nil {
			return nil, err
		}
		result[input.Name] = source
	}
	return result, nil
}

func (r *Repository) prepareTraceSource(ctx context.Context, input TraceSourceInput, records map[string][]byte) (StoredTraceSource, error) {
	if input.SnapshotID != nil {
		if input.SourceKey != "" || input.Content != nil || input.DisplayPath != "" {
			return StoredTraceSource{}, fmt.Errorf("trace source %q mixes snapshot with new source bytes", input.Name)
		}
		data, err := r.readRepositoryBlobID(ctx, *input.SnapshotID, "trace source snapshot")
		if err != nil {
			return StoredTraceSource{}, err
		}
		snapshot, err := canonicalv7.DecodeSourceSnapshot(data)
		if err != nil {
			return StoredTraceSource{}, fmt.Errorf("trace source %q snapshot %s is invalid: %w", input.Name, *input.SnapshotID, err)
		}
		bytes, err := r.readRepositoryBlobID(ctx, snapshot.Content, "trace source full Blob")
		if err != nil {
			return StoredTraceSource{}, err
		}
		return StoredTraceSource{Name: input.Name, SourceKey: snapshot.SourceKey, BlobID: snapshot.Content, ByteCount: uint64(len(bytes)), SnapshotID: *input.SnapshotID}, nil
	}
	if input.SourceKey == "" {
		return StoredTraceSource{}, fmt.Errorf("trace source %q has no source_key", input.Name)
	}
	blobID := domain.ComputeNativeBlobID(input.Content)
	snapshotBytes, err := canonicalv7.EncodeSourceSnapshot(domainv7.SourceSnapshot{Schema: domainv7.SourceSnapshotSchema, SourceKey: input.SourceKey, Content: blobID})
	if err != nil {
		return StoredTraceSource{}, err
	}
	snapshotID := domain.ComputeNativeBlobID(snapshotBytes)
	records[blobID.String()], records[snapshotID.String()] = input.Content, snapshotBytes
	return StoredTraceSource{Name: input.Name, SourceKey: input.SourceKey, DisplayPath: input.DisplayPath, BlobID: blobID, ByteCount: uint64(len(input.Content)), SnapshotID: snapshotID, New: true}, nil
}

func traceRuns(inputs []TraceRunInput, sources map[string]StoredTraceSource) ([]domainv7.OriginRun, error) {
	runs := make([]domainv7.OriginRun, 0, len(inputs))
	for _, input := range inputs {
		run := domainv7.OriginRun{Kind: input.Kind, Length: input.Length, SourceStart: input.SourceStart}
		switch input.Kind {
		case "external":
			source, ok := sources[input.SourceName]
			if !ok {
				return nil, fmt.Errorf("external trace run names unknown source %q", input.SourceName)
			}
			run.Snapshot = source.SnapshotID
		case "untraced":
			if input.SourceName != "" || input.SourceStart != 0 {
				return nil, errors.New("untraced trace run has source fields")
			}
		default:
			return nil, fmt.Errorf("unknown trace run kind %q", input.Kind)
		}
		runs = append(runs, run)
	}
	return runs, nil
}

func sortedTraceRecordIDs(records map[string][]byte) []domain.ObjectID {
	ids := make([]domain.ObjectID, 0, len(records))
	for text := range records {
		ids = append(ids, domain.ObjectID{Hex: text})
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i].String() < ids[j].String() })
	return ids
}

func sameOrigin(left, right *domain.ObjectID) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return left.Equal(*right)
}
