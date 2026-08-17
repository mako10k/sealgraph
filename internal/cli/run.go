package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/mako10k/sealgraph/internal/domain"
	"github.com/mako10k/sealgraph/internal/graph"
	"github.com/mako10k/sealgraph/internal/history"
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
	if len(args) == 0 || isHelp(args[0]) {
		printStandaloneHelp(stdout)
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
	case "derive":
		return runDerive(ctx, workDir, args[1:], stdout, stderr), true
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
	}
	return 0, false
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
	case "diff":
		return runDiff(ctx, workDir, args[1:], stdout, stderr)
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
	case "load":
		return runLoad(ctx, workDir, args[1:], stdin, stdout, stderr)
	default:
		return usageError(stderr, "unknown command %q", args[0])
	}
}

func runManifest(workDir string, args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("manifest", flag.ContinueOnError)
	flags.SetOutput(stderr)
	var source singleString
	var files stringList
	flags.Var(&source, "source", "required explicit source identity")
	flags.Var(&files, "file", "explicit relative semantic/read path (repeatable)")
	if err := flags.Parse(args); err != nil {
		return 2
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
	flags.SetOutput(stderr)
	var format singleString
	flags.Var(&format, "format", "required versioned dump format")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if flags.NArg() != 0 {
		return usageError(stderr, "load accepts no positional arguments; unexpected argument %q", flags.Arg(0))
	}
	if !format.set {
		return usageError(stderr, "load requires --format logical-v1")
	}
	if format.value != "logical-v1" {
		return usageError(stderr, "load format %q is unsupported; expected logical-v1", format.value)
	}
	input, err := io.ReadAll(stdin)
	if err != nil {
		return commandError(stderr, "load", fmt.Errorf("read logical-v1 input: %w", err))
	}
	output, err := repository.LoadLogicalV1(ctx, workDir, input)
	if err != nil {
		return commandError(stderr, "load", err)
	}
	if _, err := stdout.Write(output); err != nil {
		return commandError(stderr, "load", fmt.Errorf("write logical-v1 receipt: %w", err))
	}
	return 0
}

func runAdd(ctx context.Context, workDir string, args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		return usageError(stderr, "add requires exactly one REF")
	}
	ref := args[0]
	flags := flag.NewFlagSet("add", flag.ContinueOnError)
	flags.SetOutput(stderr)
	var content trackedString
	var contentFile trackedString
	var depends stringList
	var parent singleString
	flags.Var(&content, "content", "exact content bytes supplied as a command argument")
	flags.Var(&contentFile, "content-file", "read exact content bytes from a regular file, or '-' for stdin")
	flags.Var(&depends, "depend-on", "dependency REF or REF@SEAL (repeatable)")
	flags.Var(&parent, "parent", "exact revision parent for an absent destination REF")
	root := flags.Bool("root", false, "declare a provenance root")
	draft := flags.Bool("draft", false, "mark the candidate draft")
	if err := flags.Parse(args[1:]); err != nil {
		return 2
	}
	if flags.NArg() != 0 {
		return usageError(stderr, "add accepts exactly one REF; unexpected argument %q", flags.Arg(0))
	}
	if content.set == contentFile.set {
		return usageError(stderr, "add requires exactly one of --content or --content-file")
	}
	if parent.set {
		if parent.value == "" {
			return usageError(stderr, "add --parent requires a non-empty Seal selector")
		}
		if _, err := repository.ParseSelector(parent.value); err != nil {
			return usageError(stderr, "invalid add parent selector: %v", err)
		}
	}
	contentBytes := []byte(content.value)
	if contentFile.set {
		var err error
		contentBytes, err = readContentInput(workDir, contentFile.value, stdin)
		if err != nil {
			return usageError(stderr, "invalid --content-file: %v", err)
		}
	}
	dependencies, err := parseDependencies(depends, "")
	if err != nil {
		return usageError(stderr, "%v", err)
	}
	repo, err := repository.OpenStandalone(workDir)
	if err != nil {
		return commandError(stderr, "add", err)
	}
	candidate, err := repo.Add(ctx, repository.AddOptions{REF: ref, Content: contentBytes, Dependencies: dependencies, Parent: parent.value, Root: *root, Draft: *draft})
	if err != nil {
		return commandError(stderr, "add", err)
	}
	fmt.Fprintf(stdout, "CANDIDATE %s content=%s dependencies=%d root=%t draft=%t\n", candidate.REF, candidate.Content.ID, len(candidate.Links), candidate.Root, candidate.Draft)
	return 0
}

