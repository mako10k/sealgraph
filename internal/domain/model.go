package domain

import (
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"
)

const (
	NativeAlgorithm = "sha256"
	NativeStore     = "native"
	BlobType        = "blob"
	SealSchema      = "sealgraph/seal/v1"
	CandidateSchema = "sealgraph/candidate/v1"
	DependOn        = "depend-on"
)

// ObjectID is an algorithm-tagged immutable object identity.
type ObjectID struct {
	Algorithm string `json:"algorithm"`
	Hex       string `json:"hex"`
}

func (id ObjectID) String() string {
	return id.Algorithm + ":" + id.Hex
}

func (id ObjectID) Equal(other ObjectID) bool {
	return id.Algorithm == other.Algorithm && id.Hex == other.Hex
}

func (id ObjectID) ValidateNative() error {
	if id.Algorithm != NativeAlgorithm {
		return fmt.Errorf("object algorithm %q is not supported; expected %q", id.Algorithm, NativeAlgorithm)
	}
	if len(id.Hex) != 64 {
		return fmt.Errorf("sha256 object id has %d hex characters; expected 64", len(id.Hex))
	}
	for _, c := range id.Hex {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
			return fmt.Errorf("sha256 object id must use lower-case hexadecimal")
		}
	}
	return nil
}

func ParseObjectID(text string) (ObjectID, error) {
	id := ObjectID{Algorithm: NativeAlgorithm, Hex: text}
	if algorithm, hex, found := strings.Cut(text, ":"); found {
		id = ObjectID{Algorithm: algorithm, Hex: hex}
	}
	if err := id.ValidateNative(); err != nil {
		return ObjectID{}, fmt.Errorf("invalid object id %q: %w", text, err)
	}
	return id, nil
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
	Relation   string   `json:"relation"`
	TargetREF  string   `json:"target_ref"`
	TargetSeal ObjectID `json:"target_seal"`
}

// Attachment is a named immutable blob included in the seal.
type Attachment struct {
	Name      string     `json:"name"`
	MediaType string     `json:"media_type"`
	Blob      ContentRef `json:"blob"`
}

// SealPayload is the semantic payload encoded by sealgraph-canonical-json-v1.
type SealPayload struct {
	Schema      string       `json:"schema"`
	REF         string       `json:"ref"`
	Parent      *ObjectID    `json:"parent"`
	Content     ContentRef   `json:"content"`
	Attachments []Attachment `json:"attachments"`
	Links       []Link       `json:"links"`
	Message     string       `json:"message"`
	Root        bool         `json:"root"`
	Draft       bool         `json:"draft"`
	CreatedAt   string       `json:"created_at"`
}

// Candidate is mutable working state. Base is the REF head observed when the
// candidate was first derived and is used for compare-and-swap sealing.
type Candidate struct {
	Schema      string       `json:"schema"`
	REF         string       `json:"ref"`
	Base        *ObjectID    `json:"base"`
	Content     ContentRef   `json:"content"`
	Attachments []Attachment `json:"attachments"`
	Links       []Link       `json:"links"`
	Root        bool         `json:"root"`
	Draft       bool         `json:"draft"`
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
	if ref == "@" {
		return errors.New(`REF cannot be the single character "@"`)
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
