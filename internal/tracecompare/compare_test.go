package tracecompare

import (
	"bytes"
	"context"
	"testing"
)

func TestComparePriorityAndBoundaries(t *testing.T) {
	tests := []struct {
		name  string
		in    Input
		want  Status
		start int
	}{
		{"shifted", Input{[]byte("abcXdef"), 3, []byte("X"), []byte("QQabcXdef")}, Present, 5},
		{"shifted point before earlier copy", Input{[]byte("abcXdef"), 3, []byte("X"), []byte("XabcXdef")}, Present, 4},
		{"interval before remainder", Input{[]byte("abcXdef"), 3, []byte("X"), []byte("aaaaXaXaaaaa")}, Present, 4},
		{"balanced surrounding edits", Input{[]byte("abcXdef"), 3, []byte("X"), []byte("QQQabcXdefRRR")}, Present, 6},
		{"duplicate priority", Input{[]byte("aaaa"), 1, []byte("aa"), []byte("aaaa")}, Present, 1},
		{"remaining after shifted", Input{[]byte("abcXdef"), 3, []byte("X"), []byte("XabcYdef")}, Present, 0},
		{"last valid start", Input{[]byte("abcdef"), 4, []byte("ef"), []byte("abef")}, Present, 2},
		{"absent", Input{[]byte("abcXdef"), 3, []byte("X"), []byte("abcYdef")}, AbsentExact, -1},
		{"run longer", Input{[]byte("abc"), 1, []byte("bc"), []byte("b")}, AbsentExact, -1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Compare(context.Background(), tt.in)
			if err != nil || got.Status != tt.want || got.Start != tt.start {
				t.Fatalf("got %#v, err %v; want %s at %d", got, err, tt.want, tt.start)
			}
		})
	}
}

func TestCompareRejectsStructuralInvalidity(t *testing.T) {
	for _, in := range []Input{
		{Source: []byte("abc"), SourceStart: -1, Run: []byte("a"), Current: []byte("a")},
		{Source: []byte("abc"), SourceStart: 3, Run: []byte("a"), Current: []byte("a")},
		{Source: []byte("abc"), SourceStart: 1, Run: nil, Current: []byte("a")},
		{Source: []byte("abc"), SourceStart: 1, Run: []byte("x"), Current: []byte("x")},
	} {
		if _, err := Compare(context.Background(), in); err == nil {
			t.Fatalf("invalid input %#v was accepted", in)
		}
	}
}

func TestCompareCancellationIsUndetermined(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	got, err := Compare(ctx, Input{Source: []byte("abc"), SourceStart: 1, Run: []byte("b"), Current: []byte("abc")})
	if err != nil || got.Status != Undetermined || got.Start != -1 || got.Reason == "" {
		t.Fatalf("got %#v, err %v", got, err)
	}
}

type cancelAfterPolls struct {
	context.Context
	polls int
	limit int
}

func (c *cancelAfterPolls) Err() error {
	c.polls++
	if c.polls >= c.limit {
		return context.Canceled
	}
	return nil
}

func TestCompareCancellationDuringMatchIsUndetermined(t *testing.T) {
	ctx := &cancelAfterPolls{Context: context.Background(), limit: 3}
	got, err := Compare(ctx, Input{Source: []byte("a"), SourceStart: 0, Run: []byte("a"), Current: []byte("a")})
	if err != nil || got.Status != Undetermined || got.Start != -1 {
		t.Fatalf("got %#v, err %v", got, err)
	}
}

func TestCompareExhaustiveSmallExistenceOracle(t *testing.T) {
	for sourceLen := 1; sourceLen <= 4; sourceLen++ {
		for _, source := range allBytes(sourceLen) {
			for p := range source {
				for runLen := 1; p+runLen <= len(source); runLen++ {
					run := source[p : p+runLen]
					for currentLen := 0; currentLen <= 4; currentLen++ {
						for _, current := range allBytes(currentLen) {
							got, err := Compare(context.Background(), Input{Source: source, SourceStart: p, Run: run, Current: current})
							if err != nil {
								t.Fatal(err)
							}
							want := bytes.Contains(current, run)
							if (got.Status == Present) != want {
								t.Fatalf("source=%q p=%d run=%q current=%q got=%#v want present=%v", source, p, run, current, got, want)
							}
							if selected := expectedSelectedStart(source, p, run, current); got.Start != selected {
								t.Fatalf("source=%q p=%d run=%q current=%q selected=%d want=%d", source, p, run, current, got.Start, selected)
							}
						}
					}
				}
			}
		}
	}
}

// Compute the selected match by ranking every actual match independently of
// the production range splitting and substring search.
func expectedSelectedStart(source []byte, p int, run, current []byte) int {
	best, bestRank := -1, 4
	shifted := p + len(current) - len(source)
	lo, hi := p, shifted
	if lo > hi {
		lo, hi = hi, lo
	}
	for q := 0; q+len(run) <= len(current); q++ {
		if !bytes.Equal(current[q:q+len(run)], run) {
			continue
		}
		rank := 3
		switch {
		case q == p:
			rank = 0
		case q == shifted:
			rank = 1
		case q >= lo && q <= hi:
			rank = 2
		}
		if rank < bestRank || (rank == bestRank && (best < 0 || q < best)) {
			best, bestRank = q, rank
		}
	}
	return best
}

func allBytes(length int) [][]byte {
	if length == 0 {
		return [][]byte{{}}
	}
	short := allBytes(length - 1)
	result := make([][]byte, 0, len(short)*2)
	for _, prefix := range short {
		for _, b := range []byte{'a', 'b'} {
			value := append([]byte(nil), prefix...)
			result = append(result, append(value, b))
		}
	}
	return result
}
