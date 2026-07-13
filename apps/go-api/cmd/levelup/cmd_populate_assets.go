// cmd_populate_assets.go — sous-commande `levelup populate-assets` : peuple
// asset_translations (multilingue) depuis l'API Discovery UGC.
//
// Historique : vivait dans cmd/populate-assets (binaire standalone, réécriture
// Go du script Python, Sprint 54). Migré en sous-commande de la CLI levelup
// (lot ops 2026-07-13) : le binaire standalone n'était PAS dans l'image Docker
// prod (journal deploy 2026-07-10) → le one-off était inexécutable en prod,
// alors que la CLI levelup y est déjà embarquée (/usr/local/bin/levelup).
// Logique inchangée, seul le wrapping CLI change (flag.NewFlagSet + cfg reçu).
//
// Usage :
//
//	levelup populate-assets [--types map,playlist] [--langs fr-FR,de-DE]
//	                        [--dry-run] [--force] [--concurrency N]
//	                        [--freshness JOURS] [--title-id slug]
package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"strings"
	"sync"

	"golang.org/x/sync/semaphore"

	"levelup/go-api/internal/config"
	"levelup/go-api/internal/domain"
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/platform/duckdb"
	"levelup/go-api/internal/platform/halo"
)

// runPopulateAssets est le point d'entrée de la sous-commande.
func runPopulateAssets(cfg *config.AppConfig, args []string) error {
	fs := flag.NewFlagSet("populate-assets", flag.ExitOnError)
	var (
		typesFlag       = fs.String("types", "", "Types d'assets (ex: map,playlist) — vide = tous")
		langsFlag       = fs.String("langs", "", "Langues BCP-47 (ex: fr-FR,de-DE) — vide = toutes")
		dryRun          = fs.Bool("dry-run", false, "Simule sans écrire")
		force           = fs.Bool("force", false, "Re-fetch même si déjà présent")
		concurrencyFlag = fs.Int("concurrency", 10, "Requêtes parallèles max")
		freshnessFlag   = fs.Int("freshness", 30, "Fraîcheur en jours")
		titleID         = fs.String("title-id", titlePkg.DefaultSlug, "Slug du titre (ex: halo_infinite)")
	)
	if err := fs.Parse(args); err != nil {
		return err
	}

	types := parseAssetTypes(*typesFlag)
	langs := parseLangs(*langsFlag)

	slog.Info("populate-assets",
		"types", types,
		"langs", langs,
		"dry_run", *dryRun,
		"force", *force,
		"concurrency", *concurrencyFlag,
		"freshness_days", *freshnessFlag,
		"title_id", *titleID,
	)

	ctx := context.Background()
	if err := runPopulateAssetsPipeline(
		ctx, cfg, types, langs, *dryRun, *force, *concurrencyFlag, *freshnessFlag, *titleID,
	); err != nil {
		return fmt.Errorf("populate-assets: %w", err)
	}
	slog.Info("populate-assets: terminé avec succès")
	return nil
}

//nolint:gocyclo // orchestrateur : boucle par asset_type avec rapport final (héritée du binaire standalone)
func runPopulateAssetsPipeline(
	ctx context.Context,
	cfg *config.AppConfig,
	types []halo.AssetType,
	langs []string,
	dryRun bool,
	force bool,
	concurrency int,
	freshnessDays int,
	titleID string,
) error {
	// Ouvrir metadata.duckdb DU TITRE ciblé (MT-16 / day-one 2e titre : le flag
	// --title-id était threadé au fetch d'assets mais IGNORÉ pour les chemins DB →
	// `--title-id X` écrivait dans la metadata de Halo. Corrigé : chemins résolus
	// pour titleID. Défaut du flag = DefaultSlug → byte-identique en mono-titre.)
	pr := titlePkg.NewPathResolver(cfg.RepoRoot)
	metaPath := pr.MetadataDBPath(titleID)
	metaDB, err := duckdb.OpenReadWrite(metaPath)
	if err != nil {
		return fmt.Errorf("open metadata.duckdb: %w", err)
	}
	defer metaDB.Close()

	metaRepo := duckdb.NewMetadataRepoFromDB(metaDB)

	// Ouvrir shared_matches_v2.duckdb DU TITRE ciblé.
	sharedPath := pr.SharedDBPath(titleID)
	sharedDB, err := duckdb.OpenReadOnly(sharedPath)
	if err != nil {
		return fmt.Errorf("open shared_matches_v2.duckdb: %w", err)
	}
	defer sharedDB.Close()

	provider := halo.DefaultHaloProvider

	totals := make(map[halo.AssetType]int)

	for _, assetType := range types {
		slog.Info("populate-assets: traitement asset_type", "type", assetType)

		// 1. Récupérer les asset_ids distincts
		assetIDs, err := metaRepo.GetDistinctAssetIDs(ctx, string(assetType), sharedDB)
		if err != nil {
			return fmt.Errorf("GetDistinctAssetIDs(%s): %w", assetType, err)
		}
		if len(assetIDs) == 0 {
			slog.Warn("aucun asset trouvé", "type", assetType)
			continue
		}
		slog.Info("assets distincts trouvés", "type", assetType, "count", len(assetIDs))

		// 2. Construire cache version_id
		// Note: nécessite auth API match stats, peut échouer en mode CLI standalone
		versionCache, err := buildVersionCache(ctx, provider, assetType, assetIDs, sharedDB)
		if err != nil {
			slog.Warn("buildVersionCache failed, using default version_id", "type", assetType, "err", err)
			versionCache = make(map[string]string) // Cache vide, version_id par défaut sera utilisé
		}
		covered := len(versionCache)
		slog.Info("version_ids récupérés", "type", assetType, "covered", covered, "total", len(assetIDs))

		// 3. Fetch traductions pour chaque langue en parallèle
		count, err := fetchAllLangs(
			ctx, provider, metaRepo,
			assetType, assetIDs, versionCache, langs,
			concurrency, freshnessDays, force, dryRun, titleID,
		)
		if err != nil {
			return fmt.Errorf("fetchAllLangs(%s): %w", assetType, err)
		}

		totals[assetType] = count
		slog.Info("asset_type terminé", "type", assetType, "upserted", count)
	}

	// Rapport final
	slog.Info("=== RÉSUMÉ ===")
	for assetType, count := range totals {
		slog.Info("total upserts", "type", assetType, "count", count)
	}

	// Comptes par langue (si pas dry-run)
	if !dryRun {
		for _, assetType := range types {
			counts, err := metaRepo.GetAssetTranslationCount(ctx, string(assetType))
			if err != nil {
				slog.Warn("GetAssetTranslationCount failed", "type", assetType, "err", err)
				continue
			}
			slog.Info("traductions en DB", "type", assetType, "counts", counts)
		}
	}

	return nil
}

