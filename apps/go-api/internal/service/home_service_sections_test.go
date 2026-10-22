// Package service — home_service_sections_test.go : l'Accueil et la vue match déclarent
// leurs sections de durée (ADR 0036 I6), lues dans la ligne `http_timings` des requêtes
// lentes pour savoir où passe le temps d'une page.
package service

import (
	"context"
	"testing"
	"time"

	"levelup/go-api/internal/observability/timing"
)

// TestGetHomePage_SectionsDeDuree : chaque étape de l'Accueil porte sa section, les
// enrichissements par tuile une par liste.
func TestGetHomePage_SectionsDeDuree(t *testing.T) {
	repo := homeReplayRepo(time.Now())
	svc := withHomeMock(NewHomeService(repo), repo).WithReplay(&stubReplayService{})
	ctx, chrono := timing.WithTimings(context.Background())
	if _, err := svc.GetHomePage(ctx, "TestGT", "fr"); err != nil {
		t.Fatalf("GetHomePage: %v", err)
	}
	vues := map[string]bool{}
	for _, s := range chrono.Snapshot() {
		vues[s.Name] = true
	}
	for _, section := range []string{
		"home_load_matches", "home_spartan_identity", "home_count_matches",
		"home_playlist_ranks", "home_favorite_weapon", "home_asset_translations", "home_build_cards",
		"home_build_tiles", "home_perf_placements", "home_replay_availability", "home_build_sessions", "home_squad_teammates",
		"home_medals_recent", "home_citations_recent", "home_commendations_recent", "home_assisted_frags_recent",
		"home_analysis_pending",
	} {
		if !vues[section] {
			t.Errorf("section %q absente des durées de l'Accueil (sections vues : %v)", section, vues)
		}
	}
}

// TestMatchViewSection_FeuillesSeulement : un chargeur de la vue match ouvre `mv_<source>`
// (espaces remplacés) ; une source que son repo chronomètre déjà n'en ouvre pas.
func TestMatchViewSection_FeuillesSeulement(t *testing.T) {
	ctx, chrono := timing.WithTimings(context.Background())
	matchViewSection(ctx, "objective score")()
	for source := range matchViewSourcesTimedInRepo {
		matchViewSection(ctx, source)()
	}
	got := chrono.Snapshot()
	if len(got) != 1 || got[0].Name != "mv_objective_score" {
		t.Fatalf("sections = %+v, want la seule mv_objective_score", got)
	}
}
