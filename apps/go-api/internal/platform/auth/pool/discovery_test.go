package pool

import (
	"context"
	"testing"

	"levelup/go-api/internal/config"
	titlePkg "levelup/go-api/internal/domain/title"
)

// TestDiscoveryScan_NoStore_DiscoversNothing : depuis ADR 0023 Phase 5, le
// MultiUserTokenStore est la SEULE source de credentials. Un Discovery construit
// sans store (NewDiscovery) ne peut donc rien découvrir — même si des env vars
// SPNKR_OAUTH_REFRESH_TOKEN_* ou des résidus sync_meta existent.
func TestDiscoveryScan_NoStore_DiscoversNothing(t *testing.T) {
	t.Setenv("SPNKR_OAUTH_REFRESH_TOKEN_BOB", "refresh_bob_value")
	t.Setenv("SPNKR_OAUTH_REFRESH_TOKEN_ALICE", "refresh_alice_env")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("config.Load failed: %v", err)
	}
	resolver := titlePkg.NewPathResolver(cfg.RepoRoot)

	sources, err := NewDiscovery(cfg, resolver, titlePkg.DefaultSlug).Scan(context.Background())
	if err != nil {
		t.Fatalf("Scan failed: %v", err)
	}
	if len(sources) != 0 {
		t.Errorf("sources = %d, want 0 (aucun store attaché → aucune source legacy possible)", len(sources))
	}
}

// TestDiscoveryScan_SourcesAreStoreOnly : toute source découverte porte un
// refresh token et le label unique watcher_oauth (garde-rail de forme).
func TestDiscoveryScan_SourcesAreStoreOnly(t *testing.T) {
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("config.Load failed: %v", err)
	}
	resolver := titlePkg.NewPathResolver(cfg.RepoRoot)

	sources, err := NewDiscovery(cfg, resolver, titlePkg.DefaultSlug).Scan(context.Background())
	if err != nil {
		t.Fatalf("Scan failed: %v", err)
	}
	for _, src := range sources {
		if src.Gamertag == "" {
			t.Error("CredentialSource has empty Gamertag")
		}
		if src.RefreshToken == "" {
			t.Errorf("Gamertag %s sans refresh token — aurait dû être exclu du scan", src.Gamertag)
		}
		if src.Source != credSourceWatcherOAuth {
			t.Errorf("Gamertag %s : Source = %q, want %q", src.Gamertag, src.Source, credSourceWatcherOAuth)
		}
	}
}
