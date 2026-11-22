// gameplay_seconds_test.go — durée de jeu d'un match (:memory:) : la durée du registre moins le
// coup d'envoi, par les fragments uniques sqlfrag.SQLT0Ms et sqlfrag.SQLGameplayDurationMs.
package duckdb

import (
	"context"
	"database/sql"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/platform/duckdb/sqlfrag"
)

// newGameplayTestDB crée une DB :memory: avec un match_registry minimal et cinq matchs :
// m1 (coup d'envoi à 26,304 s), m2 (coup d'envoi inconnu), m3 (durée absente), m4 (coup d'envoi
// au-delà de la durée), m5 (coup d'envoi lu sur start_time quand start_time_utc manque).
func newGameplayTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open :memory:: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	for _, q := range []string{
		`CREATE TABLE match_registry (
			match_id VARCHAR PRIMARY KEY, start_time TIMESTAMP, start_time_utc TIMESTAMPTZ,
			real_start_time TIMESTAMP, duration_seconds INTEGER)`,
		`INSERT INTO match_registry VALUES
			('m1', '2026-11-18 12:00:00', '2026-11-18 12:00:00+00', '2026-11-18 12:00:26.304', 600),
			('m2', '2026-11-18 12:00:00', '2026-11-18 12:00:00+00', NULL, 480),
			('m3', '2026-11-18 12:00:00', '2026-11-18 12:00:00+00', '2026-11-18 12:00:20', NULL),
			('m4', '2026-11-18 12:00:00', '2026-11-18 12:00:00+00', '2026-11-18 12:00:28', 20),
			('m5', '2026-11-18 12:00:00', NULL, '2026-11-18 12:00:10', 300)`,
	} {
		if _, err := db.Exec(q); err != nil {
			t.Fatalf("%s: %v", q, err)
		}
	}
	return db
}

func TestLoadGameplaySecondsByMatch_DureeMoinsCoupDEnvoi(t *testing.T) {
	db := newGameplayTestDB(t)
	got := LoadGameplaySecondsByMatch(context.Background(), db, []string{"m1", "m2", "m3", "m4", "m5", "absent"})
	want := map[string]int{"m1": 573, "m2": 480, "m5": 290}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v (m3 sans durée, m4 sans durée de jeu et le match absent n'ont aucune valeur)", got, want)
	}
	for id, w := range want {
		if got[id] != w {
			t.Errorf("%s = %d, want %d", id, got[id], w)
		}
	}
}

// Les deux fragments rendent la même grandeur que MatchTimeline : le coup d'envoi en ms (NULL =
// inconnu) et la durée de jeu en ms (inconnu : la durée du match ; durée absente : 0).
func TestFragmentsCoupDEnvoiEtDureeDeJeu(t *testing.T) {
	db := newGameplayTestDB(t)
	rows, err := db.Query(`SELECT match_id, ` + sqlfrag.SQLT0Ms("r") + `, ` + sqlfrag.SQLGameplayDurationMs("r") +
		` FROM match_registry r ORDER BY match_id`)
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	defer rows.Close()
	type want struct {
		t0 sql.NullInt64
		gp int64
	}
	expected := map[string]want{
		"m1": {sql.NullInt64{Int64: 26304, Valid: true}, 573_696},
		"m2": {sql.NullInt64{}, 480_000},
		"m3": {sql.NullInt64{Int64: 20_000, Valid: true}, 0},
		"m4": {sql.NullInt64{Int64: 28_000, Valid: true}, 0},
		"m5": {sql.NullInt64{Int64: 10_000, Valid: true}, 290_000},
	}
	for rows.Next() {
		var id string
		var t0 sql.NullInt64
		var gp int64
		if err := rows.Scan(&id, &t0, &gp); err != nil {
			t.Fatalf("scan: %v", err)
		}
		if w := expected[id]; t0 != w.t0 || gp != w.gp {
			t.Errorf("%s : coup d'envoi %v, durée de jeu %d ; attendu %v et %d", id, t0, gp, w.t0, w.gp)
		}
	}
}

func TestLoadGameplaySecondsByMatch_DegradesWhenTableMissing(t *testing.T) {
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if got := LoadGameplaySecondsByMatch(context.Background(), db, []string{"m1"}); len(got) != 0 {
		t.Errorf("table absente : got %v, want une map vide", got)
	}
}

func TestLoadGameplaySecondsByMatch_EmptyInput(t *testing.T) {
	if got := LoadGameplaySecondsByMatch(context.Background(), nil, []string{"m1"}); len(got) != 0 {
		t.Errorf("db nil : got %v", got)
	}
	db := newGameplayTestDB(t)
	if got := LoadGameplaySecondsByMatch(context.Background(), db, nil); len(got) != 0 {
		t.Errorf("aucun id : got %v", got)
	}
}