func runDerive(ctx context.Context, workDir string, args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		return usageError(stderr, "derive requires exactly one destination REF")
	}
	ref := args[0]
	flags := flag.NewFlagSet("derive", flag.ContinueOnError)
	flags.SetOutput(stderr)
	var source singleString
	flags.Var(&source, "from", "required source Seal selector")
	if err := flags.Parse(args[1:]); err != nil {
		return 2
	}
	if flags.NArg() != 0 {
		return usageError(stderr, "derive accepts exactly one destination REF; unexpected argument %q", flags.Arg(0))
	}
	if !source.set || source.value == "" {
		return usageError(stderr, "derive requires exactly one --from SOURCE selector")
	}
	if _, err := repository.ParseSelector(source.value); err != nil {
		return usageError(stderr, "invalid derive source: %v", err)
	}
	repo, err := repository.OpenStandalone(workDir)
	if err != nil {
		return commandError(stderr, "derive", err)
	}
	candidate, err := repo.Derive(ctx, ref, source.value)
	if err != nil {
		return commandError(stderr, "derive", err)
	}
	fmt.Fprintf(stdout, "CANDIDATE %s parent=%s content=%s dependencies=%d root=%t draft=%t\n", candidate.REF, formatOptionalObjectID(candidate.ParentRevision), candidate.Content.ID, len(candidate.Links), candidate.Root, candidate.Draft)
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
	flags.SetOutput(stderr)
	var depends stringList
	flags.Var(&depends, "depend-on", "dependency REF or REF@SEAL (repeatable)")
	var message trackedString
	flags.Var(&message, "m", "optional rationale for each dependency in this invocation")
	if err := flags.Parse(args[1:]); err != nil {
		return 2
	}
	if flags.NArg() != 0 {
		return usageError(stderr, "link accepts exactly one REF; unexpected argument %q", flags.Arg(0))
	}
	if len(depends) == 0 {
		return usageError(stderr, "link requires at least one --depend-on")
	}
	dependencies, err := parseDependencies(depends, message.value)
	if err != nil {
		return usageError(stderr, "%v", err)
	}
	repo, err := repository.OpenStandalone(workDir)
	if err != nil {
		return commandError(stderr, "link", err)
	}
	candidate, err := repo.Link(ctx, ref, dependencies)
	if err != nil {
		return commandError(stderr, "link", err)
	}
	fmt.Fprintf(stdout, "CANDIDATE %s dependencies=%d\n", candidate.REF, len(candidate.Links))
	return 0
}

func runUnlink(ctx context.Context, workDir string, args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		return usageError(stderr, "unlink requires exactly one candidate REF")
	}
	ref := args[0]
	flags := flag.NewFlagSet("unlink", flag.ContinueOnError)
	flags.SetOutput(stderr)
	var upstream singleString
	flags.Var(&upstream, "upstream", "upstream REF or guarded REF@SEAL_OR_TAG")
	if err := flags.Parse(args[1:]); err != nil {
		return 2
	}
	if flags.NArg() != 0 {
		return usageError(stderr, "unlink accepts exactly one candidate REF; unexpected argument %q", flags.Arg(0))
	}
	if !upstream.set || upstream.value == "" {
		return usageError(stderr, "unlink requires exactly one --upstream REF[@SEAL_OR_TAG]")
	}
	if _, err := repository.ParseSelector(upstream.value); err != nil {
		return usageError(stderr, "invalid --upstream %q: %v", upstream.value, err)
	}
	repo, err := repository.OpenStandalone(workDir)
	if err != nil {
		return commandError(stderr, "unlink", err)
	}
	candidate, err := repo.Unlink(ctx, ref, upstream.value)
	if err != nil {
		return commandError(stderr, "unlink", err)
	}
	fmt.Fprintf(stdout, "CANDIDATE %s dependencies=%d\n", candidate.REF, len(candidate.Links))
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
	fmt.Fprintf(stdout, "TAGGED %s %s %s\n", result.REF, strconv.Quote(result.Name), result.Seal)
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
	fmt.Fprintf(stdout, "MOVED %s %s %s tags=%d\n", result.OldREF, result.NewREF, result.Head, result.Tags)
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
	fmt.Fprintf(stdout, "SEALED %s %s\n", ref, result.ID)
	return 0
}

