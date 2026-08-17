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
	if err := preflightLoadTarget(workDir, target); err != nil {
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
	repo, mappings, oldToNew, err := buildStagedRepository(ctx, staging, dump)
	if err != nil {
		return nil, err
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

func preflightLoadTarget(workDir, target string) error {
	if _, err := os.Lstat(target); err == nil {
		return fmt.Errorf("%s already exists; load requires an absent target and never merges or replaces a repository", target)
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("inspect load target %s: %w", target, err)
	}
	return rejectAbandonedLoadStaging(workDir)
}

func buildStagedRepository(ctx context.Context, staging string, dump migration.LogicalDumpV1) (*Repository, []sealMapping, map[string]domain.ObjectID, error) {
	if err := prepareRepositoryLayout(staging); err != nil {
		return nil, nil, nil, err
	}
	repo := newRepository(staging)
	if err := writeMigratedObjects(ctx, repo, dump.Objects); err != nil {
		return nil, nil, nil, err
	}
	mappings, oldToNew, err := rewriteMigratedSeals(ctx, repo, dump.Seals)
	if err != nil {
		return nil, nil, nil, err
	}
	if err := publishMigratedREFs(ctx, repo, dump.REFs, oldToNew); err != nil {
		return nil, nil, nil, err
	}
	if err := resetStagedLocks(staging); err != nil {
		return nil, nil, nil, err
	}
	return repo, mappings, oldToNew, nil
}

func writeMigratedObjects(ctx context.Context, repo *Repository, objects []migration.ObjectRecord) error {
	for _, object := range objects {
		id, err := repo.objects.WriteBlob(ctx, object.Data)
		if err != nil {
			return fmt.Errorf("write migrated material object %s: %w", object.ID, err)
		}
		if !id.Equal(object.ID) {
			return fmt.Errorf("migrated material object %s rewrote as %s", object.ID, id)
		}
	}
	return nil
}

func rewriteMigratedSeals(ctx context.Context, repo *Repository, seals []migration.SealRecord) ([]sealMapping, map[string]domain.ObjectID, error) {
	oldToNew := make(map[string]domain.ObjectID, len(seals))
	mappings := make([]sealMapping, 0, len(seals))
	for _, oldSeal := range seals {
		payload, err := rewriteSealPayload(oldSeal, oldToNew)
		if err != nil {
			return nil, nil, err
		}
		encoded, err := canonical.EncodeSeal(payload)
		if err != nil {
			return nil, nil, fmt.Errorf("rewrite old seal %s as format 4: %w", oldSeal.OldSealID, err)
		}
		newID, err := repo.objects.WriteBlob(ctx, encoded)
		if err != nil {
			return nil, nil, fmt.Errorf("write rewritten seal for %s: %w", oldSeal.OldSealID, err)
		}
		oldToNew[oldSeal.OldSealID.String()] = newID
		mappings = append(mappings, sealMapping{Old: oldSeal.OldSealID, New: newID})
	}
	return mappings, oldToNew, nil
}

func rewriteSealPayload(oldSeal migration.SealRecord, oldToNew map[string]domain.ObjectID) (domain.SealPayload, error) {
	parent, err := rewriteOptionalSealID(oldSeal.Payload.Parent, oldToNew, "parent")
	if err != nil {
		return domain.SealPayload{}, fmt.Errorf("old seal %s: %w", oldSeal.OldSealID, err)
	}
	links := make([]domain.Link, len(oldSeal.Payload.Links))
	for i, oldLink := range oldSeal.Payload.Links {
		mapped, ok := oldToNew[oldLink.TargetSeal.String()]
		if !ok {
			return domain.SealPayload{}, fmt.Errorf("old seal %s Cause target %s has no earlier mapping", oldSeal.OldSealID, oldLink.TargetSeal)
		}
		links[i] = domain.Link{TargetSeal: mapped, Message: oldLink.Message}
	}
	return domain.SealPayload{
		Schema: domain.SealSchema, ParentRevision: parent,
		Content: oldSeal.Payload.Content, Attachments: oldSeal.Payload.Attachments,
		Links: links, Root: oldSeal.Payload.Root, Draft: oldSeal.Payload.Draft,
	}, nil
}

func rewriteOptionalSealID(old *domain.ObjectID, mapping map[string]domain.ObjectID, relation string) (*domain.ObjectID, error) {
	if old == nil {
		return nil, nil
	}
	mapped, ok := mapping[old.String()]
	if !ok {
		return nil, fmt.Errorf("%s %s has no earlier mapping", relation, old)
	}
	return &mapped, nil
}

func publishMigratedREFs(ctx context.Context, repo *Repository, refs []migration.RefRecord, mapping map[string]domain.ObjectID) error {
	for _, ref := range refs {
		head, ok := mapping[ref.Head.String()]
		if !ok {
			return fmt.Errorf("REF %s old head %s has no mapping", ref.Name, ref.Head)
		}
		if err := repo.refs.Update(ctx, ref.Name, nil, &head); err != nil {
			return fmt.Errorf("publish staged REF %s: %w", ref.Name, err)
		}
	}
	return nil
}

func resetStagedLocks(staging string) error {
	if err := os.RemoveAll(filepath.Join(staging, "locks")); err != nil {
		return fmt.Errorf("clear staged transient locks: %w", err)
	}
	if err := os.Mkdir(filepath.Join(staging, "locks"), 0o755); err != nil {
		return fmt.Errorf("recreate empty staged locks directory: %w", err)
	}
	return nil
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
	names, err := validateLoadedREFs(ctx, repo, expected, mapping)
	if err != nil {
		return err
	}
	objects, err := repo.objects.List(ctx)
	if err != nil {
		return err
	}
	objectSet := make(map[string]struct{}, len(objects))
	for _, object := range objects {
		objectSet[object.ID.String()] = struct{}{}
	}
	return validateLoadedGraph(ctx, repo, names, objectSet)
}

func validateLoadedREFs(ctx context.Context, repo *Repository, expected []migration.RefRecord, mapping map[string]domain.ObjectID) ([]string, error) {
	names, err := repo.refs.List(ctx)
	if err != nil {
		return nil, err
	}
	if len(names) != len(expected) {
		return nil, fmt.Errorf("staged REF count is %d, expected %d", len(names), len(expected))
	}
	for i, record := range expected {
		if names[i] != record.Name {
			return nil, fmt.Errorf("staged REF %d is %q, expected %q", i, names[i], record.Name)
		}
		head, err := repo.refs.Resolve(ctx, record.Name)
		if err != nil {
			return nil, err
		}
		if !head.Equal(mapping[record.Head.String()]) {
			return nil, fmt.Errorf("staged REF %s target changed during validation", record.Name)
		}
	}
	return names, nil
}

type loadedGraphValidator struct {
	ctx       context.Context
	repo      *Repository
	objectSet map[string]struct{}
	state     map[string]uint8
}

func validateLoadedGraph(ctx context.Context, repo *Repository, names []string, objectSet map[string]struct{}) error {
	validator := loadedGraphValidator{ctx: ctx, repo: repo, objectSet: objectSet, state: make(map[string]uint8)}
	for _, name := range names {
		head, _ := repo.refs.Resolve(ctx, name)
		if err := validator.visit(head); err != nil {
			return fmt.Errorf("REF %s: %w", name, err)
		}
	}
	return nil
}

func (validator *loadedGraphValidator) visit(id domain.ObjectID) error {
	switch validator.state[id.String()] {
	case 1:
		return fmt.Errorf("combined revision/Cause rebuild cycle reaches %s", id)
	case 2:
		return nil
	}
	validator.state[id.String()] = 1
	payload, err := validator.repo.LoadSeal(validator.ctx, id)
	if err != nil {
		return err
	}
	if err := validator.validateMaterial(id, payload); err != nil {
		return err
	}
	if payload.ParentRevision != nil {
		if err := validator.visit(*payload.ParentRevision); err != nil {
			return err
		}
	}
	for _, link := range payload.Links {
		if err := validator.visit(link.TargetSeal); err != nil {
			return err
		}
	}
	validator.state[id.String()] = 2
	return nil
}

func (validator *loadedGraphValidator) validateMaterial(id domain.ObjectID, payload domain.SealPayload) error {
	if _, ok := validator.objectSet[payload.Content.ID.String()]; !ok {
		return fmt.Errorf("seal %s content %s is absent", id, payload.Content.ID)
	}
	for _, attachment := range payload.Attachments {
		if _, ok := validator.objectSet[attachment.Blob.ID.String()]; !ok {
			return fmt.Errorf("seal %s attachment %q object %s is absent", id, attachment.Name, attachment.Blob.ID)
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
