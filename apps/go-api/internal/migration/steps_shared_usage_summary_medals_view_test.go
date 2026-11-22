//go:build integration

package migration

// steps_shared_usage_summary_medals_view_test.go — la vue v_medals_earned : la table des médailles
// fusionnée avec les médailles que LevelUp dérive du film (Vengeur, Cible difficile), sur une base
// partagée avec résumé d'usage, sans colonnes dérivées et sans table de médailles.

import (
	"database/sql"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"
)

// baseMedailles : une base avec la table des médailles, et le résumé d'usage complet avec la vue si avecResume.
func baseMedailles(t *testing.T, avecResume bool) *sql.DB {
	t.Helper()
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open duckdb: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if _, err := db.Exec(`CREATE TABLE medals_earned (match_id VARCHAR, xuid VARCHAR, medal_name_id BIGINT,
		count SMALLINT, created_at TIMESTAMP, PRIMARY KEY (match_id, xuid, medal_name_id))`); err != nil {
		t.Fatalf("medals_earned: %v", err)
	}
	if avecResume {
		for _, fn := range []func(*sql.DB) error{
			applyMatchUsageSummary, applyMatchUsageOutcomes, applyMatchUsageShieldless, applyMatchUsageLevelUp,
		} {
			if err := fn(db); err != nil {
				t.Fatalf("resume d'usage: %v", err)
			}
		}
		if err := ApplyMedalsEarnedView(db); err != nil {
			t.Fatalf("vue: %v", err)
		}
	}
	return db
}

// ligneMedaille rend le compte de la médaille d'un joueur sur un match, lu par la vue, et si une
// ligne existe.
func ligneMedaille(t *testing.T, db *sql.DB, match, xuid string, medal int64) (int, bool) {
	t.Helper()
	var n sql.NullInt64
	err := db.QueryRow(`SELECT SUM(count) FROM v_medals_earned WHERE match_id = ? AND xuid = ? AND medal_name_id = ?`,
		match, xuid, medal).Scan(&n)
	if err != nil {
		t.Fatalf("lecture %s/%s/%d: %v", match, xuid, medal, err)
	}
	return int(n.Int64), n.Valid
}

// poserFilm écrit la ligne film d'une passe de résumé (une seule par match et par passe).
func poserFilm(t *testing.T, db *sql.DB, match, pass string, decale string) {
	t.Helper()
	if _, err := db.Exec(`INSERT INTO match_usage_films
		(match_id, summary_pass, summary_rev, artifact_schema, written_at, frame_interval_ms, frame_count, duration_ms,
		 pad_occupancies, pad_named, pad_unnamed, weapon_pads_json, powerup_pickups_json)
		VALUES (?, ?, 'us7', 70, CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP) + CAST(? AS INTERVAL), 100, 10, 1000, 0, 0, 0, '[]', '{}')`,
		match, pass, decale); err != nil {
		t.Fatalf("film: %v", err)
	}
}

// poserJoueur écrit la ligne joueur d'une passe de résumé avec ses mesures (nil = NULL).
func poserJoueur(t *testing.T, db *sql.DB, match, pass, xuid string, revenge, recharges any) {
	t.Helper()
	if _, err := db.Exec(`INSERT INTO match_usage_players
		(match_id, summary_pass, summary_rev, xuid, grapple_pulls, camo_episodes, camo_ms, camo_kills,
		 overshield_episodes, overshield_ms, overshield_kills, deployed_json, dropped_objects, grenades_thrown,
		 pad_pickups, pad_pickups_json, revenge_kills, shield_recharges)
		VALUES (?, ?, 'us7', ?, 0, 0, 0, 0, 0, 0, 0, '{}', 0, 0, 0, '{}', ?, ?)`,
		match, pass, xuid, revenge, recharges); err != nil {
		t.Fatalf("joueur: %v", err)
	}
}

