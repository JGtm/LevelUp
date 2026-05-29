//go:build integration

package migration

import (
	"database/sql"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/games/halo_infinite/rankedplaylists"
)

func setupCatalogDBForSeed(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	if _, err := db.Exec(`
		CREATE TABLE playlists_catalog (
			title_slug VARCHAR NOT NULL, playlist_asset_id VARCHAR NOT NULL,
			current_version_id VARCHAR, name_canonical VARCHAR, experience VARCHAR,
			is_ranked BOOLEAN DEFAULT FALSE, is_active BOOLEAN DEFAULT TRUE,
			first_seen_at TIMESTAMP, last_seen_at TIMESTAMP, last_fetched_at TIMESTAMP,
			PRIMARY KEY (title_slug, playlist_asset_id));
		CREATE TABLE asset_translations (
			asset_id VARCHAR, asset_type VARCHAR, lang VARCHAR, name VARCHAR,
			PRIMARY KEY (asset_id, asset_type, lang));`); err != nil {
		t.Fatalf("schema: %v", err)
	}
	return db
}

// TestApplyRankedPlaylistSeeds_FixesStuckFalse vérifie le coeur du bug récurrent :
// une playlist classée pré-existante marquée is_ranked=FALSE / experience='social'
// est corrigée (DO UPDATE) en is_ranked=TRUE / 'ranked'.
func TestApplyRankedPlaylistSeeds_FixesStuckFalse(t *testing.T) {
	db := setupCatalogDBForSeed(t)
	const arena = "edfef3ac-9cbe-4fa2-b949-8f29deafd483" // Ranked Arena
	if _, err := db.Exec(`INSERT INTO playlists_catalog
		(title_slug, playlist_asset_id, name_canonical, experience, is_ranked, is_active)
		VALUES ('halo_infinite', ?, 'Ranked Arena', 'social', FALSE, TRUE)`, arena); err != nil {
		t.Fatal(err)
	}

	if err := applyRankedPlaylistSeeds(db); err != nil {
		t.Fatalf("apply: %v", err)
	}

	var isRanked bool
	var experience string
	if err := db.QueryRow(`SELECT is_ranked, experience FROM playlists_catalog
		WHERE playlist_asset_id = ?`, arena).Scan(&isRanked, &experience); err != nil {
		t.Fatal(err)
	}
	if !isRanked || experience != "ranked" {
		t.Errorf("Ranked Arena : is_ranked=%v experience=%q ; attendu TRUE/'ranked'", isRanked, experience)
	}
}

// TestApplyRankedPlaylistSeeds_SeedsActiveAndFR vérifie que les 4 playlists
// classées actives sont présentes après seed, et que leur traduction FR existe.
func TestApplyRankedPlaylistSeeds_SeedsActiveAndFR(t *testing.T) {
	db := setupCatalogDBForSeed(t)
	if err := applyRankedPlaylistSeeds(db); err != nil {
		t.Fatalf("apply: %v", err)
	}

	var activeRanked int
	if err := db.QueryRow(`SELECT COUNT(*) FROM playlists_catalog
		WHERE is_ranked = TRUE AND is_active = TRUE`).Scan(&activeRanked); err != nil {
		t.Fatal(err)
	}
	if want := len(rankedplaylists.Active()); activeRanked != want {
		t.Errorf("playlists classées actives en catalogue = %d, attendu %d", activeRanked, want)
	}

	// Chaque active avec NameFR doit avoir une ligne fr-FR dans asset_translations.
	for _, p := range rankedplaylists.Active() {
		if p.NameFR == "" {
			continue
		}
		var fr string
		if err := db.QueryRow(`SELECT name FROM asset_translations
			WHERE asset_id = ? AND asset_type = 'playlist' AND lang = 'fr-FR'`, p.AssetID).Scan(&fr); err != nil {
			t.Errorf("FR manquant pour %s (%s): %v", p.NameEN, p.AssetID, err)
			continue
		}
		if fr != p.NameFR {
			t.Errorf("FR %s = %q, attendu %q", p.AssetID, fr, p.NameFR)
		}
	}
}

// TestApplyRankedPlaylistSeeds_Idempotent vérifie qu'un second passage ne casse rien.
func TestApplyRankedPlaylistSeeds_Idempotent(t *testing.T) {
	db := setupCatalogDBForSeed(t)
	if err := applyRankedPlaylistSeeds(db); err != nil {
		t.Fatalf("apply 1: %v", err)
	}
	if err := applyRankedPlaylistSeeds(db); err != nil {
		t.Fatalf("apply 2: %v", err)
	}
	var total int
	if err := db.QueryRow(`SELECT COUNT(*) FROM playlists_catalog`).Scan(&total); err != nil {
		t.Fatal(err)
	}
	if want := len(rankedplaylists.All()); total != want {
		t.Errorf("total playlists = %d, attendu %d (pas de doublon)", total, want)
	}
}
