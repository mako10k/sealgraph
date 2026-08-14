package cli

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/mako10k/sealgraph/internal/graph"
	"github.com/mako10k/sealgraph/internal/repository"
)

const version = "0.1.0-dev"

func RunStandalone(args []string, stdout, stderr io.Writer) int {
	workDir, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(stderr, "sealgraph: determine current directory: %v\n", err)
		return 3
	}
	return runStandaloneAt(workDir, args, stdout, stderr)
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
		return runAdd(ctx, workDir, args[1:], stdout, stderr)
	case "link":
		return runLink(ctx, workDir, args[1:], stdout, stderr)
	case "seal":
		return runSeal(ctx, workDir, args[1:], stdout, stderr)
	case "show":
		return runShow(ctx, workDir, args[1:], stdout, stderr)
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

func runAdd(ctx context.Context, workDir string, args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		return usageError(stderr, "add requires exactly one REF")
	}
	ref := args[0]
	flags := flag.NewFlagSet("add", flag.ContinueOnError)
	flags.SetOutput(stderr)
	var content trackedString
	var depends stringList
	flags.Var(&content, "content", "exact content bytes supplied as a command argument")
	flags.Var(&depends, "depend-on", "dependency REF or REF@SEAL (repeatable)")
	root := flags.Bool("root", false, "declare a provenance root")
	draft := flags.Bool("draft", false, "mark the candidate draft")
	if err := flags.Parse(args[1:]); err != nil {
		return 2
	}
	if flags.NArg() != 0 {
		return usageError(stderr, "add accepts exactly one REF; unexpected argument %q", flags.Arg(0))
	}
	if !content.set {
		return usageError(stderr, "add requires --content")
	}
	dependencies, err := parseDependencies(depends)
	if err != nil {
		return usageError(stderr, "%v", err)
	}
	repo, err := repository.OpenStandalone(workDir, nil)
	if err != nil {
		return commandError(stderr, "add", err)
	}
	candidate, err := repo.Add(ctx, repository.AddOptions{REF: ref, Content: []byte(content.value), Dependencies: dependencies, Root: *root, Draft: *draft})
	if err != nil {
		return commandError(stderr, "add", err)
	}
	fmt.Fprintf(stdout, "CANDIDATE %s content=%s dependencies=%d root=%t draft=%t\n", candidate.REF, candidate.Content.ID, len(candidate.Links), candidate.Root, candidate.Draft)
	return 0
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
	if err := flags.Parse(args[1:]); err != nil {
		return 2
	}
	if flags.NArg() != 0 {
		return usageError(stderr, "link accepts exactly one REF; unexpected argument %q", flags.Arg(0))
	}
	if len(depends) == 0 {
		return usageError(stderr, "link requires at least one --depend-on")
	}
	dependencies, err := parseDependencies(depends)
	if err != nil {
		return usageError(stderr, "%v", err)
	}
	repo, err := repository.OpenStandalone(workDir, nil)
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

func runSeal(ctx context.Context, workDir string, args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		return usageError(stderr, "seal requires exactly one REF")
	}
	if args[0] == "--all" {
		return usageError(stderr, "seal --all is forbidden; seal exactly one REF explicitly")
	}
	ref := args[0]
	flags := flag.NewFlagSet("seal", flag.ContinueOnError)
	flags.SetOutput(stderr)
	var message trackedString
	flags.Var(&message, "m", "required seal message")
	if err := flags.Parse(args[1:]); err != nil {
		return 2
	}
	if flags.NArg() != 0 {
		return usageError(stderr, "seal accepts exactly one REF; unexpected additional REF or argument %q", flags.Arg(0))
	}
	if !message.set || message.value == "" {
		return usageError(stderr, "seal requires a non-empty -m MESSAGE")
	}
	repo, err := repository.OpenStandalone(workDir, nil)
	if err != nil {
		return commandError(stderr, "seal", err)
	}
	result, err := repo.Seal(ctx, ref, message.value)
	if err != nil {
		return commandError(stderr, "seal", err)
	}
	fmt.Fprintf(stdout, "SEALED %s %s\n", ref, result.ID)
	return 0
}

