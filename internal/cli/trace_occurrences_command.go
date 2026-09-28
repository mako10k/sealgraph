package cli

import (
	"context"
	"flag"
	"fmt"
	"io"
	"strconv"

	"github.com/mako10k/sealgraph/internal/domain"
	"github.com/mako10k/sealgraph/internal/repository"
)

type traceOccurrencesOptions struct {
	ref, seal, baseline, view, cursor singleString
	runIndex, limit                   int
	output                            inspectionOutput
}

func runTraceOccurrences(ctx context.Context, workDir string, args []string, stdout, stderr io.Writer) int {
	options, err := parseTraceOccurrencesArgs(args, stdout)
	if err != nil {
		return flagUsageError(stderr, "trace occurrences", err)
	}
	repo, err := repository.OpenStandalone(workDir)
	if err != nil {
		return commandError(stderr, "trace occurrences", err)
	}
	if repo.Format() != 7 {
		return commandError(stderr, "trace occurrences", fmt.Errorf("origin trace occurrence listing requires repository format 7"))
	}
	doc, err := prepareTraceOccurrences(ctx, repo, options)
	if err != nil {
		return commandError(stderr, "trace occurrences", err)
	}
	if options.output.JSON {
		return writeInspectionJSON(stdout, stderr, "trace occurrences", doc)
	}
	printTraceOccurrencesHuman(stdout, doc)
	return 0
}

func parseTraceOccurrencesArgs(args []string, stdout io.Writer) (traceOccurrencesOptions, error) {
	args, output, err := extractInspectionFormat(args, stdout)
	if err != nil {
		return traceOccurrencesOptions{}, err
	}
	flags := flag.NewFlagSet("trace occurrences", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	var options traceOccurrencesOptions
	var run, limit singleString
	flags.Var(&options.ref, "ref", "logical REF")
	flags.Var(&options.seal, "seal", "immutable Seal selector")
	flags.Var(&options.baseline, "baseline", "candidate or head")
	flags.Var(&run, "run-index", "zero-based OriginMap run index")
	flags.Var(&options.view, "view", "snapshot, current, or both")
	flags.Var(&limit, "limit", "positive page entry limit")
	flags.Var(&options.cursor, "cursor", "opaque continuation token")
	if err := flags.Parse(args); err != nil {
		return traceOccurrencesOptions{}, err
	}
	if flags.NArg() != 0 || options.ref.set == options.seal.set {
		return traceOccurrencesOptions{}, fmt.Errorf("requires exactly one of --ref REF or --seal SELECTOR, without positional arguments")
	}
	if err := validateTraceOccurrencesSelection(options); err != nil {
		return traceOccurrencesOptions{}, err
	}
	if !run.set || !options.view.set {
		return traceOccurrencesOptions{}, fmt.Errorf("requires --run-index N and --view snapshot|current|both")
	}
	options.runIndex, err = strconv.Atoi(run.value)
	if err != nil || options.runIndex < 0 {
		return traceOccurrencesOptions{}, fmt.Errorf("--run-index requires a non-negative integer")
	}
	if options.view.value != "snapshot" && options.view.value != "current" && options.view.value != "both" {
		return traceOccurrencesOptions{}, fmt.Errorf("--view requires snapshot, current, or both")
	}
	options.limit = 100
	if limit.set {
		options.limit, err = strconv.Atoi(limit.value)
		if err != nil || options.limit <= 0 {
			return traceOccurrencesOptions{}, fmt.Errorf("--limit requires a positive integer")
		}
	}
	if options.cursor.set && options.cursor.value == "" {
		return traceOccurrencesOptions{}, fmt.Errorf("--cursor requires a non-empty token")
	}
	options.output = output
	return options, nil
}

func validateTraceOccurrencesSelection(options traceOccurrencesOptions) error {
	if options.ref.set {
		if err := domain.ValidateREF(options.ref.value); err != nil {
			return fmt.Errorf("invalid trace REF: %w", err)
		}
		if !options.baseline.set || options.baseline.value != "candidate" && options.baseline.value != "head" {
			return fmt.Errorf("--ref requires --baseline candidate|head")
		}
		return nil
	}
	if options.baseline.set {
		return fmt.Errorf("--baseline applies only with --ref")
	}
	selector, err := repository.ParseSelector(options.seal.value)
	if err != nil || selector.Kind == repository.SelectorCurrentREF {
		return fmt.Errorf("--seal requires an immutable Seal selector")
	}
	return nil
}

func (options traceOccurrencesOptions) selectedBaseline(ctx context.Context, repo *repository.Repository) (repository.TraceOwnBaseline, error) {
	if options.ref.set {
		kind := repository.TraceOwnCandidate
		if options.baseline.value == "head" {
			kind = repository.TraceOwnHead
		}
		return repository.TraceOwnBaseline{Kind: kind, REF: options.ref.value}, nil
	}
	id, err := resolveTraceCompareSeal(ctx, repo, options.seal.value)
	if err != nil {
		return repository.TraceOwnBaseline{}, err
	}
	return repository.TraceOwnBaseline{Kind: repository.TraceOwnSeal, SealID: &id}, nil
}
