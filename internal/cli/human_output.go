package cli

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/mako10k/sealgraph/internal/domain"
	domainv5 "github.com/mako10k/sealgraph/internal/domain/v5"
	"github.com/mako10k/sealgraph/internal/repository"
)

const (
	defaultHumanWidth = 100
	minimumHumanWidth = 24
	maximumHumanWidth = 120
	shortIDLength     = 12
)

type terminalSizeProvider interface {
	TerminalSize() (width int, height int)
}

type humanField struct {
	label string
	value string
}

type humanTag struct {
	name string
	seal domain.ObjectID
}

type humanPrinter struct {
	output io.Writer
	width  int
}

func newHumanPrinter(output io.Writer) humanPrinter {
	_, width, _ := outputTerminalInfo(output)
	if width <= 0 {
		width = defaultHumanWidth
	}
	width = min(max(width, minimumHumanWidth), maximumHumanWidth)
	return humanPrinter{output: output, width: width}
}

func outputTerminalInfo(output io.Writer) (terminal bool, width int, known bool) {
	if provider, ok := output.(terminalSizeProvider); ok {
		width, _ = provider.TerminalSize()
		return width > 0, width, true
	}
	file, ok := output.(*os.File)
	if !ok {
		return false, defaultHumanWidth, false
	}
	info, err := file.Stat()
	if err != nil || info.Mode()&os.ModeCharDevice == 0 {
		return false, defaultHumanWidth, true
	}
	width = terminalColumns(file.Fd())
	return width > 0, width, true
}

func isHumanTerminal(output io.Writer) bool {
	terminal, _, known := outputTerminalInfo(output)
	return known && terminal
}

func printHumanReceipt(output io.Writer, title string, fields ...humanField) {
	printer := newHumanPrinter(output)
	printer.heading(title)
	printer.fields(0, fields...)
}

func printInitHuman(output io.Writer, result repository.InitResult) {
	switch result.Outcome {
	case repository.InitInitialized:
		printHumanReceipt(output, "REPOSITORY INITIALIZED",
			humanField{"Mode", "standalone"},
			humanField{"Runtime directories", strings.Join(result.RuntimeDirectories, ", ")},
		)
	case repository.InitRuntimeBootstrapped:
		printHumanReceipt(output, "RUNTIME BOOTSTRAPPED",
			humanField{"Created directories", strings.Join(result.RuntimeDirectories, ", ")},
			humanField{"Canonical state", "unchanged"},
		)
	case repository.InitAlreadyComplete:
		printHumanReceipt(output, "REPOSITORY READY", humanField{"Changes", "none"})
	}
}

func (printer humanPrinter) heading(title string) {
	fmt.Fprintln(printer.output, title)
}

func (printer humanPrinter) note(value string) {
	fmt.Fprintf(printer.output, "  %s\n", clipDisplay(value, printer.width-2))
}

func (printer humanPrinter) fields(indent int, fields ...humanField) {
	labelWidth := 0
	for _, field := range fields {
		labelWidth = max(labelWidth, displayWidth(field.label))
	}
	prefix := strings.Repeat(" ", indent)
	if indent+labelWidth+10 > printer.width {
		for _, field := range fields {
			fmt.Fprintf(printer.output, "%s%s\n", prefix, clipDisplay(field.label, printer.width-indent))
			fmt.Fprintf(printer.output, "%s  %s\n", prefix, clipDisplay(field.value, printer.width-indent-2))
		}
		return
	}
	valueWidth := max(8, printer.width-indent-labelWidth-2)
	for _, field := range fields {
		fmt.Fprintf(printer.output, "%s%s  %s\n", prefix, padDisplay(field.label, labelWidth), clipDisplay(field.value, valueWidth))
	}
}

func (printer humanPrinter) table(indent int, headers []string, rows [][]string) {
	if len(headers) == 0 {
		return
	}
	widths := tableWidths(headers, rows, max(20, printer.width-indent))
	prefix := strings.Repeat(" ", indent)
	writeHumanRow(printer.output, prefix, headers, widths)
	separator := make([]string, len(headers))
	for index, width := range widths {
		separator[index] = strings.Repeat("-", width)
	}
	writeHumanRow(printer.output, prefix, separator, widths)
	for _, row := range rows {
		writeHumanRow(printer.output, prefix, row, widths)
	}
}

