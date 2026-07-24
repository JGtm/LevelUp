package service

import (
	"context"
	"testing"

	"levelup/go-api/internal/analysis/relations"
	"levelup/go-api/internal/domain"
)

// TestBuildExplorerFragGapSeries : somme préfixe directionnelle (frags − morts)
// + mapping d'issue canonique, ancien→récent. Vide → nil (front masque le graphe).
func TestBuildExplorerFragGapSeries(t *testing.T) {
	t.Parallel()
	if got := buildExplorerFragGapSeries(nil); got != nil {
		t.Fatalf("nil input → nil, got %v", got)
	}
	raw := []domain.RelationDuelRawRow{
		{Result: relations.ResultWin, KillsOnRival: 3, DeathsByRival: 1},  // +2 → cum 2
		{Result: relations.ResultLoss, KillsOnRival: 1, DeathsByRival: 4}, // -3 → cum -1
		{Result: 0, KillsOnRival: 2, DeathsByRival: 2},                    // 0  → cum -1
	}
	want := []domain.ExplorerFragGapPoint{
		{Cumulative: 2, Outcome: "win"},
		{Cumulative: -1, Outcome: "loss"},
		{Cumulative: -1, Outcome: "other"},
	}
	got := buildExplorerFragGapSeries(raw)
	if len(got) != len(want) {
		t.Fatalf("len = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("point %d = %+v, want %+v", i, got[i], want[i])
		}
	}
}

// mockExplorerRelations : provider relationnel de test (WR historique + timeline).
type mockExplorerRelations struct {
	wr     *float64
	duels  []domain.RelationDuelRawRow
	engErr error
	tlErr  error
}

func (m *mockExplorerRelations) GetCoreEngagement(_ context.Context, _ []string, _ []string, _ int) (domain.CoreEngagement, error) {
	return domain.CoreEngagement{PlayerWinRate: m.wr}, m.engErr
}

func (m *mockExplorerRelations) GetRivalTimeline(_ context.Context, _ string, _ []string, _ int) ([]domain.RelationDuelRawRow, error) {
	return m.duels, m.tlErr
}

// TestEnrichEncounterRelations : le WR historique alimente PlayerWinRate (repère
// des donuts) et la timeline de duels alimente FragGapSeries (courbe d'écart).
func TestEnrichEncounterRelations(t *testing.T) {
	t.Parallel()
	wr := 0.55
	prov := &mockExplorerRelations{
		wr: &wr,
		duels: []domain.RelationDuelRawRow{
			{Result: relations.ResultWin, KillsOnRival: 4, DeathsByRival: 1},  // +3
			{Result: relations.ResultLoss, KillsOnRival: 0, DeathsByRival: 2}, // -2 → cum +1
		},
	}
	svc := NewExplorerService(&mockExplorerRepo{}, "self").
		WithTargetProfileProviders(ExplorerTargetProfileDeps{Relations: prov})

	stats := &domain.ExplorerEncounterStats{CountTogether: 2}
	svc.enrichEncounterRelations(context.Background(), stats, "target-x")

	if stats.PlayerWinRate == nil || *stats.PlayerWinRate != wr {
		t.Fatalf("PlayerWinRate = %v, want %v", stats.PlayerWinRate, wr)
	}
	if len(stats.FragGapSeries) != 2 {
		t.Fatalf("FragGapSeries len = %d, want 2", len(stats.FragGapSeries))
	}
	if stats.FragGapSeries[0].Cumulative != 3 || stats.FragGapSeries[1].Cumulative != 1 {
		t.Errorf("cumulatives = %d/%d, want 3/1", stats.FragGapSeries[0].Cumulative, stats.FragGapSeries[1].Cumulative)
	}
}

// TestEnrichEncounterRelations_NoProvider : sans provider (nil), aucune mutation
// (donuts sans repère + graphe masqué côté front) — dégradation gracieuse.
func TestEnrichEncounterRelations_NoProvider(t *testing.T) {
	t.Parallel()
	svc := NewExplorerService(&mockExplorerRepo{}, "self")
	stats := &domain.ExplorerEncounterStats{CountTogether: 1}
	svc.enrichEncounterRelations(context.Background(), stats, "target-x")
	if stats.PlayerWinRate != nil || stats.FragGapSeries != nil {
		t.Errorf("no-op attendu sans provider, got wr=%v series=%v", stats.PlayerWinRate, stats.FragGapSeries)
	}
	// stats nil → no-op (pas de panic).
	svc.enrichEncounterRelations(context.Background(), nil, "target-x")
}
