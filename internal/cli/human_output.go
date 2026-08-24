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
	"github.com/mako10k/sealgraph/internal/graph"
	"github.com/mako10k/sealgraph/internal/history"
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
	text := value.String()
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

func printShowHuman(output io.Writer, result repository.ShowResult) {
	printer := newHumanPrinter(output)
	printer.heading("SEAL")
	printer.fields(0,
		humanField{"Seal ID (prefix)", shortID(result.ID)},
		humanField{"Current REF(s)", humanList(result.REFNames)},
		humanField{"Parent revision", shortOptionalID(result.Payload.ParentRevision)},
		humanField{"Root boundary", yesNo(result.Payload.Root)},
		humanField{"Draft", yesNo(result.Payload.Draft)},
	)
	fmt.Fprintln(output)
	printHumanContent(printer, result.Payload.Content, result.Content)
	printHumanAttachments(printer, result.Payload.Attachments)
	printHumanLinks(printer, result.Payload.Links)
}

func printHumanContent(printer humanPrinter, ref domain.ContentRef, content []byte) {
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
		humanField{"Blob ID (prefix)", shortID(ref.ID)},
		humanField{"Size", fmt.Sprintf("%d bytes", len(content))},
		humanField{label, quoteHumanBytes(preview)},
	)
}

func printHumanAttachments(printer humanPrinter, attachments []domain.Attachment) {
	fmt.Fprintln(printer.output)
	printer.heading(fmt.Sprintf("ATTACHMENTS (%d)", len(attachments)))
	if len(attachments) == 0 {
		printer.note("None")
		return
	}
	rows := make([][]string, 0, len(attachments))
	for _, attachment := range attachments {
		rows = append(rows, []string{quoteHumanString(attachment.Name), quoteHumanString(attachment.MediaType), shortID(attachment.Blob.ID)})
	}
	printer.table(2, []string{"NAME", "MEDIA TYPE", "BLOB ID (PREFIX)"}, rows)
}

func printHumanLinks(printer humanPrinter, links []domain.Link) {
	fmt.Fprintln(printer.output)
	printer.heading(fmt.Sprintf("CAUSES (%d)", len(links)))
	if len(links) == 0 {
		printer.note("None")
		return
	}
	rows := make([][]string, 0, len(links))
	for _, link := range links {
		rows = append(rows, []string{shortID(link.TargetSeal), quoteHumanString(link.Message)})
	}
	printer.table(2, []string{"SEAL ID (PREFIX)", "MESSAGE"}, rows)
}

