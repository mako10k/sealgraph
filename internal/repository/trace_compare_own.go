package repository

import (
	"bytes"
	"context"
	"errors"
	"fmt"

	"github.com/mako10k/sealgraph/internal/domain"
	domainv7 "github.com/mako10k/sealgraph/internal/domain/v7"
	"github.com/mako10k/sealgraph/internal/tracecompare"
	"github.com/mako10k/sealgraph/internal/workfile"
)

// TraceOwnBaselineKind identifies the immutable baseline selected for a local
// trace observation. The selection is explicit; no kind falls back to another.
type TraceOwnBaselineKind string

const (
	TraceOwnCandidate TraceOwnBaselineKind = "CANDIDATE"
	TraceOwnSeal      TraceOwnBaselineKind = "SEAL"
	TraceOwnHead      TraceOwnBaselineKind = "HEAD"
)

// TraceOwnBaseline selects exactly one Candidate, exact Seal, or current HEAD.
type TraceOwnBaseline struct {
	Kind   TraceOwnBaselineKind
	REF    string
	SealID *domain.ObjectID
}

// TraceCompareOwnOptions describes the bounded, local comparison operation.
type TraceCompareOwnOptions struct {
	Baseline TraceOwnBaseline
}

// TraceOwnRunResult is the local fact for one External OriginMap run.
type TraceOwnRunResult struct {
	RunIndex           int
	SourceSnapshotID   domain.ObjectID
	SourceKey          string
	SourceIdentity     string
	SourcePath         string
	SourceError        string
	OldStart           int
	Length             int
	CurrentBlobID      *domain.ObjectID
	Presence           tracecompare.Status
	PresenceReason     string
	SelectedMatchStart *int
	Examined           bool
}

// TraceCompareOwnResult contains only the selected baseline's local facts.
type TraceCompareOwnResult struct {
	REF             string
	Baseline        TraceOwnBaseline
	Content         domain.ObjectID
	Origin          *domain.ObjectID
	SealID          *domain.ObjectID
	CandidateDigest [32]byte
	Runs            []TraceOwnRunResult
}

type traceOwnBaselineState struct {
	selection       TraceOwnBaseline
	content         domain.ObjectID
	origin          *domain.ObjectID
	candidateDigest [32]byte
	head            *domain.ObjectID
	sealID          *domain.ObjectID
}

type traceOwnSourceState struct {
	sources     map[string]LoadedTraceSource
	paths       map[string]string
	bound       map[string]bool
	readFailed  map[string]bool
	sourceError map[string]string
	current     map[string][]byte
}

// TraceCompareOwn compares every External run in the explicitly selected
// baseline against the stable file bound to each run's SourceKey.
func (r *Repository) TraceCompareOwn(ctx context.Context, options TraceCompareOwnOptions) (TraceCompareOwnResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	state, err := r.loadTraceOwnBaseline(ctx, options.Baseline)
	if err != nil {
		return TraceCompareOwnResult{}, err
	}
	if state.origin == nil {
		if err := r.recheckTraceOwnState(ctx, state, nil, nil, nil, nil); err != nil {
			return TraceCompareOwnResult{}, err
		}
		return TraceCompareOwnResult{REF: state.selection.REF, Baseline: state.selection, Content: state.content, SealID: state.sealID, CandidateDigest: state.candidateDigest}, nil
	}
	origin, err := r.LoadOrigin(ctx, *state.origin, state.content)
	if err != nil {
		return TraceCompareOwnResult{}, err
	}
	sourceState, err := r.loadTraceOwnSources(origin)
	if err != nil {
		return TraceCompareOwnResult{}, err
	}
	if err := r.observeTraceOwnSources(ctx, origin, &sourceState); err != nil {
		return TraceCompareOwnResult{}, err
	}

	result := TraceCompareOwnResult{REF: state.selection.REF, Baseline: state.selection, Content: state.content, Origin: state.origin, SealID: state.sealID, CandidateDigest: state.candidateDigest}
	for mapIndex, run := range origin.Map.Runs {
		if run.Kind != "external" {
			continue
		}
		item, err := compareTraceOwnRun(ctx, mapIndex, run, sourceState)
		if err != nil {
			return TraceCompareOwnResult{}, err
		}
		result.Runs = append(result.Runs, item)
	}
	if err := r.recheckTraceOwnState(ctx, state, sourceState.paths, sourceState.bound, sourceState.readFailed, sourceState.current); err != nil {
		return TraceCompareOwnResult{}, err
	}
	return result, nil
}

