package service

import (
	"context"
	"testing"

	"levelup/go-api/internal/games"
	"levelup/go-api/internal/games/canonical"
)

type fakeTotalsLoader struct {
	totals []canonical.CommendationTotal
	err    error
}

func (f *fakeTotalsLoader) LoadCommendationTotals(_ context.Context) ([]canonical.CommendationTotal, error) {
	return f.totals, f.err
}

// TestCommendationTotalsService_GroupsByCategory : groupe par catégorie (alpha),
// items triés total DESC, catégorie vide → "AUTRE", TotalCount = nb de commendations.
func TestCommendationTotalsService_GroupsByCategory(t *testing.T) {
	icon := "https://cdn.test/a.png"
	svc := NewCommendationTotalsService(&fakeTotalsLoader{totals: []canonical.CommendationTotal{
		{ID: "a", Name: "Alpha", Category: "MULTIPLAYER", Total: 50, IconURL: &icon},
		{ID: "b", Name: "Bravo", Category: "MULTIPLAYER", Total: 100},
		{ID: "c", Name: "Charlie", Category: "GAME MODE", Total: 10},
		{ID: "d", Name: "Delta", Category: "", Total: 5}, // → "AUTRE"
	}})
	resp, err := svc.GetTotals(context.Background())
	if err != nil {
		t.Fatalf("GetTotals: %v", err)
	}
	if resp.TotalCount != 4 {
		t.Errorf("TotalCount = %d, want 4 (commendations distinctes)", resp.TotalCount)
	}
	if len(resp.Categories) != 3 {
		t.Fatalf("categories = %d, want 3 — %+v", len(resp.Categories), resp.Categories)
	}
	// Catégories triées alpha : AUTRE, GAME MODE, MULTIPLAYER.
	if resp.Categories[0].Category != "AUTRE" || resp.Categories[2].Category != "MULTIPLAYER" {
		t.Errorf("ordre catégories: %s .. %s", resp.Categories[0].Category, resp.Categories[2].Category)
	}
	// MULTIPLAYER items triés total DESC : Bravo (100) avant Alpha (50).
	mp := resp.Categories[2]
	if len(mp.Items) != 2 || mp.Items[0].Name != "Bravo" || mp.Items[1].Name != "Alpha" {
		t.Errorf("MULTIPLAYER mal trié: %+v", mp.Items)
	}
}

// TestCommendationTotalsService_NilLoader_Empty : loader nil → réponse vide (non-nil).
func TestCommendationTotalsService_NilLoader_Empty(t *testing.T) {
	resp, err := NewCommendationTotalsService(nil).GetTotals(context.Background())
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if resp == nil || len(resp.Categories) != 0 || resp.TotalCount != 0 {
		t.Errorf("loader nil → réponse vide attendue, got %+v", resp)
	}
}

// TestCommendationTotalsService_CapabilityNotSupported_Empty : ErrCapabilityNotSupported
// dégrade en réponse vide (pas une erreur 500).
func TestCommendationTotalsService_CapabilityNotSupported_Empty(t *testing.T) {
	svc := NewCommendationTotalsService(&fakeTotalsLoader{err: games.ErrCapabilityNotSupported})
	resp, err := svc.GetTotals(context.Background())
	if err != nil {
		t.Fatalf("ErrCapabilityNotSupported doit dégrader, got err: %v", err)
	}
	if len(resp.Categories) != 0 || resp.TotalCount != 0 {
		t.Errorf("réponse vide attendue, got %+v", resp)
	}
}

// TestCommendationTotalsService_TierProgression : la progression à vie (paliers /
// prochain seuil / maîtrisé) est calculée depuis Total + TierTargets, avec
// dégradation propre (tier_count=0) quand les paliers ne sont pas seedés.
func TestCommendationTotalsService_TierProgression(t *testing.T) {
	svc := NewCommendationTotalsService(&fakeTotalsLoader{totals: []canonical.CommendationTotal{
		// En cours : 45 sur 10/20/30/50/100 → 3 paliers atteints, prochain = 50.
		{ID: "a", Name: "Kills", Category: "MULTIPLAYER", Total: 45, TierTargets: "10,20,30,50,100"},
		// Maîtrisée : 120 >= dernier palier → tier_index == tier_count, mastered, 100%.
		{ID: "b", Name: "Assists", Category: "MULTIPLAYER", Total: 120, TierTargets: "10,20,30,50,100"},
		// Sans paliers seedés → tier_count=0, pas d'anneau, total brut conservé.
		{ID: "c", Name: "Legacy", Category: "GAME MODE", Total: 7, TierTargets: ""},
	}})
	resp, err := svc.GetTotals(context.Background())
	if err != nil {
		t.Fatalf("GetTotals: %v", err)
	}
	byID := map[string]domainItem{}
	for _, g := range resp.Categories {
		for _, it := range g.Items {
			byID[it.ID] = domainItem{it.TierIndex, it.TierCount, it.NextTierTarget, it.ProgressPct, it.IsMastered}
		}
	}
	if got := byID["a"]; got.tierIndex != 3 || got.tierCount != 5 || got.nextTarget != 50 || got.mastered {
		t.Errorf("a (en cours) = %+v, want tierIndex=3 tierCount=5 next=50 mastered=false", got)
	}
	if got := byID["b"]; got.tierIndex != 5 || got.tierCount != 5 || !got.mastered || got.pct != 100.0 {
		t.Errorf("b (maîtrisée) = %+v, want tierIndex=5 tierCount=5 mastered=true pct=100", got)
	}
	if got := byID["c"]; got.tierCount != 0 || got.mastered || got.pct != 0 {
		t.Errorf("c (sans paliers) = %+v, want tierCount=0 mastered=false pct=0", got)
	}
}

type domainItem struct {
	tierIndex, tierCount, nextTarget int
	pct                              float64
	mastered                         bool
}
