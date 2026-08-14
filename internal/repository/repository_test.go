package repository

import (
	"context"
	"strings"
	"testing"
	"time"
)

var fixedTime = time.Date(2026, 8, 14, 0, 0, 0, 0, time.UTC)

func openTestRepository(t *testing.T) *Repository {
	t.Helper()
	dir := t.TempDir()
	if _, err := InitStandalone(dir); err != nil {
		t.Fatal(err)
	}
	repo, err := OpenStandalone(dir, func() time.Time { return fixedTime })
	if err != nil {
		t.Fatal(err)
	}
	return repo
}

func sealRoot(t *testing.T, repo *Repository, ref, content, message string) SealResult {
	t.Helper()
	ctx := context.Background()
	if _, err := repo.Add(ctx, AddOptions{REF: ref, Content: []byte(content), Root: true}); err != nil {
		t.Fatal(err)
	}
	sealed, err := repo.Seal(ctx, ref, message)
	if err != nil {
		t.Fatal(err)
	}
	return sealed
}

func TestRootCanSealAndNonRootNeedsDependency(t *testing.T) {
	repo := openTestRepository(t)
	root := sealRoot(t, repo, "requirements/ROOT-001", "external requirement", "initial root")
	if !root.Payload.Root || root.Payload.Parent != nil {
		t.Fatalf("root payload = %+v", root.Payload)
	}

	ctx := context.Background()
	if _, err := repo.Add(ctx, AddOptions{REF: "design/DESIGN-001", Content: []byte("design")}); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.Seal(ctx, "design/DESIGN-001", "reviewed"); err == nil || !strings.Contains(err.Error(), "requires at least one dependency") {
		t.Fatalf("non-root seal error = %v, want dependency rejection", err)
	}
}

