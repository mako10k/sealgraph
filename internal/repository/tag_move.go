package repository

import (
	"bytes"
	"context"
	"errors"
	"fmt"

	"github.com/mako10k/sealgraph/internal/domain"
	"github.com/mako10k/sealgraph/internal/recovery"
	"github.com/mako10k/sealgraph/internal/store"
)

type TagResult struct {
	REF         string
	Name        string
	Seal        domain.ObjectID
	OperationID string
}

type MoveResult struct {
	OldREF      string
	NewREF      string
	Head        domain.ObjectID
	Tags        int
	OperationID string
}

func (r *Repository) CreateTag(ctx context.Context, selectorText, name string) (TagResult, error) {
	return withMutation(ctx, r.writer, "create immutable tag", func() (TagResult, error) {
		if err := domain.ValidateTagName(name); err != nil {
			return TagResult{}, err
		}
		selector, err := ParseSelector(selectorText)
		if err != nil {
			return TagResult{}, err
		}
		if selector.Kind == SelectorGlobalSeal {
			return TagResult{}, fmt.Errorf("tag target %q has no REF scope; use REF, REF@SEAL, or REF@TAG", selectorText)
		}
		resolved, err := r.ResolveSelector(ctx, selectorText)
		if err != nil {
			return TagResult{}, err
		}
		head, err := r.refs.Resolve(ctx, selector.REF)
		if err != nil {
			return TagResult{}, fmt.Errorf("resolve tag scope %s HEAD: %w", selector.REF, err)
		}
		recoveryRefs, err := r.recoveryRefs()
		if err != nil {
			return TagResult{}, err
		}
		before, err := recoveryRefs.Snapshot(ctx, selector.REF)
		if err != nil {
			return TagResult{}, err
		}
		after, err := recoveryRefs.PreviewTag(ctx, selector.REF, name, resolved.ID, head)
		if err != nil {
			return TagResult{}, err
		}
		if bytes.Equal(before, after) {
			return TagResult{REF: selector.REF, Name: name, Seal: resolved.ID}, nil
		}
		record, err := r.prepareRecovery("tag", []recovery.Transition{{REF: selector.REF, Before: before, After: after}})
		if err != nil {
			return TagResult{}, err
		}
		if err := r.tags.Create(ctx, selector.REF, name, resolved.ID, head); err != nil {
			return TagResult{}, err
		}
		result := TagResult{REF: selector.REF, Name: name, Seal: resolved.ID, OperationID: record.ID}
		if err := r.commitRecovery(record); err != nil {
			return result, fmt.Errorf("tag was published but recovery record %s could not be marked COMMITTED: %w", record.ID, err)
		}
		return result, nil
	})
}

func (r *Repository) Tags(ctx context.Context, ref string) ([]store.Tag, error) {
	if err := domain.ValidateREF(ref); err != nil {
		return nil, err
	}
	head, err := r.refs.Resolve(ctx, ref)
	if err != nil {
		return nil, err
	}
	if _, err := r.LoadSeal(ctx, head); err != nil {
		return nil, fmt.Errorf("REF %s head %s is not a canonical Seal: %w", ref, head, err)
	}
	tags, err := r.tags.List(ctx, ref)
	if err != nil {
		return nil, err
	}
	for _, tag := range tags {
		if _, err := r.LoadSeal(ctx, tag.Seal); err != nil {
			return nil, fmt.Errorf("tag %s@%s target %s is not a canonical Seal: %w", ref, tag.Name, tag.Seal, err)
		}
	}
	return tags, nil
}

func (r *Repository) MoveREF(ctx context.Context, oldRef, newRef string) (MoveResult, error) {
	return withMutation(ctx, r.writer, "move REF", func() (MoveResult, error) {
		if err := validateMoveNames(oldRef, newRef); err != nil {
			return MoveResult{}, err
		}
		if err := validateMoveCandidates(r.candidates, oldRef, newRef); err != nil {
			return MoveResult{}, err
		}
		if err := validateMoveSources(r.sources, oldRef, newRef); err != nil {
			return MoveResult{}, err
		}
		head, err := r.refs.Resolve(ctx, oldRef)
		if err != nil {
			return MoveResult{}, fmt.Errorf("resolve move source %s: %w", oldRef, err)
		}
		if _, err := r.LoadSeal(ctx, head); err != nil {
			return MoveResult{}, fmt.Errorf("move source %s head %s is not a canonical Seal: %w", oldRef, head, err)
		}
		tags, err := r.Tags(ctx, oldRef)
		if err != nil {
			return MoveResult{}, err
		}
		recoveryRefs, err := r.recoveryRefs()
		if err != nil {
			return MoveResult{}, err
		}
		oldBefore, err := recoveryRefs.Snapshot(ctx, oldRef)
		if err != nil {
			return MoveResult{}, err
		}
		newBefore, err := recoveryRefs.Snapshot(ctx, newRef)
		if err != nil {
			return MoveResult{}, err
		}
		if newBefore != nil {
			return MoveResult{}, fmt.Errorf("move destination REF %s already exists", newRef)
		}
		record, err := r.prepareRecovery("mv", []recovery.Transition{{REF: oldRef, Before: oldBefore, After: nil}, {REF: newRef, Before: newBefore, After: oldBefore}})
		if err != nil {
			return MoveResult{}, err
		}
		if err := r.refs.Move(ctx, oldRef, newRef); err != nil {
			return MoveResult{}, err
		}
		result := MoveResult{OldREF: oldRef, NewREF: newRef, Head: head, Tags: len(tags), OperationID: record.ID}
		if err := r.commitRecovery(record); err != nil {
			return result, fmt.Errorf("REF move committed but recovery record %s could not be marked COMMITTED: %w", record.ID, err)
		}
		return result, nil
	})
}

func validateMoveSources(sources sourceStore, oldRef, newRef string) error {
	for _, ref := range []string{oldRef, newRef} {
		if binding, _, err := sources.load(ref); err == nil {
			return fmt.Errorf("local source binding %s -> %q blocks REF-only move; inspect it with 'sealgraph source show %s', then unbind explicitly before moving the REF", ref, binding.Path, ref)
		} else if !errors.Is(err, ErrSourceNotFound) {
			return fmt.Errorf("inspect local source %s before REF move: %w", ref, err)
		}
	}
	return nil
}

func validateMoveNames(oldRef, newRef string) error {
	if err := domain.ValidateREF(oldRef); err != nil {
		return fmt.Errorf("invalid source REF: %w", err)
	}
	if err := domain.ValidateREF(newRef); err != nil {
		return fmt.Errorf("invalid destination REF: %w", err)
	}
	if oldRef == newRef {
		return fmt.Errorf("source and destination REF are both %s; choose a different absent destination", oldRef)
	}
	return nil
}

func validateMoveCandidates(candidates candidateStore, oldRef, newRef string) error {
	for _, ref := range []string{oldRef, newRef} {
		if _, err := candidates.LoadSnapshot(ref); err == nil {
			return fmt.Errorf("candidate %s blocks REF move; seal or discard it explicitly before retrying", ref)
		} else if !errors.Is(err, ErrCandidateNotFound) {
			return fmt.Errorf("inspect candidate %s before REF move: %w", ref, err)
		}
	}
	return nil
}
