//go:build integration

// Test de la migration shared_match_usage_levelup_v1 : les trois colonnes de mesures et la
// révision du calcul s'ajoutent à une base qui porte déjà les tables du résumé d'usage, et les
// deux vues `_latest` (qui figent leur SELECT * à la création) les exposent.
package migration

import (
	"database/sql"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"
)

// baseResumeUsageAvantLevelUp : une base à l'état d'avant la migration (tables et vues du résumé
// d'usage, colonne shieldless_kills comprise, aucune colonne de LevelUp).
func baseResumeUsageAvantLevelUp(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open duckdb: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	for _, fn := range []func(*sql.DB) error{
		applyMatchUsageSummary, applyMatchUsageOutcomes, applyMatchUsageShieldless,
	} {
		if err := fn(db); err != nil {
			t.Fatalf("etat anterieur: %v", err)
		}
	}
	return db
}

func TestApplyMatchUsageLevelUp_ColonnesVuesEtIdempotence(t *testing.T) {
	db := baseResumeUsageAvantLevelUp(t)
	// Une ligne ecrite AVANT la migration : ses mesures doivent se lire NULL, pas zero.
	if _, err := db.Exec(`INSERT INTO match_usage_films
		(match_id, summary_pass, summary_rev, artifact_schema, frame_interval_ms, frame_count, duration_ms,
		 pad_occupancies, pad_named, pad_unnamed, weapon_pads_json, powerup_pickups_json)
		VALUES ('m1', 'p1', 'us7', 70, 100, 10, 1000, 0, 0, 0, '[]', '{}')`); err != nil {
		t.Fatalf("film ancien: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO match_usage_players
		(match_id, summary_pass, summary_rev, xuid, grapple_pulls, camo_episodes, camo_ms, camo_kills,
		 overshield_episodes, overshield_ms, overshield_kills, deployed_json, dropped_objects, grenades_thrown,
		 pad_pickups, pad_pickups_json)
		VALUES ('m1', 'p1', 'us7', '111', 0, 0, 0, 0, 0, 0, 0, '{}', 0, 0, 0, '{}')`); err != nil {
		t.Fatalf("joueur ancien: %v", err)
	}

	for i := 0; i < 2; i++ { // la seconde application est un no-op
		if err := applyMatchUsageLevelUp(db); err != nil {
			t.Fatalf("applyMatchUsageLevelUp (passage %d): %v", i+1, err)
		}
	}

	var breaks, recharges, revenge sql.NullInt64
	if err := db.QueryRow(`SELECT shield_breaks, shield_recharges, revenge_kills
		FROM match_usage_players_latest WHERE match_id = 'm1' AND xuid = '111'`).
		Scan(&breaks, &recharges, &revenge); err != nil {
		t.Fatalf("la vue players_latest n'expose pas les colonnes de mesures: %v", err)
	}
	if breaks.Valid || recharges.Valid || revenge.Valid {
		t.Errorf("une ligne d'avant la migration doit lire NULL, lu %+v %+v %+v", breaks, recharges, revenge)
	}
	var rev sql.NullString
	if err := db.QueryRow(`SELECT levelup_rev FROM match_usage_films_latest WHERE match_id = 'm1'`).Scan(&rev); err != nil {
		t.Fatalf("la vue films_latest n'expose pas levelup_rev: %v", err)
	}
	if rev.Valid {
		t.Errorf("levelup_rev d'une passe ancienne = %q, attendu NULL", rev.String)
	}
}

func TestMatchUsageLevelUp_Enregistree(t *testing.T) {
	for _, m := range All() {
		if m.Name == "shared_match_usage_levelup_v1" {
			if m.TargetDB != TargetShared {
				t.Fatalf("cible = %v, attendu TargetShared", m.TargetDB)
			}
			return
		}
	}
	t.Fatal("migration shared_match_usage_levelup_v1 non enregistree")
}
