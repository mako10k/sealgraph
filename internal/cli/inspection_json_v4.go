package cli

import (
	"github.com/mako10k/sealgraph/internal/domain"
	domainv5 "github.com/mako10k/sealgraph/internal/domain/v5"
	"github.com/mako10k/sealgraph/internal/repository"
)

// Format 7 keeps every v3 member in its existing order and appends origin
// identity to the two views that can carry it. Origin is a nullable identity;
// old format-5/6 Seals and Candidates therefore serialize it as null.
type sealViewJSONV4 struct {
	sealViewJSONV3
	OriginMapID *string `json:"origin_map_id"`
}

type candidateViewJSONV4 struct {
	candidateViewJSONV3
	OriginMapID *string `json:"origin_map_id"`
}

func nullableOriginID(id *domain.ObjectID) *string {
	if id == nil {
		return nil
	}
	value := id.String()
	return &value
}

func sealViewV4(value domainv5.ResolvedSeal) sealViewJSONV4 {
	return sealViewJSONV4{sealViewV3(value), nullableOriginID(value.Provenance.Origin)}
}

func candidateViewV4(value repository.CandidateInspection) candidateViewJSONV4 {
	return candidateViewJSONV4{candidateViewV3(value), nullableOriginID(value.Candidate.Origin)}
}

func originChangeValue(id *domain.ObjectID) any {
	if id == nil {
		return nil
	}
	return id.String()
}

type changesJSONV4 struct {
	MaterialID    changeJSON             `json:"material_id"`
	ProvenanceID  changeJSON             `json:"provenance_id"`
	ContentBlobID changeJSON             `json:"content_blob_id"`
	Attachments   changeJSON             `json:"attachments"`
	Root          changeJSON             `json:"root"`
	Draft         changeJSON             `json:"draft"`
	CauseLinks    causeLinksChangeJSONV3 `json:"cause_links"`
	Origin        changeJSON             `json:"origin"`
}

func compareChangesV4(before *domainv5.ResolvedSeal, after domainv5.ResolvedSeal) changesJSONV4 {
	base := compareChangesV3(before, after)
	var old any
	if before != nil {
		old = originChangeValue(before.Provenance.Origin)
	}
	return changesJSONV4{
		base.MaterialID, base.ProvenanceID, base.ContentBlobID, base.Attachments, base.Root, base.Draft, base.CauseLinks,
		changed(old, originChangeValue(after.Provenance.Origin)),
	}
}

func showJSONV4(value repository.ShowResult) any {
	return struct {
		Schema      string                    `json:"schema"`
		Seal        sealViewJSONV4            `json:"seal"`
		CurrentREFs []string                  `json:"current_refs"`
		Revision    revisionObservationJSONV3 `json:"revision_observation"`
	}{"sealgraph/show/v4", sealViewV4(value.Resolved), append([]string{}, value.REFNames...), revisionJSONV3(value.Revision)}
}

func candidateShowJSONV4(value repository.CandidateInspection) any {
	var current *string
	if value.CurrentHead != nil {
		text := value.CurrentHead.String()
		current = &text
	}
	return struct {
		Schema            string                                `json:"schema"`
		Candidate         candidateViewJSONV4                   `json:"candidate"`
		CurrentREFHead    *string                               `json:"current_ref_head"`
		ExpectedHeadState repository.CandidateExpectedHeadState `json:"expected_head_state"`
	}{"sealgraph/candidate-show/v4", candidateViewV4(value), current, value.ExpectedHeadState}
}

func candidateCompareJSONV4(value repository.CandidateDiffResult) any {
	var current *string
	if value.Inspection.CurrentHead != nil {
		text := value.Inspection.CurrentHead.String()
		current = &text
	}
	baseline := struct {
		Label string `json:"label"`
		State string `json:"state"`
		Seal  any    `json:"seal"`
	}{"PUBLICATION_BASELINE", "ABSENT", nil}
	if value.Baseline != nil {
		baseline.State = "PRESENT"
		baseline.Seal = sealViewV4(*value.Baseline)
	}
	return struct {
		Schema            string                                `json:"schema"`
		REF               string                                `json:"ref"`
		CurrentREFHead    *string                               `json:"current_ref_head"`
		ExpectedHeadState repository.CandidateExpectedHeadState `json:"expected_head_state"`
		Baseline          any                                   `json:"baseline"`
		Prospective       candidateViewJSONV4                   `json:"prospective"`
		Changes           changesJSONV4                         `json:"changes"`
	}{"sealgraph/candidate-compare/v4", value.Inspection.Candidate.REF, current, value.Inspection.ExpectedHeadState, baseline, candidateViewV4(value.Inspection), compareChangesV4(value.Baseline, value.Inspection.Prospective)}
}

