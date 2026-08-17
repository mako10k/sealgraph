package cli

import (
	"fmt"
	"io"
	"sort"
	"strings"
)

type helpOption struct {
	Syntax      string
	Description string
}

type commandHelp struct {
	Path        string
	Summary     string
	Usage       []string
	Arguments   []string
	Options     []helpOption
	Details     []string
	Examples    []string
	Related     []string
	Subcommands []string
}

var commandHelpRegistry = map[string]commandHelp{
	"init": {
		Path: "init", Summary: "Initialize or explicitly bootstrap one standalone repository.",
		Usage:    []string{"sealgraph init"},
		Details:  []string{"Standalone init uses only .sealgraph and never detects or inspects Git. It does not migrate or repair canonical state."},
		Examples: []string{"sealgraph init"}, Related: []string{"concepts", "usecases"},
	},
	"manifest": {
		Path: "manifest", Summary: "Build a deterministic path/digest claim from explicit files without repository mutation.",
		Usage:     []string{"sealgraph manifest --source SOURCE --file PATH [--file PATH ...]"},
		Arguments: []string{"No positional arguments."},
		Options:   []helpOption{{"--source SOURCE", "required exactly once; explicit source identity"}, {"--file PATH", "required, repeatable; portable relative regular-file path"}},
		Details:   []string{"No glob, recursion, path normalization, Git discovery, object write, candidate edit, or seal occurs. The output is a path-digest-only claim."},
		Examples:  []string{"sealgraph manifest --source review:2026-08-17 --file docs/requirements.md --file docs/architecture.md"}, Related: []string{"add", "concepts"},
	},
	"add": {
		Path: "add", Summary: "Create or update the working candidate for exactly one REF.",
		Usage:     []string{"sealgraph add REF (--content CONTENT | --content-file PATH_OR_DASH) [--parent SELECTOR] [--root] [--draft] [--depend-on SELECTOR]..."},
		Arguments: []string{"REF (required): destination logical REF; it is not a branch or checkout target."},
		Options:   []helpOption{{"--content CONTENT", "required exactly once unless --content-file is used"}, {"--content-file PATH|-", "required exactly once unless --content is used; '-' reads exact stdin bytes"}, {"--parent SELECTOR", "optional, once; exact revision parent for an absent destination only"}, {"--root", "optional flag; declare this generation a provenance boundary"}, {"--draft", "optional flag; preserve provisional or historical Cause provenance"}, {"--depend-on SELECTOR", "optional, repeatable; replaces the dependency set when present"}},
		Details:   []string{"--content and --content-file are mutually exclusive. A root has no Cause Links. Existing dependency Links remain unless --depend-on is supplied. Bare REF dependencies resolve HEAD now; candidates persist full SealIDs."},
		Examples:  []string{"sealgraph add premise --root --content 'External premise'", "sealgraph add design/api --content-file design.md --depend-on requirements/api", "sealgraph add revised/api --parent design/api@abcd --content 'new material'"}, Related: []string{"seal", "link", "candidate show", "selectors"},
	},
	"derive": {
		Path: "derive", Summary: "Create an absent REF candidate by copying one Seal's material and using it as parent_revision.",
		Usage: []string{"sealgraph derive NEW_REF --from SOURCE_SELECTOR"}, Arguments: []string{"NEW_REF (required): absent destination REF."},
		Options: []helpOption{{"--from SOURCE_SELECTOR", "required exactly once; any valid Seal selector"}},
		Details: []string{"derive does not publish, move a REF, copy tags, or infer Seal ownership."}, Examples: []string{"sealgraph derive preserved/api --from @abcd"}, Related: []string{"selectors", "candidate show", "seal"},
	},
	"link": {
		Path: "link", Summary: "Add exact Cause dependencies to one working candidate without replacing content.",
		Usage: []string{"sealgraph link REF --depend-on SELECTOR [--depend-on SELECTOR ...] [-m LINK_MESSAGE]"}, Arguments: []string{"REF (required): candidate REF."},
		Options: []helpOption{{"--depend-on SELECTOR", "required, repeatable; resolved before candidate persistence"}, {"-m LINK_MESSAGE", "optional, once; rationale applied to each dependency in this invocation"}},
		Details: []string{"Each Link stores one exact full upstream SealID. The message is identity-bearing edge rationale, not approval, actor, or time."}, Examples: []string{"sealgraph link design/api --depend-on requirements/api -m 'API design is based on this requirement generation'"}, Related: []string{"unlink", "candidate show", "selectors", "concepts cause"},
	},
	"unlink": {
		Path: "unlink", Summary: "Remove exactly one resolved Cause target from one candidate.",
		Usage: []string{"sealgraph unlink REF --upstream SELECTOR"}, Arguments: []string{"REF (required): candidate REF."}, Options: []helpOption{{"--upstream SELECTOR", "required exactly once; exact target to remove"}},
		Details: []string{"A bare upstream REF resolves its current HEAD and will not match an older stored target. Inspect candidate show and use the displayed @SealID when removing a historical edge."}, Examples: []string{"sealgraph unlink design/api --upstream @abcd"}, Related: []string{"link", "candidate show", "selectors"},
	},
	"tag": {
		Path: "tag", Summary: "List one REF's immutable tags or create one immutable scoped tag binding.",
		Usage:     []string{"sealgraph tag REF", "sealgraph tag REF TAGNAME", "sealgraph tag REF@SEAL_OR_TAG TAGNAME"},
		Arguments: []string{"One argument lists tags. Two arguments create a binding. TAGNAME is non-empty UTF-8; @, controls, DEL, and 4-64 lower-hex names are forbidden."},
		Details:   []string{"A global @SEAL_TOKEN has no REF scope and cannot create a tag. Repeating the same binding is idempotent; retarget, delete, force, and unscoped creation are absent."}, Examples: []string{"sealgraph tag design/api", "sealgraph tag design/api reviewed/1.0", "sealgraph tag design/api@reviewed/1.0 archived"}, Related: []string{"selectors", "show"},
	},
	"mv": {
		Path: "mv", Summary: "Atomically move exactly one REF manifest and its complete tag namespace.", Usage: []string{"sealgraph mv OLD_REF NEW_REF"}, Arguments: []string{"OLD_REF and NEW_REF are required, distinct, explicit REFs; destination must be absent."},
		Details: []string{"Candidates at either name block the move. mv does not recurse, move a candidate, create an alias, or rewrite Seals or Links."}, Examples: []string{"sealgraph mv design/api archive/design-api"}, Related: []string{"candidate show", "candidate discard", "tag"},
	},
	"candidate": {
		Path: "candidate", Summary: "Inspect, compare, or explicitly discard mutable candidate state.", Usage: []string{"sealgraph candidate <show|diff|discard> ..."}, Subcommands: []string{"show", "diff", "discard"}, Details: []string{"Candidate operations never rebase, relink, repair, or seal automatically."}, Related: []string{"candidate show", "candidate diff", "candidate discard", "seal"},
	},
	"candidate show": {
		Path: "candidate show", Summary: "Inspect one candidate and its parent_revision and expected REF-head relations.", Usage: []string{"sealgraph candidate show REF [--raw-content]"}, Arguments: []string{"REF (required): exact candidate REF, not a Seal selector."}, Options: []helpOption{{"--raw-content", "optional; stdout becomes exact content bytes only"}}, Details: []string{"Inspection validates material and exact Cause targets and does not mutate or bootstrap a repository."}, Examples: []string{"sealgraph candidate show design/api"}, Related: []string{"candidate diff", "candidate discard", "seal"},
	},
	"candidate diff": {
		Path: "candidate diff", Summary: "Compare one candidate with its recorded parent_revision.", Usage: []string{"sealgraph candidate diff REF"}, Arguments: []string{"REF (required): exact candidate REF."}, Details: []string{"Publication expectation is reported separately from immutable material differences."}, Examples: []string{"sealgraph candidate diff design/api"}, Related: []string{"candidate show", "diff", "seal"},
	},
	"candidate discard": {
		Path: "candidate discard", Summary: "Explicitly remove exactly one candidate and no canonical state.", Usage: []string{"sealgraph candidate discard REF"}, Arguments: []string{"REF (required): exact candidate REF."}, Details: []string{"This removes no Seal, object, REF, tag, or descendant candidate. There is no recursive or force form."}, Examples: []string{"sealgraph candidate discard design/api"}, Related: []string{"candidate show", "add"},
	},
	"seal": {
		Path: "seal", Summary: "Publish at most one new immutable Seal for exactly one REF.", Usage: []string{"sealgraph seal REF"}, Arguments: []string{"REF (required): REF whose candidate is reviewed and published."},
		Details: []string{"A normal non-draft candidate requires every direct and reachable Cause target to be a non-draft active revision leaf. Draft may preserve intentional historical provenance. There is no --all, --force, automatic relink, or automatic stale repair."}, Examples: []string{"sealgraph seal premise", "sealgraph candidate diff design/api\nsealgraph seal design/api"}, Related: []string{"candidate show", "candidate diff", "status", "stale", "concepts stale"},
	},
	"show":    inspectionHelp("show", "Inspect one immutable Seal generation and its exact material and Cause Links.", "sealgraph show SELECTOR [--raw-content] [--format human|json]", []helpOption{{"--raw-content", "optional; exact content bytes only; conflicts with --format json"}}),
	"log":     inspectionHelp("log", "Follow parent_revision history newest-first for one current REF.", "sealgraph log REF [--format human|json]", nil),
	"linklog": inspectionHelp("linklog", "Show Cause-Link changes across parent_revision history.", "sealgraph linklog REF [--upstream SELECTOR] [--format human|json]", []helpOption{{"--upstream SELECTOR", "optional, once; filter changes involving one resolved Seal"}}),
	"diff":    inspectionHelp("diff", "Compare immutable Seal material and provenance.", "sealgraph diff REF [--format human|json]\nsealgraph diff SELECTOR SELECTOR [--format human|json]", nil),
	"status":  inspectionHelp("status", "Report candidate, draft, revision-stale, and Cause-stale facts.", "sealgraph status [REF] [--format human|json]", nil),
	"stale": {
		Path: "stale", Summary: "List stale current REF heads or the upstream-first review frontier.", Usage: []string{"sealgraph stale [--frontier] [--refs-only] [--scan] [--format human|json]"},
		Options: []helpOption{{"--frontier", "optional; keep only stale heads not blocked by another stale current head in strict Cause closure"}, {"--refs-only", "optional; stable REF-only line protocol; conflicts with --format json"}, {"--scan", "optional; bypass disposable cache reads"}, {"--format human|json", "optional, once; default human"}},
		Details: []string{"Stale is derived current review state. It is not structural impact, candidate state, approval, or an automatic repair plan. --scan does not repair canonical state."}, Examples: []string{"sealgraph stale --frontier", "sealgraph stale --frontier --refs-only --scan"}, Related: []string{"status", "show", "candidate diff", "concepts stale", "impact"},
	},
	"impact": {
		Path: "impact", Summary: "Report current downstream Cause reachability from a selected Seal or its revision ancestors.", Usage: []string{"sealgraph impact [--all-paths] [--max-paths N] SELECTOR [--format human|json]"}, Arguments: []string{"SELECTOR (required): REF, @SEAL_TOKEN, or REF@TOKEN."},
		Options: []helpOption{{"--all-paths", "optional; emit bounded distinct simple paths instead of one shortest path"}, {"--max-paths N", "optional, once; positive per-downstream limit, requires --all-paths; default 100"}, {"--format human|json", "optional, once; default human"}},
		Details: []string{"Path truncation never removes impact membership, skips graph validation, or weakens snapshot revalidation. STRUCTURAL_IMPACT is not a stale-only result."}, Examples: []string{"sealgraph impact requirements/api", "sealgraph impact --all-paths --max-paths 20 requirements/api"}, Related: []string{"selectors", "stale", "graph", "concepts structural-impact"},
	},
	"graph": inspectionHelp("graph", "Inspect the active revision DAG and exact Cause edges.", "sealgraph graph [--format human|json]", nil),
	"fsck":  inspectionHelp("fsck", "Validate the complete standalone object, REF/tag, material, revision, and Cause inventory without repair.", "sealgraph fsck [--format human|json]", nil),
	"load": {
		Path: "load", Summary: "Atomically load one canonical logical-v1 migration document into an absent target.", Usage: []string{"sealgraph load --format logical-v1 < repository.dump.json"}, Arguments: []string{"No positional arguments; stdin is the exact canonical document."}, Options: []helpOption{{"--format logical-v1", "required exactly once; no other value is accepted"}}, Details: []string{"load never merges, replaces, repairs, or directly opens a format-3 repository."}, Examples: []string{"sealgraph load --format logical-v1 < repository.dump.json"}, Related: []string{"init", "fsck"},
	},
}

