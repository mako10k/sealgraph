package traceestimate

import "testing"

func TestEstimateReplacement(t *testing.T) {
	result, err := Estimate([]byte("abc"), Range{Start: 1, Length: 1}, []byte("aXc"))
	if err != nil {
		t.Fatal(err)
	}
	if result.State != StateCandidates || len(result.Candidates) != 1 {
		t.Fatalf("result=%+v", result)
	}
	want := []Range{{Start: 1, Length: 1}}
	if !equalRanges(result.Candidates[0].CurrentRanges, want) {
		t.Fatalf("ranges=%v, want %v", result.Candidates[0].CurrentRanges, want)
	}
	if result.Method != Method || result.Candidates[0].EvidenceKind != EvidenceKind {
		t.Fatalf("method/evidence=%q/%q", result.Method, result.Candidates[0].EvidenceKind)
	}
}

func TestEstimateInsertion(t *testing.T) {
	result, err := Estimate([]byte("abc"), Range{Start: 0, Length: 3}, []byte("aXbc"))
	if err != nil {
		t.Fatal(err)
	}
	if !equalRanges(result.Candidates[0].CurrentRanges, []Range{{Start: 0, Length: 4}}) {
		t.Fatalf("ranges=%v", result.Candidates[0].CurrentRanges)
	}
}

func TestEstimatePureBoundaryInsertionIsOutsideRange(t *testing.T) {
	result, err := Estimate([]byte("abc"), Range{Start: 1, Length: 1}, []byte("aXbc"))
	if err != nil {
		t.Fatal(err)
	}
	// The insertion is at old range start and has no deleted/changed old byte.
	if !equalRanges(result.Candidates[0].CurrentRanges, []Range{{Start: 2, Length: 1}}) {
		t.Fatalf("ranges=%v", result.Candidates[0].CurrentRanges)
	}
}

func TestEstimateDeletion(t *testing.T) {
	result, err := Estimate([]byte("abc"), Range{Start: 1, Length: 1}, []byte("ac"))
	if err != nil {
		t.Fatal(err)
	}
	if result.State != StateNoCandidate || len(result.Candidates) != 0 || result.Reason == "" {
		t.Fatalf("result=%+v", result)
	}
}

func TestEstimateRepeatedBytesUsesDeterministicTieBreak(t *testing.T) {
	oldBytes, currentBytes := []byte("aaaa"), []byte("baaa")
	first, err := Estimate(oldBytes, Range{Start: 1, Length: 1}, currentBytes)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 20; i++ {
		next, err := Estimate(oldBytes, Range{Start: 1, Length: 1}, currentBytes)
		if err != nil {
			t.Fatal(err)
		}
		if next.State != first.State || !equalRanges(next.Candidates[0].CurrentRanges, first.Candidates[0].CurrentRanges) || next.Method != first.Method {
			t.Fatalf("run %d=%+v, first=%+v", i, next, first)
		}
	}
}

func TestEstimateInvalidRange(t *testing.T) {
	if _, err := Estimate([]byte("abc"), Range{Start: 3, Length: 1}, []byte("abc")); err == nil {
		t.Fatal("expected invalid range error")
	}
}

func TestEstimateModerateInput(t *testing.T) {
	oldBytes := make([]byte, 2048)
	for i := range oldBytes {
		oldBytes[i] = byte(i % 17)
	}
	currentBytes := make([]byte, 0, len(oldBytes)+3)
	currentBytes = append(currentBytes, oldBytes[:1024]...)
	currentBytes = append(currentBytes, 'x', 'y', 'z')
	currentBytes = append(currentBytes, oldBytes[1024:]...)
	result, err := Estimate(oldBytes, Range{Start: 1000, Length: 40}, currentBytes)
	if err != nil {
		t.Fatal(err)
	}
	if result.State != StateCandidates || !equalRanges(result.Candidates[0].CurrentRanges, []Range{{Start: 1000, Length: 43}}) {
		t.Fatalf("result=%+v", result)
	}
}

func equalRanges(left, right []Range) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}
