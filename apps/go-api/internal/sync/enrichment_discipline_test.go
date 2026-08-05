package sync

import (
	"context"
	"database/sql"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"
)

func discExec(t *testing.T, db *sql.DB, q string) {
	t.Helper()
	if _, err := db.Exec(q); err != nil {
		t.Fatalf("exec échec: %v\nSQL: %s", err, q)
	}
}

// dataset hétérogène : owner xJ (team0), coéquipier xM (team0), ennemi xE (team1).
func setupDisciplineShared(t *testing.T) *sql.DB {
	t.Helper()
	shared, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open shared: %v", err)
	}
	// BASCULE DU 2026-08-03 : la discipline lit `match_kill_events_latest`, un JOURNAL —
	// 1 ligne = 1 mort. L'ancienne fixture agrégeait par `kill_count` (xJ→xE valait 3 sur une
	// seule ligne) ; ici ces 3 frags sont 3 lignes, ce qui est la forme réelle de la table.
	// Une ligne de BOT (victime sans xuid) est ajoutée à dessein : elle ne doit alimenter NI
	// les suicides NI les trahisons — les deux requêtes joignent `match_participants`, où les
	// bots ne figurent pas.
	discExec(t, shared, `CREATE TABLE match_kill_events_latest (
		match_id VARCHAR, feed_killer_xuid VARCHAR, feed_killer_gamertag VARCHAR,
		victim_xuid VARCHAR, victim_gamertag VARCHAR, time_ms BIGINT)`)
	discExec(t, shared, `CREATE TABLE match_participants (match_id VARCHAR, xuid VARCHAR, team_id INTEGER)`)
	discExec(t, shared, `INSERT INTO match_participants VALUES ('m1','xJ',0),('m1','xM',0),('m1','xE',1)`)
	discExec(t, shared, `INSERT INTO match_kill_events_latest (match_id,feed_killer_xuid,victim_xuid,time_ms) VALUES
		('m1','xJ','xJ',1000),
		('m1','xJ','xM',2000),
		('m1','xJ','xE',3000),
		('m1','xJ','xE',4000),
		('m1','xJ','xE',5000),
		('m1','xE','xJ',6000),
		('m1','xE','xJ',7000),
		('m1','xM','xM',8000),
		('m1','xJ',NULL,9000)`)
	return shared
}

func setupDisciplinePlayer(t *testing.T) *sql.DB {
	t.Helper()
	player, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open player: %v", err)
	}
	discExec(t, player, `CREATE SEQUENCE IF NOT EXISTS psa_generation_seq`)
	discExec(t, player, `CREATE TABLE personal_score_awards (
		match_id VARCHAR, xuid VARCHAR, award_name VARCHAR, award_category VARCHAR,
		award_count INTEGER, award_score INTEGER, generation_id BIGINT)`)
	discExec(t, player, `CREATE VIEW personal_score_awards_latest AS
		SELECT p.* FROM personal_score_awards p
		JOIN (SELECT match_id, xuid, MAX(generation_id) AS g FROM personal_score_awards GROUP BY match_id, xuid) m
		  ON p.match_id = m.match_id AND p.xuid = m.xuid AND p.generation_id = m.g`)
	return player
}

func TestComputeAndPersistH5DisciplineAwards(t *testing.T) {
	ctx := context.Background()
	shared := setupDisciplineShared(t)
	defer shared.Close()
	player := setupDisciplinePlayer(t)
	defer player.Close()

	n, err := computeAndPersistH5DisciplineAwards(ctx, player, shared, "xJ")
	if err != nil {
		t.Fatalf("compute: %v", err)
	}
	if n != 2 {
		t.Fatalf("lignes écrites = %d, attendu 2 (1 suicide + 1 trahison)", n)
	}

	got := map[string]int{}
	rows, err := player.QueryContext(ctx, `SELECT award_name, award_count FROM personal_score_awards_latest WHERE xuid='xJ'`)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	for rows.Next() {
		var name string
		var c int
		if err := rows.Scan(&name, &c); err != nil {
			rows.Close()
			t.Fatal(err)
		}
		got[name] = c
	}
	rows.Close()
	if got["self_destruction"] != 1 {
		t.Errorf("self_destruction = %d, attendu 1 (xJ→xJ) — %v", got["self_destruction"], got)
	}
	if got["betrayed_player"] != 1 {
		t.Errorf("betrayed_player = %d, attendu 1 (xJ→xM même équipe) — %v", got["betrayed_player"], got)
	}

	// Idempotence : re-run alloue une nouvelle génération → _latest supersède (pas de doublon).
	if _, err := computeAndPersistH5DisciplineAwards(ctx, player, shared, "xJ"); err != nil {
		t.Fatalf("re-run: %v", err)
	}
	var latest int
	if err := player.QueryRowContext(ctx, `SELECT COUNT(*) FROM personal_score_awards_latest WHERE xuid='xJ'`).Scan(&latest); err != nil {
		t.Fatal(err)
	}
	if latest != 2 {
		t.Errorf("après re-run, _latest = %d, attendu 2 (supersédé, pas doublé)", latest)
	}
}

func TestComputeAndPersistH5DisciplineAwards_SkipNoSchema(t *testing.T) {
	ctx := context.Background()
	shared := setupDisciplineShared(t)
	defer shared.Close()
	player, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open player: %v", err)
	}
	defer player.Close()
	// Pas de table personal_score_awards → self-skip (0, nil).
	n, err := computeAndPersistH5DisciplineAwards(ctx, player, shared, "xJ")
	if err != nil || n != 0 {
		t.Fatalf("schéma PSA absent : attendu (0, nil), obtenu (%d, %v)", n, err)
	}
}
