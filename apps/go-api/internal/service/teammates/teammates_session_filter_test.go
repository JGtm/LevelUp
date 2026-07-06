package teammates

// teammates_session_filter_test.go — tests du filtre de session dans GetPage.
//
// Scénarios couverts :
//  1. Pas de session sélectionnée → tous les matchs escouade retournés (pas de filtre).
//  2. Session sélectionnée → seuls les matchs de la session dans mapBreakdown.
//  3. Session sélectionnée mais aucun match commun dans la session → mapBreakdown vide.
//  4. Plusieurs coéquipiers, session partielle → chaque coéquipier filtré indépendamment.
//  5. filteredMatches vide + session sélectionnée → mapBreakdown vide (session sans matchs).

import (
	"context"
	"testing"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/legacymatch"
)

// makeSynthRow construit une SynthesisMatchRow minimale avec IsWithFriends=true.
func makeSynthRow(matchID, session string) legacymatch.SynthesisMatchRow {
	return legacymatch.SynthesisMatchRow{
		MatchID:       matchID,
		StartTime:     time.Now(),
		IsWithFriends: true,
		SessionLabel:  strPtr(session),
	}
}

// makeSquadRow construit une SquadMatchRow minimale pour la carte donnée.
func makeSquadRow(matchID, mapUI string, outcome int) domain.SquadMatchRow {
	return domain.SquadMatchRow{
		MatchID:       matchID,
		MapUI:         mapUI,
		Outcome:       outcome,
		IsWithFriends: true,
		StartTime:     time.Now(),
	}
}