func inspectionHelp(path, summary, usage string, extra []helpOption) commandHelp {
	options := append([]helpOption{}, extra...)
	options = append(options, helpOption{"--format human|json", "optional, once; default human"})
	return commandHelp{Path: path, Summary: summary, Usage: strings.Split(usage, "\n"), Options: options, Details: []string{"Read-only inspection does not bootstrap, mutate, repair, relink, reseal, or inspect Git."}, Related: []string{"selectors", "concepts"}}
}

func printRootHelp(w io.Writer) {
	fmt.Fprint(w, "sealgraph — standalone provenance sealing CLI\n\nUsage:\n  sealgraph <command> [options]\n  sealgraph help [COMMAND [SUBCOMMAND]]\n  sealgraph help <concepts|selectors|usecases>\n\nCommands:\n")
	paths := make([]string, 0, len(commandHelpRegistry))
	for path := range commandHelpRegistry {
		if !strings.Contains(path, " ") {
			paths = append(paths, path)
		}
	}
	sort.Strings(paths)
	for _, path := range paths {
		entry := commandHelpRegistry[path]
		fmt.Fprintf(w, "  %-10s %s\n", path, entry.Summary)
	}
	fmt.Fprint(w, "\nTopics:\n  selectors  exact selector grammar and resolution\n  concepts   provenance concepts and common distinctions\n  usecases   copyable explicit review workflows\n\nHelp routes:\n  sealgraph --help\n  sealgraph help <command>\n  sealgraph <command> --help\n  sealgraph help candidate show\n\nEach seal operation advances exactly one REF. Navigation explains explicit next actions; it never repairs, relinks, reseals, or selects a REF automatically. Standalone operation uses only explicit inputs and .sealgraph; it does not discover or inspect Git.\n")
	fmt.Fprint(w, "\nSemantic legend:\n  CLEAN does not compare working files.\n  REF is a movable logical identity, not a branch or checkout target.\n  STRUCTURAL_IMPACT is Cause reachability; stale is current review state.\n  root marks a provenance boundary, not truth or trust.\n  log/linklog are Seal revision/Cause histories, not Git histories.\n")
}

