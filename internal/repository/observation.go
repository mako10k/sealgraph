package repository

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/mako10k/sealgraph/internal/domain"
	"github.com/mako10k/sealgraph/internal/revision"
)

type headObservation struct {
	names []string
	heads map[string]domain.ObjectID
}

func (observation headObservation) revisionHeads() []revision.Head {
	result := make([]revision.Head, 0, len(observation.names))
	for _, ref := range observation.names {
		result = append(result, revision.Head{REF: ref, Seal: observation.heads[ref]})
	}
	return result
}

func (observation headObservation) digest() string {
	hash := sha256.New()
	hash.Write([]byte("sealgraph/ref-head-observation/v1\x00format=4\x00"))
	for _, ref := range observation.names {
		hash.Write([]byte(ref))
		hash.Write([]byte{0})
		hash.Write([]byte(observation.heads[ref].String()))
		hash.Write([]byte{'\n'})
	}
	return fmt.Sprintf("%x", hash.Sum(nil))
}

func (r *Repository) observeHeads(ctx context.Context, operation string) (headObservation, error) {
	names, err := r.refs.List(ctx)
	if err != nil {
		return headObservation{}, fmt.Errorf("list current REFs for %s observation: %w", operation, err)
	}
	sort.Strings(names)
	observation := headObservation{names: append([]string(nil), names...), heads: make(map[string]domain.ObjectID, len(names))}
	for _, ref := range names {
		head, err := r.refs.Resolve(ctx, ref)
		if err != nil {
			return headObservation{}, fmt.Errorf("read current REF %s for %s observation: %w", ref, operation, err)
		}
		observation.heads[ref] = head
	}
	return observation, nil
}

func (r *Repository) revalidateHeads(ctx context.Context, observation headObservation, operation string) error {
	names, err := r.refs.List(ctx)
	if err != nil {
		return fmt.Errorf("REF heads changed or became unreadable while deriving %s: %w; rerun the command", operation, err)
	}
	sort.Strings(names)
	if !equalStrings(names, observation.names) {
		return fmt.Errorf("REF heads changed while deriving %s; rerun the command", operation)
	}
	for _, ref := range names {
		head, err := r.refs.Resolve(ctx, ref)
		if err != nil || !head.Equal(observation.heads[ref]) {
			return fmt.Errorf("REF %s changed or became unreadable while deriving %s; rerun the command", ref, operation)
		}
	}
	return nil
}

func equalStrings(left, right []string) bool {
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

func parseObservationDigest(value string) error {
	if len(value) != 64 || strings.Trim(value, "0123456789abcdef") != "" {
		return errors.New("observation digest is not 64 lower-case hexadecimal characters")
	}
	return nil
}
