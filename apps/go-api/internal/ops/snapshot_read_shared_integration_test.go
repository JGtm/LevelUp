//go:build integration

package ops

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"levelup/go-api/internal/domain/title"
)

// seedFidelityShared monte une DB shared LIVE au schéma RÉALISTE (toutes les colonnes
// référencées par mv_player_matches + les vues) et seede 2 matchs (m1 ready, m2 not).
// Hand-rollé (et non via le runner de migrations multi-titre) pour rester en package ops
// sans cycle d'import sync. OpenSnapshotShared recrée ensuite TOUTES les vues shared.
func seedFidelityShared(t *testing.T) *sql.DB {
	t.Helper()
	db := snapOpenMem(t)
	// match_registry : toutes les colonnes lues par mv_player_matches.
	snapExecT(t, db, `CREATE TABLE match_registry (
		match_id VARCHAR PRIMARY KEY, start_time TIMESTAMPTZ, end_time TIMESTAMPTZ,
		start_time_utc TIMESTAMPTZ, end_time_utc TIMESTAMPTZ,
		playlist_id VARCHAR, playlist_name VARCHAR, playlist_name_fr VARCHAR,
		map_id VARCHAR, map_name VARCHAR, map_name_fr VARCHAR,
		pair_name VARCHAR, pair_name_fr VARCHAR, pair_id VARCHAR,
		game_variant_id VARCHAR, game_variant_name VARCHAR, mode_category VARCHAR,
		is_ranked BOOLEAN, is_firefight BOOLEAN,
		duration_seconds INTEGER, playable_duration_seconds INTEGER,
		team_0_score INTEGER, team_1_score INTEGER, team_0_ps_score INTEGER, team_1_ps_score INTEGER,
		player_count INTEGER)`)
	// match_participants : toutes les colonnes lues par mv_player_matches + v_gamertag_lookup.
	snapExecT(t, db, `CREATE TABLE match_participants (
		match_id VARCHAR, xuid VARCHAR, gamertag VARCHAR, team_id INTEGER, outcome INTEGER,
		rank INTEGER, score INTEGER, kills INTEGER, deaths INTEGER, assists INTEGER,
		kda DOUBLE, accuracy DOUBLE, shots_fired INTEGER, shots_hit INTEGER,
		damage_dealt DOUBLE, damage_taken DOUBLE, personal_score INTEGER,
		time_played_seconds INTEGER, avg_life_seconds DOUBLE, headshot_kills INTEGER,
		max_killing_spree INTEGER, grenade_kills INTEGER, melee_kills INTEGER,
		power_weapon_kills INTEGER, team_mmr DOUBLE, enemy_mmr DOUBLE,
		kills_expected DOUBLE, deaths_expected DOUBLE, backfill_bits BIGINT)`)
	snapExecT(t, db, `CREATE TABLE medals_earned (match_id VARCHAR, xuid VARCHAR, medal_id BIGINT)`)
	snapExecT(t, db, `CREATE TABLE highlight_events (match_id VARCHAR, xuid VARCHAR, event_type VARCHAR)`)
	snapExecT(t, db, `CREATE TABLE killer_victim_pairs (
		match_id VARCHAR, killer_xuid VARCHAR, killer_gamertag VARCHAR,
		victim_xuid VARCHAR, victim_gamertag VARCHAR)`)
	snapExecT(t, db, `CREATE TABLE xuid_aliases (xuid VARCHAR, gamertag VARCHAR)`)
	// weapon_kills RAW (colonnes lues par v_weapon_kills : reconciled_as, generation_id).
	snapExecT(t, db, `CREATE TABLE weapon_kills (match_id VARCHAR, xuid VARCHAR, weapon_id BIGINT, reconciled_as BIGINT, generation_id BIGINT)`)
	// match_csrs RAW (colonnes lues par match_csrs_latest : written_at, id).
	snapExecT(t, db, `CREATE TABLE match_csrs (match_id VARCHAR, xuid VARCHAR, rating_value FLOAT, written_at TIMESTAMP, id BIGINT)`)
	// match_objective_stats RAW (colonnes lues par match_objective_stats_latest :
	// written_at, id) — jointe par Q12 (scoreboard MatchView) depuis v7.2.
	snapExecT(t, db, `CREATE TABLE match_objective_stats (id BIGINT, match_id VARCHAR, xuid VARCHAR, flag_captures INTEGER, written_at TIMESTAMP)`)

	// Données. m1 = ready ; m2 = not-ready. Gamertag participants/kv ≠ alias → exerce
	// la priorité de v_gamertag_lookup (alias > participant > kv).
	when := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	for _, m := range []string{"m1", "m2"} {
		snapExecT(t, db, `INSERT INTO match_registry (match_id, start_time, start_time_utc, is_ranked, is_firefight) VALUES (?, ?, ?, FALSE, FALSE)`, m, when, when)
		snapExecT(t, db, `INSERT INTO match_participants (match_id, xuid, gamertag, team_id, kills, deaths) VALUES (?, 'xKiller', 'TueurMP', 0, 5, 2)`, m)
		snapExecT(t, db, `INSERT INTO match_participants (match_id, xuid, gamertag, team_id, kills, deaths) VALUES (?, 'xVictim', 'VictimeMP', 1, 3, 5)`, m)
		snapExecT(t, db, `INSERT INTO killer_victim_pairs (match_id, killer_xuid, killer_gamertag, victim_xuid, victim_gamertag) VALUES (?, 'xKiller', 'TueurKV', 'xVictim', 'VictimeKV')`, m)
		snapExecT(t, db, `INSERT INTO weapon_kills (match_id, xuid, weapon_id, reconciled_as, generation_id) VALUES (?, 'xKiller', 200, NULL, 0)`, m)
		snapExecT(t, db, `INSERT INTO match_csrs (match_id, xuid, rating_value, written_at, id) VALUES (?, 'xKiller', 1500.0, ?, 1)`, m, when)
		// 2 générations objective stats : la vue _latest doit ne servir que la 2e.
		snapExecT(t, db, `INSERT INTO match_objective_stats (id, match_id, xuid, flag_captures, written_at) VALUES (1, ?, 'xKiller', 1, ?)`, m, when)
		snapExecT(t, db, `INSERT INTO match_objective_stats (id, match_id, xuid, flag_captures, written_at) VALUES (2, ?, 'xKiller', 3, ?)`, m, when.Add(time.Minute))
	}
	snapExecT(t, db, `INSERT INTO xuid_aliases (xuid, gamertag) VALUES ('xKiller', 'TueurAlias'), ('xVictim', 'VictimeAlias')`)
	return db
}

