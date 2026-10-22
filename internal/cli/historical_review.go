package cli

// R: parse and present explicit single-subject historical-review operations;
// factual preparation never constructs a review decision.

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/mako10k/sealgraph/internal/fsread"
	"github.com/mako10k/sealgraph/internal/historicalreview"
	"github.com/mako10k/sealgraph/internal/repository"
)

type historicalReviewMutation struct {
	Schema    string `json:"schema"`
	REF       string `json:"ref"`
	Operation string `json:"operation"`
}

func runHistoricalReview(ctx context.Context, workDir string, args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	if len(args) < 2 {
		return usageError(stderr, "review requires prepare|set|clear|show and exactly one subject")
	}
	operation := args[0]
	args, output, err := extractInspectionFormat(args[1:], stdout)
	if err != nil {
		return usageError(stderr, "%v", err)
	}
	if len(args) == 0 {
		return usageError(stderr, "review requires exactly one REF or selector")
	}
	subject := args[0]
	flags := flag.NewFlagSet("review "+operation, flag.ContinueOnError)
	flags.SetOutput(stderr)
	var file singleString
	var candidate singleBool
	if operation == "set" {
		flags.Var(&file, "file", "bounded explicit review record file or stdin")
	}
	if operation == "show" {
		flags.Var(&candidate, "candidate", "inspect Candidate instead of immutable selection")
	}
	if err := flags.Parse(args[1:]); err != nil {
		return 2
	}
	if flags.NArg() != 0 {
		return usageError(stderr, "review accepts exactly one subject")
	}
	repo, err := repository.OpenStandalone(workDir)
	if err != nil {
		return commandError(stderr, "review", err)
	}
	var result any
	switch operation {
	case "prepare":
		result, err = repo.PrepareHistoricalReview(ctx, subject)
	case "show":
		result, err = repo.InspectHistoricalReview(ctx, subject, candidate.value)
	case "set":
		if !file.set {
			return usageError(stderr, "review set requires --file PATH|-")
		}
		var input []byte
		input, err = readHistoricalReviewInput(workDir, file.value, stdin)
		if err == nil {
			_, err = repo.SetHistoricalReview(ctx, subject, input)
		}
		result = historicalReviewMutation{"sealgraph/historical-review-mutation/v1", subject, "set"}
	case "clear":
		_, err = repo.ClearHistoricalReview(ctx, subject)
		result = historicalReviewMutation{"sealgraph/historical-review-mutation/v1", subject, "clear"}
	default:
		return usageError(stderr, "unknown review operation %q", operation)
	}
	if err != nil {
		return commandError(stderr, "review "+operation, err)
	}
	if output.JSON {
		return writeInspectionJSON(stdout, stderr, "review "+operation, result)
	}
	return printHistoricalReviewHuman(stdout, stderr, result)

}

func readHistoricalReviewInput(workDir, source string, stdin io.Reader) ([]byte, error) {
	var reader io.Reader = stdin
	if source != "-" {
		path := source
		if !filepath.IsAbs(path) {
			path = filepath.Join(workDir, path)
		}
		entry, err := os.Lstat(path)
		if err != nil {
			return nil, err
		}
		if !entry.Mode().IsRegular() {
			return nil, fmt.Errorf("review input must be a regular non-symlink file")
		}
		file, err := fsread.OpenFileNoFollow(path)
		if err != nil {
			return nil, err
		}
		info, err := file.Stat()
		if err != nil || !info.Mode().IsRegular() {
			file.Close()
			return nil, fmt.Errorf("review input must be a regular non-symlink file")
		}
		defer file.Close()
		reader = file
	}
	data, err := io.ReadAll(io.LimitReader(reader, historicalreview.MaxBytes+1))
	if err != nil {
		return nil, err
	}
	if len(data) > historicalreview.MaxBytes {
		return nil, fmt.Errorf("review exceeds %d bytes", historicalreview.MaxBytes)
	}
	return data, nil
}

func printHistoricalReviewHuman(stdout, stderr io.Writer, result any) int {
	switch value := result.(type) {
	case historicalReviewMutation:
		printHumanReceipt(stdout, "REVIEW "+value.Operation, humanField{"REF", value.REF}, humanField{"Seal issued", "no"})
	case repository.HistoricalReviewPreparation:
		fmt.Fprintln(stdout, "Historical review preparation: facts only; no judgment recorded.")
		printHistoricalReviewFacts(stdout, value)
	case repository.HistoricalReviewInspection:
		printHumanReceipt(stdout, "HISTORICAL REVIEW", humanField{"Selected Seal", value.SelectedSeal.String()}, humanField{"Candidate", yesNo(value.Candidate)}, humanField{"Recorded REF", value.Record.REF}, humanField{"Recorded subject without review", value.Record.SubjectSeal.String()}, humanField{"Claim currently applicable", yesNo(value.Applicable)}, humanField{"Limitation", value.Limitation})
		fmt.Fprintln(stdout, "Recorded other heads:")
		for _, head := range value.Record.OtherHeads {
			fmt.Fprintf(stdout, "  %s %s\n", head.REF, head.Seal)
		}
		for _, entry := range value.Record.Entries {
			printHumanReceipt(stdout, "RECORDED CAUSE JUDGMENT", humanField{"Target", entry.TargetSeal.String()}, humanField{"Decision", entry.Decision}, humanField{"Difference", quoteHumanString(entry.Difference)}, humanField{"Impact", quoteHumanString(entry.Impact)}, humanField{"Reason", quoteHumanString(entry.Reason)})
			for _, id := range entry.ComparedSeals {
				fmt.Fprintf(stdout, "  Compared Seal: %s\n", id)
			}
		}
		fmt.Fprintln(stdout, "Current facts (STALE remains independent):")
		printHistoricalReviewFacts(stdout, value.Facts)
	default:
		return commandError(stderr, "review", fmt.Errorf("unsupported review result"))
	}
	return 0
}

func printHistoricalReviewFacts(stdout io.Writer, facts repository.HistoricalReviewPreparation) {
	printHumanReceipt(stdout, "REVIEW SUBJECT", humanField{"REF", facts.REF}, humanField{"Subject without review", facts.SubjectSeal.String()})
	for _, head := range facts.OtherHeads {
		fmt.Fprintf(stdout, "Other HEAD: %s %s\n", head.REF, head.Seal)
	}
	for _, target := range facts.Targets {
		printHumanReceipt(stdout, "CAUSE FACT", humanField{"Target", target.TargetSeal.String()}, humanField{"Revision state", string(target.RevisionState)}, humanField{"Draft", yesNo(target.Draft)})
		for _, id := range target.ComparedSeals {
			fmt.Fprintf(stdout, "  Comparison leaf: %s\n", id)
		}
	}
}
