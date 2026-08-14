// Package graph contains derived provenance graph logic.
//
// Canonical state is seals + current REF heads. Stale/direct/transitive impact
// information belongs here and must not be persisted as authoritative state.
package graph
