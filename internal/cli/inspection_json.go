package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"strings"

	"github.com/mako10k/sealgraph/internal/domain"
	domainv5 "github.com/mako10k/sealgraph/internal/domain/v5"
	"github.com/mako10k/sealgraph/internal/repository"
)

type inspectionOutput struct{ JSON, Explicit bool }

func extractInspectionFormat(args []string, stdout io.Writer) ([]string, inspectionOutput, error) {
	result := make([]string, 0, len(args))
	format := ""
	seen := false
	for i := 0; i < len(args); i++ {
		value := args[i]
		if value != "--format" && !strings.HasPrefix(value, "--format=") {
			result = append(result, value)
			continue
		}
		if seen {
			return nil, inspectionOutput{}, fmt.Errorf("--format may be specified only once")
		}
		seen = true
		if value == "--format" {
			i++
			if i == len(args) {
				return nil, inspectionOutput{}, fmt.Errorf("--format requires human or json")
			}
			format = args[i]
		} else {
			format = strings.TrimPrefix(value, "--format=")
		}
		if format != "human" && format != "json" {
			return nil, inspectionOutput{}, fmt.Errorf("unsupported inspection format %q; expected human or json", format)
		}
	}
	if seen {
		return result, inspectionOutput{JSON: format == "json", Explicit: true}, nil
	}
	terminal, _, known := outputTerminalInfo(stdout)
	return result, inspectionOutput{JSON: known && !terminal}, nil
}

func writeInspectionJSON(stdout, stderr io.Writer, command string, value any) int {
	var buffer bytes.Buffer
	encoder := json.NewEncoder(&buffer)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(value); err != nil {
		return commandError(stderr, command, fmt.Errorf("encode JSON output: %w", err))
	}
	if _, err := stdout.Write(literalizeInspectionUnicodeSeparators(buffer.Bytes())); err != nil {
		return commandError(stderr, command, fmt.Errorf("write JSON output: %w", err))
	}
	return 0
}

// encoding/json always escapes U+2028 and U+2029 for JavaScript compatibility.
// Format-5 inspection JSON instead uses the canonical Sealgraph string rule:
// every non-control scalar is emitted as its shortest literal UTF-8 sequence.
func literalizeInspectionUnicodeSeparators(data []byte) []byte {
	result := make([]byte, 0, len(data))
	inString := false
	for index := 0; index < len(data); {
		current := data[index]
		if !inString {
			result = append(result, current)
			index++
			if current == '"' {
				inString = true
			}
			continue
		}
		if current == '"' {
			result = append(result, current)
			index++
			inString = false
			continue
		}
		if current != '\\' {
			result = append(result, current)
			index++
			continue
		}
		if bytes.HasPrefix(data[index:], []byte(`\u2028`)) {
			result = append(result, []byte("\u2028")...)
			index += len(`\u2028`)
			continue
		}
		if bytes.HasPrefix(data[index:], []byte(`\u2029`)) {
			result = append(result, []byte("\u2029")...)
			index += len(`\u2029`)
			continue
		}
		result = append(result, current)
		index++
		if index < len(data) {
			result = append(result, data[index])
			index++
		}
	}
	return result
}

type contentV1 struct {
	Store    string `json:"store"`
	Type     string `json:"type"`
	ObjectID string `json:"object_id"`
}

func contentJSON(ref domain.ContentRef) contentV1 {
	return contentV1{Store: ref.Store, Type: ref.Type, ObjectID: ref.ID.String()}
}

