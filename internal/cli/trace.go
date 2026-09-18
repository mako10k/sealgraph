package cli

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"

	canonicalv7 "github.com/mako10k/sealgraph/internal/canonical/v7"
	"github.com/mako10k/sealgraph/internal/domain"
	domainv5 "github.com/mako10k/sealgraph/internal/domain/v5"
	"github.com/mako10k/sealgraph/internal/repository"
	"github.com/mako10k/sealgraph/internal/tracerecipe"
	"github.com/mako10k/sealgraph/internal/workfile"
)

func runTrace(ctx context.Context, workDir string, args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		return usageError(stderr, "trace requires set, clear, or show")
	}
	switch args[0] {
	case "set":
		return runTraceSet(ctx, workDir, args[1:], stdin, stdout, stderr)
	case "clear":
		return runTraceClear(ctx, workDir, args[1:], stdout, stderr)
	case "show":
		return runTraceShow(ctx, workDir, args[1:], stdout, stderr)
	default:
		return usageError(stderr, "unknown trace operation %q; expected set, clear, or show", args[0])
	}
}

func runTraceSet(ctx context.Context, workDir string, args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	args, output, err := extractInspectionFormat(args, stdout)
	if err != nil {
		return usageError(stderr, "%v", err)
	}
	var recipePath, contentPath singleString
	ref, _, err := sourceREFAndFlags(args, "trace set", func(flags *flag.FlagSet) {
		flags.Var(&recipePath, "recipe", "trace recipe file")
		flags.Var(&contentPath, "content-file", "exact content file or - for stdin")
	})
	if err != nil {
		return usageDiagnostic(stderr, "trace set", err.Error(), "provide an existing REF and exactly one --recipe PATH")
	}
	if !recipePath.set || recipePath.value == "" || contentPath.set && contentPath.value == "" {
		return usageError(stderr, "trace set requires one --recipe PATH and a non-empty --content-file when supplied")
	}
	if err := domain.ValidateREF(ref); err != nil {
		return usageError(stderr, "invalid trace REF: %v", err)
	}
	repo, err := repository.OpenStandalone(workDir)
	if err != nil {
		return commandError(stderr, "trace set", err)
	}
	if repo.Format() != 7 {
		return commandError(stderr, "trace set", fmt.Errorf("trace authoring requires repository format 7"))
	}
	recipeBytes, err := readTraceInput(workDir, recipePath.value)
	if err != nil {
		return commandError(stderr, "trace set recipe", err)
	}
	recipe, err := tracerecipe.Parse(recipeBytes)
	if err != nil {
		return commandError(stderr, "trace set recipe", err)
	}
	options := repository.TraceSetOptions{REF: ref, BeforeStore: traceSourceDisclosure(stderr)}
	if contentPath.set {
		options.ContentSet = true
		if contentPath.value == "-" {
			options.Content, err = io.ReadAll(stdin)
		} else {
			options.Content, err = readTraceInput(workDir, contentPath.value)
		}
		if err != nil {
			return commandError(stderr, "trace set content", err)
		}
	}
	if options.Sources, options.Runs, err = compileTraceRecipe(workDir, recipe); err != nil {
		return commandError(stderr, "trace set recipe", err)
	}
	result, err := repo.TraceSet(ctx, options)
	if err != nil {
		return commandError(stderr, "trace set", err)
	}
	return writeTraceSetReceipt(output, stdout, stderr, ref, result)
}

func readTraceInput(workDir, path string) ([]byte, error) {
	if path == ".sealgraph" || strings.HasPrefix(path, ".sealgraph/") {
		return nil, fmt.Errorf("path %q is within the canonical repository", path)
	}
	return workfile.ReadStable(workDir, path)
}

func compileTraceRecipe(workDir string, recipe tracerecipe.Recipe) ([]repository.TraceSourceInput, []repository.TraceRunInput, error) {
	sources := make([]repository.TraceSourceInput, 0, len(recipe.Sources))
	fileBytes := make(map[string][]byte)
	for _, source := range recipe.Sources {
		input := repository.TraceSourceInput{Name: source.Name}
		if source.SnapshotID.Hex != "" {
			id := source.SnapshotID
			input.SnapshotID = &id
		} else {
			input.SourceKey, input.DisplayPath = source.SourceKey, source.File
			content, ok := fileBytes[source.File]
			if !ok {
				var err error
				content, err = readTraceInput(workDir, source.File)
				if err != nil {
					return nil, nil, fmt.Errorf("source %q file %q: %w", source.Name, source.File, err)
				}
				fileBytes[source.File] = content
			}
			input.Content = content
		}
		sources = append(sources, input)
	}
	runs := make([]repository.TraceRunInput, len(recipe.Runs))
	for index, run := range recipe.Runs {
		runs[index] = repository.TraceRunInput{Kind: run.Kind, Length: run.Length, SourceName: run.Source, SourceStart: run.SourceStart}
	}
	return sources, runs, nil
}

