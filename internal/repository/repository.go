// Package repository coordinates standalone domain semantics, object storage,
// REF updates, candidates, and derived graph inspection.
package repository

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/mako10k/sealgraph/internal/canonical"
	"github.com/mako10k/sealgraph/internal/domain"
	"github.com/mako10k/sealgraph/internal/graph"
	"github.com/mako10k/sealgraph/internal/store"
	"github.com/mako10k/sealgraph/internal/store/native"
)

type Clock func() time.Time

type Repository struct {
	dir        string
	objects    *native.ObjectStore
	refs       *native.RefStore
	candidates candidateStore
	clock      Clock
}

func OpenStandalone(workDir string, clock Clock) (*Repository, error) {
	dir := filepath.Join(workDir, ".sealgraph")
	if err := validateLayout(dir); err != nil {
		return nil, fmt.Errorf("open standalone repository %s: %w; run 'sealgraph init' in this directory or repair it explicitly", dir, err)
	}
	if clock == nil {
		clock = time.Now
	}
	return &Repository{
		dir:        dir,
		objects:    native.NewObjectStore(dir),
		refs:       native.NewRefStore(dir),
		candidates: candidateStore{root: filepath.Join(dir, "index")},
		clock:      clock,
	}, nil
}

type Dependency struct {
	REF  string
	Seal *domain.ObjectID
}

type AddOptions struct {
	REF          string
	Content      []byte
	Dependencies []Dependency
	Root         bool
	Draft        bool
}

func (r *Repository) Add(ctx context.Context, options AddOptions) (domain.Candidate, error) {
	if err := domain.ValidateREF(options.REF); err != nil {
		return domain.Candidate{}, err
	}
	candidate, err := r.candidateForEdit(ctx, options.REF)
	if err != nil {
		return domain.Candidate{}, err
	}
	contentID, err := r.objects.WriteBlob(ctx, options.Content)
	if err != nil {
		return domain.Candidate{}, fmt.Errorf("store content for %s: %w", options.REF, err)
	}
	candidate.Content = domain.ContentRef{Store: domain.NativeStore, Type: domain.BlobType, ID: contentID}
	candidate.Root = options.Root
	candidate.Draft = options.Draft
	if options.Dependencies != nil {
		links, err := r.resolveDependencies(ctx, options.Dependencies)
		if err != nil {
			return domain.Candidate{}, err
		}
		candidate.Links = links
	}
	if err := r.candidates.Save(candidate); err != nil {
		return domain.Candidate{}, fmt.Errorf("save candidate %s: %w", options.REF, err)
	}
	return candidate, nil
}

func (r *Repository) Link(ctx context.Context, ref string, dependencies []Dependency) (domain.Candidate, error) {
	if len(dependencies) == 0 {
		return domain.Candidate{}, errors.New("at least one --depend-on target is required")
	}
	candidate, err := r.candidateForEdit(ctx, ref)
	if err != nil {
		return domain.Candidate{}, err
	}
	if candidate.Content.ID.Hex == "" {
		return domain.Candidate{}, fmt.Errorf("REF %s has no content candidate; run 'sealgraph add %s --content ...' first", ref, ref)
	}
	links, err := r.resolveDependencies(ctx, dependencies)
	if err != nil {
		return domain.Candidate{}, err
	}
	byREF := make(map[string]domain.Link, len(candidate.Links)+len(links))
	for _, link := range candidate.Links {
		byREF[link.TargetREF] = link
	}
	for _, link := range links {
		byREF[link.TargetREF] = link
	}
	candidate.Links = candidate.Links[:0]
	for _, link := range byREF {
		candidate.Links = append(candidate.Links, link)
	}
	if err := r.candidates.Save(candidate); err != nil {
		return domain.Candidate{}, fmt.Errorf("save candidate %s: %w", ref, err)
	}
	return candidate, nil
}

