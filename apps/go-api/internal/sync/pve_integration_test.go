//go:build integration

package sync

import (
	"database/sql"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"
)

func openPveDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { db.Close() })

	ddl := `
		CREATE TABLE pve_match_stats (
			match_id VARCHAR, xuid VARCHAR,
			waves_completed INTEGER, boss_kills INTEGER,
			grunt_kills INTEGER, elite_kills INTEGER,
			jackal_kills INTEGER, brute_kills INTEGER,
			hunter_kills INTEGER, skimmer_kills INTEGER,
			crawler_kills INTEGER, soldier_kills INTEGER,
			knight_kills INTEGER, warden_kills INTEGER,
			sentinel_kills INTEGER, marine_kills INTEGER,
			total_kills INTEGER, deaths INTEGER,
			damage_dealt DOUBLE, pve_bits INTEGER,
			PRIMARY KEY (match_id, xuid)
		);
		CREATE TABLE match_registry (
			match_id VARCHAR PRIMARY KEY,
			backfill_completed INTEGER DEFAULT 0
		);
	`
	if err := execScript(db, ddl); err != nil {
		t.Fatal(err)
	}
	return db
}

func TestInsertPveStats_Empty(t *testing.T) {
	db := openPveDB(t)
	n, err := InsertPveStats(db, nil)
	if err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Fatalf("expected 0, got %d", n)
	}
}

func TestInsertPveStats_Insert(t *testing.T) {
	db := openPveDB(t)
	rows := []PveMatchStatsRow{
		{MatchID: "m1", XUID: "x1", WavesCompleted: 5, BossKills: 2, TotalKills: 100, Deaths: 3, DamageDealt: 5000.0},
		{MatchID: "m2", XUID: "x1", WavesCompleted: 3, BossKills: 1, TotalKills: 50, Deaths: 5, DamageDealt: 2500.0},
	}
	n, err := InsertPveStats(db, rows)
	if err != nil {
		t.Fatal(err)
	}
	if n != 2 {
		t.Fatalf("expected 2, got %d", n)
	}

	var count int
	db.QueryRow("SELECT COUNT(*) FROM pve_match_stats").Scan(&count)
	if count != 2 {
		t.Fatalf("expected 2 rows in table, got %d", count)
	}
}

func TestInsertPveStats_Upsert(t *testing.T) {
	db := openPveDB(t)
	row := PveMatchStatsRow{MatchID: "m1", XUID: "x1", WavesCompleted: 5}
	InsertPveStats(db, []PveMatchStatsRow{row})

	row.WavesCompleted = 10
	n, err := InsertPveStats(db, []PveMatchStatsRow{row})
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("expected 1, got %d", n)
	}
}

func TestMarkPveStatsDone(t *testing.T) {
	db := openPveDB(t)
	if _, err := db.Exec("INSERT INTO match_registry (match_id) VALUES ('m1')"); err != nil {
		t.Fatal(err)
	}

	if err := MarkPveStatsDone(db, "m1"); err != nil {
		t.Fatal(err)
	}

	var bits int
	db.QueryRow("SELECT backfill_completed FROM match_registry WHERE match_id='m1'").Scan(&bits)
	if bits&int(MBitPVEStats) == 0 {
		t.Fatal("expected PVE bit set")
	}
}
