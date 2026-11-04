package cli

// R: assemble read-only daily observations without conflating provenance and files.

import (
	"bytes"
	"context"
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"reflect"
	"strconv"

	"github.com/mako10k/sealgraph/internal/repository"
)

type inspectDocument struct {
	Schema           string         `json:"schema"`
	RepositoryFormat int            `json:"repository_format"`
	Complete         bool           `json:"inventory_complete"`
	Omitted          int            `json:"omitted_refs"`
	Entries          []inspectEntry `json:"entries"`
	Limitation       string         `json:"limitation"`
}
type inspectEntry struct {
	Status          statusRecordJSON        `json:"sealed_state"`
	CandidateDigest *string                 `json:"candidate_digest"`
	Workfile        inspectWorkfile         `json:"workfile"`
	TraceState      string                  `json:"trace_state"`
	Trace           []traceCompareLocalJSON `json:"trace_baselines"`
	Reviews         []inspectReview         `json:"historical_reviews"`
	Next            []string                `json:"next_read_actions"`
}
type inspectWorkfile struct {
	State    string  `json:"state"`
	Baseline string  `json:"baseline"`
	BlobID   *string `json:"workfile_blob_id"`
}
type inspectReview struct {
	Baseline   string                                 `json:"baseline"`
	SealID     string                                 `json:"selected_seal"`
	State      string                                 `json:"state"`
	Applicable *bool                                  `json:"applicable"`
	Record     *repository.HistoricalReviewInspection `json:"inspection"`
}

func runInspect(ctx context.Context, workDir string, args []string, stdout, stderr io.Writer) int {
	args, output, err := extractInspectionFormat(args, stdout)
	if err != nil {
		return flagUsageError(stderr, "inspect", err)
	}
	ref, limit, err := parseInspectArgs(args)
	if err != nil {
		return flagUsageError(stderr, "inspect", err)
	}
	repo, err := repository.OpenStandalone(workDir)
	if err != nil {
		return commandError(stderr, "inspect", err)
	}
	doc, err := prepareInspect(ctx, repo, ref, limit)
	if err != nil {
		return commandError(stderr, "inspect", err)
	}
	if output.JSON {
		return writeInspectionJSON(stdout, stderr, "inspect", doc)
	}
	var rendered bytes.Buffer
	printInspectHuman(&rendered, doc)
	return writeOperatorOutput(stdout, stderr, "inspect", rendered.Bytes())
}

