package domain

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"unicode/utf8"
)

const (
	NativeStore     = "native"
	BlobType        = "blob"
	SealSchema      = "sealgraph/seal/v4"
	CandidateSchema = "sealgraph/candidate/v4"
)

// ObjectID is a full native SHA-256 object name. The repository config fixes
// the algorithm, so the canonical representation contains only full hex.
type ObjectID struct {
	Hex string
}

// ComputeNativeBlobID returns the native SHA-256 identity for exact blob
// payload bytes. Native objects use the Git blob envelope but no Git repository
// semantics.
func ComputeNativeBlobID(data []byte) ObjectID {
	header := []byte("blob " + strconv.Itoa(len(data)) + "\x00")
	digest := sha256.Sum256(append(header, data...))
	return ObjectID{Hex: fmt.Sprintf("%x", digest)}
}

func (id ObjectID) String() string {
	return id.Hex
}

func (id ObjectID) Equal(other ObjectID) bool {
	return id.Hex == other.Hex
}

func (id ObjectID) ValidateNative() error {
	if len(id.Hex) != 64 {
		return fmt.Errorf("native object id has %d hex characters; expected 64", len(id.Hex))
	}
	for _, c := range id.Hex {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
			return fmt.Errorf("native object id must use lower-case hexadecimal")
		}
	}
	return nil
}

func ParseObjectID(text string) (ObjectID, error) {
	id := ObjectID{Hex: text}
	if err := id.ValidateNative(); err != nil {
		return ObjectID{}, fmt.Errorf("invalid object id %q: %w", text, err)
	}
	return id, nil
}

func (id ObjectID) MarshalJSON() ([]byte, error) {
	if err := id.ValidateNative(); err != nil {
		return nil, err
	}
	return json.Marshal(id.Hex)
}

func (id *ObjectID) UnmarshalJSON(data []byte) error {
	var text string
	if err := json.Unmarshal(data, &text); err != nil {
		return fmt.Errorf("native object ID must be a JSON string: %w", err)
	}
	parsed, err := ParseObjectID(text)
	if err != nil {
		return err
	}
	*id = parsed
	return nil
}

// ContentRef identifies content in a specific object source.
type ContentRef struct {
	Store string   `json:"store"`
	Type  string   `json:"type"`
	ID    ObjectID `json:"id"`
}

func (ref ContentRef) ValidateNativeBlob() error {
	if ref.Store != NativeStore || ref.Type != BlobType {
		return fmt.Errorf("content must be a native blob, got store=%q type=%q", ref.Store, ref.Type)
	}
	return ref.ID.ValidateNative()
}

// Link commits a dependent seal to one exact upstream seal generation.
type Link struct {
	TargetSeal ObjectID `json:"target_seal"`
	Message    string   `json:"message"`
}

// Attachment is a named immutable blob included in the seal.
type Attachment struct {
	Name      string     `json:"name"`
	MediaType string     `json:"media_type"`
	Blob      ContentRef `json:"blob"`
}

// SealPayload is the REF-independent immutable format-4 payload.
type SealPayload struct {
	Schema         string       `json:"schema"`
	ParentRevision *ObjectID    `json:"parent_revision"`
	Content        ContentRef   `json:"content"`
	Attachments    []Attachment `json:"attachments"`
	Links          []Link       `json:"links"`
	Root           bool         `json:"root"`
	Draft          bool         `json:"draft"`
}

// Candidate is mutable working state. ParentRevision is immutable derivation
// topology for the next Seal. ExpectedREFHead is separate publication CAS
// state for the destination REF.
type Candidate struct {
	Schema          string       `json:"schema"`
	REF             string       `json:"ref"`
	ParentRevision  *ObjectID    `json:"parent_revision"`
	ExpectedREFHead *ObjectID    `json:"expected_ref_head"`
	Content         ContentRef   `json:"content"`
	Attachments     []Attachment `json:"attachments"`
	Links           []Link       `json:"links"`
	Root            bool         `json:"root"`
	Draft           bool         `json:"draft"`
}

// ValidateREF applies Git check-ref-format-compatible rules to the full name
// refs/seals/<logical-ref>, without invoking Git or normalizing the input.
func ValidateREF(ref string) error {
	if ref == "" {
		return errors.New("REF is empty; provide a logical name")
	}
	if !utf8.ValidString(ref) {
		return errors.New("REF is not valid UTF-8")
	}
	if strings.ContainsRune(ref, '@') {
		return errors.New(`REF cannot contain '@'; it is reserved for seal selectors`)
	}
	if strings.HasPrefix(ref, "/") || strings.HasSuffix(ref, "/") || strings.Contains(ref, "//") {
		return errors.New("REF cannot begin or end with '/', or contain consecutive '/'")
	}
	if strings.Contains(ref, "..") {
		return errors.New("REF cannot contain '..'")
	}
	if strings.Contains(ref, "@{") {
		return errors.New("REF cannot contain '@{'")
	}
	if strings.ContainsRune(ref, '\\') {
		return errors.New(`REF cannot contain '\'`)
	}
	for _, c := range ref {
		if c < 0x20 || c == 0x7f || strings.ContainsRune(" ~^:?*[", c) {
			return fmt.Errorf("REF contains forbidden character %q", c)
		}
	}
	for _, component := range strings.Split(ref, "/") {
		if strings.HasPrefix(component, ".") {
			return fmt.Errorf("REF component %q cannot begin with '.'", component)
		}
		if strings.HasSuffix(component, ".lock") {
			return fmt.Errorf("REF component %q cannot end with '.lock'", component)
		}
	}
	if strings.HasSuffix(ref, ".") {
		return errors.New("REF cannot end with '.'")
	}
	return nil
}

// ValidateTagName validates the raw, user-facing name. Filesystem encoding is
// a storage concern and is deliberately separate.
func ValidateTagName(name string) error {
	if name == "" {
		return errors.New("TAGNAME is empty")
	}
	if !utf8.ValidString(name) {
		return errors.New("TAGNAME is not valid UTF-8")
	}
	for _, c := range name {
		if c < 0x20 || c == 0x7f || c == '@' {
			return fmt.Errorf("TAGNAME contains forbidden character %q", c)
		}
	}
	if IsObjectPrefix(name) {
		return errors.New("TAGNAME cannot be 4 to 64 lower-case hex characters; that syntax is reserved for object prefixes")
	}
	return nil
}

func IsObjectPrefix(text string) bool {
	if len(text) < 4 || len(text) > 64 {
		return false
	}
	for _, c := range text {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
			return false
		}
	}
	return true
}
