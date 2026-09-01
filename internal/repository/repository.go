// Package repository coordinates the standalone format-5 runtime. Format-4
// repositories are rejected at the config boundary and are never interpreted
// by ordinary runtime readers.
package repository

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	canonicalv5 "github.com/mako10k/sealgraph/internal/canonical/v5"
	"github.com/mako10k/sealgraph/internal/domain"
	domainv5 "github.com/mako10k/sealgraph/internal/domain/v5"
	"github.com/mako10k/sealgraph/internal/recovery"
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
	recovery   *recovery.Store
	writer     writerGuard
}

func OpenStandalone(workDir string) (*Repository, error) {
	dir := filepath.Join(workDir, ".sealgraph")
	if err := validateLayout(dir); err != nil {
		return nil, fmt.Errorf("open standalone repository %s: %w; run 'sealgraph init' only for an absent format-5 repository", dir, err)
	}
	return newRepository(dir), nil
}

func newRepository(dir string) *Repository {
	candidates := candidateStore{root: filepath.Join(dir, "index")}
	return &Repository{
		dir: dir, workDir: filepath.Dir(dir), objects: native.NewObjectStore(dir),
		refs: native.NewRefStore(dir), tags: native.NewTagStore(dir), candidates: candidates,
		sources: sourceStore{candidates: candidates}, recovery: recovery.NewStore(dir),
		writer: newWriterGuard(filepath.Join(dir, "locks")),
	}
}

// CauseInput is one complete format-5 Cause Link authoring record. Selectors
// are resolved before Candidate persistence; an empty Previous slice is the
// explicit --no-previous assertion.
type CauseInput struct {
	Target   string
	Previous []string
	Messages []string
}

type AddOptions struct {
	REF             string
	Content         []byte
	Cause           *CauseInput
	Root            bool
	RootSet         bool
	ClearCauseLinks bool
	Draft           bool
	DraftSet        bool
}

type CompletionNames struct{ REFs, Candidates, Sources, RecoveryIDs []string }

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
	entries, err := r.recovery.List()
	if err != nil {
		return CompletionNames{}, err
	}
	ids := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.Record != nil {
			ids = append(ids, entry.ID)
		}
	}
	return CompletionNames{REFs: refs, Candidates: candidates, Sources: sources, RecoveryIDs: ids}, nil
}

func (r *Repository) Add(ctx context.Context, options AddOptions) (domainv5.Candidate, error) {
	return withMutation(ctx, r.writer, "add candidate", func() (domainv5.Candidate, error) { return r.addLocked(ctx, options) })
}

func (r *Repository) addLocked(ctx context.Context, options AddOptions) (domainv5.Candidate, error) {
	if err := domain.ValidateREF(options.REF); err != nil {
		return domainv5.Candidate{}, err
	}
	observation, graph, err := r.buildObservation(ctx, "add Candidate")
	if err != nil {
		return domainv5.Candidate{}, err
	}
	edit, err := r.candidateForObservedEdit(ctx, options.REF, observation, graph)
	if err != nil {
		return domainv5.Candidate{}, err
	}
	candidate, err := r.applyAddOptions(ctx, edit, options, &observation, graph)
	if err != nil {
		return domainv5.Candidate{}, err
	}
	if err := r.validateCandidateMutation(ctx, candidate, observation); err != nil {
		return domainv5.Candidate{}, err
	}
	written, err := r.objects.WriteBlob(ctx, options.Content)
	if err != nil || !written.Equal(candidate.Content) {
		return domainv5.Candidate{}, fmt.Errorf("store content for %s: id=%s err=%w", options.REF, written, err)
	}
	if err := r.accountForObservedObjectWrite(ctx, &observation, written, "add Candidate publication"); err != nil {
		return domainv5.Candidate{}, err
	}
	if err := r.revalidateHeads(ctx, observation, "add Candidate publication"); err != nil {
		return domainv5.Candidate{}, err
	}
	if err := r.candidates.SaveIfUnchanged(candidate, edit.Bytes, edit.Exists); err != nil {
		return domainv5.Candidate{}, fmt.Errorf("save candidate %s: %w", options.REF, err)
	}
	return r.candidates.Load(options.REF)
}