func parseInspectArgs(args []string) (string, int, error) {
	// REF may precede flags, as for status; flags-first is also accepted.
	ref := ""
	if len(args) > 0 && len(args[0]) > 0 && args[0][0] != '-' {
		ref, args = args[0], args[1:]
	}
	flags := flag.NewFlagSet("inspect", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	var max singleString
	flags.Var(&max, "max-refs", "positive visible REF limit, default 100")
	if err := flags.Parse(args); err != nil {
		return "", 0, err
	}
	if flags.NArg() > 1 || ref != "" && flags.NArg() != 0 {
		return "", 0, fmt.Errorf("inspect accepts at most one exact REF")
	}
	if flags.NArg() == 1 {
		ref = flags.Arg(0)
	}
	limit := 100
	if max.set {
		value, err := strconv.Atoi(max.value)
		if err != nil || value < 1 || value > 10000 {
			return "", 0, fmt.Errorf("--max-refs must be between 1 and 10000")
		}
		limit = value
	}
	return ref, limit, nil
}

func prepareInspect(ctx context.Context, repo *repository.Repository, ref string, limit int) (inspectDocument, error) {
	heads, err := repo.TraceDirectionHeads(ctx)
	if err != nil {
		return inspectDocument{}, err
	}
	before, err := repo.Status(ctx, ref)
	if err != nil {
		return inspectDocument{}, err
	}
	count := min(limit, len(before))
	doc := inspectDocument{Schema: "sealgraph/inspect/v1", RepositoryFormat: repo.Format(), Complete: count == len(before), Omitted: len(before) - count, Entries: []inspectEntry{}, Limitation: "On-demand observations only; no background notification, simultaneous filesystem snapshot, semantic approval or automatic repair. Trace is local; use trace compare/impact for directional investigation."}
	for _, status := range before[:count] {
		entry, err := inspectOne(ctx, repo, status)
		if err != nil {
			return inspectDocument{}, err
		}
		doc.Entries = append(doc.Entries, entry)
	}
	if err := revalidateInspect(ctx, repo, ref, before, doc.Entries); err != nil {
		return inspectDocument{}, err
	}
	if err := repo.RevalidateTraceDirectionHeads(ctx, heads); err != nil {
		return inspectDocument{}, err
	}
	return doc, nil
}

func inspectOne(ctx context.Context, repo *repository.Repository, status repository.RefStatus) (inspectEntry, error) {
	entry := inspectEntry{Status: statusRecord(status), TraceState: "UNSUPPORTED_FORMAT", Trace: []traceCompareLocalJSON{}, Reviews: []inspectReview{}}
	digest, err := inspectCandidateDigest(ctx, repo, status)
	if err != nil {
		return entry, err
	}
	entry.CandidateDigest = digest
	entry.Workfile, err = inspectWorkfileState(ctx, repo, status)
	if err != nil {
		return entry, err
	}
	if repo.Format() == 7 {
		entry.TraceState = "NO_BASELINE"
		baselines := inspectTraceBaselines(status)
		if len(baselines) > 0 {
			results, _, err := repo.TraceCompareOwnBatch(ctx, baselines)
			if err != nil {
				return entry, err
			}
			entry.TraceState = "OBSERVED"
			for _, value := range results {
				entry.Trace = append(entry.Trace, inspectTraceLocal(value))
			}
		}
	}
	entry.Reviews, err = inspectReviews(ctx, repo, status)
	if err != nil {
		return entry, err
	}
	entry.Next = inspectReadActions(status)
	return entry, nil
}

func inspectCandidateDigest(ctx context.Context, repo *repository.Repository, status repository.RefStatus) (*string, error) {
	if !status.Unsealed {
		return nil, nil
	}
	digest, err := repo.CandidateExactDigest(ctx, status.REF)
	if err != nil {
		return nil, err
	}
	value := hex.EncodeToString(digest[:])
	return &value, nil
}

func inspectWorkfileState(ctx context.Context, repo *repository.Repository, status repository.RefStatus) (inspectWorkfile, error) {
	result := inspectWorkfile{State: "NOT_BOUND", Baseline: "NONE"}
	if status.Source == nil {
		return result, nil
	}
	result.State, result.Baseline = status.Source.Relation, status.Source.Baseline
	if status.Source.Baseline == "NONE" || status.Source.Relation == "SOURCE_MISSING" || status.Source.Relation == "SOURCE_UNREADABLE" {
		return result, nil
	}
	compared, err := repo.SourceCompare(ctx, status.REF)
	if err != nil {
		return result, err
	}
	id := compared.WorkfileID.String()
	result.BlobID = &id
	return result, nil
}

func inspectTraceBaselines(status repository.RefStatus) []repository.TraceOwnBaseline {
	result := []repository.TraceOwnBaseline{}
	if status.Unsealed {
		result = append(result, repository.TraceOwnBaseline{Kind: repository.TraceOwnCandidate, REF: status.REF})
	}
	if status.Head != nil {
		result = append(result, repository.TraceOwnBaseline{Kind: repository.TraceOwnHead, REF: status.REF})
	}
	return result
}

func revalidateInspect(ctx context.Context, repo *repository.Repository, ref string, before []repository.RefStatus, entries []inspectEntry) error {
	after, err := repo.Status(ctx, ref)
	if err != nil {
		return err
	}
	if !reflect.DeepEqual(before, after) {
		return fmt.Errorf("inspection context changed; inspect again explicitly")
	}
	for i, entry := range entries {
		digest, err := inspectCandidateDigest(ctx, repo, after[i])
		if err != nil {
			return err
		}
		workfile, err := inspectWorkfileState(ctx, repo, after[i])
		if err != nil {
			return err
		}
		if err := revalidateInspectReviews(ctx, repo, after[i], entry); err != nil {
			return err
		}
		if err := revalidateInspectTrace(ctx, repo, after[i], entry); err != nil {
			return err
		}
		if !reflect.DeepEqual(digest, entry.CandidateDigest) || !reflect.DeepEqual(workfile, entry.Workfile) {
			return fmt.Errorf("selected Candidate or source changed during inspection; inspect again explicitly")
		}
	}
	return nil
}

// Reobserve selected local facts before emitting an aggregate; source bindings
// and file bytes remain independent observations, never a global reservation.
func revalidateInspectTrace(ctx context.Context, repo *repository.Repository, status repository.RefStatus, entry inspectEntry) error {
	if repo.Format() != 7 {
		return nil
	}
	results, _, err := repo.TraceCompareOwnBatch(ctx, inspectTraceBaselines(status))
	if err != nil {
		return err
	}
	current := []traceCompareLocalJSON{}
	for _, result := range results {
		current = append(current, inspectTraceLocal(result))
	}
	if !reflect.DeepEqual(current, entry.Trace) {
		return fmt.Errorf("Trace source or baseline changed during inspection; inspect again explicitly")
	}
	return nil
}

func inspectTraceLocal(value repository.TraceCompareOwnResult) traceCompareLocalJSON {
	result := buildTraceCompareLocal(value)
	if value.Baseline.Kind == repository.TraceOwnHead {
		result.Baseline.Kind = "head"
	}
	return result
}

func revalidateInspectReviews(ctx context.Context, repo *repository.Repository, status repository.RefStatus, entry inspectEntry) error {
	current, err := inspectReviews(ctx, repo, status)
	if err != nil {
		return err
	}
	if !reflect.DeepEqual(current, entry.Reviews) {
		return fmt.Errorf("historical review context changed during inspection; inspect again explicitly")
	}
	return nil
}
