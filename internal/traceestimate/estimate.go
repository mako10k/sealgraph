// Package traceestimate projects an old byte range through a deterministic
// single diff. It deliberately has no repository, filesystem, or CLI
// dependencies.
package traceestimate

import "fmt"

// Method is the fixed identifier for the estimator and its tie-break rules.
const Method = "single-diff-lcs-v1"

// EvidenceKind identifies the kind of evidence used for a candidate.
const EvidenceKind = "single_diff"

// State is the result state of an estimate.
type State string

const (
	StateCandidates  State = "CANDIDATES"
	StateNoCandidate State = "NO_CANDIDATE"
	StateIncomplete  State = "INCOMPLETE"
)

// Range is a half-open byte range in the current file.
type Range struct {
	Start  int
	Length int
}

// Candidate is one projected current range result.
type Candidate struct {
	CurrentRanges []Range
	EvidenceKind  string
	Method        string
	Reason        string
}

// Result is the complete estimator result. A no-candidate result has no
// candidates and a reason; an incomplete result additionally retains the
// method that was selected before the interruption.
type Result struct {
	State      State
	Method     string
	Candidates []Candidate
	Reason     string
}

// Estimate projects oldRange from oldBytes into currentBytes using one
// deterministic insertion/deletion diff. Hirschberg split ties choose the
// lowest current offset. The method returns an error only for a
// structurally invalid old range; those inputs are not an estimation result.
func Estimate(oldBytes []byte, oldRange Range, currentBytes []byte) (Result, error) {
	if oldRange.Start < 0 || oldRange.Length <= 0 {
		return Result{}, fmt.Errorf("old range must have non-negative start and positive length")
	}
	if oldRange.Start > len(oldBytes) || oldRange.Length > len(oldBytes)-oldRange.Start {
		return Result{}, fmt.Errorf("old range is outside old bytes")
	}

	start, end := oldRange.Start, oldRange.Start+oldRange.Length
	matches := hirschberg(oldBytes, currentBytes)
	projected := projectMatches(matches, start, end, len(oldBytes), len(currentBytes))

	ranges := positionsToRanges(projected)
	if len(ranges) == 0 {
		return Result{
			State:  StateNoCandidate,
			Method: Method,
			Reason: "the deterministic single diff has no current bytes corresponding to the old range",
		}, nil
	}
	return Result{
		State:  StateCandidates,
		Method: Method,
		Candidates: []Candidate{{
			CurrentRanges: ranges,
			EvidenceKind:  EvidenceKind,
			Method:        Method,
			Reason:        "projected from the deterministic single insertion/deletion diff",
		}},
	}, nil
}

type match struct{ old, current int }

// hirschberg returns one deterministic longest common subsequence. Splitting
// ties at the lowest current offset is the fixed tie-break for this method.
// Each dynamic-programming row is linear in the current input length.
func hirschberg(oldBytes, currentBytes []byte) []match {
	if len(oldBytes) == 0 || len(currentBytes) == 0 {
		return nil
	}
	if len(oldBytes) == 1 {
		for current, value := range currentBytes {
			if oldBytes[0] == value {
				return []match{{old: 0, current: current}}
			}
		}
		return nil
	}
	middle := len(oldBytes) / 2
	split := splitPoint(oldBytes, currentBytes, middle)
	result := hirschberg(oldBytes[:middle], currentBytes[:split])
	for _, item := range hirschberg(oldBytes[middle:], currentBytes[split:]) {
		result = append(result, match{old: item.old + middle, current: item.current + split})
	}
	return result
}

func splitPoint(oldBytes, currentBytes []byte, middle int) int {
	left := lcsPrefix(oldBytes[:middle], currentBytes)
	right := lcsSuffix(oldBytes[middle:], currentBytes)
	best, split := -1, 0
	for current := range left {
		value := left[current] + right[current]
		if value > best {
			best, split = value, current
		}
	}
	return split
}

func lcsPrefix(oldBytes, currentBytes []byte) []int {
	row := make([]int, len(currentBytes)+1)
	for _, old := range oldBytes {
		previous := 0
		for current, value := range currentBytes {
			above := row[current+1]
			if old == value {
				row[current+1] = previous + 1
			} else if row[current] > row[current+1] {
				row[current+1] = row[current]
			}
			previous = above
		}
	}
	return row
}

func lcsSuffix(oldBytes, currentBytes []byte) []int {
	reversedOld := reverseBytes(oldBytes)
	reversedCurrent := reverseBytes(currentBytes)
	row := lcsPrefix(reversedOld, reversedCurrent)
	result := make([]int, len(row))
	for i := range row {
		result[i] = row[len(currentBytes)-i]
	}
	return result
}

func reverseBytes(value []byte) []byte {
	result := make([]byte, len(value))
	for i := range value {
		result[len(value)-1-i] = value[i]
	}
	return result
}

func projectMatches(matches []match, start, end, oldLength, currentLength int) []int {
	projected := make([]int, 0, end-start)
	old, current := 0, 0
	for _, item := range matches {
		deletedInRange := old < end && item.old > start
		for current < item.current {
			if (old > start && old < end) || deletedInRange {
				projected = append(projected, current)
			}
			current++
		}
		if item.old >= start && item.old < end {
			projected = append(projected, item.current)
		}
		old, current = item.old+1, item.current+1
	}
	deletedInRange := old < end && oldLength > start
	for current < currentLength {
		if (old > start && old < end) || deletedInRange {
			projected = append(projected, current)
		}
		current++
	}
	return projected
}

func positionsToRanges(positions []int) []Range {
	if len(positions) == 0 {
		return nil
	}
	ranges := make([]Range, 0, len(positions))
	start, previous := positions[0], positions[0]
	for _, position := range positions[1:] {
		if position == previous+1 {
			previous = position
			continue
		}
		ranges = append(ranges, Range{Start: start, Length: previous - start + 1})
		start, previous = position, position
	}
	return append(ranges, Range{Start: start, Length: previous - start + 1})
}