type attachmentJSONV5 struct {
	Name      string `json:"name"`
	MediaType string `json:"media_type"`
	Blob      string `json:"blob"`
}
type causeLinkJSONV5 struct {
	TargetSeal string   `json:"target_seal"`
	Previous   []string `json:"previous_revision_seal_of_target_seal"`
	Messages   []string `json:"messages"`
}
type assertionJSONV5 struct {
	ObserverSeal       string   `json:"observer_seal"`
	ObserverProvenance string   `json:"observer_provenance"`
	TargetSeal         string   `json:"target_seal"`
	Previous           []string `json:"previous_revision_seal_of_target_seal"`
	Messages           []string `json:"messages"`
}
type revisionObservationJSON struct {
	TargetSeal         string            `json:"target_seal"`
	PreviousStates     []string          `json:"previous_states"`
	Assertions         []assertionJSONV5 `json:"assertions"`
	StructuralPrevious []string          `json:"structural_previous_seals"`
}
type scopedRevisionObservationJSON struct {
	TargetSeal         string            `json:"target_seal"`
	PreviousStates     []string          `json:"in_scope_previous_states"`
	Assertions         []assertionJSONV5 `json:"in_scope_assertions"`
	StructuralPrevious []string          `json:"structural_previous_seals"`
}
type revisionEdgeJSON struct {
	TargetSeal string            `json:"target_seal"`
	Previous   string            `json:"previous_revision_seal"`
	Sources    []assertionJSONV5 `json:"assertion_sources"`
}
type sealViewJSON struct {
	SealID        string             `json:"seal_id"`
	MaterialID    string             `json:"material_id"`
	ProvenanceID  string             `json:"provenance_id"`
	ContentBlobID string             `json:"content_blob_id"`
	ContentBytes  int                `json:"content_bytes"`
	Attachments   []attachmentJSONV5 `json:"attachments"`
	Root          bool               `json:"root"`
	Draft         bool               `json:"draft"`
	CauseLinks    []causeLinkJSONV5  `json:"cause_links"`
}
type candidateViewJSON struct {
	REF                     string             `json:"ref"`
	ExpectedREFHead         *string            `json:"expected_ref_head"`
	ProspectiveSealID       string             `json:"prospective_seal_id"`
	ProspectiveMaterialID   string             `json:"prospective_material_id"`
	ProspectiveProvenanceID string             `json:"prospective_provenance_id"`
	ContentBlobID           string             `json:"content_blob_id"`
	ContentBytes            int                `json:"content_bytes"`
	Attachments             []attachmentJSONV5 `json:"attachments"`
	Root                    bool               `json:"root"`
	Draft                   bool               `json:"draft"`
	CauseLinks              []causeLinkJSONV5  `json:"cause_links"`
}

func idsJSON(ids []domain.ObjectID) []string {
	result := make([]string, 0, len(ids))
	for _, id := range ids {
		result = append(result, id.String())
	}
	return result
}
func attachmentsJSON(values []domainv5.Attachment) []attachmentJSONV5 {
	result := make([]attachmentJSONV5, 0, len(values))
	for _, value := range values {
		result = append(result, attachmentJSONV5{Name: value.Name, MediaType: value.MediaType, Blob: value.Blob.String()})
	}
	return result
}
func causeLinksJSON(values []domainv5.CauseLink) []causeLinkJSONV5 {
	result := make([]causeLinkJSONV5, 0, len(values))
	for _, value := range values {
		result = append(result, causeLinkJSONV5{TargetSeal: value.TargetSeal.String(), Previous: idsJSON(value.PreviousRevisionSealOfTargetSeal), Messages: append([]string{}, value.Messages...)})
	}
	return result
}
func assertionsJSON(values []domainv5.AssertionSource) []assertionJSONV5 {
	result := make([]assertionJSONV5, 0, len(values))
	for _, value := range values {
		result = append(result, assertionJSONV5{ObserverSeal: value.ObserverSeal.String(), ObserverProvenance: value.ObserverProvenance.String(), TargetSeal: value.CauseLink.TargetSeal.String(), Previous: idsJSON(value.CauseLink.PreviousRevisionSealOfTargetSeal), Messages: append([]string{}, value.CauseLink.Messages...)})
	}
	return result
}
func revisionJSON(value domainv5.RevisionObservation) revisionObservationJSON {
	return revisionObservationJSON{TargetSeal: value.TargetSeal.String(), PreviousStates: append([]string{}, value.PreviousStates...), Assertions: assertionsJSON(value.Assertions), StructuralPrevious: idsJSON(value.StructuralPrevious)}
}
func scopedRevisionJSON(value repository.ScopedRevisionObservation) scopedRevisionObservationJSON {
	return scopedRevisionObservationJSON{TargetSeal: value.TargetSeal.String(), PreviousStates: append([]string{}, value.PreviousStates...), Assertions: assertionsJSON(value.Assertions), StructuralPrevious: idsJSON(value.StructuralPrevious)}
}
func edgeJSON(value repository.RevisionEdge) revisionEdgeJSON {
	return revisionEdgeJSON{TargetSeal: value.Target.String(), Previous: value.Previous.String(), Sources: assertionsJSON(value.Sources)}
}
func sealView(value domainv5.ResolvedSeal) sealViewJSON {
	return sealViewJSON{SealID: value.ID.String(), MaterialID: value.Seal.Material.String(), ProvenanceID: value.Seal.Provenance.String(), ContentBlobID: value.Material.Content.String(), ContentBytes: value.ContentBytes, Attachments: attachmentsJSON(value.Material.Attachments), Root: value.Provenance.Root, Draft: value.Provenance.Draft, CauseLinks: causeLinksJSON(value.Provenance.CauseLinks)}
}
func candidateView(value repository.CandidateInspection) candidateViewJSON {
	var expected *string
	if value.Candidate.ExpectedREFHead != nil {
		text := value.Candidate.ExpectedREFHead.String()
		expected = &text
	}
	return candidateViewJSON{REF: value.Candidate.REF, ExpectedREFHead: expected, ProspectiveSealID: value.Prospective.ID.String(), ProspectiveMaterialID: value.Prospective.Seal.Material.String(), ProspectiveProvenanceID: value.Prospective.Seal.Provenance.String(), ContentBlobID: value.Candidate.Content.String(), ContentBytes: len(value.Content), Attachments: attachmentsJSON(value.Candidate.Attachments), Root: value.Candidate.Root, Draft: value.Candidate.Draft, CauseLinks: causeLinksJSON(value.Candidate.CauseLinks)}
}