func printCommandHelp(w io.Writer, entry commandHelp) {
	fmt.Fprintf(w, "%s — %s\n\nUsage:\n", entry.Path, entry.Summary)
	for _, usage := range entry.Usage {
		for _, line := range strings.Split(usage, "\n") {
			fmt.Fprintf(w, "  %s\n", line)
		}
	}
	printHelpLines(w, "Arguments", entry.Arguments)
	if len(entry.Options) != 0 {
		fmt.Fprint(w, "\nOptions:\n")
		for _, option := range entry.Options {
			fmt.Fprintf(w, "  %-28s %s\n", option.Syntax, option.Description)
		}
	}
	printHelpLines(w, "Important", entry.Details)
	if len(entry.Subcommands) != 0 {
		fmt.Fprint(w, "\nSubcommands:\n")
		for _, subcommand := range entry.Subcommands {
			child := commandHelpRegistry[entry.Path+" "+subcommand]
			fmt.Fprintf(w, "  %-10s %s\n", subcommand, child.Summary)
		}
	}
	printHelpLines(w, "Examples / use cases", entry.Examples)
	if len(entry.Related) != 0 {
		fmt.Fprint(w, "\nRelated:\n")
		for _, topic := range entry.Related {
			fmt.Fprintf(w, "  sealgraph help %s\n", topic)
		}
	}
}

