// Package repository coordinates standalone format-4 domain semantics,
// object storage, REF publication, candidates, and explicit migration load.
package repository

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/mako10k/sealgraph/internal/canonical"
	"github.com/mako10k/sealgraph/internal/domain"
	"github.com/mako10k/sealgraph/internal/store"
	"github.com/mako10k/sealgraph/internal/store/native"
)

var ErrRevisionGraphPending = errors.New("format-4 revision graph is not implemented in this slice")
var ErrTagContractPending = errors.New("format-4 tag namespace is not implemented; wait for the accepted rename-safe tag contract")

type Repository struct {
	dir        string
	objects    *native.ObjectStore
	refs       store.RefStore
	candidates candidateStore
	writer     writerGuard
}

func OpenStandalone(workDir string) (*Repository, error) {
	dir := filepath.Join(workDir, ".sealgraph")
	if err := validateLayout(dir); err != nil {
		return nil, fmt.Errorf("open standalone repository %s: %w; run 'sealgraph init' in this directory or repair it explicitly", dir, err)
	}
	return newRepository(dir), nil
}

func newRepository(dir string) *Repository {
	return &Repository{
		dir:        dir,
		objects:    native.NewObjectStore(dir),
		refs:       native.NewRefStore(dir),
		candidates: candidateStore{root: filepath.Join(dir, "index")},
		writer:     newWriterGuard(filepath.Join(dir, "locks")),
	}
}

type Dependency struct {
	Selector string
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
		if err := domain.ValidateREF(options.REF); err != nil {
			return domain.Candidate{}, err
		}
		candidate, err := r.candidateForEdit(ctx, options.REF)
		if err != nil {
			return domain.Candidate{}, err
		}
		var links []domain.Link
		if options.Dependencies != nil {
			links, err = r.resolveDependencies(ctx, options.Dependencies)
			if err != nil {
				return domain.Candidate{}, err
			}
		}
		contentID, err := r.objects.WriteBlob(ctx, options.Content)
		if err != nil {
			return domain.Candidate{}, fmt.Errorf("store content for %s: %w", options.REF, err)
		}
		candidate.Content = domain.ContentRef{Store: domain.NativeStore, Type: domain.BlobType, ID: contentID}
		candidate.Root = options.Root
		candidate.Draft = options.Draft
		if options.Dependencies != nil {
			candidate.Links = links
		}
		if err := r.candidates.Save(candidate); err != nil {
			return domain.Candidate{}, fmt.Errorf("save candidate %s: %w", options.REF, err)
		}
		return candidate, nil
	})
}