// TestOpenSnapshotShared_Fidelity_integration : le schéma shared reconstruit depuis le
// snapshot retourne les MÊMES résultats que la DB live (gamertag, weapon_kills,
// killer_victim_full), filtrés au set ready (m2 not-ready exclu).
func TestOpenSnapshotShared_Fidelity_integration(t *testing.T) {
	ctx := context.Background()
	paths := title.NewPathResolver(t.TempDir(), nil)
	slug := title.DefaultSlug

	shared := seedFidelityShared(t)
	player := seedPlayerDB(t, []string{"m1"}, []string{"m2"}) // m1 ready, m2 not

	res, err := ProduceSnapshot(ctx, SnapshotOptions{
		TitleSlug:    slug,
		Paths:        paths,
		Shared:       fakeSharedOpener{db: shared},
		PlayerOpener: fakePlayerOpener{byGT: map[string]*sql.DB{"Solo": player}},
		Players:      []string{"Solo"},
		Now:          time.Date(2026, 6, 25, 9, 0, 0, 0, time.UTC),
	})
	if err != nil || !res.Produced {
		t.Fatalf("produce: res=%+v err=%v", res, err)
	}

	q, err := OpenSnapshotShared(ctx, paths, slug)
	if err != nil {
		t.Fatalf("OpenSnapshotShared: %v", err)
	}
	defer q.Close()

	scanStr := func(db *sql.DB, query string, args ...any) string {
		t.Helper()
		var s sql.NullString
		if err := db.QueryRowContext(ctx, query, args...).Scan(&s); err != nil {
			t.Fatalf("query %q: %v", query, err)
		}
		return s.String
	}
	scanInt := func(db *sql.DB, query string, args ...any) int {
		t.Helper()
		var n int
		if err := db.QueryRowContext(ctx, query, args...).Scan(&n); err != nil {
			t.Fatalf("query %q: %v", query, err)
		}
		return n
	}

	// 1. v_gamertag_lookup : résolution correcte (alias prioritaire sur participant/kv).
	if snap := scanStr(q.DB, `SELECT gamertag FROM v_gamertag_lookup WHERE xuid = 'xKiller'`); snap != "TueurAlias" {
		t.Errorf("v_gamertag_lookup xKiller = %q, attendu TueurAlias (priorité alias)", snap)
	}

	// 2. v_killer_victim_full : m1 ready présent, m2 not-ready absent.
	if n := scanInt(q.DB, `SELECT COUNT(*) FROM v_killer_victim_full WHERE match_id='m1'`); n != 1 {
		t.Errorf("v_killer_victim_full m1 = %d, attendu 1", n)
	}
	if n := scanInt(q.DB, `SELECT COUNT(*) FROM v_killer_victim_full WHERE match_id='m2'`); n != 0 {
		t.Errorf("v_killer_victim_full m2 (not-ready) = %d, attendu 0", n)
	}

	// 3. v_weapon_kills (DENSE_RANK reconstruit) : m1 présent (effective_weapon_id résolu),
	// m2 (not-ready) absent.
	if n := scanInt(q.DB, `SELECT COUNT(*) FROM v_weapon_kills WHERE match_id='m1' AND effective_weapon_id = 200`); n != 1 {
		t.Errorf("v_weapon_kills m1 = %d, attendu 1 (effective_weapon_id=200)", n)
	}
	if n := scanInt(q.DB, `SELECT COUNT(*) FROM v_weapon_kills WHERE match_id='m2'`); n != 0 {
		t.Errorf("v_weapon_kills m2 (not-ready) = %d, attendu 0 (exclu)", n)
	}

	// 4. match_csrs_latest passthrough (m1 ready).
	if n := scanInt(q.DB, `SELECT COUNT(*) FROM match_csrs_latest WHERE match_id='m1'`); n != 1 {
		t.Errorf("match_csrs_latest m1 = %d, attendu 1", n)
	}

	// 4bis. match_objective_stats_latest reconstruite (régression v7.2 : Q12 la LEFT
	// JOIN — son absence rendait TOUT match du cut scoreboard_empty). QUALIFY : seule
	// la dernière génération (flag_captures=3) est servie ; m2 not-ready exclu.
	if n := scanInt(q.DB, `SELECT COUNT(*) FROM match_objective_stats_latest WHERE match_id='m1'`); n != 1 {
		t.Errorf("match_objective_stats_latest m1 = %d, attendu 1 (dédup QUALIFY)", n)
	}
	if v := scanInt(q.DB, `SELECT flag_captures FROM match_objective_stats_latest WHERE match_id='m1' AND xuid='xKiller'`); v != 3 {
		t.Errorf("match_objective_stats_latest m1 flag_captures = %d, attendu 3 (dernière version)", v)
	}
	if n := scanInt(q.DB, `SELECT COUNT(*) FROM match_objective_stats_latest WHERE match_id='m2'`); n != 0 {
		t.Errorf("match_objective_stats_latest m2 (not-ready) = %d, attendu 0", n)
	}

	// 5. Faits de base filtrés ready.
	if n := scanInt(q.DB, `SELECT COUNT(*) FROM match_registry`); n != 1 {
		t.Errorf("match_registry = %d, attendu 1 (m1 ready ; m2 exclu)", n)
	}

	// 6. mv_player_matches reconstruit (registry × participants) : 2 lignes pour m1
	// (2 participants), gamertag résolu, m2 exclu.
	if n := scanInt(q.DB, `SELECT COUNT(*) FROM mv_player_matches WHERE match_id='m1'`); n != 2 {
		t.Errorf("mv_player_matches m1 = %d, attendu 2 (2 participants)", n)
	}
	if n := scanInt(q.DB, `SELECT COUNT(*) FROM mv_player_matches WHERE match_id='m2'`); n != 0 {
		t.Errorf("mv_player_matches m2 (not-ready) = %d, attendu 0", n)
	}
}

