package cli

// R: expose one explicitly selected immutable or current source with bounded
// quotation context, keeping byte equality separate from provenance judgment.

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/mako10k/sealgraph/internal/repository"
)

const traceReadDefaultBytes = 64 * 1024
const traceReadMaximumBytes = 1024 * 1024

type traceReadOptions struct {
	selection              traceOccurrencesOptions
	contextLines, maxBytes int
	raw                    bool
}

type traceReadDocument struct {
	Schema           string                          `json:"schema"`
	Selection        traceOccurrencesSelectionJSON   `json:"selection"`
	Run              traceOccurrencesRunJSON         `json:"run"`
	Observation      traceOccurrencesObservationJSON `json:"observation"`
	View             string                          `json:"view"`
	State            string                          `json:"state"`
	Reason           *string                         `json:"reason"`
	SourceByteLength *int                            `json:"source_byte_length"`
	SelectedStart    *int                            `json:"selected_start"`
	ExcerptStart     int                             `json:"excerpt_start"`
	ExcerptBytes     []byte                          `json:"excerpt_bytes"`
	Truncated        bool                            `json:"truncated"`
	QuoteTruncated   bool                            `json:"quote_truncated"`
	MaxBytes         int                             `json:"max_bytes"`
	Limitation       string                          `json:"limitation"`
}

func runTraceRead(ctx context.Context, workDir string, args []string, stdout, stderr io.Writer) int {
	options, err := parseTraceReadArgs(args, stdout)
	if err != nil {
		return flagUsageError(stderr, "trace read", err)
	}
	repo, err := repository.OpenStandalone(workDir)
	if err != nil {
		return commandError(stderr, "trace read", err)
	}
	if repo.Format() != 7 {
		return commandError(stderr, "trace read", fmt.Errorf("source reading requires repository format 7; inspect repository configuration and explicitly migrate if appropriate"))
	}
	baseline, err := options.selection.selectedBaseline(ctx, repo)
	if err != nil {
		return commandError(stderr, "trace read", err)
	}
	input, err := repo.LoadTraceOccurrenceInput(ctx, baseline, options.selection.runIndex, options.selection.view.value == "current")
	if err != nil {
		return commandError(stderr, "trace read", err)
	}
	doc, source := buildTraceRead(input, options)
	if err = repo.RevalidateTraceOccurrenceInput(ctx, input); err != nil {
		return commandError(stderr, "trace read", err)
	}
	if options.raw {
		if input.CurrentFailure != "" {
			return commandError(stderr, "trace read", fmt.Errorf("current source is unconfirmed: %s", input.CurrentFailure))
		}
		return writeOperatorOutput(stdout, stderr, "trace read", source)
	}
	if options.selection.output.JSON {
		return writeInspectionJSON(stdout, stderr, "trace read", doc)
	}
	var rendered bytes.Buffer
	printTraceReadHuman(&rendered, doc)
	return writeOperatorOutput(stdout, stderr, "trace read", rendered.Bytes())
}

// Read-specific flags are removed before the shared exact-baseline parser is
// used, so selection and version-bound read behavior have one implementation.
func parseTraceReadArgs(args []string, stdout io.Writer) (traceReadOptions, error) {
	options := traceReadOptions{contextLines: 3, maxBytes: traceReadDefaultBytes}
	var contextValue, maxValue singleString
	var raw singleBool
	var rest []string
	for i := 0; i < len(args); i++ {
		name, value, equal := strings.Cut(args[i], "=")
		switch name {
		case "--raw-source":
			if !equal {
				value = "true"
			}
			if err := raw.Set(value); err != nil {
				return options, err
			}
		case "--context-lines", "--max-bytes":
			if !equal {
				i++
				if i == len(args) {
					return options, fmt.Errorf("%s requires an integer", name)
				}
				value = args[i]
			}
			target := &contextValue
			if name == "--max-bytes" {
				target = &maxValue
			}
			if err := target.Set(value); err != nil {
				return options, err
			}
		case "--limit", "--cursor":
			return options, fmt.Errorf("%s is not a trace read option", name)
		default:
			rest = append(rest, args[i])
		}
	}
	selection, err := parseTraceOccurrencesArgs(rest, stdout)
	if err != nil {
		return options, err
	}
	if selection.view.value == "both" {
		return options, fmt.Errorf("--view requires snapshot or current")
	}
	options.selection, options.raw = selection, raw.value
	if options.raw && selection.output.Explicit && selection.output.JSON {
		return options, fmt.Errorf("--raw-source and --format json are mutually exclusive")
	}
	options.contextLines, err = traceReadInteger(contextValue, 3, 0, 1000, "--context-lines")
	if err != nil {
		return options, err
	}
	options.maxBytes, err = traceReadInteger(maxValue, traceReadDefaultBytes, 1, traceReadMaximumBytes, "--max-bytes")
	return options, err
}

