package breakdown

import (
	"testing"

	"levelup/go-api/internal/games/canonical"
)

// score retourne un *float64 pour les tests (helper).
func ptrF(v float64) *float64 { return &v }

func TestAvgPerformanceScoreByChain_GroupsByChain(t *testing.T) {
	t.Parallel()
	rows := []Row{
		{PerformanceScore: ptrF(80), PerformanceChain: "btb"},
		{PerformanceScore: ptrF(60), PerformanceChain: "btb"},
		{PerformanceScore: ptrF(70), PerformanceChain: "arena_slayer"},
		{PerformanceScore: ptrF(50), PerformanceChain: "arena_slayer"},
	}
	got := avgPerformanceScoreByChain(rows)
	if len(got) != 2 {
		t.Fatalf("expected 2 chains, got %d (%v)", len(got), got)
	}
	if got["btb"] == nil || *got["btb"] != 70 {
		t.Errorf("btb avg = %v, want 70", got["btb"])
	}
	if got["arena_slayer"] == nil || *got["arena_slayer"] != 60 {
		t.Errorf("arena_slayer avg = %v, want 60", got["arena_slayer"])
	}
}

func TestAvgPerformanceScoreByChain_IgnoresRowsWithoutScoreOrChain(t *testing.T) {
	t.Parallel()
	rows := []Row{
		{PerformanceScore: ptrF(80), PerformanceChain: "btb"},
		{PerformanceScore: nil, PerformanceChain: "btb"},   // skip — no score
		{PerformanceScore: ptrF(50), PerformanceChain: ""}, // skip — no chain
		{PerformanceScore: ptrF(60), PerformanceChain: "ranked"},
	}
	got := avgPerformanceScoreByChain(rows)
	if len(got) != 2 {
		t.Fatalf("expected 2 chains, got %d", len(got))
	}
	if *got["btb"] != 80 {
		t.Errorf("btb avg = %v, want 80", *got["btb"])
	}
	if *got["ranked"] != 60 {
		t.Errorf("ranked avg = %v, want 60", *got["ranked"])
	}
}

func TestAvgPerformanceScoreByChain_ReturnsNilWhenEmpty(t *testing.T) {
	t.Parallel()
	rows := []Row{
		{PerformanceScore: nil, PerformanceChain: "btb"},
		{PerformanceScore: ptrF(50), PerformanceChain: ""},
	}
	got := avgPerformanceScoreByChain(rows)
	if got != nil {
		t.Errorf("expected nil map when no valid (score, chain) pair, got %v", got)
	}
}

// TestByMap_PerfByChain_Populated vérifie que ByMap décompose la perf par chaîne
// pour une carte jouée dans plusieurs chaînes. Cas réel : "Behemoth" jouée en
// BTB et en Ranked Arena → on doit pouvoir lire la moyenne distincte de chaque.
func TestByMap_PerfByChain_Populated(t *testing.T) {
	t.Parallel()
	rows := []Row{
		{MapID: "behemoth", MapLabel: "Behemoth", Outcome: canonical.OutcomeWin,
			PerformanceScore: ptrF(70), PerformanceChain: "btb"},
		{MapID: "behemoth", MapLabel: "Behemoth", Outcome: canonical.OutcomeWin,
			PerformanceScore: ptrF(90), PerformanceChain: "btb"},
		{MapID: "behemoth", MapLabel: "Behemoth", Outcome: canonical.OutcomeLoss,
			PerformanceScore: ptrF(40), PerformanceChain: "ranked"},
		{MapID: "behemoth", MapLabel: "Behemoth", Outcome: canonical.OutcomeWin,
			PerformanceScore: ptrF(60), PerformanceChain: "ranked"},
	}
	got := ByMap(rows)
	if len(got) != 1 {
		t.Fatalf("expected 1 map aggregate, got %d", len(got))
	}
	if got[0].AvgPerformanceScore == nil || *got[0].AvgPerformanceScore != 65 {
		t.Errorf("AvgPerformanceScore (global) = %v, want 65", got[0].AvgPerformanceScore)
	}
	if got[0].PerfByChain == nil {
		t.Fatal("PerfByChain should be populated")
	}
	if got[0].PerfByChain["btb"] == nil || *got[0].PerfByChain["btb"] != 80 {
		t.Errorf("PerfByChain[btb] = %v, want 80", got[0].PerfByChain["btb"])
	}
	if got[0].PerfByChain["ranked"] == nil || *got[0].PerfByChain["ranked"] != 50 {
		t.Errorf("PerfByChain[ranked] = %v, want 50", got[0].PerfByChain["ranked"])
	}
}

