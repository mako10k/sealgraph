package store

import (
	"context"
	"errors"

	"github.com/mako10k/sealgraph/internal/domain"
)

var (
	ErrObjectNotFound        = errors.New("object not found")
	ErrAmbiguousObjectPrefix = errors.New("ambiguous object prefix")
	ErrRefNotFound           = errors.New("REF not found")
	ErrTagNotFound           = errors.New("tag not found")
	ErrTagConflict           = errors.New("immutable tag conflict")
	ErrCASMismatch           = errors.New("REF compare-and-swap mismatch")
)

// Object is the common read result used by native and Git-backed readers.
type Object struct {
	ID   domain.ObjectID
	Type string
	Data []byte
}

// ObjectReader is the critical common read boundary.
// Native standalone and Git-sidecar implementations must converge here.
type ObjectReader interface {
	ReadObject(ctx context.Context, id domain.ObjectID) (Object, error)
}

// ObjectWriter is intentionally separate because Git-sidecar content may be
// read-only from sealgraph's perspective.
type ObjectWriter interface {
	WriteBlob(ctx context.Context, data []byte) (domain.ObjectID, error)
}

// RefStore manages sealgraph logical REF heads.
// It is not a Git branch abstraction.
type RefStore interface {
	Resolve(ctx context.Context, ref string) (domain.ObjectID, error)
	Update(ctx context.Context, ref string, oldID, newID *domain.ObjectID) error
	Move(ctx context.Context, oldRef, newRef string) error
	List(ctx context.Context) ([]string, error)
}

// RecoveryRefStore exposes exact canonical manifest transitions for the
// standalone recovery coordinator. Git-backed views need not implement it.
type RecoveryRefStore interface {
	RefStore
	Snapshot(ctx context.Context, ref string) ([]byte, error)
	PreviewUpdate(ctx context.Context, ref string, oldID, newID *domain.ObjectID) ([]byte, error)
	PreviewTag(ctx context.Context, ref, name string, id, expectedHead domain.ObjectID) ([]byte, error)
	ManifestTargets(data []byte) ([]domain.ObjectID, error)
	ReplaceExact(ctx context.Context, ref string, expected, replacement []byte) error
	MoveExact(ctx context.Context, oldRef, newRef string, oldExpected, newExpected []byte) error
}

type Tag struct {
	Name string
	Seal domain.ObjectID
}

// TagStore manages immutable, REF-scoped aliases for exact seals. It is not a
// Git tag or movable ref abstraction.
type TagStore interface {
	Resolve(ctx context.Context, ref, name string) (domain.ObjectID, error)
	Create(ctx context.Context, ref, name string, id, expectedHead domain.ObjectID) error
	List(ctx context.Context, ref string) ([]Tag, error)
}
