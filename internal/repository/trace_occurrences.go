package repository

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"strings"

	"github.com/mako10k/sealgraph/internal/domain"
	"github.com/mako10k/sealgraph/internal/workfile"
)

var ErrTraceOccurrenceBaselineChanged = errors.New("selected trace baseline changed")
var ErrTraceOccurrenceBindingChanged = errors.New("trace source binding changed")

// TraceOccurrenceInput fixes one External run and the exact bytes needed to
// derive positions. Private fields retain the observations for readback.
type TraceOccurrenceInput struct {
	Baseline        TraceOwnBaseline
	SealID          *domain.ObjectID
	CandidateDigest [32]byte
	OriginMapID     domain.ObjectID
	RunIndex        int
	SourceSnapshot  domain.ObjectID
	SourceKey       string
	SourceStart     uint64
	Length          uint64
	SnapshotBlobID  domain.ObjectID
	SnapshotBytes   []byte
	Pattern         []byte
	CurrentBlobID   *domain.ObjectID
	CurrentBytes    []byte
	BindingDigest   *[32]byte
	CurrentFailure  string
	path            string
	bound           bool
}

func (r *Repository) LoadTraceOccurrenceInput(ctx context.Context, baseline TraceOwnBaseline, runIndex int, currentView bool) (TraceOccurrenceInput, error) {
	state, err := r.loadTraceOwnBaseline(ctx, baseline)
	if err != nil {
		return TraceOccurrenceInput{}, err
	}
	if state.origin == nil {
		return TraceOccurrenceInput{}, errors.New("selected baseline has no Origin Trace")
	}
	origin, err := r.LoadOrigin(ctx, *state.origin, state.content)
	if err != nil {
		return TraceOccurrenceInput{}, err
	}
	if runIndex < 0 || runIndex >= len(origin.Map.Runs) || origin.Map.Runs[runIndex].Kind != "external" {
		return TraceOccurrenceInput{}, fmt.Errorf("run index %d is not an External run", runIndex)
	}
	run := origin.Map.Runs[runIndex]
	var source *LoadedTraceSource
	for i := range origin.Sources {
		if origin.Sources[i].SnapshotID.Equal(run.Snapshot) {
			source = &origin.Sources[i]
			break
		}
	}
	if source == nil || run.SourceStart > uint64(len(source.Content)) || run.Length == 0 || run.Length > uint64(len(source.Content))-run.SourceStart {
		return TraceOccurrenceInput{}, fmt.Errorf("External run %d has invalid SourceSnapshot range", runIndex)
	}
	input := TraceOccurrenceInput{
		Baseline: baseline, SealID: state.sealID, CandidateDigest: state.candidateDigest,
		OriginMapID: origin.ID, RunIndex: runIndex, SourceSnapshot: run.Snapshot,
		SourceKey: source.Snapshot.SourceKey, SourceStart: run.SourceStart, Length: run.Length,
		SnapshotBlobID: source.Snapshot.Content, SnapshotBytes: source.Content,
		Pattern: source.Content[int(run.SourceStart):int(run.SourceStart+run.Length)],
	}
	if !currentView {
		return input, nil
	}
	binding, bindingBytes, _, err := r.traceSourceLoad(input.SourceKey)
	if errors.Is(err, ErrTraceSourceNotFound) {
		input.CurrentFailure = "SOURCE_READ_FAILED"
		return input, nil
	}
	if err != nil {
		return TraceOccurrenceInput{}, err
	}
	input.path, input.bound = binding.Path, true
	digest := sha256.Sum256(bindingBytes)
	input.BindingDigest = &digest
	current, err := workfile.ReadStable(r.workDir, binding.Path)
	if err != nil {
		input.CurrentFailure = "SOURCE_READ_FAILED"
		if strings.Contains(err.Error(), "CHANGED_DURING_READ") {
			input.CurrentFailure = "OBSERVATION_UNSTABLE"
		}
		return input, nil
	}
	input.CurrentBytes = current
	id := domain.ComputeNativeBlobID(current)
	input.CurrentBlobID = &id
	return input, nil
}

// RevalidateTraceOccurrenceInput detects selected-baseline, binding and
// current-byte changes before a page is emitted. Unselected baselines are not
// part of the observation context.
func (r *Repository) RevalidateTraceOccurrenceInput(ctx context.Context, input TraceOccurrenceInput) error {
	if err := r.revalidateTraceOccurrenceBaseline(ctx, input); err != nil {
		return err
	}
	return r.revalidateTraceOccurrenceSource(input)
}

func (r *Repository) revalidateTraceOccurrenceBaseline(ctx context.Context, input TraceOccurrenceInput) error {
	switch input.Baseline.Kind {
	case TraceOwnCandidate:
		digest, err := r.CandidateExactDigest(ctx, input.Baseline.REF)
		if err != nil || digest != input.CandidateDigest {
			return ErrTraceOccurrenceBaselineChanged
		}
	case TraceOwnHead:
		head, err := r.CurrentREFHead(ctx, input.Baseline.REF)
		if err != nil || head == nil || input.SealID == nil || !head.Equal(*input.SealID) {
			return ErrTraceOccurrenceBaselineChanged
		}
	}
	return nil
}

func (r *Repository) revalidateTraceOccurrenceSource(input TraceOccurrenceInput) error {
	if !input.bound {
		if input.CurrentFailure == "" {
			return nil
		}
		if _, err := r.TraceSourceShow(input.SourceKey); errors.Is(err, ErrTraceSourceNotFound) {
			return nil
		} else if err != nil {
			return err
		}
		return ErrTraceOccurrenceBindingChanged
	}
	binding, exact, _, err := r.traceSourceLoad(input.SourceKey)
	if err != nil || binding.Path != input.path || input.BindingDigest == nil || sha256.Sum256(exact) != *input.BindingDigest {
		return ErrTraceOccurrenceBindingChanged
	}
	current, err := workfile.ReadStable(r.workDir, binding.Path)
	if input.CurrentFailure != "" {
		if err == nil {
			return errors.New("source became readable during occurrence listing")
		}
		return nil
	}
	if err != nil || !bytes.Equal(current, input.CurrentBytes) {
		return errors.New("current source changed during occurrence listing")
	}
	return nil
}
