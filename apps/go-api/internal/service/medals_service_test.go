package service

import (
	"context"
	"errors"
	"testing"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/testutil"
)

// mockMedalsRepo implémente port.MedalsRepository pour les tests du service.
type mockMedalsRepo struct {
	catalog []domain.MedalCatalogRow
	earned  []domain.MedalEarnedRow
	catErr  error
	earnErr error
}

func (m *mockMedalsRepo) ListAllMedals(context.Context, string) ([]domain.MedalCatalogRow, error) {
	return m.catalog, m.catErr
}

func (m *mockMedalsRepo) LoadMedalTotals(context.Context, string) ([]domain.MedalEarnedRow, error) {
	return m.earned, m.earnErr
}

func TestMedalsService_GetMedalsPage(t *testing.T) {
	repo := &mockMedalsRepo{
		catalog: []domain.MedalCatalogRow{
			{MedalID: 10, Label: "Killjoy", Difficulty: "Normal", MedalType: "skill", DifficultyIndex: 0},
			{MedalID: 20, Label: "Double Kill", Difficulty: "Heroic", MedalType: "multikill", DifficultyIndex: 1},
		},
		earned: []domain.MedalEarnedRow{{MedalID: 10, TotalCount: 4}},
	}
	svc := NewMedalsService(repo)
	resp, err := svc.GetMedalsPage(context.Background(), "xuid-1")
	if err != nil {
		t.Fatalf("GetMedalsPage: %v", err)
	}
	testutil.RequireNoNilSlicesWithoutOmitempty(t, resp)

	if resp.CatalogTotal != 2 || resp.EarnedTotal != 1 || resp.TotalCount != 4 {
		t.Errorf("totaux = catalog %d earned %d count %d, want 2/1/4",
			resp.CatalogTotal, resp.EarnedTotal, resp.TotalCount)
	}
	if len(resp.Medals) != 2 {
		t.Fatalf("Medals len = %d, want 2", len(resp.Medals))
	}
	// Baseline (aucun resolver enregistré dans le package service) : catégorie = medal_type.
	for _, m := range resp.Medals {
		if m.SuperSection != "other" {
			t.Errorf("medal %d super_section = %q, want other (baseline)", m.MedalID, m.SuperSection)
		}
	}
}

func TestMedalsService_GetMedalsPage_Empty(t *testing.T) {
	svc := NewMedalsService(&mockMedalsRepo{})
	resp, err := svc.GetMedalsPage(context.Background(), "xuid-1")
	if err != nil {
		t.Fatalf("GetMedalsPage: %v", err)
	}
	testutil.RequireNoNilSlicesWithoutOmitempty(t, resp)
	if len(resp.Medals) != 0 || len(resp.Categories) != 0 {
		t.Errorf("réponse vide attendue, got Medals=%d Categories=%d", len(resp.Medals), len(resp.Categories))
	}
}

// TestMedalsService_EarnedError_Degrades : une erreur de totaux dégrade en catalogue
// seul (compteurs à 0), sans propager d'erreur (page toujours affichable).
func TestMedalsService_EarnedError_Degrades(t *testing.T) {
	repo := &mockMedalsRepo{
		catalog: []domain.MedalCatalogRow{{MedalID: 10, Label: "Killjoy", MedalType: "skill"}},
		earnErr: errors.New("shared reader indisponible"),
	}
	resp, err := NewMedalsService(repo).GetMedalsPage(context.Background(), "xuid-1")
	if err != nil {
		t.Fatalf("dégradation attendue sans erreur, got %v", err)
	}
	if len(resp.Medals) != 1 || resp.Medals[0].Count != 0 {
		t.Errorf("catalogue seul attendu (Count 0), got %+v", resp.Medals)
	}
}

// TestMedalsService_CatalogError_Propagates : une erreur de catalogue est propagée
// (le catalogue est essentiel, pas de page sans lui).
func TestMedalsService_CatalogError_Propagates(t *testing.T) {
	repo := &mockMedalsRepo{catErr: errors.New("metadata indisponible")}
	if _, err := NewMedalsService(repo).GetMedalsPage(context.Background(), "xuid-1"); err == nil {
		t.Fatal("erreur catalogue attendue, got nil")
	}
}