func TestDependencyResolutionHistoricalImmutabilityAndDirectStale(t *testing.T) {
	repo := openTestRepository(t)
	ctx := context.Background()
	rootV1 := sealRoot(t, repo, "requirements/ROOT-001", "requirement v1", "root v1")

	if _, err := repo.Add(ctx, AddOptions{
		REF: "design/api/DESIGN-001", Content: []byte("same design"),
		Dependencies: []Dependency{{REF: "requirements/ROOT-001"}},
	}); err != nil {
		t.Fatal(err)
	}
	designV1, err := repo.Seal(ctx, "design/api/DESIGN-001", "reviewed root v1")
	if err != nil {
		t.Fatal(err)
	}
	if got := designV1.Payload.Links[0].TargetSeal; !got.Equal(rootV1.ID) {
		t.Fatalf("plain --depend-on equivalent resolved %s, want concrete HEAD %s", got, rootV1.ID)
	}

	rootV2 := sealRoot(t, repo, "requirements/ROOT-001", "requirement v2", "root v2")
	if rootV2.ID.Equal(rootV1.ID) {
		t.Fatal("root supersession did not create a new identity")
	}

	oldDesign, err := repo.LoadSeal(ctx, designV1.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !oldDesign.Links[0].TargetSeal.Equal(rootV1.ID) {
		t.Fatalf("upstream supersession mutated old downstream link: %+v", oldDesign.Links[0])
	}

	statuses, err := repo.Status(ctx, "design/api/DESIGN-001")
	if err != nil {
		t.Fatal(err)
	}
	if len(statuses) != 1 || len(statuses[0].StaleDirect) != 1 || !contains(statuses[0].Labels(), "STALE_DIRECT") {
		t.Fatalf("status = %+v, want STALE_DIRECT", statuses)
	}

	if _, err := repo.Add(ctx, AddOptions{
		REF: "review/historical", Content: []byte("historical review"), Draft: true,
		Dependencies: []Dependency{{REF: "requirements/ROOT-001", Seal: &rootV1.ID}},
	}); err != nil {
		t.Fatal(err)
	}
	historical, err := repo.Seal(ctx, "review/historical", "draft against v1")
	if err != nil {
		t.Fatalf("explicit historical draft seal: %v", err)
	}
	if !historical.Payload.Links[0].TargetSeal.Equal(rootV1.ID) {
		t.Fatalf("historical link = %s, want %s", historical.Payload.Links[0].TargetSeal, rootV1.ID)
	}
	status, err := repo.Status(ctx, "review/historical")
	if err != nil {
		t.Fatal(err)
	}
	if !contains(status[0].Labels(), "DRAFT") || !contains(status[0].Labels(), "STALE_DIRECT") {
		t.Fatalf("historical draft status = %v", status[0].Labels())
	}
}

func TestNormalHistoricalSealIsRejectedAndRelinkChangesIdentity(t *testing.T) {
	repo := openTestRepository(t)
	ctx := context.Background()
	rootV1 := sealRoot(t, repo, "ROOT", "one", "v1")
	if _, err := repo.Add(ctx, AddOptions{REF: "CHILD", Content: []byte("unchanged"), Dependencies: []Dependency{{REF: "ROOT"}}}); err != nil {
		t.Fatal(err)
	}
	childV1, err := repo.Seal(ctx, "CHILD", "same review message")
	if err != nil {
		t.Fatal(err)
	}
	rootV2 := sealRoot(t, repo, "ROOT", "two", "v2")

	if _, err := repo.Add(ctx, AddOptions{REF: "NORMAL-HISTORY", Content: []byte("x"), Dependencies: []Dependency{{REF: "ROOT", Seal: &rootV1.ID}}}); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.Seal(ctx, "NORMAL-HISTORY", "not draft"); err == nil || !strings.Contains(err.Error(), "HEAD-consistent") {
		t.Fatalf("normal historical seal error = %v, want HEAD consistency rejection", err)
	}

	if _, err := repo.Link(ctx, "CHILD", []Dependency{{REF: "ROOT"}}); err != nil {
		t.Fatal(err)
	}
	childV2, err := repo.Seal(ctx, "CHILD", "same review message")
	if err != nil {
		t.Fatal(err)
	}
	if childV2.ID.Equal(childV1.ID) {
		t.Fatal("relink to a new upstream identity did not change downstream seal identity")
	}
	if !childV2.Payload.Links[0].TargetSeal.Equal(rootV2.ID) {
		t.Fatalf("relinked target = %s, want %s", childV2.Payload.Links[0].TargetSeal, rootV2.ID)
	}
	if childV2.Payload.Parent == nil || !childV2.Payload.Parent.Equal(childV1.ID) {
		t.Fatalf("parent = %v, want %s", childV2.Payload.Parent, childV1.ID)
	}
}

func TestNormalSealChecksCompleteDependencyClosure(t *testing.T) {
	repo := openTestRepository(t)
	ctx := context.Background()
	sealRoot(t, repo, "ROOT", "v1", "v1")
	if _, err := repo.Add(ctx, AddOptions{REF: "MIDDLE", Content: []byte("middle"), Dependencies: []Dependency{{REF: "ROOT"}}}); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.Seal(ctx, "MIDDLE", "middle v1"); err != nil {
		t.Fatal(err)
	}
	sealRoot(t, repo, "ROOT", "v2", "v2")
	if _, err := repo.Add(ctx, AddOptions{REF: "LEAF", Content: []byte("leaf"), Dependencies: []Dependency{{REF: "MIDDLE"}}}); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.Seal(ctx, "LEAF", "leaf"); err == nil || !strings.Contains(err.Error(), "HEAD-consistent") {
		t.Fatalf("closure-stale seal error = %v, want HEAD consistency rejection", err)
	}
}

func TestTransitiveStaleReverseImpactAndSequentialRepair(t *testing.T) {
	repo := openTestRepository(t)
	ctx := context.Background()
	rootV1 := sealRoot(t, repo, "ROOT", "root v1", "root v1")
	if _, err := repo.Add(ctx, AddOptions{REF: "MIDDLE", Content: []byte("middle"), Dependencies: []Dependency{{REF: "ROOT"}}}); err != nil {
		t.Fatal(err)
	}
	middleV1, err := repo.Seal(ctx, "MIDDLE", "middle v1")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := repo.Add(ctx, AddOptions{REF: "LEAF", Content: []byte("leaf"), Dependencies: []Dependency{{REF: "MIDDLE"}}}); err != nil {
		t.Fatal(err)
	}
	leafV1, err := repo.Seal(ctx, "LEAF", "leaf v1")
	if err != nil {
		t.Fatal(err)
	}
	rootV2 := sealRoot(t, repo, "ROOT", "root v2", "root v2")

	middleStatus, err := repo.Status(ctx, "MIDDLE")
	if err != nil {
		t.Fatal(err)
	}
	if !contains(middleStatus[0].Labels(), "STALE_DIRECT") || contains(middleStatus[0].Labels(), "STALE_TRANSITIVE") {
		t.Fatalf("middle labels = %v, want direct stale only", middleStatus[0].Labels())
	}
	leafStatus, err := repo.Status(ctx, "LEAF")
	if err != nil {
		t.Fatal(err)
	}
	if contains(leafStatus[0].Labels(), "STALE_DIRECT") || !contains(leafStatus[0].Labels(), "STALE_TRANSITIVE") {
		t.Fatalf("leaf labels = %v, want transitive stale only", leafStatus[0].Labels())
	}
	path := leafStatus[0].StaleTransitive[0]
	if len(path.Nodes) != 2 || path.Nodes[0].REF != "LEAF" || path.Nodes[1].REF != "MIDDLE" || !path.Link.TargetSeal.Equal(rootV1.ID) || !path.CurrentHead.Equal(rootV2.ID) {
		t.Fatalf("leaf stale path = %+v", path)
	}

	stale, err := repo.Stale(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(stale) != 2 || stale[0].REF != "LEAF" || stale[1].REF != "MIDDLE" {
		t.Fatalf("stale refs = %+v, want LEAF and MIDDLE", stale)
	}
	sourceHead, impacts, err := repo.Impact(ctx, "ROOT")
	if err != nil {
		t.Fatal(err)
	}
	if !sourceHead.Equal(rootV2.ID) || len(impacts) != 2 || impacts[0].REF != "LEAF" || impacts[0].Direct || impacts[1].REF != "MIDDLE" || !impacts[1].Direct {
		t.Fatalf("source=%s impacts=%+v", sourceHead, impacts)
	}

	if _, err := repo.Link(ctx, "MIDDLE", []Dependency{{REF: "ROOT"}}); err != nil {
		t.Fatal(err)
	}
	middleV2, err := repo.Seal(ctx, "MIDDLE", "middle reviewed against root v2")
	if err != nil {
		t.Fatal(err)
	}
	if middleV2.ID.Equal(middleV1.ID) {
		t.Fatal("middle relink did not change seal identity")
	}
	leafStatus, err = repo.Status(ctx, "LEAF")
	if err != nil {
		t.Fatal(err)
	}
	if !contains(leafStatus[0].Labels(), "STALE_DIRECT") || !contains(leafStatus[0].Labels(), "STALE_TRANSITIVE") {
		t.Fatalf("leaf after middle repair labels = %v, want direct and transitive stale until explicit leaf repair", leafStatus[0].Labels())
	}
	if _, err := repo.Link(ctx, "LEAF", []Dependency{{REF: "MIDDLE"}}); err != nil {
		t.Fatal(err)
	}
	leafV2, err := repo.Seal(ctx, "LEAF", "leaf reviewed against middle v2")
	if err != nil {
		t.Fatal(err)
	}
	if leafV2.ID.Equal(leafV1.ID) {
		t.Fatal("leaf relink did not change seal identity")
	}
	leafStatus, err = repo.Status(ctx, "LEAF")
	if err != nil {
		t.Fatal(err)
	}
	if labels := leafStatus[0].Labels(); len(labels) != 1 || labels[0] != "CLEAN" {
		t.Fatalf("leaf after explicit repair labels = %v, want CLEAN", labels)
	}
}

func TestShowExplicitHistoricalSeal(t *testing.T) {
	repo := openTestRepository(t)
	ctx := context.Background()
	v1 := sealRoot(t, repo, "requirements/ROOT", "v1 bytes", "v1")
	sealRoot(t, repo, "requirements/ROOT", "v2 bytes", "v2")
	shown, err := repo.Show(ctx, "requirements/ROOT", &v1.ID)
	if err != nil {
		t.Fatal(err)
	}
	if string(shown.Content) != "v1 bytes" || !shown.ID.Equal(v1.ID) {
		t.Fatalf("historical show = %+v content=%q", shown, shown.Content)
	}
}

func TestUnsealedDraftCandidateStatus(t *testing.T) {
	repo := openTestRepository(t)
	ctx := context.Background()
	sealRoot(t, repo, "ROOT", "root", "root")
	if _, err := repo.Add(ctx, AddOptions{REF: "REVIEW", Content: []byte("work"), Draft: true, Dependencies: []Dependency{{REF: "ROOT"}}}); err != nil {
		t.Fatal(err)
	}
	statuses, err := repo.Status(ctx, "REVIEW")
	if err != nil {
		t.Fatal(err)
	}
	labels := statuses[0].Labels()
	if !contains(labels, "UNSEALED") || !contains(labels, "DRAFT") {
		t.Fatalf("labels = %v, want UNSEALED and DRAFT", labels)
	}
}

func contains(values []string, wanted string) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}