func (r *Repository) loadTraceOwnSources(origin LoadedTraceOrigin) (traceOwnSourceState, error) {
	sources := make(map[string]LoadedTraceSource, len(origin.Sources))
	for _, source := range origin.Sources {
		sources[source.SnapshotID.String()] = source
	}
	return traceOwnSourceState{
		sources: sources, paths: make(map[string]string), bound: make(map[string]bool),
		readFailed: make(map[string]bool), sourceError: make(map[string]string), current: make(map[string][]byte),
	}, nil
}

func (r *Repository) observeTraceOwnSources(_ context.Context, origin LoadedTraceOrigin, state *traceOwnSourceState) error {
	seen := make(map[string]bool)
	for mapIndex, run := range origin.Map.Runs {
		if run.Kind != "external" {
			continue
		}
		source, ok := state.sources[run.Snapshot.String()]
		if !ok {
			return fmt.Errorf("origin run %d references missing SourceSnapshot %s", mapIndex, run.Snapshot)
		}
		key := source.Snapshot.SourceKey
		if seen[key] {
			continue
		}
		seen[key] = true
		if err := r.observeTraceOwnSource(key, state); err != nil {
			return err
		}
	}
	return nil
}

func (r *Repository) observeTraceOwnSource(key string, state *traceOwnSourceState) error {
	binding, err := r.TraceSourceShow(key)
	if err != nil {
		if !errors.Is(err, ErrTraceSourceNotFound) {
			return err
		}
		state.paths[key], state.sourceError[key] = "", err.Error()
		return nil
	}
	state.paths[key], state.bound[key] = binding.Path, true
	data, readErr := workfile.ReadStable(r.workDir, binding.Path)
	if readErr == nil {
		state.current[key] = data
		return nil
	}
	state.readFailed[key], state.sourceError[key] = true, readErr.Error()
	return nil
}

func compareTraceOwnRun(ctx context.Context, index int, run domainv7.OriginRun, state traceOwnSourceState) (TraceOwnRunResult, error) {
	source, ok := state.sources[run.Snapshot.String()]
	if !ok {
		return TraceOwnRunResult{}, fmt.Errorf("origin run %d references missing SourceSnapshot %s", index, run.Snapshot)
	}
	if run.SourceStart > uint64(len(source.Content)) || run.Length > uint64(len(source.Content))-run.SourceStart {
		return TraceOwnRunResult{}, fmt.Errorf("origin run %d exceeds SourceSnapshot %s", index, run.Snapshot)
	}
	key := source.Snapshot.SourceKey
	item := TraceOwnRunResult{RunIndex: index, SourceSnapshotID: run.Snapshot, SourceKey: key, SourceIdentity: key, SourcePath: state.paths[key], SourceError: state.sourceError[key], OldStart: int(run.SourceStart), Length: int(run.Length), Presence: tracecompare.Undetermined, Examined: true}
	data, ok := state.current[key]
	if !ok {
		item.PresenceReason = "SOURCE_READ_FAILED"
		return item, nil
	}
	currentID := domain.ComputeNativeBlobID(data)
	item.CurrentBlobID = &currentID
	start := int(run.SourceStart)
	comparison, err := tracecompare.Compare(ctx, tracecompare.Input{Source: source.Content, SourceStart: start, Run: source.Content[start : start+int(run.Length)], Current: data})
	if err != nil {
		return TraceOwnRunResult{}, err
	}
	item.Presence = comparison.Status
	switch comparison.Status {
	case tracecompare.Present:
		selected := comparison.Start
		item.SelectedMatchStart, item.PresenceReason = &selected, "EXACT_MATCH"
	case tracecompare.AbsentExact:
		item.PresenceReason = "NO_EXACT_MATCH"
	default:
		item.Examined, item.PresenceReason = false, "SEARCH_INTERRUPTED"
		if comparison.Reason != "" {
			item.PresenceReason = comparison.Reason
		}
	}
	return item, nil
}

