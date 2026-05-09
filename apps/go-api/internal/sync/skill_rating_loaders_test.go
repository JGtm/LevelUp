//go:build integration

package sync

import (
	"database/sql"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"
)

func openLUSRDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { db.Close() })

	ddl := `
		CREATE TABLE match_registry (
			match_id VARCHAR PRIMARY KEY,
			start_time TIMESTAMPTZ,
			playlist_name VARCHAR,
			pair_name VARCHAR,
			is_ranked BOOLEAN DEFAULT FALSE,
			is_firefight BOOLEAN DEFAULT FALSE,
			duration_seconds INTEGER
		);
		CREATE TABLE match_participants (
			match_id VARCHAR,
			xuid VARCHAR,
			outcome INTEGER,
			kills INTEGER, deaths INTEGER, assists INTEGER,
			kills_expected DOUBLE, deaths_expected DOUBLE,
			damage_dealt DOUBLE, damage_taken DOUBLE,
			accuracy DOUBLE,
			team_id INTEGER
		);
		CREATE TABLE match_skill_rank (
			match_id VARCHAR PRIMARY KEY,
			rating_type VARCHAR DEFAULT 'LUSR',
			rating_value DOUBLE,
			rating_deviation DOUBLE,
			playlist_group VARCHAR,
			start_time TIMESTAMPTZ
		);
	`
	if err := execScript(db, ddl); err != nil {
		t.Fatal(err)
	}
	return db
}

func TestLoadLUSRMatchData_Empty(t *testing.T) {
	db := openLUSRDB(t)
	data, err := loadLUSRMatchData(db, "xuid_none")
	if err != nil {
		t.Fatal(err)
	}
	if len(data) != 0 {
		t.Fatalf("expected 0, got %d", len(data))
	}
}

func TestLoadLUSRMatchData_WithData(t *testing.T) {
	db := openLUSRDB(t)
	db.Exec(`INSERT INTO match_registry VALUES
		('m1', '2025-01-01 10:00:00'::TIMESTAMPTZ, 'Ranked Arena', 'Slayer', FALSE, FALSE, 600)`)
	db.Exec(`INSERT INTO match_participants (match_id, xuid, outcome, kills, deaths, assists, kills_expected, deaths_expected, damage_dealt, damage_taken, accuracy, team_id) VALUES
		('m1', 'xuid1', 2, 15, 5, 3, 12.0, 6.0, 3000.0, 1500.0, 0.55, 0)`)

	data, err := loadLUSRMatchData(db, "xuid1")
	if err != nil {
		t.Fatal(err)
	}
	if len(data) != 1 {
		t.Fatalf("expected 1, got %d", len(data))
	}
	if data[0].Kills != 15 {
		t.Fatalf("expected kills=15, got %v", data[0].Kills)
	}
}

func TestLoadLUSRMatchData_FiltersRanked(t *testing.T) {
	db := openLUSRDB(t)
	db.Exec(`INSERT INTO match_registry VALUES
		('m1', '2025-01-01 10:00:00'::TIMESTAMPTZ, 'Ranked', 'Slayer', TRUE, FALSE, 600)`)
	db.Exec(`INSERT INTO match_participants (match_id, xuid, outcome, kills, deaths, assists, kills_expected, deaths_expected, damage_dealt, damage_taken, accuracy, team_id) VALUES
		('m1', 'xuid1', 2, 10, 5, 2, 10.0, 5.0, 2000.0, 1000.0, 0.5, 0)`)

	data, err := loadLUSRMatchData(db, "xuid1")
	if err != nil {
		t.Fatal(err)
	}
	if len(data) != 0 {
		t.Fatalf("expected 0 (ranked filtered), got %d", len(data))
	}
}

func TestLoadLUSRParticipants_Empty(t *testing.T) {
	db := openLUSRDB(t)
	result, err := loadLUSRParticipants(db, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 0 {
		t.Fatalf("expected 0, got %d", len(result))
	}
}

func TestLoadLUSRParticipants_WithData(t *testing.T) {
	db := openLUSRDB(t)
	db.Exec(`INSERT INTO match_participants (match_id, xuid, outcome, kills, deaths, assists, kills_expected, deaths_expected, damage_dealt, damage_taken, accuracy, team_id) VALUES
		('m1', 'xuid1', 2, 10, 5, 2, 10.0, 5.0, 2000.0, 1000.0, 0.5, 0),
		('m1', 'xuid2', 3, 8, 7, 1, 9.0, 6.0, 1800.0, 1200.0, 0.4, 1)`)

	result, err := loadLUSRParticipants(db, []string{"m1"})
	if err != nil {
		t.Fatal(err)
	}
	if len(result["m1"]) != 2 {
		t.Fatalf("expected 2 participants, got %d", len(result["m1"]))
	}
}

func TestLoadExistingRatingIDs_Empty(t *testing.T) {
	db := openLUSRDB(t)
	result := loadExistingRatingIDs(db, "LUSR")
	if len(result) != 0 {
		t.Fatalf("expected 0, got %d", len(result))
	}
}

func TestLoadExistingRatingIDs_WithData(t *testing.T) {
	db := openLUSRDB(t)
	db.Exec(`INSERT INTO match_skill_rank VALUES ('m1', 'LUSR', 25.0, 8.33, 'social', '2025-01-01'::TIMESTAMPTZ)`)

	result := loadExistingRatingIDs(db, "LUSR")
	if !result["m1"] {
		t.Fatal("expected m1 in result")
	}
}