func tableWidths(headers []string, rows [][]string, available int) []int {
	widths := make([]int, len(headers))
	minimums := make([]int, len(headers))
	for index, header := range headers {
		widths[index] = displayWidth(header)
		minimums[index] = min(widths[index], 2)
	}
	for _, row := range rows {
		for index := range headers {
			if index < len(row) {
				widths[index] = max(widths[index], displayWidth(row[index]))
			}
		}
	}
	contentWidth := max(len(headers)*4, available-2*(len(headers)-1))
	for sumInts(widths) > contentWidth {
		candidate := -1
		for index := range widths {
			if widths[index] > minimums[index] && (candidate < 0 || widths[index] > widths[candidate]) {
				candidate = index
			}
		}
		if candidate < 0 {
			break
		}
		widths[candidate]--
	}
	return widths
}

func writeHumanRow(output io.Writer, prefix string, cells []string, widths []int) {
	fmt.Fprint(output, prefix)
	last := len(widths) - 1
	for last > 0 && (last >= len(cells) || cells[last] == "") {
		last--
	}
	for index, width := range widths[:last+1] {
		if index != 0 {
			fmt.Fprint(output, "  ")
		}
		cell := ""
		if index < len(cells) {
			cell = cells[index]
		}
		cell = clipDisplay(cell, width)
		if index == last {
			fmt.Fprint(output, cell)
		} else {
			fmt.Fprint(output, padDisplay(cell, width))
		}
	}
	fmt.Fprintln(output)
}

func sumInts(values []int) int {
	total := 0
	for _, value := range values {
		total += value
	}
	return total
}

func displayWidth(value string) int {
	width := 0
	for _, current := range value {
		width += runeDisplayWidth(current)
	}
	return width
}

func runeDisplayWidth(value rune) int {
	if value == 0 || unicode.Is(unicode.Mn, value) || unicode.Is(unicode.Me, value) || unicode.Is(unicode.Cf, value) {
		return 0
	}
	if value < 0x20 || (value >= 0x7f && value < 0xa0) {
		return 0
	}
	if isWideRune(value) {
		return 2
	}
	return 1
}

var wideRuneRanges = [...][2]rune{
	{0x1100, 0x115f}, {0x2329, 0x232a}, {0x2e80, 0xa4cf},
	{0xac00, 0xd7a3}, {0xf900, 0xfaff}, {0xfe10, 0xfe19},
	{0xfe30, 0xfe6f}, {0xff00, 0xff60}, {0xffe0, 0xffe6},
	{0x1f300, 0x1faff}, {0x20000, 0x3fffd},
}

func isWideRune(value rune) bool {
	if value == 0x303f {
		return false
	}
	for _, interval := range wideRuneRanges {
		if value >= interval[0] && value <= interval[1] {
			return true
		}
	}
	return false
}

func clipDisplay(value string, width int) string {
	if width <= 0 {
		return ""
	}
	if displayWidth(value) <= width {
		return value
	}
	if width >= 3 && strings.HasPrefix(value, `"`) && strings.HasSuffix(value, `"`) {
		return `"` + clipDisplay(strings.TrimSuffix(strings.TrimPrefix(value, `"`), `"`), width-2) + `"`
	}
	if width == 1 {
		return "…"
	}
	var builder strings.Builder
	used := 0
	for len(value) > 0 {
		current, size := utf8.DecodeRuneInString(value)
		currentWidth := runeDisplayWidth(current)
		if used+currentWidth > width-1 {
			break
		}
		builder.WriteString(value[:size])
		value = value[size:]
		used += currentWidth
	}
	builder.WriteRune('…')
	return builder.String()
}

func padDisplay(value string, width int) string {
	return value + strings.Repeat(" ", max(0, width-displayWidth(value)))
}

func shortID(value fmt.Stringer) string {
	return shortTextID(value.String())
}

func shortTextID(text string) string {
	if len(text) <= shortIDLength {
		return text
	}
	return text[:shortIDLength]
}

func shortOptionalID(value *domain.ObjectID) string {
	if value == nil {
		return "none"
	}
	return shortID(*value)
}

func yesNo(value bool) string {
	if value {
		return "yes"
	}
	return "no"
}

func humanList(values []string) string {
	if len(values) == 0 {
		return "none"
	}
	return strings.Join(values, ", ")
}

func humanStatusLabels(status repository.RefStatus) string {
	labels := make([]string, 0, 5)
	if status.Unsealed {
		labels = append(labels, "unsealed changes")
	}
	if status.Draft {
		labels = append(labels, "draft")
	}
	if status.StaleSelf {
		labels = append(labels, "stale revision")
	}
	if len(status.StaleDirect) != 0 {
		labels = append(labels, "stale direct cause")
	}
	if len(status.StaleTransitive) != 0 {
		labels = append(labels, "stale transitive cause")
	}
	if len(labels) == 0 {
		return "clean"
	}
	return strings.Join(labels, ", ")
}

