package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/mako10k/sealgraph/internal/domain"
	"github.com/mako10k/sealgraph/internal/migration/format4extract"
	"github.com/mako10k/sealgraph/internal/pathmanifest"
	"github.com/mako10k/sealgraph/internal/repository"
)

// Version is replaced for release artifacts with -ldflags -X. Development
// builds retain an explicit non-release fallback.
var Version = "0.1.0-dev"

func RunStandalone(args []string, stdout, stderr io.Writer) int {
	workDir, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(stderr, "sealgraph: determine current directory: %v\n", err)
		return 3
	}
	return runStandaloneAtWithInput(workDir, args, os.Stdin, stdout, stderr)
}

func RunGitPlugin(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 || isHelp(args[0]) {
		printGitHelp(stdout)
		return 0
	}
	if args[0] == "--version" || args[0] == "version" {
		fmt.Fprintf(stdout, "git sealgraph %s\n", Version)
		return 0
	}
	fmt.Fprintf(stderr, "git sealgraph: command %q is not implemented; Git sidecar is outside the Phase 1 standalone scope\n", args[0])
	return 2
}

func runStandaloneAtWithInput(workDir string, args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	if len(args) >= 2 && args[0] == "__completion" && args[1] == "--bash" {
		return runBashCompletion(workDir, args[2:], stdout)
	}
	if len(args) == 0 || (len(args) == 1 && isHelp(args[0])) {
		printRootHelp(stdout)
		return 0
	}
	if args[0] == "help" {
		if printHelpTopic(stdout, args[1:]) {
			return 0
		}
		return unknownHelpTopic(stderr, args[1:])
	}
	if topic, ok := commandLocalHelpTopic(args); ok {
		printHelpTopic(stdout, topic)
		return 0
	}
	if args[0] == "--version" || args[0] == "version" {
		fmt.Fprintf(stdout, "sealgraph %s\n", Version)
		return 0
	}
	ctx := context.Background()
	if code, handled := runStandaloneMutation(ctx, workDir, args, stdin, stdout, stderr); handled {
		return code
	}
	return runStandaloneInspection(ctx, workDir, args, stdin, stdout, stderr)
}

func runStandaloneMutation(ctx context.Context, workDir string, args []string, stdin io.Reader, stdout, stderr io.Writer) (int, bool) {
	switch args[0] {
	case "init":
		return runInit(workDir, args[1:], stdout, stderr), true
	case "add":
		return runAdd(ctx, workDir, args[1:], stdin, stdout, stderr), true
	case "source":
		return runSource(ctx, workDir, args[1:], stdout, stderr), true
	case "link":
		return runLink(ctx, workDir, args[1:], stdout, stderr), true
	case "unlink":
		return runUnlink(ctx, workDir, args[1:], stdout, stderr), true
	case "tag":
		return runTag(ctx, workDir, args[1:], stdout, stderr), true
	case "mv":
		return runMove(ctx, workDir, args[1:], stdout, stderr), true
	case "candidate":
		return runCandidate(ctx, workDir, args[1:], stdout, stderr), true
	case "seal":
		return runSeal(ctx, workDir, args[1:], stdout, stderr), true
	case "recover":
		return runRecover(ctx, workDir, args[1:], stdout, stderr), true
	case "ref":
		return runREF(ctx, workDir, args[1:], stdout, stderr), true
	}
	return 0, false
}

func runREF(ctx context.Context, workDir string, args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		return usageError(stderr, "ref requires drop REF")
	}
	if args[0] != "drop" {
		return usageError(stderr, "unknown ref operation %q; expected drop", args[0])
	}
	if len(args) != 2 {
		return usageError(stderr, "ref drop requires exactly one REF")
	}
	repo, err := repository.OpenStandalone(workDir)
	if err != nil {
		return commandError(stderr, "ref drop", err)
	}
	result, err := repo.DropREF(ctx, args[1])
	if err != nil {
		return commandError(stderr, "ref drop", err)
	}
	if isHumanTerminal(stdout) {
		printHumanReceipt(stdout, "REF DROPPED",
			humanField{"REF", result.REF},
			humanField{"Previous head (prefix)", shortID(result.Head)},
			humanField{"Tags removed from namespace", strconv.Itoa(result.Tags)},
			humanField{"Recovery operation", result.OperationID},
		)
		return 0
	}
	fmt.Fprintf(stdout, "REF_DROPPED ref=%s head=%s tags=%d operation=%s\n", result.REF, result.Head, result.Tags, result.OperationID)
	return 0
}

func runSource(ctx context.Context, workDir string, args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		return usageError(stderr, "source requires one of bind, rebind, unbind, show, list, or compare")
	}
	switch args[0] {
	case "bind":
		return runSourceBind(ctx, workDir, args[1:], stdout, stderr)
	case "rebind":
		return runSourceRebind(ctx, workDir, args[1:], stdout, stderr)
	case "unbind":
		return runSourceUnbind(ctx, workDir, args[1:], stdout, stderr)
	case "show":
		return runSourceShow(workDir, args[1:], stdout, stderr)
	case "list":
		return runSourceList(workDir, args[1:], stdout, stderr)
	case "compare":
		return runSourceCompare(ctx, workDir, args[1:], stdout, stderr)
	default:
		return usageError(stderr, "source subcommand %q is unknown", args[0])
	}
}

