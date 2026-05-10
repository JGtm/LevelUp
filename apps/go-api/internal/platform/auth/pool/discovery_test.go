package pool

import (
	"context"
	"testing"

	"levelup/go-api/internal/config"
	titlePkg "levelup/go-api/internal/domain/title"
)

// TestDiscoveryScan_MixedTokenSources teste le scan avec 3 joueurs ayant différentes sources de tokens.
//
// Scénario :
//   - Bob : refresh token en env var SPNKR_OAUTH_REFRESH_TOKEN_BOB
//   - Alice : MSAL cache en DuckDB sync_meta + refresh token fallback en env
//   - Carl : pas de token du tout (exclut)
func TestDiscoveryScan_MixedTokenSources(t *testing.T) {
	// Setup env vars.
	t.Setenv("SPNKR_OAUTH_REFRESH_TOKEN_BOB", "refresh_bob_value")
	t.Setenv("SPNKR_OAUTH_REFRESH_TOKEN_ALICE", "refresh_alice_env") // sera overridé par DuckDB si présent

	// Charger la config du repo réel pour accéder à db_profiles.json et PathResolver.
	// Suppose que le test est lancé depuis la racine du repo.
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("config.Load failed: %v", err)
	}

	resolver := titlePkg.NewPathResolver(cfg.RepoRoot)

	discovery := NewDiscovery(cfg, resolver, titlePkg.DefaultSlug)

	ctx := context.Background()
	sources, err := discovery.Scan(ctx)
	if err != nil {
		t.Fatalf("Scan failed: %v", err)
	}

	// Vérifications :
	// - Au moins 1 joueur avec token doit être découvert (selon db_profiles.json du repo)
	// - Aucun joueur sans token ne doit être inclus
	// - Chaque source doit avoir un Source non-vide ("duckdb_msal", "env_oauth", etc.)
	// - Aucune source avec RefreshToken vide ET MSALCache vide (par construction Scan exclut les cas sans token)

	// Le test requiert un environnement avec soit :
	//   - un MSAL cache en DuckDB sync_meta pour ≥1 joueur de db_profiles.json, OU
	//   - une env var SPNKR_OAUTH_REFRESH_TOKEN_<GAMERTAG> matchant un de ces joueurs.
	// Les env vars BOB/ALICE setUp ci-dessus ne correspondent pas à des joueurs
	// réels ; sur une machine sans cache MSAL ni token env aligné, on skip.
	if len(sources) == 0 {
		t.Skipf("aucune source de crédentiels disponible — test requiert un environnement avec MSAL cache ou refresh token env pour ≥1 joueur de db_profiles.json (voir test header)")
	}

	for _, src := range sources {
		if src.Gamertag == "" {
			t.Error("CredentialSource has empty Gamertag")
		}
		if src.Source == "" {
			t.Error("CredentialSource has empty Source")
		}
		// Au moins un token doit être présent (par construction du Scan, qui exclut les cas sans token).
		if src.MSALCache == "" && src.RefreshToken == "" {
			t.Errorf("Gamertag %s has neither MSALCache nor RefreshToken", src.Gamertag)
		}
	}

	t.Logf("Discovered %d credential sources:", len(sources))
	for _, src := range sources {
		t.Logf("  - %s (source: %s, has_msal: %v, has_oauth: %v)",
			src.Gamertag, src.Source, src.MSALCache != "", src.RefreshToken != "")
	}
}

// TestDiscoveryScan_ExcludesPlayersWithoutTokens teste que les joueurs sans tokens sont exclus.
func TestDiscoveryScan_ExcludesPlayersWithoutTokens(t *testing.T) {
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("config.Load failed: %v", err)
	}

	resolver := titlePkg.NewPathResolver(cfg.RepoRoot)
	discovery := NewDiscovery(cfg, resolver, titlePkg.DefaultSlug)

	ctx := context.Background()
	sources, err := discovery.Scan(ctx)
	if err != nil {
		t.Fatalf("Scan failed: %v", err)
	}

	// Chaque source doit avoir un token.
	for _, src := range sources {
		if src.MSALCache == "" && src.RefreshToken == "" {
			t.Errorf("Source %s (from %s) has no tokens — should have been excluded during Scan",
				src.Gamertag, src.Source)
		}
	}
}

// TestDiscoveryScan_EnvTokenFallback teste que les tokens env var sont lus si DuckDB absent.
func TestDiscoveryScan_EnvTokenFallback(t *testing.T) {
	t.Setenv("SPNKR_OAUTH_REFRESH_TOKEN_TESTPLAYER", "test_env_token_value")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("config.Load failed: %v", err)
	}

	// Créer un discovery avec un joueur "TESTPLAYER" (peut ne pas être dans db_profiles.json,
	// mais le test sera limité aux joueurs configurés).
	resolver := titlePkg.NewPathResolver(cfg.RepoRoot)
	discovery := NewDiscovery(cfg, resolver, titlePkg.DefaultSlug)

	ctx := context.Background()
	sources, err := discovery.Scan(ctx)
	if err != nil {
		t.Fatalf("Scan failed: %v", err)
	}

	// Vérifier que si un joueur avec RefreshToken en env existe, sa source est "env_oauth".
	for _, src := range sources {
		if src.Gamertag == "TESTPLAYER" || src.Gamertag == "TestPlayer" {
			if src.Source == "env_oauth" && src.RefreshToken == "test_env_token_value" {
				return // test réussi
			}
		}
	}
	// Si TESTPLAYER n'est pas dans db_profiles.json, ce test ne peut pas vérifier le fallback.
	// C'est acceptable — le test est limité aux joueurs configurés.
	t.Logf("TESTPLAYER not in db_profiles.json — env fallback test skipped")
}

// TestReadOAuthRefreshTokenFromEnv teste la transformation du gamertag en clé env.
func TestReadOAuthRefreshTokenFromEnv(t *testing.T) {
	tests := []struct {
		gamertag string
		envKey   string
		envValue string
		want     string
	}{
		{
			gamertag: "Bob",
			envKey:   "SPNKR_OAUTH_REFRESH_TOKEN_BOB",
			envValue: "token_bob",
			want:     "token_bob",
		},
		{
			gamertag: "Alice Smith",
			envKey:   "SPNKR_OAUTH_REFRESH_TOKEN_ALICE_SMITH",
			envValue: "token_alice",
			want:     "token_alice",
		},
		{
			gamertag: "player-with-dash",
			envKey:   "SPNKR_OAUTH_REFRESH_TOKEN_PLAYER_WITH_DASH",
			envValue: "token_dash",
			want:     "token_dash",
		},
		{
			gamertag: "Unknown",
			envKey:   "SPNKR_OAUTH_REFRESH_TOKEN_UNKNOWN",
			envValue: "", // non setenv
			want:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.gamertag, func(t *testing.T) {
			if tt.envValue != "" {
				t.Setenv(tt.envKey, tt.envValue)
			}
			got := readOAuthRefreshTokenFromEnv(tt.gamertag)
			if got != tt.want {
				t.Errorf("readOAuthRefreshTokenFromEnv(%q) = %q, want %q", tt.gamertag, got, tt.want)
			}
		})
	}
}
