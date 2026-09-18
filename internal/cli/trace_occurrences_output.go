package cli

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"

	"github.com/mako10k/sealgraph/internal/repository"
	"github.com/mako10k/sealgraph/internal/traceoccurrence"
)

type traceOccurrencesBaselineJSON struct {
	Kind            string  `json:"kind"`
	SealID          *string `json:"seal_id"`
	CandidateDigest *string `json:"candidate_digest"`
	OriginMapID     string  `json:"origin_map_id"`
}
type traceOccurrencesSelectionJSON struct {
	Kind      string                       `json:"kind"`
	Requested string                       `json:"requested"`
	Baseline  traceOccurrencesBaselineJSON `json:"baseline"`
}
type traceOccurrencesRunJSON struct {
	RunIndex         int    `json:"run_index"`
	SourceSnapshotID string `json:"source_snapshot_id"`
	SourceKey        string `json:"source_key"`
	SourceStart      uint64 `json:"source_start"`
	Length           uint64 `json:"length"`
	PatternSHA256    string `json:"pattern_sha256"`
}
type traceOccurrencesObservationJSON struct {
	SnapshotBlobID    string  `json:"snapshot_blob_id"`
	CurrentBlobID     *string `json:"current_blob_id"`
	CurrentByteLength *int    `json:"current_byte_length"`
	BindingDigest     *string `json:"binding_digest"`
}
type traceOccurrencesEntryJSON struct {
	View   string `json:"view"`
	Start  int    `json:"start"`
	Length int    `json:"length"`
}
type traceOccurrencesPageJSON struct {
	Limit      int                         `json:"limit"`
	Entries    []traceOccurrencesEntryJSON `json:"entries"`
	State      string                      `json:"state"`
	HasMore    *bool                       `json:"has_more"`
	NextCursor *string                     `json:"next_cursor"`
	Reason     *string                     `json:"reason"`
}
type traceOccurrencesDocument struct {
	Schema      string                          `json:"schema"`
	Selection   traceOccurrencesSelectionJSON   `json:"selection"`
	Run         traceOccurrencesRunJSON         `json:"run"`
	View        string                          `json:"view"`
	Observation traceOccurrencesObservationJSON `json:"observation"`
	Page        traceOccurrencesPageJSON        `json:"page"`
}

func prepareTraceOccurrences(ctx context.Context, repo *repository.Repository, options traceOccurrencesOptions) (traceOccurrencesDocument, error) {
	var previous *traceOccurrencesCursor
	if options.cursor.set {
		decoded, err := decodeTraceOccurrencesCursor(options.cursor.value)
		if err != nil {
			return traceOccurrencesDocument{}, err
		}
		previous = &decoded
	}
	baseline, err := options.selectedBaseline(ctx, repo)
	if err != nil {
		return traceOccurrencesDocument{}, err
	}
	if previous != nil {
		if err := precheckTraceOccurrencesCursor(ctx, repo, options, baseline, *previous); err != nil {
			return traceOccurrencesDocument{}, err
		}
	}
	input, err := repo.LoadTraceOccurrenceInput(ctx, baseline, options.runIndex, options.view.value != "snapshot")
	if err != nil {
		return traceOccurrencesDocument{}, err
	}
	doc, cursorContext := newTraceOccurrencesDocument(input, options)
	if previous != nil {
		if !sameTraceOccurrencesContext(*previous, cursorContext) || input.CurrentFailure != "" {
			return traceOccurrencesDocument{}, errPageContextChanged
		}
		if !validTraceOccurrencesLast(input, options.view.value, *previous) {
			return traceOccurrencesDocument{}, errPageTokenInvalid
		}
	}
	if input.CurrentFailure != "" {
		setTraceOccurrencesIncomplete(&doc, input.CurrentFailure)
		if doc.View == "both" {
			entries, _, err := collectTraceOccurrences(input, "snapshot", doc.Page.Limit, nil)
			if err != nil {
				return traceOccurrencesDocument{}, err
			}
			doc.Page.Entries = entries
		}
	} else if err := fillTraceOccurrencesPage(&doc, input, previous, cursorContext); err != nil {
		return traceOccurrencesDocument{}, err
	}
	if err := ctx.Err(); err != nil {
		return traceOccurrencesDocument{}, err
	}
	if err := repo.RevalidateTraceOccurrenceInput(ctx, input); err != nil {
		if previous != nil || errors.Is(err, repository.ErrTraceOccurrenceBaselineChanged) || errors.Is(err, repository.ErrTraceOccurrenceBindingChanged) {
			return traceOccurrencesDocument{}, errPageContextChanged
		}
		setTraceOccurrencesIncomplete(&doc, "OBSERVATION_UNSTABLE")
	}
	return doc, nil
}