// TestOpenSnapshotShared_Incomplete_integration : si une relation requise manque du
// snapshot (ici xuid_aliases jamais seedée), OpenSnapshotShared retourne
// ErrSnapshotIncomplete (le caller dégrade vers live).
func TestOpenSnapshotShared_Incomplete_integration(t *testing.T) {
	ctx := context.Background()
	paths := title.NewPathResolver(t.TempDir(), nil)
	slug := title.DefaultSlug

	// Shared SANS xuid_aliases ni weapon_kills/match_csrs (schéma minimal = 5 tables).
	shared := snapOpenMem(t)
	seedSharedSchema(t, shared)
	seedSharedMatch(t, shared, "m1", time.Now())
	player := seedPlayerDB(t, []string{"m1"}, nil)

	res, err := ProduceSnapshot(ctx, SnapshotOptions{
		TitleSlug: slug, Paths: paths,
		Shared:       fakeSharedOpener{db: shared},
		PlayerOpener: fakePlayerOpener{byGT: map[string]*sql.DB{"Solo": player}},
		Players:      []string{"Solo"},
	})
	if err != nil || !res.Produced {
		t.Fatalf("produce: res=%+v err=%v", res, err)
	}

	if _, err := OpenSnapshotShared(ctx, paths, slug); !errors.Is(err, ErrSnapshotIncomplete) {
		t.Fatalf("err = %v, attendu ErrSnapshotIncomplete (xuid_aliases absente)", err)
	}
}