func sourceREFAndFlags(args []string, command string, configure func(*flag.FlagSet)) (string, *flag.FlagSet, error) {
	if len(args) == 0 {
		return "", nil, fmt.Errorf("%s requires exactly one REF", command)
	}
	flags := flag.NewFlagSet(command, flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	configure(flags)
	if err := flags.Parse(args[1:]); err != nil {
		return "", flags, err
	}
	if flags.NArg() != 0 {
		return "", flags, fmt.Errorf("%s accepts exactly one REF; unexpected argument %q", command, flags.Arg(0))
	}
	return args[0], flags, nil
}

func runSourceBind(ctx context.Context, workDir string, args []string, stdout, stderr io.Writer) int {
	args, output, formatErr := extractInspectionFormat(args, stdout)
	if formatErr != nil {
		return usageError(stderr, "%v", formatErr)
	}
	var path singleString
	ref, _, err := sourceREFAndFlags(args, "source bind", func(flags *flag.FlagSet) {
		flags.Var(&path, "file", "working-directory-relative source path")
	})
	if err != nil {
		return usageDiagnostic(stderr, "source bind", err.Error(), "provide one REF and one --file PATH")
	}
	if !path.set || path.value == "" {
		return usageError(stderr, "source bind requires exactly one non-empty --file PATH")
	}
	repo, err := repository.OpenStandalone(workDir)
	if err != nil {
		return commandError(stderr, "source bind", err)
	}
	binding, err := repo.SourceBind(ctx, ref, path.value)
	if err != nil {
		return commandError(stderr, "source bind", err)
	}
	if output.JSON {
		return writeInspectionJSON(stdout, stderr, "source bind", sourceMutationJSON("bind", binding.REF, "", binding.Path))
	}
	if isHumanTerminal(stdout) {
		printHumanReceipt(stdout, "LOCAL SOURCE BOUND",
			humanField{"REF", binding.REF},
			humanField{"Source file", quoteHumanString(binding.Path)},
			humanField{"Candidate", "unchanged"},
		)
		return 0
	}
	fmt.Fprintf(stdout, "SOURCE_BOUND ref=%s before=null after=%s candidate=UNCHANGED\n", binding.REF, quoteHumanString(binding.Path))
	return 0
}

func runSourceRebind(ctx context.Context, workDir string, args []string, stdout, stderr io.Writer) int {
	args, output, formatErr := extractInspectionFormat(args, stdout)
	if formatErr != nil {
		return usageError(stderr, "%v", formatErr)
	}
	var oldPath, newPath singleString
	ref, _, err := sourceREFAndFlags(args, "source rebind", func(flags *flag.FlagSet) {
		flags.Var(&oldPath, "from", "required exact currently observed path")
		flags.Var(&newPath, "file", "new working-directory-relative source path")
	})
	if err != nil {
		return usageDiagnostic(stderr, "source rebind", err.Error(), "provide one REF, --from OLD_PATH, and --file NEW_PATH")
	}
	if !oldPath.set || oldPath.value == "" || !newPath.set || newPath.value == "" {
		return usageError(stderr, "source rebind requires non-empty --from OLD_PATH and --file NEW_PATH")
	}
	repo, err := repository.OpenStandalone(workDir)
	if err != nil {
		return commandError(stderr, "source rebind", err)
	}
	binding, err := repo.SourceRebind(ctx, ref, oldPath.value, newPath.value)
	if err != nil {
		return commandError(stderr, "source rebind", err)
	}
	if output.JSON {
		return writeInspectionJSON(stdout, stderr, "source rebind", sourceMutationJSON("rebind", binding.REF, oldPath.value, binding.Path))
	}
	if isHumanTerminal(stdout) {
		printHumanReceipt(stdout, "LOCAL SOURCE REBOUND",
			humanField{"REF", binding.REF},
			humanField{"Previous file", quoteHumanString(oldPath.value)},
			humanField{"Source file", quoteHumanString(binding.Path)},
			humanField{"Candidate", "unchanged"},
		)
		return 0
	}
	fmt.Fprintf(stdout, "SOURCE_REBOUND ref=%s before=%s after=%s candidate=UNCHANGED\n", binding.REF, quoteHumanString(oldPath.value), quoteHumanString(binding.Path))
	return 0
}

func runSourceUnbind(ctx context.Context, workDir string, args []string, stdout, stderr io.Writer) int {
	args, output, formatErr := extractInspectionFormat(args, stdout)
	if formatErr != nil {
		return usageError(stderr, "%v", formatErr)
	}
	var oldPath singleString
	ref, _, err := sourceREFAndFlags(args, "source unbind", func(flags *flag.FlagSet) {
		flags.Var(&oldPath, "from", "required exact currently observed path")
	})
	if err != nil {
		return usageDiagnostic(stderr, "source unbind", err.Error(), "provide one REF and --from PATH")
	}
	if !oldPath.set || oldPath.value == "" {
		return usageError(stderr, "source unbind requires exactly one non-empty --from PATH")
	}
	repo, err := repository.OpenStandalone(workDir)
	if err != nil {
		return commandError(stderr, "source unbind", err)
	}
	binding, err := repo.SourceUnbind(ctx, ref, oldPath.value)
	if err != nil {
		return commandError(stderr, "source unbind", err)
	}
	if output.JSON {
		return writeInspectionJSON(stdout, stderr, "source unbind", sourceMutationJSON("unbind", binding.REF, binding.Path, ""))
	}
	if isHumanTerminal(stdout) {
		printHumanReceipt(stdout, "LOCAL SOURCE UNBOUND",
			humanField{"REF", binding.REF},
			humanField{"Previous file", quoteHumanString(binding.Path)},
			humanField{"Candidate", "unchanged"},
		)
		return 0
	}
	fmt.Fprintf(stdout, "SOURCE_UNBOUND ref=%s before=%s after=null candidate=UNCHANGED\n", binding.REF, quoteHumanString(binding.Path))
	return 0
}

func runSourceShow(workDir string, args []string, stdout, stderr io.Writer) int {
	args, output, err := extractInspectionFormat(args, stdout)
	if err != nil {
		return usageError(stderr, "%v", err)
	}
	if len(args) != 1 {
		return usageError(stderr, "source show requires exactly one REF")
	}
	repo, err := repository.OpenStandalone(workDir)
	if err != nil {
		return commandError(stderr, "source show", err)
	}
	binding, err := repo.SourceShow(args[0])
	if err != nil {
		return commandError(stderr, "source show", err)
	}
	if output.JSON {
		return writeInspectionJSON(stdout, stderr, "source show", sourceJSON("show", []repository.SourceBinding{binding}))
	}
	printSources(stdout, []repository.SourceBinding{binding})
	return 0
}

func runSourceList(workDir string, args []string, stdout, stderr io.Writer) int {
	args, output, err := extractInspectionFormat(args, stdout)
	if err != nil {
		return usageError(stderr, "%v", err)
	}
	if len(args) != 0 {
		return usageError(stderr, "source list accepts no positional arguments")
	}
	repo, err := repository.OpenStandalone(workDir)
	if err != nil {
		return commandError(stderr, "source list", err)
	}
	bindings, err := repo.SourceList()
	if err != nil {
		return commandError(stderr, "source list", err)
	}
	if output.JSON {
		return writeInspectionJSON(stdout, stderr, "source list", sourceJSON("list", bindings))
	}
	printSources(stdout, bindings)
	return 0
}

func runSourceCompare(ctx context.Context, workDir string, args []string, stdout, stderr io.Writer) int {
	args, output, err := extractInspectionFormat(args, stdout)
	if err != nil {
		return usageError(stderr, "%v", err)
	}
	if len(args) != 1 {
		return usageError(stderr, "source compare requires exactly one REF")
	}
	if err := domain.ValidateREF(args[0]); err != nil {
		return usageError(stderr, "invalid source comparison REF: %v", err)
	}
	repo, err := repository.OpenStandalone(workDir)
	if err != nil {
		return commandError(stderr, "source compare", err)
	}
	result, err := repo.SourceCompare(ctx, args[0])
	if err != nil {
		return commandError(stderr, "source compare", err)
	}
	if output.JSON {
		return writeInspectionJSON(stdout, stderr, "source compare", sourceCompareJSON(result))
	}
	printSourceCompareHuman(stdout, result)
	return 0
}

func printSources(stdout io.Writer, bindings []repository.SourceBinding) {
	printSourcesHuman(stdout, bindings)
}

func runStandaloneInspection(ctx context.Context, workDir string, args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	switch args[0] {
	case "manifest":
		return runManifest(workDir, args[1:], stdout, stderr)
	case "show":
		return runShow(ctx, workDir, args[1:], stdout, stderr)
	case "log":
		return runLog(ctx, workDir, args[1:], stdout, stderr)
	case "linklog":
		return runLinkLog(ctx, workDir, args[1:], stdout, stderr)
	case "compare":
		return runCompare(ctx, workDir, args[1:], stdout, stderr)
	case "status":
		return runStatus(ctx, workDir, args[1:], stdout, stderr)
	case "stale":
		return runStale(ctx, workDir, args[1:], stdout, stderr)
	case "impact":
		return runImpact(ctx, workDir, args[1:], stdout, stderr)
	case "graph":
		return runGraph(ctx, workDir, args[1:], stdout, stderr)
	case "fsck":
		return runFsck(ctx, workDir, args[1:], stdout, stderr)
	case "migrate":
		return runMigrate(ctx, workDir, args[1:], stdout, stderr)
	case "load":
		return runLoad(ctx, workDir, args[1:], stdin, stdout, stderr)
	case "load-receipt":
		return runLoadReceipt(ctx, workDir, args[1:], stdout, stderr)
	default:
		if code, ok := gitMisuseDiagnostic(stderr, args); ok {
			return code
		}
		return unknownCommandError(stderr, args[0])
	}
}

func runMigrate(ctx context.Context, workDir string, args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		return usageError(stderr, "migrate requires extract")
	}
	if args[0] != "extract" {
		return usageError(stderr, "unknown migrate operation %q; expected extract", args[0])
	}
	return runMigrateExtract(ctx, workDir, args[1:], stdout, stderr)
}

