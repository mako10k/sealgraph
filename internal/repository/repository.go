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

type Repository struct {
	dir        string
	workDir    string
	objects    *native.ObjectStore
	refs       store.RefStore
	tags       store.TagStore
	candidates candidateStore
	sources    sourceStore
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
	candidates := candidateStore{root: filepath.Join(dir, "index")}
	return &Repository{
		dir:        dir,
		workDir:    filepath.Dir(dir),
		objects:    native.NewObjectStore(dir),
		refs:       native.NewRefStore(dir),
		tags:       native.NewTagStore(dir),
		candidates: candidates,
		sources:    sourceStore{candidates: candidates},
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
	Parent       string
	Root         bool
	Draft        bool
}

type CompletionNames struct {
	REFs       []string
	Candidates []string
	Sources    []string
}

// CompletionNames returns names from repository metadata only. It never opens
// a bound workfile, writes cache state, or bootstraps a repository.
func (r *Repository) CompletionNames(ctx context.Context) (CompletionNames, error) {
	refs, err := r.refs.List(ctx)
	if err != nil {
		return CompletionNames{}, err
	}
	candidates, err := r.candidates.List()
	if err != nil {
		return CompletionNames{}, err
	}
	bindings, err := r.sources.list()
	if err != nil {
		return CompletionNames{}, err
	}
	sources := make([]string, 0, len(bindings))
	for _, binding := range bindings {
		sources = append(sources, binding.REF)
	}
	return CompletionNames{REFs: refs, Candidates: candidates, Sources: sources}, nil
}

func (r *Repository) Add(ctx context.Context, options AddOptions) (domain.Candidate, error) {
	return withMutation(ctx, r.writer, "add candidate", func() (domain.Candidate, error) {
		return r.addLocked(ctx, options, false, true, true)
	})
}

func (r *Repository) addLocked(ctx context.Context, options AddOptions, preserve bool, rootSet, draftSet bool) (domain.Candidate, error) {
	if err := domain.ValidateREF(options.REF); err != nil {
		return domain.Candidate{}, err
	}
	candidate, err := r.candidateForAdd(ctx, options.REF, options.Parent)
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
	if !preserve || rootSet {
		candidate.Root = options.Root
	}
	if !preserve || draftSet {
		candidate.Draft = options.Draft
	}
	if options.Dependencies != nil {
		candidate.Links = links
	}
	if err := r.candidates.Save(candidate); err != nil {
		return domain.Candidate{}, fmt.Errorf("save candidate %s: %w", options.REF, err)
	}
	return candidate, nil
}

func (r *Repository) candidateForAdd(ctx context.Context, ref, parentSelector string) (domain.Candidate, error) {
	if parentSelector == "" {
		return r.candidateForEdit(ctx, ref)
	}
	if err := r.requireAbsentDestination(ctx, ref); err != nil {
		return domain.Candidate{}, err
	}
	parent, err := r.ResolveSelector(ctx, parentSelector)
	if err != nil {
		return domain.Candidate{}, fmt.Errorf("resolve explicit parent %q: %w", parentSelector, err)
	}
	if err := r.validateRevisionChain(ctx, parent.ID); err != nil {
		return domain.Candidate{}, fmt.Errorf("validate explicit parent %s: %w", parent.ID, err)
	}
	parentID := parent.ID
	return domain.Candidate{
		Schema: domain.CandidateSchema, REF: ref, ParentRevision: &parentID,
		Attachments: []domain.Attachment{}, Links: []domain.Link{},
	}, nil
}

func (r *Repository) Derive(ctx context.Context, ref, sourceSelector string) (domain.Candidate, error) {
	return withMutation(ctx, r.writer, "derive candidate", func() (domain.Candidate, error) {
		if err := domain.ValidateREF(ref); err != nil {
			return domain.Candidate{}, err
		}
		if err := r.requireAbsentDestination(ctx, ref); err != nil {
			return domain.Candidate{}, err
		}
		source, err := r.ResolveSelector(ctx, sourceSelector)
		if err != nil {
			return domain.Candidate{}, fmt.Errorf("resolve derive source %q: %w", sourceSelector, err)
		}
		if err := r.validateRevisionChain(ctx, source.ID); err != nil {
			return domain.Candidate{}, fmt.Errorf("validate derive source %s: %w", source.ID, err)
		}
		if err := r.validateSealMaterial(ctx, source.ID, source.Payload); err != nil {
			return domain.Candidate{}, err
		}
		parent := source.ID
		candidate := domain.Candidate{
			Schema: domain.CandidateSchema, REF: ref, ParentRevision: &parent,
			Content: source.Payload.Content, Attachments: append([]domain.Attachment(nil), source.Payload.Attachments...),
			Links: append([]domain.Link(nil), source.Payload.Links...), Root: source.Payload.Root, Draft: source.Payload.Draft,
		}
		if err := r.candidates.Save(candidate); err != nil {
			return domain.Candidate{}, fmt.Errorf("save derived candidate %s: %w", ref, err)
		}
		return candidate, nil
	})
}

func (r *Repository) requireAbsentDestination(ctx context.Context, ref string) error {
	if _, err := r.candidates.LoadSnapshot(ref); err == nil {
		return fmt.Errorf("destination candidate %s already exists; discard it explicitly or choose another REF", ref)
	} else if !errors.Is(err, ErrCandidateNotFound) {
		return fmt.Errorf("destination candidate %s is not safely absent: %w", ref, err)
	}
	if head, err := r.refs.Resolve(ctx, ref); err == nil {
		return fmt.Errorf("destination REF %s already exists at %s; explicit parent creation requires an absent destination", ref, head)
	} else if !errors.Is(err, store.ErrRefNotFound) {
		return fmt.Errorf("destination REF %s is not safely absent: %w", ref, err)
	}
	return nil
}

func (r *Repository) validateSealMaterial(ctx context.Context, id domain.ObjectID, payload domain.SealPayload) error {
	if _, err := r.readRepositoryBlob(ctx, payload.Content, fmt.Sprintf("content for seal %s", id)); err != nil {
		return err
	}
	for _, attachment := range payload.Attachments {
		if _, err := r.readRepositoryBlob(ctx, attachment.Blob, fmt.Sprintf("attachment %q for seal %s", attachment.Name, id)); err != nil {
			return err
		}
	}
	return nil
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
		var admission *headObservation
		if !candidate.Draft && !candidate.Root {
			observed, err := r.requireActiveLeafClosure(ctx, candidate.Links)
			if err != nil {
				return SealResult{}, err
			}
			admission = &observed
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
		if admission != nil {
			if err := r.revalidateHeads(ctx, *admission, "seal admission"); err != nil {
				return SealResult{}, fmt.Errorf("seal object %s was written but REF %s was not advanced: %w", sealID, ref, err)
			}
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
		if err := r.validateRevisionChain(ctx, id); err != nil {
			return fmt.Errorf("Cause target %s has invalid revision ancestry: %w", id, err)
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
		id, err = r.tags.Resolve(ctx, selector.REF, selector.Token)
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
