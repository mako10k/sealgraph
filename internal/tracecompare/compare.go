// Package tracecompare contains the pure byte comparison used by origin trace.
package tracecompare

import (
	"bytes"
	"context"
	"fmt"
	"math"
)

// Status is the outcome of an exact run search.
type Status string

const (
	Present      Status = "PRESENT"
	AbsentExact  Status = "ABSENT_EXACT"
	Undetermined Status = "UNDETERMINED"
)

// Input is a complete, fixed comparison snapshot. The comparator does not
// read files, refs, or Git state.
type Input struct {
	Source      []byte
	SourceStart int
	Run         []byte
	Current     []byte
}

// Result contains the selected byte start when the run is present. Start is
// -1 for ABSENT_EXACT and UNDETERMINED.
type Result struct {
	Status Status
	Start  int
	Reason string
}

// Compare searches for the exact sealed run in Current according to the
// deterministic priority defined by the origin trace comparison contract.
func Compare(ctx context.Context, in Input) (Result, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return undetermined(err), nil
	}
	if err := validateInput(in); err != nil {
		return Result{}, err
	}

	l := len(in.Run)
	n := len(in.Current)
	if n < l {
		return Result{Status: AbsentExact, Start: -1}, nil
	}
	maxStart := n - l
	first := in.SourceStart
	second, secondOK := checkedAdd(first, n-len(in.Source))
	if result, ok := searchPoints(ctx, in.Current, in.Run, first, second, secondOK, maxStart); ok {
		return result, nil
	}
	if err := ctx.Err(); err != nil {
		return undetermined(err), nil
	}

	lo, hi, intervalOK := clippedInterval(first, second, secondOK, maxStart)
	if intervalOK {
		if result, ok := scanRange(ctx, in.Current, in.Run, lo, hi, first, second); ok {
			return result, nil
		}
		if err := ctx.Err(); err != nil {
			return undetermined(err), nil
		}
	}
	if result, ok := searchRemainder(ctx, in.Current, in.Run, lo, hi, intervalOK, maxStart, first, second); ok {
		return result, nil
	}
	if err := ctx.Err(); err != nil {
		return undetermined(err), nil
	}
	return Result{Status: AbsentExact, Start: -1}, nil
}

func validateInput(in Input) error {
	if in.SourceStart < 0 || in.SourceStart > len(in.Source) {
		return fmt.Errorf("source_start %d is outside source length %d", in.SourceStart, len(in.Source))
	}
	if len(in.Run) == 0 {
		return fmt.Errorf("sealed run must be nonempty")
	}
	if len(in.Run) > len(in.Source)-in.SourceStart {
		return fmt.Errorf("source_start %d and run length %d exceed source length %d", in.SourceStart, len(in.Run), len(in.Source))
	}
	if !bytes.Equal(in.Source[in.SourceStart:in.SourceStart+len(in.Run)], in.Run) {
		return fmt.Errorf("sealed run does not match source at source_start %d", in.SourceStart)
	}
	return nil
}

func searchPoints(ctx context.Context, current, run []byte, first, second int, secondOK bool, maxStart int) (Result, bool) {
	// First and second priority positions are checked individually. Invalid
	// positions are skipped, never clamped to a different position.
	firstValid := first <= maxStart
	if firstValid {
		if result, ok := check(ctx, current, run, first); ok {
			return result, true
		}
	}
	secondValid := secondOK && second >= 0 && second <= maxStart
	if secondValid && (!firstValid || second != first) {
		if result, ok := check(ctx, current, run, second); ok {
			return result, true
		}
	}
	return Result{}, false
}

func clippedInterval(first, second int, secondOK bool, maxStart int) (int, int, bool) {
	// The closed interval is based on the raw positions, then intersected
	// with valid starts. If checked arithmetic cannot represent the shifted
	// position, the remaining scan still supplies complete coverage.
	if !secondOK {
		return 0, -1, false
	}
	lo, hi := first, second
	if lo > hi {
		lo, hi = hi, lo
	}
	if lo < 0 {
		lo = 0
	}
	if hi > maxStart {
		hi = maxStart
	}
	return lo, hi, lo <= hi
}

func searchRemainder(ctx context.Context, current, run []byte, lo, hi int, intervalOK bool, maxStart, first, second int) (Result, bool) {
	// The remaining valid starts are scanned in ascending order, split around
	// the already inspected interval and point positions.
	if !intervalOK {
		return scanRange(ctx, current, run, 0, maxStart, first, second)
	}
	for _, r := range [][2]int{{0, lo - 1}, {hi + 1, maxStart}} {
		if result, ok := scanRange(ctx, current, run, r[0], r[1], first, second); ok {
			return result, true
		}
	}
	return Result{}, false
}

func check(ctx context.Context, current, run []byte, start int) (Result, bool) {
	if err := ctx.Err(); err != nil {
		return Result{}, false
	}
	if bytes.Equal(current[start:start+len(run)], run) {
		if ctx.Err() != nil {
			return Result{}, false
		}
		return Result{Status: Present, Start: start}, true
	}
	return Result{}, false
}

func scanRange(ctx context.Context, current, run []byte, lo, hi, first, second int) (Result, bool) {
	if lo > hi {
		return Result{}, false
	}
	// Exclude points already checked so the range search has no duplicate work.
	for _, point := range []int{first, second} {
		if point < lo || point > hi {
			continue
		}
		if point > lo {
			if result, ok := scanRange(ctx, current, run, lo, point-1, math.MinInt, math.MinInt); ok {
				return result, ok
			}
		}
		lo = point + 1
	}
	if lo > hi {
		return Result{}, false
	}
	if err := ctx.Err(); err != nil {
		return Result{}, false
	}
	end, ok := checkedAdd(hi, len(run))
	if !ok || end > len(current) {
		return Result{}, false
	}
	idx := bytes.Index(current[lo:end], run)
	if idx < 0 {
		return Result{}, false
	}
	if ctx.Err() != nil {
		return Result{}, false
	}
	return Result{Status: Present, Start: lo + idx}, true
}

func checkedAdd(a, b int) (int, bool) {
	if b > 0 && a > math.MaxInt-b {
		return 0, false
	}
	if b < 0 && a < math.MinInt-b {
		return 0, false
	}
	return a + b, true
}

func undetermined(err error) Result {
	return Result{Status: Undetermined, Start: -1, Reason: err.Error()}
}