func precheckTraceOccurrencesCursor(ctx context.Context, repo *repository.Repository, options traceOccurrencesOptions, baseline repository.TraceOwnBaseline, cursor traceOccurrencesCursor) error {
	requested, kind := options.ref.value, "ref"
	if options.seal.set {
		requested, kind = options.seal.value, "seal"
	}
	if cursor.Kind != kind || cursor.Requested != requested || cursor.RunIndex != options.runIndex || cursor.View != options.view.value || cursor.Limit != options.limit {
		return errPageContextChanged
	}
	switch baseline.Kind {
	case repository.TraceOwnCandidate:
		if cursor.Baseline != "candidate" {
			return errPageContextChanged
		}
		digest, err := repo.CandidateExactDigest(ctx, baseline.REF)
		if err != nil || hex.EncodeToString(digest[:]) != cursor.CandidateDigest {
			return errPageContextChanged
		}
	case repository.TraceOwnHead:
		if cursor.Baseline != "seal" {
			return errPageContextChanged
		}
		head, err := repo.CurrentREFHead(ctx, baseline.REF)
		if err != nil || head == nil || head.String() != cursor.SealID {
			return errPageContextChanged
		}
	case repository.TraceOwnSeal:
		if cursor.Baseline != "seal" || baseline.SealID == nil || baseline.SealID.String() != cursor.SealID {
			return errPageContextChanged
		}
	}
	return nil
}

func newTraceOccurrencesDocument(input repository.TraceOccurrenceInput, options traceOccurrencesOptions) (traceOccurrencesDocument, traceOccurrencesCursor) {
	requested, kind := options.ref.value, "ref"
	if options.seal.set {
		requested, kind = options.seal.value, "seal"
	}
	baselineKind := "candidate"
	var sealID, candidateDigest *string
	if input.SealID != nil {
		baselineKind = "seal"
		id := input.SealID.String()
		sealID = &id
	} else {
		digest := hex.EncodeToString(input.CandidateDigest[:])
		candidateDigest = &digest
	}
	patternDigest := sha256.Sum256(input.Pattern)
	doc := traceOccurrencesDocument{
		Schema:      "sealgraph/trace-occurrences/v1",
		Selection:   traceOccurrencesSelectionJSON{Kind: kind, Requested: requested, Baseline: traceOccurrencesBaselineJSON{Kind: baselineKind, SealID: sealID, CandidateDigest: candidateDigest, OriginMapID: input.OriginMapID.String()}},
		Run:         traceOccurrencesRunJSON{RunIndex: input.RunIndex, SourceSnapshotID: input.SourceSnapshot.String(), SourceKey: input.SourceKey, SourceStart: input.SourceStart, Length: input.Length, PatternSHA256: hex.EncodeToString(patternDigest[:])},
		View:        options.view.value,
		Observation: traceOccurrencesObservationJSON{SnapshotBlobID: input.SnapshotBlobID.String()},
		Page:        traceOccurrencesPageJSON{Limit: options.limit, Entries: []traceOccurrencesEntryJSON{}},
	}
	if input.CurrentBlobID != nil {
		id, length := input.CurrentBlobID.String(), len(input.CurrentBytes)
		doc.Observation.CurrentBlobID, doc.Observation.CurrentByteLength = &id, &length
	}
	if input.BindingDigest != nil {
		digest := hex.EncodeToString(input.BindingDigest[:])
		doc.Observation.BindingDigest = &digest
	}
	cursor := traceOccurrencesCursor{
		Kind: kind, Requested: requested, Baseline: baselineKind, OriginMapID: input.OriginMapID.String(),
		RunIndex: input.RunIndex, PatternSHA256: doc.Run.PatternSHA256, View: options.view.value,
		Limit: options.limit, SnapshotBlobID: input.SnapshotBlobID.String(),
	}
	if sealID != nil {
		cursor.SealID = *sealID
	}
	if candidateDigest != nil {
		cursor.CandidateDigest = *candidateDigest
	}
	if doc.Observation.CurrentBlobID != nil {
		cursor.CurrentBlobID, cursor.CurrentLength = *doc.Observation.CurrentBlobID, *doc.Observation.CurrentByteLength
	}
	if doc.Observation.BindingDigest != nil {
		cursor.BindingDigest = *doc.Observation.BindingDigest
	}
	return doc, cursor
}

