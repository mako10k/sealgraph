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
	tags       *native.TagStore
	candidates candidateStore
	writer     writerGuard
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
		tags:       native.NewTagStore(dir),
		candidates: candidateStore{root: filepath.Join(dir, "index")},
		writer:     newWriterGuard(filepath.Join(dir, "locks")),
		clock:      clock,
	}, nil
}

type Dependency struct {
	REF      string
	Revision string
	Message  string
}

type AddOptions struct {
	REF          string
	Content      []byte
	Dependencies []Dependency
	Root         bool
	Draft        bool
}

func (r *Repository) Add(ctx context.Context, options AddOptions) (domain.Candidate, error) {
	return withMutation(ctx, r.writer, "add candidate", func() (domain.Candidate, error) {
		return r.add(ctx, options)
	})
}

func (r *Repository) add(ctx context.Context, options AddOptions) (domain.Candidate, error) {
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
	return withMutation(ctx, r.writer, "link candidate", func() (domain.Candidate, error) {
		return r.link(ctx, ref, dependencies)
	})
}

func (r *Repository) link(ctx context.Context, ref string, dependencies []Dependency) (domain.Candidate, error) {
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
		if dependency.Revision == "" {
			resolved, err := r.refs.Resolve(ctx, dependency.REF)
			if err != nil {
				return nil, fmt.Errorf("resolve dependency %s HEAD: %w; seal that REF first or select an existing REF", dependency.REF, err)
			}
			id = resolved
		} else {
			resolved, err := r.ResolveSealID(ctx, dependency.REF, dependency.Revision)
			if err != nil {
				return nil, fmt.Errorf("resolve dependency %s@%s: %w", dependency.REF, dependency.Revision, err)
			}
			id = resolved
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
		links = append(links, domain.Link{TargetREF: dependency.REF, TargetSeal: id, Message: dependency.Message})
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
	return withMutation(ctx, r.writer, "seal REF", func() (SealResult, error) {
		return r.seal(ctx, ref, message)
	})
}

func (r *Repository) seal(ctx context.Context, ref, message string) (SealResult, error) {
	if err := domain.ValidateREF(ref); err != nil {
		return SealResult{}, err
	}
	if message == "" {
		return SealResult{}, errors.New("seal message is required; pass -m MESSAGE")
	}
	snapshot, err := r.candidates.LoadSnapshot(ref)
	if err != nil {
		if errors.Is(err, ErrCandidateNotFound) {
			return SealResult{}, fmt.Errorf("REF %s has no working candidate; run 'sealgraph add' or 'sealgraph link' first", ref)
		}
		return SealResult{}, err
	}
	candidate := snapshot.Candidate
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
	payload, _ = domain.NormalizeSeal(payload)
	result := SealResult{ID: sealID, Payload: payload}
	if err := r.candidates.RemoveIfUnchanged(ref, snapshot.Bytes); err != nil {
		return result, fmt.Errorf("REF %s was published at seal %s, but its candidate was retained because cleanup could not prove it was unchanged: %w; inspect the candidate explicitly", ref, sealID, err)
	}
	return result, nil
}

func (r *Repository) requireHeadConsistentClosure(ctx context.Context, links []domain.Link) error {
	const (
		visiting = 1
		complete = 2
	)
	state := make(map[string]int)
	var path []graph.SealIdentity
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
			key := link.TargetSeal.String()
			if state[key] == visiting {
				cycleStart := 0
				for i, identity := range path {
					if identity.Seal.Equal(link.TargetSeal) {
						cycleStart = i
						break
					}
				}
				cycle := append([]graph.SealIdentity(nil), path[cycleStart:]...)
				cycle = append(cycle, graph.SealIdentity{REF: link.TargetREF, Seal: link.TargetSeal})
				return fmt.Errorf("normal seal dependency closure is invalid: %w; inspect the immutable objects and REF heads explicitly", &graph.CycleError{Path: cycle})
			}
			if state[key] == complete {
				continue
			}
			seal, err := r.LoadSeal(ctx, link.TargetSeal)
			if err != nil {
				return fmt.Errorf("traverse dependency %s@%s: %w", link.TargetREF, link.TargetSeal, err)
			}
			if seal.REF != link.TargetREF {
				return fmt.Errorf("dependency seal %s belongs to %s, not %s", link.TargetSeal, seal.REF, link.TargetREF)
			}
			if seal.Draft {
				return fmt.Errorf("normal seal requires a non-draft dependency closure, but %s@%s is draft; keep the candidate draft or supersede %s with a non-draft seal and relink explicitly", link.TargetREF, link.TargetSeal, link.TargetREF)
			}
			state[key] = visiting
			path = append(path, graph.SealIdentity{REF: link.TargetREF, Seal: link.TargetSeal})
			if err := visit(seal.Links); err != nil {
				return err
			}
			path = path[:len(path)-1]
			state[key] = complete
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

func (r *Repository) Show(ctx context.Context, ref, revision string) (ShowResult, error) {
	if err := domain.ValidateREF(ref); err != nil {
		return ShowResult{}, err
	}
	id, err := r.ResolveSealID(ctx, ref, revision)
	if err != nil {
		return ShowResult{}, err
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
	REF             string
	Head            *domain.ObjectID
	Unsealed        bool
	Draft           bool
	StaleDirect     []graph.DirectStaleLink
	StaleTransitive []graph.StalePath
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
	if len(s.StaleTransitive) > 0 {
		labels = append(labels, "STALE_TRANSITIVE")
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
			inspection, inspectErr := graph.Inspect(
				ctx,
				graph.SealIdentity{REF: ref, Seal: head},
				payload,
				r.refs,
				r.LoadSeal,
			)
			if inspectErr != nil {
				return nil, fmt.Errorf("inspect provenance graph for %s: %w; inspect the immutable objects and REF heads explicitly", ref, inspectErr)
			}
			status.StaleDirect = inspection.Direct
			status.StaleTransitive = inspection.Transitive
		} else if !errors.Is(err, store.ErrRefNotFound) {
			return nil, err
		}
		result = append(result, status)
	}
	return result, nil
}

// Stale returns only current REF statuses with derived direct or transitive
// staleness. Candidate-only UNSEALED state is intentionally excluded.
func (r *Repository) Stale(ctx context.Context) ([]RefStatus, error) {
	statuses, err := r.Status(ctx, "")
	if err != nil {
		return nil, err
	}
	result := make([]RefStatus, 0, len(statuses))
	for _, status := range statuses {
		if len(status.StaleDirect) > 0 || len(status.StaleTransitive) > 0 {
			result = append(result, status)
		}
	}
	return result, nil
}

type GraphLink struct {
	Link        domain.Link
	CurrentHead domain.ObjectID
}

type GraphNode struct {
	Status RefStatus
	Links  []GraphLink
}

// Graph returns current REF nodes and their concrete direct links. Historical
// seals remain reachable through the link IDs but are not promoted to REF
// heads by this read operation.
func (r *Repository) Graph(ctx context.Context) ([]GraphNode, error) {
	statuses, err := r.Status(ctx, "")
	if err != nil {
		return nil, err
	}
	nodes := make([]GraphNode, 0, len(statuses))
	for _, status := range statuses {
		node := GraphNode{Status: status}
		if status.Head != nil {
			payload, err := r.LoadSeal(ctx, *status.Head)
			if err != nil {
				return nil, fmt.Errorf("load current seal for graph node %s: %w", status.REF, err)
			}
			for _, link := range payload.Links {
				head, err := r.refs.Resolve(ctx, link.TargetREF)
				if err != nil {
					return nil, fmt.Errorf("resolve graph link %s: %w", link.TargetREF, err)
				}
				node.Links = append(node.Links, GraphLink{Link: link, CurrentHead: head})
			}
		}
		nodes = append(nodes, node)
	}
	return nodes, nil
}

// Impact derives all current downstream REFs whose exact immutable dependency
// closure names sourceREF. It returns the source's current head separately.
func (r *Repository) Impact(ctx context.Context, sourceREF string) (domain.ObjectID, []graph.Impact, error) {
	if err := domain.ValidateREF(sourceREF); err != nil {
		return domain.ObjectID{}, nil, err
	}
	sourceHead, err := r.refs.Resolve(ctx, sourceREF)
	if err != nil {
		return domain.ObjectID{}, nil, fmt.Errorf("resolve impact source %s: %w", sourceREF, err)
	}
	refs, err := r.refs.List(ctx)
	if err != nil {
		return domain.ObjectID{}, nil, err
	}
	current := make([]graph.SealIdentity, 0, len(refs))
	for _, ref := range refs {
		head, err := r.refs.Resolve(ctx, ref)
		if err != nil {
			return domain.ObjectID{}, nil, err
		}
		payload, err := r.LoadSeal(ctx, head)
		if err != nil {
			return domain.ObjectID{}, nil, fmt.Errorf("load current seal for %s: %w", ref, err)
		}
		identity := graph.SealIdentity{REF: ref, Seal: head}
		if _, err := graph.Inspect(ctx, identity, payload, r.refs, r.LoadSeal); err != nil {
			return domain.ObjectID{}, nil, fmt.Errorf("inspect provenance graph for %s: %w; inspect the immutable objects and REF heads explicitly", ref, err)
		}
		current = append(current, identity)
	}
	impacts, err := graph.ReverseImpact(ctx, sourceREF, current, r.LoadSeal)
	if err != nil {
		return domain.ObjectID{}, nil, fmt.Errorf("derive reverse impact for %s: %w", sourceREF, err)
	}
	return sourceHead, impacts, nil
}

func ParseSelector(text string) (string, string, error) {
	if index := strings.LastIndexByte(text, '@'); index >= 0 {
		if index == 0 || index == len(text)-1 {
			return "", "", fmt.Errorf("selector %q must be REF@TOKEN", text)
		}
		ref, revision := text[:index], text[index+1:]
		if err := domain.ValidateREF(ref); err != nil {
			return "", "", err
		}
		if !domain.IsObjectPrefix(revision) {
			if err := domain.ValidateTagName(revision); err != nil {
				return "", "", fmt.Errorf("invalid selector token: %w", err)
			}
		}
		return ref, revision, nil
	}
	if err := domain.ValidateREF(text); err != nil {
		return "", "", err
	}
	return text, "", nil
}

// ResolveSealID resolves current HEAD, a unique object prefix, or a REF-scoped
// immutable tag and validates that the result is a canonical seal owned by ref.
func (r *Repository) ResolveSealID(ctx context.Context, ref, revision string) (domain.ObjectID, error) {
	if err := domain.ValidateREF(ref); err != nil {
		return domain.ObjectID{}, err
	}
	var id domain.ObjectID
	var err error
	switch {
	case revision == "":
		id, err = r.refs.Resolve(ctx, ref)
	case domain.IsObjectPrefix(revision):
		id, err = r.objects.ResolvePrefix(ctx, revision)
	default:
		if validateErr := domain.ValidateTagName(revision); validateErr != nil {
			return domain.ObjectID{}, validateErr
		}
		id, err = r.tags.Resolve(ctx, ref, revision)
	}
	if err != nil {
		return domain.ObjectID{}, fmt.Errorf("resolve %s selector %q: %w", ref, revision, err)
	}
	payload, err := r.LoadSeal(ctx, id)
	if err != nil {
		return domain.ObjectID{}, fmt.Errorf("resolved object %s is not a canonical seal: %w", id, err)
	}
	if payload.REF != ref {
		return domain.ObjectID{}, fmt.Errorf("seal %s belongs to REF %s, not %s", id, payload.REF, ref)
	}
	return id, nil
}

func (r *Repository) CreateTag(ctx context.Context, ref, revision, name string) (domain.ObjectID, error) {
	return withMutation(ctx, r.writer, "create tag", func() (domain.ObjectID, error) {
		return r.createTag(ctx, ref, revision, name)
	})
}

func (r *Repository) createTag(ctx context.Context, ref, revision, name string) (domain.ObjectID, error) {
	id, err := r.ResolveSealID(ctx, ref, revision)
	if err != nil {
		return domain.ObjectID{}, err
	}
	if err := r.tags.Create(ctx, ref, name, id); err != nil {
		return domain.ObjectID{}, err
	}
	return id, nil
}

func (r *Repository) ListTags(ctx context.Context, ref string) ([]store.Tag, error) {
	if _, err := r.refs.Resolve(ctx, ref); err != nil {
		return nil, fmt.Errorf("resolve tag scope %s: %w", ref, err)
	}
	tags, err := r.tags.List(ctx, ref)
	if err != nil {
		return nil, err
	}
	for _, tag := range tags {
		payload, err := r.LoadSeal(ctx, tag.Seal)
		if err != nil || payload.REF != ref {
			return nil, fmt.Errorf("tag %s@%s does not target a readable seal owned by its REF", ref, tag.Name)
		}
	}
	return tags, nil
}