func humanRelation(value string) string {
	replacer := strings.NewReplacer("WORKFILE_", "", "SOURCE_", "source ", "_", " ")
	return strings.ToLower(replacer.Replace(value))
}

func printShowHuman(output io.Writer, result repository.ShowResult, format int) {
	printer := newHumanPrinter(output)
	printer.heading("SEAL")
	printer.fields(0,
		humanField{"Seal ID (prefix)", shortID(result.Resolved.ID)},
		humanField{"Material ID (prefix)", shortID(result.Resolved.Seal.Material)},
		humanField{"Provenance ID (prefix)", shortID(result.Resolved.Seal.Provenance)},
		humanField{"Current REF(s)", humanList(result.REFNames)},
		humanField{"Observed previous state", strings.Join(result.Revision.PreviousStates, ", ")},
		humanField{"Root boundary", yesNo(result.Resolved.Provenance.Root)},
		humanField{"Draft", yesNo(result.Resolved.Provenance.Draft)},
	)
	if format == 6 {
		printer.fields(0, humanField{"Seal schema", result.Resolved.Seal.Schema}, humanField{"Provenance schema", result.Resolved.Provenance.Schema})
	}
	fmt.Fprintln(output)
	printHumanContent(printer, result.Resolved.Material.Content, result.Content)
	printHumanAttachments(printer, result.Resolved.Material.Attachments)
	metadataOwner := ""
	if format == 6 {
		metadataOwner = result.Resolved.Provenance.Schema
	}
	printHumanLinks(printer, result.Resolved.Provenance.CauseLinks, metadataOwner)
}

func printHumanContent(printer humanPrinter, id domain.ObjectID, content []byte) {
	preview := content
	truncated := len(content) > contentPreviewLimit
	if truncated {
		preview = content[:contentPreviewLimit]
	}
	label := "Preview (escaped)"
	if truncated {
		label = "Preview (first 256 bytes)"
	}
	printer.heading("CONTENT")
	printer.fields(2,
		humanField{"Blob ID (prefix)", shortID(id)},
		humanField{"Size", fmt.Sprintf("%d bytes", len(content))},
		humanField{label, quoteHumanBytes(preview)},
	)
}

func printHumanAttachments(printer humanPrinter, attachments []domainv5.Attachment) {
	fmt.Fprintln(printer.output)
	printer.heading(fmt.Sprintf("ATTACHMENTS (%d)", len(attachments)))
	if len(attachments) == 0 {
		printer.note("None")
		return
	}
	rows := make([][]string, 0, len(attachments))
	for _, attachment := range attachments {
		rows = append(rows, []string{quoteHumanString(attachment.Name), quoteHumanString(attachment.MediaType), shortID(attachment.Blob)})
	}
	printer.table(2, []string{"NAME", "MEDIA TYPE", "BLOB ID (PREFIX)"}, rows)
}

func printHumanLinks(printer humanPrinter, links []domainv5.CauseLink, metadataOwner string) {
	fmt.Fprintln(printer.output)
	printer.heading(fmt.Sprintf("CAUSES (%d)", len(links)))
	if len(links) == 0 {
		printer.note("None")
		return
	}
	if metadataOwner != "" {
		printHumanLinksWithMetadata(printer, links, metadataOwner)
		return
	}
	rows := make([][]string, 0, len(links))
	for _, link := range links {
		rows = append(rows, []string{shortID(link.TargetSeal), fmt.Sprintf("%d", len(link.PreviousRevisionSealOfTargetSeal)), fmt.Sprintf("%d", len(link.Messages))})
	}
	printer.table(2, []string{"TARGET SEAL", "PREVIOUS", "MESSAGES"}, rows)
}