func traceSourceDisclosure(stderr io.Writer) func([]repository.StoredTraceSource) error {
	return func(sources []repository.StoredTraceSource) error {
		for _, source := range sources {
			if _, err := fmt.Fprintf(stderr, "retaining full source file %q (%d bytes)\n", source.DisplayPath, source.ByteCount); err != nil {
				return err
			}
		}
		return nil
	}
}

type traceMutationReceipt struct {
	Schema        string                  `json:"schema"`
	Operation     string                  `json:"operation"`
	REF           string                  `json:"ref"`
	BeforeDigest  string                  `json:"before_digest"`
	AfterDigest   string                  `json:"after_digest"`
	Changed       bool                    `json:"changed"`
	StoredSources []traceStoredSourceJSON `json:"stored_sources"`
}

type traceStoredSourceJSON struct {
	SourceKey     string  `json:"source_key"`
	SnapshotID    string  `json:"snapshot_id"`
	ContentBlobID string  `json:"content_blob_id"`
	ByteLength    uint64  `json:"byte_length"`
	InputFile     *string `json:"input_file"`
}

func writeTraceSetReceipt(output inspectionOutput, stdout, stderr io.Writer, ref string, result repository.TraceSetResult) int {
	before, after := fmt.Sprintf("%x", result.BeforeCandidateSHA256), fmt.Sprintf("%x", result.AfterCandidateSHA256)
	stored := make([]repository.StoredTraceSource, len(result.StoredSources))
	copy(stored, result.StoredSources)
	sort.Slice(stored, func(i, j int) bool {
		left, right := stored[i], stored[j]
		if left.SnapshotID.String() != right.SnapshotID.String() {
			return left.SnapshotID.String() < right.SnapshotID.String()
		}
		if left.New != right.New {
			return !left.New
		}
		return left.DisplayPath < right.DisplayPath
	})
	entries := make([]traceStoredSourceJSON, len(stored))
	for i, source := range stored {
		entries[i] = traceStoredSourceJSON{SourceKey: source.SourceKey, SnapshotID: source.SnapshotID.String(), ContentBlobID: source.BlobID.String(), ByteLength: source.ByteCount}
		if source.New {
			entries[i].InputFile = &stored[i].DisplayPath
		}
	}
	receipt := traceMutationReceipt{Schema: "sealgraph/trace-mutation/v2", Operation: "set", REF: ref, BeforeDigest: before, AfterDigest: after, Changed: before != after, StoredSources: entries}
	if output.JSON {
		return writeInspectionJSON(stdout, stderr, "trace set", receipt)
	}
	printHumanReceipt(stdout, "TRACE SET", humanField{"REF", ref}, humanField{"Origin map (prefix)", shortID(result.OriginID)}, humanField{"Candidate changed", yesNo(receipt.Changed)}, humanField{"Full sources", strconv.Itoa(len(entries))})
	for _, source := range stored {
		if source.New {
			fmt.Fprintf(stdout, "  Full source file %q (%d bytes)\n", source.DisplayPath, source.ByteCount)
		} else {
			fmt.Fprintf(stdout, "  Full source snapshot %s (existing Snapshot reused, %d bytes)\n", source.SnapshotID, source.ByteCount)
		}
	}
	return 0
}

func runTraceClear(ctx context.Context, workDir string, args []string, stdout, stderr io.Writer) int {
	args, output, err := extractInspectionFormat(args, stdout)
	if err != nil {
		return usageError(stderr, "%v", err)
	}
	if len(args) != 1 {
		return usageError(stderr, "trace clear requires exactly one REF")
	}
	repo, err := repository.OpenStandalone(workDir)
	if err != nil {
		return commandError(stderr, "trace clear", err)
	}
	result, err := repo.TraceClear(ctx, args[0])
	if err != nil {
		return commandError(stderr, "trace clear", err)
	}
	receipt := traceMutationReceipt{Schema: "sealgraph/trace-mutation/v2", Operation: "clear", REF: args[0], BeforeDigest: fmt.Sprintf("%x", result.BeforeCandidateSHA256), AfterDigest: fmt.Sprintf("%x", result.AfterCandidateSHA256), Changed: result.Changed, StoredSources: []traceStoredSourceJSON{}}
	if output.JSON {
		return writeInspectionJSON(stdout, stderr, "trace clear", receipt)
	}
	printHumanReceipt(stdout, "TRACE CLEARED", humanField{"REF", args[0]}, humanField{"Candidate changed", yesNo(result.Changed)})
	return 0
}

