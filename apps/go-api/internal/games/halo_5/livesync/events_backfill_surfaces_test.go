package livesync

import (
	"context"
	"database/sql"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/games/canonical"
	"levelup/go-api/internal/persist"
)

// openEventsBackfillDB : DuckDB in-memory avec les tables minimales touchées par le
// backfill (formes alignées sur le shared h5 : colonnes utilisées seulement).
func openEventsBackfillDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	db.SetMaxOpenConns(1)
	for _, s := range []string{
		`CREATE TABLE weapon_accuracy (match_id VARCHAR, xuid VARCHAR, weapon_id UBIGINT, shots_fired INTEGER, shots_landed INTEGER, drops INTEGER)`,
		`CREATE TABLE highlight_events (match_id VARCHAR, event_type VARCHAR, time_ms INTEGER, xuid VARCHAR, type_hint VARCHAR)`,
		`CREATE TABLE match_registry (match_id VARCHAR, start_time_utc TIMESTAMP, start_time TIMESTAMP)`,
		`CREATE TABLE xuid_aliases (gamertag VARCHAR, xuid VARCHAR)`,
	} {
		if _, err := db.Exec(s); err != nil {
			t.Fatalf("schema: %v\n%s", err, s)
		}
	}
	return db
}

func ebExec(t *testing.T, db *sql.DB, q string) {
	t.Helper()
	if _, err := db.Exec(q); err != nil {
		t.Fatalf("exec: %v\n%s", err, q)
	}
}

func ebCount(t *testing.T, db *sql.DB, q string) int {
	t.Helper()
	var n int
	if err := db.QueryRow(q).Scan(&n); err != nil {
		t.Fatalf("count: %v\n%s", err, q)
	}
	return n
}

func ebStr(s string) *string { return &s }

func TestWriteEventDerivedSurfaces_IdempotentPreservesMedals(t *testing.T) {
	db := openEventsBackfillDB(t)
	defer db.Close()
	ctx := context.Background()

	// Pré-existant (collect initial) : médaille + kill — NE DOIVENT PAS être touchés.
	ebExec(t, db, `INSERT INTO highlight_events VALUES ('m1','medal',1000,'xJ','{}')`)
	ebExec(t, db, `INSERT INTO highlight_events VALUES ('m1','kill',1200,'xJ',NULL)`)

	wacc := []persist.WeaponAccuracyInsert{
		{MatchID: "m1", XUID: "xJ", WeaponID: 100, ShotsFired: 15, ShotsLanded: 7, Drops: 2},
	}
	hl := []persist.HighlightEventInsert{
		{MatchID: "m1", EventType: "assist", TimeMS: 1300, XUID: ebStr("xM")},
		{MatchID: "m1", EventType: "mode", TimeMS: 1400, XUID: ebStr("xJ")},
	}

	w, e, err := WriteEventDerivedSurfaces(ctx, db, "m1", wacc, hl)
	if err != nil {
		t.Fatalf("write: %v", err)
	}
	if w != 1 || e != 2 {
		t.Fatalf("retours = %d/%d, attendu 1/2", w, e)
	}
	ebAssertSurfaces(t, db)

	// Idempotence : ré-écrire ne duplique rien (DELETE ciblé puis ré-INSERT).
	if _, _, err := WriteEventDerivedSurfaces(ctx, db, "m1", wacc, hl); err != nil {
		t.Fatalf("rewrite: %v", err)
	}
	ebAssertSurfaces(t, db)
}

func ebAssertSurfaces(t *testing.T, db *sql.DB) {
	t.Helper()
	if n := ebCount(t, db, `SELECT COUNT(*) FROM weapon_accuracy WHERE match_id='m1'`); n != 1 {
		t.Errorf("weapon_accuracy = %d, attendu 1", n)
	}
	if n := ebCount(t, db, `SELECT COUNT(*) FROM highlight_events WHERE match_id='m1' AND event_type IN ('assist','mode')`); n != 2 {
		t.Errorf("assist+mode = %d, attendu 2", n)
	}
	if n := ebCount(t, db, `SELECT COUNT(*) FROM highlight_events WHERE match_id='m1' AND event_type='medal'`); n != 1 {
		t.Errorf("médaille préservée = %d, attendu 1", n)
	}
	if n := ebCount(t, db, `SELECT COUNT(*) FROM highlight_events WHERE match_id='m1' AND event_type='kill'`); n != 1 {
		t.Errorf("kill préservé = %d, attendu 1", n)
	}
}

func TestMatchesMissingWeaponAccuracy(t *testing.T) {
	db := openEventsBackfillDB(t)
	defer db.Close()
	ctx := context.Background()
	ebExec(t, db, `INSERT INTO match_registry (match_id, start_time_utc) VALUES
		('old1', TIMESTAMP '2024-01-01'), ('old2', TIMESTAMP '2024-02-01'), ('done1', TIMESTAMP '2024-03-01')`)
	ebExec(t, db, `INSERT INTO weapon_accuracy VALUES ('done1','xJ',100,10,5,1)`) // déjà traité → exclu

	ids, err := matchesMissingWeaponAccuracy(ctx, db, 0)
	if err != nil {
		t.Fatalf("enum: %v", err)
	}
	if len(ids) != 2 {
		t.Fatalf("ids = %v, attendu 2 (done1 exclu)", ids)
	}
	if ids[0] != "old2" || ids[1] != "old1" { // récents d'abord
		t.Errorf("ordre = %v, attendu [old2 old1]", ids)
	}
}

func TestRunEventsBackfill_EndToEnd(t *testing.T) {
	db := openEventsBackfillDB(t)
	defer db.Close()
	ctx := context.Background()
	ebExec(t, db, `INSERT INTO match_registry (match_id, start_time_utc) VALUES ('m1', TIMESTAMP '2024-01-01'), ('m2', TIMESTAMP '2024-02-01')`)
	ebExec(t, db, `INSERT INTO weapon_accuracy VALUES ('m2','xJ',100,1,1,1)`) // m2 déjà fait → skip
	ebExec(t, db, `INSERT INTO xuid_aliases VALUES ('JGtm','xJ')`)

	sf, sl := 12, 6
	fetch := func(_ context.Context, _ string) ([]canonical.MatchEvent, error) {
		return []canonical.MatchEvent{{
			Type:        canonical.MatchEventWeaponDrop,
			TimeMs:      1000,
			Player:      &canonical.PlayerIdentity{Gamertag: "JGtm"},
			Weapon:      &canonical.AssetReference{Kind: "weapon", ID: "100"},
			ShotsFired:  &sf,
			ShotsLanded: &sl,
		}}, nil
	}
	stats, err := RunEventsBackfill(ctx, db, fetch, 0, nil)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if stats.Matches != 1 {
		t.Errorf("Matches = %d, attendu 1 (m2 exclu car déjà weapon_accuracy)", stats.Matches)
	}
	if stats.Updated != 1 || stats.WeaponRows != 1 {
		t.Errorf("stats = %+v, attendu maj=1 weapon_rows=1", stats)
	}
	// m1 a maintenant une ligne weapon_accuracy avec le xuid RÉSOLU depuis xuid_aliases.
	if n := ebCount(t, db, `SELECT COUNT(*) FROM weapon_accuracy WHERE match_id='m1' AND xuid='xJ'`); n != 1 {
		t.Errorf("m1 weapon_accuracy(xJ) = %d, attendu 1", n)
	}
}
