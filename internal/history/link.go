package history

import (
	"sort"

	"github.com/mako10k/sealgraph/internal/domain"
)

type LinkChangeKind string

const (
	LinkAdd     LinkChangeKind = "ADD"
	LinkRemove  LinkChangeKind = "REMOVE"
	LinkRepoint LinkChangeKind = "REPOINT"
)

// LinkChange is a semantic dependency change from an older seal state to a
// newer seal state. Before and After are concrete seal identities.
type LinkChange struct {
	Kind       LinkChangeKind
	Relation   string
	TargetREF  string
	BeforeSeal *domain.ObjectID
	AfterSeal  *domain.ObjectID
}

// LinkLogEntry records the link transition from an entry's parent to that
// entry. Entries with no changes are retained so a no-op link generation is
// visible rather than silently omitted.
type LinkLogEntry struct {
	Entry   Entry
	Changes []LinkChange
}

// DiffLinks derives add, remove, and repoint events in deterministic
// relation/target-REF order.
func DiffLinks(before, after []domain.Link) []LinkChange {
	type linkKey struct {
		relation  string
		targetREF string
	}
	beforeByKey := make(map[linkKey]domain.Link, len(before))
	afterByKey := make(map[linkKey]domain.Link, len(after))
	keys := make(map[linkKey]bool, len(before)+len(after))
	for _, link := range before {
		key := linkKey{relation: link.Relation, targetREF: link.TargetREF}
		beforeByKey[key] = link
		keys[key] = true
	}
	for _, link := range after {
		key := linkKey{relation: link.Relation, targetREF: link.TargetREF}
		afterByKey[key] = link
		keys[key] = true
	}
	ordered := make([]linkKey, 0, len(keys))
	for key := range keys {
		ordered = append(ordered, key)
	}
	sort.Slice(ordered, func(i, j int) bool {
		if ordered[i].relation != ordered[j].relation {
			return ordered[i].relation < ordered[j].relation
		}
		return ordered[i].targetREF < ordered[j].targetREF
	})

	var changes []LinkChange
	for _, key := range ordered {
		oldLink, hadBefore := beforeByKey[key]
		newLink, hasAfter := afterByKey[key]
		switch {
		case !hadBefore && hasAfter:
			afterID := newLink.TargetSeal
			changes = append(changes, LinkChange{Kind: LinkAdd, Relation: key.relation, TargetREF: key.targetREF, AfterSeal: &afterID})
		case hadBefore && !hasAfter:
			beforeID := oldLink.TargetSeal
			changes = append(changes, LinkChange{Kind: LinkRemove, Relation: key.relation, TargetREF: key.targetREF, BeforeSeal: &beforeID})
		case !oldLink.TargetSeal.Equal(newLink.TargetSeal):
			beforeID, afterID := oldLink.TargetSeal, newLink.TargetSeal
			changes = append(changes, LinkChange{Kind: LinkRepoint, Relation: key.relation, TargetREF: key.targetREF, BeforeSeal: &beforeID, AfterSeal: &afterID})
		}
	}
	return changes
}

// DeriveLinkLog compares each generation with its parent. The optional
// upstream filter matches one exact logical target REF.
func DeriveLinkLog(entries []Entry, upstream string) []LinkLogEntry {
	result := make([]LinkLogEntry, 0, len(entries))
	for i, entry := range entries {
		var parentLinks []domain.Link
		if i+1 < len(entries) {
			parentLinks = entries[i+1].Payload.Links
		}
		changes := DiffLinks(parentLinks, entry.Payload.Links)
		if upstream != "" {
			filtered := changes[:0]
			for _, change := range changes {
				if change.TargetREF == upstream {
					filtered = append(filtered, change)
				}
			}
			changes = filtered
		}
		result = append(result, LinkLogEntry{Entry: entry, Changes: changes})
	}
	return result
}
