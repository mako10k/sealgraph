package cli

import (
	"bytes"
	"context"
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
	"github.com/mako10k/sealgraph/internal/repository"
)

const version = "0.1.0-dev"

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
		fmt.Fprintf(stdout, "git sealgraph %s\n", version)
		return 0
	}
	fmt.Fprintf(stderr, "git sealgraph: command %q is not implemented; Git sidecar is outside the Phase 1 standalone scope\n", args[0])
	return 2
}

func runStandaloneAt(workDir string, args []string, stdout, stderr io.Writer) int {
	return runStandaloneAtWithInput(workDir, args, strings.NewReader(""), stdout, stderr)
}

func runStandaloneAtWithInput(workDir string, args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	if len(args) == 0 || isHelp(args[0]) {
		printStandaloneHelp(stdout)
		return 0
	}
	if args[0] == "--version" || args[0] == "version" {
		fmt.Fprintf(stdout, "sealgraph %s\n", version)
		return 0
	}
	ctx := context.Background()
	switch args[0] {
	case "init":
		if len(args) != 1 {
			return usageError(stderr, "init accepts no arguments")
		}
		created, err := repository.InitStandalone(workDir)
		if err != nil {
			return commandError(stderr, "init", err)
		}
		if created {
			fmt.Fprintf(stdout, "Initialized standalone sealgraph repository in %s/.sealgraph\n", workDir)
		} else {
			fmt.Fprintf(stdout, "Standalone sealgraph repository already initialized in %s/.sealgraph\n", workDir)
		}
		return 0
	case "add":
		return runAdd(ctx, workDir, args[1:], stdin, stdout, stderr)
	case "link":
		return runLink(ctx, workDir, args[1:], stdout, stderr)
	case "unlink":
		return runUnlink(ctx, workDir, args[1:], stdout, stderr)
	case "candidate":
		return runCandidate(ctx, workDir, args[1:], stdout, stderr)
	case "tag":
		return runTag(ctx, workDir, args[1:], stdout, stderr)
	case "seal":
		return runSeal(ctx, workDir, args[1:], stdout, stderr)
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
	default:
		return usageError(stderr, "unknown command %q", args[0])
	}
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
	flags.Var(&content, "content", "exact content bytes supplied as a command argument")
	flags.Var(&contentFile, "content-file", "read exact content bytes from a regular file, or '-' for stdin")
	flags.Var(&depends, "depend-on", "dependency REF or REF@SEAL (repeatable)")
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
	candidate, err := repo.Add(ctx, repository.AddOptions{REF: ref, Content: contentBytes, Dependencies: dependencies, Root: *root, Draft: *draft})
	if err != nil {
		return commandError(stderr, "add", err)
	}
	fmt.Fprintf(stdout, "CANDIDATE %s content=%s dependencies=%d root=%t draft=%t\n", candidate.REF, candidate.Content.ID, len(candidate.Links), candidate.Root, candidate.Draft)
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
	upstreamREF, revision, err := repository.ParseSelector(upstream.value)
	if err != nil {
		return usageError(stderr, "invalid --upstream %q: %v", upstream.value, err)
	}
	repo, err := repository.OpenStandalone(workDir)
	if err != nil {
		return commandError(stderr, "unlink", err)
	}
	candidate, err := repo.Unlink(ctx, ref, upstreamREF, revision)
	if err != nil {
		return commandError(stderr, "unlink", err)
	}
	fmt.Fprintf(stdout, "CANDIDATE %s dependencies=%d\n", candidate.REF, len(candidate.Links))
	return 0
}

