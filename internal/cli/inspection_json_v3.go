package cli

import (
	"reflect"
	"sort"

	domainv5 "github.com/mako10k/sealgraph/internal/domain/v5"
	"github.com/mako10k/sealgraph/internal/repository"
)

type causeLinkJSONV6 struct {
	TargetSeal string              `json:"target_seal"`
	Previous   []string            `json:"previous_revision_seal_of_target_seal"`
	Messages   []string            `json:"messages"`
	Metadata   []metadataEntryJSON `json:"metadata"`
}

type assertionJSONV6 struct {
	ObserverSeal             string              `json:"observer_seal"`
	ObserverSealSchema       string              `json:"observer_seal_schema"`
	ObserverProvenance       string              `json:"observer_provenance"`
	ObserverProvenanceSchema string              `json:"observer_provenance_schema"`
	TargetSeal               string              `json:"target_seal"`
	Previous                 []string            `json:"previous_revision_seal_of_target_seal"`
	Messages                 []string            `json:"messages"`
	Metadata                 []metadataEntryJSON `json:"metadata"`
}

type revisionObservationJSONV3 struct {
	TargetSeal         string            `json:"target_seal"`
	PreviousStates     []string          `json:"previous_states"`
	Assertions         []assertionJSONV6 `json:"assertions"`
	StructuralPrevious []string          `json:"structural_previous_seals"`
}

type scopedRevisionObservationJSONV3 struct {
	TargetSeal         string            `json:"target_seal"`
	PreviousStates     []string          `json:"in_scope_previous_states"`
	Assertions         []assertionJSONV6 `json:"in_scope_assertions"`
	StructuralPrevious []string          `json:"structural_previous_seals"`
}

type revisionEdgeJSONV3 struct {
	TargetSeal string            `json:"target_seal"`
	Previous   string            `json:"previous_revision_seal"`
	Sources    []assertionJSONV6 `json:"assertion_sources"`
}

type sealViewJSONV3 struct {
	SealID           string             `json:"seal_id"`
	SealSchema       string             `json:"seal_schema"`
	MaterialID       string             `json:"material_id"`
	ProvenanceID     string             `json:"provenance_id"`
	ProvenanceSchema string             `json:"provenance_schema"`
	ContentBlobID    string             `json:"content_blob_id"`
	ContentBytes     int                `json:"content_bytes"`
	Attachments      []attachmentJSONV5 `json:"attachments"`
	Root             bool               `json:"root"`
	Draft            bool               `json:"draft"`
	CauseLinks       []causeLinkJSONV6  `json:"cause_links"`
}

type candidateViewJSONV3 struct {
	REF                     string             `json:"ref"`
	CandidateSchema         string             `json:"candidate_schema"`
	ExpectedREFHead         *string            `json:"expected_ref_head"`
	ProspectiveSealID       string             `json:"prospective_seal_id"`
	ProspectiveMaterialID   string             `json:"prospective_material_id"`
	ProspectiveProvenanceID string             `json:"prospective_provenance_id"`
	ContentBlobID           string             `json:"content_blob_id"`
	ContentBytes            int                `json:"content_bytes"`
	Attachments             []attachmentJSONV5 `json:"attachments"`
	Root                    bool               `json:"root"`
	Draft                   bool               `json:"draft"`
	CauseLinks              []causeLinkJSONV6  `json:"cause_links"`
}

func metadataJSON(values []domainv5.MetadataEntry) []metadataEntryJSON {
	result := make([]metadataEntryJSON, 0, len(values))
	for _, value := range values {
		result = append(result, metadataEntryJSON{Namespace: value.Namespace, Schema: value.Schema, Value: value.Value})
	}
	return result
}

func causeLinksJSONV3(values []domainv5.CauseLink) []causeLinkJSONV6 {
	result := make([]causeLinkJSONV6, 0, len(values))
	for _, value := range values {
		result = append(result, causeLinkJSONV6{value.TargetSeal.String(), idsJSON(value.PreviousRevisionSealOfTargetSeal), append([]string{}, value.Messages...), metadataJSON(value.Metadata)})
	}
	return result
}