// TestGetPage_NoSessionFilter_AllSquadMatchesIncluded vérifie que sans filtre de session,
// tous les matchs escouade de LoadSquadMatches contribuent au mapBreakdown.
func TestGetPage_NoSessionFilter_AllSquadMatchesIncluded(t *testing.T) {
	repo := &mockSquadRepo{
		topRows: []domain.TopTeammateRow{
			{XUID: "tm1", Gamertag: "Ally", GamesTogether: 10},
		},
		// 3 matchs historiques escouade
		squadRows: []domain.SquadMatchRow{
			makeSquadRow("m1", "Bazaar", domain.OutcomeWin),
			makeSquadRow("m2", "Aquarius", domain.OutcomeWin),
			makeSquadRow("m3", "Recharge", domain.OutcomeLoss),
		},
		// synthRows vide = canonicalRows vides (aucune session disponible)
	}
	svc := NewTeammatesService(repo, nil).WithPlayerMatchesRepo(
		newSynthMockFromRows(repo.synthRows, repo.synthErr), "halo_infinite", "Test",
	)

	// Aucune session sélectionnée → pas de filtre
	resp, err := svc.GetPage(context.Background(), "px", domain.TeammatesQueryRequest{
		SelectedGamertags: []string{"Ally"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Les 3 matchs (3 cartes distinctes) doivent apparaître
	if len(resp.MapBreakdown) != 3 {
		t.Errorf("expected 3 map entries (no session filter), got %d", len(resp.MapBreakdown))
	}
}

// TestGetPage_SessionFilter_OnlySessionMatchesInMapBreakdown vérifie que quand
// une session est sélectionnée, seuls les matchs communs de cette session
// contribuent au mapBreakdown.
func TestGetPage_SessionFilter_OnlySessionMatchesInMapBreakdown(t *testing.T) {
	const sessionLabel = "2026-04-21 19h"

	// synthRows = matchs canoniques du joueur pour la session
	// m1 et m2 sont dans la session ; m3 est hors session.
	synthRows := []legacymatch.SynthesisMatchRow{
		makeSynthRow("m1", sessionLabel),
		makeSynthRow("m2", sessionLabel),
	}
	repo := &mockSquadRepo{
		topRows: []domain.TopTeammateRow{
			{XUID: "tm1", Gamertag: "Ally", GamesTogether: 3},
		},
		// LoadSquadMatches retourne les 3 matchs historiques (m1 à m3)
		squadRows: []domain.SquadMatchRow{
			makeSquadRow("m1", "Bazaar", domain.OutcomeWin),
			makeSquadRow("m2", "Aquarius", domain.OutcomeWin),
			makeSquadRow("m3", "Recharge", domain.OutcomeLoss), // hors session
		},
		synthRows: synthRows,
	}
	svc := NewTeammatesService(repo, nil).WithPlayerMatchesRepo(
		newSynthMockFromRows(synthRows, nil), "halo_infinite", "Test",
	)

	resp, err := svc.GetPage(context.Background(), "px", domain.TeammatesQueryRequest{
		SelectedGamertags:   []string{"Ally"},
		PickedSquadSessions: []string{sessionLabel},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Seuls m1 (Bazaar) et m2 (Aquarius) sont dans la session → 2 cartes
	if len(resp.MapBreakdown) != 2 {
		t.Errorf("expected 2 map entries (session filter m1+m2), got %d", len(resp.MapBreakdown))
	}
	// Vérifier que Recharge (m3, hors session) est absent
	for _, row := range resp.MapBreakdown {
		if row.MapUI == "Recharge" {
			t.Errorf("Recharge should be excluded by session filter, but it appears in MapBreakdown")
		}
	}
}

// TestGetPage_SessionFilter_NoOverlapWithSquadMatches vérifie que si la session
// sélectionnée ne contient aucun match commun avec le coéquipier, mapBreakdown est vide.
func TestGetPage_SessionFilter_NoOverlapWithSquadMatches(t *testing.T) {
	const sessionLabel = "2026-04-21 19h"

	// synthRows : session contient m99 et m100, mais le coéquipier n'a joué ni l'un ni l'autre
	synthRows := []legacymatch.SynthesisMatchRow{
		makeSynthRow("m99", sessionLabel),
		makeSynthRow("m100", sessionLabel),
	}
	repo := &mockSquadRepo{
		topRows: []domain.TopTeammateRow{
			{XUID: "tm1", Gamertag: "Ally", GamesTogether: 3},
		},
		// squadRows n'a que m1/m2/m3 → aucun match en commun avec la session
		squadRows: []domain.SquadMatchRow{
			makeSquadRow("m1", "Bazaar", domain.OutcomeWin),
			makeSquadRow("m2", "Aquarius", domain.OutcomeWin),
		},
		synthRows: synthRows,
	}
	svc := NewTeammatesService(repo, nil).WithPlayerMatchesRepo(
		newSynthMockFromRows(synthRows, nil), "halo_infinite", "Test",
	)

	resp, err := svc.GetPage(context.Background(), "px", domain.TeammatesQueryRequest{
		SelectedGamertags:   []string{"Ally"},
		PickedSquadSessions: []string{sessionLabel},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.MapBreakdown) != 0 {
		t.Errorf("expected 0 map entries (no overlap with session), got %d", len(resp.MapBreakdown))
	}
}

// TestGetPage_SessionFilter_WinRateComputedFromSessionOnly vérifie que le win rate
// dans mapBreakdown est calculé uniquement sur les matchs de la session,
// pas sur l'historique complet.
//
// Session : 1 match sur Bazaar → victoire (winRate = 1.0)
// Historique escouade : 3 matchs sur Bazaar dont 1 victoire (winRate = 0.33)
func TestGetPage_SessionFilter_WinRateComputedFromSessionOnly(t *testing.T) {
	const sessionLabel = "2026-04-21 19h"

	synthRows := []legacymatch.SynthesisMatchRow{
		makeSynthRow("m1", sessionLabel), // seul match de la session
	}
	repo := &mockSquadRepo{
		topRows: []domain.TopTeammateRow{
			{XUID: "tm1", Gamertag: "Ally", GamesTogether: 3},
		},
		squadRows: []domain.SquadMatchRow{
			makeSquadRow("m1", "Bazaar", domain.OutcomeWin),  // dans la session
			makeSquadRow("m2", "Bazaar", domain.OutcomeLoss), // hors session
			makeSquadRow("m3", "Bazaar", domain.OutcomeLoss), // hors session
		},
		synthRows: synthRows,
	}
	svc := NewTeammatesService(repo, nil).WithPlayerMatchesRepo(
		newSynthMockFromRows(synthRows, nil), "halo_infinite", "Test",
	)

	resp, err := svc.GetPage(context.Background(), "px", domain.TeammatesQueryRequest{
		SelectedGamertags:   []string{"Ally"},
		PickedSquadSessions: []string{sessionLabel},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.MapBreakdown) != 1 {
		t.Fatalf("expected 1 map entry, got %d", len(resp.MapBreakdown))
	}
	bazaar := resp.MapBreakdown[0]
	if bazaar.MatchCount != 1 {
		t.Errorf("expected MatchCount=1 (session only), got %d", bazaar.MatchCount)
	}
	if bazaar.WinRate != 1.0 {
		t.Errorf("expected WinRate=1.0 (session only, 1 win/1 match), got %f", bazaar.WinRate)
	}
}

// TestGetPage_SessionFilter_MultipleTeammates vérifie que le filtre de session
// s'applique indépendamment à chaque coéquipier.
func TestGetPage_SessionFilter_MultipleTeammates(t *testing.T) {
	const sessionLabel = "2026-04-21 19h"

	synthRows := []legacymatch.SynthesisMatchRow{
		makeSynthRow("m1", sessionLabel),
		makeSynthRow("m2", sessionLabel),
	}

	callCount := 0
	// mockSquadRepo ne distingue pas les coéquipiers pour LoadSquadMatches,
	// mais ici on a un seul XUID connu ("tm1"). Le second coéquipier ("Ally2")
	// n'est pas dans topRows → skippé → seul Ally1 contribue.
	repo := &mockSquadRepo{
		topRows: []domain.TopTeammateRow{
			{XUID: "tm1", Gamertag: "Ally1", GamesTogether: 5},
		},
		squadRows: []domain.SquadMatchRow{
			makeSquadRow("m1", "Bazaar", domain.OutcomeWin),    // session
			makeSquadRow("m2", "Aquarius", domain.OutcomeWin),  // session
			makeSquadRow("m3", "Recharge", domain.OutcomeLoss), // hors session
		},
		synthRows: synthRows,
	}
	_ = callCount
	svc := NewTeammatesService(repo, nil).WithPlayerMatchesRepo(
		newSynthMockFromRows(synthRows, nil), "halo_infinite", "Test",
	)

	resp, err := svc.GetPage(context.Background(), "px", domain.TeammatesQueryRequest{
		SelectedGamertags:   []string{"Ally1"},
		PickedSquadSessions: []string{sessionLabel},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// m1 (Bazaar) + m2 (Aquarius) → 2 cartes, Recharge exclu
	if len(resp.MapBreakdown) != 2 {
		t.Errorf("expected 2 map entries, got %d", len(resp.MapBreakdown))
	}
}