type showDocument struct {
	Schema      string                  `json:"schema"`
	Seal        sealViewJSON            `json:"seal"`
	CurrentREFs []string                `json:"current_refs"`
	Revision    revisionObservationJSON `json:"revision_observation"`
}

func showJSON(result repository.ShowResult) showDocument {
	return showDocument{Schema: "sealgraph/show/v2", Seal: sealView(result.Resolved), CurrentREFs: append([]string{}, result.REFNames...), Revision: revisionJSON(result.Revision)}
}

type candidateShowDocument struct {
	Schema            string                                `json:"schema"`
	Candidate         candidateViewJSON                     `json:"candidate"`
	CurrentREFHead    *string                               `json:"current_ref_head"`
	ExpectedHeadState repository.CandidateExpectedHeadState `json:"expected_head_state"`
}

func candidateShowJSON(value repository.CandidateInspection) candidateShowDocument {
	var current *string
	if value.CurrentHead != nil {
		text := value.CurrentHead.String()
		current = &text
	}
	return candidateShowDocument{Schema: "sealgraph/candidate-show/v2", Candidate: candidateView(value), CurrentREFHead: current, ExpectedHeadState: value.ExpectedHeadState}
}

type changeJSON struct {
	Changed bool `json:"changed"`
	Before  any  `json:"before"`
	After   any  `json:"after"`
}
type changesJSON struct {
	MaterialID    changeJSON `json:"material_id"`
	ProvenanceID  changeJSON `json:"provenance_id"`
	ContentBlobID changeJSON `json:"content_blob_id"`
	Attachments   changeJSON `json:"attachments"`
	Root          changeJSON `json:"root"`
	Draft         changeJSON `json:"draft"`
	CauseLinks    changeJSON `json:"cause_links"`
}

func changed(before, after any) changeJSON {
	return changeJSON{Changed: !reflect.DeepEqual(before, after), Before: before, After: after}
}
func compareChanges(before *domainv5.ResolvedSeal, after domainv5.ResolvedSeal) changesJSON {
	var material, provenance, content, attachments, root, draft, links any
	if before != nil {
		material = before.Seal.Material.String()
		provenance = before.Seal.Provenance.String()
		content = before.Material.Content.String()
		attachments = attachmentsJSON(before.Material.Attachments)
		root = before.Provenance.Root
		draft = before.Provenance.Draft
		links = causeLinksJSON(before.Provenance.CauseLinks)
	}
	return changesJSON{MaterialID: changed(material, after.Seal.Material.String()), ProvenanceID: changed(provenance, after.Seal.Provenance.String()), ContentBlobID: changed(content, after.Material.Content.String()), Attachments: changed(attachments, attachmentsJSON(after.Material.Attachments)), Root: changed(root, after.Provenance.Root), Draft: changed(draft, after.Provenance.Draft), CauseLinks: changed(links, causeLinksJSON(after.Provenance.CauseLinks))}
}