func (r *Repository) applyAddOptions(ctx context.Context, edit candidateEdit, options AddOptions, observation *headObservation, graph *observedGraph) (domainv5.Candidate, error) {
	if edit.Initial && !options.RootSet && !options.Root {
		return domainv5.Candidate{}, fmt.Errorf("new Candidate %s requires explicit --root --clear-cause-links or --non-root with one complete --target group", options.REF)
	}
	candidate := edit.Candidate
	candidate.Content = domain.ComputeNativeBlobID(options.Content)
	if options.DraftSet {
		candidate.Draft = options.Draft
	}
	if options.RootSet || options.Root {
		candidate.Root = options.Root
	}
	if options.Root && !options.ClearCauseLinks {
		return domainv5.Candidate{}, errors.New("--root requires --clear-cause-links so the root transition is explicit and atomic")
	}
	if options.ClearCauseLinks {
		candidate.CauseLinks = []domainv5.CauseLink{}
	}
	if options.Cause != nil {
		if candidate.Root {
			return domainv5.Candidate{}, errors.New("a root Candidate cannot contain a Cause Link")
		}
		link, err := r.resolveCauseInputObserved(ctx, *options.Cause, observation, graph)
		if err != nil {
			return domainv5.Candidate{}, err
		}
		candidate.CauseLinks = replaceCauseLink(candidate.CauseLinks, link)
	}
	return candidate, nil
}

type candidateEdit struct {
	Candidate domainv5.Candidate
	Bytes     []byte
	Exists    bool
	Initial   bool
}

func (r *Repository) candidateForObservedEdit(ctx context.Context, ref string, observation headObservation, graph *observedGraph) (candidateEdit, error) {
	if err := domain.ValidateREF(ref); err != nil {
		return candidateEdit{}, err
	}
	if snapshot, err := r.candidates.LoadSnapshot(ref); err == nil {
		return candidateEdit{Candidate: snapshot.Candidate, Bytes: snapshot.Bytes, Exists: true}, nil
	} else if !errors.Is(err, ErrCandidateNotFound) {
		return candidateEdit{}, err
	}
	head, ok := observation.heads[ref]
	if !ok {
		return candidateEdit{Candidate: domainv5.Candidate{Schema: domainv5.CandidateSchema, REF: ref, Attachments: []domainv5.Attachment{}, CauseLinks: []domainv5.CauseLink{}}, Initial: true}, nil
	}
	resolved, ok := graph.nodes[head.String()]
	if !ok {
		return candidateEdit{}, fmt.Errorf("load observed current Seal for %s: %s is absent from graph", ref, head)
	}
	expected := head
	return candidateEdit{Candidate: domainv5.Candidate{
		Schema: domainv5.CandidateSchema, REF: ref, ExpectedREFHead: &expected,
		Content: resolved.Material.Content, Attachments: append([]domainv5.Attachment(nil), resolved.Material.Attachments...),
		Root: resolved.Provenance.Root, Draft: resolved.Provenance.Draft,
		CauseLinks: cloneCauseLinks(resolved.Provenance.CauseLinks),
	}}, nil
}

func (r *Repository) requireAbsentDestination(ctx context.Context, ref string) error {
	if _, err := r.candidates.LoadSnapshot(ref); err == nil {
		return fmt.Errorf("destination Candidate %s already exists", ref)
	} else if !errors.Is(err, ErrCandidateNotFound) {
		return err
	}
	if head, err := r.refs.Resolve(ctx, ref); err == nil {
		return fmt.Errorf("destination REF %s already exists at %s", ref, head)
	} else if !errors.Is(err, store.ErrRefNotFound) {
		return err
	}
	return nil
}

func (r *Repository) resolveCauseInputObserved(ctx context.Context, input CauseInput, observation *headObservation, graph *observedGraph) (domainv5.CauseLink, error) {
	if input.Target == "" {
		return domainv5.CauseLink{}, errors.New("Cause Link target is empty")
	}
	target, err := r.resolveSelectorObserved(ctx, input.Target, observation, graph)
	if err != nil {
		return domainv5.CauseLink{}, fmt.Errorf("resolve Cause target %q: %w", input.Target, err)
	}
	previous := make([]domain.ObjectID, 0, len(input.Previous))
	for _, selector := range input.Previous {
		resolved, err := r.resolveSelectorObserved(ctx, selector, observation, graph)
		if err != nil {
			return domainv5.CauseLink{}, fmt.Errorf("resolve previous revision %q: %w", selector, err)
		}
		previous = append(previous, resolved.ID)
	}
	provenance := domainv5.Provenance{Schema: domainv5.ProvenanceSchema, CauseLinks: []domainv5.CauseLink{{
		TargetSeal: target.ID, PreviousRevisionSealOfTargetSeal: previous, Messages: append([]string(nil), input.Messages...),
	}}}
	normalized, err := domainv5.NormalizeProvenance(provenance)
	if err != nil {
		return domainv5.CauseLink{}, fmt.Errorf("validate Cause Link for %s: %w", target.ID, err)
	}
	return normalized.CauseLinks[0], nil
}

