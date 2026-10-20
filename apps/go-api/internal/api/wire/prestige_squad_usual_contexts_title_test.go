//go:build integration

package wire

// prestige_squad_usual_contexts_title_test.go — l'indice « playlists habituelles » d'une
// escouade exclut la Campagne du titre du JOUEUR RÉSOLU, sur le vrai chemin HTTP (revue
// adversariale des lots B, constat R2-2 ; lot B-C8 du backlog 2026-09-26).
//
// Le web appelle GET /squads SANS title_slug (apps/web/src/lib/prestige.ts) : le handler
// transmettait donc "" à SquadUsualContexts, et l'exclusion de la Campagne (D-5, lot B4)
// ne s'appliquait pas. Le test de B4 passait "halo_5" en dur au fournisseur. Ici, la
// requête ne porte aucun titre : seul le PlayerDB résolu le connaît.

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/api/handlers"
	"levelup/go-api/internal/platform/duckdb"
	prestigedb "levelup/go-api/internal/platform/duckdb/prestige"
	"levelup/go-api/internal/prestige"
)

const (
	bc8Joueur = "Joueur"
	bc8XUIDA  = "2533274800000101"
	bc8XUIDB  = "2533274800000102"
)

// bc8Env : bases migrées, titre du joueur posé sur son PlayerDB, un match d'arène et un
// match de Campagne joués par tout le roster, une escouade dont Joueur est membre.
func bc8Env(t *testing.T, titleSlug string) *duckdb.PlayerDB {
	t.Helper()
	pdb := setupProgressionEnv(t).pdb
	pdb.TitleSlug = titleSlug
	ctx := context.Background()
	camp := analysis.CampaignExcludedVariantIDs("halo_5")[0]
	for _, m := range []struct {
		id, variant, playlistID, playlist string
		at                                time.Time
	}{
		{"arena1", "aaaaaaaa-0000-0000-0000-000000000001", "pl-arene", "Arene", time.Date(2026, 9, 1, 10, 0, 0, 0, time.UTC)},
		{"camp1", camp, "pl-campagne", "Campagne", time.Date(2026, 9, 2, 10, 0, 0, 0, time.UTC)},
	} {
		if _, err := pdb.Shared.Exec(ctx, `INSERT INTO match_registry
			(match_id, start_time, start_time_utc, game_variant_id, playlist_id, playlist_name)
			VALUES (?, ?, ?, ?, ?, ?)`, m.id, m.at, m.at, m.variant, m.playlistID, m.playlist); err != nil {
			t.Fatalf("insert match_registry %s : %v", m.id, err)
		}
		for _, xuid := range []string{bc8XUIDA, bc8XUIDB} {
			if _, err := pdb.Shared.Exec(ctx, `INSERT INTO match_participants (
					match_id, xuid, gamertag, team_id, outcome, kills, deaths, assists,
					kda, accuracy, personal_score, time_played_seconds, headshot_kills, damage_dealt, damage_taken
				) VALUES (?, ?, ?, 1, 2, 5, 1, 0, 1.0, 0.5, 1000, 600, 1, 500, 100)`,
				m.id, xuid, "gt-"+xuid); err != nil {
				t.Fatalf("insert match_participants %s/%s : %v", m.id, xuid, err)
			}
		}
	}
	squads := prestigedb.NewPrestigeSquadRepo(pdb.SharedSocial)
	now := time.Now().UTC().Truncate(time.Second)
	if err := squads.Create(ctx, prestige.Squad{ID: "sq-bc8", Name: "Alpha", CreatedBy: bc8Joueur, CreatedAt: now}); err != nil {
		t.Fatalf("création escouade : %v", err)
	}
	for _, m := range []prestige.SquadMember{
		{SquadID: "sq-bc8", Xuid: bc8XUIDA, UserID: bc8Joueur, JoinedAt: now},
		{SquadID: "sq-bc8", Xuid: bc8XUIDB, JoinedAt: now},
	} {
		if err := squads.AddMember(ctx, m); err != nil {
			t.Fatalf("ajout membre %s : %v", m.Xuid, err)
		}
	}
	return pdb
}

// playlistsHabituelles appelle GET /squads?user_id=Joueur — SANS title_slug, comme le web —
// par le handler réel, le service Prestige paresseux et le bundle.
func playlistsHabituelles(t *testing.T, pdb *duckdb.PlayerDB) []string {
	t.Helper()
	bundle := &PrestigeBundle{
		enabled:    true,
		socialRepo: prestigedb.NewPrestigeSocialRepo(pdb.SharedSocial),
		squadRepo:  prestigedb.NewPrestigeSquadRepo(pdb.SharedSocial),
		resolve:    func(context.Context, string) (*duckdb.PlayerDB, error) { return pdb, nil },
	}
	lazy := NewLazyPrestigeService(bundle, func(context.Context) string { return bc8Joueur })
	r := chi.NewRouter()
	handlers.NewPrestigeHandler(lazy, nil).Mount(r)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/squads?user_id="+bc8Joueur, nil))
	if w.Code != http.StatusOK {
		t.Fatalf("GET /squads : statut %d — corps %s", w.Code, w.Body.String())
	}
	var body struct {
		Squads []struct {
			UsualPlaylists []string `json:"usual_playlists"`
		} `json:"squads"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil || len(body.Squads) != 1 {
		t.Fatalf("réponse illisible (err %v) : %s", err, w.Body.String())
	}
	return body.Squads[0].UsualPlaylists
}

func TestSquadUsualContexts_TitreDuJoueurResolu(t *testing.T) {
	for _, tc := range []struct {
		slug string
		want int // playlists habituelles attendues : la Campagne ne compte pas en Halo 5
	}{{"halo_5", 1}, {"halo_infinite", 2}} {
		got := playlistsHabituelles(t, bc8Env(t, tc.slug))
		if len(got) != tc.want {
			t.Errorf("%s : playlists habituelles %v, attendu %d (titre du PlayerDB résolu, "+
				"requête sans title_slug)", tc.slug, got, tc.want)
		}
	}
}
