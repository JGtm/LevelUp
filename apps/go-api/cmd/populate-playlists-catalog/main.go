// cmd/populate-playlists-catalog — CLI bootstrap du catalogue Playlists/Pairs/Maps.
//
// Phase G du plan PLAN_PLAYLISTS_CATALOG.md.
//
// Workflow :
//  1. SELECT DISTINCT playlist_id, pair_id, map_id, game_variant_id depuis
//     shared.match_registry → INSERT OR IGNORE dans catalog_fetch_queue.
//  2. Lance CatalogFetcherService.Drain() pour vider la queue via DiscoveryUGC.
//  3. Affiche les compteurs finaux (X playlists, Y pairs, Z maps, W variants, E erreurs).
//
// Usage :
//
//	./populate-playlists-catalog \
//	    --metadata-db data/titles/halo_infinite/warehouse/metadata.duckdb \
//	    --shared-db   data/titles/halo_infinite/warehouse/shared_matches_v2.duckdb \
//	    --player-db   data/titles/halo_infinite/players/JGtm/stats.duckdb \
//	    --reset-attempts
package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/games/halo_infinite"
	"levelup/go-api/internal/migration"
	"levelup/go-api/internal/ops"
	"levelup/go-api/internal/platform/auth"
	"levelup/go-api/internal/platform/duckdb"
	"levelup/go-api/internal/platform/halo"
	"levelup/go-api/internal/service"
)

func main() {
	titleSlug := flag.String("title", "halo_infinite", "title slug à bootstrapper")
	metadataDBPath := flag.String("metadata-db", "data/titles/halo_infinite/warehouse/metadata.duckdb", "chemin metadata.duckdb")
	sharedDBPath := flag.String("shared-db", "data/titles/halo_infinite/warehouse/shared_matches_v2.duckdb", "chemin shared_matches_v2.duckdb")
	rulesPath := flag.String("experience-rules", "config/titles/halo_infinite/catalog/experience_rules.toml", "chemin experience_rules.toml")
	dryRun := flag.Bool("dry-run", false, "ne fait que le seed de la queue, pas de fetch DiscoveryUGC")
	rateLimit := flag.Int("rate-limit", 60, "requêtes max par minute vers Discovery UGC (défaut 60, use 300+ pour batch)")
	authFile := flag.String("auth-file", "data/auth/watcher_tokens.json", "chemin tokens.json pour accès XSTS/OAuth")
	playerDB := flag.String("player-db", "", "chemin stats.duckdb d'un joueur (lecture du oauth_refresh_token depuis sync_meta)")
	envFile := flag.String("env-file", ".env.local", "chemin .env.local (SPNKR_OAUTH_REFRESH_TOKEN_* et SPNKR_AZURE_CLIENT_ID)")
	fromMatchRegistry := flag.Bool("from-match-registry", false, "peuple les tables catalog directement depuis match_registry (bypass Discovery UGC API)")
	flag.Parse()

	// Charger .env.local pour exposer SPNKR_OAUTH_REFRESH_TOKEN_* et SPNKR_AZURE_CLIENT_ID.
	loadEnvLocal(*envFile)

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))
	slog.SetDefault(logger)

	ctx := context.Background()

	metadataDB, err := sql.Open("duckdb", *metadataDBPath)
	if err != nil {
		fatal("open metadata DB: %v", err)
	}
	defer metadataDB.Close()

	sharedDB, err := sql.Open("duckdb", *sharedDBPath+"?access_mode=READ_ONLY")
	if err != nil {
		fatal("open shared DB: %v", err)
	}
	defer sharedDB.Close()

	// Migrations metadata pour s'assurer que les tables catalogue existent.
	if err := migration.RunForDB(metadataDB, migration.TargetMetadata); err != nil {
		fatal("migrate metadata: %v", err)
	}

	// 1. Seed la queue depuis match_registry (logique partagée avec l'action
	// admin POST /admin/actions/catalog/ugc-drain).
	seeded, err := ops.SeedCatalogQueueFromRegistry(ctx, metadataDB, sharedDB, *titleSlug)
	if err != nil {
		fatal("seed queue: %v", err)
	}
	slog.InfoContext(ctx, "queue seeded from match_registry",
		"playlists", seeded.Playlists,
		"pairs", seeded.Pairs,
		"maps", seeded.Maps,
		"game_variants", seeded.GameVariants)

	if *dryRun {
		slog.InfoContext(ctx, "dry-run: skip drain")
		return
	}

	// 2. Mode --from-match-registry : seed direct depuis match_registry, sans appel HTTP.
	if *fromMatchRegistry {
		slog.InfoContext(ctx, "from-match-registry: seed catalog tables depuis match_registry")
		catalogResult, err := ops.CatalogRefreshFromRegistry(ctx, metadataDB, sharedDB, *titleSlug)
		if err != nil {
			fatal("populate catalog from match_registry: %v", err)
		}
		slog.InfoContext(ctx, "catalog seedé depuis match_registry",
			"playlists", catalogResult.Playlists,
			"pairs", catalogResult.Pairs,
			"maps", catalogResult.Maps,
			"game_variants", catalogResult.GameVariants)
		return
	}

	// 2. Charger les tokens Halo (requis par Discovery UGC).
	var haloTokens *domain.HaloTokens
	haloTokens, err = loadHaloTokens(ctx, *authFile, *playerDB)
	if err != nil {
		slog.WarnContext(ctx, "impossible de charger les tokens Halo — requêtes Discovery UGC pourraient échouer (401)",
			"err", err)
	} else {
		slog.InfoContext(ctx, "tokens Halo chargés")
	}

	// 3. Construire le resolver + adapter Halo.
	provider := halo.NewHaloProvider().WithRateLimit(*rateLimit).WithTokens(haloTokens)
	fetcher := halo.NewCatalogFetcher(provider)
	adapter, err := halo_infinite.NewCatalogAdapter(fetcher, *rulesPath)
	if err != nil {
		fatal("init catalog adapter: %v", err)
	}
	resolver := games.NewStaticResolver(*titleSlug)
	resolver.RegisterCatalog(adapter)

	// 4. Lance le drain (loop jusqu'à plus aucune nouvelle résolution).
	svc := service.NewCatalogFetcherService(duckdb.NewCatalogWriter(metadataDB), resolver)
	t0 := time.Now()
	totalPlaylists, totalPairs, totalMaps, totalGV, totalErrors := 0, 0, 0, 0, 0
	for pass := 1; ; pass++ {
		res, err := svc.Drain(ctx, *titleSlug)
		if err != nil {
			fatal("drain pass %d: %v", pass, err)
		}
		if res.Playlists+res.Pairs+res.Maps+res.GameVariants == 0 {
			break // plus aucune nouvelle résolution (reste = non résolvables)
		}
		totalPlaylists += res.Playlists
		totalPairs += res.Pairs
		totalMaps += res.Maps
		totalGV += res.GameVariants
		totalErrors += res.Errors
		slog.InfoContext(ctx, "drain pass complete",
			"pass", pass,
			"playlists", res.Playlists, "pairs", res.Pairs,
			"maps", res.Maps, "game_variants", res.GameVariants,
			"errors", res.Errors)
		if pass > 10 {
			slog.WarnContext(ctx, "drain: too many passes, stopping")
			break
		}
	}

	slog.InfoContext(ctx, "bootstrap complete",
		"duration", time.Since(t0).String(),
		"playlists", totalPlaylists, "pairs", totalPairs,
		"maps", totalMaps, "game_variants", totalGV,
		"errors", totalErrors)
}

