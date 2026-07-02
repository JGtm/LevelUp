// Package duckdb — patterns_repo_db_test.go : test bout-en-bout de PatternsRepo
// sur des DuckDB :memory: (player + shared). Couvre les 3 loaders SQL cross-DB
// (loadShared / loadEnrichments / loadSkillRanks) + l'orchestration LoadRows
// (merge + deltas), que les tests unitaires purs ne touchaient pas.
package duckdb

import (
	"context"
	"database/sql"
	"testing"

	"levelup/go-api/internal/migration"
)

// newPatternsTestPDB construit un PlayerDB minimal câblé sur deux DuckDB
// :memory : une "shared" (match_registry + match_participants) et une "player"
// (player_match_enrichment + match_skill_rank). SharedReader nil → SharedReadDB()
// retombe sur LegacySharedReader(pdb.Shared).
func newPatternsTestPDB(t *testing.T) *PlayerDB {
	t.Helper()
	ctx := context.Background()

	sharedSQL, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open shared: %v", err)
	}
	t.Cleanup(func() { _ = sharedSQL.Close() })

	playerSQL, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open player: %v", err)
	}
	t.Cleanup(func() { _ = playerSQL.Close() })

	// Schéma fidèle au vrai match_registry prod (cf. migration/steps_shared.go) :
	// start_time + start_time_utc (TIMESTAMPTZ), pair_name/pair_name_fr (mode),
	// duration_seconds. Le repo lit played_at via COALESCE(start_time_utc,
	// start_time AT TIME ZONE 'UTC') et le mode via COALESCE(pair_name_fr, pair_name).
	mustExec(t, sharedSQL, `CREATE TABLE match_registry (
		match_id VARCHAR, start_time TIMESTAMP, start_time_utc TIMESTAMPTZ,
		pair_name VARCHAR, pair_name_fr VARCHAR,
		map_id VARCHAR, duration_seconds INTEGER)`)
	mustExec(t, sharedSQL, `CREATE TABLE match_participants (
		match_id VARCHAR, xuid VARCHAR, outcome INTEGER, kills INTEGER,
		deaths INTEGER, assists INTEGER, accuracy DOUBLE, damage_dealt DOUBLE,
		damage_taken DOUBLE, headshot_kills INTEGER, team_mmr DOUBLE)`)
	// m1 = plus récent, non ranked (team_mmr NULL) ; m2 = plus ancien, ranked.
	mustExec(t, sharedSQL, `INSERT INTO match_registry VALUES
		('m1', TIMESTAMP '2026-01-02 12:00:00', TIMESTAMPTZ '2026-01-02 12:00:00+00', 'Slayer', NULL, 'map1', 600),
		('m2', TIMESTAMP '2026-01-01 12:00:00', TIMESTAMPTZ '2026-01-01 12:00:00+00', 'Oddball', NULL, 'map2', 500)`)
	mustExec(t, sharedSQL, `INSERT INTO match_participants VALUES
		('m1', 'p1', 2, 10, 5, 4, 0.55, 2000, 1500, 3, NULL),
		('m2', 'p1', 3, 8, 4, 2, 0.40, 1200, 1300, 1, 1500)`)

	mustExec(t, playerSQL, `CREATE TABLE player_match_enrichment (
		match_id VARCHAR, performance_score DOUBLE, session_id VARCHAR,
		is_with_friends BOOLEAN, engagement_score DOUBLE, engagement_score_brut DOUBLE)`)
	// Append-only #23046 : convertit player_match_enrichment (id PK + stage +
	// written_at) et crée la vue player_match_enrichment_latest (lue par le repo).
	if err := migration.EnsurePlayerMatchEnrichmentAppendOnly(playerSQL); err != nil {
		t.Fatalf("EnsurePlayerMatchEnrichmentAppendOnly: %v", err)
	}
	mustExec(t, playerSQL, `INSERT INTO player_match_enrichment
		(match_id, performance_score, session_id, is_with_friends, engagement_score, engagement_score_brut)
		VALUES ('m1', 0.8, 's1', TRUE, 0.6, 0.1)`)

	mustExec(t, playerSQL, `CREATE TABLE match_skill_rank (
		match_id VARCHAR, rating_value DOUBLE, rating_type VARCHAR)`)
	mustExec(t, playerSQL, `INSERT INTO match_skill_rank VALUES
		('m1', 1550, 'LUSR'),
		('m2', 1500, 'LUSR')`)
	// Lot B (ADR 0026) : le repo lit désormais la vue _latest. Le seed n'a qu'une
	// version par match → pass-through suffit (pas de colonnes id/written_at ici).
	mustExec(t, playerSQL, `CREATE VIEW match_skill_rank_latest AS SELECT * FROM match_skill_rank`)
	_ = ctx

	return &PlayerDB{
		Player: newTestDB(playerSQL, ":memory:"),
		Shared: newTestDB(sharedSQL, ":memory:"),
		XUID:   "p1",
	}
}