type baselineJSON struct {
	Label string `json:"label"`
	State string `json:"state"`
	Seal  any    `json:"seal"`
}
type candidateCompareDocument struct {
	Schema            string                                `json:"schema"`
	REF               string                                `json:"ref"`
	CurrentREFHead    *string                               `json:"current_ref_head"`
	ExpectedHeadState repository.CandidateExpectedHeadState `json:"expected_head_state"`
	Baseline          baselineJSON                          `json:"baseline"`
	Prospective       candidateViewJSON                     `json:"prospective"`
	Changes           changesJSON                           `json:"changes"`
}

func candidateCompareJSON(value repository.CandidateDiffResult) candidateCompareDocument {
	show := candidateShowJSON(value.Inspection)
	baseline := baselineJSON{Label: "PUBLICATION_BASELINE", State: "ABSENT"}
	if value.Baseline != nil {
		baseline.State = "PRESENT"
		baseline.Seal = sealView(*value.Baseline)
	}
	return candidateCompareDocument{Schema: "sealgraph/candidate-compare/v2", REF: value.Inspection.Candidate.REF, CurrentREFHead: show.CurrentREFHead, ExpectedHeadState: value.Inspection.ExpectedHeadState, Baseline: baseline, Prospective: candidateView(value.Inspection), Changes: compareChanges(value.Baseline, value.Inspection.Prospective)}
}

type compareDocument struct {
	Schema  string       `json:"schema"`
	From    sealViewJSON `json:"from"`
	To      sealViewJSON `json:"to"`
	Changes changesJSON  `json:"changes"`
}

func compareJSON(value repository.SealComparison) compareDocument {
	return compareDocument{Schema: "sealgraph/compare/v2", From: sealView(value.From), To: sealView(value.To), Changes: compareChanges(&value.From, value.To)}
}

type staleJSON struct {
	Self       bool       `json:"self"`
	Direct     []string   `json:"direct_target_seal_ids"`
	Transitive [][]string `json:"transitive_paths"`
}
type localSourceJSON struct {
	Path     string `json:"path"`
	Baseline string `json:"baseline"`
	Relation string `json:"relation"`
}
type statusRecordJSON struct {
	REF         string           `json:"ref"`
	Head        *string          `json:"head_seal_id"`
	Candidate   string           `json:"candidate_to_head"`
	Draft       bool             `json:"draft"`
	Stale       staleJSON        `json:"stale"`
	Labels      []string         `json:"sealed_state_labels"`
	LocalSource *localSourceJSON `json:"local_source"`
}
type staleStatusRecordJSON struct {
	REF    string    `json:"ref"`
	Head   string    `json:"head_seal_id"`
	Draft  bool      `json:"draft"`
	Stale  staleJSON `json:"stale"`
	Labels []string  `json:"sealed_state_labels"`
}

func statusStale(value repository.RefStatus) staleJSON {
	paths := make([][]string, 0, len(value.StaleTransitive))
	for _, path := range value.StaleTransitive {
		paths = append(paths, idsJSON(path))
	}
	return staleJSON{Self: value.StaleSelf, Direct: idsJSON(value.StaleDirect), Transitive: paths}
}
func labels(value repository.RefStatus) []string {
	result := value.Labels()
	for i := range result {
		if result[i] == "CLEAN" {
			result[i] = "SEALED_STATE_CLEAN"
		}
	}
	return result
}
func statusRecord(value repository.RefStatus) statusRecordJSON {
	var head *string
	if value.Head != nil {
		text := value.Head.String()
		head = &text
	}
	candidate := "NO_CANDIDATE"
	if value.Unsealed {
		candidate = "UNSEALED"
	}
	var source *localSourceJSON
	if value.Source != nil {
		source = &localSourceJSON{Path: value.Source.Path, Baseline: value.Source.Baseline, Relation: value.Source.Relation}
	}
	return statusRecordJSON{REF: value.REF, Head: head, Candidate: candidate, Draft: value.Draft, Stale: statusStale(value), Labels: labels(value), LocalSource: source}
}
func staleRecord(value repository.RefStatus) staleStatusRecordJSON {
	return staleStatusRecordJSON{REF: value.REF, Head: value.Head.String(), Draft: value.Draft, Stale: statusStale(value), Labels: labels(value)}
}

