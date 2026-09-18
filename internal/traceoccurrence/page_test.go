package traceoccurrence

import (
	"errors"
	"reflect"
	"testing"
)

func TestPageOverlappingLimitOne(t *testing.T) {
	data, pattern := []byte("aaaa"), []byte("aa")
	starts, more, err := Page(data, pattern, -1, 1)
	if err != nil || !reflect.DeepEqual(starts, []int{0}) || !more {
		t.Fatalf("first page = %v, %t, %v", starts, more, err)
	}
	starts, more, err = Page(data, pattern, starts[len(starts)-1], 1)
	if err != nil || !reflect.DeepEqual(starts, []int{1}) || !more {
		t.Fatalf("second page = %v, %t, %v", starts, more, err)
	}
	starts, more, err = Page(data, pattern, starts[len(starts)-1], 1)
	if err != nil || !reflect.DeepEqual(starts, []int{2}) || more {
		t.Fatalf("last page = %v, %t, %v", starts, more, err)
	}
}

func TestPageLastOffsetAndShorterData(t *testing.T) {
	starts, more, err := Page([]byte("abcab"), []byte("ab"), 0, 10)
	if err != nil || !reflect.DeepEqual(starts, []int{3}) || more {
		t.Fatalf("after offset = %v, %t, %v", starts, more, err)
	}
	starts, more, err = Page([]byte("a"), []byte("aa"), -1, 1)
	if err != nil || len(starts) != 0 || more {
		t.Fatalf("shorter data = %v, %t, %v", starts, more, err)
	}
}

func TestPageBinaryBytes(t *testing.T) {
	data := []byte{0, 255, 0, 255, 0}
	starts, more, err := Page(data, []byte{0, 255, 0}, -1, 10)
	if err != nil || !reflect.DeepEqual(starts, []int{0, 2}) || more {
		t.Fatalf("binary data = %v, %t, %v", starts, more, err)
	}
}

func TestPageRejectsInvalidLimit(t *testing.T) {
	for _, limit := range []int{0, -1} {
		starts, more, err := Page([]byte("abc"), []byte("a"), -1, limit)
		if !errors.Is(err, errInvalidLimit) || starts != nil || more {
			t.Fatalf("limit %d = %v, %t, %v", limit, starts, more, err)
		}
	}
}

func TestPageRejectsEmptyPatternAndInvalidAfter(t *testing.T) {
	if _, _, err := Page([]byte("abc"), nil, -1, 1); !errors.Is(err, errEmptyPattern) {
		t.Fatalf("empty pattern error = %v", err)
	}
	if _, _, err := Page([]byte("abc"), []byte("a"), -2, 1); !errors.Is(err, errInvalidAfter) {
		t.Fatalf("invalid after error = %v", err)
	}
}
