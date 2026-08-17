package repository

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/mako10k/sealgraph/internal/canonical"
	"github.com/mako10k/sealgraph/internal/domain"
	"github.com/mako10k/sealgraph/internal/migration"
)

const logicalLoadReceiptV1Schema = "sealgraph/logical-load-receipt/v1"

type sealMapping struct {
	Old domain.ObjectID
	New domain.ObjectID
}

type collapseGroup struct {
	New domain.ObjectID
	Old []domain.ObjectID
}

// LoadLogicalV1 rebuilds one complete format-4 repository from the versioned
// migration document. It never opens a format-3 repository and publishes only
// to an absent target via an atomic no-replace directory rename.
func LoadLogicalV1(ctx context.Context, workDir string, input []byte) ([]byte, error) {
	target := filepath.Join(workDir, ".sealgraph")
	if _, err := os.Lstat(target); err == nil {
		return nil, fmt.Errorf("%s already exists; load requires an absent target and never merges or replaces a repository", target)
	} else if !os.IsNotExist(err) {
		return nil, fmt.Errorf("inspect load target %s: %w", target, err)
	}
	if err := rejectAbandonedLoadStaging(workDir); err != nil {
		return nil, err
	}
	dump, err := migration.DecodeLogicalDumpV1(input)
	if err != nil {
		return nil, err
	}
	if len(dump.Tags) != 0 {
		return nil, fmt.Errorf("logical dump contains %d tag record(s): %w; no tag was dropped or deferred", len(dump.Tags), ErrTagContractPending)
	}

	staging, err := os.MkdirTemp(workDir, ".sealgraph-load-")
	if err != nil {
		return nil, fmt.Errorf("create format-4 load staging directory: %w", err)
	}
	published := false
	defer func() {
		if !published {
			_ = os.RemoveAll(staging)
		}
	}()
	if err := prepareRepositoryLayout(staging); err != nil {
		return nil, err
	}
	repo := newRepository(staging)
	for _, object := range dump.Objects {
		id, err := repo.objects.WriteBlob(ctx, object.Data)
		if err != nil {
			return nil, fmt.Errorf("write migrated material object %s: %w", object.ID, err)
		}
		if !id.Equal(object.ID) {
			return nil, fmt.Errorf("migrated material object %s rewrote as %s", object.ID, id)
		}
	}

	oldToNew := make(map[string]domain.ObjectID, len(dump.Seals))
	mappings := make([]sealMapping, 0, len(dump.Seals))
	for _, oldSeal := range dump.Seals {
		var parent *domain.ObjectID
		if oldSeal.Payload.Parent != nil {
			mapped, ok := oldToNew[oldSeal.Payload.Parent.String()]
			if !ok {
				return nil, fmt.Errorf("old seal %s parent %s has no earlier mapping", oldSeal.OldSealID, oldSeal.Payload.Parent)
			}
			copy := mapped
			parent = &copy
		}
		links := make([]domain.Link, len(oldSeal.Payload.Links))
		for i, oldLink := range oldSeal.Payload.Links {
			mapped, ok := oldToNew[oldLink.TargetSeal.String()]
			if !ok {
				return nil, fmt.Errorf("old seal %s Cause target %s has no earlier mapping", oldSeal.OldSealID, oldLink.TargetSeal)
			}
			links[i] = domain.Link{TargetSeal: mapped, Message: oldLink.Message}
		}
		payload := domain.SealPayload{
			Schema: domain.SealSchema, ParentRevision: parent,
			Content: oldSeal.Payload.Content, Attachments: oldSeal.Payload.Attachments,
			Links: links, Root: oldSeal.Payload.Root, Draft: oldSeal.Payload.Draft,
		}
		encoded, err := canonical.EncodeSeal(payload)
		if err != nil {
			return nil, fmt.Errorf("rewrite old seal %s as format 4: %w", oldSeal.OldSealID, err)
		}
		newID, err := repo.objects.WriteBlob(ctx, encoded)
		if err != nil {
			return nil, fmt.Errorf("write rewritten seal for %s: %w", oldSeal.OldSealID, err)
		}
		oldToNew[oldSeal.OldSealID.String()] = newID
		mappings = append(mappings, sealMapping{Old: oldSeal.OldSealID, New: newID})
	}
	for _, ref := range dump.REFs {
		head, ok := oldToNew[ref.Head.String()]
		if !ok {
			return nil, fmt.Errorf("REF %s old head %s has no mapping", ref.Name, ref.Head)
		}
		if err := repo.refs.Update(ctx, ref.Name, nil, &head); err != nil {
			return nil, fmt.Errorf("publish staged REF %s: %w", ref.Name, err)
		}
	}
	// REF publication uses transient lock directories inside staging. They are
	// not migrated state; publish one empty runtime lock directory instead.
	if err := os.RemoveAll(filepath.Join(staging, "locks")); err != nil {
		return nil, fmt.Errorf("clear staged transient locks: %w", err)
	}
	if err := os.Mkdir(filepath.Join(staging, "locks"), 0o755); err != nil {
		return nil, fmt.Errorf("recreate empty staged locks directory: %w", err)
	}
	if err := validateLoadedRepository(ctx, repo, dump.REFs, oldToNew); err != nil {
		return nil, fmt.Errorf("validate staged format-4 repository: %w", err)
	}
	receipt := encodeLoadReceipt(input, mappings, dump.REFs, oldToNew)
	if err := renameNoReplace(staging, target); err != nil {
		return nil, fmt.Errorf("publish complete format-4 repository without replacement: %w", err)
	}
	published = true
	return receipt, nil
}