func assertionsJSONV3(values []domainv5.AssertionSource) []assertionJSONV6 {
	result := make([]assertionJSONV6, 0, len(values))
	for _, value := range values {
		result = append(result, assertionJSONV6{
			value.ObserverSeal.String(), value.ObserverSealSchema, value.ObserverProvenance.String(), value.ObserverProvenanceSchema,
			value.CauseLink.TargetSeal.String(), idsJSON(value.CauseLink.PreviousRevisionSealOfTargetSeal), append([]string{}, value.CauseLink.Messages...), metadataJSON(value.CauseLink.Metadata),
		})
	}
	return result
}

func revisionJSONV3(value domainv5.RevisionObservation) revisionObservationJSONV3 {
	return revisionObservationJSONV3{value.TargetSeal.String(), append([]string{}, value.PreviousStates...), assertionsJSONV3(value.Assertions), idsJSON(value.StructuralPrevious)}
}

func scopedRevisionJSONV3(value repository.ScopedRevisionObservation) scopedRevisionObservationJSONV3 {
	return scopedRevisionObservationJSONV3{value.TargetSeal.String(), append([]string{}, value.PreviousStates...), assertionsJSONV3(value.Assertions), idsJSON(value.StructuralPrevious)}
}

func edgeJSONV3(value repository.RevisionEdge) revisionEdgeJSONV3 {
	return revisionEdgeJSONV3{value.Target.String(), value.Previous.String(), assertionsJSONV3(value.Sources)}
}

func sealViewV3(value domainv5.ResolvedSeal) sealViewJSONV3 {
	return sealViewJSONV3{
		value.ID.String(), value.Seal.Schema, value.Seal.Material.String(), value.Seal.Provenance.String(), value.Provenance.Schema,
		value.Material.Content.String(), value.ContentBytes, attachmentsJSON(value.Material.Attachments), value.Provenance.Root, value.Provenance.Draft, causeLinksJSONV3(value.Provenance.CauseLinks),
	}
}

func candidateViewV3(value repository.CandidateInspection) candidateViewJSONV3 {
	var expected *string
	if value.Candidate.ExpectedREFHead != nil {
		text := value.Candidate.ExpectedREFHead.String()
		expected = &text
	}
	return candidateViewJSONV3{
		value.Candidate.REF, value.Candidate.Schema, expected, value.Prospective.ID.String(), value.Prospective.Seal.Material.String(),
		value.Prospective.Seal.Provenance.String(), value.Candidate.Content.String(), len(value.Content), attachmentsJSON(value.Candidate.Attachments),
		value.Candidate.Root, value.Candidate.Draft, causeLinksJSONV3(value.Candidate.CauseLinks),
	}
}

type metadataChangeJSON struct {
	Namespace string             `json:"namespace"`
	Before    *metadataEntryJSON `json:"before"`
	After     *metadataEntryJSON `json:"after"`
}

type causeLinkChangeJSONV3 struct {
	Target   string               `json:"target_seal"`
	Before   *causeLinkJSONV6     `json:"before"`
	After    *causeLinkJSONV6     `json:"after"`
	Previous changeJSON           `json:"previous_revision_seal_of_target_seal"`
	Messages changeJSON           `json:"messages"`
	Metadata []metadataChangeJSON `json:"metadata"`
}

type causeLinksChangeJSONV3 struct {
	Changed bool                    `json:"changed"`
	Before  []causeLinkJSONV6       `json:"before"`
	After   []causeLinkJSONV6       `json:"after"`
	Records []causeLinkChangeJSONV3 `json:"records"`
}

type changesJSONV3 struct {
	MaterialID    changeJSON             `json:"material_id"`
	ProvenanceID  changeJSON             `json:"provenance_id"`
	ContentBlobID changeJSON             `json:"content_blob_id"`
	Attachments   changeJSON             `json:"attachments"`
	Root          changeJSON             `json:"root"`
	Draft         changeJSON             `json:"draft"`
	CauseLinks    causeLinksChangeJSONV3 `json:"cause_links"`
}

