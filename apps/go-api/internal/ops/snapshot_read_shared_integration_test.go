//go:build integration

package ops

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/domain/title"
)

// seedFidelityShared monte une DB shared LIVE complète (tables de base + xuid_aliases +
// weapon_kills + match_csrs + les vues que le producteur exporte / que le reader recrée)
// et seede 2 matchs (m1 ready, m2 not-ready) avec gamertags croisés.
func seedFidelityShared(t *testing.T) *sql.DB {
	t.Helper()
	db := snapOpenMem(t)
	// Schéma réaliste (colonnes référencées par v_gamertag_lookup / v_killer_victim_full).
	snapExecT(t, db, `CREATE TABLE match_registry (
		match_id VARCHAR PRIMARY KEY, start_time TIMESTAMPTZ, start_time_utc TIMESTAMPTZ,
		is_ranked BOOLEAN, is_firefight BOOLEAN)`)
	snapExecT(t, db, `CREATE TABLE match_participants (match_id VARCHAR, xuid VARCHAR, gamertag VARCHAR, kills INTEGER)`)
	snapExecT(t, db, `CREATE TABLE medals_earned (match_id VARCHAR, xuid VARCHAR, medal_id BIGINT)`)
	snapExecT(t, db, `CREATE TABLE highlight_events (match_id VARCHAR, xuid VARCHAR, event_type VARCHAR)`)
	snapExecT(t, db, `CREATE TABLE killer_victim_pairs (
		match_id VARCHAR, killer_xuid VARCHAR, killer_gamertag VARCHAR,
		victim_xuid VARCHAR, victim_gamertag VARCHAR)`)
	snapExecT(t, db, `CREATE TABLE xuid_aliases (xuid VARCHAR, gamertag VARCHAR)`)
	snapExecT(t, db, `CREATE TABLE weapon_kills (match_id VARCHAR, xuid VARCHAR, weapon_id BIGINT, effective_weapon_id BIGINT, kills INTEGER)`)
	snapExecT(t, db, `CREATE TABLE match_csrs (match_id VARCHAR, xuid VARCHAR, rating_value FLOAT)`)
	// Vues que le PRODUCTEUR lit pour l'export (collapsed côté live).
	snapExecT(t, db, `CREATE VIEW v_weapon_kills AS SELECT * FROM weapon_kills`)
	snapExecT(t, db, `CREATE VIEW match_csrs_latest AS SELECT * FROM match_csrs`)

	// Données. m1 = ready ; m2 = not-ready. Le gamertag des participants/kv diffère de
	// l'alias pour exercer la priorité de v_gamertag_lookup (alias > participant > kv).
	when := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	for _, m := range []string{"m1", "m2"} {
		snapExecT(t, db, `INSERT INTO match_registry VALUES (?, ?, ?, FALSE, FALSE)`, m, when, when)
		snapExecT(t, db, `INSERT INTO match_participants VALUES (?, 'xKiller', 'TueurMP', 5)`, m)
		snapExecT(t, db, `INSERT INTO match_participants VALUES (?, 'xVictim', 'VictimeMP', 3)`, m)
		snapExecT(t, db, `INSERT INTO killer_victim_pairs VALUES (?, 'xKiller', 'TueurKV', 'xVictim', 'VictimeKV')`, m)
		snapExecT(t, db, `INSERT INTO weapon_kills VALUES (?, 'xKiller', 200, 200, 4)`, m)
		snapExecT(t, db, `INSERT INTO match_csrs VALUES (?, 'xKiller', 1500.0)`, m)
	}
	snapExecT(t, db, `INSERT INTO xuid_aliases VALUES ('xKiller', 'TueurAlias'), ('xVictim', 'VictimeAlias')`)
	// Vues dérivées canoniques côté live (pour comparer terme à terme aux résultats du
	// snapshot — c'est ça la fidélité, pas une valeur hardcodée).
	snapExecT(t, db, analysis.GamertagLookupViewSQL())
	snapExecT(t, db, `CREATE VIEW v_match_full AS SELECT mr.* FROM match_registry mr`)
	snapExecT(t, db, `CREATE VIEW v_killer_victim_full AS
		SELECT kvp.*, k.gamertag AS killer_gamertag, v.gamertag AS victim_gamertag
		FROM killer_victim_pairs kvp
		LEFT JOIN v_gamertag_lookup k ON kvp.killer_xuid = k.xuid
		LEFT JOIN v_gamertag_lookup v ON kvp.victim_xuid = v.xuid`)
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

	// 1. v_gamertag_lookup : même résolution que live (alias prioritaire).
	const gtQ = `SELECT gamertag FROM v_gamertag_lookup WHERE xuid = 'xKiller'`
	if live, snap := scanStr(shared, gtQ), scanStr(q.DB, gtQ); live != snap || snap != "TueurAlias" {
		t.Errorf("v_gamertag_lookup xKiller: live=%q snapshot=%q (attendu TueurAlias)", live, snap)
	}

	// 2. v_killer_victim_full : FIDÉLITÉ — la même requête sur le snapshot et sur live
	// retourne le même résultat (la sémantique exacte de la vue, quirks inclus).
	const kvQ = `SELECT killer_gamertag FROM v_killer_victim_full WHERE match_id = 'm1' LIMIT 1`
	if live, snap := scanStr(shared, kvQ), scanStr(q.DB, kvQ); live != snap {
		t.Errorf("v_killer_victim_full killer_gamertag (m1): live=%q snapshot=%q (divergence)", live, snap)
	}

	// 3. v_weapon_kills passthrough : kills de m1 présents, m2 (not-ready) absent.
	if n := scanInt(q.DB, `SELECT COALESCE(SUM(kills),0) FROM v_weapon_kills WHERE match_id='m1'`); n != 4 {
		t.Errorf("v_weapon_kills m1 kills = %d, attendu 4", n)
	}
	if n := scanInt(q.DB, `SELECT COUNT(*) FROM v_weapon_kills WHERE match_id='m2'`); n != 0 {
		t.Errorf("v_weapon_kills m2 (not-ready) = %d, attendu 0 (exclu)", n)
	}

	// 4. match_csrs_latest passthrough (m1 ready).
	if n := scanInt(q.DB, `SELECT COUNT(*) FROM match_csrs_latest WHERE match_id='m1'`); n != 1 {
		t.Errorf("match_csrs_latest m1 = %d, attendu 1", n)
	}

	// 5. Faits de base filtrés ready.
	if n := scanInt(q.DB, `SELECT COUNT(*) FROM match_registry`); n != 1 {
		t.Errorf("match_registry = %d, attendu 1 (m1 ready ; m2 exclu)", n)
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
