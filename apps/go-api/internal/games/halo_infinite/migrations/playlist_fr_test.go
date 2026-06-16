//go:build integration

// playlist_fr_test.go — déplacé depuis internal/migration (Phase 1.5 b7/b10) avec
// applyPlaylistFRSeeds (mode_playlist_fr.go).
package migrations

import (
	"database/sql"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"
)

func setupAssetTranslationsForPlaylistFR(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	if _, err := db.Exec(`CREATE TABLE asset_translations (
		asset_id VARCHAR, asset_type VARCHAR, lang VARCHAR, name VARCHAR,
		PRIMARY KEY (asset_id, asset_type, lang))`); err != nil {
		t.Fatalf("schema: %v", err)
	}
	return db
}

// TestApplyPlaylistFRSeeds_FixesCorruptedFRRow vérifie qu'une ligne fr-FR
// contenant l'EN raw est mise à jour avec la traduction canonique.
func TestApplyPlaylistFRSeeds_FixesCorruptedFRRow(t *testing.T) {
	db := setupAssetTranslationsForPlaylistFR(t)
	const playlistID = "uuid-quick-play"
	if _, err := db.Exec(`INSERT INTO asset_translations VALUES
		(?, 'playlist', 'en-US', 'Quick Play'),
		(?, 'playlist', 'fr-FR', 'Quick Play')`, playlistID, playlistID); err != nil {
		t.Fatal(err)
	}

	if err := applyPlaylistFRSeeds(db); err != nil {
		t.Fatalf("apply: %v", err)
	}

	var got string
	if err := db.QueryRow(`SELECT name FROM asset_translations
		WHERE asset_id = ? AND asset_type = 'playlist' AND lang = 'fr-FR'`,
		playlistID).Scan(&got); err != nil {
		t.Fatalf("query: %v", err)
	}
	if got != "Partie rapide" {
		t.Errorf("name fr-FR = %q, want %q", got, "Partie rapide")
	}
}

// TestApplyPlaylistFRSeeds_PreservesAlreadyTranslated garde-fou : ne pas
// écraser une traduction FR déjà correcte.
func TestApplyPlaylistFRSeeds_PreservesAlreadyTranslated(t *testing.T) {
	db := setupAssetTranslationsForPlaylistFR(t)
	const playlistID = "uuid-bteam"
	if _, err := db.Exec(`INSERT INTO asset_translations VALUES
		(?, 'playlist', 'en-US', 'Big Team Battle'),
		(?, 'playlist', 'fr-FR', 'Mon Custom FR Personnalisé')`, playlistID, playlistID); err != nil {
		t.Fatal(err)
	}

	if err := applyPlaylistFRSeeds(db); err != nil {
		t.Fatalf("apply: %v", err)
	}

	var got string
	if err := db.QueryRow(`SELECT name FROM asset_translations
		WHERE asset_id = ? AND lang = 'fr-FR'`, playlistID).Scan(&got); err != nil {
		t.Fatalf("query: %v", err)
	}
	if got != "Mon Custom FR Personnalisé" {
		t.Errorf("préservation FR custom : got %q", got)
	}
}

// TestApplyPlaylistFRSeeds_InsertsMissingFRRow crée une ligne fr-FR pour les
// asset_id qui n'avaient qu'une ligne en-US.
func TestApplyPlaylistFRSeeds_InsertsMissingFRRow(t *testing.T) {
	db := setupAssetTranslationsForPlaylistFR(t)
	const playlistID = "uuid-only-en"
	if _, err := db.Exec(`INSERT INTO asset_translations VALUES
		(?, 'playlist', 'en-US', 'Ranked Arena')`, playlistID); err != nil {
		t.Fatal(err)
	}

	if err := applyPlaylistFRSeeds(db); err != nil {
		t.Fatalf("apply: %v", err)
	}

	var got string
	if err := db.QueryRow(`SELECT name FROM asset_translations
		WHERE asset_id = ? AND lang = 'fr-FR'`, playlistID).Scan(&got); err != nil {
		t.Fatalf("query INSERT row: %v", err)
	}
	if got != "Arène classée" {
		t.Errorf("INSERT row fr-FR = %q, want %q", got, "Arène classée")
	}
}

// TestApplyPlaylistFRSeeds_Idempotent : exécuter deux fois donne le même résultat.
func TestApplyPlaylistFRSeeds_Idempotent(t *testing.T) {
	db := setupAssetTranslationsForPlaylistFR(t)
	const playlistID = "uuid-quick-play"
	if _, err := db.Exec(`INSERT INTO asset_translations VALUES
		(?, 'playlist', 'en-US', 'Quick Play'),
		(?, 'playlist', 'fr-FR', 'Quick Play')`, playlistID, playlistID); err != nil {
		t.Fatal(err)
	}

	if err := applyPlaylistFRSeeds(db); err != nil {
		t.Fatalf("apply 1: %v", err)
	}
	if err := applyPlaylistFRSeeds(db); err != nil {
		t.Fatalf("apply 2: %v", err)
	}

	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM asset_translations
		WHERE asset_id = ? AND lang = 'fr-FR'`, playlistID).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Errorf("après 2 runs : %d rows fr-FR, want 1", count)
	}
}

// TestApplyPlaylistFRSeeds_NoAssetTranslationsTable : robustesse — échec propre
// si la table n'existe pas encore.
func TestApplyPlaylistFRSeeds_NoAssetTranslationsTable(t *testing.T) {
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer db.Close()
	err = applyPlaylistFRSeeds(db)
	if err == nil {
		t.Fatal("attendu erreur quand asset_translations absente")
	}
}
