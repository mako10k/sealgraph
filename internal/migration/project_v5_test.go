package migration

import (
	"strings"
	"testing"

	canonicalv5 "github.com/mako10k/sealgraph/internal/canonical/v5"
	"github.com/mako10k/sealgraph/internal/domain"
	domainv5 "github.com/mako10k/sealgraph/internal/domain/v5"
)

func TestValidateProjectedGraphRejectsMixedCauseRevisionCycle(t *testing.T) {
	first := domain.ObjectID{Hex: strings.Repeat("a", 64)}
	second := domain.ObjectID{Hex: strings.Repeat("b", 64)}
	firstProvenance := encodeTestProvenance(t, second, []domain.ObjectID{first})
	secondProvenance, err := canonicalv5.EncodeProvenance(domainv5.Provenance{
		Schema: domainv5.ProvenanceSchema,
		Root:   true,
	})
	if err != nil {
		t.Fatal(err)
	}

	err = validateProjectedGraph([]ProjectedSeal{
		{NewID: first, ProvenanceBytes: firstProvenance},
		{NewID: second, ProvenanceBytes: secondProvenance},
	})
	if err == nil || !strings.Contains(err.Error(), "combined Cause/revision cycle") {
		t.Fatalf("err=%v", err)
	}
}

func encodeTestProvenance(t *testing.T, target domain.ObjectID, previous []domain.ObjectID) []byte {
	t.Helper()
	data, err := canonicalv5.EncodeProvenance(domainv5.Provenance{
		Schema: domainv5.ProvenanceSchema,
		CauseLinks: []domainv5.CauseLink{{
			TargetSeal:                       target,
			PreviousRevisionSealOfTargetSeal: previous,
			Messages:                         []string{},
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	return data
}
