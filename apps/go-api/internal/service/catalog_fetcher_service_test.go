//go:build integration

// catalog_fetcher_service_test.go — tests Phase F plan catalogue.
package service

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/games"
	"levelup/go-api/internal/games/canonical"
	"levelup/go-api/internal/migration"
)

// stubCatalogAdapter implémente games.TitleCatalogAdapter pour tests d'intégration.
type stubCatalogAdapter struct {
	playlists    map[string]canonical.CanonicalPlaylist
	pairs        map[string]canonical.CanonicalPair
	maps         map[string]canonical.CanonicalMap
	gameVariants map[string]canonical.CanonicalGameVariant
	failOn       map[string]error // key = "asset_type:asset_id" → erreur à retourner
}

func (a *stubCatalogAdapter) TitleSlug() string { return "halo_infinite" }

func (a *stubCatalogAdapter) FetchPlaylist(_ context.Context, id, _ string) (canonical.CanonicalPlaylist, error) {
	if err, ok := a.failOn["playlist:"+id]; ok {
		return canonical.CanonicalPlaylist{}, err
	}
	if pl, ok := a.playlists[id]; ok {
		return pl, nil
	}
	return canonical.CanonicalPlaylist{}, errors.New("playlist not found in stub")
}

func (a *stubCatalogAdapter) FetchPair(_ context.Context, id, _ string) (canonical.CanonicalPair, error) {
	if err, ok := a.failOn["pair:"+id]; ok {
		return canonical.CanonicalPair{}, err
	}
	if p, ok := a.pairs[id]; ok {
		return p, nil
	}
	return canonical.CanonicalPair{}, errors.New("pair not found in stub")
}

func (a *stubCatalogAdapter) FetchMap(_ context.Context, id, _ string) (canonical.CanonicalMap, error) {
	if err, ok := a.failOn["map:"+id]; ok {
		return canonical.CanonicalMap{}, err
	}
	if m, ok := a.maps[id]; ok {
		return m, nil
	}
	return canonical.CanonicalMap{}, errors.New("map not found in stub")
}

func (a *stubCatalogAdapter) FetchGameVariant(_ context.Context, id, _ string) (canonical.CanonicalGameVariant, error) {
	if gv, ok := a.gameVariants[id]; ok {
		return gv, nil
	}
	return canonical.CanonicalGameVariant{}, errors.New("game variant not found in stub")
}

func (a *stubCatalogAdapter) ClassifyExperience(_ canonical.CanonicalPlaylist) canonical.Experience {
	return canonical.ExperienceUnknown
}

// stubResolver retourne le stub adapter pour tout slug.
type stubResolver struct {
	adapter games.TitleCatalogAdapter
}

func (r *stubResolver) Data(_ string) (games.TitleDataAdapter, error)         { return nil, nil }
func (r *stubResolver) Semantic(_ string) (games.TitleSemanticAdapter, error) { return nil, nil }
func (r *stubResolver) AssetURL(_ string) (games.TitleAssetURLAdapter, error) { return nil, nil }
func (r *stubResolver) Catalog(_ string) (games.TitleCatalogAdapter, error)   { return r.adapter, nil }
func (r *stubResolver) DefaultSlug() string                                   { return "halo_infinite" }

func setupCatalogTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	if err := migration.RunForDB(db, migration.TargetMetadata); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

