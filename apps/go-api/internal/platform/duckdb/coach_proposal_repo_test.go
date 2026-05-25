//go:build integration

// Tests d'intégration de CoachProposalRepo (stats.duckdb, Phase 3 ADR 0020).

package duckdb

import (
	"context"
	"errors"
	"testing"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/migration"
	"levelup/go-api/internal/prestige"
	"levelup/go-api/internal/progression/coach_advisor"
)

func newCoachProposalRepoForTest(t *testing.T) *CoachProposalRepo {
	t.Helper()
	db := setupPrestigeDB(t, migration.TargetPlayer)
	return NewCoachProposalRepo(db)
}

func sampleProposal(id, metric, axis string, strength float64) coach_advisor.Proposal {
	return coach_advisor.Proposal{
		ID:            id,
		UserID:        "u1",
		TitleSlug:     "halo_infinite",
		Kind:          coach_advisor.ProposalKindChallenge,
		TemplateID:    "tpl_" + id,
		SuggestedTier: prestige.TierHeroic,
		SourceSignal:  coach_advisor.SignalLOWESSPositive,
		SourceMetric:  metric,
		RadarAxis:     axis,
		Strength:      strength,
		Origin:        coach_advisor.OriginCatalog,
		ReasonKeyEN:   "coach.proposal.lowess.en",
		ReasonKeyFR:   "coach.proposal.lowess.fr",
		ReasonParams:  `{"metric":"` + metric + `"}`,
	}
}

func TestCoachProposalRepo_CreateAndGet_Roundtrip(t *testing.T) {
	repo := newCoachProposalRepoForTest(t)
	ctx := context.Background()
	p := sampleProposal("p1", "accuracy", "combat", 0.7)

	if err := repo.Create(ctx, p); err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := repo.Get(ctx, "p1")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.ID != "p1" || got.UserID != "u1" || got.TitleSlug != "halo_infinite" {
		t.Errorf("identity mismatch: %+v", got)
	}
	if got.Kind != coach_advisor.ProposalKindChallenge {
		t.Errorf("Kind: got %q, want challenge", got.Kind)
	}
	if got.Status != coach_advisor.ProposalPending {
		t.Errorf("Status default: got %q, want pending", got.Status)
	}
	if got.TemplateID != "tpl_p1" {
		t.Errorf("TemplateID: got %q", got.TemplateID)
	}
	if got.SuggestedTier != prestige.TierHeroic {
		t.Errorf("SuggestedTier: got %q", got.SuggestedTier)
	}
	if got.SourceMetric != "accuracy" || got.RadarAxis != "combat" {
		t.Errorf("source mismatch: metric=%q axis=%q", got.SourceMetric, got.RadarAxis)
	}
	if got.Strength != 0.7 {
		t.Errorf("Strength: got %f, want 0.7", got.Strength)
	}
	if got.Origin != coach_advisor.OriginCatalog {
		t.Errorf("Origin: got %q", got.Origin)
	}
	if got.ExpiresAt != nil {
		t.Errorf("ExpiresAt should be nil (no expiration by age), got %v", got.ExpiresAt)
	}
}

func TestCoachProposalRepo_Get_NotFound_ReturnsErrProposalNotFound(t *testing.T) {
	repo := newCoachProposalRepoForTest(t)
	_, err := repo.Get(context.Background(), "missing")
	if !errors.Is(err, coach_advisor.ErrProposalNotFound) {
		t.Errorf("expected ErrProposalNotFound, got %v", err)
	}
}

