package v6

import domainv6 "github.com/mako10k/sealgraph/internal/domain/v6"

// NormalizeProvenance exposes the established metadata-bearing Cause Link
// validation to the format-7 codec without changing format-6 bytes.
func NormalizeProvenance(value domainv6.Provenance) (domainv6.Provenance, error) {
	return normalizeProvenance(value)
}

// NormalizeCandidate exposes the established metadata-bearing Candidate
// validation to the format-7 codec without changing format-6 bytes.
func NormalizeCandidate(value domainv6.Candidate) (domainv6.Candidate, error) {
	return normalizeCandidate(value)
}

// AppendCauseLinks writes the established canonical metadata-bearing Link
// member shape after the caller has normalized the value.
func AppendCauseLinks(dst []byte, links []domainv6.CauseLink) ([]byte, error) {
	return appendCauseLinks(dst, links)
}