func TestCatalogFetcherService_Drain_PlaylistAndPair(t *testing.T) {
	ctx := context.Background()
	db := setupCatalogTestDB(t)

	// Seed la queue avec 1 playlist et 1 pair à drainer.
	db.Exec(`INSERT INTO catalog_fetch_queue (title_slug, asset_type, asset_id) VALUES ('halo_infinite', 'playlist', 'pl-1')`)
	db.Exec(`INSERT INTO catalog_fetch_queue (title_slug, asset_type, asset_id) VALUES ('halo_infinite', 'pair', 'pa-1')`)

	adapter := &stubCatalogAdapter{
		playlists: map[string]canonical.CanonicalPlaylist{
			"pl-1": {AssetID: "pl-1", VersionID: "v1", NameCanonical: "Quick Play", Experience: canonical.ExperienceSocial},
		},
		pairs: map[string]canonical.CanonicalPair{
			"pa-1": {AssetID: "pa-1", VersionID: "v1", NameCanonical: "Arena:Slayer on Bazaar",
				MapAssetID: "ma-1", GameVariantAssetID: "gv-1", ModeCategory: "Assassin",
				ModeLabels: map[string]string{"en": "Slayer", "fr": "Assassin"},
			},
		},
	}
	svc := NewCatalogFetcherService(db, &stubResolver{adapter: adapter}, 5)

	res, err := svc.Drain(ctx, "halo_infinite")
	if err != nil {
		t.Fatalf("Drain: %v", err)
	}
	if res.Playlists != 1 {
		t.Errorf("Playlists = %d, want 1", res.Playlists)
	}
	if res.Pairs != 1 {
		t.Errorf("Pairs = %d, want 1", res.Pairs)
	}
	if res.Errors != 0 {
		t.Errorf("Errors = %d, want 0", res.Errors)
	}

	// Playlist + pair upserted.
	var n int
	db.QueryRow(`SELECT COUNT(*) FROM playlists_catalog WHERE title_slug = 'halo_infinite'`).Scan(&n)
	if n != 1 {
		t.Errorf("playlists_catalog count = %d, want 1", n)
	}
	db.QueryRow(`SELECT COUNT(*) FROM map_mode_pair_definitions WHERE title_slug = 'halo_infinite'`).Scan(&n)
	if n != 1 {
		t.Errorf("map_mode_pair_definitions count = %d, want 1", n)
	}

	// Map et game_variant ré-enqueués (référencés par le pair).
	db.QueryRow(`SELECT COUNT(*) FROM catalog_fetch_queue WHERE asset_type = 'map'`).Scan(&n)
	if n != 1 {
		t.Errorf("re-enqueued maps = %d, want 1", n)
	}
	db.QueryRow(`SELECT COUNT(*) FROM catalog_fetch_queue WHERE asset_type = 'game_variant'`).Scan(&n)
	if n != 1 {
		t.Errorf("re-enqueued game_variants = %d, want 1", n)
	}

	// Labels normalisés multi-langues persistés.
	db.QueryRow(`SELECT COUNT(*) FROM pair_mode_label_translations WHERE pair_asset_id = 'pa-1'`).Scan(&n)
	if n != 2 {
		t.Errorf("pair_mode_label_translations count = %d, want 2 (en + fr)", n)
	}
}

func TestCatalogFetcherService_Drain_TransientError_RetryablePotrf(t *testing.T) {
	ctx := context.Background()
	db := setupCatalogTestDB(t)

	db.Exec(`INSERT INTO catalog_fetch_queue (title_slug, asset_type, asset_id) VALUES ('halo_infinite', 'playlist', 'pl-fail')`)

	adapter := &stubCatalogAdapter{
		failOn: map[string]error{
			"playlist:pl-fail": errors.New("503 service unavailable"),
		},
	}
	svc := NewCatalogFetcherService(db, &stubResolver{adapter: adapter}, 5)

	res, err := svc.Drain(ctx, "halo_infinite")
	if err != nil {
		t.Fatalf("Drain: %v", err)
	}
	if res.Errors != 1 {
		t.Errorf("Errors = %d, want 1", res.Errors)
	}

	// La ligne reste en queue avec attempts=1 et last_error renseigné.
	var attempts int
	var lastError sql.NullString
	err = db.QueryRow(`SELECT attempts, last_error FROM catalog_fetch_queue WHERE asset_id = 'pl-fail'`).Scan(&attempts, &lastError)
	if err != nil {
		t.Fatalf("queue read: %v", err)
	}
	if attempts != 1 {
		t.Errorf("attempts = %d, want 1", attempts)
	}
	if !lastError.Valid || lastError.String == "" {
		t.Error("last_error not set")
	}
}

func TestCatalogFetcherService_Drain_MaxRetriesReached_Skipped(t *testing.T) {
	ctx := context.Background()
	db := setupCatalogTestDB(t)

	db.Exec(`INSERT INTO catalog_fetch_queue (title_slug, asset_type, asset_id, attempts) VALUES ('halo_infinite', 'playlist', 'pl-x', 5)`)

	adapter := &stubCatalogAdapter{}
	svc := NewCatalogFetcherService(db, &stubResolver{adapter: adapter}, 5)

	res, err := svc.Drain(ctx, "halo_infinite")
	if err != nil {
		t.Fatalf("Drain: %v", err)
	}
	if res.Playlists != 0 || res.Errors != 0 {
		t.Errorf("Drain doit ignorer attempts >= maxRetries (got Playlists=%d Errors=%d)", res.Playlists, res.Errors)
	}

	// Toujours en queue, intacte.
	var n int
	db.QueryRow(`SELECT COUNT(*) FROM catalog_fetch_queue WHERE asset_id = 'pl-x'`).Scan(&n)
	if n != 1 {
		t.Errorf("queue count = %d, want 1 (max retries skip)", n)
	}
}

func TestCatalogFetcherService_Drain_EmptyQueue_NoError(t *testing.T) {
	ctx := context.Background()
	db := setupCatalogTestDB(t)
	svc := NewCatalogFetcherService(db, &stubResolver{adapter: &stubCatalogAdapter{}}, 5)
	res, err := svc.Drain(ctx, "halo_infinite")
	if err != nil {
		t.Fatalf("Drain empty: %v", err)
	}
	if res.Playlists+res.Pairs+res.Maps+res.GameVariants+res.Errors != 0 {
		t.Errorf("Drain empty queue should have 0 counters, got %+v", res)
	}
}