func setTraceOccurrencesIncomplete(doc *traceOccurrencesDocument, reason string) {
	doc.Page = traceOccurrencesPageJSON{Limit: doc.Page.Limit, Entries: []traceOccurrencesEntryJSON{}, State: "INCOMPLETE", Reason: &reason}
}

func validTraceOccurrencesLast(input repository.TraceOccurrenceInput, view string, cursor traceOccurrencesCursor) bool {
	if cursor.LastView == "snapshot" && view == "current" || cursor.LastView == "current" && view == "snapshot" {
		return false
	}
	data := input.SnapshotBytes
	if cursor.LastView == "current" {
		data = input.CurrentBytes
	}
	start := cursor.LastStart
	return start <= len(data)-len(input.Pattern) && bytes.Equal(data[start:start+len(input.Pattern)], input.Pattern)
}

func fillTraceOccurrencesPage(doc *traceOccurrencesDocument, input repository.TraceOccurrenceInput, previous *traceOccurrencesCursor, cursorContext traceOccurrencesCursor) error {
	entries, hasMore, err := collectTraceOccurrences(input, doc.View, doc.Page.Limit, previous)
	if err != nil {
		return err
	}
	doc.Page.Entries, doc.Page.State, doc.Page.HasMore = entries, "READY", &hasMore
	if hasMore {
		last := entries[len(entries)-1]
		cursorContext.LastView, cursorContext.LastStart = last.View, last.Start
		token, err := encodeTraceOccurrencesCursor(cursorContext)
		if err != nil {
			return err
		}
		doc.Page.NextCursor = &token
	}
	return nil
}

func collectTraceOccurrences(input repository.TraceOccurrenceInput, view string, limit int, previous *traceOccurrencesCursor) ([]traceOccurrencesEntryJSON, bool, error) {
	entries := make([]traceOccurrencesEntryJSON, 0, min(limit, 100))
	if view != "current" && (previous == nil || previous.LastView == "snapshot") {
		after := -1
		if previous != nil {
			after = previous.LastStart
		}
		starts, more, err := traceoccurrence.Page(input.SnapshotBytes, input.Pattern, after, limit)
		if err != nil {
			return nil, false, err
		}
		for _, start := range starts {
			entries = append(entries, traceOccurrencesEntryJSON{View: "snapshot", Start: start, Length: len(input.Pattern)})
		}
		if more || view == "snapshot" {
			return entries, more, nil
		}
	}
	remaining := limit - len(entries)
	if remaining == 0 {
		lookahead, _, err := traceoccurrence.Page(input.CurrentBytes, input.Pattern, -1, 1)
		return entries, len(lookahead) != 0, err
	}
	after := -1
	if previous != nil && previous.LastView == "current" {
		after = previous.LastStart
	}
	starts, more, err := traceoccurrence.Page(input.CurrentBytes, input.Pattern, after, remaining)
	if err != nil {
		return nil, false, err
	}
	for _, start := range starts {
		entries = append(entries, traceOccurrencesEntryJSON{View: "current", Start: start, Length: len(input.Pattern)})
	}
	return entries, more, nil
}

func printTraceOccurrencesHuman(out io.Writer, doc traceOccurrencesDocument) {
	fmt.Fprintf(out, "Trace occurrences: %s %s, %s run %d, view=%s\n", doc.Selection.Kind, doc.Selection.Requested, doc.Selection.Baseline.Kind, doc.Run.RunIndex, doc.View)
	current, more := "unread", "unknown"
	if doc.Observation.CurrentBlobID != nil {
		current = *doc.Observation.CurrentBlobID
	}
	if doc.Page.HasMore != nil {
		more = fmt.Sprintf("%t", *doc.Page.HasMore)
	}
	fmt.Fprintf(out, "Snapshot Blob: %s; current Blob: %s; page=%s; has_more=%s\n", doc.Observation.SnapshotBlobID, current, doc.Page.State, more)
	for _, entry := range doc.Page.Entries {
		fmt.Fprintf(out, "  %s byte [%d,%d)\n", entry.View, entry.Start, entry.Start+entry.Length)
	}
	if doc.Page.NextCursor != nil {
		fmt.Fprintf(out, "Next cursor: %s\n", *doc.Page.NextCursor)
	}
	if doc.Page.Reason != nil {
		fmt.Fprintf(out, "Incomplete reason: %s\n", *doc.Page.Reason)
	}
}