func runShow(ctx context.Context, workDir string, args []string, stdout, stderr io.Writer) int {
	args, outputJSON, err := extractInspectionFormat(args)
	if err != nil {
		return usageError(stderr, "%v", err)
	}
	if len(args) == 0 {
		return usageError(stderr, "show requires exactly one REF or REF@SEAL")
	}
	selector := args[0]
	flags := flag.NewFlagSet("show", flag.ContinueOnError)
	flags.SetOutput(stderr)
	rawContent := flags.Bool("raw-content", false, "write exact content bytes only")
	if err := flags.Parse(args[1:]); err != nil {
		return 2
	}
	if flags.NArg() != 0 {
		return usageError(stderr, "show accepts exactly one REF or REF@SEAL; unexpected argument %q", flags.Arg(0))
	}
	if _, err := repository.ParseSelector(selector); err != nil {
		return usageError(stderr, "invalid selector: %v", err)
	}
	if outputJSON && *rawContent {
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
	if outputJSON {
		return writeInspectionJSON(stdout, stderr, "show", showJSON(result))
	}
	fmt.Fprintf(stdout, "SEAL %s\n", result.ID)
	if len(result.REFNames) == 0 {
		fmt.Fprintln(stdout, "CURRENT_REFS -")
	} else {
		fmt.Fprintf(stdout, "CURRENT_REFS %s\n", strings.Join(result.REFNames, ","))
	}
	if result.Payload.ParentRevision == nil {
		fmt.Fprintln(stdout, "PARENT_REVISION -")
	} else {
		fmt.Fprintf(stdout, "PARENT_REVISION %s\n", result.Payload.ParentRevision)
	}
	printContentSummary(stdout, result.Payload.Content, result.Content)
	fmt.Fprintf(stdout, "ROOT %t\nDRAFT %t\nATTACHMENTS %d\n", result.Payload.Root, result.Payload.Draft, len(result.Payload.Attachments))
	for _, attachment := range result.Payload.Attachments {
		fmt.Fprintf(stdout, "  attachment name=%s media_type=%s blob=%s\n", quoteHumanString(attachment.Name), quoteHumanString(attachment.MediaType), formatContentRef(attachment.Blob))
	}
	fmt.Fprintf(stdout, "DEPENDENCIES %d\n", len(result.Payload.Links))
	for _, link := range result.Payload.Links {
		fmt.Fprintf(stdout, "  depend-on @%s message=%s\n", link.TargetSeal, quoteHumanString(link.Message))
	}
	return 0
}

func runCandidate(ctx context.Context, workDir string, args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		return usageError(stderr, "candidate requires show, diff, or discard")
	}
	switch args[0] {
	case "show":
		return runCandidateShow(ctx, workDir, args[1:], stdout, stderr)
	case "diff":
		return runCandidateDiff(ctx, workDir, args[1:], stdout, stderr)
	case "discard":
		return runCandidateDiscard(ctx, workDir, args[1:], stdout, stderr)
	default:
		return usageError(stderr, "unknown candidate operation %q; expected show, diff, or discard", args[0])
	}
}

func runCandidateShow(ctx context.Context, workDir string, args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		return usageError(stderr, "candidate show requires exactly one logical REF")
	}
	ref := args[0]
	if err := domain.ValidateREF(ref); err != nil {
		return usageError(stderr, "invalid candidate REF: %v", err)
	}
	flags := flag.NewFlagSet("candidate show", flag.ContinueOnError)
	flags.SetOutput(stderr)
	rawContent := flags.Bool("raw-content", false, "write exact candidate content bytes only")
	if err := flags.Parse(args[1:]); err != nil {
		return 2
	}
	if flags.NArg() != 0 {
		return usageError(stderr, "candidate show accepts exactly one REF; unexpected argument %q", flags.Arg(0))
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
	printCandidateInspection(stdout, inspection)
	return 0
}

