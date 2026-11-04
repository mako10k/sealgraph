package cli

// R: render bounded, read-only text differences between two exact Seal contents.

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"
	"unicode/utf8"

	"github.com/mako10k/sealgraph/internal/repository"
)

const compareTextLimit = 64 * 1024
const compareTextCells = 1024 * 1024

func extractCompareText(args []string) ([]string, bool, error) {
	result := make([]string, 0, len(args))
	text := false
	for _, arg := range args {
		if arg != "--text" {
			result = append(result, arg)
			continue
		}
		if text {
			return nil, false, fmt.Errorf("--text may be specified only once")
		}
		text = true
	}
	return result, text, nil
}

func renderTextComparison(ctx context.Context, repo *repository.Repository, diff repository.SealComparison, from, to string) ([]byte, error) {
	left, err := repo.Show(ctx, "@"+diff.From.ID.String())
	if err != nil {
		return nil, err
	}
	right, err := repo.Show(ctx, "@"+diff.To.ID.String())
	if err != nil {
		return nil, err
	}
	observed, err := repo.DiffSelectors(ctx, from, to)
	if err != nil {
		return nil, err
	}
	if !observed.From.ID.Equal(diff.From.ID) || !observed.To.ID.Equal(diff.To.ID) {
		return nil, fmt.Errorf("comparison selection changed during text inspection; inspect the selected versions and retry explicitly")
	}
	var out bytes.Buffer
	printSealDiffHuman(&out, diff, repo.Format())
	fmt.Fprintln(&out, "\nTEXT COMPARISON (escaped lines; - before, + after)")
	fmt.Fprintf(&out, "Exact before: %s\nExact after:  %s\n", diff.From.ID, diff.To.ID)
	printTextChanges(&out, left.Content, right.Content)
	fmt.Fprintf(&out, "\nFull exact originals (read-only):\n  sealgraph show @%s --raw-content\n  sealgraph show @%s --raw-content\n", diff.From.ID, diff.To.ID)
	fmt.Fprintf(&out, "Cause records and recorded rationales (separate from raw content):\n  sealgraph show @%s --format json\n  sealgraph show @%s --format json\n", diff.From.ID, diff.To.ID)
	fmt.Fprintln(&out, "Cause records and review claims do not establish semantic correctness or approval.")
	return out.Bytes(), nil
}

func printableCompareText(data []byte) bool {
	if !utf8.Valid(data) {
		return false
	}
	for _, value := range data {
		if value < 32 && value != '\n' && value != '\r' && value != '\t' || value == 127 {
			return false
		}
	}
	return true
}

type compareTextOutput struct {
	out       io.Writer
	remaining int
	omitted   bool
}

func (w *compareTextOutput) line(marker, line string) {
	// Bound before escaping too: one giant line must not allocate an unbounded preview.
	if len(line) > compareTextLimit {
		line = line[:compareTextLimit]
		w.omitted = true
	}
	value := marker + " " + quoteHumanString(line) + "\n"
	if len(value) > w.remaining {
		w.omitted = true
		return
	}
	fmt.Fprint(w.out, value)
	w.remaining -= len(value)
}

func printTextChanges(out io.Writer, left, right []byte) {
	fmt.Fprintf(out, "Content bytes: before=%d after=%d; input preview limit=%d bytes per side; displayed line budget=%d bytes\n", len(left), len(right), compareTextLimit, compareTextLimit)
	if bytes.Equal(left, right) {
		fmt.Fprintln(out, "Content bytes unchanged.")
		return
	}
	if !printableCompareText(left) || !printableCompareText(right) {
		fmt.Fprintln(out, "Binary or non-text content: line comparison unavailable; extract exact originals below.")
		return
	}
	clipped := len(left) > compareTextLimit || len(right) > compareTextLimit
	left = left[:min(len(left), compareTextLimit)]
	right = right[:min(len(right), compareTextLimit)]
	// Trim a clipped UTF-8 suffix so an otherwise valid original remains readable.
	left = trimCompareUTF8(left)
	right = trimCompareUTF8(right)
	before, after := strings.SplitAfter(string(left), "\n"), strings.SplitAfter(string(right), "\n")
	output := compareTextOutput{out: out, remaining: compareTextLimit}
	printAlignedText(&output, before, after)
	if clipped || output.omitted {
		fmt.Fprintln(out, "OMITTED: bounded preview is incomplete; use the full exact-original commands below. Unshown bytes were not compared as lines.")
	}
}

func trimCompareUTF8(value []byte) []byte {
	for !utf8.Valid(value) {
		value = value[:len(value)-1]
	}
	return value
}

func printAlignedText(out *compareTextOutput, before, after []string) {
	if (len(before)+1)*(len(after)+1) > compareTextCells {
		fmt.Fprintln(out.out, "Line alignment unavailable: LCS cell budget exceeded; showing bounded before/after lines, not a minimal diff.")
		for _, line := range before {
			out.line("-", line)
		}
		for _, line := range after {
			out.line("+", line)
		}
		return
	}
	width := len(after) + 1
	lcs := make([]uint32, (len(before)+1)*width)
	for i := len(before) - 1; i >= 0; i-- {
		for j := len(after) - 1; j >= 0; j-- {
			if before[i] == after[j] {
				lcs[i*width+j] = lcs[(i+1)*width+j+1] + 1
			} else {
				lcs[i*width+j] = max(lcs[(i+1)*width+j], lcs[i*width+j+1])
			}
		}
	}
	emitAlignedText(out, before, after, lcs, width)
}

func emitAlignedText(out *compareTextOutput, before, after []string, lcs []uint32, width int) {
	i, j := 0, 0
	common := 0
	flush := func() {
		if common > 0 {
			fmt.Fprintf(out.out, "  [%d unchanged lines]\n", common)
			common = 0
		}
	}
	for i < len(before) && j < len(after) {
		if before[i] == after[j] {
			i++
			j++
			common++
			continue
		}
		flush()
		if lcs[(i+1)*width+j] >= lcs[i*width+j+1] {
			out.line("-", before[i])
			i++
		} else {
			out.line("+", after[j])
			j++
		}
	}
	flush()
	for ; i < len(before); i++ {
		out.line("-", before[i])
	}
	for ; j < len(after); j++ {
		out.line("+", after[j])
	}
}