func traceReadInteger(value singleString, fallback, minimum, maximum int, name string) (int, error) {
	if !value.set {
		return fallback, nil
	}
	result, err := strconv.Atoi(value.value)
	if err != nil || result < minimum || result > maximum {
		return 0, fmt.Errorf("%s requires an integer from %d through %d", name, minimum, maximum)
	}
	return result, nil
}

func buildTraceRead(input repository.TraceOccurrenceInput, options traceReadOptions) (traceReadDocument, []byte) {
	metadata, _ := newTraceOccurrencesDocument(input, options.selection)
	doc := traceReadDocument{Schema: "sealgraph/trace-read/v1", Selection: metadata.Selection, Run: metadata.Run, Observation: metadata.Observation, View: options.selection.view.value, State: "SAVED_RANGE", MaxBytes: options.maxBytes, ExcerptBytes: []byte{}, Limitation: "A current match is one representative byte occurrence, not proof of historical identity, movement, semantic validity or approval. Excerpt bytes are base64 in JSON. Use --raw-source for the complete selected source."}
	if input.Baseline.Kind == repository.TraceOwnHead {
		doc.Selection.Baseline.Kind = "head"
	}
	source := input.SnapshotBytes
	start := int(input.SourceStart)
	if doc.View == "current" {
		source = input.CurrentBytes
		doc.State = "PRESENT"
		if input.CurrentFailure != "" {
			doc.State = "INCOMPLETE"
			reason := input.CurrentFailure
			if input.BindingDigest == nil {
				reason = "SOURCE_UNBOUND"
			}
			doc.Reason = &reason
			return doc, nil
		}
		start = bytes.Index(source, input.Pattern)
		if start < 0 {
			doc.State = "ABSENT_EXACT"
		}
	}
	length := len(source)
	doc.SourceByteLength = &length
	if start < 0 {
		doc.Truncated = len(source) != 0
		return doc, source
	}
	doc.SelectedStart = &start
	begin, end := traceReadContext(source, start, len(input.Pattern), options.contextLines)
	if start-begin > options.maxBytes/3 {
		begin = start - options.maxBytes/3
	}
	displayedEnd := end
	if displayedEnd-begin > options.maxBytes {
		displayedEnd = begin + options.maxBytes
	}
	doc.ExcerptStart = begin
	doc.ExcerptBytes = source[begin:displayedEnd]
	doc.Truncated = begin > 0 || displayedEnd < len(source)
	doc.QuoteTruncated = start+len(input.Pattern) > displayedEnd
	return doc, source
}

func traceReadContext(source []byte, start, length, lines int) (int, int) {
	begin, end := start, start+length
	breaks := 0
	for begin > 0 {
		if source[begin-1] == '\n' {
			if breaks >= lines {
				break
			}
			breaks++
		}
		begin--
	}
	breaks = 0
	for end < len(source) {
		c := source[end]
		end++
		if c == '\n' {
			if breaks >= lines {
				break
			}
			breaks++
		}
	}
	return begin, end
}

func printTraceReadHuman(out io.Writer, doc traceReadDocument) {
	fmt.Fprintf(out, "SOURCE CONTEXT %s state=%s\n", doc.View, doc.State)
	fmt.Fprintf(out, "Snapshot %s source key=%s saved blob=%s\n", doc.Run.SourceSnapshotID, quoteHumanString(doc.Run.SourceKey), doc.Observation.SnapshotBlobID)
	fmt.Fprintf(out, "Baseline %s; run=%d saved start=%d length=%d\n", doc.Selection.Baseline.Kind, doc.Run.RunIndex, doc.Run.SourceStart, doc.Run.Length)
	if doc.Selection.Baseline.SealID != nil {
		fmt.Fprintf(out, "Seal %s\n", *doc.Selection.Baseline.SealID)
	}
	if doc.Selection.Baseline.CandidateDigest != nil {
		fmt.Fprintf(out, "Candidate digest %s\n", *doc.Selection.Baseline.CandidateDigest)
	}
	if doc.Observation.CurrentBlobID != nil {
		fmt.Fprintf(out, "Current blob %s\n", *doc.Observation.CurrentBlobID)
	}
	if doc.Reason != nil {
		fmt.Fprintf(out, "Unconfirmed: %s\n", *doc.Reason)
	}
	if doc.SelectedStart != nil {
		fmt.Fprintf(out, "Selected byte start=%d; excerpt start=%d bytes=%d; source bytes=%d\n", *doc.SelectedStart, doc.ExcerptStart, len(doc.ExcerptBytes), *doc.SourceByteLength)
		fmt.Fprintf(out, "Excerpt (escaped): %s\n", quoteHumanBytes(doc.ExcerptBytes))
	}
	fmt.Fprintf(out, "Truncated=%t quote_truncated=%t max_bytes=%d\n%s\n", doc.Truncated, doc.QuoteTruncated, doc.MaxBytes, doc.Limitation)
}
