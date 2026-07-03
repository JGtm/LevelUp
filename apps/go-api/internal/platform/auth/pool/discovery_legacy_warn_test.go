// Package pool — discovery_legacy_warn_test.go : le label (et donc le warn)
// « legacy sync_meta utilisée » ne doit refléter que les valeurs réellement
// ADOPTÉES — pas la simple présence de résidus en DB (plan anti-bruit
// 2026-06-11). Build tag cgo : crée une vraie player DB DuckDB.
//
//go:build cgo

package pool

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/observability"
	"levelup/go-api/internal/platform/auth"
	"levelup/go-api/internal/platform/duckdb"
)

// seedSyncMetaRT crée la player DB au chemin canonique avec un sync_meta
// contenant un oauth_refresh_token legacy.
func seedSyncMetaRT(t *testing.T, dbPath, rt string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		t.Fatal(err)
	}
	db, err := duckdb.OpenReadWrite(dbPath)
	if err != nil {
		t.Fatalf("create player db: %v", err)
	}
	defer db.Close()
	ctx := context.Background()
	if _, err := db.Exec(ctx, "CREATE TABLE IF NOT EXISTS sync_meta (key VARCHAR, value VARCHAR)"); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(ctx,
		"INSERT INTO sync_meta (key, value) VALUES ('oauth_refresh_token', ?)", rt); err != nil {
		t.Fatal(err)
	}
}

// TestDiscoveryScan_SyncMetaResidue_NotAdoptedWhenStoreCovers : le store
// fournit déjà le RT → le résidu sync_meta est ignoré, le label ne contient
// pas duckdb_oauth (= pas de warn « à migrer » mensonger).
func TestDiscoveryScan_SyncMetaResidue_NotAdoptedWhenStoreCovers(t *testing.T) {
	cfg := fakeConfigWithPlayers(t, "Alice", "1111")
	resolver := titlePkg.NewPathResolver(cfg.RepoRoot)
	seedSyncMetaRT(t, resolver.PlayerDBPath(titlePkg.DefaultSlug, "Alice"), "rt-legacy-STALE")

	storeDir := filepath.Join(cfg.RepoRoot, "data", "auth", "watcher_tokens")
	if err := os.MkdirAll(storeDir, 0o700); err != nil {
		t.Fatal(err)
	}
	store := auth.NewMultiUserTokenStore(storeDir)
	if err := store.Upsert(&auth.UserTokens{
		XUID: "1111", Gamertag: "Alice", OAuthRefreshToken: "rt-store-FRESH",
	}); err != nil {
		t.Fatal(err)
	}

	counterName := "legacy_source_used_" + credSourceDuckDBOAuth
	before := observability.LoadCounter(counterName)
	d := NewDiscoveryWithStores(cfg, resolver, titlePkg.DefaultSlug, store, nil)
	sources, err := d.Scan(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(sources) != 1 {
		t.Fatalf("sources = %d, want 1", len(sources))
	}
	src := sources[0]
	if src.RefreshToken != "rt-store-FRESH" {
		t.Errorf("RefreshToken = %q, want rt-store-FRESH", src.RefreshToken)
	}
	if strings.Contains(src.Source, credSourceDuckDBOAuth) {
		t.Errorf("Source = %q : duckdb_oauth ne doit pas apparaître quand le store couvre le RT (résidu non adopté)", src.Source)
	}
	// D1a : résidu non adopté → le compteur legacy_source_used ne bouge PAS.
	if after := observability.LoadCounter(counterName); after != before {
		t.Errorf("compteur %s = %d, attendu %d (résidu non adopté → aucun comptage)", counterName, after, before)
	}
}

// TestDiscoveryScan_SyncMetaAdopted_WhenStoreEmpty : sans store, la valeur
// sync_meta est réellement adoptée → label duckdb_oauth (et warn légitime).
func TestDiscoveryScan_SyncMetaAdopted_WhenStoreEmpty(t *testing.T) {
	cfg := fakeConfigWithPlayers(t, "Alice", "1111")
	resolver := titlePkg.NewPathResolver(cfg.RepoRoot)
	seedSyncMetaRT(t, resolver.PlayerDBPath(titlePkg.DefaultSlug, "Alice"), "rt-legacy")

	counterName := "legacy_source_used_" + credSourceDuckDBOAuth
	before := observability.LoadCounter(counterName)
	d := NewDiscoveryWithStores(cfg, resolver, titlePkg.DefaultSlug, nil, nil)
	sources, err := d.Scan(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(sources) != 1 {
		t.Fatalf("sources = %d, want 1", len(sources))
	}
	src := sources[0]
	if src.RefreshToken != "rt-legacy" {
		t.Errorf("RefreshToken = %q, want rt-legacy", src.RefreshToken)
	}
	if !strings.Contains(src.Source, credSourceDuckDBOAuth) {
		t.Errorf("Source = %q : duckdb_oauth attendu quand la valeur legacy est adoptée", src.Source)
	}
	// D1a : adoption réelle du sync_meta legacy → compteur legacy_source_used +1.
	if after := observability.LoadCounter(counterName); after != before+1 {
		t.Errorf("compteur %s = %d, attendu %d (adoption legacy → +1)", counterName, after, before+1)
	}
}