func mustExec(t *testing.T, db *sql.DB, q string) {
	t.Helper()
	if _, err := db.ExecContext(context.Background(), q); err != nil {
		t.Fatalf("exec %q: %v", q, err)
	}
}

// TestPatternsRepo_LoadRows_EndToEnd vérifie le chemin complet : 3 loaders SQL +
// merge + deltas, contre de vraies DuckDB :memory.
func TestPatternsRepo_LoadRows_EndToEnd(t *testing.T) {
	repo := NewPatternsRepo(newPatternsTestPDB(t))

	rows, err := repo.LoadRows(context.Background(), 50)
	if err != nil {
		t.Fatalf("LoadRows: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("len(rows) = %d, want 2", len(rows))
	}

	// Ordre DESC (played_at) restauré en sortie : m1 (récent) puis m2 (ancien).
	if rows[0].MatchID != "m1" || rows[1].MatchID != "m2" {
		t.Fatalf("ordre = [%s, %s], want [m1, m2]", rows[0].MatchID, rows[1].MatchID)
	}

	m1, m2 := rows[0], rows[1]

	// loadShared : KDA = (10 + 4/2)/5 = 2.4 ; team_mmr NULL → non ranked.
	if m1.KDA != 2.4 {
		t.Errorf("m1.KDA = %v, want 2.4", m1.KDA)
	}
	if m1.IsRanked {
		t.Errorf("m1.IsRanked = true, want false (team_mmr NULL)")
	}
	if !m2.IsRanked {
		t.Errorf("m2.IsRanked = false, want true (team_mmr non NULL)")
	}
	if m1.Mode != "Slayer" || m1.MapID != "map1" {
		t.Errorf("m1 mode/map = %q/%q, want Slayer/map1", m1.Mode, m1.MapID)
	}

	// loadEnrichments : m1 a un enrichissement, m2 non.
	if m1.SessionID != "s1" || !m1.IsWithFriends || m1.PerfScore == nil || *m1.PerfScore != 0.8 {
		t.Errorf("m1 enrichissements mal hydratés: %+v", m1)
	}
	if m2.SessionID != "" || m2.PerfScore != nil {
		t.Errorf("m2 ne devrait pas avoir d'enrichissement: %+v", m2)
	}

	// loadSkillRanks + deltas : m1 (récent) = 1550-1500 = 50 ; m2 (le plus ancien) = nil.
	if m1.DeltaLUSR == nil || *m1.DeltaLUSR != 50 {
		t.Errorf("m1.DeltaLUSR = %v, want 50", m1.DeltaLUSR)
	}
	if m2.DeltaLUSR != nil {
		t.Errorf("m2.DeltaLUSR = %v, want nil (match le plus ancien)", m2.DeltaLUSR)
	}
}

// TestPatternsRepo_LoadRows_EmptyPlayer : joueur sans match → nil, nil (pas d'erreur).
func TestPatternsRepo_LoadRows_EmptyPlayer(t *testing.T) {
	pdb := newPatternsTestPDB(t)
	pdb.XUID = "ghost" // aucun match pour ce xuid
	repo := NewPatternsRepo(pdb)

	rows, err := repo.LoadRows(context.Background(), 50)
	if err != nil {
		t.Fatalf("LoadRows: %v", err)
	}
	if rows != nil {
		t.Errorf("rows = %v, want nil (aucun match)", rows)
	}
}
