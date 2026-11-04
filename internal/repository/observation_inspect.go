package repository

// R: expose separate, coherent read-only observation dimensions without changing admission.

import (
	"context"
	"crypto/sha256"
	"fmt"
	"reflect"
	"sort"

	"github.com/mako10k/sealgraph/internal/domain"
	domainv5 "github.com/mako10k/sealgraph/internal/domain/v5"
)

type ObservationOptions struct {
	Selector  string
	Inventory bool
	MaxPaths  int
}
type ObservationResult struct {
	Schema string            `json:"schema"`
	Scope  string            `json:"scope"`
	REFs   []ObservationREF  `json:"refs"`
	Seals  []ObservationSeal `json:"seals"`
}
type ObservationREF struct {
	REF       string                `json:"ref"`
	Head      *domain.ObjectID      `json:"head"`
	HeadDraft *bool                 `json:"head_draft"`
	Candidate *ObservationCandidate `json:"candidate"`
	Source    *SourceStatus         `json:"source"`
	Legacy    ObservationLegacy     `json:"legacy_stale"`
}
type ObservationLegacy struct {
	Self       bool                `json:"self"`
	Direct     []domain.ObjectID   `json:"direct"`
	Transitive [][]domain.ObjectID `json:"transitive"`
}
type ObservationCandidate struct {
	Draft                 bool                       `json:"draft"`
	ProspectiveSeal       domain.ObjectID            `json:"prospective_seal"`
	ExactDigest           string                     `json:"exact_digest"`
	ExpectedREFHead       *domain.ObjectID           `json:"expected_ref_head"`
	ExpectedHeadState     CandidateExpectedHeadState `json:"expected_head_state"`
	CurrentHeadComparison string                     `json:"current_head_comparison"`
	DiagnosticBasis       string                     `json:"diagnostic_basis"`
	Reasons               []ObservationReason        `json:"reasons"`
}
type ObservationSeal struct {
	SealID              domain.ObjectID              `json:"seal_id"`
	Active              bool                         `json:"active"`
	ActiveLeaf          bool                         `json:"active_leaf"`
	Successors          []domain.ObjectID            `json:"observed_successors"`
	Previous            domainv5.RevisionObservation `json:"previous"`
	SuccessorAssertions []domainv5.AssertionSource   `json:"successor_assertions"`
	Reasons             []ObservationReason          `json:"reasons"`
}
type ObservationReason struct {
	Subject       domain.ObjectID     `json:"subject"`
	Target        domain.ObjectID     `json:"target"`
	Kind          string              `json:"kind"`
	Direct        bool                `json:"direct"`
	Paths         [][]domain.ObjectID `json:"paths"`
	PathsComplete bool                `json:"paths_complete"`
}

func (r *Repository) Observe(ctx context.Context, options ObservationOptions) (ObservationResult, error) {
	if options.MaxPaths == 0 {
		options.MaxPaths = 100
	}
	if options.MaxPaths < 1 || options.MaxPaths > 10000 || (options.Inventory && options.Selector != "") {
		return ObservationResult{}, fmt.Errorf("observe requires positive max-paths and either selector or inventory")
	}
	physical, err := capturePhysicalRepositoryObservation(r.dir)
	if err != nil {
		return ObservationResult{}, err
	}
	observation, graph, err := r.buildObservation(ctx, "observe")
	if err != nil {
		return ObservationResult{}, err
	}
	result := ObservationResult{Schema: "sealgraph/observation/v1", Scope: "CURRENT", REFs: []ObservationREF{}, Seals: []ObservationSeal{}}
	names, selected, err := r.observationSelection(ctx, options, &observation, graph)
	if err != nil {
		return result, err
	}
	if options.Selector != "" {
		result.Scope = "SELECTED_SEAL"
		if len(names) > 0 {
			result.Scope = "SELECTED_REF"
		}
	}
	refs, sources, err := r.observationREFs(ctx, names, observation, graph, options.MaxPaths)
	if err != nil {
		return result, err
	}
	result.REFs = refs
	selected, err = r.observationInventory(ctx, options.Inventory, physical, selected)
	if err != nil {
		return result, err
	}
	if options.Inventory {
		result.Scope = "RETAINED_INVENTORY"
	}
	sortIDs(selected)
	for _, id := range selected {
		seal, err := r.observationSeal(ctx, id, graph, options.MaxPaths)
		if err != nil {
			return result, err
		}
		result.Seals = append(result.Seals, seal)
	}
	finalNames, _, err := r.observationSelection(ctx, options, &observation, graph)
	if err != nil || !reflect.DeepEqual(names, finalNames) {
		return result, fmt.Errorf("REF/Candidate/source inventory changed during observation; rerun")
	}
	if err := r.revalidateObservationSources(ctx, result.REFs, sources, observation, graph, options.MaxPaths); err != nil {
		return result, err
	}
	if err := r.validateFsckFinalObservation(ctx, observation, physical); err != nil {
		return result, err
	}
	return result, nil
}