func rejectAbandonedLoadStaging(workDir string) error {
	entries, err := os.ReadDir(workDir)
	if err != nil {
		return fmt.Errorf("inspect load staging parent %s: %w", workDir, err)
	}
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".sealgraph-load-") {
			return fmt.Errorf("found prior load staging path %s; inspect and remove it explicitly before retrying", filepath.Join(workDir, entry.Name()))
		}
	}
	return nil
}

func prepareRepositoryLayout(dir string) error {
	for _, relative := range []string{"objects", filepath.Join("refs", "seals"), filepath.Join("refs", "tags"), "index", "locks"} {
		if err := os.MkdirAll(filepath.Join(dir, relative), 0o755); err != nil {
			return fmt.Errorf("prepare format-4 repository layout: %w", err)
		}
	}
	if err := os.WriteFile(filepath.Join(dir, "config"), []byte(configBytes), 0o644); err != nil {
		return fmt.Errorf("write format-4 repository config: %w", err)
	}
	return nil
}

func validateLoadedRepository(ctx context.Context, repo *Repository, expected []migration.RefRecord, mapping map[string]domain.ObjectID) error {
	if err := validateLayout(repo.dir); err != nil {
		return err
	}
	names, err := repo.refs.List(ctx)
	if err != nil {
		return err
	}
	if len(names) != len(expected) {
		return fmt.Errorf("staged REF count is %d, expected %d", len(names), len(expected))
	}
	for i, record := range expected {
		if names[i] != record.Name {
			return fmt.Errorf("staged REF %d is %q, expected %q", i, names[i], record.Name)
		}
		head, err := repo.refs.Resolve(ctx, record.Name)
		if err != nil {
			return err
		}
		if !head.Equal(mapping[record.Head.String()]) {
			return fmt.Errorf("staged REF %s target changed during validation", record.Name)
		}
	}
	objects, err := repo.objects.List(ctx)
	if err != nil {
		return err
	}
	objectSet := make(map[string]struct{}, len(objects))
	for _, object := range objects {
		objectSet[object.ID.String()] = struct{}{}
	}
	state := make(map[string]uint8)
	var visit func(domain.ObjectID) error
	visit = func(id domain.ObjectID) error {
		switch state[id.String()] {
		case 1:
			return fmt.Errorf("combined revision/Cause rebuild cycle reaches %s", id)
		case 2:
			return nil
		}
		state[id.String()] = 1
		payload, err := repo.LoadSeal(ctx, id)
		if err != nil {
			return err
		}
		if _, ok := objectSet[payload.Content.ID.String()]; !ok {
			return fmt.Errorf("seal %s content %s is absent", id, payload.Content.ID)
		}
		for _, attachment := range payload.Attachments {
			if _, ok := objectSet[attachment.Blob.ID.String()]; !ok {
				return fmt.Errorf("seal %s attachment %q object %s is absent", id, attachment.Name, attachment.Blob.ID)
			}
		}
		if payload.ParentRevision != nil {
			if err := visit(*payload.ParentRevision); err != nil {
				return err
			}
		}
		for _, link := range payload.Links {
			if err := visit(link.TargetSeal); err != nil {
				return err
			}
		}
		state[id.String()] = 2
		return nil
	}
	for _, name := range names {
		head, _ := repo.refs.Resolve(ctx, name)
		if err := visit(head); err != nil {
			return fmt.Errorf("REF %s: %w", name, err)
		}
	}
	return nil
}