// loadHaloTokens charge les tokens Halo pour les appels Discovery UGC.
//
// Ordre de résolution :
//  1. XSTS token depuis authFile (watcher_tokens.json) si encore valide → échange → Spartan
//  2. OAuth access_token depuis authFile si encore valide → échange complet
//  3. oauth_refresh_token depuis playerDBPath (sync_meta) → refresh → échange complet
func loadHaloTokens(ctx context.Context, authFile, playerDBPath string) (*domain.HaloTokens, error) {
	const margin = 5 * time.Minute

	// Chemin 1 & 2 : watcher_tokens.json
	store := auth.NewTokenStore(authFile)
	if stored, err := store.Load(); err == nil {
		if stored.IsXSTSValid(margin) {
			slog.DebugContext(ctx, "loadHaloTokens: XSTS valide → Spartan token")
			if tokens, err := auth.ExchangeXSTSForHaloTokens(ctx, stored.XSTSToken); err == nil {
				return tokens, nil
			}
		}
		if stored.IsOAuthValid(margin) {
			slog.DebugContext(ctx, "loadHaloTokens: OAuth access_token valide → échange complet")
			if result, err := auth.ExchangeAccessToken(ctx, stored.AccessToken); err == nil {
				return result.Tokens, nil
			}
		}
	}

	// Chemin 3 : oauth_refresh_token depuis player DuckDB
	if playerDBPath != "" {
		if tokens, err := loadTokensFromPlayerDB(ctx, playerDBPath); err == nil {
			return tokens, nil
		} else {
			slog.DebugContext(ctx, "loadHaloTokens: player DB auth échoué", "err", err)
		}
	}

	return nil, fmt.Errorf("aucun token valide trouvé (auth-file=%s, player-db=%s)", authFile, playerDBPath)
}