func replaceCauseLink(links []domainv5.CauseLink, replacement domainv5.CauseLink) []domainv5.CauseLink {
	result := make([]domainv5.CauseLink, 0, len(links)+1)
	for _, link := range links {
		if !link.TargetSeal.Equal(replacement.TargetSeal) {
			result = append(result, link)
		}
	}
	return append(result, replacement)
}

func cloneCauseLinks(links []domainv5.CauseLink) []domainv5.CauseLink {
	result := make([]domainv5.CauseLink, len(links))
	for i, link := range links {
		result[i] = domainv5.CauseLink{TargetSeal: link.TargetSeal,
			PreviousRevisionSealOfTargetSeal: append([]domain.ObjectID(nil), link.PreviousRevisionSealOfTargetSeal...),
			Messages:                         append([]string(nil), link.Messages...)}
	}
	return result
}

func (r *Repository) Link(ctx context.Context, ref string, input CauseInput) (domainv5.Candidate, error) {
	return withMutation(ctx, r.writer, "link candidate", func() (domainv5.Candidate, error) {
		observation, graph, err := r.buildObservation(ctx, "link Candidate")
		if err != nil {
			return domainv5.Candidate{}, err
		}
		edit, err := r.candidateForObservedEdit(ctx, ref, observation, graph)
		if err != nil {
			return domainv5.Candidate{}, err
		}
		candidate := edit.Candidate
		if candidate.Content.Hex == "" {
			return domainv5.Candidate{}, fmt.Errorf("REF %s has no content Candidate; run 'sealgraph add' first", ref)
		}
		if candidate.Root {
			return domainv5.Candidate{}, fmt.Errorf("Candidate %s is root; use add --non-root with one complete --target group", ref)
		}
		link, err := r.resolveCauseInputObserved(ctx, input, &observation, graph)
		if err != nil {
			return domainv5.Candidate{}, err
		}
		candidate.CauseLinks = replaceCauseLink(candidate.CauseLinks, link)
		if err := r.validateCandidateMutation(ctx, candidate, observation); err != nil {
			return domainv5.Candidate{}, err
		}
		if err := r.candidates.SaveIfUnchanged(candidate, edit.Bytes, edit.Exists); err != nil {
			return domainv5.Candidate{}, fmt.Errorf("save Candidate %s: %w", ref, err)
		}
		return r.candidates.Load(ref)
	})
}

func (r *Repository) validateCandidateMutation(ctx context.Context, candidate domainv5.Candidate, observation headObservation) error {
	prospective, err := prospectiveSeal(candidate)
	if err != nil {
		return fmt.Errorf("prospective Candidate is invalid: %w", err)
	}
	heads := cloneHeads(observation.heads)
	heads[candidate.REF] = prospective.ID
	if _, err := r.buildObservedGraph(ctx, heads, map[string]domainv5.ResolvedSeal{prospective.ID.String(): prospective}); err != nil {
		return fmt.Errorf("prospective Candidate graph is invalid: %w", err)
	}
	return r.revalidateHeads(ctx, observation, "Candidate graph validation")
}

type SealResult struct {
	ID          domain.ObjectID
	Resolved    domainv5.ResolvedSeal
	OperationID string
}