func TestCoachProposalRepo_ListByUser_FiltersByStatus(t *testing.T) {
	repo := newCoachProposalRepoForTest(t)
	ctx := context.Background()
	now := time.Now().UTC()

	p1 := sampleProposal("p1", "accuracy", "combat", 0.6)
	p2 := sampleProposal("p2", "kda", "combat", 0.8)
	p3 := sampleProposal("p3", "win_rate", "score", 0.7)
	for _, p := range []coach_advisor.Proposal{p1, p2, p3} {
		if err := repo.Create(ctx, p); err != nil {
			t.Fatalf("Create %s: %v", p.ID, err)
		}
	}
	if err := repo.MarkDismissed(ctx, "p2", now); err != nil {
		t.Fatalf("MarkDismissed: %v", err)
	}

	pending, err := repo.ListByUser(ctx, "u1", "halo_infinite", coach_advisor.ProposalPending)
	if err != nil {
		t.Fatalf("ListByUser pending: %v", err)
	}
	if len(pending) != 2 {
		t.Errorf("pending count: got %d, want 2", len(pending))
	}

	dismissed, err := repo.ListByUser(ctx, "u1", "halo_infinite", coach_advisor.ProposalDismissed)
	if err != nil {
		t.Fatalf("ListByUser dismissed: %v", err)
	}
	if len(dismissed) != 1 || dismissed[0].ID != "p2" {
		t.Errorf("dismissed: got %v", dismissed)
	}

	all, err := repo.ListByUser(ctx, "u1", "halo_infinite", "")
	if err != nil {
		t.Fatalf("ListByUser all: %v", err)
	}
	if len(all) != 3 {
		t.Errorf("all count: got %d, want 3", len(all))
	}
}

func TestCoachProposalRepo_ListPendingBySignalScope_MatchesByMetricOrAxis(t *testing.T) {
	repo := newCoachProposalRepoForTest(t)
	ctx := context.Background()

	// p1 : metric=accuracy, axis=combat
	// p2 : metric=kda,      axis=combat
	// p3 : metric=accuracy, axis=score
	// p4 : metric=win_rate, axis=score
	for i, p := range []coach_advisor.Proposal{
		sampleProposal("p1", "accuracy", "combat", 0.5),
		sampleProposal("p2", "kda", "combat", 0.6),
		sampleProposal("p3", "accuracy", "score", 0.7),
		sampleProposal("p4", "win_rate", "score", 0.8),
	} {
		if err := repo.Create(ctx, p); err != nil {
			t.Fatalf("Create p%d: %v", i+1, err)
		}
	}

	got, err := repo.ListPendingBySignalScope(ctx, "u1", "halo_infinite", "accuracy", "combat")
	if err != nil {
		t.Fatalf("ListPendingBySignalScope: %v", err)
	}
	// Match p1 (both), p2 (axis), p3 (metric). p4 ne match ni metric ni axis.
	if len(got) != 3 {
		t.Fatalf("got %d, want 3", len(got))
	}
	seen := map[string]bool{}
	for _, p := range got {
		seen[p.ID] = true
	}
	for _, id := range []string{"p1", "p2", "p3"} {
		if !seen[id] {
			t.Errorf("missing %s in result", id)
		}
	}
	if seen["p4"] {
		t.Errorf("p4 (no overlap) should not be in result")
	}
}

func TestCoachProposalRepo_MarkAccepted_TransitionsStatusAndRef(t *testing.T) {
	repo := newCoachProposalRepoForTest(t)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)

	if err := repo.Create(ctx, sampleProposal("p1", "accuracy", "combat", 0.7)); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := repo.MarkAccepted(ctx, "p1", "ch_xyz", now); err != nil {
		t.Fatalf("MarkAccepted: %v", err)
	}

	got, _ := repo.Get(ctx, "p1")
	if got.Status != coach_advisor.ProposalAccepted {
		t.Errorf("Status: got %q, want accepted", got.Status)
	}
	if got.ResolvedRef != "ch_xyz" {
		t.Errorf("ResolvedRef: got %q, want ch_xyz", got.ResolvedRef)
	}
	if got.ResolvedAt == nil || !got.ResolvedAt.Equal(now) {
		t.Errorf("ResolvedAt: got %v, want %v", got.ResolvedAt, now)
	}
}