func printHumanLinksWithMetadata(printer humanPrinter, links []domainv5.CauseLink, ownerSchema string) {
	for index, link := range links {
		fmt.Fprintf(printer.output, "  Cause %d\n", index+1)
		printer.fields(4,
			humanField{"Target Seal ID (prefix)", shortID(link.TargetSeal)},
			humanField{"Previous revisions", humanList(idsJSON(link.PreviousRevisionSealOfTargetSeal))},
			humanField{"Messages", humanList(link.Messages)},
		)
		if len(link.Metadata) == 0 {
			label := "none"
			if ownerSchema == domainv5.ProvenanceSchema || ownerSchema == domainv5.CandidateSchema {
				label = "projected empty from v5"
			}
			fmt.Fprintf(printer.output, "    Metadata: %s\n", label)
			continue
		}
		fmt.Fprintf(printer.output, "    Metadata (%d)\n", len(link.Metadata))
		for _, entry := range link.Metadata {
			schema := "none"
			if entry.Schema != nil {
				schema = quoteHumanString(*entry.Schema)
			}
			fmt.Fprintf(printer.output, "      Namespace: %s\n      Schema: %s\n      Value: %s\n", quoteHumanString(entry.Namespace), schema, entry.Value)
		}
	}
}

func printCandidateInspectionHuman(output io.Writer, inspection repository.CandidateInspection, format int) {
	printer := newHumanPrinter(output)
	candidate := inspection.Candidate
	printer.heading("CANDIDATE")
	printer.fields(0,
		humanField{"REF", candidate.REF},
		humanField{"Expected REF head", shortOptionalID(candidate.ExpectedREFHead)},
		humanField{"Current REF head", shortOptionalID(inspection.CurrentHead)},
		humanField{"Prospective Seal", shortID(inspection.Prospective.ID)},
		humanField{"Publication check", strings.ToLower(strings.ReplaceAll(string(inspection.ExpectedHeadState), "_", " "))},
		humanField{"Root boundary", yesNo(candidate.Root)},
		humanField{"Draft", yesNo(candidate.Draft)},
	)
	if format == 6 {
		printer.fields(0, humanField{"Candidate schema", candidate.Schema}, humanField{"Prospective Seal schema", inspection.Prospective.Seal.Schema}, humanField{"Prospective provenance schema", inspection.Prospective.Provenance.Schema})
	}
	fmt.Fprintln(output)
	printHumanContent(printer, candidate.Content, inspection.Content)
	printHumanAttachments(printer, candidate.Attachments)
	metadataOwner := ""
	if format == 6 {
		metadataOwner = candidate.Schema
	}
	printHumanLinks(printer, candidate.CauseLinks, metadataOwner)
}

func printSourcesHuman(output io.Writer, bindings []repository.SourceBinding) {
	printer := newHumanPrinter(output)
	printer.heading("LOCAL SOURCES")
	if len(bindings) == 0 {
		printer.note("No local source bindings.")
		return
	}
	rows := make([][]string, 0, len(bindings))
	for _, binding := range bindings {
		rows = append(rows, []string{binding.REF, quoteHumanString(binding.Path)})
	}
	printer.table(0, []string{"REF", "SOURCE FILE"}, rows)
}

func printTagsHuman(output io.Writer, ref string, tags []humanTag) {
	printer := newHumanPrinter(output)
	printer.heading("IMMUTABLE TAGS")
	printer.fields(0, humanField{"REF", ref})
	if len(tags) == 0 {
		printer.note("No tags.")
		return
	}
	rows := make([][]string, 0, len(tags))
	for _, tag := range tags {
		rows = append(rows, []string{quoteHumanString(tag.name), shortID(tag.seal)})
	}
	printer.table(0, []string{"TAG NAME", "SEAL ID (PREFIX)"}, rows)
}

func printSourceCompareHuman(output io.Writer, result repository.SourceCompareResult) {
	printer := newHumanPrinter(output)
	baselineID := "none"
	if result.BaselineContent != nil {
		baselineID = shortID(result.BaselineContent.ID)
	}
	printer.heading("SOURCE COMPARISON")
	printer.fields(0,
		humanField{"REF", result.REF},
		humanField{"Source file", quoteHumanString(result.Path)},
		humanField{"Compared with", strings.ToLower(result.Baseline)},
		humanField{"Result", humanRelation(result.Relation)},
		humanField{"Baseline blob", baselineID},
		humanField{"Workfile blob", shortID(result.WorkfileID)},
		humanField{"Workfile size", fmt.Sprintf("%d bytes", result.WorkfileBytes)},
	)
}

