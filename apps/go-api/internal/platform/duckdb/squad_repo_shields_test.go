// Package duckdb — squad_repo_shields_test.go : Q32f contre une base DuckDB en mémoire au schéma
// livré (migrations de la base partagée), lecture par la vue `_latest`.
package duckdb

import (
	"context"
	"database/sql"
	"fmt"
	"testing"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/migration"
)

// poserPasseBouclier écrit une passe du résumé d'usage d'un match : la ligne film (écrite `decale`
// plus tard que maintenant, pour ordonner les passes) puis une ligne joueur par entrée de bris
// (nil = NULL, non mesuré). L'entier d'une entrée est son nombre de bris, le second ses recharges.
func poserPasseBouclier(t *testing.T, db *sql.DB, match, pass, decale string, joueurs map[string][2]any) {
	t.Helper()
	if _, err := db.Exec(`INSERT INTO match_usage_films
		(match_id, summary_pass, summary_rev, artifact_schema, written_at, frame_interval_ms, frame_count, duration_ms,
		 pad_occupancies, pad_named, pad_unnamed, weapon_pads_json, powerup_pickups_json)
		VALUES (?, ?, 'us7', 70, CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP) + CAST(? AS INTERVAL), 100, 10, 1000, 0, 0, 0, '[]', '{}')`,
		match, pass, decale); err != nil {
		t.Fatalf("film %s/%s: %v", match, pass, err)
	}
	for xuid, v := range joueurs {
		if _, err := db.Exec(`INSERT INTO match_usage_players
			(match_id, summary_pass, summary_rev, xuid, grapple_pulls, camo_episodes, camo_ms, camo_kills,
			 overshield_episodes, overshield_ms, overshield_kills, deployed_json, dropped_objects, grenades_thrown,
			 pad_pickups, pad_pickups_json, shield_breaks, shield_recharges)
			VALUES (?, ?, 'us7', ?, 0, 0, 0, 0, 0, 0, 0, '{}', 0, 0, 0, '{}', ?, ?)`,
			match, pass, xuid, v[0], v[1]); err != nil {
			t.Fatalf("joueur %s/%s/%s: %v", match, pass, xuid, err)
		}
	}
}

func lireBoucliers(t *testing.T, db *sql.DB, matchIDs []string) []domain.MatchShieldRow {
	t.Helper()
	rows, err := db.QueryContext(context.Background(),
		fmt.Sprintf(Q32fMatchShieldsTemplate, placeholderList(len(matchIDs))), appendStrArgs(nil, matchIDs)...)
	if err != nil {
		t.Fatalf("Q32f: %v", err)
	}
	defer rows.Close()
	got, err := scanMatchShields(rows)
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	return got
}

// TestQ32fMatchShields : la passe courante seule est lue ; une ligne sans aucune valeur (NULL des
// deux colonnes) ne rend pas son match mesuré ; une colonne NULL à côté de l'autre vaut zéro ; un
// match sans résumé est absent.
func TestQ32fMatchShields(t *testing.T) {
	db, err := sql.Open("duckdb", "")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	FillMigratedMemory(t, db, migration.TargetShared)

	// m1 : une passe périmée (valeurs fausses), puis la passe courante.
	poserPasseBouclier(t, db, "m1", "m1-old", "-1 hour", map[string][2]any{"a": {99, 99}, "b": {99, 99}})
	poserPasseBouclier(t, db, "m1", "m1-new", "0 seconds", map[string][2]any{
		"a": {7, 2}, "b": {0, 0}, "c": {nil, nil}, "d": {4, nil},
	})
	// m2 : passe d'avant la mesure, toutes les valeurs NULL : non mesuré.
	poserPasseBouclier(t, db, "m2", "m2-p1", "0 seconds", map[string][2]any{"a": {nil, nil}, "b": {nil, nil}})
	// m3 : aucun résumé.

	got := lireBoucliers(t, db, []string{"m1", "m2", "m3"})
	want := []domain.MatchShieldRow{
		{MatchID: "m1", XUID: "a", Breaks: 7, Recharges: 2},
		{MatchID: "m1", XUID: "b"},
		{MatchID: "m1", XUID: "d", Breaks: 4},
	}
	if fmt.Sprint(got) != fmt.Sprint(want) {
		t.Errorf("Q32f = %+v, attendu %+v", got, want)
	}
}