func runMigrateExtract(ctx context.Context, workDir string, args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("migrate extract", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	var sourceFormat, format singleString
	flags.Var(&sourceFormat, "source-format", "required source repository format")
	flags.Var(&format, "format", "required migration document format")
	if err := flags.Parse(args); err != nil {
		return flagUsageError(stderr, "migrate extract", err)
	}
	if flags.NArg() != 0 {
		return usageError(stderr, "migrate extract accepts no positional arguments; unexpected argument %q", flags.Arg(0))
	}
	if !sourceFormat.set || sourceFormat.value != "4" {
		return usageError(stderr, "migrate extract requires --source-format 4")
	}
	if !format.set || format.value != "universal-blob-v1" {
		return usageError(stderr, "migrate extract requires --format universal-blob-v1")
	}
	result, err := format4extract.Extract(ctx, workDir)
	if err != nil {
		return commandError(stderr, "migrate extract", err)
	}
	written, err := stdout.Write(result.Document)
	if err != nil || written != len(result.Document) {
		if err == nil {
			err = io.ErrShortWrite
		}
		return commandError(stderr, "migrate extract", fmt.Errorf("write universal-blob-v1 document: %w", err))
	}
	for _, warning := range result.Warnings {
		fmt.Fprintln(stderr, warning)
	}
	return 0
}

func runManifest(workDir string, args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("manifest", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	var source singleString
	var files stringList
	flags.Var(&source, "source", "required explicit source identity")
	flags.Var(&files, "file", "explicit relative semantic/read path (repeatable)")
	if err := flags.Parse(args); err != nil {
		return flagUsageError(stderr, "manifest", err)
	}
	if flags.NArg() != 0 {
		return usageError(stderr, "manifest accepts no positional arguments; unexpected argument %q", flags.Arg(0))
	}
	if !source.set || source.value == "" {
		return usageError(stderr, "manifest requires exactly one non-empty --source SOURCE")
	}
	if len(files) == 0 {
		return usageError(stderr, "manifest requires at least one --file PATH")
	}
	output, err := pathmanifest.Build(workDir, source.value, files)
	if err != nil {
		return commandError(stderr, "manifest", err)
	}
	if _, err := stdout.Write(output); err != nil {
		return commandError(stderr, "manifest", fmt.Errorf("write path manifest: %w", err))
	}
	return 0
}

func runInit(workDir string, args []string, stdout, stderr io.Writer) int {
	if len(args) != 0 {
		return usageError(stderr, "init accepts no arguments")
	}
	result, err := repository.InitStandalone(workDir)
	if err != nil {
		return commandError(stderr, "init", err)
	}
	if isHumanTerminal(stdout) {
		printInitHuman(stdout, result)
		return 0
	}
	switch result.Outcome {
	case repository.InitInitialized:
		fmt.Fprintln(stdout, "INITIALIZED standalone repository runtime=index,locks")
	case repository.InitRuntimeBootstrapped:
		fmt.Fprintf(stdout, "BOOTSTRAPPED_RUNTIME %s\n", strings.Join(result.RuntimeDirectories, ","))
	case repository.InitAlreadyComplete:
		fmt.Fprintln(stdout, "ALREADY_COMPLETE")
	}
	return 0
}

func runLoad(ctx context.Context, workDir string, args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("load", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	var format singleString
	flags.Var(&format, "format", "required versioned dump format")
	if err := flags.Parse(args); err != nil {
		return flagUsageError(stderr, "load", err)
	}
	if flags.NArg() != 0 {
		return usageError(stderr, "load accepts no positional arguments; unexpected argument %q", flags.Arg(0))
	}
	if !format.set {
		return usageError(stderr, "load requires --format universal-blob-v1")
	}
	if format.value != "universal-blob-v1" {
		return usageError(stderr, "load format %q is unsupported; expected universal-blob-v1", format.value)
	}
	input, err := io.ReadAll(stdin)
	if err != nil {
		return commandError(stderr, "load", fmt.Errorf("read universal-blob-v1 input: %w", err))
	}
	result, err := repository.LoadUniversalBlobV1(ctx, workDir, input)
	if err != nil {
		return commandError(stderr, "load", err)
	}
	for _, warning := range result.Warnings {
		fmt.Fprintln(stderr, warning)
	}
	if written, err := stdout.Write(result.Receipt); err != nil || written != len(result.Receipt) {
		if err == nil {
			err = io.ErrShortWrite
		}
		return commandError(stderr, "load", fmt.Errorf("LOAD_PUBLISHED_RECEIPT_UNDELIVERED: target and durable receipt are valid; recover with load-receipt rather than retrying load: %w", err))
	}
	return 0
}

func runLoadReceipt(ctx context.Context, workDir string, args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("load-receipt", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	var sourceDigest singleString
	flags.Var(&sourceDigest, "source-document-sha256", "required source migration document digest")
	if err := flags.Parse(args); err != nil {
		return flagUsageError(stderr, "load-receipt", err)
	}
	if flags.NArg() != 0 {
		return usageError(stderr, "load-receipt accepts no positional arguments; unexpected argument %q", flags.Arg(0))
	}
	if !sourceDigest.set || sourceDigest.value == "" {
		return usageError(stderr, "load-receipt requires --source-document-sha256 HEX")
	}
	receipt, err := repository.RecoverUniversalLoadReceipt(ctx, workDir, sourceDigest.value)
	if err != nil {
		return commandError(stderr, "load-receipt", err)
	}
	if written, err := stdout.Write(receipt); err != nil || written != len(receipt) {
		if err == nil {
			err = io.ErrShortWrite
		}
		return commandError(stderr, "load-receipt", fmt.Errorf("write recovered receipt: %w", err))
	}
	return 0
}

type addCLIOptions struct {
	ref         string
	content     trackedString
	contentFile trackedString
	target      singleString
	previous    stringList
	messages    stringList
	noPrevious  singleBool
	root        singleBool
	nonRoot     singleBool
	clearCauses singleBool
	draft       singleBool
	bindSource  bool
}

func runAdd(ctx context.Context, workDir string, args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	if len(args) > 0 && (args[0] == "." || args[0] == "-A" || args[0] == "-u") {
		fmt.Fprintf(stderr, "error: 'add %s' assumes Git worktree-wide staging and is not supported\n", args[0])
		fmt.Fprintln(stderr, "reason: sealgraph add updates exactly one named REF candidate and never discovers files")
		fmt.Fprintln(stderr, "hint: retry with `sealgraph add REF` or `sealgraph add REF --content-file PATH`")
		fmt.Fprintln(stderr, "help: sealgraph help add")
		return 2
	}
	options, code := parseAddCLIOptions(args, stderr)
	if code != 0 {
		return code
	}
	cause, err := parseCauseInput(options.target, options.previous, options.messages, options.noPrevious)
	if err != nil {
		return usageDiagnostic(stderr, "add", err.Error(), "construct one complete Cause record from the add command help; no selector was inferred")
	}
	repo, err := repository.OpenStandalone(workDir)
	if err != nil {
		return commandError(stderr, "add", err)
	}
	if !options.content.set && (!options.contentFile.set || options.contentFile.value != "-") {
		return runLocalSourceAdd(ctx, repo, options, cause, stdout, stderr)
	}
	return runExplicitBytesAdd(ctx, repo, workDir, options, cause, stdin, stdout, stderr)
}

func parseAddCLIOptions(args []string, stderr io.Writer) (addCLIOptions, int) {
	if len(args) == 0 {
		return addCLIOptions{}, usageError(stderr, "add requires exactly one REF")
	}
	options := addCLIOptions{ref: args[0]}
	flags := flag.NewFlagSet("add", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	flags.Var(&options.content, "content", "exact content bytes supplied as a command argument")
	flags.Var(&options.contentFile, "content-file", "read exact content bytes from a regular file, or '-' for stdin")
	flags.Var(&options.target, "target", "one exact Cause target selector")
	flags.Var(&options.previous, "previous", "previous revision selector for that target (repeatable)")
	flags.Var(&options.messages, "m", "Cause Link message (repeatable)")
	flags.Var(&options.noPrevious, "no-previous", "explicitly assert no previous revision for the target")
	flags.Var(&options.root, "root", "declare a provenance root")
	flags.Var(&options.nonRoot, "non-root", "declare a non-root provenance generation")
	flags.Var(&options.clearCauses, "clear-cause-links", "remove all Cause Links atomically with --root")
	flags.Var(&options.draft, "draft", "mark the candidate draft")
	flags.BoolVar(&options.bindSource, "bind-source", false, "persist the named local source after candidate update")
	if err := flags.Parse(args[1:]); err != nil {
		return addCLIOptions{}, flagUsageError(stderr, "add", err)
	}
	if flags.NArg() != 0 {
		return addCLIOptions{}, usageError(stderr, "add accepts exactly one REF; unexpected argument %q", flags.Arg(0))
	}
	if options.content.set && options.contentFile.set {
		return addCLIOptions{}, usageError(stderr, "add accepts at most one of --content or --content-file")
	}
	if options.bindSource && (options.content.set || (options.contentFile.set && options.contentFile.value == "-")) {
		return addCLIOptions{}, usageError(stderr, "add --bind-source requires a named file source and cannot be used with --content or --content-file -")
	}
	for _, item := range []struct {
		name  string
		value singleBool
	}{{"--root", options.root}, {"--non-root", options.nonRoot}, {"--clear-cause-links", options.clearCauses}} {
		if item.value.set && !item.value.value {
			return addCLIOptions{}, usageError(stderr, "add %s=false is invalid; omit the flag instead of negating its explicit declaration", item.name)
		}
	}
	if options.root.set && options.nonRoot.set {
		return addCLIOptions{}, usageError(stderr, "add accepts exactly one of --root or --non-root")
	}
	if options.root.set && !options.clearCauses.value {
		return addCLIOptions{}, usageError(stderr, "add --root requires --clear-cause-links")
	}
	if options.clearCauses.set && !options.root.value {
		return addCLIOptions{}, usageError(stderr, "--clear-cause-links is valid only with --root")
	}
	return options, 0
}

func runLocalSourceAdd(ctx context.Context, repo *repository.Repository, options addCLIOptions, cause *repository.CauseInput, stdout, stderr io.Writer) int {
	path := ""
	if options.contentFile.set {
		path = options.contentFile.value
	}
	result, err := repo.AddLocalSource(ctx, repository.LocalSourceAddOptions{
		REF: options.ref, Path: path, BindSource: options.bindSource, PreserveSemantics: !options.contentFile.set,
		Cause: cause, Root: options.root.value, RootSet: options.root.set || options.nonRoot.set,
		ClearCauseLinks: options.clearCauses.value, Draft: options.draft.value, DraftSet: options.draft.set,
	})
	if err != nil {
		return commandError(stderr, "add", err)
	}
	candidate := result.Candidate
	nextSource := "ready"
	if result.SourceBinding == "NONE" {
		nextSource = "requires-explicit-file-or-source-bind"
	}
	if isHumanTerminal(stdout) {
		printHumanReceipt(stdout, "CANDIDATE UPDATED",
			humanField{"REF", candidate.REF},
			humanField{"Content blob (prefix)", shortID(candidate.Content)},
			humanField{"Causes", strconv.Itoa(len(candidate.CauseLinks))},
			humanField{"Root boundary", yesNo(candidate.Root)},
			humanField{"Draft", yesNo(candidate.Draft)},
			humanField{"Source mode", result.SourceMode},
			humanField{"Source file", quoteHumanString(result.SourcePath)},
			humanField{"Source binding", strings.ToLower(result.SourceBinding)},
			humanField{"Next refresh", strings.ReplaceAll(nextSource, "-", " ")},
		)
		return 0
	}
	fmt.Fprintf(stdout, "CANDIDATE %s content=%s cause_links=%d root=%t draft=%t source_mode=%s source_path=%s source_binding=%s next_source=%s\n", candidate.REF, candidate.Content, len(candidate.CauseLinks), candidate.Root, candidate.Draft, result.SourceMode, quoteHumanString(result.SourcePath), result.SourceBinding, nextSource)
	return 0
}

func runExplicitBytesAdd(ctx context.Context, repo *repository.Repository, workDir string, options addCLIOptions, cause *repository.CauseInput, stdin io.Reader, stdout, stderr io.Writer) int {
	contentBytes := []byte(options.content.value)
	if options.contentFile.set {
		var err error
		contentBytes, err = readContentInput(workDir, options.contentFile.value, stdin)
		if err != nil {
			return usageError(stderr, "invalid --content-file: %v", err)
		}
	}
	candidate, err := repo.Add(ctx, repository.AddOptions{REF: options.ref, Content: contentBytes, Cause: cause,
		Root: options.root.value, RootSet: options.root.set || options.nonRoot.set, ClearCauseLinks: options.clearCauses.value,
		Draft: options.draft.value, DraftSet: options.draft.set})
	if err != nil {
		return commandError(stderr, "add", err)
	}
	if isHumanTerminal(stdout) {
		printHumanReceipt(stdout, "CANDIDATE UPDATED",
			humanField{"REF", candidate.REF},
			humanField{"Content blob (prefix)", shortID(candidate.Content)},
			humanField{"Causes", strconv.Itoa(len(candidate.CauseLinks))},
			humanField{"Root boundary", yesNo(candidate.Root)},
			humanField{"Draft", yesNo(candidate.Draft)},
			humanField{"Source mode", "explicit bytes"},
			humanField{"Source binding", "none"},
		)
		return 0
	}
	fmt.Fprintf(stdout, "CANDIDATE %s content=%s cause_links=%d root=%t draft=%t source_mode=explicit-bytes source_binding=NONE\n", candidate.REF, candidate.Content, len(candidate.CauseLinks), candidate.Root, candidate.Draft)
	return 0
}

func readContentInput(workDir, source string, stdin io.Reader) ([]byte, error) {
	if source == "-" {
		data, err := io.ReadAll(stdin)
		if err != nil {
			return nil, fmt.Errorf("read stdin: %w", err)
		}
		return data, nil
	}
	if source == "" {
		return nil, fmt.Errorf("path is empty")
	}
	path := source
	if !filepath.IsAbs(path) {
		path = filepath.Join(workDir, path)
	}
	info, err := os.Lstat(path)
	if err != nil {
		return nil, fmt.Errorf("inspect %q: %w", source, err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return nil, fmt.Errorf("%q is not a regular non-symlink file", source)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %q: %w", source, err)
	}
	return data, nil
}

func runLink(ctx context.Context, workDir string, args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		return usageError(stderr, "link requires exactly one REF")
	}
	ref := args[0]
	flags := flag.NewFlagSet("link", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	var target singleString
	var previous, messages stringList
	var noPrevious singleBool
	flags.Var(&target, "target", "exact Cause target selector")
	flags.Var(&previous, "previous", "previous revision selector (repeatable)")
	flags.Var(&messages, "m", "Cause Link message (repeatable)")
	flags.Var(&noPrevious, "no-previous", "explicitly assert no previous revision")
	if err := flags.Parse(args[1:]); err != nil {
		return flagUsageError(stderr, "link", err)
	}
	if flags.NArg() != 0 {
		return usageError(stderr, "link accepts exactly one REF; unexpected argument %q", flags.Arg(0))
	}
	cause, err := parseCauseInput(target, previous, messages, noPrevious)
	if err != nil {
		return usageDiagnostic(stderr, "link", err.Error(), "construct one complete Cause record from the link command help; no selector was inferred")
	}
	if cause == nil {
		return usageError(stderr, "link requires exactly one --target and one of --previous or --no-previous")
	}
	repo, err := repository.OpenStandalone(workDir)
	if err != nil {
		return commandError(stderr, "link", err)
	}
	candidate, err := repo.Link(ctx, ref, *cause)
	if err != nil {
		return commandError(stderr, "link", err)
	}
	if isHumanTerminal(stdout) {
		printHumanReceipt(stdout, "CANDIDATE CAUSES UPDATED",
			humanField{"REF", candidate.REF},
			humanField{"Causes", strconv.Itoa(len(candidate.CauseLinks))},
		)
		return 0
	}
	fmt.Fprintf(stdout, "CANDIDATE %s cause_links=%d\n", candidate.REF, len(candidate.CauseLinks))
	return 0
}

func runUnlink(ctx context.Context, workDir string, args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		return usageError(stderr, "unlink requires exactly one candidate REF")
	}
	ref := args[0]
	flags := flag.NewFlagSet("unlink", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	var target singleString
	flags.Var(&target, "target", "exact Cause target selector")
	if err := flags.Parse(args[1:]); err != nil {
		return flagUsageError(stderr, "unlink", err)
	}
	if flags.NArg() != 0 {
		return usageError(stderr, "unlink accepts exactly one candidate REF; unexpected argument %q", flags.Arg(0))
	}
	if !target.set || target.value == "" {
		return usageError(stderr, "unlink requires exactly one --target TARGET_SELECTOR")
	}
	if _, err := repository.ParseSelector(target.value); err != nil {
		return usageError(stderr, "invalid --target %q: %v", target.value, err)
	}
	repo, err := repository.OpenStandalone(workDir)
	if err != nil {
		return commandError(stderr, "unlink", err)
	}
	candidate, err := repo.Unlink(ctx, ref, target.value)
	if err != nil {
		return commandError(stderr, "unlink", err)
	}
	if isHumanTerminal(stdout) {
		printHumanReceipt(stdout, "CANDIDATE CAUSE REMOVED",
			humanField{"REF", candidate.REF},
			humanField{"Causes remaining", strconv.Itoa(len(candidate.CauseLinks))},
		)
		return 0
	}
	fmt.Fprintf(stdout, "CANDIDATE %s cause_links=%d\n", candidate.REF, len(candidate.CauseLinks))
	return 0
}

func runTag(ctx context.Context, workDir string, args []string, stdout, stderr io.Writer) int {
	if len(args) != 1 && len(args) != 2 {
		return usageError(stderr, "tag requires REF [TAGNAME] or SELECTOR TAGNAME")
	}
	if len(args) == 1 {
		return runTagList(ctx, workDir, args[0], stdout, stderr)
	}
	selector, err := repository.ParseSelector(args[0])
	if err != nil {
		return usageError(stderr, "invalid tag selector: %v", err)
	}
	if selector.Kind == repository.SelectorGlobalSeal {
		return usageError(stderr, "tag target %q has no REF scope; use REF, REF@SEAL, or REF@TAG", args[0])
	}
	if err := domain.ValidateTagName(args[1]); err != nil {
		return usageError(stderr, "invalid TAGNAME: %v", err)
	}
	repo, err := repository.OpenStandalone(workDir)
	if err != nil {
		return commandError(stderr, "tag", err)
	}
	result, err := repo.CreateTag(ctx, args[0], args[1])
	if err != nil {
		return commandError(stderr, "tag", err)
	}
	if isHumanTerminal(stdout) {
		printHumanReceipt(stdout, "TAG CREATED",
			humanField{"REF", result.REF},
			humanField{"Tag name", quoteHumanString(result.Name)},
			humanField{"Seal ID (prefix)", shortID(result.Seal)},
			humanField{"Recovery operation", result.OperationID},
		)
		return 0
	}
	fmt.Fprintf(stdout, "TAGGED %s %s %s", result.REF, strconv.Quote(result.Name), result.Seal)
	printOperationID(stdout, result.OperationID)
	return 0
}

func runTagList(ctx context.Context, workDir, ref string, stdout, stderr io.Writer) int {
	if err := domain.ValidateREF(ref); err != nil {
		return usageError(stderr, "invalid tag REF: %v", err)
	}
	repo, err := repository.OpenStandalone(workDir)
	if err != nil {
		return commandError(stderr, "tag", err)
	}
	tags, err := repo.Tags(ctx, ref)
	if err != nil {
		return commandError(stderr, "tag", err)
	}
	if isHumanTerminal(stdout) {
		humanTags := make([]humanTag, 0, len(tags))
		for _, tag := range tags {
			humanTags = append(humanTags, humanTag{name: tag.Name, seal: tag.Seal})
		}
		printTagsHuman(stdout, ref, humanTags)
		return 0
	}
	for _, tag := range tags {
		fmt.Fprintf(stdout, "TAG %s %s %s\n", ref, strconv.Quote(tag.Name), tag.Seal)
	}
	return 0
}

func runMove(ctx context.Context, workDir string, args []string, stdout, stderr io.Writer) int {
	if len(args) != 2 {
		return usageError(stderr, "mv requires exactly OLD_REF NEW_REF")
	}
	for _, ref := range args {
		if err := domain.ValidateREF(ref); err != nil {
			return usageError(stderr, "invalid mv REF %q: %v", ref, err)
		}
	}
	if args[0] == args[1] {
		return usageError(stderr, "mv source and destination are both %s; choose a different absent destination", args[0])
	}
	repo, err := repository.OpenStandalone(workDir)
	if err != nil {
		return commandError(stderr, "mv", err)
	}
	result, err := repo.MoveREF(ctx, args[0], args[1])
	if err != nil {
		return commandError(stderr, "mv", err)
	}
	if isHumanTerminal(stdout) {
		printHumanReceipt(stdout, "REF MOVED",
			humanField{"From", result.OldREF},
			humanField{"To", result.NewREF},
			humanField{"Head Seal ID (prefix)", shortID(result.Head)},
			humanField{"Tags moved", strconv.Itoa(result.Tags)},
			humanField{"Recovery operation", result.OperationID},
		)
		return 0
	}
	fmt.Fprintf(stdout, "MOVED %s %s %s tags=%d", result.OldREF, result.NewREF, result.Head, result.Tags)
	printOperationID(stdout, result.OperationID)
	return 0
}

func runSeal(ctx context.Context, workDir string, args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		return usageError(stderr, "seal requires exactly one REF")
	}
	if args[0] == "--all" {
		return usageError(stderr, "seal --all is forbidden; seal exactly one REF explicitly")
	}
	if len(args) != 1 {
		return usageError(stderr, "seal accepts exactly one REF; unexpected additional REF or argument %q", args[1])
	}
	ref := args[0]
	repo, err := repository.OpenStandalone(workDir)
	if err != nil {
		return commandError(stderr, "seal", err)
	}
	result, err := repo.Seal(ctx, ref)
	if err != nil {
		return commandError(stderr, "seal", err)
	}
	if isHumanTerminal(stdout) {
		printHumanReceipt(stdout, "SEALED",
			humanField{"REF", ref},
			humanField{"Seal ID (prefix)", shortID(result.ID)},
			humanField{"Recovery operation", result.OperationID},
		)
		return 0
	}
	fmt.Fprintf(stdout, "SEALED %s %s operation=%s\n", ref, result.ID, result.OperationID)
	return 0
}

func printOperationID(stdout io.Writer, id string) {
	if id != "" {
		fmt.Fprintf(stdout, " operation=%s", id)
	}
	fmt.Fprintln(stdout)
}

func runRecover(ctx context.Context, workDir string, args []string, stdout, stderr io.Writer) int {
	args, output, err := extractInspectionFormat(args, stdout)
	if err != nil {
		return usageError(stderr, "%v", err)
	}
	if len(args) == 0 {
		return usageError(stderr, "recover requires show [OPERATION_ID] or one exact OPERATION_ID")
	}
	repo, err := repository.OpenStandalone(workDir)
	if err != nil {
		return commandError(stderr, "recover", err)
	}
	if args[0] == "show" {
		if len(args) > 2 {
			return usageError(stderr, "recover show accepts at most one exact OPERATION_ID")
		}
		id := ""
		if len(args) == 2 {
			id = args[1]
		}
		inspections, err := repo.RecoveryShow(ctx, id)
		if err != nil {
			return commandError(stderr, "recover show", err)
		}
		if output.JSON {
			return writeInspectionJSON(stdout, stderr, "recover show", recoveryInspectionsJSON(inspections))
		}
		printRecoveryInspectionsHuman(stdout, inspections)
		return 0
	}
	if len(args) != 1 {
		return usageError(stderr, "recover requires exactly one full OPERATION_ID")
	}
	result, err := repo.Recover(ctx, args[0])
	if err != nil {
		return commandError(stderr, "recover", err)
	}
	if output.JSON {
		return writeInspectionJSON(stdout, stderr, "recover", map[string]any{"schema": "sealgraph/recover/v1", "operation_id": result.ID, "kind": result.Kind, "result": "RECOVERED"})
	}
	if isHumanTerminal(stdout) {
		printHumanReceipt(stdout, "RECOVERED",
			humanField{"Operation ID", result.ID},
			humanField{"Kind", result.Kind},
		)
		return 0
	}
	fmt.Fprintf(stdout, "RECOVERED operation=%s kind=%s\n", result.ID, result.Kind)
	return 0
}

func runShow(ctx context.Context, workDir string, args []string, stdout, stderr io.Writer) int {
	args, output, err := extractInspectionFormat(args, stdout)
	if err != nil {
		return usageError(stderr, "%v", err)
	}
	if len(args) == 0 {
		return usageError(stderr, "show requires exactly one REF or REF@SEAL")
	}
	selector := args[0]
	flags := flag.NewFlagSet("show", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	rawContent := flags.Bool("raw-content", false, "write exact content bytes only")
	if err := flags.Parse(args[1:]); err != nil {
		return flagUsageError(stderr, "show", err)
	}
	if flags.NArg() != 0 {
		return usageError(stderr, "show accepts exactly one REF or REF@SEAL; unexpected argument %q", flags.Arg(0))
	}
	if _, err := repository.ParseSelector(selector); err != nil {
		return usageDiagnostic(stderr, "show", fmt.Sprintf("invalid selector: %v", err), "use REF, @SEAL_TOKEN, or REF@TOKEN; inspect the exact grammar with `sealgraph help selectors`")
	}
	if output.Explicit && output.JSON && *rawContent {
		return usageError(stderr, "show --format json and --raw-content are mutually exclusive")
	}
	repo, err := repository.OpenStandalone(workDir)
	if err != nil {
		return commandError(stderr, "show", err)
	}
	result, err := repo.Show(ctx, selector)
	if err != nil {
		return commandError(stderr, "show", err)
	}
	if *rawContent {
		return writeRawContent(stdout, stderr, "show", result.Content)
	}
	if output.JSON && !*rawContent {
		return writeInspectionJSON(stdout, stderr, "show", showJSON(result))
	}
	printShowHuman(stdout, result)
	return 0
}

func runCandidate(ctx context.Context, workDir string, args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		return usageError(stderr, "candidate requires show, compare, or discard")
	}
	switch args[0] {
	case "show":
		return runCandidateShow(ctx, workDir, args[1:], stdout, stderr)
	case "compare":
		return runCandidateCompare(ctx, workDir, args[1:], stdout, stderr)
	case "diff":
		return gitCandidateDiffDiagnostic(stderr)
	case "discard":
		return runCandidateDiscard(ctx, workDir, args[1:], stdout, stderr)
	default:
		return usageError(stderr, "unknown candidate operation %q; expected show, compare, or discard", args[0])
	}
}

func runCandidateShow(ctx context.Context, workDir string, args []string, stdout, stderr io.Writer) int {
	args, output, formatErr := extractInspectionFormat(args, stdout)
	if formatErr != nil {
		return usageError(stderr, "%v", formatErr)
	}
	if len(args) == 0 {
		return usageError(stderr, "candidate show requires exactly one logical REF")
	}
	ref := args[0]
	if err := domain.ValidateREF(ref); err != nil {
		return usageError(stderr, "invalid candidate REF: %v", err)
	}
	flags := flag.NewFlagSet("candidate show", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	rawContent := flags.Bool("raw-content", false, "write exact candidate content bytes only")
	if err := flags.Parse(args[1:]); err != nil {
		return flagUsageError(stderr, "candidate show", err)
	}
	if flags.NArg() != 0 {
		return usageError(stderr, "candidate show accepts exactly one REF; unexpected argument %q", flags.Arg(0))
	}
	if output.Explicit && output.JSON && *rawContent {
		return usageError(stderr, "candidate show --format json and --raw-content are mutually exclusive")
	}
	repo, err := repository.OpenStandalone(workDir)
	if err != nil {
		return commandError(stderr, "candidate show", err)
	}
	inspection, err := repo.InspectCandidate(ctx, ref)
	if err != nil {
		return commandError(stderr, "candidate show", err)
	}
	if *rawContent {
		return writeRawContent(stdout, stderr, "candidate show", inspection.Content)
	}
	if output.JSON {
		return writeInspectionJSON(stdout, stderr, "candidate show", candidateShowJSON(inspection))
	}
	printCandidateInspection(stdout, inspection)
	return 0
}

func runCandidateCompare(ctx context.Context, workDir string, args []string, stdout, stderr io.Writer) int {
	args, output, formatErr := extractInspectionFormat(args, stdout)
	if formatErr != nil {
		return usageError(stderr, "%v", formatErr)
	}
	if len(args) != 1 {
		return usageError(stderr, "candidate compare requires exactly one logical REF")
	}
	ref := args[0]
	if err := domain.ValidateREF(ref); err != nil {
		return usageError(stderr, "invalid candidate REF: %v", err)
	}
	repo, err := repository.OpenStandalone(workDir)
	if err != nil {
		return commandError(stderr, "candidate compare", err)
	}
	result, err := repo.DiffCandidate(ctx, ref)
	if err != nil {
		return commandError(stderr, "candidate compare", err)
	}
	if output.JSON {
		return writeInspectionJSON(stdout, stderr, "candidate compare", candidateCompareJSON(result))
	}
	printCandidateDiff(stdout, result)
	return 0
}

func runCandidateDiscard(ctx context.Context, workDir string, args []string, stdout, stderr io.Writer) int {
	if len(args) != 1 {
		return usageError(stderr, "candidate discard requires exactly one logical REF")
	}
	ref := args[0]
	if err := domain.ValidateREF(ref); err != nil {
		return usageError(stderr, "invalid candidate REF: %v", err)
	}
	repo, err := repository.OpenStandalone(workDir)
	if err != nil {
		return commandError(stderr, "candidate discard", err)
	}
	if err := repo.DiscardCandidate(ctx, ref); err != nil {
		return commandError(stderr, "candidate discard", err)
	}
	if isHumanTerminal(stdout) {
		printHumanReceipt(stdout, "CANDIDATE DISCARDED", humanField{"REF", ref})
		return 0
	}
	fmt.Fprintf(stdout, "DISCARDED CANDIDATE %s\n", ref)
	return 0
}

func printCandidateInspection(stdout io.Writer, inspection repository.CandidateInspection) {
	printCandidateInspectionHuman(stdout, inspection)
}

func printCandidateDiff(stdout io.Writer, result repository.CandidateDiffResult) {
	printCandidateDiffHuman(stdout, result)
}

func runLog(ctx context.Context, workDir string, args []string, stdout, stderr io.Writer) int {
	args, output, err := extractInspectionFormat(args, stdout)
	if err != nil {
		return usageError(stderr, "%v", err)
	}
	flags := flag.NewFlagSet("log", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	var allPaths singleBool
	var maxPaths singleString
	flags.Var(&allPaths, "all-paths", "show bounded maximal revision paths")
	flags.Var(&maxPaths, "max-paths", "positive path limit; valid only with --all-paths")
	if err := flags.Parse(args); err != nil {
		return flagUsageError(stderr, "log", err)
	}
	if flags.NArg() != 1 {
		return usageError(stderr, "log requires exactly one current logical REF")
	}
	ref := flags.Arg(0)
	if err := domain.ValidateREF(ref); err != nil {
		return usageError(stderr, "invalid log REF: %v", err)
	}
	limit, err := parseImpactLimit(allPaths.value, maxPaths)
	if err != nil {
		return usageError(stderr, "%v", err)
	}
	repo, err := repository.OpenStandalone(workDir)
	if err != nil {
		return commandError(stderr, "log", err)
	}
	result, err := repo.Log(ctx, ref, allPaths.value, limit)
	if err != nil {
		return commandError(stderr, "log", err)
	}
	if output.JSON {
		return writeInspectionJSON(stdout, stderr, "log", logJSON(result))
	}
	printLogHuman(stdout, result)
	return 0
}

func runLinkLog(ctx context.Context, workDir string, args []string, stdout, stderr io.Writer) int {
	args, output, err := extractInspectionFormat(args, stdout)
	if err != nil {
		return usageError(stderr, "%v", err)
	}
	if len(args) == 0 {
		return usageError(stderr, "linklog requires exactly one current logical REF")
	}
	if err := domain.ValidateREF(args[0]); err != nil {
		return usageError(stderr, "invalid linklog REF: %v", err)
	}
	flags := flag.NewFlagSet("linklog", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	var upstream singleString
	flags.Var(&upstream, "upstream", "show changes involving one exact upstream Seal selector")
	if err := flags.Parse(args[1:]); err != nil {
		return flagUsageError(stderr, "linklog", err)
	}
	if flags.NArg() != 0 {
		return usageError(stderr, "linklog accepts exactly one REF; unexpected argument %q", flags.Arg(0))
	}
	repo, err := repository.OpenStandalone(workDir)
	if err != nil {
		return commandError(stderr, "linklog", err)
	}
	if upstream.set {
		if _, err := repository.ParseSelector(upstream.value); err != nil {
			return usageError(stderr, "invalid linklog upstream selector: %v", err)
		}
	}
	result, err := repo.LinkLog(ctx, args[0], upstream.value)
	if err != nil {
		return commandError(stderr, "linklog", err)
	}
	if output.JSON {
		return writeInspectionJSON(stdout, stderr, "linklog", linkLogJSON(result))
	}
	printLinkLogHuman(stdout, result)
	return 0
}

func runCompare(ctx context.Context, workDir string, args []string, stdout, stderr io.Writer) int {
	args, output, err := extractInspectionFormat(args, stdout)
	if err != nil {
		return usageError(stderr, "%v", err)
	}
	if len(args) != 2 {
		return usageError(stderr, "SECOND_SELECTOR_REQUIRED: format-5 compare requires exactly two explicit Seal selectors")
	}
	for _, arg := range args {
		if _, err := repository.ParseSelector(arg); err != nil {
			return usageError(stderr, "invalid compare selector %q: %v", arg, err)
		}
	}
	repo, err := repository.OpenStandalone(workDir)
	if err != nil {
		return commandError(stderr, "compare", err)
	}
	result, err := repo.DiffSelectors(ctx, args[0], args[1])
	if err != nil {
		return commandError(stderr, "compare", err)
	}
	if output.JSON {
		return writeInspectionJSON(stdout, stderr, "compare", compareJSON(result))
	}
	printSealDiffHuman(stdout, result)
	return 0
}

const contentPreviewLimit = 256

func quoteHumanString(value string) string { return quoteHumanBytes([]byte(value)) }

func quoteHumanBytes(value []byte) string {
	const hex = "0123456789abcdef"
	var builder strings.Builder
	builder.Grow(len(value) + 2)
	builder.WriteByte('"')
	for _, current := range value {
		switch current {
		case '"':
			builder.WriteString(`\"`)
		case '\\':
			builder.WriteString(`\\`)
		case '\n':
			builder.WriteString(`\n`)
		case '\r':
			builder.WriteString(`\r`)
		case '\t':
			builder.WriteString(`\t`)
		default:
			if current >= 0x20 && current <= 0x7e {
				builder.WriteByte(current)
				continue
			}
			builder.WriteString(`\x`)
			builder.WriteByte(hex[current>>4])
			builder.WriteByte(hex[current&0x0f])
		}
	}
	builder.WriteByte('"')
	return builder.String()
}

func writeRawContent(stdout io.Writer, stderr io.Writer, command string, content []byte) int {
	if _, err := stdout.Write(content); err != nil {
		return commandError(stderr, command, fmt.Errorf("write raw content: %w", err))
	}
	return 0
}

func runStatus(ctx context.Context, workDir string, args []string, stdout, stderr io.Writer) int {
	args, output, err := extractInspectionFormat(args, stdout)
	if err != nil {
		return usageError(stderr, "%v", err)
	}
	if len(args) > 1 {
		return usageError(stderr, "status accepts at most one REF")
	}
	ref := ""
	if len(args) == 1 {
		ref = args[0]
	}
	repo, err := repository.OpenStandalone(workDir)
	if err != nil {
		return commandError(stderr, "status", err)
	}
	statuses, err := repo.Status(ctx, ref)
	if err != nil {
		return commandError(stderr, "status", err)
	}
	if output.JSON {
		return writeInspectionJSON(stdout, stderr, "status", statusesJSON("sealgraph/status/v3", statuses, false, false))
	}
	printStatusesHuman(stdout, "SEALED_STATE", statuses)
	return 0
}

func runStale(ctx context.Context, workDir string, args []string, stdout, stderr io.Writer) int {
	if len(args) == 1 && isHelp(args[0]) {
		printStaleHelp(stdout)
		return 0
	}
	args, output, formatErr := extractInspectionFormat(args, stdout)
	if formatErr != nil {
		return usageError(stderr, "%v", formatErr)
	}
	flags := flag.NewFlagSet("stale", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	var frontier singleBool
	var refsOnly singleBool
	var scan singleBool
	flags.Var(&frontier, "frontier", "select the upstream-most stale review frontier")
	flags.Var(&refsOnly, "refs-only", "emit one logical REF per line")
	flags.Var(&scan, "scan", "bypass cache and rebuild the canonical revision index")
	if err := flags.Parse(args); err != nil {
		return flagUsageError(stderr, "stale", err)
	}
	if flags.NArg() != 0 {
		return usageError(stderr, "stale accepts no positional arguments; unexpected argument %q", flags.Arg(0))
	}
	if output.Explicit && output.JSON && refsOnly.value {
		return usageError(stderr, "stale --format json and --refs-only are mutually exclusive")
	}
	repo, err := repository.OpenStandalone(workDir)
	if err != nil {
		return commandError(stderr, "stale", err)
	}
	statuses, warning, err := repo.Stale(ctx, frontier.value, scan.value)
	if err != nil {
		return commandError(stderr, "stale", err)
	}
	if warning != "" {
		fmt.Fprintf(stderr, "sealgraph stale: warning: %s\n", warning)
	}
	if refsOnly.value {
		for _, status := range statuses {
			fmt.Fprintln(stdout, status.REF)
		}
		return 0
	}
	if output.JSON && !refsOnly.value {
		return writeInspectionJSON(stdout, stderr, "stale", statusesJSON("sealgraph/stale/v2", statuses, frontier.value, scan.value))
	}
	printStatusesHuman(stdout, "STALE_REVIEW_STATE", statuses)
	return 0
}

func runImpact(ctx context.Context, workDir string, args []string, stdout, stderr io.Writer) int {
	args, output, formatErr := extractInspectionFormat(args, stdout)
	if formatErr != nil {
		return usageError(stderr, "%v", formatErr)
	}
	flags := flag.NewFlagSet("impact", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	var allPaths singleBool
	var maxPaths singleString
	var assertedBy stringList
	flags.Var(&allPaths, "all-paths", "show bounded distinct simple Cause paths")
	flags.Var(&maxPaths, "max-paths", "positive path limit per downstream Seal; valid only with --all-paths")
	flags.Var(&assertedBy, "asserted-by", "revision assertion observer selector (repeatable)")
	if err := flags.Parse(args); err != nil {
		return flagUsageError(stderr, "impact", err)
	}
	if flags.NArg() != 1 {
		return usageError(stderr, "impact requires exactly one Seal selector")
	}
	selector := flags.Arg(0)
	if _, err := repository.ParseSelector(selector); err != nil {
		return usageError(stderr, "invalid impact selector: %v", err)
	}
	limit, err := parseImpactLimit(allPaths.value, maxPaths)
	if err != nil {
		hint := "add --all-paths before --max-paths, or omit --max-paths"
		if maxPaths.set && selector != "" {
			hint = fmt.Sprintf("use `sealgraph impact --all-paths --max-paths %s %s`", maxPaths.value, selector)
		}
		return usageDiagnostic(stderr, "impact", err.Error(), hint)
	}
	repo, err := repository.OpenStandalone(workDir)
	if err != nil {
		return commandError(stderr, "impact", err)
	}
	result, err := repo.Impact(ctx, selector, assertedBy, allPaths.value, limit)
	if err != nil {
		return commandError(stderr, "impact", err)
	}
	if output.JSON {
		return writeInspectionJSON(stdout, stderr, "impact", impactJSON(result))
	}
	printImpactsHuman(stdout, result)
	return 0
}

func runGraph(ctx context.Context, workDir string, args []string, stdout, stderr io.Writer) int {
	args, output, err := extractInspectionFormat(args, stdout)
	if err != nil {
		return usageError(stderr, "%v", err)
	}
	if len(args) != 0 {
		return usageError(stderr, "graph accepts no arguments")
	}
	repo, err := repository.OpenStandalone(workDir)
	if err != nil {
		return commandError(stderr, "graph", err)
	}
	nodes, err := repo.Graph(ctx)
	if err != nil {
		return commandError(stderr, "graph", err)
	}
	if output.JSON {
		return writeInspectionJSON(stdout, stderr, "graph", graphJSON(nodes))
	}
	printGraphHuman(stdout, nodes)
	return 0
}

func runFsck(ctx context.Context, workDir string, args []string, stdout, stderr io.Writer) int {
	args, output, err := extractInspectionFormat(args, stdout)
	if err != nil {
		return usageError(stderr, "%v", err)
	}
	if len(args) != 0 {
		return usageError(stderr, "fsck accepts no positional arguments")
	}
	repo, err := repository.OpenStandalone(workDir)
	if err != nil {
		return commandError(stderr, "fsck", err)
	}
	report, err := repo.Fsck(ctx)
	if err != nil {
		return commandError(stderr, "fsck", err)
	}
	if output.JSON {
		return writeInspectionJSON(stdout, stderr, "fsck", fsckJSON(report))
	}
	printFsckHuman(stdout, report)
	return 0
}

func parseImpactLimit(allPaths bool, value singleString) (int, error) {
	if value.set && !allPaths {
		return 0, errors.New("--max-paths is valid only with --all-paths")
	}
	if !value.set {
		return 100, nil
	}
	parsed, err := strconv.Atoi(value.value)
	if err != nil || parsed <= 0 {
		return 0, fmt.Errorf("--max-paths requires a positive integer, got %q", value.value)
	}
	return parsed, nil
}

func parseCauseInput(target singleString, previous, messages []string, noPrevious singleBool) (*repository.CauseInput, error) {
	if noPrevious.set && !noPrevious.value {
		return nil, errors.New("--no-previous=false is invalid; omit it or assert explicit previous revisions")
	}
	if !target.set {
		if len(previous) != 0 || len(messages) != 0 || noPrevious.set {
			return nil, errors.New("--previous, --no-previous, and -m require exactly one --target")
		}
		return nil, nil
	}
	if target.value == "" {
		return nil, errors.New("--target requires a non-empty selector")
	}
	if _, err := repository.ParseSelector(target.value); err != nil {
		return nil, fmt.Errorf("invalid --target %q: %w", target.value, err)
	}
	if noPrevious.value == (len(previous) != 0) {
		return nil, errors.New("a target requires exactly one of at least one --previous or --no-previous")
	}
	for _, selector := range previous {
		if _, err := repository.ParseSelector(selector); err != nil {
			return nil, fmt.Errorf("invalid --previous %q: %w", selector, err)
		}
	}
	return &repository.CauseInput{Target: target.value, Previous: append([]string(nil), previous...), Messages: append([]string(nil), messages...)}, nil
}

type trackedString struct {
	value string
	set   bool
}

func (v *trackedString) String() string         { return v.value }
func (v *trackedString) Set(value string) error { v.value, v.set = value, true; return nil }

type singleString struct {
	value string
	set   bool
}

type singleBool struct {
	value bool
	set   bool
}

func (v *singleBool) String() string { return strconv.FormatBool(v.value) }
func (v *singleBool) IsBoolFlag() bool {
	return true
}
func (v *singleBool) Set(value string) error {
	if v.set {
		return fmt.Errorf("may be specified only once")
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return err
	}
	v.value, v.set = parsed, true
	return nil
}

func (v *singleString) String() string { return v.value }
func (v *singleString) Set(value string) error {
	if v.set {
		return fmt.Errorf("may be specified only once")
	}
	v.value, v.set = value, true
	return nil
}

type stringList []string

func (v *stringList) String() string         { return strings.Join(*v, ",") }
func (v *stringList) Set(value string) error { *v = append(*v, value); return nil }

func commandError(stderr io.Writer, command string, err error) int {
	message := err.Error()
	fmt.Fprintf(stderr, "error: sealgraph %s: %s\n", command, message)
	reason, hints := domainNavigation(command, message)
	if reason != "" {
		fmt.Fprintf(stderr, "reason: %s\n", reason)
	}
	for _, hint := range hints {
		fmt.Fprintf(stderr, "hint: %s\n", hint)
	}
	fmt.Fprintf(stderr, "help: sealgraph help %s\n", command)
	return 1
}

func usageError(stderr io.Writer, format string, args ...any) int {
	message := fmt.Sprintf(format, args...)
	topic := inferUsageTopic(message)
	return usageDiagnostic(stderr, topic, message, "construct the invocation from the command help; no value or REF was inferred")
}

func usageDiagnostic(stderr io.Writer, topic, message, hint string) int {
	fmt.Fprintf(stderr, "error: %s\n", message)
	if topic != "" {
		fmt.Fprintf(stderr, "usage: %s\n", commandUsage(topic))
	}
	if hint != "" {
		fmt.Fprintf(stderr, "hint: %s\n", hint)
	}
	fmt.Fprintf(stderr, "help: sealgraph help%s\n", helpSuffix(topic))
	return 2
}

func flagUsageError(stderr io.Writer, topic string, err error) int {
	return usageDiagnostic(stderr, topic, err.Error(), "check the option spelling, value, repeatability, and positional argument order")
}

func helpSuffix(topic string) string {
	if topic == "" {
		return ""
	}
	return " " + topic
}

func inferUsageTopic(message string) string {
	paths := make([]string, 0, len(commandHelpRegistry))
	for path := range commandHelpRegistry {
		paths = append(paths, path)
	}
	sort.Slice(paths, func(i, j int) bool { return len(paths[i]) > len(paths[j]) })
	for _, path := range paths {
		if strings.HasPrefix(message, path+" ") || strings.Contains(message, " "+path+" ") {
			return path
		}
	}
	return ""
}

func commandLocalHelpTopic(args []string) ([]string, bool) {
	if len(args) >= 2 && isHelp(args[1]) {
		if _, ok := commandHelpRegistry[args[0]]; ok {
			return []string{args[0]}, true
		}
	}
	if len(args) >= 3 && isHelp(args[2]) {
		if _, ok := commandHelpRegistry[args[0]+" "+args[1]]; ok {
			return []string{args[0], args[1]}, true
		}
	}
	return nil, false
}

func unknownHelpTopic(stderr io.Writer, path []string) int {
	text := strings.Join(path, " ")
	if text == "" {
		return usageDiagnostic(stderr, "", "help topic is empty", "list commands and topics with `sealgraph help`")
	}
	return usageDiagnostic(stderr, "", fmt.Sprintf("unknown help topic %q", text), "list commands and topics with `sealgraph help`")
}

func unknownCommandError(stderr io.Writer, command string) int {
	hint := "list available commands with `sealgraph help`"
	if suggestion := nearestCommand(command); suggestion != "" {
		fmt.Fprintf(stderr, "error: unknown command %q\n", command)
		fmt.Fprintf(stderr, "hint: possible command: %s (review it before running; it was not executed)\n", suggestion)
		fmt.Fprintf(stderr, "help: sealgraph help %s\n", suggestion)
		return 2
	}
	return usageDiagnostic(stderr, "", fmt.Sprintf("unknown command %q", command), hint)
}

func gitMisuseDiagnostic(stderr io.Writer, args []string) (int, bool) {
	if len(args) == 0 {
		return 0, false
	}
	command := args[0]
	switch command {
	case "diff":
		fmt.Fprintln(stderr, "error: 'diff' is Git-shaped vocabulary and is not a sealgraph command")
		if containsArg(args[1:], "--draft") {
			fmt.Fprintln(stderr, "reason: DRAFT is an identity-bearing provenance property, not mutable candidate state")
		} else if containsArg(args[1:], "--cached") || containsArg(args[1:], "--staged") {
			fmt.Fprintln(stderr, "reason: cached/staged assumes Git index semantics; sealgraph candidates and source bindings are distinct states")
		} else {
			fmt.Fprintln(stderr, "reason: sealgraph comparison reports semantic material and provenance rather than a Git patch")
		}
		fmt.Fprintln(stderr, "hint: compare immutable Seals with `sealgraph compare FROM_SELECTOR TO_SELECTOR`")
		fmt.Fprintln(stderr, "hint: compare candidate state with `sealgraph candidate compare REF`")
		fmt.Fprintln(stderr, "hint: compare a bound workfile with `sealgraph source compare REF`")
		fmt.Fprintln(stderr, "help: sealgraph help compare")
		return 2, true
	case "rm", "remove":
		fmt.Fprintf(stderr, "error: '%s' is a Git-shaped deletion operation and is not a sealgraph command\n", command)
		fmt.Fprintln(stderr, "reason: sealgraph does not delete workfiles or stage file deletion")
		fmt.Fprintln(stderr, "hint: discard only unsealed state with `sealgraph candidate discard REF`")
		fmt.Fprintln(stderr, "hint: inspect then remove only a binding with `sealgraph source show REF` and `sealgraph source unbind REF --from PATH`")
		fmt.Fprintln(stderr, "hint: drop only the logical REF handle with `sealgraph ref drop REF` (candidate and source binding must be handled separately)")
		fmt.Fprintln(stderr, "help: sealgraph help concepts")
		return 2, true
	case "commit":
		return gitConceptDiagnostic(stderr, "commit", "a Seal publication advances exactly one REF from one reviewed candidate", "sealgraph candidate compare REF", "sealgraph seal REF"), true
	case "checkout", "switch", "branch":
		return gitConceptDiagnostic(stderr, command, "a sealgraph REF is a movable logical identity, not a checked-out branch", "sealgraph status", "sealgraph show REF"), true
	case "reset", "restore", "clean":
		return gitConceptDiagnostic(stderr, command, "sealgraph has no combined worktree/index reset operation", "sealgraph candidate discard REF", "sealgraph source show REF"), true
	}
	return 0, false
}

func gitCandidateDiffDiagnostic(stderr io.Writer) int {
	fmt.Fprintln(stderr, "error: 'candidate diff' is retired Git-shaped vocabulary")
	fmt.Fprintln(stderr, "reason: candidate comparison is semantic pre-publication state comparison, not a Git patch")
	fmt.Fprintln(stderr, "hint: retry explicitly with `sealgraph candidate compare REF`")
	fmt.Fprintln(stderr, "help: sealgraph help candidate compare")
	return 2
}

func gitConceptDiagnostic(stderr io.Writer, command, reason, first, second string) int {
	fmt.Fprintf(stderr, "error: '%s' is Git-shaped vocabulary and is not a sealgraph command\n", command)
	fmt.Fprintf(stderr, "reason: %s\n", reason)
	fmt.Fprintf(stderr, "hint: inspect the intended state with `%s`\n", first)
	fmt.Fprintf(stderr, "hint: use the explicit sealgraph operation `%s` only after review\n", second)
	fmt.Fprintln(stderr, "help: sealgraph help concepts")
	return 2
}

func containsArg(args []string, target string) bool {
	for _, arg := range args {
		if arg == target {
			return true
		}
	}
	return false
}

func nearestCommand(input string) string {
	best, bestDistance := "", 3
	for path := range commandHelpRegistry {
		if strings.Contains(path, " ") {
			continue
		}
		distance := editDistance(input, path)
		if distance < bestDistance || (distance == bestDistance && path < best) {
			best, bestDistance = path, distance
		}
	}
	return best
}

func editDistance(left, right string) int {
	previous := make([]int, len(right)+1)
	for index := range previous {
		previous[index] = index
	}
	for i := 1; i <= len(left); i++ {
		current := make([]int, len(right)+1)
		current[0] = i
		for j := 1; j <= len(right); j++ {
			cost := 1
			if left[i-1] == right[j-1] {
				cost = 0
			}
			current[j] = min(current[j-1]+1, previous[j]+1, previous[j-1]+cost)
		}
		previous = current
	}
	return previous[len(right)]
}

func domainNavigation(command, message string) (string, []string) {
	if reason, hints, ok := localSourceNavigation(message); ok {
		return reason, hints
	}
	switch {
	case strings.Contains(message, "has no working candidate") || strings.Contains(message, "candidate not found"):
		if command == "candidate discard" {
			return "discard requires one existing candidate and never treats absence as successful", []string{"inspect current candidate and REF state with `sealgraph status` before choosing an explicit action"}
		}
		return "the operation requires explicit mutable candidate state for that REF", []string{"create or update it with `sealgraph add` or `sealgraph link`, or inspect the intended REF first"}
	case strings.Contains(message, "REF not found") || strings.Contains(message, "has no head or candidate"):
		return "the selected logical REF has no current repository state", []string{"inspect current state with `sealgraph status` or select an existing REF or explicit Seal selector"}
	case strings.Contains(message, "active revision leaf") || strings.Contains(message, "non-draft Cause") || strings.Contains(message, "Cause closure"):
		return "normal non-draft publication requires every reachable Cause to be a non-draft active revision leaf", []string{"inspect the named target with `sealgraph show @SEAL_ID` and current review state with `sealgraph stale --frontier`", "relink and review explicitly from upstream to downstream; if historical provenance is intentional, keep the candidate draft"}
	case strings.Contains(message, "open standalone repository"):
		return "this command requires a valid standalone .sealgraph repository", []string{"run `sealgraph init` only when initializing this directory; otherwise inspect and repair repository state explicitly"}
	case strings.Contains(message, "outside the observed revision closure"):
		return "REF@hex is scoped to revision assertions observed from the REF's current HEAD", []string{"use the unscoped @SEAL_TOKEN only when selecting a Seal outside that observed closure intentionally"}
	case strings.Contains(message, "changed while deriving") || strings.Contains(message, "changed or became unreadable"):
		return "the coherent repository observation changed before output could be committed", []string{"inspect current state and rerun the read-only command; no partial result is authoritative"}
	default:
		return "the requested operation could not satisfy its repository or provenance contract", []string{fmt.Sprintf("inspect the failure and the contract in `sealgraph help %s`; no repair was performed", command)}
	}
}

func localSourceNavigation(message string) (string, []string, bool) {
	switch {
	case strings.Contains(message, "has no local source binding") || strings.Contains(message, "local source binding not found"):
		return "the REF has no machine-local source binding; no file path was inferred", []string{"inspect all bindings with `sealgraph source list`", "bind one exact path with `sealgraph source bind REF --file PATH`, or provide `sealgraph add REF --content-file PATH` explicitly"}, true
	case strings.Contains(message, "is already") && strings.Contains(message, "source rebind"):
		return "source bind is create-only and does not silently retarget an existing binding", []string{"inspect the exact current path with `sealgraph source show REF`", "replace it with `sealgraph source rebind REF --from OLD_PATH --file NEW_PATH`"}, true
	case strings.Contains(message, "not expected") && strings.Contains(message, "local source"):
		return "the local source changed after the operator's observation; no compare-and-change mutation was applied", []string{"read the current binding with `sealgraph source show REF` and retry only with its exact path"}, true
	case strings.Contains(message, "is bound to") && strings.Contains(message, "explicit source"):
		return "candidate input and the next contentless-refresh source would disagree", []string{"inspect the binding with `sealgraph source show REF`", "use `source rebind` to change it atomically or `source unbind` before a one-time explicit file add"}, true
	case strings.Contains(message, "CHANGED_DURING_READ"):
		return "the named working file changed or was replaced during exact-byte observation", []string{"wait until the file is stable, inspect its current identity, and rerun; no plausible candidate or status result was produced"}, true
	case strings.Contains(message, "blocks REF-only move"):
		return "mv changes only the logical REF manifest and never moves a working file or local source binding", []string{"inspect the binding with `sealgraph source show REF`", "explicitly unbind it, move the REF, then bind the new REF to the intended path"}, true
	case strings.Contains(message, "binding was not published"):
		return "the candidate was published before the requested local binding and may already contain the new bytes", []string{"inspect `sealgraph candidate show REF` and `sealgraph source show REF` before retrying; do not assume both local files changed"}, true
	default:
		return "", nil, false
	}
}

func isHelp(value string) bool { return value == "help" || value == "--help" || value == "-h" }

func printStaleHelp(w io.Writer) {
	fmt.Fprint(w, `Usage:
  sealgraph stale [--frontier] [--refs-only] [--scan] [--format human|json]

The result is a coherent current-head observation. --scan bypasses the
disposable revision cache; no form repairs, relinks, or seals provenance.
--refs-only is a separate stable line protocol and cannot be combined with an
explicit --format json. Without --format, terminals use human output and
non-terminal destinations use versioned JSON.
`)
}

func printGitHelp(w io.Writer) {
	fmt.Fprint(w, `git sealgraph — Git sidecar integration for sealgraph

Usage:
  git sealgraph <command> [options]

This separate surface is planned and is not implemented by the standalone Phase 1 slice.
`)
}
