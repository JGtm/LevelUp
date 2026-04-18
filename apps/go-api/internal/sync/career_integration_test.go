//go:build integration

package sync

import (
	"context"
	"database/sql"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"
)

func openCareerDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { db.Close() })

	ddl := `
		CREATE TABLE career_progression (
			xuid VARCHAR, rank INTEGER,
			rank_name VARCHAR, rank_tier VARCHAR,
			current_xp INTEGER, xp_for_next_rank INTEGER, xp_total INTEGER,
			is_max_rank BOOLEAN, adornment_path VARCHAR, spartan_id VARCHAR,
			recorded_at TIMESTAMPTZ
		);
		CREATE TABLE sync_meta (key VARCHAR PRIMARY KEY, value VARCHAR, updated_at TIMESTAMPTZ);
	`
	if _, err := db.Exec(ddl); err != nil {
		t.Fatal(err)
	}
	return db
}

func TestSyncCareerRank_EmptyXUID(t *testing.T) {
	_, err := syncCareerRank(context.Background(), nil, "")
	if err == nil {
		t.Fatal("expected error for empty xuid")
	}
}

func TestSyncCareerRank_NonNumericXUID(t *testing.T) {
	_, err := syncCareerRank(context.Background(), nil, "abc123")
	if err == nil {
		t.Fatal("expected error for non-numeric xuid")
	}
}

func TestSyncCareerRank_ShortXUID(t *testing.T) {
	_, err := syncCareerRank(context.Background(), nil, "12345")
	if err == nil {
		t.Fatal("expected error for short xuid")
	}
}

func TestSaveCareerRank(t *testing.T) {
	db := openCareerDB(t)
	data := &CareerRankData{
		XUID:            "1234567890123456",
		CurrentRank:     42,
		CurrentRankName: "Diamond",
		CurrentRankTier: "IV",
		CurrentXP:       5000,
		XPForNextRank:   8000,
		XPTotal:         50000,
		IsMaxRank:       false,
	}
	err := saveCareerRank(db, data)
	if err != nil {
		t.Fatal(err)
	}

	var count int
	db.QueryRow("SELECT COUNT(*) FROM career_progression").Scan(&count)
	if count != 1 {
		t.Fatalf("expected 1, got %d", count)
	}

	var val string
	db.QueryRow("SELECT value FROM sync_meta WHERE key='current_rank'").Scan(&val)
	if val != "42" {
		t.Fatalf("expected current_rank=42, got %s", val)
	}
}

func TestParseCareerRank_Full(t *testing.T) {
	body := map[string]interface{}{
		"Rank": map[string]interface{}{
			"Value": float64(42),
			"Name":  "Diamond",
			"Tier":  "IV",
		},
		"RewardTrack": map[string]interface{}{
			"CurrentProgress":      float64(5000),
			"NextLevelRequirement": float64(8000),
			"TotalEarned":          float64(50000),
			"IsMaxRank":            true,
		},
		"AdornmentPath": "/path/to/adornment",
		"SpartanId":     "spartan123",
	}
	data := parseCareerRank(body, "xuid1")
	if data.CurrentRank != 42 {
		t.Fatalf("expected rank=42, got %d", data.CurrentRank)
	}
	if !data.IsMaxRank {
		t.Fatal("expected IsMaxRank=true")
	}
	if data.SpartanID != "spartan123" {
		t.Fatalf("expected spartanId, got %s", data.SpartanID)
	}
}
