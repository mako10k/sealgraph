package repository

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"sort"

	"github.com/mako10k/sealgraph/internal/domain"
	"github.com/mako10k/sealgraph/internal/recovery"
	"github.com/mako10k/sealgraph/internal/store"
)

type RecoveryInspection struct {
	ID          string
	Kind        string
	Journal     recovery.State
	Status      string
	Transitions []RecoveryTransitionInspection
	Corrupt     string
}

type RecoveryTransitionInspection struct {
	REF     string
	Current string
}

type RecoveryResult struct {
	ID   string
	Kind string
}

type DropREFResult struct {
	REF         string
	Head        domain.ObjectID
	Tags        int
	OperationID string
}

func (r *Repository) DropREF(ctx context.Context, ref string) (DropREFResult, error) {
	return withMutation(ctx, r.writer, "drop REF", func() (DropREFResult, error) {
		if err := domain.ValidateREF(ref); err != nil {
			return DropREFResult{}, err
		}
		if _, err := r.candidates.LoadSnapshot(ref); err == nil {
			return DropREFResult{}, fmt.Errorf("candidate %s blocks REF drop; seal or discard it explicitly before retrying", ref)
		} else if !errors.Is(err, ErrCandidateNotFound) {
			return DropREFResult{}, fmt.Errorf("inspect candidate %s before REF drop: %w", ref, err)
		}
		if binding, _, err := r.sources.load(ref); err == nil {
			return DropREFResult{}, fmt.Errorf("local source binding %s -> %q blocks REF drop; inspect it, then unbind explicitly before retrying", ref, binding.Path)
		} else if !errors.Is(err, ErrSourceNotFound) {
			return DropREFResult{}, fmt.Errorf("inspect local source %s before REF drop: %w", ref, err)
		}
		head, err := r.refs.Resolve(ctx, ref)
		if err != nil {
			return DropREFResult{}, err
		}
		tags, err := r.Tags(ctx, ref)
		if err != nil {
			return DropREFResult{}, err
		}
		refs, err := r.recoveryRefs()
		if err != nil {
			return DropREFResult{}, err
		}
		before, err := refs.Snapshot(ctx, ref)
		if err != nil {
			return DropREFResult{}, err
		}
		record, err := r.prepareRecovery("ref-drop", []recovery.Transition{{REF: ref, Before: before, After: nil}})
		if err != nil {
			return DropREFResult{}, err
		}
		if err := refs.ReplaceExact(ctx, ref, before, nil); err != nil {
			return DropREFResult{}, err
		}
		result := DropREFResult{REF: ref, Head: head, Tags: len(tags), OperationID: record.ID}
		if err := r.commitRecovery(record); err != nil {
			return result, fmt.Errorf("REF %s was dropped but recovery record %s could not be marked COMMITTED: %w", ref, record.ID, err)
		}
		return result, nil
	})
}

func (r *Repository) recoveryRefs() (store.RecoveryRefStore, error) {
	refs, ok := r.refs.(store.RecoveryRefStore)
	if !ok {
		return nil, fmt.Errorf("local REF recovery is unsupported by this repository backend")
	}
	return refs, nil
}

func (r *Repository) prepareRecovery(kind string, transitions []recovery.Transition) (recovery.Record, error) {
	sort.Slice(transitions, func(i, j int) bool { return transitions[i].REF < transitions[j].REF })
	return r.recovery.Prepare(kind, transitions)
}

func (r *Repository) commitRecovery(record recovery.Record) error {
	_, err := r.recovery.Commit(record)
	return err
}

func (r *Repository) RecoveryShow(ctx context.Context, id string) ([]RecoveryInspection, error) {
	if id != "" {
		record, err := r.recovery.Load(id)
		if err != nil {
			return nil, err
		}
		inspection, err := r.inspectRecoveryRecord(ctx, record)
		return []RecoveryInspection{inspection}, err
	}
	entries, err := r.recovery.List()
	if err != nil {
		return nil, err
	}
	result := make([]RecoveryInspection, 0, len(entries))
	for _, entry := range entries {
		if entry.Err != nil {
			result = append(result, RecoveryInspection{ID: entry.ID, Status: "CORRUPT", Corrupt: entry.Err.Error()})
			continue
		}
		inspection, inspectErr := r.inspectRecoveryRecord(ctx, *entry.Record)
		if inspectErr != nil {
			result = append(result, RecoveryInspection{ID: entry.ID, Kind: entry.Record.Kind, Journal: entry.Record.State, Status: "CORRUPT", Corrupt: inspectErr.Error()})
			continue
		}
		result = append(result, inspection)
	}
	return result, nil
}