func (r *Repository) observationSelection(ctx context.Context, options ObservationOptions, observation *headObservation, graph *observedGraph) ([]string, []domain.ObjectID, error) {
	candidates, err := r.candidates.List()
	if err != nil {
		return nil, nil, err
	}
	bindings, err := r.sources.list()
	if err != nil {
		return nil, nil, err
	}
	sourceNames := []string{}
	for _, binding := range bindings {
		sourceNames = append(sourceNames, binding.REF)
	}
	if options.Selector != "" {
		selector, err := ParseSelector(options.Selector)
		if err != nil {
			return nil, nil, err
		}
		if selector.Kind == SelectorCurrentREF {
			names, err := statusNames(observation.names, candidates, sourceNames, selector.REF)
			ids := []domain.ObjectID{}
			if id, ok := observation.heads[selector.REF]; ok {
				ids = append(ids, id)
			}
			return names, ids, err
		}
		selected, err := r.resolveSelectorObserved(ctx, options.Selector, observation, graph)
		return []string{}, []domain.ObjectID{selected.ID}, err
	}
	names, err := statusNames(observation.names, candidates, sourceNames, "")
	ids := []domain.ObjectID{}
	for id := range graph.nodes {
		ids = append(ids, domain.ObjectID{Hex: id})
	}
	return names, ids, err
}

func (r *Repository) observationREF(ctx context.Context, ref string, observation headObservation, graph *observedGraph, max int) (ObservationREF, error) {
	status, err := r.statusForREF(ref, observation, graph)
	if err != nil {
		return ObservationREF{}, err
	}
	result := ObservationREF{REF: ref, Head: status.Head, Source: status.Source, Legacy: ObservationLegacy{status.StaleSelf, status.StaleDirect, status.StaleTransitive}}
	if status.Head != nil {
		draft := graph.nodes[status.Head.String()].Provenance.Draft
		result.HeadDraft = &draft
	}
	if !status.Unsealed {
		return result, nil
	}
	snapshot, err := r.candidates.LoadSnapshot(ref)
	if err != nil {
		return result, err
	}
	inspection, err := r.InspectCandidate(ctx, ref)
	if err != nil {
		return result, err
	}
	if !reflect.DeepEqual(snapshot.Candidate, inspection.Candidate) {
		return result, fmt.Errorf("Candidate changed during observation; rerun")
	}
	digest := sha256.Sum256(snapshot.Bytes)
	comparison := "NO_HEAD"
	if status.Head != nil {
		comparison = "DIFFERENT"
		if inspection.Prospective.ID.Equal(*status.Head) {
			comparison = "SAME"
		}
	}
	causes := []domain.ObjectID{}
	for _, link := range inspection.Candidate.CauseLinks {
		causes = appendUniqueID(causes, link.TargetSeal)
	}
	reasons, err := r.observationReasons(ctx, inspection.Prospective.ID, causes, graph, max)
	if err != nil {
		return result, err
	}
	result.Candidate = &ObservationCandidate{Draft: snapshot.Candidate.Draft, ProspectiveSeal: inspection.Prospective.ID, ExactDigest: fmt.Sprintf("%x", digest), ExpectedREFHead: snapshot.Candidate.ExpectedREFHead, ExpectedHeadState: candidateExpectedHeadState(snapshot.Candidate.ExpectedREFHead, status.Head), CurrentHeadComparison: comparison, DiagnosticBasis: "CURRENT_OBSERVATION_NOT_ADMISSION", Reasons: reasons}
	return result, nil
}

func (r *Repository) observationSeal(ctx context.Context, id domain.ObjectID, graph *observedGraph, max int) (ObservationSeal, error) {
	resolved, err := r.LoadSeal(ctx, id)
	if err != nil {
		return ObservationSeal{}, err
	}
	causes := []domain.ObjectID{}
	for _, link := range resolved.Provenance.CauseLinks {
		causes = appendUniqueID(causes, link.TargetSeal)
	}
	reasons, err := r.observationReasons(ctx, id, causes, graph, max)
	if err != nil {
		return ObservationSeal{}, err
	}
	result := ObservationSeal{SealID: id, Active: graph.active[id.String()], ActiveLeaf: graph.isActiveLeaf(id), Successors: append([]domain.ObjectID{}, graph.children[id.String()]...), Previous: graph.revisionObservation(id), SuccessorAssertions: []domainv5.AssertionSource{}, Reasons: reasons}
	for _, successor := range result.Successors {
		for _, assertion := range graph.assertions[successor.String()] {
			for _, previous := range assertion.CauseLink.PreviousRevisionSealOfTargetSeal {
				if previous.Equal(id) {
					result.SuccessorAssertions = append(result.SuccessorAssertions, assertion)
				}
			}
		}
	}
	return result, nil
}