// TestByMap_PerfByChain_NilWhenNoChain : si aucune row n'a de PerformanceChain
// (cas legacy DB pas encore backfillée), PerfByChain doit être nil.
func TestByMap_PerfByChain_NilWhenNoChain(t *testing.T) {
	t.Parallel()
	rows := []Row{
		{MapID: "m1", MapLabel: "Bazaar", Outcome: canonical.OutcomeWin,
			PerformanceScore: ptrF(70)},
	}
	got := ByMap(rows)
	if got[0].PerfByChain != nil {
		t.Errorf("PerfByChain should be nil when no row has a chain, got %v", got[0].PerfByChain)
	}
	// AvgPerformanceScore reste calculé sur le score global même sans chaîne.
	if got[0].AvgPerformanceScore == nil || *got[0].AvgPerformanceScore != 70 {
		t.Errorf("AvgPerformanceScore = %v, want 70", got[0].AvgPerformanceScore)
	}
}

// TestByModeCategory_PerfByChain_Populated : une catégorie "Assassin" peut
// englober arena_slayer + arena_objectif (sous-modes mixés). PerfByChain doit
// décomposer la moyenne par chaîne.
func TestByModeCategory_PerfByChain_Populated(t *testing.T) {
	t.Parallel()
	rows := []Row{
		{ModeCategory: "Assassin", Outcome: canonical.OutcomeWin,
			PerformanceScore: ptrF(60), PerformanceChain: "arena_slayer"},
		{ModeCategory: "Assassin", Outcome: canonical.OutcomeLoss,
			PerformanceScore: ptrF(80), PerformanceChain: "arena_slayer"},
		{ModeCategory: "Assassin", Outcome: canonical.OutcomeWin,
			PerformanceScore: ptrF(90), PerformanceChain: "arena_objectif"},
	}
	got := ByModeCategory(rows)
	if len(got) != 1 {
		t.Fatalf("expected 1 category aggregate, got %d", len(got))
	}
	if got[0].PerfByChain == nil {
		t.Fatal("PerfByChain should be populated for ByModeCategory")
	}
	if *got[0].PerfByChain["arena_slayer"] != 70 {
		t.Errorf("PerfByChain[arena_slayer] = %v, want 70", *got[0].PerfByChain["arena_slayer"])
	}
	if *got[0].PerfByChain["arena_objectif"] != 90 {
		t.Errorf("PerfByChain[arena_objectif] = %v, want 90", *got[0].PerfByChain["arena_objectif"])
	}
}

// TestByMode_NoPerfByChain : ByMode (sous-mode) ne doit PAS peupler PerfByChain
// car en pratique 1 sous-mode → 1 chaîne unique. La granularité de
// AvgPerformanceScore est déjà suffisante.
func TestByMode_NoPerfByChain(t *testing.T) {
	t.Parallel()
	rows := []Row{
		{ModeName: "Slayer", Outcome: canonical.OutcomeWin,
			PerformanceScore: ptrF(75), PerformanceChain: "arena_slayer"},
		{ModeName: "Slayer", Outcome: canonical.OutcomeLoss,
			PerformanceScore: ptrF(45), PerformanceChain: "arena_slayer"},
	}
	got := ByMode(rows)
	if got[0].PerfByChain != nil {
		t.Errorf("ByMode.PerfByChain should be nil (1 sous-mode → 1 chaîne), got %v", got[0].PerfByChain)
	}
	// AvgPerformanceScore reste correctement calculé.
	if *got[0].AvgPerformanceScore != 60 {
		t.Errorf("AvgPerformanceScore = %v, want 60", *got[0].AvgPerformanceScore)
	}
}
