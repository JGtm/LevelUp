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
	"levelup/go-api/internal/platform/auth"
	"levelup/go-api/internal/platform/halo"
	"levelup/go-api/internal/service"
)

func main() {
	titleSlug := flag.String("title", "halo_infinite", "title slug à bootstrapper")
	metadataDBPath := flag.String("metadata-db", "data/titles/halo_infinite/warehouse/metadata.duckdb", "chemin metadata.duckdb")
	sharedDBPath := flag.String("shared-db", "data/titles/halo_infinite/warehouse/shared_matches_v2.duckdb", "chemin shared_matches_v2.duckdb")
	rulesPath := flag.String("experience-rules", "config/titles/halo_infinite/catalog/experience_rules.toml", "chemin experience_rules.toml")
	maxRetries := flag.Int("max-retries", 5, "nombre max de tentatives par asset avant skip")
	dryRun := flag.Bool("dry-run", false, "ne fait que le seed de la queue, pas de fetch DiscoveryUGC")
	rateLimit := flag.Int("rate-limit", 60, "requêtes max par minute vers Discovery UGC (défaut 60, use 300+ pour batch)")
	authFile := flag.String("auth-file", "data/auth/watcher_tokens.json", "chemin tokens.json pour accès XSTS/OAuth")
	playerDB := flag.String("player-db", "", "chemin stats.duckdb d'un joueur (lecture du oauth_refresh_token depuis sync_meta)")
	resetAttempts := flag.Bool("reset-attempts", false, "remet attempts=0 sur toutes les entrées avant de drainer")
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

	// 1. Seed la queue depuis match_registry.
	seeded, err := seedQueueFromMatchRegistry(ctx, metadataDB, sharedDB, *titleSlug)
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
		catalogResult, err := populateCatalogFromMatchRegistry(ctx, metadataDB, sharedDB, *titleSlug)
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

	// 1b. Reset attempts si demandé (après un run précédent avec erreurs 401).
	if *resetAttempts {
		res, err := metadataDB.ExecContext(ctx,
			`UPDATE catalog_fetch_queue SET attempts = 0, last_error = NULL WHERE title_slug = ?`,
			*titleSlug,
		)
		if err != nil {
			fatal("reset attempts: %v", err)
		}
		n, _ := res.RowsAffected()
		slog.InfoContext(ctx, "attempts reset", "rows", n)
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

	// 4. Lance le drain (loop jusqu'à queue vide ou tous skipped).
	svc := service.NewCatalogFetcherService(metadataDB, resolver, *maxRetries)
	t0 := time.Now()
	totalPlaylists, totalPairs, totalMaps, totalGV, totalErrors := 0, 0, 0, 0, 0
	for pass := 1; ; pass++ {
		res, err := svc.Drain(ctx, *titleSlug)
		if err != nil {
			fatal("drain pass %d: %v", pass, err)
		}
		if res.Playlists+res.Pairs+res.Maps+res.GameVariants+res.Errors == 0 {
			break // queue vide ou tous en max-retries
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

	provider := auth.NewMSALProvider()

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

type seedCounters struct {
	Playlists, Pairs, Maps, GameVariants int
}

// seedQueueFromMatchRegistry insère dans catalog_fetch_queue tous les asset IDs
// distincts vus dans shared.match_registry, sans appel HTTP.
func seedQueueFromMatchRegistry(ctx context.Context, metadataDB, sharedDB *sql.DB, titleSlug string) (seedCounters, error) {
	var counters seedCounters
	type seedSpec struct {
		assetType  string
		col        string
		versionCol string
		counter    *int
	}
	specs := []seedSpec{
		{"playlist", "playlist_id", "playlist_version_id", &counters.Playlists},
		{"pair", "pair_id", "pair_version_id", &counters.Pairs},
		{"map", "map_id", "map_version_id", &counters.Maps},
		{"game_variant", "game_variant_id", "game_variant_version_id", &counters.GameVariants},
	}
	for _, s := range specs {
		// Essai avec version_id — fallback sans si la colonne n'existe pas encore
		// (migration add_match_registry_version_ids non encore appliquée).
		query := fmt.Sprintf(
			`SELECT DISTINCT %s, COALESCE(%s, '') FROM match_registry WHERE %s IS NOT NULL AND %s != ''`,
			s.col, s.versionCol, s.col, s.col,
		)
		rows, err := sharedDB.QueryContext(ctx, query)
		if err != nil {
			// Colonne version_id absente → fallback : seed sans version_id
			slog.WarnContext(ctx, "seed: version_id column absent, falling back to id-only seed",
				"asset_type", s.assetType, "err", err)
			query = fmt.Sprintf(
				`SELECT DISTINCT %s, '' FROM match_registry WHERE %s IS NOT NULL AND %s != ''`,
				s.col, s.col, s.col,
			)
			rows, err = sharedDB.QueryContext(ctx, query)
			if err != nil {
				return counters, fmt.Errorf("select %s (fallback): %w", s.assetType, err)
			}
		}
		var inserted int
		for rows.Next() {
			var id, ver string
			if err := rows.Scan(&id, &ver); err != nil {
				rows.Close()
				return counters, err
			}
			res, err := metadataDB.ExecContext(ctx,
				`INSERT OR IGNORE INTO catalog_fetch_queue (title_slug, asset_type, asset_id, version_id) VALUES (?, ?, ?, ?)`,
				titleSlug, s.assetType, id, ver)
			if err != nil {
				slog.WarnContext(ctx, "seed: insert queue failed", "err", err, "asset_type", s.assetType, "asset_id", id)
				continue
			}
			n, _ := res.RowsAffected()
			if n > 0 {
				inserted++
			}
		}
		rows.Close()
		*s.counter = inserted
	}
	return counters, nil
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

type catalogSeedResult struct {
	Playlists, Pairs, Maps, GameVariants int
}

// populateCatalogFromMatchRegistry peuple les tables catalog depuis match_registry
// sans appel HTTP. Les noms en/fr sont pris directement des colonnes de match_registry.
func populateCatalogFromMatchRegistry(ctx context.Context, metadataDB, sharedDB *sql.DB, titleSlug string) (catalogSeedResult, error) {
	var r catalogSeedResult

	// --- Playlists ---
	rows, err := sharedDB.QueryContext(ctx, `
		SELECT
			playlist_id,
			COALESCE(MAX(playlist_version_id) FILTER (WHERE playlist_version_id IS NOT NULL AND playlist_version_id != ''), '') AS version_id,
			FIRST(playlist_name ORDER BY start_time DESC NULLS LAST) AS name,
			FIRST(is_ranked   ORDER BY start_time DESC NULLS LAST) AS is_ranked,
			MIN(start_time) AS first_seen,
			MAX(start_time) AS last_seen
		FROM match_registry
		WHERE playlist_id IS NOT NULL AND playlist_id != '' AND playlist_name IS NOT NULL
		GROUP BY playlist_id
	`)
	if err != nil {
		return r, fmt.Errorf("query playlists: %w", err)
	}
	for rows.Next() {
		var id, versionID, name string
		var isRanked bool
		var firstSeen, lastSeen time.Time
		if err := rows.Scan(&id, &versionID, &name, &isRanked, &firstSeen, &lastSeen); err != nil {
			rows.Close()
			return r, fmt.Errorf("scan playlist: %w", err)
		}
		experience := classifyExperienceFromName(name, isRanked)
		_, err := metadataDB.ExecContext(ctx, `
			INSERT INTO playlists_catalog
				(title_slug, playlist_asset_id, current_version_id, name_canonical, experience, is_ranked, is_active, first_seen_at, last_seen_at, last_fetched_at)
			VALUES (?, ?, ?, ?, ?, ?, TRUE, ?, ?, CURRENT_TIMESTAMP)
			ON CONFLICT (title_slug, playlist_asset_id) DO UPDATE SET
				current_version_id = CASE WHEN excluded.current_version_id != '' THEN excluded.current_version_id ELSE current_version_id END,
				name_canonical     = excluded.name_canonical,
				experience         = excluded.experience,
				is_ranked          = excluded.is_ranked,
				last_seen_at       = excluded.last_seen_at,
				last_fetched_at    = excluded.last_fetched_at
		`, titleSlug, id, versionID, name, experience, isRanked, firstSeen, lastSeen)
		if err != nil {
			slog.WarnContext(ctx, "upsert playlist", "id", id, "err", err)
			continue
		}
		r.Playlists++
	}
	rows.Close()

	// --- Maps ---
	rows, err = sharedDB.QueryContext(ctx, `
		SELECT
			map_id,
			COALESCE(MAX(map_version_id) FILTER (WHERE map_version_id IS NOT NULL AND map_version_id != ''), '') AS version_id,
			FIRST(map_name ORDER BY start_time DESC NULLS LAST) AS name
		FROM match_registry
		WHERE map_id IS NOT NULL AND map_id != '' AND map_name IS NOT NULL
		GROUP BY map_id
	`)
	if err != nil {
		return r, fmt.Errorf("query maps: %w", err)
	}
	for rows.Next() {
		var id, versionID, name string
		if err := rows.Scan(&id, &versionID, &name); err != nil {
			rows.Close()
			return r, fmt.Errorf("scan map: %w", err)
		}
		_, err := metadataDB.ExecContext(ctx, `
			INSERT INTO maps_catalog
				(title_slug, map_asset_id, current_version_id, name_canonical, last_fetched_at)
			VALUES (?, ?, ?, ?, CURRENT_TIMESTAMP)
			ON CONFLICT (title_slug, map_asset_id) DO UPDATE SET
				current_version_id = CASE WHEN excluded.current_version_id != '' THEN excluded.current_version_id ELSE current_version_id END,
				name_canonical     = excluded.name_canonical,
				last_fetched_at    = excluded.last_fetched_at
		`, titleSlug, id, versionID, name)
		if err != nil {
			slog.WarnContext(ctx, "upsert map", "id", id, "err", err)
			continue
		}
		r.Maps++
	}
	rows.Close()

	// --- Game Variants ---
	rows, err = sharedDB.QueryContext(ctx, `
		SELECT
			game_variant_id,
			COALESCE(MAX(game_variant_version_id) FILTER (WHERE game_variant_version_id IS NOT NULL AND game_variant_version_id != ''), '') AS version_id,
			FIRST(game_variant_name ORDER BY start_time DESC NULLS LAST) AS name
		FROM match_registry
		WHERE game_variant_id IS NOT NULL AND game_variant_id != '' AND game_variant_name IS NOT NULL
		GROUP BY game_variant_id
	`)
	if err != nil {
		return r, fmt.Errorf("query game_variants: %w", err)
	}
	for rows.Next() {
		var id, versionID, name string
		if err := rows.Scan(&id, &versionID, &name); err != nil {
			rows.Close()
			return r, fmt.Errorf("scan game_variant: %w", err)
		}
		modeCanonical := classifyModeCanonicalFromName(name)
		_, err := metadataDB.ExecContext(ctx, `
			INSERT INTO game_variants_catalog
				(title_slug, game_variant_asset_id, current_version_id, name_canonical, mode_canonical, last_fetched_at)
			VALUES (?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
			ON CONFLICT (title_slug, game_variant_asset_id) DO UPDATE SET
				current_version_id = CASE WHEN excluded.current_version_id != '' THEN excluded.current_version_id ELSE current_version_id END,
				name_canonical     = excluded.name_canonical,
				mode_canonical     = excluded.mode_canonical,
				last_fetched_at    = excluded.last_fetched_at
		`, titleSlug, id, versionID, name, modeCanonical)
		if err != nil {
			slog.WarnContext(ctx, "upsert game_variant", "id", id, "err", err)
			continue
		}
		r.GameVariants++
	}
	rows.Close()

	// --- Pairs (+ pair_mode_label_translations) ---
	// Essai avec pair_name_fr ; fallback sans si la colonne n'existe pas.
	pairQuery := `
		SELECT
			pair_id,
			COALESCE(MAX(pair_version_id) FILTER (WHERE pair_version_id IS NOT NULL AND pair_version_id != ''), '') AS version_id,
			FIRST(pair_name    ORDER BY start_time DESC NULLS LAST) AS name,
			FIRST(pair_name_fr ORDER BY start_time DESC NULLS LAST) AS name_fr,
			FIRST(map_id           ORDER BY start_time DESC NULLS LAST) AS map_id,
			FIRST(game_variant_id  ORDER BY start_time DESC NULLS LAST) AS game_variant_id,
			FIRST(mode_category    ORDER BY start_time DESC NULLS LAST) AS mode_category
		FROM match_registry
		WHERE pair_id IS NOT NULL AND pair_id != '' AND pair_name IS NOT NULL
		GROUP BY pair_id`
	rows, err = sharedDB.QueryContext(ctx, pairQuery)
	hasFR := true
	if err != nil {
		// pair_name_fr absent → fallback sans cette colonne
		hasFR = false
		slog.WarnContext(ctx, "pair_name_fr absent, fallback sans FR", "err", err)
		pairQuery = `
			SELECT
				pair_id,
				COALESCE(MAX(pair_version_id) FILTER (WHERE pair_version_id IS NOT NULL AND pair_version_id != ''), '') AS version_id,
				FIRST(pair_name   ORDER BY start_time DESC NULLS LAST) AS name,
				FIRST(map_id          ORDER BY start_time DESC NULLS LAST) AS map_id,
				FIRST(game_variant_id ORDER BY start_time DESC NULLS LAST) AS game_variant_id,
				FIRST(mode_category   ORDER BY start_time DESC NULLS LAST) AS mode_category
			FROM match_registry
			WHERE pair_id IS NOT NULL AND pair_id != '' AND pair_name IS NOT NULL
			GROUP BY pair_id`
		rows, err = sharedDB.QueryContext(ctx, pairQuery)
		if err != nil {
			return r, fmt.Errorf("query pairs (fallback): %w", err)
		}
	}
	for rows.Next() {
		var id, versionID, name string
		var nameFR sql.NullString
		var mapID, gameVariantID, modeCategory sql.NullString
		var scanErr error
		if hasFR {
			scanErr = rows.Scan(&id, &versionID, &name, &nameFR, &mapID, &gameVariantID, &modeCategory)
		} else {
			scanErr = rows.Scan(&id, &versionID, &name, &mapID, &gameVariantID, &modeCategory)
		}
		if scanErr != nil {
			rows.Close()
			return r, fmt.Errorf("scan pair: %w", scanErr)
		}
		_, err := metadataDB.ExecContext(ctx, `
			INSERT INTO map_mode_pair_definitions
				(title_slug, pair_asset_id, current_version_id, name_canonical, map_asset_id, game_variant_asset_id, mode_category, last_fetched_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
			ON CONFLICT (title_slug, pair_asset_id) DO UPDATE SET
				current_version_id    = CASE WHEN excluded.current_version_id != '' THEN excluded.current_version_id ELSE current_version_id END,
				name_canonical        = excluded.name_canonical,
				map_asset_id          = CASE WHEN excluded.map_asset_id != '' THEN excluded.map_asset_id ELSE map_asset_id END,
				game_variant_asset_id = CASE WHEN excluded.game_variant_asset_id != '' THEN excluded.game_variant_asset_id ELSE game_variant_asset_id END,
				mode_category         = CASE WHEN excluded.mode_category != '' THEN excluded.mode_category ELSE mode_category END,
				last_fetched_at       = excluded.last_fetched_at
		`, titleSlug, id, versionID, name, mapID.String, gameVariantID.String, modeCategory.String)
		if err != nil {
			slog.WarnContext(ctx, "upsert pair", "id", id, "err", err)
			continue
		}
		r.Pairs++

		// pair_mode_label_translations — EN
		if name != "" {
			metadataDB.ExecContext(ctx, `
				INSERT INTO pair_mode_label_translations (title_slug, pair_asset_id, lang, label)
				VALUES (?, ?, 'en', ?)
				ON CONFLICT (title_slug, pair_asset_id, lang) DO UPDATE SET label = excluded.label
			`, titleSlug, id, name) //nolint:errcheck
		}
		// pair_mode_label_translations — FR
		if nameFR.Valid && nameFR.String != "" {
			metadataDB.ExecContext(ctx, `
				INSERT INTO pair_mode_label_translations (title_slug, pair_asset_id, lang, label)
				VALUES (?, ?, 'fr', ?)
				ON CONFLICT (title_slug, pair_asset_id, lang) DO UPDATE SET label = excluded.label
			`, titleSlug, id, nameFR.String) //nolint:errcheck
		}
	}
	rows.Close()

	return r, nil
}

// classifyExperienceFromName dérive l'expérience depuis le nom de playlist et is_ranked.
func classifyExperienceFromName(name string, isRanked bool) string {
	if isRanked {
		return "ranked"
	}
	lower := strings.ToLower(name)
	switch {
	case strings.Contains(lower, "big team") || strings.Contains(lower, "btb"):
		return "btb"
	case strings.Contains(lower, "firefight"):
		return "firefight"
	case strings.Contains(lower, "action sack"):
		return "action_sack"
	case strings.Contains(lower, "limited") || strings.Contains(lower, "event"):
		return "limited_time"
	default:
		return "social"
	}
}

// classifyModeCanonicalFromName dérive le mode canonique depuis le nom de game_variant.
func classifyModeCanonicalFromName(name string) string {
	lower := strings.ToLower(name)
	switch {
	case strings.Contains(lower, "slayer"):
		return "slayer"
	case strings.Contains(lower, "ctf") || strings.Contains(lower, "capture the flag"):
		return "ctf"
	case strings.Contains(lower, "oddball"):
		return "oddball"
	case strings.Contains(lower, "king of the hill") || strings.Contains(lower, "koth"):
		return "koth"
	case strings.Contains(lower, "stronghold"):
		return "strongholds"
	case strings.Contains(lower, "extraction"):
		return "extraction"
	case strings.Contains(lower, "fiesta"):
		return "fiesta"
	case strings.Contains(lower, "firefight") || strings.Contains(lower, "kotr"):
		return "firefight_kotr"
	case strings.Contains(lower, "attrition"):
		return "attrition"
	case strings.Contains(lower, "stockpile"):
		return "stockpile"
	case strings.Contains(lower, "total control"):
		return "total_control"
	default:
		return "unknown"
	}
}
