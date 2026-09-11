package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"

	domainv5 "github.com/mako10k/sealgraph/internal/domain/v5"
	"github.com/mako10k/sealgraph/internal/repository"
)

type metadataEntryJSON struct {
	Namespace string          `json:"namespace"`
	Schema    *string         `json:"schema"`
	Value     json.RawMessage `json:"value"`
}

type linkMetadataMutationReceipt struct {
	Schema                  string             `json:"schema"`
	Action                  string             `json:"action"`
	REF                     string             `json:"ref"`
	TargetSeal              string             `json:"target_seal"`
	Namespace               string             `json:"namespace"`
	Before                  *metadataEntryJSON `json:"before"`
	After                   *metadataEntryJSON `json:"after"`
	CandidateSchema         string             `json:"candidate_schema"`
	ProspectiveProvenanceID string             `json:"prospective_provenance_id"`
	ProspectiveSealID       string             `json:"prospective_seal_id"`
}

func linkMetadataReceiptJSON(result repository.LinkMetadataMutationResult) linkMetadataMutationReceipt {
	return linkMetadataMutationReceipt{
		Schema: "sealgraph/link-metadata-mutation/v1", Action: result.Action, REF: result.REF,
		TargetSeal: result.TargetSeal, Namespace: result.Namespace,
		Before: metadataEntryReceipt(result.Before), After: metadataEntryReceipt(result.After),
		CandidateSchema:         result.Candidate.Schema,
		ProspectiveProvenanceID: result.Prospective.Seal.Provenance.String(),
		ProspectiveSealID:       result.Prospective.ID.String(),
	}
}

func metadataEntryReceipt(entry *domainv5.MetadataEntry) *metadataEntryJSON {
	if entry == nil {
		return nil
	}
	return &metadataEntryJSON{Namespace: entry.Namespace, Schema: entry.Schema, Value: entry.Value}
}

type repositoryMigrationReceipt struct {
	Schema                string `json:"schema"`
	FromFormat            int    `json:"from_format"`
	ToFormat              int    `json:"to_format"`
	Result                string `json:"result"`
	RetainedSealsV5       int    `json:"retained_seals_v5"`
	RetainedProvenancesV1 int    `json:"retained_provenances_v1"`
	RetainedCandidatesV5  int    `json:"retained_candidates_v5"`
}

func writeCommittedMigrationJSON(stdout, stderr io.Writer, value repositoryMigrationReceipt) int {
	var buffer bytes.Buffer
	encoder := json.NewEncoder(&buffer)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(value); err != nil {
		return commandError(stderr, "migrate repository", fmt.Errorf("encode committed migration receipt: %w", err))
	}
	data := literalizeInspectionUnicodeSeparators(buffer.Bytes())
	return writeCommittedMigrationBytes(stdout, stderr, data)
}

func writeCommittedMigrationBytes(stdout, stderr io.Writer, data []byte) int {
	written, err := stdout.Write(data)
	if err != nil || written != len(data) {
		if err == nil {
			err = io.ErrShortWrite
		}
		fmt.Fprintf(stderr, "error: sealgraph migrate repository: MIGRATION_COMMITTED_OUTPUT_UNDELIVERED: repository is already format 6; do not retry migration: %v\n", err)
		return 3
	}
	return 0
}