type statusDocument struct {
	Schema   string             `json:"schema"`
	Statuses []statusRecordJSON `json:"statuses"`
}
type staleDocument struct {
	Schema   string                  `json:"schema"`
	Frontier bool                    `json:"frontier"`
	Scan     bool                    `json:"scan"`
	Statuses []staleStatusRecordJSON `json:"statuses"`
}

func statusesJSON(schema string, statuses []repository.RefStatus, frontier, scan bool) any {
	if schema == "sealgraph/status/v3" {
		items := make([]statusRecordJSON, 0, len(statuses))
		for _, value := range statuses {
			items = append(items, statusRecord(value))
		}
		return statusDocument{Schema: schema, Statuses: items}
	}
	items := make([]staleStatusRecordJSON, 0, len(statuses))
	for _, value := range statuses {
		items = append(items, staleRecord(value))
	}
	return staleDocument{Schema: schema, Frontier: frontier, Scan: scan, Statuses: items}
}

type graphCauseJSON struct {
	Target string                   `json:"target_seal_id"`
	State  repository.RevisionState `json:"revision_state"`
}
type graphNodeJSON struct {
	SealID   string                   `json:"seal_id"`
	State    repository.RevisionState `json:"revision_state"`
	REFs     []string                 `json:"refs"`
	Revision revisionObservationJSON  `json:"revision_observation"`
	Causes   []graphCauseJSON         `json:"causes"`
}
type graphDocument struct {
	Schema string          `json:"schema"`
	Nodes  []graphNodeJSON `json:"nodes"`
}

func graphJSON(nodes []repository.GraphNode) graphDocument {
	items := make([]graphNodeJSON, 0, len(nodes))
	for _, node := range nodes {
		causes := make([]graphCauseJSON, 0, len(node.Causes))
		for _, cause := range node.Causes {
			causes = append(causes, graphCauseJSON{Target: cause.Target.String(), State: cause.State})
		}
		items = append(items, graphNodeJSON{SealID: node.Resolved.ID.String(), State: node.State, REFs: append([]string{}, node.REFs...), Revision: revisionJSON(node.Revision), Causes: causes})
	}
	return graphDocument{Schema: "sealgraph/graph/v2", Nodes: items}
}

type proofJSON struct {
	SealIDs      []string                        `json:"seal_ids"`
	Edges        []revisionEdgeJSON              `json:"edges"`
	Observations []scopedRevisionObservationJSON `json:"target_observations"`
}
type impactPathJSON struct {
	CauseSealIDs []string  `json:"cause_seal_ids"`
	Matched      string    `json:"matched_revision_seal_id"`
	Proof        proofJSON `json:"revision_proof"`
}
type impactRecordJSON struct {
	Head      string           `json:"head_seal_id"`
	REFs      []string         `json:"refs"`
	Paths     []impactPathJSON `json:"paths"`
	Truncated bool             `json:"paths_truncated"`
}
type impactDocument struct {
	Schema    string             `json:"schema"`
	Source    string             `json:"source_seal_id"`
	Scope     string             `json:"assertion_scope"`
	Observers []string           `json:"asserted_by_observer_seal_ids"`
	AllPaths  bool               `json:"all_paths"`
	MaxPaths  any                `json:"max_paths"`
	Impacts   []impactRecordJSON `json:"impacts"`
}

func proofValue(value repository.RevisionProof) proofJSON {
	edges := make([]revisionEdgeJSON, 0, len(value.Edges))
	for _, edge := range value.Edges {
		edges = append(edges, edgeJSON(edge))
	}
	observations := make([]scopedRevisionObservationJSON, 0, len(value.Observations))
	for _, observation := range value.Observations {
		observations = append(observations, scopedRevisionJSON(observation))
	}
	return proofJSON{SealIDs: idsJSON(value.SealIDs), Edges: edges, Observations: observations}
}
func impactJSON(value repository.ImpactResult) impactDocument {
	items := make([]impactRecordJSON, 0, len(value.Impacts))
	for _, impact := range value.Impacts {
		paths := make([]impactPathJSON, 0, len(impact.Paths))
		for _, path := range impact.Paths {
			paths = append(paths, impactPathJSON{CauseSealIDs: idsJSON(path.CauseSealIDs), Matched: path.MatchedRevision.String(), Proof: proofValue(path.Proof)})
		}
		items = append(items, impactRecordJSON{Head: impact.Head.String(), REFs: append([]string{}, impact.REFs...), Paths: paths, Truncated: impact.Truncated})
	}
	var max any
	if value.AllPaths {
		max = value.MaxPaths
	}
	return impactDocument{Schema: "sealgraph/impact/v2", Source: value.Source.String(), Scope: value.AssertionScope, Observers: idsJSON(value.ObserverSealIDs), AllPaths: value.AllPaths, MaxPaths: max, Impacts: items}
}

