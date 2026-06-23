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