func (r *Repository) candidateForEdit(ctx context.Context, ref string) (domain.Candidate, error) {
	if err := domain.ValidateREF(ref); err != nil {
		return domain.Candidate{}, err
	}
	if candidate, err := r.candidates.Load(ref); err == nil {
		return candidate, nil
	} else if !errors.Is(err, ErrCandidateNotFound) {
		return domain.Candidate{}, err
	}
	head, err := r.refs.Resolve(ctx, ref)
	if errors.Is(err, store.ErrRefNotFound) {
		return domain.Candidate{Schema: domain.CandidateSchema, REF: ref, Attachments: []domain.Attachment{}, Links: []domain.Link{}}, nil
	}
	if err != nil {
		return domain.Candidate{}, err
	}
	seal, err := r.LoadSeal(ctx, head)
	if err != nil {
		return domain.Candidate{}, fmt.Errorf("load current seal for %s: %w", ref, err)
	}
	if seal.REF != ref {
		return domain.Candidate{}, fmt.Errorf("REF %s points to seal for %s; repair the REF explicitly", ref, seal.REF)
	}
	base := head
	return domain.Candidate{
		Schema: domain.CandidateSchema, REF: ref, Base: &base, Content: seal.Content,
		Attachments: append([]domain.Attachment(nil), seal.Attachments...),
		Links:       append([]domain.Link(nil), seal.Links...), Root: seal.Root, Draft: seal.Draft,
	}, nil
}

func (r *Repository) resolveDependencies(ctx context.Context, dependencies []Dependency) ([]domain.Link, error) {
	links := make([]domain.Link, 0, len(dependencies))
	for _, dependency := range dependencies {
		if err := domain.ValidateREF(dependency.REF); err != nil {
			return nil, fmt.Errorf("invalid dependency REF %q: %w", dependency.REF, err)
		}
		var id domain.ObjectID
		if dependency.Seal == nil {
			resolved, err := r.refs.Resolve(ctx, dependency.REF)
			if err != nil {
				return nil, fmt.Errorf("resolve dependency %s HEAD: %w; seal that REF first or select an existing REF", dependency.REF, err)
			}
			id = resolved
		} else {
			id = *dependency.Seal
			if err := id.ValidateNative(); err != nil {
				return nil, fmt.Errorf("dependency %s has invalid seal ID: %w", dependency.REF, err)
			}
			if _, err := r.refs.Resolve(ctx, dependency.REF); err != nil {
				return nil, fmt.Errorf("explicit historical dependency %s has no readable current HEAD: %w; restore the REF or select another dependency", dependency.REF, err)
			}
		}
		seal, err := r.LoadSeal(ctx, id)
		if err != nil {
			return nil, fmt.Errorf("read dependency %s@%s: %w", dependency.REF, id, err)
		}
		if seal.REF != dependency.REF {
			return nil, fmt.Errorf("seal %s belongs to REF %s, not requested dependency %s", id, seal.REF, dependency.REF)
		}
		links = append(links, domain.Link{Relation: domain.DependOn, TargetREF: dependency.REF, TargetSeal: id})
	}
	normalized, err := domain.NormalizeLinks(links)
	if err != nil {
		return nil, err
	}
	return normalized, nil
}

type SealResult struct {
	ID      domain.ObjectID
	Payload domain.SealPayload
}