func printCandidateInspectionHuman(output io.Writer, inspection repository.CandidateInspection) {
	printer := newHumanPrinter(output)
	candidate := inspection.Candidate
	printer.heading("CANDIDATE")
	printer.fields(0,
		humanField{"REF", candidate.REF},
		humanField{"Parent revision", shortOptionalID(candidate.ParentRevision)},
		humanField{"Expected REF head", shortOptionalID(candidate.ExpectedREFHead)},
		humanField{"Current REF head", shortOptionalID(inspection.CurrentHead)},
		humanField{"Publication check", strings.ToLower(strings.ReplaceAll(string(inspection.ExpectedHeadState), "_", " "))},
		humanField{"Root boundary", yesNo(candidate.Root)},
		humanField{"Draft", yesNo(candidate.Draft)},
	)
	fmt.Fprintln(output)
	printHumanContent(printer, candidate.Content, inspection.Content)
	printHumanAttachments(printer, candidate.Attachments)
	printHumanLinks(printer, candidate.Links)
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

func printGraphHuman(output io.Writer, nodes []repository.GraphNode) {
	printer := newHumanPrinter(output)
	printer.heading("REVISION_CAUSE_GRAPH")
	if len(nodes) == 0 {
		printer.note("No active revision or Cause nodes.")
		return
	}
	rows := make([][]string, 0, len(nodes)*2)
	for _, node := range nodes {
		rows = append(rows, []string{"Seal", shortID(node.ID), humanGraphState(string(node.State)), humanList(node.REFs), shortOptionalID(node.Parent)})
		for _, link := range node.Links {
			rows = append(rows, []string{"  Cause", shortID(link.Target), humanGraphState(string(link.State)), "", ""})
		}
	}
	printer.table(0, []string{"RELATION", "SEAL ID (PREFIX)", "STATE", "CURRENT REF(S)", "PARENT"}, rows)
}

func humanGraphState(value string) string {
	return strings.ToLower(strings.ReplaceAll(value, "_", " "))
}

func printImpactsHuman(output io.Writer, source domain.ObjectID, impacts []graph.Impact, limit int) {
	printer := newHumanPrinter(output)
	printer.heading("STRUCTURAL_IMPACT")
	printer.fields(0, humanField{"Source Seal ID (prefix)", shortID(source)})
	if len(impacts) == 0 {
		printer.note("No current downstream Seals are impacted.")
		return
	}
	rows := make([][]string, 0, len(impacts))
	for _, impact := range impacts {
		rows = append(rows, []string{shortID(impact.Head), humanList(impact.REFs), strconv.Itoa(len(impact.Paths))})
	}
	fmt.Fprintln(output)
	printer.table(0, []string{"DOWNSTREAM SEAL", "CURRENT REF(S)", "PATHS"}, rows)
	for _, impact := range impacts {
		fmt.Fprintf(output, "\n  %s\n", shortID(impact.Head))
		for index, path := range impact.Paths {
			fmt.Fprintf(output, "    Path %d\n", index+1)
			for depth, id := range path {
				marker := "starts at"
				if depth != 0 {
					marker = "causes"
				}
				fmt.Fprintf(output, "%s%s  %s\n", strings.Repeat("  ", depth+3), marker, shortID(id))
			}
		}
		if impact.Truncated {
			fmt.Fprintf(output, "    Additional paths omitted (maximum %d).\n", limit)
		}
	}
}

func printLogHuman(output io.Writer, ref string, entries []history.Entry) {
	printer := newHumanPrinter(output)
	printer.heading("REVISION HISTORY")
	printer.fields(0, humanField{"REF", ref}, humanField{"Revisions", strconv.Itoa(len(entries))})
	for index, entry := range entries {
		label := fmt.Sprintf("Revision %d", index+1)
		if index == 0 {
			label += " (current)"
		}
		fmt.Fprintf(output, "\n%s\n", label)
		printer.fields(2,
			humanField{"Seal ID (prefix)", shortID(entry.ID)},
			humanField{"Parent revision", shortOptionalID(entry.Payload.ParentRevision)},
			humanField{"Content blob", shortID(entry.Payload.Content.ID)},
			humanField{"Root boundary", yesNo(entry.Payload.Root)},
			humanField{"Draft", yesNo(entry.Payload.Draft)},
		)
		printHumanLinksIndented(printer, entry.Payload.Links, 2)
	}
}

func printHumanLinksIndented(printer humanPrinter, links []domain.Link, indent int) {
	fmt.Fprintf(printer.output, "%sCauses (%d)\n", strings.Repeat(" ", indent), len(links))
	if len(links) == 0 {
		fmt.Fprintf(printer.output, "%sNone\n", strings.Repeat(" ", indent+2))
		return
	}
	rows := make([][]string, 0, len(links))
	for _, link := range links {
		rows = append(rows, []string{shortID(link.TargetSeal), quoteHumanString(link.Message)})
	}
	printer.table(indent+2, []string{"SEAL ID (PREFIX)", "MESSAGE"}, rows)
}

func printLinkLogHuman(output io.Writer, ref, upstream string, entries []history.LinkLogEntry) {
	printer := newHumanPrinter(output)
	printer.heading("CAUSE LINK HISTORY")
	fields := []humanField{{"REF", ref}, {"Revisions", strconv.Itoa(len(entries))}}
	if upstream != "" {
		fields = append(fields, humanField{"Upstream Seal ID (prefix)", abbreviateIDText(upstream)})
	}
	printer.fields(0, fields...)
	for index, entry := range entries {
		fmt.Fprintf(output, "\nRevision %d\n", index+1)
		printer.fields(2,
			humanField{"Seal ID (prefix)", shortID(entry.Entry.ID)},
			humanField{"Parent revision", shortOptionalID(entry.Entry.Payload.ParentRevision)},
		)
		printHumanLinkChanges(printer, entry.Changes, 2)
	}
}

func abbreviateIDText(value string) string {
	if len(value) <= shortIDLength {
		return value
	}
	return value[:shortIDLength]
}

func printSealDiffHuman(output io.Writer, diff history.SealDiff) {
	printer := newHumanPrinter(output)
	printer.heading("SEAL COMPARISON")
	printer.fields(0, humanField{"From Seal ID (prefix)", shortID(diff.From)}, humanField{"To Seal ID (prefix)", shortID(diff.To)})
	rows := [][]string{
		humanValueChangeRow("Content blob", diff.Content.Changed, shortID(diff.Content.Before.ID), shortID(diff.Content.After.ID)),
		humanValueChangeRow("Root boundary", diff.Root.Changed, yesNo(diff.Root.Before), yesNo(diff.Root.After)),
		humanValueChangeRow("Draft", diff.Draft.Changed, yesNo(diff.Draft.Before), yesNo(diff.Draft.After)),
		humanValueChangeRow("Parent revision", diff.Parent.Changed, shortOptionalID(diff.Parent.Before), shortOptionalID(diff.Parent.After)),
	}
	fmt.Fprintln(output)
	printer.table(0, []string{"FIELD", "RESULT", "BEFORE", "AFTER"}, rows)
	printHumanAttachmentChanges(printer, diff.Attachments)
	printHumanLinkChanges(printer, diff.Links, 0)
}

func humanValueChangeRow(field string, changed bool, before, after string) []string {
	result := "unchanged"
	if changed {
		result = "changed"
	}
	return []string{field, result, before, after}
}

func printCandidateDiffHuman(output io.Writer, result repository.CandidateDiffResult) {
	printer := newHumanPrinter(output)
	inspection, diff := result.Inspection, result.Diff
	candidate := inspection.Candidate
	printer.heading("CANDIDATE COMPARISON")
	printer.fields(0,
		humanField{"REF", candidate.REF},
		humanField{"Compared with", shortOptionalID(candidate.ParentRevision)},
		humanField{"Expected REF head", shortOptionalID(candidate.ExpectedREFHead)},
		humanField{"Current REF head", shortOptionalID(inspection.CurrentHead)},
		humanField{"Publication check", strings.ToLower(strings.ReplaceAll(string(inspection.ExpectedHeadState), "_", " "))},
	)
	contentBefore := "none"
	if !diff.Initial {
		contentBefore = shortID(diff.Content.Before.ID)
	}
	rows := [][]string{
		humanValueChangeRow("Content blob", diff.Initial || diff.Content.Changed, contentBefore, shortID(diff.Content.After.ID)),
		humanValueChangeRow("Root boundary", diff.Initial || diff.Root.Changed, initialBool(diff.Initial, diff.Root.Before), yesNo(diff.Root.After)),
		humanValueChangeRow("Draft", diff.Initial || diff.Draft.Changed, initialBool(diff.Initial, diff.Draft.Before), yesNo(diff.Draft.After)),
	}
	fmt.Fprintln(output)
	printer.table(0, []string{"FIELD", "RESULT", "BEFORE", "CANDIDATE"}, rows)
	printHumanAttachmentChanges(printer, diff.Attachments)
	printHumanLinkChanges(printer, diff.Links, 0)
}

func initialBool(initial, value bool) string {
	if initial {
		return "none"
	}
	return yesNo(value)
}

func printHumanAttachmentChanges(printer humanPrinter, changes []history.AttachmentChangeRecord) {
	fmt.Fprintln(printer.output)
	printer.heading("ATTACHMENT CHANGES")
	if len(changes) == 0 {
		printer.note("None")
		return
	}
	rows := make([][]string, 0, len(changes))
	for _, change := range changes {
		before, after := "none", "none"
		if change.Before != nil {
			before = shortID(change.Before.Blob.ID)
		}
		if change.After != nil {
			after = shortID(change.After.Blob.ID)
		}
		rows = append(rows, []string{string(change.Kind), quoteHumanString(change.Name), before, after})
	}
	printer.table(2, []string{"CHANGE", "NAME", "BEFORE BLOB", "AFTER BLOB"}, rows)
}

func printHumanLinkChanges(printer humanPrinter, changes []history.LinkChange, indent int) {
	fmt.Fprintln(printer.output)
	fmt.Fprintf(printer.output, "%sCAUSE CHANGES\n", strings.Repeat(" ", indent))
	if len(changes) == 0 {
		fmt.Fprintf(printer.output, "%sNone\n", strings.Repeat(" ", indent+2))
		return
	}
	rows := make([][]string, 0, len(changes))
	for _, change := range changes {
		before, after, message := shortOptionalID(change.BeforeSeal), shortOptionalID(change.AfterSeal), change.AfterMessage
		if message == "" {
			message = change.BeforeMessage
		}
		rows = append(rows, []string{string(change.Kind), shortID(change.TargetSeal), before, after, quoteHumanString(message)})
	}
	printer.table(indent+2, []string{"CHANGE", "TARGET", "BEFORE", "AFTER", "MESSAGE"}, rows)
}

func printFsckHuman(output io.Writer, report repository.FsckReport) {
	printer := newHumanPrinter(output)
	printer.heading("REPOSITORY CHECK: OK")
	rows := [][]string{
		{"Objects", strconv.Itoa(report.Objects)},
		{"Seals", strconv.Itoa(report.Seals)},
		{"Material objects", strconv.Itoa(report.MaterialObjects)},
		{"REFs", strconv.Itoa(report.REFs)},
		{"Tags", strconv.Itoa(report.Tags)},
		{"Active Seals", strconv.Itoa(report.ActiveSeals)},
		{"Historical or detached Seals", strconv.Itoa(len(report.HistoricalOrDetachedSeals))},
		{"Unreferenced objects", strconv.Itoa(len(report.UnreferencedObjects))},
	}
	printer.table(0, []string{"INVENTORY", "COUNT"}, rows)
	ids := make([][]string, 0, len(report.HistoricalOrDetachedSeals)+len(report.UnreferencedObjects))
	for _, id := range report.HistoricalOrDetachedSeals {
		ids = append(ids, []string{"Historical or detached Seal", shortID(id)})
	}
	for _, id := range report.UnreferencedObjects {
		ids = append(ids, []string{"Unreferenced object", shortID(id)})
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
