//go:build integration

package duckdb

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"levelup/go-api/internal/domain"
)

// seedStatLeaderboardData crée des tables minimales (match_participants,
// v_gamertag_lookup, match_registry) et insère un dataset déterministe :
//   - xa "Alpha" : 12 matchs, 10 kills/match (haut du classement)
//   - xb "Bravo" : 10 matchs, 5 kills/match
//   - xc "Charlie" : 5 matchs seulement → exclu par le seuil min (10)
//   - bid(bot)   : 12 matchs → exclu par le filtre NOT LIKE 'bid(%'
//
// Les matchs de Alpha sont en playlist "Ranked Arena", ceux de Bravo en
// "Ranked Slayer" (pour tester le filtre playlist ILIKE).
func seedStatLeaderboardData(t *testing.T, db *DB) {
	t.Helper()
	ctx := context.Background()
	ddl := []string{
		`CREATE TABLE match_participants (
			match_id VARCHAR, xuid VARCHAR, kills INTEGER, deaths INTEGER,
			assists INTEGER, damage_dealt DOUBLE, shots_hit INTEGER, shots_fired INTEGER
		)`,
		`CREATE TABLE v_gamertag_lookup (xuid VARCHAR, gamertag VARCHAR)`,
		`CREATE TABLE match_registry (match_id VARCHAR, playlist_name VARCHAR, season_id VARCHAR)`,
	}
	for _, q := range ddl {
		if _, err := db.Exec(ctx, q); err != nil {
			t.Fatalf("DDL %q: %v", q, err)
		}
	}

	type player struct {
		xuid, gt, playlist string
		matches, killsEach int
	}
	players := []player{
		{"xa", "Alpha", "Ranked Arena", 12, 10},
		{"xb", "Bravo", "Ranked Slayer", 10, 5},
		{"xc", "Charlie", "Ranked Arena", 5, 99}, // sous le seuil min matchs
		{"bid(007)", "BotSeven", "Ranked Arena", 12, 50},
	}
	for _, p := range players {
		if p.gt != "" && !strings.HasPrefix(p.xuid, "bid(") {
			if _, err := db.Exec(ctx, `INSERT INTO v_gamertag_lookup VALUES (?, ?)`, p.xuid, p.gt); err != nil {
				t.Fatalf("insert lookup: %v", err)
			}
		}
		for i := 0; i < p.matches; i++ {
			mid := fmt.Sprintf("%s-m%d", p.xuid, i)
			if _, err := db.Exec(ctx,
				`INSERT INTO match_participants VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
				mid, p.xuid, p.killsEach, 5, 2, 1000.0, 50, 100,
			); err != nil {
				t.Fatalf("insert participant: %v", err)
			}
			if _, err := db.Exec(ctx, `INSERT INTO match_registry VALUES (?, ?, ?)`, mid, p.playlist, "CsrSeason13"); err != nil {
				t.Fatalf("insert registry: %v", err)
			}
		}
	}
}

// TestGetStatLeaderboard_AggregationFiltersAndLocal valide l'agrégation des stats
// communautaires : tri DESC, seuil min de matchs, exclusion des bots, mise en
// évidence du joueur local, et filtre playlist optionnel.
func TestGetStatLeaderboard_AggregationFiltersAndLocal(t *testing.T) {
	shared := openMemDB(t)
	seedStatLeaderboardData(t, shared)
	repo := NewLeaderboardRepo(&PlayerDB{Shared: shared, XUID: "xa"})
	ctx := context.Background()

	// Kills, sans filtre playlist : Alpha (120) puis Bravo (50). Charlie exclu
	// (5 matchs < 10), bot exclu.
	res, err := repo.GetStatLeaderboard(ctx, domain.LeaderboardKills, "", "", 50)
	if err != nil {
		t.Fatalf("GetStatLeaderboard: %v", err)
	}
	if len(res) != 2 {
		t.Fatalf("len = %d, attendu 2 (Alpha, Bravo) ; got %+v", len(res), res)
	}
	if res[0].Gamertag != "Alpha" || res[0].Rank != 1 || res[0].Value != 120 {
		t.Errorf("rang 1 = %+v, attendu Alpha/1/120", res[0])
	}
	if res[1].Gamertag != "Bravo" || res[1].Rank != 2 || res[1].Value != 50 {
		t.Errorf("rang 2 = %+v, attendu Bravo/2/50", res[1])
	}
	if !res[0].IsLocal {
		t.Error("Alpha (xuid local xa) devrait avoir IsLocal=true")
	}
	if res[1].IsLocal {
		t.Error("Bravo ne devrait pas être local")
	}
	for _, e := range res {
		if e.Gamertag == "Charlie" || e.Gamertag == "BotSeven" {
			t.Errorf("%s ne devrait pas figurer (seuil/bot)", e.Gamertag)
		}
	}

	// Filtre playlist "Arena" : seul Alpha a des matchs Ranked Arena (>= seuil).
	arena, err := repo.GetStatLeaderboard(ctx, domain.LeaderboardKills, "Arena", "", 50)
	if err != nil {
		t.Fatalf("GetStatLeaderboard(Arena): %v", err)
	}
	if len(arena) != 1 || arena[0].Gamertag != "Alpha" {
		t.Fatalf("filtre Arena = %+v, attendu [Alpha]", arena)
	}

	// Filtre saison "CsrSeason13" (toutes les données seedées) : Alpha + Bravo.
	s13, err := repo.GetStatLeaderboard(ctx, domain.LeaderboardKills, "", "CsrSeason13", 50)
	if err != nil {
		t.Fatalf("GetStatLeaderboard(CsrSeason13): %v", err)
	}
	if len(s13) != 2 {
		t.Errorf("filtre saison CsrSeason13 = %d entrées, attendu 2", len(s13))
	}

	// Filtre saison inexistante → aucun résultat.
	none, err := repo.GetStatLeaderboard(ctx, domain.LeaderboardKills, "", "CsrSeason99", 50)
	if err != nil {
		t.Fatalf("GetStatLeaderboard(CsrSeason99): %v", err)
	}
	if len(none) != 0 {
		t.Errorf("filtre saison inexistante = %d entrées, attendu 0", len(none))
	}

	// Catégorie inconnue → erreur claire.
	if _, err := repo.GetStatLeaderboard(ctx, domain.LeaderboardCategory("inexistante"), "", "", 10); err == nil {
		t.Error("catégorie inconnue devrait retourner une erreur")
	}
}
