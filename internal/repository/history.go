package repository

import (
	"context"
	"fmt"

	"github.com/mako10k/sealgraph/internal/domain"
	"github.com/mako10k/sealgraph/internal/history"
)

// Log returns the complete immutable parent history for the current head of
// one logical REF, newest first.
func (r *Repository) Log(ctx context.Context, ref string) ([]history.Entry, error) {
	if err := domain.ValidateREF(ref); err != nil {
		return nil, err
	}
	head, err := r.refs.Resolve(ctx, ref)
	if err != nil {
		return nil, fmt.Errorf("resolve history head for %s: %w; seal that REF first or select an existing REF", ref, err)
	}
	entries, err := history.Walk(ctx, ref, head, r.LoadSeal)
	if err != nil {
		return nil, fmt.Errorf("read immutable history for %s: %w; inspect the named seal objects and REF head explicitly before retrying", ref, err)
	}
	return entries, nil
}

// LinkLog derives dependency changes from the validated parent history. An
// upstream filter matches an exact logical target REF.
func (r *Repository) LinkLog(ctx context.Context, ref, upstream string) ([]history.LinkLogEntry, error) {
	if upstream != "" {
		if err := domain.ValidateREF(upstream); err != nil {
			return nil, fmt.Errorf("invalid upstream REF %q: %w", upstream, err)
		}
	}
	entries, err := r.Log(ctx, ref)
	if err != nil {
		return nil, err
	}
	return history.DeriveLinkLog(entries, upstream), nil
}

// DiffCurrent compares the current head of ref with its immutable parent.
func (r *Repository) DiffCurrent(ctx context.Context, ref string) (history.SealDiff, error) {
	if err := domain.ValidateREF(ref); err != nil {
		return history.SealDiff{}, err
	}
	head, err := r.refs.Resolve(ctx, ref)
	if err != nil {
		return history.SealDiff{}, fmt.Errorf("resolve current head for diff %s: %w; seal that REF first or select an existing REF", ref, err)
	}
	current, err := r.LoadSeal(ctx, head)
	if err != nil {
		return history.SealDiff{}, fmt.Errorf("load current generation %s@%s: %w", ref, head, err)
	}
	if current.REF != ref {
		return history.SealDiff{}, fmt.Errorf("current seal %s belongs to REF %s, not %s; repair the REF explicitly", head, current.REF, ref)
	}
	if current.Parent == nil {
		return history.SealDiff{}, fmt.Errorf("current seal %s@%s has no parent to compare; provide two explicit REF@SEAL generations after the REF is superseded", ref, head)
	}
	return r.DiffExact(ctx, ref, *current.Parent, head)
}

// DiffExact compares two exact canonical generations owned by one logical REF.
func (r *Repository) DiffExact(ctx context.Context, ref string, fromID, toID domain.ObjectID) (history.SealDiff, error) {
	if err := domain.ValidateREF(ref); err != nil {
		return history.SealDiff{}, err
	}
	from, err := r.LoadSeal(ctx, fromID)
	if err != nil {
		return history.SealDiff{}, fmt.Errorf("load older diff generation %s@%s: %w; inspect that immutable object explicitly", ref, fromID, err)
	}
	if from.REF != ref {
		return history.SealDiff{}, fmt.Errorf("older seal %s belongs to REF %s, not %s; select two generations owned by one logical REF", fromID, from.REF, ref)
	}
	to, err := r.LoadSeal(ctx, toID)
	if err != nil {
		return history.SealDiff{}, fmt.Errorf("load newer diff generation %s@%s: %w; inspect that immutable object explicitly", ref, toID, err)
	}
	if to.REF != ref {
		return history.SealDiff{}, fmt.Errorf("newer seal %s belongs to REF %s, not %s; select two generations owned by one logical REF", toID, to.REF, ref)
	}
	return history.DiffSeals(fromID, from, toID, to)
}
