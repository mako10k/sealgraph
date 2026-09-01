package repository

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/mako10k/sealgraph/internal/domain"
)

type headObservation struct {
	names                   []string
	heads                   map[string]domain.ObjectID
	manifests               map[string][]byte
	tags                    map[string]map[string]domain.ObjectID
	objectIDs               []string
	objectInventoryCaptured bool
}

func (r *Repository) observeHeads(ctx context.Context, operation string) (headObservation, error) {
	names, err := r.refs.List(ctx)
	if err != nil {
		return headObservation{}, fmt.Errorf("list current REFs for %s observation: %w", operation, err)
	}
	sort.Strings(names)
	recoveryRefs, err := r.recoveryRefs()
	if err != nil {
		return headObservation{}, err
	}
	observation := headObservation{
		names: append([]string(nil), names...), heads: make(map[string]domain.ObjectID, len(names)),
		manifests: make(map[string][]byte, len(names)), tags: make(map[string]map[string]domain.ObjectID, len(names)),
	}
	for _, ref := range names {
		manifest, err := recoveryRefs.Snapshot(ctx, ref)
		if err != nil {
			return headObservation{}, fmt.Errorf("capture current REF %s manifest for %s observation: %w", ref, operation, err)
		}
		observation.manifests[ref] = manifest
		targets, err := recoveryRefs.ManifestTargets(manifest)
		if err != nil {
			return headObservation{}, fmt.Errorf("decode captured REF %s head for %s observation: %w", ref, operation, err)
		}
		if len(targets) == 0 {
			return headObservation{}, fmt.Errorf("captured REF %s manifest has no head", ref)
		}
		observation.heads[ref] = targets[0]
		manifestTags, err := recoveryRefs.ManifestTags(manifest)
		if err != nil {
			return headObservation{}, fmt.Errorf("decode captured REF %s tags for %s observation: %w", ref, operation, err)
		}
		observation.tags[ref] = make(map[string]domain.ObjectID, len(manifestTags))
		for _, tag := range manifestTags {
			observation.tags[ref][tag.Name] = tag.Seal
		}
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
	recoveryRefs, err := r.recoveryRefs()
	if err != nil {
		return err
	}
	for _, ref := range names {
		head, err := r.refs.Resolve(ctx, ref)
		manifest, manifestErr := recoveryRefs.Snapshot(ctx, ref)
		if err != nil || manifestErr != nil || !head.Equal(observation.heads[ref]) || !bytes.Equal(manifest, observation.manifests[ref]) {
			return fmt.Errorf("REF %s changed or became unreadable while deriving %s; rerun the command", ref, operation)
		}
	}
	if observation.objectInventoryCaptured {
		ids, err := r.captureObjectInventory(ctx, operation)
		if err != nil || !equalStrings(ids, observation.objectIDs) {
			return fmt.Errorf("loose-object inventory changed or became unreadable while deriving %s; rerun the command", operation)
		}
	}
	return nil
}

func (r *Repository) ensureObjectInventory(ctx context.Context, observation *headObservation, operation string) error {
	if observation.objectInventoryCaptured {
		return nil
	}
	ids, err := r.captureObjectInventory(ctx, operation)
	if err != nil {
		return err
	}
	observation.objectIDs = ids
	observation.objectInventoryCaptured = true
	return nil
}

func (r *Repository) accountForObservedObjectWrite(ctx context.Context, observation *headObservation, id domain.ObjectID, operation string) error {
	if !observation.objectInventoryCaptured {
		return nil
	}
	expected := append([]string(nil), observation.objectIDs...)
	found := false
	for _, existing := range expected {
		if existing == id.String() {
			found = true
			break
		}
	}
	if !found {
		expected = append(expected, id.String())
		sort.Strings(expected)
	}
	current, err := r.captureObjectInventory(ctx, operation)
	if err != nil || !equalStrings(current, expected) {
		return fmt.Errorf("loose-object inventory changed unexpectedly while deriving %s; rerun the command", operation)
	}
	observation.objectIDs = current
	return nil
}

func (r *Repository) captureObjectInventory(ctx context.Context, operation string) ([]string, error) {
	objects, err := r.objects.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("capture loose-object inventory for %s: %w", operation, err)
	}
	ids := make([]string, len(objects))
	for i, object := range objects {
		ids[i] = object.ID.String()
	}
	sort.Strings(ids)
	return ids, nil
}

func resolveObservedObjectPrefix(prefix string, ids []string) (domain.ObjectID, error) {
	var match string
	for _, id := range ids {
		if !strings.HasPrefix(id, prefix) {
			continue
		}
		if match != "" && match != id {
			return domain.ObjectID{}, fmt.Errorf("ambiguous object prefix %q; use more hexadecimal characters", prefix)
		}
		match = id
	}
	if match == "" {
		return domain.ObjectID{}, fmt.Errorf("object not found: prefix %s", prefix)
	}
	return domain.ObjectID{Hex: match}, nil
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