func printHelpLines(w io.Writer, title string, lines []string) {
	if len(lines) == 0 {
		return
	}
	fmt.Fprintf(w, "\n%s:\n", title)
	for _, line := range lines {
		for _, part := range strings.Split(line, "\n") {
			fmt.Fprintf(w, "  %s\n", part)
		}
	}
}

func printSelectorsHelp(w io.Writer) {
	fmt.Fprint(w, `selectors — immutable Seal selection grammar

Forms:
  REF          resolve the REF's current HEAD at operation time
  @SEAL_TOKEN  resolve a repository-wide unique native ODB prefix
  REF@TOKEN    resolve a Seal inside one REF UI scope

Rules:
  A hexadecimal token is 4 through 64 lower-case hex characters.
  REF@hex requires the selected Seal to be current HEAD or a parent_revision
  ancestor of that HEAD. Use @SEAL_TOKEN for a sibling or detached Seal.
  REF@non-hex resolves an immutable tag in that REF's namespace.
  A bare hexadecimal Seal token is not accepted because it can be a valid REF.
  There is no @latest shortcut. Bare REF is the explicit current-HEAD form.
  Links and candidates persist only the resolved full 64-character SealID;
  selector spelling and dynamic HEAD references are never persisted.

Examples:
  sealgraph show requirements/api
  sealgraph show @1a2b
  sealgraph show requirements/api@reviewed/1.0
  sealgraph link design/api --depend-on @1a2b

Related:
  sealgraph help concepts ref
  sealgraph help concepts seal
  sealgraph help tag
`)
}