func printStatusesHuman(output io.Writer, title string, statuses []repository.RefStatus) {
	printer := newHumanPrinter(output)
	printer.heading(title)
	if len(statuses) == 0 {
		printer.note("No matching REF, candidate, or local source state.")
		return
	}
	rows := make([][]string, 0, len(statuses))
	for _, status := range statuses {
		candidate := "none"
		if status.Unsealed {
			candidate = "unsealed"
		}
		rows = append(rows, []string{status.REF, candidate, shortOptionalID(status.Head), humanStatusLabels(status)})
	}
	printer.table(0, []string{"REF", "CANDIDATE", "SEAL ID (PREFIX)", "SEALED STATE"}, rows)
	sourceRows := make([][]string, 0, len(statuses))
	for _, status := range statuses {
		if status.Source != nil {
			sourceRows = append(sourceRows, []string{status.REF, strings.ToLower(status.Source.Baseline), humanRelation(status.Source.Relation), quoteHumanString(status.Source.Path)})
		}
	}
	if len(sourceRows) != 0 {
		fmt.Fprintln(output)
		printer.heading("LOCAL WORKFILES")
		printer.table(0, []string{"REF", "BASELINE", "RESULT", "SOURCE FILE"}, sourceRows)
	}
}

func printGraphHuman(output io.Writer, nodes []repository.GraphNode, format int) {
	printer := newHumanPrinter(output)
	printer.heading("REVISION_CAUSE_GRAPH")
	if len(nodes) == 0 {
		printer.note("No active revision or Cause nodes.")
		return
	}
	rows := make([][]string, 0, len(nodes)*2)
	for _, node := range nodes {
		row := []string{"Seal", shortID(node.Resolved.ID), humanGraphState(string(node.State)), humanList(node.REFs)}
		if format == 6 {
			row = append(row, node.Resolved.Seal.Schema+" / "+node.Resolved.Provenance.Schema)
		}
		rows = append(rows, row)
		for _, link := range node.Causes {
			rows = append(rows, []string{"  Cause", shortID(link.Target), humanGraphState(string(link.State)), ""})
		}
	}
	headings := []string{"RELATION", "SEAL ID (PREFIX)", "STATE", "CURRENT REF(S)"}
	if format == 6 {
		headings = append(headings, "SCHEMAS")
	}
	printer.table(0, headings, rows)
}

func humanGraphState(value string) string {
	return strings.ToLower(strings.ReplaceAll(value, "_", " "))
}

func printImpactsHuman(output io.Writer, result repository.ImpactResult) {
	printer := newHumanPrinter(output)
	printer.heading("STRUCTURAL_IMPACT")
	printer.fields(0, humanField{"Source Seal ID (prefix)", shortID(result.Source)}, humanField{"Revision assertions", strings.ToLower(strings.ReplaceAll(result.AssertionScope, "_", " "))})
	if len(result.Impacts) == 0 {
		printer.note("No current downstream Seals are impacted.")
		return
	}
	rows := make([][]string, 0, len(result.Impacts))
	for _, impact := range result.Impacts {
		rows = append(rows, []string{shortID(impact.Head), humanList(impact.REFs), strconv.Itoa(len(impact.Paths))})
	}
	fmt.Fprintln(output)
	printer.table(0, []string{"DOWNSTREAM SEAL", "CURRENT REF(S)", "PATHS"}, rows)
	for _, impact := range result.Impacts {
		fmt.Fprintf(output, "\n  %s\n", shortID(impact.Head))
		for index, path := range impact.Paths {
			fmt.Fprintf(output, "    Path %d\n", index+1)
			for depth, id := range path.CauseSealIDs {
				marker := "starts at"
				if depth != 0 {
					marker = "causes"
				}
				fmt.Fprintf(output, "%s%s  %s\n", strings.Repeat("  ", depth+3), marker, shortID(id))
			}
		}
		if impact.Truncated {
			fmt.Fprintf(output, "    Additional paths omitted (maximum %d).\n", result.MaxPaths)
		}
	}
}

func printLogHuman(output io.Writer, result repository.LogResult, format int) {
	printer := newHumanPrinter(output)
	printer.heading("REVISION HISTORY")
	printer.fields(0, humanField{"REF", result.REF}, humanField{"Revisions", strconv.Itoa(len(result.Entries))})
	for index, entry := range result.Entries {
		label := fmt.Sprintf("Revision %d", index+1)
		if index == 0 {
			label += " (current)"
		}
		fmt.Fprintf(output, "\n%s\n", label)
		printer.fields(2,
			humanField{"Seal ID (prefix)", shortID(entry.Resolved.ID)},
			humanField{"Minimum depth", strconv.Itoa(entry.MinimumDepth)},
			humanField{"Content blob", shortID(entry.Resolved.Material.Content)},
			humanField{"Previous edges", strconv.Itoa(len(entry.OutgoingEdges))},
			humanField{"Root boundary", yesNo(entry.Resolved.Provenance.Root)},
			humanField{"Draft", yesNo(entry.Resolved.Provenance.Draft)},
		)
		if format == 6 {
			printer.fields(2, humanField{"Seal schema", entry.Resolved.Seal.Schema}, humanField{"Provenance schema", entry.Resolved.Provenance.Schema})
		}
		metadataOwner := ""
		if format == 6 {
			metadataOwner = entry.Resolved.Provenance.Schema
		}
		printHumanLinksIndented(printer, entry.Resolved.Provenance.CauseLinks, 2, metadataOwner)
	}
}