func (r *Repository) Seal(ctx context.Context, ref string) (SealResult, error) {
	return withMutation(ctx, r.writer, "seal REF", func() (SealResult, error) {
		if err := domain.ValidateREF(ref); err != nil {
			return SealResult{}, err
		}
		snapshot, err := r.candidates.LoadSnapshot(ref)
		if err != nil {
			if errors.Is(err, ErrCandidateNotFound) {
				return SealResult{}, fmt.Errorf("REF %s has no working Candidate; run 'sealgraph add' or 'sealgraph link' first", ref)
			}
			return SealResult{}, err
		}
		candidate := snapshot.Candidate
		observation, err := r.validateSealCandidate(ctx, candidate)
		if err != nil {
			return SealResult{}, err
		}
		material := domainv5.Material{Schema: domainv5.MaterialSchema, Content: candidate.Content, Attachments: candidate.Attachments}
		materialBytes, err := canonicalv5.EncodeMaterial(material)
		if err != nil {
			return SealResult{}, fmt.Errorf("canonicalize Material for %s: %w", ref, err)
		}
		materialID, err := r.objects.WriteBlob(ctx, materialBytes)
		if err != nil {
			return SealResult{}, fmt.Errorf("store immutable Material for %s: %w", ref, err)
		}
		provenance := domainv5.Provenance{Schema: domainv5.ProvenanceSchema, Root: candidate.Root, Draft: candidate.Draft, CauseLinks: candidate.CauseLinks}
		provenanceBytes, err := canonicalv5.EncodeProvenance(provenance)
		if err != nil {
			return SealResult{}, fmt.Errorf("canonicalize Provenance for %s: %w", ref, err)
		}
		provenanceID, err := r.objects.WriteBlob(ctx, provenanceBytes)
		if err != nil {
			return SealResult{}, fmt.Errorf("store immutable Provenance for %s: %w", ref, err)
		}
		seal := domainv5.Seal{Schema: domainv5.SealSchema, Material: materialID, Provenance: provenanceID}
		sealBytes, err := canonicalv5.EncodeSeal(seal)
		if err != nil {
			return SealResult{}, fmt.Errorf("canonicalize Seal for %s: %w", ref, err)
		}
		sealID, err := r.objects.WriteBlob(ctx, sealBytes)
		if err != nil {
			return SealResult{}, fmt.Errorf("store immutable Seal for %s: %w", ref, err)
		}
		if err := r.revalidateHeads(ctx, observation, "seal admission"); err != nil {
			return SealResult{}, fmt.Errorf("Seal Blob %s was written but REF %s was not advanced: %w", sealID, ref, err)
		}
		recoveryRefs, err := r.recoveryRefs()
		if err != nil {
			return SealResult{}, err
		}
		before, err := recoveryRefs.Snapshot(ctx, ref)
		if err != nil {
			return SealResult{}, err
		}
		after, err := recoveryRefs.PreviewUpdate(ctx, ref, candidate.ExpectedREFHead, &sealID)
		if err != nil {
			return SealResult{}, err
		}
		record, err := r.prepareRecovery("seal", []recovery.Transition{{REF: ref, Before: before, After: after}})
		if err != nil {
			return SealResult{}, fmt.Errorf("Seal Blob %s was written but REF %s was not advanced because recovery preparation failed: %w", sealID, ref, err)
		}
		if err := r.refs.Update(ctx, ref, candidate.ExpectedREFHead, &sealID); err != nil {
			return SealResult{}, fmt.Errorf("Seal Blob %s was written but REF %s was not advanced: %w", sealID, ref, err)
		}
		content, err := r.readRepositoryBlobID(ctx, candidate.Content, fmt.Sprintf("published content for %s", ref))
		if err != nil {
			return SealResult{ID: sealID, OperationID: record.ID}, fmt.Errorf("REF %s was published at Seal %s but published content readback failed: %w", ref, sealID, err)
		}
		resolved := domainv5.ResolvedSeal{ID: sealID, Seal: seal, Material: material, Provenance: provenance, ContentBytes: len(content)}
		result := SealResult{ID: sealID, Resolved: resolved, OperationID: record.ID}
		if err := r.commitRecovery(record); err != nil {
			return result, fmt.Errorf("REF %s was published at Seal %s but recovery record %s could not be marked COMMITTED: %w", ref, sealID, record.ID, err)
		}
		if err := r.candidates.RemoveIfUnchanged(ref, snapshot.Bytes); err != nil {
			return result, fmt.Errorf("REF %s was published at Seal %s, but its Candidate was retained: %w", ref, sealID, err)
		}
		return result, nil
	})
}