// buildVersionCache construit le mapping asset_id → version_id via l'API match stats.
// Pour chaque asset, récupère un match_id représentatif et fetch ses stats.
func buildVersionCache(
	ctx context.Context,
	provider *halo.HaloProvider,
	assetType halo.AssetType,
	assetIDs []string,
	sharedDB *duckdb.DB,
) (map[string]string, error) {
	// Mapping asset_id → match_id représentatif
	assetToMatch, err := getRepresentativeMatches(ctx, assetType, sharedDB)
	_ = assetIDs // garde la variable pour la trace explicite des assets traités
	if err != nil {
		return nil, err
	}

	// Dédupliquer les match_ids
	uniqueMatches := make(map[string]bool)
	for _, matchID := range assetToMatch {
		uniqueMatches[matchID] = true
	}

	slog.Info("fetch version_ids via match stats", "unique_matches", len(uniqueMatches))

	cache := make(map[string]string)
	var mu sync.Mutex

	// Paralléliser les fetches de match stats
	sem := semaphore.NewWeighted(10) // Concurrency fixe pour version cache
	var wg sync.WaitGroup

	for matchID := range uniqueMatches {
		matchID := matchID
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = sem.Acquire(ctx, 1)
			defer sem.Release(1)

			// Sprint 54 : match stats publiques, pas besoin de tokens réels
			stats, err := provider.FetchMatchStats(ctx, matchID, nil)
			if err != nil {
				slog.Warn("fetch match stats failed", "match_id", matchID, "err", err)
				return
			}

			// Extraire version_ids depuis MatchInfo
			matchInfo, ok := stats["MatchInfo"].(map[string]interface{})
			if !ok {
				return
			}

			// Mapper asset_type → clé JSON
			jsonKey := halo.AssetTypeToMatchInfoKey[assetType]
			assetRef, ok := matchInfo[jsonKey].(map[string]interface{})
			if !ok {
				return
			}

			assetID, _ := assetRef["AssetId"].(string)
			versionID, _ := assetRef["VersionId"].(string)

			if assetID != "" && versionID != "" {
				mu.Lock()
				cache[assetID] = versionID
				mu.Unlock()
			}
		}()
	}

	wg.Wait()
	return cache, nil
}

// getRepresentativeMatches retourne un mapping asset_id → match_id représentatif.
func getRepresentativeMatches(
	ctx context.Context,
	assetType halo.AssetType,
	sharedDB *duckdb.DB,
) (map[string]string, error) {
	columnMap := map[halo.AssetType]string{
		halo.AssetTypeMap:         "map_id",
		halo.AssetTypePlaylist:    "playlist_id",
		halo.AssetTypePair:        "pair_id",
		halo.AssetTypeGameVariant: "game_variant_id",
	}

	column := columnMap[assetType]
	query := fmt.Sprintf(`
		SELECT %s, match_id
		FROM (
			SELECT %s, match_id, ROW_NUMBER() OVER (PARTITION BY %s) AS rn
			FROM match_registry
			WHERE %s IS NOT NULL
		)
		WHERE rn = 1
	`, column, column, column, column)

	rows, err := sharedDB.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query representative matches: %w", err)
	}
	defer rows.Close()

	result := make(map[string]string)
	for rows.Next() {
		var assetID, matchID string
		if err := rows.Scan(&assetID, &matchID); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		result[assetID] = matchID
	}

	return result, rows.Err()
}

