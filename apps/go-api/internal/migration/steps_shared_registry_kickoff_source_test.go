//go:build cgo

package migration

import (
	"database/sql"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"
)

// Une base existante (vue `v_match_full` créée avant la colonne, coups d'envoi déjà reportés depuis un
// rejeu) migrée : la colonne naît `film` là où `t0_quality = 'film_movement'`, NULL ailleurs, et la vue
// l'expose. Une seconde application ne change rien.
func TestRealStartSource_AmorceEtVue(t *testing.T) {
	db, err := sql.Open("duckdb", "")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	db.SetMaxOpenConns(1)
	for _, q := range []string{
		`CREATE TABLE match_registry (match_id VARCHAR PRIMARY KEY, start_time TIMESTAMP NOT NULL,
			real_start_time TIMESTAMP, t0_quality VARCHAR)`,
		`CREATE INDEX idx_mr_start_time ON match_registry(start_time)`,
		`INSERT INTO match_registry VALUES
			('film', TIMESTAMP '2026-11-18 12:00:00', TIMESTAMP '2026-11-18 12:00:26', 'film_movement'),
			('api',  TIMESTAMP '2026-11-18 12:00:00', TIMESTAMP '2026-11-18 12:00:20', 'ok'),
			('neg',  TIMESTAMP '2026-11-18 12:00:00', NULL, 'negative')`,
		VMatchFullViewSQL,
	} {
		if _, err := db.Exec(q); err != nil {
			t.Fatalf("%s: %v", q, err)
		}
	}
	for i := 0; i < 2; i++ {
		if err := applyRealStartSource(db); err != nil {
			t.Fatalf("application %d: %v", i+1, err)
		}
	}
	got := map[string]string{}
	rows, err := db.Query(`SELECT match_id, COALESCE(real_start_source, '') FROM v_match_full ORDER BY match_id`)
	if err != nil {
		t.Fatalf("la vue n'expose pas la colonne : %v", err)
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var id, src string
		if err := rows.Scan(&id, &src); err != nil {
			t.Fatalf("scan: %v", err)
		}
		got[id] = src
	}
	if got["film"] != "film" || got["api"] != "" || got["neg"] != "" {
		t.Fatalf("sources = %v, attendu film pour le seul film_movement", got)
	}
}

// Une base sans `t0_quality` (titre qui ne l'a jamais reçue) se migre sans amorce.
func TestRealStartSource_SansQualiteRienAAmorcer(t *testing.T) {
	db, err := sql.Open("duckdb", "")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	db.SetMaxOpenConns(1)
	if _, err := db.Exec(`CREATE TABLE match_registry (match_id VARCHAR PRIMARY KEY, start_time TIMESTAMP NOT NULL)`); err != nil {
		t.Fatalf("schema: %v", err)
	}
	if err := applyRealStartSource(db); err != nil {
		t.Fatalf("application: %v", err)
	}
	if ok, err := columnExists(db, "match_registry", "real_start_source"); err != nil || !ok {
		t.Fatalf("colonne absente (err %v)", err)
	}
}