// TestOpenSnapshotShared_SchemaContract_integration : CONTRAT du schéma snapshot
// shared. Toutes les relations (tables + vues) que les lectures shared de l'app
// référencent — en particulier les requêtes MatchView Q12/Q13 (match_participants,
// v_gamertag_lookup, v_weapon_kills, match_csrs_latest, match_objective_stats_latest,
// medals_earned, highlight_events, v_killer_victim_full...) — doivent exister dans le
// schéma :memory: reconstruit. Une table nouvellement jointe par une requête MatchView
// (précédent : Q12 → match_objective_stats_latest en v7.2) doit être ajoutée à
// l'export (sharedSnapshotMatchKeyedRaw & co) ET à la lecture (OpenSnapshotShared),
// sinon CE TEST casse — au lieu d'un Catalog Error silencieux en prod
// (scoreboard_empty sur tous les matchs du cut).
func TestOpenSnapshotShared_SchemaContract_integration(t *testing.T) {
	ctx := context.Background()
	paths := title.NewPathResolver(t.TempDir(), nil)
	slug := title.DefaultSlug

	shared := seedFidelityShared(t)
	player := seedPlayerDB(t, []string{"m1"}, nil)

	res, err := ProduceSnapshot(ctx, SnapshotOptions{
		TitleSlug:    slug,
		Paths:        paths,
		Shared:       fakeSharedOpener{db: shared},
		PlayerOpener: fakePlayerOpener{byGT: map[string]*sql.DB{"Solo": player}},
		Players:      []string{"Solo"},
		Now:          time.Date(2026, 6, 25, 9, 0, 0, 0, time.UTC),
	})
	if err != nil || !res.Produced {
		t.Fatalf("produce: res=%+v err=%v", res, err)
	}

	q, err := OpenSnapshotShared(ctx, paths, slug)
	if err != nil {
		t.Fatalf("OpenSnapshotShared: %v", err)
	}
	defer q.Close()

	expected := []string{
		// Tables de base matérialisées (sharedSnapshotTables + MatchKeyedRaw + Global).
		"match_registry", "match_participants", "medals_earned", "highlight_events",
		"killer_victim_pairs", "weapon_kills", "match_csrs", "match_objective_stats",
		"xuid_aliases",
		// Vues reconstruites aux noms live.
		"v_gamertag_lookup", "v_match_full", "v_killer_victim_full", "v_weapon_kills",
		"match_csrs_latest", "match_objective_stats_latest", "mv_player_matches",
	}
	for _, rel := range expected {
		var n int
		if err := q.DB.QueryRowContext(ctx,
			`SELECT COUNT(*) FROM information_schema.tables WHERE table_name = ?`, rel).Scan(&n); err != nil {
			t.Fatalf("information_schema %s: %v", rel, err)
		}
		if n == 0 {
			t.Errorf("relation %q absente du schéma snapshot reconstruit (contrat lectures shared/MatchView)", rel)
		}
	}
}