func runTag(ctx context.Context, workDir string, args []string, stdout, stderr io.Writer) int {
	if len(args) != 1 && len(args) != 2 {
		return usageError(stderr, "tag requires REF [TAGNAME] or REF@SEAL_OR_TAG TAGNAME")
	}
	ref, revision, err := repository.ParseSelector(args[0])
	if err != nil {
		return usageError(stderr, "invalid tag selector: %v", err)
	}
	repo, err := repository.OpenStandalone(workDir)
	if err != nil {
		return commandError(stderr, "tag", err)
	}
	if len(args) == 1 {
		if revision != "" {
			return usageError(stderr, "tag listing requires a logical REF without @TOKEN")
		}
		tags, err := repo.ListTags(ctx, ref)
		if err != nil {
			return commandError(stderr, "tag", err)
		}
		for _, tag := range tags {
			fmt.Fprintf(stdout, "%s %s\n", tag.Name, tag.Seal)
		}
		return 0
	}
	id, err := repo.CreateTag(ctx, ref, revision, args[1])
	if err != nil {
		return commandError(stderr, "tag", err)
	}
	fmt.Fprintf(stdout, "TAGGED %s@%s %s\n", ref, args[1], id)
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
	ref, revision, err := repository.ParseSelector(selector)
	if err != nil {
		return usageError(stderr, "invalid selector: %v", err)
	}
	repo, err := repository.OpenStandalone(workDir)
	if err != nil {
		return commandError(stderr, "show", err)
	}
	result, err := repo.Show(ctx, ref, revision)
	if err != nil {
		return commandError(stderr, "show", err)
	}
	if *rawContent {
		return writeRawContent(stdout, stderr, "show", result.Content)
	}
	fmt.Fprintf(stdout, "REF %s\nSEAL %s\n", result.Payload.REF, result.ID)
	if result.Payload.Parent == nil {
		fmt.Fprintln(stdout, "PARENT -")
	} else {
		fmt.Fprintf(stdout, "PARENT %s\n", result.Payload.Parent)
	}
	printContentSummary(stdout, result.Payload.Content, result.Content)
	fmt.Fprintf(stdout, "ROOT %t\nDRAFT %t\nATTACHMENTS %d\n", result.Payload.Root, result.Payload.Draft, len(result.Payload.Attachments))
	for _, attachment := range result.Payload.Attachments {
		fmt.Fprintf(stdout, "  attachment name=%s media_type=%s blob=%s\n", quoteHumanString(attachment.Name), quoteHumanString(attachment.MediaType), formatContentRef(attachment.Blob))
	}
	fmt.Fprintf(stdout, "DEPENDENCIES %d\n", len(result.Payload.Links))
	for _, link := range result.Payload.Links {
		fmt.Fprintf(stdout, "  depend-on %s@%s message=%s\n", link.TargetREF, link.TargetSeal, quoteHumanString(link.Message))
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
	ref, revision, err := repository.ParseSelector(args[0])
	if err != nil {
		return usageError(stderr, "invalid candidate REF: %v", err)
	}
	if revision != "" {
		return usageError(stderr, "candidate show accepts a logical REF without @TOKEN")
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
	ref, revision, err := repository.ParseSelector(args[0])
	if err != nil {
		return usageError(stderr, "invalid candidate REF: %v", err)
	}
	if revision != "" {
		return usageError(stderr, "candidate diff accepts a logical REF without @TOKEN")
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
	ref, revision, err := repository.ParseSelector(args[0])
	if err != nil {
		return usageError(stderr, "invalid candidate REF: %v", err)
	}
	if revision != "" {
		return usageError(stderr, "candidate discard accepts a logical REF without @TOKEN")
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
	fmt.Fprintf(stdout, "REF %s\nCANDIDATE\nBASE %s\nCURRENT_HEAD %s\nBASE_STATE %s\n", candidate.REF, formatOptionalObjectID(candidate.Base), formatOptionalObjectID(inspection.CurrentHead), inspection.BaseState)
	printContentSummary(stdout, candidate.Content, inspection.Content)
	fmt.Fprintf(stdout, "ROOT %t\nDRAFT %t\nATTACHMENTS %d\n", candidate.Root, candidate.Draft, len(candidate.Attachments))
	for _, attachment := range candidate.Attachments {
		fmt.Fprintf(stdout, "  attachment name=%s media_type=%s blob=%s\n", quoteHumanString(attachment.Name), quoteHumanString(attachment.MediaType), formatContentRef(attachment.Blob))
	}
	fmt.Fprintf(stdout, "DEPENDENCIES %d\n", len(candidate.Links))
	for _, link := range candidate.Links {
		fmt.Fprintf(stdout, "  depend-on %s@%s message=%s\n", link.TargetREF, link.TargetSeal, quoteHumanString(link.Message))
	}
}

func printCandidateDiff(stdout io.Writer, result repository.CandidateDiffResult) {
	inspection, diff := result.Inspection, result.Diff
	candidate := inspection.Candidate
	fmt.Fprintf(stdout, "REF %s\nFROM %s\nTO CANDIDATE\nCURRENT_HEAD %s\nBASE_STATE %s\n", candidate.REF, formatOptionalObjectID(candidate.Base), formatOptionalObjectID(inspection.CurrentHead), inspection.BaseState)
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
	if len(args) != 1 {
		return usageError(stderr, "log requires exactly one current logical REF")
	}
	ref, revision, err := repository.ParseSelector(args[0])
	if err != nil {
		return usageError(stderr, "invalid log REF: %v", err)
	}
	if revision != "" {
		return usageError(stderr, "log starts from a current logical REF, not REF@SEAL; use show for one historical generation")
	}
	repo, err := repository.OpenStandalone(workDir)
	if err != nil {
		return commandError(stderr, "log", err)
	}
	entries, err := repo.Log(ctx, ref)
	if err != nil {
		return commandError(stderr, "log", err)
	}
	printLog(stdout, ref, entries)
	return 0
}

func runLinkLog(ctx context.Context, workDir string, args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		return usageError(stderr, "linklog requires exactly one current logical REF")
	}
	ref, revision, err := repository.ParseSelector(args[0])
	if err != nil {
		return usageError(stderr, "invalid linklog REF: %v", err)
	}
	if revision != "" {
		return usageError(stderr, "linklog starts from a current logical REF, not REF@SEAL")
	}
	flags := flag.NewFlagSet("linklog", flag.ContinueOnError)
	flags.SetOutput(stderr)
	upstream := flags.String("upstream", "", "show changes for one exact upstream logical REF")
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
	entries, err := repo.LinkLog(ctx, ref, *upstream)
	if err != nil {
		return commandError(stderr, "linklog", err)
	}
	printLinkLog(stdout, ref, *upstream, entries)
	return 0
}

func runDiff(ctx context.Context, workDir string, args []string, stdout, stderr io.Writer) int {
	if len(args) != 1 && len(args) != 2 {
		return usageError(stderr, "diff requires one current REF or two explicit REF@SEAL generations")
	}
	var ref string
	var fromRevision, toRevision string
	if len(args) == 1 {
		parsedREF, revision, err := repository.ParseSelector(args[0])
		if err != nil {
			return usageError(stderr, "invalid diff REF: %v", err)
		}
		if revision != "" {
			return usageError(stderr, "one-argument diff requires a current logical REF without @SEAL; provide two explicit selectors to compare historical generations")
		}
		ref = parsedREF
	} else {
		fromREF, parsedFromRevision, err := repository.ParseSelector(args[0])
		if err != nil {
			return usageError(stderr, "invalid older diff selector: %v", err)
		}
		toREF, parsedToRevision, err := repository.ParseSelector(args[1])
		if err != nil {
			return usageError(stderr, "invalid newer diff selector: %v", err)
		}
		if parsedFromRevision == "" || parsedToRevision == "" {
			return usageError(stderr, "two-argument diff requires two explicit REF@SEAL selectors; current HEAD shorthand is only available as 'diff REF'")
		}
		if fromREF != toREF {
			return usageError(stderr, "diff compares generations of one logical REF; got %s and %s", fromREF, toREF)
		}
		ref, fromRevision, toRevision = fromREF, parsedFromRevision, parsedToRevision
	}
	repo, err := repository.OpenStandalone(workDir)
	if err != nil {
		return commandError(stderr, "diff", err)
	}
	var result history.SealDiff
	if fromRevision == "" {
		result, err = repo.DiffCurrent(ctx, ref)
	} else {
		fromID, resolveErr := repo.ResolveSealID(ctx, ref, fromRevision)
		if resolveErr != nil {
			return commandError(stderr, "diff", resolveErr)
		}
		toID, resolveErr := repo.ResolveSealID(ctx, ref, toRevision)
		if resolveErr != nil {
			return commandError(stderr, "diff", resolveErr)
		}
		result, err = repo.DiffExact(ctx, ref, fromID, toID)
	}
	if err != nil {
		return commandError(stderr, "diff", err)
	}
	printSealDiff(stdout, result)
	return 0
}

func printLog(stdout io.Writer, ref string, entries []history.Entry) {
	fmt.Fprintf(stdout, "REF %s\n", ref)
	for _, entry := range entries {
		fmt.Fprintf(stdout, "SEAL %s\n", entry.ID)
		fmt.Fprintf(stdout, "  PARENT %s\n", formatOptionalObjectID(entry.Payload.Parent))
		fmt.Fprintf(stdout, "  ROOT %t\n", entry.Payload.Root)
		fmt.Fprintf(stdout, "  DRAFT %t\n", entry.Payload.Draft)
		fmt.Fprintf(stdout, "  CONTENT %s\n", formatContentRef(entry.Payload.Content))
		fmt.Fprintf(stdout, "  DEPENDENCIES %d\n", len(entry.Payload.Links))
		for _, link := range entry.Payload.Links {
			fmt.Fprintf(stdout, "    depend-on %s@%s message=%s\n", link.TargetREF, link.TargetSeal, quoteHumanString(link.Message))
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
		fmt.Fprintf(stdout, "  PARENT %s\n", formatOptionalObjectID(entry.Entry.Payload.Parent))
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
	fmt.Fprintf(stdout, "REF %s\nFROM %s\nTO %s\n", diff.REF, diff.From, diff.To)
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
		fmt.Fprintf(stdout, "%sADD %s new=%s message=%s\n", prefix, change.TargetREF, formatOptionalObjectID(change.AfterSeal), quoteHumanString(change.AfterMessage))
	case history.LinkRemove:
		fmt.Fprintf(stdout, "%sREMOVE %s old=%s message=%s\n", prefix, change.TargetREF, formatOptionalObjectID(change.BeforeSeal), quoteHumanString(change.BeforeMessage))
	case history.LinkRepoint:
		fmt.Fprintf(stdout, "%sREPOINT %s old=%s new=%s\n", prefix, change.TargetREF, formatOptionalObjectID(change.BeforeSeal), formatOptionalObjectID(change.AfterSeal))
	case history.LinkMessage:
		fmt.Fprintf(stdout, "%sMESSAGE_CHANGE %s old=%s new=%s\n", prefix, change.TargetREF, quoteHumanString(change.BeforeMessage), quoteHumanString(change.AfterMessage))
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

func printStringChange(stdout io.Writer, name string, change history.ValueChange[string]) {
	if change.Changed {
		fmt.Fprintf(stdout, "%s CHANGED old=%s new=%s\n", name, quoteHumanString(change.Before), quoteHumanString(change.After))
	} else {
		fmt.Fprintf(stdout, "%s UNCHANGED value=%s\n", name, quoteHumanString(change.Before))
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
	flags := flag.NewFlagSet("stale", flag.ContinueOnError)
	flags.SetOutput(stderr)
	var frontier singleBool
	var refsOnly singleBool
	flags.Var(&frontier, "frontier", "select the upstream-most stale review frontier")
	flags.Var(&refsOnly, "refs-only", "emit one logical REF per line")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if flags.NArg() != 0 {
		return usageError(stderr, "stale accepts no positional arguments; unexpected argument %q", flags.Arg(0))
	}
	repo, err := repository.OpenStandalone(workDir)
	if err != nil {
		return commandError(stderr, "stale", err)
	}
	var statuses []repository.RefStatus
	if frontier.value {
		statuses, err = repo.StaleFrontier(ctx)
	} else {
		statuses, err = repo.Stale(ctx)
	}
	if err != nil {
		return commandError(stderr, "stale", err)
	}
	var output bytes.Buffer
	if refsOnly.value {
		for _, status := range statuses {
			fmt.Fprintln(&output, status.REF)
		}
	} else if len(statuses) == 0 {
		fmt.Fprintln(&output, "CLEAN")
	} else {
		printStatuses(&output, statuses)
	}
	if _, err := stdout.Write(output.Bytes()); err != nil {
		return commandError(stderr, "stale", fmt.Errorf("write output: %w", err))
	}
	return 0
}

func runImpact(ctx context.Context, workDir string, args []string, stdout, stderr io.Writer) int {
	if len(args) != 1 {
		return usageError(stderr, "impact requires exactly one current REF")
	}
	ref, revision, err := repository.ParseSelector(args[0])
	if err != nil {
		return usageError(stderr, "invalid impact REF: %v", err)
	}
	if revision != "" {
		return usageError(stderr, "impact accepts a logical REF, not a historical REF@SEAL selector")
	}
	repo, err := repository.OpenStandalone(workDir)
	if err != nil {
		return commandError(stderr, "impact", err)
	}
	head, impacts, err := repo.Impact(ctx, ref)
	if err != nil {
		return commandError(stderr, "impact", err)
	}
	fmt.Fprintf(stdout, "SOURCE %s@%s\n", ref, head)
	if len(impacts) == 0 {
		fmt.Fprintln(stdout, "NO_IMPACT")
		return 0
	}
	for _, impact := range impacts {
		kind := "TRANSITIVE"
		if impact.Direct {
			kind = "DIRECT"
		}
		fmt.Fprintf(stdout, "%s %s@%s path=%s\n", kind, impact.REF, impact.Head, formatSealPath(impact.Path))
	}
	return 0
}

func runGraph(ctx context.Context, workDir string, args []string, stdout, stderr io.Writer) int {
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
	if len(nodes) == 0 {
		fmt.Fprintln(stdout, "EMPTY")
		return 0
	}
	for _, node := range nodes {
		head := "-"
		if node.Status.Head != nil {
			head = node.Status.Head.String()
		}
		fmt.Fprintf(stdout, "REF %s@%s %s\n", node.Status.REF, head, strings.Join(node.Status.Labels(), ","))
		for _, link := range node.Links {
			state := "HEAD"
			if !link.Link.TargetSeal.Equal(link.CurrentHead) {
				state = "HISTORICAL head=" + link.CurrentHead.String()
			}
			fmt.Fprintf(stdout, "  depend-on %s@%s %s message=%s\n", link.Link.TargetREF, link.Link.TargetSeal, state, quoteHumanString(link.Link.Message))
		}
	}
	return 0
}

func printStatuses(stdout io.Writer, statuses []repository.RefStatus) {
	for _, status := range statuses {
		fmt.Fprintf(stdout, "%s %s", status.REF, strings.Join(status.Labels(), ","))
		if status.Head != nil {
			fmt.Fprintf(stdout, " %s", status.Head)
		}
		fmt.Fprintln(stdout)
		for _, stale := range status.StaleDirect {
			fmt.Fprintf(stdout, "  %s sealed=%s head=%s\n", stale.Link.TargetREF, stale.Link.TargetSeal, stale.CurrentHead)
		}
		for _, stale := range status.StaleTransitive {
			path := append([]repositoryPathIdentity(nil), makePathIdentities(stale.Nodes)...)
			path = append(path, repositoryPathIdentity{REF: stale.Link.TargetREF, Seal: stale.Link.TargetSeal.String()})
			fmt.Fprintf(stdout, "  transitive path=%s head=%s@%s\n", formatRepositoryPath(path), stale.Link.TargetREF, stale.CurrentHead)
		}
	}
}

type repositoryPathIdentity struct {
	REF  string
	Seal string
}

func makePathIdentities(nodes []graph.SealIdentity) []repositoryPathIdentity {
	result := make([]repositoryPathIdentity, len(nodes))
	for i, node := range nodes {
		result[i] = repositoryPathIdentity{REF: node.REF, Seal: node.Seal.String()}
	}
	return result
}

func formatRepositoryPath(path []repositoryPathIdentity) string {
	parts := make([]string, len(path))
	for i, identity := range path {
		parts[i] = identity.REF + "@" + identity.Seal
	}
	return strings.Join(parts, " -> ")
}

func formatSealPath(path []graph.SealIdentity) string {
	return formatRepositoryPath(makePathIdentities(path))
}

func parseDependencies(values []string, message string) ([]repository.Dependency, error) {
	if values == nil {
		return nil, nil
	}
	dependencies := make([]repository.Dependency, 0, len(values))
	for _, value := range values {
		ref, revision, err := repository.ParseSelector(value)
		if err != nil {
			return nil, fmt.Errorf("invalid --depend-on %q: %w", value, err)
		}
		dependencies = append(dependencies, repository.Dependency{REF: ref, Revision: revision, Message: message})
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
  sealgraph add REF (--content CONTENT | --content-file PATH_OR_DASH) [--root] [--draft] [--depend-on REF[@SEAL_OR_TAG]]...
  sealgraph link REF --depend-on REF[@SEAL_OR_TAG]... [-m LINK_MESSAGE]
  sealgraph unlink REF --upstream REF[@SEAL_OR_TAG]
  sealgraph candidate show REF [--raw-content]
  sealgraph candidate diff REF
  sealgraph candidate discard REF
  sealgraph tag REF [TAGNAME]
  sealgraph tag REF@SEAL_OR_TAG TAGNAME
  sealgraph seal REF
  sealgraph show REF[@SEAL] [--raw-content]
  sealgraph log REF
  sealgraph linklog REF [--upstream REF]
  sealgraph diff REF
  sealgraph diff REF@OLD REF@NEW
  sealgraph status [REF]
  sealgraph stale [--frontier] [--refs-only]
  sealgraph impact REF
  sealgraph graph

Each seal operation advances exactly one logical REF.

stale reports a validated observation of current REF heads.
The result is not a reservation or batch plan.
Re-run after each link or seal operation; seal revalidates dependencies before publication.
`)
}

func printStaleHelp(w io.Writer) {
	fmt.Fprint(w, `Usage:
  sealgraph stale [--frontier] [--refs-only]

Reports current REF heads with derived direct or transitive staleness.
--frontier selects only stale REFs with no direct upstream REF that is stale.
--refs-only emits the selected logical REFs, one per line, with no other text.

Reports a validated observation of current REF heads.
The result is not a reservation or batch plan.
Re-run after each link or seal operation; seal revalidates dependencies before publication.
`)
}

func printGitHelp(w io.Writer) {
	fmt.Fprint(w, `git sealgraph — Git sidecar integration for sealgraph

Usage:
  git sealgraph <command> [options]

This separate surface is planned and is not implemented by the standalone Phase 1 slice.
`)
}
