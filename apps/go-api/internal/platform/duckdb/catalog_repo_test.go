//go:build integration

// catalog_repo_test.go — tests Phase F plan catalogue (lecture).
package duckdb

import (
	"context"
	"database/sql"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/migration"
)

func setupCatalogRepoDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	if err := migration.RunForDB(db, migration.TargetMetadata); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	// La migration seed_ranked_playlists_catalog pré-remplit playlists_catalog
	// avec les playlists ranked réelles HI. On repart d'un catalogue vide pour
	// que les comptages des tests portent uniquement sur le jeu seedé ci-dessous.
	db.Exec(`DELETE FROM playlists_catalog`)
	// Seed.
	db.Exec(`INSERT INTO playlists_catalog (title_slug, playlist_asset_id, name_canonical, experience, is_ranked, is_active)
	         VALUES ('halo_infinite', 'pl-1', 'Quick Play', 'social', FALSE, TRUE)`)
	db.Exec(`INSERT INTO playlists_catalog (title_slug, playlist_asset_id, name_canonical, experience, is_ranked, is_active)
	         VALUES ('halo_infinite', 'pl-2', 'Ranked Arena', 'ranked', TRUE, TRUE)`)
	db.Exec(`INSERT INTO playlists_catalog (title_slug, playlist_asset_id, name_canonical, is_active)
	         VALUES ('halo_infinite', 'pl-old', 'Old Playlist', FALSE)`)
	db.Exec(`INSERT INTO playlists_catalog (title_slug, playlist_asset_id, name_canonical, is_active)
	         VALUES ('synthetic_title_b', 'pl-1', 'Synth Quick Play', TRUE)`)

	db.Exec(`INSERT INTO maps_catalog (title_slug, map_asset_id, name_canonical, image_url)
	         VALUES ('halo_infinite', 'm-1', 'Bazaar', '/api/v1/assets/maps/bazaar.png')`)
	db.Exec(`INSERT INTO maps_catalog (title_slug, map_asset_id, name_canonical)
	         VALUES ('halo_infinite', 'm-2', 'Live Fire')`)

	db.Exec(`INSERT INTO map_mode_pair_definitions (title_slug, pair_asset_id, name_canonical, map_asset_id, game_variant_asset_id, mode_category)
	         VALUES ('halo_infinite', 'pa-1', 'Arena:Slayer on Bazaar', 'm-1', 'gv-1', 'Assassin')`)
	db.Exec(`INSERT INTO playlist_pair_links (title_slug, playlist_asset_id, pair_asset_id, weight)
	         VALUES ('halo_infinite', 'pl-1', 'pa-1', 1.0)`)
	return db
}

func TestCatalogRepo_PlaylistsByTitle_FullCatalog(t *testing.T) {
	ctx := context.Background()
	db := setupCatalogRepoDB(t)
	r := NewCatalogRepo(db, nil)

	pls, err := r.PlaylistsByTitle(ctx, "halo_infinite", "", false)
	if err != nil {
		t.Fatalf("PlaylistsByTitle: %v", err)
	}
	// Doit retourner les 2 actives, exclure pl-old (is_active=false).
	if len(pls) != 2 {
		t.Errorf("len = %d, want 2 (excl is_active=false)", len(pls))
	}
	for _, p := range pls {
		if p.PlaylistAssetID == "pl-old" {
			t.Error("pl-old (inactive) ne devrait pas être retourné")
		}
		if p.TitleSlug != "halo_infinite" {
			t.Errorf("title_slug = %q (cross-leak)", p.TitleSlug)
		}
	}
}

func TestCatalogRepo_PlaylistsByTitle_TitleIsolation(t *testing.T) {
	ctx := context.Background()
	db := setupCatalogRepoDB(t)
	r := NewCatalogRepo(db, nil)

	pls, err := r.PlaylistsByTitle(ctx, "synthetic_title_b", "", false)
	if err != nil {
		t.Fatalf("PlaylistsByTitle: %v", err)
	}
	if len(pls) != 1 {
		t.Errorf("len = %d, want 1 (synth only)", len(pls))
	}
	if pls[0].Name != "Synth Quick Play" {
		t.Errorf("name = %q (isolation broken)", pls[0].Name)
	}
}

func TestCatalogRepo_PairsByPlaylist(t *testing.T) {
	ctx := context.Background()
	db := setupCatalogRepoDB(t)
	r := NewCatalogRepo(db, nil)

	pairs, err := r.PairsByPlaylist(ctx, "halo_infinite", "pl-1")
	if err != nil {
		t.Fatalf("PairsByPlaylist: %v", err)
	}
	if len(pairs) != 1 {
		t.Errorf("len = %d, want 1", len(pairs))
	}
	if pairs[0].PairAssetID != "pa-1" {
		t.Errorf("pair = %q", pairs[0].PairAssetID)
	}
	if pairs[0].Weight != 1.0 {
		t.Errorf("weight = %f, want 1.0", pairs[0].Weight)
	}
	if pairs[0].ModeCategory != "Assassin" {
		t.Errorf("mode_category = %q, want Assassin", pairs[0].ModeCategory)
	}
}

