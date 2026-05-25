//go:build integration

// Package migration — steps_player_spartan_identity_test.go : vérifie la
// migration create_spartan_identity_table (schéma + backfill depuis
// career_progression). PLAN_SPARTAN_IDENTITY_REFACTOR §11 Phase 4.
//
// Tag `integration` car nécessite le driver DuckDB (CGO).
package migration

import (
	"database/sql"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"
)

// TestRunForDB_Player_SpartanIdentity_Schema : la table est créée par
// ApplySchema, avec les 8 colonnes attendues.
func TestRunForDB_Player_SpartanIdentity_Schema(t *testing.T) {
	db := openMemDB(t)

	if err := RunForDB(db, TargetPlayer); err != nil {
		t.Fatalf("RunForDB(Player): %v", err)
	}

	for _, col := range []string{
		"xuid", "spartan_id", "banner_image_url", "emblem_image_url",
		"backdrop_image_url", "last_refreshed_at", "last_attempt_at",
		"last_attempt_status",
	} {
		mustHaveColumn(t, db, "spartan_identity", col)
	}
}

// TestBackfillSpartanIdentity_NoCareerProgression : si la table source est
// absente, no-op gracieux.
func TestBackfillSpartanIdentity_NoCareerProgression(t *testing.T) {
	db := openMemDB(t)
	// On crée juste spartan_identity sans career_progression.
	if _, err := db.Exec(`CREATE TABLE spartan_identity (
		xuid VARCHAR PRIMARY KEY,
		spartan_id VARCHAR, banner_image_url VARCHAR, emblem_image_url VARCHAR,
		backdrop_image_url VARCHAR, last_refreshed_at TIMESTAMP,
		last_attempt_at TIMESTAMP, last_attempt_status VARCHAR
	)`); err != nil {
		t.Fatalf("seed spartan_identity: %v", err)
	}
	if err := backfillSpartanIdentityFromCareerProgression(db); err != nil {
		t.Fatalf("backfill should be no-op, got error: %v", err)
	}
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM spartan_identity`).Scan(&count); err != nil {
		t.Fatalf("count spartan_identity: %v", err)
	}
	if count != 0 {
		t.Errorf("spartan_identity = %d rows, want 0 (career_progression absent)", count)
	}
}

// TestBackfillSpartanIdentity_CopiesFromCareerProgression : avec données
// historiques en career_progression, le backfill remplit spartan_identity
// avec les dernières valeurs non-vides (ARG_MAX par recorded_at).
func TestBackfillSpartanIdentity_CopiesFromCareerProgression(t *testing.T) {
	db := openMemDB(t)
	seedPlayerSchemaForSpartanBackfill(t, db)

	// Seed 3 snapshots avec valeurs évolutives — on attend que la backfill
	// prenne la dernière valeur non-vide par colonne.
	xuid := "2533274823110022"
	rows := []struct {
		ts          string
		spartanID   string
		bannerURL   string
		emblemURL   string
		backdropURL string
	}{
		{"2026-05-01 10:00:00", "OLD-TAG", "banner-v1.png", "emblem-v1.png", "backdrop-v1.png"},
		{"2026-05-10 12:00:00", "OLD-TAG", "", "emblem-v2.png", ""},                // banner/backdrop vides
		{"2026-05-20 14:00:00", "NEW-TAG", "banner-v3.png", "", "backdrop-v3.png"}, // emblem vide
	}
	for _, r := range rows {
		if _, err := db.Exec(`
			INSERT INTO career_progression (xuid, rank, current_xp, spartan_id, banner_image_url, emblem_image_url, backdrop_image_url, recorded_at)
			VALUES (?, 10, 100, ?, ?, ?, ?, ?)
		`, xuid, r.spartanID, r.bannerURL, r.emblemURL, r.backdropURL, r.ts); err != nil {
			t.Fatalf("insert snapshot %s: %v", r.ts, err)
		}
	}

	if err := backfillSpartanIdentityFromCareerProgression(db); err != nil {
		t.Fatalf("backfill: %v", err)
	}

	var (
		gotSpartanID, gotBanner, gotEmblem, gotBackdrop sql.NullString
		gotStatus                                       sql.NullString
	)
	err := db.QueryRow(`
		SELECT spartan_id, banner_image_url, emblem_image_url, backdrop_image_url, last_attempt_status
		FROM spartan_identity WHERE xuid = ?`, xuid).Scan(
		&gotSpartanID, &gotBanner, &gotEmblem, &gotBackdrop, &gotStatus)
	if err != nil {
		t.Fatalf("load backfilled row: %v", err)
	}
	// Last non-empty values :
	// - spartan_id : NEW-TAG (recorded_at=2026-05-20)
	// - banner_image_url : banner-v3.png (2026-05-20)
	// - emblem_image_url : emblem-v2.png (2026-05-10, car v3 vide)
	// - backdrop_image_url : backdrop-v3.png (2026-05-20)
	if !gotSpartanID.Valid || gotSpartanID.String != "NEW-TAG" {
		t.Errorf("spartan_id = %v, want NEW-TAG", gotSpartanID)
	}
	if !gotBanner.Valid || gotBanner.String != "banner-v3.png" {
		t.Errorf("banner_image_url = %v, want banner-v3.png", gotBanner)
	}
	if !gotEmblem.Valid || gotEmblem.String != "emblem-v2.png" {
		t.Errorf("emblem_image_url = %v, want emblem-v2.png (v3 was empty)", gotEmblem)
	}
	if !gotBackdrop.Valid || gotBackdrop.String != "backdrop-v3.png" {
		t.Errorf("backdrop_image_url = %v, want backdrop-v3.png", gotBackdrop)
	}
	if !gotStatus.Valid || gotStatus.String != "ok" {
		t.Errorf("last_attempt_status = %v, want ok", gotStatus)
	}
}

// TestBackfillSpartanIdentity_Idempotent : second appel ne dupplique pas
// la row (skip si xuid déjà présent).
func TestBackfillSpartanIdentity_Idempotent(t *testing.T) {
	db := openMemDB(t)
	seedPlayerSchemaForSpartanBackfill(t, db)
	xuid := "2533274823110022"
	if _, err := db.Exec(`INSERT INTO career_progression (xuid, rank, current_xp, spartan_id, recorded_at) VALUES (?, 10, 100, 'TAG', '2026-05-20 14:00:00')`, xuid); err != nil {
		t.Fatalf("seed career_progression: %v", err)
	}

	// 1er backfill : INSERT.
	if err := backfillSpartanIdentityFromCareerProgression(db); err != nil {
		t.Fatalf("backfill 1: %v", err)
	}
	// 2nd backfill : should skip.
	if err := backfillSpartanIdentityFromCareerProgression(db); err != nil {
		t.Fatalf("backfill 2: %v", err)
	}
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM spartan_identity`).Scan(&count); err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 1 {
		t.Errorf("spartan_identity count = %d, want 1 (idempotent)", count)
	}
}

// seedPlayerSchemaForSpartanBackfill crée career_progression + spartan_identity
// avec les colonnes attendues par le backfill.
func seedPlayerSchemaForSpartanBackfill(t *testing.T, db *sql.DB) {
	t.Helper()
	if _, err := db.Exec(`CREATE TABLE career_progression (
		xuid VARCHAR,
		rank INTEGER,
		current_xp INTEGER,
		spartan_id VARCHAR DEFAULT '',
		banner_image_url VARCHAR DEFAULT '',
		emblem_image_url VARCHAR DEFAULT '',
		backdrop_image_url VARCHAR DEFAULT '',
		recorded_at TIMESTAMP
	)`); err != nil {
		t.Fatalf("create career_progression: %v", err)
	}
	if _, err := db.Exec(`CREATE TABLE spartan_identity (
		xuid VARCHAR PRIMARY KEY,
		spartan_id VARCHAR, banner_image_url VARCHAR, emblem_image_url VARCHAR,
		backdrop_image_url VARCHAR, last_refreshed_at TIMESTAMP,
		last_attempt_at TIMESTAMP, last_attempt_status VARCHAR
	)`); err != nil {
		t.Fatalf("create spartan_identity: %v", err)
	}
}
