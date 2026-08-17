package history

import (
	"sort"

	"github.com/mako10k/sealgraph/internal/domain"
)

type LinkChangeKind string

const (
	LinkAdd     LinkChangeKind = "ADD"
	LinkRemove  LinkChangeKind = "REMOVE"
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

// DeriveLinkLog compares each generation with its parent. The optional
// target filter matches one exact full target SealID.
func DeriveLinkLog(entries []Entry, target string) []LinkLogEntry {
	result := make([]LinkLogEntry, 0, len(entries))
	for i, entry := range entries {
		var parentLinks []domain.Link
		if i+1 < len(entries) {
			parentLinks = entries[i+1].Payload.Links
		}
		changes := DiffLinks(parentLinks, entry.Payload.Links)
		if target != "" {
			filtered := changes[:0]
			for _, change := range changes {
				if change.TargetSeal.String() == target {
					filtered = append(filtered, change)
				}
			}
			changes = filtered
		}
		result = append(result, LinkLogEntry{Entry: entry, Changes: changes})
	}
	return result
}