type logEntryJSON struct {
	MinimumDepth int                     `json:"minimum_depth"`
	Seal         sealViewJSON            `json:"seal"`
	Revision     revisionObservationJSON `json:"revision_observation"`
	Edges        []revisionEdgeJSON      `json:"outgoing_revision_edges"`
}
type logPathJSON struct {
	SealIDs []string           `json:"seal_ids"`
	Edges   []revisionEdgeJSON `json:"edges"`
}
type logDocument struct {
	Schema    string         `json:"schema"`
	REF       string         `json:"ref"`
	Head      string         `json:"head_seal_id"`
	AllPaths  bool           `json:"all_paths"`
	MaxPaths  any            `json:"max_paths"`
	Entries   []logEntryJSON `json:"entries"`
	Paths     []logPathJSON  `json:"paths"`
	Truncated bool           `json:"paths_truncated"`
}

func logJSON(value repository.LogResult) logDocument {
	entries := make([]logEntryJSON, 0, len(value.Entries))
	for _, entry := range value.Entries {
		edges := make([]revisionEdgeJSON, 0, len(entry.OutgoingEdges))
		for _, edge := range entry.OutgoingEdges {
			edges = append(edges, edgeJSON(edge))
		}
		entries = append(entries, logEntryJSON{MinimumDepth: entry.MinimumDepth, Seal: sealView(entry.Resolved), Revision: revisionJSON(entry.Revision), Edges: edges})
	}
	paths := make([]logPathJSON, 0, len(value.Paths))
	for _, path := range value.Paths {
		edges := make([]revisionEdgeJSON, 0, len(path.Edges))
		for _, edge := range path.Edges {
			edges = append(edges, edgeJSON(edge))
		}
		paths = append(paths, logPathJSON{SealIDs: idsJSON(path.SealIDs), Edges: edges})
	}
	var max any
	if value.AllPaths {
		max = value.MaxPaths
	}
	return logDocument{Schema: "sealgraph/log/v2", REF: value.REF, Head: value.Head.String(), AllPaths: value.AllPaths, MaxPaths: max, Entries: entries, Paths: paths, Truncated: value.Truncated}
}

type causeChangeJSON struct {
	Target string `json:"target_seal"`
	Before any    `json:"before"`
	After  any    `json:"after"`
}
type linkLogEntryJSON struct {
	Depth       int                     `json:"minimum_newer_depth"`
	Newer       string                  `json:"newer_seal_id"`
	Previous    string                  `json:"previous_seal_id"`
	Sources     []assertionJSONV5       `json:"supporting_assertions"`
	Observation revisionObservationJSON `json:"target_revision_observation"`
	Changes     []causeChangeJSON       `json:"changes"`
}
type linkLogDocument struct {
	Schema   string             `json:"schema"`
	REF      string             `json:"ref"`
	Head     string             `json:"head_seal_id"`
	Upstream any                `json:"upstream_seal_id"`
	Entries  []linkLogEntryJSON `json:"entries"`
}

func linkLogJSON(value repository.LinkLogResult) linkLogDocument {
	entries := make([]linkLogEntryJSON, 0, len(value.Entries))
	for _, entry := range value.Entries {
		changes := make([]causeChangeJSON, 0, len(entry.Changes))
		for _, change := range entry.Changes {
			var before, after any
			if change.Before != nil {
				before = causeLinksJSON([]domainv5.CauseLink{*change.Before})[0]
			}
			if change.After != nil {
				after = causeLinksJSON([]domainv5.CauseLink{*change.After})[0]
			}
			changes = append(changes, causeChangeJSON{Target: change.Target.String(), Before: before, After: after})
		}
		entries = append(entries, linkLogEntryJSON{Depth: entry.MinimumNewerDepth, Newer: entry.Newer.String(), Previous: entry.Previous.String(), Sources: assertionsJSON(entry.SupportingAssertions), Observation: revisionJSON(entry.TargetRevision), Changes: changes})
	}
	var upstream any
	if value.Upstream != nil {
		upstream = value.Upstream.String()
	}
	return linkLogDocument{Schema: "sealgraph/linklog/v2", REF: value.REF, Head: value.Head.String(), Upstream: upstream, Entries: entries}
}

