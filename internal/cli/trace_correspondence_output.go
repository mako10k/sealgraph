package cli

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"

	"github.com/mako10k/sealgraph/internal/repository"
)

type traceCorrespondenceRangeJSON struct {
	Start  uint64 `json:"start"`
	Length uint64 `json:"length"`
}

type traceCorrespondenceJSON struct {
	Schema         string                         `json:"schema"`
	SourceSnapshot string                         `json:"source_snapshot"`
	SourceStart    uint64                         `json:"source_start"`
	Length         uint64                         `json:"length"`
	CurrentBlob    string                         `json:"current_blob"`
	CurrentRanges  []traceCorrespondenceRangeJSON `json:"current_ranges"`
	Deleted        bool                           `json:"deleted"`
	Reason         string                         `json:"reason"`
	DeclaredAt     string                         `json:"declared_at"`
}

type traceCorrespondenceRecordJSON struct {
	ID     string                  `json:"id"`
	Record traceCorrespondenceJSON `json:"record"`
}

type traceCorrespondenceListJSON struct {
	Schema  string                          `json:"schema"`
	Records []traceCorrespondenceRecordJSON `json:"records"`
}

type traceCorrespondenceMutationJSON struct {
	Schema    string `json:"schema"`
	Operation string `json:"operation"`
	ID        string `json:"id"`
	Changed   bool   `json:"changed"`
}

func traceCorrespondenceRecordsJSON(records []repository.TraceCorrespondenceRecord) []traceCorrespondenceRecordJSON {
	items := make([]traceCorrespondenceRecordJSON, 0, len(records))
	for _, record := range records {
		wire := traceCorrespondenceJSON{
			Schema: record.Record.Schema, SourceSnapshot: record.Record.SourceSnapshot.String(), SourceStart: record.Record.SourceStart,
			Length: record.Record.Length, CurrentBlob: record.Record.CurrentBlob.String(), CurrentRanges: []traceCorrespondenceRangeJSON{},
			Deleted: record.Record.Deleted, Reason: record.Record.Reason, DeclaredAt: record.Record.DeclaredAt,
		}
		for _, current := range record.Record.CurrentRanges {
			wire.CurrentRanges = append(wire.CurrentRanges, traceCorrespondenceRangeJSON{Start: current.Start, Length: current.Length})
		}
		items = append(items, traceCorrespondenceRecordJSON{ID: record.ID, Record: wire})
	}
	return items
}

func traceCorrespondenceDigest(records []repository.TraceCorrespondenceRecord) (string, error) {
	data, err := json.Marshal(traceCorrespondenceRecordsJSON(records))
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256(data)
	return hex.EncodeToString(digest[:]), nil
}