func TestCoachProposalRepo_MarkSuperseded_OnlyAffectsPending(t *testing.T) {
	repo := newCoachProposalRepoForTest(t)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)

	if err := repo.Create(ctx, sampleProposal("p1", "accuracy", "combat", 0.5)); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := repo.Create(ctx, sampleProposal("p2", "accuracy", "combat", 0.7)); err != nil {
		t.Fatalf("Create p2: %v", err)
	}
	// p2 supersède p1 ; p2 reste pending.
	if err := repo.MarkSuperseded(ctx, "p1", "p2", now); err != nil {
		t.Fatalf("MarkSuperseded: %v", err)
	}
	// Re-appel idempotent : aucune erreur, mais ne touche pas (filter status='pending').
	if err := repo.MarkSuperseded(ctx, "p1", "p2", now); err != nil {
		t.Fatalf("MarkSuperseded 2e: %v", err)
	}

	got, _ := repo.Get(ctx, "p1")
	if got.Status != coach_advisor.ProposalSuperseded {
		t.Errorf("Status: got %q, want superseded", got.Status)
	}
	if got.SupersededBy != "p2" {
		t.Errorf("SupersededBy: got %q, want p2", got.SupersededBy)
	}
}

func TestCoachProposalRepo_ListPendingByAxis_ForObsolescence(t *testing.T) {
	repo := newCoachProposalRepoForTest(t)
	ctx := context.Background()
	now := time.Now().UTC()

	for _, p := range []coach_advisor.Proposal{
		sampleProposal("p1", "accuracy", "combat", 0.6),
		sampleProposal("p2", "kda", "combat", 0.7),
		sampleProposal("p3", "win_rate", "score", 0.8),
	} {
		if err := repo.Create(ctx, p); err != nil {
			t.Fatalf("Create %s: %v", p.ID, err)
		}
	}

	combat, err := repo.ListPendingByAxis(ctx, "u1", "halo_infinite", "combat")
	if err != nil {
		t.Fatalf("ListPendingByAxis: %v", err)
	}
	if len(combat) != 2 {
		t.Errorf("combat axis: got %d, want 2", len(combat))
	}

	// Obsoletes both
	for _, p := range combat {
		if err := repo.MarkObsoleted(ctx, p.ID, now); err != nil {
			t.Fatalf("MarkObsoleted %s: %v", p.ID, err)
		}
	}
	combat2, _ := repo.ListPendingByAxis(ctx, "u1", "halo_infinite", "combat")
	if len(combat2) != 0 {
		t.Errorf("after obsolete, combat pending: got %d, want 0", len(combat2))
	}
}

func TestCoachProposalRepo_Create_WithArcKind(t *testing.T) {
	repo := newCoachProposalRepoForTest(t)
	ctx := context.Background()

	arc := coach_advisor.Proposal{
		ID:             "arc1",
		UserID:         "u1",
		TitleSlug:      "halo_infinite",
		Kind:           coach_advisor.ProposalKindArc,
		ChallengesSpec: `[{"template_id":"a"},{"template_id":"b"}]`,
		SuggestedTier:  prestige.TierLegendary,
		SourceSignal:   coach_advisor.SignalCombatPatternActive,
		RadarAxis:      "combat",
		Strength:       0.85,
		Origin:         coach_advisor.OriginSynthesized,
	}
	if err := repo.Create(ctx, arc); err != nil {
		t.Fatalf("Create arc: %v", err)
	}

	got, _ := repo.Get(ctx, "arc1")
	if got.Kind != coach_advisor.ProposalKindArc {
		t.Errorf("Kind: got %q, want arc", got.Kind)
	}
	if got.ChallengesSpec == "" {
		t.Error("ChallengesSpec lost across round-trip")
	}
	if got.TemplateID != "" {
		t.Errorf("TemplateID should be empty for arc, got %q", got.TemplateID)
	}
	if got.Origin != coach_advisor.OriginSynthesized {
		t.Errorf("Origin: got %q", got.Origin)
	}
}