func compareJSONV4(value repository.SealComparison) any {
	return struct {
		Schema  string         `json:"schema"`
		From    sealViewJSONV4 `json:"from"`
		To      sealViewJSONV4 `json:"to"`
		Changes changesJSONV4  `json:"changes"`
	}{"sealgraph/compare/v4", sealViewV4(value.From), sealViewV4(value.To), compareChangesV4(&value.From, value.To)}
}

func graphJSONV4(nodes []repository.GraphNode) any {
	items := make([]struct {
		SealID   string                    `json:"seal_id"`
		State    repository.RevisionState  `json:"revision_state"`
		REFs     []string                  `json:"refs"`
		Revision revisionObservationJSONV3 `json:"revision_observation"`
		Causes   []graphCauseJSON          `json:"causes"`
	}, 0, len(nodes))
	for _, node := range nodes {
		causes := make([]graphCauseJSON, 0, len(node.Causes))
		for _, cause := range node.Causes {
			causes = append(causes, graphCauseJSON{cause.Target.String(), cause.State})
		}
		items = append(items, struct {
			SealID   string                    `json:"seal_id"`
			State    repository.RevisionState  `json:"revision_state"`
			REFs     []string                  `json:"refs"`
			Revision revisionObservationJSONV3 `json:"revision_observation"`
			Causes   []graphCauseJSON          `json:"causes"`
		}{node.Resolved.ID.String(), node.State, append([]string{}, node.REFs...), revisionJSONV3(node.Revision), causes})
	}
	return struct {
		Schema string `json:"schema"`
		Nodes  any    `json:"nodes"`
	}{"sealgraph/graph/v4", items}
}

func impactJSONV4(value repository.ImpactResult) any {
	base := impactJSONV3(value).(struct {
		Schema    string               `json:"schema"`
		Source    string               `json:"source_seal_id"`
		Scope     string               `json:"assertion_scope"`
		Observers []string             `json:"asserted_by_observer_seal_ids"`
		AllPaths  bool                 `json:"all_paths"`
		MaxPaths  any                  `json:"max_paths"`
		Impacts   []impactRecordJSONV3 `json:"impacts"`
	})
	base.Schema = "sealgraph/impact/v4"
	return base
}

func logJSONV4(value repository.LogResult) any {
	base := logJSONV3(value).(struct {
		Schema    string           `json:"schema"`
		REF       string           `json:"ref"`
		Head      string           `json:"head_seal_id"`
		AllPaths  bool             `json:"all_paths"`
		MaxPaths  any              `json:"max_paths"`
		Entries   []logEntryJSONV3 `json:"entries"`
		Paths     []logPathJSONV3  `json:"paths"`
		Truncated bool             `json:"paths_truncated"`
	})
	entries := make([]struct {
		MinimumDepth int                       `json:"minimum_depth"`
		Seal         sealViewJSONV4            `json:"seal"`
		Revision     revisionObservationJSONV3 `json:"revision_observation"`
		Edges        []revisionEdgeJSONV3      `json:"outgoing_revision_edges"`
	}, 0, len(value.Entries))
	for i, entry := range value.Entries {
		old := base.Entries[i]
		entries = append(entries, struct {
			MinimumDepth int                       `json:"minimum_depth"`
			Seal         sealViewJSONV4            `json:"seal"`
			Revision     revisionObservationJSONV3 `json:"revision_observation"`
			Edges        []revisionEdgeJSONV3      `json:"outgoing_revision_edges"`
		}{old.MinimumDepth, sealViewV4(entry.Resolved), old.Revision, old.Edges})
	}
	return struct {
		Schema    string `json:"schema"`
		REF       string `json:"ref"`
		Head      string `json:"head_seal_id"`
		AllPaths  bool   `json:"all_paths"`
		MaxPaths  any    `json:"max_paths"`
		Entries   any    `json:"entries"`
		Paths     any    `json:"paths"`
		Truncated bool   `json:"paths_truncated"`
	}{"sealgraph/log/v4", base.REF, base.Head, base.AllPaths, base.MaxPaths, entries, base.Paths, base.Truncated}
}

func linkLogJSONV4(value repository.LinkLogResult) any {
	base := linkLogJSONV3(value).(struct {
		Schema   string               `json:"schema"`
		REF      string               `json:"ref"`
		Head     string               `json:"head_seal_id"`
		Upstream any                  `json:"upstream_seal_id"`
		Entries  []linkLogEntryJSONV3 `json:"entries"`
	})
	base.Schema = "sealgraph/linklog/v4"
	return base
}
