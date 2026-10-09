//go:build integration

// replay_oracle_repo_test.go — l'oracle du banc de verite : les colonnes officielles que la
// cuisson ne lit pas, et les stats d'objectif lues sur la vue `_latest`.
//
// Lancer avec : go test -tags=integration ./internal/platform/duckdb/ -run TestReplayOracle

package duckdb

import (
	"context"
	"database/sql"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/migration"
)

// seedReplayOracle cree le schema minimal : les colonnes que `participants` lit (INSERT NOMMES,
// meme lecon que seedReplayFacts), et la table append-only des stats d'objectif avec sa vue
// canonique (`migration.MatchObjectiveStatsLatestViewSQL`, jamais un DDL inline).
func seedReplayOracle(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open :memory: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	ddl := `
		CREATE TABLE match_participants (
			match_id VARCHAR, xuid VARCHAR, personal_score INTEGER, score INTEGER,
			shots_fired INTEGER, shots_hit INTEGER, headshot_kills SMALLINT, melee_kills SMALLINT,
			grenade_kills SMALLINT, power_weapon_kills SMALLINT, time_played_seconds INTEGER,
			present_at_beginning BOOLEAN, present_at_completion BOOLEAN);
		CREATE TABLE match_objective_stats (
			id BIGINT, match_id VARCHAR, xuid VARCHAR, written_at TIMESTAMP,
			flag_captures INTEGER, flag_grabs INTEGER, time_as_flag_carrier_seconds DOUBLE,
			zone_captures INTEGER);
		` + migration.MatchObjectiveStatsLatestViewSQL("match_objective_stats") + `;`
	if _, err := db.ExecContext(context.Background(), ddl); err != nil {
		t.Fatalf("ddl: %v", err)
	}
	return db
}

// TestReplayOracleForMatch — les colonnes officielles (NULL -> nil), les stats d'objectif de la
// DERNIERE ecriture (vue _latest), et les colonnes NULL absentes de la carte.
func TestReplayOracleForMatch(t *testing.T) {
	db := seedReplayOracle(t)
	ctx := context.Background()
	if _, err := db.ExecContext(ctx, `
		INSERT INTO match_participants (match_id, xuid, personal_score, score, shots_fired, shots_hit,
			headshot_kills, melee_kills, grenade_kills, power_weapon_kills, time_played_seconds,
			present_at_beginning, present_at_completion) VALUES
			('m1', '111', 1200, 1250, 300, 120, 4, 1, 2, 0, 600, TRUE, FALSE),
			('m1', '222', NULL, 10, NULL, NULL, 0, 0, 0, 0, NULL, NULL, TRUE),
			('m2', '111', 1, 1, 1, 1, 1, 1, 1, 1, 1, TRUE, TRUE);
		INSERT INTO match_objective_stats (id, match_id, xuid, written_at, flag_captures, flag_grabs,
			time_as_flag_carrier_seconds, zone_captures) VALUES
			(1, 'm1', '111', TIMESTAMP '2026-09-01 10:00:00', 1, 2, 10.5, NULL),
			(2, 'm1', '111', TIMESTAMP '2026-09-02 10:00:00', 3, 4, 20.25, NULL);`); err != nil {
		t.Fatalf("seed: %v", err)
	}
	got, err := NewReplayOracleRepo(db).OracleForMatch(ctx, "m1")
	if err != nil {
		t.Fatalf("OracleForMatch: %v", err)
	}
	if got.MatchID != "m1" || len(got.Players) != 2 {
		t.Fatalf("oracle = %+v, veut m1 et 2 joueurs", got)
	}
	p := got.Players[0]
	if p.XUID != "111" || p.PersonalScore == nil || *p.PersonalScore != 1200 || *p.Score != 1250 ||
		*p.ShotsFired != 300 || *p.ShotsHit != 120 || *p.HeadshotKills != 4 || *p.TimePlayedSeconds != 600 ||
		!*p.PresentAtBeginning || *p.PresentAtCompletion {
		t.Errorf("ligne 111 = %+v", p)
	}
	if p.Objectives["flag_captures"] != 3 || p.Objectives["flag_grabs"] != 4 ||
		p.Objectives["time_as_flag_carrier_seconds"] != 20.25 {
		t.Errorf("stats d'objectif = %v, veut la DERNIERE ecriture (3, 4, 20.25)", p.Objectives)
	}
	if _, ok := p.Objectives["zone_captures"]; ok {
		t.Errorf("une colonne NULL ne doit pas figurer dans la carte : %v", p.Objectives)
	}
	for _, c := range []string{"id", "match_id", "xuid", "written_at"} {
		if _, ok := p.Objectives[c]; ok {
			t.Errorf("colonne d'identite %q prise pour une stat : %v", c, p.Objectives)
		}
	}
	q := got.Players[1]
	if q.PersonalScore != nil || q.ShotsFired != nil || q.PresentAtBeginning != nil || q.Objectives != nil {
		t.Errorf("NULL doit rester nil, et un joueur sans stats sans carte : %+v", q)
	}
}

// TestReplayOracleUnknownMatchIsEmpty — un match inconnu rend un oracle vide, sans erreur.
func TestReplayOracleUnknownMatchIsEmpty(t *testing.T) {
	db := seedReplayOracle(t)
	got, err := NewReplayOracleRepo(db).OracleForMatch(context.Background(), "absent")
	if err != nil || !got.Empty() {
		t.Fatalf("oracle = %+v, err = %v ; veut vide sans erreur", got, err)
	}
}
