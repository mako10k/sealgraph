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
		Details:  []string{"New repositories use repository format 5. Standalone init uses only .sealgraph and never detects or inspects Git. Existing format-4 repositories fail before mutation with FORMAT4_REQUIRES_MIGRATION and the explicit extract/load commands."},
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
		Usage:     []string{"sealgraph add REF [--content CONTENT | --content-file PATH_OR_DASH] [--bind-source] [--draft] [--root --clear-cause-links | --non-root] [--target TARGET (--previous PREVIOUS ... | --no-previous) [-m MESSAGE ...]]"},
		Arguments: []string{"REF (required): destination logical REF; it is not a branch or checkout target."},
		Options:   []helpOption{{"--content CONTENT", "optional exact bytes; conflicts with --content-file and --bind-source"}, {"--content-file PATH|-", "optional exact file/stdin source; named PATH may use --bind-source"}, {"--bind-source", "bind the named source after candidate publication; never retargets"}, {"--draft", "optional flag; preserve provisional or historical Cause provenance"}, {"--root --clear-cause-links", "declare a root and atomically remove every Cause Link"}, {"--non-root", "declare non-root; the result must contain a Cause Link"}, {"--target TARGET", "one complete Cause Link target; resolves to an exact Seal"}, {"--previous PREVIOUS", "repeatable exact previous revision asserted for TARGET"}, {"--no-previous", "assert that TARGET has no previous revision; conflicts with --previous"}, {"-m MESSAGE", "repeatable identity-bearing message for the target record"}},
		Details:   []string{"--content and --content-file are mutually exclusive. Without explicit content, add refreshes from a bound source while preserving semantic state. A new root requires --root --clear-cause-links. A new non-root requires --non-root and one complete --target group. Existing Candidates preserve omitted root mode and omitted Cause group. All selectors in a group resolve before persistence."},
		Examples:  []string{"sealgraph add premise --root --clear-cause-links --content 'External premise'", "sealgraph add docs/spec.md --root --clear-cause-links --bind-source", "sealgraph add design/api --content-file design.md --non-root --target requirements/api --no-previous", "sealgraph add design/api --target requirements/api --previous @abcd -m 'reviewed revision relation'"}, Related: []string{"source", "seal", "link", "candidate show", "selectors"},
	},
	"source": {
		Path: "source", Summary: "Manage and compare non-canonical local REF-to-file source bindings.", Usage: []string{"sealgraph source <bind|rebind|unbind|show|list|compare> ..."},
		Subcommands: []string{"bind", "rebind", "unbind", "show", "list", "compare"}, Details: []string{"A local source binding is not Git tracked state and never changes a candidate by itself."}, Related: []string{"add", "status"},
	},
	"source bind": {
		Path: "source bind", Summary: "Create one absent local source binding.", Usage: []string{"sealgraph source bind REF --file PATH [--format human|json]"}, Options: []helpOption{{"--file PATH", "required portable relative regular-file path"}, {"--format human|json", "optional, once; default terminal=human, non-terminal=JSON"}}, Details: []string{"Same binding is idempotent. A different existing path requires source rebind."}, Related: []string{"source show", "source rebind", "add"},
	},
	"source rebind": {
		Path: "source rebind", Summary: "Atomically replace one observed local source binding.", Usage: []string{"sealgraph source rebind REF --from OLD_PATH --file NEW_PATH [--format human|json]"}, Options: []helpOption{{"--from OLD_PATH", "required exact observed current path"}, {"--file NEW_PATH", "required validated new source path"}, {"--format human|json", "optional, once; default terminal=human, non-terminal=JSON"}}, Related: []string{"source show", "source bind"},
	},
	"source unbind": {
		Path: "source unbind", Summary: "Remove one exact observed local source binding.", Usage: []string{"sealgraph source unbind REF --from PATH [--format human|json]"}, Options: []helpOption{{"--from PATH", "required exact observed current path"}, {"--format human|json", "optional, once; default terminal=human, non-terminal=JSON"}}, Details: []string{"No candidate, REF, object, or Seal is removed."}, Related: []string{"source show", "source bind"},
	},
	"source show":    inspectionHelp("source show", "Show one local source binding without opening its source file.", "sealgraph source show REF [--format human|json]", nil),
	"source list":    inspectionHelp("source list", "List local source bindings without opening source files.", "sealgraph source list [--format human|json]", nil),
	"source compare": inspectionHelp("source compare", "Compare one bound workfile with its candidate-or-HEAD content baseline.", "sealgraph source compare REF [--format human|json]", nil),
	"link": {
		Path: "link", Summary: "Create or replace one complete Cause Link record on one Candidate.",
		Usage: []string{"sealgraph link REF --target TARGET (--previous PREVIOUS ... | --no-previous) [-m MESSAGE ...]"}, Arguments: []string{"REF (required): candidate REF."},
		Options: []helpOption{{"--target TARGET", "required exactly once; resolves to the record's exact target Seal"}, {"--previous PREVIOUS", "repeatable; exact previous revision asserted for TARGET"}, {"--no-previous", "assert no previous revision for TARGET; conflicts with --previous"}, {"-m MESSAGE", "repeatable identity-bearing message in this exact record"}},
		Details: []string{"The operation replaces the whole record for that exact target and preserves every other Candidate field and Cause Link. It never unions messages or previous revisions with an existing record."}, Examples: []string{"sealgraph link design/api --target requirements/api --no-previous -m 'API design is based on this generation'", "sealgraph link design/api --target requirements/api --previous @abcd"}, Related: []string{"unlink", "candidate show", "selectors", "concepts cause"},
	},
	"link-metadata": {
		Path: "link-metadata", Summary: "Set or remove one namespaced metadata entry on one exact Candidate Cause Link.",
		Usage: []string{"sealgraph link-metadata <set|remove> ..."}, Subcommands: []string{"set", "remove"},
		Details: []string{"Metadata is opaque, identity-bearing canonical JSON. These operations require repository format 6 and never create a Seal or move a REF."},
		Related: []string{"link-metadata set", "link-metadata remove", "candidate show", "link"},
	},
	"link-metadata set": {
		Path: "link-metadata set", Summary: "Add or replace one complete metadata entry on one exact Candidate Cause Link.",
		Usage:   []string{"sealgraph link-metadata set REF --target TARGET --namespace NAMESPACE (--schema SCHEMA | --no-schema) (--value-json JSON | --value-file PATH_OR_DASH) [--format human|json]"},
		Options: []helpOption{{"--target TARGET", "required exact Cause target selector"}, {"--namespace NAMESPACE", "required exact namespace"}, {"--schema SCHEMA", "non-empty schema; conflicts with --no-schema"}, {"--no-schema", "store a null schema; conflicts with --schema"}, {"--value-json JSON", "one complete JSON value; conflicts with --value-file"}, {"--value-file PATH_OR_DASH", "read one JSON value from a regular non-symlink file or stdin (-)"}, {"--format human|json", "optional, once; default terminal=human, non-terminal=JSON"}},
		Details: []string{"The value is canonicalized and duplicate object keys are rejected. Every unselected Candidate and Link field is preserved; an identical set is idempotent."},
		Related: []string{"link-metadata remove", "candidate show", "link"},
	},
	"link-metadata remove": {
		Path: "link-metadata remove", Summary: "Remove one existing namespace from one exact Candidate Cause Link.",
		Usage:   []string{"sealgraph link-metadata remove REF --target TARGET --namespace NAMESPACE [--format human|json]"},
		Options: []helpOption{{"--target TARGET", "required exact Cause target selector"}, {"--namespace NAMESPACE", "required exact namespace"}, {"--format human|json", "optional, once; default terminal=human, non-terminal=JSON"}},
		Details: []string{"Removing an absent namespace fails. The Cause Link itself and every unselected field are preserved."},
		Related: []string{"link-metadata set", "candidate show", "unlink"},
	},
	"unlink": {
		Path: "unlink", Summary: "Remove exactly one resolved Cause target from one candidate.",
		Usage: []string{"sealgraph unlink REF --target TARGET"}, Arguments: []string{"REF (required): candidate REF."}, Options: []helpOption{{"--target TARGET", "required exactly once; exact target record to remove"}},
		Details: []string{"A bare target REF resolves its current HEAD and will not match an older stored target. Inspect candidate show and use the displayed @SealID when removing a historical record."}, Examples: []string{"sealgraph unlink design/api --target @abcd"}, Related: []string{"link", "candidate show", "selectors"},
	},
	"tag": {
		Path: "tag", Summary: "List one REF's immutable tags or create one immutable scoped tag binding.",
		Usage:     []string{"sealgraph tag REF", "sealgraph tag REF TAGNAME", "sealgraph tag REF@SEAL_OR_TAG TAGNAME"},
		Arguments: []string{"One argument lists tags. Two arguments create a binding. TAGNAME is non-empty UTF-8; @, controls, DEL, and 4-64 lower-hex names are forbidden."},
		Details:   []string{"A global @SEAL_TOKEN has no REF scope and cannot create a tag. Repeating the same binding is idempotent; retarget, delete, force, and unscoped creation are absent."}, Examples: []string{"sealgraph tag design/api", "sealgraph tag design/api reviewed/1.0", "sealgraph tag design/api@reviewed/1.0 archived"}, Related: []string{"selectors", "show"},
	},
	"mv": {
		Path: "mv", Summary: "Atomically move exactly one REF manifest and its complete tag namespace.", Usage: []string{"sealgraph mv OLD_REF NEW_REF"}, Arguments: []string{"OLD_REF and NEW_REF are required, distinct, explicit REFs; destination must be absent."},
		Details: []string{"This is REF_ONLY and PATH_NOT_MOVED. Candidates or local source bindings at either name block the move. mv does not recurse, move a file/candidate, create an alias, or rewrite Seals or Links."}, Examples: []string{"sealgraph mv design/api archive/design-api"}, Related: []string{"source show", "source unbind", "candidate show", "candidate discard", "tag"},
	},
	"candidate": {
		Path: "candidate", Summary: "Inspect, compare, or explicitly discard mutable candidate state.", Usage: []string{"sealgraph candidate <show|compare|discard> ..."}, Subcommands: []string{"show", "compare", "discard"}, Details: []string{"Candidate operations never rebase, relink, repair, or seal automatically."}, Related: []string{"candidate show", "candidate compare", "candidate discard", "seal"},
	},
	"candidate show": {
		Path: "candidate show", Summary: "Inspect one Candidate, its prospective typed IDs, Cause assertions, and expected REF-head relation.", Usage: []string{"sealgraph candidate show REF [--raw-content] [--format human|json]"}, Arguments: []string{"REF (required): exact candidate REF, not a Seal selector."}, Options: []helpOption{{"--raw-content", "optional; stdout becomes exact content bytes only; conflicts with explicit --format json"}, {"--format human|json", "optional, once; default terminal=human, non-terminal=JSON"}}, Details: []string{"Inspection validates Material, Provenance, prospective Seal, and exact Cause targets and does not mutate or bootstrap a repository."}, Examples: []string{"sealgraph candidate show design/api"}, Related: []string{"candidate compare", "candidate discard", "seal"},
	},
	"candidate compare": {
		Path: "candidate compare", Summary: "Compare one Candidate with its explicit publication baseline.", Usage: []string{"sealgraph candidate compare REF [--format human|json]"}, Arguments: []string{"REF (required): exact candidate REF."}, Options: []helpOption{{"--format human|json", "optional, once; default terminal=human, non-terminal=JSON"}}, Details: []string{"The baseline is Candidate.expected_ref_head, not an inferred revision predecessor. Publication concurrency is reported separately from immutable material and provenance differences."}, Examples: []string{"sealgraph candidate compare design/api"}, Related: []string{"candidate show", "compare", "seal"},
	},
	"candidate discard": {
		Path: "candidate discard", Summary: "Explicitly remove exactly one candidate and no canonical state.", Usage: []string{"sealgraph candidate discard REF"}, Arguments: []string{"REF (required): exact candidate REF."}, Details: []string{"This removes no Seal, object, REF, tag, or descendant candidate. There is no recursive or force form."}, Examples: []string{"sealgraph candidate discard design/api"}, Related: []string{"candidate show", "add"},
	},
	"seal": {
		Path: "seal", Summary: "Publish at most one new immutable Seal for exactly one REF.", Usage: []string{"sealgraph seal REF"}, Arguments: []string{"REF (required): REF whose candidate is reviewed and published."},
		Details: []string{"A normal non-draft candidate requires every direct and reachable Cause target to be a non-draft active revision leaf. Draft may preserve intentional historical provenance. There is no --all, --force, automatic relink, or automatic stale repair."}, Examples: []string{"sealgraph seal premise", "sealgraph candidate compare design/api\nsealgraph seal design/api"}, Related: []string{"candidate show", "candidate compare", "status", "stale", "concepts stale"},
	},
	"recover": {
		Path: "recover", Summary: "Inspect or restore one explicitly selected local REF operation.", Usage: []string{"sealgraph recover show [OPERATION_ID] [--format human|json]", "sealgraph recover OPERATION_ID [--format human|json]"},
		Subcommands: []string{"show"}, Details: []string{"Recovery restores exact prior REF-manifest state only when current state still equals the recorded after-state. It never changes Seal bytes or selects the latest operation implicitly."}, Related: []string{"status", "show", "concepts"},
	},
	"recover show": inspectionHelp("recover show", "List local recovery records or inspect one exact operation ID.", "sealgraph recover show [OPERATION_ID] [--format human|json]", nil),
	"ref": {
		Path: "ref", Summary: "Manage one explicit logical REF handle without deleting immutable graph material.", Usage: []string{"sealgraph ref drop REF"}, Subcommands: []string{"drop"}, Details: []string{"REF operations never delete workfiles, candidates, source bindings, Seals, content, Links, or downstream Seals."}, Related: []string{"ref drop", "recover", "candidate discard", "source unbind"},
	},
	"ref drop": {
		Path: "ref drop", Summary: "Remove exactly one current REF manifest with an explicit recovery receipt.", Usage: []string{"sealgraph ref drop REF"}, Arguments: []string{"REF (required): exact current logical REF; no selector, prefix, or batch form."}, Details: []string{"Candidate or source-binding state blocks the operation. The complete tag namespace leaves the active namespace with the REF. Immutable objects remain valid."}, Examples: []string{"sealgraph ref drop obsolete/spec"}, Related: []string{"recover", "candidate discard", "source unbind", "show"},
	},
	"show":    inspectionHelp("show", "Inspect one immutable Seal generation and its exact material and Cause Links.", "sealgraph show SELECTOR [--raw-content] [--format human|json]", []helpOption{{"--raw-content", "optional; exact content bytes only; conflicts with explicit --format json"}}),
	"log":     inspectionHelp("log", "Follow observed Cause-scoped revision assertions for one current REF.", "sealgraph log [--all-paths] [--max-paths N] REF [--format human|json]", []helpOption{{"--all-paths", "emit bounded complete leaf-terminated revision paths"}, {"--max-paths N", "positive bound; requires --all-paths; default 100"}}),
	"linklog": inspectionHelp("linklog", "Show exact Cause-Link record changes across observed revision assertions.", "sealgraph linklog [--upstream TARGET_SELECTOR] REF [--format human|json]", []helpOption{{"--upstream TARGET_SELECTOR", "optional, once; retain changes for one resolved target Seal"}}),
	"compare": inspectionHelp("compare", "Compare two explicit immutable Seal selections.", "sealgraph compare FROM_SELECTOR TO_SELECTOR [--format human|json]", nil),
	"status":  inspectionHelp("status", "Report separate candidate/HEAD, local workfile/baseline, draft, and stale facts.", "sealgraph status [REF] [--format human|json]", nil),
	"stale": {
		Path: "stale", Summary: "List stale current REF heads or the upstream-first review frontier.", Usage: []string{"sealgraph stale [--frontier] [--refs-only] [--scan] [--format human|json]"},
		Options: []helpOption{{"--frontier", "optional; keep only stale heads not blocked by another stale current head in strict Cause closure"}, {"--refs-only", "optional; stable REF-only line protocol; conflicts with explicit --format json"}, {"--scan", "optional; bypass disposable cache reads"}, {"--format human|json", "optional, once; default terminal=human, non-terminal=JSON"}},
		Details: []string{"Stale is derived current review state. It is not structural impact, candidate state, approval, or an automatic repair plan. --scan does not repair canonical state."}, Examples: []string{"sealgraph stale --frontier", "sealgraph stale --frontier --refs-only --scan"}, Related: []string{"status", "show", "candidate compare", "concepts stale", "impact"},
	},
	"impact": {
		Path: "impact", Summary: "Report downstream Cause reachability through an explicit revision-assertion scope.", Usage: []string{"sealgraph impact [--asserted-by OBSERVER_SELECTOR ...] [--all-paths] [--max-paths N] SELECTOR [--format human|json]"}, Arguments: []string{"SELECTOR (required): REF, @SEAL_TOKEN, or REF@TOKEN."},
		Options: []helpOption{{"--asserted-by OBSERVER_SELECTOR", "optional, repeatable; restrict revision proof edges to these observer Seals"}, {"--all-paths", "optional; emit bounded distinct first-match Cause paths"}, {"--max-paths N", "optional, once; positive per-downstream limit, requires --all-paths; default 100"}, {"--format human|json", "optional, once; default terminal=human, non-terminal=JSON"}},
		Details: []string{"Path truncation never removes impact membership, skips graph validation, or weakens snapshot revalidation. STRUCTURAL_IMPACT is not a stale-only result."}, Examples: []string{"sealgraph impact requirements/api", "sealgraph impact --all-paths --max-paths 20 requirements/api"}, Related: []string{"selectors", "stale", "graph", "concepts structural-impact"},
	},
	"graph": inspectionHelp("graph", "Inspect observed Cause and Cause-scoped revision edges.", "sealgraph graph [--format human|json]", nil),
	"fsck":  inspectionHelp("fsck", "Validate the complete standalone object, REF/tag, material, revision, and Cause inventory without repair.", "sealgraph fsck [--format human|json]", nil),
	"migrate": {
		Path: "migrate", Summary: "Run an explicitly isolated one-way repository migration step.", Usage: []string{"sealgraph migrate extract --source-format 4 --format universal-blob-v1", "sealgraph migrate repository --from 5 --to 6 [--format human|json]"}, Subcommands: []string{"extract", "repository"}, Details: []string{"Each migration command accepts one exact source/target contract. No downgrade, inference, force, or batch path exists."}, Related: []string{"migrate extract", "migrate repository", "load"},
	},
	"migrate extract": {
		Path: "migrate extract", Summary: "Read one retained format-4 source and emit a canonical migration document.", Usage: []string{"sealgraph migrate extract --source-format 4 --format universal-blob-v1 > repository.dump.json"}, Arguments: []string{"No positional arguments; the source is exactly .sealgraph below the current directory."}, Options: []helpOption{{"--source-format 4", "required exactly once; no other source format is accepted"}, {"--format universal-blob-v1", "required exactly once; no other document format is accepted"}}, Details: []string{"This is the only migration command that opens format 4. It has no source mutation operation, never inspects Git, rejects every Candidate or corrupt/unrecognized canonical entry, validates two equal complete source captures, and writes the document only after both captures agree."}, Examples: []string{"sealgraph migrate extract --source-format 4 --format universal-blob-v1 > repository.dump.json"}, Related: []string{"load", "init", "fsck"},
	},
	"migrate repository": {
		Path: "migrate repository", Summary: "Atomically change one validated repository config from format 5 to format 6 without rewriting retained records.",
		Usage:   []string{"sealgraph migrate repository --from 5 --to 6 [--format human|json]"},
		Options: []helpOption{{"--from 5", "required exactly once"}, {"--to 6", "required exactly once"}, {"--format human|json", "optional, once; default terminal=human, non-terminal=JSON"}},
		Details: []string{"The command validates and snapshots format-5 state, atomically replaces only config, then reopens and fscks format 6. If receipt delivery fails after commit, do not rerun migration; use fsck."},
		Related: []string{"fsck", "migrate extract"},
	},
	"load": {
		Path: "load", Summary: "Atomically import one canonical format-4 migration document into an absent format-5 target.", Usage: []string{"sealgraph load --format universal-blob-v1 < repository.dump.json"}, Arguments: []string{"No positional arguments; stdin is the exact canonical document emitted by migrate extract."}, Options: []helpOption{{"--format universal-blob-v1", "required exactly once; no other value is accepted"}}, Details: []string{"First extract in the retained format-4 repository with `sealgraph migrate extract --source-format 4 --format universal-blob-v1 > repository.dump.json`. Load consumes only that document; it never opens format 4, merges, replaces, or repairs an existing target. If receipt stdout delivery alone fails after publication, recover it with load-receipt; never retry load."}, Examples: []string{"sealgraph load --format universal-blob-v1 < repository.dump.json"}, Related: []string{"migrate extract", "load-receipt", "init", "fsck"},
	},
	"load-receipt": {
		Path: "load-receipt", Summary: "Recover one durable migration receipt without modifying repository state.", Usage: []string{"sealgraph load-receipt --source-document-sha256 HEX"}, Arguments: []string{"No positional arguments."}, Options: []helpOption{{"--source-document-sha256 HEX", "required exactly once; 64 lower-case hexadecimal source-document digest"}}, Details: []string{"The command requires the matching regular receipt, validates its exact canonical bytes, runs complete format-5 fsck, and requires the current repository digest to match the receipt before emitting its exact bytes. It never creates, repairs, or republishes state."}, Examples: []string{"sealgraph load-receipt --source-document-sha256 0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"}, Related: []string{"load", "fsck"},
	},
}