func TestCatalogRepo_MapsByTitle(t *testing.T) {
	ctx := context.Background()
	db := setupCatalogRepoDB(t)
	r := NewCatalogRepo(db, nil)

	maps, err := r.MapsByTitle(ctx, "halo_infinite", "", false)
	if err != nil {
		t.Fatalf("MapsByTitle: %v", err)
	}
	if len(maps) != 2 {
		t.Errorf("len = %d, want 2", len(maps))
	}
}

// TestCatalogRepo_playlistsPlayedByXUID : split+merge cross-DB.
// Vérifie que seules les playlists avec ≥ 1 match joué sont retournées,
// hydratées avec leur MatchCount.
func TestCatalogRepo_playlistsPlayedByXUID(t *testing.T) {
	ctx := context.Background()
	pdb := newTestPlayerDB(t)

	// Création locale de playlists_catalog (pas dans seedMetaDBSchema
	// pour éviter une dépendance globale aux tests qui n'en ont pas besoin).
	if _, err := pdb.Metadata.Exec(ctx, `
		CREATE TABLE playlists_catalog (
			title_slug VARCHAR NOT NULL,
			playlist_asset_id VARCHAR NOT NULL,
			current_version_id VARCHAR,
			name_canonical VARCHAR,
			experience VARCHAR,
			is_ranked BOOLEAN DEFAULT FALSE,
			is_active BOOLEAN DEFAULT TRUE,
			PRIMARY KEY (title_slug, playlist_asset_id))`); err != nil {
		t.Fatalf("create playlists_catalog: %v", err)
	}

	// Seed metadata.playlists_catalog (3 playlists actives).
	for _, p := range []struct {
		id, name, exp string
		ranked        bool
	}{
		{"pl-quick", "Quick Play", "social", false},
		{"pl-ranked", "Ranked Arena", "ranked", true},
		{"pl-never", "Never Played", "social", false},
	} {
		if _, err := pdb.Metadata.Exec(ctx,
			`INSERT INTO playlists_catalog
				(title_slug, playlist_asset_id, name_canonical, experience, is_ranked, is_active)
			 VALUES ('halo_infinite', ?, ?, ?, ?, TRUE)`,
			p.id, p.name, p.exp, p.ranked); err != nil {
			t.Fatalf("seed playlists_catalog: %v", err)
		}
	}

	// Seed shared : 3 matchs sur pl-quick (joués) + 1 sur pl-ranked.
	// pl-never reste sans match → exclue du résultat.
	for _, m := range []struct {
		matchID, playlistID string
	}{
		{"mq_1", "pl-quick"},
		{"mq_2", "pl-quick"},
		{"mq_3", "pl-quick"},
		{"mr_1", "pl-ranked"},
	} {
		if _, err := pdb.Shared.Exec(ctx,
			`INSERT INTO shared.match_registry (match_id, playlist_id) VALUES (?, ?)`,
			m.matchID, m.playlistID); err != nil {
			t.Fatalf("seed match_registry: %v", err)
		}
		if _, err := pdb.Shared.Exec(ctx,
			`INSERT INTO shared.match_participants (match_id, xuid) VALUES (?, ?)`,
			m.matchID, pTestXUID); err != nil {
			t.Fatalf("seed match_participants: %v", err)
		}
	}

	// Construction du repo avec SharedReader legacy autour de pdb.Shared.
	repo := NewCatalogRepo(pdb.Metadata.SQLDb(), LegacySharedReader(pdb.Shared))

	pls, err := repo.PlaylistsByTitle(ctx, "halo_infinite", pTestXUID, true)
	if err != nil {
		t.Fatalf("PlaylistsByTitle onlyPlayed: %v", err)
	}
	if len(pls) != 2 {
		t.Fatalf("attendu 2 playlists jouées, obtenu %d : %v", len(pls), pls)
	}
	got := map[string]int{}
	for _, p := range pls {
		got[p.PlaylistAssetID] = p.MatchCount
	}
	if got["pl-quick"] != 3 {
		t.Errorf("pl-quick MatchCount = %d, want 3", got["pl-quick"])
	}
	if got["pl-ranked"] != 1 {
		t.Errorf("pl-ranked MatchCount = %d, want 1", got["pl-ranked"])
	}
	if _, ok := got["pl-never"]; ok {
		t.Error("pl-never (sans match) ne devrait pas être retournée")
	}
}

func TestCatalogRepo_CountCatalogEntries(t *testing.T) {
	ctx := context.Background()
	db := setupCatalogRepoDB(t)
	r := NewCatalogRepo(db, nil)

	n, err := r.CountCatalogEntries(ctx, "halo_infinite")
	if err != nil {
		t.Fatalf("CountCatalogEntries: %v", err)
	}
	if n != 2 {
		t.Errorf("count = %d, want 2 (active only)", n)
	}

	n, _ = r.CountCatalogEntries(ctx, "synthetic_title_b")
	if n != 1 {
		t.Errorf("synth count = %d, want 1", n)
	}

	n, _ = r.CountCatalogEntries(ctx, "nonexistent")
	if n != 0 {
		t.Errorf("nonexistent count = %d, want 0", n)
	}
}
