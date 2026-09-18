package cli

import (
	"context"
	"flag"
	"fmt"
	"io"

	"github.com/mako10k/sealgraph/internal/repository"
)

func runTraceCorrespondence(ctx context.Context, workDir string, args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		return usageError(stderr, "trace correspondence requires put, show, list, or remove")
	}
	args, output, err := extractInspectionFormat(args, stdout)
	if err != nil {
		return usageError(stderr, "%v", err)
	}
	if len(args) == 0 {
		return usageError(stderr, "trace correspondence requires put, show, list, or remove")
	}
	repo, err := repository.OpenStandalone(workDir)
	if err != nil {
		return commandError(stderr, "trace correspondence", err)
	}
	if repo.Format() != 7 {
		return commandError(stderr, "trace correspondence", fmt.Errorf("trace correspondences require repository format 7"))
	}
	switch args[0] {
	case "put":
		return runTraceCorrespondencePut(ctx, workDir, repo, args[1:], output, stdout, stderr)
	case "show":
		return runTraceCorrespondenceShow(repo, args[1:], output, stdout, stderr)
	case "list":
		return runTraceCorrespondenceList(repo, args[1:], output, stdout, stderr)
	case "remove":
		return runTraceCorrespondenceRemove(ctx, repo, args[1:], output, stdout, stderr)
	default:
		return usageError(stderr, "unknown trace correspondence operation %q; expected put, show, list, or remove", args[0])
	}
}

func runTraceCorrespondencePut(ctx context.Context, workDir string, repo *repository.Repository, args []string, output inspectionOutput, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("trace correspondence put", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	var file singleString
	flags.Var(&file, "file", "declaration JSON file")
	if err := flags.Parse(args); err != nil {
		return flagUsageError(stderr, "trace correspondence put", err)
	}
	if flags.NArg() != 0 || !file.set || file.value == "" {
		return usageError(stderr, "trace correspondence put requires exactly one --file PATH")
	}
	data, err := readTraceInput(workDir, file.value)
	if err != nil {
		return commandError(stderr, "trace correspondence put", err)
	}
	options, err := repository.ParseTraceCorrespondenceInput(data)
	if err != nil {
		return commandError(stderr, "trace correspondence put", err)
	}
	result, err := repo.TraceCorrespondencePut(ctx, options)
	if err != nil {
		return commandError(stderr, "trace correspondence put", err)
	}
	return writeTraceCorrespondenceMutation(output, stdout, stderr, "put", result.ID, result.Changed)
}

func runTraceCorrespondenceShow(repo *repository.Repository, args []string, output inspectionOutput, stdout, stderr io.Writer) int {
	if len(args) != 1 {
		return usageError(stderr, "trace correspondence show requires exactly one ID")
	}
	record, err := repo.TraceCorrespondenceShow(args[0])
	if err != nil {
		return commandError(stderr, "trace correspondence show", err)
	}
	items := traceCorrespondenceRecordsJSON([]repository.TraceCorrespondenceRecord{{ID: args[0], Record: record}})
	return writeTraceCorrespondenceList(output, stdout, stderr, items)
}

func runTraceCorrespondenceList(repo *repository.Repository, args []string, output inspectionOutput, stdout, stderr io.Writer) int {
	if len(args) != 0 {
		return usageError(stderr, "trace correspondence list accepts no positional arguments")
	}
	records, err := repo.TraceCorrespondenceList()
	if err != nil {
		return commandError(stderr, "trace correspondence list", err)
	}
	return writeTraceCorrespondenceList(output, stdout, stderr, traceCorrespondenceRecordsJSON(records))
}

func runTraceCorrespondenceRemove(ctx context.Context, repo *repository.Repository, args []string, output inspectionOutput, stdout, stderr io.Writer) int {
	if len(args) != 1 {
		return usageError(stderr, "trace correspondence remove requires exactly one ID")
	}
	changed, err := repo.TraceCorrespondenceRemove(ctx, args[0])
	if err != nil {
		return commandError(stderr, "trace correspondence remove", err)
	}
	return writeTraceCorrespondenceMutation(output, stdout, stderr, "remove", args[0], changed)
}

func writeTraceCorrespondenceMutation(output inspectionOutput, stdout, stderr io.Writer, operation, id string, changed bool) int {
	if output.JSON {
		return writeInspectionJSON(stdout, stderr, "trace correspondence "+operation, traceCorrespondenceMutationJSON{Schema: "sealgraph/trace-correspondence-mutation/v1", Operation: operation, ID: id, Changed: changed})
	}
	fmt.Fprintf(stdout, "Trace correspondence %s: %s changed=%t\n", operation, id, changed)
	return 0
}

func writeTraceCorrespondenceList(output inspectionOutput, stdout, stderr io.Writer, records []traceCorrespondenceRecordJSON) int {
	if output.JSON {
		return writeInspectionJSON(stdout, stderr, "trace correspondence", traceCorrespondenceListJSON{Schema: "sealgraph/trace-correspondence-list/v1", Records: records})
	}
	for _, record := range records {
		fmt.Fprintf(stdout, "Trace correspondence %s snapshot=%s old=[%d,%d) current=%s ranges=%v deleted=%t reason=%s declared_at=%s\n", record.ID, record.Record.SourceSnapshot, record.Record.SourceStart, record.Record.SourceStart+record.Record.Length, record.Record.CurrentBlob, record.Record.CurrentRanges, record.Record.Deleted, quoteHumanString(record.Record.Reason), record.Record.DeclaredAt)
	}
	return 0
}
