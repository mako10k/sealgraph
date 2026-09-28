package cli

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
)

// The cursor is opaque to callers. The checksum detects accidental or direct
// token alteration; all context fields are independently checked against the
// freshly observed selected baseline and file before use.
type traceOccurrencesCursor struct {
	Kind            string `json:"kind"`
	Requested       string `json:"requested"`
	Baseline        string `json:"baseline"`
	SealID          string `json:"seal_id"`
	CandidateDigest string `json:"candidate_digest"`
	OriginMapID     string `json:"origin_map_id"`
	RunIndex        int    `json:"run_index"`
	PatternSHA256   string `json:"pattern_sha256"`
	View            string `json:"view"`
	Limit           int    `json:"limit"`
	SnapshotBlobID  string `json:"snapshot_blob_id"`
	CurrentBlobID   string `json:"current_blob_id"`
	CurrentLength   int    `json:"current_length"`
	BindingDigest   string `json:"binding_digest"`
	LastView        string `json:"last_view"`
	LastStart       int    `json:"last_start"`
}

var errPageTokenInvalid = errors.New("PAGE_TOKEN_INVALID: restart listing from the first page")
var errPageContextChanged = errors.New("PAGE_CONTEXT_CHANGED: restart listing from the first page")

func encodeTraceOccurrencesCursor(value traceOccurrencesCursor) (string, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256(data)
	buffer := make([]byte, 0, len(data)+len(digest))
	buffer = append(buffer, data...)
	buffer = append(buffer, digest[:]...)
	return base64.RawURLEncoding.EncodeToString(buffer), nil
}

func decodeTraceOccurrencesCursor(token string) (traceOccurrencesCursor, error) {
	data, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil || len(data) < sha256.Size+2 {
		return traceOccurrencesCursor{}, errPageTokenInvalid
	}
	payload, claimed := data[:len(data)-sha256.Size], data[len(data)-sha256.Size:]
	actual := sha256.Sum256(payload)
	if !bytes.Equal(claimed, actual[:]) {
		return traceOccurrencesCursor{}, errPageTokenInvalid
	}
	var value traceOccurrencesCursor
	if err := json.Unmarshal(payload, &value); err != nil {
		return traceOccurrencesCursor{}, errPageTokenInvalid
	}
	canonical, err := json.Marshal(value)
	if err != nil || !bytes.Equal(payload, canonical) || !validTraceOccurrencesCursorShape(value) {
		return traceOccurrencesCursor{}, errPageTokenInvalid
	}
	return value, nil
}

func validTraceOccurrencesCursorShape(value traceOccurrencesCursor) bool {
	if value.Kind != "ref" && value.Kind != "seal" || value.Requested == "" || value.RunIndex < 0 || value.Limit <= 0 || value.LastStart < 0 {
		return false
	}
	return validTraceOccurrencesCursorView(value) && validTraceOccurrencesCursorIdentity(value)
}

func validTraceOccurrencesCursorView(value traceOccurrencesCursor) bool {
	if value.View != "snapshot" && value.View != "current" && value.View != "both" || value.LastView != "snapshot" && value.LastView != "current" {
		return false
	}
	if value.View == "snapshot" && value.LastView != "snapshot" || value.View == "current" && value.LastView != "current" {
		return false
	}
	if value.View == "snapshot" {
		return value.CurrentBlobID == "" && value.CurrentLength == 0 && value.BindingDigest == ""
	}
	return value.CurrentLength >= 0 && validTraceCursorHex(value.CurrentBlobID) && validTraceCursorHex(value.BindingDigest)
}

func validTraceOccurrencesCursorIdentity(value traceOccurrencesCursor) bool {
	if !validTraceCursorHex(value.OriginMapID) || !validTraceCursorHex(value.PatternSHA256) || !validTraceCursorHex(value.SnapshotBlobID) {
		return false
	}
	if value.Baseline == "candidate" {
		if value.Kind != "ref" || !validTraceCursorHex(value.CandidateDigest) || value.SealID != "" {
			return false
		}
	} else if value.Baseline != "seal" || !validTraceCursorHex(value.SealID) || value.CandidateDigest != "" {
		return false
	}
	return true
}

func validTraceCursorHex(value string) bool {
	if len(value) != 64 || strings.ToLower(value) != value {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}

func sameTraceOccurrencesContext(left, right traceOccurrencesCursor) bool {
	left.LastView, left.LastStart = "", 0
	right.LastView, right.LastStart = "", 0
	return left == right
}