func runShow(ctx context.Context, workDir string, args []string, stdout, stderr io.Writer) int {
	if len(args) != 1 {
		return usageError(stderr, "show requires exactly one REF or REF@SEAL")
	}
	ref, explicit, err := repository.ParseSelector(args[0])
	if err != nil {
		return usageError(stderr, "invalid selector: %v", err)
	}
	repo, err := repository.OpenStandalone(workDir, nil)
	if err != nil {
		return commandError(stderr, "show", err)
	}
	result, err := repo.Show(ctx, ref, explicit)
	if err != nil {
		return commandError(stderr, "show", err)
	}
	fmt.Fprintf(stdout, "REF %s\nSEAL %s\n", result.Payload.REF, result.ID)
	if result.Payload.Parent == nil {
		fmt.Fprintln(stdout, "PARENT -")
	} else {
		fmt.Fprintf(stdout, "PARENT %s\n", result.Payload.Parent)
	}
	fmt.Fprintf(stdout, "CONTENT %s\n%s\nMESSAGE %s\nROOT %t\nDRAFT %t\nCREATED_AT %s\nDEPENDENCIES %d\n", result.Payload.Content.ID, result.Content, result.Payload.Message, result.Payload.Root, result.Payload.Draft, result.Payload.CreatedAt, len(result.Payload.Links))
	for _, link := range result.Payload.Links {
		fmt.Fprintf(stdout, "  %s %s@%s\n", link.Relation, link.TargetREF, link.TargetSeal)
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
	repo, err := repository.OpenStandalone(workDir, nil)
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
	if len(args) != 0 {
		return usageError(stderr, "stale accepts no arguments")
	}
	repo, err := repository.OpenStandalone(workDir, nil)
	if err != nil {
		return commandError(stderr, "stale", err)
	}
	statuses, err := repo.Stale(ctx)
	if err != nil {
		return commandError(stderr, "stale", err)
	}
	if len(statuses) == 0 {
		fmt.Fprintln(stdout, "CLEAN")
		return 0
	}
	printStatuses(stdout, statuses)
	return 0
}

func runImpact(ctx context.Context, workDir string, args []string, stdout, stderr io.Writer) int {
	if len(args) != 1 {
		return usageError(stderr, "impact requires exactly one current REF")
	}
	ref, explicit, err := repository.ParseSelector(args[0])
	if err != nil {
		return usageError(stderr, "invalid impact REF: %v", err)
	}
	if explicit != nil {
		return usageError(stderr, "impact accepts a logical REF, not a historical REF@SEAL selector")
	}
	repo, err := repository.OpenStandalone(workDir, nil)
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
	repo, err := repository.OpenStandalone(workDir, nil)
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
			fmt.Fprintf(stdout, "  %s %s@%s %s\n", link.Link.Relation, link.Link.TargetREF, link.Link.TargetSeal, state)
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

func parseDependencies(values []string) ([]repository.Dependency, error) {
	if values == nil {
		return nil, nil
	}
	dependencies := make([]repository.Dependency, 0, len(values))
	for _, value := range values {
		ref, seal, err := repository.ParseSelector(value)
		if err != nil {
			return nil, fmt.Errorf("invalid --depend-on %q: %w", value, err)
		}
		dependencies = append(dependencies, repository.Dependency{REF: ref, Seal: seal})
	}
	return dependencies, nil
}

type trackedString struct {
	value string
	set   bool
}

func (v *trackedString) String() string         { return v.value }
func (v *trackedString) Set(value string) error { v.value, v.set = value, true; return nil }

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
  sealgraph add REF --content CONTENT [--root] [--draft] [--depend-on REF[@SEAL]]...
  sealgraph link REF --depend-on REF[@SEAL]...
  sealgraph seal REF -m MESSAGE
  sealgraph show REF[@SEAL]
  sealgraph status [REF]
  sealgraph stale
  sealgraph impact REF
  sealgraph graph

Each seal operation advances exactly one logical REF.
`)
}

func printGitHelp(w io.Writer) {
	fmt.Fprint(w, `git sealgraph — Git sidecar integration for sealgraph

Usage:
  git sealgraph <command> [options]

This separate surface is planned and is not implemented by the standalone Phase 1 slice.
`)
}
