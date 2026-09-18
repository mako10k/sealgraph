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

type traceCompareCLIOptions struct {
	ref            singleString
	seal           singleString
	maxGraphVisits int
	estimate       bool
	output         inspectionOutput
}

func runTraceCompare(ctx context.Context, workDir string, args []string, stdout, stderr io.Writer) int {
	options, err := parseTraceCompareArgs(args, stdout)
	if err != nil {
		return flagUsageError(stderr, "trace compare", err)
	}
	repo, err := repository.OpenStandalone(workDir)
	if err != nil {
		return commandError(stderr, "trace compare", err)
	}
	if repo.Format() != 7 {
		return commandError(stderr, "trace compare", fmt.Errorf("origin trace comparison requires repository format 7"))
	}
	var document traceCompareV2Document
	if options.estimate {
		document, err = prepareTraceCompareWithEstimate(ctx, repo, options.ref, options.seal, options.maxGraphVisits)
	} else {
		document, err = prepareTraceCompareNoEstimate(ctx, repo, options.ref, options.seal, options.maxGraphVisits)
	}
	if err != nil {
		return commandError(stderr, "trace compare", err)
	}
	if options.output.JSON {
		return writeInspectionJSON(stdout, stderr, "trace compare", document)
	}
	printTraceCompareV2Human(stdout, document)
	return 0
}

func parseTraceCompareArgs(args []string, stdout io.Writer) (traceCompareCLIOptions, error) {
	args, output, err := extractInspectionFormat(args, stdout)
	if err != nil {
		return traceCompareCLIOptions{}, err
	}
	flags := flag.NewFlagSet("trace compare", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	var ref, seal, visits singleString
	var estimate singleBool
	flags.Var(&ref, "ref", "logical REF")
	flags.Var(&seal, "seal", "immutable Seal selector")
	flags.Var(&visits, "max-graph-visits", "positive graph visit limit")
	flags.Var(&estimate, "estimate", "also estimate changed current ranges")
	if err := flags.Parse(args); err != nil {
		return traceCompareCLIOptions{}, err
	}
	if flags.NArg() != 0 || ref.set == seal.set || ref.set && ref.value == "" || seal.set && seal.value == "" {
		return traceCompareCLIOptions{}, fmt.Errorf("trace compare requires exactly one of --ref REF or --seal SELECTOR and no positional arguments")
	}
	if ref.set {
		if err := domain.ValidateREF(ref.value); err != nil {
			return traceCompareCLIOptions{}, fmt.Errorf("invalid trace REF: %w", err)
		}
	} else {
		selector, err := repository.ParseSelector(seal.value)
		if err != nil || selector.Kind == repository.SelectorCurrentREF {
			return traceCompareCLIOptions{}, fmt.Errorf("trace compare --seal requires an immutable Seal selector")
		}
	}
	if !visits.set {
		return traceCompareCLIOptions{}, fmt.Errorf("trace compare requires --max-graph-visits N")
	}
	maxGraphVisits, err := strconv.Atoi(visits.value)
	if err != nil || maxGraphVisits <= 0 {
		return traceCompareCLIOptions{}, fmt.Errorf("--max-graph-visits requires a positive integer")
	}
	return traceCompareCLIOptions{ref: ref, seal: seal, maxGraphVisits: maxGraphVisits, estimate: estimate.value, output: output}, nil
}
