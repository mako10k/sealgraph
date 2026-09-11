package repository

import (
	"sort"

	"github.com/mako10k/sealgraph/internal/domain"
	"github.com/mako10k/sealgraph/internal/migration"
)

type projectedSeal = migration.ProjectedSeal
type migrationProjection = migration.Projection

func projectUniversalDump(dump migration.UniversalBlobV1) (migrationProjection, error) {
	return migration.ProjectUniversalBlobV1(dump)
}

func sortMigrationIDs(values []domain.ObjectID) {
	sort.Slice(values, func(i, j int) bool { return values[i].String() < values[j].String() })
}
