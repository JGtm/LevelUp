//go:build integration

// Package sync — career_test.go : tests pour les fonctions de carrière.
//
// Sprint 47 T13 — parseCareerRank (pur) + saveCareerRank (DuckDB in-memory).
// Note : ce package importe DuckDB transitif → ne compile pas sur Windows (contrainte
// build constraint windows-amd64). Ces tests sont conçus pour tourner en CI Linux.
package sync

import (
	"database/sql"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"
)

// openMemForCareer ouvre une DuckDB in-memory avec la table career_progression.
func openMemForCareer(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("openMemForCareer: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS career_progression (
		xuid VARCHAR NOT NULL,
		rank INTEGER,
		rank_name VARCHAR,
		rank_tier VARCHAR,
		current_xp INTEGER,
		xp_for_next_rank INTEGER,
		xp_total INTEGER,
		is_max_rank BOOLEAN,
		adornment_path VARCHAR,
		spartan_id VARCHAR,
		banner_image_url VARCHAR,
		emblem_image_url VARCHAR,
		backdrop_image_url VARCHAR,
		recorded_at TIMESTAMP
	)`)
	if err != nil {
		t.Fatalf("CREATE career_progression: %v", err)
	}
	return db
}

// ── Tests parseCareerRank (pure) ─────────────────────────────────────────────

func TestParseCareerRank_FullBody(t *testing.T) {
	body := map[string]interface{}{
		"Rank": map[string]interface{}{
			"Value": float64(15),
			"Name":  "Sergeant",
			"Tier":  "Gold",
		},
		"RewardTrack": map[string]interface{}{
			"CurrentProgress":      float64(1200),
			"NextLevelRequirement": float64(5000),
			"TotalEarned":          float64(42000),
			"IsMaxRank":            false,
		},
		"AdornmentPath": "path/to/adornment",
		"SpartanId":     "spartan-abc",
	}

	result := parseCareerRank(body, "xuid-test-001")

	if result == nil {
		t.Fatal("parseCareerRank retourne nil")
	}
	if result.XUID != "xuid-test-001" {
		t.Errorf("XUID attendu 'xuid-test-001', obtenu %q", result.XUID)
	}
	if result.CurrentRank != 15 {
		t.Errorf("CurrentRank attendu 15, obtenu %d", result.CurrentRank)
	}
	if result.CurrentRankName != "Sergeant" {
		t.Errorf("CurrentRankName attendu 'Sergeant', obtenu %q", result.CurrentRankName)
	}
	if result.CurrentXP != 1200 {
		t.Errorf("CurrentXP attendu 1200, obtenu %d", result.CurrentXP)
	}
	if result.XPTotal != 42000 {
		t.Errorf("XPTotal attendu 42000, obtenu %d", result.XPTotal)
	}
	if result.AdornmentPath != "path/to/adornment" {
		t.Errorf("AdornmentPath attendu 'path/to/adornment', obtenu %q", result.AdornmentPath)
	}
}

func TestParseCareerRank_EmptyBody(t *testing.T) {
	result := parseCareerRank(map[string]interface{}{}, "xuid-empty")
	if result == nil {
		t.Fatal("parseCareerRank retourne nil sur body vide")
	}
	if result.CurrentRank != 0 {
		t.Errorf("CurrentRank attendu 0, obtenu %d", result.CurrentRank)
	}
	if result.XUID != "xuid-empty" {
		t.Errorf("XUID non préservé sur body vide")
	}
}

func TestParseCareerRank_MaxRank(t *testing.T) {
	body := map[string]interface{}{
		"Rank": map[string]interface{}{
			"Value": float64(272),
			"Name":  "Reclaimer",
			"Tier":  "Onyx",
		},
		"RewardTrack": map[string]interface{}{
			"IsMaxRank": true,
		},
	}
	result := parseCareerRank(body, "xuid-max")
	if !result.IsMaxRank {
		t.Error("IsMaxRank devrait être true")
	}
	if result.CurrentRankTier != "Onyx" {
		t.Errorf("CurrentRankTier attendu 'Onyx', obtenu %q", result.CurrentRankTier)
	}
}

// ── Tests saveCareerRank ─────────────────────────────────────────────────────

func TestSaveCareerRank_Insert(t *testing.T) {
	db := openMemForCareer(t)

	data := &CareerRankData{
		XUID:            "xuid-save-001",
		CurrentRank:     10,
		CurrentRankName: "Private",
		CurrentXP:       500,
		XPTotal:         10000,
		BannerImageURL:  "https://example.test/banner.png",
	}

	if err := saveCareerRank(t.Context(), db, data); err != nil {
		t.Fatalf("saveCareerRank: %v", err)
	}

	var cnt int
	if err := db.QueryRow("SELECT COUNT(*) FROM career_progression WHERE xuid = 'xuid-save-001'").Scan(&cnt); err != nil {
		t.Fatalf("COUNT: %v", err)
	}
	if cnt != 1 {
		t.Errorf("attendu 1 ligne insérée, obtenu %d", cnt)
	}

	var bannerURL string
	if err := db.QueryRow("SELECT banner_image_url FROM career_progression WHERE xuid = 'xuid-save-001'").Scan(&bannerURL); err != nil {
		t.Fatalf("SELECT banner_image_url: %v", err)
	}
	if bannerURL != "https://example.test/banner.png" {
		t.Errorf("banner_image_url attendu https://example.test/banner.png, obtenu %q", bannerURL)
	}
}

func TestSaveCareerRank_MultipleSnapshots(t *testing.T) {
	db := openMemForCareer(t)

	for i, rank := range []int{10, 11, 12} {
		data := &CareerRankData{
			XUID:        "xuid-multi",
			CurrentRank: rank,
			CurrentXP:   i * 1000,
		}
		if err := saveCareerRank(t.Context(), db, data); err != nil {
			t.Fatalf("saveCareerRank rank=%d: %v", rank, err)
		}
	}

	var cnt int
	if err := db.QueryRow("SELECT COUNT(*) FROM career_progression WHERE xuid = 'xuid-multi'").Scan(&cnt); err != nil {
		t.Fatalf("COUNT: %v", err)
	}
	if cnt != 3 {
		t.Errorf("attendu 3 snapshots, obtenu %d", cnt)
	}
}