func (r *Repository) revalidateObservationSources(ctx context.Context, entries []ObservationREF, sources map[string]SourceCompareResult, observation headObservation, graph *observedGraph, max int) error {
	for _, entry := range entries {
		afterEntry, err := r.observationREF(ctx, entry.REF, observation, graph, max)
		if err != nil || !reflect.DeepEqual(entry, afterEntry) {
			return fmt.Errorf("REF/Candidate/source changed for %s during observation; rerun", entry.REF)
		}
		status, err := r.statusForREF(entry.REF, observation, graph)
		if err != nil {
			return err
		}
		if !reflect.DeepEqual(status.Source, entry.Source) {
			return fmt.Errorf("workfile observation changed for %s; rerun", entry.REF)
		}
		if source, ok := sources[entry.REF]; ok {
			after, err := r.SourceCompare(ctx, entry.REF)
			if err != nil || !reflect.DeepEqual(source, after) {
				return fmt.Errorf("workfile bytes changed for %s; rerun", entry.REF)
			}
		}
	}
	return nil
}

// observationReasons traverses all Cause nodes, retaining target membership even
// when alternative paths exceed the per-target bound. Candidate assertions never
// modify graph: every reason is diagnosed against the captured current graph.
func (r *Repository) observationReasons(ctx context.Context, subject domain.ObjectID, causes []domain.ObjectID, graph *observedGraph, max int) ([]ObservationReason, error) {
	records := map[string]*ObservationReason{}
	var visit func(domain.ObjectID, []domain.ObjectID) error
	visit = func(id domain.ObjectID, path []domain.ObjectID) error {
		for _, ancestor := range path {
			if ancestor.Equal(id) {
				return fmt.Errorf("Cause cycle in observation at %s", id)
			}
		}
		next := append(append([]domain.ObjectID{}, path...), id)
		if !graph.isActiveLeaf(id) {
			kind := "OUTSIDE_ACTIVE"
			if graph.active[id.String()] {
				kind = "ACTIVE_NON_LEAF"
			}
			key := id.String() + kind
			record := records[key]
			if record == nil {
				record = &ObservationReason{Subject: subject, Target: id, Kind: kind, Paths: [][]domain.ObjectID{}, PathsComplete: true}
				records[key] = record
			}
			record.Direct = record.Direct || len(path) == 1
			if len(record.Paths) < max {
				record.Paths = append(record.Paths, next)
			} else {
				record.PathsComplete = false
			}
		}
		resolved, err := r.LoadSeal(ctx, id)
		if err != nil {
			return err
		}
		for _, link := range resolved.Provenance.CauseLinks {
			if err := visit(link.TargetSeal, next); err != nil {
				return err
			}
		}
		return nil
	}
	sortIDs(causes)
	for _, id := range causes {
		if err := visit(id, []domain.ObjectID{subject}); err != nil {
			return nil, err
		}
	}
	keys := []string{}
	for key := range records {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	result := []ObservationReason{}
	for _, key := range keys {
		result = append(result, *records[key])
	}
	return result, nil
}

func (r *Repository) observationInventory(ctx context.Context, enabled bool, physical physicalRepositoryObservation, selected []domain.ObjectID) ([]domain.ObjectID, error) {
	if !enabled {
		return selected, nil
	}
	inventory, err := fsckInventoryFromPhysicalFormat(ctx, physical, r.format)
	if err != nil {
		return nil, err
	}
	for id := range inventory.seals {
		selected = appendUniqueID(selected, domain.ObjectID{Hex: id})
	}
	return selected, nil
}

func (r *Repository) observationREFs(ctx context.Context, names []string, observation headObservation, graph *observedGraph, max int) ([]ObservationREF, map[string]SourceCompareResult, error) {
	refs := []ObservationREF{}
	sources := make(map[string]SourceCompareResult)
	for _, name := range names {
		entry, err := r.observationREF(ctx, name, observation, graph, max)
		if err != nil {
			return nil, nil, err
		}
		refs = append(refs, entry)
		if entry.Source != nil {
			if source, err := r.SourceCompare(ctx, name); err == nil {
				sources[name] = source
			}
		}
	}
	return refs, sources, nil
}