func (r *Repository) Seal(ctx context.Context, ref, message string) (SealResult, error) {
	if err := domain.ValidateREF(ref); err != nil {
		return SealResult{}, err
	}
	if message == "" {
		return SealResult{}, errors.New("seal message is required; pass -m MESSAGE")
	}
	candidate, err := r.candidates.Load(ref)
	if err != nil {
		if errors.Is(err, ErrCandidateNotFound) {
			return SealResult{}, fmt.Errorf("REF %s has no working candidate; run 'sealgraph add' or 'sealgraph link' first", ref)
		}
		return SealResult{}, err
	}
	current, currentErr := r.refs.Resolve(ctx, ref)
	if candidate.Base == nil {
		if currentErr == nil {
			return SealResult{}, fmt.Errorf("REF %s advanced to %s after candidate creation; recreate the candidate before sealing", ref, current)
		}
		if !errors.Is(currentErr, store.ErrRefNotFound) {
			return SealResult{}, currentErr
		}
	} else {
		if currentErr != nil || !current.Equal(*candidate.Base) {
			return SealResult{}, fmt.Errorf("REF %s HEAD changed after candidate creation; recreate the candidate from the current head before sealing", ref)
		}
	}
	if _, err := r.objects.ReadObject(ctx, candidate.Content.ID); err != nil {
		return SealResult{}, fmt.Errorf("candidate content for %s is unreadable: %w", ref, err)
	}
	if candidate.Root && len(candidate.Links) != 0 {
		return SealResult{}, fmt.Errorf("root REF %s cannot have dependencies; remove them explicitly before sealing", ref)
	}
	if !candidate.Root && len(candidate.Links) == 0 {
		return SealResult{}, fmt.Errorf("non-root REF %s requires at least one dependency; use 'sealgraph link' or recreate it with --root", ref)
	}
	for _, link := range candidate.Links {
		if link.TargetREF == ref {
			return SealResult{}, fmt.Errorf("REF %s cannot depend on itself; remove the link explicitly", ref)
		}
		seal, err := r.LoadSeal(ctx, link.TargetSeal)
		if err != nil {
			return SealResult{}, fmt.Errorf("dependency %s@%s is unreadable: %w", link.TargetREF, link.TargetSeal, err)
		}
		if seal.REF != link.TargetREF {
			return SealResult{}, fmt.Errorf("dependency target %s belongs to REF %s, not %s", link.TargetSeal, seal.REF, link.TargetREF)
		}
	}
	if !candidate.Draft {
		if err := r.requireHeadConsistentClosure(ctx, candidate.Links); err != nil {
			return SealResult{}, err
		}
	}
	createdAt := r.clock().UTC().Truncate(time.Second).Format("2006-01-02T15:04:05Z")
	payload := domain.SealPayload{
		Schema: domain.SealSchema, REF: ref, Parent: candidate.Base, Content: candidate.Content,
		Attachments: candidate.Attachments, Links: candidate.Links, Message: message,
		Root: candidate.Root, Draft: candidate.Draft, CreatedAt: createdAt,
	}
	bytes, err := canonical.EncodeSeal(payload)
	if err != nil {
		return SealResult{}, fmt.Errorf("canonicalize seal %s: %w", ref, err)
	}
	sealID, err := r.objects.WriteBlob(ctx, bytes)
	if err != nil {
		return SealResult{}, fmt.Errorf("store immutable seal %s: %w", ref, err)
	}
	if err := r.refs.Update(ctx, ref, candidate.Base, &sealID); err != nil {
		return SealResult{}, fmt.Errorf("seal object %s was written but REF %s was not advanced: %w", sealID, ref, err)
	}
	if err := r.candidates.Remove(ref); err != nil {
		return SealResult{}, fmt.Errorf("REF %s advanced to seal %s, but its candidate could not be cleared: %w", ref, sealID, err)
	}
	payload, _ = domain.NormalizeSeal(payload)
	return SealResult{ID: sealID, Payload: payload}, nil
}

func (r *Repository) requireHeadConsistentClosure(ctx context.Context, links []domain.Link) error {
	visited := make(map[string]bool)
	var visit func([]domain.Link) error
	visit = func(current []domain.Link) error {
		for _, link := range current {
			head, err := r.refs.Resolve(ctx, link.TargetREF)
			if err != nil {
				return fmt.Errorf("normal seal requires HEAD-consistent dependency closure, but %s HEAD cannot be resolved: %w; use a draft historical workflow or repair the REF explicitly", link.TargetREF, err)
			}
			if !head.Equal(link.TargetSeal) {
				return fmt.Errorf("normal seal requires HEAD-consistent dependency closure: %s targets %s but HEAD is %s; relink it or mark the candidate draft", link.TargetREF, link.TargetSeal, head)
			}
			if visited[link.TargetSeal.String()] {
				continue
			}
			visited[link.TargetSeal.String()] = true
			seal, err := r.LoadSeal(ctx, link.TargetSeal)
			if err != nil {
				return fmt.Errorf("traverse dependency %s@%s: %w", link.TargetREF, link.TargetSeal, err)
			}
			if seal.REF != link.TargetREF {
				return fmt.Errorf("dependency seal %s belongs to %s, not %s", link.TargetSeal, seal.REF, link.TargetREF)
			}
			if err := visit(seal.Links); err != nil {
				return err
			}
		}
		return nil
	}
	return visit(links)
}

func (r *Repository) LoadSeal(ctx context.Context, id domain.ObjectID) (domain.SealPayload, error) {
	object, err := r.objects.ReadObject(ctx, id)
	if err != nil {
		return domain.SealPayload{}, err
	}
	if object.Type != domain.BlobType {
		return domain.SealPayload{}, fmt.Errorf("seal object %s has type %s, expected blob", id, object.Type)
	}
	payload, err := canonical.DecodeSeal(object.Data)
	if err != nil {
		return domain.SealPayload{}, fmt.Errorf("object %s is not a valid canonical seal: %w", id, err)
	}
	return payload, nil
}

