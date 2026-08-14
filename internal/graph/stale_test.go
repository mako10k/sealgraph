package graph

import (
	"reflect"
	"testing"
)

func TestStaleFrontierHandlesChainsDiamondsAndDuplicates(t *testing.T) {
	tests := []struct {
		name      string
		stale     []string
		upstreams map[string][]string
		want      []string
	}{
		{
			name:  "chain selects upstream-most stale REF",
			stale: []string{"C", "B"},
			upstreams: map[string][]string{
				"B": {"A"},
				"C": {"B"},
			},
			want: []string{"B"},
		},
		{
			name:  "diamond selects both upstream arms",
			stale: []string{"D", "C", "B", "D"},
			upstreams: map[string][]string{
				"B": {"A"},
				"C": {"A"},
				"D": {"B", "C", "B"},
			},
			want: []string{"B", "C"},
		},
		{
			name:  "stale historical child of fresh current upstream is frontier",
			stale: []string{"C"},
			upstreams: map[string][]string{
				"C": {"B"},
			},
			want: []string{"C"},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := StaleFrontier(test.stale, test.upstreams); !reflect.DeepEqual(got, test.want) {
				t.Fatalf("StaleFrontier() = %v, want %v", got, test.want)
			}
		})
	}
}