func (r *Repository) validateSealCandidate(ctx context.Context, candidate domainv5.Candidate) (headObservation, error) {
	prospective, err := prospectiveSeal(candidate)
	if err != nil {
		return headObservation{}, fmt.Errorf("derive prospective Seal for %s: %w", candidate.REF, err)
	}
	if candidate.ExpectedREFHead != nil && candidate.ExpectedREFHead.Equal(prospective.ID) {
		return headObservation{}, fmt.Errorf("SEAL_ID_UNCHANGED: Candidate %s resolves to its current HEAD %s; inspect or discard the unchanged Candidate", candidate.REF, prospective.ID)
	}
	if err := r.validatePublicationExpectation(ctx, candidate); err != nil {
		return headObservation{}, err
	}
	if _, err := r.readRepositoryBlobID(ctx, candidate.Content, fmt.Sprintf("Candidate content for %s", candidate.REF)); err != nil {
		return headObservation{}, err
	}
	for _, attachment := range candidate.Attachments {
		if _, err := r.readRepositoryBlobID(ctx, attachment.Blob, fmt.Sprintf("Candidate attachment %q for %s", attachment.Name, candidate.REF)); err != nil {
			return headObservation{}, err
		}
	}
	return r.validateProspectiveCandidate(ctx, candidate)
}

func (r *Repository) validatePublicationExpectation(ctx context.Context, candidate domainv5.Candidate) error {
	current, err := r.refs.Resolve(ctx, candidate.REF)
	if candidate.ExpectedREFHead == nil {
		if err == nil {
			return fmt.Errorf("REF %s appeared at %s after Candidate creation; recreate the Candidate before sealing", candidate.REF, current)
		}
		if !errors.Is(err, store.ErrRefNotFound) {
			return err
		}
		return nil
	}
	if err != nil || !current.Equal(*candidate.ExpectedREFHead) {
		return fmt.Errorf("REF %s HEAD changed after Candidate creation; recreate the Candidate from the current head before sealing", candidate.REF)
	}
	return nil
}

func (r *Repository) LoadSeal(ctx context.Context, id domain.ObjectID) (domainv5.ResolvedSeal, error) {
	sealObject, err := r.objects.ReadObject(ctx, id)
	if err != nil {
		return domainv5.ResolvedSeal{}, err
	}
	if sealObject.Type != domain.BlobType {
		return domainv5.ResolvedSeal{}, fmt.Errorf("Seal object %s has type %s, expected blob", id, sealObject.Type)
	}
	seal, err := canonicalv5.DecodeSeal(sealObject.Data)
	if err != nil {
		return domainv5.ResolvedSeal{}, fmt.Errorf("object %s is not a canonical format-5 Seal Blob: %w", id, err)
	}
	materialObject, err := r.objects.ReadObject(ctx, seal.Material)
	if err != nil {
		return domainv5.ResolvedSeal{}, fmt.Errorf("read Material %s for Seal %s: %w", seal.Material, id, err)
	}
	material, err := canonicalv5.DecodeMaterial(materialObject.Data)
	if err != nil {
		return domainv5.ResolvedSeal{}, fmt.Errorf("object %s named as Material by Seal %s is invalid: %w", seal.Material, id, err)
	}
	provenanceObject, err := r.objects.ReadObject(ctx, seal.Provenance)
	if err != nil {
		return domainv5.ResolvedSeal{}, fmt.Errorf("read Provenance %s for Seal %s: %w", seal.Provenance, id, err)
	}
	provenance, err := canonicalv5.DecodeProvenance(provenanceObject.Data)
	if err != nil {
		return domainv5.ResolvedSeal{}, fmt.Errorf("object %s named as Provenance by Seal %s is invalid: %w", seal.Provenance, id, err)
	}
	content, err := r.readRepositoryBlobID(ctx, material.Content, fmt.Sprintf("content for Seal %s", id))
	if err != nil {
		return domainv5.ResolvedSeal{}, err
	}
	for _, attachment := range material.Attachments {
		if _, err := r.readRepositoryBlobID(ctx, attachment.Blob, fmt.Sprintf("attachment %q for Seal %s", attachment.Name, id)); err != nil {
			return domainv5.ResolvedSeal{}, err
		}
	}
	return domainv5.ResolvedSeal{ID: id, Seal: seal, Material: material, Provenance: provenance, ContentBytes: len(content)}, nil
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
	Resolved domainv5.ResolvedSeal
}