func printHumanLinksIndented(printer humanPrinter, links []domainv5.CauseLink, indent int, metadataOwner string) {
	fmt.Fprintf(printer.output, "%sCauses (%d)\n", strings.Repeat(" ", indent), len(links))
	if len(links) == 0 {
		fmt.Fprintf(printer.output, "%sNone\n", strings.Repeat(" ", indent+2))
		return
	}
	if metadataOwner != "" {
		printHumanLinksWithMetadata(printer, links, metadataOwner)
		return
	}
	rows := make([][]string, 0, len(links))
	for _, link := range links {
		rows = append(rows, []string{shortID(link.TargetSeal), strconv.Itoa(len(link.PreviousRevisionSealOfTargetSeal)), strconv.Itoa(len(link.Messages))})
	}
	printer.table(indent+2, []string{"TARGET SEAL", "PREVIOUS", "MESSAGES"}, rows)
}

func printLinkLogHuman(output io.Writer, result repository.LinkLogResult, format int) {
	printer := newHumanPrinter(output)
	printer.heading("CAUSE LINK HISTORY")
	fields := []humanField{{"REF", result.REF}, {"Revision edges", strconv.Itoa(len(result.Entries))}}
	if result.Upstream != nil {
		fields = append(fields, humanField{"Upstream Seal ID (prefix)", shortID(*result.Upstream)})
	}
	printer.fields(0, fields...)
	for index, entry := range result.Entries {
		fmt.Fprintf(output, "\nRevision edge %d\n", index+1)
		printer.fields(2,
			humanField{"Newer Seal", shortID(entry.Newer)},
			humanField{"Previous Seal", shortID(entry.Previous)},
			humanField{"Supporting observers", strconv.Itoa(len(entry.SupportingAssertions))},
			humanField{"Cause changes", strconv.Itoa(len(entry.Changes))},
		)
		if format == 6 {
			printer.fields(2, humanField{"Newer schemas", entry.NewerSealSchema + " / " + entry.NewerProvenanceSchema}, humanField{"Previous schemas", entry.PreviousSealSchema + " / " + entry.PreviousProvenanceSchema})
			records := make([]causeLinkChangeJSONV3, 0, len(entry.Changes))
			for _, change := range entry.Changes {
				records = append(records, causeLinkChangeRecordV3(change.Target.String(), change.Before, change.After))
			}
			printHumanCauseChanges(printer, records)
		}
	}
}

func printSealDiffHuman(output io.Writer, diff repository.SealComparison, format int) {
	printer := newHumanPrinter(output)
	printer.heading("SEAL COMPARISON")
	printer.fields(0, humanField{"From Seal ID (prefix)", shortID(diff.From.ID)}, humanField{"To Seal ID (prefix)", shortID(diff.To.ID)})
	rows := [][]string{
		humanValueChangeRow("Material", !diff.From.Seal.Material.Equal(diff.To.Seal.Material), shortID(diff.From.Seal.Material), shortID(diff.To.Seal.Material)),
		humanValueChangeRow("Provenance", !diff.From.Seal.Provenance.Equal(diff.To.Seal.Provenance), shortID(diff.From.Seal.Provenance), shortID(diff.To.Seal.Provenance)),
		humanValueChangeRow("Content blob", !diff.From.Material.Content.Equal(diff.To.Material.Content), shortID(diff.From.Material.Content), shortID(diff.To.Material.Content)),
		humanValueChangeRow("Root boundary", diff.From.Provenance.Root != diff.To.Provenance.Root, yesNo(diff.From.Provenance.Root), yesNo(diff.To.Provenance.Root)),
		humanValueChangeRow("Draft", diff.From.Provenance.Draft != diff.To.Provenance.Draft, yesNo(diff.From.Provenance.Draft), yesNo(diff.To.Provenance.Draft)),
	}
	if format == 6 {
		rows = append(rows, humanValueChangeRow("Seal schema", diff.From.Seal.Schema != diff.To.Seal.Schema, diff.From.Seal.Schema, diff.To.Seal.Schema), humanValueChangeRow("Provenance schema", diff.From.Provenance.Schema != diff.To.Provenance.Schema, diff.From.Provenance.Schema, diff.To.Provenance.Schema))
	}
	fmt.Fprintln(output)
	printer.table(0, []string{"FIELD", "RESULT", "BEFORE", "AFTER"}, rows)
	if format == 6 {
		printHumanCauseChanges(printer, causeLinkChangesV3(diff.From.Provenance.CauseLinks, diff.To.Provenance.CauseLinks).Records)
	}
}