type traceShowSelection struct {
	Kind           string  `json:"kind"`
	Requested      string  `json:"requested"`
	ResolvedSealID *string `json:"resolved_seal_id"`
}

type traceShowBaseline struct {
	Kind            string  `json:"kind"`
	REF             *string `json:"ref"`
	SealID          *string `json:"seal_id"`
	CandidateDigest *string `json:"candidate_digest"`
	ContentBlobID   string  `json:"content_blob_id"`
	OriginMapID     *string `json:"origin_map_id"`
}

type traceShowSnapshot struct {
	ID     string          `json:"id"`
	Record json.RawMessage `json:"record"`
}

type traceShowInspection struct {
	Baseline        traceShowBaseline   `json:"baseline"`
	OriginMap       *json.RawMessage    `json:"origin_map"`
	SourceSnapshots []traceShowSnapshot `json:"source_snapshots"`
}

type traceShowDocument struct {
	Schema    string                `json:"schema"`
	Selection traceShowSelection    `json:"selection"`
	Baselines []traceShowInspection `json:"baselines"`
}

func runTraceShow(ctx context.Context, workDir string, args []string, stdout, stderr io.Writer) int {
	args, output, err := extractInspectionFormat(args, stdout)
	if err != nil {
		return usageError(stderr, "%v", err)
	}
	flags := flag.NewFlagSet("trace show", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	var ref, seal singleString
	flags.Var(&ref, "ref", "logical REF")
	flags.Var(&seal, "seal", "immutable Seal selector")
	if err := flags.Parse(args); err != nil {
		return flagUsageError(stderr, "trace show", err)
	}
	if flags.NArg() != 0 || ref.set == seal.set || ref.set && ref.value == "" || seal.set && seal.value == "" {
		return usageError(stderr, "trace show requires exactly one of --ref REF or --seal SELECTOR")
	}
	if ref.set {
		if err := domain.ValidateREF(ref.value); err != nil {
			return usageError(stderr, "invalid trace REF: %v", err)
		}
	} else {
		selector, err := repository.ParseSelector(seal.value)
		if err != nil || selector.Kind == repository.SelectorCurrentREF {
			return usageError(stderr, "trace show --seal requires an immutable Seal selector")
		}
	}
	repo, err := repository.OpenStandalone(workDir)
	if err != nil {
		return commandError(stderr, "trace show", err)
	}
	document, err := buildTraceShow(ctx, repo, ref, seal)
	if err != nil {
		return commandError(stderr, "trace show", err)
	}
	if output.JSON {
		return writeInspectionJSON(stdout, stderr, "trace show", document)
	}
	printTraceShowHuman(stdout, document)
	return 0
}

func buildTraceShow(ctx context.Context, repo *repository.Repository, ref, seal singleString) (traceShowDocument, error) {
	doc := traceShowDocument{Schema: "sealgraph/trace-show/v1", Baselines: []traceShowInspection{}}
	if seal.set {
		selected, err := repo.ResolveSelector(ctx, seal.value)
		if err != nil {
			return traceShowDocument{}, err
		}
		id := selected.ID.String()
		doc.Selection = traceShowSelection{Kind: "seal", Requested: seal.value, ResolvedSealID: &id}
		item, err := traceSealInspection(ctx, repo, selected.Resolved, nil)
		if err != nil {
			return traceShowDocument{}, err
		}
		doc.Baselines = append(doc.Baselines, item)
		return doc, nil
	}
	doc.Selection = traceShowSelection{Kind: "ref", Requested: ref.value}
	beforeHead, err := repo.CurrentREFHead(ctx, ref.value)
	if err != nil {
		return traceShowDocument{}, err
	}
	beforeDigest, digestErr := repo.CandidateExactDigest(ctx, ref.value)
	if digestErr != nil && !errors.Is(digestErr, repository.ErrCandidateNotFound) {
		return traceShowDocument{}, digestErr
	}
	if digestErr == nil {
		candidate, err := repo.InspectCandidate(ctx, ref.value)
		if err != nil {
			return traceShowDocument{}, err
		}
		digest := fmt.Sprintf("%x", beforeDigest)
		item, err := traceInspection(ctx, repo, "candidate", &ref.value, nil, &digest, candidate.Candidate.Content, candidate.Candidate.Origin)
		if err != nil {
			return traceShowDocument{}, err
		}
		doc.Baselines = append(doc.Baselines, item)
	}
	if beforeHead != nil {
		resolved, err := repo.LoadSeal(ctx, *beforeHead)
		if err != nil {
			return traceShowDocument{}, err
		}
		item, err := traceSealInspection(ctx, repo, resolved, &ref.value)
		if err != nil {
			return traceShowDocument{}, err
		}
		doc.Baselines = append(doc.Baselines, item)
		id := beforeHead.String()
		doc.Selection.ResolvedSealID = &id
	}
	if len(doc.Baselines) == 0 {
		return traceShowDocument{}, fmt.Errorf("REF %s has no Candidate or HEAD", ref.value)
	}
	if err := verifyTraceShowBaselines(ctx, repo, ref.value, beforeHead, beforeDigest, digestErr == nil); err != nil {
		return traceShowDocument{}, err
	}
	return doc, nil
}

func verifyTraceShowBaselines(ctx context.Context, repo *repository.Repository, ref string, head *domain.ObjectID, digest [32]byte, candidatePresent bool) error {
	afterHead, err := repo.CurrentREFHead(ctx, ref)
	if err != nil {
		return err
	}
	if head == nil && afterHead != nil || head != nil && (afterHead == nil || !head.Equal(*afterHead)) {
		return fmt.Errorf("trace show REF %s changed during observation", ref)
	}
	afterDigest, err := repo.CandidateExactDigest(ctx, ref)
	if candidatePresent {
		if err != nil || afterDigest != digest {
			return fmt.Errorf("trace show Candidate %s changed during observation: %v", ref, err)
		}
	} else if !errors.Is(err, repository.ErrCandidateNotFound) {
		return fmt.Errorf("trace show Candidate %s appeared or became unreadable: %v", ref, err)
	}
	return nil
}

func traceSealInspection(ctx context.Context, repo *repository.Repository, resolved domainv5.ResolvedSeal, ref *string) (traceShowInspection, error) {
	id := resolved.ID.String()
	return traceInspection(ctx, repo, "seal", ref, &id, nil, resolved.Material.Content, resolved.Provenance.Origin)
}

func traceInspection(ctx context.Context, repo *repository.Repository, kind string, ref, sealID, digest *string, contentID domain.ObjectID, originID *domain.ObjectID) (traceShowInspection, error) {
	item := traceShowInspection{Baseline: traceShowBaseline{Kind: kind, REF: ref, SealID: sealID, CandidateDigest: digest, ContentBlobID: contentID.String()}, SourceSnapshots: []traceShowSnapshot{}}
	if originID == nil {
		return item, nil
	}
	loaded, err := repo.LoadOrigin(ctx, *originID, contentID)
	if err != nil {
		return traceShowInspection{}, err
	}
	id := originID.String()
	item.Baseline.OriginMapID = &id
	mapBytes, err := canonicalv7.EncodeOriginMap(loaded.Map)
	if err != nil {
		return traceShowInspection{}, err
	}
	mapRecord := json.RawMessage(mapBytes)
	item.OriginMap = &mapRecord
	for _, source := range loaded.Sources {
		snapshotBytes, err := canonicalv7.EncodeSourceSnapshot(source.Snapshot)
		if err != nil {
			return traceShowInspection{}, err
		}
		item.SourceSnapshots = append(item.SourceSnapshots, traceShowSnapshot{ID: source.SnapshotID.String(), Record: json.RawMessage(snapshotBytes)})
	}
	return item, nil
}

func printTraceShowHuman(stdout io.Writer, doc traceShowDocument) {
	fmt.Fprintf(stdout, "TRACE %s %s\n", doc.Selection.Kind, doc.Selection.Requested)
	for _, item := range doc.Baselines {
		fmt.Fprintf(stdout, "%s content=%s", strings.ToUpper(item.Baseline.Kind), item.Baseline.ContentBlobID)
		if item.Baseline.OriginMapID == nil {
			fmt.Fprint(stdout, " origin=NO_TRACE")
		} else {
			fmt.Fprintf(stdout, " origin=%s sources=%d", *item.Baseline.OriginMapID, len(item.SourceSnapshots))
		}
		fmt.Fprintln(stdout)
	}
}
