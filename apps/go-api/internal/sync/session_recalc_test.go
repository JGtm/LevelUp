//go:build integration

// Package sync — session_recalc_test.go : tests d'intégration du recalcul de sessions.
package sync

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/sync/testutil"
)

// seedSharedForRecalc est réservé à de futurs helpers de test.

func TestLookupFriendXUIDs_Found(t *testing.T) {
	db := testutil.NewInMemoryShared(t)

	_, err := db.Exec(`INSERT INTO xuid_aliases (xuid, gamertag) VALUES ('xuid-1', 'PlayerOne')`)
	if err != nil {
		t.Fatalf("seed: %v", err)
	}

	xuids := LookupFriendXUIDs(t.Context(), db, []string{"PlayerOne"})
	if len(xuids) != 1 || xuids[0] != "xuid-1" {
		t.Errorf("expected [xuid-1], got %v", xuids)
	}
}

func TestLookupFriendXUIDs_CaseInsensitive(t *testing.T) {
	db := testutil.NewInMemoryShared(t)

	_, err := db.Exec(`INSERT INTO xuid_aliases (xuid, gamertag) VALUES ('xuid-2', 'PlayerTwo')`)
	if err != nil {
		t.Fatalf("seed: %v", err)
	}

	xuids := LookupFriendXUIDs(t.Context(), db, []string{"playertwo"})
	if len(xuids) != 1 || xuids[0] != "xuid-2" {
		t.Errorf("expected [xuid-2], got %v", xuids)
	}
}

func TestLookupFriendXUIDs_NotFound(t *testing.T) {
	db := testutil.NewInMemoryShared(t)

	xuids := LookupFriendXUIDs(t.Context(), db, []string{"Inconnu"})
	if len(xuids) != 0 {
		t.Errorf("expected empty, got %v", xuids)
	}
}

func TestLookupFriendXUIDs_EmptySlice(t *testing.T) {
	db := testutil.NewInMemoryShared(t)
	xuids := LookupFriendXUIDs(t.Context(), db, nil)
	if xuids != nil {
		t.Errorf("expected nil, got %v", xuids)
	}
	xuids = LookupFriendXUIDs(t.Context(), db, []string{})
	if xuids != nil {
		t.Errorf("expected nil for empty slice, got %v", xuids)
	}
}

func TestLookupFriendXUIDs_PartialMatch(t *testing.T) {
	db := testutil.NewInMemoryShared(t)

	_, err := db.Exec(`INSERT INTO xuid_aliases (xuid, gamertag) VALUES ('xuid-3', 'PlayerThree')`)
	if err != nil {
		t.Fatalf("seed: %v", err)
	}

	// Un gamertag trouvé, un absent → seul le trouvé retourné.
	xuids := LookupFriendXUIDs(t.Context(), db, []string{"PlayerThree", "GhostPlayer"})
	if len(xuids) != 1 || xuids[0] != "xuid-3" {
		t.Errorf("expected [xuid-3], got %v", xuids)
	}
}

func TestLoadSessionMatchRowsDirect_EmptyDB(t *testing.T) {
	db := testutil.NewInMemoryShared(t)

	rows, err := loadSessionMatchRowsDirect(context.Background(), db, "xuid-nobody")
	if err != nil {
		t.Fatalf("loadSessionMatchRowsDirect: %v", err)
	}
	if len(rows) != 0 {
		t.Errorf("expected empty result, got %d rows", len(rows))
	}
}