func causeLinkPointer(value *domainv5.CauseLink) *causeLinkJSONV6 {
	if value == nil {
		return nil
	}
	item := causeLinksJSONV3([]domainv5.CauseLink{*value})[0]
	return &item
}

func causeLinkChangesV3(before, after []domainv5.CauseLink) causeLinksChangeJSONV3 {
	left, right := linksByTarget(before), linksByTarget(after)
	targets := make([]string, 0, len(left)+len(right))
	seen := map[string]bool{}
	for target := range left {
		targets = append(targets, target)
		seen[target] = true
	}
	for target := range right {
		if !seen[target] {
			targets = append(targets, target)
		}
	}
	sort.Strings(targets)
	records := []causeLinkChangeJSONV3{}
	for _, target := range targets {
		if reflect.DeepEqual(left[target], right[target]) {
			continue
		}
		records = append(records, causeLinkChangeRecordV3(target, left[target], right[target]))
	}
	return causeLinksChangeJSONV3{len(records) != 0, causeLinksJSONV3(before), causeLinksJSONV3(after), records}
}

func linksByTarget(values []domainv5.CauseLink) map[string]*domainv5.CauseLink {
	result := map[string]*domainv5.CauseLink{}
	for i := range values {
		result[values[i].TargetSeal.String()] = &values[i]
	}
	return result
}

func causeLinkChangeRecordV3(target string, before, after *domainv5.CauseLink) causeLinkChangeJSONV3 {
	var previousBefore, previousAfter, messagesBefore, messagesAfter any
	if before != nil {
		previousBefore = idsJSON(before.PreviousRevisionSealOfTargetSeal)
		messagesBefore = append([]string{}, before.Messages...)
	}
	if after != nil {
		previousAfter = idsJSON(after.PreviousRevisionSealOfTargetSeal)
		messagesAfter = append([]string{}, after.Messages...)
	}
	return causeLinkChangeJSONV3{target, causeLinkPointer(before), causeLinkPointer(after), changed(previousBefore, previousAfter), changed(messagesBefore, messagesAfter), metadataChangesV3(before, after)}
}