func TestMedalsEarnedView_Fusion(t *testing.T) {
	db := baseMedailles(t, true)
	// m-herite : Vengeur hérité seul, résumé écrit sans mesure -> l'hérité reste.
	// m-derive : Vengeur hérité 8 (A) et 5 (B), dérivé 2 pour A et 0 pour B -> dérivé seul.
	for _, q := range []string{
		`INSERT INTO medals_earned VALUES ('m-herite', 'A', 9000000001, 4, now()), ('m-herite', 'A', 111, 3, now())`,
		`INSERT INTO medals_earned VALUES ('m-derive', 'A', 9000000001, 8, now()), ('m-derive', 'B', 9000000001, 5, now()),
			('m-derive', 'A', 222, 1, now())`,
	} {
		if _, err := db.Exec(q); err != nil {
			t.Fatalf("seed: %v", err)
		}
	}
	poserFilm(t, db, "m-herite", "p1", "0 seconds")
	poserJoueur(t, db, "m-herite", "p1", "A", nil, nil)
	poserFilm(t, db, "m-derive", "p2", "0 seconds")
	poserJoueur(t, db, "m-derive", "p2", "A", 2, 3)
	poserJoueur(t, db, "m-derive", "p2", "B", 0, 0)

	if n, ok := ligneMedaille(t, db, "m-herite", "A", 9000000001); !ok || n != 4 {
		t.Errorf("Vengeur hérité sans valeur dérivée = %d (présent %v), attendu 4", n, ok)
	}
	if n, ok := ligneMedaille(t, db, "m-herite", "A", 111); !ok || n != 3 {
		t.Errorf("médaille ordinaire = %d (présent %v), attendu 3", n, ok)
	}
	if n, ok := ligneMedaille(t, db, "m-derive", "A", 9000000001); !ok || n != 2 {
		t.Errorf("Vengeur dérivé de A = %d (présent %v), attendu 2 (l'hérité 8 est remplacé)", n, ok)
	}
	if _, ok := ligneMedaille(t, db, "m-derive", "B", 9000000001); ok {
		t.Errorf("Vengeur de B : dérivé à zéro, l'hérité 5 ne doit plus apparaître")
	}
	if n, ok := ligneMedaille(t, db, "m-derive", "A", 9000000002); !ok || n != 3 {
		t.Errorf("Cible difficile de A = %d (présent %v), attendu 3", n, ok)
	}
	if _, ok := ligneMedaille(t, db, "m-derive", "B", 9000000002); ok {
		t.Errorf("Cible difficile de B : zéro recharge, aucune ligne")
	}
	if n, ok := ligneMedaille(t, db, "m-derive", "A", 222); !ok || n != 1 {
		t.Errorf("médaille ordinaire d'un match dérivé = %d (présent %v), attendu 1", n, ok)
	}
}

// TestMedalsEarnedView_SuitLaPasseCourante : une passe plus récente supplante la précédente, y
// compris quand elle ne porte aucune mesure (la vue ne mélange jamais deux passes).
func TestMedalsEarnedView_SuitLaPasseCourante(t *testing.T) {
	db := baseMedailles(t, true)
	poserFilm(t, db, "m1", "p1", "0 seconds")
	poserJoueur(t, db, "m1", "p1", "A", 1, 1)
	if n, _ := ligneMedaille(t, db, "m1", "A", 9000000001); n != 1 {
		t.Fatalf("passe 1 : Vengeur = %d, attendu 1", n)
	}
	poserFilm(t, db, "m1", "p2", "1 hour")
	poserJoueur(t, db, "m1", "p2", "A", nil, nil)
	if _, ok := ligneMedaille(t, db, "m1", "A", 9000000001); ok {
		t.Errorf("la vue sert la passe précédente : Vengeur encore présent")
	}
}

func TestMedalsEarnedView_SansColonnesDeriveesRendLaTable(t *testing.T) {
	for _, avecResume := range []bool{false, true} {
		db := baseMedailles(t, false)
		if avecResume {
			// Tables du résumé d'usage d'avant la migration des mesures : aucune colonne dérivée.
			for _, fn := range []func(*sql.DB) error{applyMatchUsageSummary, applyMatchUsageOutcomes, applyMatchUsageShieldless} {
				if err := fn(db); err != nil {
					t.Fatalf("resume: %v", err)
				}
			}
		}
		if _, err := db.Exec(`INSERT INTO medals_earned VALUES ('m', 'A', 9000000001, 4, now())`); err != nil {
			t.Fatalf("seed: %v", err)
		}
		if err := ApplyMedalsEarnedView(db); err != nil {
			t.Fatalf("ApplyMedalsEarnedView: %v", err)
		}
		if n, ok := ligneMedaille(t, db, "m", "A", 9000000001); !ok || n != 4 {
			t.Errorf("avec résumé=%v : Vengeur = %d (présent %v), attendu 4 (table seule)", avecResume, n, ok)
		}
	}
}

func TestMedalsEarnedView_SansTableDeMedaillesNeFaitRien(t *testing.T) {
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open duckdb: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := ApplyMedalsEarnedView(db); err != nil {
		t.Fatalf("sans medals_earned : %v", err)
	}
}

func TestMedalsEarnedView_Idempotente(t *testing.T) {
	db := baseMedailles(t, true)
	for i := 0; i < 2; i++ {
		if err := ApplyMedalsEarnedView(db); err != nil {
			t.Fatalf("passage %d: %v", i+1, err)
		}
	}
}