func TestLoadSessionMatchRowsDirect_WithData(t *testing.T) {
	db := testutil.NewInMemoryShared(t)

	t1 := time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)
	t2 := time.Date(2024, 1, 1, 11, 0, 0, 0, time.UTC)

	_, err := db.Exec(`INSERT INTO match_registry (match_id, start_time, is_ranked) VALUES ('m1', ?, FALSE), ('m2', ?, TRUE)`,
		t1, t2)
	if err != nil {
		t.Fatalf("seed registry: %v", err)
	}
	_, err = db.Exec(`INSERT INTO match_participants (match_id, xuid, gamertag, team_id, outcome, rank, score, time_played_seconds) VALUES
		('m1', 'xuid-a', 'PlayerA', 0, 2, 1, 100, 600),
		('m2', 'xuid-a', 'PlayerA', 0, 3, 2, 80, 600)`)
	if err != nil {
		t.Fatalf("seed participants: %v", err)
	}

	rows, err := loadSessionMatchRowsDirect(context.Background(), db, "xuid-a")
	if err != nil {
		t.Fatalf("loadSessionMatchRowsDirect: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(rows))
	}

	// Vérifie l'ordre chronologique.
	if rows[0].MatchID != "m1" || rows[1].MatchID != "m2" {
		t.Errorf("expected m1, m2 order; got %q, %q", rows[0].MatchID, rows[1].MatchID)
	}
	if rows[1].IsRanked != true {
		t.Errorf("expected m2 is_ranked=true, got %v", rows[1].IsRanked)
	}
}

// ─── Exclusion de la Campagne (E2 du plan .ai/PLAN_BACKLOG_2026-10-15.md, DB-27) ───
//
// Chaque cas sème, pour le même joueur, un match d'arène (arena1, 01/09) et un match de
// Campagne PLUS RÉCENT (camp1, 02/09), puis lit par le VRAI chargeur, le titre posé dans le
// contexte comme le fait le moteur de sync. Halo 5 : la Campagne disparaît. Halo Infinite
// (aucune variante masquée) : le résolveur est neutre, les deux matchs restent.

const campagneXUID = "2533274800000001"

var campagneTitres = []struct {
	slug    string
	masquee bool
}{{"halo_5", true}, {"halo_infinite", false}, {"", true}}

func campagneAttendu(masquee bool, sans, avec []string) []string {
	if masquee {
		return sans
	}
	return avec
}