func (r *Repository) loadTraceOwnBaseline(ctx context.Context, selection TraceOwnBaseline) (traceOwnBaselineState, error) {
	if selection.Kind != TraceOwnSeal && selection.REF == "" {
		return traceOwnBaselineState{}, errors.New("trace own baseline REF is required")
	}
	if selection.REF != "" {
		if err := domain.ValidateREF(selection.REF); err != nil {
			return traceOwnBaselineState{}, err
		}
	}
	state := traceOwnBaselineState{selection: selection}
	switch selection.Kind {
	case TraceOwnCandidate:
		inspection, err := r.InspectCandidate(ctx, selection.REF)
		if err != nil {
			return traceOwnBaselineState{}, err
		}
		state.content = inspection.Candidate.Content
		state.origin = inspection.Candidate.Origin
		state.head = inspection.CurrentHead
		state.candidateDigest, err = r.CandidateExactDigest(ctx, selection.REF)
		if err != nil {
			return traceOwnBaselineState{}, err
		}
	case TraceOwnSeal:
		if selection.SealID == nil {
			return traceOwnBaselineState{}, errors.New("exact Seal baseline requires SealID")
		}
		resolved, err := r.LoadSeal(ctx, *selection.SealID)
		if err != nil {
			return traceOwnBaselineState{}, err
		}
		state.content = resolved.Material.Content
		_, err = r.readRepositoryBlobID(ctx, state.content, "trace own Seal content")
		if err != nil {
			return traceOwnBaselineState{}, err
		}
		state.origin = resolved.Provenance.Origin
		sealID := *selection.SealID
		state.sealID = &sealID
	case TraceOwnHead:
		head, err := r.CurrentREFHead(ctx, selection.REF)
		if err != nil {
			return traceOwnBaselineState{}, err
		}
		if head == nil {
			return traceOwnBaselineState{}, fmt.Errorf("REF %s has no HEAD", selection.REF)
		}
		resolved, err := r.LoadSeal(ctx, *head)
		if err != nil {
			return traceOwnBaselineState{}, err
		}
		state.head = head
		state.sealID = head
		state.content = resolved.Material.Content
		_, err = r.readRepositoryBlobID(ctx, state.content, "trace own HEAD content")
		if err != nil {
			return traceOwnBaselineState{}, err
		}
		state.origin = resolved.Provenance.Origin
	default:
		return traceOwnBaselineState{}, fmt.Errorf("unsupported trace own baseline %q", selection.Kind)
	}
	return state, nil
}

func (r *Repository) recheckTraceOwnState(ctx context.Context, state traceOwnBaselineState, paths map[string]string, bound map[string]bool, readFailed map[string]bool, current map[string][]byte) error {
	if err := r.recheckTraceOwnBaseline(ctx, state); err != nil {
		return err
	}
	return r.recheckTraceOwnSources(paths, bound, readFailed, current)
}

func (r *Repository) recheckTraceOwnBaseline(ctx context.Context, state traceOwnBaselineState) error {
	switch state.selection.Kind {
	case TraceOwnCandidate:
		digest, err := r.CandidateExactDigest(ctx, state.selection.REF)
		if err != nil || !bytes.Equal(digest[:], state.candidateDigest[:]) {
			return fmt.Errorf("Candidate %s changed during trace comparison", state.selection.REF)
		}
		head, err := r.CurrentREFHead(ctx, state.selection.REF)
		if err != nil || !sameTraceOwnID(head, state.head) {
			return fmt.Errorf("HEAD for Candidate %s changed during trace comparison", state.selection.REF)
		}
	case TraceOwnHead:
		head, err := r.CurrentREFHead(ctx, state.selection.REF)
		if err != nil || head == nil || state.head == nil || !head.Equal(*state.head) {
			return fmt.Errorf("HEAD for %s changed during trace comparison", state.selection.REF)
		}
	}
	return nil
}

func (r *Repository) recheckTraceOwnSources(paths map[string]string, bound map[string]bool, readFailed map[string]bool, current map[string][]byte) error {
	for key, path := range paths {
		if !bound[key] {
			if _, err := r.TraceSourceShow(key); err == nil {
				return fmt.Errorf("source binding %s appeared during trace comparison", key)
			} else if !errors.Is(err, ErrTraceSourceNotFound) {
				return err
			}
			continue
		}
		binding, err := r.TraceSourceShow(key)
		if err != nil || binding.Path != path {
			return fmt.Errorf("source binding %s changed during trace comparison", key)
		}
		data, err := workfile.ReadStable(r.workDir, path)
		if err != nil {
			if readFailed[key] {
				continue
			}
			return fmt.Errorf("source %s changed during trace comparison: %w", key, err)
		}
		if readFailed[key] {
			return fmt.Errorf("source %s changed during trace comparison: read succeeded after initial failure", key)
		}
		if initial, ok := current[key]; ok && !bytes.Equal(initial, data) {
			return fmt.Errorf("source %s changed during trace comparison", key)
		}
	}
	return nil
}

func sameTraceOwnID(a, b *domain.ObjectID) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return a.Equal(*b)
}
