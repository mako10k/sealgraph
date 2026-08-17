package history

import (
	"context"
	"fmt"
	"sort"

	"github.com/mako10k/sealgraph/internal/domain"
)

type LinkChangeKind string

const (
	LinkAdd     LinkChangeKind = "ADD"
	LinkRemove  LinkChangeKind = "REMOVE"
	LinkRepoint LinkChangeKind = "REPOINT"
	LinkMessage LinkChangeKind = "MESSAGE_CHANGE"
)

// LinkChange is a semantic dependency change from an older seal state to a
// newer seal state. Before and After are concrete seal identities.
type LinkChange struct {
	Kind          LinkChangeKind
	TargetSeal    domain.ObjectID
	BeforeSeal    *domain.ObjectID
	AfterSeal     *domain.ObjectID
	BeforeMessage string
	AfterMessage  string
}

// LinkLogEntry records the link transition from an entry's parent to that
// entry. Entries with no changes are retained so a no-op link generation is
// visible rather than silently omitted.
type LinkLogEntry struct {
	Entry   Entry
	Changes []LinkChange
}

// DiffLinks derives exact-target add, remove, and message events in
// deterministic SealID order. Ancestry-based repoint presentation belongs to
// the revision-graph slice.
func DiffLinks(before, after []domain.Link) []LinkChange {
	type linkKey string
	beforeByKey := make(map[linkKey]domain.Link, len(before))
	afterByKey := make(map[linkKey]domain.Link, len(after))
	keys := make(map[linkKey]bool, len(before)+len(after))
	for _, link := range before {
		key := linkKey(link.TargetSeal.String())
		beforeByKey[key] = link
		keys[key] = true
	}
	for _, link := range after {
		key := linkKey(link.TargetSeal.String())
		afterByKey[key] = link
		keys[key] = true
	}
	ordered := make([]linkKey, 0, len(keys))
	for key := range keys {
		ordered = append(ordered, key)
	}
	sort.Slice(ordered, func(i, j int) bool { return ordered[i] < ordered[j] })

	var changes []LinkChange
	for _, key := range ordered {
		oldLink, hadBefore := beforeByKey[key]
		newLink, hasAfter := afterByKey[key]
		switch {
		case !hadBefore && hasAfter:
			afterID := newLink.TargetSeal
			changes = append(changes, LinkChange{Kind: LinkAdd, TargetSeal: afterID, AfterSeal: &afterID, AfterMessage: newLink.Message})
		case hadBefore && !hasAfter:
			beforeID := oldLink.TargetSeal
			changes = append(changes, LinkChange{Kind: LinkRemove, TargetSeal: beforeID, BeforeSeal: &beforeID, BeforeMessage: oldLink.Message})
		default:
			if oldLink.Message != newLink.Message {
				changes = append(changes, LinkChange{Kind: LinkMessage, TargetSeal: oldLink.TargetSeal, BeforeMessage: oldLink.Message, AfterMessage: newLink.Message})
			}
		}
	}
	return changes
}

// DeriveLinkLog compares each generation with its parent. Unambiguous
// ancestry-related remove/add pairs are presented as repoints. The optional
// target filter matches one exact full target SealID on either side.
func DeriveLinkLog(ctx context.Context, entries []Entry, target string, load LoadSealFunc) ([]LinkLogEntry, error) {
	result := make([]LinkLogEntry, 0, len(entries))
	for i, entry := range entries {
		var parentLinks []domain.Link
		if i+1 < len(entries) {
			parentLinks = entries[i+1].Payload.Links
		}
		changes, err := withRepoints(ctx, DiffLinks(parentLinks, entry.Payload.Links), load)
		if err != nil {
			return nil, fmt.Errorf("derive Link changes at %s: %w", entry.ID, err)
		}
		if target != "" {
			filtered := changes[:0]
			for _, change := range changes {
				if change.TargetSeal.String() == target || optionalIDString(change.BeforeSeal) == target || optionalIDString(change.AfterSeal) == target {
					filtered = append(filtered, change)
				}
			}
			changes = filtered
		}
		result = append(result, LinkLogEntry{Entry: entry, Changes: changes})
	}
	return result, nil
}

func withRepoints(ctx context.Context, changes []LinkChange, load LoadSealFunc) ([]LinkChange, error) {
	removeIndexes, addIndexes := changeIndexes(changes)
	removeMatches := make(map[int][]int)
	addMatches := make(map[int][]int)
	for _, removeIndex := range removeIndexes {
		for _, addIndex := range addIndexes {
			related, err := isAncestor(ctx, *changes[removeIndex].BeforeSeal, *changes[addIndex].AfterSeal, load)
			if err != nil {
				return nil, err
			}
			if related {
				removeMatches[removeIndex] = append(removeMatches[removeIndex], addIndex)
				addMatches[addIndex] = append(addMatches[addIndex], removeIndex)
			}
		}
	}
	consumed := make(map[int]bool)
	result := make([]LinkChange, 0, len(changes))
	for _, removeIndex := range removeIndexes {
		matches := removeMatches[removeIndex]
		if len(matches) != 1 || len(addMatches[matches[0]]) != 1 {
			continue
		}
		addIndex := matches[0]
		before, after := changes[removeIndex], changes[addIndex]
		consumed[removeIndex], consumed[addIndex] = true, true
		result = append(result, LinkChange{
			Kind: LinkRepoint, TargetSeal: *after.AfterSeal,
			BeforeSeal: before.BeforeSeal, AfterSeal: after.AfterSeal,
			BeforeMessage: before.BeforeMessage, AfterMessage: after.AfterMessage,
		})
	}
	for index, change := range changes {
		if !consumed[index] {
			result = append(result, change)
		}
	}
	sort.Slice(result, func(i, j int) bool { return linkChangeKey(result[i]) < linkChangeKey(result[j]) })
	return result, nil
}

func changeIndexes(changes []LinkChange) ([]int, []int) {
	var removes, adds []int
	for index, change := range changes {
		switch change.Kind {
		case LinkRemove:
			removes = append(removes, index)
		case LinkAdd:
			adds = append(adds, index)
		}
	}
	return removes, adds
}

func isAncestor(ctx context.Context, ancestor, descendant domain.ObjectID, load LoadSealFunc) (bool, error) {
	seen := make(map[string]bool)
	current := descendant
	for {
		if current.Equal(ancestor) {
			return true, nil
		}
		if seen[current.String()] {
			return false, fmt.Errorf("parent_revision cycle reaches %s", current)
		}
		seen[current.String()] = true
		payload, err := load(ctx, current)
		if err != nil {
			return false, err
		}
		if payload.ParentRevision == nil {
			return false, nil
		}
		current = *payload.ParentRevision
	}
}

func optionalIDString(id *domain.ObjectID) string {
	if id == nil {
		return ""
	}
	return id.String()
}

func linkChangeKey(change LinkChange) string {
	return string(change.Kind) + "\x00" + optionalIDString(change.BeforeSeal) + "\x00" + optionalIDString(change.AfterSeal) + "\x00" + change.TargetSeal.String()
}
