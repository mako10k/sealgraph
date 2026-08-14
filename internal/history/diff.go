package history

import (
	"fmt"
	"sort"

	"github.com/mako10k/sealgraph/internal/domain"
)

// ValueChange compares a scalar canonical field.
type ValueChange[T comparable] struct {
	Before  T
	After   T
	Changed bool
}

type ParentChange struct {
	Before  *domain.ObjectID
	After   *domain.ObjectID
	Changed bool
}

type AttachmentChangeKind string

const (
	AttachmentAdd    AttachmentChangeKind = "ADD"
	AttachmentRemove AttachmentChangeKind = "REMOVE"
	AttachmentChange AttachmentChangeKind = "CHANGE"
)

type AttachmentChangeRecord struct {
	Kind   AttachmentChangeKind
	Name   string
	Before *domain.Attachment
	After  *domain.Attachment
}

// SealDiff is a semantic comparison from one exact generation to another.
// Empty attachment/link change slices mean those canonical sets are unchanged.
type SealDiff struct {
	REF         string
	From        domain.ObjectID
	To          domain.ObjectID
	Content     ValueChange[domain.ContentRef]
	Attachments []AttachmentChangeRecord
	Links       []LinkChange
	Message     ValueChange[string]
	Root        ValueChange[bool]
	Draft       ValueChange[bool]
	Parent      ParentChange
	CreatedAt   ValueChange[string]
}

func valueChange[T comparable](before, after T) ValueChange[T] {
	return ValueChange[T]{Before: before, After: after, Changed: before != after}
}

func parentChange(before, after *domain.ObjectID) ParentChange {
	result := ParentChange{Before: copyObjectID(before), After: copyObjectID(after)}
	switch {
	case before == nil && after == nil:
	case before == nil || after == nil:
		result.Changed = true
	default:
		result.Changed = !before.Equal(*after)
	}
	return result
}

func copyObjectID(id *domain.ObjectID) *domain.ObjectID {
	if id == nil {
		return nil
	}
	copy := *id
	return &copy
}

// DiffSeals compares all material canonical fields between two generations of
// the same logical REF. Schema equality follows from canonical v2 decoding.
func DiffSeals(fromID domain.ObjectID, from domain.SealPayload, toID domain.ObjectID, to domain.SealPayload) (SealDiff, error) {
	if from.REF != to.REF {
		return SealDiff{}, fmt.Errorf("cannot compare seal generations from different REFs: %s and %s", from.REF, to.REF)
	}
	return SealDiff{
		REF:         from.REF,
		From:        fromID,
		To:          toID,
		Content:     valueChange(from.Content, to.Content),
		Attachments: diffAttachments(from.Attachments, to.Attachments),
		Links:       DiffLinks(from.Links, to.Links),
		Message:     valueChange(from.Message, to.Message),
		Root:        valueChange(from.Root, to.Root),
		Draft:       valueChange(from.Draft, to.Draft),
		Parent:      parentChange(from.Parent, to.Parent),
		CreatedAt:   valueChange(from.CreatedAt, to.CreatedAt),
	}, nil
}

func diffAttachments(before, after []domain.Attachment) []AttachmentChangeRecord {
	beforeByName := make(map[string]domain.Attachment, len(before))
	afterByName := make(map[string]domain.Attachment, len(after))
	names := make(map[string]bool, len(before)+len(after))
	for _, attachment := range before {
		beforeByName[attachment.Name] = attachment
		names[attachment.Name] = true
	}
	for _, attachment := range after {
		afterByName[attachment.Name] = attachment
		names[attachment.Name] = true
	}
	ordered := make([]string, 0, len(names))
	for name := range names {
		ordered = append(ordered, name)
	}
	sort.Strings(ordered)

	var changes []AttachmentChangeRecord
	for _, name := range ordered {
		oldAttachment, hadBefore := beforeByName[name]
		newAttachment, hasAfter := afterByName[name]
		switch {
		case !hadBefore && hasAfter:
			afterCopy := newAttachment
			changes = append(changes, AttachmentChangeRecord{Kind: AttachmentAdd, Name: name, After: &afterCopy})
		case hadBefore && !hasAfter:
			beforeCopy := oldAttachment
			changes = append(changes, AttachmentChangeRecord{Kind: AttachmentRemove, Name: name, Before: &beforeCopy})
		case oldAttachment != newAttachment:
			beforeCopy, afterCopy := oldAttachment, newAttachment
			changes = append(changes, AttachmentChangeRecord{Kind: AttachmentChange, Name: name, Before: &beforeCopy, After: &afterCopy})
		}
	}
	return changes
}