func encodeLoadReceipt(input []byte, mappings []sealMapping, refs []migration.RefRecord, oldToNew map[string]domain.ObjectID) []byte {
	sort.Slice(mappings, func(i, j int) bool { return mappings[i].Old.String() < mappings[j].Old.String() })
	byNew := make(map[string][]domain.ObjectID)
	for _, mapping := range mappings {
		byNew[mapping.New.String()] = append(byNew[mapping.New.String()], mapping.Old)
	}
	var collapsed []collapseGroup
	for newID, oldIDs := range byNew {
		if len(oldIDs) < 2 {
			continue
		}
		sort.Slice(oldIDs, func(i, j int) bool { return oldIDs[i].String() < oldIDs[j].String() })
		collapsed = append(collapsed, collapseGroup{New: domain.ObjectID{Hex: newID}, Old: oldIDs})
	}
	sort.Slice(collapsed, func(i, j int) bool { return collapsed[i].New.String() < collapsed[j].New.String() })
	digest := sha256.Sum256(input)
	b := make([]byte, 0, 1024)
	b = append(b, `{"schema":`...)
	b, _ = canonical.AppendString(b, logicalLoadReceiptV1Schema)
	b = append(b, `,"source_dump_sha256":`...)
	b, _ = canonical.AppendString(b, fmt.Sprintf("%x", digest))
	b = append(b, `,"old_to_new_seals":[`...)
	for i, mapping := range mappings {
		if i > 0 {
			b = append(b, ',')
		}
		b = append(b, `{"old_seal_id":`...)
		b, _ = canonical.AppendString(b, mapping.Old.String())
		b = append(b, `,"new_seal_id":`...)
		b, _ = canonical.AppendString(b, mapping.New.String())
		b = append(b, '}')
	}
	b = append(b, `],"collapsed_seals":[`...)
	for i, group := range collapsed {
		if i > 0 {
			b = append(b, ',')
		}
		b = append(b, `{"new_seal_id":`...)
		b, _ = canonical.AppendString(b, group.New.String())
		b = append(b, `,"old_seal_ids":[`...)
		for j, oldID := range group.Old {
			if j > 0 {
				b = append(b, ',')
			}
			b, _ = canonical.AppendString(b, oldID.String())
		}
		b = append(b, ']', '}')
	}
	b = append(b, `],"refs":[`...)
	for i, ref := range refs {
		if i > 0 {
			b = append(b, ',')
		}
		b = append(b, `{"name":`...)
		b, _ = canonical.AppendString(b, ref.Name)
		b = append(b, `,"head":`...)
		b, _ = canonical.AppendString(b, oldToNew[ref.Head.String()].String())
		b = append(b, '}')
	}
	b = append(b, `],"tags":[],"published_repository_format":4}`...)
	b = append(b, '\n')
	return b
}