func runCandidateDiff(ctx context.Context, workDir string, args []string, stdout, stderr io.Writer) int {
	if len(args) != 1 {
		return usageError(stderr, "candidate diff requires exactly one logical REF")
	}
	ref := args[0]
	if err := domain.ValidateREF(ref); err != nil {
		return usageError(stderr, "invalid candidate REF: %v", err)
	}
	repo, err := repository.OpenStandalone(workDir)
	if err != nil {
		return commandError(stderr, "candidate diff", err)
	}
	result, err := repo.DiffCandidate(ctx, ref)
	if err != nil {
		return commandError(stderr, "candidate diff", err)
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
	fmt.Fprintf(stdout, "DISCARDED CANDIDATE %s\n", ref)
	return 0
}

func printCandidateInspection(stdout io.Writer, inspection repository.CandidateInspection) {
	candidate := inspection.Candidate
	fmt.Fprintf(stdout, "REF %s\nCANDIDATE\nPARENT_REVISION %s\nEXPECTED_REF_HEAD %s\nCURRENT_HEAD %s\nEXPECTED_HEAD_STATE %s\n", candidate.REF, formatOptionalObjectID(candidate.ParentRevision), formatOptionalObjectID(candidate.ExpectedREFHead), formatOptionalObjectID(inspection.CurrentHead), inspection.ExpectedHeadState)
	printContentSummary(stdout, candidate.Content, inspection.Content)
	fmt.Fprintf(stdout, "ROOT %t\nDRAFT %t\nATTACHMENTS %d\n", candidate.Root, candidate.Draft, len(candidate.Attachments))
	for _, attachment := range candidate.Attachments {
		fmt.Fprintf(stdout, "  attachment name=%s media_type=%s blob=%s\n", quoteHumanString(attachment.Name), quoteHumanString(attachment.MediaType), formatContentRef(attachment.Blob))
	}
	fmt.Fprintf(stdout, "DEPENDENCIES %d\n", len(candidate.Links))
	for _, link := range candidate.Links {
		fmt.Fprintf(stdout, "  depend-on @%s message=%s\n", link.TargetSeal, quoteHumanString(link.Message))
	}
}

func printCandidateDiff(stdout io.Writer, result repository.CandidateDiffResult) {
	inspection, diff := result.Inspection, result.Diff
	candidate := inspection.Candidate
	fmt.Fprintf(stdout, "REF %s\nFROM %s\nTO CANDIDATE\nPARENT_REVISION %s\nEXPECTED_REF_HEAD %s\nCURRENT_HEAD %s\nEXPECTED_HEAD_STATE %s\n", candidate.REF, formatOptionalObjectID(candidate.ParentRevision), formatOptionalObjectID(candidate.ParentRevision), formatOptionalObjectID(candidate.ExpectedREFHead), formatOptionalObjectID(inspection.CurrentHead), inspection.ExpectedHeadState)
	if diff.Initial {
		fmt.Fprintf(stdout, "CONTENT ADD new=%s\n", formatContentRef(diff.Content.After))
		fmt.Fprintf(stdout, "ATTACHMENTS SET count=%d\n", len(candidate.Attachments))
		for _, change := range diff.Attachments {
			printAttachmentChange(stdout, change)
		}
		fmt.Fprintf(stdout, "LINKS SET count=%d\n", len(candidate.Links))
		for _, change := range diff.Links {
			printLinkChange(stdout, "  LINK_", change)
		}
		fmt.Fprintf(stdout, "ROOT SET value=%t\nDRAFT SET value=%t\n", candidate.Root, candidate.Draft)
		return
	}
	printMaterialStateDiff(stdout, diff.Content, diff.Attachments, diff.Links)
	printBoolChange(stdout, "ROOT", diff.Root)
	printBoolChange(stdout, "DRAFT", diff.Draft)
}

func runLog(ctx context.Context, workDir string, args []string, stdout, stderr io.Writer) int {
	args, outputJSON, err := extractInspectionFormat(args)
	if err != nil {
		return usageError(stderr, "%v", err)
	}
	if len(args) != 1 {
		return usageError(stderr, "log requires exactly one current logical REF")
	}
	if err := domain.ValidateREF(args[0]); err != nil {
		return usageError(stderr, "invalid log REF: %v", err)
	}
	repo, err := repository.OpenStandalone(workDir)
	if err != nil {
		return commandError(stderr, "log", err)
	}
	entries, err := repo.Log(ctx, args[0])
	if err != nil {
		return commandError(stderr, "log", err)
	}
	if outputJSON {
		return writeInspectionJSON(stdout, stderr, "log", logJSON(args[0], entries))
	}
	printLog(stdout, args[0], entries)
	return 0
}

func runLinkLog(ctx context.Context, workDir string, args []string, stdout, stderr io.Writer) int {
	args, outputJSON, err := extractInspectionFormat(args)
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
	flags.SetOutput(stderr)
	var upstream singleString
	flags.Var(&upstream, "upstream", "show changes involving one exact upstream Seal selector")
	if err := flags.Parse(args[1:]); err != nil {
		return 2
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
	entries, target, err := repo.LinkLog(ctx, args[0], upstream.value)
	if err != nil {
		return commandError(stderr, "linklog", err)
	}
	if outputJSON {
		return writeInspectionJSON(stdout, stderr, "linklog", linkLogJSON(args[0], target, entries))
	}
	printLinkLog(stdout, args[0], target, entries)
	return 0
}

func runDiff(ctx context.Context, workDir string, args []string, stdout, stderr io.Writer) int {
	args, outputJSON, err := extractInspectionFormat(args)
	if err != nil {
		return usageError(stderr, "%v", err)
	}
	if len(args) != 1 && len(args) != 2 {
		return usageError(stderr, "diff requires one current REF or two Seal selectors")
	}
	for _, arg := range args {
		if _, err := repository.ParseSelector(arg); err != nil {
			return usageError(stderr, "invalid diff selector %q: %v", arg, err)
		}
	}
	repo, err := repository.OpenStandalone(workDir)
	if err != nil {
		return commandError(stderr, "diff", err)
	}
	var result history.SealDiff
	if len(args) == 1 {
		selector, _ := repository.ParseSelector(args[0])
		if selector.Kind != repository.SelectorCurrentREF {
			return usageError(stderr, "one-argument diff requires a current REF; provide two selectors for an explicit comparison")
		}
		result, err = repo.DiffCurrent(ctx, selector.REF)
	} else {
		result, err = repo.DiffSelectors(ctx, args[0], args[1])
	}
	if err != nil {
		return commandError(stderr, "diff", err)
	}
	if outputJSON {
		return writeInspectionJSON(stdout, stderr, "diff", diffJSON(result))
	}
	printSealDiff(stdout, result)
	return 0
}

func printLog(stdout io.Writer, ref string, entries []history.Entry) {
	fmt.Fprintf(stdout, "REF %s\n", ref)
	for _, entry := range entries {
		fmt.Fprintf(stdout, "SEAL %s\n", entry.ID)
		fmt.Fprintf(stdout, "  PARENT_REVISION %s\n", formatOptionalObjectID(entry.Payload.ParentRevision))
		fmt.Fprintf(stdout, "  ROOT %t\n", entry.Payload.Root)
		fmt.Fprintf(stdout, "  DRAFT %t\n", entry.Payload.Draft)
		fmt.Fprintf(stdout, "  CONTENT %s\n", formatContentRef(entry.Payload.Content))
		fmt.Fprintf(stdout, "  DEPENDENCIES %d\n", len(entry.Payload.Links))
		for _, link := range entry.Payload.Links {
			fmt.Fprintf(stdout, "    depend-on @%s message=%s\n", link.TargetSeal, quoteHumanString(link.Message))
		}
	}
}

func printLinkLog(stdout io.Writer, ref, upstream string, entries []history.LinkLogEntry) {
	fmt.Fprintf(stdout, "REF %s\n", ref)
	if upstream != "" {
		fmt.Fprintf(stdout, "UPSTREAM %s\n", upstream)
	}
	for _, entry := range entries {
		fmt.Fprintf(stdout, "SEAL %s\n", entry.Entry.ID)
		fmt.Fprintf(stdout, "  PARENT_REVISION %s\n", formatOptionalObjectID(entry.Entry.Payload.ParentRevision))
		if len(entry.Changes) == 0 {
			fmt.Fprintln(stdout, "  NO_LINK_CHANGES")
			continue
		}
		for _, change := range entry.Changes {
			printLinkChange(stdout, "  LINK_", change)
		}
	}
}

func printSealDiff(stdout io.Writer, diff history.SealDiff) {
	fmt.Fprintf(stdout, "FROM %s\nTO %s\n", diff.From, diff.To)
	printMaterialStateDiff(stdout, diff.Content, diff.Attachments, diff.Links)
	printBoolChange(stdout, "ROOT", diff.Root)
	printBoolChange(stdout, "DRAFT", diff.Draft)
	if diff.Parent.Changed {
		fmt.Fprintf(stdout, "PARENT CHANGED old=%s new=%s\n", formatOptionalObjectID(diff.Parent.Before), formatOptionalObjectID(diff.Parent.After))
	} else {
		fmt.Fprintf(stdout, "PARENT UNCHANGED value=%s\n", formatOptionalObjectID(diff.Parent.Before))
	}
}

func printMaterialStateDiff(stdout io.Writer, content history.ValueChange[domain.ContentRef], attachments []history.AttachmentChangeRecord, links []history.LinkChange) {
	if content.Changed {
		fmt.Fprintf(stdout, "CONTENT CHANGED old=%s new=%s\n", formatContentRef(content.Before), formatContentRef(content.After))
	} else {
		fmt.Fprintf(stdout, "CONTENT UNCHANGED value=%s\n", formatContentRef(content.Before))
	}
	if len(attachments) == 0 {
		fmt.Fprintln(stdout, "ATTACHMENTS UNCHANGED")
	} else {
		fmt.Fprintf(stdout, "ATTACHMENTS CHANGED count=%d\n", len(attachments))
		for _, change := range attachments {
			printAttachmentChange(stdout, change)
		}
	}
	if len(links) == 0 {
		fmt.Fprintln(stdout, "LINKS UNCHANGED")
	} else {
		fmt.Fprintf(stdout, "LINKS CHANGED count=%d\n", len(links))
		for _, change := range links {
			printLinkChange(stdout, "  LINK_", change)
		}
	}
}

func printLinkChange(stdout io.Writer, prefix string, change history.LinkChange) {
	switch change.Kind {
	case history.LinkAdd:
		fmt.Fprintf(stdout, "%sADD target=%s message=%s\n", prefix, change.TargetSeal, quoteHumanString(change.AfterMessage))
	case history.LinkRemove:
		fmt.Fprintf(stdout, "%sREMOVE target=%s message=%s\n", prefix, change.TargetSeal, quoteHumanString(change.BeforeMessage))
	case history.LinkRepoint:
		fmt.Fprintf(stdout, "%sREPOINT old=%s new=%s old_message=%s new_message=%s\n", prefix, formatOptionalObjectID(change.BeforeSeal), formatOptionalObjectID(change.AfterSeal), quoteHumanString(change.BeforeMessage), quoteHumanString(change.AfterMessage))
	case history.LinkMessage:
		fmt.Fprintf(stdout, "%sMESSAGE_CHANGE target=%s old=%s new=%s\n", prefix, change.TargetSeal, quoteHumanString(change.BeforeMessage), quoteHumanString(change.AfterMessage))
	}
}

func printAttachmentChange(stdout io.Writer, change history.AttachmentChangeRecord) {
	switch change.Kind {
	case history.AttachmentAdd:
		fmt.Fprintf(stdout, "  ATTACHMENT_ADD name=%s media_type=%s blob=%s\n", quoteHumanString(change.Name), quoteHumanString(change.After.MediaType), formatContentRef(change.After.Blob))
	case history.AttachmentRemove:
		fmt.Fprintf(stdout, "  ATTACHMENT_REMOVE name=%s media_type=%s blob=%s\n", quoteHumanString(change.Name), quoteHumanString(change.Before.MediaType), formatContentRef(change.Before.Blob))
	case history.AttachmentChange:
		fmt.Fprintf(stdout, "  ATTACHMENT_CHANGE name=%s old_media_type=%s new_media_type=%s old_blob=%s new_blob=%s\n", quoteHumanString(change.Name), quoteHumanString(change.Before.MediaType), quoteHumanString(change.After.MediaType), formatContentRef(change.Before.Blob), formatContentRef(change.After.Blob))
	}
}

func printBoolChange(stdout io.Writer, name string, change history.ValueChange[bool]) {
	if change.Changed {
		fmt.Fprintf(stdout, "%s CHANGED old=%t new=%t\n", name, change.Before, change.After)
	} else {
		fmt.Fprintf(stdout, "%s UNCHANGED value=%t\n", name, change.Before)
	}
}

func formatOptionalObjectID(id *domain.ObjectID) string {
	if id == nil {
		return "-"
	}
	return id.String()
}

func formatContentRef(ref domain.ContentRef) string {
	return ref.Store + "/" + ref.Type + "@" + ref.ID.String()
}

const contentPreviewLimit = 256

func printContentSummary(stdout io.Writer, ref domain.ContentRef, content []byte) {
	preview := content
	truncated := len(content) > contentPreviewLimit
	if truncated {
		preview = content[:contentPreviewLimit]
	}
	fmt.Fprintf(stdout, "CONTENT %s bytes=%d\n", formatContentRef(ref), len(content))
	fmt.Fprintf(stdout, "CONTENT_PREVIEW %s truncated=%t\n", quoteHumanBytes(preview), truncated)
}

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
	args, outputJSON, err := extractInspectionFormat(args)
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
	if outputJSON {
		return writeInspectionJSON(stdout, stderr, "status", statusesJSON("sealgraph/status/v1", statuses, nil))
	}
	fmt.Fprintln(stdout, "SEALED_STATE")
	if len(statuses) == 0 {
		fmt.Fprintln(stdout, "CLEAN")
		return 0
	}
	printStatuses(stdout, statuses)
	return 0
}

func runStale(ctx context.Context, workDir string, args []string, stdout, stderr io.Writer) int {
	if len(args) == 1 && isHelp(args[0]) {
		printStaleHelp(stdout)
		return 0
	}
	args, outputJSON, formatErr := extractInspectionFormat(args)
	if formatErr != nil {
		return usageError(stderr, "%v", formatErr)
	}
	flags := flag.NewFlagSet("stale", flag.ContinueOnError)
	flags.SetOutput(stderr)
	var frontier singleBool
	var refsOnly singleBool
	var scan singleBool
	flags.Var(&frontier, "frontier", "select the upstream-most stale review frontier")
	flags.Var(&refsOnly, "refs-only", "emit one logical REF per line")
	flags.Var(&scan, "scan", "bypass cache and rebuild the canonical revision index")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if flags.NArg() != 0 {
		return usageError(stderr, "stale accepts no positional arguments; unexpected argument %q", flags.Arg(0))
	}
	if outputJSON && refsOnly.value {
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
	if outputJSON {
		return writeInspectionJSON(stdout, stderr, "stale", statusesJSON("sealgraph/stale/v1", statuses, map[string]any{"frontier": frontier.value, "scan": scan.value}))
	}
	printStatuses(stdout, statuses)
	return 0
}

func runImpact(ctx context.Context, workDir string, args []string, stdout, stderr io.Writer) int {
	args, outputJSON, formatErr := extractInspectionFormat(args)
	if formatErr != nil {
		return usageError(stderr, "%v", formatErr)
	}
	flags := flag.NewFlagSet("impact", flag.ContinueOnError)
	flags.SetOutput(stderr)
	var allPaths singleBool
	var maxPaths singleString
	flags.Var(&allPaths, "all-paths", "show bounded distinct simple Cause paths")
	flags.Var(&maxPaths, "max-paths", "positive path limit per downstream Seal; valid only with --all-paths")
	if err := flags.Parse(args); err != nil {
		return 2
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
		return usageError(stderr, "%v", err)
	}
	repo, err := repository.OpenStandalone(workDir)
	if err != nil {
		return commandError(stderr, "impact", err)
	}
	source, impacts, err := repo.Impact(ctx, selector, allPaths.value, limit)
	if err != nil {
		return commandError(stderr, "impact", err)
	}
	if outputJSON {
		return writeInspectionJSON(stdout, stderr, "impact", impactJSON(source, impacts, allPaths.value, limit))
	}
	fmt.Fprintln(stdout, "STRUCTURAL_IMPACT")
	printImpacts(stdout, source, impacts, limit)
	return 0
}

func runGraph(ctx context.Context, workDir string, args []string, stdout, stderr io.Writer) int {
	args, outputJSON, err := extractInspectionFormat(args)
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
	if outputJSON {
		return writeInspectionJSON(stdout, stderr, "graph", graphJSON(nodes))
	}
	fmt.Fprintln(stdout, "REVISION_CAUSE_GRAPH")
	printGraph(stdout, nodes)
	return 0
}

func runFsck(ctx context.Context, workDir string, args []string, stdout, stderr io.Writer) int {
	args, outputJSON, err := extractInspectionFormat(args)
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
	if outputJSON {
		return writeInspectionJSON(stdout, stderr, "fsck", fsckJSON(report))
	}
	fmt.Fprintf(stdout, "FSCK_OK objects=%d seals=%d material_objects=%d refs=%d tags=%d active_seals=%d historical_or_detached=%d unreferenced_objects=%d\n", report.Objects, report.Seals, report.MaterialObjects, report.REFs, report.Tags, report.ActiveSeals, len(report.HistoricalOrDetachedSeals), len(report.UnreferencedObjects))
	for _, id := range report.HistoricalOrDetachedSeals {
		fmt.Fprintf(stdout, "HISTORICAL_OR_DETACHED_SEAL %s\n", id)
	}
	for _, id := range report.UnreferencedObjects {
		fmt.Fprintf(stdout, "UNREFERENCED_OBJECT %s\n", id)
	}
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

func printImpacts(stdout io.Writer, source domain.ObjectID, impacts []graph.Impact, limit int) {
	fmt.Fprintf(stdout, "SOURCE %s\n", source)
	for _, impact := range impacts {
		fmt.Fprintf(stdout, "IMPACT %s refs=%s paths=%d\n", impact.Head, strings.Join(impact.REFs, ","), len(impact.Paths))
		for _, path := range impact.Paths {
			parts := make([]string, len(path))
			for i, id := range path {
				parts[i] = id.String()
			}
			fmt.Fprintf(stdout, "  PATH %s\n", strings.Join(parts, " -> "))
		}
		if impact.Truncated {
			fmt.Fprintf(stdout, "  PATHS_TRUNCATED max=%d\n", limit)
		}
	}
}

func printGraph(stdout io.Writer, nodes []repository.GraphNode) {
	for _, node := range nodes {
		refs := "-"
		if len(node.REFs) != 0 {
			refs = strings.Join(node.REFs, ",")
		}
		fmt.Fprintf(stdout, "SEAL %s state=%s refs=%s parent=%s\n", node.ID, node.State, refs, formatOptionalObjectID(node.Parent))
		for _, link := range node.Links {
			fmt.Fprintf(stdout, "  CAUSE %s state=%s\n", link.Target, link.State)
		}
	}
}

func printStatuses(stdout io.Writer, statuses []repository.RefStatus) {
	for _, status := range statuses {
		fmt.Fprintf(stdout, "%s %s", status.REF, strings.Join(status.Labels(), ","))
		if status.Head != nil {
			fmt.Fprintf(stdout, " %s", status.Head)
		}
		fmt.Fprintln(stdout)
	}
}

func parseDependencies(values []string, message string) ([]repository.Dependency, error) {
	if values == nil {
		return nil, nil
	}
	dependencies := make([]repository.Dependency, 0, len(values))
	for _, value := range values {
		if _, err := repository.ParseSelector(value); err != nil {
			return nil, fmt.Errorf("invalid --depend-on %q: %w", value, err)
		}
		dependencies = append(dependencies, repository.Dependency{Selector: value, Message: message})
	}
	return dependencies, nil
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
	fmt.Fprintf(stderr, "sealgraph %s: %v\n", command, err)
	return 1
}

func usageError(stderr io.Writer, format string, args ...any) int {
	fmt.Fprintf(stderr, "sealgraph: %s\n", fmt.Sprintf(format, args...))
	return 2
}

func isHelp(value string) bool { return value == "help" || value == "--help" || value == "-h" }

func printStandaloneHelp(w io.Writer) {
	fprintf := fmt.Fprintf
	fprintf(w, `sealgraph — standalone provenance sealing CLI

Usage:
  sealgraph init
  sealgraph manifest --source SOURCE --file PATH [--file PATH ...]
  sealgraph add REF (--content CONTENT | --content-file PATH_OR_DASH) [--parent SELECTOR] [--root] [--draft] [--depend-on SELECTOR]...
  sealgraph derive NEW_REF --from SOURCE_SELECTOR
  sealgraph link REF --depend-on SELECTOR... [-m LINK_MESSAGE]
  sealgraph unlink REF --upstream SELECTOR
  sealgraph tag REF [TAGNAME]
  sealgraph tag REF@SEAL_OR_TAG TAGNAME
  sealgraph mv OLD_REF NEW_REF
  sealgraph candidate show REF [--raw-content]
  sealgraph candidate diff REF
  sealgraph candidate discard REF
  sealgraph seal REF
  sealgraph show SELECTOR [--raw-content] [--format human|json]
  sealgraph log REF [--format human|json]
  sealgraph linklog REF [--upstream SELECTOR] [--format human|json]
  sealgraph diff REF [--format human|json]
  sealgraph diff SELECTOR SELECTOR [--format human|json]
  sealgraph status [REF] [--format human|json]
  sealgraph stale [--frontier] [--refs-only] [--scan] [--format human|json]
  sealgraph impact [--all-paths] [--max-paths N] SELECTOR [--format human|json]
  sealgraph graph [--format human|json]
  sealgraph fsck [--format human|json]
  sealgraph load --format logical-v1 < repository.dump.json

manifest emits a path/digest claim only; it does not write repository state or
store named files as attachments. Each seal operation advances exactly one
logical REF. Tags are immutable, and
mv moves one REF manifest with its complete tag namespace without moving a
candidate or creating an old-name alias.

Semantic legend:
  CLEAN does not compare working files; it means no candidate and no derived
  stale relation for the selected sealed state.
  REF is a movable logical identity, not a branch or checkout target.
  STRUCTURAL_IMPACT is provenance reachability; stale is current review state.
  root marks a provenance boundary, not truth or trust.
  log/linklog are Seal revision/Cause histories, not commit/reflog histories.
  Standalone commands use only explicit inputs and .sealgraph; standalone init
  does not discover or inspect Git.
`)
}

func printStaleHelp(w io.Writer) {
	fmt.Fprint(w, `Usage:
  sealgraph stale [--frontier] [--refs-only] [--scan] [--format human|json]

The result is a coherent current-head observation. --scan bypasses the
disposable revision cache; no form repairs, relinks, or seals provenance.
--refs-only is a separate stable line protocol and cannot be combined with JSON.
`)
}

func printGitHelp(w io.Writer) {
	fmt.Fprint(w, `git sealgraph — Git sidecar integration for sealgraph

Usage:
  git sealgraph <command> [options]

This separate surface is planned and is not implemented by the standalone Phase 1 slice.
`)
}