func humanValueChangeRow(field string, changed bool, before, after string) []string {
	result := "unchanged"
	if changed {
		result = "changed"
	}
	return []string{field, result, before, after}
}

func printCandidateDiffHuman(output io.Writer, result repository.CandidateDiffResult, format int) {
	printer := newHumanPrinter(output)
	inspection := result.Inspection
	candidate := inspection.Candidate
	printer.heading("CANDIDATE COMPARISON")
	printer.fields(0,
		humanField{"REF", candidate.REF},
		humanField{"Compared with", shortOptionalID(candidate.ExpectedREFHead)},
		humanField{"Expected REF head", shortOptionalID(candidate.ExpectedREFHead)},
		humanField{"Current REF head", shortOptionalID(inspection.CurrentHead)},
		humanField{"Publication check", strings.ToLower(strings.ReplaceAll(string(inspection.ExpectedHeadState), "_", " "))},
	)
	if format == 6 {
		printer.fields(0, humanField{"Candidate schema", candidate.Schema}, humanField{"Prospective Seal schema", inspection.Prospective.Seal.Schema}, humanField{"Prospective provenance schema", inspection.Prospective.Provenance.Schema})
	}
	contentBefore, materialBefore, provenanceBefore := "none", "none", "none"
	rootBefore, draftBefore := "none", "none"
	if result.Baseline != nil {
		contentBefore = shortID(result.Baseline.Material.Content)
		materialBefore = shortID(result.Baseline.Seal.Material)
		provenanceBefore = shortID(result.Baseline.Seal.Provenance)
		rootBefore = yesNo(result.Baseline.Provenance.Root)
		draftBefore = yesNo(result.Baseline.Provenance.Draft)
	}
	rows := [][]string{
		humanValueChangeRow("Material", result.Baseline == nil || materialBefore != shortID(inspection.Prospective.Seal.Material), materialBefore, shortID(inspection.Prospective.Seal.Material)),
		humanValueChangeRow("Provenance", result.Baseline == nil || provenanceBefore != shortID(inspection.Prospective.Seal.Provenance), provenanceBefore, shortID(inspection.Prospective.Seal.Provenance)),
		humanValueChangeRow("Content blob", result.Baseline == nil || contentBefore != shortID(candidate.Content), contentBefore, shortID(candidate.Content)),
		humanValueChangeRow("Root boundary", result.Baseline == nil || rootBefore != yesNo(candidate.Root), rootBefore, yesNo(candidate.Root)),
		humanValueChangeRow("Draft", result.Baseline == nil || draftBefore != yesNo(candidate.Draft), draftBefore, yesNo(candidate.Draft)),
	}
	fmt.Fprintln(output)
	printer.table(0, []string{"FIELD", "RESULT", "BEFORE", "CANDIDATE"}, rows)
	if format == 6 {
		var before []domainv5.CauseLink
		if result.Baseline != nil {
			before = result.Baseline.Provenance.CauseLinks
		}
		printHumanCauseChanges(printer, causeLinkChangesV3(before, candidate.CauseLinks).Records)
	}
}

func printHumanCauseChanges(printer humanPrinter, records []causeLinkChangeJSONV3) {
	if len(records) == 0 {
		printer.note("Cause Links unchanged.")
		return
	}
	for _, record := range records {
		action := "changed"
		if record.Before == nil {
			action = "target added"
		} else if record.After == nil {
			action = "target removed"
		}
		fmt.Fprintf(printer.output, "\n  Cause target %s: %s\n", shortTextID(record.Target), action)
		printer.fields(4,
			humanField{"Previous revisions", humanChangeLabel(record.Previous)},
			humanField{"Messages", humanChangeLabel(record.Messages)},
		)
		for _, metadata := range record.Metadata {
			fmt.Fprintf(printer.output, "    Metadata namespace %s: %s\n", quoteHumanString(metadata.Namespace), metadataChangeLabel(metadata))
			printHumanMetadataSide(printer.output, "before", metadata.Before)
			printHumanMetadataSide(printer.output, "after", metadata.After)
		}
	}
}