func metadataChangesV3(before, after *domainv5.CauseLink) []metadataChangeJSON {
	left, right := map[string]*metadataEntryJSON{}, map[string]*metadataEntryJSON{}
	if before != nil {
		for i := range before.Metadata {
			left[before.Metadata[i].Namespace] = metadataEntryReceipt(&before.Metadata[i])
		}
	}
	if after != nil {
		for i := range after.Metadata {
			right[after.Metadata[i].Namespace] = metadataEntryReceipt(&after.Metadata[i])
		}
	}
	names := make([]string, 0, len(left)+len(right))
	seen := map[string]bool{}
	for name := range left {
		names = append(names, name)
		seen[name] = true
	}
	for name := range right {
		if !seen[name] {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	result := []metadataChangeJSON{}
	for _, name := range names {
		if !reflect.DeepEqual(left[name], right[name]) {
			result = append(result, metadataChangeJSON{name, left[name], right[name]})
		}
	}
	return result
}

func compareChangesV3(before *domainv5.ResolvedSeal, after domainv5.ResolvedSeal) changesJSONV3 {
	var material, provenance, content, attachments, root, draft any
	var links []domainv5.CauseLink
	if before != nil {
		material, provenance, content = before.Seal.Material.String(), before.Seal.Provenance.String(), before.Material.Content.String()
		attachments, root, draft, links = attachmentsJSON(before.Material.Attachments), before.Provenance.Root, before.Provenance.Draft, before.Provenance.CauseLinks
	}
	return changesJSONV3{changed(material, after.Seal.Material.String()), changed(provenance, after.Seal.Provenance.String()), changed(content, after.Material.Content.String()), changed(attachments, attachmentsJSON(after.Material.Attachments)), changed(root, after.Provenance.Root), changed(draft, after.Provenance.Draft), causeLinkChangesV3(links, after.Provenance.CauseLinks)}
}

func showJSONV3(value repository.ShowResult) any {
	return struct {
		Schema      string                    `json:"schema"`
		Seal        sealViewJSONV3            `json:"seal"`
		CurrentREFs []string                  `json:"current_refs"`
		Revision    revisionObservationJSONV3 `json:"revision_observation"`
	}{"sealgraph/show/v3", sealViewV3(value.Resolved), append([]string{}, value.REFNames...), revisionJSONV3(value.Revision)}
}

func candidateShowJSONV3(value repository.CandidateInspection) any {
	var current *string
	if value.CurrentHead != nil {
		text := value.CurrentHead.String()
		current = &text
	}
	return struct {
		Schema            string                                `json:"schema"`
		Candidate         candidateViewJSONV3                   `json:"candidate"`
		CurrentREFHead    *string                               `json:"current_ref_head"`
		ExpectedHeadState repository.CandidateExpectedHeadState `json:"expected_head_state"`
	}{"sealgraph/candidate-show/v3", candidateViewV3(value), current, value.ExpectedHeadState}
}

func candidateCompareJSONV3(value repository.CandidateDiffResult) any {
	show := candidateShowJSONV3(value.Inspection).(struct {
		Schema            string                                `json:"schema"`
		Candidate         candidateViewJSONV3                   `json:"candidate"`
		CurrentREFHead    *string                               `json:"current_ref_head"`
		ExpectedHeadState repository.CandidateExpectedHeadState `json:"expected_head_state"`
	})
	baseline := struct {
		Label string `json:"label"`
		State string `json:"state"`
		Seal  any    `json:"seal"`
	}{"PUBLICATION_BASELINE", "ABSENT", nil}
	if value.Baseline != nil {
		baseline.State = "PRESENT"
		baseline.Seal = sealViewV3(*value.Baseline)
	}
	return struct {
		Schema            string                                `json:"schema"`
		REF               string                                `json:"ref"`
		CurrentREFHead    *string                               `json:"current_ref_head"`
		ExpectedHeadState repository.CandidateExpectedHeadState `json:"expected_head_state"`
		Baseline          any                                   `json:"baseline"`
		Prospective       candidateViewJSONV3                   `json:"prospective"`
		Changes           changesJSONV3                         `json:"changes"`
	}{"sealgraph/candidate-compare/v3", value.Inspection.Candidate.REF, show.CurrentREFHead, value.Inspection.ExpectedHeadState, baseline, candidateViewV3(value.Inspection), compareChangesV3(value.Baseline, value.Inspection.Prospective)}
}

func compareJSONV3(value repository.SealComparison) any {
	return struct {
		Schema  string         `json:"schema"`
		From    sealViewJSONV3 `json:"from"`
		To      sealViewJSONV3 `json:"to"`
		Changes changesJSONV3  `json:"changes"`
	}{"sealgraph/compare/v3", sealViewV3(value.From), sealViewV3(value.To), compareChangesV3(&value.From, value.To)}
}

type graphNodeJSONV3 struct {
	SealID   string                    `json:"seal_id"`
	State    repository.RevisionState  `json:"revision_state"`
	REFs     []string                  `json:"refs"`
	Revision revisionObservationJSONV3 `json:"revision_observation"`
	Causes   []graphCauseJSON          `json:"causes"`
}

func graphJSONV3(nodes []repository.GraphNode) any {
	items := make([]graphNodeJSONV3, 0, len(nodes))
	for _, node := range nodes {
		causes := []graphCauseJSON{}
		for _, cause := range node.Causes {
			causes = append(causes, graphCauseJSON{cause.Target.String(), cause.State})
		}
		items = append(items, graphNodeJSONV3{node.Resolved.ID.String(), node.State, append([]string{}, node.REFs...), revisionJSONV3(node.Revision), causes})
	}
	return struct {
		Schema string            `json:"schema"`
		Nodes  []graphNodeJSONV3 `json:"nodes"`
	}{"sealgraph/graph/v3", items}
}

type proofJSONV3 struct {
	SealIDs      []string                          `json:"seal_ids"`
	Edges        []revisionEdgeJSONV3              `json:"edges"`
	Observations []scopedRevisionObservationJSONV3 `json:"target_observations"`
}
type impactPathJSONV3 struct {
	CauseSealIDs []string    `json:"cause_seal_ids"`
	Matched      string      `json:"matched_revision_seal_id"`
	Proof        proofJSONV3 `json:"revision_proof"`
}
type impactRecordJSONV3 struct {
	Head      string             `json:"head_seal_id"`
	REFs      []string           `json:"refs"`
	Paths     []impactPathJSONV3 `json:"paths"`
	Truncated bool               `json:"paths_truncated"`
}

func proofValueV3(value repository.RevisionProof) proofJSONV3 {
	edges := []revisionEdgeJSONV3{}
	for _, edge := range value.Edges {
		edges = append(edges, edgeJSONV3(edge))
	}
	observations := []scopedRevisionObservationJSONV3{}
	for _, item := range value.Observations {
		observations = append(observations, scopedRevisionJSONV3(item))
	}
	return proofJSONV3{idsJSON(value.SealIDs), edges, observations}
}
func impactJSONV3(value repository.ImpactResult) any {
	items := []impactRecordJSONV3{}
	for _, impact := range value.Impacts {
		paths := []impactPathJSONV3{}
		for _, path := range impact.Paths {
			paths = append(paths, impactPathJSONV3{idsJSON(path.CauseSealIDs), path.MatchedRevision.String(), proofValueV3(path.Proof)})
		}
		items = append(items, impactRecordJSONV3{impact.Head.String(), append([]string{}, impact.REFs...), paths, impact.Truncated})
	}
	var max any
	if value.AllPaths {
		max = value.MaxPaths
	}
	return struct {
		Schema    string               `json:"schema"`
		Source    string               `json:"source_seal_id"`
		Scope     string               `json:"assertion_scope"`
		Observers []string             `json:"asserted_by_observer_seal_ids"`
		AllPaths  bool                 `json:"all_paths"`
		MaxPaths  any                  `json:"max_paths"`
		Impacts   []impactRecordJSONV3 `json:"impacts"`
	}{"sealgraph/impact/v3", value.Source.String(), value.AssertionScope, idsJSON(value.ObserverSealIDs), value.AllPaths, max, items}
}

type logEntryJSONV3 struct {
	MinimumDepth int                       `json:"minimum_depth"`
	Seal         sealViewJSONV3            `json:"seal"`
	Revision     revisionObservationJSONV3 `json:"revision_observation"`
	Edges        []revisionEdgeJSONV3      `json:"outgoing_revision_edges"`
}
type logPathJSONV3 struct {
	SealIDs []string             `json:"seal_ids"`
	Edges   []revisionEdgeJSONV3 `json:"edges"`
}

func logJSONV3(value repository.LogResult) any {
	entries := []logEntryJSONV3{}
	for _, entry := range value.Entries {
		edges := []revisionEdgeJSONV3{}
		for _, edge := range entry.OutgoingEdges {
			edges = append(edges, edgeJSONV3(edge))
		}
		entries = append(entries, logEntryJSONV3{entry.MinimumDepth, sealViewV3(entry.Resolved), revisionJSONV3(entry.Revision), edges})
	}
	paths := []logPathJSONV3{}
	for _, path := range value.Paths {
		edges := []revisionEdgeJSONV3{}
		for _, edge := range path.Edges {
			edges = append(edges, edgeJSONV3(edge))
		}
		paths = append(paths, logPathJSONV3{idsJSON(path.SealIDs), edges})
	}
	var max any
	if value.AllPaths {
		max = value.MaxPaths
	}
	return struct {
		Schema    string           `json:"schema"`
		REF       string           `json:"ref"`
		Head      string           `json:"head_seal_id"`
		AllPaths  bool             `json:"all_paths"`
		MaxPaths  any              `json:"max_paths"`
		Entries   []logEntryJSONV3 `json:"entries"`
		Paths     []logPathJSONV3  `json:"paths"`
		Truncated bool             `json:"paths_truncated"`
	}{"sealgraph/log/v3", value.REF, value.Head.String(), value.AllPaths, max, entries, paths, value.Truncated}
}

type linkLogEntryJSONV3 struct {
	Depth                    int                       `json:"minimum_newer_depth"`
	Newer                    string                    `json:"newer_seal_id"`
	NewerSealSchema          string                    `json:"newer_seal_schema"`
	NewerProvenanceID        string                    `json:"newer_provenance_id"`
	NewerProvenanceSchema    string                    `json:"newer_provenance_schema"`
	Previous                 string                    `json:"previous_seal_id"`
	PreviousSealSchema       string                    `json:"previous_seal_schema"`
	PreviousProvenanceID     string                    `json:"previous_provenance_id"`
	PreviousProvenanceSchema string                    `json:"previous_provenance_schema"`
	Sources                  []assertionJSONV6         `json:"supporting_assertions"`
	Observation              revisionObservationJSONV3 `json:"target_revision_observation"`
	Changes                  []causeLinkChangeJSONV3   `json:"changes"`
}

func linkLogJSONV3(value repository.LinkLogResult) any {
	entries := []linkLogEntryJSONV3{}
	for _, entry := range value.Entries {
		changes := []causeLinkChangeJSONV3{}
		for _, change := range entry.Changes {
			changes = append(changes, causeLinkChangeRecordV3(change.Target.String(), change.Before, change.After))
		}
		entries = append(entries, linkLogEntryJSONV3{entry.MinimumNewerDepth, entry.Newer.String(), entry.NewerSealSchema, entry.NewerProvenanceID, entry.NewerProvenanceSchema, entry.Previous.String(), entry.PreviousSealSchema, entry.PreviousProvenanceID, entry.PreviousProvenanceSchema, assertionsJSONV3(entry.SupportingAssertions), revisionJSONV3(entry.TargetRevision), changes})
	}
	var upstream any
	if value.Upstream != nil {
		upstream = value.Upstream.String()
	}
	return struct {
		Schema   string               `json:"schema"`
		REF      string               `json:"ref"`
		Head     string               `json:"head_seal_id"`
		Upstream any                  `json:"upstream_seal_id"`
		Entries  []linkLogEntryJSONV3 `json:"entries"`
	}{"sealgraph/linklog/v3", value.REF, value.Head.String(), upstream, entries}
}

type fsckDocumentV3 struct {
	Schema        string   `json:"schema"`
	Result        string   `json:"result"`
	Blobs         int      `json:"blobs"`
	Seals         int      `json:"seals"`
	SealsV5       int      `json:"seals_v5"`
	SealsV6       int      `json:"seals_v6"`
	Materials     int      `json:"materials"`
	Provenances   int      `json:"provenances"`
	ProvenancesV1 int      `json:"provenances_v1"`
	ProvenancesV2 int      `json:"provenances_v2"`
	CandidatesV5  int      `json:"candidates_v5"`
	CandidatesV6  int      `json:"candidates_v6"`
	REFs          int      `json:"refs"`
	Tags          int      `json:"tags"`
	Active        int      `json:"active_seals"`
	Historical    []string `json:"historical_or_detached_seal_ids"`
	Unreferenced  []string `json:"unreferenced_blob_ids"`
}

func fsckJSONV3(value repository.FsckReport) fsckDocumentV3 {
	return fsckDocumentV3{"sealgraph/fsck/v3", "ok", value.Blobs, value.Seals, value.SealsV5, value.SealsV6, value.Materials, value.Provenances, value.ProvenancesV1, value.ProvenancesV2, value.CandidatesV5, value.CandidatesV6, value.REFs, value.Tags, value.ActiveSeals, idsJSON(value.HistoricalOrDetachedSeals), idsJSON(value.UnreferencedBlobs)}
}

func formatAwareJSON(format int, v2, v3 any) any {
	if format == 6 {
		return v3
	}
	return v2
}
