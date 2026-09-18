// Package traceoccurrence provides pure byte occurrence paging primitives.
package traceoccurrence

import (
	"bytes"
	"errors"
)

var (
	errEmptyPattern = errors.New("pattern must not be empty")
	errInvalidAfter = errors.New("after must be -1 or a non-negative byte offset")
	errInvalidLimit = errors.New("limit must be positive")
)

// Page returns at most limit overlapping occurrences of pattern in data.
// Occurrences are returned in ascending byte offset order and start strictly
// after after. Use after == -1 for the first page. hasMore is true only when a
// further occurrence was found after the returned page.
func Page(data, pattern []byte, after, limit int) (starts []int, hasMore bool, err error) {
	if len(pattern) == 0 {
		return nil, false, errEmptyPattern
	}
	if limit <= 0 {
		return nil, false, errInvalidLimit
	}
	if after < -1 {
		return nil, false, errInvalidAfter
	}
	if len(pattern) > len(data) || after >= len(data)-1 {
		return []int{}, false, nil
	}

	first := after + 1
	last := len(data) - len(pattern)
	if first > last {
		return []int{}, false, nil
	}
	capacity := min(limit, last-first+1)
	starts = make([]int, 0, capacity)
	for offset := first; offset <= last; offset++ {
		if !bytes.Equal(data[offset:offset+len(pattern)], pattern) {
			continue
		}
		if len(starts) == limit {
			return starts, true, nil
		}
		starts = append(starts, offset)
	}
	return starts, false, nil
}