var conceptHelp = map[string]string{
	"ref":               "REF is a movable logical lookup/publication name, not immutable Seal identity, a branch, or a checkout target. Each REF has at most one current HEAD.",
	"seal":              "Seal is an immutable snapshot of exact material, direct Cause SealIDs, optional parent_revision, root, and draft. REF names, tags, actor, and time are not Seal bytes.",
	"parent-revision":   "parent_revision is one exact derivation parent. It does not mean replacement, preference, ownership, approval, or Cause dependency.",
	"cause":             "A Cause Link records one exact upstream Seal generation. It is distinct from parent_revision and persists a full SealID, never a dynamic REF HEAD.",
	"root":              "root is an explicit provenance boundary for one generation. It does not mean true, trusted, or approved.",
	"draft":             "draft is explicit provisional sealing. It may preserve historical or non-leaf Causes, remains observable, and does not propagate automatically.",
	"candidate":         "candidate is mutable unsealed state for one destination REF. It is not history or approval and is inspected, changed, discarded, or sealed explicitly.",
	"tag":               "tag is an immutable REF-scoped alias to one exact Seal. It is external to Seal bytes and is not a branch, dynamic Link, or approval claim.",
	"selector":          "selector resolves one immutable Seal using REF, @SEAL_TOKEN, or REF@TOKEN. Run `sealgraph help selectors` for the exact grammar.",
	"stale":             "stale is derived from immutable Seals plus current REF heads. Revision self-stale and direct/transitive Cause stale are review facts, not persisted state or automatic work.",
	"structural-impact": "STRUCTURAL_IMPACT is downstream Cause reachability from a selected generation or its revision ancestors. It is broader than current stale state.",
	"history":           "Revision history follows parent_revision. Cause history compares exact Links across revisions. Neither is Git commit or reflog history.",
}

func printConceptsHelp(w io.Writer, topic string) bool {
	if topic != "" {
		text, ok := conceptHelp[topic]
		if !ok {
			return false
		}
		fmt.Fprintf(w, "concepts %s\n\n%s\n", topic, text)
		return true
	}
	fmt.Fprint(w, "concepts — SealGraph domain semantics\n\n")
	order := []string{"ref", "seal", "parent-revision", "cause", "root", "draft", "candidate", "tag", "selector", "stale", "structural-impact", "history"}
	for _, name := range order {
		fmt.Fprintf(w, "%s:\n  %s\n\n", name, conceptHelp[name])
	}
	fmt.Fprint(w, "CLEAN means no candidate and no derived stale relation; it does not compare working files. Standalone SealGraph never searches for Git.\n")
	return true
}

func printUseCasesHelp(w io.Writer) {
	fmt.Fprint(w, `usecases — explicit provenance workflows

Create the first root:
  sealgraph add premise --root --content 'External premise'
  sealgraph seal premise

Seal content based on an upstream HEAD:
  sealgraph add design/api --content-file design.md --depend-on requirements/api
  sealgraph seal design/api

Add dependency rationale:
  sealgraph link design/api --depend-on requirements/api -m 'API design is based on this requirement generation'

Review a candidate explicitly:
  sealgraph candidate show design/api
  sealgraph candidate diff design/api
  sealgraph seal design/api

Review stale provenance upstream-first:
  sealgraph stale --frontier
  sealgraph status design/api
  sealgraph show requirements/api
  sealgraph candidate diff design/api
  sealgraph seal design/api

Inspect structural impact:
  sealgraph impact requirements/api
  sealgraph impact --all-paths --max-paths 20 requirements/api

Select a historical generation explicitly:
  sealgraph show requirements/api@reviewed/1.0
  sealgraph show @1a2b
  sealgraph link design/api --depend-on @1a2b

These are review/navigation examples, not automatic repair procedures. Relink,
draft selection, and each one-REF seal remain explicit operator decisions.
`)
}

func printHelpTopic(w io.Writer, path []string) bool {
	if len(path) == 0 {
		printRootHelp(w)
		return true
	}
	key := strings.Join(path, " ")
	if entry, ok := commandHelpRegistry[key]; ok {
		printCommandHelp(w, entry)
		return true
	}
	switch path[0] {
	case "selectors":
		if len(path) == 1 {
			printSelectorsHelp(w)
			return true
		}
	case "usecases":
		if len(path) == 1 {
			printUseCasesHelp(w)
			return true
		}
	case "concepts":
		if len(path) <= 2 {
			topic := ""
			if len(path) == 2 {
				topic = path[1]
			}
			return printConceptsHelp(w, topic)
		}
	}
	return false
}

func commandUsage(path string) string {
	entry, ok := commandHelpRegistry[path]
	if !ok || len(entry.Usage) == 0 {
		return "sealgraph " + path
	}
	return entry.Usage[0]
}