func (r *Repository) ResolveSelector(ctx context.Context, text string) (ResolvedSelector, error) {
	observation, graph, err := r.buildObservation(ctx, "selector")
	if err != nil {
		return ResolvedSelector{}, err
	}
	resolved, err := r.resolveSelectorObserved(ctx, text, &observation, graph)
	if err != nil {
		return ResolvedSelector{}, err
	}
	if err := r.revalidateHeads(ctx, observation, "selector"); err != nil {
		return ResolvedSelector{}, err
	}
	return resolved, nil
}

func (r *Repository) resolveSelectorObserved(ctx context.Context, text string, observation *headObservation, graph *observedGraph) (ResolvedSelector, error) {
	selector, err := ParseSelector(text)
	if err != nil {
		return ResolvedSelector{}, err
	}
	var id domain.ObjectID
	switch selector.Kind {
	case SelectorCurrentREF:
		var ok bool
		id, ok = observation.heads[selector.REF]
		if !ok {
			err = fmt.Errorf("REF not found: %s", selector.REF)
		}
	case SelectorGlobalSeal:
		if err = r.ensureObjectInventory(ctx, observation, "selector resolution"); err == nil {
			id, err = resolveObservedObjectPrefix(selector.Token, observation.objectIDs)
		}
	case SelectorScopedSeal:
		head, ok := observation.heads[selector.REF]
		if !ok {
			err = fmt.Errorf("REF not found: %s", selector.REF)
		} else {
			id, err = resolveScopedSealPrefix(graph, head, selector.Token)
		}
	case SelectorScopedTag:
		refTags, ok := observation.tags[selector.REF]
		if !ok {
			err = fmt.Errorf("REF not found: %s", selector.REF)
		} else if id, ok = refTags[selector.Token]; !ok {
			err = fmt.Errorf("%w: %s@%s", store.ErrTagNotFound, selector.REF, selector.Token)
		}
	}
	if err != nil {
		return ResolvedSelector{}, fmt.Errorf("resolve selector %q: %w", text, err)
	}
	resolved, err := r.LoadSeal(ctx, id)
	if err != nil {
		return ResolvedSelector{}, fmt.Errorf("selector %q resolved object %s that is not a canonical format-5 Seal: %w", text, id, err)
	}
	if selector.Kind == SelectorScopedSeal {
		head, ok := observation.heads[selector.REF]
		if !ok || !graph.revisionReachable(head, id) {
			return ResolvedSelector{}, fmt.Errorf("selector %q is outside the observed revision closure of REF %s", text, selector.REF)
		}
	}
	return ResolvedSelector{Selector: selector, ID: id, Resolved: resolved}, nil
}

func resolveScopedSealPrefix(graph *observedGraph, head domain.ObjectID, prefix string) (domain.ObjectID, error) {
	match := ""
	for text := range graph.nodes {
		id := domain.ObjectID{Hex: text}
		if !strings.HasPrefix(text, prefix) || !graph.revisionReachable(head, id) {
			continue
		}
		if match != "" && match != text {
			return domain.ObjectID{}, fmt.Errorf("ambiguous Seal prefix %q in observed revision closure; use more hexadecimal characters", prefix)
		}
		match = text
	}
	if match == "" {
		return domain.ObjectID{}, fmt.Errorf("Seal prefix %s is absent from the observed revision closure", prefix)
	}
	return domain.ObjectID{Hex: match}, nil
}

type ShowResult struct {
	Resolved domainv5.ResolvedSeal
	Content  []byte
	REFNames []string
	Revision domainv5.RevisionObservation
}

func (r *Repository) Show(ctx context.Context, selector string) (ShowResult, error) {
	observation, graph, err := r.buildObservation(ctx, "show")
	if err != nil {
		return ShowResult{}, err
	}
	selected, err := r.resolveSelectorObserved(ctx, selector, &observation, graph)
	if err != nil {
		return ShowResult{}, err
	}
	content, err := r.readRepositoryBlobID(ctx, selected.Resolved.Material.Content, fmt.Sprintf("content for Seal %s", selected.ID))
	if err != nil {
		return ShowResult{}, err
	}
	aliases := graph.refsFor(selected.ID)
	if err := r.revalidateHeads(ctx, observation, "show"); err != nil {
		return ShowResult{}, err
	}
	return ShowResult{Resolved: selected.Resolved, Content: content, REFNames: aliases, Revision: graph.revisionObservation(selected.ID)}, nil
}