// loadTokensFromPlayerDB tente d'obtenir un access_token Microsoft depuis sync_meta
// du player stats.duckdb (MSAL cache ou oauth_refresh_token), puis l'échange pour des tokens Halo.
// Même logique que warm_bp_assets/main.go:readAccessTokenFromProfiles.
func loadTokensFromPlayerDB(ctx context.Context, dbPath string) (*domain.HaloTokens, error) {
	db, err := sql.Open("duckdb", dbPath+"?access_mode=READ_ONLY")
	if err != nil {
		return nil, fmt.Errorf("open player DB: %w", err)
	}
	db.SetMaxOpenConns(1)
	defer db.Close()

	var cacheJSON, refreshToken string
	_ = db.QueryRowContext(ctx, `SELECT value FROM sync_meta WHERE key = 'msal_token_cache'`).Scan(&cacheJSON)
	_ = db.QueryRowContext(ctx, `SELECT value FROM sync_meta WHERE key = 'oauth_refresh_token'`).Scan(&refreshToken)

	provider := auth.NewSISUProvider()

	// Extraire le gamertag depuis le chemin pour vérifier l'env var.
	gamertag := extractGamertag(dbPath)

	var accessToken string
	if cacheJSON != "" {
		if tok, err := provider.TrySilentRefresh(ctx, cacheJSON); err == nil && tok != "" {
			slog.DebugContext(ctx, "loadTokensFromPlayerDB: MSAL silent refresh OK")
			accessToken = tok
		}
	}
	if accessToken == "" && refreshToken != "" {
		if tok, err := provider.TryOAuthRefresh(ctx, refreshToken); err == nil && tok != "" {
			slog.DebugContext(ctx, "loadTokensFromPlayerDB: OAuth v2 refresh OK")
			accessToken = tok
		}
	}
	// Fallback : SPNKR_OAUTH_REFRESH_TOKEN_<GAMERTAG> depuis .env.local / environnement.
	if accessToken == "" && gamertag != "" {
		envKey := "SPNKR_OAUTH_REFRESH_TOKEN_" + strings.ToUpper(gamertag)
		if envRT := os.Getenv(envKey); envRT != "" {
			if tok, err := provider.TryOAuthRefresh(ctx, envRT); err == nil && tok != "" {
				slog.DebugContext(ctx, "loadTokensFromPlayerDB: env var OAuth refresh OK", "key", envKey)
				accessToken = tok
			}
		}
	}
	if accessToken == "" {
		return nil, fmt.Errorf("aucun access_token obtenu depuis player DB (msal_cache=%v oauth_rt=%v gamertag=%q)",
			cacheJSON != "", refreshToken != "", gamertag)
	}

	result, err := auth.ExchangeAccessToken(ctx, accessToken)
	if err != nil {
		return nil, fmt.Errorf("access_token → Halo tokens: %w", err)
	}
	return result.Tokens, nil
}

// extractGamertag extrait le nom du dossier joueur depuis un chemin stats.duckdb.
// Ex: "data/titles/halo_infinite/players/JGtm/stats.duckdb" → "JGtm"
func extractGamertag(dbPath string) string {
	parts := strings.Split(strings.ReplaceAll(dbPath, "\\", "/"), "/")
	for i, part := range parts {
		if part == "players" && i+1 < len(parts) {
			return parts[i+1]
		}
	}
	return ""
}

// loadEnvLocal lit un fichier .env.local et injecte les variables dans l'environnement
// du processus, sans écraser les variables déjà définies.
func loadEnvLocal(path string) {
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		idx := strings.IndexByte(line, '=')
		if idx <= 0 {
			continue
		}
		key := strings.TrimSpace(line[:idx])
		val := strings.TrimSpace(line[idx+1:])
		if len(val) >= 2 && ((val[0] == '"' && val[len(val)-1] == '"') || (val[0] == '\'' && val[len(val)-1] == '\'')) {
			val = val[1 : len(val)-1]
		}
		if os.Getenv(key) == "" {
			_ = os.Setenv(key, val)
		}
	}
}

func fatal(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "FATAL: "+format+"\n", args...)
	os.Exit(1)
}

// =============================================================================
// Seed direct depuis match_registry (bypass Discovery UGC API)
// =============================================================================
