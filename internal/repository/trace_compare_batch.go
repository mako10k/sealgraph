package repository

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/mako10k/sealgraph/internal/domain"
)

// TraceOwnSourceObservation is the one local source observation shared by all
// baselines in one TraceCompareOwnBatch execution.
type TraceOwnSourceObservation struct {
	SourceKey      string
	SourceIdentity string
	Path           *string
	Bound          bool
	ReadFailed     bool
	ObservedAt     time.Time
	CurrentBlobID  *domain.ObjectID
	ByteLength     *int
	Error          *string
}

// TraceCompareOwnBatch observes every source_key once, then compares all
// explicitly selected baselines against those same bytes.
func (r *Repository) TraceCompareOwnBatch(ctx context.Context, selections []TraceOwnBaseline) ([]TraceCompareOwnResult, []TraceOwnSourceObservation, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	states, origins, err := r.loadTraceOwnBatch(ctx, selections)
	if err != nil {
		return nil, nil, err
	}
	sourceState, observations, err := r.observeTraceOwnBatch(origins)
	if err != nil {
		return nil, nil, err
	}

	results, err := compareTraceOwnBatch(ctx, states, origins, sourceState.traceOwnSourceState)
	if err != nil {
		return nil, nil, err
	}
	for _, state := range states {
		if err := r.recheckTraceOwnState(ctx, state, sourceState.paths, sourceState.bound, sourceState.readFailed, sourceState.current); err != nil {
			return nil, nil, err
		}
	}
	return results, observations, nil
}

func (r *Repository) loadTraceOwnBatch(ctx context.Context, selections []TraceOwnBaseline) ([]traceOwnBaselineState, []LoadedTraceOrigin, error) {
	states := make([]traceOwnBaselineState, 0, len(selections))
	origins := make([]LoadedTraceOrigin, 0, len(selections))
	for _, selection := range selections {
		state, err := r.loadTraceOwnBaseline(ctx, selection)
		if err != nil {
			return nil, nil, err
		}
		states = append(states, state)
		if state.origin == nil {
			origins = append(origins, LoadedTraceOrigin{})
			continue
		}
		origin, err := r.LoadOrigin(ctx, *state.origin, state.content)
		if err != nil {
			return nil, nil, err
		}
		origins = append(origins, origin)
	}
	return states, origins, nil
}

func (r *Repository) observeTraceOwnBatch(origins []LoadedTraceOrigin) (traceOwnBatchSourceState, []TraceOwnSourceObservation, error) {
	state := newTraceOwnBatchSourceState()
	for _, origin := range origins {
		loaded, err := r.loadTraceOwnSources(origin)
		if err != nil {
			return state, nil, err
		}
		for key, source := range loaded.sources {
			state.sources[key] = source
		}
		for index, run := range origin.Map.Runs {
			if run.Kind != "external" {
				continue
			}
			source, ok := loaded.sources[run.Snapshot.String()]
			if !ok {
				return state, nil, fmt.Errorf("origin run %d references missing SourceSnapshot %s", index, run.Snapshot)
			}
			state.keys[source.Snapshot.SourceKey] = true
		}
	}
	keys := make([]string, 0, len(state.keys))
	for key := range state.keys {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	observations := make([]TraceOwnSourceObservation, 0, len(keys))
	for _, key := range keys {
		if err := r.observeTraceOwnSource(key, &state.traceOwnSourceState); err != nil {
			return state, nil, err
		}
		observations = append(observations, makeTraceOwnSourceObservation(key, time.Now().UTC(), state.traceOwnSourceState))
	}
	return state, observations, nil
}

func compareTraceOwnBatch(ctx context.Context, states []traceOwnBaselineState, origins []LoadedTraceOrigin, sourceState traceOwnSourceState) ([]TraceCompareOwnResult, error) {
	results := make([]TraceCompareOwnResult, 0, len(states))
	for index, state := range states {
		result := TraceCompareOwnResult{REF: state.selection.REF, Baseline: state.selection, Content: state.content, Origin: state.origin, SealID: state.sealID, CandidateDigest: state.candidateDigest}
		for mapIndex, run := range origins[index].Map.Runs {
			if run.Kind != "external" {
				continue
			}
			item, err := compareTraceOwnRun(ctx, mapIndex, run, sourceState)
			if err != nil {
				return nil, err
			}
			result.Runs = append(result.Runs, item)
		}
		results = append(results, result)
	}
	return results, nil
}

type traceOwnBatchSourceState struct {
	traceOwnSourceState
	keys map[string]bool
}

func newTraceOwnBatchSourceState() traceOwnBatchSourceState {
	return traceOwnBatchSourceState{traceOwnSourceState: traceOwnSourceState{
		sources: make(map[string]LoadedTraceSource), paths: make(map[string]string), bound: make(map[string]bool),
		readFailed: make(map[string]bool), sourceError: make(map[string]string), current: make(map[string][]byte),
	}, keys: make(map[string]bool)}
}

func makeTraceOwnSourceObservation(key string, observedAt time.Time, state traceOwnSourceState) TraceOwnSourceObservation {
	observation := TraceOwnSourceObservation{SourceKey: key, SourceIdentity: key, Bound: state.bound[key], ReadFailed: state.readFailed[key], ObservedAt: observedAt}
	if path := state.paths[key]; path != "" {
		observation.Path = &path
	}
	if sourceError := state.sourceError[key]; sourceError != "" {
		observation.Error = &sourceError
	}
	if data, ok := state.current[key]; ok {
		id := domain.ComputeNativeBlobID(data)
		length := len(data)
		observation.CurrentBlobID, observation.ByteLength = &id, &length
	}
	return observation
}
