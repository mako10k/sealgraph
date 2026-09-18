package cli

import (
	"context"
	"flag"
	"fmt"
	"io"
	"sort"

	"github.com/mako10k/sealgraph/internal/repository"
)

// runTraceSource implements the local source binding operations owned by the
// trace command. The repository methods perform path validation and the
// expected-old-path compare-and-swap; this layer only parses arguments and
// commits the versioned output contract.
func runTraceSource(ctx context.Context, workDir string, args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		return usageError(stderr, "trace source requires bind, rebind, unbind, show, or list")
	}
	switch args[0] {
	case "bind":
		return runTraceSourceBind(ctx, workDir, args[1:], stdout, stderr)
	case "rebind":
		return runTraceSourceRebind(ctx, workDir, args[1:], stdout, stderr)
	case "unbind":
		return runTraceSourceUnbind(ctx, workDir, args[1:], stdout, stderr)
	case "show":
		return runTraceSourceShow(workDir, args[1:], stdout, stderr)
	case "list":
		return runTraceSourceList(ctx, workDir, args[1:], stdout, stderr)
	default:
		return usageError(stderr, "unknown trace source operation %q; expected bind, rebind, unbind, show, or list", args[0])
	}
}

type traceSourceBindingJSON struct {
	Schema    string `json:"schema"`
	SourceKey string `json:"source_key"`
	Path      string `json:"path"`
}

type traceSourceMutationJSON struct {
	Schema    string                  `json:"schema"`
	Operation string                  `json:"operation"`
	SourceKey string                  `json:"source_key"`
	Before    *traceSourceBindingJSON `json:"before"`
	After     *traceSourceBindingJSON `json:"after"`
	Changed   bool                    `json:"changed"`
}

type traceSourceListJSON struct {
	Schema   string                   `json:"schema"`
	Bindings []traceSourceBindingJSON `json:"bindings"`
}

func traceSourceBindingJSONValue(binding repository.TraceSourceBinding) *traceSourceBindingJSON {
	return &traceSourceBindingJSON{Schema: "sealgraph/trace-source-binding/v1", SourceKey: binding.SourceKey, Path: binding.Path}
}

func traceSourceMutationOutput(stdout, stderr io.Writer, output inspectionOutput, command, operation, sourceKey string, before, after *repository.TraceSourceBinding, changed bool) int {
	var beforeJSON, afterJSON *traceSourceBindingJSON
	if before != nil {
		beforeJSON = traceSourceBindingJSONValue(*before)
	}
	if after != nil {
		afterJSON = traceSourceBindingJSONValue(*after)
	}
	if output.JSON {
		return writeInspectionJSON(stdout, stderr, command, traceSourceMutationJSON{
			Schema: "sealgraph/trace-source-mutation/v1", Operation: operation, SourceKey: sourceKey,
			Before: beforeJSON, After: afterJSON, Changed: changed,
		})
	}
	if before == nil && after != nil {
		printHumanReceipt(stdout, "TRACE SOURCE BOUND", humanField{"Source key", sourceKey}, humanField{"Path", quoteHumanString(after.Path)}, humanField{"Changed", yesNo(changed)})
	} else if before != nil && after == nil {
		printHumanReceipt(stdout, "TRACE SOURCE UNBOUND", humanField{"Source key", sourceKey}, humanField{"Previous path", quoteHumanString(before.Path)}, humanField{"Changed", yesNo(changed)})
	} else if before != nil && after != nil {
		printHumanReceipt(stdout, "TRACE SOURCE REBOUND", humanField{"Source key", sourceKey}, humanField{"Previous path", quoteHumanString(before.Path)}, humanField{"Path", quoteHumanString(after.Path)}, humanField{"Changed", yesNo(changed)})
	}
	return 0
}

