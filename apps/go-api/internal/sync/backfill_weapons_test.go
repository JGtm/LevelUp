//go:build integration

package sync

import (
	"database/sql"
	"testing"

	"levelup/go-api/internal/analysis"

	_ "github.com/duckdb/duckdb-go/v2"
)

func openWeaponDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { db.Close() })

	ddl := `
		CREATE TABLE highlight_events (
			match_id VARCHAR, xuid VARCHAR,
			event_type VARCHAR, time_ms INTEGER
		);
		CREATE TABLE killer_victim_pairs (
			match_id VARCHAR, killer_xuid VARCHAR,
			weapon_type VARCHAR, time_ms INTEGER
		);
		CREATE TABLE match_participants (
			match_id VARCHAR, xuid VARCHAR,
			team_id INTEGER, rank_in_team INTEGER
		);
		CREATE TABLE match_registry (
			match_id VARCHAR PRIMARY KEY,
			backfill_completed INTEGER DEFAULT 0
		);
		CREATE TABLE weapon_kills (
			match_id VARCHAR, xuid VARCHAR,
			time_ms INTEGER, weapon_id UBIGINT,
			reconciled_as UBIGINT, delta_ms INTEGER,
			confidence VARCHAR, attribution_path VARCHAR,
			swap_detected BOOLEAN, delayed_damage BOOLEAN,
			player_index INTEGER
		);
	`
	if err := execScript(db, ddl); err != nil {
		t.Fatal(err)
	}
	return db
}

func TestGetKillsForPlayer_Empty(t *testing.T) {
	db := openWeaponDB(t)
	kills, err := getKillsForPlayer(db, "m1", "xuid1")
	if err != nil {
		t.Fatal(err)
	}
	if len(kills) != 0 {
		t.Fatalf("expected 0, got %d", len(kills))
	}
}

func TestGetKillsForPlayer_WithData(t *testing.T) {
	db := openWeaponDB(t)
	db.Exec(`INSERT INTO highlight_events VALUES ('m1', 'xuid1', 'kill', 5000)`)
	db.Exec(`INSERT INTO highlight_events VALUES ('m1', 'xuid1', 'kill', 10000)`)
	db.Exec(`INSERT INTO highlight_events VALUES ('m1', 'xuid2', 'kill', 7000)`)

	kills, err := getKillsForPlayer(db, "m1", "xuid1")
	if err != nil {
		t.Fatal(err)
	}
	if len(kills) != 2 {
		t.Fatalf("expected 2, got %d", len(kills))
	}
	if kills[0].TimeMS != 5000 {
		t.Fatalf("expected time_ms=5000, got %d", kills[0].TimeMS)
	}
}

func TestGetXuidToPI_Empty(t *testing.T) {
	db := openWeaponDB(t)
	result, err := getXuidToPI(db, "m1")
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 0 {
		t.Fatalf("expected 0, got %d", len(result))
	}
}

func TestGetXuidToPI_WithData(t *testing.T) {
	db := openWeaponDB(t)
	db.Exec(`INSERT INTO match_participants VALUES ('m1', 'xuid1', 0, 1)`)
	db.Exec(`INSERT INTO match_participants VALUES ('m1', 'xuid2', 0, 2)`)
	db.Exec(`INSERT INTO match_participants VALUES ('m1', 'xuid3', 1, 1)`)

	result, err := getXuidToPI(db, "m1")
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 3 {
		t.Fatalf("expected 3, got %d", len(result))
	}
	if result["xuid1"] != 0 {
		t.Fatalf("expected xuid1=0, got %d", result["xuid1"])
	}
}

func TestInsertWeaponKills_Empty(t *testing.T) {
	db := openWeaponDB(t)
	err := InsertWeaponKills(db, "m1", "xuid1", nil)
	if err != nil {
		t.Fatal(err)
	}
}

func TestInsertWeaponKills_WithData(t *testing.T) {
	db := openWeaponDB(t)
	rows := []WeaponKillRow{
		{TimeMS: 5000, WeaponID: ptrU64(123), Confidence: "high", AttributionPath: "fire_event"},
		{TimeMS: 10000, WeaponID: ptrU64(456), Confidence: "medium", AttributionPath: "timeline"},
	}
	err := InsertWeaponKills(db, "m1", "xuid1", rows)
	if err != nil {
		t.Fatal(err)
	}
	var count int
	db.QueryRow("SELECT COUNT(*) FROM weapon_kills").Scan(&count)
	if count != 2 {
		t.Fatalf("expected 2, got %d", count)
	}
}

func TestMarkWeaponKillsDone(t *testing.T) {
	db := openWeaponDB(t)
	db.Exec("INSERT INTO match_registry (match_id) VALUES ('m1')")

	err := MarkWeaponKillsDone(db, "m1", false)
	if err != nil {
		t.Fatal(err)
	}

	var bits int
	db.QueryRow("SELECT backfill_completed FROM match_registry WHERE match_id='m1'").Scan(&bits)
	if bits&int(MBitWeaponKills) == 0 {
		t.Fatal("expected weapon kills bit set")
	}
}

func TestMarkWeaponKillsDone_NoFilm(t *testing.T) {
	db := openWeaponDB(t)
	db.Exec("INSERT INTO match_registry (match_id) VALUES ('m1')")

	err := MarkWeaponKillsDone(db, "m1", true)
	if err != nil {
		t.Fatal(err)
	}

	var bits int
	db.QueryRow("SELECT backfill_completed FROM match_registry WHERE match_id='m1'").Scan(&bits)
	if bits&int(MBitWeaponKillsNoFilm) == 0 {
		t.Fatal("expected weapon kills no-film bit set")
	}
}

func ptrU64(v uint64) *uint64 { return &v }

func TestAttributionsToRows(t *testing.T) {
	attrs := []analysis.KillAttribution{
		{XUID: "xuid1", TimeMS: 5000, WeaponID: ptrU64(123), Confidence: "high"},
		{XUID: "xuid2", TimeMS: 7000, WeaponID: ptrU64(456), Confidence: "low"},
		{XUID: "xuid1", TimeMS: 10000, WeaponID: ptrU64(789), Confidence: "medium"},
	}
	rows := attributionsToRows(attrs, "xuid1")
	if len(rows) != 2 {
		t.Fatalf("expected 2, got %d", len(rows))
	}
}