func (r *Repository) inspectRecoveryRecord(ctx context.Context, record recovery.Record) (RecoveryInspection, error) {
	refs, err := r.recoveryRefs()
	if err != nil {
		return RecoveryInspection{}, err
	}
	inspection := RecoveryInspection{ID: record.ID, Kind: record.Kind, Journal: record.State}
	allBefore, allAfter := true, true
	for _, transition := range record.Transitions {
		for _, state := range [][]byte{transition.Before, transition.After} {
			if state == nil {
				continue
			}
			targets, err := refs.ManifestTargets(state)
			if err != nil {
				return RecoveryInspection{}, fmt.Errorf("recorded REF %s manifest is invalid: %w", transition.REF, err)
			}
			for _, target := range targets {
				if _, err := r.LoadSeal(ctx, target); err != nil {
					return RecoveryInspection{}, fmt.Errorf("recorded REF %s target %s is not a canonical Seal: %w", transition.REF, target, err)
				}
			}
		}
		current, err := refs.Snapshot(ctx, transition.REF)
		if err != nil {
			return RecoveryInspection{}, err
		}
		state := "INTERVENED"
		if bytes.Equal(current, transition.Before) {
			state = "BEFORE"
			allAfter = false
		} else if bytes.Equal(current, transition.After) {
			state = "AFTER"
			allBefore = false
		} else {
			allBefore, allAfter = false, false
		}
		inspection.Transitions = append(inspection.Transitions, RecoveryTransitionInspection{REF: transition.REF, Current: state})
	}
	switch {
	case allAfter && record.State == recovery.Prepared:
		inspection.Status = "PREPARED_APPLIED_RECOVERABLE"
	case allAfter:
		inspection.Status = "RECOVERABLE"
	case allBefore && record.State == recovery.Prepared:
		inspection.Status = "PREPARED_NOT_APPLIED"
	case allBefore:
		inspection.Status = "ALREADY_RECOVERED"
	default:
		inspection.Status = "INTERVENED"
	}
	return inspection, nil
}

func (r *Repository) Recover(ctx context.Context, id string) (RecoveryResult, error) {
	return withMutation(ctx, r.writer, "recover REF operation", func() (RecoveryResult, error) {
		record, err := r.recovery.Load(id)
		if err != nil {
			return RecoveryResult{}, err
		}
		inspection, err := r.inspectRecoveryRecord(ctx, record)
		if err != nil {
			return RecoveryResult{}, err
		}
		if inspection.Status != "RECOVERABLE" && inspection.Status != "PREPARED_APPLIED_RECOVERABLE" {
			return RecoveryResult{}, fmt.Errorf("operation %s is %s, not recoverable; inspect current sealed state and `sealgraph recover show %s`", id, inspection.Status, id)
		}
		refs, err := r.recoveryRefs()
		if err != nil {
			return RecoveryResult{}, err
		}
		if record.Kind == "mv" {
			if err := recoverMove(ctx, refs, record.Transitions); err != nil {
				return RecoveryResult{}, err
			}
		} else {
			transition := record.Transitions[0]
			if err := refs.ReplaceExact(ctx, transition.REF, transition.After, transition.Before); err != nil {
				return RecoveryResult{}, err
			}
		}
		return RecoveryResult{ID: id, Kind: record.Kind}, nil
	})
}

func recoverMove(ctx context.Context, refs store.RecoveryRefStore, transitions []recovery.Transition) error {
	if len(transitions) != 2 {
		return fmt.Errorf("mv recovery requires exactly two REF transitions")
	}
	var oldTransition, newTransition *recovery.Transition
	for i := range transitions {
		transition := &transitions[i]
		if transition.Before != nil && transition.After == nil {
			oldTransition = transition
		} else if transition.Before == nil && transition.After != nil {
			newTransition = transition
		}
	}
	if oldTransition == nil || newTransition == nil {
		return fmt.Errorf("mv recovery record does not describe one exact rename")
	}
	return refs.MoveExact(ctx, newTransition.REF, oldTransition.REF, newTransition.After, oldTransition.After)
}