type ShowResult struct {
	ID      domain.ObjectID
	Payload domain.SealPayload
	Content []byte
}

func (r *Repository) Show(ctx context.Context, ref string, explicit *domain.ObjectID) (ShowResult, error) {
	if err := domain.ValidateREF(ref); err != nil {
		return ShowResult{}, err
	}
	id := domain.ObjectID{}
	if explicit == nil {
		resolved, err := r.refs.Resolve(ctx, ref)
		if err != nil {
			return ShowResult{}, fmt.Errorf("resolve REF %s: %w", ref, err)
		}
		id = resolved
	} else {
		id = *explicit
	}
	payload, err := r.LoadSeal(ctx, id)
	if err != nil {
		return ShowResult{}, err
	}
	if payload.REF != ref {
		return ShowResult{}, fmt.Errorf("seal %s belongs to REF %s, not %s", id, payload.REF, ref)
	}
	content, err := r.objects.ReadObject(ctx, payload.Content.ID)
	if err != nil {
		return ShowResult{}, fmt.Errorf("read content %s for seal %s: %w", payload.Content.ID, id, err)
	}
	return ShowResult{ID: id, Payload: payload, Content: content.Data}, nil
}

type RefStatus struct {
	REF         string
	Head        *domain.ObjectID
	Unsealed    bool
	Draft       bool
	StaleDirect []graph.DirectStaleLink
}

func (s RefStatus) Labels() []string {
	var labels []string
	if s.Unsealed {
		labels = append(labels, "UNSEALED")
	}
	if s.Draft {
		labels = append(labels, "DRAFT")
	}
	if len(s.StaleDirect) > 0 {
		labels = append(labels, "STALE_DIRECT")
	}
	if len(labels) == 0 {
		labels = append(labels, "CLEAN")
	}
	return labels
}

func (r *Repository) Status(ctx context.Context, onlyREF string) ([]RefStatus, error) {
	refs, err := r.refs.List(ctx)
	if err != nil {
		return nil, err
	}
	candidates, err := r.candidates.List()
	if err != nil {
		return nil, err
	}
	all := make(map[string]bool, len(refs)+len(candidates))
	for _, ref := range refs {
		all[ref] = true
	}
	for _, ref := range candidates {
		all[ref] = true
	}
	if onlyREF != "" {
		if err := domain.ValidateREF(onlyREF); err != nil {
			return nil, err
		}
		if !all[onlyREF] {
			return nil, fmt.Errorf("REF %s has no head or candidate", onlyREF)
		}
		all = map[string]bool{onlyREF: true}
	}
	names := make([]string, 0, len(all))
	for ref := range all {
		names = append(names, ref)
	}
	sort.Strings(names)
	result := make([]RefStatus, 0, len(names))
	for _, ref := range names {
		status := RefStatus{REF: ref}
		if candidate, err := r.candidates.Load(ref); err == nil {
			status.Unsealed = true
			status.Draft = candidate.Draft
		} else if !errors.Is(err, ErrCandidateNotFound) {
			return nil, err
		}
		head, err := r.refs.Resolve(ctx, ref)
		if err == nil {
			headCopy := head
			status.Head = &headCopy
			payload, err := r.LoadSeal(ctx, head)
			if err != nil {
				return nil, fmt.Errorf("read current seal for %s: %w", ref, err)
			}
			if payload.REF != ref {
				return nil, fmt.Errorf("REF %s points to a seal for %s", ref, payload.REF)
			}
			status.Draft = status.Draft || payload.Draft
			status.StaleDirect, err = graph.DirectStale(ctx, payload, r.refs)
			if err != nil {
				return nil, err
			}
		} else if !errors.Is(err, store.ErrRefNotFound) {
			return nil, err
		}
		result = append(result, status)
	}
	return result, nil
}

func ParseSelector(text string) (string, *domain.ObjectID, error) {
	if index := strings.LastIndexByte(text, '@'); index > 0 && index < len(text)-1 {
		if id, err := domain.ParseObjectID(text[index+1:]); err == nil {
			ref := text[:index]
			if err := domain.ValidateREF(ref); err != nil {
				return "", nil, err
			}
			return ref, &id, nil
		}
	}
	if err := domain.ValidateREF(text); err != nil {
		return "", nil, err
	}
	return text, nil, nil
}
