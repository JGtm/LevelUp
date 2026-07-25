package service

import (
	"context"
	"errors"
	"testing"

	"levelup/go-api/internal/ctxkeys"
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

// TestMedalsService_GetMedalsPage_GhostMedalMasked (V72-33) : une médaille gagnée en
// match réel mais enregistrée comme fantôme (sans nom ni icône exploitables, cf.
// halo_5.GhostMedalIDs) est exclue de la réponse — la médaille normale du même
// catalogue est conservée, et les totaux n'incluent pas le fantôme. Slug de test
// dédié (pas halo_5/halo_infinite) pour ne pas muter l'état global partagé par les
// autres tests du package (RegisterGhostMedalIDs est un registre boot-once).
func TestMedalsService_GetMedalsPage_GhostMedalMasked(t *testing.T) {
	const testSlug = "test_ghost_title"
	const ghostID = int64(999888777)
	RegisterGhostMedalIDs(testSlug, map[int64]bool{ghostID: true})

	repo := &mockMedalsRepo{
		catalog: []domain.MedalCatalogRow{
			{MedalID: 10, Label: "Killjoy", Difficulty: "Normal", MedalType: "skill"},
		},
		earned: []domain.MedalEarnedRow{
			{MedalID: 10, TotalCount: 2},
			{MedalID: ghostID, TotalCount: 5}, // gagnée mais absente du catalogue + masquée
		},
	}
	ctx := ctxkeys.WithTitleSlug(context.Background(), testSlug)
	resp, err := NewMedalsService(repo).GetMedalsPage(ctx, "xuid-1")
	if err != nil {
		t.Fatalf("GetMedalsPage: %v", err)
	}
	if len(resp.Medals) != 1 || resp.Medals[0].MedalID != 10 {
		t.Fatalf("Medals = %+v, want [medal 10 seul] (fantôme %d masqué)", resp.Medals, ghostID)
	}
	if resp.CatalogTotal != 1 || resp.TotalCount != 2 {
		t.Errorf("totaux = catalog %d count %d, want 1/2 (fantôme exclu des agrégats)",
			resp.CatalogTotal, resp.TotalCount)
	}
}

// TestMedalsService_GetMedalsPage_NoGhostRegistered_NormalMedalKept : un titre sans
// allowlist fantôme enregistrée conserve toutes ses médailles obtenues (comportement
// par défaut inchangé — aucune régression sur les titres non concernés par V72-33).
func TestMedalsService_GetMedalsPage_NoGhostRegistered_NormalMedalKept(t *testing.T) {
	repo := &mockMedalsRepo{
		catalog: []domain.MedalCatalogRow{{MedalID: 10, Label: "Killjoy", MedalType: "skill"}},
		earned:  []domain.MedalEarnedRow{{MedalID: 10, TotalCount: 2}},
	}
	ctx := ctxkeys.WithTitleSlug(context.Background(), "test_no_ghost_title")
	resp, err := NewMedalsService(repo).GetMedalsPage(ctx, "xuid-1")
	if err != nil {
		t.Fatalf("GetMedalsPage: %v", err)
	}
	if len(resp.Medals) != 1 || resp.Medals[0].MedalID != 10 {
		t.Fatalf("Medals = %+v, want [medal 10] (médaille normale conservée)", resp.Medals)
	}
}

// TestFilterGhostMedals couvre la fonction pure (0 dépendance service) : ID masqué
// retiré, ID inconnu conservé, allowlist vide/nil = no-op.
func TestFilterGhostMedals(t *testing.T) {
	items := []domain.MedalSummaryItem{
		{MedalID: 1, Name: "A"},
		{MedalID: 2, Name: "#2"},
		{MedalID: 3, Name: "C"},
	}
	out := filterGhostMedals(items, map[int64]bool{2: true})
	if len(out) != 2 || out[0].MedalID != 1 || out[1].MedalID != 3 {
		t.Fatalf("out = %+v, want medals [1,3] (2 masqué)", out)
	}

	// Aucun ID masqué pour ce titre → no-op (slice indépendante de l'appel ci-dessus,
	// filterGhostMedals ne mute jamais son entrée).
	fresh := []domain.MedalSummaryItem{{MedalID: 1}, {MedalID: 2}, {MedalID: 3}}
	out2 := filterGhostMedals(fresh, nil)
	if len(out2) != 3 {
		t.Errorf("ghosts vide : len = %d, want 3 (no-op)", len(out2))
	}
}

// TestRegisterGhostMedalIDs_NoopGuards : slug vide ou allowlist vide/nil n'altère pas
// le registre (miroir de RegisterMedalCategoryResolver).
func TestRegisterGhostMedalIDs_NoopGuards(t *testing.T) {
	before := len(ghostMedalIDsBySlug)
	RegisterGhostMedalIDs("", map[int64]bool{1: true})
	RegisterGhostMedalIDs("some_slug_never_used_elsewhere", nil)
	RegisterGhostMedalIDs("some_slug_never_used_elsewhere", map[int64]bool{})
	if len(ghostMedalIDsBySlug) != before {
		t.Errorf("no-op attendu (slug vide ou ids vide), registre a changé : %d → %d",
			before, len(ghostMedalIDsBySlug))
	}
}