func (r *Repository) Link(ctx context.Context, ref string, dependencies []Dependency) (domain.Candidate, error) {
	return withMutation(ctx, r.writer, "link candidate", func() (domain.Candidate, error) {
		if len(dependencies) == 0 {
			return domain.Candidate{}, errors.New("at least one --depend-on selector is required")
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
		candidate.Links = append(candidate.Links, links...)
		candidate.Links, err = domain.NormalizeLinks(candidate.Links)
		if err != nil {
			return domain.Candidate{}, fmt.Errorf("update candidate %s Cause links: %w; unlink the exact old target before adding it again", ref, err)
		}
		if err := r.candidates.Save(candidate); err != nil {
			return domain.Candidate{}, fmt.Errorf("save candidate %s: %w", ref, err)
		}
		return candidate, nil
	})
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
	parent, expected := head, head
	return domain.Candidate{
		Schema: domain.CandidateSchema, REF: ref,
		ParentRevision: &parent, ExpectedREFHead: &expected,
		Content: seal.Content, Attachments: append([]domain.Attachment(nil), seal.Attachments...),
		Links: append([]domain.Link(nil), seal.Links...), Root: seal.Root, Draft: seal.Draft,
	}, nil
}

func (r *Repository) resolveDependencies(ctx context.Context, dependencies []Dependency) ([]domain.Link, error) {
	links := make([]domain.Link, 0, len(dependencies))
	for _, dependency := range dependencies {
		resolved, err := r.ResolveSelector(ctx, dependency.Selector)
		if err != nil {
			return nil, fmt.Errorf("resolve dependency %q: %w", dependency.Selector, err)
		}
		links = append(links, domain.Link{TargetSeal: resolved.ID, Message: dependency.Message})
	}
	return domain.NormalizeLinks(links)
}

type SealResult struct {
	ID      domain.ObjectID
	Payload domain.SealPayload
}

func (r *Repository) Seal(ctx context.Context, ref string) (SealResult, error) {
	return withMutation(ctx, r.writer, "seal REF", func() (SealResult, error) {
		if err := domain.ValidateREF(ref); err != nil {
			return SealResult{}, err
		}
		snapshot, err := r.candidates.LoadSnapshot(ref)
		if err != nil {
			if errors.Is(err, ErrCandidateNotFound) {
				return SealResult{}, fmt.Errorf("REF %s has no working candidate; run 'sealgraph add' or 'sealgraph link' first", ref)
			}
			return SealResult{}, err
		}
		candidate := snapshot.Candidate
		if err := r.validatePublicationExpectation(ctx, candidate); err != nil {
			return SealResult{}, err
		}
		if _, err := r.readRepositoryBlob(ctx, candidate.Content, fmt.Sprintf("candidate content for %s", ref)); err != nil {
			return SealResult{}, err
		}
		for _, attachment := range candidate.Attachments {
			if _, err := r.readRepositoryBlob(ctx, attachment.Blob, fmt.Sprintf("candidate attachment %q for %s", attachment.Name, ref)); err != nil {
				return SealResult{}, err
			}
		}
		if candidate.ParentRevision != nil {
			if err := r.validateRevisionChain(ctx, *candidate.ParentRevision); err != nil {
				return SealResult{}, fmt.Errorf("candidate parent revision %s is invalid: %w", candidate.ParentRevision, err)
			}
		}
		if err := r.validateCauseClosure(ctx, candidate.Links, !candidate.Draft); err != nil {
			return SealResult{}, err
		}
		if !candidate.Draft && !candidate.Root {
			return SealResult{}, fmt.Errorf("normal non-root publication requires active-revision-leaf admission: %w; keep the candidate unchanged until FORMAT4_REVISION_GRAPH is implemented", ErrRevisionGraphPending)
		}
		payload := domain.SealPayload{
			Schema: domain.SealSchema, ParentRevision: candidate.ParentRevision,
			Content: candidate.Content, Attachments: candidate.Attachments, Links: candidate.Links,
			Root: candidate.Root, Draft: candidate.Draft,
		}
		encoded, err := canonical.EncodeSeal(payload)
		if err != nil {
			return SealResult{}, fmt.Errorf("canonicalize seal for %s: %w", ref, err)
		}
		sealID, err := r.objects.WriteBlob(ctx, encoded)
		if err != nil {
			return SealResult{}, fmt.Errorf("store immutable seal for %s: %w", ref, err)
		}
		if err := r.refs.Update(ctx, ref, candidate.ExpectedREFHead, &sealID); err != nil {
			return SealResult{}, fmt.Errorf("seal object %s was written but REF %s was not advanced: %w", sealID, ref, err)
		}
		payload, _ = domain.NormalizeSeal(payload)
		result := SealResult{ID: sealID, Payload: payload}
		if err := r.candidates.RemoveIfUnchanged(ref, snapshot.Bytes); err != nil {
			return result, fmt.Errorf("REF %s was published at seal %s, but its candidate was retained because cleanup could not prove it was unchanged: %w; inspect the candidate explicitly", ref, sealID, err)
		}
		return result, nil
	})
}

func (r *Repository) validatePublicationExpectation(ctx context.Context, candidate domain.Candidate) error {
	current, err := r.refs.Resolve(ctx, candidate.REF)
	if candidate.ExpectedREFHead == nil {
		if err == nil {
			return fmt.Errorf("REF %s appeared at %s after candidate creation; recreate the candidate before sealing", candidate.REF, current)
		}
		if !errors.Is(err, store.ErrRefNotFound) {
			return err
		}
		return nil
	}
	if err != nil || !current.Equal(*candidate.ExpectedREFHead) {
		return fmt.Errorf("REF %s HEAD changed after candidate creation; recreate the candidate from the current head before sealing", candidate.REF)
	}
	if candidate.ParentRevision == nil || !candidate.ParentRevision.Equal(*candidate.ExpectedREFHead) {
		return fmt.Errorf("existing REF %s candidate must use expected head %s as its parent revision; alternate-parent override is not available in this slice", candidate.REF, candidate.ExpectedREFHead)
	}
	return nil
}

func (r *Repository) validateRevisionChain(ctx context.Context, head domain.ObjectID) error {
	seen := make(map[string]struct{})
	current := &head
	for current != nil {
		if _, exists := seen[current.String()]; exists {
			return fmt.Errorf("revision parent cycle reaches %s", current)
		}
		seen[current.String()] = struct{}{}
		payload, err := r.LoadSeal(ctx, *current)
		if err != nil {
			return err
		}
		current = payload.ParentRevision
	}
	return nil
}

func (r *Repository) validateCauseClosure(ctx context.Context, links []domain.Link, requireNonDraft bool) error {
	state := make(map[string]uint8)
	var visit func(domain.ObjectID) error
	visit = func(id domain.ObjectID) error {
		switch state[id.String()] {
		case 1:
			return fmt.Errorf("Cause cycle reaches seal %s", id)
		case 2:
			return nil
		}
		state[id.String()] = 1
		payload, err := r.LoadSeal(ctx, id)
		if err != nil {
			return fmt.Errorf("Cause target %s is unreadable: %w", id, err)
		}
		if requireNonDraft && payload.Draft {
			return fmt.Errorf("normal seal requires a non-draft Cause closure, but %s is draft; keep the candidate draft or select a non-draft Cause explicitly", id)
		}
		for _, link := range payload.Links {
			if err := visit(link.TargetSeal); err != nil {
				return err
			}
		}
		state[id.String()] = 2
		return nil
	}
	for _, link := range links {
		if err := visit(link.TargetSeal); err != nil {
			return err
		}
	}
	return nil
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

type SelectorKind uint8

const (
	SelectorCurrentREF SelectorKind = iota
	SelectorGlobalSeal
	SelectorScopedSeal
	SelectorScopedTag
)

type Selector struct {
	Raw   string
	Kind  SelectorKind
	REF   string
	Token string
}

func ParseSelector(text string) (Selector, error) {
	if strings.HasPrefix(text, "@") {
		token := strings.TrimPrefix(text, "@")
		if !domain.IsObjectPrefix(token) {
			return Selector{}, fmt.Errorf("selector %q requires @ followed by 4 to 64 lower-case hexadecimal characters", text)
		}
		return Selector{Raw: text, Kind: SelectorGlobalSeal, Token: token}, nil
	}
	if index := strings.LastIndexByte(text, '@'); index >= 0 {
		if index == len(text)-1 {
			return Selector{}, fmt.Errorf("selector %q has an empty token", text)
		}
		ref, token := text[:index], text[index+1:]
		if err := domain.ValidateREF(ref); err != nil {
			return Selector{}, err
		}
		if domain.IsObjectPrefix(token) {
			return Selector{Raw: text, Kind: SelectorScopedSeal, REF: ref, Token: token}, nil
		}
		if err := domain.ValidateTagName(token); err != nil {
			return Selector{}, fmt.Errorf("invalid selector token: %w", err)
		}
		return Selector{Raw: text, Kind: SelectorScopedTag, REF: ref, Token: token}, nil
	}
	if err := domain.ValidateREF(text); err != nil {
		return Selector{}, err
	}
	return Selector{Raw: text, Kind: SelectorCurrentREF, REF: text}, nil
}

type ResolvedSelector struct {
	Selector Selector
	ID       domain.ObjectID
	Payload  domain.SealPayload
}

func (r *Repository) ResolveSelector(ctx context.Context, text string) (ResolvedSelector, error) {
	selector, err := ParseSelector(text)
	if err != nil {
		return ResolvedSelector{}, err
	}
	var id domain.ObjectID
	switch selector.Kind {
	case SelectorCurrentREF:
		id, err = r.refs.Resolve(ctx, selector.REF)
	case SelectorGlobalSeal, SelectorScopedSeal:
		id, err = r.objects.ResolvePrefix(ctx, selector.Token)
	case SelectorScopedTag:
		return ResolvedSelector{}, fmt.Errorf("cannot resolve %q: %w", text, ErrTagContractPending)
	}
	if err != nil {
		return ResolvedSelector{}, fmt.Errorf("resolve selector %q: %w", text, err)
	}
	payload, err := r.LoadSeal(ctx, id)
	if err != nil {
		return ResolvedSelector{}, fmt.Errorf("selector %q resolved object %s that is not a canonical format-4 Seal: %w", text, id, err)
	}
	if selector.Kind == SelectorScopedSeal {
		head, err := r.refs.Resolve(ctx, selector.REF)
		if err != nil {
			return ResolvedSelector{}, fmt.Errorf("resolve selector scope %s HEAD: %w", selector.REF, err)
		}
		if err := r.requireAncestor(ctx, head, id); err != nil {
			return ResolvedSelector{}, fmt.Errorf("selector %q is outside the current parent ancestry: %w; use @%s for an unscoped sibling or detached Seal", text, err, id)
		}
	}
	return ResolvedSelector{Selector: selector, ID: id, Payload: payload}, nil
}

func (r *Repository) requireAncestor(ctx context.Context, head, selected domain.ObjectID) error {
	seen := make(map[string]struct{})
	current := head
	for {
		if current.Equal(selected) {
			return nil
		}
		if _, exists := seen[current.String()]; exists {
			return fmt.Errorf("revision parent cycle reaches %s", current)
		}
		seen[current.String()] = struct{}{}
		payload, err := r.LoadSeal(ctx, current)
		if err != nil {
			return err
		}
		if payload.ParentRevision == nil {
			return fmt.Errorf("seal %s is not an ancestor of %s", selected, head)
		}
		current = *payload.ParentRevision
	}
}

type ShowResult struct {
	ID       domain.ObjectID
	Payload  domain.SealPayload
	Content  []byte
	REFNames []string
}

func (r *Repository) Show(ctx context.Context, selector string) (ShowResult, error) {
	resolved, err := r.ResolveSelector(ctx, selector)
	if err != nil {
		return ShowResult{}, err
	}
	content, err := r.readRepositoryBlob(ctx, resolved.Payload.Content, fmt.Sprintf("content %s for seal %s", resolved.Payload.Content.ID, resolved.ID))
	if err != nil {
		return ShowResult{}, err
	}
	for _, attachment := range resolved.Payload.Attachments {
		if _, err := r.readRepositoryBlob(ctx, attachment.Blob, fmt.Sprintf("attachment %q for seal %s", attachment.Name, resolved.ID)); err != nil {
			return ShowResult{}, err
		}
	}
	aliases, err := r.refsPointingTo(ctx, resolved.ID)
	if err != nil {
		return ShowResult{}, err
	}
	return ShowResult{ID: resolved.ID, Payload: resolved.Payload, Content: content, REFNames: aliases}, nil
}

func (r *Repository) refsPointingTo(ctx context.Context, id domain.ObjectID) ([]string, error) {
	names, err := r.refs.List(ctx)
	if err != nil {
		return nil, err
	}
	var aliases []string
	for _, name := range names {
		head, err := r.refs.Resolve(ctx, name)
		if err != nil {
			return nil, err
		}
		if head.Equal(id) {
			aliases = append(aliases, name)
		}
	}
	sort.Strings(aliases)
	return aliases, nil
}

type RefStatus struct {
	REF      string
	Head     *domain.ObjectID
	Unsealed bool
	Draft    bool
}

func (s RefStatus) Labels() []string {
	var labels []string
	if s.Unsealed {
		labels = append(labels, "UNSEALED")
	}
	if s.Draft {
		labels = append(labels, "DRAFT")
	}
	if len(labels) == 0 {
		labels = append(labels, "CLEAN")
	}
	return labels
}

func (r *Repository) Status(ctx context.Context, onlyREF string) ([]RefStatus, error) {
	return nil, ErrRevisionGraphPending
}

func (r *Repository) RevisionGraphUnavailable() error { return ErrRevisionGraphPending }

func (r *Repository) CreateTag(context.Context, string, string) (domain.ObjectID, error) {
	return domain.ObjectID{}, ErrTagContractPending
}

func (r *Repository) ListTags(context.Context, string) ([]store.Tag, error) {
	return nil, ErrTagContractPending
}