func humanChangeLabel(change changeJSON) string {
	if change.Changed {
		return "changed"
	}
	return "unchanged"
}

func metadataChangeLabel(change metadataChangeJSON) string {
	if change.Before == nil {
		return "added"
	}
	if change.After == nil {
		return "removed"
	}
	return "changed"
}

func printHumanMetadataSide(output io.Writer, label string, entry *metadataEntryJSON) {
	if entry == nil {
		fmt.Fprintf(output, "      %s: none\n", label)
		return
	}
	schema := "none"
	if entry.Schema != nil {
		schema = quoteHumanString(*entry.Schema)
	}
	fmt.Fprintf(output, "      %s schema: %s\n      %s value: %s\n", label, schema, label, entry.Value)
}

func printFsckHuman(output io.Writer, report repository.FsckReport, format int) {
	printer := newHumanPrinter(output)
	printer.heading("REPOSITORY CHECK: OK")
	rows := [][]string{
		{"Blobs", strconv.Itoa(report.Blobs)},
		{"Seals", strconv.Itoa(report.Seals)},
		{"Materials", strconv.Itoa(report.Materials)},
		{"Provenances", strconv.Itoa(report.Provenances)},
		{"REFs", strconv.Itoa(report.REFs)},
		{"Tags", strconv.Itoa(report.Tags)},
		{"Active Seals", strconv.Itoa(report.ActiveSeals)},
		{"Historical or detached Seals", strconv.Itoa(len(report.HistoricalOrDetachedSeals))},
		{"Unreferenced Blobs", strconv.Itoa(len(report.UnreferencedBlobs))},
	}
	if format == 6 {
		rows = append(rows,
			[]string{"Seals v5", strconv.Itoa(report.SealsV5)}, []string{"Seals v6", strconv.Itoa(report.SealsV6)},
			[]string{"Provenances v1", strconv.Itoa(report.ProvenancesV1)}, []string{"Provenances v2", strconv.Itoa(report.ProvenancesV2)},
			[]string{"Candidates v5", strconv.Itoa(report.CandidatesV5)}, []string{"Candidates v6", strconv.Itoa(report.CandidatesV6)},
		)
	}
	printer.table(0, []string{"INVENTORY", "COUNT"}, rows)
	ids := make([][]string, 0, len(report.HistoricalOrDetachedSeals)+len(report.UnreferencedBlobs))
	for _, id := range report.HistoricalOrDetachedSeals {
		ids = append(ids, []string{"Historical or detached Seal", shortID(id)})
	}
	for _, id := range report.UnreferencedBlobs {
		ids = append(ids, []string{"Unreferenced Blob", shortID(id)})
	}
	if len(ids) != 0 {
		fmt.Fprintln(output)
		printer.table(0, []string{"ITEM", "ID (PREFIX)"}, ids)
	}
}

func printRecoveryInspectionsHuman(output io.Writer, inspections []repository.RecoveryInspection) {
	printer := newHumanPrinter(output)
	printer.heading("LOCAL RECOVERY OPERATIONS")
	if len(inspections) == 0 {
		printer.note("No local recovery operations.")
		return
	}
	rows := make([][]string, 0, len(inspections))
	for _, inspection := range inspections {
		rows = append(rows, []string{inspection.ID, inspection.Kind, string(inspection.Journal), inspection.Status})
	}
	printer.table(0, []string{"OPERATION ID", "KIND", "JOURNAL", "STATUS"}, rows)
	for _, inspection := range inspections {
		if len(inspection.Transitions) == 0 && inspection.Corrupt == "" {
			continue
		}
		fmt.Fprintf(output, "\n  %s\n", inspection.ID)
		if inspection.Corrupt != "" {
			printer.fields(4, humanField{"Error", quoteHumanString(inspection.Corrupt)})
		}
		transitionRows := make([][]string, 0, len(inspection.Transitions))
		for _, transition := range inspection.Transitions {
			transitionRows = append(transitionRows, []string{transition.REF, transition.Current})
		}
		if len(transitionRows) != 0 {
			printer.table(4, []string{"REF", "CURRENT STATE"}, transitionRows)
		}
	}
}