func traceSourceFlags(args []string, stdout io.Writer, command string, configure func(*flag.FlagSet)) (string, *flag.FlagSet, inspectionOutput, error) {
	args, output, err := extractInspectionFormat(args, stdout)
	if err != nil {
		return "", nil, output, err
	}
	if len(args) == 0 {
		return "", nil, output, fmt.Errorf("%s requires exactly one source key", command)
	}
	flags := flag.NewFlagSet(command, flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	configure(flags)
	if err := flags.Parse(args[1:]); err != nil {
		return "", flags, output, err
	}
	if flags.NArg() != 0 {
		return "", flags, output, fmt.Errorf("%s accepts exactly one source key; unexpected argument %q", command, flags.Arg(0))
	}
	return args[0], flags, output, nil
}

func runTraceSourceBind(ctx context.Context, workDir string, args []string, stdout, stderr io.Writer) int {
	var path singleString
	key, _, output, err := traceSourceFlags(args, stdout, "trace source bind", func(f *flag.FlagSet) { f.Var(&path, "file", "source file path") })
	if err != nil {
		return usageDiagnostic(stderr, "trace source bind", err.Error(), "provide KEY and --file PATH")
	}
	if !path.set || path.value == "" || key == "" {
		return usageError(stderr, "trace source bind requires a non-empty KEY and --file PATH")
	}
	repo, err := repository.OpenStandalone(workDir)
	if err != nil {
		return commandError(stderr, "trace source bind", err)
	}
	var before *repository.TraceSourceBinding
	if existing, showErr := repo.TraceSourceShow(key); showErr == nil {
		before = &existing
	}
	after, err := repo.TraceSourceBind(ctx, key, path.value)
	if err != nil {
		return commandError(stderr, "trace source bind", err)
	}
	changed := before == nil || before.Path != after.Path
	return traceSourceMutationOutput(stdout, stderr, output, "trace source bind", "bind", key, before, &after, changed)
}

func runTraceSourceRebind(ctx context.Context, workDir string, args []string, stdout, stderr io.Writer) int {
	var oldPath, path singleString
	key, _, output, err := traceSourceFlags(args, stdout, "trace source rebind", func(f *flag.FlagSet) {
		f.Var(&oldPath, "from", "expected current source path")
		f.Var(&path, "file", "new source file path")
	})
	if err != nil {
		return usageDiagnostic(stderr, "trace source rebind", err.Error(), "provide KEY, --from OLD_PATH, and --file PATH")
	}
	if key == "" || !oldPath.set || oldPath.value == "" || !path.set || path.value == "" {
		return usageError(stderr, "trace source rebind requires KEY, --from OLD_PATH, and --file PATH")
	}
	repo, err := repository.OpenStandalone(workDir)
	if err != nil {
		return commandError(stderr, "trace source rebind", err)
	}
	before := repository.TraceSourceBinding{SourceKey: key, Path: oldPath.value}
	after, err := repo.TraceSourceRebind(ctx, key, oldPath.value, path.value)
	if err != nil {
		return commandError(stderr, "trace source rebind", err)
	}
	return traceSourceMutationOutput(stdout, stderr, output, "trace source rebind", "rebind", key, &before, &after, before.Path != after.Path)
}

func runTraceSourceUnbind(ctx context.Context, workDir string, args []string, stdout, stderr io.Writer) int {
	var oldPath singleString
	key, _, output, err := traceSourceFlags(args, stdout, "trace source unbind", func(f *flag.FlagSet) { f.Var(&oldPath, "from", "expected current source path") })
	if err != nil {
		return usageDiagnostic(stderr, "trace source unbind", err.Error(), "provide KEY and --from PATH")
	}
	if key == "" || !oldPath.set || oldPath.value == "" {
		return usageError(stderr, "trace source unbind requires KEY and --from PATH")
	}
	repo, err := repository.OpenStandalone(workDir)
	if err != nil {
		return commandError(stderr, "trace source unbind", err)
	}
	before := repository.TraceSourceBinding{SourceKey: key, Path: oldPath.value}
	if _, err := repo.TraceSourceUnbind(ctx, key, oldPath.value); err != nil {
		return commandError(stderr, "trace source unbind", err)
	}
	return traceSourceMutationOutput(stdout, stderr, output, "trace source unbind", "unbind", key, &before, nil, true)
}

func runTraceSourceShow(workDir string, args []string, stdout, stderr io.Writer) int {
	args, output, err := extractInspectionFormat(args, stdout)
	if err != nil {
		return usageError(stderr, "%v", err)
	}
	if len(args) != 1 || args[0] == "" {
		return usageError(stderr, "trace source show requires exactly one source key")
	}
	repo, err := repository.OpenStandalone(workDir)
	if err != nil {
		return commandError(stderr, "trace source show", err)
	}
	binding, err := repo.TraceSourceShow(args[0])
	if err != nil {
		return commandError(stderr, "trace source show", err)
	}
	return traceSourceListOutput(stdout, stderr, output, []repository.TraceSourceBinding{binding})
}

func runTraceSourceList(ctx context.Context, workDir string, args []string, stdout, stderr io.Writer) int {
	args, output, err := extractInspectionFormat(args, stdout)
	if err != nil {
		return usageError(stderr, "%v", err)
	}
	if len(args) != 0 {
		return usageError(stderr, "trace source list accepts no positional arguments")
	}
	repo, err := repository.OpenStandalone(workDir)
	if err != nil {
		return commandError(stderr, "trace source list", err)
	}
	bindings, err := repo.TraceSourceList()
	if err != nil {
		return commandError(stderr, "trace source list", err)
	}
	return traceSourceListOutput(stdout, stderr, output, bindings)
}

func traceSourceListOutput(stdout, stderr io.Writer, output inspectionOutput, bindings []repository.TraceSourceBinding) int {
	sort.Slice(bindings, func(i, j int) bool { return bindings[i].SourceKey < bindings[j].SourceKey })
	items := make([]traceSourceBindingJSON, 0, len(bindings))
	for _, binding := range bindings {
		items = append(items, *traceSourceBindingJSONValue(binding))
	}
	if output.JSON {
		return writeInspectionJSON(stdout, stderr, "trace source list", traceSourceListJSON{Schema: "sealgraph/trace-source-list/v1", Bindings: items})
	}
	for _, binding := range bindings {
		fmt.Fprintf(stdout, "%s -> %s\n", binding.SourceKey, quoteHumanString(binding.Path))
	}
	return 0
}
