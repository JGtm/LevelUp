package prestige

import (
	"context"
	"testing"
	"time"
)

// service_arc_completion_test.go — couvre la clôture d'arc et son bonus PP :
//   - maybeCompleteArc (toutes étapes finies / partielles / arc déjà clôturé)
//   - enrichArcReward via ListArcs (ObjectivesPP + CompletionBonusPP)
//   - EvaluateForUser : la dernière étape complétée clôt l'arc et crédite le bonus.
//
// Les stubs (stubChallengeRepo, stubArcRepo, stubPrestigeRepo) et le harness
// buildCoverageService() viennent de service_coverage_test.go (même package).

func countBySource(events []PrestigeEvent, source string) int {
	n := 0
	for _, e := range events {
		if e.SourceType == source {
			n++
		}
	}
	return n
}

func TestMaybeCompleteArc_AllCompleted_CreditsBonus(t *testing.T) {
	svc, chRepo, arcRepo, _, prRepo, _ := buildCoverageService()
	const arcID = "arc_done"
	arcRepo.stored[arcID] = Arc{ID: arcID, UserID: "u", TitleSlug: "halo_infinite"}
	// 3 objectifs Héroïque/Full tous complétés → objectivesPP = 3*75 = 225,
	// bonus = round(225 * 0.5) = 113.
	for _, id := range []string{"c1", "c2", "c3"} {
		chRepo.stored[id] = Challenge{
			ID: id, ArcID: arcID, Tier: TierHeroic, DataTier: DataFull, Status: StatusCompleted,
		}
	}

	var out EvaluationOutcome
	svc.maybeCompleteArc(context.Background(), arcID, "u", "halo_infinite", svc.deps.Now(), &out)

	if out.ArcCompletedID != arcID {
		t.Fatalf("ArcCompletedID = %q, want %q", out.ArcCompletedID, arcID)
	}
	if out.ArcPPCredited != 113 {
		t.Errorf("ArcPPCredited = %d, want 113", out.ArcPPCredited)
	}
	if n := countBySource(prRepo.emitted, SourceArc); n != 1 {
		t.Fatalf("arc events = %d, want 1", n)
	}
	ev := prRepo.emitted[len(prRepo.emitted)-1]
	if ev.SourceID != arcID || ev.PPAmount != 113 {
		t.Errorf("arc event = %+v, want source=%s pp=113", ev, arcID)
	}
}

func TestMaybeCompleteArc_NotAllCompleted_NoOp(t *testing.T) {
	svc, chRepo, arcRepo, _, prRepo, _ := buildCoverageService()
	const arcID = "arc_partial"
	arcRepo.stored[arcID] = Arc{ID: arcID, UserID: "u", TitleSlug: "halo_infinite"}
	chRepo.stored["c1"] = Challenge{ID: "c1", ArcID: arcID, Tier: TierHeroic, DataTier: DataFull, Status: StatusCompleted}
	chRepo.stored["c2"] = Challenge{ID: "c2", ArcID: arcID, Tier: TierHeroic, DataTier: DataFull, Status: StatusActive}

	var out EvaluationOutcome
	svc.maybeCompleteArc(context.Background(), arcID, "u", "halo_infinite", svc.deps.Now(), &out)

	if out.ArcCompletedID != "" {
		t.Errorf("ArcCompletedID = %q, want empty (arc en cours)", out.ArcCompletedID)
	}
	if n := countBySource(prRepo.emitted, SourceArc); n != 0 {
		t.Errorf("arc events = %d, want 0", n)
	}
}

func TestMaybeCompleteArc_AlreadyCompleted_Idempotent(t *testing.T) {
	svc, chRepo, arcRepo, _, prRepo, _ := buildCoverageService()
	const arcID = "arc_already"
	done := svc.deps.Now()
	arcRepo.stored[arcID] = Arc{ID: arcID, UserID: "u", TitleSlug: "halo_infinite", CompletedAt: &done}
	chRepo.stored["c1"] = Challenge{ID: "c1", ArcID: arcID, Tier: TierMythic, DataTier: DataFull, Status: StatusCompleted}

	var out EvaluationOutcome
	svc.maybeCompleteArc(context.Background(), arcID, "u", "halo_infinite", svc.deps.Now(), &out)

	if out.ArcCompletedID != "" || countBySource(prRepo.emitted, SourceArc) != 0 {
		t.Errorf("arc déjà clôturé : attendu no-op, got id=%q events=%d",
			out.ArcCompletedID, countBySource(prRepo.emitted, SourceArc))
	}
}