// fetchAllLangs fetch les traductions pour toutes les langues en parallèle.
//
//nolint:funlen // orchestrateur i18n : 1 boucle externe + 1 sub-pipeline retry/fallback par lang
func fetchAllLangs(
	ctx context.Context,
	provider *halo.HaloProvider,
	repo *duckdb.MetadataRepo,
	assetType halo.AssetType,
	assetIDs []string,
	versionCache map[string]string,
	langs []string,
	concurrency int,
	freshnessDays int,
	force bool,
	dryRun bool,
	titleID string,
) (int, error) {
	totalCount := 0
	var mu sync.Mutex

	sem := semaphore.NewWeighted(int64(concurrency))
	var wg sync.WaitGroup

	for _, lang := range langs {
		lang := lang

		// Lire l'état courant de la DB pour cette langue
		existing, err := repo.GetExistingTranslations(ctx, string(assetType), lang, freshnessDays)
		if err != nil {
			return 0, fmt.Errorf("GetExistingTranslations(%s, %s): %w", assetType, lang, err)
		}

		toFetch := filterToFetch(assetIDs, existing, force)
		alreadyCount := len(assetIDs) - len(toFetch)

		if len(toFetch) == 0 {
			slog.Info("langue déjà complète", "type", assetType, "lang", lang, "already", alreadyCount)
			continue
		}

		slog.Info("langue à traiter",
			"type", assetType,
			"lang", lang,
			"to_fetch", len(toFetch),
			"already", alreadyCount,
		)

		wg.Add(1)
		go func() {
			defer wg.Done()

			count := 0
			for _, assetID := range toFetch {
				assetID := assetID
				_ = sem.Acquire(ctx, 1)

				// Déterminer version_id (fallback à "1" si non disponible dans cache)
				versionID, ok := versionCache[assetID]
				if !ok {
					versionID = "1" // Version par défaut quand match stats non accessible
					slog.Debug("using default version_id", "asset_id", assetID, "version_id", versionID)
				}

				asset, err := provider.FetchAsset(ctx, assetType, titleID, assetID, versionID, lang)
				sem.Release(1)

				if err != nil {
					slog.Debug("fetch asset failed",
						"type", assetType,
						"asset_id", assetID,
						"lang", lang,
						"err", err,
					)
					continue
				}

				if asset.PublicName == "" {
					slog.Debug("PublicName vide",
						"type", assetType,
						"asset_id", assetID,
						"lang", lang,
					)
					continue
				}

				if !dryRun {
					if err := repo.UpsertAssetTranslation(
						ctx, assetID, string(assetType), lang, asset.PublicName, asset.Description,
					); err != nil {
						slog.Warn("upsert failed",
							"type", assetType,
							"asset_id", assetID,
							"lang", lang,
							"err", err,
						)
						continue
					}
				}

				count++
				if count%100 == 0 {
					slog.Info("progress", "type", assetType, "lang", lang, "fetched", count)
				}
			}

			mu.Lock()
			totalCount += count
			mu.Unlock()

			slog.Info("langue terminée", "type", assetType, "lang", lang, "upserted", count)
		}()
	}

	wg.Wait()
	return totalCount, nil
}

// filterToFetch retourne les asset_ids à fetch (non présents ou force=true).
func filterToFetch(assetIDs []string, existing map[string]bool, force bool) []string {
	if force {
		return assetIDs
	}
	var result []string
	for _, id := range assetIDs {
		if !existing[id] {
			result = append(result, id)
		}
	}
	return result
}

// parseAssetTypes parse la string "--types map,playlist" en slice d'AssetType.
func parseAssetTypes(raw string) []halo.AssetType {
	if raw == "" {
		return []halo.AssetType{
			halo.AssetTypeMap,
			halo.AssetTypePlaylist,
			halo.AssetTypePair,
			halo.AssetTypeGameVariant,
		}
	}
	parts := strings.Split(raw, ",")
	var result []halo.AssetType
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, halo.AssetType(p))
		}
	}
	return result
}

// parseLangs parse la string "--langs fr-FR,de-DE" en slice de codes BCP-47.
func parseLangs(raw string) []string {
	if raw == "" {
		return domain.TargetLanguages
	}
	parts := strings.Split(raw, ",")
	validSet := make(map[string]bool)
	for _, lang := range domain.TargetLanguages {
		validSet[lang] = true
	}

	var result []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if validSet[p] {
			result = append(result, p)
		} else if p != "" {
			slog.Warn("langue invalide ignorée", "lang", p)
		}
	}
	if len(result) == 0 {
		return []string{domain.DefaultLanguage}
	}
	return result
}