type fsckDocument struct {
	Schema       string   `json:"schema"`
	Result       string   `json:"result"`
	Blobs        int      `json:"blobs"`
	Seals        int      `json:"seals"`
	Materials    int      `json:"materials"`
	Provenances  int      `json:"provenances"`
	REFs         int      `json:"refs"`
	Tags         int      `json:"tags"`
	Active       int      `json:"active_seals"`
	Historical   []string `json:"historical_or_detached_seal_ids"`
	Unreferenced []string `json:"unreferenced_blob_ids"`
}

func fsckJSON(value repository.FsckReport) fsckDocument {
	return fsckDocument{Schema: "sealgraph/fsck/v2", Result: "ok", Blobs: value.Blobs, Seals: value.Seals, Materials: value.Materials, Provenances: value.Provenances, REFs: value.REFs, Tags: value.Tags, Active: value.ActiveSeals, Historical: idsJSON(value.HistoricalOrDetachedSeals), Unreferenced: idsJSON(value.UnreferencedBlobs)}
}

func sourceCompareJSON(value repository.SourceCompareResult) any {
	var baseline any
	if value.BaselineContent != nil {
		baseline = contentJSON(*value.BaselineContent)
	}
	return struct {
		Schema          string `json:"schema"`
		REF             string `json:"ref"`
		Path            string `json:"path"`
		Baseline        string `json:"baseline"`
		BaselineContent any    `json:"baseline_content"`
		WorkfileContent any    `json:"workfile_content"`
		Relation        string `json:"relation"`
	}{"sealgraph/source-compare/v1", value.REF, value.Path, value.Baseline, baseline, struct {
		Store    string `json:"store"`
		Type     string `json:"type"`
		ObjectID string `json:"object_id"`
		Bytes    int    `json:"bytes"`
	}{domain.NativeStore, domain.BlobType, value.WorkfileID.String(), value.WorkfileBytes}, value.Relation}
}
func sourceJSON(operation string, bindings []repository.SourceBinding) any {
	type item struct {
		REF  string `json:"ref"`
		Path string `json:"path"`
	}
	items := make([]item, 0, len(bindings))
	for _, binding := range bindings {
		items = append(items, item{binding.REF, binding.Path})
	}
	return struct {
		Schema    string `json:"schema"`
		Operation string `json:"operation"`
		Bindings  []item `json:"bindings"`
	}{"sealgraph/source/v1", operation, items}
}
func sourceMutationJSON(operation, ref, before, after string) any {
	var beforeValue, afterValue any
	if before != "" {
		beforeValue = before
	}
	if after != "" {
		afterValue = after
	}
	return struct {
		Schema    string `json:"schema"`
		Operation string `json:"operation"`
		REF       string `json:"ref"`
		Before    any    `json:"before_path"`
		After     any    `json:"after_path"`
		Candidate string `json:"candidate"`
	}{"sealgraph/source/v1", operation, ref, beforeValue, afterValue, "UNCHANGED"}
}
func recoveryInspectionsJSON(values []repository.RecoveryInspection) any {
	type transition struct {
		REF     string `json:"ref"`
		Current string `json:"current"`
	}
	type item struct {
		ID          string       `json:"operation_id"`
		Kind        string       `json:"kind"`
		Journal     string       `json:"journal_state"`
		Status      string       `json:"status"`
		Transitions []transition `json:"transitions"`
		Error       string       `json:"error"`
	}
	items := make([]item, 0, len(values))
	for _, value := range values {
		transitions := make([]transition, 0, len(value.Transitions))
		for _, current := range value.Transitions {
			transitions = append(transitions, transition{current.REF, current.Current})
		}
		items = append(items, item{value.ID, value.Kind, string(value.Journal), value.Status, transitions, value.Corrupt})
	}
	return struct {
		Schema     string `json:"schema"`
		Operations []item `json:"operations"`
	}{"sealgraph/recover/v1", items}
}
