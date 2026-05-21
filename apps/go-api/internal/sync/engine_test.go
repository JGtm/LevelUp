//go:build integration

// Package sync — engine_test.go : tests pour les fonctions utilitaires du moteur de sync.
//
// Sprint 47 T15 — couvrir loadKnownMatchIDs et les fonctions de cycle sync.
package sync

import (
	"database/sql"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"
)

// ── Tests loadKnownMatchIDs ──────────────────────────────────────────────────

func TestLoadKnownMatchIDs_EmptyTable(t *testing.T) {
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer db.Close()

	_, err = db.Exec(`CREATE TABLE player_match_enrichment (
		match_id VARCHAR,
		performance_score FLOAT,
		session_id VARCHAR,
		session_label VARCHAR,
		is_with_friends BOOLEAN
	)`)
	if err != nil {
		t.Fatalf("CREATE: %v", err)
	}

	known, err := loadKnownMatchIDs(t.Context(), db)
	if err != nil {
		t.Fatalf("loadKnownMatchIDs: %v", err)
	}
	if len(known) != 0 {
		t.Errorf("attendu map vide, obtenu %d entrées", len(known))
	}
}

func TestLoadKnownMatchIDs_WithMatches(t *testing.T) {
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer db.Close()

	_, _ = db.Exec(`CREATE TABLE player_match_enrichment (
		match_id VARCHAR,
		performance_score FLOAT,
		session_id VARCHAR,
		session_label VARCHAR,
		is_with_friends BOOLEAN
	)`)
	_, _ = db.Exec(`INSERT INTO player_match_enrichment (match_id) VALUES ('aabbccdd-0000-0000-0000-000000000001')`)
	_, _ = db.Exec(`INSERT INTO player_match_enrichment (match_id) VALUES ('aabbccdd-0000-0000-0000-000000000002')`)

	known, err := loadKnownMatchIDs(t.Context(), db)
	if err != nil {
		t.Fatalf("loadKnownMatchIDs: %v", err)
	}
	if len(known) != 2 {
		t.Errorf("attendu 2 match_id, obtenu %d", len(known))
	}
	if !known["aabbccdd-0000-0000-0000-000000000001"] {
		t.Error("match_id #1 manquant dans la map")
	}
}

func TestLoadKnownMatchIDs_MissingTable(t *testing.T) {
	// Si la table n'existe pas → retourne map vide sans erreur
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer db.Close()

	known, err := loadKnownMatchIDs(t.Context(), db)
	if err != nil {
		t.Fatalf("attendu nil, obtenu %v", err)
	}
	if known == nil {
		t.Error("map ne devrait pas être nil")
	}
}