func inspectionHelp(path, summary, usage string, extra []helpOption) commandHelp {
	options := append([]helpOption{}, extra...)
	options = append(options, helpOption{"--format human|json", "optional, once; default terminal=human, non-terminal=JSON"})
	return commandHelp{Path: path, Summary: summary, Usage: strings.Split(usage, "\n"), Options: options, Details: []string{"Read-only inspection does not bootstrap, mutate, repair, relink, reseal, or inspect Git. Human output uses aligned columns and abbreviated IDs; JSON keeps full IDs."}, Related: []string{"selectors", "concepts"}}
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
	fmt.Fprint(w, "\nOutput:\n  terminal: aligned human display with abbreviated IDs\n  pipe/file: versioned JSON where supported\n  override: --format human|json\n\nSemantic legend:\n  status separates CANDIDATE_TO_HEAD from WORKFILE_TO_BASELINE.\n  local source binding is not Git tracked membership.\n  REF is a movable logical identity, not a branch or checkout target.\n  STRUCTURAL_IMPACT is Cause reachability; stale is current review state.\n  root marks a provenance boundary, not truth or trust.\n  log/linklog are Seal revision/Cause histories, not Git histories.\n")
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
  REF@hex requires the selected Seal to be current HEAD or reachable through
  observed Cause-scoped revision assertions. Use @SEAL_TOKEN for another Seal.
  REF@non-hex resolves an immutable tag in that REF's namespace.
  A bare hexadecimal Seal token is not accepted because it can be a valid REF.
  There is no @latest shortcut. Bare REF is the explicit current-HEAD form.
  Links and candidates persist only the resolved full 64-character SealID;
  selector spelling and dynamic HEAD references are never persisted.

Examples:
  sealgraph show requirements/api
  sealgraph show @1a2b
  sealgraph show requirements/api@reviewed/1.0
  sealgraph link design/api --target @1a2b --no-previous

Related:
  sealgraph help concepts ref
  sealgraph help concepts seal
  sealgraph help tag
`)
}

var conceptHelp = map[string]string{
	"ref":                "REF is a movable logical lookup/publication name, not immutable Seal identity, a branch, or a checkout target. Each REF has at most one current HEAD.",
	"seal":               "Seal is an immutable typed Blob naming one exact Material and one exact Provenance. REF names, tags, actor, and time are not Seal bytes.",
	"revision-assertion": "A Cause Link may assert zero or more previous revisions of its exact target. The assertion belongs to that observer Link; there is no intrinsic parent on a Seal.",
	"cause":              "A Cause Link records one exact target Seal, its observer-local previous-revision assertion array, and messages. It persists full SealIDs, never dynamic REF HEAD pointers.",
	"root":               "root is an explicit provenance boundary for one generation. It does not mean true, trusted, or approved.",
	"draft":              "draft is explicit provisional sealing. It may preserve historical or non-leaf Causes, remains observable, and does not propagate automatically.",
	"candidate":          "candidate is mutable unsealed state for one destination REF. It is not history or approval and is inspected, changed, discarded, or sealed explicitly.",
	"tag":                "tag is an immutable REF-scoped alias to one exact Seal. It is external to Seal bytes and is not a branch, dynamic Link, or approval claim.",
	"selector":           "selector resolves one immutable Seal using REF, @SEAL_TOKEN, or REF@TOKEN. Run `sealgraph help selectors` for the exact grammar.",
	"stale":              "stale is derived from immutable Seals plus current REF heads. Revision self-stale and direct/transitive Cause stale are review facts, not persisted state or automatic work.",
	"structural-impact":  "STRUCTURAL_IMPACT is downstream Cause reachability from a selected generation or its revision ancestors. It is broader than current stale state.",
	"history":            "Revision history follows the union of observed Cause-scoped assertions and can branch. Cause history compares exact whole Link records across those structural edges. Neither is Git commit or reflog history.",
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
	order := []string{"ref", "seal", "revision-assertion", "cause", "root", "draft", "candidate", "tag", "selector", "stale", "structural-impact", "history"}
	for _, name := range order {
		fmt.Fprintf(w, "%s:\n  %s\n\n", name, conceptHelp[name])
	}
	fmt.Fprint(w, "CLEAN means no candidate and no derived stale relation; it does not compare working files. Standalone SealGraph never searches for Git.\n")
	return true
}

func printUseCasesHelp(w io.Writer) {
	fmt.Fprint(w, `usecases — explicit provenance workflows

Create the first root:
  sealgraph add premise --root --clear-cause-links --content 'External premise'
  sealgraph seal premise

Seal content based on an upstream HEAD:
  sealgraph add design/api --content-file design.md --non-root \
    --target requirements/api --no-previous
  sealgraph seal design/api

Replace one complete Cause record with revision evidence and rationale:
  sealgraph link design/api --target requirements/api \
    --previous @1a2b -m 'API design reviewed this revision relation'

Review a candidate explicitly:
  sealgraph candidate show design/api
  sealgraph candidate compare design/api
  sealgraph seal design/api

Review stale provenance upstream-first:
  sealgraph stale --frontier
  sealgraph status design/api
  sealgraph show requirements/api
  sealgraph candidate compare design/api
  sealgraph seal design/api

Inspect structural impact:
  sealgraph impact requirements/api
  sealgraph impact --all-paths --max-paths 20 requirements/api

Select a historical generation explicitly:
  sealgraph show requirements/api@reviewed/1.0
  sealgraph show @1a2b
  sealgraph link design/api --target @1a2b --no-previous

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