func TestListArcs_EnrichesReward(t *testing.T) {
	svc, chRepo, arcRepo, _, _, _ := buildCoverageService()
	arcRepo.list = []Arc{{ID: "arc_x", UserID: "u", TitleSlug: "halo_infinite"}}
	// Héroïque (75) + Légendaire (125) = 200 ; bonus = round(200 * 0.5) = 100.
	chRepo.stored["c1"] = Challenge{ID: "c1", ArcID: "arc_x", Tier: TierHeroic, DataTier: DataFull, Status: StatusActive}
	chRepo.stored["c2"] = Challenge{ID: "c2", ArcID: "arc_x", Tier: TierLegendary, DataTier: DataFull, Status: StatusCompleted}

	arcs, err := svc.ListArcs(context.Background(), "u", "halo_infinite")
	if err != nil {
		t.Fatalf("ListArcs: %v", err)
	}
	if len(arcs) != 1 {
		t.Fatalf("arcs = %d, want 1", len(arcs))
	}
	if arcs[0].ObjectivesPP != 200 {
		t.Errorf("ObjectivesPP = %d, want 200", arcs[0].ObjectivesPP)
	}
	if arcs[0].CompletionBonusPP != 100 {
		t.Errorf("CompletionBonusPP = %d, want 100", arcs[0].CompletionBonusPP)
	}
}

// TestService_EvaluateForUser_ClosesArcAndCreditsBonus vérifie le câblage
// complet : compléter la dernière étape d'un arc via l'évaluateur clôt l'arc
// et émet un événement PP de type "arc" en plus de celui du défi.
func TestService_EvaluateForUser_ClosesArcAndCreditsBonus(t *testing.T) {
	svc, chRepo, arcRepo, _, prRepo, bp := buildCoverageService()
	bp.matches = make([]MatchData, 12)
	for i := range bp.matches {
		bp.matches[i] = MatchData{MetricValue: 2.0, StartedAt: time.Now()}
	}
	const arcID = "arc_evt"
	arcRepo.stored[arcID] = Arc{ID: arcID, UserID: "u1", TitleSlug: "halo_infinite"}

	// Une étape déjà complétée + l'étape active qui se complète ce run.
	done := svc.deps.Now()
	chRepo.stored["done1"] = Challenge{
		ID: "done1", UserID: "u1", TitleSlug: "halo_infinite", ArcID: arcID,
		Tier: TierHeroic, DataTier: DataFull, Status: StatusCompleted, CompletedAt: &done,
	}
	last := Challenge{
		ID: "last", UserID: "u1", TitleSlug: "halo_infinite", ArcID: arcID,
		Metric: "FieldKDA", Target: 1.5, Status: StatusActive,
		EvalType: EvalThreshold, WindowType: WindowSession,
		Tier: TierHeroic, DataTier: DataFull, Mode: ModeLibre,
	}
	chRepo.stored["last"] = last
	chRepo.listResult = []Challenge{last} // ListActiveChallenges ne renvoie que l'actif

	outcomes, err := svc.EvaluateForUser(context.Background(), "u1", "halo_infinite")
	if err != nil {
		t.Fatal(err)
	}
	if len(outcomes) != 1 || outcomes[0].NewStatus != StatusCompleted {
		t.Fatalf("attendu 1 défi complété, got %+v", outcomes)
	}
	if outcomes[0].ArcCompletedID != arcID {
		t.Errorf("ArcCompletedID = %q, want %q", outcomes[0].ArcCompletedID, arcID)
	}
	// objectivesPP = 75 (done1) + 75 (last) = 150 ; bonus = round(150*0.5) = 75.
	if outcomes[0].ArcPPCredited != 75 {
		t.Errorf("ArcPPCredited = %d, want 75", outcomes[0].ArcPPCredited)
	}
	if c := countBySource(prRepo.emitted, SourceChallenge); c != 1 {
		t.Errorf("challenge events = %d, want 1", c)
	}
	if a := countBySource(prRepo.emitted, SourceArc); a != 1 {
		t.Errorf("arc events = %d, want 1", a)
	}
}
