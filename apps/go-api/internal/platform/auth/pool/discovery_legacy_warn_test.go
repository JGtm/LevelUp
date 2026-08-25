// Package pool — discovery_legacy_warn_test.go : RATCHET ADR 0023 Phase 5.
//
// Historique : le scan adoptait un oauth_refresh_token résiduel de sync_meta
// quand le store ne couvrait pas le joueur (et loguait « legacy sync_meta
// utilisée »). Depuis le 2026-08-25, cette source n'existe plus — ce fichier
// vérifie qu'un résidu sync_meta n'est JAMAIS adopté, store rempli OU vide.
// Toute réintroduction du fallback fait échouer ces tests.
//
// Build tag cgo : crée une vraie player DB DuckDB portant le résidu.
//
//go:build cgo

package pool

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	titlePkg "levelup/go-api/internal/domain/title"
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
// fournit le RT → le résidu sync_meta est ignoré.
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

	d := NewDiscoveryWithStore(cfg, resolver, titlePkg.DefaultSlug, store)
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
	if src.Source != credSourceWatcherOAuth {
		t.Errorf("Source = %q, want %q (source unique ADR 0023)", src.Source, credSourceWatcherOAuth)
	}
}

// TestDiscoveryScan_SyncMetaNeverAdopted_WhenStoreEmpty : RATCHET Phase 5 —
// store vide + résidu sync_meta présent → le joueur est EXCLU (le fallback
// legacy n'existe plus). C'est la non-régression du retrait.
func TestDiscoveryScan_SyncMetaNeverAdopted_WhenStoreEmpty(t *testing.T) {
	cfg := fakeConfigWithPlayers(t, "Alice", "1111")
	resolver := titlePkg.NewPathResolver(cfg.RepoRoot)
	seedSyncMetaRT(t, resolver.PlayerDBPath(titlePkg.DefaultSlug, "Alice"), "rt-legacy")

	storeDir := filepath.Join(cfg.RepoRoot, "data", "auth", "watcher_tokens")
	store := auth.NewMultiUserTokenStore(storeDir) // aucune entrée

	d := NewDiscoveryWithStore(cfg, resolver, titlePkg.DefaultSlug, store)
	sources, err := d.Scan(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(sources) != 0 {
		t.Fatalf("sources = %d (%+v), want 0 — le résidu sync_meta ne doit JAMAIS être adopté (ADR 0023 Phase 5)",
			len(sources), sources)
	}
}

// TestDiscoveryScan_EnvVarNeverAdopted : RATCHET Phase 5 — une env var
// SPNKR_OAUTH_REFRESH_TOKEN_* présente ne doit plus jamais peupler le pool.
func TestDiscoveryScan_EnvVarNeverAdopted(t *testing.T) {
	cfg := fakeConfigWithPlayers(t, "Alice", "1111")
	resolver := titlePkg.NewPathResolver(cfg.RepoRoot)
	t.Setenv("SPNKR_OAUTH_REFRESH_TOKEN_ALICE", "rt-from-env")

	storeDir := filepath.Join(cfg.RepoRoot, "data", "auth", "watcher_tokens")
	store := auth.NewMultiUserTokenStore(storeDir) // aucune entrée

	d := NewDiscoveryWithStore(cfg, resolver, titlePkg.DefaultSlug, store)
	sources, err := d.Scan(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(sources) != 0 {
		t.Fatalf("sources = %d (%+v), want 0 — l'env var n'est plus une source de credentials", len(sources), sources)
	}
}
