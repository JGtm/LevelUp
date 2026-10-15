//go:build integration

// campaign_exclusion_test.go — les chargeurs LUSR (v1 et ombre v2) excluent la Campagne (E2 du
// plan .ai/PLAN_BACKLOG_2026-10-15.md, DB-27 du backlog 2026-09-26).
//
// Un match d'arène (arena1, 01/09) et un match de Campagne PLUS RÉCENT (camp1, 02/09), joués par
// le même joueur, le titre posé dans le contexte comme le fait le moteur de sync. Halo 5 : la
// Campagne disparaît de la trajectoire ; Halo Infinite (aucune variante masquée) : le résolveur
// est neutre, les deux matchs restent.

package skill

import (
	"context"
	"database/sql"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/ctxkeys"
)

func openCampagneLUSRDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { db.Close() })
	camp := analysis.CampaignExcludedVariantIDs("halo_5")[0]
	for _, s := range []string{
		`CREATE TABLE match_registry (match_id VARCHAR PRIMARY KEY, start_time TIMESTAMP,
			start_time_utc TIMESTAMPTZ, real_start_time TIMESTAMP, playlist_name VARCHAR, pair_name VARCHAR,
			is_ranked BOOLEAN DEFAULT FALSE, is_firefight BOOLEAN DEFAULT FALSE, duration_seconds INTEGER,
			game_variant_id VARCHAR)`,
		`CREATE TABLE match_participants (match_id VARCHAR, xuid VARCHAR, team_id INTEGER, outcome INTEGER,
			kills INTEGER, deaths INTEGER, assists INTEGER, kills_expected DOUBLE, deaths_expected DOUBLE,
			damage_dealt DOUBLE, damage_taken DOUBLE, accuracy DOUBLE)`,
		`INSERT INTO match_registry VALUES
			('arena1', '2026-09-01 10:00:00', '2026-09-01 10:00:00+00', NULL, 'Quick Play', 'Arena:Slayer', FALSE, FALSE, 600,
			 'aaaaaaaa-0000-0000-0000-000000000001'),
			('camp1', '2026-09-02 10:00:00', '2026-09-02 10:00:00+00', NULL, 'Campaign', 'Arena:Slayer', FALSE, FALSE, 600,
			 '` + camp + `')`,
		`INSERT INTO match_participants VALUES
			('arena1', 'xuid1', 0, 2, 5, 1, 2, 4.0, 2.0, 500.0, 100.0, 0.5),
			('camp1', 'xuid1', 0, 2, 50, 0, 9, 4.0, 2.0, 5000.0, 100.0, 0.9)`,
	} {
		if _, err := db.Exec(s); err != nil {
			t.Fatalf("fixture Campagne : %v\n%s", err, s)
		}
	}
	return db
}

var campagneTitres = []struct {
	slug    string
	masquee bool
}{{"halo_5", true}, {"halo_infinite", false}}

func campagneNombre(masquee bool) int {
	if masquee {
		return 1
	}
	return 2
}

func TestCampaignExclusion_LoadLUSRMatchData(t *testing.T) {
	for _, tc := range campagneTitres {
		ctx := ctxkeys.WithTitleSlug(context.Background(), tc.slug)
		data, err := loadLUSRMatchData(ctx, openCampagneLUSRDB(t), "xuid1")
		if err != nil {
			t.Fatalf("%s : %v", tc.slug, err)
		}
		if len(data) != campagneNombre(tc.masquee) {
			t.Errorf("%s : %d matchs LUSR, attendu %d", tc.slug, len(data), campagneNombre(tc.masquee))
		}
	}
}

func TestCampaignExclusion_LoadShadowMatches(t *testing.T) {
	for _, tc := range campagneTitres {
		ctx := ctxkeys.WithTitleSlug(context.Background(), tc.slug)
		data, err := loadShadowMatches(ctx, openCampagneLUSRDB(t), "xuid1")
		if err != nil {
			t.Fatalf("%s : %v", tc.slug, err)
		}
		if len(data) != campagneNombre(tc.masquee) {
			t.Errorf("%s : %d matchs de l'ombre v2, attendu %d", tc.slug, len(data), campagneNombre(tc.masquee))
		}
	}
}