func campagneEgal(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// seedCampagneArenaEtCampagne sème arena1 (arène) et camp1 (Campagne, plus récent) pour
// campagneXUID dans un shared de test.
func seedCampagneArenaEtCampagne(t *testing.T, db *sql.DB) {
	t.Helper()
	camp := analysis.CampaignExcludedVariantIDs("halo_5")[0]
	for _, s := range []string{
		`INSERT INTO match_registry (match_id, start_time, start_time_utc, end_time, duration_seconds,
			game_variant_id, pair_name) VALUES
			('arena1', '2026-09-01 10:00:00', '2026-09-01 10:00:00+00', '2026-09-01 10:10:00', 600,
			 'aaaaaaaa-0000-0000-0000-000000000001', 'Arena:Slayer'),
			('camp1', '2026-09-02 10:00:00', '2026-09-02 10:00:00+00', '2026-09-02 10:10:00', 600,
			 '` + camp + `', 'Arena:Slayer')`,
		`INSERT INTO match_participants (match_id, xuid, team_id, outcome, kills, deaths, assists,
			damage_dealt, damage_taken, time_played_seconds, kda, accuracy) VALUES
			('arena1', '` + campagneXUID + `', 0, 2, 5, 1, 2, 500, 100, 600, 1.0, 0.5),
			('camp1', '` + campagneXUID + `', 0, 2, 50, 0, 9, 5000, 100, 600, 1.0, 0.9)`,
	} {
		if _, err := db.Exec(s); err != nil {
			t.Fatalf("fixture Campagne : %v\n%s", err, s)
		}
	}
}

func campagneCtx(slug string) context.Context {
	if slug == "" { // contexte SANS titre : la Campagne reste exclue (jamais le repli sur Infinite)
		return context.Background()
	}
	return ctxkeys.WithTitleSlug(context.Background(), slug)
}

// E2.5 — recalcul des sessions : une session ne s'étend pas sur un match de Campagne.
func TestCampaignExclusion_SessionMatches(t *testing.T) {
	for _, tc := range campagneTitres {
		db := testutil.NewInMemoryShared(t)
		seedCampagneArenaEtCampagne(t, db)
		rows, err := loadSessionMatchRowsDirect(campagneCtx(tc.slug), db, campagneXUID)
		if err != nil {
			t.Fatalf("%s : %v", tc.slug, err)
		}
		var got []string
		for _, r := range rows {
			got = append(got, r.MatchID)
		}
		if want := campagneAttendu(tc.masquee, []string{"arena1"}, []string{"arena1", "camp1"}); !campagneEgal(got, want) {
			t.Errorf("%s : matchs de session %v, attendu %v", tc.slug, got, want)
		}
	}
}

// E2.5 — modèle des assistances : la Campagne n'entre pas dans l'échantillon d'entraînement.
func TestCampaignExclusion_AssistsSamples(t *testing.T) {
	for _, tc := range campagneTitres {
		db := testutil.NewInMemoryShared(t)
		seedCampagneArenaEtCampagne(t, db)
		samples, err := loadAssistsSamples(campagneCtx(tc.slug), db, campagneXUID)
		if err != nil {
			t.Fatalf("%s : %v", tc.slug, err)
		}
		if want := map[bool]int{true: 1, false: 2}[tc.masquee]; len(samples) != want {
			t.Errorf("%s : %d échantillons, attendu %d", tc.slug, len(samples), want)
		}
	}
}

// E2.5 — score de performance : l'historique de référence ne porte pas la Campagne.
func TestCampaignExclusion_PerfHistory(t *testing.T) {
	for _, tc := range campagneTitres {
		db := testutil.NewInMemoryShared(t)
		seedCampagneArenaEtCampagne(t, db)
		hist, err := loadHistoryForPerf(campagneCtx(tc.slug), db, campagneXUID)
		if err != nil {
			t.Fatalf("%s : %v", tc.slug, err)
		}
		var got []string
		for _, h := range hist {
			got = append(got, h.MatchID)
		}
		if want := campagneAttendu(tc.masquee, []string{"arena1"}, []string{"arena1", "camp1"}); !campagneEgal(got, want) {
			t.Errorf("%s : historique %v, attendu %v", tc.slug, got, want)
		}
	}
}

// E2.5 — engagement : les matchs scorés (et leur historique de percentile) sans la Campagne.
func TestCampaignExclusion_EngagementMatches(t *testing.T) {
	for _, tc := range campagneTitres {
		db := testutil.NewInMemoryShared(t)
		seedCampagneArenaEtCampagne(t, db)
		if _, err := db.Exec(`CREATE TABLE highlight_events (match_id VARCHAR)`); err != nil {
			t.Fatalf("fixture highlight_events : %v", err)
		}
		rows, err := loadMatchesForEngagement(campagneCtx(tc.slug), db, campagneXUID)
		if err != nil {
			t.Fatalf("%s : %v", tc.slug, err)
		}
		var got []string
		for _, r := range rows {
			got = append(got, r.MatchID)
		}
		if want := campagneAttendu(tc.masquee, []string{"arena1"}, []string{"arena1", "camp1"}); !campagneEgal(got, want) {
			t.Errorf("%s : matchs scorés %v, attendu %v", tc.slug, got, want)
		}
	}
}

// E2.10 — le titre des lectures qui excluent la Campagne est celui des BASES (moteur, profil),
// jamais celui du ctx entrant : un ctx qui porte Infinite (requête par défaut, header) ne doit
// pas lever l'exclusion sur la base d'un joueur Halo 5.
func TestCampaignExclusion_TitreDesBasesPrimeSurLeCtx(t *testing.T) {
	entrant := ctxkeys.WithTitleSlug(context.Background(), "halo_infinite")
	if got := ctxkeys.TitleSlug((&SyncEngine{titleSlug: "halo_5"}).pinTitle(entrant)); got != "halo_5" {
		t.Errorf("SyncEngine.pinTitle : titre %q, attendu halo_5", got)
	}
	if got := ctxkeys.TitleSlug((&MatchRecomputer{}).WithTitle("halo_5").titledCtx(entrant)); got != "halo_5" {
		t.Errorf("MatchRecomputer.titledCtx : titre %q, attendu halo_5", got)
	}
	if got := ctxkeys.TitleSlug((&MatchRecomputer{}).titledCtx(entrant)); got != "halo_infinite" {
		t.Errorf("MatchRecomputer sans titre : le ctx entrant doit rester inchangé, obtenu %q", got)
	}
}
