package repository

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"

	canonicalv6 "github.com/mako10k/sealgraph/internal/canonical/v6"
	domainv5 "github.com/mako10k/sealgraph/internal/domain/v5"
)

type LinkMetadataMutationResult struct {
	Action, REF, Namespace string
	TargetSeal             string
	Before, After          *domainv5.MetadataEntry
	Candidate              domainv5.Candidate
	Prospective            domainv5.ResolvedSeal
}

func (r *Repository) SetLinkMetadata(ctx context.Context, ref, target, namespace string, schema *string, value json.RawMessage) (LinkMetadataMutationResult, error) {
	entry := domainv5.MetadataEntry{Namespace: namespace, Schema: schema, Value: append([]byte(nil), value...)}
	normalized, err := canonicalv6.NormalizeMetadata([]domainv5.MetadataEntry{entry})
	if err != nil {
		return LinkMetadataMutationResult{}, err
	}
	return r.mutateLinkMetadata(ctx, "SET", ref, target, namespace, &normalized[0])
}

func (r *Repository) RemoveLinkMetadata(ctx context.Context, ref, target, namespace string) (LinkMetadataMutationResult, error) {
	return r.mutateLinkMetadata(ctx, "REMOVED", ref, target, namespace, nil)
}

func (r *Repository) mutateLinkMetadata(ctx context.Context, action, ref, target, namespace string, replacement *domainv5.MetadataEntry) (LinkMetadataMutationResult, error) {
	if r.format != 6 {
		return LinkMetadataMutationResult{}, errors.New("Link metadata mutation requires repository format 6")
	}
	if replacement == nil {
		if _, err := canonicalv6.NormalizeMetadata([]domainv5.MetadataEntry{{Namespace: namespace, Value: json.RawMessage("null")}}); err != nil {
			return LinkMetadataMutationResult{}, err
		}
	}
	return withMutation(ctx, r.writer, "mutate Link metadata", func() (LinkMetadataMutationResult, error) {
		return r.mutateLinkMetadataLocked(ctx, action, ref, target, namespace, replacement)
	})
}

func (r *Repository) mutateLinkMetadataLocked(ctx context.Context, action, ref, target, namespace string, replacement *domainv5.MetadataEntry) (LinkMetadataMutationResult, error) {
	observation, graph, err := r.buildObservation(ctx, "Link metadata mutation")
	if err != nil {
		return LinkMetadataMutationResult{}, err
	}
	resolved, err := r.resolveSelectorObserved(ctx, target, &observation, graph)
	if err != nil {
		return LinkMetadataMutationResult{}, fmt.Errorf("resolve metadata target %q: %w", target, err)
	}
	edit, err := r.candidateForObservedEdit(ctx, ref, observation, graph)
	if err != nil {
		return LinkMetadataMutationResult{}, err
	}
	if edit.Initial {
		return LinkMetadataMutationResult{}, fmt.Errorf("REF %s is absent; Link metadata requires an existing REF or Candidate baseline", ref)
	}
	if edit.Candidate.Root {
		return LinkMetadataMutationResult{}, fmt.Errorf("Candidate %s is root and has no Cause Link metadata", ref)
	}
	linkIndex := causeLinkIndex(edit.Candidate.CauseLinks, resolved.ID.String())
	if linkIndex < 0 {
		return LinkMetadataMutationResult{}, fmt.Errorf("Candidate %s has no Cause Link to resolved target %s; inspect the Candidate and use an exact selector", ref, resolved.ID)
	}
	candidate := edit.Candidate
	metadata := cloneMetadata(candidate.CauseLinks[linkIndex].Metadata)
	entryIndex := metadataIndex(metadata, namespace)
	var before *domainv5.MetadataEntry
	if entryIndex >= 0 {
		value := metadata[entryIndex]
		before = &value
	}
	if action == "REMOVED" && entryIndex < 0 {
		return LinkMetadataMutationResult{}, fmt.Errorf("metadata namespace %q is absent from target %s", namespace, resolved.ID)
	}
	metadata = replaceMetadata(metadata, entryIndex, replacement)
	normalized, err := canonicalv6.NormalizeMetadata(metadata)
	if err != nil {
		return LinkMetadataMutationResult{}, err
	}
	candidate.CauseLinks[linkIndex].Metadata = normalized
	prospective, err := r.prospectiveSeal(candidate)
	if err != nil {
		return LinkMetadataMutationResult{}, err
	}
	result := LinkMetadataMutationResult{Action: action, REF: ref, Namespace: namespace, TargetSeal: resolved.ID.String(), Before: before, After: cloneMetadataEntry(replacement), Candidate: candidate, Prospective: prospective}
	if replacement != nil && before != nil && metadataEntriesEqual(*before, *replacement) && edit.Candidate.Schema == "sealgraph/candidate/v6" {
		return result, nil
	}
	if err := r.validateCandidateMutation(ctx, candidate, observation); err != nil {
		return LinkMetadataMutationResult{}, err
	}
	if err := r.candidates.SaveIfUnchanged(candidate, edit.Bytes, edit.Exists); err != nil {
		return LinkMetadataMutationResult{}, err
	}
	result.Candidate, err = r.candidates.Load(ref)
	return result, err
}

func causeLinkIndex(links []domainv5.CauseLink, target string) int {
	for i, link := range links {
		if link.TargetSeal.String() == target {
			return i
		}
	}
	return -1
}

func metadataIndex(entries []domainv5.MetadataEntry, namespace string) int {
	for i, entry := range entries {
		if entry.Namespace == namespace {
			return i
		}
	}
	return -1
}

func replaceMetadata(entries []domainv5.MetadataEntry, index int, replacement *domainv5.MetadataEntry) []domainv5.MetadataEntry {
	if index >= 0 {
		entries = append(entries[:index], entries[index+1:]...)
	}
	if replacement != nil {
		entries = append(entries, *cloneMetadataEntry(replacement))
	}
	return entries
}

func cloneMetadataEntry(entry *domainv5.MetadataEntry) *domainv5.MetadataEntry {
	if entry == nil {
		return nil
	}
	cloned := cloneMetadata([]domainv5.MetadataEntry{*entry})[0]
	return &cloned
}

func metadataEntriesEqual(left, right domainv5.MetadataEntry) bool {
	if left.Namespace != right.Namespace || (left.Schema == nil) != (right.Schema == nil) || !bytes.Equal(left.Value, right.Value) {
		return false
	}
	return left.Schema == nil || *left.Schema == *right.Schema
}
